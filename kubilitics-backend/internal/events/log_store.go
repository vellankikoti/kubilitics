package events

import (
	"context"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
)

// StoredLog represents a single persisted log line with parsed metadata.
type StoredLog struct {
	LogID         string   `db:"log_id"         json:"log_id"`
	Timestamp     int64    `db:"timestamp"       json:"timestamp"`
	ClusterID     string   `db:"cluster_id"      json:"cluster_id"`
	Namespace     string   `db:"namespace"        json:"namespace"`
	PodName       string   `db:"pod_name"        json:"pod_name"`
	ContainerName string   `db:"container_name"  json:"container_name"`
	Level         string   `db:"level"           json:"level"`
	Message       string   `db:"message"         json:"message"`
	RawLine       string   `db:"raw_line"        json:"raw_line"`
	IsStructured  bool     `db:"is_structured"   json:"is_structured"`
	Fields        JSONText `db:"fields"          json:"fields"`
	OwnerKind     string   `db:"owner_kind"      json:"owner_kind"`
	OwnerName     string   `db:"owner_name"      json:"owner_name"`
}

// LogQuery defines filters for querying stored logs.
type LogQuery struct {
	ClusterID  string
	Namespace  string
	PodName    string // specific pod
	OwnerKind  string // for cross-pod: "Deployment"
	OwnerName  string // for cross-pod: "checkout-api"
	Level      string // ERROR, WARN, INFO, DEBUG
	Search     string // full-text search in message
	FieldQuery string // key=value search in fields JSON
	From       int64
	To         int64
	Limit      int
	Offset     int
}

// InsertLogs batch-inserts log lines using INSERT OR IGNORE for dedup.
func (s *Store) InsertLogs(ctx context.Context, logs []StoredLog) error {
	if len(logs) == 0 {
		return nil
	}

	const q = `
		INSERT OR IGNORE INTO stored_logs (
			log_id, timestamp, cluster_id, namespace, pod_name, container_name,
			level, message, raw_line, is_structured, fields,
			owner_kind, owner_name
		) VALUES (
			:log_id, :timestamp, :cluster_id, :namespace, :pod_name, :container_name,
			:level, :message, :raw_line, :is_structured, :fields,
			:owner_kind, :owner_name
		)`

	// Batch in groups of 100 to stay within SQLite variable limits.
	batchSize := 100
	for i := 0; i < len(logs); i += batchSize {
		end := i + batchSize
		if end > len(logs) {
			end = len(logs)
		}
		batch := logs[i:end]

		tx, err := s.db.BeginTxx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin tx for log batch: %w", err)
		}

		for _, l := range batch {
			if _, err := tx.NamedExecContext(ctx, q, l); err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("insert log %s: %w", l.LogID, err)
			}
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit log batch: %w", err)
		}
	}

	return nil
}

// QueryLogs returns logs matching the given query filters with dynamic WHERE.
func (s *Store) QueryLogs(ctx context.Context, q LogQuery) ([]StoredLog, error) {
	where, args := buildLogWhere(q)

	limit := q.Limit
	if limit <= 0 {
		limit = 200
	}

	query := `SELECT * FROM stored_logs` + where + ` ORDER BY timestamp DESC LIMIT ? OFFSET ?`
	args = append(args, limit, q.Offset)

	var logs []StoredLog
	if err := s.db.SelectContext(ctx, &logs, query, args...); err != nil {
		return nil, fmt.Errorf("query logs: %w", err)
	}
	return logs, nil
}

// PruneLogs deletes logs older than retentionDays. Returns the number of rows deleted.
func (s *Store) PruneLogs(ctx context.Context, retentionDays int) (int64, error) {
	cutoff := UnixMillis() - int64(retentionDays)*24*60*60*1000
	res, err := s.db.ExecContext(ctx, `DELETE FROM stored_logs WHERE timestamp < ?`, cutoff)
	if err != nil {
		return 0, fmt.Errorf("prune logs: %w", err)
	}
	return res.RowsAffected()
}

// buildLogWhere constructs a dynamic WHERE clause from a LogQuery.
func buildLogWhere(q LogQuery) (string, []interface{}) {
	var clauses []string
	var args []interface{}

	if q.ClusterID != "" {
		clauses = append(clauses, "cluster_id = ?")
		args = append(args, q.ClusterID)
	}
	if q.Namespace != "" {
		clauses = append(clauses, "namespace = ?")
		args = append(args, q.Namespace)
	}
	if q.PodName != "" {
		clauses = append(clauses, "pod_name = ?")
		args = append(args, q.PodName)
	}
	if q.OwnerKind != "" {
		clauses = append(clauses, "owner_kind = ?")
		args = append(args, q.OwnerKind)
	}
	if q.OwnerName != "" {
		clauses = append(clauses, "owner_name = ?")
		args = append(args, q.OwnerName)
	}
	if q.Level != "" {
		clauses = append(clauses, "level = ?")
		args = append(args, q.Level)
	}
	if q.Search != "" {
		clauses = append(clauses, "message LIKE ?")
		args = append(args, "%"+q.Search+"%")
	}
	if q.FieldQuery != "" {
		// Support key=value search in the JSON fields column.
		parts := strings.SplitN(q.FieldQuery, "=", 2)
		if len(parts) == 2 {
			// Use SQLite json_extract for field queries.
			clauses = append(clauses, "json_extract(fields, ?) = ?")
			args = append(args, "$."+parts[0], parts[1])
		}
	}
	if q.From > 0 {
		clauses = append(clauses, "timestamp >= ?")
		args = append(args, q.From)
	}
	if q.To > 0 {
		clauses = append(clauses, "timestamp <= ?")
		args = append(args, q.To)
	}

	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

// Ensure sqlx is used (prevent import removal during refactors).
var _ *sqlx.DB
