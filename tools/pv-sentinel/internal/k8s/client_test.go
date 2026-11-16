package k8s

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
)

func pod(ns, name, uid, node string, owners []metav1.OwnerReference) corev1.Pod {
	return corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       ns,
			Name:            name,
			UID:             types.UID(uid),
			OwnerReferences: owners,
		},
		Spec: corev1.PodSpec{NodeName: node},
	}
}

// objs converts pod values into the runtime.Object slice the fake clientset
// wants. DeepCopyObject has a pointer receiver, so a corev1.Pod value does not
// satisfy runtime.Object and cannot be passed directly.
func objs(pods ...corev1.Pod) []runtime.Object {
	out := make([]runtime.Object, 0, len(pods))
	for i := range pods {
		p := pods[i]
		out = append(out, &p)
	}
	return out
}

func attachment(name, node string, pvName *string) runtime.Object {
	return &storagev1.VolumeAttachment{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: storagev1.VolumeAttachmentSpec{
			NodeName: node,
			Source:   storagev1.VolumeAttachmentSource{PersistentVolumeName: pvName},
		},
	}
}

func TestFilterPodsOnNode(t *testing.T) {
	pods := []corev1.Pod{
		pod("default", "a", "aaaaaaaa-0000-0000-0000-000000000000", "node-1", nil),
		pod("default", "b", "bbbbbbbb-0000-0000-0000-000000000000", "node-2", nil),
		pod("kube-system", "c", "cccccccc-0000-0000-0000-000000000000", "node-1", nil),
		pod("", "unscheduled", "dddddddd-0000-0000-0000-000000000000", "", nil),
	}

	got := filterPodsOnNode(pods, "node-1")
	if len(got) != 2 {
		t.Fatalf("filterPodsOnNode() = %d pods, want 2", len(got))
	}
	names := map[string]bool{}
	for _, p := range got {
		if p.Spec.NodeName != "node-1" {
			t.Errorf("pod %s on node %q leaked into node-1 result", p.Name, p.Spec.NodeName)
		}
		names[p.Name] = true
	}
	if !names["a"] || !names["c"] {
		t.Errorf("got %v, want a and c", names)
	}

	if empty := filterPodsOnNode(nil, "node-1"); empty == nil {
		t.Error("filterPodsOnNode(nil) should return an empty slice, not nil")
	}
	if none := filterPodsOnNode(pods, "no-such-node"); len(none) != 0 {
		t.Errorf("unknown node = %d pods, want 0", len(none))
	}
}

func TestPodStatus(t *testing.T) {
	runningReady := pod("default", "a", "aaaaaaaa-0000-0000-0000-000000000000", "n", nil)
	runningReady.Status.Phase = corev1.PodRunning
	runningReady.Status.ContainerStatuses = []corev1.ContainerStatus{{Ready: true}}

	runningNotReady := pod("default", "a2", "a2aaaaaa-0000-0000-0000-000000000000", "n", nil)
	runningNotReady.Status.Phase = corev1.PodRunning
	runningNotReady.Status.ContainerStatuses = []corev1.ContainerStatus{{Ready: true}, {Ready: false}}

	pending := pod("default", "b", "bbbbbbbb-0000-0000-0000-000000000000", "n", nil)
	pending.Status.Phase = corev1.PodPending

	noStatuses := pod("default", "c", "cccccccc-0000-0000-0000-000000000000", "n", nil)
	noStatuses.Status.Phase = corev1.PodRunning

	tests := []struct {
		name string
		pod  *corev1.Pod
		want string
	}{
		{"nil pod", nil, "Missing/false"},
		{"running and ready", &runningReady, "Running/true"},
		{"running, one container not ready", &runningNotReady, "Running/false"},
		{"pending", &pending, "Pending/false"},
		{"running with no container statuses", &noStatuses, "Running/false"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PodStatus(tt.pod); got != tt.want {
				t.Errorf("PodStatus() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewAndNode(t *testing.T) {
	if got := New(nil, "node-9").Node(); got != "node-9" {
		t.Errorf("Node() = %q, want node-9", got)
	}
}

// TestNilClientsetContract asserts every method fails with ErrNoClient rather
// than panicking, which is what lets the binary run outside a cluster.
func TestNilClientsetContract(t *testing.T) {
	ctx := context.Background()
	c := New(nil, "node-1")

	if _, err := c.PodsOnNode(ctx); err != ErrNoClient {
		t.Errorf("PodsOnNode err = %v, want ErrNoClient", err)
	}
	if _, err := c.PodByUID(ctx, "aaaaaaaa-0000-0000-0000-000000000000"); err != ErrNoClient {
		t.Errorf("PodByUID err = %v, want ErrNoClient", err)
	}
	if _, err := c.FindPodByUID(ctx, "aaaaaaaa-0000-0000-0000-000000000000"); err != ErrNoClient {
		t.Errorf("FindPodByUID err = %v, want ErrNoClient", err)
	}
	if _, err := c.PodByName(ctx, "default", "x"); err != ErrNoClient {
		t.Errorf("PodByName err = %v, want ErrNoClient", err)
	}
	if err := c.DeletePod(ctx, "default", "x", 30); err != ErrNoClient {
		t.Errorf("DeletePod err = %v, want ErrNoClient", err)
	}
	if _, err := c.VolumeAttachmentsOnNode(ctx); err != ErrNoClient {
		t.Errorf("VolumeAttachmentsOnNode err = %v, want ErrNoClient", err)
	}
	if _, err := c.WaitPodReady(ctx, "default", "x", "u", time.Second, time.Millisecond); err != ErrNoClient {
		t.Errorf("WaitPodReady err = %v, want ErrNoClient", err)
	}
}

func TestPodsOnNode(t *testing.T) {
	fakeCS := fake.NewSimpleClientset(objs(
		pod("default", "on-node", "aaaaaaaa-0000-0000-0000-000000000000", "node-1", nil),
		pod("other", "off-node", "bbbbbbbb-0000-0000-0000-000000000000", "node-2", nil),
	)...)
	c := New(fakeCS, "node-1")

	got, err := c.PodsOnNode(context.Background())
	if err != nil {
		t.Fatalf("PodsOnNode: %v", err)
	}
	if len(got) != 1 || got[0].Name != "on-node" {
		t.Errorf("got %v, want just on-node", got)
	}
}

func TestPodByUID_EmptyUID(t *testing.T) {
	fakeCS := fake.NewSimpleClientset(objs(
		pod("default", "a", "aaaaaaaa-0000-0000-0000-000000000000", "node-1", nil),
	)...)
	c := New(fakeCS, "node-1")

	// An empty UID means the mount path never yielded one; it must not match a
	// pod whose UID field is also empty.
	if _, err := c.PodByUID(context.Background(), ""); !apierrors.IsNotFound(err) {
		t.Errorf("PodByUID(\"\") err = %v, want NotFound", err)
	}
}

func TestPodByUID(t *testing.T) {
	fakeCS := fake.NewSimpleClientset(objs(
		pod("default", "on-node", "bbbbbbbb-0000-0000-0000-000000000000", "node-1", nil),
		pod("other", "off-node", "cccccccc-0000-0000-0000-000000000000", "node-2", nil),
	)...)
	c := New(fakeCS, "node-1")

	got, err := c.PodByUID(context.Background(), "bbbbbbbb-0000-0000-0000-000000000000")
	if err != nil {
		t.Fatalf("PodByUID: %v", err)
	}
	if got.Name != "on-node" || got.Namespace != "default" {
		t.Errorf("got %s/%s, want default/on-node", got.Namespace, got.Name)
	}

	// The off-node pod exists in the clientset but is not on this node, so the
	// node scoping must keep it unreachable.
	if _, err := c.PodByUID(context.Background(), "cccccccc-0000-0000-0000-000000000000"); !apierrors.IsNotFound(err) {
		t.Errorf("PodByUID for off-node pod = %v, want NotFound", err)
	}

	if _, err := c.PodByUID(context.Background(), "99999999-0000-0000-0000-000000000000"); !apierrors.IsNotFound(err) {
		t.Errorf("PodByUID for unknown uid = %v, want NotFound", err)
	}
}

func TestDeletePod(t *testing.T) {
	fakeCS := fake.NewSimpleClientset(objs(
		pod("default", "gone", "aaaaaaaa-0000-0000-0000-000000000000", "node-1", nil),
	)...)
	c := New(fakeCS, "node-1")

	if err := c.DeletePod(context.Background(), "default", "gone", 30); err != nil {
		t.Fatalf("DeletePod: %v", err)
	}
	if _, err := c.PodByName(context.Background(), "default", "gone"); !apierrors.IsNotFound(err) {
		t.Errorf("pod still present after delete: %v", err)
	}
	if err := c.DeletePod(context.Background(), "default", "never-existed", 30); err == nil {
		t.Error("DeletePod of a missing pod should error")
	}
}

func TestVolumeAttachmentsOnNode(t *testing.T) {
	// The target cluster populates the classic spec.nodeName +
	// spec.source.persistentVolumeName pair; the CSI 1.30 attachedTo/volumeName
	// fields are empty there, so the test asserts against the classic shape.
	pvName := "pvc-aaaaaaaa-0000-0000-0000-000000000000"
	otherPV := "pvc-bbbbbbbb-0000-0000-0000-000000000000"
	fakeCS := fake.NewSimpleClientset(
		attachment("va-on-node", "node-1", &pvName),
		attachment("va-off-node", "node-2", &otherPV),
		attachment("va-inline", "node-1", nil), // nil PV name: inline ephemeral volume
	)
	c := New(fakeCS, "node-1")

	got, err := c.VolumeAttachmentsOnNode(context.Background())
	if err != nil {
		t.Fatalf("VolumeAttachmentsOnNode: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d attachments, want 2 (node-1 only)", len(got))
	}
	for _, va := range got {
		if va.Spec.NodeName != "node-1" {
			t.Errorf("attachment %s on node %q leaked", va.Name, va.Spec.NodeName)
		}
	}
}

func TestWithTimeout(t *testing.T) {
	ctx, cancel := WithTimeout()
	defer cancel()

	dl, ok := ctx.Deadline()
	if !ok {
		t.Fatal("WithTimeout context has no deadline")
	}
	if got := time.Until(dl); got <= 0 || got > k8sTimeout {
		t.Errorf("deadline = %s, want <= %s", got, k8sTimeout)
	}
}

func TestNewFromEnv_OutsideCluster(t *testing.T) {
	// Off a cluster there is no in-cluster config. The contract is a non-nil,
	// inert client plus an error, so main can warn and keep serving.
	c, err := NewFromEnv("node-1")
	if err == nil {
		t.Skip("running with in-cluster config available; nothing to assert")
	}
	if c == nil {
		t.Fatal("NewFromEnv returned a nil client alongside an error")
	}
	if c.Node() != "node-1" {
		t.Errorf("Node() = %q, want node-1", c.Node())
	}
	if _, err := c.PodsOnNode(context.Background()); err != ErrNoClient {
		t.Errorf("inert client PodsOnNode err = %v, want ErrNoClient", err)
	}
}
