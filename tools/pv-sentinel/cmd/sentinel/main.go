// Command sentinel is the pv-sentinel agent: one privileged-capable pod per
// node that write-probes every CSI volume mounted on that node, reports the ones
// that return EROFS despite advertising rw, and (in remediate mode) deletes the
// owning pod to force CSI to detach and reattach the volume.
//
// The same cycle runs a second detector for stuck PVC volume expansions
// (internal/resize), remediated on iSCSI volumes by rescanning the session so
// the kernel re-reads the LUN size. Each detector's mode is gated separately —
// MODE plus the MODE_RO_MOUNT / MODE_RESIZE overrides — because deleting a pod
// and rescanning a session are not remotely the same kind of action.
//
// Configuration is environment-only — no flag package — matching the house
// convention. The manifest injects NODE_NAME via the downward API.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"pv-sentinel/internal/k8s"
	"pv-sentinel/internal/metrics"
	"pv-sentinel/internal/probe"
	"pv-sentinel/internal/remediate"
	"pv-sentinel/internal/resize"
)

// config is the full environment contract for the binary.
type config struct {
	Port         string
	LogLevel     slog.Level
	Node         string
	MountsPath   string
	FSTypes      []string
	RunOnce      bool
	RunOnceGrace time.Duration
	Interval     time.Duration
	// ResizeThreshold is how long a PVC may advertise a requested size larger
	// than its reported capacity before the expansion counts as stuck. See
	// internal/resize: a resize that is progressing resolves in seconds to
	// minutes, so a gap that has not lasted this long is in flight, not stuck.
	ResizeThreshold time.Duration
	Gate            remediate.Gate
}

// envOr returns the value of key, or def when unset or empty.
func envOr(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

// envInt reads an integer env var, applying def when unset.
func envInt(key string, def int) (int, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return def, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s must be a valid integer, got %q: %w", key, raw, err)
	}
	return n, nil
}

// envDuration reads a Go duration env var, applying def when unset.
func envDuration(key string, def time.Duration) (time.Duration, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return def, nil
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("%s must be a duration like 300s, got %q: %w", key, raw, err)
	}
	return d, nil
}

// envBool reads a boolean env var, applying def when unset.
func envBool(key string, def bool) (bool, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return def, nil
	}
	b, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("%s must be a bool, got %q: %w", key, raw, err)
	}
	return b, nil
}

// resolveNode picks the node this instance reports on. The manifest injects
// NODE_NAME via the downward API; NODE is the manual override; the hostname is
// the fallback so a local run still labels its output correctly.
//
// It takes the env getter as a parameter rather than reading os.Getenv directly
// so the precedence is testable without mutating process state.
func resolveNode(getenv func(string) string) (string, error) {
	if v := strings.TrimSpace(getenv("NODE_NAME")); v != "" {
		return v, nil
	}
	if v := strings.TrimSpace(getenv("NODE")); v != "" {
		return v, nil
	}
	hn, err := os.Hostname()
	if err != nil {
		return "", fmt.Errorf("cannot determine node: set NODE_NAME or NODE, hostname failed: %w", err)
	}
	return hn, nil
}

// parseLogLevel maps the LOG_LEVEL env value onto a slog level, defaulting to
// info for anything unrecognised rather than refusing to start.
func parseLogLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// loadConfig assembles and validates the config from the environment.
func loadConfig() (config, error) {
	var cfg config

	port, err := envInt("PORT", 8080)
	if err != nil {
		return cfg, err
	}
	if port < 1 || port > 65535 {
		return cfg, fmt.Errorf("PORT must be between 1 and 65535, got %d", port)
	}
	cfg.Port = strconv.Itoa(port)
	cfg.LogLevel = parseLogLevel(envOr("LOG_LEVEL", "info"))

	node, err := resolveNode(os.Getenv)
	if err != nil {
		return cfg, err
	}
	cfg.Node = node

	cfg.MountsPath = envOr("MOUNTS_PATH", "/proc/mounts")

	fsTypes := strings.Split(envOr("FS_TYPES", strings.Join(probe.DefaultFSTypes, ",")), ",")
	for i := range fsTypes {
		fsTypes[i] = strings.ToLower(strings.TrimSpace(fsTypes[i]))
	}
	cfg.FSTypes = fsTypes

	runOnce, err := envBool("RUN_ONCE", false)
	if err != nil {
		return cfg, err
	}
	cfg.RunOnce = runOnce

	if cfg.RunOnceGrace, err = envDuration("RUN_ONCE_GRACE", 10*time.Second); err != nil {
		return cfg, err
	}
	if cfg.Interval, err = envDuration("INTERVAL", 300*time.Second); err != nil {
		return cfg, err
	}
	if cfg.Interval <= 0 {
		return cfg, errors.New("INTERVAL must be > 0")
	}

	// 10m is the default because that is the shape of a real resize: the
	// controller and the node complete a genuine expansion in seconds to
	// minutes, so a gap still open after ten minutes is not progressing. A
	// threshold of 0 would call every in-flight resize stuck and make the
	// metric noise, which is what the threshold exists to prevent.
	if cfg.ResizeThreshold, err = envDuration("RESIZE_THRESHOLD", 10*time.Minute); err != nil {
		return cfg, err
	}
	if cfg.ResizeThreshold <= 0 {
		return cfg, errors.New("RESIZE_THRESHOLD must be > 0")
	}

	gate, err := gateFromEnv(gateDefaults())
	if err != nil {
		return cfg, err
	}
	cfg.Gate = gate

	return cfg, nil
}

// gateDefaults is the starting gate before env overrides are applied.
func gateDefaults() remediate.Gate { return remediate.DefaultGate() }

// gateFromEnv layers the REMEDIATE_* environment onto a base gate and validates
// the result. Split out from loadConfig so the gate rules are testable in
// isolation, and so a bad COOLDOWN is reported as a bad COOLDOWN rather than as
// a generic config failure.
func gateFromEnv(base remediate.Gate) (remediate.Gate, error) {
	g := base
	g.Mode = envOr("MODE", remediate.DefaultMode)

	// Per-detector overrides. Both default to empty, which means "inherit MODE"
	// — so a manifest written before the resize detector existed keeps behaving
	// exactly as it did. Setting one without the other is how an operator enables
	// the non-disruptive fix (a session rescan) while pod deletion stays in
	// report; see Gate.ModeFor.
	g.RoMountMode = envOr("MODE_RO_MOUNT", "")
	g.ResizeMode = envOr("MODE_RESIZE", "")

	// The binary path is config, not a $PATH lookup, so which iscsiadm runs is
	// decided by the manifest rather than by whatever an image layer left behind.
	g.ISCSIADMPath = envOr("ISCSIADM_PATH", remediate.DefaultISCSIADMPath)

	allowBare, err := envBool("ALLOW_BARE_PODS", false)
	if err != nil {
		return g, err
	}
	g.AllowBarePods = allowBare

	// Both window defaults here said 60*time.Hour while the README documented 60m
	// for both. The doc is the correct side: every other row in that table matches
	// the code exactly (INTERVAL 300s, MAX_PER_NODE 3, GRACE_PERIOD 30, POLL_TIMEOUT
	// 180s, POLL_INTERVAL 5s), so the two that disagreed are the two that shared one
	// suspicious literal — a Hour-for-Minute slip, not two independent doc typos. A
	// 60h cooldown on a process that scans every 300s also means a volume can be
	// acted on once every two and a half days.
	if g.Cooldown, err = envDuration("COOLDOWN", 60*time.Minute); err != nil {
		return g, err
	}
	maxPerNode, err := envInt("MAX_PER_NODE", 3)
	if err != nil {
		return g, err
	}
	g.MaxPerNode = maxPerNode

	if g.NodeWindow, err = envDuration("NODE_WINDOW", 60*time.Minute); err != nil {
		return g, err
	}
	grace, err := envInt("GRACE_PERIOD", 30)
	if err != nil {
		return g, err
	}
	g.GracePeriod = int64(grace)

	if g.PollTimeout, err = envDuration("POLL_TIMEOUT", 180*time.Second); err != nil {
		return g, err
	}
	if g.PollInterval, err = envDuration("POLL_INTERVAL", 5*time.Second); err != nil {
		return g, err
	}

	if err := g.Validate(); err != nil {
		return g, err
	}
	return g, nil
}

// scanOnce performs one detect-and-maybe-remediate cycle and publishes it.
//
// Both detectors run in the same cycle over the same `now`: the mount detector
// and the resize detector describe the same node at the same instant, and the
// freeidp case is exactly a pair of findings that belong to one volume — the
// mount that failed and the expansion that never finished. Splitting them onto
// separate tickers would let the two halves of one incident disagree.
func scanOnce(
	ctx context.Context,
	cfg config,
	cl *k8s.Client,
	rem *remediate.Remediator,
	det *resize.Detector,
	resc *remediate.Rescanner,
	m *metrics.Metrics,
	log *slog.Logger,
) (probe.Report, error) {
	content, err := probe.ReadMounts(cfg.MountsPath)
	if err != nil {
		return probe.Report{}, fmt.Errorf("read mount table %s: %w", cfg.MountsPath, err)
	}

	now := time.Now()
	vols := probe.Discover(content, cfg.FSTypes)
	rep := probe.NewReport(cfg.Node, probe.ProbeAll(vols, now), now)

	// Cross-check the API's attachment list. A PV the API says is attached to
	// this node but that no mount entry covers is a distinct failure from
	// read-only — the volume never got mounted at all.
	if cl != nil {
		ctx, cancel := k8s.WithTimeout()
		defer cancel()
		if vas, err := cl.VolumeAttachmentsOnNode(ctx); err == nil {
			mounted := make(map[string]bool, len(rep.MountedPV))
			for _, pv := range rep.MountedPV {
				mounted[pv] = true
			}
			for _, va := range vas {
				// PersistentVolumeName is a *string in the storage/v1 API and is
				// nil when the attachment references an inline ephemeral volume.
				if va.Spec.Source.PersistentVolumeName == nil {
					continue
				}
				pv := *va.Spec.Source.PersistentVolumeName
				if pv != "" && !mounted[pv] {
					rep.AttachedNotMounted = append(rep.AttachedNotMounted, pv)
				}
			}
		} else {
			log.Warn("volume attachment cross-check unavailable", "error", err)
		}
	}

	m.SetNode(cfg.Node)
	m.RecordReport(rep)
	log.Info("scan complete", "node", cfg.Node, "summary", metrics.Summary(rep))

	for _, res := range rem.HandleAll(ctx, rep, now, probe.Probe) {
		if res.Action == remediate.ActionNone {
			continue // healthy volumes are already covered by the summary line
		}
		m.RecordResult(res)
		log.Info("remediation",
			"pv", res.PVName,
			"pod", res.Pod,
			"action", res.Action,
			"before", res.Before,
			"after", res.After,
			"detail", res.Detail,
		)
	}

	// The second detector. A failure here is logged rather than returned: the
	// mount report above is already published, and a PVC list that will not
	// answer — RBAC too narrow, or no API access at all — must not turn a
	// working agent's scan into an error. The metric simply stops advancing,
	// which is the honest signal.
	if det != nil && resc != nil {
		rrep, err := det.Detect(ctx, now)
		if err != nil {
			log.Warn("resize detection unavailable", "error", err)
		} else {
			m.RecordResize(rrep)
			log.Info("resize scan complete", "node", cfg.Node, "summary", resize.Summary(rrep))
			for _, res := range resc.HandleAll(ctx, rrep.Findings, now) {
				if res.Action == remediate.ActionNone {
					continue
				}
				m.RecordResizeResult(res)
				log.Info("resize remediation",
					"pvc", res.PVC,
					"pv", res.PVName,
					"action", res.Action,
					"detail", res.Detail,
				)
			}
		}
	}

	return rep, nil
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		slog.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	slog.SetDefault(log)

	m := metrics.New()
	m.SetNode(cfg.Node)

	cl, err := k8s.NewFromEnv(cfg.Node)
	if err != nil {
		log.Warn("kubernetes API unavailable, reporting continues", "error", err)
	}
	rem := remediate.New(cl, cfg.Gate, log)

	// The resize detector needs a client to list anything, so it exists only when
	// the API does; Remediator is built unconditionally because it degrades to
	// reporting on its own. The rescanner takes the same gate, from which it reads
	// MODE_RESIZE.
	var det *resize.Detector
	if cl != nil {
		det = resize.New(cl, cfg.ResizeThreshold)
	}
	resc := remediate.NewRescanner(cfg.Gate, log)

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {
			log.Error("failed to write health response", "error", err)
		}
	})
	mux.Handle("/metrics", m.Handler())

	server := &http.Server{
		Addr:           ":" + cfg.Port,
		Handler:        mux,
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   15 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1024,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("http server stopped", "error", err)
		}
	}()

	if cfg.RunOnce {
		// One-shot audit mode: scan, log, then hold the port open briefly so a
		// scraper can read /metrics before the process exits.
		if _, err := scanOnce(ctx, cfg, cl, rem, det, resc, m, log); err != nil {
			log.Error("scan failed", "error", err)
			os.Exit(1)
		}
		time.Sleep(cfg.RunOnceGrace)
		return
	}

	go func() {
		ticker := time.NewTicker(cfg.Interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if _, err := scanOnce(ctx, cfg, cl, rem, det, resc, m, log); err != nil {
					log.Error("scan failed", "error", err)
				}
			}
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit
	log.Info("shutting down")

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelShutdown()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error("server forced to shutdown", "error", err)
		return
	}
}
