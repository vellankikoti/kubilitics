package rest

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/kubilitics/kubilitics-backend/internal/api/middleware"
	"github.com/kubilitics/kubilitics-backend/internal/auth"
	"github.com/kubilitics/kubilitics-backend/internal/repository"
	"github.com/kubilitics/kubilitics-backend/internal/service"
)

// ScannerHandler handles /api/v1/scanner/* endpoints.
type ScannerHandler struct {
	svc      service.ScannerService
	authMode string
	repo     *repository.SQLiteRepository
}

// NewScannerHandler creates a new scanner handler.
func NewScannerHandler(svc service.ScannerService, authMode string, repo *repository.SQLiteRepository) *ScannerHandler {
	return &ScannerHandler{svc: svc, authMode: authMode, repo: repo}
}

// wrapWithRBAC mirrors internal/api/rest.Handler.wrapWithRBAC exactly — see
// the identical helper in internal/events/api.go for why this is duplicated
// rather than imported (package layering).
func (h *ScannerHandler) wrapWithRBAC(handler http.HandlerFunc, minRole string) http.Handler {
	if h.authMode == "" || h.authMode == "disabled" || h.repo == nil {
		return http.HandlerFunc(handler)
	}
	switch minRole {
	case auth.RoleAdmin:
		return middleware.RequireAdmin(h.repo)(http.HandlerFunc(handler))
	case auth.RoleOperator:
		return middleware.RequireOperator(h.repo)(http.HandlerFunc(handler))
	case auth.RoleViewer:
		return middleware.RequireViewer(h.repo)(http.HandlerFunc(handler))
	default:
		return http.HandlerFunc(handler)
	}
}

// RegisterRoutes registers scanner routes on the API router.
//
// SECURITY (docs/ai/STABILIZATION-PLAN.md Phase 0.4): StartScan's own inline
// admin check (`claims != nil && claims.Role != "admin" ...`) skipped the
// check entirely when claims was nil — i.e. in optional-auth mode, a fully
// unauthenticated caller could trigger an arbitrary filesystem/image/helm
// scan. Every route here is now wrapped with the same RequireRole pattern
// used throughout handler.go, which denies-by-default on nil claims (see
// middleware.RequireRole) instead of only acting when claims happen to be
// present. The other 7 routes had no RBAC at all; scan results/findings can
// contain vulnerability paths and secrets-adjacent data, so they're now
// gated at Viewer — consistent with every other read endpoint in the app.
func (h *ScannerHandler) RegisterRoutes(router *mux.Router) {
	router.Handle("/scanner/runs", h.wrapWithRBAC(h.StartScan, auth.RoleAdmin)).Methods("POST")
	router.Handle("/scanner/runs", h.wrapWithRBAC(h.ListRuns, auth.RoleViewer)).Methods("GET")
	router.Handle("/scanner/runs/{runId}", h.wrapWithRBAC(h.GetRun, auth.RoleViewer)).Methods("GET")
	router.Handle("/scanner/runs/{runId}/findings", h.wrapWithRBAC(h.ListRunFindings, auth.RoleViewer)).Methods("GET")
	router.Handle("/scanner/runs/{runId}/report", h.wrapWithRBAC(h.GetReport, auth.RoleViewer)).Methods("GET")
	router.Handle("/scanner/findings", h.wrapWithRBAC(h.ListAllFindings, auth.RoleViewer)).Methods("GET")
	router.Handle("/scanner/stats", h.wrapWithRBAC(h.GetStats, auth.RoleViewer)).Methods("GET")
	router.Handle("/scanner/tools", h.wrapWithRBAC(h.ListTools, auth.RoleViewer)).Methods("GET")
}

// allowedTargetTypes restricts scan target types.
var allowedTargetTypes = map[string]bool{
	"directory":       true,
	"container_image": true,
	"helm_chart":      true,
}

// allowedReportFormats restricts report download formats.
var allowedReportFormats = map[string]string{
	"json":     "json",
	"html":     "html",
	"markdown": "markdown",
	"md":       "markdown",
}

type startScanRequest struct {
	TargetType string   `json:"target_type"`
	TargetPath string   `json:"target_path"`
	Scanners   []string `json:"scanners,omitempty"`
}

// StartScan handles POST /scanner/runs — starts a new scan asynchronously.
// Admin role is enforced by the RequireAdmin wrapper in RegisterRoutes, not
// here — an inline claims-nil check previously lived in this function and
// silently did nothing when claims was nil (docs/ai/STABILIZATION-PLAN.md
// Phase 0.4), which the shared RequireRole middleware does not get wrong.
func (h *ScannerHandler) StartScan(w http.ResponseWriter, r *http.Request) {
	var req startScanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.TargetType == "" {
		req.TargetType = "directory"
	}
	if !allowedTargetTypes[req.TargetType] {
		respondError(w, http.StatusBadRequest, "Invalid target_type; allowed: directory, container_image, helm_chart")
		return
	}

	if req.TargetPath == "" {
		req.TargetPath = "."
	}

	// Path validation: resolve to absolute and ensure it doesn't escape via traversal
	if req.TargetType == "directory" || req.TargetType == "helm_chart" {
		cleaned := filepath.Clean(req.TargetPath)
		// Block obvious path traversal attempts
		if strings.Contains(cleaned, "..") {
			respondError(w, http.StatusBadRequest, "Path traversal not allowed")
			return
		}
		// Block scanning system-sensitive directories
		abs, err := filepath.Abs(cleaned)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid path")
			return
		}
		// docs/ai/STABILIZATION-PLAN.md Phase 0.4 flagged this blocklist as
		// not covering macOS user home dirs (/Users/*). Deliberately NOT
		// added here: the only real caller (ScanDashboard.tsx via
		// scannerApi.ts) always passes target_path="." for a self-scan of
		// the backend's own working directory, which during local
		// development is itself under /Users/<name>/... on macOS — a
		// blanket /Users block would silently break that, the feature's
		// only current use. A real fix needs an explicit allowlist of
		// legitimate scan roots (the plan's own suggested direction), which
		// needs product input on what those roots are; tracking as a
		// follow-up rather than guessing. Added /private and /boot (no
		// known legitimate caller needs them) and fixed a prefix-boundary
		// gap: HasPrefix(abs, "/home") also matched "/homebrew-data".
		for _, blocked := range []string{"/etc", "/var", "/root", "/home", "/proc", "/sys", "/dev", "/private", "/boot"} {
			if abs == blocked || strings.HasPrefix(abs, blocked+"/") {
				respondError(w, http.StatusForbidden, "Scanning system directories is not allowed")
				return
			}
		}
		req.TargetPath = cleaned
	}

	// Validate scanner names if provided
	validScanners := map[string]bool{"trivy": true, "semgrep": true, "gitleaks": true, "kubescape": true}
	for _, s := range req.Scanners {
		if !validScanners[s] {
			respondError(w, http.StatusBadRequest, "Unknown scanner: "+s)
			return
		}
	}

	run, err := h.svc.StartScan(r.Context(), req.TargetType, req.TargetPath, req.Scanners)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to start scan: "+err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(run)
}

// ListRuns handles GET /scanner/runs — lists scan runs with pagination.
func (h *ScannerHandler) ListRuns(w http.ResponseWriter, r *http.Request) {
	limit, offset := parseScannerPagination(r)
	runs, total, err := h.svc.ListScanRuns(r.Context(), limit, offset)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list scan runs: "+err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"runs":  runs,
		"total": total,
	})
}

// GetRun handles GET /scanner/runs/{runId} — gets a single scan run.
func (h *ScannerHandler) GetRun(w http.ResponseWriter, r *http.Request) {
	runID := mux.Vars(r)["runId"]
	run, err := h.svc.GetScanRun(r.Context(), runID)
	if err != nil {
		respondError(w, http.StatusNotFound, "Scan run not found")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(run)
}

// ListRunFindings handles GET /scanner/runs/{runId}/findings — findings for a specific run.
func (h *ScannerHandler) ListRunFindings(w http.ResponseWriter, r *http.Request) {
	runID := mux.Vars(r)["runId"]
	severity := r.URL.Query().Get("severity")
	tool := r.URL.Query().Get("tool")
	status := r.URL.Query().Get("status")
	limit, offset := parseScannerPagination(r)

	findings, total, err := h.svc.ListFindings(r.Context(), runID, severity, tool, status, limit, offset)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list findings: "+err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"findings": findings,
		"total":    total,
	})
}

// ListAllFindings handles GET /scanner/findings — all findings across runs.
func (h *ScannerHandler) ListAllFindings(w http.ResponseWriter, r *http.Request) {
	severity := r.URL.Query().Get("severity")
	tool := r.URL.Query().Get("tool")
	status := r.URL.Query().Get("status")
	limit, offset := parseScannerPagination(r)

	findings, total, err := h.svc.ListAllFindings(r.Context(), severity, tool, status, limit, offset)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list findings: "+err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"findings": findings,
		"total":    total,
	})
}

// GetStats handles GET /scanner/stats — aggregated scan statistics.
func (h *ScannerHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.svc.GetStats(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get stats: "+err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// GetReport handles GET /scanner/runs/{runId}/report — download scan report.
func (h *ScannerHandler) GetReport(w http.ResponseWriter, r *http.Request) {
	runID := mux.Vars(r)["runId"]
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}

	// Whitelist format to prevent header injection
	safeFormat, ok := allowedReportFormats[format]
	if !ok {
		respondError(w, http.StatusBadRequest, "Invalid format; allowed: json, html, markdown")
		return
	}

	data, contentType, err := h.svc.GetReport(r.Context(), runID, safeFormat)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to generate report: "+err.Error())
		return
	}

	ext := safeFormat
	if ext == "markdown" {
		ext = "md"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", "attachment; filename=\"scan-report-"+runID+"."+ext+"\"")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	// nosemgrep: go.lang.security.audit.xss.no-direct-write-to-responsewriter.no-direct-write-to-responsewriter
	_, _ = w.Write(data) //nolint:errcheck // Content-Type: application/json|text/plain — not HTML; server-generated content only
}

// ListTools handles GET /scanner/tools — lists available scanner tools.
func (h *ScannerHandler) ListTools(w http.ResponseWriter, r *http.Request) {
	tools := h.svc.AvailableTools()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tools": tools,
	})
}

// parseScannerPagination extracts limit/offset from query params (scanner-specific to avoid naming conflicts).
func parseScannerPagination(r *http.Request) (limit, offset int) {
	limit = 50
	offset = 0
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 && l <= 500 {
		limit = l
	}
	if o, err := strconv.Atoi(r.URL.Query().Get("offset")); err == nil && o >= 0 {
		offset = o
	}
	return
}
