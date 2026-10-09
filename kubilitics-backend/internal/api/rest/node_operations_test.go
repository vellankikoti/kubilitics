package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"

	"github.com/kubilitics/kubilitics-backend/internal/config"
	"github.com/kubilitics/kubilitics-backend/internal/k8s"
	"github.com/kubilitics/kubilitics-backend/internal/models"
)

// docs/ai/STABILIZATION-PLAN.md Phase 2 item 6: node_operations.go
// (cordon/drain — "a bad drain can cause an outage") had zero test
// coverage. Writing these tests also surfaced a real gap: DrainNode did
// not require X-Confirm-Destructive, unlike DeleteResource/ApplyManifest,
// despite being at least as destructive — fixed in the same change as
// these tests (see node_operations.go).

// newNodeOpsTestClient builds a *k8s.Client with both a working typed
// Clientset (seeded with pods, for List/Evict/Delete) and a working
// dynamic client (seeded with the node, for PatchResource during cordon).
func newNodeOpsTestClient(t *testing.T, node *corev1.Node, pods ...*corev1.Pod) *k8s.Client {
	t.Helper()
	podObjs := make([]runtime.Object, 0, len(pods))
	for _, p := range pods {
		podObjs = append(podObjs, p)
	}
	clientset := fake.NewSimpleClientset(podObjs...)
	// client-go's fake EvictV1 only records the "create .../eviction"
	// action (see fake_pod_expansion.go) — it never actually removes the
	// pod, unlike a real API server's eviction handler. Without this
	// reactor, DrainNode's eviction path would appear to succeed in every
	// test but never be verifiable as having actually removed anything.
	// Deletes via the Tracker directly (not clientset.CoreV1()...Delete())
	// — calling back into the Clientset's own mutating methods from inside
	// a reactor re-enters Fake.Invokes' locking and deadlocks.
	podsGVR := corev1.SchemeGroupVersion.WithResource("pods")
	clientset.PrependReactor("create", "pods", func(action ktesting.Action) (bool, runtime.Object, error) {
		if action.GetSubresource() != "eviction" {
			return false, nil, nil
		}
		createAction, ok := action.(ktesting.CreateAction)
		if !ok {
			return false, nil, nil
		}
		eviction, ok := createAction.GetObject().(*policyv1.Eviction)
		if !ok {
			return false, nil, nil
		}
		if err := clientset.Tracker().Delete(podsGVR, eviction.Namespace, eviction.Name); err != nil {
			return true, nil, err
		}
		return true, eviction, nil
	})

	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("add corev1 to scheme: %v", err)
	}
	if err := appsv1.AddToScheme(scheme); err != nil {
		t.Fatalf("add appsv1 to scheme: %v", err)
	}
	node.TypeMeta = metav1.TypeMeta{APIVersion: "v1", Kind: "Node"}
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(node)
	if err != nil {
		t.Fatalf("convert node to unstructured: %v", err)
	}
	dyn := dynamicfake.NewSimpleDynamicClient(scheme, &unstructured.Unstructured{Object: obj})

	client := k8s.NewClientForTest(clientset)
	client.Dynamic = dyn
	return client
}

func newNodeOpsTestHandler(t *testing.T, client *k8s.Client, clusterID string) *mux.Router {
	t.Helper()
	cluster := &models.Cluster{ID: clusterID, Name: clusterID, Context: "test-ctx", Status: "connected"}
	mockService := &mockClusterServiceWithClient{clusters: []*models.Cluster{cluster}, client: client}
	h := NewHandler(mockService, nil, &config.Config{}, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	router := mux.NewRouter()
	api := router.PathPrefix("/api/v1").Subrouter()
	SetupRoutes(api, h)
	return router
}

func TestHandler_CordonNode_Success(t *testing.T) {
	clusterID := "test-cluster"
	node := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node-1"}}
	client := newNodeOpsTestClient(t, node)
	router := newNodeOpsTestHandler(t, client, clusterID)

	body, _ := json.Marshal(CordonNodeRequest{Unschedulable: true})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/clusters/"+clusterID+"/resources/nodes/node-1/cordon", strings.NewReader(string(body)))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	got, err := client.GetResource(context.Background(), "nodes", "", "node-1")
	if err != nil {
		t.Fatalf("GetResource after cordon: %v", err)
	}
	unschedulable, found, err := unstructured.NestedBool(got.Object, "spec", "unschedulable")
	if err != nil || !found || !unschedulable {
		t.Errorf("expected spec.unschedulable=true after cordon, found=%v err=%v value=%v", found, err, unschedulable)
	}
}

func TestHandler_CordonNode_Uncordon(t *testing.T) {
	clusterID := "test-cluster"
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "node-1"},
		Spec:       corev1.NodeSpec{Unschedulable: true},
	}
	client := newNodeOpsTestClient(t, node)
	router := newNodeOpsTestHandler(t, client, clusterID)

	body, _ := json.Marshal(CordonNodeRequest{Unschedulable: false})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/clusters/"+clusterID+"/resources/nodes/node-1/uncordon", strings.NewReader(string(body)))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	got, err := client.GetResource(context.Background(), "nodes", "", "node-1")
	if err != nil {
		t.Fatalf("GetResource after uncordon: %v", err)
	}
	unschedulable, _, _ := unstructured.NestedBool(got.Object, "spec", "unschedulable")
	if unschedulable {
		t.Error("expected spec.unschedulable=false after uncordon")
	}
}

func TestHandler_CordonNode_NotFound(t *testing.T) {
	clusterID := "test-cluster"
	// Seed a different node, so GVR resolution works but the named node 404s.
	client := newNodeOpsTestClient(t, &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "other-node"}})
	router := newNodeOpsTestHandler(t, client, clusterID)

	body, _ := json.Marshal(CordonNodeRequest{Unschedulable: true})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/clusters/"+clusterID+"/resources/nodes/ghost/cordon", strings.NewReader(string(body)))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for cordoning a nonexistent node, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandler_DrainNode_RequiresDestructiveHeader(t *testing.T) {
	clusterID := "test-cluster"
	node := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node-1"}}
	client := newNodeOpsTestClient(t, node)
	router := newNodeOpsTestHandler(t, client, clusterID)

	body, _ := json.Marshal(DrainNodeRequest{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/clusters/"+clusterID+"/resources/nodes/node-1/drain", strings.NewReader(string(body)))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 without destructive header, got %d: %s", rec.Code, rec.Body.String())
	}

	// Confirm the node was never even cordoned — the header check must run
	// before any mutation, not just before the response.
	got, err := client.GetResource(context.Background(), "nodes", "", "node-1")
	if err != nil {
		t.Fatalf("GetResource: %v", err)
	}
	unschedulable, _, _ := unstructured.NestedBool(got.Object, "spec", "unschedulable")
	if unschedulable {
		t.Error("expected the node to remain uncordoned when drain was rejected for missing the header")
	}
}

func TestHandler_DrainNode_EvictsEligiblePods_SkipsDaemonSetAndTerminal(t *testing.T) {
	clusterID := "test-cluster"
	node := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node-1"}}

	regularPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "app-pod", Namespace: "default",
			OwnerReferences: []metav1.OwnerReference{{Kind: "ReplicaSet", Name: "app-rs"}},
		},
		Spec:   corev1.PodSpec{NodeName: "node-1"},
		Status: corev1.PodStatus{Phase: corev1.PodRunning},
	}
	dsPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ds-pod", Namespace: "kube-system",
			OwnerReferences: []metav1.OwnerReference{{Kind: "DaemonSet", Name: "node-exporter"}},
		},
		Spec:   corev1.PodSpec{NodeName: "node-1"},
		Status: corev1.PodStatus{Phase: corev1.PodRunning},
	}
	terminalPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "done-job", Namespace: "default"},
		Spec:       corev1.PodSpec{NodeName: "node-1"},
		Status:     corev1.PodStatus{Phase: corev1.PodSucceeded},
	}
	// Note: a pod scheduled on a different node is deliberately NOT included
	// here — client-go's fake ObjectTracker does not honor FieldSelector at
	// all (confirmed: k8s.io/client-go/testing's List never references it),
	// so a fake-backed test cannot verify DrainNode's "spec.nodeName=<node>"
	// scoping actually filters server-side. That's a real K8s API server
	// behavior this test harness cannot exercise, not a gap in this test.

	client := newNodeOpsTestClient(t, node, regularPod, dsPod, terminalPod)
	router := newNodeOpsTestHandler(t, client, clusterID)

	body, _ := json.Marshal(DrainNodeRequest{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/clusters/"+clusterID+"/resources/nodes/node-1/drain", strings.NewReader(string(body)))
	req.Header.Set(DestructiveConfirmHeader, "true")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp DrainNodeResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(resp.Evicted) != 1 || resp.Evicted[0] != "default/app-pod" {
		t.Errorf("expected exactly the regular pod to be evicted, got %v", resp.Evicted)
	}
	skippedJoined := strings.Join(resp.Skipped, "|")
	if !strings.Contains(skippedJoined, "ds-pod") {
		t.Errorf("expected the DaemonSet pod to be skipped, got %v", resp.Skipped)
	}
	if !strings.Contains(skippedJoined, "done-job") {
		t.Errorf("expected the terminal pod to be skipped, got %v", resp.Skipped)
	}

	// The node must also end up cordoned as a side effect of drain.
	got, err := client.GetResource(context.Background(), "nodes", "", "node-1")
	if err != nil {
		t.Fatalf("GetResource after drain: %v", err)
	}
	unschedulable, _, _ := unstructured.NestedBool(got.Object, "spec", "unschedulable")
	if !unschedulable {
		t.Error("expected the node to be cordoned as part of drain")
	}

	// The evicted pod must actually be gone from the fake cluster.
	if _, err := client.Clientset.CoreV1().Pods("default").Get(context.Background(), "app-pod", metav1.GetOptions{}); err == nil {
		t.Error("expected the evicted pod to be deleted from the cluster")
	}
	// The skipped pods must still exist, untouched.
	if _, err := client.Clientset.CoreV1().Pods("kube-system").Get(context.Background(), "ds-pod", metav1.GetOptions{}); err != nil {
		t.Errorf("expected the DaemonSet pod to still exist, got: %v", err)
	}
}

func TestHandler_DrainNode_WithoutControllerIsSkippedUnlessForced(t *testing.T) {
	clusterID := "test-cluster"
	node := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node-1"}}
	bareStaticPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "bare-pod", Namespace: "default"}, // no OwnerReferences
		Spec:       corev1.PodSpec{NodeName: "node-1"},
		Status:     corev1.PodStatus{Phase: corev1.PodRunning},
	}
	client := newNodeOpsTestClient(t, node, bareStaticPod)
	router := newNodeOpsTestHandler(t, client, clusterID)

	body, _ := json.Marshal(DrainNodeRequest{Force: false})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/clusters/"+clusterID+"/resources/nodes/node-1/drain", strings.NewReader(string(body)))
	req.Header.Set(DestructiveConfirmHeader, "true")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	var resp DrainNodeResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Evicted) != 0 {
		t.Errorf("expected an unmanaged pod to be skipped without force=true, got evicted=%v", resp.Evicted)
	}
	if _, err := client.Clientset.CoreV1().Pods("default").Get(context.Background(), "bare-pod", metav1.GetOptions{}); err != nil {
		t.Errorf("expected the unmanaged pod to still exist without force=true, got: %v", err)
	}
}
