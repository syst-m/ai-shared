---
name: ade-sandbox-update-rollout
description: End-to-end "update the sandboxes" flow — discover the latest agent harness versions (claude-code / codex / opencode), reconcile env vars and the k8s Secret against ~/.zshrc, bump the Dockerfile pins, build + push both multi-arch images, roll the StatefulSets, and run a functional verification pass. Use for "update the harness versions and roll it out", "bump claude/codex/opencode in the containers", "bring the sandboxes up to date", or as the umbrella over `ade-sandbox-deploy` when the trigger is a *version/config update* rather than a plain re-deploy of unchanged images.
---

# ADE sandbox update + rollout

The orchestration skill for **changing** what the sandboxes run, then shipping it.
`ade-sandbox-deploy` is the pure build→push→deploy→verify pipeline for images
that are already correct; this skill is what you run when the *content* is
changing — new harness versions, an env-var fix, a rotated credential — and you
want the whole chain (discover → reconcile → bump → build → push → rollout →
prove) done in order. Steps 4–6 below are exactly `ade-sandbox-deploy` Steps
1–3; they're inlined here so this skill is self-contained, but read the deploy
skill's ground rules (insecure registry, amd64-only nodes, RBAC-before-cmux)
before the first run.

## Ground rules

- **No secrets in the repo.** The Dockerfiles and `k8s/*.yaml` carry no key
  values. Credentials live only in the `ade-agent-keys` Secret (created out of
  band) and in `~/.zshrc`. Never `echo` a secret value into a log or commit it;
  print key *names* only.
- **`~/.zshrc` is the source of truth** for the laptop's active env. The k8s
  ConfigMap (`k8s/agent-config.yaml`) must mirror its active (uncommented)
  values so the pods behave identically. Cross-reference before you ship.
- **Both Dockerfiles move together.** `Dockerfile.cmux` and `Dockerfile.orca`
  share the same three harness `ARG`s — bump both to the same versions or the
  two runtimes drift apart.
- **Don't trust a build's exit 0.** A `curl (35) SSL_ERROR_SYSCALL` on any
  download (github.com, dl.k8s.io, nodejs.org) fails the build even with
  `--retry 3`; verify the registry digest actually changed (Step 5).

## Step 1 — Discover the latest harness versions

Query npm for the three agent CLIs; compare against the current `ARG` pins in
both Dockerfiles. Only bump if a newer version exists and you intend to take it.

```sh
npm view @anthropic-ai/claude-code version
npm view @openai/codex version
npm view opencode-ai version
```

The current pins are in the runtime-stage `ARG` block of each Dockerfile
(`CLAUDE_CODE_VERSION`, `CODEX_VERSION`, `OPENCODE_VERSION`).

## Step 2 — Reconcile env vars (zshrc ↔ ConfigMap)

Read the active exports in `~/.zshrc` and diff them against `k8s/agent-config.yaml`
`ade-agent-env`. Fix protocol/value mismatches (a real one seen: ConfigMap had
`OPENAI_BASE_URL: http://…` while zshrc and the codex config both used `https://…`).
Carry over only the active, uncommented exports — commented model overrides in
zshrc are intentionally not mirrored.

```sh
grep -E '^[[:space:]]*export (ANTHROPIC|OPENAI|CLAUDE_CODE)' ~/.zshrc
```

File-only keys (`codex-config.toml`, `claude-settings.json`, `mcp.json`) live in
the `ade-agent-files` ConfigMap, not the env one — `envFrom` would reject them as
env-var names.

## Step 3 — Refresh the Secret (no values logged)

`k8s/create-agent-secret.sh` builds `ade-agent-keys` from env-or-zshrc and
server-side-applies it (idempotent). The shell the Bash tool runs inherits the
Claude Code process env, which may have *some* keys set and others unset — to
force the script to read the authoritative zshrc values, blank the env first:

```sh
env -u ANTHROPIC_API_KEY -u ANTHROPIC_AUTH_TOKEN -u OPENAI_API_KEY \
  bash k8s/create-agent-secret.sh
```

The script's `resolve()` follows a lone `${NAME}` reference (zshrc exports
`OPENAI_API_KEY="${ANTHROPIC_AUTH_TOKEN}"`), so a plain grep no longer captures
that literally. `GH_TOKEN` comes from `gh auth token` (Keychain on macOS), not
zshrc; it's soft-required (`optional: true` in both StatefulSets). Output is
`secret/ade-agent-keys unchanged` when values already match — that's the
confirmation you want, and it logs no secret values.

## Step 4 — Bump the Dockerfile pins

Edit the three harness `ARG`s in **both** `Dockerfile.cmux` and `Dockerfile.orca`
to the versions from Step 1. These are plain version strings (no checksum pair —
npm resolves and the registry serves them), unlike the `*_SHA*`-paired toolchain
ARGs (go/jdk/kubectl/gh/cmuxd/node) documented in `ade-sandbox-deploy`
§"Version bumps".

## Step 5 — Build + push both images (multi-arch)

From the repo root. The two builds are independent — run them as two background
shells to share wall-clock and the buildx cache safely. Cached layers make an
unchanged rebuild finish in seconds; only changed layers rebuild.

```sh
docker buildx build --builder multi-arch --platform linux/amd64,linux/arm64 \
  -t 10.10.111.134:5000/ade-sandbox/cmux:latest --push -f Dockerfile.cmux .
docker buildx build --builder multi-arch --platform linux/amd64,linux/arm64 \
  -t 10.10.111.134:5000/ade-sandbox/orca:latest --push -f Dockerfile.orca .
```

If a build dies with `curl: (35) OpenSSL SSL_connect: SSL_ERROR_SYSCALL` on a
download, it's a transient network flake (common when two multi-arch builds hit
github.com/dl.k8s.io at once) — just re-run it; cached layers make the retry
cheap. Confirm the builder exists first:
`docker buildx inspect multi-arch | grep Status:` (create with
`docker buildx create --name multi-arch --driver docker-container --bootstrap`).

## Step 6 — Verify the push (don't trust exit 0)

`buildx` can exit 0 while the registry's blob cache lies. Query the OCI index
directly and expect **two rows per image** (amd64 + arm64), with digests that
*differ* from the previous run:

```sh
for img in cmux orca; do
  echo "=== $img ==="
  curl -s -H "Accept: application/vnd.oci.image.index.v1+json" \
    "http://10.10.111.134:5000/v2/ade-sandbox/$img/manifests/latest" \
  | python3 -c "import json,sys; d=json.load(sys.stdin); [print(m['platform']['architecture'], m['digest'][:25]) for m in d['manifests'] if m['platform']['architecture']!='unknown']"
done
```

Also grep the build log for the installed versions to prove the bump took:
`grep -iE "claude-code@|opencode-ai@" /tmp/<build>.log`. A missing platform or an
unchanged digest when you expected new content = stale registry cache; restart the
registry pod (`kubectl -n registry-ns delete pod -l app=local-registry`, RWO PVC →
delete not rollout) and re-push.

## Step 7 — Roll out the StatefulSets

`imagePullPolicy: Always` means a restart pulls the new `:latest` — no manifest
edit. Orca takes 60–90s to bind, so give it the longer timeout.

```sh
kubectl -n ade-sandbox rollout restart statefulset/orca-dev statefulset/cmux-dev
kubectl -n ade-sandbox rollout status statefulset/cmux-dev --timeout=180s
kubectl -n ade-sandbox rollout status statefulset/orca-dev --timeout=240s
```

## Step 8 — Functional verification

The point is to prove the pods *work*, not just that they're `Running`. Run this
suite and compare the reported versions against the Step 4 pins.

```sh
# Endpoints (LB IPs from `kubectl -n ade-sandbox get svc`; default cmux .139, orca .138)
nc -z -G 5 10.10.111.139 2222 && echo "cmux OK"
nc -z -G 5 10.10.111.138 6768 && echo "orca OK"

# cmux: arch + every tool resolves to the pinned version
kubectl -n ade-sandbox exec cmux-dev-0 -- bash -lc '
  uname -m
  claude --version; codex --version; opencode --version
  go version; java -version 2>&1 | head -1; python3.12 --version
  bun --version; uv --version; kubectl version --client | head -1
  gh --version | head -1; cmuxd-remote version'

# cmux over SSH (NOT just kubectl exec — exec and ssh see different env; see
# cmux-sandbox-connect). A fresh pod rotates the sshd host key: drop the stale
# known_hosts entry first or BatchMode refuses the changed key.
ssh-keygen -R '[10.10.111.139]:2222' >/dev/null 2>&1
ssh -p 2222 -o BatchMode=yes -o StrictHostKeyChecking=accept-new cmux@10.10.111.139 '
  echo "shell: $(whoami)@$(hostname)"
  ls ~/.mcp.json ~/.claude/settings.json ~/.codex/config.toml
  gh auth status 2>&1 | head -3
  kubectl get pods -n ade-sandbox
  kubectl auth can-i get pods -n kube-system   # "no" is correct: namespace-scoped'

# orca: arch + tools, and the ready contract with pairing available
kubectl -n ade-sandbox exec orca-dev-0 -- bash -lc 'uname -m; claude --version; codex --version; opencode --version; bun --version'
kubectl -n ade-sandbox logs orca-dev-0 --tail=200 | grep -c orca_server_ready   # expect >=1
```

A passing run: both pods `1/1 Running`, `x86_64` from both `uname -m`, every tool
matching the Step 4 pins, all three cmux config files present, `gh auth status`
→ `Logged in to github.com account syst-m (GH_TOKEN)`, scoped kubectl (`no` for
kube-system), both `nc` checks OK, and an `orca_server_ready` line with
`"pairing":{"available":true}`.

## Gotchas learned

- **curl (35) SSL flake** on a download fails the build despite `--retry 3` —
  re-run, don't edit the Dockerfile.
- **Secret script and the tool shell**: the Bash tool's env inherits Claude
  Code's own keys, not zshrc's. Blank them (`env -u …`) to force zshrc values.
- **ssh host key rotates** on every pod replacement — `ssh-keygen -R
  '[<lb-ip>]:2222'` before the BatchMode check, and expect to re-pair Orca.
- **Check `~/.mcp.json` as a file**, not from the ConfigMap — it's `subPath`
  mounted; a bare `$HOME` path isn't a volume, so an init-container `cp` of it
  dies with the init container.
- **`gh --version` succeeds with no credentials** — only `gh auth status` proves
  the token reached the pod.
