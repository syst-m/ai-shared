package remediate

import (
	"context"
	"errors"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"

	"pv-sentinel/internal/k8s"
	"pv-sentinel/internal/probe"
)

const (
	testUID  = "aaaaaaaa-0000-0000-0000-000000000000"
	testNode = "node-1"
)

func ownedPod(ns, name, uid, node string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      name,
			UID:       types.UID(uid),
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "apps/v1",
				Kind:       "ReplicaSet",
				Name:       "web-6d4f7c8d9b",
			}},
		},
		Spec: corev1.PodSpec{NodeName: node},
	}
}

func barePod(ns, name, uid, node string) *corev1.Pod {
	p := ownedPod(ns, name, uid, node)
	p.OwnerReferences = nil
	return p
}

func remediateGate() Gate {
	g := DefaultGate()
	g.Mode = ModeRemediate
	return g
}

// tightGate is DefaultGate with the durations shortened to hours, so the test
// can place attempts at exact offsets without tripping the 60m/60h defaults.
func tightGate() Gate {
	return Gate{
		Mode:         ModeRemediate,
		Cooldown:     time.Hour,
		MaxPerNode:   3,
		NodeWindow:   time.Hour,
		GracePeriod:  30,
		PollTimeout:  time.Minute,
		PollInterval: time.Second,
	}
}

func TestDefaultGateValidates(t *testing.T) {
	if err := DefaultGate().Validate(); err != nil {
		t.Fatalf("DefaultGate() does not validate: %v", err)
	}
	if err := remediateGate().Validate(); err != nil {
		t.Fatalf("ModeRemediate gate does not validate: %v", err)
	}
	if err := tightGate().Validate(); err != nil {
		t.Fatalf("tightGate() does not validate: %v", err)
	}
}

func TestGate_Validate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Gate)
		wantErr bool
	}{
		{"default", func(*Gate) {}, false},
		{"remediate", func(g *Gate) { g.Mode = ModeRemediate }, false},
		{"bogus mode", func(g *Gate) { g.Mode = "yolo" }, true},
		{"empty mode", func(g *Gate) { g.Mode = "" }, true},
		{"zero cooldown", func(g *Gate) { g.Cooldown = 0 }, true},
		{"zero max per node", func(g *Gate) { g.MaxPerNode = 0 }, true},
		{"negative max per node", func(g *Gate) { g.MaxPerNode = -1 }, true},
		{"zero node window", func(g *Gate) { g.NodeWindow = 0 }, true},
		{"zero poll timeout", func(g *Gate) { g.PollTimeout = 0 }, true},
		{"zero poll interval", func(g *Gate) { g.PollInterval = 0 }, true},
		{"negative grace period", func(g *Gate) { g.GracePeriod = -1 }, true},
		{"zero grace period is fine", func(g *Gate) { g.GracePeriod = 0 }, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := DefaultGate()
			tt.mutate(&g)
			if err := g.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() err = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestDecide is the exhaustive branch table. Every action constant must be
// reachable from exactly one documented condition, in the documented order.
func TestDecide(t *testing.T) {
	now := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	ro := probe.Volume{PVName: "pvc-x", Status: probe.StatusReadOnly}
	healthy := probe.Volume{PVName: "pvc-x", Status: probe.StatusHealthy}
	unknown := probe.Volume{PVName: "pvc-x", Status: probe.StatusUnknown}
	owned := ownedPod("default", "web-1", testUID, testNode)
	bare := barePod("default", "solo", testUID, testNode)
	allowBare := tightGate()
	allowBare.AllowBarePods = true

	tests := []struct {
		name         string
		gate         Gate
		vol          probe.Volume
		pod          *corev1.Pod
		lastAttempt  time.Time
		recentOnNode int
		wantAction   string
		wantDelete   bool
	}{
		{
			name: "healthy volume needs nothing even in remediate mode",
			gate: remediateGate(), vol: healthy, pod: owned,
			wantAction: ActionNone,
		},
		{
			name: "read-only in report mode is a dry run",
			gate: DefaultGate(), vol: ro, pod: owned,
			wantAction: ActionDryRun,
		},
		{
			name: "unknown in report mode is also a dry run",
			gate: DefaultGate(), vol: unknown, pod: owned,
			wantAction: ActionDryRun,
		},
		{
			name: "unknown in remediate mode is still actionable",
			gate: tightGate(), vol: unknown, pod: owned,
			wantAction: ActionDeleted, wantDelete: true,
		},
		{
			name: "recent attempt on the same PV is in cooldown",
			gate: tightGate(), vol: ro, pod: owned,
			lastAttempt: now.Add(-30 * time.Minute), // cooldown is 1h here
			wantAction:  ActionCooldown,
		},
		{
			name: "an attempt older than the cooldown is not in cooldown",
			gate: tightGate(), vol: ro, pod: owned,
			lastAttempt: now.Add(-2 * time.Hour),
			wantAction:  ActionDeleted, wantDelete: true,
		},
		{
			name: "no resolvable owner",
			gate: tightGate(), vol: ro, pod: nil,
			wantAction: ActionSkippedNoPod,
		},
		{
			// The bare-pod check precedes the rate limit on purpose: a pod we
			// would refuse to delete must not consume node deletion budget.
			name: "bare pod refused by default",
			gate: tightGate(), vol: ro, pod: bare,
			wantAction: ActionSkippedBare,
		},
		{
			name: "bare pod allowed when AllowBarePods",
			gate: allowBare, vol: ro, pod: bare,
			wantAction: ActionDeleted, wantDelete: true,
		},
		{
			name: "bare pod refused even when the node is rate limited",
			gate: tightGate(), vol: ro, pod: bare, recentOnNode: 99,
			wantAction: ActionSkippedBare,
		},
		{
			name: "node budget exhausted",
			gate: tightGate(), vol: ro, pod: owned, recentOnNode: 3,
			wantAction: ActionRateLimited,
		},
		{
			name: "node budget one under the cap still deletes",
			gate: tightGate(), vol: ro, pod: owned, recentOnNode: 2,
			wantAction: ActionDeleted, wantDelete: true,
		},
		{
			name: "healthy beats cooldown",
			gate: tightGate(), vol: healthy, pod: owned,
			lastAttempt: now, wantAction: ActionNone,
		},
		{
			name: "dry run beats cooldown",
			gate: DefaultGate(), vol: ro, pod: owned,
			lastAttempt: now, wantAction: ActionDryRun,
		},
		{
			name: "cooldown beats a missing pod",
			gate: tightGate(), vol: ro, pod: nil,
			lastAttempt: now, wantAction: ActionCooldown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Decide(tt.gate, tt.vol, tt.pod, now, tt.lastAttempt, tt.recentOnNode)
			if got.Action != tt.wantAction {
				t.Errorf("Action = %q, want %q (detail %q)", got.Action, tt.wantAction, got.Detail)
			}
			if got.ShouldDelete != tt.wantDelete {
				t.Errorf("ShouldDelete = %v, want %v", got.ShouldDelete, tt.wantDelete)
			}
			if got.Action != ActionNone && got.Action != ActionDeleted && got.Detail == "" {
				t.Errorf("Action %q must explain itself, detail is empty", got.Action)
			}
		})
	}
}

func TestRemediator_AttemptBookkeeping(t *testing.T) {
	r := New(nil, tightGate(), nil)
	now := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)

	if _, ok := r.LastAttempt("pvc-a"); ok {
		t.Error("LastAttempt should be unset before any attempt")
	}
	r.MarkAttempt("pvc-a", now.Add(-2*time.Hour))
	r.MarkAttempt("pvc-b", now.Add(-30*time.Minute))
	r.MarkAttempt("pvc-c", now)

	if got, ok := r.LastAttempt("pvc-a"); !ok || !got.Equal(now.Add(-2*time.Hour)) {
		t.Errorf("LastAttempt(pvc-a) = %v %v, want %v true", got, ok, now.Add(-2*time.Hour))
	}

	// NodeWindow is 1h, so the pvc-a attempt from two hours ago has aged out and
	// only pvc-b and pvc-c count.
	if got := r.RecentOnNode(now); got != 2 {
		t.Errorf("RecentOnNode = %d, want 2 (pvc-b and pvc-c are inside the 1h window)", got)
	}
	// Three hours later all three have aged out, which is what lets the node's
	// deletion budget refill instead of staying permanently spent.
	if got := r.RecentOnNode(now.Add(3 * time.Hour)); got != 0 {
		t.Errorf("RecentOnNode at +3h = %d, want 0 (every attempt is older than the 1h window)", got)
	}
}

func TestRemediator_GateAccessor(t *testing.T) {
	g := tightGate()
	if got := New(nil, g, nil).Gate(); got != g {
		t.Errorf("Gate() = %+v, want %+v", got, g)
	}
}

// newRemediator builds a Remediator over a fake clientset seeded with pods, plus
// a waitFunc standing in for controller recreation — the fake deletes objects and
// never resurrects them with a new UID, so the success path is unreachable
// without the seam.
func newRemediator(t *testing.T, g Gate, pods ...*corev1.Pod) (*Remediator, *k8s.Client) {
	t.Helper()
	objs := make([]runtime.Object, 0, len(pods))
	for _, p := range pods {
		objs = append(objs, p)
	}
	fakeCS := fake.NewSimpleClientset(objs...)
	cl := k8s.New(fakeCS, testNode)
	r := New(cl, g, nil)
	r.SetWaitFunc(func(context.Context, string, string, string, time.Duration, time.Duration) (string, error) {
		return "Running/true", nil
	})
	return r, cl
}

func TestRemediator_Handle_HealthySkipsAPI(t *testing.T) {
	// A healthy volume must short-circuit before any API call. The client here
	// is nil, so reaching the API at all would surface as an error.
	r := New(nil, remediateGate(), nil)
	res := r.Handle(context.Background(),
		probe.Volume{PVName: "pvc-a", Status: probe.StatusHealthy}, time.Now(), nil)

	if res.Action != ActionNone {
		t.Errorf("Action = %q, want %q", res.Action, ActionNone)
	}
	if res.Before != probe.StatusHealthy {
		t.Errorf("Before = %q, want %q", res.Before, probe.StatusHealthy)
	}
}

func TestRemediator_Handle_NoClient(t *testing.T) {
	r := New(nil, remediateGate(), nil)
	res := r.Handle(context.Background(),
		probe.Volume{PVName: "pvc-a", Status: probe.StatusReadOnly, PodUID: testUID}, time.Now(), nil)

	if res.Action != ActionFailed {
		t.Errorf("Action = %q, want %q", res.Action, ActionFailed)
	}
	if res.Detail != k8s.ErrNoClient.Error() {
		t.Errorf("Detail = %q, want %q", res.Detail, k8s.ErrNoClient.Error())
	}
}

func TestRemediator_Handle_DryRunDeletesNothing(t *testing.T) {
	r, cl := newRemediator(t, DefaultGate(), ownedPod("default", "web-1", testUID, testNode))
	res := r.Handle(context.Background(),
		probe.Volume{PVName: "pvc-a", MountPath: "/tmp/x", Status: probe.StatusReadOnly, PodUID: testUID},
		time.Now(), nil)

	if res.Action != ActionDryRun {
		t.Fatalf("Action = %q, want %q", res.Action, ActionDryRun)
	}
	if res.Pod != "default/web-1" {
		t.Errorf("Pod = %q, want default/web-1", res.Pod)
	}
	if res.After != "" {
		t.Errorf("After = %q, want empty (nothing was done)", res.After)
	}
	// The pod must still exist — report mode must not mutate the cluster.
	if _, err := cl.PodByName(context.Background(), "default", "web-1"); err != nil {
		t.Errorf("report mode deleted a pod: %v", err)
	}
	// A dry run must not start a cooldown either, or flipping to remediate mode
	// later would be silently gated by months of report cycles.
	if _, ok := r.LastAttempt("pvc-a"); ok {
		t.Error("dry run recorded an attempt; report mode must not start cooldowns")
	}
}

func TestRemediator_Handle_DeletesAndReprobes(t *testing.T) {
	r, cl := newRemediator(t, tightGate(), ownedPod("default", "web-1", testUID, testNode))
	now := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)

	reprobe := func(string) (string, string) { return probe.StatusHealthy, "" }
	res := r.Handle(context.Background(),
		probe.Volume{PVName: "pvc-a", MountPath: t.TempDir(), Status: probe.StatusReadOnly, PodUID: testUID},
		now, reprobe)

	if res.Action != ActionDeleted {
		t.Fatalf("Action = %q, want %q (detail %q)", res.Action, ActionDeleted, res.Detail)
	}
	if res.Before != probe.StatusReadOnly || res.After != probe.StatusHealthy {
		t.Errorf("before/after = %q/%q, want read_only/healthy", res.Before, res.After)
	}
	if res.Pod != "default/web-1" {
		t.Errorf("Pod = %q, want default/web-1", res.Pod)
	}
	if res.Detail != "pod Running/true" {
		t.Errorf("Detail = %q, want %q", res.Detail, "pod Running/true")
	}
	// The pod is gone from the fake, proving the delete actually fired.
	if _, err := cl.PodByName(context.Background(), "default", "web-1"); err == nil {
		t.Error("pod still exists after remediation")
	}
	// And the attempt is recorded, so the next cycle hits cooldown.
	if got, ok := r.LastAttempt("pvc-a"); !ok || !got.Equal(now) {
		t.Errorf("LastAttempt(pvc-a) = %v %v, want %v true", got, ok, now)
	}
}

func TestRemediator_Handle_WaitFailureReportsFailed(t *testing.T) {
	fakeCS := fake.NewSimpleClientset(ownedPod("default", "web-1", testUID, testNode))
	cl := k8s.New(fakeCS, testNode)
	r := New(cl, tightGate(), nil)
	r.SetWaitFunc(func(context.Context, string, string, string, time.Duration, time.Duration) (string, error) {
		return "Pending/false", errors.New("boom")
	})

	res := r.Handle(context.Background(),
		probe.Volume{PVName: "pvc-a", MountPath: t.TempDir(), Status: probe.StatusReadOnly, PodUID: testUID},
		time.Now(), nil)

	if res.Action != ActionFailed {
		t.Fatalf("Action = %q, want %q", res.Action, ActionFailed)
	}
	// The attempt still counts: a fix that timed out must not be retried
	// immediately on the next cycle.
	if _, ok := r.LastAttempt("pvc-a"); !ok {
		t.Error("a failed remediation should still start the cooldown")
	}
}

func TestRemediator_HandleAll_OrderAndActions(t *testing.T) {
	dir := t.TempDir()
	r, _ := newRemediator(t, DefaultGate(), ownedPod("default", "web-1", testUID, testNode))
	rep := probe.Report{
		Node: testNode,
		Volumes: []probe.Volume{
			{PVName: "pvc-a", MountPath: dir, Status: probe.StatusHealthy, PodUID: testUID},
			{PVName: "pvc-b", MountPath: dir, Status: probe.StatusReadOnly, PodUID: testUID},
			{PVName: "pvc-c", MountPath: dir, Status: probe.StatusUnknown, PodUID: ""},
		},
	}

	got := r.HandleAll(context.Background(), rep, time.Now(), nil)
	if len(got) != len(rep.Volumes) {
		t.Fatalf("HandleAll returned %d results, want %d", len(got), len(rep.Volumes))
	}
	want := []struct {
		pv, action string
	}{
		{"pvc-a", ActionNone},
		{"pvc-b", ActionDryRun},
		// pvc-c has no pod, but the mode check precedes the pod check, so in
		// report mode it reports as a dry run rather than skipped_no_pod.
		{"pvc-c", ActionDryRun},
	}
	for i, w := range want {
		if got[i].PVName != w.pv {
			t.Errorf("result %d is for %q, want %q", i, got[i].PVName, w.pv)
		}
		if got[i].Action != w.action {
			t.Errorf("result %d (%s) Action = %q, want %q", i, w.pv, got[i].Action, w.action)
		}
	}
}
