CREATE TABLE IF NOT EXISTS spans (
    span_id          TEXT PRIMARY KEY,
    trace_id         TEXT NOT NULL,
    parent_span_id   TEXT NOT NULL DEFAULT '',

    -- Service info
    service_name     TEXT NOT NULL DEFAULT '',
    operation_name   TEXT NOT NULL DEFAULT '',    -- e.g., "GET /api/users", "SELECT users"
    span_kind        TEXT NOT NULL DEFAULT '',    -- client, server, producer, consumer, internal

    -- Timing
    start_time       INTEGER NOT NULL,           -- unix nanoseconds
    end_time         INTEGER NOT NULL,
    duration_ns      INTEGER NOT NULL,           -- end - start

    -- Status
    status_code      TEXT NOT NULL DEFAULT 'OK', -- OK, ERROR, UNSET
    status_message   TEXT NOT NULL DEFAULT '',

    -- HTTP (common instrumentation)
    http_method      TEXT NOT NULL DEFAULT '',
    http_url         TEXT NOT NULL DEFAULT '',
    http_status_code INTEGER,
    http_route       TEXT NOT NULL DEFAULT '',

    -- Database (common instrumentation)
    db_system        TEXT NOT NULL DEFAULT '',    -- postgresql, redis, mysql
    db_statement     TEXT NOT NULL DEFAULT '',

    -- K8s context (from resource attributes)
    k8s_pod_name     TEXT NOT NULL DEFAULT '',
    k8s_namespace    TEXT NOT NULL DEFAULT '',
    k8s_node_name    TEXT NOT NULL DEFAULT '',
    k8s_container    TEXT NOT NULL DEFAULT '',
    k8s_deployment   TEXT NOT NULL DEFAULT '',

    -- User context (from span attributes)
    user_id          TEXT NOT NULL DEFAULT '',

    -- Cluster link
    cluster_id       TEXT NOT NULL DEFAULT '',

    -- All attributes as JSON
    attributes       TEXT NOT NULL DEFAULT '{}',  -- full attribute map
    events           TEXT NOT NULL DEFAULT '[]',  -- span events (logs)

    -- Correlation
    linked_event_ids TEXT NOT NULL DEFAULT '[]'   -- JSON array of correlated wide_event IDs
);

CREATE INDEX IF NOT EXISTS idx_spans_trace ON spans(trace_id);
CREATE INDEX IF NOT EXISTS idx_spans_service ON spans(service_name);
CREATE INDEX IF NOT EXISTS idx_spans_time ON spans(start_time);
CREATE INDEX IF NOT EXISTS idx_spans_cluster ON spans(cluster_id);
CREATE INDEX IF NOT EXISTS idx_spans_pod ON spans(k8s_pod_name);
CREATE INDEX IF NOT EXISTS idx_spans_status ON spans(status_code);
CREATE INDEX IF NOT EXISTS idx_spans_operation ON spans(operation_name);
CREATE INDEX IF NOT EXISTS idx_spans_duration ON spans(duration_ns);
CREATE INDEX IF NOT EXISTS idx_spans_http ON spans(http_method, http_route);
CREATE INDEX IF NOT EXISTS idx_spans_user ON spans(user_id);

-- Trace summary table for fast trace listing
CREATE TABLE IF NOT EXISTS traces (
    trace_id         TEXT PRIMARY KEY,
    root_service     TEXT NOT NULL DEFAULT '',
    root_operation   TEXT NOT NULL DEFAULT '',
    start_time       INTEGER NOT NULL,
    duration_ns      INTEGER NOT NULL,           -- total trace duration
    span_count       INTEGER NOT NULL DEFAULT 1,
    error_count      INTEGER NOT NULL DEFAULT 0,
    service_count    INTEGER NOT NULL DEFAULT 1,
    status           TEXT NOT NULL DEFAULT 'OK',
    cluster_id       TEXT NOT NULL DEFAULT '',
    services         TEXT NOT NULL DEFAULT '[]',  -- JSON array of service names
    updated_at       INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_traces_time ON traces(start_time);
CREATE INDEX IF NOT EXISTS idx_traces_service ON traces(root_service);
CREATE INDEX IF NOT EXISTS idx_traces_status ON traces(status);
CREATE INDEX IF NOT EXISTS idx_traces_duration ON traces(duration_ns);
CREATE INDEX IF NOT EXISTS idx_traces_cluster ON traces(cluster_id);
