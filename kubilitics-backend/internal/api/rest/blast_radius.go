package rest

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"github.com/kubilitics/kubilitics-backend/internal/pkg/logger"
	"github.com/kubilitics/kubilitics-backend/internal/pkg/validate"
	"github.com/kubilitics/kubilitics-backend/internal/service"
)

// GetBlastRadius handles GET /clusters/{clusterId}/blast-radius/{namespace}/{kind}/{name}.
// It computes the dependency graph and criticality score for the specified resource.
func (h *Handler) GetBlastRadius(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["clusterId"]
	if !validate.ClusterID(clusterID) {
		respondError(w, http.StatusBadRequest, "Invalid clusterId")
		return
	}

	namespace := vars["namespace"]
	kind := vars["kind"]
	name := vars["name"]

	if namespace == "" || kind == "" || name == "" {
		respondError(w, http.StatusBadRequest, "namespace, kind, and name are required")
		return
	}

	// Normalize kind to title case (e.g. "deployments" -> "Deployment")
	kind = normalizeKind(kind)
	if kind == "" {
		respondError(w, http.StatusBadRequest, "Unsupported resource kind")
		return
	}

	client, err := h.getClientFromRequest(r.Context(), r, clusterID, h.cfg)
	if err != nil {
		requestID := logger.FromContext(r.Context())
		respondErrorWithCode(w, http.StatusNotFound, ErrCodeNotFound, err.Error(), requestID)
		return
	}

	timeoutSec := 30
	if h.cfg != nil && h.cfg.TopologyTimeoutSec > 0 {
		timeoutSec = h.cfg.TopologyTimeoutSec
	}
	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(timeoutSec)*time.Second)
	defer cancel()

	brService := service.NewBlastRadiusService()
	result, err := brService.ComputeBlastRadius(ctx, client, namespace, kind, name)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			respondError(w, http.StatusServiceUnavailable, "Blast radius computation timed out")
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// normalizeKind converts plural/lowercase resource kind strings to their
// canonical Kubernetes kind (singular, PascalCase).
func normalizeKind(kind string) string {
	switch kind {
	case "Deployment", "deployment", "deployments":
		return "Deployment"
	case "StatefulSet", "statefulset", "statefulsets":
		return "StatefulSet"
	case "DaemonSet", "daemonset", "daemonsets":
		return "DaemonSet"
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
	default:
		return ""
	}
}
