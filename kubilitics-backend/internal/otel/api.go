package otel

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// OTelHandler provides HTTP handlers for OTel trace ingestion and queries.
type OTelHandler struct {
	receiver *Receiver
	store    *Store
}

// NewOTelHandler creates a new OTelHandler.
func NewOTelHandler(receiver *Receiver, store *Store) *OTelHandler {
	return &OTelHandler{receiver: receiver, store: store}
}

// SetupOTelRoutes registers OTel-related routes on the given router.
// The router is expected to be the /api/v1 subrouter.
func SetupOTelRoutes(router *mux.Router, handler *OTelHandler) {
	// OTLP receiver: POST /api/v1/traces (on subrouter)
	router.HandleFunc("/traces", handler.ReceiveTraces).Methods("POST")

	// Trace query APIs (cluster-scoped)
	router.HandleFunc("/clusters/{clusterId}/traces", handler.ListTraces).Methods("GET")
	router.HandleFunc("/clusters/{clusterId}/traces/services", handler.GetServiceMap).Methods("GET")
	router.HandleFunc("/clusters/{clusterId}/traces/{traceId}", handler.GetTrace).Methods("GET")
}

// SetupOTLPStandardRoute registers the OTLP standard endpoint POST /v1/traces
// on the ROOT router (not the /api/v1 subrouter). This is the standard OTLP/HTTP
// endpoint that OTel SDKs expect when configured with OTEL_EXPORTER_OTLP_ENDPOINT.
func SetupOTLPStandardRoute(rootRouter *mux.Router, handler *OTelHandler) {
	rootRouter.HandleFunc("/v1/traces", handler.ReceiveTraces).Methods("POST")
}

// ReceiveTraces handles POST /v1/traces (OTLP JSON).
func (h *OTelHandler) ReceiveTraces(w http.ResponseWriter, r *http.Request) {
	// Limit request body to 10MB to prevent OOM from oversized payloads.
	r.Body = http.MaxBytesReader(w, r.Body, 10*1024*1024)

	var req OTLPTraceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON or payload too large"}`, http.StatusBadRequest)
		return
	}

	clusterIDHint := r.Header.Get("X-Kubilitics-Cluster-Id")
	if err := h.receiver.ProcessTraces(r.Context(), &req, clusterIDHint); err != nil {
		if errors.Is(err, ErrRateLimited) {
			w.Header().Set("Retry-After", "5")
			http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
			return
		}
		http.Error(w, `{"error":"failed to process traces"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("{}"))
}

// ListTraces handles GET /clusters/{clusterId}/traces.
func (h *OTelHandler) ListTraces(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["clusterId"]
	params := r.URL.Query()

	q := TraceQuery{
		ClusterID: clusterID,
		Service:   params.Get("service"),
		Operation: params.Get("operation"),
		Status:    params.Get("status"),
		UserID:    params.Get("user_id"),
	}

	if v := params.Get("min_duration"); v != "" {
		q.MinDuration, _ = strconv.ParseInt(v, 10, 64)
	}
	if v := params.Get("max_duration"); v != "" {
		q.MaxDuration, _ = strconv.ParseInt(v, 10, 64)
	}
	if v := params.Get("from"); v != "" {
		q.From, _ = strconv.ParseInt(v, 10, 64)
	}
	if v := params.Get("to"); v != "" {
		q.To, _ = strconv.ParseInt(v, 10, 64)
	}
	if v := params.Get("limit"); v != "" {
		q.Limit, _ = strconv.Atoi(v)
	}
	if v := params.Get("offset"); v != "" {
		q.Offset, _ = strconv.Atoi(v)
	}

	traces, err := h.store.QueryTraces(r.Context(), q)
	if err != nil {
		http.Error(w, `{"error":"failed to query traces"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(traces)
}

// GetTrace handles GET /clusters/{clusterId}/traces/{traceId}.
func (h *OTelHandler) GetTrace(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	traceID := vars["traceId"]

	detail, err := h.store.GetTrace(r.Context(), traceID)
	if err != nil {
		http.Error(w, `{"error":"trace not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(detail)
}

// GetServiceMap handles GET /clusters/{clusterId}/traces/services.
func (h *OTelHandler) GetServiceMap(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["clusterId"]
	params := r.URL.Query()

	var from, to int64
	if v := params.Get("from"); v != "" {
		from, _ = strconv.ParseInt(v, 10, 64)
	}
	if v := params.Get("to"); v != "" {
		to, _ = strconv.ParseInt(v, 10, 64)
	}

	// Default to last 1 hour if no range specified
	if from == 0 && to == 0 {
		to = int64(^uint64(0) >> 1) // max int64
	}

	svcMap, err := h.store.GetServiceMap(r.Context(), clusterID, from, to)
	if err != nil {
		http.Error(w, `{"error":"failed to get service map"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(svcMap)
}
