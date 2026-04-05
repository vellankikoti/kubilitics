/**
 * API client for OpenTelemetry Traces endpoints.
 * Covers trace listing, trace detail, and service map.
 */
import { backendRequest } from './client';

/* ─── Types ────────────────────────────────────────────────────────────── */

export interface Span {
  span_id: string;
  trace_id: string;
  parent_span_id: string;
  service_name: string;
  operation_name: string;
  span_kind: string;
  start_time: number;
  end_time: number;
  duration_ns: number;
  status_code: string;
  status_message: string;
  http_method: string;
  http_url: string;
  http_status_code: number | null;
  http_route: string;
  db_system: string;
  db_statement: string;
  k8s_pod_name: string;
  k8s_namespace: string;
  k8s_node_name: string;
  k8s_container: string;
  k8s_deployment: string;
  user_id: string;
  cluster_id: string;
  attributes: Record<string, unknown>;
  events: unknown[];
  linked_event_ids: string[];
}

export interface TraceSummary {
  trace_id: string;
  root_service: string;
  root_operation: string;
  start_time: number;
  duration_ns: number;
  span_count: number;
  error_count: number;
  service_count: number;
  status: string;
  services: string[];
}

export interface TraceDetail {
  summary: TraceSummary;
  spans: Span[];
}

export interface ServiceNode {
  name: string;
  span_count: number;
  error_count: number;
  avg_duration_ns: number;
}

export interface ServiceEdge {
  source: string;
  target: string;
  count: number;
}

export interface ServiceMap {
  nodes: ServiceNode[];
  edges: ServiceEdge[];
}

export interface TraceQueryParams {
  service?: string;
  operation?: string;
  status?: string;
  min_duration?: number;
  from?: number;
  to?: number;
  limit?: number;
}

/* ─── Helpers ──────────────────────────────────────────────────────────── */

function clusterPath(clusterId: string, subpath: string): string {
  return `clusters/${encodeURIComponent(clusterId)}/${subpath}`;
}

function buildQueryString(params: Record<string, string | number | undefined>): string {
  const entries = Object.entries(params).filter(([, v]) => v !== undefined && v !== '');
  if (entries.length === 0) return '';
  return '?' + entries.map(([k, v]) => `${k}=${encodeURIComponent(String(v))}`).join('&');
}

/* ─── API Functions ────────────────────────────────────────────────────── */

/** GET /clusters/{clusterId}/traces — list traces with optional filters */
export async function listTraces(
  baseUrl: string,
  clusterId: string,
  params: TraceQueryParams,
): Promise<TraceSummary[]> {
  const qs = buildQueryString(params as Record<string, string | number | undefined>);
  const path = clusterPath(clusterId, `traces${qs}`);
  const result = await backendRequest<TraceSummary[]>(baseUrl, path);
  return result ?? [];
}

/** GET /clusters/{clusterId}/traces/{traceId} — get full trace with all spans */
export async function getTrace(
  baseUrl: string,
  clusterId: string,
  traceId: string,
): Promise<TraceDetail> {
  const path = clusterPath(clusterId, `traces/${encodeURIComponent(traceId)}`);
  const result = await backendRequest<TraceDetail>(baseUrl, path);
  result.spans = result.spans ?? [];
  return result;
}

/** GET /clusters/{clusterId}/resource-traces — traces for a specific K8s resource */
export async function getResourceTraces(
  baseUrl: string,
  clusterId: string,
  params: { kind: string; name: string; namespace?: string; from?: number; to?: number; limit?: number },
): Promise<TraceSummary[]> {
  const qs = buildQueryString(params as Record<string, string | number | undefined>);
  const path = clusterPath(clusterId, `resource-traces${qs}`);
  const result = await backendRequest<TraceSummary[]>(baseUrl, path);
  return result ?? [];
}

/** GET /clusters/{clusterId}/events-intelligence/{eventId}/traces — traces linked to an event */
export async function getLinkedTraces(
  baseUrl: string,
  clusterId: string,
  eventId: string,
): Promise<TraceSummary[]> {
  const path = clusterPath(clusterId, `events-intelligence/${encodeURIComponent(eventId)}/traces`);
  const result = await backendRequest<TraceSummary[]>(baseUrl, path);
  return result ?? [];
}

/** GET /clusters/{clusterId}/traces/services — service dependency map */
export async function getServiceMap(
  baseUrl: string,
  clusterId: string,
  from?: number,
  to?: number,
): Promise<ServiceMap> {
  const params: Record<string, string | number | undefined> = { from, to };
  const qs = buildQueryString(params);
  const path = clusterPath(clusterId, `traces/services${qs}`);
  const result = await backendRequest<ServiceMap>(baseUrl, path);
  result.nodes = result.nodes ?? [];
  result.edges = result.edges ?? [];
  return result;
}
