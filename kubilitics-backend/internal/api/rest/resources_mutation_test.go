package rest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/kubilitics/kubilitics-backend/internal/config"
	"github.com/kubilitics/kubilitics-backend/internal/k8s"
	"github.com/kubilitics/kubilitics-backend/internal/models"
)

// docs/ai/STABILIZATION-PLAN.md Phase 2 item 1: resources.go's generic
// DeleteResource/PatchResource/ApplyManifest is the shared mutation path for
// every resource kind in the app, and had zero real test coverage — the
// existing DeleteResource tests in handler_resources_test.go use
// k8s.NewClientForTest, which leaves Client.Dynamic nil, so every call into
// the actual K8s-mutation code path panics and the test just recover()s and
// logs "panic expected" without ever exercising the real behavior. These
// tests wire up a working fake dynamic client so the mutation actually runs
// to completion and the result is asserted on, not just routing+validation.

// newMutableTestClient returns a *k8s.Client whose Dynamic field is a real
// (fake) dynamic client, so GetResource/PatchResource/DeleteResource/
// ApplyYAML's actual K8s-mutation code paths run instead of panicking on a
// nil Dynamic.
func newMutableTestClient(t *testing.T, objects ...runtime.Object) *k8s.Client {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("add corev1 to scheme: %v", err)
	}
	if err := appsv1.AddToScheme(scheme); err != nil {
		t.Fatalf("add appsv1 to scheme: %v", err)
	}
	dyn := dynamicfake.NewSimpleDynamicClient(scheme, objects...)
	client := k8s.NewClientForTest(fake.NewSimpleClientset())
	client.Dynamic = dyn
	return client
}

func newMutationTestHandler(t *testing.T, client *k8s.Client, clusterID string) (*Handler, *mux.Router) {
	t.Helper()
	cluster := &models.Cluster{ID: clusterID, Name: clusterID, Context: "test-ctx", Status: "connected"}
	mockService := &mockClusterServiceWithClient{clusters: []*models.Cluster{cluster}, client: client}
	h := NewHandler(mockService, nil, &config.Config{}, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	router := mux.NewRouter()
	api := router.PathPrefix("/api/v1").Subrouter()
	SetupRoutes(api, h)
	return h, router
}

func toUnstructuredDeployment(d *appsv1.Deployment) *unstructured.Unstructured {
	d.TypeMeta = metav1.TypeMeta{APIVersion: "apps/v1", Kind: "Deployment"}
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(d)
	if err != nil {
		panic(err)
	}
	return &unstructured.Unstructured{Object: obj}
}

func toUnstructuredPod(p *corev1.Pod) *unstructured.Unstructured {
	p.TypeMeta = metav1.TypeMeta{APIVersion: "v1", Kind: "Pod"}
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(p)
	if err != nil {
		panic(err)
	}
	return &unstructured.Unstructured{Object: obj}
}

func TestHandler_PatchResource_Success(t *testing.T) {
	clusterID := "test-cluster"
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "default"},
		Spec:       appsv1.DeploymentSpec{Replicas: int32Ptr(2)},
	}
	client := newMutableTestClient(t, toUnstructuredDeployment(deployment))
	_, router := newMutationTestHandler(t, client, clusterID)

	body := strings.NewReader(`{"spec":{"replicas":5}}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/clusters/"+clusterID+"/resources/deployments/default/web", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var result map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	resource, ok := result["resource"].(map[string]interface{})
	if !ok {
		t.Fatalf("response missing resource field: %v", result)
	}
	spec, ok := resource["spec"].(map[string]interface{})
	if !ok {
		t.Fatalf("resource missing spec field: %v", resource)
	}
	if replicas, ok := spec["replicas"].(float64); !ok || replicas != 5 {
		t.Errorf("expected replicas=5 in response, got %v", spec["replicas"])
	}

	// Verify the patch actually persisted against the dynamic client, not
	// just that the handler echoed back a locally-mutated copy.
	got, err := client.GetResource(req.Context(), "deployments", "default", "web")
	if err != nil {
		t.Fatalf("GetResource after patch: %v", err)
	}
	gotReplicas, found, err := unstructured.NestedInt64(got.Object, "spec", "replicas")
	if err != nil || !found {
		t.Fatalf("spec.replicas not found after patch: err=%v found=%v", err, found)
	}
	if gotReplicas != 5 {
		t.Errorf("expected persisted replicas=5, got %d", gotReplicas)
	}
}

func TestHandler_PatchResource_InvalidJSON(t *testing.T) {
	clusterID := "test-cluster"
	deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "default"}}
	client := newMutableTestClient(t, toUnstructuredDeployment(deployment))
	_, router := newMutationTestHandler(t, client, clusterID)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/clusters/"+clusterID+"/resources/deployments/default/web", strings.NewReader(`not json`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid JSON, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandler_PatchResource_EmptyBody(t *testing.T) {
	clusterID := "test-cluster"
	deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "default"}}
	client := newMutableTestClient(t, toUnstructuredDeployment(deployment))
	_, router := newMutationTestHandler(t, client, clusterID)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/clusters/"+clusterID+"/resources/deployments/default/web", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty patch body, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandler_PatchResource_NotFound(t *testing.T) {
	clusterID := "test-cluster"
	client := newMutableTestClient(t) // no objects seeded
	_, router := newMutationTestHandler(t, client, clusterID)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/clusters/"+clusterID+"/resources/deployments/default/ghost", strings.NewReader(`{"spec":{"replicas":1}}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for patching a nonexistent resource, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandler_DeleteResource_ActuallyDeletes(t *testing.T) {
	clusterID := "test-cluster"
	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "victim", Namespace: "default"}}
	client := newMutableTestClient(t, toUnstructuredPod(pod))
	_, router := newMutationTestHandler(t, client, clusterID)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/clusters/"+clusterID+"/resources/pods/default/victim", nil)
	req.Header.Set(DestructiveConfirmHeader, "true")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// The real regression check: the object must actually be gone from the
	// (fake) cluster, not just that the handler returned 200.
	if _, err := client.GetResource(req.Context(), "pods", "default", "victim"); err == nil {
		t.Error("expected pod to be deleted, but GetResource still found it")
	}
}

func TestHandler_DeleteResource_NotFound(t *testing.T) {
	clusterID := "test-cluster"
	client := newMutableTestClient(t) // nothing seeded
	_, router := newMutationTestHandler(t, client, clusterID)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/clusters/"+clusterID+"/resources/pods/default/ghost", nil)
	req.Header.Set(DestructiveConfirmHeader, "true")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for deleting a nonexistent resource, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandler_ApplyManifest_RequiresDestructiveHeader(t *testing.T) {
	clusterID := "test-cluster"
	client := newMutableTestClient(t)
	_, router := newMutationTestHandler(t, client, clusterID)

	body := `{"yaml":"apiVersion: v1\nkind: Pod\nmetadata:\n  name: x\n  namespace: default\n"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/clusters/"+clusterID+"/apply", strings.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 without destructive header, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandler_ApplyManifest_CreatesResource(t *testing.T) {
	clusterID := "test-cluster"
	client := newMutableTestClient(t) // empty cluster
	_, router := newMutationTestHandler(t, client, clusterID)

	yaml := "apiVersion: v1\nkind: Pod\nmetadata:\n  name: fresh-pod\n  namespace: default\nspec:\n  containers:\n  - name: c\n    image: nginx\n"
	reqBody, err := json.Marshal(map[string]string{"yaml": yaml})
	if err != nil {
		t.Fatalf("marshal request body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/clusters/"+clusterID+"/apply", strings.NewReader(string(reqBody)))
	req.Header.Set(DestructiveConfirmHeader, "true")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// The real regression check: the pod must actually exist afterward.
	if _, err := client.GetResource(req.Context(), "pods", "default", "fresh-pod"); err != nil {
		t.Errorf("expected applied pod to exist, GetResource failed: %v", err)
	}
}

func TestHandler_ApplyManifest_UpdatesExistingResource(t *testing.T) {
	clusterID := "test-cluster"
	existing := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "cfg", Namespace: "default"},
		Data:       map[string]string{"k": "old"},
	}
	existing.TypeMeta = metav1.TypeMeta{APIVersion: "v1", Kind: "ConfigMap"}
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(existing)
	if err != nil {
		t.Fatalf("convert to unstructured: %v", err)
	}
	client := newMutableTestClient(t, &unstructured.Unstructured{Object: obj})
	_, router := newMutationTestHandler(t, client, clusterID)

	yaml := "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: cfg\n  namespace: default\ndata:\n  k: new\n"
	reqBody, err := json.Marshal(map[string]string{"yaml": yaml})
	if err != nil {
		t.Fatalf("marshal request body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/clusters/"+clusterID+"/apply", strings.NewReader(string(reqBody)))
	req.Header.Set(DestructiveConfirmHeader, "true")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var result map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	resources, ok := result["resources"].([]interface{})
	if !ok || len(resources) != 1 {
		t.Fatalf("expected 1 applied resource, got %v", result["resources"])
	}
	first, ok := resources[0].(map[string]interface{})
	if !ok || first["action"] != "updated" {
		t.Errorf("expected action=updated for an existing resource, got %v", first)
	}

	got, err := client.GetResource(req.Context(), "configmaps", "default", "cfg")
	if err != nil {
		t.Fatalf("GetResource after apply: %v", err)
	}
	data, found, err := unstructured.NestedStringMap(got.Object, "data")
	if err != nil || !found {
		t.Fatalf("configmap data not found after apply: err=%v found=%v", err, found)
	}
	if data["k"] != "new" {
		t.Errorf("expected data.k=new after update, got %q", data["k"])
	}
}

func TestHandler_ApplyManifest_InvalidYAML(t *testing.T) {
	clusterID := "test-cluster"
	client := newMutableTestClient(t)
	_, router := newMutationTestHandler(t, client, clusterID)

	reqBody, err := json.Marshal(map[string]string{"yaml": "not: valid: yaml: : :"})
	if err != nil {
		t.Fatalf("marshal request body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/clusters/"+clusterID+"/apply", strings.NewReader(string(reqBody)))
	req.Header.Set(DestructiveConfirmHeader, "true")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code == http.StatusOK {
		t.Errorf("expected a non-200 response for invalid YAML, got 200: %s", rec.Body.String())
	}
}

func int32Ptr(v int32) *int32 { return &v }
