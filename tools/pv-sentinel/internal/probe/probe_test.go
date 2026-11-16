package probe

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// realMounts is a /proc/mounts excerpt captured from the k8s-ctrl node. It
// contains the doubled view a host-root mount produces (each volume listed at
// both /var/lib/kubelet/... and /host/var/lib/kubelet/...), which is the exact
// shape that made a naive count report 22 volumes where 11 exist.
const realMounts = `proc /proc proc rw,nosuid,nodev,noexec,relatime 0 0
sysfs /sys sysfs rw,nosuid,nodev,noexec,relatime 0 0
/dev/sda /var/lib/kubelet/pods/e704e317-c538-409d-a144-362a4755ff39/volumes/kubernetes.io~csi/pvc-c7b1d832-6ca4-4830-bfe8-ba08cce33d0f/mount ext4 rw,relatime,stripe=2048 0 0
/dev/sda /host/var/lib/kubelet/pods/e704e317-c538-409d-a144-362a4755ff39/volumes/kubernetes.io~csi/pvc-c7b1d832-6ca4-4830-bfe8-ba08cce33d0f/mount ext4 rw,relatime,stripe=2048 0 0
/dev/sdd /var/lib/kubelet/pods/334c3464-a72c-48b5-ad17-bdc420597d6d/volumes/kubernetes.io~csi/pvc-adea4c45-3d82-4be5-80d9-160613a19f62/mount ext4 rw,relatime,stripe=2048 0 0
/dev/sdd /host/var/lib/kubelet/pods/334c3464-a72c-48b5-ad17-bdc420597d6d/volumes/kubernetes.io~csi/pvc-adea4c45-3d82-4be5-80d9-160613a19f62/mount ext4 rw,relatime,stripe=2048 0 0
/dev/sdb /var/lib/kubelet/pods/7476fa7e-62a0-4bd1-8783-85141dcccb7a/volumes/kubernetes.io~csi/pvc-53e85256-5c88-4412-9ef5-0386c7ff8e18/mount ext4 rw,relatime,stripe=2048 0 0
/dev/sdb /host/var/lib/kubelet/pods/7476fa7e-62a0-4bd1-8783-85141dcccb7a/volumes/kubernetes.io~csi/pvc-53e85256-5c88-4412-9ef5-0386c7ff8e18/mount ext4 rw,relatime,stripe=2048 0 0
/dev/sdc /var/lib/kubelet/pods/960302fa-c68f-403a-975c-fbc10d5dae08/volumes/kubernetes.io~csi/pvc-23794761-b156-48f9-86d4-9586d6974bf5/mount ext4 rw,relatime,stripe=2048 0 0
/dev/sdc /host/var/lib/kubelet/pods/960302fa-c68f-403a-975c-fbc10d5dae08/volumes/kubernetes.io~csi/pvc-23794761-b156-48f9-86d4-9586d6974bf5/mount ext4 rw,relatime,stripe=2048 0 0
tmpfs /var/lib/kubelet/pods/11111111-2222-3333-4444-555555555555/etc-hosts tmpfs rw,mode=755 0 0
`

func TestParseMounts(t *testing.T) {
	tests := []struct {
		name    string
		content string
		allowed []string
		want    int
	}{
		{"real cluster excerpt, default fs types", realMounts, nil, 8},
		{"ext4 only", realMounts, []string{"ext4"}, 8},
		{"xfs only excludes every ext4 line", realMounts, []string{"xfs"}, 0},
		{"empty", "", nil, 0},
		{"malformed short line", "dev /path\n", []string{"ext4"}, 0},
		{"comment line ignored", "# comment\n/dev/x /a/b/volumes/kubernetes.io~csi/pvc-a/mount ext4 rw 0 0\n", nil, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseMounts(tt.content, tt.allowed)
			if len(got) != tt.want {
				t.Errorf("ParseMounts() = %d mounts, want %d", len(got), tt.want)
			}
		})
	}
}

func TestParseMounts_Fields(t *testing.T) {
	mounts := ParseMounts(realMounts, nil)
	if len(mounts) == 0 {
		t.Fatal("no mounts parsed")
	}
	m := mounts[0]
	if m.Device != "/dev/sda" {
		t.Errorf("Device = %q, want /dev/sda", m.Device)
	}
	if m.FSType != "ext4" {
		t.Errorf("FSType = %q, want ext4", m.FSType)
	}
	if m.Options != "rw,relatime,stripe=2048" {
		t.Errorf("Options = %q, want rw,relatime,stripe=2048", m.Options)
	}
	if !strings.Contains(m.Path, "pvc-c7b1d832") {
		t.Errorf("Path = %q, want it to contain pvc-c7b1d832", m.Path)
	}
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"/host", "/"},
		{"/host/var/lib/kubelet/pods/x", "/var/lib/kubelet/pods/x"},
		{"/var/lib/kubelet/pods/x", "/var/lib/kubelet/pods/x"},
		{"/host/", "/"},
		{"/host/var//lib", "/var/lib"},
	}
	for _, tt := range tests {
		if got := NormalizePath(tt.in); got != tt.want {
			t.Errorf("NormalizePath(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestLooksLikeUID(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"e704e317-c538-409d-a144-362a4755ff39", true},
		{"11111111-2222-3333-4444-555555555555", true},
		{"not-a-uid", false},
		{"", false},
		{"E704E317-C538-409D-A144-362A4755FF39", false}, // uppercase is not canonical
		{"e704e317c538-409d-a144-362a4755ff39", false},  // missing a hyphen
		{"e704e317-c538-409d-a144-362a4755ff3", false},  // too short
	}
	for _, tt := range tests {
		if got := looksLikeUID(tt.in); got != tt.want {
			t.Errorf("looksLikeUID(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestParseKubeletPath(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		wantUID string
		wantPV  string
		wantOK  bool
	}{
		{
			"standard",
			"/var/lib/kubelet/pods/e704e317-c538-409d-a144-362a4755ff39/volumes/kubernetes.io~csi/pvc-c7b1d832-6ca4-4830-bfe8-ba08cce33d0f/mount",
			"e704e317-c538-409d-a144-362a4755ff39", "pvc-c7b1d832-6ca4-4830-bfe8-ba08cce33d0f", true,
		},
		{
			"host-prefixed form normalizes to the same result",
			"/host/var/lib/kubelet/pods/e704e317-c538-409d-a144-362a4755ff39/volumes/kubernetes.io~csi/pvc-c7b1d832-6ca4-4830-bfe8-ba08cce33d0f/mount",
			"e704e317-c538-409d-a144-362a4755ff39", "pvc-c7b1d832-6ca4-4830-bfe8-ba08cce33d0f", true,
		},
		{"not a pods path", "/var/lib/kubelet/plugins/kubernetes.io/csi/x", "", "", false},
		{"uid too short", "/var/lib/kubelet/pods/short/volumes/kubernetes.io~csi/pvc-a/mount", "", "", false},
		{"secret volume dir, not csi", "/var/lib/kubelet/pods/e704e317-c538-409d-a144-362a4755ff39/volumes/kubernetes.io~secret/x", "", "", false},
		{"missing mount segment", "/var/lib/kubelet/pods/e704e317-c538-409d-a144-362a4755ff39/volumes/kubernetes.io~csi", "", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uid, pv, ok := parseKubeletPath(tt.in)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if !tt.wantOK {
				return
			}
			if uid != tt.wantUID {
				t.Errorf("uid = %q, want %q", uid, tt.wantUID)
			}
			if pv != tt.wantPV {
				t.Errorf("pv = %q, want %q", pv, tt.wantPV)
			}
		})
	}
}

// TestDiscover_DedupesDoubledView is the regression test for the 22-lines-for-11-
// volumes trap: the mount table lists each volume twice, and Discover must
// report each PV exactly once.
func TestDiscover_DedupesDoubledView(t *testing.T) {
	vols := Discover(realMounts, nil)
	if len(vols) != 4 {
		t.Fatalf("Discover() = %d volumes, want 4 (8 mount lines, each PV listed twice)", len(vols))
	}
	seen := map[string]bool{}
	for _, v := range vols {
		if seen[v.PVName] {
			t.Errorf("duplicate PV %s in output", v.PVName)
		}
		seen[v.PVName] = true
		if v.PodUID == "" {
			t.Errorf("volume %s has empty PodUID", v.PVName)
		}
	}
	for _, want := range []string{
		"pvc-c7b1d832-6ca4-4830-bfe8-ba08cce33d0f",
		"pvc-adea4c45-3d82-4be5-80d9-160613a19f62",
		"pvc-53e85256-5c88-4412-9ef5-0386c7ff8e18",
		"pvc-23794761-b156-48f9-86d4-9586d6974bf5",
	} {
		if !seen[want] {
			t.Errorf("missing expected PV %s", want)
		}
	}
}

func TestDiscover_SortedDeterministically(t *testing.T) {
	first := Discover(realMounts, nil)
	second := Discover(realMounts, nil)
	for i := range first {
		if first[i].PVName != second[i].PVName {
			t.Fatalf("Discover not deterministic at %d: %q vs %q", i, first[i].PVName, second[i].PVName)
		}
	}
}

func TestVolume_OptionsClaimedRW(t *testing.T) {
	tests := []struct {
		opts string
		want bool
	}{
		{"rw,relatime,stripe=2048", true},
		{"ro,relatime", false},
		{"rw", true},
		{"defaults", false},
		{"", false},
	}
	for _, tt := range tests {
		v := Volume{Options: tt.opts}
		if got := v.OptionsClaimedRW(); got != tt.want {
			t.Errorf("OptionsClaimedRW(%q) = %v, want %v", tt.opts, got, tt.want)
		}
	}
}

func TestProbe_WritableDir(t *testing.T) {
	dir := t.TempDir()
	status, detail := Probe(dir)
	if status != StatusHealthy {
		t.Fatalf("Probe() = %q (%s), want %q", status, detail, StatusHealthy)
	}
	// The scratch file must not be left behind.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("probe left %d file(s) behind, want 0", len(entries))
	}
}

// TestProbe_NonexistentDirIsNotReadOnly guards the tool's core claim: only EROFS
// may be reported as read_only. A missing directory must land in unknown, or
// every transient mount race would look like a broken volume.
func TestProbe_NonexistentDirIsNotReadOnly(t *testing.T) {
	status, detail := Probe(filepath.Join(t.TempDir(), "does-not-exist"))
	if status == StatusReadOnly {
		t.Fatalf("Probe() on a missing dir = read_only; must not fabricate read-only from %s", detail)
	}
	if status != StatusUnknown {
		t.Fatalf("Probe() = %q, want %q", status, StatusUnknown)
	}
	if detail == "" {
		t.Error("unknown status must carry the error text")
	}
}

func TestProbeAll_StampsAndFills(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	vols := []Volume{{PVName: "pvc-a", MountPath: dir}}

	out := ProbeAll(vols, now)
	if len(out) != 1 {
		t.Fatalf("len = %d, want 1", len(out))
	}
	if out[0].Status != StatusHealthy {
		t.Errorf("Status = %q, want %q", out[0].Status, StatusHealthy)
	}
	if !out[0].ProbedAt.Equal(now) {
		t.Errorf("ProbedAt = %v, want %v", out[0].ProbedAt, now)
	}
}

func TestNewReport(t *testing.T) {
	now := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	vols := []Volume{
		{PVName: "pvc-a", MountPath: "", Status: StatusHealthy},
		{PVName: "pvc-b", MountPath: "", Status: StatusReadOnly},
		{PVName: "pvc-c", MountPath: "", Status: StatusUnknown},
		{PVName: "pvc-d", MountPath: "", Status: "bogus"},
	}
	r := NewReport("node-1", vols, now)

	if r.Node != "node-1" || !r.At.Equal(now) {
		t.Errorf("node/time not carried: %q %v", r.Node, r.At)
	}
	if r.Healthy != 1 || r.ReadOnly != 1 || r.Unknown != 2 {
		t.Errorf("tally = healthy:%d readonly:%d unknown:%d, want 1/1/2", r.Healthy, r.ReadOnly, r.Unknown)
	}
	if len(r.MountedPV) != 4 {
		t.Errorf("MountedPV = %d, want 4", len(r.MountedPV))
	}
}

func TestReadMounts(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "mounts")
	if err := os.WriteFile(p, []byte(realMounts), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	content, err := ReadMounts(p)
	if err != nil {
		t.Fatalf("ReadMounts: %v", err)
	}
	if content != realMounts {
		t.Error("ReadMounts did not return the file contents verbatim")
	}
	if _, err := ReadMounts(filepath.Join(dir, "missing")); err == nil {
		t.Error("ReadMounts on a missing file should error")
	}
}

func TestUnescapeMountPath(t *testing.T) {
	if got := unescapeMountPath(`/var/lib/kubelet/my\040dir`); got != "/var/lib/kubelet/my dir" {
		t.Errorf("unescapeMountPath() = %q, want %q", got, "/var/lib/kubelet/my dir")
	}
}
