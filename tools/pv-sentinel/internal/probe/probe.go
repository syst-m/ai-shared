// Package probe discovers CSI-provisioned volume mounts on a single node and
// determines whether each one is actually writable.
//
// The reason this package exists is that mount options lie. When an ext4 journal
// aborts the kernel remounts the filesystem read-only, but /proc/mounts keeps
// reporting "rw,relatime" for as long as the stale kubelet mount survives. Six
// volumes on the target cluster reported rw for days while every write returned
// EROFS. Anything that answers "is this volume healthy?" by parsing mount flags
// gets it wrong. The only reliable answer is to create a file and see whether
// the kernel lets you.
package probe

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"
)

// Status of a volume as reported by a write probe.
const (
	StatusHealthy  = "healthy"
	StatusReadOnly = "read_only"
	StatusUnknown  = "unknown"
)

// ProbeFileName is the scratch file created inside each mount to test
// writability. Dot-prefixed so it never passes for application data.
const ProbeFileName = ".pv-sentinel-probe"

// CSIVolumeDir is the kubelet directory segment that marks a path as a CSI
// volume mount rather than a configmap, secret, or subPath bind mount.
const CSIVolumeDir = "kubernetes.io~csi"

// DefaultFSTypes are the filesystem types worth probing. The target cluster
// provisions ext4 and xfs via democratic-csi; anything else is not a real
// volume mount and probing it would produce noise.
var DefaultFSTypes = []string{"ext4", "xfs"}

// Mount is one parsed line from /proc/mounts.
type Mount struct {
	Device  string
	Path    string
	FSType  string
	Options string
}

// Volume is one discovered CSI volume on this node.
type Volume struct {
	PVName    string // pvc-<uuid>, as it appears in the kubelet path
	Device    string // /dev/sda, as reported by the mount table
	FSType    string // ext4 | xfs
	Options   string // raw options field, e.g. "rw,relatime,stripe=2048"
	MountPath string // absolute path as visible from this pod
	PodUID    string // kubelet pod UID taken from the path
	Status    string // filled by Probe
	Detail    string // errno text when Status is not healthy
	ProbedAt  time.Time
}

// OptionsClaimedRW reports whether the mount options advertise a read-write
// mount. It exists purely to surface the lie: a volume with Status
// StatusReadOnly and OptionsClaimedRW true is one where the mount table says
// rw and the kernel disagrees.
func (v Volume) OptionsClaimedRW() bool {
	for _, opt := range strings.Split(v.Options, ",") {
		if opt == "rw" {
			return true
		}
	}
	return false
}

// unescapeMountPath decodes the octal escapes the kernel uses in /proc/mounts
// for characters that are legal in a path but would break the line format.
func unescapeMountPath(s string) string {
	r := strings.NewReplacer(`\040`, " ", `\011`, "\t", `\012`, "\n", `\134`, `\`)
	return r.Replace(s)
}

// NormalizePath collapses the double view created by mounting the host root:
// the same volume appears once at /var/lib/kubelet/... and again at
// /host/var/lib/kubelet/..., so a naive count reports every volume twice. The
// cluster's own CSI node pods measure 22 mount lines for 11 real volumes for
// exactly this reason.
func NormalizePath(p string) string {
	p = filepath.Clean(p)
	switch {
	case p == "/host":
		return "/"
	case strings.HasPrefix(p, "/host/"):
		return p[len("/host"):]
	default:
		return p
	}
}

// looksLikeUID reports whether s is shaped like a Kubernetes pod UID (a 36
// character hyphenated UUID). Requiring the shape keeps unrelated /pods/
// segments from being mistaken for volume owners.
func looksLikeUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	// Ranging over bytes, not runes: a UID is ASCII by construction, and
	// ranging over a string would yield code points and byte offsets that
	// disagree for any non-ASCII input.
	for i, c := range []byte(s) {
		switch {
		case i == 8 || i == 13 || i == 18 || i == 23:
			// The four hyphen slots of a 36-character UUID.
			if c != '-' {
				return false
			}
		case c >= '0' && c <= '9', c >= 'a' && c <= 'f':
			// A hex digit where a hex digit belongs.
		default:
			return false
		}
	}
	return true
}

// ParseMounts parses /proc/mounts content, returning only CSI volume mounts
// whose fsType is allowed. An empty or nil allowed means DefaultFSTypes.
func ParseMounts(content string, allowed []string) []Mount {
	if len(allowed) == 0 {
		allowed = DefaultFSTypes
	}
	ok := make(map[string]bool, len(allowed))
	for _, f := range allowed {
		ok[strings.ToLower(strings.TrimSpace(f))] = true
	}

	out := make([]Mount, 0, 8)
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		m := Mount{
			Device:  unescapeMountPath(fields[0]),
			Path:    unescapeMountPath(fields[1]),
			FSType:  strings.ToLower(fields[2]),
			Options: fields[3],
		}
		if !ok[m.FSType] {
			continue
		}
		if !strings.Contains(m.Path, "volumes/"+CSIVolumeDir+"/") {
			continue
		}
		out = append(out, m)
	}
	return out
}

// volumesDir is the fixed path segment between a pod UID and the volume plugin
// directory: kubelet lays out /pods/<uid>/volumes/<plugin>/<name>/mount.
const volumesDir = "volumes"

// parseKubeletPath extracts the pod UID and PV name from a kubelet volume mount
// path, in either the /var/lib/kubelet/... form or the /host/... form. It
// returns ok=false for anything that is not a kubelet CSI mount.
//
// The full segment layout is pods/<uid>/volumes/<plugin>/<pv>/mount, so the
// plugin lands at index 2 and the PV name at index 3. Requiring the literal
// "volumes" segment is what keeps plugin/ paths and other pod subdirectories
// (etc-hosts, plugins/kubernetes.io/csi/...) from being read as volume owners.
func parseKubeletPath(p string) (uid, pv string, ok bool) {
	normalized := NormalizePath(p)
	i := strings.Index(normalized, "/pods/")
	if i < 0 {
		return "", "", false
	}
	parts := strings.Split(normalized[i+len("/pods/"):], "/")
	if len(parts) < 4 {
		return "", "", false
	}
	uid = parts[0]
	if !looksLikeUID(uid) {
		return "", "", false
	}
	if parts[1] != volumesDir {
		return "", "", false
	}
	if parts[2] != CSIVolumeDir {
		return "", "", false
	}
	pv = parts[3]
	if pv == "" {
		return "", "", false
	}
	return uid, pv, true
}

// Discover returns the distinct CSI volumes visible in /proc/mounts content,
// deduped by PV name and sorted for stable output.
func Discover(content string, fsTypes []string) []Volume {
	mounts := ParseMounts(content, fsTypes)
	seen := make(map[string]bool, len(mounts))
	out := make([]Volume, 0, len(mounts))
	for _, m := range mounts {
		uid, pv, ok := parseKubeletPath(m.Path)
		if !ok || seen[pv] {
			continue
		}
		seen[pv] = true
		out = append(out, Volume{
			PVName:    pv,
			Device:    m.Device,
			FSType:    m.FSType,
			Options:   m.Options,
			MountPath: m.Path,
			PodUID:    uid,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].PVName < out[j].PVName })
	return out
}

// ReadMounts reads a mount table from disk. It is separate from Discover so
// that a captured /proc/mounts file can be replayed against the parser.
func ReadMounts(path string) (string, error) {
	//nolint:gosec // G304: the path is operator-supplied config, not user input.
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Probe attempts to create and remove a scratch file inside dir.
//
// EROFS is the only failure that means "read-only filesystem". Everything else
// is reported as StatusUnknown with the error preserved, because folding
// EACCES or ENOENT into the read-only bucket would manufacture false positives
// — which is the failure mode this whole tool is trying to avoid.
func Probe(dir string) (status string, detail string) {
	//nolint:gosec // G304: dir comes from this node's own mount table.
	f, err := os.OpenFile(filepath.Join(dir, ProbeFileName), os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		var errno syscall.Errno
		if errors.As(err, &errno) && errno == syscall.EROFS {
			return StatusReadOnly, errno.Error()
		}
		return StatusUnknown, err.Error()
	}
	if err := f.Close(); err != nil {
		return StatusUnknown, err.Error()
	}
	if err := os.Remove(filepath.Join(dir, ProbeFileName)); err != nil {
		return StatusUnknown, err.Error()
	}
	return StatusHealthy, ""
}

// ProbeAll probes each volume and fills Status, Detail, and ProbedAt, returning
// the same slice. now is injected so callers and tests control the timestamp.
func ProbeAll(vols []Volume, now time.Time) []Volume {
	for i := range vols {
		vols[i].Status, vols[i].Detail = Probe(vols[i].MountPath)
		vols[i].ProbedAt = now
	}
	return vols
}

// Report is the result of one scan of one node.
type Report struct {
	Node      string
	At        time.Time
	Volumes   []Volume
	Healthy   int
	ReadOnly  int
	Unknown   int
	MountedPV []string

	// AttachedNotMounted lists PVs the API says are attached to this node but
	// that no mount table entry covers. That is a distinct failure from
	// read-only: the volume never got mounted at all.
	AttachedNotMounted []string
}

// NewReport tallies a probed volume set into a Report.
func NewReport(node string, vols []Volume, at time.Time) Report {
	r := Report{Node: node, At: at, Volumes: vols}
	r.MountedPV = make([]string, 0, len(vols))
	for _, v := range vols {
		r.MountedPV = append(r.MountedPV, v.PVName)
		switch v.Status {
		case StatusHealthy:
			r.Healthy++
		case StatusReadOnly:
			r.ReadOnly++
		default:
			r.Unknown++
		}
	}
	return r
}
