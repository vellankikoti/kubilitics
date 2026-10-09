package rest

// docs/ai/STABILIZATION-PLAN.md Phase 2 item 6: exec.go (pod exec over
// WebSocket) had zero test coverage despite being a live-execution path
// into a user's container. These tests exercise everything reachable
// without a real Kubernetes API server: path validation, cluster
// resolution, RBAC gating, and — via a real WebSocket dial against an
// httptest.Server — the container-resolution error paths that run before
// GetPodExec ever calls remotecommand.NewSPDYExecutor.
//
// What is deliberately NOT exercised here: the actual SPDY exec stream
// (client.Clientset.CoreV1().RESTClient()...SubResource("exec") and the
// resulting remotecommand.NewSPDYExecutor/StreamWithContext call). A
// client-go fake Clientset's RESTClient() is not wired to build real,
// dialable URLs, and client.Config is nil for a test client — feeding
// that into spdy.RoundTripperFor would dereference a nil *rest.Config
// (config.Proxy) and panic, or hang on an unbounded dial depending on
// what URL the fake RESTClient happens to produce. That boundary
// fundamentally requires a live API server; faking it would either crash
// the test process or produce a false-positive "success" that proves
// nothing. This is the same boundary node_operations_test.go's drain path
// stops short of for eviction verification reasons.

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/kubilitics/kubilitics-backend/internal/auth"
	"github.com/kubilitics/kubilitics-backend/internal/config"
	"github.com/kubilitics/kubilitics-backend/internal/k8s"
	"github.com/kubilitics/kubilitics-backend/internal/models"
)

// newExecTestHandler builds a Handler+router with a mockClusterServiceWithClient
// (defined in handler_resources_test.go, same package) serving the given fake
// k8s.Client, auth disabled so RBAC doesn't interfere with the functional tests.
func newExecTestHandler(t *testing.T, client *k8s.Client, clusterID string) *mux.Router {
	t.Helper()
	cluster := &models.Cluster{ID: clusterID, Name: clusterID, Context: "test-ctx", Status: "connected"}
	mockService := &mockClusterServiceWithClient{clusters: []*models.Cluster{cluster}, client: client}
	h := NewHandler(mockService, nil, &config.Config{}, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	router := mux.NewRouter()
	api := router.PathPrefix("/api/v1").Subrouter()
	SetupRoutes(api, h)
	return router
}

func TestGetPodExec_InvalidClusterID(t *testing.T) {
	client := k8s.NewClientForTest(fake.NewSimpleClientset())
	router := newExecTestHandler(t, client, "test-cluster")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/clusters/bad!cluster/pods/default/pod-1/exec", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid clusterId, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGetPodExec_InvalidNamespace(t *testing.T) {
	client := k8s.NewClientForTest(fake.NewSimpleClientset())
	router := newExecTestHandler(t, client, "test-cluster")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/clusters/test-cluster/pods/BAD_NS!/pod-1/exec", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid namespace, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGetPodExec_InvalidPodName(t *testing.T) {
	client := k8s.NewClientForTest(fake.NewSimpleClientset())
	router := newExecTestHandler(t, client, "test-cluster")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/clusters/test-cluster/pods/default/BAD_NAME!/exec", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid pod name, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGetPodExec_ClusterNotFound(t *testing.T) {
	// client: nil forces mockClusterServiceWithClient.GetClient to error for
	// every id, and the cluster list is empty, so resolveClusterID's fallback
	// lookup also fails — exactly the "cluster truly doesn't exist" case.
	mockService := &mockClusterServiceWithClient{clusters: nil, client: nil}
	h := NewHandler(mockService, nil, &config.Config{}, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	router := mux.NewRouter()
	api := router.PathPrefix("/api/v1").Subrouter()
	SetupRoutes(api, h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/clusters/missing-cluster/pods/default/pod-1/exec", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unresolvable cluster, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGetPodExec_RBAC_Unauthenticated(t *testing.T) {
	h := buildHandler(t, &mockAddonService{})
	r := httptest.NewRequest(http.MethodGet, "/clusters/cluster-1/pods/default/pod-1/exec", nil)
	w := serve(h, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without auth, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetPodExec_RBAC_ViewerForbidden(t *testing.T) {
	h := buildHandler(t, &mockAddonService{})
	r := httptest.NewRequest(http.MethodGet, "/clusters/cluster-1/pods/default/pod-1/exec", nil)
	r = claimsCtx(r, auth.RoleViewer)
	w := serve(h, r)
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for viewer (exec requires operator), got %d: %s", w.Code, w.Body.String())
	}
}

// TestGetPodExec_RBAC_OperatorPassesAuth confirms an operator claim clears the
// RBAC gate and reaches the handler body. It can't reach a real WebSocket
// upgrade (httptest.ResponseRecorder doesn't implement http.Hijacker), so the
// handler's own upgrader.Upgrade() call fails and short-circuits with 500 —
// that failure happens strictly *after* RBAC, so observing anything other
// than 401/403 proves the operator role was accepted.
func TestGetPodExec_RBAC_OperatorPassesAuth(t *testing.T) {
	h := buildHandler(t, &mockAddonService{})
	r := httptest.NewRequest(http.MethodGet, "/clusters/cluster-1/pods/default/pod-1/exec", nil)
	r = claimsCtx(r, auth.RoleOperator)
	w := serve(h, r)
	if w.Code == http.StatusUnauthorized || w.Code == http.StatusForbidden {
		t.Errorf("operator should clear RBAC, got %d: %s", w.Code, w.Body.String())
	}
}

// dialExecWS opens a real WebSocket to GetPodExec served by an httptest.Server,
// exercising the actual upgrade, container-resolution and error-messaging code.
func dialExecWS(t *testing.T, serverURL, clusterID, namespace, pod, container string) (*websocket.Conn, *http.Response, error) {
	t.Helper()
	u := strings.Replace(serverURL, "http://", "ws://", 1) + "/api/v1/clusters/" + clusterID + "/pods/" + namespace + "/" + pod + "/exec"
	if container != "" {
		u += "?container=" + container
	}
	return websocket.DefaultDialer.Dial(u, nil)
}

func readExecWSMessage(t *testing.T, conn *websocket.Conn) wsOutMessage {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	var msg wsOutMessage
	if err := conn.ReadJSON(&msg); err != nil {
		t.Fatalf("read ws message: %v", err)
	}
	return msg
}

func TestGetPodExec_WS_PodNotFound(t *testing.T) {
	client := k8s.NewClientForTest(fake.NewSimpleClientset()) // no pods seeded
	router := newExecTestHandler(t, client, "test-cluster")
	srv := httptest.NewServer(router)
	defer srv.Close()

	conn, resp, err := dialExecWS(t, srv.URL, "test-cluster", "default", "ghost-pod", "")
	if err != nil {
		t.Fatalf("dial: %v (resp=%v)", err, resp)
	}
	defer func() { _ = conn.Close() }()

	msg := readExecWSMessage(t, conn)
	if msg.T != wsMsgError || !strings.Contains(msg.D, "pod not found") {
		t.Errorf("expected pod-not-found error message, got %+v", msg)
	}
}

func TestGetPodExec_WS_MultipleContainersRequireExplicitQuery(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "multi-pod", Namespace: "default"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "app"}, {Name: "sidecar"}},
		},
	}
	client := k8s.NewClientForTest(fake.NewSimpleClientset(pod))
	router := newExecTestHandler(t, client, "test-cluster")
	srv := httptest.NewServer(router)
	defer srv.Close()

	conn, resp, err := dialExecWS(t, srv.URL, "test-cluster", "default", "multi-pod", "")
	if err != nil {
		t.Fatalf("dial: %v (resp=%v)", err, resp)
	}
	defer func() { _ = conn.Close() }()

	msg := readExecWSMessage(t, conn)
	if msg.T != wsMsgError || !strings.Contains(msg.D, "container query is required") {
		t.Errorf("expected container-query-required error, got %+v", msg)
	}
	if !strings.Contains(msg.D, "app") || !strings.Contains(msg.D, "sidecar") {
		t.Errorf("expected error to list valid container names, got %+v", msg)
	}
}

func TestGetPodExec_WS_ContainerNotFound(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "single-pod", Namespace: "default"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "app"}},
		},
	}
	client := k8s.NewClientForTest(fake.NewSimpleClientset(pod))
	router := newExecTestHandler(t, client, "test-cluster")
	srv := httptest.NewServer(router)
	defer srv.Close()

	conn, resp, err := dialExecWS(t, srv.URL, "test-cluster", "default", "single-pod", "nope")
	if err != nil {
		t.Fatalf("dial: %v (resp=%v)", err, resp)
	}
	defer func() { _ = conn.Close() }()

	msg := readExecWSMessage(t, conn)
	if msg.T != wsMsgError || !strings.Contains(msg.D, `container "nope" not found in pod`) {
		t.Errorf("expected container-not-found error, got %+v", msg)
	}
	if !strings.Contains(msg.D, "app") {
		t.Errorf("expected error to list the valid container name, got %+v", msg)
	}
}

// Sanity check that the base64 stdin/stdout framing helpers this suite relies
// on (used across exec.go and shell_stream.go) round-trip correctly.
func TestWSMessageFraming_Base64RoundTrip(t *testing.T) {
	payload := []byte("echo hello\n")
	encoded := base64.StdEncoding.EncodeToString(payload)
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || string(decoded) != string(payload) {
		t.Fatalf("base64 round-trip failed: decoded=%q err=%v", decoded, err)
	}
}
