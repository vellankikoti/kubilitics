/**
 * Generic K8s resource CRUD + specialized resource operations
 * (list, get, create, update, delete, apply YAML, rollout, search, consumers, etc.)
 */
import { backendRequest, BackendApiError, CONFIRM_DESTRUCTIVE_HEADER } from './client';
import type {
  BackendResourceListResponse,
  RolloutHistoryRevision,
  SearchResponse,
  ConsumersResponse,
  TLSSecretInfo,
  NodeDrainResult,
} from './types';

/**
 * GET /api/v1/clusters/{clusterId}/crd-instances/{crdName} — list instances of a CRD by full name (e.g. certificates.cert-manager.io).
 * Query: namespace, limit, continue, labelSelector, fieldSelector.
 */
export async function listCRDInstances(
  baseUrl: string,
  clusterId: string,
  crdName: string,
  params?: { namespace?: string; limit?: number; continue?: string; labelSelector?: string; fieldSelector?: string }
): Promise<BackendResourceListResponse> {
  const search = new URLSearchParams();
  if (params?.namespace !== undefined && params.namespace !== '') search.set('namespace', params.namespace);
  if (params?.limit != null) search.set('limit', String(params.limit));
  if (params?.continue) search.set('continue', params.continue);
  if (params?.labelSelector) search.set('labelSelector', params.labelSelector);
  if (params?.fieldSelector) search.set('fieldSelector', params.fieldSelector);
  const query = search.toString();
  const path = `clusters/${encodeURIComponent(clusterId)}/crd-instances/${encodeURIComponent(crdName)}${query ? `?${query}` : ''}`;
  return backendRequest<BackendResourceListResponse>(baseUrl, path);
}

export interface ResourceListParams {
  namespace?: string;
  namespaces?: string[];
  limit?: number;
  continue?: string;
  labelSelector?: string;
  fieldSelector?: string;
  /** Server-side full-text search (name, namespace). Requires informer cache. */
  search?: string;
  /** Server-side sort field: name | namespace | creationTimestamp */
  sortBy?: string;
  /** Server-side sort direction */
  sortOrder?: 'asc' | 'desc';
  /** Offset-based pagination offset (0-indexed). Requires informer cache. */
  offset?: number;
  /**
   * LOADING-2 (docs/PRODUCTION-RELIABILITY-AUDIT.md): React Query's per-query
   * cancellation signal, forwarded to backendRequest so an unmounted/stale query
   * actually cancels its in-flight fetch instead of being silently ignored.
   * Not a URL parameter — never serialized into the query string.
   */
  signal?: AbortSignal;
}

/**
 * GET /api/v1/clusters/{clusterId}/resources/{kind} — list resources by kind.
 * Query: namespace (single), namespaces (comma-separated for project scope), limit, continue,
 * labelSelector, fieldSelector, search, sortBy, sortOrder, offset (server-side pagination).
 */
export async function listResources(
  baseUrl: string,
  clusterId: string,
  kind: string,
  params?: ResourceListParams
): Promise<BackendResourceListResponse> {
  const search = new URLSearchParams();
  if (params?.namespaces !== undefined) {
    search.set('namespaces', params.namespaces.length ? params.namespaces.join(',') : '');
  } else if (params?.namespace !== undefined && params.namespace !== '') {
    search.set('namespace', params.namespace);
  }
  if (params?.limit != null) search.set('limit', String(params.limit));
  if (params?.continue) search.set('continue', params.continue);
  if (params?.labelSelector) search.set('labelSelector', params.labelSelector);
  if (params?.fieldSelector) search.set('fieldSelector', params.fieldSelector);
  if (params?.search) search.set('search', params.search);
  if (params?.sortBy) search.set('sortBy', params.sortBy);
  if (params?.sortOrder) search.set('sortOrder', params.sortOrder);
  if (params?.offset !== undefined) search.set('offset', String(params.offset));
  const query = search.toString();
  const path = `clusters/${encodeURIComponent(clusterId)}/resources/${encodeURIComponent(kind)}${query ? `?${query}` : ''}`;
  try {
    // Only pass a 3rd `init` arg when a signal is actually present, preserving
    // the exact 2-arg call shape (and existing call-site tests) otherwise.
    return params?.signal
      ? await backendRequest<BackendResourceListResponse>(baseUrl, path, { signal: params.signal })
      : await backendRequest<BackendResourceListResponse>(baseUrl, path);
  } catch (err) {
    // When a CRD/resource type doesn't exist in the cluster the backend returns 404.
    // Return an empty list instead of throwing so callers don't flood the console with errors.
    if (err instanceof BackendApiError && err.status === 404) {
      return { items: [], metadata: { total: 0, resourceVersion: '' } };
    }
    throw err;
  }
}

/**
 * GET /api/v1/clusters/{clusterId}/resources/{kind}/{namespace}/{name} — get single resource.
 * For cluster-scoped resources (IngressClass, Node, etc.) pass namespace as ''; path uses '-' sentinel.
 */
export async function getResource(
  baseUrl: string,
  clusterId: string,
  kind: string,
  namespace: string,
  name: string,
  // LOADING-2: forwarded React Query cancellation signal — see ResourceListParams.signal.
  signal?: AbortSignal
): Promise<Record<string, unknown>> {
  const ns = namespace === '' ? '-' : namespace;
  const path = `clusters/${encodeURIComponent(clusterId)}/resources/${encodeURIComponent(kind)}/${encodeURIComponent(ns)}/${encodeURIComponent(name)}`;
  return signal
    ? backendRequest<Record<string, unknown>>(baseUrl, path, { signal })
    : backendRequest<Record<string, unknown>>(baseUrl, path);
}

/**
 * PATCH /api/v1/clusters/{clusterId}/resources/{kind}/{namespace}/{name}
 * Body: JSON merge-patch (e.g. { spec: { replicas: 3 } } for scaling).
 * For cluster-scoped resources pass namespace as ''; path uses '-' sentinel.
 */
export async function patchResource(
  baseUrl: string,
  clusterId: string,
  kind: string,
  namespace: string,
  name: string,
  patch: Record<string, unknown>
): Promise<Record<string, unknown>> {
  const ns = namespace === '' ? '-' : namespace;
  const path = `clusters/${encodeURIComponent(clusterId)}/resources/${encodeURIComponent(kind)}/${encodeURIComponent(ns)}/${encodeURIComponent(name)}`;
  return backendRequest<Record<string, unknown>>(baseUrl, path, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(patch),
  });
}

/**
 * DELETE /api/v1/clusters/{clusterId}/resources/{kind}/{namespace}/{name}
 * D1.2: Requires X-Confirm-Destructive: true (call only after user confirmation).
 * For cluster-scoped resources pass namespace as ''; path uses '-' sentinel.
 */
export async function deleteResource(
  baseUrl: string,
  clusterId: string,
  kind: string,
  namespace: string,
  name: string
): Promise<{ message: string; cluster_id: string; kind: string; namespace: string; name: string }> {
  const ns = namespace === '' ? '-' : namespace;
  const path = `clusters/${encodeURIComponent(clusterId)}/resources/${encodeURIComponent(kind)}/${encodeURIComponent(ns)}/${encodeURIComponent(name)}`;
  return backendRequest(baseUrl, path, {
    method: 'DELETE',
    headers: { [CONFIRM_DESTRUCTIVE_HEADER]: 'true' },
  });
}

/**
 * POST /api/v1/clusters/{clusterId}/apply — apply YAML manifest.
 * D1.2: Requires X-Confirm-Destructive: true (review YAML before applying).
 */
export async function applyManifest(
  baseUrl: string,
  clusterId: string,
  yaml: string
): Promise<{ message: string; cluster_id: string; resources: Array<{ kind: string; namespace: string; name: string; action: string }> }> {
  const path = `clusters/${encodeURIComponent(clusterId)}/apply`;
  return backendRequest(baseUrl, path, {
    method: 'POST',
    headers: { [CONFIRM_DESTRUCTIVE_HEADER]: 'true' },
    body: JSON.stringify({ yaml }),
  });
}

/**
 * GET /api/v1/clusters/{clusterId}/search?q=...&limit=25
 * Global search for command palette: returns resources matching name or namespace (case-insensitive).
 */
export async function searchResources(
  baseUrl: string,
  clusterId: string,
  q: string,
  limit?: number
): Promise<SearchResponse> {
  const params = new URLSearchParams({ q: q.trim() });
  if (limit != null && limit > 0) params.set('limit', String(limit));
  const path = `clusters/${encodeURIComponent(clusterId)}/search?${params.toString()}`;
  return backendRequest<SearchResponse>(baseUrl, path);
}

/**
 * GET /api/v1/clusters/{clusterId}/resources/deployments/{namespace}/{name}/rollout-history
 */
export async function getDeploymentRolloutHistory(
  baseUrl: string,
  clusterId: string,
  namespace: string,
  name: string
): Promise<{ revisions: RolloutHistoryRevision[] }> {
  const path = `clusters/${encodeURIComponent(clusterId)}/resources/deployments/${encodeURIComponent(namespace)}/${encodeURIComponent(name)}/rollout-history`;
  return backendRequest<{ revisions: RolloutHistoryRevision[] }>(baseUrl, path);
}

/**
 * GET /api/v1/clusters/{clusterId}/resources/services/{namespace}/{name}/endpoints
 * Returns the Endpoints resource with the same name as the service.
 */
export async function getServiceEndpoints(
  baseUrl: string,
  clusterId: string,
  namespace: string,
  name: string
): Promise<Record<string, unknown>> {
  const path = `clusters/${encodeURIComponent(clusterId)}/resources/services/${encodeURIComponent(namespace)}/${encodeURIComponent(name)}/endpoints`;
  return backendRequest<Record<string, unknown>>(baseUrl, path);
}

/**
 * GET /api/v1/clusters/{clusterId}/resources/configmaps/{namespace}/{name}/consumers
 */
export async function getConfigMapConsumers(
  baseUrl: string,
  clusterId: string,
  namespace: string,
  name: string
): Promise<ConsumersResponse> {
  const path = `clusters/${encodeURIComponent(clusterId)}/resources/configmaps/${encodeURIComponent(namespace)}/${encodeURIComponent(name)}/consumers`;
  return backendRequest<ConsumersResponse>(baseUrl, path);
}

/**
 * GET /api/v1/clusters/{clusterId}/resources/secrets/{namespace}/{name}/consumers
 */
export async function getSecretConsumers(
  baseUrl: string,
  clusterId: string,
  namespace: string,
  name: string
): Promise<ConsumersResponse> {
  const path = `clusters/${encodeURIComponent(clusterId)}/resources/secrets/${encodeURIComponent(namespace)}/${encodeURIComponent(name)}/consumers`;
  return backendRequest<ConsumersResponse>(baseUrl, path);
}

export async function getSecretTLSInfo(
  baseUrl: string,
  clusterId: string,
  namespace: string,
  name: string
): Promise<TLSSecretInfo> {
  const path = `clusters/${encodeURIComponent(clusterId)}/resources/secrets/${encodeURIComponent(namespace)}/${encodeURIComponent(name)}/tls-info`;
  return backendRequest<TLSSecretInfo>(baseUrl, path);
}

/**
 * GET .../resources/persistentvolumeclaims/{namespace}/{name}/consumers
 */
export async function getPVCConsumers(
  baseUrl: string,
  clusterId: string,
  namespace: string,
  name: string
): Promise<ConsumersResponse> {
  const path = `clusters/${encodeURIComponent(clusterId)}/resources/persistentvolumeclaims/${encodeURIComponent(namespace)}/${encodeURIComponent(name)}/consumers`;
  return backendRequest<ConsumersResponse>(baseUrl, path);
}

/**
 * GET .../resources/storageclasses/pv-counts
 * Returns { [storageClassName]: count }
 */
export async function getStorageClassPVCounts(
  baseUrl: string,
  clusterId: string
): Promise<Record<string, number>> {
  const path = `clusters/${encodeURIComponent(clusterId)}/resources/storageclasses/pv-counts`;
  return backendRequest<Record<string, number>>(baseUrl, path);
}

/**
 * GET .../resources/namespaces/counts
 * Returns { [namespaceName]: { pods, services } } for list-page display.
 */
export async function getNamespaceCounts(
  baseUrl: string,
  clusterId: string
): Promise<Record<string, { pods: number; services: number }>> {
  const path = `clusters/${encodeURIComponent(clusterId)}/resources/namespaces/counts`;
  return backendRequest<Record<string, { pods: number; services: number }>>(baseUrl, path);
}

/**
 * GET .../resources/serviceaccounts/token-counts
 * Returns { "namespace/name": tokenCount } for service account token secrets (type=service-account-token).
 */
export async function getServiceAccountTokenCounts(
  baseUrl: string,
  clusterId: string
): Promise<Record<string, number>> {
  const path = `clusters/${encodeURIComponent(clusterId)}/resources/serviceaccounts/token-counts`;
  return backendRequest<Record<string, number>>(baseUrl, path);
}

/**
 * POST /api/v1/clusters/{clusterId}/resources/deployments/{namespace}/{name}/rollback
 * Body: { revision?: number } — optional; omit to roll back to previous revision.
 */
export async function postDeploymentRollback(
  baseUrl: string,
  clusterId: string,
  namespace: string,
  name: string,
  body?: { revision?: number }
): Promise<Record<string, unknown>> {
  const path = `clusters/${encodeURIComponent(clusterId)}/resources/deployments/${encodeURIComponent(namespace)}/${encodeURIComponent(name)}/rollback`;
  return backendRequest<Record<string, unknown>>(baseUrl, path, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body ?? {}),
  });
}

/**
 * POST /api/v1/clusters/{clusterId}/resources/nodes/{name}/cordon
 * Cordons or uncordons a node by setting spec.unschedulable.
 */
export async function postNodeCordon(
  baseUrl: string,
  clusterId: string,
  name: string,
  unschedulable: boolean
): Promise<Record<string, unknown>> {
  const path = `clusters/${encodeURIComponent(clusterId)}/resources/nodes/${encodeURIComponent(name)}/cordon`;
  return backendRequest<Record<string, unknown>>(baseUrl, path, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ unschedulable }),
  });
}

/**
 * POST /api/v1/clusters/{clusterId}/resources/nodes/{name}/drain
 * Cordons then evicts all eligible pods from the node.
 *
 * docs/ai/STABILIZATION-PLAN.md Phase 2 item 6: drain is at least as
 * destructive as delete/apply (both already send this header) — it evicts,
 * or with force=true hard-deletes, every eligible pod on the node. The
 * backend now rejects this call without X-Confirm-Destructive: true.
 */
export async function postNodeDrain(
  baseUrl: string,
  clusterId: string,
  name: string,
  options?: { gracePeriodSeconds?: number; force?: boolean; ignoreDaemonSets?: boolean }
): Promise<NodeDrainResult> {
  const path = `clusters/${encodeURIComponent(clusterId)}/resources/nodes/${encodeURIComponent(name)}/drain`;
  return backendRequest<NodeDrainResult>(baseUrl, path, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', [CONFIRM_DESTRUCTIVE_HEADER]: 'true' },
    body: JSON.stringify({
      gracePeriodSeconds: options?.gracePeriodSeconds ?? -1,
      force: options?.force ?? false,
      ignoreDaemonSets: options?.ignoreDaemonSets ?? true,
    }),
  });
}

/**
 * POST /api/v1/clusters/{clusterId}/resources/cronjobs/{namespace}/{name}/trigger
 * Creates a one-off Job from the CronJob's jobTemplate.
 */
export async function postCronJobTrigger(
  baseUrl: string,
  clusterId: string,
  namespace: string,
  name: string
): Promise<Record<string, unknown>> {
  const path = `clusters/${encodeURIComponent(clusterId)}/resources/cronjobs/${encodeURIComponent(namespace)}/${encodeURIComponent(name)}/trigger`;
  return backendRequest<Record<string, unknown>>(baseUrl, path, { method: 'POST' });
}

/**
 * GET /api/v1/clusters/{clusterId}/resources/cronjobs/{namespace}/{name}/jobs?limit=5
 * Returns last N child jobs owned by this CronJob (for expandable row drill-down).
 */
export async function getCronJobJobs(
  baseUrl: string,
  clusterId: string,
  namespace: string,
  name: string,
  limit = 5
): Promise<{ items: Record<string, unknown>[] }> {
  const path = `clusters/${encodeURIComponent(clusterId)}/resources/cronjobs/${encodeURIComponent(namespace)}/${encodeURIComponent(name)}/jobs?limit=${limit}`;
  const res = await backendRequest<{ items: Record<string, unknown>[] }>(baseUrl, path);
  return res ?? { items: [] };
}

/**
 * POST /api/v1/clusters/{clusterId}/resources/jobs/{namespace}/{name}/retry
 * Creates a new Job with the same spec (retry).
 */
export async function postJobRetry(
  baseUrl: string,
  clusterId: string,
  namespace: string,
  name: string
): Promise<Record<string, unknown>> {
  const path = `clusters/${encodeURIComponent(clusterId)}/resources/jobs/${encodeURIComponent(namespace)}/${encodeURIComponent(name)}/retry`;
  return backendRequest<Record<string, unknown>>(baseUrl, path, { method: 'POST' });
}
