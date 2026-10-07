package rest

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"github.com/kubilitics/kubilitics-backend/internal/intelligence/diff"
	"github.com/kubilitics/kubilitics-backend/internal/pkg/validate"
	topologyv2 "github.com/kubilitics/kubilitics-backend/internal/topology/v2"
	topologyv2builder "github.com/kubilitics/kubilitics-backend/internal/topology/v2/builder"
)

// CreateTopologySnapshot handles POST /clusters/{clusterId}/topology/snapshot.
// It fetches the current topology, converts it to a snapshot, and stores it.
// Optional query parameter: namespace (default: cluster-wide, i.e. empty string).
func (h *Handler) CreateTopologySnapshot(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["clusterId"]
	if !validate.ClusterID(clusterID) {
		respondError(w, http.StatusBadRequest, "Invalid clusterId")
		return
	}

	namespace := r.URL.Query().Get("namespace")

	if h.snapshotStore == nil {
		respondError(w, http.StatusServiceUnavailable, "Snapshot store not configured")
		return
	}

	// Resolve cluster name for topology options
	clusterName := clusterID
	if c, err := h.clusterService.GetCluster(r.Context(), clusterID); err == nil && c != nil && c.Name != "" {
		clusterName = c.Name
	}

	// Build topology
	opts := topologyv2.Options{
		ClusterID:   clusterID,
		ClusterName: clusterName,
		Mode:        topologyv2.ViewModeCluster,
		Namespace:   namespace,
	}

	timeoutSec := 10
	if h.cfg != nil && h.cfg.TopologyTimeoutSec > 0 {
		timeoutSec = h.cfg.TopologyTimeoutSec
	}
	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(timeoutSec)*time.Second)
	defer cancel()

	client, err := h.getClientFromRequest(ctx, r, clusterID, h.cfg)
	if err != nil {
		respondError(w, http.StatusServiceUnavailable, "Cluster not connected")
		return
	}

	topoResp, err := topologyv2builder.BuildTopology(ctx, opts, client, h.clusterService.GetInformerManager(clusterID))
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			respondTimeout(w, r, http.StatusServiceUnavailable, "", "CreateTopologySnapshot", clusterID, "Topology build timed out")
			return
		}
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to build topology: %v", err))
		return
	}

	// Convert topology response to snapshot
	snapshot := diff.TopologyResponseToSnapshot(clusterID, namespace, topoResp)

	// Store snapshot
	if err := h.snapshotStore.Save(snapshot); err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to save snapshot: %v", err))
		return
	}

	// Return snapshot summary (without full node/edge data to keep response small)
	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"id":         snapshot.ID,
		"cluster_id": snapshot.ClusterID,
		"namespace":  snapshot.Namespace,
		"metadata":   snapshot.Metadata,
		"created_at": snapshot.CreatedAt,
	})
}

// GetTopologyDiff handles GET /clusters/{clusterId}/topology/diff.
// Query parameters:
//   - from: ISO 8601 date (required) — start of diff window
//   - to:   ISO 8601 date (required) — end of diff window
//   - namespace: optional namespace filter (default: cluster-wide)
//
// Finds the closest snapshots to the given dates and computes the diff.
func (h *Handler) GetTopologyDiff(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["clusterId"]
	if !validate.ClusterID(clusterID) {
		respondError(w, http.StatusBadRequest, "Invalid clusterId")
		return
	}

	namespace := r.URL.Query().Get("namespace")

	if h.snapshotStore == nil {
		respondError(w, http.StatusServiceUnavailable, "Snapshot store not configured")
		return
	}

	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")
	if fromStr == "" || toStr == "" {
		respondError(w, http.StatusBadRequest, "Both 'from' and 'to' query parameters are required (ISO 8601 date)")
		return
	}

	fromTime, err := parseISODate(fromStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, fmt.Sprintf("Invalid 'from' date: %v", err))
		return
	}

	toTime, err := parseISODate(toStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, fmt.Sprintf("Invalid 'to' date: %v", err))
		return
	}

	if toTime.Before(fromTime) {
		respondError(w, http.StatusBadRequest, "'to' date must be after 'from' date")
		return
	}

	// Fetch snapshots in the date range
	snapshots, err := h.snapshotStore.GetByDateRange(clusterID, namespace, fromTime, toTime)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to fetch snapshots: %v", err))
		return
	}

	if len(snapshots) < 2 {
		respondError(w, http.StatusNotFound, "At least 2 snapshots are required in the date range to compute a diff")
		return
	}

	// Use the earliest and latest snapshots in the range
	fromSnap := &snapshots[0]
	toSnap := &snapshots[len(snapshots)-1]

	result := diff.ComputeDiff(fromSnap, toSnap)

	respondJSON(w, http.StatusOK, result)
}

// parseISODate parses an ISO 8601 date string.
// Supports both date-only (2026-04-01) and datetime (2026-04-01T12:00:00Z) formats.
func parseISODate(s string) (time.Time, error) {
	// Try full RFC 3339 first
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	// Try date-only
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("expected ISO 8601 format (e.g. 2026-04-01 or 2026-04-01T12:00:00Z)")
}
