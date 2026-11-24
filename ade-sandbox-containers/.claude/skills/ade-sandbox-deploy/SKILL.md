---
name: ade-sandbox-deploy
description: Rebuild, publish, and (re)deploy the Orca + cmux dev sandbox containers to the k8s `ade-sandbox` namespace, end to end, with verification at every step. Use for "rebuild and publish the sandbox images", "roll out the sandbox to k8s", "update the dev containers", "bring the sandbox back up", or after editing `Dockerfile.orca`/`Dockerfile.cmux` or anything under `k8s/`. Not for one-off `docker run` experiments or for changing the pinned tool versions themselves (that's just editing the `ARG` pairs).
---

# ADE sandbox deploy

Rebuild → publish → deploy → verify, for the two sandbox runtimes (`orca`, `cmux`)
in namespace `ade-sandbox`. Every step has a real failure mode documented in
`../../README.md` and `../../k8s/README.md`; this skill is the ordered checklist
that catches them before they cost a rebuild cycle.

## Ground rules

- **Registry is plain HTTP** (`10.10.111.134:5000`). The Docker daemon must list it
  under `insecure-registries`, and the `docker-container` buildx builder needs its
  own buildkitd config (`[registry."10.10.111.134:5000"] http = true, insecure = true`).
  `docker buildx imagetools inspect` does **not** respect the daemon's insecure
  list — query the registry API directly with `curl` instead (see Step 3).
- **Cluster nodes are all `linux/amd64`.** The multi-arch build's `amd64` entry is
  what actually runs; `arm64` is for local Apple Silicon `docker run` testing.
- **Order matters**: namespace → Services (wait for `EXTERNAL-IP`) → Secret → RBAC →
  StatefulSets. Orca reads `ORCA_PAIRING_ADDRESS` at startup and advertises it to
  clients immediately; if the LB IP isn't assigned yet, it advertises nothing useful.
- **`k8s/rbac.yaml` before `k8s/cmux-statefulset.yaml`.** The cmux pod names
  `serviceAccountName: cmux-dev`; applying the StatefulSet first leaves the pod in
  `CreateContainerConfigError` until the ServiceAccount exists. Orca has no
  `serviceAccountName` and needs no RBAC — don't add one to it reflexively.
- **PVCs are `reclaimPolicy: Delete`.** Deleting a PVC wipes 120Gi of workspace
  irreversibly. Deleting/restarting a StatefulSet does **not** delete its PVCs.
- **`$HOME` outside `~/workspace` is ephemeral** (only `workspace` is a volume
  mount). Orca's pairing state and cmux's sshd host key reset on every pod
  replacement — expect to re-pair Orca and `ssh-keygen -R '[<lb-ip>]:2222'` after.

## Step 0 — Prereqs

```sh
docker buildx ls | grep -q '^multi-arch' || echo "MISSING: multi-arch builder"
docker buildx inspect multi-arch | grep -q 'Status:.*running' || docker buildx inspect multi-arch --bootstrap
kubectl cluster-info | head -1
curl -s http://10.10.111.134:5000/v2/_catalog >/dev/null && echo "registry reachable"
```

If the builder doesn't exist yet:
```sh
docker buildx create --name multi-arch --driver docker-container --bootstrap
```

## Step 1 — Rebuild + push (multi-arch)

Run from the repo root (`ade-sandbox-containers/`). Both builds are independent —
run them in parallel (two background shells) to save wall-clock time; they share
the buildx cache safely.

```sh
docker buildx use multi-arch
docker buildx build --platform linux/amd64,linux/arm64 \
  -t 10.10.111.134:5000/ade-sandbox/orca:latest --push -f Dockerfile.orca .
docker buildx build --platform linux/amd64,linux/arm64 \
  -t 10.10.111.134:5000/ade-sandbox/cmux:latest --push -f Dockerfile.cmux .
```

Unchanged Dockerfiles will hit the buildx cache and finish in seconds — that's
expected and fine, it still re-pushes the manifest.

## Step 2 — Confirm the push actually uploaded (don't trust exit 0)

`buildx` exits 0 even when the registry's in-memory blob-descriptor cache lies
about a blob being present (see `k8s/README.md` §"If a push silently uploads
nothing"). Always verify the OCI index directly:

```sh
for img in orca cmux; do
  echo "=== $img ==="
  curl -s -H "Accept: application/vnd.oci.image.index.v1+json" \
    "http://10.10.111.134:5000/v2/ade-sandbox/$img/manifests/latest" \
  | python3 -c "import json,sys; d=json.load(sys.stdin); [print(m['platform'], m['digest'][:25]) for m in d['manifests'] if m['platform']['architecture']!='unknown']"
done
```

Expect exactly two rows per image: `amd64` and `arm64`. If a digest looks
unchanged from a previous run when you expected new content, or a platform is
missing, that's the stale-cache symptom — restart the registry pod and re-push:

```sh
kubectl -n registry-ns delete pod -l app=local-registry   # RWO PVC: delete, don't just rollout restart
# wait for the registry pod to be Running, then re-run Step 1
```

## Step 3 — Deploy / update the namespace

Fresh namespace (first time only):

```sh
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/services.yaml
kubectl -n ade-sandbox get svc -w   # wait until both *-lb have an EXTERNAL-IP
kubectl -n ade-sandbox create secret generic cmux-ssh-key \
  --from-file=publicKey=$HOME/.ssh/id_ed25519.pub
kubectl apply -f k8s/rbac.yaml             # ServiceAccount cmux-dev (see ground rules)
kubectl apply -f k8s/orca-statefulset.yaml -f k8s/cmux-statefulset.yaml
```

If `k8s/create-agent-secret.sh` predates the `GH_TOKEN` key, re-run it before the
rollout — it now also pulls the gh token via `gh auth token` (on macOS the token is
in the Keychain, not in `~/.zshrc`, so the script can't read it from the rc file).
Both StatefulSets mark the key `optional: true`, so a missing value warns instead
of blocking the pod, and `gh` inside the sandbox stays unauthenticated.

Re-deploy after a rebuild (namespace/services/secret already exist):

```sh
kubectl -n ade-sandbox rollout restart statefulset/orca-dev statefulset/cmux-dev
kubectl -n ade-sandbox rollout status statefulset/cmux-dev --timeout=120s
kubectl -n ade-sandbox rollout status statefulset/orca-dev --timeout=180s   # Orca takes 60-90s to bind
```

`imagePullPolicy: Always` on both StatefulSets means a restart is all that's
needed to pick up a new `:latest` — no manifest edit required.

If the LB services were ever deleted/recreated and got new IPs, update
`ORCA_PAIRING_ADDRESS` in `k8s/orca-statefulset.yaml` to match before the next
rollout, and re-verify the cmux connect string in `k8s/README.md`.

## Step 4 — Verify

```sh
# 1. Pods up, on amd64 nodes
kubectl -n ade-sandbox get pods -o wide

# 2. Architecture inside the pod is x86_64 (confirms the amd64 manifest entry is what got pulled)
kubectl -n ade-sandbox exec cmux-dev-0 -- uname -m
kubectl -n ade-sandbox exec orca-dev-0 -- uname -m

# 3. Toolchain + agent CLIs resolve to the pinned versions
kubectl -n ade-sandbox exec cmux-dev-0 -- bash -lc '
  claude --version; codex --version; opencode --version
  go version; java -version 2>&1 | head -1; python3.12 --version
  bun --version; uv --version; kubectl version --client | head -1; gh --version | head -1
  cmuxd-remote version'

# 4. Orca emitted its ready contract with pairing available
kubectl -n ade-sandbox logs orca-dev-0 | grep orca_server_ready

# 5. External endpoints actually reachable (not just "pod Running")
nc -z -G 5 10.10.111.139 2222 && echo "cmux OK"
nc -z -G 5 10.10.111.138 6768 && echo "orca OK"

# 6. Agent integrations, over SSH (not kubectl exec — see cmux-sandbox-connect on
#    why exec and ssh see different environments). Config files, gh auth, and the
#    scoped kubectl from k8s/rbac.yaml.
ssh -p 2222 -o BatchMode=yes cmux@10.10.111.139 '
  ls -l ~/.mcp.json ~/.claude/settings.json ~/.codex/config.toml
  gh auth status
  kubectl get pods -n ade-sandbox
  kubectl auth can-i get pods -n kube-system   # "no" is correct: namespace-scoped
'
```

`~/.mcp.json` must be checked as a **file**, not inferred from the ConfigMap. It is
mounted via `subPath`, and the earlier pattern of `cp`-ing it from the init
container silently lost it — a bare `$HOME` path isn't a volume, so the init
container's copy dies with the init container. `gh auth status` likewise can't be
inferred from the binary: `gh --version` succeeds with no credentials at all.

A passing run looks like: both pods `1/1 Running`, `x86_64` from both `uname -m`
calls, every tool version matching the `ARG` pins in the Dockerfiles, an
`orca_server_ready` line with `"pairing":{"available":true,...}`, both `nc`
checks succeeding, and check 6 showing all three config files, `Logged in to
github.com account syst-m (GH_TOKEN)`, the namespace pod list, and `no` for
kube-system.

## Step 5 — Teardown (only when the data is really wanted gone)

```sh
kubectl -n ade-sandbox delete -f k8s/cmux-statefulset.yaml -f k8s/orca-statefulset.yaml
# PVCs survive the StatefulSet delete above. Deleting them destroys 120Gi each, irreversibly:
kubectl -n ade-sandbox delete pvc -l 'app in (orca-dev,cmux-dev)'
kubectl -n ade-sandbox delete -f k8s/services.yaml -f k8s/namespace.yaml
```

## Version bumps

Edit the `ARG *_VERSION` + `*_SHA*` pairs at the top of each Dockerfile — never
edit a running image. Checksum sources: go.dev/dl JSON, Adoptium API,
`dl.k8s.io/release/<ver>/bin/linux/<arch>/kubectl.sha256`, Orca's
`latest-linux*.yml` release manifests, cmux's `cmuxd-remote-checksums.txt`.
Then re-run Steps 1–4.
