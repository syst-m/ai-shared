package main

import (
	"os"
	"strings"
	"testing"
	"time"

	"pv-sentinel/internal/probe"
	"pv-sentinel/internal/remediate"
)

// envFromMap returns a getenv func backed by m, so precedence rules can be
// tested without mutating the real process environment.
func envFromMap(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestEnvOr(t *testing.T) {
	t.Setenv("ENVOR_SET", "  spaced  ")
	t.Setenv("ENVOR_BLANK", "   ")

	tests := []struct {
		key, want string
	}{
		{"ENVOR_SET", "spaced"}, // trimmed, not passed through verbatim
		{"ENVOR_BLANK", "fallback"},
		{"ENVOR_UNSET", "fallback"},
	}
	for _, tt := range tests {
		if got := envOr(tt.key, "fallback"); got != tt.want {
			t.Errorf("envOr(%q) = %q, want %q", tt.key, got, tt.want)
		}
	}
	if got := envOr("ENVOR_UNSET", ""); got != "" {
		t.Errorf("envOr with empty default = %q, want empty", got)
	}
}

func TestEnvInt(t *testing.T) {
	t.Setenv("EI_SET", " 4321 ")
	t.Setenv("EI_BLANK", "  ")
	t.Setenv("EI_BAD", "not-a-number")
	t.Setenv("EI_FLOAT", "1.5")

	tests := []struct {
		name    string
		key     string
		def     int
		want    int
		wantErr bool
	}{
		{"set", "EI_SET", 8080, 4321, false},
		{"blank falls back", "EI_BLANK", 8080, 8080, false},
		{"unset falls back", "EI_UNSET", 8080, 8080, false},
		{"non-numeric errors", "EI_BAD", 8080, 0, true},
		{"float errors", "EI_FLOAT", 8080, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := envInt(tt.key, tt.def)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.key) {
				t.Errorf("error %q must name the offending variable", err.Error())
			}
		})
	}
}

func TestEnvDuration(t *testing.T) {
	t.Setenv("ED_SET", "90s")
	t.Setenv("ED_BAD", "90") // a bare number is not a Go duration

	tests := []struct {
		name    string
		key     string
		def     time.Duration
		want    time.Duration
		wantErr bool
	}{
		{"set", "ED_SET", 300 * time.Second, 90 * time.Second, false},
		{"unset falls back", "ED_UNSET", 300 * time.Second, 300 * time.Second, false},
		{"unitless errors", "ED_BAD", time.Second, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := envDuration(tt.key, tt.def)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.key) {
				t.Errorf("error %q must name the offending variable", err.Error())
			}
		})
	}
}

func TestEnvBool(t *testing.T) {
	t.Setenv("EB_TRUE", "true")
	t.Setenv("EB_ONE", "1")
	t.Setenv("EB_BAD", "yes") // strconv.ParseBool rejects "yes"

	tests := []struct {
		name    string
		key     string
		def     bool
		want    bool
		wantErr bool
	}{
		{"true", "EB_TRUE", false, true, false},
		{"one", "EB_ONE", false, true, false},
		{"unset falls back", "EB_UNSET", true, true, false},
		{"yes errors", "EB_BAD", false, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := envBool(tt.key, tt.def)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResolveNode(t *testing.T) {
	hn, err := os.Hostname()
	if err != nil {
		t.Skipf("hostname unavailable: %v", err)
	}

	tests := []struct {
		name string
		env  map[string]string
		want string
	}{
		{"NODE_NAME wins over NODE", map[string]string{"NODE_NAME": "from-downward-api", "NODE": "manual"}, "from-downward-api"},
		{"NODE used when NODE_NAME unset", map[string]string{"NODE": "manual"}, "manual"},
		{"NODE_NAME blank falls through to NODE", map[string]string{"NODE_NAME": "  ", "NODE": "manual"}, "manual"},
		{"both blank falls through to hostname", map[string]string{"NODE_NAME": "", "NODE": ""}, hn},
		{"nothing set uses hostname", map[string]string{}, hn},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveNode(envFromMap(tt.env))
			if err != nil {
				t.Fatalf("resolveNode: %v", err)
			}
			if got != tt.want {
				t.Errorf("resolveNode() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"debug", "DEBUG"},
		{"DEBUG", "DEBUG"},
		{"  debug  ", "DEBUG"},
		{"info", "INFO"},
		{"warn", "WARN"},
		{"warning", "WARN"},
		{"error", "ERROR"},
		{"", "INFO"},
		{"nonsense", "INFO"}, // unrecognised must not stop the process from starting
	}
	for _, tt := range tests {
		if got := parseLogLevel(tt.in).String(); got != tt.want {
			t.Errorf("parseLogLevel(%q) = %s, want %s", tt.in, got, tt.want)
		}
	}
}

func TestGateDefaultsMatchPackage(t *testing.T) {
	if got := gateDefaults(); got != remediate.DefaultGate() {
		t.Errorf("gateDefaults() = %+v, want %+v", got, remediate.DefaultGate())
	}
}

func TestGateFromEnv_Defaults(t *testing.T) {
	g, err := gateFromEnv(remediate.DefaultGate())
	if err != nil {
		t.Fatalf("gateFromEnv: %v", err)
	}
	if g.Mode != remediate.ModeReport {
		t.Errorf("Mode = %q, want %q", g.Mode, remediate.ModeReport)
	}
	if g.AllowBarePods {
		t.Error("AllowBarePods must default to false")
	}
	if g.MaxPerNode != 3 {
		t.Errorf("MaxPerNode = %d, want 3", g.MaxPerNode)
	}
	if g.GracePeriod != 30 {
		t.Errorf("GracePeriod = %d, want 30", g.GracePeriod)
	}
	if g.PollTimeout != 180*time.Second {
		t.Errorf("PollTimeout = %v, want 180s", g.PollTimeout)
	}
	if g.PollInterval != 5*time.Second {
		t.Errorf("PollInterval = %v, want 5s", g.PollInterval)
	}
	// Unset overrides mean "inherit MODE", which is the pre-split behaviour; an
	// empty string is the value that expresses it, not a missing field.
	if g.RoMountMode != "" || g.ResizeMode != "" {
		t.Errorf("per-detector modes = %q/%q, want both empty (inherit MODE)", g.RoMountMode, g.ResizeMode)
	}
	if g.ISCSIADMPath != remediate.DefaultISCSIADMPath {
		t.Errorf("ISCSIADMPath = %q, want %q", g.ISCSIADMPath, remediate.DefaultISCSIADMPath)
	}
}

func TestGateFromEnv_Overrides(t *testing.T) {
	t.Setenv("MODE", "remediate")
	t.Setenv("MODE_RO_MOUNT", "report")
	t.Setenv("MODE_RESIZE", "remediate")
	t.Setenv("ISCSIADM_PATH", "/sbin/iscsiadm")
	t.Setenv("ALLOW_BARE_PODS", "true")
	t.Setenv("COOLDOWN", "90m")
	t.Setenv("MAX_PER_NODE", "7")
	t.Setenv("NODE_WINDOW", "12h")
	t.Setenv("GRACE_PERIOD", "5")
	t.Setenv("POLL_TIMEOUT", "45s")
	t.Setenv("POLL_INTERVAL", "2s")

	g, err := gateFromEnv(remediate.DefaultGate())
	if err != nil {
		t.Fatalf("gateFromEnv: %v", err)
	}
	want := remediate.Gate{
		Mode:          remediate.ModeRemediate,
		RoMountMode:   remediate.ModeReport,
		ResizeMode:    remediate.ModeRemediate,
		ISCSIADMPath:  "/sbin/iscsiadm",
		AllowBarePods: true,
		Cooldown:      90 * time.Minute,
		MaxPerNode:    7,
		NodeWindow:    12 * time.Hour,
		GracePeriod:   5,
		PollTimeout:   45 * time.Second,
		PollInterval:  2 * time.Second,
	}
	if g != want {
		t.Errorf("gateFromEnv() =\n%+v\nwant\n%+v", g, want)
	}
	// The two overrides have to resolve independently, or the split is cosmetic.
	if g.ModeFor(remediate.DetectorROMount) != remediate.ModeReport {
		t.Errorf("ModeFor(ro_mount) = %q, want %q", g.ModeFor(remediate.DetectorROMount), remediate.ModeReport)
	}
	if g.ModeFor(remediate.DetectorResize) != remediate.ModeRemediate {
		t.Errorf("ModeFor(resize) = %q, want %q", g.ModeFor(remediate.DetectorResize), remediate.ModeRemediate)
	}
}

func TestGateFromEnv_RejectsInvalidMode(t *testing.T) {
	t.Setenv("MODE", "yolo")
	if _, err := gateFromEnv(remediate.DefaultGate()); err == nil {
		t.Fatal("gateFromEnv should reject an unknown MODE")
	}
}

// TestGateFromEnv_RejectsInvalidOverrides covers the two per-detector modes.
// An empty override is legal ("inherit MODE"), so this is the only way to learn
// whether gateFromEnv validates them at all rather than accepting any string.
func TestGateFromEnv_RejectsInvalidOverrides(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		val     string
		wantErr bool
	}{
		{"MODE_RO_MOUNT not a mode", "MODE_RO_MOUNT", "fix", true},
		{"MODE_RESIZE not a mode", "MODE_RESIZE", "fix", true},
		// Empty is the documented way to ask for "inherit MODE", so it must not
		// be rejected — that is what keeps an existing manifest working.
		{"MODE_RO_MOUNT empty is legal", "MODE_RO_MOUNT", "", false},
		{"MODE_RESIZE empty is legal", "MODE_RESIZE", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("MODE", remediate.ModeReport)
			t.Setenv(tt.key, tt.val)
			g, err := gateFromEnv(remediate.DefaultGate())
			if tt.wantErr {
				if err == nil {
					t.Fatalf("gateFromEnv() with %s=%s should fail", tt.key, tt.val)
				}
				if !strings.Contains(err.Error(), tt.key) {
					t.Errorf("error %q must name %s", err.Error(), tt.key)
				}
				return
			}
			if err != nil {
				t.Fatalf("gateFromEnv() with %s=%q: %v", tt.key, tt.val, err)
			}
			if g.ModeFor(remediate.DetectorROMount) != remediate.ModeReport {
				t.Errorf("ModeFor(ro_mount) = %q, want it to inherit report", g.ModeFor(remediate.DetectorROMount))
			}
		})
	}
}

func TestGateFromEnv_ReportsBadCooldownByName(t *testing.T) {
	t.Setenv("COOLDOWN", "soon")
	_, err := gateFromEnv(remediate.DefaultGate())
	if err == nil {
		t.Fatal("gateFromEnv should reject a non-duration COOLDOWN")
	}
	if !strings.Contains(err.Error(), "COOLDOWN") {
		t.Errorf("error %q must name COOLDOWN, not just say the config is bad", err.Error())
	}
}

func TestLoadConfig_Defaults(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("NODE_NAME", "node-7")
	t.Setenv("RUN_ONCE", "")
	t.Setenv("MODE", "")

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want 8080", cfg.Port)
	}
	if cfg.Node != "node-7" {
		t.Errorf("Node = %q, want node-7", cfg.Node)
	}
	if cfg.MountsPath != "/proc/mounts" {
		t.Errorf("MountsPath = %q, want /proc/mounts", cfg.MountsPath)
	}
	if strings.Join(cfg.FSTypes, ",") != strings.Join(probe.DefaultFSTypes, ",") {
		t.Errorf("FSTypes = %v, want %v", cfg.FSTypes, probe.DefaultFSTypes)
	}
	if cfg.RunOnce {
		t.Error("RunOnce must default to false")
	}
	if cfg.RunOnceGrace != 10*time.Second {
		t.Errorf("RunOnceGrace = %v, want 10s", cfg.RunOnceGrace)
	}
	if cfg.Interval != 300*time.Second {
		t.Errorf("Interval = %v, want 300s", cfg.Interval)
	}
	if cfg.ResizeThreshold != 10*time.Minute {
		t.Errorf("ResizeThreshold = %v, want 10m", cfg.ResizeThreshold)
	}
	if cfg.Gate.Mode != remediate.ModeReport {
		t.Errorf("Gate.Mode = %q, want %q", cfg.Gate.Mode, remediate.ModeReport)
	}
}

func TestLoadConfig_All(t *testing.T) {
	t.Setenv("PORT", "9999")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("NODE", "manual-node")
	t.Setenv("MOUNTS_PATH", "/tmp/fake-mounts")
	t.Setenv("FS_TYPES", "EXT4, xfs ")
	t.Setenv("RUN_ONCE", "true")
	t.Setenv("RUN_ONCE_GRACE", "1s")
	t.Setenv("INTERVAL", "45s")
	t.Setenv("MODE", "remediate")
	t.Setenv("MODE_RESIZE", "report")
	t.Setenv("RESIZE_THRESHOLD", "25m")

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if cfg.Port != "9999" {
		t.Errorf("Port = %q, want 9999", cfg.Port)
	}
	if cfg.LogLevel.String() != "DEBUG" {
		t.Errorf("LogLevel = %s, want DEBUG", cfg.LogLevel)
	}
	// NODE_NAME is unset here, so NODE is the next in precedence.
	if cfg.Node != "manual-node" {
		t.Errorf("Node = %q, want manual-node", cfg.Node)
	}
	if cfg.MountsPath != "/tmp/fake-mounts" {
		t.Errorf("MountsPath = %q, want /tmp/fake-mounts", cfg.MountsPath)
	}
	// Case and padding are normalised so "EXT4, xfs " matches the parser's keys.
	if got := strings.Join(cfg.FSTypes, ","); got != "ext4,xfs" {
		t.Errorf("FSTypes = %q, want ext4,xfs", got)
	}
	if !cfg.RunOnce {
		t.Error("RunOnce = false, want true")
	}
	if cfg.Interval != 45*time.Second {
		t.Errorf("Interval = %v, want 45s", cfg.Interval)
	}
	if cfg.ResizeThreshold != 25*time.Minute {
		t.Errorf("ResizeThreshold = %v, want 25m", cfg.ResizeThreshold)
	}
	if cfg.Gate.Mode != remediate.ModeRemediate {
		t.Errorf("Gate.Mode = %q, want %q", cfg.Gate.Mode, remediate.ModeRemediate)
	}
	// The override has to survive the whole loadConfig path, not just
	// gateFromEnv, or the manifest's per-detector setting is silently dropped.
	if cfg.Gate.ResizeMode != remediate.ModeReport {
		t.Errorf("Gate.ResizeMode = %q, want %q", cfg.Gate.ResizeMode, remediate.ModeReport)
	}
}

func TestLoadConfig_Errors(t *testing.T) {
	tests := []struct {
		name string
		key  string
		val  string
	}{
		{"port not a number", "PORT", "http"},
		{"port zero", "PORT", "0"},
		{"port too high", "PORT", "70000"},
		{"negative interval", "INTERVAL", "-5s"},
		{"zero interval", "INTERVAL", "0s"},
		{"bad interval", "INTERVAL", "5"},
		{"bad mode", "MODE", "yolo"},
		{"bad run_once", "RUN_ONCE", "sure"},
		{"bad grace", "RUN_ONCE_GRACE", "10"},
		{"bad resize threshold", "RESIZE_THRESHOLD", "10"},
		{"negative resize threshold", "RESIZE_THRESHOLD", "-5m"},
		{"zero resize threshold", "RESIZE_THRESHOLD", "0s"},
		{"bad resize mode", "MODE_RESIZE", "yolo"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("NODE_NAME", "node-1")
			t.Setenv(tt.key, tt.val)
			if _, err := loadConfig(); err == nil {
				t.Fatalf("loadConfig() with %s=%s should fail", tt.key, tt.val)
			}
		})
	}
}
