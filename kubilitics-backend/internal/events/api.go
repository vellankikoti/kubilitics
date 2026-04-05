package events

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// OTelStore defines the interface for querying OTel trace data.
// This avoids a direct import cycle between events and otel packages.
type OTelStore interface {
	QuerySpansByLinkedEvent(ctx context.Context, clusterID, eventID string, limit int) ([]TraceSummaryView, error)
	QuerySpansByResource(ctx context.Context, clusterID, resourceKind, resourceName, namespace string, from, to int64, limit int) ([]TraceSummaryView, error)
}

// TraceSummaryView is a local mirror of otel.TraceSummary to avoid import cycles.
type TraceSummaryView struct {
	TraceID       string  `db:"trace_id" json:"trace_id"`
	RootService   string  `db:"root_service" json:"root_service"`
	RootOperation string  `db:"root_operation" json:"root_operation"`
	StartTime     int64   `db:"start_time" json:"start_time"`
	DurationNs    int64   `db:"duration_ns" json:"duration_ns"`
	SpanCount     int     `db:"span_count" json:"span_count"`
	ErrorCount    int     `db:"error_count" json:"error_count"`
	ServiceCount  int     `db:"service_count" json:"service_count"`
	Status        string  `db:"status" json:"status"`
	ClusterID     string  `db:"cluster_id" json:"cluster_id"`
	Services      JSONText `db:"services" json:"services"`
	UpdatedAt     int64   `db:"updated_at" json:"updated_at"`
}

// EventSubscriber is satisfied by both *Pipeline and *PipelineManager,
// allowing the handler to stream events regardless of single- vs multi-cluster mode.
type EventSubscriber interface {
	Subscribe() <-chan *WideEvent
	Unsubscribe(ch <-chan *WideEvent)
}

// EventsHandler handles REST API requests for the Events Intelligence subsystem.
type EventsHandler struct {
	subscriber EventSubscriber
	store      *Store
	otelStore  OTelStore
	manager    *PipelineManager // set when created from manager, nil for single-pipeline mode
}

// NewEventsHandler creates a new EventsHandler from a single Pipeline.
func NewEventsHandler(pipeline *Pipeline) *EventsHandler {
	return &EventsHandler{
		subscriber: pipeline,
		store:      pipeline.Store(),
	}
}

// NewEventsHandlerFromManager creates a new EventsHandler backed by a
// PipelineManager (multi-cluster). All clusters share the same event store.
func NewEventsHandlerFromManager(mgr *PipelineManager) *EventsHandler {
	return &EventsHandler{
		subscriber: mgr,
		store:      mgr.GetStore(),
		manager:    mgr,
	}
}

// SetOTelStore sets the OTel store for cross-domain trace queries.
func (h *EventsHandler) SetOTelStore(otelStore OTelStore) {
	h.otelStore = otelStore
}

// SetupEventsRoutes registers all Events Intelligence routes on the given router.
// The router should already be scoped to /api/v1/clusters/{clusterId}.
func SetupEventsRoutes(router *mux.Router, h *EventsHandler) {
	// Events
	router.HandleFunc("/clusters/{clusterId}/events-intelligence/stream", h.StreamEvents).Methods("GET")
	router.HandleFunc("/clusters/{clusterId}/events-intelligence/query", h.QueryEvents).Methods("GET")
	router.HandleFunc("/clusters/{clusterId}/events-intelligence/analyze", h.AnalyzeEvents).Methods("POST")
	router.HandleFunc("/clusters/{clusterId}/events-intelligence/stats", h.GetStats).Methods("GET")
	router.HandleFunc("/clusters/{clusterId}/events-intelligence/{eventId}", h.GetEvent).Methods("GET")
	router.HandleFunc("/clusters/{clusterId}/events-intelligence/{eventId}/chain", h.GetCausalChain).Methods("GET")
	router.HandleFunc("/clusters/{clusterId}/events-intelligence/{eventId}/relationships", h.GetRelationships).Methods("GET")
	router.HandleFunc("/clusters/{clusterId}/events-intelligence/{eventId}/traces", h.GetLinkedTraces).Methods("GET")

	// Resource traces
	router.HandleFunc("/clusters/{clusterId}/resource-traces", h.GetResourceTraces).Methods("GET")

	// Changes
	router.HandleFunc("/clusters/{clusterId}/changes/recent", h.GetRecentChanges).Methods("GET")

	// Incidents
	router.HandleFunc("/clusters/{clusterId}/incidents", h.ListIncidents).Methods("GET")
	router.HandleFunc("/clusters/{clusterId}/incidents/{incidentId}", h.GetIncident).Methods("GET")
	router.HandleFunc("/clusters/{clusterId}/incidents/{incidentId}/events", h.GetIncidentEvents).Methods("GET")

	// Insights
	router.HandleFunc("/clusters/{clusterId}/insights/active", h.GetActiveInsights).Methods("GET")
	router.HandleFunc("/clusters/{clusterId}/insights/{insightId}/dismiss", h.DismissInsight).Methods("POST")

	// Time-travel
	router.HandleFunc("/clusters/{clusterId}/state/at", h.GetStateAt).Methods("GET")

	// Log persistence & cross-pod search
	router.HandleFunc("/clusters/{clusterId}/logs/search", h.SearchLogs).Methods("GET")
	router.HandleFunc("/clusters/{clusterId}/logs/aggregate", h.AggregateLogs).Methods("GET")

	// System-wide health (not cluster-scoped)
	router.HandleFunc("/system/events-health", h.GetSystemHealth).Methods("GET")
}

// ---------------------------------------------------------------------------
// Event handlers
// ---------------------------------------------------------------------------

// QueryEvents returns events matching query parameters.
func (h *EventsHandler) QueryEvents(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["clusterId"]
	ctx := r.Context()

	q := EventQuery{
		ClusterID:    clusterID,
		Namespace:    r.URL.Query().Get("namespace"),
		ResourceKind: r.URL.Query().Get("kind"),
		ResourceName: r.URL.Query().Get("name"),
		EventType:    r.URL.Query().Get("type"),
		Reason:       r.URL.Query().Get("reason"),
		NodeName:     r.URL.Query().Get("node"),
	}

	if v := r.URL.Query().Get("from"); v != "" {
		if ts, err := strconv.ParseInt(v, 10, 64); err == nil {
			q.Since = &ts
		}
	}
	if v := r.URL.Query().Get("to"); v != "" {
		if ts, err := strconv.ParseInt(v, 10, 64); err == nil {
			q.Until = &ts
		}
	}
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			q.Limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			q.Offset = n
		}
	}

	events, err := h.store.QueryEvents(ctx, q)
	if err != nil {
		http.Error(w, fmt.Sprintf("query events: %v", err), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, events)
}

// GetStats returns aggregate event statistics.
func (h *EventsHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["clusterId"]
	ctx := r.Context()

	// Default: last 24 hours.
	until := time.Now().UnixMilli()
	since := until - 24*60*60*1000

	if v := r.URL.Query().Get("from"); v != "" {
		if ts, err := strconv.ParseInt(v, 10, 64); err == nil {
			since = ts
		}
	}
	if v := r.URL.Query().Get("to"); v != "" {
		if ts, err := strconv.ParseInt(v, 10, 64); err == nil {
			until = ts
		}
	}

	stats, err := h.store.GetStats(ctx, clusterID, since, until)
	if err != nil {
		http.Error(w, fmt.Sprintf("get stats: %v", err), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, stats)
}

// GetEvent returns a single event with full context: relationships, related
// events (same resource UID within +/- 1 hour, or same correlation group),
// recent changes on the resource, and any linked incident.
func (h *EventsHandler) GetEvent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["clusterId"]
	eventID := vars["eventId"]
	ctx := r.Context()

	event, err := h.store.GetEvent(ctx, eventID)
	if err != nil {
		http.Error(w, fmt.Sprintf("get event: %v", err), http.StatusNotFound)
		return
	}

	// Build context: relationships, related events, changes, incident.
	rels, _ := h.store.GetRelationships(ctx, eventID)

	// Related events: same resource_uid within +/- 1 hour.
	var relatedEvents []WideEvent
	if event.ResourceUID != "" {
		oneHourMs := int64(3600000)
		from := event.Timestamp - oneHourMs
		to := event.Timestamp + oneHourMs
		relatedEvents, _ = h.store.QueryEvents(ctx, EventQuery{
			ClusterID:   clusterID,
			ResourceUID: event.ResourceUID,
			Since:       &from,
			Until:       &to,
			Limit:       20,
		})
	}

	// Also include events from the same correlation group.
	if event.CorrelationGroupID != "" {
		corrEvents, _ := h.store.GetEventsByCorrelationGroup(ctx, event.CorrelationGroupID)
		// Merge correlation group events, deduplicating by event ID.
		seen := make(map[string]bool, len(relatedEvents))
		for _, e := range relatedEvents {
			seen[e.EventID] = true
		}
		for _, e := range corrEvents {
			if !seen[e.EventID] {
				relatedEvents = append(relatedEvents, e)
				seen[e.EventID] = true
			}
		}
	}

	// Exclude the event itself from related events.
	filtered := relatedEvents[:0]
	for _, e := range relatedEvents {
		if e.EventID != eventID {
			filtered = append(filtered, e)
		}
	}
	relatedEvents = filtered

	// Recent changes for this resource in the last hour.
	changes, _ := h.store.GetRecentChangesForResource(ctx, clusterID, event.ResourceNamespace, event.ResourceKind, event.OwnerName, 5)

	// Linked incident (if any).
	incident, _ := h.store.GetIncidentForEvent(ctx, eventID)

	ec := EventContext{
		Event:         *event,
		RelatedEvents: relatedEvents,
		Relationships: rels,
		Changes:       changes,
		Incident:      incident,
	}

	respondJSON(w, http.StatusOK, ec)
}

// GetCausalChain walks the caused_by_event_id chain to the root event.
func (h *EventsHandler) GetCausalChain(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	eventID := vars["eventId"]
	ctx := r.Context()

	chain := CausalChain{
		RootEventID: eventID,
	}

	currentID := eventID
	visited := make(map[string]bool)
	const maxDepth = 20

	for depth := 0; depth < maxDepth; depth++ {
		if visited[currentID] {
			break
		}
		visited[currentID] = true

		event, err := h.store.GetEvent(ctx, currentID)
		if err != nil {
			break
		}

		link := ChainLink{
			EventID:      event.EventID,
			Timestamp:    event.Timestamp,
			Reason:       event.Reason,
			ResourceKind: event.ResourceKind,
			ResourceName: event.ResourceName,
		}

		// Get relationship info if available.
		rels, _ := h.store.GetRelationships(ctx, currentID)
		for _, rel := range rels {
			if rel.RelationshipType == "caused_by" && rel.SourceEventID == currentID {
				link.RelationshipType = "caused_by"
				link.Confidence = rel.Confidence
				break
			}
		}

		chain.Links = append(chain.Links, link)
		chain.RootEventID = currentID

		if event.CausedByEventID == nil || *event.CausedByEventID == "" {
			break
		}
		currentID = *event.CausedByEventID
	}

	chain.Depth = len(chain.Links)

	// Reverse so root is first.
	for i, j := 0, len(chain.Links)-1; i < j; i, j = i+1, j-1 {
		chain.Links[i], chain.Links[j] = chain.Links[j], chain.Links[i]
	}
	if len(chain.Links) > 0 {
		chain.RootEventID = chain.Links[0].EventID
	}

	respondJSON(w, http.StatusOK, chain)
}

// GetRelationships returns all relationships for an event.
func (h *EventsHandler) GetRelationships(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	eventID := vars["eventId"]
	ctx := r.Context()

	rels, err := h.store.GetRelationships(ctx, eventID)
	if err != nil {
		http.Error(w, fmt.Sprintf("get relationships: %v", err), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, rels)
}

// AnalyzeEvents performs aggregate analysis on events.
func (h *EventsHandler) AnalyzeEvents(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["clusterId"]
	ctx := r.Context()

	var aq AnalyzeQuery
	if err := json.NewDecoder(r.Body).Decode(&aq); err != nil {
		http.Error(w, fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest)
		return
	}
	aq.ClusterID = clusterID

	// Validate group_by column to prevent SQL injection.
	allowedGroupBy := map[string]bool{
		"reason": true, "resource_kind": true, "severity": true,
		"event_type": true, "resource_namespace": true, "node_name": true,
		"source_component": true, "owner_kind": true,
	}
	groupBy := aq.GroupBy
	if groupBy == "" {
		groupBy = "reason"
	}
	if !allowedGroupBy[groupBy] {
		http.Error(w, fmt.Sprintf("invalid group_by column: %s", groupBy), http.StatusBadRequest)
		return
	}

	topN := aq.TopN
	if topN <= 0 {
		topN = 20
	}

	// Build dynamic query.
	var whereClauses []string
	var args []interface{}

	whereClauses = append(whereClauses, "cluster_id = ?")
	args = append(args, clusterID)

	if aq.Namespace != "" {
		whereClauses = append(whereClauses, "resource_namespace = ?")
		args = append(args, aq.Namespace)
	}
	if aq.Since != nil {
		whereClauses = append(whereClauses, "timestamp >= ?")
		args = append(args, *aq.Since)
	}
	if aq.Until != nil {
		whereClauses = append(whereClauses, "timestamp <= ?")
		args = append(args, *aq.Until)
	}

	where := strings.Join(whereClauses, " AND ")
	query := fmt.Sprintf(
		`SELECT %s AS group_key, COUNT(*) AS count,
		 MIN(timestamp) AS first_seen, MAX(timestamp) AS last_seen,
		 AVG(health_score) AS avg_health
		 FROM wide_events WHERE %s
		 GROUP BY %s ORDER BY count DESC LIMIT ?`,
		groupBy, where, groupBy,
	)
	args = append(args, topN)

	var results []AnalyzeResult
	if err := h.store.db.SelectContext(ctx, &results, query, args...); err != nil {
		http.Error(w, fmt.Sprintf("analyze events: %v", err), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, results)
}

// ---------------------------------------------------------------------------
// Changes
// ---------------------------------------------------------------------------

// GetRecentChanges returns recent resource changes.
func (h *EventsHandler) GetRecentChanges(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["clusterId"]
	ctx := r.Context()

	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}

	changes, err := h.store.GetRecentChanges(ctx, clusterID, limit)
	if err != nil {
		http.Error(w, fmt.Sprintf("get recent changes: %v", err), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, changes)
}

// ---------------------------------------------------------------------------
// Incidents
// ---------------------------------------------------------------------------

// ListIncidents returns active incidents and recently resolved ones.
func (h *EventsHandler) ListIncidents(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["clusterId"]
	ctx := r.Context()

	active, err := h.store.GetActiveIncidents(ctx, clusterID)
	if err != nil {
		http.Error(w, fmt.Sprintf("get active incidents: %v", err), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, active)
}

// GetIncident returns a single incident by ID.
func (h *EventsHandler) GetIncident(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	incidentID := vars["incidentId"]
	ctx := r.Context()

	inc, err := h.store.GetIncident(ctx, incidentID)
	if err != nil {
		http.Error(w, fmt.Sprintf("get incident: %v", err), http.StatusNotFound)
		return
	}

	respondJSON(w, http.StatusOK, inc)
}

// GetIncidentEvents returns all events linked to an incident.
func (h *EventsHandler) GetIncidentEvents(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	incidentID := vars["incidentId"]
	ctx := r.Context()

	events, err := h.store.GetIncidentEvents(ctx, incidentID)
	if err != nil {
		http.Error(w, fmt.Sprintf("get incident events: %v", err), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, events)
}

// ---------------------------------------------------------------------------
// Insights
// ---------------------------------------------------------------------------

// GetActiveInsights returns all non-dismissed insights.
func (h *EventsHandler) GetActiveInsights(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["clusterId"]
	ctx := r.Context()

	insights, err := h.store.GetActiveInsights(ctx, clusterID)
	if err != nil {
		http.Error(w, fmt.Sprintf("get active insights: %v", err), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, insights)
}

// DismissInsight marks an insight as dismissed.
func (h *EventsHandler) DismissInsight(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	insightID := vars["insightId"]
	ctx := r.Context()

	if err := h.store.DismissInsight(ctx, insightID); err != nil {
		http.Error(w, fmt.Sprintf("dismiss insight: %v", err), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "dismissed"})
}

// ---------------------------------------------------------------------------
// Time-travel
// ---------------------------------------------------------------------------

// GetStateAt returns the nearest state snapshot at or before the given timestamp.
func (h *EventsHandler) GetStateAt(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["clusterId"]
	ctx := r.Context()

	tStr := r.URL.Query().Get("t")
	if tStr == "" {
		http.Error(w, "missing 't' query parameter (unix ms)", http.StatusBadRequest)
		return
	}

	ts, err := strconv.ParseInt(tStr, 10, 64)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid 't' parameter: %v", err), http.StatusBadRequest)
		return
	}

	snap, err := h.store.GetSnapshotAt(ctx, clusterID, ts)
	if err != nil {
		http.Error(w, fmt.Sprintf("get state at %d: %v", ts, err), http.StatusNotFound)
		return
	}

	respondJSON(w, http.StatusOK, snap)
}

// ---------------------------------------------------------------------------
// Log Search
// ---------------------------------------------------------------------------

// SearchLogs returns stored log lines matching query parameters.
func (h *EventsHandler) SearchLogs(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["clusterId"]
	ctx := r.Context()

	q := LogQuery{
		ClusterID:  clusterID,
		Namespace:  r.URL.Query().Get("namespace"),
		PodName:    r.URL.Query().Get("pod"),
		OwnerKind:  r.URL.Query().Get("owner_kind"),
		OwnerName:  r.URL.Query().Get("owner_name"),
		Level:      r.URL.Query().Get("level"),
		Search:     r.URL.Query().Get("search"),
		FieldQuery: r.URL.Query().Get("field"),
	}

	if v := r.URL.Query().Get("from"); v != "" {
		if ts, err := strconv.ParseInt(v, 10, 64); err == nil {
			q.From = ts
		}
	}
	if v := r.URL.Query().Get("to"); v != "" {
		if ts, err := strconv.ParseInt(v, 10, 64); err == nil {
			q.To = ts
		}
	}
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			q.Limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			q.Offset = n
		}
	}

	logs, err := h.store.QueryLogs(ctx, q)
	if err != nil {
		http.Error(w, fmt.Sprintf("search logs: %v", err), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, logs)
}

// AggregateLogs returns cross-pod aggregated logs for a workload, sorted by
// timestamp. This powers the "show me all ERROR logs across all checkout-api pods"
// use case.
func (h *EventsHandler) AggregateLogs(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["clusterId"]
	ctx := r.Context()

	ownerKind := r.URL.Query().Get("owner_kind")
	ownerName := r.URL.Query().Get("owner_name")
	namespace := r.URL.Query().Get("namespace")

	if ownerKind == "" || ownerName == "" {
		http.Error(w, "owner_kind and owner_name are required", http.StatusBadRequest)
		return
	}

	q := LogQuery{
		ClusterID: clusterID,
		Namespace: namespace,
		OwnerKind: ownerKind,
		OwnerName: ownerName,
		Level:     r.URL.Query().Get("level"),
		Limit:     500,
	}

	if v := r.URL.Query().Get("from"); v != "" {
		if ts, err := strconv.ParseInt(v, 10, 64); err == nil {
			q.From = ts
		}
	}
	if v := r.URL.Query().Get("to"); v != "" {
		if ts, err := strconv.ParseInt(v, 10, 64); err == nil {
			q.To = ts
		}
	}
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			q.Limit = n
		}
	}

	logs, err := h.store.QueryLogs(ctx, q)
	if err != nil {
		http.Error(w, fmt.Sprintf("aggregate logs: %v", err), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, logs)
}

// ---------------------------------------------------------------------------
// Linked Traces
// ---------------------------------------------------------------------------

// GetLinkedTraces returns traces correlated with a specific event, either via
// linked_event_ids or by matching the event's resource pod + time window.
func (h *EventsHandler) GetLinkedTraces(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["clusterId"]
	eventID := vars["eventId"]
	ctx := r.Context()

	if h.otelStore == nil {
		respondJSON(w, http.StatusOK, []TraceSummaryView{})
		return
	}

	// Try linked_event_ids first
	traces, err := h.otelStore.QuerySpansByLinkedEvent(ctx, clusterID, eventID, 10)
	if err != nil {
		http.Error(w, fmt.Sprintf("query linked traces: %v", err), http.StatusInternalServerError)
		return
	}

	// If no linked traces found, fall back to pod + time window match
	if len(traces) == 0 {
		event, err := h.store.GetEvent(ctx, eventID)
		if err != nil {
			respondJSON(w, http.StatusOK, []TraceSummaryView{})
			return
		}

		// Use event's resource name as pod name, +/- 5 minutes in nanoseconds
		windowMs := int64(300_000)
		fromNs := (event.Timestamp - windowMs) * 1_000_000
		toNs := (event.Timestamp + windowMs) * 1_000_000

		traces, _ = h.otelStore.QuerySpansByResource(
			ctx, clusterID, event.ResourceKind, event.ResourceName,
			event.ResourceNamespace, fromNs, toNs, 10,
		)
	}

	if traces == nil {
		traces = []TraceSummaryView{}
	}
	respondJSON(w, http.StatusOK, traces)
}

// GetResourceTraces returns traces for a specific K8s resource (pod, deployment, service).
func (h *EventsHandler) GetResourceTraces(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["clusterId"]
	ctx := r.Context()

	kind := r.URL.Query().Get("kind")
	name := r.URL.Query().Get("name")
	namespace := r.URL.Query().Get("namespace")

	if kind == "" || name == "" {
		http.Error(w, "kind and name are required", http.StatusBadRequest)
		return
	}

	if h.otelStore == nil {
		respondJSON(w, http.StatusOK, []TraceSummaryView{})
		return
	}

	var from, to int64
	if v := r.URL.Query().Get("from"); v != "" {
		if ts, err := strconv.ParseInt(v, 10, 64); err == nil {
			from = ts
		}
	}
	if v := r.URL.Query().Get("to"); v != "" {
		if ts, err := strconv.ParseInt(v, 10, 64); err == nil {
			to = ts
		}
	}

	limit := 20
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}

	traces, err := h.otelStore.QuerySpansByResource(ctx, clusterID, kind, name, namespace, from, to, limit)
	if err != nil {
		http.Error(w, fmt.Sprintf("query resource traces: %v", err), http.StatusInternalServerError)
		return
	}

	if traces == nil {
		traces = []TraceSummaryView{}
	}
	respondJSON(w, http.StatusOK, traces)
}

// ---------------------------------------------------------------------------
// System Health
// ---------------------------------------------------------------------------

// GetSystemHealth returns the health of the entire Events Intelligence system
// across all clusters. This endpoint is NOT cluster-scoped.
func (h *EventsHandler) GetSystemHealth(w http.ResponseWriter, r *http.Request) {
	if h.manager == nil {
		http.Error(w, "events health not available (single-pipeline mode)", http.StatusNotImplemented)
		return
	}
	health := h.manager.Health(r.Context())
	respondJSON(w, http.StatusOK, health)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// respondJSON writes a JSON response with the given status code.
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}
}
