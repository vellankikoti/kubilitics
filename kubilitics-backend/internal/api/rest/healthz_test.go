package rest

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/kubilitics/kubilitics-backend/internal/repository"
)

// Locks in the behavior the Helm chart's readinessProbe now depends on
// (Task 9, quality-attributes audit): /healthz/ready must actually reflect
// backend health, not just "process is up" — unlike the legacy /health
// endpoint, which unconditionally returns 200 regardless of DB state.
func TestHealthzHandler_Live_AlwaysReturns200(t *testing.T) {
	h := NewHealthzHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/healthz/live", nil)
	rec := httptest.NewRecorder()

	h.Live(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestHealthzHandler_Ready_ReturnsOKWhenNoRepoConfigured(t *testing.T) {
	// A nil repo means "no DB dependency to check" — Ready must not panic
	// and must report healthy, matching Live's behavior in that case.
	h := NewHealthzHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/healthz/ready", nil)
	rec := httptest.NewRecorder()

	h.Ready(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 with nil repo, got %d", rec.Code)
	}
}

// This is the exact behavior the Helm chart's readinessProbe now depends
// on (fixed to point at /healthz/ready instead of the legacy /health,
// which always returns 200 regardless of DB state) — a readiness probe
// that can never fail is not a readiness probe.
func TestHealthzHandler_Ready_Returns503WhenDatabaseUnreachable(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	repo, err := repository.NewSQLiteRepository(dbPath)
	if err != nil {
		t.Fatalf("failed to create test repo: %v", err)
	}
	// Close the connection to force Ping to fail, simulating a real
	// database-unavailable condition.
	if err := repo.Close(); err != nil {
		t.Fatalf("failed to close test repo: %v", err)
	}

	h := NewHealthzHandler(repo)
	req := httptest.NewRequest(http.MethodGet, "/healthz/ready", nil)
	rec := httptest.NewRecorder()

	h.Ready(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when database is unreachable, got %d (body: %s)", rec.Code, rec.Body.String())
	}
}
