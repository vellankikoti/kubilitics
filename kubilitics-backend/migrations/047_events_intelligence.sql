-- Events Intelligence: wide event collection, causality, incidents, insights
-- =============================================================================

-- Wide Events — the core denormalized event table
CREATE TABLE IF NOT EXISTS wide_events (
    event_id              TEXT PRIMARY KEY,
    timestamp             INTEGER NOT NULL,              -- unix ms
    cluster_id            TEXT NOT NULL,

    -- Event fields
    event_type            TEXT NOT NULL DEFAULT '',       -- Normal, Warning
    reason                TEXT NOT NULL DEFAULT '',       -- K8s reason string
    message               TEXT NOT NULL DEFAULT '',
    source_component      TEXT NOT NULL DEFAULT '',       -- kubelet, scheduler, etc.
    source_host           TEXT NOT NULL DEFAULT '',
    event_count           INTEGER NOT NULL DEFAULT 1,
    first_seen            INTEGER NOT NULL DEFAULT 0,    -- unix ms
    last_seen             INTEGER NOT NULL DEFAULT 0,    -- unix ms

    -- Resource fields
    resource_kind         TEXT NOT NULL DEFAULT '',
    resource_name         TEXT NOT NULL DEFAULT '',
    resource_namespace    TEXT NOT NULL DEFAULT '',
    resource_uid          TEXT NOT NULL DEFAULT '',
    resource_api_version  TEXT NOT NULL DEFAULT '',
    owner_kind            TEXT NOT NULL DEFAULT '',
    owner_name            TEXT NOT NULL DEFAULT '',

    -- K8s context
    node_name             TEXT NOT NULL DEFAULT '',

    -- Kubilitics enrichment
    health_score          REAL,                          -- nullable: resource health at event time
    is_spof               INTEGER NOT NULL DEFAULT 0,    -- boolean
    blast_radius          INTEGER NOT NULL DEFAULT 0,    -- number of affected resources
    severity              TEXT NOT NULL DEFAULT 'info',  -- info, low, medium, high, critical

    -- Causality
    caused_by_event_id    TEXT,                          -- FK to another event
    correlation_group_id  TEXT NOT NULL DEFAULT '',       -- groups related events

    -- Extensible dimensions (JSON)
    dimensions            TEXT NOT NULL DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS idx_wide_events_cluster_time     ON wide_events(cluster_id, timestamp);
CREATE INDEX IF NOT EXISTS idx_wide_events_resource         ON wide_events(resource_kind, resource_name, resource_namespace);
CREATE INDEX IF NOT EXISTS idx_wide_events_type_reason      ON wide_events(event_type, reason);
CREATE INDEX IF NOT EXISTS idx_wide_events_severity         ON wide_events(severity, timestamp);
CREATE INDEX IF NOT EXISTS idx_wide_events_correlation      ON wide_events(correlation_group_id);
CREATE INDEX IF NOT EXISTS idx_wide_events_node             ON wide_events(node_name, timestamp);
CREATE INDEX IF NOT EXISTS idx_wide_events_namespace_time   ON wide_events(resource_namespace, timestamp);
CREATE INDEX IF NOT EXISTS idx_wide_events_caused_by        ON wide_events(caused_by_event_id);

-- Changes — tracks field-level resource mutations
CREATE TABLE IF NOT EXISTS changes (
    change_id        TEXT PRIMARY KEY,
    timestamp        INTEGER NOT NULL,                   -- unix ms
    cluster_id       TEXT NOT NULL,
    resource_kind    TEXT NOT NULL DEFAULT '',
    resource_name    TEXT NOT NULL DEFAULT '',
    resource_namespace TEXT NOT NULL DEFAULT '',
    resource_uid     TEXT NOT NULL DEFAULT '',

    change_type      TEXT NOT NULL DEFAULT '',            -- created, updated, deleted, scaled, restarted
    field_changes    TEXT NOT NULL DEFAULT '[]',          -- JSON array of FieldChange
    change_source    TEXT NOT NULL DEFAULT '',            -- user, controller, hpa, etc.

    events_caused    INTEGER NOT NULL DEFAULT 0,
    health_impact    REAL,                               -- delta health score

    event_id         TEXT,                               -- FK to wide_events
    FOREIGN KEY (event_id) REFERENCES wide_events(event_id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_changes_cluster_time       ON changes(cluster_id, timestamp);
CREATE INDEX IF NOT EXISTS idx_changes_resource           ON changes(resource_kind, resource_name, resource_namespace);
CREATE INDEX IF NOT EXISTS idx_changes_event              ON changes(event_id);
CREATE INDEX IF NOT EXISTS idx_changes_type               ON changes(change_type, timestamp);

-- Event Relationships — causal / temporal / topological links between events
CREATE TABLE IF NOT EXISTS event_relationships (
    source_event_id    TEXT NOT NULL,
    target_event_id    TEXT NOT NULL,
    relationship_type  TEXT NOT NULL DEFAULT '',          -- caused, correlated, followed_by, same_root
    confidence         REAL NOT NULL DEFAULT 1.0,        -- 0.0–1.0
    metadata           TEXT NOT NULL DEFAULT '{}',       -- JSON
    PRIMARY KEY (source_event_id, target_event_id),
    FOREIGN KEY (source_event_id) REFERENCES wide_events(event_id) ON DELETE CASCADE,
    FOREIGN KEY (target_event_id) REFERENCES wide_events(event_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_event_rel_target ON event_relationships(target_event_id);
CREATE INDEX IF NOT EXISTS idx_event_rel_type   ON event_relationships(relationship_type);

-- Incidents — groups of correlated events forming an operational incident
CREATE TABLE IF NOT EXISTS incidents (
    incident_id        TEXT PRIMARY KEY,
    started_at         INTEGER NOT NULL,                 -- unix ms
    ended_at           INTEGER,                          -- unix ms, null if ongoing
    status             TEXT NOT NULL DEFAULT 'active',   -- active, mitigated, resolved
    severity           TEXT NOT NULL DEFAULT 'medium',   -- low, medium, high, critical

    cluster_id         TEXT NOT NULL,
    namespace          TEXT NOT NULL DEFAULT '',

    health_before      REAL,
    health_after       REAL,
    health_lowest      REAL,

    root_cause_kind    TEXT NOT NULL DEFAULT '',
    root_cause_name    TEXT NOT NULL DEFAULT '',
    root_cause_summary TEXT NOT NULL DEFAULT '',

    ttd                INTEGER,                          -- time to detect (ms)
    ttr                INTEGER,                          -- time to resolve (ms)

    dimensions         TEXT NOT NULL DEFAULT '{}'        -- JSON
);

CREATE INDEX IF NOT EXISTS idx_incidents_cluster_status   ON incidents(cluster_id, status);
CREATE INDEX IF NOT EXISTS idx_incidents_severity         ON incidents(severity, started_at);
CREATE INDEX IF NOT EXISTS idx_incidents_started          ON incidents(started_at);

-- Incident Events — junction table linking events to incidents
CREATE TABLE IF NOT EXISTS incident_events (
    incident_id TEXT NOT NULL,
    event_id    TEXT NOT NULL,
    role        TEXT NOT NULL DEFAULT 'member',          -- root_cause, symptom, member, resolution
    PRIMARY KEY (incident_id, event_id),
    FOREIGN KEY (incident_id) REFERENCES incidents(incident_id) ON DELETE CASCADE,
    FOREIGN KEY (event_id)    REFERENCES wide_events(event_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_incident_events_event ON incident_events(event_id);

-- Insights — automated observations and recommendations
CREATE TABLE IF NOT EXISTS insights (
    insight_id  TEXT PRIMARY KEY,
    timestamp   INTEGER NOT NULL,                        -- unix ms
    cluster_id  TEXT NOT NULL,
    rule        TEXT NOT NULL DEFAULT '',                 -- rule that generated this
    severity    TEXT NOT NULL DEFAULT 'info',             -- info, low, medium, high, critical
    title       TEXT NOT NULL DEFAULT '',
    detail      TEXT NOT NULL DEFAULT '',
    status      TEXT NOT NULL DEFAULT 'active'            -- active, dismissed, resolved
);

CREATE INDEX IF NOT EXISTS idx_insights_cluster_status ON insights(cluster_id, status);
CREATE INDEX IF NOT EXISTS idx_insights_severity       ON insights(severity, timestamp);

-- State Snapshots — periodic aggregate cluster state captures
CREATE TABLE IF NOT EXISTS state_snapshots (
    snapshot_id        TEXT PRIMARY KEY,
    timestamp          INTEGER NOT NULL,                 -- unix ms
    cluster_id         TEXT NOT NULL,

    total_pods         INTEGER NOT NULL DEFAULT 0,
    running_pods       INTEGER NOT NULL DEFAULT 0,
    total_nodes        INTEGER NOT NULL DEFAULT 0,
    ready_nodes        INTEGER NOT NULL DEFAULT 0,
    health_score       REAL NOT NULL DEFAULT 0,
    spof_count         INTEGER NOT NULL DEFAULT 0,
    warning_events     INTEGER NOT NULL DEFAULT 0,
    error_events       INTEGER NOT NULL DEFAULT 0,

    namespace_states   TEXT NOT NULL DEFAULT '[]',       -- JSON
    deployment_states  TEXT NOT NULL DEFAULT '[]'        -- JSON
);

CREATE INDEX IF NOT EXISTS idx_state_snapshots_cluster_time ON state_snapshots(cluster_id, timestamp);
