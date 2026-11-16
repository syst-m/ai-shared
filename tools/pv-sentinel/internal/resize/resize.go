// Package resize is pv-sentinel's second detector: it finds PersistentVolumeClaims
// whose volume expansion is stuck rather than merely in flight.
//
// The case it exists for is real, not hypothetical. On node um760-1 the
// freeidp namespace's freeidp-pg-2 PVC was expanded 5Gi -> 10Gi and stayed that
// way: the PVC advertised spec.resources.requests 10Gi against status.capacity
// 5Gi, its conditions said Resizing=True and Unused=False/PodUsingPVC, the
// VolumeAttachment csi-6819fb6c... still named um760-1, and the pod was Pending
// with no sandbox because the mount never completed. The pod's own event named
// the cause:
//
//	MountVolume.MountDevice failed for volume "pvc-4e4658fb-4d16-4f14-a2b7-f87913b92894":
//	  rpc error: code = Internal
//	  stdout: "Filesystem at /dev/sdd is mounted on .../globalmount; on-line resizing required
//	           old_desc_blocks = 1, new_desc_blocks = 2"
//	  stderr: "resize2fs 1.47.0 (5-Feb-2023)
//	           resize2fs: Permission denied to resize filesystem"
//
// old_desc_blocks = 1, new_desc_blocks = 2 is exactly the 5Gi -> 10Gi doubling.
// resize2fs returns EPERM because the kernel's view of /dev/sdd is still 5Gi
// while the array presents 10Gi: the iSCSI session on the initiator was never
// rescanned after the array-side growth. So the spec-vs-status gap is the
// symptom and a stale device size is the cause, and this package reports both —
// the state from the two numbers, the cause from the conditions and the
// volume's transport.
//
// Why a threshold at all: a resize that is genuinely progressing resolves in
// seconds to minutes. Reporting every in-flight resize as a problem is noise,
// so a gap only becomes "stuck" once it has been continuously observed for
// RESIZE_THRESHOLD. That is also why the detector keeps state (when each gap was
// first seen) rather than being a pure function of one API snapshot.
package resize

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"

	"pv-sentinel/internal/k8s"
)

// States of a PVC's requested-vs-reported size.
const (
	// StateCurrent means status.capacity equals spec.resources.requests: the
	// volume is the size it was asked to be.
	StateCurrent = "current"
	// StateGrowing means the volume is smaller than requested and the gap has
	// not yet lasted long enough to call it stuck.
	StateGrowing = "growing"
	// StateStuck means the volume has been smaller than requested for at least
	// the threshold. This is the only state remediation acts on.
	StateStuck = "stuck"
	// StateShrunk means status.capacity is larger than the request. Kubernetes
	// does not support shrinking a PVC, so this is a report-only curiosity:
	// usually a leftover from a resized-then-recreated claim or a provider that
	// rounds up. Acting on it would be wrong, so it never produces a cause.
	StateShrunk = "shrunk"
	// StateUnknown means the request or the capacity is not readable, so no
	// comparison is possible.
	StateUnknown = "unknown"
)

// Causes of a stuck expansion. A cause is only ever set on StateStuck: without
// the threshold being crossed there is nothing to explain yet.
const (
	// CauseStaleDeviceSize means the block device on this node is smaller than
	// the volume the array presents, so the node-side mount/resize cannot
	// proceed. This is the one cause the local remediation can fix: rescanning
	// the iSCSI session makes the kernel re-read the LUN size.
	CauseStaleDeviceSize = "stale_device_size"
	// CauseControllerPending means the CSI controller acknowledged the resize
	// but has not completed it (ControllerResizePending=True). The fix is on the
	// storage side, not this node, so a rescan here would not help.
	CauseControllerPending = "controller_resize_pending"
	// CauseNodePending means the controller grew the volume and the node has
	// not grown the filesystem, on a transport this node cannot rescan (local,
	// or a CSI driver that is not iSCSI).
	CauseNodePending = "node_resize_pending"
	// CauseNotStarted means the resize is neither controller- nor node-pending:
	// nothing has picked the request up at all.
	CauseNotStarted = "not_started"
	// CauseNotCSI means the volume is not a CSI volume, so the iSCSI rescan
	// remediation does not apply to it whatever the conditions say.
	CauseNotCSI = "not_csi"
)

// Transports a CSI volume is served over, as far as the PV spec reveals. The
// remediation is transport-specific, so the detector has to name it.
const (
	ProtocolISCSI   = "iscsi"
	ProtocolNFS     = "nfs"
	ProtocolLocal   = "local"
	ProtocolUnknown = "unknown"
	// ProtocolNotCSI is for in-tree (non-CSI) volumes: hostPath, in-tree nfs,
	// in-tree local, and so on.
	ProtocolNotCSI = "not_csi"
)

// Condition is one PVC status condition, flattened. The API type carries more
// (LastTransitionTime, Reason) but the detector only ever reads these three, and
// flattening them keeps Candidate constructible in tests without building the
// full condition struct.
type Condition struct {
	Type    string
	Status  string
	Message string
}

// Candidate is one PVC on this node whose expansion is worth looking at, reduced
// to the fields the classification needs.
type Candidate struct {
	PVName    string
	Namespace string
	PVCName   string
	Protocol  string
	// SpecBytes is spec.resources.requests.storage in bytes; SpecSet says
	// whether that request exists at all.
	SpecBytes int64
	SpecSet   bool
	// StatusBytes is status.capacity.storage in bytes; StatusSet says whether
	// the controller has reported a capacity yet.
	StatusBytes int64
	StatusSet   bool
	Conditions  []Condition
}

// Verdict is the classification of a Candidate: what state the expansion is in
// and, when it is stuck, why.
type Verdict struct {
	State string
	Cause string
}

// Key identifies the candidate across scans. namespace/pvc is the stable
// identity: the PV name changes when a volume is recreated, and bookkeeping
// keyed on the PV would treat a recreated volume as brand new.
func (c Candidate) Key() string {
	if c.Namespace == "" && c.PVCName == "" {
		return c.PVName
	}
	return c.Namespace + "/" + c.PVCName
}

// Protocol names the transport behind a PersistentVolume.
//
// The CSI driver name is the authority, but driver names are not a registry —
// there is no canonical list to switch on — so this matches the well-known
// substrings and then falls back to looking for iSCSI markers in the volume's
// attributes. The fallback matters because a CSI driver that wraps iSCSI does
// not always say so in its name, while its attributes carry the target iqn and
// portals, which is exactly what the rescan operates on.
func Protocol(pv corev1.PersistentVolume) string {
	src := pv.Spec.PersistentVolumeSource
	if src.CSI == nil {
		return ProtocolNotCSI
	}
	driver := strings.ToLower(src.CSI.Driver)
	switch {
	case strings.Contains(driver, "iscsi"):
		return ProtocolISCSI
	case strings.Contains(driver, "nfs"):
		return ProtocolNFS
	case strings.Contains(driver, "local"):
		return ProtocolLocal
	}
	// The driver name said nothing useful: look for iSCSI evidence in the volume
	// attributes before giving up, since that is where the target iqn and portals
	// live and they are what decides whether a rescan can help.
	//
	// The field is VolumeAttributes here; "volume context" is the CSI RPC's name
	// for the same map, and the k8s type has no field by that name.
	blob := strings.ToLower(fmt.Sprintf("%v", src.CSI.VolumeAttributes))
	if strings.Contains(blob, "iqn") || strings.Contains(blob, "portal") || strings.Contains(blob, "iscsi") {
		return ProtocolISCSI
	}
	return ProtocolUnknown
}

// condition returns the named condition's status and message, or ("", "") when
// the candidate does not carry it.
func (c Candidate) condition(condType string) (string, string) {
	for _, cond := range c.Conditions {
		if cond.Type == condType {
			return cond.Status, cond.Message
		}
	}
	return "", ""
}

// causeFor explains a stuck expansion from the PVC's conditions and the volume's
// transport. It is only called for a gap that has lasted past the threshold.
//
// The ordering is deliberate. Resizing=True/FileSystemResizePending=True on an
// iSCSI volume is the freeidp signature — the controller grew the volume, the
// node cannot complete the mount, and the fix is a session rescan — so it is
// checked before the generic node-pending branch. ControllerResizePending comes
// next because a resize the controller has not acted on cannot be fixed on this
// node whatever the transport is.
func causeFor(c Candidate) string {
	if c.Protocol == ProtocolNotCSI {
		return CauseNotCSI
	}
	resizing, _ := c.condition("Resizing")
	fsPending, _ := c.condition("FileSystemResizePending")
	controllerPending, _ := c.condition("ControllerResizePending")

	switch {
	case strings.EqualFold(fsPending, "true"), strings.EqualFold(resizing, "true"):
		if c.Protocol == ProtocolISCSI {
			return CauseStaleDeviceSize
		}
		return CauseNodePending
	case strings.EqualFold(controllerPending, "true"):
		return CauseControllerPending
	}
	return CauseNotStarted
}

// Classify decides a Candidate's state. firstSeen is when this same gap was
// first observed and now is the current time; both are passed in rather than
// read from a clock so the threshold arithmetic is testable.
//
// A spec smaller than status is not a resize in progress — Kubernetes has no
// shrink path — so it is reported and never remediated.
func Classify(c Candidate, firstSeen, now time.Time, threshold time.Duration) Verdict {
	if !c.SpecSet || !c.StatusSet {
		return Verdict{State: StateUnknown}
	}
	switch {
	case c.SpecBytes == c.StatusBytes:
		return Verdict{State: StateCurrent}
	case c.SpecBytes < c.StatusBytes:
		return Verdict{State: StateShrunk}
	}
	// spec > status: the volume has not reached its requested size.
	age := now.Sub(firstSeen)
	if age < 0 {
		age = 0
	}
	if age < threshold {
		return Verdict{State: StateGrowing}
	}
	return Verdict{State: StateStuck, Cause: causeFor(c)}
}

// Finding is one stuck expansion, with the age that made it stuck. The
// remediation consumes this; the metric exports it.
//
// Only stuck candidates become Findings. "growing" is a normal transient and
// logging one per scan for every resize in the cluster is the noise the
// threshold exists to suppress, so it is counted but not itemised.
type Finding struct {
	PVName      string
	Namespace   string
	PVCName     string
	Protocol    string
	SpecBytes   int64
	StatusBytes int64
	State       string
	Cause       string
	Since       time.Time
	Age         time.Duration
}

// Key identifies the finding; see Candidate.Key.
func (f Finding) Key() string {
	if f.Namespace == "" && f.PVCName == "" {
		return f.PVName
	}
	return f.Namespace + "/" + f.PVCName
}

// Report is one detector pass over this node's volumes.
type Report struct {
	Node     string
	At       time.Time
	Findings []Finding
	// Counts of every classified state, so the metric and the log line agree
	// with each other and with the set of findings.
	Current, Growing, Stuck, Shrunk, Unknown int
}

// Summary renders a Report as one log-safe line. Nothing here is secret, but the
// format stays consistent with the mount detector's summary so one grep pattern
// covers both detectors.
func Summary(r Report) string {
	return fmt.Sprintf("current=%d growing=%d stuck=%d shrunk=%d unknown=%d",
		r.Current, r.Growing, r.Stuck, r.Shrunk, r.Unknown)
}

// BuildCandidates joins the node's VolumeAttachments to their PVs and their
// claims. Anything that does not resolve all the way to a PVC is skipped rather
// than guessed at: an attachment whose PV is gone, or a PV with no claimRef, has
// no spec to compare against and inventing one would produce a false finding.
//
// Duplicates are collapsed — a PVC attached twice (a multi-attach volume, or two
// attachments in the same list) must not be counted twice.
func BuildCandidates(
	vas []storagev1.VolumeAttachment,
	pvs []corev1.PersistentVolume,
	pvcs []corev1.PersistentVolumeClaim,
) []Candidate {
	byPV := make(map[string]corev1.PersistentVolume, len(pvs))
	for _, pv := range pvs {
		byPV[pv.Name] = pv
	}
	byPVC := make(map[string]corev1.PersistentVolumeClaim, len(pvcs))
	for _, pvc := range pvcs {
		byPVC[pvc.Namespace+"/"+pvc.Name] = pvc
	}

	seen := make(map[string]bool, len(vas))
	out := make([]Candidate, 0, len(vas))
	for _, va := range vas {
		// PersistentVolumeName is a *string: the source is either a PV name or
		// (for inline ephemeral volumes) nothing at all.
		if va.Spec.Source.PersistentVolumeName == nil {
			continue
		}
		pvName := *va.Spec.Source.PersistentVolumeName
		if pvName == "" {
			continue
		}
		pv, ok := byPV[pvName]
		if !ok {
			continue
		}
		ref := pv.Spec.ClaimRef
		if ref == nil {
			continue
		}
		pvc, ok := byPVC[ref.Namespace+"/"+ref.Name]
		if !ok {
			continue
		}
		c := Candidate{
			PVName:    pvName,
			Namespace: pvc.Namespace,
			PVCName:   pvc.Name,
			Protocol:  Protocol(pv),
		}
		if q, ok := pvc.Spec.Resources.Requests[corev1.ResourceStorage]; ok {
			c.SpecBytes, c.SpecSet = q.Value(), true
		}
		if q, ok := pvc.Status.Capacity[corev1.ResourceStorage]; ok {
			c.StatusBytes, c.StatusSet = q.Value(), true
		}
		for _, cond := range pvc.Status.Conditions {
			c.Conditions = append(c.Conditions, Condition{
				Type:    string(cond.Type),
				Status:  string(cond.Status),
				Message: cond.Message,
			})
		}
		if seen[c.Key()] {
			continue
		}
		seen[c.Key()] = true
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key() < out[j].Key() })
	return out
}

// ClassifyAll classifies every candidate against the first-seen bookkeeping in
// first and returns the aggregate Report. first is updated in place: a gap that
// is new this round is stamped now, and a claim that is no longer a gap is
// removed so it cannot carry a stale firstSeen into its next life.
//
// The two passes are the point. Registering gaps before classifying means a
// first observation of a gap is measured from now (so it is "growing"), not from
// the zero time (which would make every newly-seen volume instantly stuck).
func ClassifyAll(candidates []Candidate, first map[string]time.Time, now time.Time, threshold time.Duration) Report {
	gaps := make(map[string]bool, len(candidates))
	for _, c := range candidates {
		if !c.SpecSet || !c.StatusSet || c.SpecBytes <= c.StatusBytes {
			continue
		}
		key := c.Key()
		if _, ok := first[key]; !ok {
			first[key] = now
		}
		gaps[key] = true
	}

	r := Report{At: now}
	for _, c := range candidates {
		v := Classify(c, first[c.Key()], now, threshold)
		switch v.State {
		case StateCurrent:
			r.Current++
		case StateGrowing:
			r.Growing++
		case StateStuck:
			r.Stuck++
			r.Findings = append(r.Findings, Finding{
				PVName:      c.PVName,
				Namespace:   c.Namespace,
				PVCName:     c.PVCName,
				Protocol:    c.Protocol,
				SpecBytes:   c.SpecBytes,
				StatusBytes: c.StatusBytes,
				State:       v.State,
				Cause:       v.Cause,
				Since:       first[c.Key()],
				Age:         now.Sub(first[c.Key()]),
			})
		case StateShrunk:
			r.Shrunk++
		case StateUnknown:
			r.Unknown++
		}
	}
	for k := range first {
		if !gaps[k] {
			delete(first, k)
		}
	}
	return r
}

// Detector watches the PVCs whose volumes are attached to one node and decides
// whether each requested expansion is progressing or stuck.
//
// It holds the first-seen time per claim, which is the whole reason it is a
// struct and not a function: "stuck" is a statement about duration, and a
// duration needs a memory. The map is pruned from the current scan on every
// Detect so a claim that leaves the node stops accumulating age — otherwise a
// deleted PVC would keep a stale firstSeen forever and its replacement would be
// born already-stuck.
type Detector struct {
	cl        *k8s.Client
	threshold time.Duration

	mu    sync.Mutex
	first map[string]time.Time
}

// New returns a Detector over cl's node. A threshold <= 0 makes every gap
// instantly stuck, which is useful in tests and a poor setting in production;
// main.go defaults it to 10m.
func New(cl *k8s.Client, threshold time.Duration) *Detector {
	return &Detector{cl: cl, threshold: threshold, first: map[string]time.Time{}}
}

// Detect lists this node's volumes and classifies each one's expansion.
//
// It reuses k8s.Client.VolumeAttachmentsOnNode — the same VolumeAttachment→node
// mapping the mount detector uses — because that is the only enumeration that
// sees the freeidp case: the volume is attached but not mounted, so it is absent
// from /proc/mounts and invisible to the probe. The VolumeAttachment is the one
// place that says "this node owns pvc-4e4658fb…".
func (d *Detector) Detect(ctx context.Context, now time.Time) (Report, error) {
	attachments, err := d.cl.VolumeAttachmentsOnNode(ctx)
	if err != nil {
		return Report{}, fmt.Errorf("resize detection needs volume attachments: %w", err)
	}
	pvs, err := d.cl.PersistentVolumes(ctx)
	if err != nil {
		return Report{}, fmt.Errorf("resize detection needs persistent volumes: %w", err)
	}
	pvcs, err := d.cl.PersistentVolumeClaims(ctx)
	if err != nil {
		return Report{}, fmt.Errorf("resize detection needs persistent volume claims: %w", err)
	}

	candidates := BuildCandidates(attachments, pvs, pvcs)

	d.mu.Lock()
	defer d.mu.Unlock()
	r := ClassifyAll(candidates, d.first, now, d.threshold)
	r.Node = d.cl.Node()
	return r, nil
}

// FirstSeen reports when a claim's current gap was first observed. It exists for
// tests and for the metric's "since" value; an absent claim reports false.
func (d *Detector) FirstSeen(namespace, name string) (time.Time, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	t, ok := d.first[namespace+"/"+name]
	return t, ok
}
