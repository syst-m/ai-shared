// Package remediate decides what to do about a volume that failed its write
// probe, and does it.
//
// The fix is not fsck. The volumes this tool was written for had aborted ext4
// journals on otherwise clean filesystems — e2fsck -n reported them clean, and
// running e2fsck -fy on a healthy mounted filesystem is the destructive move.
// The defect was a stale kubelet mount holding the aborted journal. Deleting the
// consuming pod makes CSI detach the volume; the next attach mounts the device
// fresh, ext4 replays the journal at mount time, and the volume is genuinely
// writable again. So the remediation is: find the pod using the volume, and
// delete it.
package remediate

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"pv-sentinel/internal/k8s"
	"pv-sentinel/internal/probe"
)

// Mode controls whether the remediator acts or only reports.
const (
	ModeReport    = "report"
	ModeRemediate = "remediate"
)

// Action taken, or declined, for one volume.
const (
	ActionNone         = "none"             // volume healthy, nothing to do
	ActionDryRun       = "dry_run"          // would remediate, but mode is report
	ActionCooldown     = "cooldown"         // remediated too recently
	ActionSkippedNoPod = "skipped_no_pod"   // volume mounted but no pod owns it
	ActionSkippedBare  = "skipped_bare_pod" // pod has no ownerRefs to recreate it
	ActionRateLimited  = "rate_limited"     // per-node cap reached
	ActionDeleted      = "pod_deleted"      // pod deleted, re-probe result in After
	ActionFailed       = "failed"           // remediation attempted and errored
)

// DefaultMode is the mode a fresh deployment starts in: detect and report,
// delete nothing. Flipping to ModeRemediate is a deliberate, reviewable change.
const DefaultMode = ModeReport

// Detectors that can be gated independently. See Gate.ModeFor.
const (
	DetectorROMount = "ro_mount" // the write-probe detector, remediated by deleting the pod
	DetectorResize  = "resize"   // the expansion detector, remediated by an iSCSI rescan
)

// DefaultISCSIADMPath is where open-iscsi installs the client binary on Alpine
// and on every other distro that ships it in a package. It is a path, not a
// discovery: searching $PATH at runtime would let a stray binary in an image
// layer become part of the security surface.
const DefaultISCSIADMPath = "/usr/sbin/iscsiadm"

// ModeFor resolves the mode for one detector.
//
// MODE is the default for every detector; MODE_RO_MOUNT and MODE_RESIZE override
// it for their own detector when set. The reason for the split is that the two
// remediations have very different blast radii: deleting a pod is disruptive and,
// for a pod with no controller, permanent — which is why ALLOW_BARE_PODS exists at
// all — while `iscsiadm --mode session --rescan` is a read-only query to the
// initiator that touches no mounted filesystem. Under a single global switch an
// operator who wants to enable only the safe fix cannot ask for it: flipping
// MODE=remediate to get rescans silently turns on pod deletion too.
//
// Unset overrides mean "inherit MODE", which is exactly the pre-existing
// behaviour, so an existing manifest keeps doing what it did before.
func (g Gate) ModeFor(detector string) string {
	switch detector {
	case DetectorROMount:
		if g.RoMountMode != "" {
			return g.RoMountMode
		}
	case DetectorResize:
		if g.ResizeMode != "" {
			return g.ResizeMode
		}
	}
	return g.Mode
}

// Gate collects every knob that decides whether a fix is allowed to happen.
type Gate struct {
	// Mode is report (default) or remediate, for every detector that does not
	// override it.
	Mode string
	// RoMountMode overrides Mode for the read-only-mount detector (MODE_RO_MOUNT).
	// Empty means inherit Mode.
	RoMountMode string
	// ResizeMode overrides Mode for the resize detector (MODE_RESIZE). Empty
	// means inherit Mode.
	ResizeMode string
	// ISCSIADMPath is the iscsiadm binary the resize remediation execs. Empty
	// means DefaultISCSIADMPath.
	ISCSIADMPath string
	// AllowBarePods permits deleting pods with no ownerRefs. Off by default:
	// a bare pod has no controller to recreate it, so deleting it strands the
	// volume in a worse state than the one being fixed.
	AllowBarePods bool
	// Cooldown is how long to wait before remediating the same PV twice.
	Cooldown time.Duration
	// MaxPerNode caps deletions across the whole node within NodeWindow, so a
	// node-wide storage event cannot cascade into evicting every pod at once.
	MaxPerNode int
	// NodeWindow is the window MaxPerNode counts over.
	NodeWindow time.Duration
	// GracePeriod is the pod deletion grace period in seconds.
	GracePeriod int64
	// PollTimeout bounds how long to wait for a deleted pod to come back ready.
	PollTimeout time.Duration
	// PollInterval is how often to check while waiting.
	PollInterval time.Duration
}

// DefaultGate returns the gate a fresh deployment should run with.
func DefaultGate() Gate {
	return Gate{
		Mode:          DefaultMode,
		ISCSIADMPath:  DefaultISCSIADMPath,
		AllowBarePods: false,
		Cooldown:      60 * time.Minute,
		MaxPerNode:    3,
		NodeWindow:    60 * time.Minute,
		GracePeriod:   30,
		PollTimeout:   180 * time.Second,
		PollInterval:  5 * time.Second,
	}
}

// Validate rejects gate configurations that would misbehave rather than fail
// loudly at startup.
func (g Gate) Validate() error {
	if !validMode(g.Mode) {
		return fmt.Errorf("invalid MODE %q: must be %q or %q", g.Mode, ModeReport, ModeRemediate)
	}
	// An empty override means "inherit MODE" — which is what every existing
	// manifest asks for — so only a non-empty value can be wrong here.
	if g.RoMountMode != "" && !validMode(g.RoMountMode) {
		return fmt.Errorf("invalid MODE_RO_MOUNT %q: must be %q or %q", g.RoMountMode, ModeReport, ModeRemediate)
	}
	if g.ResizeMode != "" && !validMode(g.ResizeMode) {
		return fmt.Errorf("invalid MODE_RESIZE %q: must be %q or %q", g.ResizeMode, ModeReport, ModeRemediate)
	}
	// ISCSIADMPath may be empty: it then resolves to DefaultISCSIADMPath. A path
	// that points at a missing binary is not a config error but a runtime one, and
	// Rescan reports it as such rather than refusing to start.
	if g.Cooldown <= 0 {
		return errors.New("COOLDOWN must be > 0")
	}
	if g.MaxPerNode < 1 {
		return errors.New("MAX_PER_NODE must be >= 1")
	}
	if g.NodeWindow <= 0 {
		return errors.New("NODE_WINDOW must be > 0")
	}
	if g.PollTimeout <= 0 {
		return errors.New("POLL_TIMEOUT must be > 0")
	}
	if g.PollInterval <= 0 {
		return errors.New("POLL_INTERVAL must be > 0")
	}
	if g.GracePeriod < 0 {
		return errors.New("GRACE_PERIOD must be >= 0")
	}
	return nil
}

// validMode reports whether s is an accepted mode value. Every mode field —
// MODE and both per-detector overrides — is checked through it so the three
// cannot drift apart.
func validMode(s string) bool { return s == ModeReport || s == ModeRemediate }

// Decision is the pure verdict for one volume: what to do and why.
type Decision struct {
	Action       string
	Detail       string
	ShouldDelete bool
}

// Decide applies the gates to one volume without touching the cluster. Keeping
// the rules in a pure function is what makes the ownerRef and cooldown behaviour
// testable without a live API server.
//
// pod may be nil when the mount has no resolvable owner. recentOnNode is how
// many volumes on this node were already remediated inside NodeWindow.
//
// The mode consulted is this detector's (Gate.ModeFor), not MODE directly, so
// MODE_RO_MOUNT gates pod deletion on its own. The value is identical whenever
// the override is unset, which is the pre-existing behaviour.
func Decide(g Gate, v probe.Volume, pod *corev1.Pod, now, lastAttempt time.Time, recentOnNode int) Decision {
	mode := g.ModeFor(DetectorROMount)
	switch {
	case v.Status == probe.StatusHealthy:
		return Decision{Action: ActionNone}

	case mode != ModeRemediate:
		return Decision{
			Action: ActionDryRun,
			Detail: fmt.Sprintf("mode=%s, pod not deleted", mode),
		}

	case !lastAttempt.IsZero() && now.Sub(lastAttempt) < g.Cooldown:
		return Decision{
			Action: ActionCooldown,
			Detail: fmt.Sprintf("last attempt %s ago, cooldown %s",
				now.Sub(lastAttempt).Round(time.Second), g.Cooldown),
		}

	case pod == nil:
		return Decision{
			Action: ActionSkippedNoPod,
			Detail: "no pod owns this mount",
		}

	case len(pod.OwnerReferences) == 0 && !g.AllowBarePods:
		return Decision{
			Action: ActionSkippedBare,
			Detail: fmt.Sprintf("pod %s/%s has no ownerRefs; deleting it would not recreate it",
				pod.Namespace, pod.Name),
		}

	case recentOnNode >= g.MaxPerNode:
		return Decision{
			Action: ActionRateLimited,
			Detail: fmt.Sprintf("%d remediations on this node within %s, max %d",
				recentOnNode, g.NodeWindow, g.MaxPerNode),
		}

	default:
		return Decision{Action: ActionDeleted, ShouldDelete: true}
	}
}

// Result is the outcome for one volume.
type Result struct {
	PVName string
	Pod    string // "namespace/name", empty when no pod resolved
	Action string
	Before string // probe status before any action
	After  string // probe status after, empty when nothing was done
	Detail string
}

// Remediator turns probe results into actions, remembering what it already did
// so a volume that keeps failing does not get its pod deleted every cycle.
// waitFunc matches k8s.Client.WaitPodReady. It is a field so tests can stand in
// for controller recreation, which client-go's fake clientset cannot simulate:
// the fake deletes the object and never brings back a replacement with a new UID.
type waitFunc func(ctx context.Context, namespace, name, deletedUID string, timeout, interval time.Duration) (string, error)

type Remediator struct {
	cl   *k8s.Client
	g    Gate
	log  *slog.Logger
	wait waitFunc

	mu   sync.Mutex
	last map[string]time.Time // PV name -> last delete attempt
}

// New builds a Remediator. cl may be nil, in which case remediation of real
// pods fails with k8s.ErrNoClient while reporting continues.
func New(cl *k8s.Client, g Gate, log *slog.Logger) *Remediator {
	if log == nil {
		log = slog.Default()
	}
	return &Remediator{cl: cl, g: g, log: log, last: make(map[string]time.Time)}
}

// SetWaitFunc overrides how a deleted pod is waited on. Tests use it to simulate
// a controller recreating the pod; production leaves it nil and uses the client.
func (r *Remediator) SetWaitFunc(f waitFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.wait = f
}

// waitReady resolves the wait implementation and calls it.
func (r *Remediator) waitReady(ctx context.Context, ns, name, deletedUID string) (string, error) {
	r.mu.Lock()
	f := r.wait
	r.mu.Unlock()

	if f == nil {
		if r.cl == nil {
			return "", k8s.ErrNoClient
		}
		f = r.cl.WaitPodReady
	}
	return f(ctx, ns, name, deletedUID, r.g.PollTimeout, r.g.PollInterval)
}

// Gate returns the configured gate.
func (r *Remediator) Gate() Gate { return r.g }

// MarkAttempt records a delete attempt for a PV, starting its cooldown.
func (r *Remediator) MarkAttempt(pv string, now time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.last[pv] = now
}

// LastAttempt returns the last recorded delete attempt for a PV.
func (r *Remediator) LastAttempt(pv string) (time.Time, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.last[pv]
	return t, ok
}

// RecentOnNode counts recorded delete attempts inside the node window. Because
// a Remediator is per-node, its attempt map *is* the node's history.
func (r *Remediator) RecentOnNode(now time.Time) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for _, t := range r.last {
		if now.Sub(t) <= r.g.NodeWindow {
			n++
		}
	}
	return n
}

// resolvePod maps a volume's kubelet pod UID back to a pod. A not-found is
// reported as (nil, nil): an unresolvable owner is a normal state to reason
// about, not an error.
func (r *Remediator) resolvePod(ctx context.Context, v probe.Volume) (*corev1.Pod, error) {
	if v.PodUID == "" {
		return nil, nil
	}
	if r.cl == nil {
		return nil, k8s.ErrNoClient
	}
	pod, err := r.cl.PodByUID(ctx, v.PodUID)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return pod, nil
}

// Handle evaluates one volume and, if the gates allow, fixes it.
//
// reprobe is injected so tests control what the volume looks like after the
// delete; production passes probe.Probe. Healthy volumes short-circuit before
// any API call, which keeps a scan of a healthy node at one request.
func (r *Remediator) Handle(
	ctx context.Context,
	v probe.Volume,
	now time.Time,
	reprobe func(string) (string, string),
) Result {
	res := Result{PVName: v.PVName, Before: v.Status}

	if v.Status == probe.StatusHealthy {
		res.Action = ActionNone
		return res
	}

	pod, err := r.resolvePod(ctx, v)
	if err != nil {
		res.Action = ActionFailed
		res.Detail = err.Error()
		return res
	}
	if pod != nil {
		res.Pod = pod.Namespace + "/" + pod.Name
	}

	last, _ := r.LastAttempt(v.PVName)
	dec := Decide(r.g, v, pod, now, last, r.RecentOnNode(now))
	res.Action = dec.Action
	res.Detail = dec.Detail
	if !dec.ShouldDelete {
		return res
	}

	r.MarkAttempt(v.PVName, now)
	if err := r.cl.DeletePod(ctx, pod.Namespace, pod.Name, r.g.GracePeriod); err != nil {
		res.Action = ActionFailed
		res.Detail = err.Error()
		return res
	}

	status, err := r.waitReady(ctx, pod.Namespace, pod.Name, string(pod.UID))
	if err != nil {
		res.Action = ActionFailed
		res.Detail = fmt.Sprintf("pod deleted, wait failed: %v (last=%s)", err, status)
		return res
	}

	res.Action = ActionDeleted
	if reprobe != nil {
		res.After, _ = reprobe(v.MountPath)
	}
	res.Detail = fmt.Sprintf("pod %s", status)
	return res
}

// HandleAll handles every volume in a report and returns one Result per volume,
// in the same order.
func (r *Remediator) HandleAll(
	ctx context.Context,
	rep probe.Report,
	now time.Time,
	reprobe func(string) (string, string),
) []Result {
	out := make([]Result, 0, len(rep.Volumes))
	for _, v := range rep.Volumes {
		out = append(out, r.Handle(ctx, v, now, reprobe))
	}
	return out
}
