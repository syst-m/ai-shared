# ade-sandbox-containers

Remote runtime sandbox images with **optional ADE alignment**. Both images ship
the same pinned dev toolchain; only the ADE (Agent Development Environment)
layer differs, so a workload can pick its orchestrator without changing the
base environment.

| Image | ADE layer | Registry |
|---|---|---|
| `Dockerfile.orca` | Orca (onorca.dev) headless `orca serve` runtime | `10.10.111.134:5000/ade-sandbox/orca:latest` |
| `Dockerfile.cmux` | cmux (manaflow-ai/cmux) remote daemon `cmuxd-remote` | `10.10.111.134:5000/ade-sandbox/cmux:latest` |

Both are pushed as **multi-arch manifests: `linux/amd64` (k8s) + `linux/arm64`**.

## Shared toolchain (identical in both images)

| Tool | Version | Source / pin |
|---|---|---|
| Base OS | Ubuntu 24.04 (noble) | `ubuntu:24.04` |
| Bun | 1.4.2 | `oven/bun:1.4.2` |
| uv / uvx | 0.12.21 | `ghcr.io/astral-sh/uv:0.12.21` |
| Python | 3.12.3 (+venv, pip) | Ubuntu `python3.12` |
| Go | 1.26.8 | go.dev tarball, sha256-pinned |
| JDK | Temurin 25.0.4.1+1 (LTS) | Adoptium tarball, sha256-pinned |
| kubectl | v1.37.1 | dl.k8s.io, sha256-pinned |
| gh CLI | 2.102.0 | GitHub release, sha256-pinned |
| git / openssh-client | 2.43.0 | apt |

Design practices: multi-stage builds, every downloaded artifact checksum-verified
(sha256/sha512 from official sources), root-owned toolchains the service user
cannot modify, `tini` as PID 1 (zombie reaping), non-root service user
(`uid/gid 1000`), `HEALTHCHECK`, no build tooling in the final image.

## Orca image (`Dockerfile.orca`)

Runs the Orca runtime headless via the official `.deb` payload
(`/opt/Orca/orca-ide` + the `orca-ide` CLI shim the upstream postinst puts on
PATH for headless servers). Orca auto-starts its own Xvfb when `DISPLAY` is
unset; `LIBGL_ALWAYS_SOFTWARE=1` is set.

The deb is extracted with `dpkg-deb -x` at build time — no arch-specific binary
is executed during the build, which is what makes the cross-arch build work on
Apple Silicon (Rosetta cannot exec the amd64 AppImage's static-pie runtime).

`ELECTRON_DISABLE_SANDBOX=1` is baked in: Docker's default seccomp profile
blocks the Chromium namespace sandbox. The container already runs as non-root;
to re-enable the sandbox, use a permissive seccomp profile and unset the var.

Run:

```sh
docker run -d -p 6768:6768 \
  -e ORCA_PAIRING_ADDRESS=host.example.com \
  10.10.111.134:5000/ade-sandbox/orca:latest
```

| Env | Default | Purpose |
|---|---|---|
| `ORCA_PORT` | `6768` | WebSocket listener port |
| `ORCA_PAIRING_ADDRESS` | — | Address advertised to pairing clients (LAN/Tailscale/reverse-proxy URL) |

Default command emits the versioned `orca_server_ready` JSON contract on
stdout (pairing URL included). Any container args override it
(e.g. `docker run <image> bash`). Verified: container reaches `healthy` and
emits the ready JSON with `pairing.available: true`.

## cmux image (`Dockerfile.cmux`)

cmux itself is a **macOS-only** app; the Linux-side runtime is `cmuxd-remote`,
the Go daemon the cmux app bootstraps onto remote hosts. This image bakes in
`cmuxd-remote` v0.65.0 (upstream prebuilt, sha256-pinned) — no Electron, no
Xvfb, much slimmer than the Orca image.

Two headless modes, both fully unprivileged (sshd runs as the `cmux` user on
port 2222 with key-only auth):

**1. SSH mode (default)** — for `cmux ssh` from the macOS app:

```sh
docker run -d -p 2222:2222 \
  -e CMUX_SSH_PUBLIC_KEY="ssh-ed25519 AAAA... you@laptop" \
  10.10.111.134:5000/ade-sandbox/cmux:latest
```

Then from your Mac: `cmux ssh -p 2222 cmux@<host>` (or add an `~/.ssh/config`
entry). Alternatively mount a real `authorized_keys` at
`/home/cmux/.ssh/authorized_keys`. The baked `cmuxd-remote` is found on
non-interactive SSH, so the app skips its upload step.

**2. WebSocket PTY mode** (cloud-image path) — set `CMUX_WS_LEASE_FILE` to a
backend-written lease file and the entrypoint runs
`cmuxd-remote serve --ws` on `:7777` instead.

| Env | Default | Purpose |
|---|---|---|
| `CMUX_SSH_PUBLIC_KEY` | — | Appended to `authorized_keys` at start |
| `CMUX_SSH_PORT` | `2222` | sshd port |
| `CMUX_WS_LEASE_FILE` | — | Enables WS PTY transport with this lease file |
| `CMUX_WS_PORT` | `7777` | WS PTY listen port |

Verified at the current pin: `cmuxd-remote version` inside the container prints
the pinned version, and the full toolchain resolves
(`go`, `java`, `python3.12`, `bun`, `uv`, `kubectl`, `gh`).
`SetEnv PATH` is set in sshd_config so toolchains resolve in SSH sessions.

## Building

Multi-arch build + push (requires a buildx builder whose BuildKit container
trusts the HTTP registry — see note below):

```sh
docker buildx build --builder multi-arch --platform linux/amd64,linux/arm64 \
  -t 10.10.111.134:5000/ade-sandbox/orca:latest --push -f Dockerfile.orca .

docker buildx build --builder multi-arch --platform linux/amd64,linux/arm64 \
  -t 10.10.111.134:5000/ade-sandbox/cmux:latest --push -f Dockerfile.cmux .
```

Local arm64 build (runs natively on Apple Silicon):

```sh
docker buildx build --builder multi-arch --platform linux/arm64 \
  -t ade-sandbox-orca:arm64 --load -f Dockerfile.orca .
docker buildx build --builder multi-arch --platform linux/arm64 \
  -t ade-sandbox-cmux:arm64 --load -f Dockerfile.cmux .
```

Notes:

- `10.10.111.134:5000` is plain HTTP. The Docker daemon must list it under
  `insecure-registries`, and `docker-container` buildx builders need their own
  buildkitd config:
  `[registry."10.10.111.134:5000"] http = true, insecure = true`.
- Version bumps: update the `ARG *_VERSION` + `*_SHA*` pairs. Checksum sources:
  go.dev/dl JSON, Adoptium API, `dl.k8s.io/release/<ver>/bin/linux/<arch>/kubectl.sha256`,
  Orca `latest-linux*.yml` release manifests, cmux `cmuxd-remote-checksums.txt`.
- k8s: the amd64 manifest entry is what your cluster pulls; no `nodeSelector`
  tricks needed on amd64 nodes.
