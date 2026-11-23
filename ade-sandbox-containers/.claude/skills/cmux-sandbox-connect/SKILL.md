---
name: cmux-sandbox-connect
description: Connect the cmux dev sandbox container to the running cmux app as an extra workspace, and keep the connection working across pod rollouts. Use for "connect to the cmux dev container", "add the sandbox as a cmux workspace", "open the dev container in cmux", or after any `kubectl rollout restart statefulset/cmux-dev` / `ade-sandbox-deploy` run that replaced the cmux pod. Not for building or deploying the images (that's `ade-sandbox-deploy`).
---

# cmux sandbox connect

Attach the `cmux-dev` sandbox container as a workspace in the live cmux app.

The container's SSH endpoint is `cmux@10.10.111.139:2222` (from `cmux-dev-lb`).
cmux has a first-class command for this — do not hand-roll a plain `ssh`
terminal, and do not use the cmux-cua MCP tools (those drive the app's UI via
accessibility, which is the wrong tool when a CLI exists).

```sh
cmux ssh cmux@10.10.111.139 --port 2222 --name "cmux-dev (k8s)" --no-focus
```

The `cmux` binary is on PATH inside a cmux session; from outside it is
`/Applications/cmux.app/Contents/Resources/bin/cmux`. Add `--identity <path>`
if the key isn't the default `~/.ssh/id_ed25519`, and `-A` to forward the agent.

## The one thing that breaks: host key rotation

`~/.ssh` inside the pod is **ephemeral** — only `~/workspace` is a volume. Every
pod replacement regenerates the sshd host key, so the next SSH connection fails
with `REMOTE HOST IDENTIFICATION HAS CHANGED` and refuses to connect until the
stale entry is removed:

```sh
ssh-keygen -R '[10.10.111.139]:2222'
```

Note the quoting: the host must be `[ip]:port` and single-quoted, or `ssh-keygen`
does not match the entry.

**Order matters when a rollout is also pending.** Do the rollout first, then
clear the key once, then connect. Clearing the key before a rollout means
clearing it twice.

## Procedure

1. **Confirm the pod is the one you expect** (not a stale revision):
   ```sh
   kubectl -n ade-sandbox get pod cmux-dev-0 -o wide
   nc -z -G 5 10.10.111.139 2222 && echo "port open"
   ```

2. **Verify key auth non-interactively** before involving the app:
   ```sh
   ssh -p 2222 -o BatchMode=yes -o ConnectTimeout=8 \
     -o StrictHostKeyChecking=accept-new cmux@10.10.111.139 \
     'hostname; uname -m; claude --version'
   ```
   If this fails on host key verification, run the `ssh-keygen -R` above. If it
   fails on `Permission denied (publickey)`, the `cmux-ssh-key` Secret does not
   contain your public key — see `ade-sandbox-deploy`.

   **Do not use `kubectl exec` to check agent configuration.** `kubectl exec`
   and an SSH shell get their environments from different places, and only one
   of them is the one `cmux ssh` uses. sshd builds a session environment from a
   fixed whitelist plus `SetEnv` and does not pass the rest of its environment to
   sessions, so every var the pod spec injects has to appear in `SetEnv` to
   reach an SSH shell. A pod can therefore have all nine agent vars under
   `kubectl exec` and still give `claude` none of them over SSH — which is the
   exact failure that made claude prompt for configuration in a workspace that
   looked healthy. Check the path the user actually uses:
   ```sh
   ssh -p 2222 -o BatchMode=yes cmux@10.10.111.139 \
     'echo "AK=${#ANTHROPIC_API_KEY} AT=${#ANTHROPIC_AUTH_TOKEN} OK=${#OPENAI_API_KEY} GH=${#GH_TOKEN}"'
   ```
   Non-zero lengths mean the credentials arrived; `0` means sshd's `SetEnv` is
   not carrying them, regardless of what the pod spec says.

   Then the config and cluster access, in the same session:
   ```sh
   ssh -p 2222 -o BatchMode=yes cmux@10.10.111.139 '
     ls -l ~/.mcp.json ~/.claude/settings.json ~/.codex/config.toml
     gh auth status
     kubectl get pods -n ade-sandbox
   '
   ```
   All three files must exist. `~/.mcp.json` is `subPath`-mounted from the
   ConfigMap rather than copied by the init container — a bare `$HOME` path is
   not a volume, so a file the init container writes there dies with it. `gh
   auth status` is the only proof of gh login (`gh --version` succeeds with no
   token), and `kubectl get pods` works only because the pod runs as the
   `cmux-dev` ServiceAccount; on a pod without it, kubectl answers `Forbidden`.

3. **Attach it to cmux:**
   ```sh
   cmux ssh cmux@10.10.111.139 --port 2222 --name "cmux-dev (k8s)" --no-focus
   ```
   Prints `OK workspace:<ref> surface:<ref>`.

4. **Confirm cmux sees it:**
   ```sh
   cmux workspace list
   ```
   Expect the new workspace with `[ssh:connected]`.

## Managing the connection afterwards

```sh
cmux workspace list                                 # workspaces in the current window
cmux workspace close --workspace workspace:1000010003
cmux ssh-session-list --all-workspaces              # can be slow; prefer `workspace list`
cmux ssh-session-cleanup --all-workspaces --session-id <id>
cmux list-windows                                   # windows + their workspace counts
```

Refs must be passed in `workspace:<n>` form. A bare `1000010003` fails with
`Error: Workspace index not found` — the CLI does not accept the bare number
even though `workspace list` prints it that way.

After a pod rollout, an existing SSH workspace usually auto-reconnects. If it
sits paused (host briefly unreachable during the recreate), clear the host key
first, then close the dead workspace and re-run `cmux ssh` — there is no
`workspace reconnect` subcommand, and re-running `cmux ssh` is what rebuilds the
link. Reconnecting without clearing the key just re-runs into the same
verification failure.

## When `cmux ssh` fails with a zombie carrier

```
Error: ssh_failed: cmux-tui link did not report a socket within the connect timeout.
```

The workspace looks fine but won't accept input, and re-running `cmux ssh` fails
even though plain `ssh` to the same host works. Cause: a previous
`cmux-tui remote ssh <dest> --carrier` process survived its own workspace being
closed (despite `--exit-with-parent`) and still holds the state dir
(`~/.cmuxterm/cmux-tui-client`) and the SSH ControlPath socket, so a new link
can't claim them and times out.

Diagnose by counting the carrier's children — this is the reliable test:

```sh
pgrep -fl "cmux-tui remote ssh"
pgrep -P <carrier_pid>
```

A **healthy** carrier has an `ssh -T ... -p 2222` child (the live tunnel). A
**zombie** has none. `[ssh:connected]` in `workspace list` does *not* prove
liveness — a phantom workspace reports connected with no carrier behind it.

Fix — kill the zombie and clear its stale control socket:

```sh
kill <carrier_pid>                                  # SIGTERM is enough
rm -f ~/.cmux/ssh/<controlpath>                     # the socket file it leaves behind
```

Then re-run `cmux ssh`. Verify by confirming the new carrier *has* an `ssh` child.

Two follow-ups this leaves behind:

- **Failed `cmux ssh` calls still create the workspace.** Each errored attempt
  leaves a phantom `[ssh:connected]` workspace that must be closed individually,
  or the window accumulates duplicates of the same host.
- If several `cmux ssh` attempts are needed, close the phantoms before retrying
  rather than after — one good workspace plus zero stale ones is the target state.

## Related

- `ade-sandbox-deploy` — rebuild/push/rollout the images. Run it first if the
  sandbox is down or stale; this skill assumes a running `cmux-dev-0`.
- `k8s/README.md` — the ephemeral-`$HOME` tradeoff and the `ssh-keygen -R` note.
