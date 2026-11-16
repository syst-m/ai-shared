# Examples

Deployment variants of the DaemonSet, for reading side by side rather than
choosing between. The canonical manifest lives one directory up at
[`../k8s-manifests/daemonset.yaml`](../k8s-manifests/daemonset.yaml) and is what
runs in the lab; this directory holds the postures that manifest is *not* in.

| File | Posture | What it changes |
|------|---------|-----------------|
| `daemonset-remediate.yaml` | `MODE_RESIZE=remediate`, `MODE_RO_MOUNT=report` | Turns on the iSCSI rescan while leaving pod deletion in report |

## Why the modes are split

`MODE` is a single default for every detector, and `MODE_RO_MOUNT` /
`MODE_RESIZE` override it per detector (an unset override inherits `MODE`). The
split exists because the two detectors' actions are not comparable in cost:

| Detector | Action | Cost of being wrong |
|----------|--------|---------------------|
| `ro-mount` | delete the pod consuming a read-only volume | Disruptive; for a bare pod with no `ownerRefs`, permanent — nothing recreates it |
| `resize` | `iscsiadm --mode session --rescan` | Near-zero: a read-only initiator query that makes the kernel re-read a LUN's size. Touches no filesystem, deletes nothing |

So the two flip independently, and the cheap one goes first. That ordering is the
posture `daemonset-remediate.yaml` shows.

## Before enabling `MODE_RESIZE=remediate`

The image has to actually contain `iscsiadm`. The `Dockerfile` installs `open-iscsi`
behind `ARG TARGETARCH`, because Alpine ships that package for x86_64 only while CI
builds arm64 too. On an image without the binary, `remediate` is not silently
skipped — it returns `ErrNoISCSIADM` and counts a `failed` remediation, which is the
correct answer but not a fix. Check the image, then flip the mode.
