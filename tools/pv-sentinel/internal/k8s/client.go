// Package k8s wraps the Kubernetes API surface pv-sentinel needs: resolving a
// kubelet pod UID back to a real pod, deleting pods, and reading CSI
// VolumeAttachments.
//
// The clientset is stored as kubernetes.Interface rather than the concrete
// *kubernetes.Clientset so tests can inject client-go's fake clientset. A nil
// clientset is legal and makes every call fail with ErrNoClient, which lets the
// binary start up and serve /healthz on a machine that is not in a cluster.
package k8s

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// k8sTimeout is the default timeout for Kubernetes API calls.
const k8sTimeout = 30 * time.Second

// ErrNoClient is returned by every method when the client was built without a
// clientset, i.e. when the process is not running inside a cluster.
var ErrNoClient = errors.New("kubernetes client not initialized - cannot query the API")

// Client is the Kubernetes surface pv-sentinel uses, scoped to one node.
type Client struct {
	clientset kubernetes.Interface
	node      string
}

// New returns a Client for an existing clientset. Passing a nil clientset is
// allowed; every call then returns ErrNoClient.
func New(clientset kubernetes.Interface, node string) *Client {
	return &Client{clientset: clientset, node: node}
}

// NewFromEnv builds a Client from the in-cluster service account config.
//
// It follows the in-cluster-first pattern: when the config is unavailable the
// returned Client is non-nil but inert, and the error explains why. Callers can
// therefore log a warning and keep running in report-only mode rather than
// crashing on a laptop or a node with a broken mount.
func NewFromEnv(node string) (*Client, error) {
	config, err := rest.InClusterConfig()
	if err == nil {
		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			return nil, fmt.Errorf("failed to create clientset: %w", err)
		}
		return New(clientset, node), nil
	}
	slog.Warn("Not running in Kubernetes cluster, API operations will fail", "error", err)
	return New(nil, node), fmt.Errorf("kubernetes not available: %w", err)
}

// Node returns the node this client is scoped to.
func (c *Client) Node() string { return c.node }

// filterPodsOnNode keeps only pods scheduled onto node. It is separated from
// the API call because client-go's fake clientset ignores field selectors:
// filtering in Go makes the behaviour identical in production and in tests.
func filterPodsOnNode(pods []corev1.Pod, node string) []corev1.Pod {
	out := make([]corev1.Pod, 0, len(pods))
	for _, p := range pods {
		if p.Spec.NodeName == node {
			out = append(out, p)
		}
	}
	return out
}

// PodsOnNode returns the pods scheduled onto this client's node, across all
// namespaces.
func (c *Client) PodsOnNode(ctx context.Context) ([]corev1.Pod, error) {
	if c.clientset == nil {
		return nil, ErrNoClient
	}
	list, err := c.clientset.CoreV1().Pods(corev1.NamespaceAll).List(ctx, metav1.ListOptions{
		FieldSelector: "spec.nodeName=" + c.node,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods on node %s: %w", c.node, err)
	}
	return filterPodsOnNode(list.Items, c.node), nil
}

// PodByUID resolves a kubelet pod UID to the pod that owns it. The kubelet path
// /var/lib/kubelet/pods/<uid>/volumes/... carries the UID but not the name or
// namespace, and the PV name is not the PVC name, so UID is the only reliable
// join key between a mount and a pod.
func (c *Client) PodByUID(ctx context.Context, uid string) (*corev1.Pod, error) {
	if c.clientset == nil {
		return nil, ErrNoClient
	}
	if uid == "" {
		return nil, apierrors.NewNotFound(corev1.Resource("pod"), "")
	}
	pods, err := c.PodsOnNode(ctx)
	if err != nil {
		return nil, err
	}
	for i := range pods {
		if string(pods[i].UID) == uid {
			return &pods[i], nil
		}
	}
	return nil, apierrors.NewNotFound(corev1.Resource("pod"), uid)
}

// PodByName fetches a pod by namespace and name.
func (c *Client) PodByName(ctx context.Context, namespace, name string) (*corev1.Pod, error) {
	if c.clientset == nil {
		return nil, ErrNoClient
	}
	pod, err := c.clientset.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get pod %s/%s: %w", namespace, name, err)
	}
	return pod, nil
}

// DeletePod deletes a pod with the given grace period.
func (c *Client) DeletePod(ctx context.Context, namespace, name string, graceSeconds int64) error {
	if c.clientset == nil {
		return ErrNoClient
	}
	err := c.clientset.CoreV1().Pods(namespace).Delete(ctx, name, metav1.DeleteOptions{
		GracePeriodSeconds: &graceSeconds,
	})
	if err != nil {
		return fmt.Errorf("failed to delete pod %s/%s: %w", namespace, name, err)
	}
	return nil
}

// PodStatus renders a pod as "phase/allContainersReady", e.g. "Running/true".
// A pod that reports no container statuses is reported not-ready, since that is
// the state before a recreated container has started.
func PodStatus(pod *corev1.Pod) string {
	if pod == nil {
		return "Missing/false"
	}
	ready := false
	if pod.Status.Phase == corev1.PodRunning && len(pod.Status.ContainerStatuses) > 0 {
		ready = true
		for _, cs := range pod.Status.ContainerStatuses {
			if !cs.Ready {
				ready = false
				break
			}
		}
	}
	return fmt.Sprintf("%s/%t", pod.Status.Phase, ready)
}

// FindPodByUID is the API-backed lookup used during remediation. It exists so
// callers get a single entry point that respects the clientset-nil contract.
func (c *Client) FindPodByUID(ctx context.Context, uid string) (*corev1.Pod, error) {
	return c.PodByUID(ctx, uid)
}

// WaitPodReady polls until a pod with the given name exists with a UID different
// from deletedUID, is Running, and has every container ready — or until the
// timeout elapses. It returns the last observed status string.
//
// Waiting on a *different* UID is the point: a controller recreates the pod with
// a fresh UID, and observing the old pod still in Terminating would otherwise be
// mistaken for a successful recreation.
func (c *Client) WaitPodReady(ctx context.Context, namespace, name, deletedUID string, timeout, interval time.Duration) (string, error) {
	if c.clientset == nil {
		return "", ErrNoClient
	}
	deadline := time.Now().Add(timeout)
	last := "Missing/false"
	for {
		pod, err := c.PodByName(ctx, namespace, name)
		if err == nil {
			last = PodStatus(pod)
			if string(pod.UID) != deletedUID && pod.Status.Phase == corev1.PodRunning {
				if PodStatus(pod) == "Running/true" {
					return last, nil
				}
			}
		} else if !apierrors.IsNotFound(err) {
			return last, err
		}
		if time.Now().After(deadline) {
			return last, fmt.Errorf("pod %s/%s did not become ready within %s (last=%s)", namespace, name, timeout, last)
		}
		select {
		case <-ctx.Done():
			return last, ctx.Err()
		case <-time.After(interval):
		}
	}
}

// VolumeAttachmentsOnNode returns CSI VolumeAttachments whose spec names this
// client's node.
//
// This is the only API-side source of the PV-to-node mapping on the target
// cluster: the PersistentVolumes themselves carry no nodeAffinity, and the
// newer status.attachedTo field is unpopulated, so both read as unbound.
func (c *Client) VolumeAttachmentsOnNode(ctx context.Context) ([]storagev1.VolumeAttachment, error) {
	if c.clientset == nil {
		return nil, ErrNoClient
	}
	list, err := c.clientset.StorageV1().VolumeAttachments().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list volume attachments: %w", err)
	}
	out := make([]storagev1.VolumeAttachment, 0, len(list.Items))
	for _, va := range list.Items {
		if va.Spec.NodeName == c.node {
			out = append(out, va)
		}
	}
	return out, nil
}

// PersistentVolumes lists every PersistentVolume.
//
// The list is cluster-scoped and unfiltered because the join the resize detector
// needs is VolumeAttachment -> PV name -> PV, and a field selector on a name we
// have already listed would be a second round-trip for nothing. The fake
// clientset honours a plain List, so this path is identical in tests.
func (c *Client) PersistentVolumes(ctx context.Context) ([]corev1.PersistentVolume, error) {
	if c.clientset == nil {
		return nil, ErrNoClient
	}
	list, err := c.clientset.CoreV1().PersistentVolumes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list persistent volumes: %w", err)
	}
	return list.Items, nil
}

// PersistentVolumeClaims lists every PersistentVolumeClaim across all
// namespaces.
//
// Cluster-scoped for the same reason as PersistentVolumes: the claim a PV points
// at is named by the PV, so filtering by namespace here would only cost a query
// per namespace. It is also the reason rbac.yaml needs persistentvolumeclaims
// get/list — the role had persistentvolumes but not the claims.
func (c *Client) PersistentVolumeClaims(ctx context.Context) ([]corev1.PersistentVolumeClaim, error) {
	if c.clientset == nil {
		return nil, ErrNoClient
	}
	list, err := c.clientset.CoreV1().PersistentVolumeClaims(corev1.NamespaceAll).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list persistent volume claims: %w", err)
	}
	return list.Items, nil
}

// WithTimeout wraps a background context with the package-wide API timeout.
func WithTimeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), k8sTimeout)
}
