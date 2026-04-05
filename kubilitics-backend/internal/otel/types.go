// Package otel provides OpenTelemetry trace ingestion, storage, and query
// capabilities for the Kubilitics Operational Intelligence Platform.
package otel

import "github.com/kubilitics/kubilitics-backend/internal/events"

// Span represents a single span stored in the spans table.
type Span struct {
	SpanID         string          `db:"span_id" json:"span_id"`
	TraceID        string          `db:"trace_id" json:"trace_id"`
	ParentSpanID   string          `db:"parent_span_id" json:"parent_span_id"`
	ServiceName    string          `db:"service_name" json:"service_name"`
	OperationName  string          `db:"operation_name" json:"operation_name"`
	SpanKind       string          `db:"span_kind" json:"span_kind"`
	StartTime      int64           `db:"start_time" json:"start_time"`
	EndTime        int64           `db:"end_time" json:"end_time"`
	DurationNs     int64           `db:"duration_ns" json:"duration_ns"`
	StatusCode     string          `db:"status_code" json:"status_code"`
	StatusMessage  string          `db:"status_message" json:"status_message"`
	HTTPMethod     string          `db:"http_method" json:"http_method"`
	HTTPURL        string          `db:"http_url" json:"http_url"`
	HTTPStatusCode *int            `db:"http_status_code" json:"http_status_code"`
	HTTPRoute      string          `db:"http_route" json:"http_route"`
	DBSystem       string          `db:"db_system" json:"db_system"`
	DBStatement    string          `db:"db_statement" json:"db_statement"`
	K8sPodName     string          `db:"k8s_pod_name" json:"k8s_pod_name"`
	K8sNamespace   string          `db:"k8s_namespace" json:"k8s_namespace"`
	K8sNodeName    string          `db:"k8s_node_name" json:"k8s_node_name"`
	K8sContainer   string          `db:"k8s_container" json:"k8s_container"`
	K8sDeployment  string          `db:"k8s_deployment" json:"k8s_deployment"`
	UserID         string          `db:"user_id" json:"user_id"`
	ClusterID      string          `db:"cluster_id" json:"cluster_id"`
	Attributes     events.JSONText `db:"attributes" json:"attributes"`
	Events         events.JSONText `db:"events" json:"events"`
	LinkedEventIDs events.JSONText `db:"linked_event_ids" json:"linked_event_ids"`
}

// TraceSummary is a denormalized trace record for fast trace listing.
type TraceSummary struct {
	TraceID       string          `db:"trace_id" json:"trace_id"`
	RootService   string          `db:"root_service" json:"root_service"`
	RootOperation string          `db:"root_operation" json:"root_operation"`
	StartTime     int64           `db:"start_time" json:"start_time"`
	DurationNs    int64           `db:"duration_ns" json:"duration_ns"`
	SpanCount     int             `db:"span_count" json:"span_count"`
	ErrorCount    int             `db:"error_count" json:"error_count"`
	ServiceCount  int             `db:"service_count" json:"service_count"`
	Status        string          `db:"status" json:"status"`
	ClusterID     string          `db:"cluster_id" json:"cluster_id"`
	Services      events.JSONText `db:"services" json:"services"`
	UpdatedAt     int64           `db:"updated_at" json:"updated_at"`
}

// TraceQuery defines filters for querying traces.
type TraceQuery struct {
	ClusterID   string `json:"cluster_id"`
	Service     string `json:"service"`
	Operation   string `json:"operation"`
	Status      string `json:"status"`       // OK, ERROR
	MinDuration int64  `json:"min_duration"` // nanoseconds
	MaxDuration int64  `json:"max_duration"`
	From        int64  `json:"from"` // unix ns
	To          int64  `json:"to"`
	UserID      string `json:"user_id"`
	Limit       int    `json:"limit"`
	Offset      int    `json:"offset"`
}

// SpanQuery defines filters for querying spans.
type SpanQuery struct {
	TraceID string `json:"trace_id"`
}

// TraceDetail holds a trace summary with all its spans (for waterfall view).
type TraceDetail struct {
	Summary TraceSummary `json:"summary"`
	Spans   []Span      `json:"spans"`
}

// ServiceNode represents a service in the service map.
type ServiceNode struct {
	Name        string `json:"name"`
	SpanCount   int    `json:"span_count"`
	ErrorCount  int    `json:"error_count"`
	AvgDuration int64  `json:"avg_duration_ns"`
}

// ServiceEdge represents a call between two services.
type ServiceEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Count  int    `json:"count"`
}

// ServiceMap holds the service dependency graph.
type ServiceMap struct {
	Nodes []ServiceNode `json:"nodes"`
	Edges []ServiceEdge `json:"edges"`
}
