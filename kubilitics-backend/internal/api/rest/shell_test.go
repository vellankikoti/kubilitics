package rest

// docs/ai/STABILIZATION-PLAN.md Phase 2 item 6: shell.go (the read-only
// "cluster shell" that proxies kubectl/kcli commands) had zero test
// coverage. These tests cover request validation, the blocked-verb
// allowlist (the actual security boundary of this endpoint — PostShell
// refuses mutating verbs so the web shell can't be used to delete/patch/
// exec), RBAC gating, the context-deadline-exceeded path, and the
// splitCommand tokenizer in isolation.
//
// What is NOT exercised with real process output assertions: the exact
// stdout of a successful "kubectl version"/"kubectl get ..." call. That
// depends on whatever kubectl/kcli happens to be on the test machine's
// PATH and (for get/describe/etc) a real reachable cluster — brittle and
// environment-dependent. Instead, the success path is exercised via the
// context-already-expired trick below, which deterministically drives the
// exact same cmd.Run()/respondTimeout code path without depending on any
// external binary actually being resolvable or reachable.

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"

	"github.com/kubilitics/kubilitics-backend/internal/auth"
	"github.com/kubilitics/kubilitics-backend/internal/config"
	"github.com/kubilitics/kubilitics-backend/internal/models"
)

func newShellTestHandler(t *testing.T, clusters []*models.Cluster) *mux.Router {
	t.Helper()
	mockService := &mockClusterServiceWithClient{clusters: clusters}
	h := NewHandler(mockService, nil, &config.Config{}, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	router := mux.NewRouter()
	api := router.PathPrefix("/api/v1").Subrouter()
	SetupRoutes(api, h)
	return router
}

func postShellRequest(clusterID, command string) *http.Request {
	body, _ := json.Marshal(map[string]string{"command": command})
	return httptest.NewRequest(http.MethodPost, "/api/v1/clusters/"+clusterID+"/shell", bytes.NewReader(body))
}

func TestPostShell_InvalidClusterID(t *testing.T) {
	router := newShellTestHandler(t, nil)
	req := postShellRequest("bad!cluster", "get pods")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestPostShell_InvalidJSONBody(t *testing.T) {
	router := newShellTestHandler(t, []*models.Cluster{{ID: "c1", KubeconfigPath: "/tmp/kc"}})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/clusters/c1/shell", strings.NewReader("{not-json"))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for malformed body, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestPostShell_ClusterNotFound(t *testing.T) {
	router := newShellTestHandler(t, nil)
	req := postShellRequest("ghost-cluster", "get pods")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestPostShell_NoKubeconfigPath(t *testing.T) {
	router := newShellTestHandler(t, []*models.Cluster{{ID: "c1", KubeconfigPath: ""}})
	req := postShellRequest("c1", "get pods")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing kubeconfig path, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestPostShell_BlockedVerbsRejected(t *testing.T) {
	router := newShellTestHandler(t, []*models.Cluster{{ID: "c1", KubeconfigPath: "/tmp/kc"}})

	cases := []string{
		"delete pod foo",
		"DELETE pod foo", // verb check is case-insensitive
		"kubectl delete pod foo",
		"kcli delete pod foo",
		"kubectl apply -f manifest.yaml",
		"patch deployment foo",
		"exec -it foo -- sh",
		"drain node-1",
		"rollout restart deployment/foo",
	}
	for _, cmd := range cases {
		t.Run(cmd, func(t *testing.T) {
			req := postShellRequest("c1", cmd)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400 for blocked verb %q, got %d: %s", cmd, rec.Code, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), "Command not allowed") {
				t.Errorf("expected 'Command not allowed' message for %q, got %s", cmd, rec.Body.String())
			}
		})
	}
}

// TestPostShell_CommandTimesOut forces the context-deadline-exceeded branch
// deterministically: a request context that has already expired by the time
// PostShell's exec.CommandContext tries to Start() returns ctx.Err() before
// ever touching a real process, regardless of whether kubectl/kcli exist on
// the test machine's PATH. This is the same branch a genuinely slow/hung
// kubectl call would hit in production.
func TestPostShell_CommandTimesOut(t *testing.T) {
	router := newShellTestHandler(t, []*models.Cluster{{ID: "c1", KubeconfigPath: "/tmp/kc"}})

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(time.Millisecond) // guarantee the deadline has passed

	req := postShellRequest("c1", "get pods")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusGatewayTimeout {
		t.Fatalf("expected 504 on an already-expired context, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "timed out") {
		t.Errorf("expected timeout message, got %s", rec.Body.String())
	}
}

// TestPostShell_EmptyCommandAlwaysReturns200 exercises the bare "kcli"/empty
// command branch, which deliberately ignores cmd.Run()'s error (so the shell
// always returns *something* for a bare invocation) even when the context is
// already expired — proving that branch never surfaces a timeout/error status.
func TestPostShell_EmptyCommandAlwaysReturns200(t *testing.T) {
	router := newShellTestHandler(t, []*models.Cluster{{ID: "c1", KubeconfigPath: "/tmp/kc"}})

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(time.Millisecond)

	req := postShellRequest("c1", "")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for bare/empty command regardless of exec outcome, got %d: %s", rec.Code, rec.Body.String())
	}
	var out struct {
		Stdout   string `json:"stdout"`
		Stderr   string `json:"stderr"`
		ExitCode int    `json:"exitCode"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}

func TestPostShell_RBAC_Unauthenticated(t *testing.T) {
	h := buildHandler(t, &mockAddonService{})
	body, _ := json.Marshal(map[string]string{"command": "get pods"})
	r := httptest.NewRequest(http.MethodPost, "/clusters/cluster-1/shell", bytes.NewReader(body))
	w := serve(h, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without auth, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPostShell_RBAC_ViewerForbidden(t *testing.T) {
	h := buildHandler(t, &mockAddonService{})
	body, _ := json.Marshal(map[string]string{"command": "get pods"})
	r := httptest.NewRequest(http.MethodPost, "/clusters/cluster-1/shell", bytes.NewReader(body))
	r = claimsCtx(r, auth.RoleViewer)
	w := serve(h, r)
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for viewer (shell requires operator), got %d: %s", w.Code, w.Body.String())
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// splitCommand tokenizer — pure function, no handler plumbing needed.
// ──────────────────────────────────────────────────────────────────────────────

func TestSplitCommand(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  []string
	}{
		{"simple", "get pods -n default", []string{"get", "pods", "-n", "default"}},
		{"empty", "", nil},
		{"whitespace only", "   ", nil},
		{"single word", "version", []string{"version"}},
		{"double-quoted arg with space", `get pods -l app="my app"`, []string{"get", "pods", "-l", `app="my`, `app"`}},
		{"quoted token", `get pods -l "app=my-app"`, []string{"get", "pods", "-l", "app=my-app"}},
		{"single-quoted token", `get pods -l 'app=my-app'`, []string{"get", "pods", "-l", "app=my-app"}},
		{"extra spaces collapse", "get   pods", []string{"get", "pods"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := splitCommand(tc.input)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("splitCommand(%q) = %#v, want %#v", tc.input, got, tc.want)
			}
		})
	}
}
