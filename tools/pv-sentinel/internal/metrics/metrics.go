// Package metrics exposes the Prometheus surface for a pv-sentinel pod.
package metrics

import (
	"net/http"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"pv-sentinel/internal/probe"
	"pv-sentinel/internal/remediate"
	"pv-sentinel/internal/resize"
)

// namespace prefixes every metric this binary exports.
const namespace = "pv_sentinel"

// Metrics is the Prometheus surface for one pv-sentinel pod.
type Metrics struct {
	registry *prometheus.Registry

	// node and volumes remember the last published per-volume series set so it
	// can be aged out on the next scan; the registry offers no way to enumerate
	// a GaugeVec's children.
	node    string
	volumes []string

	// resizePVCs is the same cache for the second detector: the resize gauge is
	// keyed on a PVC, not a PV, so it ages out separately. See RecordResize.
	resizePVCs []string

	volumeStatus       *prometheus.GaugeVec
	probesTotal        *prometheus.CounterVec
	remediationsTotal  *prometheus.CounterVec
	healthyVolumes     prometheus.Gauge
	readOnlyVolumes    prometheus.Gauge
	unknownVolumes     prometheus.Gauge
	attachedNotMounted prometheus.Gauge
	volumesDiscovered  prometheus.Gauge
	lastScan           prometheus.Gauge

	resizeStatus            *prometheus.GaugeVec
	stuckExpansions         prometheus.Gauge
	resizeLastScan          prometheus.Gauge
	resizeRemediationsTotal *prometheus.CounterVec
}

// New builds and registers the full metric set on a private registry, so tests
// and multiple instances never collide with the default registry.
func New() *Metrics {
	m := &Metrics{
		registry: prometheus.NewRegistry(),

		volumeStatus: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "volume_status",
			Help: "Observed status of one volume on this node; always 1, the labels carry the state. " +
				"A volume whose labels say read_only while /proc/mounts says rw is the case this tool exists to catch.",
		}, []string{"node", "pv", "fs", "status"}),

		probesTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "probes_total",
			Help:      "Write probes performed, by result.",
		}, []string{"status"}),

		remediationsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "remediations_total",
			Help:      "Remediation decisions taken, by action.",
		}, []string{"action"}),

		healthyVolumes: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "healthy_volumes",
			Help:      "Volumes on this node that accepted a write probe.",
		}),
		readOnlyVolumes: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "read_only_volumes",
			Help:      "Volumes on this node that returned EROFS despite their mount options.",
		}),
		unknownVolumes: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "unknown_volumes",
			Help:      "Volumes on this node whose probe failed for a reason other than EROFS.",
		}),
		attachedNotMounted: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "attached_not_mounted_volumes",
			Help:      "PVs the API says are attached to this node but that no mount table entry covers.",
		}),
		volumesDiscovered: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "volumes_discovered",
			Help:      "Distinct CSI volumes discovered on this node, after deduplication.",
		}),
		lastScan: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "last_scan_timestamp_seconds",
			Help:      "Unix time of the most recent completed scan.",
		}),

		// The second detector gets its own family rather than more labels on
		// volume_status: a PVC whose expansion is stuck is not a volume whose
		// write probe failed, and the two detectors disagree about which object
		// is even being described (a PV mount vs a PVC claim). Sharing one family
		// would mean a status label set that means different things per row.
		//
		// There is no `state` label because only stuck claims get a series: the
		// presence of the series is the signal, and the state of every other claim
		// is a count on stuck_expansions' sibling gauges. A state label here would
		// be a label that can only ever hold one value — cardinality for nothing.
		resizeStatus: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "resize_status",
			Help: "One series per PVC on this node whose expansion is stuck; always 1, so the series existing is the alert. " +
				"The pv label lets a stuck claim be joined to the volume_status row for the same volume.",
		}, []string{"node", "pv", "pvc"}),

		stuckExpansions: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "stuck_expansions",
			Help:      "PVCs on this node whose requested size exceeds the reported capacity past RESIZE_THRESHOLD.",
		}),
		resizeLastScan: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "resize_last_scan_timestamp_seconds",
			Help:      "Unix time of the most recent completed resize scan.",
		}),
		resizeRemediationsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "resize_remediations_total",
			Help:      "Resize remediation decisions taken, by action.",
		}, []string{"action"}),
	}

	m.registry.MustRegister(
		m.volumeStatus, m.probesTotal, m.remediationsTotal,
		m.healthyVolumes, m.readOnlyVolumes, m.unknownVolumes,
		m.attachedNotMounted, m.volumesDiscovered, m.lastScan,
		m.resizeStatus, m.stuckExpansions, m.resizeLastScan, m.resizeRemediationsTotal,
	)
	return m
}

// Handler serves the registry at /metrics.
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

// RecordReport folds one node scan into the gauges and probe counters.
func (m *Metrics) RecordReport(r probe.Report) {
	node := r.Node

	// Drop the previous per-volume series so a volume that disappeared stops
	// being reported. Without this, deleted volumes linger forever.
	for _, prev := range m.cachedVolumes(node) {
		m.volumeStatus.DeletePartialMatch(prometheus.Labels{"node": node, "pv": prev})
	}
	m.volumes = append(m.volumes[:0], r.MountedPV...)

	m.volumesDiscovered.Set(float64(len(r.Volumes)))
	m.healthyVolumes.Set(float64(r.Healthy))
	m.readOnlyVolumes.Set(float64(r.ReadOnly))
	m.unknownVolumes.Set(float64(r.Unknown))
	m.attachedNotMounted.Set(float64(len(r.AttachedNotMounted)))
	m.lastScan.Set(float64(r.At.Unix()))

	for _, v := range r.Volumes {
		m.volumeStatus.WithLabelValues(node, v.PVName, v.FSType, v.Status).Set(1)
		m.probesTotal.WithLabelValues(v.Status).Inc()
	}
}

// RecordResult counts one remediation decision.
func (m *Metrics) RecordResult(res remediate.Result) {
	m.remediationsTotal.WithLabelValues(res.Action).Inc()
}

// RecordResize folds one resize-detector pass into the resize gauges.
//
// The series aging mirrors RecordReport for the same reason: a claim that stops
// being stuck — the resize completed, the PVC went away — must stop being
// reported, or the metric keeps an alert firing for a volume that is fine now.
// The cache is separate from `volumes` because the two detectors key on
// different objects, a PV mount and a PVC claim, and one node has both.
func (m *Metrics) RecordResize(r resize.Report) {
	node := r.Node

	for _, prev := range m.cachedResizePVCs(node) {
		m.resizeStatus.DeletePartialMatch(prometheus.Labels{"node": node, "pvc": prev})
	}
	m.resizePVCs = m.resizePVCs[:0]
	for _, f := range r.Findings {
		m.resizePVCs = append(m.resizePVCs, f.PVCName)
	}

	m.stuckExpansions.Set(float64(r.Stuck))
	m.resizeLastScan.Set(float64(r.At.Unix()))
	for _, f := range r.Findings {
		m.resizeStatus.WithLabelValues(node, f.PVName, f.PVCName).Set(1)
	}
}

// RecordResizeResult counts one resize remediation decision.
func (m *Metrics) RecordResizeResult(res remediate.RescanResult) {
	m.resizeRemediationsTotal.WithLabelValues(res.Action).Inc()
}

// cachedVolumes is the small amount of state needed to age out per-volume
// series; the registry itself offers no way to enumerate a GaugeVec's children.
func (m *Metrics) cachedVolumes(node string) []string {
	if m.node != node || len(m.volumes) == 0 {
		return nil
	}
	return m.volumes
}

// cachedResizePVCs is the same thing for the resize gauge's pvc label.
func (m *Metrics) cachedResizePVCs(node string) []string {
	if m.node != node || len(m.resizePVCs) == 0 {
		return nil
	}
	return m.resizePVCs
}

// SetNode records which node this instance serves, used to validate that a
// report belongs to the cached series set before aging it out.
func (m *Metrics) SetNode(node string) { m.node = node }

// Summary renders the one-line human-readable scan summary that main logs.
func Summary(r probe.Report) string {
	return "healthy=" + strconv.Itoa(r.Healthy) +
		" read_only=" + strconv.Itoa(r.ReadOnly) +
		" unknown=" + strconv.Itoa(r.Unknown) +
		" attached_not_mounted=" + strconv.Itoa(len(r.AttachedNotMounted))
}
