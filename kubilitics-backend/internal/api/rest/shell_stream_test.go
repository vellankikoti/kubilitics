package rest

// docs/ai/STABILIZATION-PLAN.md Phase 2 item 6: shell_stream.go (the
// interactive PTY "cloud shell" WebSocket) had zero test coverage. These
// tests cover the validation/lookup branches that run before the
// WebSocket upgrade, RBAC gating, and — via a real WebSocket dial — a full
// round trip through the actual PTY plumbing (pty.StartWithSize, the
// stdin/stdout JSON framing, the writer/reader goroutines). The PTY round
// trip spawns a real local bash process; it does NOT contact any
// Kubernetes API server (KUBECONFIG is only set as an env var, never used
// unless the shell session itself runs a kubectl command), so it is a
// legitimate, non-mocked exercise of this file's own code, not a fake
// stand-in for a live-cluster-dependent boundary.

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"

	"github.com/kubilitics/kubilitics-backend/internal/auth"
	"github.com/kubilitics/kubilitics-backend/internal/config"
	"github.com/kubilitics/kubilitics-backend/internal/models"
)

func newShellStreamTestHandler(t *testing.T, clusters []*models.Cluster) *mux.Router {
	t.Helper()
	mockService := &mockClusterServiceWithClient{clusters: clusters}
	h := NewHandler(mockService, nil, &config.Config{}, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	router := mux.NewRouter()
	api := router.PathPrefix("/api/v1").Subrouter()
	SetupRoutes(api, h)
	return router
}

func TestGetShellStream_InvalidClusterID(t *testing.T) {
	router := newShellStreamTestHandler(t, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/clusters/bad!cluster/shell/stream", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGetShellStream_ClusterNotFound(t *testing.T) {
	router := newShellStreamTestHandler(t, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/clusters/ghost-cluster/shell/stream", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGetShellStream_NoKubeconfigPath(t *testing.T) {
	router := newShellStreamTestHandler(t, []*models.Cluster{{ID: "c1", KubeconfigPath: ""}})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/clusters/c1/shell/stream", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing kubeconfig path, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGetShellStream_RBAC_Unauthenticated(t *testing.T) {
	h := buildHandler(t, &mockAddonService{})
	r := httptest.NewRequest(http.MethodGet, "/clusters/cluster-1/shell/stream", nil)
	w := serve(h, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without auth, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetShellStream_RBAC_ViewerForbidden(t *testing.T) {
	h := buildHandler(t, &mockAddonService{})
	r := httptest.NewRequest(http.MethodGet, "/clusters/cluster-1/shell/stream", nil)
	r = claimsCtx(r, auth.RoleViewer)
	w := serve(h, r)
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for viewer (shell stream requires operator), got %d: %s", w.Code, w.Body.String())
	}
}

// TestGetShellStream_PTYEchoRoundTrip drives the real PTY shell end-to-end:
// dial, send a stdin frame, and verify the echoed marker comes back on
// stdout. This exercises pty.StartWithSize, the rcfile-based PS1/alias
// setup, the single-writer-goroutine WebSocket pump, and the stdin/resize
// JSON protocol — all without touching any Kubernetes API.
func TestGetShellStream_PTYEchoRoundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("spawns a real PTY shell process; skipped in -short mode")
	}

	router := newShellStreamTestHandler(t, []*models.Cluster{{
		ID:             "stream-cluster",
		KubeconfigPath: "/tmp/kubilitics-shell-test-kubeconfig", // never read by a shell that doesn't run kubectl
	}})
	srv := httptest.NewServer(router)
	defer srv.Close()

	wsURL := strings.Replace(srv.URL, "http://", "ws://", 1) + "/api/v1/clusters/stream-cluster/shell/stream"
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v (resp=%v)", err, resp)
	}
	defer func() { _ = conn.Close() }()

	const marker = "kubilitics_shell_stream_test_marker_42"
	stdin := wsInMessage{T: wsMsgStdin, D: base64.StdEncoding.EncodeToString([]byte("echo " + marker + "\n"))}
	if err := conn.WriteJSON(stdin); err != nil {
		t.Fatalf("write stdin: %v", err)
	}

	_ = conn.SetReadDeadline(time.Now().Add(15 * time.Second))
	found := false
	for i := 0; i < 500 && !found; i++ {
		var out wsOutMessage
		if err := conn.ReadJSON(&out); err != nil {
			t.Fatalf("read ws message (marker not yet found): %v", err)
		}
		switch out.T {
		case wsMsgStdout:
			dec, decErr := base64.StdEncoding.DecodeString(out.D)
			if decErr == nil && strings.Contains(string(dec), marker) {
				found = true
			}
		case wsMsgError:
			t.Fatalf("shell reported an error: %s", out.D)
		case wsMsgExit:
			t.Fatal("shell exited before echoing the marker")
		}
	}
	if !found {
		t.Fatal("never observed the echoed marker on stdout")
	}
}
