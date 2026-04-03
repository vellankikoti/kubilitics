package rest

import (
	"net/http"
	"sort"

	"github.com/gorilla/mux"

	"github.com/kubilitics/kubilitics-backend/internal/intelligence/compliance"
	"github.com/kubilitics/kubilitics-backend/internal/pkg/validate"
)

// complianceEngine is a package-level singleton so that every request reuses
// the same framework registry. It is safe for concurrent use because Engine
// is read-only after construction.
var complianceEngine = compliance.NewEngine()

// GetCompliance handles GET /clusters/{clusterId}/compliance?framework=cis-1.8.
// It builds ClusterComplianceData from the cluster's graph snapshot and evaluates
// the requested compliance framework.
func (h *Handler) GetCompliance(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["clusterId"]
	if !validate.ClusterID(clusterID) {
		respondError(w, http.StatusBadRequest, "Invalid clusterId")
		return
	}

	framework := r.URL.Query().Get("framework")
	if framework == "" {
		respondError(w, http.StatusBadRequest, "Query parameter 'framework' is required (e.g. cis-1.8, soc2)")
		return
	}

	namespaceFilter := r.URL.Query().Get("namespace")

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

	data := buildComplianceData(snap, namespaceFilter)

	result, err := complianceEngine.Evaluate(framework, clusterID, data)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// GetComplianceFrameworks handles GET /clusters/{clusterId}/compliance/frameworks.
// It returns the list of available compliance frameworks.
func (h *Handler) GetComplianceFrameworks(w http.ResponseWriter, _ *http.Request) {
	names := complianceEngine.ListFrameworks()
	sort.Strings(names)
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"frameworks": names,
	})
}
