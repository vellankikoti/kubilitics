package rest

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/kubilitics/kubilitics-backend/internal/graph"
	"github.com/kubilitics/kubilitics-backend/internal/models"
	"github.com/kubilitics/kubilitics-backend/internal/pkg/logger"
	"github.com/kubilitics/kubilitics-backend/internal/pkg/validate"
)

// GetBlastRadius handles GET /clusters/{clusterId}/blast-radius/{namespace}/{kind}/{name}.
// It reads from the pre-built graph engine snapshot instead of computing per-request.
func (h *Handler) GetBlastRadius(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["clusterId"]
	if !validate.ClusterID(clusterID) {
		respondError(w, http.StatusBadRequest, "Invalid clusterId")
		return
	}

	namespace := vars["namespace"]
	kind := normalizeKind(vars["kind"])
	name := vars["name"]

	if namespace == "" || kind == "" || name == "" {
		respondError(w, http.StatusBadRequest, "namespace, kind, and name are required")
		return
	}

	engine := h.getOrStartGraphEngine(r, clusterID)
	if engine == nil {
		respondError(w, http.StatusServiceUnavailable, "Blast radius graph not available for this cluster")
		return
	}

	snap := engine.Snapshot()
	if !snap.Status().Ready {
		respondError(w, http.StatusServiceUnavailable, "Dependency graph is still building")
		return
	}

	target := models.ResourceRef{Kind: kind, Name: name, Namespace: namespace}

	// Optional failure_mode query parameter (auto-detected from kind if not provided)
	failureMode := r.URL.Query().Get("failure_mode")
	if failureMode == "" {
		switch kind {
		case "Pod":
			failureMode = graph.FailureModePodCrash
		case "Namespace":
			failureMode = graph.FailureModeNamespaceDeletion
		default:
			failureMode = graph.FailureModeWorkloadDeletion
		}
	}
	if !graph.ValidFailureMode(failureMode) {
		respondError(w, http.StatusBadRequest, "Invalid failure_mode. Must be one of: pod-crash, workload-deletion, namespace-deletion")
		return
	}

	result, err := snap.ComputeBlastRadiusWithMode(target, failureMode)
	if err != nil {
		requestID := logger.FromContext(r.Context())
		respondErrorWithCode(w, http.StatusNotFound, ErrCodeNotFound, err.Error(), requestID)
		return
	}

	// Strip audit trail if not requested
	if r.URL.Query().Get("audit") != "true" {
		result.AuditTrail = nil
	}

	respondJSON(w, http.StatusOK, result)
}

// GetBlastRadiusSummary handles GET /clusters/{clusterId}/blast-radius/summary.
// Returns the top-N highest blast-radius resources from the graph snapshot.
func (h *Handler) GetBlastRadiusSummary(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["clusterId"]
	if !validate.ClusterID(clusterID) {
		respondError(w, http.StatusBadRequest, "Invalid clusterId")
		return
	}

	engine := h.getOrStartGraphEngine(r, clusterID)
	if engine == nil {
		respondError(w, http.StatusServiceUnavailable, "Blast radius graph not available for this cluster")
		return
	}

	snap := engine.Snapshot()
	summary := snap.GetSummary(20)
	respondJSON(w, http.StatusOK, summary)
}

// GetGraphStatus handles GET /clusters/{clusterId}/blast-radius/graph-status.
// Returns the current status of the dependency graph engine.
func (h *Handler) GetGraphStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["clusterId"]
	if !validate.ClusterID(clusterID) {
		respondError(w, http.StatusBadRequest, "Invalid clusterId")
		return
	}

	engine := h.getOrStartGraphEngine(r, clusterID)
	if engine == nil {
		respondJSON(w, http.StatusOK, models.GraphStatus{Error: "graph engine not initialized"})
		return
	}

	respondJSON(w, http.StatusOK, engine.Status())
}

// getGraphEngine returns the graph engine for a cluster, or nil.
func (h *Handler) getGraphEngine(clusterID string) *graph.ClusterGraphEngine {
	h.graphEnginesMu.RLock()
	defer h.graphEnginesMu.RUnlock()
	if h.graphEngines == nil {
		return nil
	}
	return h.graphEngines[clusterID]
}

// getOrStartGraphEngine returns an existing engine or lazily starts one.
// Uses the request context to resolve the K8s client (same as other handlers).
// Uses a double-checked locking pattern to avoid the TOCTOU race on the map.
func (h *Handler) getOrStartGraphEngine(r *http.Request, clusterID string) *graph.ClusterGraphEngine {
	// Fast path: engine already exists.
	h.graphEnginesMu.RLock()
	if h.graphEngines != nil {
		if engine, ok := h.graphEngines[clusterID]; ok {
			h.graphEnginesMu.RUnlock()
			return engine
		}
	}
	h.graphEnginesMu.RUnlock()

	// Slow path: acquire write lock and re-check before inserting (TOCTOU guard).
	h.graphEnginesMu.Lock()
	defer h.graphEnginesMu.Unlock()

	if h.graphEngines == nil {
		h.graphEngines = make(map[string]*graph.ClusterGraphEngine)
	}
	// Re-check: another goroutine may have inserted while we were waiting.
	if engine, ok := h.graphEngines[clusterID]; ok {
		return engine
	}

	// Lazy init: resolve client the same way the handler does
	client, err := h.getClientFromRequest(r.Context(), r, clusterID, h.cfg)
	if err != nil {
		return nil
	}
	engine := graph.NewClusterGraphEngine(clusterID, client.Clientset, slog.Default())
	// Gap 2 fix: actively invalidate BOTH V1 and V2 topology caches when K8s
	// resources change. The onRebuild callback fires after every debounced graph
	// rebuild (triggered by informer add/update/delete events). This ensures the
	// next topology request gets fresh data instead of waiting for TTL expiry.
	engine.SetOnRebuild(func(cid string) {
		TopologyCacheInvalidateForCluster(cid)      // V2 handler cache (sync.Map)
		h.topologyService.InvalidateForCluster(cid) // V1 service cache (topologycache.Cache)
	})
	engine.Start(context.Background())
	h.graphEngines[clusterID] = engine
	slog.Default().Info("Lazily started graph engine", "cluster", clusterID)
	return engine
}

// normalizeKind converts plural/lowercase resource kind strings to their
// canonical Kubernetes kind (singular, PascalCase).
func normalizeKind(kind string) string {
	switch kind {
	case "Pod", "pod", "pods":
		return "Pod"
	case "Deployment", "deployment", "deployments":
		return "Deployment"
	case "ReplicaSet", "replicaset", "replicasets":
		return "ReplicaSet"
	case "StatefulSet", "statefulset", "statefulsets":
		return "StatefulSet"
	case "DaemonSet", "daemonset", "daemonsets":
		return "DaemonSet"
	case "Job", "job", "jobs":
		return "Job"
	case "CronJob", "cronjob", "cronjobs":
		return "CronJob"
	case "Service", "service", "services":
		return "Service"
	case "ConfigMap", "configmap", "configmaps":
		return "ConfigMap"
	case "Secret", "secret", "secrets":
		return "Secret"
	case "Ingress", "ingress", "ingresses":
		return "Ingress"
	case "NetworkPolicy", "networkpolicy", "networkpolicies":
		return "NetworkPolicy"
	case "PersistentVolumeClaim", "persistentvolumeclaim", "persistentvolumeclaims", "pvc":
		return "PersistentVolumeClaim"
	case "PersistentVolume", "persistentvolume", "persistentvolumes", "pv":
		return "PersistentVolume"
	case "Node", "node", "nodes":
		return "Node"
	case "Namespace", "namespace", "namespaces":
		return "Namespace"
	case "ServiceAccount", "serviceaccount", "serviceaccounts":
		return "ServiceAccount"
	case "HorizontalPodAutoscaler", "horizontalpodautoscaler", "horizontalpodautoscalers", "hpa":
		return "HorizontalPodAutoscaler"
	default:
		return ""
	}
}
