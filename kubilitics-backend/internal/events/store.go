package events

import (
	"context"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
)

// Store provides persistence operations for the Events Intelligence subsystem.
// It wraps a *sqlx.DB connected to the application's SQLite database.
type Store struct {
	db *sqlx.DB
}

// NewStore creates a new events Store.
func NewStore(db *sqlx.DB) *Store {
	return &Store{db: db}
}

// ---------------------------------------------------------------------------
// Wide Events
// ---------------------------------------------------------------------------

// InsertEvent inserts or replaces a wide event.
func (s *Store) InsertEvent(ctx context.Context, e *WideEvent) error {
	const q = `
		INSERT OR REPLACE INTO wide_events (
			event_id, timestamp, cluster_id,
			event_type, reason, message, source_component, source_host,
			event_count, first_seen, last_seen,
			resource_kind, resource_name, resource_namespace, resource_uid, resource_api_version,
			owner_kind, owner_name, node_name,
			health_score, is_spof, blast_radius, severity,
			caused_by_event_id, correlation_group_id, dimensions
		) VALUES (
			:event_id, :timestamp, :cluster_id,
			:event_type, :reason, :message, :source_component, :source_host,
			:event_count, :first_seen, :last_seen,
			:resource_kind, :resource_name, :resource_namespace, :resource_uid, :resource_api_version,
			:owner_kind, :owner_name, :node_name,
			:health_score, :is_spof, :blast_radius, :severity,
			:caused_by_event_id, :correlation_group_id, :dimensions
		)`
	_, err := s.db.NamedExecContext(ctx, q, e)
	if err != nil {
		return fmt.Errorf("insert event %s: %w", e.EventID, err)
	}
	return nil
}

// QueryEvents returns events matching the given query filters.
// Filters are applied dynamically via WHERE clause construction.
func (s *Store) QueryEvents(ctx context.Context, q EventQuery) ([]WideEvent, error) {
	where, params := buildEventWhere(q)

	limit := q.Limit
	if limit <= 0 {
		limit = 200
	}
	params["limit"] = limit
	params["offset"] = q.Offset

	query := `SELECT * FROM wide_events` + where + ` ORDER BY timestamp DESC LIMIT :limit OFFSET :offset`

	query, args, err := sqlx.Named(query, params)
	if err != nil {
		return nil, fmt.Errorf("build named query: %w", err)
	}
	query = s.db.Rebind(query)

	var events []WideEvent
	if err := s.db.SelectContext(ctx, &events, query, args...); err != nil {
		return nil, fmt.Errorf("query events: %w", err)
	}
	return events, nil
}

// GetEvent retrieves a single event by ID.
func (s *Store) GetEvent(ctx context.Context, eventID string) (*WideEvent, error) {
	var e WideEvent
	err := s.db.GetContext(ctx, &e, `SELECT * FROM wide_events WHERE event_id = ?`, eventID)
	if err != nil {
		return nil, fmt.Errorf("get event %s: %w", eventID, err)
	}
	return &e, nil
}

// FindRecentEvent looks up the most recent event matching the given resource
// and reason within a cluster.
func (s *Store) FindRecentEvent(ctx context.Context, clusterID, resourceKind, resourceName, namespace, reason string) (*WideEvent, error) {
	const q = `
		SELECT * FROM wide_events
		WHERE cluster_id = ? AND resource_kind = ? AND resource_name = ?
		  AND resource_namespace = ? AND reason = ?
		ORDER BY timestamp DESC
		LIMIT 1`
	var e WideEvent
	if err := s.db.GetContext(ctx, &e, q, clusterID, resourceKind, resourceName, namespace, reason); err != nil {
		return nil, fmt.Errorf("find recent event: %w", err)
	}
	return &e, nil
}

// GetEventsByCorrelationGroup returns all events in a correlation group.
func (s *Store) GetEventsByCorrelationGroup(ctx context.Context, groupID string) ([]WideEvent, error) {
	var events []WideEvent
	err := s.db.SelectContext(ctx, &events,
		`SELECT * FROM wide_events WHERE correlation_group_id = ? ORDER BY timestamp ASC`, groupID)
	if err != nil {
		return nil, fmt.Errorf("get events by correlation group %s: %w", groupID, err)
	}
	return events, nil
}

// GetStats returns aggregate event statistics for a cluster within a time range.
func (s *Store) GetStats(ctx context.Context, clusterID string, since, until int64) (*EventStats, error) {
	stats := &EventStats{
		ByType:     make(map[string]int64),
		BySeverity: make(map[string]int64),
		ByReason:   make(map[string]int64),
		Since:      since,
		Until:      until,
	}

	const baseWhere = ` WHERE cluster_id = ? AND timestamp >= ? AND timestamp <= ?`

	// Total count
	err := s.db.GetContext(ctx, &stats.TotalEvents,
		`SELECT COUNT(*) FROM wide_events`+baseWhere, clusterID, since, until)
	if err != nil {
		return nil, fmt.Errorf("count events: %w", err)
	}

	// Group by type
	type kv struct {
		Key   string `db:"k"`
		Count int64  `db:"c"`
	}
	var byType []kv
	err = s.db.SelectContext(ctx, &byType,
		`SELECT event_type AS k, COUNT(*) AS c FROM wide_events`+baseWhere+` GROUP BY event_type`,
		clusterID, since, until)
	if err != nil {
		return nil, fmt.Errorf("group by type: %w", err)
	}
	for _, r := range byType {
		stats.ByType[r.Key] = r.Count
	}

	// Group by severity
	var bySev []kv
	err = s.db.SelectContext(ctx, &bySev,
		`SELECT severity AS k, COUNT(*) AS c FROM wide_events`+baseWhere+` GROUP BY severity`,
		clusterID, since, until)
	if err != nil {
		return nil, fmt.Errorf("group by severity: %w", err)
	}
	for _, r := range bySev {
		stats.BySeverity[r.Key] = r.Count
	}

	// Group by reason (top 20)
	var byReason []kv
	err = s.db.SelectContext(ctx, &byReason,
		`SELECT reason AS k, COUNT(*) AS c FROM wide_events`+baseWhere+` GROUP BY reason ORDER BY c DESC LIMIT 20`,
		clusterID, since, until)
	if err != nil {
		return nil, fmt.Errorf("group by reason: %w", err)
	}
	for _, r := range byReason {
		stats.ByReason[r.Key] = r.Count
	}

	return stats, nil
}

// ---------------------------------------------------------------------------
// Changes
// ---------------------------------------------------------------------------

// InsertChange inserts or replaces a change record.
func (s *Store) InsertChange(ctx context.Context, c *Change) error {
	const q = `
		INSERT OR REPLACE INTO changes (
			change_id, timestamp, cluster_id,
			resource_kind, resource_name, resource_namespace, resource_uid,
			change_type, field_changes, change_source,
			events_caused, health_impact, event_id
		) VALUES (
			:change_id, :timestamp, :cluster_id,
			:resource_kind, :resource_name, :resource_namespace, :resource_uid,
			:change_type, :field_changes, :change_source,
			:events_caused, :health_impact, :event_id
		)`
	_, err := s.db.NamedExecContext(ctx, q, c)
	if err != nil {
		return fmt.Errorf("insert change %s: %w", c.ChangeID, err)
	}
	return nil
}

// GetRecentChanges returns the most recent changes for a cluster.
func (s *Store) GetRecentChanges(ctx context.Context, clusterID string, limit int) ([]Change, error) {
	if limit <= 0 {
		limit = 50
	}
	var changes []Change
	err := s.db.SelectContext(ctx, &changes,
		`SELECT * FROM changes WHERE cluster_id = ? ORDER BY timestamp DESC LIMIT ?`,
		clusterID, limit)
	if err != nil {
		return nil, fmt.Errorf("get recent changes: %w", err)
	}
	return changes, nil
}

// GetRecentChangesForResource returns recent changes for a specific resource,
// filtered by cluster, namespace, kind, and optionally resource name.
func (s *Store) GetRecentChangesForResource(ctx context.Context, clusterID, namespace, kind, name string, limit int) ([]Change, error) {
	if limit <= 0 {
		limit = 5
	}

	var clauses []string
	var args []interface{}

	clauses = append(clauses, "cluster_id = ?")
	args = append(args, clusterID)

	if namespace != "" {
		clauses = append(clauses, "resource_namespace = ?")
		args = append(args, namespace)
	}
	if kind != "" {
		clauses = append(clauses, "resource_kind = ?")
		args = append(args, kind)
	}
	if name != "" {
		clauses = append(clauses, "resource_name = ?")
		args = append(args, name)
	}

	query := `SELECT * FROM changes WHERE ` + strings.Join(clauses, " AND ") + ` ORDER BY timestamp DESC LIMIT ?`
	args = append(args, limit)

	var changes []Change
	if err := s.db.SelectContext(ctx, &changes, query, args...); err != nil {
		return nil, fmt.Errorf("get recent changes for resource: %w", err)
	}
	return changes, nil
}

// GetIncidentForEvent returns the incident (if any) linked to the given event.
func (s *Store) GetIncidentForEvent(ctx context.Context, eventID string) (*Incident, error) {
	const q = `
		SELECT i.* FROM incidents i
		INNER JOIN incident_events ie ON ie.incident_id = i.incident_id
		WHERE ie.event_id = ?
		LIMIT 1`
	var inc Incident
	if err := s.db.GetContext(ctx, &inc, q, eventID); err != nil {
		return nil, fmt.Errorf("get incident for event %s: %w", eventID, err)
	}
	return &inc, nil
}

// UpdateChangeImpact updates the events_caused and health_impact for a change.
func (s *Store) UpdateChangeImpact(ctx context.Context, changeID string, eventsCaused int, healthImpact *float64) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE changes SET events_caused = ?, health_impact = ? WHERE change_id = ?`,
		eventsCaused, healthImpact, changeID)
	if err != nil {
		return fmt.Errorf("update change impact %s: %w", changeID, err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Incidents
// ---------------------------------------------------------------------------

// InsertIncident inserts or replaces an incident.
func (s *Store) InsertIncident(ctx context.Context, inc *Incident) error {
	const q = `
		INSERT OR REPLACE INTO incidents (
			incident_id, started_at, ended_at, status, severity,
			cluster_id, namespace,
			health_before, health_after, health_lowest,
			root_cause_kind, root_cause_name, root_cause_summary,
			ttd, ttr, dimensions
		) VALUES (
			:incident_id, :started_at, :ended_at, :status, :severity,
			:cluster_id, :namespace,
			:health_before, :health_after, :health_lowest,
			:root_cause_kind, :root_cause_name, :root_cause_summary,
			:ttd, :ttr, :dimensions
		)`
	_, err := s.db.NamedExecContext(ctx, q, inc)
	if err != nil {
		return fmt.Errorf("insert incident %s: %w", inc.IncidentID, err)
	}
	return nil
}

// LinkEventToIncident associates an event with an incident.
func (s *Store) LinkEventToIncident(ctx context.Context, ie *IncidentEvent) error {
	const q = `
		INSERT OR REPLACE INTO incident_events (incident_id, event_id, role)
		VALUES (:incident_id, :event_id, :role)`
	_, err := s.db.NamedExecContext(ctx, q, ie)
	if err != nil {
		return fmt.Errorf("link event %s to incident %s: %w", ie.EventID, ie.IncidentID, err)
	}
	return nil
}

// GetActiveIncidents returns all active (non-resolved) incidents for a cluster.
func (s *Store) GetActiveIncidents(ctx context.Context, clusterID string) ([]Incident, error) {
	var incidents []Incident
	err := s.db.SelectContext(ctx, &incidents,
		`SELECT * FROM incidents WHERE cluster_id = ? AND status != 'resolved' ORDER BY started_at DESC`,
		clusterID)
	if err != nil {
		return nil, fmt.Errorf("get active incidents: %w", err)
	}
	return incidents, nil
}

// GetIncident retrieves a single incident by ID.
func (s *Store) GetIncident(ctx context.Context, incidentID string) (*Incident, error) {
	var inc Incident
	err := s.db.GetContext(ctx, &inc, `SELECT * FROM incidents WHERE incident_id = ?`, incidentID)
	if err != nil {
		return nil, fmt.Errorf("get incident %s: %w", incidentID, err)
	}
	return &inc, nil
}

// GetIncidentEvents returns all events linked to an incident.
func (s *Store) GetIncidentEvents(ctx context.Context, incidentID string) ([]WideEvent, error) {
	const q = `
		SELECT e.* FROM wide_events e
		INNER JOIN incident_events ie ON ie.event_id = e.event_id
		WHERE ie.incident_id = ?
		ORDER BY e.timestamp ASC`
	var events []WideEvent
	err := s.db.SelectContext(ctx, &events, q, incidentID)
	if err != nil {
		return nil, fmt.Errorf("get incident events %s: %w", incidentID, err)
	}
	return events, nil
}

// ---------------------------------------------------------------------------
// Relationships
// ---------------------------------------------------------------------------

// InsertRelationship inserts or replaces an event relationship.
func (s *Store) InsertRelationship(ctx context.Context, rel *EventRelationship) error {
	const q = `
		INSERT OR REPLACE INTO event_relationships (
			source_event_id, target_event_id, relationship_type, confidence, metadata
		) VALUES (
			:source_event_id, :target_event_id, :relationship_type, :confidence, :metadata
		)`
	_, err := s.db.NamedExecContext(ctx, q, rel)
	if err != nil {
		return fmt.Errorf("insert relationship %s->%s: %w", rel.SourceEventID, rel.TargetEventID, err)
	}
	return nil
}

// GetRelationships returns all relationships where the given event is either
// the source or the target.
func (s *Store) GetRelationships(ctx context.Context, eventID string) ([]EventRelationship, error) {
	var rels []EventRelationship
	err := s.db.SelectContext(ctx, &rels,
		`SELECT * FROM event_relationships WHERE source_event_id = ? OR target_event_id = ?`,
		eventID, eventID)
	if err != nil {
		return nil, fmt.Errorf("get relationships for %s: %w", eventID, err)
	}
	return rels, nil
}

// ---------------------------------------------------------------------------
// Insights
// ---------------------------------------------------------------------------

// InsertInsight inserts or replaces an insight.
func (s *Store) InsertInsight(ctx context.Context, ins *Insight) error {
	const q = `
		INSERT OR REPLACE INTO insights (
			insight_id, timestamp, cluster_id, rule, severity, title, detail, status
		) VALUES (
			:insight_id, :timestamp, :cluster_id, :rule, :severity, :title, :detail, :status
		)`
	_, err := s.db.NamedExecContext(ctx, q, ins)
	if err != nil {
		return fmt.Errorf("insert insight %s: %w", ins.InsightID, err)
	}
	return nil
}

// GetActiveInsights returns all non-dismissed insights for a cluster.
func (s *Store) GetActiveInsights(ctx context.Context, clusterID string) ([]Insight, error) {
	var insights []Insight
	err := s.db.SelectContext(ctx, &insights,
		`SELECT * FROM insights WHERE cluster_id = ? AND status = 'active' ORDER BY timestamp DESC`,
		clusterID)
	if err != nil {
		return nil, fmt.Errorf("get active insights: %w", err)
	}
	return insights, nil
}

// DismissInsight marks an insight as dismissed.
func (s *Store) DismissInsight(ctx context.Context, insightID string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE insights SET status = 'dismissed' WHERE insight_id = ?`, insightID)
	if err != nil {
		return fmt.Errorf("dismiss insight %s: %w", insightID, err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// State Snapshots
// ---------------------------------------------------------------------------

// InsertSnapshot inserts or replaces a state snapshot.
func (s *Store) InsertSnapshot(ctx context.Context, snap *StateSnapshot) error {
	const q = `
		INSERT OR REPLACE INTO state_snapshots (
			snapshot_id, timestamp, cluster_id,
			total_pods, running_pods, total_nodes, ready_nodes,
			health_score, spof_count, warning_events, error_events,
			namespace_states, deployment_states
		) VALUES (
			:snapshot_id, :timestamp, :cluster_id,
			:total_pods, :running_pods, :total_nodes, :ready_nodes,
			:health_score, :spof_count, :warning_events, :error_events,
			:namespace_states, :deployment_states
		)`
	_, err := s.db.NamedExecContext(ctx, q, snap)
	if err != nil {
		return fmt.Errorf("insert snapshot %s: %w", snap.SnapshotID, err)
	}
	return nil
}

// GetSnapshotAt returns the closest snapshot to the given timestamp for a cluster.
func (s *Store) GetSnapshotAt(ctx context.Context, clusterID string, ts int64) (*StateSnapshot, error) {
	const q = `
		SELECT * FROM state_snapshots
		WHERE cluster_id = ? AND timestamp <= ?
		ORDER BY timestamp DESC
		LIMIT 1`
	var snap StateSnapshot
	if err := s.db.GetContext(ctx, &snap, q, clusterID, ts); err != nil {
		return nil, fmt.Errorf("get snapshot at %d: %w", ts, err)
	}
	return &snap, nil
}

// ---------------------------------------------------------------------------
// DB Size
// ---------------------------------------------------------------------------

// GetDBSizeBytes returns the current SQLite database size in bytes using PRAGMA introspection.
func (s *Store) GetDBSizeBytes() (int64, error) {
	var pageCount, pageSize int64
	if err := s.db.Get(&pageCount, "PRAGMA page_count"); err != nil {
		return 0, fmt.Errorf("get page_count: %w", err)
	}
	if err := s.db.Get(&pageSize, "PRAGMA page_size"); err != nil {
		return 0, fmt.Errorf("get page_size: %w", err)
	}
	return pageCount * pageSize, nil
}

// ---------------------------------------------------------------------------
// Maintenance
// ---------------------------------------------------------------------------

// PruneOldEvents deletes events older than the given unix-ms timestamp.
// Returns the number of rows deleted.
func (s *Store) PruneOldEvents(ctx context.Context, olderThanMs int64) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM wide_events WHERE timestamp < ?`, olderThanMs)
	if err != nil {
		return 0, fmt.Errorf("prune old events: %w", err)
	}
	return res.RowsAffected()
}

// PruneOldSnapshots deletes state snapshots older than the given unix-ms timestamp.
// Returns the number of rows deleted.
func (s *Store) PruneOldSnapshots(ctx context.Context, olderThanMs int64) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM state_snapshots WHERE timestamp < ?`, olderThanMs)
	if err != nil {
		return 0, fmt.Errorf("prune old snapshots: %w", err)
	}
	return res.RowsAffected()
}

// CountEvents returns the total number of wide events in the database.
func (s *Store) CountEvents(ctx context.Context) (int64, error) {
	var count int64
	if err := s.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM wide_events`); err != nil {
		return 0, fmt.Errorf("count events: %w", err)
	}
	return count, nil
}

// CountEventsSince returns the number of events for a cluster since the given unix ms timestamp.
func (s *Store) CountEventsSince(ctx context.Context, clusterID string, sinceMs int64) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM wide_events WHERE cluster_id = ? AND timestamp >= ?`
	if err := s.db.GetContext(ctx, &count, query, clusterID, sinceMs); err != nil {
		return 0, fmt.Errorf("count events since: %w", err)
	}
	return count, nil
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// buildEventWhere constructs a dynamic WHERE clause from an EventQuery.
func buildEventWhere(q EventQuery) (string, map[string]interface{}) {
	var clauses []string
	params := make(map[string]interface{})

	if q.ClusterID != "" {
		clauses = append(clauses, "cluster_id = :cluster_id")
		params["cluster_id"] = q.ClusterID
	}
	if q.Namespace != "" {
		clauses = append(clauses, "resource_namespace = :namespace")
		params["namespace"] = q.Namespace
	}
	if q.ResourceKind != "" {
		clauses = append(clauses, "resource_kind = :resource_kind")
		params["resource_kind"] = q.ResourceKind
	}
	if q.ResourceName != "" {
		clauses = append(clauses, "resource_name = :resource_name")
		params["resource_name"] = q.ResourceName
	}
	if q.EventType != "" {
		clauses = append(clauses, "event_type = :event_type")
		params["event_type"] = q.EventType
	}
	if q.Reason != "" {
		clauses = append(clauses, "reason = :reason")
		params["reason"] = q.Reason
	}
	if q.Severity != "" {
		clauses = append(clauses, "severity = :severity")
		params["severity"] = q.Severity
	}
	if q.NodeName != "" {
		clauses = append(clauses, "node_name = :node_name")
		params["node_name"] = q.NodeName
	}
	if q.ResourceUID != "" {
		clauses = append(clauses, "resource_uid = :resource_uid")
		params["resource_uid"] = q.ResourceUID
	}
	if q.Since != nil {
		clauses = append(clauses, "timestamp >= :since")
		params["since"] = *q.Since
	}
	if q.Until != nil {
		clauses = append(clauses, "timestamp <= :until")
		params["until"] = *q.Until
	}

	if len(clauses) == 0 {
		return "", params
	}
	return " WHERE " + strings.Join(clauses, " AND "), params
}
