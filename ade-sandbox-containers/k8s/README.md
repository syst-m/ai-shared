# k8s sandbox deployment

Two long-lived dev containers in namespace `ade-sandbox`, one per ADE runtime.
Each is a single-replica StatefulSet with its own 120Gi persistent volume.

| Workload | Runtime | Image | External | Port |
|---|---|---|---|---|
| `statefulset/orca-dev` | Orca headless `serve` | `ade-sandbox/orca:latest` | `10.10.111.138` | 6768 |
| `statefulset/cmux-dev` | `cmuxd-remote` over sshd | `ade-sandbox/cmux:latest` | `10.10.111.139` | 2222 |

Both images bake in the same pinned toolchain plus the agent CLIs
(`claude`, `codex`, `opencode`) under `/opt/node/bin` — see `../README.md`.

## Layout

```
k8s/
  namespace.yaml           # ade-sandbox
  services.yaml            # headless (governing) + LoadBalancer per runtime
  agent-config.yaml        # ade-agent-env (env vars) + ade-agent-files (codex/claude files)
  orca-statefulset.yaml    # 1 replica, 120Gi iscsi-ext4 -> /home/orca/workspace
  cmux-statefulset.yaml    # 1 replica, 120Gi iscsi-ext4 -> /home/cmux/workspace
  create-agent-secret.sh   # builds ade-agent-keys from ~/.zshrc (values via env, never argv)
```

## Apply

Order matters: the Services must exist before the StatefulSets, because Orca
advertises its pairing address at startup and needs the LB IP already assigned.

```sh
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/services.yaml
kubectl -n ade-sandbox get svc -w          # wait for EXTERNAL-IP on both *-lb

# authorized key for cmux ssh (out-of-band; nothing key-shaped is committed)
kubectl -n ade-sandbox create secret generic cmux-ssh-key \
  --from-file=publicKey=$HOME/.ssh/id_ed25519.pub

# agent CLI credentials (ANTHROPIC_API_KEY / ANTHROPIC_AUTH_TOKEN / OPENAI_API_KEY),
# read out of ~/.zshrc into a Secret. The values are passed via the environment,
# never argv, so they stay out of the process list and shell history.
k8s/create-agent-secret.sh

kubectl apply -f k8s/agent-config.yaml
kubectl apply -f k8s/orca-statefulset.yaml -f k8s/cmux-statefulset.yaml
kubectl -n ade-sandbox get pods -w
```

## Agent credentials and config

The agent CLIs (`claude`, `codex`) get their settings from three objects, and the
split is deliberate:

| Object | Holds | Consumed by |
|---|---|---|
| Secret `ade-agent-keys` | the API keys | `valueFrom.secretKeyRef` in both StatefulSets |
| ConfigMap `ade-agent-env` | non-secret scalars (`ANTHROPIC_BASE_URL`, `CLAUDE_CODE_*`, `OPENAI_BASE_URL`) | `envFrom` in both StatefulSets |
| ConfigMap `ade-agent-files` | `codex-config.toml`, `claude-settings.json` | mounted read-only, copied into `$HOME` by the `seed-agent-config` initContainer |

No secret value is committed to this repo — the StatefulSets reference the Secret by
name, and `create-agent-secret.sh` is the only place the values are read.

`ade-agent-env` and `ade-agent-files` are separate ConfigMaps because `envFrom`
exports *every* key as an env var, and a file key like `codex-config.toml` is not a
valid env var name — kubelet skips it with an `InvalidVariableNames` warning.

On **cmux**, sshd's `SetEnv` is generated at container start from the full container
environment, which is what makes these vars visible to an SSH-launched shell — see
`../Dockerfile.cmux`. **Orca has no sshd**, so `envFrom`/`secretKeyRef` alone cover it.

Namespaces are written explicitly in every manifest. `kubectl apply -f a.yaml
-f b.yaml` does **not** carry a namespace from one file into the next — anything
without an explicit `metadata.namespace` silently lands in `default`.

## Storage

`volumeClaimTemplates` on StorageClass `iscsi-ext4` (default class, `ReadWriteOnce`,
`allowVolumeExpansion: true`), 120Gi each, mounted at `~/workspace` only.

Two consequences worth knowing:

- **`reclaimPolicy: Delete`.** Deleting a PVC destroys the volume and everything
  in it. `kubectl delete pvc workspace-0-... -n ade-sandbox` is not reversible.
  Scaling the StatefulSet to 0 keeps the PVC; deleting the StatefulSet does not.
- **Everything outside `~/workspace` is ephemeral.** `$HOME/.config`, `$HOME/.npm`,
  shell history, `go install` output, and `$HOME/.ssh` are on the container's
  writable layer and reset on every pod replacement. In particular:
  - Orca's profile and paired-device state reset on restart — re-pair after a rollout.
  - cmux regenerates its sshd host key, so clear the stale entry afterwards:
    `ssh-keygen -R '[10.10.111.139]:2222'`

  If either becomes annoying, the fix is a second small `volumeClaimTemplate`
  mounted at `$HOME/.config` (or `.ssh`) rather than moving the whole home dir.

`fsGroup: 1000` on the pod makes kubelet chown the freshly-formatted ext4 volume,
otherwise the uid-1000 service user cannot write to a new volume.

## Connecting

```sh
# cmux: from the macOS app
cmux ssh -p 2222 cmux@10.10.111.139

# Orca: read the ready contract (pairing URL, version) from the pod log
kubectl -n ade-sandbox logs orca-dev-0 | head -5
```

In-cluster DNS for the pods themselves:
`orca-dev-0.orca-dev.ade-sandbox.svc.cluster.local`,
`cmux-dev-0.cmux-dev.ade-sandbox.svc.cluster.local`.

## Updating the images

The images are mutable `:latest` with `imagePullPolicy: Always`, so a rebuild and
push does nothing until the pods restart:

```sh
docker buildx build --builder multi-arch --platform linux/amd64,linux/arm64 \
  -t 10.10.111.134:5000/ade-sandbox/orca:latest --push -f Dockerfile.orca .
kubectl -n ade-sandbox rollout restart statefulset/orca-dev
kubectl -n ade-sandbox rollout status  statefulset/orca-dev
```

### If a push silently uploads nothing

The registry (`registry-ns/local-registry`) runs with
`storage.cache.blobdescriptor: inmemory`. That cache maps blob digest -> size and
survives deletion of the blobs underneath it. If anything removes blob data out
from under the running registry — a `registry garbage-collect`, or a push that
died on a full disk — then `HEAD /v2/.../blobs/sha256:<d>` keeps answering
`200 OK` with the correct `Content-Length` for a blob that is no longer on disk.
buildx trusts that HEAD, skips the upload, pushes the manifest anyway and exits 0.
The image looks published but is unrecoverable: pulls fail with
`short read: expected N bytes but got 0`.

Symptom check, and the fix:

```sh
# 200 here while the file is absent == stale cache
curl -sI http://10.10.111.134:5000/v2/ade-sandbox/cmux/blobs/sha256:<digest> | head -1

# clearing the cache requires a registry restart
kubectl -n registry-ns rollout restart deploy/local-registry
```

The registry PVC is `ReadWriteOnce`, so `rollout restart` deadlocks: the new pod
cannot attach the volume until the old one releases it. Delete the old pod
explicitly. Re-push **after** the restart — buildx re-dedups against the registry
on every push, so the same command uploads correctly once HEAD tells the truth.
Deleting the repo directory under the registry does *not* help; the cache is not
repo-scoped.

## Verify

```sh
kubectl -n ade-sandbox get sts,pods,pvc,svc -o wide
kubectl -n ade-sandbox exec cmux-dev-0 -- bash -lc '
  claude --version; codex --version; opencode --version
  go version; java -version 2>&1 | head -1; python3.12 --version
  bun --version; uv --version; kubectl version --client; gh --version | head -1'
kubectl -n ade-sandbox exec orca-dev-0 -- df -h /home/orca/workspace
```

## Teardown

```sh
kubectl -n ade-sandbox delete -f k8s/cmux-statefulset.yaml -f k8s/orca-statefulset.yaml
# The PVCs survive deleting the StatefulSets. Delete them only when the data is
# really wanted gone — iscsi-ext4 reclaims (wipes) on PVC deletion:
kubectl -n ade-sandbox delete pvc -l 'app in (orca-dev,cmux-dev)'
kubectl -n ade-sandbox delete -f k8s/services.yaml -f k8s/namespace.yaml
```
