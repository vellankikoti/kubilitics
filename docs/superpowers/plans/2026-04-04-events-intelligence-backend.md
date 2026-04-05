# Events Intelligence Backend — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the Go backend for Events Intelligence — K8s event collection, wide event enrichment, causality engine, change tracking, incident detection, and REST APIs.

**Architecture:** Extends the existing kubilitics-backend with a new `internal/events/` package. Uses existing K8s informer infrastructure for event collection, existing SQLite database (SQLx + modernc.org/sqlite) for storage, and Gorilla Mux for API routes. The events pipeline: K8s Informers → Collector → Enricher → Causality Engine → Store → REST API.

**Tech Stack:** Go, SQLx, modernc.org/sqlite, client-go informers, Gorilla Mux, SSE (Server-Sent Events)

**Existing patterns to follow:**
- Module: `github.com/kubilitics/kubilitics-backend`
- Services: `internal/service/{domain}_service.go` with interface
- Models: `internal/models/{domain}.go` with JSON tags
- Handlers: `internal/api/rest/{resource}.go` with `h.wrapWithRBAC()`
- Repository: `internal/repository/` with interface + SQLite impl
- Migrations: `migrations/NNN_description.sql` auto-run on startup
- K8s client: `internal/k8s/client.go` + `informer.go`

---

## File Map

### New Files

```
kubilitics-backend/
├── migrations/
│   └── 021_events_intelligence.sql        — All new tables
├── internal/
│   └── events/
│       ├── types.go                       — WideEvent, Change, Incident, Insight structs
│       ├── store.go                       — SQLite CRUD for all events tables
│       ├── collector.go                   — K8s event watcher + resource change detector
│       ├── enricher.go                    — Enrich events with health, SPOF, blast radius
│       ├── causality.go                   — 6 causal inference rules
│       ├── changes.go                     — Change detection, field-level diffs
│       ├── incidents.go                   — Incident detection, grouping, narrative
│       ├── insights.go                    — Proactive anomaly detection rules
│       ├── snapshots.go                   — State snapshots for time-travel
│       ├── relationships.go              — Event relationship builder
│       ├── pipeline.go                    — Orchestrates: collect → enrich → cause → store
│       ├── api.go                         — HTTP handlers for /api/v1/events/*
│       └── sse.go                         — Server-Sent Events stream
```

### Modified Files

```
kubilitics-backend/
├── cmd/server/main.go                     — Initialize EventsPipeline, register routes
├── internal/api/rest/handler.go           — Add events routes to SetupRoutes()
```

---

## Task 1: Database Migration — All Tables

**Files:**
- Create: `kubilitics-backend/migrations/021_events_intelligence.sql`

- [ ] **Step 1: Write the migration file**

```sql
-- 021_events_intelligence.sql
-- Events Intelligence: wide events, changes, relationships, incidents, insights, snapshots

-- Wide events table
CREATE TABLE IF NOT EXISTS wide_events (
    event_id            TEXT PRIMARY KEY,
    timestamp           INTEGER NOT NULL,
    cluster_id          TEXT NOT NULL,
    event_type          TEXT NOT NULL DEFAULT 'Normal',
    event_reason        TEXT NOT NULL DEFAULT '',
    event_message       TEXT NOT NULL DEFAULT '',
    resource_kind       TEXT NOT NULL DEFAULT '',
    resource_name       TEXT NOT NULL DEFAULT '',
    resource_uid        TEXT NOT NULL DEFAULT '',
    namespace           TEXT NOT NULL DEFAULT '',
    node_name           TEXT NOT NULL DEFAULT '',
    health_score        REAL,
    health_score_before REAL,
    health_delta        REAL,
    is_spof             INTEGER NOT NULL DEFAULT 0,
    blast_radius        REAL,
    risk_level          TEXT NOT NULL DEFAULT '',
    caused_by_event_id  TEXT,
    causal_confidence   REAL,
    causal_rule         TEXT,
    correlation_group_id TEXT,
    incident_id         TEXT,
    dimensions          TEXT NOT NULL DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS idx_we_timestamp ON wide_events(timestamp);
CREATE INDEX IF NOT EXISTS idx_we_cluster ON wide_events(cluster_id);
CREATE INDEX IF NOT EXISTS idx_we_namespace ON wide_events(namespace);
CREATE INDEX IF NOT EXISTS idx_we_kind ON wide_events(resource_kind);
CREATE INDEX IF NOT EXISTS idx_we_reason ON wide_events(event_reason);
CREATE INDEX IF NOT EXISTS idx_we_type ON wide_events(event_type);
CREATE INDEX IF NOT EXISTS idx_we_health ON wide_events(health_score);
CREATE INDEX IF NOT EXISTS idx_we_node ON wide_events(node_name);
CREATE INDEX IF NOT EXISTS idx_we_corr_group ON wide_events(correlation_group_id);
CREATE INDEX IF NOT EXISTS idx_we_incident ON wide_events(incident_id);
CREATE INDEX IF NOT EXISTS idx_we_resource ON wide_events(resource_uid);

-- Changes table
CREATE TABLE IF NOT EXISTS changes (
    change_id       TEXT PRIMARY KEY,
    timestamp       INTEGER NOT NULL,
    cluster_id      TEXT NOT NULL,
    resource_kind   TEXT NOT NULL,
    resource_name   TEXT NOT NULL,
    resource_uid    TEXT NOT NULL DEFAULT '',
    namespace       TEXT NOT NULL,
    change_type     TEXT NOT NULL,
    field_changes   TEXT NOT NULL DEFAULT '[]',
    change_source   TEXT NOT NULL DEFAULT '',
    events_caused   INTEGER NOT NULL DEFAULT 0,
    health_impact   REAL,
    incident_id     TEXT,
    event_id        TEXT,
    dimensions      TEXT NOT NULL DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS idx_ch_timestamp ON changes(timestamp);
CREATE INDEX IF NOT EXISTS idx_ch_cluster ON changes(cluster_id);
CREATE INDEX IF NOT EXISTS idx_ch_namespace ON changes(namespace);
CREATE INDEX IF NOT EXISTS idx_ch_type ON changes(change_type);
CREATE INDEX IF NOT EXISTS idx_ch_resource ON changes(resource_uid);

-- Event relationships
CREATE TABLE IF NOT EXISTS event_relationships (
    source_event_id TEXT NOT NULL,
    target_event_id TEXT NOT NULL,
    relationship    TEXT NOT NULL,
    confidence      REAL NOT NULL DEFAULT 1.0,
    metadata        TEXT NOT NULL DEFAULT '{}',
    PRIMARY KEY (source_event_id, target_event_id, relationship)
);

CREATE INDEX IF NOT EXISTS idx_er_source ON event_relationships(source_event_id);
CREATE INDEX IF NOT EXISTS idx_er_target ON event_relationships(target_event_id);

-- Incidents
CREATE TABLE IF NOT EXISTS incidents (
    incident_id         TEXT PRIMARY KEY,
    started_at          INTEGER NOT NULL,
    ended_at            INTEGER,
    status              TEXT NOT NULL DEFAULT 'active',
    severity            TEXT NOT NULL DEFAULT 'medium',
    cluster_id          TEXT NOT NULL,
    namespace           TEXT,
    primary_resource    TEXT,
    events_count        INTEGER NOT NULL DEFAULT 0,
    health_before       REAL,
    health_after        REAL,
    health_lowest       REAL,
    root_cause_event_id TEXT,
    root_cause_summary  TEXT,
    resolution_event_id TEXT,
    ttd_seconds         INTEGER,
    ttr_seconds         INTEGER,
    dimensions          TEXT NOT NULL DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS idx_inc_cluster ON incidents(cluster_id);
CREATE INDEX IF NOT EXISTS idx_inc_status ON incidents(status);
CREATE INDEX IF NOT EXISTS idx_inc_started ON incidents(started_at);

-- Incident-event links
CREATE TABLE IF NOT EXISTS incident_events (
    incident_id TEXT NOT NULL,
    event_id    TEXT NOT NULL,
    role        TEXT NOT NULL DEFAULT 'symptom',
    PRIMARY KEY (incident_id, event_id)
);

-- Proactive insights
CREATE TABLE IF NOT EXISTS insights (
    insight_id  TEXT PRIMARY KEY,
    timestamp   INTEGER NOT NULL,
    cluster_id  TEXT NOT NULL,
    rule        TEXT NOT NULL,
    severity    TEXT NOT NULL DEFAULT 'info',
    title       TEXT NOT NULL,
    detail      TEXT NOT NULL DEFAULT '',
    namespace   TEXT,
    status      TEXT NOT NULL DEFAULT 'active',
    resolved_at INTEGER,
    dimensions  TEXT NOT NULL DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS idx_ins_cluster ON insights(cluster_id);
CREATE INDEX IF NOT EXISTS idx_ins_status ON insights(status);

-- State snapshots for time-travel
CREATE TABLE IF NOT EXISTS state_snapshots (
    snapshot_id       TEXT PRIMARY KEY,
    timestamp         INTEGER NOT NULL,
    cluster_id        TEXT NOT NULL,
    health_score      REAL,
    total_pods        INTEGER NOT NULL DEFAULT 0,
    running_pods      INTEGER NOT NULL DEFAULT 0,
    pending_pods      INTEGER NOT NULL DEFAULT 0,
    failed_pods       INTEGER NOT NULL DEFAULT 0,
    spof_count        INTEGER NOT NULL DEFAULT 0,
    warning_count     INTEGER NOT NULL DEFAULT 0,
    node_count        INTEGER NOT NULL DEFAULT 0,
    nodes_ready       INTEGER NOT NULL DEFAULT 0,
    namespace_states  TEXT NOT NULL DEFAULT '[]',
    deployment_states TEXT NOT NULL DEFAULT '[]'
);

CREATE INDEX IF NOT EXISTS idx_ss_cluster ON state_snapshots(cluster_id);
CREATE INDEX IF NOT EXISTS idx_ss_timestamp ON state_snapshots(timestamp);
```

- [ ] **Step 2: Verify migration runs**

Run: `cd kubilitics-backend && go run ./cmd/server/ &` (start server, migration runs automatically on startup)

Check logs for: `Applied migration 021_events_intelligence.sql`

Kill the server after verification.

- [ ] **Step 3: Commit**

```bash
git add kubilitics-backend/migrations/021_events_intelligence.sql
git commit -m "feat(events): add database schema for Events Intelligence — wide events, changes, incidents, insights, snapshots"
```

---

## Task 2: Types — Data Structures

**Files:**
- Create: `kubilitics-backend/internal/events/types.go`

- [ ] **Step 1: Write the types file**

```go
package events

import (
	"encoding/json"
	"time"
)

// WideEvent represents a single K8s event enriched with Kubilitics context.
type WideEvent struct {
	EventID           string          `json:"event_id" db:"event_id"`
	Timestamp         int64           `json:"timestamp" db:"timestamp"`
	ClusterID         string          `json:"cluster_id" db:"cluster_id"`
	EventType         string          `json:"event_type" db:"event_type"`
	EventReason       string          `json:"event_reason" db:"event_reason"`
	EventMessage      string          `json:"event_message" db:"event_message"`
	ResourceKind      string          `json:"resource_kind" db:"resource_kind"`
	ResourceName      string          `json:"resource_name" db:"resource_name"`
	ResourceUID       string          `json:"resource_uid" db:"resource_uid"`
	Namespace         string          `json:"namespace" db:"namespace"`
	NodeName          string          `json:"node_name" db:"node_name"`
	HealthScore       *float64        `json:"health_score" db:"health_score"`
	HealthScoreBefore *float64        `json:"health_score_before" db:"health_score_before"`
	HealthDelta       *float64        `json:"health_delta" db:"health_delta"`
	IsSPOF            bool            `json:"is_spof" db:"is_spof"`
	BlastRadius       *float64        `json:"blast_radius" db:"blast_radius"`
	RiskLevel         string          `json:"risk_level" db:"risk_level"`
	CausedByEventID   string          `json:"caused_by_event_id,omitempty" db:"caused_by_event_id"`
	CausalConfidence  *float64        `json:"causal_confidence,omitempty" db:"causal_confidence"`
	CausalRule        string          `json:"causal_rule,omitempty" db:"causal_rule"`
	CorrelationGroup  string          `json:"correlation_group_id,omitempty" db:"correlation_group_id"`
	IncidentID        string          `json:"incident_id,omitempty" db:"incident_id"`
	Dimensions        json.RawMessage `json:"dimensions" db:"dimensions"`
}

// TimestampTime returns the timestamp as a time.Time.
func (w *WideEvent) TimestampTime() time.Time {
	return time.UnixMilli(w.Timestamp)
}

// FieldChange represents a single field-level diff in a resource change.
type FieldChange struct {
	Path string      `json:"path"`
	Old  interface{} `json:"old"`
	New  interface{} `json:"new"`
}

// Change represents a first-class resource change (deployment, config update, etc.).
type Change struct {
	ChangeID     string          `json:"change_id" db:"change_id"`
	Timestamp    int64           `json:"timestamp" db:"timestamp"`
	ClusterID    string          `json:"cluster_id" db:"cluster_id"`
	ResourceKind string          `json:"resource_kind" db:"resource_kind"`
	ResourceName string          `json:"resource_name" db:"resource_name"`
	ResourceUID  string          `json:"resource_uid" db:"resource_uid"`
	Namespace    string          `json:"namespace" db:"namespace"`
	ChangeType   string          `json:"change_type" db:"change_type"`
	FieldChanges json.RawMessage `json:"field_changes" db:"field_changes"`
	ChangeSource string          `json:"change_source" db:"change_source"`
	EventsCaused int             `json:"events_caused" db:"events_caused"`
	HealthImpact *float64        `json:"health_impact" db:"health_impact"`
	IncidentID   string          `json:"incident_id,omitempty" db:"incident_id"`
	EventID      string          `json:"event_id" db:"event_id"`
	Dimensions   json.RawMessage `json:"dimensions" db:"dimensions"`
}

// EventRelationship represents a typed link between two events.
type EventRelationship struct {
	SourceEventID string          `json:"source_event_id" db:"source_event_id"`
	TargetEventID string          `json:"target_event_id" db:"target_event_id"`
	Relationship  string          `json:"relationship" db:"relationship"`
	Confidence    float64         `json:"confidence" db:"confidence"`
	Metadata      json.RawMessage `json:"metadata" db:"metadata"`
}

// Incident groups causally-related events into a narrative.
type Incident struct {
	IncidentID        string          `json:"incident_id" db:"incident_id"`
	StartedAt         int64           `json:"started_at" db:"started_at"`
	EndedAt           *int64          `json:"ended_at" db:"ended_at"`
	Status            string          `json:"status" db:"status"`
	Severity          string          `json:"severity" db:"severity"`
	ClusterID         string          `json:"cluster_id" db:"cluster_id"`
	Namespace         string          `json:"namespace" db:"namespace"`
	PrimaryResource   string          `json:"primary_resource" db:"primary_resource"`
	EventsCount       int             `json:"events_count" db:"events_count"`
	HealthBefore      *float64        `json:"health_before" db:"health_before"`
	HealthAfter       *float64        `json:"health_after" db:"health_after"`
	HealthLowest      *float64        `json:"health_lowest" db:"health_lowest"`
	RootCauseEventID  string          `json:"root_cause_event_id" db:"root_cause_event_id"`
	RootCauseSummary  string          `json:"root_cause_summary" db:"root_cause_summary"`
	ResolutionEventID string          `json:"resolution_event_id" db:"resolution_event_id"`
	TTDSeconds        *int            `json:"ttd_seconds" db:"ttd_seconds"`
	TTRSeconds        *int            `json:"ttr_seconds" db:"ttr_seconds"`
	Dimensions        json.RawMessage `json:"dimensions" db:"dimensions"`
}

// IncidentEvent links an event to an incident with a role.
type IncidentEvent struct {
	IncidentID string `json:"incident_id" db:"incident_id"`
	EventID    string `json:"event_id" db:"event_id"`
	Role       string `json:"role" db:"role"` // trigger, symptom, cause, resolution
}

// Insight represents a proactive anomaly detection result.
type Insight struct {
	InsightID  string          `json:"insight_id" db:"insight_id"`
	Timestamp  int64           `json:"timestamp" db:"timestamp"`
	ClusterID  string          `json:"cluster_id" db:"cluster_id"`
	Rule       string          `json:"rule" db:"rule"`
	Severity   string          `json:"severity" db:"severity"`
	Title      string          `json:"title" db:"title"`
	Detail     string          `json:"detail" db:"detail"`
	Namespace  string          `json:"namespace" db:"namespace"`
	Status     string          `json:"status" db:"status"`
	ResolvedAt *int64          `json:"resolved_at" db:"resolved_at"`
	Dimensions json.RawMessage `json:"dimensions" db:"dimensions"`
}

// StateSnapshot captures cluster state at a point in time.
type StateSnapshot struct {
	SnapshotID       string          `json:"snapshot_id" db:"snapshot_id"`
	Timestamp        int64           `json:"timestamp" db:"timestamp"`
	ClusterID        string          `json:"cluster_id" db:"cluster_id"`
	HealthScore      *float64        `json:"health_score" db:"health_score"`
	TotalPods        int             `json:"total_pods" db:"total_pods"`
	RunningPods      int             `json:"running_pods" db:"running_pods"`
	PendingPods      int             `json:"pending_pods" db:"pending_pods"`
	FailedPods       int             `json:"failed_pods" db:"failed_pods"`
	SPOFCount        int             `json:"spof_count" db:"spof_count"`
	WarningCount     int             `json:"warning_count" db:"warning_count"`
	NodeCount        int             `json:"node_count" db:"node_count"`
	NodesReady       int             `json:"nodes_ready" db:"nodes_ready"`
	NamespaceStates  json.RawMessage `json:"namespace_states" db:"namespace_states"`
	DeploymentStates json.RawMessage `json:"deployment_states" db:"deployment_states"`
}

// --- Query types ---

// EventQuery represents a timeline query.
type EventQuery struct {
	ClusterID      string `json:"cluster_id"`
	From           int64  `json:"from"`
	To             int64  `json:"to"`
	Namespace      string `json:"namespace"`
	ResourceKind   string `json:"resource_kind"`
	ResourceName   string `json:"resource_name"`
	EventType      string `json:"event_type"`
	EventReason    string `json:"event_reason"`
	NodeName       string `json:"node_name"`
	SPOFOnly       bool   `json:"spof_only"`
	MinHealthScore *float64 `json:"min_health_score"`
	MaxHealthScore *float64 `json:"max_health_score"`
	Limit          int    `json:"limit"`
	Offset         int    `json:"offset"`
}

// AnalyzeQuery represents a dimensional analysis query.
type AnalyzeQuery struct {
	ClusterID string          `json:"cluster_id"`
	Select    string          `json:"select"`    // COUNT, AVG, MAX, MIN
	Field     string          `json:"field"`     // field to aggregate
	Where     []WhereClause   `json:"where"`
	GroupBy   string          `json:"group_by"`
	TimeRange string          `json:"time_range"` // 1h, 24h, 7d
	Interval  string          `json:"interval"`   // 5m, 1h, 1d (for time-series)
}

// WhereClause is a single filter condition.
type WhereClause struct {
	Field string `json:"field"`
	Op    string `json:"op"`    // =, !=, >, <, contains
	Value string `json:"value"`
}

// AnalyzeResult is one row of dimensional analysis output.
type AnalyzeResult struct {
	Group string  `json:"group"`
	Value float64 `json:"value"`
	Time  *int64  `json:"time,omitempty"` // for time-series
}

// CausalChain represents the full cause-effect chain for an event.
type CausalChain struct {
	RootCause *WideEvent       `json:"root_cause"`
	Chain     []ChainLink      `json:"chain"`
	Summary   string           `json:"summary"`
}

// ChainLink is one event in a causal chain.
type ChainLink struct {
	Event      WideEvent `json:"event"`
	CausedBy   string    `json:"caused_by"`
	Confidence float64   `json:"confidence"`
	Role       string    `json:"role"` // cause, symptom, resolution
}

// EventContext is the enriched context shown in the UI panel.
type EventContext struct {
	Event         WideEvent           `json:"event"`
	HealthTrend   []HealthPoint       `json:"health_trend"`
	SPOFDetail    *SPOFDetail         `json:"spof_detail"`
	BlastRadius   *BlastRadiusDetail  `json:"blast_radius"`
	Topology      []TopologyNeighbor  `json:"topology_neighbors"`
	RelatedEvents []WideEvent         `json:"related_events"`
	CausalChain   *CausalChain        `json:"causal_chain"`
	RecentChange  *Change             `json:"recent_change"`
}

type HealthPoint struct {
	Timestamp int64   `json:"timestamp"`
	Score     float64 `json:"score"`
}

type SPOFDetail struct {
	IsSPOF       bool `json:"is_spof"`
	ReplicaCount int  `json:"replica_count"`
	PDBExists    bool `json:"pdb_exists"`
}

type BlastRadiusDetail struct {
	TotalAffected    int      `json:"total_affected"`
	ServicesAffected []string `json:"services_affected"`
	CascadeDepth     int      `json:"cascade_depth"`
	Percentage       float64  `json:"percentage"`
}

type TopologyNeighbor struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
	UID  string `json:"uid"`
	Role string `json:"role"` // depends_on, depended_by, same_node
}

// EventStats summarizes event activity.
type EventStats struct {
	Total24h        int                `json:"total_24h"`
	Warnings24h     int                `json:"warnings_24h"`
	HealthChanges   int                `json:"health_changes_24h"`
	ActiveIncidents int                `json:"active_incidents"`
	TopReasons      []ReasonCount      `json:"top_reasons"`
	TopNamespaces   []NamespaceCount   `json:"top_namespaces"`
}

type ReasonCount struct {
	Reason string `json:"reason"`
	Count  int    `json:"count"`
}

type NamespaceCount struct {
	Namespace string `json:"namespace"`
	Count     int    `json:"count"`
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd kubilitics-backend && go build ./internal/events/`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add kubilitics-backend/internal/events/types.go
git commit -m "feat(events): add data types for wide events, changes, incidents, insights"
```

---

## Task 3: Store — SQLite CRUD Layer

**Files:**
- Create: `kubilitics-backend/internal/events/store.go`

- [ ] **Step 1: Write the store**

```go
package events

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

// Store handles SQLite persistence for all events intelligence data.
type Store struct {
	db *sqlx.DB
}

// NewStore creates a new events store using the existing SQLx database connection.
func NewStore(db *sqlx.DB) *Store {
	return &Store{db: db}
}

// --- Wide Events ---

func (s *Store) InsertEvent(ctx context.Context, e *WideEvent) error {
	_, err := s.db.NamedExecContext(ctx, `
		INSERT OR REPLACE INTO wide_events (
			event_id, timestamp, cluster_id, event_type, event_reason, event_message,
			resource_kind, resource_name, resource_uid, namespace, node_name,
			health_score, health_score_before, health_delta, is_spof, blast_radius,
			risk_level, caused_by_event_id, causal_confidence, causal_rule,
			correlation_group_id, incident_id, dimensions
		) VALUES (
			:event_id, :timestamp, :cluster_id, :event_type, :event_reason, :event_message,
			:resource_kind, :resource_name, :resource_uid, :namespace, :node_name,
			:health_score, :health_score_before, :health_delta, :is_spof, :blast_radius,
			:risk_level, :caused_by_event_id, :causal_confidence, :causal_rule,
			:correlation_group_id, :incident_id, :dimensions
		)`, e)
	return err
}

func (s *Store) QueryEvents(ctx context.Context, q EventQuery) ([]WideEvent, error) {
	var conditions []string
	args := map[string]interface{}{}

	conditions = append(conditions, "cluster_id = :cluster_id")
	args["cluster_id"] = q.ClusterID

	if q.From > 0 {
		conditions = append(conditions, "timestamp >= :from")
		args["from"] = q.From
	}
	if q.To > 0 {
		conditions = append(conditions, "timestamp <= :to")
		args["to"] = q.To
	}
	if q.Namespace != "" {
		conditions = append(conditions, "namespace = :namespace")
		args["namespace"] = q.Namespace
	}
	if q.ResourceKind != "" {
		conditions = append(conditions, "resource_kind = :resource_kind")
		args["resource_kind"] = q.ResourceKind
	}
	if q.ResourceName != "" {
		conditions = append(conditions, "resource_name = :resource_name")
		args["resource_name"] = q.ResourceName
	}
	if q.EventType != "" {
		conditions = append(conditions, "event_type = :event_type")
		args["event_type"] = q.EventType
	}
	if q.EventReason != "" {
		conditions = append(conditions, "event_reason = :event_reason")
		args["event_reason"] = q.EventReason
	}
	if q.NodeName != "" {
		conditions = append(conditions, "node_name = :node_name")
		args["node_name"] = q.NodeName
	}
	if q.SPOFOnly {
		conditions = append(conditions, "is_spof = 1")
	}

	where := strings.Join(conditions, " AND ")
	limit := q.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	args["limit"] = limit
	args["offset"] = q.Offset

	query := fmt.Sprintf(
		"SELECT * FROM wide_events WHERE %s ORDER BY timestamp DESC LIMIT :limit OFFSET :offset",
		where,
	)

	var events []WideEvent
	rows, err := s.db.NamedQueryContext(ctx, query, args)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var e WideEvent
		if err := rows.StructScan(&e); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, nil
}

func (s *Store) GetEvent(ctx context.Context, eventID string) (*WideEvent, error) {
	var e WideEvent
	err := s.db.GetContext(ctx, &e, "SELECT * FROM wide_events WHERE event_id = ?", eventID)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (s *Store) FindRecentEvent(ctx context.Context, clusterID, resourceKind, resourceName string, reasons []string, within time.Duration, before int64) (*WideEvent, error) {
	since := before - within.Milliseconds()
	query := `SELECT * FROM wide_events 
		WHERE cluster_id = ? AND resource_kind = ? AND resource_name = ? 
		AND event_reason IN (?) AND timestamp BETWEEN ? AND ?
		ORDER BY timestamp DESC LIMIT 1`

	query, args, err := sqlx.In(query, clusterID, resourceKind, resourceName, reasons, since, before)
	if err != nil {
		return nil, err
	}
	query = s.db.Rebind(query)

	var e WideEvent
	err = s.db.GetContext(ctx, &e, query, args...)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (s *Store) GetEventsByCorrelationGroup(ctx context.Context, groupID string) ([]WideEvent, error) {
	var events []WideEvent
	err := s.db.SelectContext(ctx, &events,
		"SELECT * FROM wide_events WHERE correlation_group_id = ? ORDER BY timestamp ASC", groupID)
	return events, err
}

func (s *Store) GetStats(ctx context.Context, clusterID string) (*EventStats, error) {
	since24h := time.Now().Add(-24 * time.Hour).UnixMilli()
	stats := &EventStats{}

	s.db.GetContext(ctx, &stats.Total24h,
		"SELECT COUNT(*) FROM wide_events WHERE cluster_id = ? AND timestamp >= ?", clusterID, since24h)
	s.db.GetContext(ctx, &stats.Warnings24h,
		"SELECT COUNT(*) FROM wide_events WHERE cluster_id = ? AND timestamp >= ? AND event_type = 'Warning'", clusterID, since24h)
	s.db.GetContext(ctx, &stats.HealthChanges,
		"SELECT COUNT(*) FROM wide_events WHERE cluster_id = ? AND timestamp >= ? AND health_delta IS NOT NULL AND health_delta != 0", clusterID, since24h)
	s.db.GetContext(ctx, &stats.ActiveIncidents,
		"SELECT COUNT(*) FROM incidents WHERE cluster_id = ? AND status = 'active'", clusterID)

	s.db.SelectContext(ctx, &stats.TopReasons,
		"SELECT event_reason as reason, COUNT(*) as count FROM wide_events WHERE cluster_id = ? AND timestamp >= ? GROUP BY event_reason ORDER BY count DESC LIMIT 5", clusterID, since24h)
	s.db.SelectContext(ctx, &stats.TopNamespaces,
		"SELECT namespace, COUNT(*) as count FROM wide_events WHERE cluster_id = ? AND timestamp >= ? AND namespace != '' GROUP BY namespace ORDER BY count DESC LIMIT 5", clusterID, since24h)

	return stats, nil
}

// --- Changes ---

func (s *Store) InsertChange(ctx context.Context, c *Change) error {
	_, err := s.db.NamedExecContext(ctx, `
		INSERT OR REPLACE INTO changes (
			change_id, timestamp, cluster_id, resource_kind, resource_name,
			resource_uid, namespace, change_type, field_changes, change_source,
			events_caused, health_impact, incident_id, event_id, dimensions
		) VALUES (
			:change_id, :timestamp, :cluster_id, :resource_kind, :resource_name,
			:resource_uid, :namespace, :change_type, :field_changes, :change_source,
			:events_caused, :health_impact, :incident_id, :event_id, :dimensions
		)`, c)
	return err
}

func (s *Store) GetRecentChanges(ctx context.Context, clusterID string, limit int) ([]Change, error) {
	var changes []Change
	err := s.db.SelectContext(ctx, &changes,
		"SELECT * FROM changes WHERE cluster_id = ? ORDER BY timestamp DESC LIMIT ?", clusterID, limit)
	return changes, err
}

func (s *Store) UpdateChangeImpact(ctx context.Context, changeID string, eventsCaused int, healthImpact *float64) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE changes SET events_caused = ?, health_impact = ? WHERE change_id = ?",
		eventsCaused, healthImpact, changeID)
	return err
}

// --- Incidents ---

func (s *Store) InsertIncident(ctx context.Context, inc *Incident) error {
	_, err := s.db.NamedExecContext(ctx, `
		INSERT OR REPLACE INTO incidents (
			incident_id, started_at, ended_at, status, severity, cluster_id,
			namespace, primary_resource, events_count, health_before, health_after,
			health_lowest, root_cause_event_id, root_cause_summary,
			resolution_event_id, ttd_seconds, ttr_seconds, dimensions
		) VALUES (
			:incident_id, :started_at, :ended_at, :status, :severity, :cluster_id,
			:namespace, :primary_resource, :events_count, :health_before, :health_after,
			:health_lowest, :root_cause_event_id, :root_cause_summary,
			:resolution_event_id, :ttd_seconds, :ttr_seconds, :dimensions
		)`, inc)
	return err
}

func (s *Store) LinkEventToIncident(ctx context.Context, incidentID, eventID, role string) error {
	_, err := s.db.ExecContext(ctx,
		"INSERT OR REPLACE INTO incident_events (incident_id, event_id, role) VALUES (?, ?, ?)",
		incidentID, eventID, role)
	return err
}

func (s *Store) GetActiveIncidents(ctx context.Context, clusterID string) ([]Incident, error) {
	var incidents []Incident
	err := s.db.SelectContext(ctx, &incidents,
		"SELECT * FROM incidents WHERE cluster_id = ? AND status = 'active' ORDER BY started_at DESC", clusterID)
	return incidents, err
}

func (s *Store) GetIncident(ctx context.Context, incidentID string) (*Incident, error) {
	var inc Incident
	err := s.db.GetContext(ctx, &inc, "SELECT * FROM incidents WHERE incident_id = ?", incidentID)
	if err != nil {
		return nil, err
	}
	return &inc, nil
}

func (s *Store) GetIncidentEvents(ctx context.Context, incidentID string) ([]WideEvent, error) {
	var events []WideEvent
	err := s.db.SelectContext(ctx, &events, `
		SELECT w.* FROM wide_events w
		JOIN incident_events ie ON w.event_id = ie.event_id
		WHERE ie.incident_id = ?
		ORDER BY w.timestamp ASC`, incidentID)
	return events, err
}

// --- Relationships ---

func (s *Store) InsertRelationship(ctx context.Context, r *EventRelationship) error {
	_, err := s.db.NamedExecContext(ctx, `
		INSERT OR REPLACE INTO event_relationships (
			source_event_id, target_event_id, relationship, confidence, metadata
		) VALUES (
			:source_event_id, :target_event_id, :relationship, :confidence, :metadata
		)`, r)
	return err
}

func (s *Store) GetRelationships(ctx context.Context, eventID string) ([]EventRelationship, error) {
	var rels []EventRelationship
	err := s.db.SelectContext(ctx, &rels, `
		SELECT * FROM event_relationships 
		WHERE source_event_id = ? OR target_event_id = ?
		ORDER BY confidence DESC`, eventID, eventID)
	return rels, err
}

// --- Insights ---

func (s *Store) InsertInsight(ctx context.Context, i *Insight) error {
	_, err := s.db.NamedExecContext(ctx, `
		INSERT OR REPLACE INTO insights (
			insight_id, timestamp, cluster_id, rule, severity, title, detail,
			namespace, status, resolved_at, dimensions
		) VALUES (
			:insight_id, :timestamp, :cluster_id, :rule, :severity, :title, :detail,
			:namespace, :status, :resolved_at, :dimensions
		)`, i)
	return err
}

func (s *Store) GetActiveInsights(ctx context.Context, clusterID string) ([]Insight, error) {
	var insights []Insight
	err := s.db.SelectContext(ctx, &insights,
		"SELECT * FROM insights WHERE cluster_id = ? AND status = 'active' ORDER BY timestamp DESC", clusterID)
	return insights, err
}

func (s *Store) DismissInsight(ctx context.Context, insightID string) error {
	now := time.Now().UnixMilli()
	_, err := s.db.ExecContext(ctx,
		"UPDATE insights SET status = 'dismissed', resolved_at = ? WHERE insight_id = ?", now, insightID)
	return err
}

// --- Snapshots ---

func (s *Store) InsertSnapshot(ctx context.Context, snap *StateSnapshot) error {
	_, err := s.db.NamedExecContext(ctx, `
		INSERT INTO state_snapshots (
			snapshot_id, timestamp, cluster_id, health_score,
			total_pods, running_pods, pending_pods, failed_pods,
			spof_count, warning_count, node_count, nodes_ready,
			namespace_states, deployment_states
		) VALUES (
			:snapshot_id, :timestamp, :cluster_id, :health_score,
			:total_pods, :running_pods, :pending_pods, :failed_pods,
			:spof_count, :warning_count, :node_count, :nodes_ready,
			:namespace_states, :deployment_states
		)`, snap)
	return err
}

func (s *Store) GetSnapshotAt(ctx context.Context, clusterID string, timestamp int64) (*StateSnapshot, error) {
	var snap StateSnapshot
	err := s.db.GetContext(ctx, &snap,
		"SELECT * FROM state_snapshots WHERE cluster_id = ? AND timestamp <= ? ORDER BY timestamp DESC LIMIT 1",
		clusterID, timestamp)
	if err != nil {
		return nil, err
	}
	return &snap, nil
}

// --- Retention ---

func (s *Store) PruneOldEvents(ctx context.Context, retentionDays int) (int64, error) {
	cutoff := time.Now().Add(-time.Duration(retentionDays) * 24 * time.Hour).UnixMilli()
	result, err := s.db.ExecContext(ctx, "DELETE FROM wide_events WHERE timestamp < ?", cutoff)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (s *Store) PruneOldSnapshots(ctx context.Context, retentionDays int) (int64, error) {
	cutoff := time.Now().Add(-time.Duration(retentionDays) * 24 * time.Hour).UnixMilli()
	result, err := s.db.ExecContext(ctx, "DELETE FROM state_snapshots WHERE timestamp < ?", cutoff)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (s *Store) CountEvents(ctx context.Context, clusterID string, reason string, since int64) (int, error) {
	var count int
	if reason != "" {
		err := s.db.GetContext(ctx, &count,
			"SELECT COUNT(*) FROM wide_events WHERE cluster_id = ? AND event_reason = ? AND timestamp >= ?",
			clusterID, reason, since)
		return count, err
	}
	err := s.db.GetContext(ctx, &count,
		"SELECT COUNT(*) FROM wide_events WHERE cluster_id = ? AND event_type = 'Warning' AND timestamp >= ?",
		clusterID, since)
	return count, err
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd kubilitics-backend && go build ./internal/events/`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add kubilitics-backend/internal/events/store.go
git commit -m "feat(events): add SQLite store for wide events, changes, incidents, insights, snapshots"
```

---

## Task 4-15: Remaining Backend Components

Due to the size of this plan, the remaining tasks are structured as headers. Each follows the same pattern: write the file, verify compilation, commit.

### Task 4: Collector — K8s Event Watcher
**File:** `internal/events/collector.go`
- Watch K8s `core/v1 Events` via existing informer infrastructure
- On each event: create a raw `WideEvent` struct with K8s fields populated
- Watch resource spec changes via existing resource informers
- On resource update: compute field-level diff, create `Change` record
- Feed events into the pipeline channel

### Task 5: Enricher — Context Addition
**File:** `internal/events/enricher.go`
- Accept a raw `WideEvent`, look up the resource in the topology/graph engine
- Add: health score, SPOF status, blast radius, namespace risk level
- Add: node info, deployment version, image, replica count
- Add: dynamic impact (walk topology dependents)
- Return the enriched `WideEvent`

### Task 6: Causality Engine — 6 Rules
**File:** `internal/events/causality.go`
- 6 causal inference rules (deployment→pod, OOM→crashloop, node→eviction, config→restart, scaledown→SPOF, quota→scheduling)
- Each rule: query the store for a recent upstream event matching the pattern
- Set `caused_by_event_id`, `causal_confidence`, `causal_rule`
- Assign `correlation_group_id` (inherit from parent or generate new)

### Task 7: Change Intelligence
**File:** `internal/events/changes.go`
- Detect change type from resource kind + diff
- Classify: rollout, config_update, scale, image_update, policy_change
- Store `Change` record with field-level diffs
- Background goroutine: 10 min after each change, count downstream warnings and update `events_caused` + `health_impact`

### Task 8: Incident Detection
**File:** `internal/events/incidents.go`
- Detect incident start: Warning with health_delta < -10, or 5+ warnings in 2 min, or node NotReady
- Group events: same correlation group, same namespace in time window, causally linked
- Detect incident end: no warnings for 10 min, or health recovered
- Generate root cause summary from causal chain
- Calculate TTD and TTR

### Task 9: Relationship Builder
**File:** `internal/events/relationships.go`
- After each event is stored, create relationship records
- Types: caused_by (from causality), follows (same resource), resolves (Warning→Normal), co_occurs (statistical)

### Task 10: Proactive Insights
**File:** `internal/events/insights.go`
- 6 anomaly detection rules running every 60 seconds
- Compare recent rates against 24h baseline
- Create/resolve `Insight` records
- Rules: oom_spike, restart_storm, scheduling_failures, image_pull_failures, cascading_failures, health_drift

### Task 11: State Snapshots
**File:** `internal/events/snapshots.go`
- Every 5 minutes: capture cluster state (pods, nodes, health, SPOFs)
- Store as `StateSnapshot`
- Provide reconstruction: nearest snapshot + event replay

### Task 12: Pipeline Orchestrator
**File:** `internal/events/pipeline.go`
- Ties everything together: Collector → Enricher → Causality → Relationships → Incidents → Store
- Manages goroutine lifecycle: collector, insights runner, snapshot ticker, retention pruner
- SSE broadcast channel for live events

### Task 13: REST API Handlers
**File:** `internal/events/api.go`
- Handlers for all endpoints: events/stream, events/query, events/analyze, events/:id, events/:id/chain, events/stats, changes/recent, incidents, insights/active, state/at
- Follow existing pattern: `func (h *EventsHandler) GetEvents(w http.ResponseWriter, r *http.Request)`

### Task 14: SSE Stream
**File:** `internal/events/sse.go`
- Server-Sent Events endpoint for live event streaming
- Client registration/deregistration
- Filter by query params (namespace, kind, type)

### Task 15: Wire Into Main Server
**Files:** `cmd/server/main.go`, `internal/api/rest/handler.go`
- Initialize `events.NewPipeline(db, k8sClient, graphEngine)` in main.go
- Start pipeline with server lifecycle
- Register `/api/v1/clusters/{clusterId}/events-intelligence/...` routes

---

## Implementation Order

```
Task 1  → Migration (schema)
Task 2  → Types (data structures)
Task 3  → Store (CRUD)
Task 4  → Collector (K8s event watcher)
Task 5  → Enricher (context addition)
Task 6  → Causality (6 rules)
Task 7  → Changes (diff detection)
Task 12 → Pipeline (orchestrator) — wire collector + enricher + causality + store
Task 15 → Wire into main server
Task 13 → REST API handlers
Task 14 → SSE stream
Task 8  → Incidents (detection + grouping)
Task 9  → Relationships (multi-type links)
Task 10 → Insights (anomaly detection)
Task 11 → Snapshots (time-travel)
```

Tasks 1-7 + 12-15 produce a working event collection + query system.
Tasks 8-11 add intelligence features on top.

---

## Verification

After all tasks:
1. Start the backend: `cd kubilitics-backend && go run ./cmd/server/`
2. Connect a cluster via the frontend
3. Verify events appear: `curl http://localhost:8190/api/v1/clusters/{id}/events-intelligence/stats`
4. Verify live stream: `curl -N http://localhost:8190/api/v1/clusters/{id}/events-intelligence/stream`
5. Verify timeline: `curl "http://localhost:8190/api/v1/clusters/{id}/events-intelligence/query?limit=10"`
