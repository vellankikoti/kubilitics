# Events Intelligence — Observability 2.0 + System Intelligence for Kubilitics

## Overview

A new intelligence pillar that transforms Kubernetes operations and application logs into queryable wide events enriched with Kubilitics context, causal reasoning, incident narratives, and proactive insights. Three layers:

1. **Events Intelligence page** — K8s events as infrastructure-level wide events with causality
2. **Enhanced Log Viewer** — Structured JSON app logs with system context integration
3. **System Intelligence Engine** — Causality, incidents, impact analysis, anomaly detection, time-travel

## Why

Kubilitics today tells you **what** the state is (Health Scores, SPOF count, Risk level) but not **what changed, why it changed, and what was affected**. When a health score drops from 87 to 61, users have no way to trace the cause, understand the cascade, or replay the sequence.

Traditional observability shows events. System intelligence **explains** them.

## Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                      Frontend (React)                        │
│  ┌────────────────┐ ┌──────────────┐ ┌────────────────────┐ │
│  │ Events Timeline │ │ Incident     │ │ Enhanced Log       │ │
│  │ + Analyze Mode  │ │ Narrative    │ │ Viewer + System    │ │
│  │ + Time Travel   │ │ View         │ │ Context Markers    │ │
│  └───────┬─────────┘ └──────┬───────┘ └────────┬───────────┘ │
│          │ REST/SSE         │                   │ REST/WS     │
├──────────┼──────────────────┼───────────────────┼────────────┤
│          │        Backend (Go)                  │             │
│  ┌───────▼──────────────────▼───────────────────▼──────────┐ │
│  │                Wide Event Pipeline                       │ │
│  │  K8s Informers → Enrichment → Causality → Storage       │ │
│  │                                                          │ │
│  │  ┌─────────────┐ ┌──────────────┐ ┌──────────────────┐  │ │
│  │  │ Causality    │ │ Incident     │ │ Proactive        │  │ │
│  │  │ Engine       │ │ Detector     │ │ Insights Engine  │  │ │
│  │  │ (6 rules)    │ │ (grouping)   │ │ (anomaly rules)  │  │ │
│  │  └─────────────┘ └──────────────┘ └──────────────────┘  │ │
│  │  ┌─────────────┐ ┌──────────────┐ ┌──────────────────┐  │ │
│  │  │ Change       │ │ Impact       │ │ State Snapshot   │  │ │
│  │  │ Intelligence │ │ Analyzer     │ │ (time-travel)    │  │ │
│  │  └─────────────┘ └──────────────┘ └──────────────────┘  │ │
│  └──────────────────────────────────────────────────────────┘ │
│  ┌──────────────────────────────────────────────────────────┐ │
│  │                    SQLite Storage                         │ │
│  │  wide_events | changes | event_relationships | incidents │ │
│  │  incident_events | state_snapshots | insights            │ │
│  └──────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────┘
```

---

## Part 1: Wide Events + Collection

### Wide Event Schema

Every K8s event is captured as a wide event with 40+ fields:

```json
{
  "event_id": "evt_a8f3c2",
  "timestamp": "2026-04-04T03:42:01Z",
  "cluster_id": "docker-desktop",

  "event.type": "Warning",
  "event.reason": "OOMKilled",
  "event.message": "Container exceeded memory limit",
  "event.count": 1,
  "event.source_component": "kubelet",
  "event.source_host": "worker-node-03",

  "resource.kind": "Pod",
  "resource.name": "checkout-api-7d4f8b-x2k9p",
  "resource.namespace": "production",
  "resource.uid": "uid-abc-123",

  "k8s.node": "worker-node-03",
  "k8s.node_pool": "default-pool",
  "k8s.container": "checkout",
  "k8s.image": "checkout-api:v2.4.1",
  "k8s.replica_count": 1,

  "kubilitics.health_score": 61,
  "kubilitics.health_score_before": 87,
  "kubilitics.health_delta": -26,
  "kubilitics.is_spof": true,
  "kubilitics.blast_radius_pct": 18,
  "kubilitics.risk_level": "high",
  "kubilitics.namespace_health": 72,

  "deploy.version": "v2.4.1",
  "deploy.commit": "a3f82c1",
  "deploy.config_generation": 14,

  "causality.caused_by_event_id": "evt_parent",
  "causality.confidence": 0.9,
  "causality.rule": "deployment_causes_pod_lifecycle",
  "causality.correlation_group_id": "grp_abc123",

  "impact.services_affected": ["payment-svc", "order-svc"],
  "impact.services_count": 2,
  "impact.pods_affected": 12,
  "impact.cascade_depth": 2,

  "incident.id": "inc_042",
  "incident.role": "symptom"
}
```

### Collection Pipeline

The Go backend watches three K8s sources:

1. **Event stream** — `core/v1 Events` via informer. Captures: pod lifecycle, scheduling, scaling, image pulls, probes, OOMKills, evictions.
2. **Resource watches** — Deployment/StatefulSet/Service/ConfigMap spec changes via existing informers. When a resource spec changes, emit a wide event with `field_path`, `old_value`, `new_value`, and store a first-class `change` record.
3. **Health score changes** — When Kubilitics recalculates health scores, emit a synthetic wide event with before/after scores.

Each raw event passes through the pipeline:
1. **Enrich** — Look up resource in topology graph → add SPOF status, blast radius, node info, deployment version
2. **Causality** — Run causal inference rules → link to parent event, assign correlation group
3. **Impact** — Walk topology dependents → compute dynamic impact (services, pods, cascade depth)
4. **Incident** — Check if event belongs to an active incident or triggers a new one
5. **Store** — Write to SQLite with all enrichments

---

## Part 2: Causality Engine

### Causal Inference Rules

The enricher runs 6 rules on each event to find its probable cause:

| Rule | Pattern | Confidence |
|------|---------|------------|
| `deployment_causes_pod_lifecycle` | Pod event within 5 min of owning Deployment's rollout | 0.90 |
| `oom_causes_crashloop` | CrashLoopBackOff follows OOMKill on same container within 2 min | 0.95 |
| `node_causes_eviction` | Pod eviction follows Node NotReady within 5 min on same node | 0.95 |
| `config_causes_restart` | Pod restart follows mounted ConfigMap/Secret change within 2 min | 0.85 |
| `scaledown_causes_spof` | SPOF event follows HPA/manual scale-down to 1 replica | 0.90 |
| `quota_causes_scheduling` | FailedScheduling follows ResourceQuota exceeded in same namespace | 0.85 |

### Correlation Groups

When events are causally linked, they share a `correlation_group_id`. This allows querying the entire causal chain:

```sql
SELECT * FROM wide_events 
WHERE correlation_group_id = 'grp_abc123' 
ORDER BY timestamp ASC
```

### Causal Chain API

```
GET /api/v1/events/:id/chain
Returns:
{
  "root_cause": { ...event... },
  "chain": [
    { "event": {...}, "caused_by": "evt_parent", "confidence": 0.95, "role": "cause" },
    { "event": {...}, "caused_by": "evt_abc", "confidence": 0.90, "role": "symptom" },
    ...
  ],
  "summary": "Deployment rollout of checkout-api:v2.4.1 caused OOMKill, creating a SPOF and dropping health score by 26 points"
}
```

---

## Part 3: Change Intelligence

### Changes as First-Class Data

Changes (deployments, config updates, scaling) are stored in their own table with field-level diffs:

```sql
CREATE TABLE changes (
  change_id        TEXT PRIMARY KEY,
  timestamp        INTEGER NOT NULL,
  cluster_id       TEXT NOT NULL,
  resource_kind    TEXT NOT NULL,
  resource_name    TEXT NOT NULL,
  namespace        TEXT NOT NULL,
  change_type      TEXT NOT NULL,      -- rollout, config_update, scale, image_update, policy_change
  field_changes    TEXT NOT NULL,      -- JSON: [{path, old, new}]
  change_source    TEXT,               -- kubectl, helm, argocd, ci-pipeline, hpa
  events_caused    INTEGER DEFAULT 0,  -- retrospective impact count
  health_impact    REAL,               -- health delta within 10 min
  incident_id      TEXT,
  event_id         TEXT,               -- link to the wide event
  dimensions       TEXT NOT NULL
);
```

### Field-Level Diff

```json
{
  "field_changes": [
    {"path": "spec.replicas", "old": 3, "new": 1},
    {"path": "spec.template.spec.containers[0].image", "old": "checkout-api:v2.3.0", "new": "checkout-api:v2.4.1"},
    {"path": "spec.template.spec.containers[0].resources.limits.memory", "old": "512Mi", "new": "256Mi"}
  ]
}
```

### Retrospective Impact Scoring

A background goroutine runs 10 minutes after each change:
- Count Warning events in the causal chain that trace back to this change
- Compute health score delta in the 10-minute window
- Update `events_caused` and `health_impact` fields

---

## Part 4: Event Relationships

### Relationship Types

```sql
CREATE TABLE event_relationships (
  source_event_id   TEXT NOT NULL,
  target_event_id   TEXT NOT NULL,
  relationship      TEXT NOT NULL,
  confidence        REAL DEFAULT 1.0,
  metadata          TEXT,
  PRIMARY KEY (source_event_id, target_event_id, relationship)
);
```

| Relationship | Meaning | Detection |
|---|---|---|
| `caused_by` | A directly caused B | Causal engine rules |
| `impacts` | A's failure affects B | Topology graph dependents |
| `follows` | B happened after A on same resource | Same resource_uid within 5 min |
| `triggered_by` | A triggered automated response B | HPA/VPA action follows threshold |
| `co_occurs` | A and B happen together suspiciously often | Same 30-second window, >3 co-occurrences in 24h |
| `resolves` | B fixes the problem A reported | Warning→Normal for same resource+reason within 10 min |

---

## Part 5: Incident Narratives

### Incident Detection

An incident starts when:
1. A Warning event with `health_delta < -10`, OR
2. More than 5 Warning events in same namespace within 2 min, OR
3. A node goes NotReady

An incident ends when:
1. No new Warning events for 10 minutes in affected scope, OR
2. Health score recovers to within 5 points of pre-incident level

### Incident Storage

```sql
CREATE TABLE incidents (
  incident_id          TEXT PRIMARY KEY,
  started_at           INTEGER NOT NULL,
  ended_at             INTEGER,
  status               TEXT NOT NULL,       -- active, resolved, false_positive
  severity             TEXT NOT NULL,       -- critical, high, medium, low
  namespace            TEXT,
  primary_resource     TEXT,
  events_count         INTEGER DEFAULT 0,
  health_before        REAL,
  health_after         REAL,
  health_lowest        REAL,
  root_cause_event_id  TEXT,
  root_cause_summary   TEXT,
  resolution_event_id  TEXT,
  ttd_seconds          INTEGER,            -- time to detect
  ttr_seconds          INTEGER,            -- time to resolve
  dimensions           TEXT
);

CREATE TABLE incident_events (
  incident_id  TEXT NOT NULL,
  event_id     TEXT NOT NULL,
  role         TEXT NOT NULL,              -- trigger, symptom, cause, resolution
  PRIMARY KEY (incident_id, event_id)
);
```

### Incident Narrative Structure

Each incident is rendered as a 3-phase story:
1. **Root Cause** — The change or failure that started the cascade
2. **Cascade** — The chain of symptoms and downstream effects
3. **Resolution** — What fixed it and when health recovered

---

## Part 6: Dynamic Impact Analysis

### Computation

When a Warning event occurs, walk the topology graph from the affected resource:

1. Get the resource node from topology
2. Walk dependents up to 3 levels deep
3. Filter to currently-degraded or would-be-affected resources
4. Compute: services affected, pods affected, namespace spread, cascade depth

### Impact Fields on Wide Events

```json
{
  "impact.services_affected": ["payment-svc", "order-svc", "notification-svc"],
  "impact.services_count": 3,
  "impact.namespaces_affected": ["production", "platform"],
  "impact.pods_affected": 12,
  "impact.cascade_depth": 2,
  "impact.estimated_request_impact": "high"
}
```

---

## Part 7: Proactive Insights

### Rules-Based Anomaly Detection

Runs every 60 seconds, compares recent rates against trailing baselines:

| Rule | Trigger | Severity |
|------|---------|----------|
| `oom_spike` | OOMKills in 30 min > 3x 24h baseline, and >= 3 | warning |
| `restart_storm` | >10 pod restarts in same namespace in 5 min | warning |
| `scheduling_failures` | >5 FailedScheduling in 10 min | warning |
| `image_pull_failures` | >3 ImagePullBackOff in 5 min | warning |
| `cascading_failures` | >3 different resources in same causal chain | critical |
| `health_drift` | Health score trending down >5 points over 1 hour | info |

### Insights Storage

```sql
CREATE TABLE insights (
  insight_id    TEXT PRIMARY KEY,
  timestamp     INTEGER NOT NULL,
  rule          TEXT NOT NULL,
  severity      TEXT NOT NULL,
  title         TEXT NOT NULL,
  detail        TEXT,
  namespace     TEXT,
  status        TEXT NOT NULL,       -- active, dismissed, resolved
  resolved_at   INTEGER,
  dimensions    TEXT
);
```

---

## Part 8: Time-Travel Debugging

### State Snapshots

A goroutine captures cluster state every 5 minutes:

```sql
CREATE TABLE state_snapshots (
  snapshot_id       TEXT PRIMARY KEY,
  timestamp         INTEGER NOT NULL,
  cluster_id        TEXT NOT NULL,
  health_score      REAL,
  total_pods        INTEGER,
  running_pods      INTEGER,
  pending_pods      INTEGER,
  failed_pods       INTEGER,
  spof_count        INTEGER,
  warning_count     INTEGER,
  node_count        INTEGER,
  nodes_ready       INTEGER,
  namespace_states  TEXT NOT NULL,     -- JSON
  deployment_states TEXT NOT NULL      -- JSON (top 50 by pod count)
);
```

### Reconstruction

To show state at time T:
1. Find the nearest snapshot before T
2. Replay all wide events between snapshot and T to compute deltas
3. Present the reconstructed state

**Volume**: 288 snapshots/day × ~2KB = ~4MB/week.

---

## Part 9: Enhanced Log Viewer

### JSON Detection + Display

When log lines arrive:
1. Attempt `JSON.parse()` on each line
2. If valid JSON with recognized structure (`level`/`msg`/`message`), treat as structured
3. Non-JSON lines display as plain text (unchanged)

### Structured Log Features

- **Collapsed view**: Timestamp, level badge, message, top 3-4 fields inline
- **Expanded view**: All key-value pairs with clickable filter buttons
- **Search bar**: `key = value` queries with autocomplete
- **Field facet sidebar**: Auto-extracted dimensions with value counts
- **Log ↔ Events link**: `trace_id`/`request_id` links to Events Intelligence

### System Context Markers

Inline markers in the log stream showing system events at the correct timestamp:

```
10:30:02  INFO   connecting to database...         db.host=postgres-0
──── ⚡ SYSTEM EVENT: postgres-0 OOMKilled (10:30:01) ─────────────
10:30:03  ERROR  connection timeout after 5000ms   db.host=postgres-0
```

Implementation: Log viewer fetches events for the pod's scope (and topology dependencies) for the visible time window, merges into the log stream by timestamp.

---

## Storage Summary

```sql
-- Core (from original design)
CREATE TABLE wide_events (...);          -- 40+ field wide events

-- Causality + Relationships
ALTER TABLE wide_events ADD COLUMN caused_by_event_id TEXT;
ALTER TABLE wide_events ADD COLUMN causal_confidence REAL;
ALTER TABLE wide_events ADD COLUMN causal_rule TEXT;
ALTER TABLE wide_events ADD COLUMN correlation_group_id TEXT;
ALTER TABLE wide_events ADD COLUMN incident_id TEXT;
CREATE TABLE event_relationships (...);  -- Multi-type relationships

-- Change Intelligence
CREATE TABLE changes (...);              -- First-class change tracking with diffs

-- Incident Narratives
CREATE TABLE incidents (...);            -- Incident detection + grouping
CREATE TABLE incident_events (...);      -- Event-to-incident links

-- Proactive Insights
CREATE TABLE insights (...);             -- Anomaly detection results

-- Time-Travel
CREATE TABLE state_snapshots (...);      -- Periodic cluster state snapshots
```

**Retention**: 7 days default for events/insights. 30 days for incidents and snapshots. Configurable.

---

## API Endpoints

```
-- Events
GET  /api/v1/events/stream          -- SSE live stream
GET  /api/v1/events/query           -- Timeline with filters
POST /api/v1/events/analyze         -- Dimensional analysis
GET  /api/v1/events/:id             -- Single event + context panel
GET  /api/v1/events/:id/chain       -- Causal chain
GET  /api/v1/events/:id/relationships -- All relationships
GET  /api/v1/events/stats           -- Summary stats

-- Changes
GET  /api/v1/changes/recent         -- Recent changes with diffs + impact
GET  /api/v1/changes/:id            -- Single change with downstream events

-- Incidents
GET  /api/v1/incidents              -- List incidents (active + recent)
GET  /api/v1/incidents/:id          -- Full incident narrative
GET  /api/v1/incidents/:id/events   -- All events in incident

-- Insights
GET  /api/v1/insights/active        -- Currently active insights
POST /api/v1/insights/:id/dismiss   -- Dismiss an insight

-- Time-Travel
GET  /api/v1/state/at?t=<timestamp> -- Reconstructed state at point in time
GET  /api/v1/state/snapshots        -- List available snapshots
```

---

## Component Inventory

### New Backend Files

```
internal/events/collector.go       -- K8s event watcher + resource change detection
internal/events/enricher.go        -- Enrich raw events with Kubilitics context
internal/events/causality.go       -- Causal inference engine (6 rules)
internal/events/changes.go         -- Change detection + diff + impact scoring
internal/events/relationships.go   -- Multi-type relationship builder
internal/events/incidents.go       -- Incident detection + grouping + narrative
internal/events/impact.go          -- Dynamic blast radius computation
internal/events/insights.go        -- Proactive anomaly rules engine
internal/events/snapshots.go       -- State snapshot collection + time-travel
internal/events/store.go           -- SQLite storage layer (all tables)
internal/events/query.go           -- Query engine (timeline + analyze)
internal/events/retention.go       -- Background cleanup goroutine
internal/events/api.go             -- HTTP handlers for all endpoints
internal/events/sse.go             -- Server-Sent Events stream handler
internal/events/types.go           -- Go types for all data structures
```

### New Frontend Files

```
-- Pages
src/pages/EventsIntelligence.tsx           -- Main page (3 modes)

-- Event Components
src/components/events/EventTimeline.tsx     -- Timeline view
src/components/events/EventRow.tsx          -- Single event row
src/components/events/EventAnalyze.tsx      -- Analyze mode query builder
src/components/events/EventContextPanel.tsx -- Right slide-out context
src/components/events/EventFilters.tsx      -- Left sidebar filters
src/components/events/EventStats.tsx        -- Bottom stats bar
src/components/events/AnalyzeChart.tsx      -- Bar/line chart
src/components/events/PresetQueries.tsx     -- Quick query buttons
src/components/events/CausalChain.tsx       -- Directed cause→effect graph
src/components/events/ChangePanel.tsx       -- Recent changes with diffs
src/components/events/IncidentView.tsx      -- Incident narrative view
src/components/events/IncidentCard.tsx      -- Single incident summary
src/components/events/ImpactPanel.tsx       -- Dynamic impact analysis
src/components/events/InsightsBanner.tsx    -- Proactive insights alerts
src/components/events/TimeTravelSlider.tsx  -- State reconstruction scrubber

-- Log Components
src/components/logs/StructuredLogRow.tsx    -- JSON log row with fields
src/components/logs/LogFieldFacets.tsx      -- Auto-extracted field sidebar
src/components/logs/LogQueryBar.tsx         -- Structured search input
src/components/logs/SystemEventMarker.tsx   -- Inline system events in logs

-- Hooks + Services
src/hooks/useEventsQuery.ts                -- React Query hook for events
src/hooks/useEventsStream.ts               -- SSE hook for live stream
src/hooks/useEventAnalysis.ts              -- React Query for analyze
src/hooks/useIncidents.ts                  -- React Query for incidents
src/hooks/useInsights.ts                   -- React Query for insights
src/hooks/useTimeTravel.ts                 -- React Query for state snapshots
src/hooks/useLogParser.ts                  -- JSON log detection + extraction
src/services/api/events.ts                 -- API client for all endpoints
src/stores/eventsStore.ts                  -- Zustand store for filter/query state
src/types/events.ts                        -- TypeScript types

-- Modified Files
src/App.tsx                                -- Add /events route
src/components/layout/Sidebar.tsx          -- Add "Events" nav item
src/components/resources/LogViewer.tsx      -- Upgrade with JSON + system markers
```

---

## Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Storage | SQLite with 7 tables | Zero deps, offline-first, fits desktop. JSON columns for flexibility. |
| Data source (v1) | K8s events + resource watches | Immediate value, no user setup. OTel is Phase B. |
| Causality | Rules-based (6 rules) | Deterministic, explainable, no ML needed. Covers 80% of K8s causal patterns. |
| Incident detection | Threshold-based | Simple, predictable. ML-based grouping is Phase B. |
| Anomaly detection | Rate comparison vs. baseline | Simple, no training data needed. Works from day 1. |
| State snapshots | Every 5 min | 4MB/week, trivial. Fast reconstruction with event replay. |
| Log parsing | JSON-only, client-side | No backend changes. Modern apps emit JSON. |
| Context panel | Live-pulled | Always current state. Avoids stale data. |

---

## Out of Scope (v1)

- OpenTelemetry span ingestion (Phase B)
- K8s audit log collection (Phase C)
- Bi-directional page integration (enhancement)
- Cross-cluster event correlation (Fleet feature)
- ML-based incident grouping
- Non-JSON log parsing
- Log aggregation across pods
- Log persistence/indexing

---

## Success Criteria

1. User can answer "why did my health score drop?" in under 30 seconds via causal chain
2. User can see incidents as narratives (root cause → cascade → resolution) not flat event lists
3. User can see "what changed" with field-level diffs and retrospective impact scores
4. User gets proactive alerts for anomalous patterns before they become incidents
5. User can time-travel to any point in the last 7 days and see cluster state
6. User can see system events inline in pod logs explaining application errors
7. Zero additional setup required — works immediately after cluster connection
