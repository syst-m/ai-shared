# pv-sentinel

Detects broken volumes on a Kubernetes node and remediates them. Two detectors run
in one cycle: a volume mounted but returning `EROFS`, and a PVC whose expansion was
requested but never landed on the device.

It exists because **mount options lie**. On the cluster this was built for, six
volumes reported `rw,relatime,stripe=2048` in `/proc/mounts` for days while every
write returned `EROFS` — an aborted ext4 journal makes the kernel remount the
filesystem read-only, but the mount table keeps advertising `rw` for as long as
the stale kubelet mount survives. Anything that answers "is this volume healthy?"
by parsing mount flags gets it wrong. The only reliable answer is to create a
file and see whether the kernel lets you.

## Why a DaemonSet

The second finding is that the API cannot tell you where a volume is mounted. The
cluster's 15 PVs have no `.spec.nodeAffinity` (all read as unbound) and the CSI
1.30 `VolumeAttachment.spec.attachedTo` field is empty. The working mapping is
the classic `VolumeAttachment.spec.nodeName` + `.spec.source.persistentVolumeName`,
which matches the node-local mount table 1:1 — but a control-plane-only
Deployment can enumerate those attachments without being able to probe a single
one. Node-local access is mandatory, so this runs one pod per node.

## Why not fsck

The fix is not `e2fsck`. Running `e2fsck -fy` against a healthy mounted volume is
the destructive move — it can lose data on a filesystem that was never broken. The
actual fix is to delete the consuming pod: CSI detaches the device, kubelet mounts
it fresh, ext4 replays the journal at mount time, and the volume comes back
genuinely read-write. That is what this service automates.

This replaces `pv-doctor`, an ad-hoc sleep-loop pod with `e2fsprogs` installed at
runtime during the v0.8.15 recovery — a shell we happened to be inside rather than
a service. pv-doctor mounted all of `/` and ran `privileged: true`; this mounts
`/var/lib/kubelet` with `mountPropagation: HostToContainer` and runs as uid 0
with no added capabilities, which is the narrowest grant that still works.

## Stuck volume expansions

The second detector answers a different question. A PVC expansion is *requested* in
`.spec.resources.requests.storage` and *reported* in `.status.capacity.storage`;
between the two sits the controller resize, the node resize, and — on a block
device — a `resize2fs` against a filesystem the kernel must first learn is bigger.
If the two numbers differ for longer than `RESIZE_THRESHOLD` (default `10m`) the
expansion is **stuck**, not merely in flight. The threshold is the whole design: a
resize that is genuinely progressing resolves in seconds to minutes, so reporting
every in-flight resize as a problem is noise, and a gap that has survived ten
minutes is a real one.

The case this models is `freeidp-pg-2` on node `um760-1`, expanded 5Gi→10Gi and
parked there: `spec.resources.requests: 10Gi` against `status.capacity: 5Gi`, with
`Resizing=True` and `Unused=False/PodUsingPVC` on the claim, a VolumeAttachment
(`csi-6819fb6c…`) pointing at `um760-1`, and a pod stuck `Pending` with
`PodReadyToStartContainers: False — "no pod sandbox exists"`. The pod event names
the cause:

```
MountVolume.MountDevice failed for volume "pvc-4e4658fb-…" :
  rpc error: code = Internal
  stdout: "Filesystem at /dev/sdd is mounted on …/globalmount; on-line resizing required
           old_desc_blocks = 1, new_desc_blocks = 2"
  stderr: "resize2fs 1.47.0 (5-Feb-2023)
           resize2fs: Permission denied to resize filesystem"
```

`old_desc_blocks = 1, new_desc_blocks = 2` is exactly the 5Gi→10Gi doubling, so the
CSI plugin knew the new size; `Permission denied` (EPERM) is the kernel refusing to
grow a filesystem on a block device it still believes is 5Gi, because the iSCSI
session was never rescanned after the array grew the LUN. **The spec-vs-status gap
is the symptom; the stale device size is the cause** — so `Classify` returns both,
a `State` (`stuck`) and a `Cause` (`stale_device_size`), and remediation keys off
the cause rather than the state. The other causes it can name are
`controller_resize_pending`, `node_resize_pending`, `not_started`, and `not_csi`.

It enumerates through the existing `VolumeAttachment`→node mapping rather than
inventing a second one, and that is not a convenience — it is the only way. In the
case above the volume is attached but *not mounted* (no sandbox exists), so
`/proc/mounts`, which is what the first detector reads, would not show it at all.
The attachment says the volume is on this node; only the PVC says what state the
expansion is in. Hence the RBAC addition: `persistentvolumeclaims` `get,list`.

The fix is `iscsiadm --mode session --rescan`, which makes the initiator re-read
the LUN size so the pending mount and resize can complete. It is batched: one
rescan per cycle covers every stuck iSCSI volume on the node, because the command
rescans all sessions on the host regardless of how many are sick. Non-CSI volumes
(`not_csi`) are skipped — a rescan cannot help an NFS mount — and that check runs
*before* the mode check, so a dry run does not claim it would have acted on a volume
it would in fact refuse.

## Layout

| Path | Responsibility |
|---|---|
| `cmd/sentinel` | Config from env, HTTP server (`/healthz`, `/metrics`), scan loop, signal handling |
| `internal/probe` | Parse `/proc/mounts`, discover CSI volumes, dedupe the double view, write-probe each one |
| `internal/resize` | Pure classification of PVC spec-vs-status into `current`/`growing`/`stuck`/`shrunk`/`unknown` plus a cause, with the threshold timer |
| `internal/remediate` | Map a mount path's pod UID to the owning pod, apply the gates, delete, wait, re-probe; and the iSCSI rescan path for resize findings |
| `internal/k8s` | `kubernetes.Interface` wrapper scoped to one node |
| `internal/metrics` | Prometheus collectors |

Detection detail worth knowing: the CSI node container sees every mount twice —
once at `/var/lib/kubelet/…` and again at `/host/var/lib/kubelet/…` — so a naive
line count reports 22 mounts for 11 real volumes. `NormalizePath` strips the
`/host` prefix and volumes are deduped by PV name.

## Modes

`MODE=report` (default) runs detection and exports everything but changes nothing.
`MODE=remediate` enables the fixes. Shipping in report mode is what makes the first
deploy non-scary without making report-only a permanent limitation.

`RUN_ONCE=true` scans once, sleeps `RUN_ONCE_GRACE`, and exits — useful for
validating a build against the current cluster before it ever runs as a daemon.

### One MODE, or one per detector

Adding the resize detector forced a decision that did not exist when MODE governed
exactly one action. Three options:

| | | |
|---|---|---|
| **(a)** | Keep `MODE` global, both detectors obey it | Simplest, and wrong on the merits |
| **(b)** | Replace it with `MODE_RO_MOUNT` / `MODE_RESIZE` | Clean, but breaks the existing contract |
| **(c)** | Keep `MODE` as the default, add per-detector overrides | **Chosen** |

**(c)** — `MODE` stays and is the default for every detector; `MODE_RO_MOUNT` and
`MODE_RESIZE` override it per detector, and an unset override means "inherit MODE".

The reason (a) is not enough is that the two fixes are not the same kind of action.
Deleting a pod is disruptive and, for a bare pod with no controller, *permanent* —
the whole reason `ALLOW_BARE_PODS` exists and defaults to false is the
`freeidp-pg-2` incident where a deleted pod had nothing to recreate it. An iSCSI
session rescan is a read-only initiator query: it re-reads LUN sizes and touches no
filesystem, no pod, no mount. Forcing an operator who wants rescans automated to also
automate pod deletion — or one who wants pod deletion held back to also hold back
rescans — is a false coupling. (b) is the cleaner API but breaks the contract the
manifest and README already ship with, and every existing deployment's env. (c) is
backward compatible *and* separable: today's manifest behaves identically, and
`MODE=report` with `MODE_RESIZE=remediate` gives a week of rescans-only observation
before the global flip.

Argue from the next flip, not from today: the cluster runs `MODE=report`, so nothing
is auto-remediating and every choice here is currently zero-risk. The choice that
matters is what happens the day someone flips MODE to `remediate` — with (a) that
single keystroke silently enables a second class of action they may not have
reviewed; with (c) the two flips are independent, and each can be reviewed on its
own merits.

The manifest sets both overrides explicitly, to the same value `MODE` already has.
That redundancy is deliberate: it makes the split visible in the deployed YAML
instead of implicit in the code, so reading the DaemonSet tells you what each
detector would do.

## Configuration

All config is environment variables; there is no flag package.

| Variable | Default | Purpose |
|---|---|---|
| `PORT` | `8080` | HTTP port for `/healthz` and `/metrics` |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error`; unrecognised falls back to info |
| `NODE_NAME` | — | Downward API `spec.nodeName`; highest precedence for the node name |
| `NODE` | — | Fallback node name if `NODE_NAME` is unset |
| `MOUNTS_PATH` | `/proc/mounts` | Mount table to parse |
| `FS_TYPES` | `ext4,xfs` | Filesystem types worth probing; anything else is noise |
| `RUN_ONCE` | `false` | Scan once and exit |
| `RUN_ONCE_GRACE` | `10s` | How long to stay up after a run-once scan, so `/metrics` can be scraped |
| `INTERVAL` | `300s` | Seconds between scans |
| `MODE` | `report` | `report` or `remediate`; the default for every detector |
| `MODE_RO_MOUNT` | — | Override `MODE` for the read-only-mount detector only; empty inherits it |
| `MODE_RESIZE` | — | Override `MODE` for the resize detector only; empty inherits it |
| `ISCSIADM_PATH` | `/usr/sbin/iscsiadm` | Binary the resize remediation execs; a path, not a `$PATH` lookup |
| `RESIZE_THRESHOLD` | `10m` | How long a spec-vs-status gap may stay open before the expansion counts as stuck |
| `ALLOW_BARE_PODS` | `false` | Permit deleting pods with no `ownerRefs` |
| `COOLDOWN` | `60m` | Minimum interval between remediations of the same PV; shared by both detectors |
| `MAX_PER_NODE` | `3` | Max remediations per node per `NODE_WINDOW` |
| `NODE_WINDOW` | `60m` | Window the per-node cap is measured over |
| `GRACE_PERIOD` | `30` | `GracePeriodSeconds` passed to pod deletion |
| `POLL_TIMEOUT` | `180s` | How long to wait for the recreated pod to be Ready |
| `POLL_INTERVAL` | `5s` | Poll interval while waiting |

`ISCSIADM_PATH` is a full path rather than a name resolved through `$PATH` so that
*which* iscsiadm runs is decided by the manifest, not by whatever an image layer
happened to leave behind — and so a missing binary is a config-shaped, testable
answer (`ErrNoISCSIADM`) instead of an `exec` accident. `COOLDOWN` is shared by both
detectors rather than split: a per-PVC cooldown is the same question in both cases
("have we already acted on this volume recently?"), and a second knob that asks it
twice is a second thing to misconfigure.

### Gate rules

Order matters, and it is deliberate:

1. A **healthy** volume needs nothing — short-circuits before any API call.
2. **Not in remediate mode** → `dry_run`. A dry run does not start a cooldown, so
   months of report cycles cannot silently gate the first real remediation.
3. **Cooldown** — a volume that keeps breaking does not get its pod deleted every
   cycle.
4. **No resolvable owner pod** → `skipped_no_pod`.
5. **Bare pod** (no `ownerRefs`) → `skipped_bare_pod` unless `ALLOW_BARE_PODS`.
   This check precedes the rate limit on purpose: a pod we would refuse to delete
   must not consume the node's deletion budget. Refusing bare pods is what stops
   the `freeidp-pg-2` failure mode, where a deleted pod had no controller to
   recreate it and never came back.
6. **Node budget exhausted** → `rate_limited`, so a node-wide event cannot cascade
   into deleting every pod on the node at once.
7. Otherwise → `pod_deleted`.

### Gate rules (resize)

Same shape, different order, because the questions differ:

1. **Not stuck** → `none`. A growing or current expansion needs nothing.
2. **Cause is not `stale_device_size`** → `skipped_not_iscsi`. This precedes the
   mode check on purpose. A dry run's whole job is to say what it *would* do, and
   here the honest answer for an NFS volume is "nothing, in any mode" — reporting
   `dry_run` would claim an action we would never take.
3. **Not in remediate mode** (for this detector) → `dry_run`, and no cooldown is
   started, exactly as above.
4. **Cooldown** — `COOLDOWN` since the last rescan attempt for this PVC.
5. Otherwise → `iscsi_rescanned`.

Attempts are recorded *before* the exec, so a rescan that fails still gates the next
one; retrying a failing `iscsiadm` every 300s is a way to make a bad situation
louder without fixing it. Findings in one cycle collapse to **one** exec — the
command rescans every session on the host — and each pending result is rewritten
with that one outcome.

## Metrics

`pv_sentinel_volume_status{node,pv,fs,status}` is the one that surfaces the bug
this tool exists for: a volume with `status="read_only"` while `/proc/mounts`
still says `rw` is exactly the `OptionsClaimedRW` + EROFS combination.

```
pv_sentinel_volume_status{node,pv,fs,status}
pv_sentinel_probes_total{status}
pv_sentinel_remediations_total{action}
pv_sentinel_healthy_volumes
pv_sentinel_read_only_volumes
pv_sentinel_unknown_volumes
pv_sentinel_attached_not_mounted_volumes
pv_sentinel_volumes_discovered
pv_sentinel_last_scan_timestamp_seconds

pv_sentinel_resize_status{node,pv,pvc}
pv_sentinel_stuck_expansions
pv_sentinel_resize_last_scan_timestamp_seconds
pv_sentinel_resize_remediations_total{action}
```

Per-volume series are aged out with `DeletePartialMatch` on the `node` label, so a
volume that gets detached stops being reported instead of showing as broken forever.
The same aging applies to `resize_status` per `(node,pvc)`: when an expansion
completes, its series disappears rather than lingering as a stuck claim.

`resize_status` is a **separate family**, not another value of `volume_status`,
because the two answer different questions about different objects — one per
*volume on this node*, one per *claim being resized* — and joining a PVC's resize
into a PV-keyed gauge would mean emitting rows for volumes the mount detector
never saw (the freeidp case: attached, never mounted). Its value is always `1`;
the series existing *is* the alert. There is deliberately no `state` label on it:
only stuck claims get a series, so `state` could only ever hold one value —
cardinality for nothing. The breakdown across `current`/`growing`/`shrunk` lives in
the log line and in `stuck_expansions`, not in labels.

## Deploy

```
kubectl apply -f k8s-manifests/rbac.yaml
kubectl apply -f k8s-manifests/daemonset.yaml
kubectl apply -f k8s-manifests/service.yaml
```

The image is built for `linux/amd64`; the build host is arm64, so a plain
`docker build` produces an image that schedules and then fails at runtime.

### The image and `iscsiadm`

The resize fix execs a host binary, so the image has to contain it. The final stage
is `alpine:3.20` — it has a shell (busybox `/bin/sh`), so this was never a distroless
problem — but it installed `ca-certificates` and nothing else, so `iscsiadm` was
absent and `MODE_RESIZE=remediate` could only ever have reported `failed`. Alpine's
`open-iscsi` package (2.1.9-r3, GPL-2.0-only) exists in v3.20 `main` **for x86_64
only**, while CI builds both amd64 and arm64. The base image is therefore unchanged
and the package is added behind `ARG TARGETARCH`:

```dockerfile
ARG TARGETARCH
RUN if [ "$TARGETARCH" = "amd64" ]; then \
        apk --no-cache add ca-certificates open-iscsi; \
    else \
        apk --no-cache add ca-certificates; \
    fi && ...
```

The cluster is amd64, so every node that needs the fix gets the binary; on arm64 it
is simply absent and the remediation returns `ErrNoISCSIADM` rather than pretending
it ran. The alternative — swapping to a fat base image that ships iSCSI tooling —
would trade a 3MB layer for a ~80MB one to solve a problem that only exists on one
architecture. **The image has not been rebuilt or pushed**; the manifest keeps
`MODE_RESIZE: "report"` and the flip belongs in the same change that ships the new
image.

## Tests

```
go test -race ./...
~/go/bin/golangci-lint run
```

The k8s client is tested against `k8s.io/client-go/kubernetes/fake`. Two fake
limitations shaped the production code rather than the tests: the fake ignores
field selectors, so `PodsOnNode` filters in Go via a pure `filterPodsOnNode`; and
it deletes objects without ever resurrecting them under a new UID, so `Remediator`
has a `SetWaitFunc` seam to stand in for controller recreation.

`internal/resize` is pure — `Classify(candidate, firstSeen, now, threshold)` takes
the clock as an argument rather than calling it, so the threshold boundary is a
table of timestamps, not a `time.Sleep`. `Rescanner` has the same seam shape as
`Remediator`: `SetExecFunc` stands in for `exec.CommandContext`, which lets the
tests assert the exact argv (`--mode session --rescan`) and drive the
`ErrNoISCSIADM` and non-zero-exit paths without a binary present. The real
`defaultExec` is covered too, by execing `/bin/true` and by pointing at a path that
does not exist.
