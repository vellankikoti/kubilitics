package rest

// docs/ai/STABILIZATION-PLAN.md Phase 2 item 6: portforward.go (real
// kubectl port-forward subprocess management) had zero test coverage.
// These tests cover request validation, cluster resolution, RBAC gating,
// the session store (pfStore/pfLookup/pfDelete/session.stop), and the
// "subprocess exits before the port opens" error path — driven with a
// deliberately broken KUBECONFIG so a real `kubectl` on PATH fails fast
// and deterministically, without ever needing a live cluster.
//
// What is NOT exercised: a genuinely successful port-forward (kubectl
// actually binding the local port and proxying to a real pod). That
// requires a live API server and a schedulable pod — fundamentally an
// integration-test/live-cluster concern, not a unit-testable one.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"

	"github.com/kubilitics/kubilitics-backend/internal/auth"
	"github.com/kubilitics/kubilitics-backend/internal/config"
	"github.com/kubilitics/kubilitics-backend/internal/models"
)

func newPortForwardTestHandler(t *testing.T, clusters []*models.Cluster) *mux.Router {
	t.Helper()
	mockService := &mockClusterServiceWithClient{clusters: clusters}
	h := NewHandler(mockService, nil, &config.Config{}, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	router := mux.NewRouter()
	api := router.PathPrefix("/api/v1").Subrouter()
	SetupRoutes(api, h)
	return router
}

// freePort asks the OS for a currently-unused TCP port on localhost.
func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen for free port: %v", err)
	}
	defer func() { _ = l.Close() }()
	return l.Addr().(*net.TCPAddr).Port
}

func postPortForwardRequest(clusterID string, req portForwardStartReq) *http.Request {
	body, _ := json.Marshal(req)
	return httptest.NewRequest(http.MethodPost, "/api/v1/clusters/"+clusterID+"/port-forward", bytes.NewReader(body))
}

func TestPostPortForward_InvalidClusterID(t *testing.T) {
	router := newPortForwardTestHandler(t, nil)
	req := postPortForwardRequest("bad!cluster", portForwardStartReq{Name: "pod-1", Namespace: "default", LocalPort: 18080, RemotePort: 80})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestPostPortForward_ClusterNotResolvable(t *testing.T) {
	router := newPortForwardTestHandler(t, nil)
	req := postPortForwardRequest("ghost-cluster", portForwardStartReq{Name: "pod-1", Namespace: "default", LocalPort: 18080, RemotePort: 80})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestPostPortForward_MissingFields(t *testing.T) {
	cluster := &models.Cluster{ID: "c1", KubeconfigPath: "/tmp/kc"}
	router := newPortForwardTestHandler(t, []*models.Cluster{cluster})

	cases := []portForwardStartReq{
		{Name: "", Namespace: "default", LocalPort: 18080, RemotePort: 80},
		{Name: "pod-1", Namespace: "", LocalPort: 18080, RemotePort: 80},
		{Name: "pod-1", Namespace: "default", LocalPort: 0, RemotePort: 80},
		{Name: "pod-1", Namespace: "default", LocalPort: 18080, RemotePort: 0},
	}
	for i, tc := range cases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			req := postPortForwardRequest("c1", tc)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400 for %+v, got %d: %s", tc, rec.Code, rec.Body.String())
			}
		})
	}
}

// TestPostPortForward_KubectlExitsPrematurely drives the real subprocess
// path with a KubeconfigPath that doesn't exist, so a real kubectl binary
// on PATH fails to load its config and exits almost immediately — the
// handler's procExited race against the 2s port-probe loop resolves to
// "exited prematurely" deterministically, regardless of cluster reachability.
func TestPostPortForward_KubectlExitsPrematurely(t *testing.T) {
	if _, err := net.LookupHost("localhost"); err != nil {
		t.Skip("no local network resolution available")
	}
	cluster := &models.Cluster{
		ID:             "c1",
		KubeconfigPath: "/nonexistent/kubilitics-test-kubeconfig-does-not-exist",
	}
	router := newPortForwardTestHandler(t, []*models.Cluster{cluster})

	req := postPortForwardRequest("c1", portForwardStartReq{
		Name:       "pod-1",
		Namespace:  "default",
		LocalPort:  freePort(t),
		RemotePort: 80,
	})
	rec := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		router.ServeHTTP(rec, req)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("PostPortForward did not return within 10s — kubectl may be missing from PATH or hung")
	}

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for a kubectl process that exits before opening the port, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "port-forward") {
		t.Errorf("expected error message to mention port-forward, got %s", rec.Body.String())
	}
}

func TestDeletePortForward_NotFound(t *testing.T) {
	cluster := &models.Cluster{ID: "c1", KubeconfigPath: "/tmp/kc"}
	router := newPortForwardTestHandler(t, []*models.Cluster{cluster})

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/clusters/c1/port-forward/no-such-session", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown session, got %d: %s", rec.Code, rec.Body.String())
	}
}

// TestDeletePortForward_StopsSession exercises the real session store and
// stop() logic directly: a fake session (no subprocess) is registered via
// the package-level store, and DeletePortForward must cancel it, mark it
// stopped, and remove it from the store.
func TestDeletePortForward_StopsSession(t *testing.T) {
	cluster := &models.Cluster{ID: "pf-stop-cluster", KubeconfigPath: "/tmp/kc"}
	router := newPortForwardTestHandler(t, []*models.Cluster{cluster})

	cancelCalled := false
	sess := &portForwardSession{
		cancel:       func() { cancelCalled = true },
		lastActivity: time.Now(),
	}
	pfStore(cluster.ID, "sess-under-test", sess)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/clusters/"+cluster.ID+"/port-forward/sess-under-test", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !cancelCalled {
		t.Error("expected session.stop() to invoke the stored cancel func")
	}
	sess.mu.Lock()
	stopped := sess.stopped
	sess.mu.Unlock()
	if !stopped {
		t.Error("expected the session to be marked stopped")
	}
	if _, found := pfLookup(cluster.ID, "sess-under-test"); found {
		t.Error("expected the session to be removed from the store after delete")
	}
}

func TestPostPortForward_RBAC_Unauthenticated(t *testing.T) {
	h := buildHandler(t, &mockAddonService{})
	body, _ := json.Marshal(portForwardStartReq{Name: "pod-1", Namespace: "default", LocalPort: 18080, RemotePort: 80})
	r := httptest.NewRequest(http.MethodPost, "/clusters/cluster-1/port-forward", bytes.NewReader(body))
	w := serve(h, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without auth, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPostPortForward_RBAC_ViewerForbidden(t *testing.T) {
	h := buildHandler(t, &mockAddonService{})
	body, _ := json.Marshal(portForwardStartReq{Name: "pod-1", Namespace: "default", LocalPort: 18080, RemotePort: 80})
	r := httptest.NewRequest(http.MethodPost, "/clusters/cluster-1/port-forward", bytes.NewReader(body))
	r = claimsCtx(r, auth.RoleViewer)
	w := serve(h, r)
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for viewer (port-forward requires operator), got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeletePortForward_RBAC_ViewerForbidden(t *testing.T) {
	h := buildHandler(t, &mockAddonService{})
	r := httptest.NewRequest(http.MethodDelete, "/clusters/cluster-1/port-forward/sess-1", nil)
	r = claimsCtx(r, auth.RoleViewer)
	w := serve(h, r)
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for viewer (port-forward delete requires operator), got %d: %s", w.Code, w.Body.String())
	}
}
