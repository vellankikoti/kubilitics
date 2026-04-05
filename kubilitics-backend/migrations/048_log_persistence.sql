-- 048_log_persistence.sql
-- Structured log storage for cross-pod search and aggregation.

CREATE TABLE IF NOT EXISTS stored_logs (
    log_id          TEXT PRIMARY KEY,
    timestamp       INTEGER NOT NULL,       -- unix ms
    cluster_id      TEXT NOT NULL,
    namespace       TEXT NOT NULL,
    pod_name        TEXT NOT NULL,
    container_name  TEXT NOT NULL DEFAULT '',

    -- Parsed fields (for indexed search)
    level           TEXT NOT NULL DEFAULT '',    -- ERROR, WARN, INFO, DEBUG
    message         TEXT NOT NULL DEFAULT '',

    -- Full structured log
    raw_line        TEXT NOT NULL,               -- original log line
    is_structured   INTEGER NOT NULL DEFAULT 0,  -- 1 if valid JSON
    fields          TEXT NOT NULL DEFAULT '{}',  -- parsed JSON fields

    -- Ownership for cross-pod queries
    owner_kind      TEXT NOT NULL DEFAULT '',    -- Deployment, StatefulSet, etc.
    owner_name      TEXT NOT NULL DEFAULT ''     -- checkout-api
);

CREATE INDEX IF NOT EXISTS idx_logs_cluster_time ON stored_logs(cluster_id, timestamp);
CREATE INDEX IF NOT EXISTS idx_logs_namespace ON stored_logs(namespace);
CREATE INDEX IF NOT EXISTS idx_logs_pod ON stored_logs(pod_name);
CREATE INDEX IF NOT EXISTS idx_logs_level ON stored_logs(level);
CREATE INDEX IF NOT EXISTS idx_logs_owner ON stored_logs(owner_kind, owner_name);
CREATE INDEX IF NOT EXISTS idx_logs_message ON stored_logs(message);
