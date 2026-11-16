package remediate

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"pv-sentinel/internal/probe"
	"pv-sentinel/internal/resize"
)

var rescNow = time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)

// stuckFinding builds a finding the way the resize detector would: the freeidp
// numbers, stuck for an hour, with the cause the test asks for.
func stuckFinding(pvcName, cause string) resize.Finding {
	return resize.Finding{
		PVName:      "pvc-4e4658fb-4d16-4f14-a2b7-f87913b92894",
		Namespace:   "freeidp",
		PVCName:     pvcName,
		Protocol:    resize.ProtocolISCSI,
		SpecBytes:   10 << 30,
		StatusBytes: 5 << 30,
		State:       resize.StateStuck,
		Cause:       cause,
		Since:       rescNow.Add(-time.Hour),
		Age:         time.Hour,
	}
}

// recorder is an execFunc that counts calls and returns a canned result.
type recorder struct {
	calls  int
	gotCmd []string // name + args of the last call
	out    string
	err    error
}

func (r *recorder) execFunc() execFunc {
	return func(_ context.Context, name string, args ...string) ([]byte, error) {
		r.calls++
		r.gotCmd = append([]string{name}, args...)
		return []byte(r.out), r.err
	}
}

func TestRescanArgs(t *testing.T) {
	// Pinned because the whole remediation is this one command: an argument
	// change here changes what runs on every node in the cluster.
	got := RescanArgs()
	want := []string{"--mode", "session", "--rescan"}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("args = %v, want %v", got, want)
	}
}

func TestGate_ModeFor(t *testing.T) {
	tests := []struct {
		name     string
		gate     Gate
		detector string
		want     string
	}{
		{"no overrides inherits MODE (report)", Gate{Mode: ModeReport}, DetectorROMount, ModeReport},
		{"no overrides inherits MODE (remediate)", Gate{Mode: ModeRemediate}, DetectorResize, ModeRemediate},
		{"ro override wins for ro_mount", Gate{Mode: ModeReport, RoMountMode: ModeRemediate}, DetectorROMount, ModeRemediate},
		{"ro override does not affect resize", Gate{Mode: ModeReport, RoMountMode: ModeRemediate}, DetectorResize, ModeReport},
		{"resize override wins for resize", Gate{Mode: ModeReport, ResizeMode: ModeRemediate}, DetectorResize, ModeRemediate},
		{"resize override does not affect ro_mount", Gate{Mode: ModeRemediate, ResizeMode: ModeReport}, DetectorROMount, ModeRemediate},
		{"both overrides are independent", Gate{Mode: ModeReport, RoMountMode: ModeRemediate, ResizeMode: ModeReport}, DetectorROMount, ModeRemediate},
		{"unknown detector falls back to MODE", Gate{Mode: ModeRemediate, RoMountMode: ModeReport}, "mystery", ModeRemediate},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.gate.ModeFor(tt.detector); got != tt.want {
				t.Errorf("ModeFor(%q) = %q, want %q", tt.detector, got, tt.want)
			}
		})
	}
}

func TestGate_ISCSIADMPath(t *testing.T) {
	tests := []struct {
		name string
		gate Gate
		want string
	}{
		{"empty resolves to the default", Gate{}, DefaultISCSIADMPath},
		{"explicit path wins", Gate{ISCSIADMPath: "/sbin/iscsiadm"}, "/sbin/iscsiadm"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.gate.iscsiadmPath(); got != tt.want {
				t.Errorf("iscsiadmPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDecideRescan(t *testing.T) {
	stale := resize.CauseStaleDeviceSize
	report := Gate{Mode: ModeReport, Cooldown: time.Hour}
	fix := Gate{Mode: ModeRemediate, Cooldown: time.Hour}
	// The configuration the README argues for: global report, resize-only fix.
	resizeOnly := Gate{Mode: ModeReport, ResizeMode: ModeRemediate, Cooldown: time.Hour}
	// The inverse: mounts get the fix, rescans stay in report.
	mountOnly := Gate{Mode: ModeRemediate, ResizeMode: ModeReport, Cooldown: time.Hour}

	tests := []struct {
		name        string
		gate        Gate
		finding     resize.Finding
		lastAttempt time.Time
		wantAction  string
		wantRun     bool
	}{
		{
			name:       "a growing finding is none",
			gate:       fix,
			finding:    resize.Finding{Namespace: "ns", PVCName: "pvc", State: resize.StateGrowing},
			wantAction: ActionNone,
			wantRun:    false,
		},
		{
			name:       "a not-csi cause is skipped even in remediate mode",
			gate:       fix,
			finding:    stuckFinding("pvc", resize.CauseControllerPending),
			wantAction: ActionSkippedNotCSI,
			wantRun:    false,
		},
		{
			name:       "a node-pending cause is skipped in report mode too",
			gate:       report,
			finding:    stuckFinding("pvc", resize.CauseNodePending),
			wantAction: ActionSkippedNotCSI,
			wantRun:    false,
		},
		{
			name:       "a not-csi cause outranks dry_run",
			gate:       report,
			finding:    stuckFinding("pvc", resize.CauseNotCSI),
			wantAction: ActionSkippedNotCSI,
			wantRun:    false,
		},
		{
			name:       "report mode is a dry run",
			gate:       report,
			finding:    stuckFinding("pvc", stale),
			wantAction: ActionDryRun,
			wantRun:    false,
		},
		{
			name:       "MODE_RESIZE=remediate enables it under MODE=report",
			gate:       resizeOnly,
			finding:    stuckFinding("pvc", stale),
			wantAction: ActionRescanned,
			wantRun:    true,
		},
		{
			name:       "MODE_RESIZE=report disables it under MODE=remediate",
			gate:       mountOnly,
			finding:    stuckFinding("pvc", stale),
			wantAction: ActionDryRun,
			wantRun:    false,
		},
		{
			name:       "remediate mode with no prior attempt runs it",
			gate:       fix,
			finding:    stuckFinding("pvc", stale),
			wantAction: ActionRescanned,
			wantRun:    true,
		},
		{
			name:        "inside the cooldown it waits",
			gate:        fix,
			finding:     stuckFinding("pvc", stale),
			lastAttempt: rescNow.Add(-time.Minute),
			wantAction:  ActionCooldown,
			wantRun:     false,
		},
		{
			name:        "outside the cooldown it runs again",
			gate:        fix,
			finding:     stuckFinding("pvc", stale),
			lastAttempt: rescNow.Add(-2 * time.Hour),
			wantAction:  ActionRescanned,
			wantRun:     true,
		},
		{
			name:        "a dry run does not start a cooldown",
			gate:        report,
			finding:     stuckFinding("pvc", stale),
			lastAttempt: rescNow.Add(-time.Second),
			wantAction:  ActionDryRun,
			wantRun:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DecideRescan(tt.gate, tt.finding, rescNow, tt.lastAttempt)
			if got.Action != tt.wantAction {
				t.Errorf("Action = %q, want %q", got.Action, tt.wantAction)
			}
			if got.ShouldRescan != tt.wantRun {
				t.Errorf("ShouldRescan = %t, want %t", got.ShouldRescan, tt.wantRun)
			}
			// Every declined action has to say why, or the log line is useless.
			if got.Action != ActionNone && got.Action != ActionRescanned && got.Detail == "" {
				t.Errorf("action %q has an empty Detail", got.Action)
			}
		})
	}
}

func TestRescanner_Rescan(t *testing.T) {
	tests := []struct {
		name        string
		gate        Gate
		out         string
		err         error
		wantDetail  string
		wantErrText string
	}{
		{
			name:       "successful rescan reports the output",
			gate:       Gate{Mode: ModeRemediate, Cooldown: time.Hour},
			out:        "rescanning [ip:port,targetnode] complete\n",
			wantDetail: "rescanning [ip:port,targetnode] complete",
		},
		{
			name:       "multi-line output collapses to one line",
			gate:       Gate{Mode: ModeRemediate, Cooldown: time.Hour},
			out:        "session1\n\n  session2\n\t  session3\n",
			wantDetail: "session1 session2 session3",
		},
		{
			name:       "a custom path is the name passed to exec",
			gate:       Gate{Mode: ModeRemediate, Cooldown: time.Hour, ISCSIADMPath: "/opt/sbin/iscsiadm"},
			out:        "ok",
			wantDetail: "ok",
		},
		{
			name:        "a failing rescan wraps the error with the path",
			gate:        Gate{Mode: ModeRemediate, Cooldown: time.Hour},
			out:         "iscsiadm: could not log into no ports",
			err:         errors.New("exit status 21"),
			wantDetail:  "iscsiadm: could not log into no ports",
			wantErrText: "rescan via /usr/sbin/iscsiadm: exit status 21",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := &recorder{out: tt.out, err: tt.err}
			r := NewRescanner(tt.gate, nil)
			r.SetExecFunc(rec.execFunc())

			detail, err := r.Rescan(t.Context())
			switch {
			case tt.err == nil && err != nil:
				t.Fatalf("Rescan: unexpected error %v", err)
			case tt.err != nil:
				if err == nil || err.Error() != tt.wantErrText {
					t.Fatalf("error = %v, want %q", err, tt.wantErrText)
				}
				if !errors.Is(err, tt.err) {
					t.Errorf("error does not wrap the cause: %v", err)
				}
			}
			if detail != tt.wantDetail {
				t.Errorf("detail = %q, want %q", detail, tt.wantDetail)
			}

			wantCmd := append([]string{tt.gate.iscsiadmPath()}, RescanArgs()...)
			if fmt.Sprint(rec.gotCmd) != fmt.Sprint(wantCmd) {
				t.Errorf("exec called with %v, want %v", rec.gotCmd, wantCmd)
			}
			if rec.calls != 1 {
				t.Errorf("exec called %d times, want 1", rec.calls)
			}
		})
	}
}

// TestRescanTruncation covers the bound on log/metric-sized output: iscsiadm
// prints a line per session, so the detail must not grow with the node.
func TestRescanTruncation(t *testing.T) {
	long := strings.Repeat("x", maxExecDetail+500)
	r := NewRescanner(Gate{Mode: ModeRemediate, Cooldown: time.Hour}, nil)
	rec := &recorder{out: long}
	r.SetExecFunc(rec.execFunc())

	detail, err := r.Rescan(t.Context())
	if err != nil {
		t.Fatalf("Rescan: %v", err)
	}
	// The ellipsis is one rune but three UTF-8 bytes, so the cap is on the
	// truncated body and the marker is counted by its byte length.
	if want := maxExecDetail + len("…"); len(detail) != want {
		t.Fatalf("len(detail) = %d, want %d", len(detail), want)
	}
	if !strings.HasSuffix(detail, "…") {
		t.Errorf("truncated detail does not end with an ellipsis: %q", detail[len(detail)-8:])
	}
}

// TestDefaultExecMissingBinary is the image constraint made testable: on an
// image without iscsiadm, the fix reports ErrNoISCSIADM rather than pretending
// the remediation ran.
func TestDefaultExecMissingBinary(t *testing.T) {
	path := t.TempDir() + "/iscsiadm"
	_, err := defaultExec(t.Context(), path, RescanArgs()...)
	if !errors.Is(err, ErrNoISCSIADM) {
		t.Fatalf("error = %v, want ErrNoISCSIADM", err)
	}
	if !strings.Contains(err.Error(), path) {
		t.Errorf("error %q does not name the missing path", err)
	}
}

// TestDefaultExecRunsABinary proves the exec path itself with a binary that
// exists, so the os.Stat branch above is not the only one ever exercised.
func TestDefaultExecRunsABinary(t *testing.T) {
	sh := "/bin/sh"
	if _, err := os.Stat(sh); err != nil {
		t.Skipf("no %s on this platform: %v", sh, err)
	}
	out, err := defaultExec(t.Context(), sh, "-c", "echo hello")
	if err != nil {
		t.Fatalf("defaultExec: %v", err)
	}
	if strings.TrimSpace(string(out)) != "hello" {
		t.Errorf("output = %q, want hello", string(out))
	}
}

func TestRescanner_HandleAll(t *testing.T) {
	tests := []struct {
		name     string
		gate     Gate
		findings []resize.Finding
		out      string
		err      error
		want     []string // expected action per finding, in order
		wantExec int
	}{
		{
			name:     "no findings, no exec",
			gate:     Gate{Mode: ModeRemediate, Cooldown: time.Hour},
			findings: nil,
			wantExec: 0,
		},
		{
			name:     "report mode dry-runs two findings with no exec at all",
			gate:     Gate{Mode: ModeReport, Cooldown: time.Hour},
			findings: []resize.Finding{stuckFinding("a", resize.CauseStaleDeviceSize), stuckFinding("b", resize.CauseStaleDeviceSize)},
			want:     []string{ActionDryRun, ActionDryRun},
			wantExec: 0,
		},
		{
			name:     "remediate mode rescans once for two findings",
			gate:     Gate{Mode: ModeRemediate, Cooldown: time.Hour},
			findings: []resize.Finding{stuckFinding("a", resize.CauseStaleDeviceSize), stuckFinding("b", resize.CauseStaleDeviceSize)},
			out:      "rescan complete",
			want:     []string{ActionRescanned, ActionRescanned},
			wantExec: 1,
		},
		{
			name:     "a mixed batch execs once and reports each finding separately",
			gate:     Gate{Mode: ModeRemediate, Cooldown: time.Hour},
			findings: []resize.Finding{stuckFinding("a", resize.CauseStaleDeviceSize), stuckFinding("b", resize.CauseControllerPending)},
			out:      "rescan complete",
			want:     []string{ActionRescanned, ActionSkippedNotCSI},
			wantExec: 1,
		},
		{
			name:     "a failed exec marks every pending finding failed",
			gate:     Gate{Mode: ModeRemediate, Cooldown: time.Hour},
			findings: []resize.Finding{stuckFinding("a", resize.CauseStaleDeviceSize), stuckFinding("b", resize.CauseStaleDeviceSize)},
			err:      errors.New("exit status 21"),
			want:     []string{ActionFailed, ActionFailed},
			wantExec: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := &recorder{out: tt.out, err: tt.err}
			r := NewRescanner(tt.gate, nil)
			r.SetExecFunc(rec.execFunc())

			got := r.HandleAll(t.Context(), tt.findings, rescNow)
			if len(got) != len(tt.findings) {
				t.Fatalf("len(results) = %d, want %d", len(got), len(tt.findings))
			}
			for i, want := range tt.want {
				if got[i].Action != want {
					t.Errorf("results[%d].Action = %q, want %q", i, got[i].Action, want)
				}
			}
			if rec.calls != tt.wantExec {
				t.Errorf("exec called %d times, want %d", rec.calls, tt.wantExec)
			}
			for _, res := range got {
				if res.PVC == "" {
					t.Errorf("result for %q has an empty PVC field", res.PVName)
				}
			}
		})
	}
}

// TestRescanner_HandleAllCooldown is the second half of the cooldown contract:
// the first cycle records attempts, the next one inside the window declines to
// exec again. DecideRescan needs the recorded attempt to say so, so this is the
// only way to test the two together.
func TestRescanner_HandleAllCooldown(t *testing.T) {
	g := Gate{Mode: ModeRemediate, Cooldown: time.Hour}
	findings := []resize.Finding{stuckFinding("a", resize.CauseStaleDeviceSize)}
	ctx := t.Context()

	first := &recorder{out: "ok"}
	r := NewRescanner(g, nil)
	r.SetExecFunc(first.execFunc())
	if res := r.HandleAll(ctx, findings, rescNow); res[0].Action != ActionRescanned {
		t.Fatalf("first cycle = %+v", res[0])
	}

	second := &recorder{out: "ok"}
	r.SetExecFunc(second.execFunc())
	res := r.HandleAll(ctx, findings, rescNow.Add(time.Minute))
	if res[0].Action != ActionCooldown {
		t.Fatalf("second cycle = %q, want %s", res[0].Action, ActionCooldown)
	}
	if second.calls != 0 {
		t.Errorf("cooldown cycle still exec'd %d times", second.calls)
	}

	// Past the window it runs again.
	third := &recorder{out: "ok"}
	r.SetExecFunc(third.execFunc())
	if res := r.HandleAll(ctx, findings, rescNow.Add(2*time.Hour)); res[0].Action != ActionRescanned {
		t.Errorf("post-cooldown cycle = %q, want %s", res[0].Action, ActionRescanned)
	}
}

func TestRescanner_AttemptBookkeepingAndGate(t *testing.T) {
	g := Gate{Mode: ModeRemediate, Cooldown: time.Hour, ISCSIADMPath: "/usr/sbin/iscsiadm"}
	r := NewRescanner(g, nil)
	if r.Gate() != g {
		t.Errorf("Gate() = %+v, want %+v", r.Gate(), g)
	}
	if _, ok := r.LastAttempt("freeidp/x"); ok {
		t.Fatal("LastAttempt on a fresh rescanner reported an attempt")
	}
	r.MarkAttempt("freeidp/x", rescNow)
	got, ok := r.LastAttempt("freeidp/x")
	if !ok || !got.Equal(rescNow) {
		t.Errorf("LastAttempt = %v, %t; want %v, true", got, ok, rescNow)
	}
}

// TestRemediator_DecideUsesRoMountOverride pins the wiring change in Decide:
// with MODE=report and MODE_RO_MOUNT=remediate, pod deletion must actually be
// allowed, and the dry_run detail must name the resolved mode.
func TestRemediator_DecideUsesRoMountOverride(t *testing.T) {
	g := Gate{Mode: ModeReport, RoMountMode: ModeRemediate, Cooldown: time.Hour,
		MaxPerNode: 3, NodeWindow: time.Hour, GracePeriod: 30,
		PollTimeout: time.Minute, PollInterval: time.Second}
	v := probe.Volume{
		PVName:    "pvc-1",
		MountPath: "/var/lib/kubelet/pods/" + testUID + "/volumes/kubernetes.io~csi/pvc-1/mount",
		Status:    probe.StatusReadOnly,
		PodUID:    testUID,
	}
	pod := ownedPod("default", "web-1", testUID, testNode)

	dec := Decide(g, v, pod, rescNow, time.Time{}, 0)
	if dec.Action != ActionDeleted || !dec.ShouldDelete {
		t.Fatalf("Decide = %+v, want pod_deleted", dec)
	}

	// Same volume, override removed: back to a dry run naming MODE. This is the
	// pre-existing behaviour, unchanged by the split.
	g.RoMountMode = ""
	dec = Decide(g, v, pod, rescNow, time.Time{}, 0)
	if dec.Action != ActionDryRun {
		t.Fatalf("Decide = %q, want dry_run", dec.Action)
	}
	if want := "mode=report, pod not deleted"; dec.Detail != want {
		t.Errorf("Detail = %q, want %q", dec.Detail, want)
	}
}
