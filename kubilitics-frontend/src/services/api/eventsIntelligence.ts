/**
 * API client for Events Intelligence endpoints.
 * Covers wide events, causal chains, incidents, insights, changes, and time-travel.
 */
import { backendRequest } from './client';

/* ─── Core Types (match Go types.go) ────────────────────────────────────── */

export interface WideEvent {
  event_id: string;
  timestamp: number;
  cluster_id: string;

  event_type: string;
  reason: string;
  message: string;
  source_component: string;
  source_host: string;
  event_count: number;
  first_seen: number;
  last_seen: number;

  resource_kind: string;
  resource_name: string;
  resource_namespace: string;
  resource_uid: string;
  resource_api_version: string;
  owner_kind: string;
  owner_name: string;

  node_name: string;

  health_score?: number | null;
  is_spof: number;
  blast_radius: number;
  severity: string;

  caused_by_event_id?: string | null;
  correlation_group_id: string;

  dimensions: Record<string, unknown> | null;
}

export interface Change {
  change_id: string;
  timestamp: number;
  cluster_id: string;
  resource_kind: string;
  resource_name: string;
  resource_namespace: string;
  resource_uid: string;
  change_type: string;
  field_changes: FieldChange[] | null;
  change_source: string;
  events_caused: number;
  health_impact?: number | null;
  event_id?: string | null;
}

export interface FieldChange {
  field: string;
  old_value: string;
  new_value: string;
}

export interface EventRelationship {
  source_event_id: string;
  target_event_id: string;
  relationship_type: string;
  confidence: number;
  metadata: Record<string, unknown> | null;
}

export interface Incident {
  incident_id: string;
  started_at: number;
  ended_at?: number | null;
  status: string;
  severity: string;
  cluster_id: string;
  namespace: string;
  health_before?: number | null;
  health_after?: number | null;
  health_lowest?: number | null;
  root_cause_kind: string;
  root_cause_name: string;
  root_cause_summary: string;
  ttd?: number | null;
  ttr?: number | null;
  dimensions: Record<string, unknown> | null;
}

export interface Insight {
  insight_id: string;
  timestamp: number;
  cluster_id: string;
  rule: string;
  severity: string;
  title: string;
  detail: string;
  status: string;
}

export interface StateSnapshot {
  snapshot_id: string;
  timestamp: number;
  cluster_id: string;
  total_pods: number;
  running_pods: number;
  total_nodes: number;
  ready_nodes: number;
  health_score: number;
  spof_count: number;
  warning_events: number;
  error_events: number;
  namespace_states: Record<string, unknown> | null;
  deployment_states: Record<string, unknown> | null;
}

export interface EventStats {
  total_events: number;
  by_type: Record<string, number>;
  by_severity: Record<string, number>;
  by_reason: Record<string, number>;
  since: number;
  until: number;
}

export interface ChainLink {
  event_id: string;
  timestamp: number;
  reason: string;
  resource_kind: string;
  resource_name: string;
  relationship_type: string;
  confidence: number;
}

export interface CausalChain {
  root_event_id: string;
  links: ChainLink[];
  depth: number;
}

export interface EventContext {
  event: WideEvent;
  related_events: WideEvent[];
  relationships: EventRelationship[];
  changes: Change[];
  incident?: Incident | null;
}

export interface AnalyzeResult {
  group_key: string;
  count: number;
  first_seen: number;
  last_seen: number;
  avg_health?: number | null;
}

/* ─── Query Parameters ──────────────────────────────────────────────────── */

export interface EventQueryParams {
  from?: number;
  to?: number;
  namespace?: string;
  kind?: string;
  type?: string;
  reason?: string;
  name?: string;
  node?: string;
  limit?: number;
  offset?: number;
}

export interface AnalyzeQuery {
  namespace?: string;
  since?: number;
  until?: number;
  group_by?: string;
  top_n?: number;
}

/* ─── API Functions ─────────────────────────────────────────────────────── */

function clusterPath(clusterId: string, subpath: string): string {
  return `clusters/${encodeURIComponent(clusterId)}/${subpath}`;
}

function buildQueryString(params: Record<string, string | number | undefined>): string {
  const entries = Object.entries(params).filter(([, v]) => v !== undefined && v !== '');
  if (entries.length === 0) return '';
  return '?' + entries.map(([k, v]) => `${k}=${encodeURIComponent(String(v))}`).join('&');
}

/** GET /events-intelligence/query */
export async function queryEvents(
  baseUrl: string,
  clusterId: string,
  params: EventQueryParams,
): Promise<WideEvent[]> {
  const qs = buildQueryString(params as Record<string, string | number | undefined>);
  const path = clusterPath(clusterId, `events-intelligence/query${qs}`);
  const result = await backendRequest<WideEvent[]>(baseUrl, path);
  return result ?? [];
}

/** GET /events-intelligence/stats */
export async function getEventStats(
  baseUrl: string,
  clusterId: string,
): Promise<EventStats> {
  const path = clusterPath(clusterId, 'events-intelligence/stats');
  const result = await backendRequest<EventStats>(baseUrl, path);
  result.by_type = result.by_type ?? {};
  result.by_severity = result.by_severity ?? {};
  result.by_reason = result.by_reason ?? {};
  return result;
}

/** GET /events-intelligence/{eventId} — returns EventContext */
export async function getEvent(
  baseUrl: string,
  clusterId: string,
  eventId: string,
): Promise<EventContext> {
  const path = clusterPath(clusterId, `events-intelligence/${encodeURIComponent(eventId)}`);
  const result = await backendRequest<EventContext>(baseUrl, path);
  result.related_events = result.related_events ?? [];
  result.relationships = result.relationships ?? [];
  result.changes = result.changes ?? [];
  return result;
}

/** GET /events-intelligence/{eventId}/chain */
export async function getCausalChain(
  baseUrl: string,
  clusterId: string,
  eventId: string,
): Promise<CausalChain> {
  const path = clusterPath(clusterId, `events-intelligence/${encodeURIComponent(eventId)}/chain`);
  const result = await backendRequest<CausalChain>(baseUrl, path);
  result.links = result.links ?? [];
  return result;
}

/** GET /events-intelligence/{eventId}/relationships */
export async function getEventRelationships(
  baseUrl: string,
  clusterId: string,
  eventId: string,
): Promise<EventRelationship[]> {
  const path = clusterPath(clusterId, `events-intelligence/${encodeURIComponent(eventId)}/relationships`);
  const result = await backendRequest<EventRelationship[]>(baseUrl, path);
  return result ?? [];
}

/** POST /events-intelligence/analyze */
export async function analyzeEvents(
  baseUrl: string,
  clusterId: string,
  query: AnalyzeQuery,
): Promise<AnalyzeResult[]> {
  const path = clusterPath(clusterId, 'events-intelligence/analyze');
  const result = await backendRequest<AnalyzeResult[]>(baseUrl, path, {
    method: 'POST',
    body: JSON.stringify(query),
  });
  return result ?? [];
}

/** GET /changes/recent */
export async function getRecentChanges(
  baseUrl: string,
  clusterId: string,
  limit?: number,
): Promise<Change[]> {
  const qs = limit ? `?limit=${limit}` : '';
  const path = clusterPath(clusterId, `changes/recent${qs}`);
  const result = await backendRequest<Change[]>(baseUrl, path);
  return result ?? [];
}

/** GET /incidents */
export async function getIncidents(
  baseUrl: string,
  clusterId: string,
): Promise<Incident[]> {
  const path = clusterPath(clusterId, 'incidents');
  const result = await backendRequest<Incident[]>(baseUrl, path);
  return result ?? [];
}

/** GET /incidents/{incidentId} */
export async function getIncident(
  baseUrl: string,
  clusterId: string,
  id: string,
): Promise<Incident> {
  const path = clusterPath(clusterId, `incidents/${encodeURIComponent(id)}`);
  return backendRequest<Incident>(baseUrl, path);
}

/** GET /incidents/{incidentId}/events */
export async function getIncidentEvents(
  baseUrl: string,
  clusterId: string,
  id: string,
): Promise<WideEvent[]> {
  const path = clusterPath(clusterId, `incidents/${encodeURIComponent(id)}/events`);
  const result = await backendRequest<WideEvent[]>(baseUrl, path);
  return result ?? [];
}

/** GET /insights/active */
export async function getActiveInsights(
  baseUrl: string,
  clusterId: string,
): Promise<Insight[]> {
  const path = clusterPath(clusterId, 'insights/active');
  const result = await backendRequest<Insight[]>(baseUrl, path);
  return result ?? [];
}

/** POST /insights/{insightId}/dismiss */
export async function dismissInsight(
  baseUrl: string,
  clusterId: string,
  id: string,
): Promise<void> {
  const path = clusterPath(clusterId, `insights/${encodeURIComponent(id)}/dismiss`);
  await backendRequest<unknown>(baseUrl, path, { method: 'POST' });
}

/** GET /state/at?t=<unix_ms> */
export async function getStateAt(
  baseUrl: string,
  clusterId: string,
  timestamp: number,
): Promise<StateSnapshot> {
  const path = clusterPath(clusterId, `state/at?t=${timestamp}`);
  return backendRequest<StateSnapshot>(baseUrl, path);
}

/* ─── System Health ────────────────────────────────────────────────────── */

export interface PipelineHealth {
  cluster_id: string;
  status: string;
  last_event_time: number;
  events_last_5min: number;
  uptime_seconds: number;
  collector_ok: boolean;
  insights_ok: boolean;
  snapshots_ok: boolean;
}

export interface SystemHealth {
  status: string;
  pipelines: PipelineHealth[];
  db_size_mb: number;
  total_events: number;
  uptime_seconds: number;
}

/** GET /system/events-health — system-wide pipeline health (not cluster-scoped) */
export async function getSystemHealth(baseUrl: string): Promise<SystemHealth> {
  const result = await backendRequest<SystemHealth>(baseUrl, 'system/events-health');
  result.pipelines = result.pipelines ?? [];
  return result;
}

/* ─── Log Persistence & Cross-Pod Search ───────────────────────────────── */

export interface StoredLog {
  log_id: string;
  timestamp: number;
  cluster_id: string;
  namespace: string;
  pod_name: string;
  container_name: string;
  level: string;
  message: string;
  raw_line: string;
  is_structured: boolean;
  fields: Record<string, unknown>;
  owner_kind: string;
  owner_name: string;
}

export interface LogSearchParams {
  namespace?: string;
  pod?: string;
  owner_kind?: string;
  owner_name?: string;
  level?: string;
  search?: string;
  field?: string;
  from?: number;
  to?: number;
  limit?: number;
  offset?: number;
}

/** GET /logs/search — search stored logs with filters */
export async function searchLogs(
  baseUrl: string,
  clusterId: string,
  params: LogSearchParams,
): Promise<StoredLog[]> {
  const qs = buildQueryString(params as Record<string, string | number | undefined>);
  const path = clusterPath(clusterId, `logs/search${qs}`);
  const result = await backendRequest<StoredLog[]>(baseUrl, path);
  return result ?? [];
}

/** GET /logs/aggregate — cross-pod aggregated logs for a workload */
export async function aggregateLogs(
  baseUrl: string,
  clusterId: string,
  ownerKind: string,
  ownerName: string,
  namespace: string,
  options?: { level?: string; from?: number; to?: number; limit?: number },
): Promise<StoredLog[]> {
  const params: Record<string, string | number | undefined> = {
    owner_kind: ownerKind,
    owner_name: ownerName,
    namespace,
    ...options,
  };
  const qs = buildQueryString(params);
  const path = clusterPath(clusterId, `logs/aggregate${qs}`);
  const result = await backendRequest<StoredLog[]>(baseUrl, path);
  return result ?? [];
}
