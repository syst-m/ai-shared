package remediate

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"pv-sentinel/internal/resize"
)

// Actions produced by the resize remediation. ActionNone, ActionDryRun,
// ActionCooldown and ActionFailed are shared with the mount remediation: an
// action vocabulary split per remediation would double the metric's label set for
// no information gain.
const (
	// ActionRescanned means iscsiadm ran and exited 0.
	ActionRescanned = "iscsi_rescanned"
	// ActionSkippedNotCSI means the finding is real but this node's fix does not
	// apply to it: the volume is not served over iSCSI, so rescanning the iSCSI
	// session cannot change its size.
	ActionSkippedNotCSI = "skipped_not_iscsi"
)

// maxExecDetail caps how much command output is kept in a Result and a log line.
// iscsiadm prints a line per session, so a node with many targets produces output
// proportional to the number of volumes attached — and a metric label or a log
// field that grows with the cluster is the mistake the repo's metric rules exist
// to prevent.
const maxExecDetail = 512

// RescanArgs are the arguments to the one command this remediation runs.
//
// `iscsiadm --mode session --rescan` re-reads every session on the initiator,
// which is what makes the kernel notice that a LUN grew. It is a query plus a
// rescan of existing sessions: it does not log in, does not format, and does not
// touch a mounted filesystem, which is why this remediation needs a far less
// careful gate than deleting a pod does.
func RescanArgs() []string { return []string{"--mode", "session", "--rescan"} }

// execFunc is the seam for running the rescan binary, mirroring waitFunc in
// remediate.go: production execs, tests return a canned result and count calls.
type execFunc func(ctx context.Context, name string, args ...string) ([]byte, error)

// ErrNoISCSIADM is reported when the configured binary is not on disk. It exists
// so "this image cannot run the fix" is a named condition an operator can grep
// for, distinct from a rescan that ran and failed: one is an image problem, the
// other a storage problem, and a log line that conflates them sends the operator
// to the wrong place.
var ErrNoISCSIADM = errors.New("iscsiadm not available")

// RescanDecision is the pure verdict for one resize finding.
type RescanDecision struct {
	Action       string
	Detail       string
	ShouldRescan bool
}

// DecideRescan applies the gates to one stuck-expansion finding.
//
// The ordering has one deliberate inversion relative to Decide: the
// transport check comes *before* the mode check. A dry run means "we would have
// acted", and for a volume that is not on iSCSI we would not have acted, so
// reporting dry_run there would be a lie. Answering skipped_not_iscsi in report
// mode is what lets an operator read one week of metrics and learn which fixes
// this node is actually eligible for before turning the mode on.
//
// After that the order matches Decide: mode, then cooldown, then act.
func DecideRescan(g Gate, f resize.Finding, now, lastAttempt time.Time) RescanDecision {
	mode := g.ModeFor(DetectorResize)
	switch {
	case f.State != resize.StateStuck:
		return RescanDecision{Action: ActionNone, Detail: "finding is " + f.State + ", not stuck"}

	case f.Cause != resize.CauseStaleDeviceSize:
		return RescanDecision{
			Action: ActionSkippedNotCSI,
			Detail: fmt.Sprintf("cause=%s, an iSCSI rescan cannot fix it", f.Cause),
		}

	case mode != ModeRemediate:
		return RescanDecision{
			Action: ActionDryRun,
			Detail: fmt.Sprintf("mode=%s, rescan not run", mode),
		}

	case !lastAttempt.IsZero() && now.Sub(lastAttempt) < g.Cooldown:
		return RescanDecision{
			Action: ActionCooldown,
			Detail: fmt.Sprintf("last rescan %s ago, cooldown %s",
				now.Sub(lastAttempt).Round(time.Second), g.Cooldown),
		}

	default:
		return RescanDecision{Action: ActionRescanned, ShouldRescan: true}
	}
}

// RescanResult is the outcome for one finding.
type RescanResult struct {
	PVC    string // "namespace/name"
	PVName string
	Action string
	Detail string
}

// Rescanner runs the iSCSI rescan on this node for the findings the resize
// detector produced.
//
// It is a separate type from Remediator rather than a method on it because the
// two remediations have different state: Remediator keys its cooldown on a PV
// and needs a pod, while this keys on a PVC and needs no pod at all — in the
// freeidp case the pod is Pending and may not even exist yet.
type Rescanner struct {
	g    Gate
	log  *slog.Logger
	exec execFunc

	mu   sync.Mutex
	last map[string]time.Time // "namespace/pvc" -> last rescan attempt
}

// NewRescanner builds a Rescanner from the gate. The mode it consults is
// MODE_RESIZE (falling back to MODE), so a deployment that sets neither leaves
// this remediation in report mode.
func NewRescanner(g Gate, log *slog.Logger) *Rescanner {
	if log == nil {
		log = slog.Default()
	}
	return &Rescanner{g: g, log: log, last: make(map[string]time.Time)}
}

// SetExecFunc overrides how the binary is run. Tests use it to avoid execing a
// real iscsiadm; production leaves it nil and uses exec.CommandContext.
func (r *Rescanner) SetExecFunc(f execFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.exec = f
}

// Gate returns the configured gate.
func (r *Rescanner) Gate() Gate { return r.g }

// iscsiadmPath resolves the configured binary, falling back to the default so a
// hand-built Gate still names a real path.
func (g Gate) iscsiadmPath() string {
	if g.ISCSIADMPath == "" {
		return DefaultISCSIADMPath
	}
	return g.ISCSIADMPath
}

// defaultExec runs the command and returns its combined output.
//
// The existence check comes first so a missing binary is reported as
// ErrNoISCSIADM naming the path. os/exec's own sentinel (exec.ErrNotFound) says
// the same thing in a wording that reads like a Go plumbing error; the operator
// needs to see which binary is missing, because on this image that is the answer.
func defaultExec(ctx context.Context, name string, args ...string) ([]byte, error) {
	if _, err := os.Stat(name); err != nil {
		return nil, fmt.Errorf("%s: %w", name, ErrNoISCSIADM)
	}
	// #nosec G204 -- name is operator config (ISCSIADM_PATH), defaulting to
	// DefaultISCSIADMPath; it is not derived from any API object or request.
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}

// Rescan runs the node-wide rescan once and returns a summarised version of its
// output.
func (r *Rescanner) Rescan(ctx context.Context) (string, error) {
	r.mu.Lock()
	f := r.exec
	r.mu.Unlock()
	if f == nil {
		f = defaultExec
	}

	path := r.g.iscsiadmPath()
	out, err := f(ctx, path, RescanArgs()...)
	detail := summarize(string(out))
	if err != nil {
		return detail, fmt.Errorf("rescan via %s: %w", path, err)
	}
	return detail, nil
}

// summarize collapses whitespace and truncates, so a multi-line iscsiadm dump
// becomes one bounded log field.
func summarize(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) <= maxExecDetail {
		return s
	}
	return s[:maxExecDetail] + "…"
}

// MarkAttempt records a rescan for a finding key, starting its cooldown.
func (r *Rescanner) MarkAttempt(key string, now time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.last[key] = now
}

// LastAttempt returns the last recorded rescan for a key.
func (r *Rescanner) LastAttempt(key string) (time.Time, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.last[key]
	return t, ok
}

// HandleAll turns a set of findings into results, running the rescan at most
// once per cycle.
//
// One exec for many findings is the point: `iscsiadm --mode session --rescan`
// rescans every session on the initiator, so N stuck volumes on this node need
// one command, not N. Executing per finding would multiply the process spawns
// for an identical effect and make the cooldown look like the throttle when the
// command itself is already global.
//
// An empty finding list never execs, so a healthy node costs nothing.
func (r *Rescanner) HandleAll(
	ctx context.Context,
	findings []resize.Finding,
	now time.Time,
) []RescanResult {
	if len(findings) == 0 {
		return nil
	}

	results := make([]RescanResult, len(findings))
	pending := make([]int, 0, len(findings))
	for i, f := range findings {
		last, _ := r.LastAttempt(f.Key())
		dec := DecideRescan(r.g, f, now, last)
		results[i] = RescanResult{
			PVC:    f.Key(),
			PVName: f.PVName,
			Action: dec.Action,
			Detail: dec.Detail,
		}
		if dec.ShouldRescan {
			pending = append(pending, i)
		}
	}
	if len(pending) == 0 {
		return results
	}

	// Mark before running, as Remediator.Handle does: an attempt that crashes the
	// process must still count, or the next cycle would repeat it immediately.
	for _, i := range pending {
		r.MarkAttempt(findings[i].Key(), now)
	}

	detail, err := r.Rescan(ctx)
	for _, i := range pending {
		if err != nil {
			results[i].Action = ActionFailed
			results[i].Detail = err.Error()
			continue
		}
		results[i].Action = ActionRescanned
		results[i].Detail = detail
	}
	return results
}
