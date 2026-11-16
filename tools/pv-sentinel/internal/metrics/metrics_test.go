package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	dto "github.com/prometheus/client_model/go"

	"pv-sentinel/internal/probe"
	"pv-sentinel/internal/remediate"
	"pv-sentinel/internal/resize"
)

// registered lists every metric this package must export. The test asserts the
// set exactly, so removing or renaming a metric fails rather than silently
// shrinking the monitoring surface.
var registered = []string{
	"pv_sentinel_volume_status",
	"pv_sentinel_probes_total",
	"pv_sentinel_remediations_total",
	"pv_sentinel_healthy_volumes",
	"pv_sentinel_read_only_volumes",
	"pv_sentinel_unknown_volumes",
	"pv_sentinel_attached_not_mounted_volumes",
	"pv_sentinel_volumes_discovered",
	"pv_sentinel_last_scan_timestamp_seconds",
	"pv_sentinel_resize_status",
	"pv_sentinel_stuck_expansions",
	"pv_sentinel_resize_last_scan_timestamp_seconds",
	"pv_sentinel_resize_remediations_total",
}

// gather flattens the registry into name -> family. testutil.ToFloat64 only
// works on a collector with exactly one series, which rules out every *Vec here.
func gather(t *testing.T, m *Metrics) map[string]*dto.MetricFamily {
	t.Helper()
	fams, err := m.registry.Gather()
	if err != nil {
		t.Fatalf("Gather: %v", err)
	}
	out := make(map[string]*dto.MetricFamily, len(fams))
	for _, f := range fams {
		out[f.GetName()] = f
	}
	return out
}

// gaugeValue reads the value of a single-series gauge.
func gaugeValue(t *testing.T, m *Metrics, family string) float64 {
	t.Helper()
	f, ok := gather(t, m)[family]
	if !ok {
		t.Fatalf("metric %s is not registered", family)
	}
	if len(f.GetMetric()) != 1 {
		t.Fatalf("metric %s has %d series, want exactly 1", family, len(f.GetMetric()))
	}
	return f.GetMetric()[0].GetGauge().GetValue()
}

// counterValues reads every series of a CounterVec keyed by its label values.
func counterValues(t *testing.T, m *Metrics, family string) map[string]float64 {
	t.Helper()
	f, ok := gather(t, m)[family]
	if !ok {
		t.Fatalf("metric %s is not registered", family)
	}
	out := make(map[string]float64, len(f.GetMetric()))
	for _, mt := range f.GetMetric() {
		key := ""
		for _, lp := range mt.GetLabel() {
			if key != "" {
				key += "|"
			}
			key += lp.GetValue()
		}
		out[key] = mt.GetCounter().GetValue()
	}
	return out
}

// TestNewRegistersEverything seeds one report and one result before gathering:
// prometheus omits a *Vec with zero children from Gather(), so an unseeded
// registry would report only the plain gauges and hide every vec.
func TestNewRegistersEverything(t *testing.T) {
	m := New()
	m.SetNode("node-1")
	m.RecordReport(sampleReport())
	m.RecordResult(remediate.Result{Action: remediate.ActionNone})
	m.RecordResize(sampleResizeReport())
	m.RecordResizeResult(remediate.RescanResult{Action: remediate.ActionDryRun})

	fams := gather(t, m)
	for _, name := range registered {
		if _, ok := fams[name]; !ok {
			t.Errorf("metric %s is not registered", name)
		}
	}
	if len(fams) != len(registered) {
		t.Errorf("registry has %d metric families, want %d", len(fams), len(registered))
	}
}

// TestNewTwice guards against the default-registry collision class: New must use
// a private registry so two instances (or a test and a live server) never
// trigger MustRegister's duplicate-registration panic.
func TestNewTwice(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("New() panics when called twice: %v", r)
		}
	}()
	_ = New()
	_ = New()
}

func sampleReport() probe.Report {
	return probe.Report{
		Node: "node-1",
		At:   time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC),
		Volumes: []probe.Volume{
			{PVName: "pvc-a", FSType: "ext4", Status: probe.StatusHealthy},
			{PVName: "pvc-b", FSType: "ext4", Status: probe.StatusHealthy},
			{PVName: "pvc-c", FSType: "xfs", Status: probe.StatusReadOnly},
			{PVName: "pvc-d", FSType: "ext4", Status: probe.StatusUnknown},
		},
		Healthy:            2,
		ReadOnly:           1,
		Unknown:            1,
		MountedPV:          []string{"pvc-a", "pvc-b", "pvc-c", "pvc-d"},
		AttachedNotMounted: []string{"pvc-e", "pvc-f"},
	}
}

// sampleResizeReport is the freeidp case: one stuck iSCSI expansion on a node
// that also has three healthy claims and one resize still in flight.
func sampleResizeReport() resize.Report {
	at := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	return resize.Report{
		Node: "node-1",
		At:   at,
		Findings: []resize.Finding{{
			PVName:      "pvc-4e4658fb-4d16-4f14-a2b7-f87913b92894",
			Namespace:   "freeidp",
			PVCName:     "freeidp-pg-2",
			Protocol:    resize.ProtocolISCSI,
			SpecBytes:   10 << 30,
			StatusBytes: 5 << 30,
			State:       resize.StateStuck,
			Cause:       resize.CauseStaleDeviceSize,
			Since:       at.Add(-time.Hour),
			Age:         time.Hour,
		}},
		Current: 3,
		Stuck:   1,
	}
}

func TestRecordReport(t *testing.T) {
	m := New()
	rep := sampleReport()
	m.SetNode(rep.Node)
	m.RecordReport(rep)

	if got := gaugeValue(t, m, "pv_sentinel_volumes_discovered"); got != 4 {
		t.Errorf("volumes_discovered = %v, want 4", got)
	}
	if got := gaugeValue(t, m, "pv_sentinel_healthy_volumes"); got != 2 {
		t.Errorf("healthy_volumes = %v, want 2", got)
	}
	if got := gaugeValue(t, m, "pv_sentinel_read_only_volumes"); got != 1 {
		t.Errorf("read_only_volumes = %v, want 1", got)
	}
	if got := gaugeValue(t, m, "pv_sentinel_unknown_volumes"); got != 1 {
		t.Errorf("unknown_volumes = %v, want 1", got)
	}
	if got := gaugeValue(t, m, "pv_sentinel_attached_not_mounted_volumes"); got != 2 {
		t.Errorf("attached_not_mounted_volumes = %v, want 2", got)
	}
	if got := gaugeValue(t, m, "pv_sentinel_last_scan_timestamp_seconds"); got != float64(rep.At.Unix()) {
		t.Errorf("last_scan_timestamp_seconds = %v, want %v", got, rep.At.Unix())
	}

	probes := counterValues(t, m, "pv_sentinel_probes_total")
	for status, want := range map[string]float64{
		probe.StatusHealthy: 2, probe.StatusReadOnly: 1, probe.StatusUnknown: 1,
	} {
		if probes[status] != want {
			t.Errorf("probes_total{status=%q} = %v, want %v", status, probes[status], want)
		}
	}

	// The per-volume series must carry the node, pv, fs and status labels — the
	// combination that lets a dashboard find "rw in the table, EROFS in practice".
	statuses := map[string]string{}
	fam := gather(t, m)["pv_sentinel_volume_status"]
	for _, mt := range fam.GetMetric() {
		labels := map[string]string{}
		for _, lp := range mt.GetLabel() {
			labels[lp.GetName()] = lp.GetValue()
		}
		if labels["node"] != "node-1" {
			t.Errorf("volume_status series has node %q, want node-1", labels["node"])
		}
		if labels["pv"] == "" || labels["fs"] == "" || labels["status"] == "" {
			t.Errorf("volume_status series is missing a label: %v", labels)
		}
		statuses[labels["pv"]] = labels["status"]
	}
	for pv, want := range map[string]string{
		"pvc-a": probe.StatusHealthy, "pvc-b": probe.StatusHealthy,
		"pvc-c": probe.StatusReadOnly, "pvc-d": probe.StatusUnknown,
	} {
		if statuses[pv] != want {
			t.Errorf("volume_status for %s = %q, want %q", pv, statuses[pv], want)
		}
	}
}

// TestRecordReport_AgesOutStaleSeries covers the case a Prometheus gauge most
// often gets wrong: a volume that no longer exists must stop being reported, or
// the dashboard shows a broken volume forever after it is fixed and detached.
func TestRecordReport_AgesOutStaleSeries(t *testing.T) {
	m := New()
	m.SetNode("node-1")
	m.RecordReport(probe.Report{
		Node:      "node-1",
		At:        time.Now(),
		Volumes:   []probe.Volume{{PVName: "pvc-a", FSType: "ext4", Status: probe.StatusHealthy}},
		Healthy:   1,
		MountedPV: []string{"pvc-a"},
	})
	if got := len(gather(t, m)["pv_sentinel_volume_status"].GetMetric()); got != 1 {
		t.Fatalf("volume_status has %d series after first report, want 1", got)
	}

	// pvc-a flips to read_only and pvc-b appears.
	m.RecordReport(probe.Report{
		Node:      "node-1",
		At:        time.Now(),
		Volumes:   []probe.Volume{{PVName: "pvc-a", FSType: "ext4", Status: probe.StatusReadOnly}},
		ReadOnly:  1,
		MountedPV: []string{"pvc-a"},
	})
	fam := gather(t, m)["pv_sentinel_volume_status"]
	if got := len(fam.GetMetric()); got != 1 {
		t.Fatalf("volume_status has %d series after second report, want 1 (the healthy series must be deleted)", got)
	}
	labels := map[string]string{}
	for _, mt := range fam.GetMetric() {
		for _, lp := range mt.GetLabel() {
			labels[lp.GetName()] = lp.GetValue()
		}
	}
	if labels["status"] != probe.StatusReadOnly {
		t.Errorf("status label = %q, want %q", labels["status"], probe.StatusReadOnly)
	}
}

// TestRecordReport_NodeChange makes sure a cached series set from one node is
// never used to age out another node's metrics.
func TestRecordReport_NodeChange(t *testing.T) {
	m := New()
	m.SetNode("node-1")
	m.RecordReport(probe.Report{
		Node: "node-1", At: time.Now(),
		Volumes: []probe.Volume{{PVName: "pvc-a", FSType: "ext4", Status: probe.StatusHealthy}},
		Healthy: 1, MountedPV: []string{"pvc-a"},
	})
	m.SetNode("node-2")
	m.RecordReport(probe.Report{
		Node: "node-2", At: time.Now(),
		Volumes: []probe.Volume{{PVName: "pvc-z", FSType: "ext4", Status: probe.StatusHealthy}},
		Healthy: 1, MountedPV: []string{"pvc-z"},
	})
	if got := len(gather(t, m)["pv_sentinel_volume_status"].GetMetric()); got != 2 {
		t.Errorf("volume_status has %d series, want 2 (node-1's series must survive)", got)
	}
}

func TestRecordResult(t *testing.T) {
	m := New()
	for _, action := range []string{
		remediate.ActionNone, remediate.ActionNone, remediate.ActionDryRun, remediate.ActionDeleted,
	} {
		m.RecordResult(remediate.Result{Action: action})
	}
	got := counterValues(t, m, "pv_sentinel_remediations_total")
	want := map[string]float64{
		remediate.ActionNone: 2, remediate.ActionDryRun: 1, remediate.ActionDeleted: 1,
	}
	for action, w := range want {
		if got[action] != w {
			t.Errorf("remediations_total{action=%q} = %v, want %v", action, got[action], w)
		}
	}
}

func TestRecordResize(t *testing.T) {
	m := New()
	m.SetNode("node-1")
	rep := sampleResizeReport()
	m.RecordResize(rep)

	if got := gaugeValue(t, m, "pv_sentinel_stuck_expansions"); got != 1 {
		t.Errorf("stuck_expansions = %v, want 1", got)
	}
	if got := gaugeValue(t, m, "pv_sentinel_resize_last_scan_timestamp_seconds"); got != float64(rep.At.Unix()) {
		t.Errorf("resize_last_scan_timestamp_seconds = %v, want %v", got, rep.At.Unix())
	}

	fam := gather(t, m)["pv_sentinel_resize_status"]
	if got := len(fam.GetMetric()); got != 1 {
		t.Fatalf("resize_status has %d series, want 1", got)
	}
	labels := map[string]string{}
	for _, lp := range fam.GetMetric()[0].GetLabel() {
		labels[lp.GetName()] = lp.GetValue()
	}
	// The three labels are the whole point: which node, which volume, which
	// claim. Without pvc the count says "something is stuck here" and stops.
	for key, want := range map[string]string{
		"node": "node-1",
		"pv":   "pvc-4e4658fb-4d16-4f14-a2b7-f87913b92894",
		"pvc":  "freeidp-pg-2",
	} {
		if labels[key] != want {
			t.Errorf("resize_status{%s} = %q, want %q", key, labels[key], want)
		}
	}
	if got := fam.GetMetric()[0].GetGauge().GetValue(); got != 1 {
		t.Errorf("resize_status value = %v, want 1 (the series existing is the signal)", got)
	}
}

// TestRecordResize_AgesOutStaleSeries is the resize half of the same guarantee
// TestRecordReport_AgesOutStaleSeries gives for mounts: a completed resize must
// stop being reported. The two caches are separate, so this also proves the
// resize path ages independently of the volume path.
func TestRecordResize_AgesOutStaleSeries(t *testing.T) {
	m := New()
	m.SetNode("node-1")
	m.RecordReport(sampleReport())
	m.RecordResize(sampleResizeReport())
	if got := len(gather(t, m)["pv_sentinel_resize_status"].GetMetric()); got != 1 {
		t.Fatalf("resize_status has %d series after the first pass, want 1", got)
	}

	// The resize completes: the claim is now current, so it produces no finding.
	m.RecordResize(resize.Report{Node: "node-1", At: time.Now(), Current: 4})
	// A *Vec with no children is omitted from the exposition entirely, so the
	// family being absent is the passing answer; present-but-empty also passes.
	if f, ok := gather(t, m)["pv_sentinel_resize_status"]; ok && len(f.GetMetric()) != 0 {
		t.Errorf("resize_status has %d series after the resize completed, want 0", len(f.GetMetric()))
	}
	if got := gaugeValue(t, m, "pv_sentinel_stuck_expansions"); got != 0 {
		t.Errorf("stuck_expansions = %v, want 0", got)
	}
	// The mount gauge must be untouched by the resize pass.
	if got := gaugeValue(t, m, "pv_sentinel_volumes_discovered"); got != 4 {
		t.Errorf("volumes_discovered = %v, want 4 (the resize pass must not age out mount series)", got)
	}
}

func TestRecordResizeResult(t *testing.T) {
	m := New()
	for _, action := range []string{
		remediate.ActionDryRun, remediate.ActionDryRun,
		remediate.ActionRescanned, remediate.ActionSkippedNotCSI,
	} {
		m.RecordResizeResult(remediate.RescanResult{Action: action})
	}
	got := counterValues(t, m, "pv_sentinel_resize_remediations_total")
	want := map[string]float64{
		remediate.ActionDryRun:        2,
		remediate.ActionRescanned:     1,
		remediate.ActionSkippedNotCSI: 1,
	}
	for action, w := range want {
		if got[action] != w {
			t.Errorf("resize_remediations_total{action=%q} = %v, want %v", action, got[action], w)
		}
	}
}

func TestSummary(t *testing.T) {
	rep := sampleReport()
	want := "healthy=2 read_only=1 unknown=1 attached_not_mounted=2"
	if got := Summary(rep); got != want {
		t.Errorf("Summary() = %q, want %q", got, want)
	}

	empty := Summary(probe.Report{})
	if empty != "healthy=0 read_only=0 unknown=0 attached_not_mounted=0" {
		t.Errorf("Summary(empty) = %q", empty)
	}
}

func TestHandler(t *testing.T) {
	m := New()
	m.SetNode("node-1")
	m.RecordReport(sampleReport())
	// Seed a remediation too: prometheus omits a *Vec with zero children from
	// the exposition, so without this the remediations counter is absent from
	// the body and the assertion below would be checking a metric that was
	// never emitted.
	m.RecordResult(remediate.Result{Action: remediate.ActionNone})
	m.RecordResize(sampleResizeReport())
	m.RecordResizeResult(remediate.RescanResult{Action: remediate.ActionDryRun})

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	m.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /metrics = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, name := range registered {
		if !strings.Contains(body, name) {
			t.Errorf("/metrics output is missing %s", name)
		}
	}
}
