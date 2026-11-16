package resize

import (
	"errors"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"pv-sentinel/internal/k8s"
)

// The freeidp numbers, used verbatim so the fixture is the incident rather than
// an abstraction of it.
const (
	gib      = int64(1 << 30)
	pvName   = "pvc-4e4658fb-4d16-4f14-a2b7-f87913b92894"
	nsName   = "freeidp"
	pvcName  = "freeidp-pg-2"
	nodeName = "um760-1"
	fiveGiB  = 5 * gib
	tenGiB   = 10 * gib
)

var t0 = time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)

// csiPV builds a CSI PersistentVolume. driver "" with ctx nil yields an empty
// source, which is how a non-CSI volume is represented.
func csiPV(name, driver string, attrs map[string]string) corev1.PersistentVolume {
	return corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: corev1.PersistentVolumeSpec{
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				CSI: &corev1.CSIPersistentVolumeSource{Driver: driver, VolumeAttributes: attrs},
			},
			ClaimRef: &corev1.ObjectReference{Namespace: nsName, Name: pvcName},
		},
	}
}

// hostPathPV is an in-tree volume: the CSI branch of the source stays nil.
func hostPathPV(name string) corev1.PersistentVolume {
	return corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: corev1.PersistentVolumeSpec{
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				HostPath: &corev1.HostPathVolumeSource{Path: "/srv"},
			},
			ClaimRef: &corev1.ObjectReference{Namespace: nsName, Name: pvcName},
		},
	}
}

func pvc(ns, name, spec, status string, conds ...corev1.PersistentVolumeClaimCondition) corev1.PersistentVolumeClaim {
	p := corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: name},
	}
	if spec != "" {
		p.Spec.Resources.Requests = corev1.ResourceList{
			corev1.ResourceStorage: resource.MustParse(spec),
		}
	}
	if status != "" {
		p.Status.Capacity = corev1.ResourceList{
			corev1.ResourceStorage: resource.MustParse(status),
		}
	}
	p.Status.Conditions = conds
	return p
}

func ptr(s string) *string { return &s }

func TestProtocol(t *testing.T) {
	tests := []struct {
		name string
		pv   corev1.PersistentVolume
		want string
	}{
		{
			name: "driver name says iscsi",
			pv:   csiPV(pvName, "iscsi.example.com", nil),
			want: ProtocolISCSI,
		},
		{
			name: "driver name says nfs",
			pv:   csiPV(pvName, "nfs.csi.k8s.io", nil),
			want: ProtocolNFS,
		},
		{
			name: "driver name says local",
			pv:   csiPV(pvName, "sig-storage-local-csi-driver", nil),
			want: ProtocolLocal,
		},
		{
			name: "opaque driver name, iqn in the volume context",
			pv:   csiPV(pvName, "com.example.array", map[string]string{"targetPortal": "10.10.111.7:3260", "iqn": "iqn.2004-04.com.example:sdd"}),
			want: ProtocolISCSI,
		},
		{
			name: "opaque driver name, portal in the volume context",
			pv:   csiPV(pvName, "com.example.array", map[string]string{"portal": "10.10.111.7:3260"}),
			want: ProtocolISCSI,
		},
		{
			name: "opaque driver name, no evidence either way",
			pv:   csiPV(pvName, "com.example.array", map[string]string{"volumeType": "ssd"}),
			want: ProtocolUnknown,
		},
		{
			name: "in-tree hostpath is not csi",
			pv:   hostPathPV(pvName),
			want: ProtocolNotCSI,
		},
		{
			name: "driver name wins over a contradictory volume context",
			pv:   csiPV(pvName, "nfs.csi.k8s.io", map[string]string{"iqn": "iqn.leftover"}),
			want: ProtocolNFS,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Protocol(tt.pv); got != tt.want {
				t.Errorf("Protocol() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestClassify(t *testing.T) {
	iscsiCandidate := func(spec, status int64, conds ...Condition) Candidate {
		c := Candidate{
			PVName: pvName, Namespace: nsName, PVCName: pvcName,
			Protocol: ProtocolISCSI, Conditions: conds,
		}
		if spec >= 0 {
			c.SpecBytes, c.SpecSet = spec, true
		}
		if status >= 0 {
			c.StatusBytes, c.StatusSet = status, true
		}
		return c
	}
	resizing := Condition{Type: "Resizing", Status: "True"}

	tests := []struct {
		name      string
		cand      Candidate
		firstSeen time.Time
		want      Verdict
	}{
		{
			name:      "no request and no capacity is unknown",
			cand:      iscsiCandidate(-1, -1),
			firstSeen: t0,
			want:      Verdict{State: StateUnknown},
		},
		{
			name:      "request without capacity is unknown",
			cand:      iscsiCandidate(tenGiB, -1),
			firstSeen: t0,
			want:      Verdict{State: StateUnknown},
		},
		{
			name:      "capacity without request is unknown",
			cand:      iscsiCandidate(-1, fiveGiB),
			firstSeen: t0,
			want:      Verdict{State: StateUnknown},
		},
		{
			name:      "equal sizes are current",
			cand:      iscsiCandidate(tenGiB, tenGiB),
			firstSeen: t0,
			want:      Verdict{State: StateCurrent},
		},
		{
			name:      "a shrink is reported and never diagnosed",
			cand:      iscsiCandidate(fiveGiB, tenGiB, resizing),
			firstSeen: t0.Add(-time.Hour),
			want:      Verdict{State: StateShrunk},
		},
		{
			name:      "a fresh gap is growing",
			cand:      iscsiCandidate(tenGiB, fiveGiB, resizing),
			firstSeen: t0,
			want:      Verdict{State: StateGrowing},
		},
		{
			name:      "a gap one second short of the threshold is still growing",
			cand:      iscsiCandidate(tenGiB, fiveGiB, resizing),
			firstSeen: t0.Add(-9*time.Minute - 59*time.Second),
			want:      Verdict{State: StateGrowing},
		},
		{
			name:      "a gap at exactly the threshold is stuck on an iscsi volume",
			cand:      iscsiCandidate(tenGiB, fiveGiB, resizing),
			firstSeen: t0.Add(-10 * time.Minute),
			want:      Verdict{State: StateStuck, Cause: CauseStaleDeviceSize},
		},
		{
			name:      "a long-standing gap is stuck",
			cand:      iscsiCandidate(tenGiB, fiveGiB, resizing),
			firstSeen: t0.Add(-72 * time.Hour),
			want:      Verdict{State: StateStuck, Cause: CauseStaleDeviceSize},
		},
		{
			name:      "a stuck nfs volume is node pending, not a stale device",
			cand:      Candidate{PVName: pvName, Namespace: nsName, PVCName: pvcName, Protocol: ProtocolNFS, SpecBytes: tenGiB, SpecSet: true, StatusBytes: fiveGiB, StatusSet: true, Conditions: []Condition{resizing}},
			firstSeen: t0.Add(-time.Hour),
			want:      Verdict{State: StateStuck, Cause: CauseNodePending},
		},
		{
			name:      "a stuck volume with no protocol is not csi",
			cand:      Candidate{PVName: pvName, Namespace: nsName, PVCName: pvcName, Protocol: ProtocolNotCSI, SpecBytes: tenGiB, SpecSet: true, StatusBytes: fiveGiB, StatusSet: true, Conditions: []Condition{resizing}},
			firstSeen: t0.Add(-time.Hour),
			want:      Verdict{State: StateStuck, Cause: CauseNotCSI},
		},
		{
			name:      "a negative age is clamped to zero, not to a huge duration",
			cand:      iscsiCandidate(tenGiB, fiveGiB, resizing),
			firstSeen: t0.Add(time.Hour),
			want:      Verdict{State: StateGrowing},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Classify(tt.cand, tt.firstSeen, t0, 10*time.Minute)
			if got != tt.want {
				t.Errorf("Classify() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

// TestClassify_ZeroThreshold covers the threshold<=0 setting: every gap is
// immediately stuck, which is what tests and one-shot runs want.
func TestClassify_ZeroThreshold(t *testing.T) {
	c := Candidate{
		PVName: pvName, Namespace: nsName, PVCName: pvcName, Protocol: ProtocolISCSI,
		SpecBytes: tenGiB, SpecSet: true, StatusBytes: fiveGiB, StatusSet: true,
		Conditions: []Condition{{Type: "Resizing", Status: "True"}},
	}
	got := Classify(c, t0, t0, 0)
	if got.State != StateStuck || got.Cause != CauseStaleDeviceSize {
		t.Fatalf("Classify(threshold=0) = %+v, want stuck/stale_device_size", got)
	}
}

func TestCauseFor(t *testing.T) {
	tests := []struct {
		name     string
		protocol string
		conds    []Condition
		want     string
	}{
		{
			name:     "iscsi with Resizing=True is the freeidp signature",
			protocol: ProtocolISCSI,
			conds:    []Condition{{Type: "Resizing", Status: "True", Message: "Waiting on online resize"}},
			want:     CauseStaleDeviceSize,
		},
		{
			name:     "iscsi with FileSystemResizePending=True is the same signature",
			protocol: ProtocolISCSI,
			conds:    []Condition{{Type: "FileSystemResizePending", Status: "True"}},
			want:     CauseStaleDeviceSize,
		},
		{
			name:     "controller pending alone is the controller's problem",
			protocol: ProtocolISCSI,
			conds:    []Condition{{Type: "ControllerResizePending", Status: "True"}},
			want:     CauseControllerPending,
		},
		{
			name:     "resizing wins over controller pending",
			protocol: ProtocolISCSI,
			conds: []Condition{
				{Type: "ControllerResizePending", Status: "True"},
				{Type: "Resizing", Status: "True"},
			},
			want: CauseStaleDeviceSize,
		},
		{
			name:     "lowercase true is still true",
			protocol: ProtocolISCSI,
			conds:    []Condition{{Type: "Resizing", Status: "true"}},
			want:     CauseStaleDeviceSize,
		},
		{
			name:     "a False condition is not a True one",
			protocol: ProtocolISCSI,
			conds:    []Condition{{Type: "Resizing", Status: "False"}, {Type: "FileSystemResizePending", Status: "False"}, {Type: "ControllerResizePending", Status: "False"}},
			want:     CauseNotStarted,
		},
		{
			name:     "no conditions at all means nothing picked it up",
			protocol: ProtocolISCSI,
			conds:    nil,
			want:     CauseNotStarted,
		},
		{
			name:     "local volume with Resizing=True cannot be rescanned",
			protocol: ProtocolLocal,
			conds:    []Condition{{Type: "Resizing", Status: "True"}},
			want:     CauseNodePending,
		},
		{
			name:     "unknown protocol with Resizing=True is node pending",
			protocol: ProtocolUnknown,
			conds:    []Condition{{Type: "Resizing", Status: "True"}},
			want:     CauseNodePending,
		},
		{
			name:     "not-csi short-circuits before the conditions are read",
			protocol: ProtocolNotCSI,
			conds:    []Condition{{Type: "Resizing", Status: "True"}},
			want:     CauseNotCSI,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := Candidate{
				PVName: pvName, Namespace: nsName, PVCName: pvcName, Protocol: tt.protocol,
				SpecBytes: tenGiB, SpecSet: true, StatusBytes: fiveGiB, StatusSet: true,
				Conditions: tt.conds,
			}
			if got := causeFor(c); got != tt.want {
				t.Errorf("causeFor() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildCandidates(t *testing.T) {
	pv := csiPV(pvName, "iscsi.example.com", nil)
	claim := pvc(nsName, pvcName, "10Gi", "5Gi",
		corev1.PersistentVolumeClaimCondition{Type: corev1.PersistentVolumeClaimResizing, Status: corev1.ConditionTrue})
	va := storagev1.VolumeAttachment{
		ObjectMeta: metav1.ObjectMeta{Name: "csi-" + pvName},
		Spec: storagev1.VolumeAttachmentSpec{
			NodeName: nodeName,
			Source:   storagev1.VolumeAttachmentSource{PersistentVolumeName: ptr(pvName)},
		},
	}
	t.Run("the freeidp triple joins into one candidate", func(t *testing.T) {
		got := BuildCandidates([]storagev1.VolumeAttachment{va},
			[]corev1.PersistentVolume{pv},
			[]corev1.PersistentVolumeClaim{claim})
		if len(got) != 1 {
			t.Fatalf("len = %d, want 1", len(got))
		}
		c := got[0]
		if c.PVName != pvName || c.Namespace != nsName || c.PVCName != pvcName {
			t.Errorf("identity = %q %q/%q", c.PVName, c.Namespace, c.PVCName)
		}
		if c.Protocol != ProtocolISCSI {
			t.Errorf("Protocol = %q, want iscsi", c.Protocol)
		}
		if !c.SpecSet || c.SpecBytes != tenGiB {
			t.Errorf("spec = %d (set=%t), want %d", c.SpecBytes, c.SpecSet, tenGiB)
		}
		if !c.StatusSet || c.StatusBytes != fiveGiB {
			t.Errorf("status = %d (set=%t), want %d", c.StatusBytes, c.StatusSet, fiveGiB)
		}
		if len(c.Conditions) != 1 || c.Conditions[0].Type != "Resizing" || c.Conditions[0].Status != "True" {
			t.Errorf("conditions = %+v", c.Conditions)
		}
	})

	t.Run("unresolvable attachments are skipped", func(t *testing.T) {
		named := func(n string) storagev1.VolumeAttachment {
			return storagev1.VolumeAttachment{
				ObjectMeta: metav1.ObjectMeta{Name: "va-" + n},
				Spec:       storagev1.VolumeAttachmentSpec{NodeName: nodeName},
			}
		}
		// Source with no PV name.
		empty := named("empty")
		// Attachment pointing at a PV that is not in the list.
		dangling := named("dangling")
		dangling.Spec.Source.PersistentVolumeName = ptr("pv-gone")
		// A PV with no claimRef.
		unclaimed := corev1.PersistentVolume{ObjectMeta: metav1.ObjectMeta{Name: "pv-unclaimed"}}
		unclaimedVA := named("unclaimed")
		unclaimedVA.Spec.Source.PersistentVolumeName = ptr("pv-unclaimed")
		// A PV whose claimRef points at a PVC that is not in the list.
		orphan := corev1.PersistentVolume{
			ObjectMeta: metav1.ObjectMeta{Name: "pv-orphan"},
			Spec: corev1.PersistentVolumeSpec{
				ClaimRef: &corev1.ObjectReference{Namespace: nsName, Name: "missing-claim"},
			},
		}
		orphanVA := named("orphan")
		orphanVA.Spec.Source.PersistentVolumeName = ptr("pv-orphan")

		vas := []storagev1.VolumeAttachment{empty, dangling, unclaimedVA, orphanVA}
		pvs := []corev1.PersistentVolume{pv, unclaimed, orphan}
		if got := BuildCandidates(vas, pvs, []corev1.PersistentVolumeClaim{claim}); len(got) != 0 {
			t.Fatalf("BuildCandidates() = %+v, want no candidates", got)
		}
	})

	t.Run("a pvc attached twice yields one candidate", func(t *testing.T) {
		second := va
		second.Name = "csi-second"
		got := BuildCandidates([]storagev1.VolumeAttachment{va, second},
			[]corev1.PersistentVolume{pv},
			[]corev1.PersistentVolumeClaim{claim})
		if len(got) != 1 {
			t.Fatalf("len = %d, want 1 (multi-attach must not double-count)", len(got))
		}
	})

	t.Run("output is sorted by key so reports are stable", func(t *testing.T) {
		pvB := csiPV("pv-b", "iscsi.example.com", nil)
		pvB.Spec.ClaimRef = &corev1.ObjectReference{Namespace: nsName, Name: "aaa-claim"}
		pvA := csiPV("pv-a", "iscsi.example.com", nil)
		pvA.Spec.ClaimRef = &corev1.ObjectReference{Namespace: nsName, Name: "zzz-claim"}
		claimB := pvc(nsName, "aaa-claim", "1Gi", "1Gi")
		claimA := pvc(nsName, "zzz-claim", "1Gi", "1Gi")
		vas := []storagev1.VolumeAttachment{
			{ObjectMeta: metav1.ObjectMeta{Name: "va-b"}, Spec: storagev1.VolumeAttachmentSpec{NodeName: nodeName, Source: storagev1.VolumeAttachmentSource{PersistentVolumeName: ptr("pv-b")}}},
			{ObjectMeta: metav1.ObjectMeta{Name: "va-a"}, Spec: storagev1.VolumeAttachmentSpec{NodeName: nodeName, Source: storagev1.VolumeAttachmentSource{PersistentVolumeName: ptr("pv-a")}}},
		}
		got := BuildCandidates(vas,
			[]corev1.PersistentVolume{pvB, pvA},
			[]corev1.PersistentVolumeClaim{claimB, claimA})
		if len(got) != 2 || got[0].PVCName != "aaa-claim" || got[1].PVCName != "zzz-claim" {
			t.Fatalf("order = %v", []string{got[0].PVCName, got[1].PVCName})
		}
	})
}

func TestClassifyAll(t *testing.T) {
	cand := func(pvcN string, spec, status int64, conds ...Condition) Candidate {
		return Candidate{
			PVName: "pv-" + pvcN, Namespace: nsName, PVCName: pvcN, Protocol: ProtocolISCSI,
			SpecBytes: spec, SpecSet: true, StatusBytes: status, StatusSet: true,
			Conditions: conds,
		}
	}
	resizing := []Condition{{Type: "Resizing", Status: "True"}}

	t.Run("first observation of a gap is growing, not stuck", func(t *testing.T) {
		first := map[string]time.Time{}
		r := ClassifyAll([]Candidate{cand("a", tenGiB, fiveGiB, resizing...)}, first, t0, 10*time.Minute)
		if r.Growing != 1 || r.Stuck != 0 {
			t.Fatalf("counts = %+v, want growing=1 stuck=0", r)
		}
		if got, ok := first[nsName+"/a"]; !ok || !got.Equal(t0) {
			t.Errorf("first[%s/a] = %v, want %v", nsName, got, t0)
		}
	})

	t.Run("the same gap past the threshold is stuck and becomes a finding", func(t *testing.T) {
		first := map[string]time.Time{nsName + "/a": t0.Add(-time.Hour)}
		r := ClassifyAll([]Candidate{cand("a", tenGiB, fiveGiB, resizing...)}, first, t0, 10*time.Minute)
		if r.Stuck != 1 || len(r.Findings) != 1 {
			t.Fatalf("counts = %+v findings = %+v", r, r.Findings)
		}
		f := r.Findings[0]
		if f.Cause != CauseStaleDeviceSize || f.State != StateStuck {
			t.Errorf("finding = %+v", f)
		}
		if !f.Since.Equal(t0.Add(-time.Hour)) || f.Age != time.Hour {
			t.Errorf("since=%v age=%v, want since=%v age=1h", f.Since, f.Age, t0.Add(-time.Hour))
		}
	})

	t.Run("counts cover every state and the findings match the stuck count", func(t *testing.T) {
		cands := []Candidate{
			cand("cur", tenGiB, tenGiB),
			cand("shrunk", fiveGiB, tenGiB),
			// No request and no capacity: neither field set, so there is nothing
			// to compare.
			{PVName: "pv-unknown", Namespace: nsName, PVCName: "unknown", Protocol: ProtocolISCSI},
		}
		first := map[string]time.Time{}
		r := ClassifyAll(cands, first, t0, 10*time.Minute)
		if r.Current != 1 || r.Shrunk != 1 || r.Unknown != 1 || r.Stuck != 0 || r.Growing != 0 {
			t.Fatalf("counts = %+v", r)
		}
		if len(r.Findings) != r.Stuck {
			t.Errorf("len(findings)=%d but stuck=%d", len(r.Findings), r.Stuck)
		}
	})

	t.Run("a gap that resolves stops accumulating age", func(t *testing.T) {
		first := map[string]time.Time{nsName + "/a": t0.Add(-time.Hour)}
		r := ClassifyAll([]Candidate{cand("a", tenGiB, tenGiB)}, first, t0, 10*time.Minute)
		if r.Current != 1 {
			t.Fatalf("counts = %+v", r)
		}
		if _, ok := first[nsName+"/a"]; ok {
			t.Error("first-seen entry survived a resolved gap; the next resize would inherit this age")
		}
	})

	t.Run("a claim that leaves the node is pruned from the bookkeeping", func(t *testing.T) {
		first := map[string]time.Time{
			nsName + "/gone": t0.Add(-30 * 24 * time.Hour),
			nsName + "/here": t0,
		}
		r := ClassifyAll([]Candidate{cand("here", tenGiB, fiveGiB, resizing...)}, first, t0, 10*time.Minute)
		if r.Growing != 1 {
			t.Fatalf("counts = %+v", r)
		}
		if _, ok := first[nsName+"/gone"]; ok {
			t.Error("departed claim kept its firstSeen")
		}
	})

	t.Run("an empty scan yields an empty report and clears the bookkeeping", func(t *testing.T) {
		first := map[string]time.Time{nsName + "/a": t0.Add(-time.Hour)}
		r := ClassifyAll(nil, first, t0, 10*time.Minute)
		if r.Current+r.Growing+r.Stuck+r.Shrunk+r.Unknown != 0 || len(r.Findings) != 0 {
			t.Fatalf("report = %+v", r)
		}
		if len(first) != 0 {
			t.Errorf("first = %v, want empty", first)
		}
	})
}

func TestFindingKey(t *testing.T) {
	tests := []struct {
		name string
		f    Finding
		want string
	}{
		{"namespaced", Finding{PVName: pvName, Namespace: "freeidp", PVCName: "freeidp-pg-2"}, "freeidp/freeidp-pg-2"},
		{"no namespace falls back to the pv", Finding{PVName: pvName}, pvName},
		{"empty everything is empty", Finding{}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.f.Key(); got != tt.want {
				t.Errorf("Key() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSummary(t *testing.T) {
	r := Report{Current: 3, Growing: 1, Stuck: 2, Shrunk: 1, Unknown: 4}
	got := Summary(r)
	want := "current=3 growing=1 stuck=2 shrunk=1 unknown=4"
	if got != want {
		t.Errorf("Summary() = %q, want %q", got, want)
	}
}

func TestDetector_Detect(t *testing.T) {
	pv := csiPV(pvName, "iscsi.example.com", nil)
	claim := pvc(nsName, pvcName, "10Gi", "5Gi",
		corev1.PersistentVolumeClaimCondition{Type: corev1.PersistentVolumeClaimResizing, Status: corev1.ConditionTrue})
	va := storagev1.VolumeAttachment{
		ObjectMeta: metav1.ObjectMeta{Name: "csi-" + pvName},
		Spec: storagev1.VolumeAttachmentSpec{
			NodeName: nodeName,
			Source:   storagev1.VolumeAttachmentSource{PersistentVolumeName: ptr(pvName)},
		},
	}
	// A volume attached to a *different* node must not appear in this node's
	// report; that filter is the whole reason the detector goes through
	// VolumeAttachmentsOnNode instead of listing every PVC.
	other := pvc("other-ns", "other-pvc", "10Gi", "5Gi")
	otherPV := csiPV("pv-other", "iscsi.example.com", nil)
	otherPV.Spec.ClaimRef = &corev1.ObjectReference{Namespace: "other-ns", Name: "other-pvc"}
	otherVA := storagev1.VolumeAttachment{
		ObjectMeta: metav1.ObjectMeta{Name: "csi-pv-other"},
		Spec: storagev1.VolumeAttachmentSpec{
			NodeName: "some-other-node",
			Source:   storagev1.VolumeAttachmentSource{PersistentVolumeName: ptr("pv-other")},
		},
	}

	fake := fake.NewSimpleClientset(&va, &otherVA, &pv, &otherPV, &claim, &other)
	cl := k8s.New(fake, nodeName)
	d := New(cl, 10*time.Minute)

	ctx := t.Context()

	first, err := d.Detect(ctx, t0)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if first.Node != nodeName {
		t.Errorf("Node = %q, want %q", first.Node, nodeName)
	}
	if !first.At.Equal(t0) {
		t.Errorf("At = %v, want %v", first.At, t0)
	}
	if first.Growing != 1 || first.Stuck != 0 {
		t.Fatalf("first pass counts = %+v, want growing=1", first)
	}

	// The same gap, observed after the threshold has elapsed, is now stuck.
	second, err := d.Detect(ctx, t0.Add(30*time.Minute))
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if second.Stuck != 1 || len(second.Findings) != 1 {
		t.Fatalf("second pass = %+v, want stuck=1", second)
	}
	f := second.Findings[0]
	if f.Cause != CauseStaleDeviceSize || f.Protocol != ProtocolISCSI {
		t.Errorf("finding = %+v, want stale_device_size on iscsi", f)
	}
	if f.SpecBytes != tenGiB || f.StatusBytes != fiveGiB {
		t.Errorf("sizes = %d/%d, want %d/%d", f.SpecBytes, f.StatusBytes, tenGiB, fiveGiB)
	}
	if f.Age != 30*time.Minute {
		t.Errorf("Age = %v, want 30m", f.Age)
	}
	if strings.Contains(Summary(second), "growing=1") {
		t.Errorf("summary still reports the gap as growing: %s", Summary(second))
	}
}

func TestDetector_DetectNoClient(t *testing.T) {
	// An inert client must fail with the k8s sentinel wrapped, not panic: the
	// binary runs on nodes with no API access and reports why.
	d := New(k8s.New(nil, nodeName), time.Minute)
	_, err := d.Detect(t.Context(), t0)
	if !errors.Is(err, k8s.ErrNoClient) {
		t.Fatalf("error = %v, want ErrNoClient", err)
	}
	if !strings.Contains(err.Error(), "volume attachments") {
		t.Errorf("error %q does not say which list failed", err)
	}
}

func TestDetector_FirstSeen(t *testing.T) {
	d := New(k8s.New(nil, nodeName), time.Minute)
	if _, ok := d.FirstSeen(nsName, pvcName); ok {
		t.Fatal("FirstSeen on an empty detector reported a time")
	}
	d.mu.Lock()
	d.first[nsName+"/"+pvcName] = t0
	d.mu.Unlock()
	got, ok := d.FirstSeen(nsName, pvcName)
	if !ok || !got.Equal(t0) {
		t.Errorf("FirstSeen = %v, %t; want %v, true", got, ok, t0)
	}
}
