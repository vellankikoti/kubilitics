package rest

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/kubilitics/kubilitics-backend/internal/autopilot"
	"github.com/kubilitics/kubilitics-backend/internal/pkg/logger"
	"github.com/kubilitics/kubilitics-backend/internal/pkg/validate"
)

// autopilotScheduler is a package-level reference set during server initialization.
// The handler methods access it to trigger scans and retrieve findings.
var autopilotScheduler *autopilot.Scheduler

// autopilotRepo is a package-level reference to the autopilot repository.
var autopilotRepo autopilot.AutoPilotRepository

// autopilotRegistry is a package-level reference to the rule registry.
var autopilotRegistry *autopilot.RuleRegistry

// InitAutoPilot sets the package-level autopilot dependencies.
// Must be called during server startup before routes are served.
func InitAutoPilot(scheduler *autopilot.Scheduler, repo autopilot.AutoPilotRepository, registry *autopilot.RuleRegistry) {
	autopilotScheduler = scheduler
	autopilotRepo = repo
	autopilotRegistry = registry
}

// GetAutoPilotFindings handles GET /clusters/{clusterId}/autopilot/findings.
// Runs detection NOW against the current graph snapshot and returns findings.
func (h *Handler) GetAutoPilotFindings(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["clusterId"]
	if !validate.ClusterID(clusterID) {
		respondError(w, http.StatusBadRequest, "Invalid clusterId")
		return
	}

	engine := h.getOrStartGraphEngine(r, clusterID)
	if engine == nil {
		respondError(w, http.StatusServiceUnavailable, "Graph engine not available for this cluster")
		return
	}

	snap := engine.Snapshot()
	if !snap.Status().Ready {
		respondError(w, http.StatusServiceUnavailable, "Dependency graph is still building")
		return
	}

	if autopilotRegistry == nil {
		respondError(w, http.StatusServiceUnavailable, "AutoPilot not initialized")
		return
	}

	findings := autopilotRegistry.DetectAll(snap)
	if findings == nil {
		findings = []autopilot.Finding{}
	}
	respondJSON(w, http.StatusOK, findings)
}

// GetAutoPilotActions handles GET /clusters/{clusterId}/autopilot/actions.
// Returns paginated list of actions with optional status filter.
func (h *Handler) GetAutoPilotActions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["clusterId"]
	if !validate.ClusterID(clusterID) {
		respondError(w, http.StatusBadRequest, "Invalid clusterId")
		return
	}

	if autopilotRepo == nil {
		respondError(w, http.StatusServiceUnavailable, "AutoPilot not initialized")
		return
	}

	status := r.URL.Query().Get("status")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	actions, err := autopilotRepo.ListActions(clusterID, status, limit, offset)
	if err != nil {
		requestID := logger.FromContext(r.Context())
		respondErrorWithCode(w, http.StatusInternalServerError, ErrCodeInternalError, err.Error(), requestID)
		return
	}

	if actions == nil {
		actions = []autopilot.ActionRecord{}
	}
	respondJSON(w, http.StatusOK, actions)
}

// GetAutoPilotAction handles GET /clusters/{clusterId}/autopilot/actions/{actionId}.
func (h *Handler) GetAutoPilotAction(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["clusterId"]
	if !validate.ClusterID(clusterID) {
		respondError(w, http.StatusBadRequest, "Invalid clusterId")
		return
	}

	actionID := vars["actionId"]
	if actionID == "" {
		respondError(w, http.StatusBadRequest, "actionId is required")
		return
	}

	if autopilotRepo == nil {
		respondError(w, http.StatusServiceUnavailable, "AutoPilot not initialized")
		return
	}

	action, err := autopilotRepo.GetAction(actionID)
	if err != nil {
		requestID := logger.FromContext(r.Context())
		respondErrorWithCode(w, http.StatusNotFound, ErrCodeNotFound, err.Error(), requestID)
		return
	}

	respondJSON(w, http.StatusOK, action)
}

// PostAutoPilotApprove handles POST /clusters/{clusterId}/autopilot/actions/{actionId}/approve.
func (h *Handler) PostAutoPilotApprove(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["clusterId"]
	if !validate.ClusterID(clusterID) {
		respondError(w, http.StatusBadRequest, "Invalid clusterId")
		return
	}

	actionID := vars["actionId"]
	if actionID == "" {
		respondError(w, http.StatusBadRequest, "actionId is required")
		return
	}

	if autopilotRepo == nil {
		respondError(w, http.StatusServiceUnavailable, "AutoPilot not initialized")
		return
	}

	// Verify the action exists and is pending
	action, err := autopilotRepo.GetAction(actionID)
	if err != nil {
		requestID := logger.FromContext(r.Context())
		respondErrorWithCode(w, http.StatusNotFound, ErrCodeNotFound, err.Error(), requestID)
		return
	}

	if action.Status != "pending" {
		respondError(w, http.StatusBadRequest, "Only pending actions can be approved")
		return
	}

	if err := autopilotRepo.UpdateActionStatus(actionID, "applied"); err != nil {
		requestID := logger.FromContext(r.Context())
		respondErrorWithCode(w, http.StatusInternalServerError, ErrCodeInternalError, err.Error(), requestID)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "applied", "action_id": actionID})
}

// PostAutoPilotDismiss handles POST /clusters/{clusterId}/autopilot/actions/{actionId}/dismiss.
func (h *Handler) PostAutoPilotDismiss(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["clusterId"]
	if !validate.ClusterID(clusterID) {
		respondError(w, http.StatusBadRequest, "Invalid clusterId")
		return
	}

	actionID := vars["actionId"]
	if actionID == "" {
		respondError(w, http.StatusBadRequest, "actionId is required")
		return
	}

	if autopilotRepo == nil {
		respondError(w, http.StatusServiceUnavailable, "AutoPilot not initialized")
		return
	}

	action, err := autopilotRepo.GetAction(actionID)
	if err != nil {
		requestID := logger.FromContext(r.Context())
		respondErrorWithCode(w, http.StatusNotFound, ErrCodeNotFound, err.Error(), requestID)
		return
	}

	if action.Status != "pending" {
		respondError(w, http.StatusBadRequest, "Only pending actions can be dismissed")
		return
	}

	if err := autopilotRepo.UpdateActionStatus(actionID, "dismissed"); err != nil {
		requestID := logger.FromContext(r.Context())
		respondErrorWithCode(w, http.StatusInternalServerError, ErrCodeInternalError, err.Error(), requestID)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "dismissed", "action_id": actionID})
}

// GetAutoPilotConfig handles GET /clusters/{clusterId}/autopilot/config.
func (h *Handler) GetAutoPilotConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["clusterId"]
	if !validate.ClusterID(clusterID) {
		respondError(w, http.StatusBadRequest, "Invalid clusterId")
		return
	}

	if autopilotRepo == nil {
		respondError(w, http.StatusServiceUnavailable, "AutoPilot not initialized")
		return
	}

	configs, err := autopilotRepo.ListRuleConfigs(clusterID)
	if err != nil {
		requestID := logger.FromContext(r.Context())
		respondErrorWithCode(w, http.StatusInternalServerError, ErrCodeInternalError, err.Error(), requestID)
		return
	}

	if configs == nil {
		configs = []autopilot.RuleConfig{}
	}
	respondJSON(w, http.StatusOK, configs)
}

// PutAutoPilotRuleConfig handles PUT /clusters/{clusterId}/autopilot/config/{ruleId}.
func (h *Handler) PutAutoPilotRuleConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["clusterId"]
	if !validate.ClusterID(clusterID) {
		respondError(w, http.StatusBadRequest, "Invalid clusterId")
		return
	}

	ruleID := vars["ruleId"]
	if ruleID == "" {
		respondError(w, http.StatusBadRequest, "ruleId is required")
		return
	}

	if autopilotRepo == nil {
		respondError(w, http.StatusServiceUnavailable, "AutoPilot not initialized")
		return
	}

	var cfg autopilot.RuleConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	cfg.RuleID = ruleID

	// Validate mode
	switch cfg.Mode {
	case "auto", "approval", "audit":
		// valid
	default:
		respondError(w, http.StatusBadRequest, "Mode must be auto, approval, or audit")
		return
	}

	if err := autopilotRepo.UpsertRuleConfig(clusterID, cfg); err != nil {
		requestID := logger.FromContext(r.Context())
		respondErrorWithCode(w, http.StatusInternalServerError, ErrCodeInternalError, err.Error(), requestID)
		return
	}

	respondJSON(w, http.StatusOK, cfg)
}

// PostAutoPilotScan handles POST /clusters/{clusterId}/autopilot/scan.
// Triggers a manual scan and returns findings.
func (h *Handler) PostAutoPilotScan(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["clusterId"]
	if !validate.ClusterID(clusterID) {
		respondError(w, http.StatusBadRequest, "Invalid clusterId")
		return
	}

	if autopilotScheduler == nil {
		respondError(w, http.StatusServiceUnavailable, "AutoPilot not initialized")
		return
	}

	findings, err := autopilotScheduler.RunOnce(clusterID)
	if err != nil {
		requestID := logger.FromContext(r.Context())
		respondErrorWithCode(w, http.StatusInternalServerError, ErrCodeInternalError, err.Error(), requestID)
		return
	}

	if findings == nil {
		findings = []autopilot.Finding{}
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"findings": findings,
		"count":    len(findings),
	})
}

// GetAutoPilotRules handles GET /clusters/{clusterId}/autopilot/rules.
// Returns available rules with metadata.
func (h *Handler) GetAutoPilotRules(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["clusterId"]
	if !validate.ClusterID(clusterID) {
		respondError(w, http.StatusBadRequest, "Invalid clusterId")
		return
	}

	if autopilotRegistry == nil {
		respondError(w, http.StatusServiceUnavailable, "AutoPilot not initialized")
		return
	}

	var rules []autopilot.RuleMeta
	for _, rule := range autopilotRegistry.Rules() {
		rules = append(rules, autopilot.RuleMeta{
			ID:          rule.ID(),
			Name:        rule.Name(),
			Description: rule.Description(),
			Severity:    rule.Severity(),
		})
	}

	if rules == nil {
		rules = []autopilot.RuleMeta{}
	}
	respondJSON(w, http.StatusOK, rules)
}
