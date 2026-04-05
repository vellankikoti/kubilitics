// Package events provides wide event collection, storage, causality tracking,
// incident correlation, and insight generation for the Events Intelligence
// subsystem of Kubilitics.
package events

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// JSONText is a JSONText that implements sql.Scanner and driver.Valuer
// so SQLite TEXT columns can be scanned into it by sqlx.
type JSONText []byte

func (j JSONText) MarshalJSON() ([]byte, error) {
	if len(j) == 0 {
		return []byte("{}"), nil
	}
	return []byte(j), nil
}

func (j *JSONText) UnmarshalJSON(data []byte) error {
	*j = JSONText(data)
	return nil
}

func (j *JSONText) Scan(src interface{}) error {
	switch v := src.(type) {
	case string:
		*j = JSONText(v)
	case []byte:
		*j = JSONText(v)
	case nil:
		*j = JSONText("{}")
	default:
		return fmt.Errorf("JSONText.Scan: unsupported type %T", src)
	}
	return nil
}

func (j JSONText) Value() (driver.Value, error) {
	if len(j) == 0 {
		return "{}", nil
	}
	return string(j), nil
}

// ---------------------------------------------------------------------------
// Core data types (match migration 047_events_intelligence.sql)
// ---------------------------------------------------------------------------

// WideEvent is the denormalized event record that captures a Kubernetes event
// along with Kubilitics enrichment (health, SPOF, blast radius) and causality.
type WideEvent struct {
	EventID            string          `db:"event_id"             json:"event_id"`
	Timestamp          int64           `db:"timestamp"            json:"timestamp"`
	ClusterID          string          `db:"cluster_id"           json:"cluster_id"`

	// Event fields
	EventType          string          `db:"event_type"           json:"event_type"`
	Reason             string          `db:"reason"               json:"reason"`
	Message            string          `db:"message"              json:"message"`
	SourceComponent    string          `db:"source_component"     json:"source_component"`
	SourceHost         string          `db:"source_host"          json:"source_host"`
	EventCount         int             `db:"event_count"          json:"event_count"`
	FirstSeen          int64           `db:"first_seen"           json:"first_seen"`
	LastSeen           int64           `db:"last_seen"            json:"last_seen"`

	// Resource fields
	ResourceKind       string          `db:"resource_kind"        json:"resource_kind"`
	ResourceName       string          `db:"resource_name"        json:"resource_name"`
	ResourceNamespace  string          `db:"resource_namespace"   json:"resource_namespace"`
	ResourceUID        string          `db:"resource_uid"         json:"resource_uid"`
	ResourceAPIVersion string          `db:"resource_api_version" json:"resource_api_version"`
	OwnerKind          string          `db:"owner_kind"           json:"owner_kind"`
	OwnerName          string          `db:"owner_name"           json:"owner_name"`

	// K8s context
	NodeName           string          `db:"node_name"            json:"node_name"`

	// Kubilitics enrichment
	HealthScore        *float64        `db:"health_score"         json:"health_score,omitempty"`
	IsSPOF             int             `db:"is_spof"              json:"is_spof"`
	BlastRadius        int             `db:"blast_radius"         json:"blast_radius"`
	Severity           string          `db:"severity"             json:"severity"`

	// Causality
	CausedByEventID    *string         `db:"caused_by_event_id"   json:"caused_by_event_id,omitempty"`
	CorrelationGroupID string          `db:"correlation_group_id" json:"correlation_group_id"`

	// Extensible dimensions
	Dimensions         JSONText `db:"dimensions"           json:"dimensions"`
}

// Change tracks a field-level resource mutation.
type Change struct {
	ChangeID          string          `db:"change_id"           json:"change_id"`
	Timestamp         int64           `db:"timestamp"           json:"timestamp"`
	ClusterID         string          `db:"cluster_id"          json:"cluster_id"`
	ResourceKind      string          `db:"resource_kind"       json:"resource_kind"`
	ResourceName      string          `db:"resource_name"       json:"resource_name"`
	ResourceNamespace string          `db:"resource_namespace"  json:"resource_namespace"`
	ResourceUID       string          `db:"resource_uid"        json:"resource_uid"`

	ChangeType        string          `db:"change_type"         json:"change_type"`
	FieldChanges      JSONText `db:"field_changes"       json:"field_changes"`
	ChangeSource      string          `db:"change_source"       json:"change_source"`

	EventsCaused      int             `db:"events_caused"       json:"events_caused"`
	HealthImpact      *float64        `db:"health_impact"       json:"health_impact,omitempty"`

	EventID           *string         `db:"event_id"            json:"event_id,omitempty"`
}

// EventRelationship links two events with a typed, scored relationship.
type EventRelationship struct {
	SourceEventID    string          `db:"source_event_id"    json:"source_event_id"`
	TargetEventID    string          `db:"target_event_id"    json:"target_event_id"`
	RelationshipType string          `db:"relationship_type"  json:"relationship_type"`
	Confidence       float64         `db:"confidence"         json:"confidence"`
	Metadata         JSONText `db:"metadata"           json:"metadata"`
}

// Incident represents a group of correlated events forming an operational incident.
type Incident struct {
	IncidentID       string          `db:"incident_id"        json:"incident_id"`
	StartedAt        int64           `db:"started_at"         json:"started_at"`
	EndedAt          *int64          `db:"ended_at"           json:"ended_at,omitempty"`
	Status           string          `db:"status"             json:"status"`
	Severity         string          `db:"severity"           json:"severity"`

	ClusterID        string          `db:"cluster_id"         json:"cluster_id"`
	Namespace        string          `db:"namespace"          json:"namespace"`

	HealthBefore     *float64        `db:"health_before"      json:"health_before,omitempty"`
	HealthAfter      *float64        `db:"health_after"       json:"health_after,omitempty"`
	HealthLowest     *float64        `db:"health_lowest"      json:"health_lowest,omitempty"`

	RootCauseKind    string          `db:"root_cause_kind"    json:"root_cause_kind"`
	RootCauseName    string          `db:"root_cause_name"    json:"root_cause_name"`
	RootCauseSummary string          `db:"root_cause_summary" json:"root_cause_summary"`

	TTD              *int64          `db:"ttd"                json:"ttd,omitempty"`
	TTR              *int64          `db:"ttr"                json:"ttr,omitempty"`

	Dimensions       JSONText `db:"dimensions"         json:"dimensions"`
}

// IncidentEvent links an event to an incident with a role.
type IncidentEvent struct {
	IncidentID string `db:"incident_id" json:"incident_id"`
	EventID    string `db:"event_id"    json:"event_id"`
	Role       string `db:"role"        json:"role"`
}

// Insight is an automated observation or recommendation.
type Insight struct {
	InsightID string `db:"insight_id" json:"insight_id"`
	Timestamp int64  `db:"timestamp"  json:"timestamp"`
	ClusterID string `db:"cluster_id" json:"cluster_id"`
	Rule      string `db:"rule"       json:"rule"`
	Severity  string `db:"severity"   json:"severity"`
	Title     string `db:"title"      json:"title"`
	Detail    string `db:"detail"     json:"detail"`
	Status    string `db:"status"     json:"status"`
}

// StateSnapshot captures aggregate cluster state at a point in time.
type StateSnapshot struct {
	SnapshotID       string          `db:"snapshot_id"        json:"snapshot_id"`
	Timestamp        int64           `db:"timestamp"          json:"timestamp"`
	ClusterID        string          `db:"cluster_id"         json:"cluster_id"`

	TotalPods        int             `db:"total_pods"         json:"total_pods"`
	RunningPods      int             `db:"running_pods"       json:"running_pods"`
	TotalNodes       int             `db:"total_nodes"        json:"total_nodes"`
	ReadyNodes       int             `db:"ready_nodes"        json:"ready_nodes"`
	HealthScore      float64         `db:"health_score"       json:"health_score"`
	SPOFCount        int             `db:"spof_count"         json:"spof_count"`
	WarningEvents    int             `db:"warning_events"     json:"warning_events"`
	ErrorEvents      int             `db:"error_events"       json:"error_events"`

	NamespaceStates  JSONText `db:"namespace_states"   json:"namespace_states"`
	DeploymentStates JSONText `db:"deployment_states"  json:"deployment_states"`
}

// ---------------------------------------------------------------------------
// Query types
// ---------------------------------------------------------------------------

// EventQuery defines filters for querying wide events.
type EventQuery struct {
	ClusterID     string   `json:"cluster_id"`
	Namespace     string   `json:"namespace,omitempty"`
	ResourceKind  string   `json:"resource_kind,omitempty"`
	ResourceName  string   `json:"resource_name,omitempty"`
	EventType     string   `json:"event_type,omitempty"`
	Reason        string   `json:"reason,omitempty"`
	Severity      string   `json:"severity,omitempty"`
	NodeName      string   `json:"node_name,omitempty"`
	ResourceUID   string   `json:"resource_uid,omitempty"`
	Since         *int64   `json:"since,omitempty"`          // unix ms
	Until         *int64   `json:"until,omitempty"`          // unix ms
	Limit         int      `json:"limit,omitempty"`
	Offset        int      `json:"offset,omitempty"`
}

// WhereClause holds a SQL fragment and its corresponding named parameter values.
type WhereClause struct {
	SQL    string
	Params map[string]interface{}
}

// AnalyzeQuery defines parameters for event analysis.
type AnalyzeQuery struct {
	ClusterID    string `json:"cluster_id"`
	Namespace    string `json:"namespace,omitempty"`
	Since        *int64 `json:"since,omitempty"`
	Until        *int64 `json:"until,omitempty"`
	GroupBy      string `json:"group_by,omitempty"`       // reason, resource_kind, severity, etc.
	TopN         int    `json:"top_n,omitempty"`
}

// AnalyzeResult holds aggregated event analysis output.
type AnalyzeResult struct {
	GroupKey   string   `db:"group_key"  json:"group_key"`
	Count     int64    `db:"count"      json:"count"`
	FirstSeen int64    `db:"first_seen" json:"first_seen"`
	LastSeen  int64    `db:"last_seen"  json:"last_seen"`
	AvgHealth *float64 `db:"avg_health" json:"avg_health,omitempty"`
}

// ---------------------------------------------------------------------------
// Response / enrichment types
// ---------------------------------------------------------------------------

// CausalChain represents a chain of causally linked events.
type CausalChain struct {
	RootEventID string      `json:"root_event_id"`
	Links       []ChainLink `json:"links"`
	Depth       int         `json:"depth"`
}

// ChainLink is one hop in a causal chain.
type ChainLink struct {
	EventID          string  `json:"event_id"`
	Timestamp        int64   `json:"timestamp"`
	Reason           string  `json:"reason"`
	ResourceKind     string  `json:"resource_kind"`
	ResourceName     string  `json:"resource_name"`
	RelationshipType string  `json:"relationship_type"`
	Confidence       float64 `json:"confidence"`
}

// EventContext provides surrounding context for a single event.
type EventContext struct {
	Event          WideEvent           `json:"event"`
	RelatedEvents  []WideEvent         `json:"related_events"`
	Relationships  []EventRelationship `json:"relationships"`
	Changes        []Change            `json:"changes"`
	Incident       *Incident           `json:"incident,omitempty"`
	Neighbors      []TopologyNeighbor  `json:"neighbors,omitempty"`
}

// EventStats holds aggregate statistics for a cluster's events.
type EventStats struct {
	TotalEvents   int64            `json:"total_events"`
	ByType        map[string]int64 `json:"by_type"`
	BySeverity    map[string]int64 `json:"by_severity"`
	ByReason      map[string]int64 `json:"by_reason"`
	Since         int64            `json:"since"`
	Until         int64            `json:"until"`
}

// ---------------------------------------------------------------------------
// Supporting types
// ---------------------------------------------------------------------------

// FieldChange describes a single field mutation within a Change.
type FieldChange struct {
	Field    string `json:"field"`
	OldValue string `json:"old_value"`
	NewValue string `json:"new_value"`
}

// HealthPoint is a timestamped health score.
type HealthPoint struct {
	Timestamp int64   `json:"timestamp"`
	Score     float64 `json:"score"`
}

// SPOFDetail describes a single point of failure.
type SPOFDetail struct {
	ResourceKind string `json:"resource_kind"`
	ResourceName string `json:"resource_name"`
	Namespace    string `json:"namespace"`
	Reason       string `json:"reason"`
}

// BlastRadiusDetail describes the blast radius of a resource.
type BlastRadiusDetail struct {
	ResourceKind    string   `json:"resource_kind"`
	ResourceName    string   `json:"resource_name"`
	Namespace       string   `json:"namespace"`
	AffectedCount   int      `json:"affected_count"`
	AffectedKinds   []string `json:"affected_kinds"`
}

// TopologyNeighbor represents a neighboring resource in the topology graph.
type TopologyNeighbor struct {
	ResourceKind string `json:"resource_kind"`
	ResourceName string `json:"resource_name"`
	Namespace    string `json:"namespace"`
	Direction    string `json:"direction"` // upstream, downstream
	Relationship string `json:"relationship"`
}

// ---------------------------------------------------------------------------
// Time helpers
// ---------------------------------------------------------------------------

// UnixMillis returns the current time as Unix milliseconds.
func UnixMillis() int64 {
	return time.Now().UnixMilli()
}

// TimeFromMillis converts Unix milliseconds to time.Time.
func TimeFromMillis(ms int64) time.Time {
	return time.UnixMilli(ms)
}
