/**
 * useFleetOverview — Fetches all clusters and their health summaries for the Fleet Dashboard.
 *
 * TASK-ENT-004: Fleet Dashboard
 *
 * FLEET-N1 (docs/FLEET-N1-IMPLEMENTATION.md): previously issued GET
 * /api/v1/clusters followed by N parallel GET /clusters/{id}/summary calls
 * (one per cluster) — quantified at ~21.6s cold vs ~1.8s aggregate at 50
 * synthetic clusters. Now a single GET /api/v1/fleet/overview request
 * serves the whole fleet; the backend already did this N-way fan-out
 * server-side, it just wasn't returning everything this hook needed.
 * - Aggregates totals: nodes, pods, healthy/degraded/failed cluster counts.
 * - Polls every 30s via TanStack Query refetchInterval.
 */
import { useQuery } from '@tanstack/react-query';
import { useBackendConfigStore, getEffectiveBackendBaseUrl } from '@/stores/backendConfigStore';
import { getFleetOverview } from '@/services/backendApiClient';
import type { BackendFleetClusterInfo } from '@/services/backendApiClient';

/** Shape of a single cluster in the fleet view. */
export interface FleetCluster {
  id: string;
  name: string;
  context: string;
  /**
   * UX-2 (docs/PRODUCTION-HARDENING-ROADMAP.md, Phase 7): 'unknown' is a
   * distinct state from 'healthy' — it means Fleet has no basis to claim the
   * cluster is fine (missing/unrecognized status, and no summary data yet).
   * Never collapse 'unknown' into 'healthy' at a render site.
   */
  status: 'healthy' | 'warning' | 'error' | 'unknown';
  provider?: string;
  version?: string;
  region?: string;
  nodeCount: number;
  podCount: number;
  healthScore: number;
  healthGrade: string;
  deploymentCount: number;
  serviceCount: number;
  lastConnected?: string;
  healthReason?: string;
  /** True when the live cluster fetch succeeded (backend ClusterSummary.reachable). Undefined = not yet known. */
  reachable?: boolean;
  /** True when the displayed counts/health came from the backend's last-known-good cache, not a live fetch. */
  stale?: boolean;
  /** ISO timestamp of the stale snapshot; set only when stale=true. */
  staleAsOf?: string;
  /** The underlying reason surfaced by the backend when reachable=false. */
  errorMessage?: string;
  /** True when this cluster's /summary request itself failed (network/timeout) — distinct from the
   * summary succeeding and reporting bad health. Status still falls back to cluster.status in this
   * case; this flag lets the UI additionally say "health detail unavailable" rather than staying silent. */
  summaryUnavailable?: boolean;
  /** cluster-connection-state plan, Task 7: same freshness fields the
   * Picker already exposes via presence.RegisteredCluster, so Fleet and
   * Picker rows can render identical "last seen" copy. */
  lastCheckedAt?: string;
  lastSuccessAt?: string;
  lastError?: string;
}

/** Aggregate metrics across the fleet. */
export interface FleetAggregates {
  totalClusters: number;
  totalNodes: number;
  totalPods: number;
  totalDeployments: number;
  healthyClusters: number;
  degradedClusters: number;
  failedClusters: number;
  /** UX-2: clusters whose health could not be determined — never folded into healthyClusters. */
  unknownClusters: number;
}

export interface FleetOverviewResult {
  clusters: FleetCluster[];
  aggregates: FleetAggregates;
  isLoading: boolean;
  isError: boolean;
  error: Error | null;
}

const FLEET_POLL_INTERVAL = 30_000;

// UX-2 (docs/PRODUCTION-HARDENING-ROADMAP.md, Phase 7): both mappers
// previously defaulted a missing/unrecognized status string to 'healthy'
// ("if (!status) return 'healthy'") — a cluster Fleet had no information
// about rendered identically to one confirmed healthy. 'connected'/known
// health labels still map to 'healthy' (that is a real, positive signal);
// only the "we don't actually know" case changed, to 'unknown'.
function mapBackendStatus(status?: string): 'healthy' | 'warning' | 'error' | 'unknown' {
  if (!status) return 'unknown';
  const s = status.toLowerCase();
  if (s === 'error' || s === 'failed' || s === 'disconnected' || s === 'unreachable') return 'error';
  if (s === 'warning' || s === 'degraded') return 'warning';
  if (s === 'connected' || s === 'healthy') return 'healthy';
  return 'unknown';
}

// Mirrors ClusterHealthWidget.tsx's STATUS_CONFIG classification (same backend
// healthscore vocabulary: excellent/healthy/good, fair/degraded/poor,
// unhealthy/critical) so Fleet and the Dashboard agree on what each label means.
function mapHealthStatus(healthStatus?: string): 'healthy' | 'warning' | 'error' | 'unknown' {
  if (!healthStatus) return 'unknown';
  const s = healthStatus.toLowerCase();
  if (s === 'excellent' || s === 'healthy' || s === 'good') return 'healthy';
  if (s === 'fair' || s === 'degraded' || s === 'poor' || s === 'warning') return 'warning';
  if (s === 'unhealthy' || s === 'critical' || s === 'error' || s === 'failed') return 'error';
  return 'unknown';
}

function mergeCluster(c: BackendFleetClusterInfo): FleetCluster {
  // summary_unavailable (the /summary fetch itself failed) takes precedence
  // over the raw backend cluster.status string — mirrors the previous
  // two-query version's mapHealthStatus-over-mapBackendStatus precedence.
  const status = c.summary_unavailable
    ? mapBackendStatus(c.status)
    : mapHealthStatus(c.healthStatus);

  return {
    id: c.id,
    name: c.name,
    context: c.context ?? '',
    status,
    provider: c.provider,
    version: c.version,
    region: undefined, // backend does not expose region currently
    nodeCount: c.nodes,
    podCount: c.pods,
    healthScore: 0, // not computed server-side; unchanged from the pre-fix behavior
    healthGrade: status === 'healthy' ? 'A' : status === 'warning' ? 'C' : status === 'error' ? 'F' : '?',
    deploymentCount: c.deployments,
    serviceCount: c.services,
    lastConnected: c.last_connected,
    healthReason: c.healthReason,
    reachable: c.reachable,
    stale: c.stale,
    staleAsOf: c.stale_as_of,
    errorMessage: c.error_message,
    summaryUnavailable: !!c.summary_unavailable,
    lastCheckedAt: c.last_checked_at,
    lastSuccessAt: c.last_success_at,
    lastError: c.last_error,
  };
}

function computeAggregates(clusters: FleetCluster[]): FleetAggregates {
  return {
    totalClusters: clusters.length,
    totalNodes: clusters.reduce((sum, c) => sum + c.nodeCount, 0),
    totalPods: clusters.reduce((sum, c) => sum + c.podCount, 0),
    totalDeployments: clusters.reduce((sum, c) => sum + c.deploymentCount, 0),
    healthyClusters: clusters.filter((c) => c.status === 'healthy').length,
    degradedClusters: clusters.filter((c) => c.status === 'warning').length,
    failedClusters: clusters.filter((c) => c.status === 'error').length,
    unknownClusters: clusters.filter((c) => c.status === 'unknown').length,
  };
}

export function useFleetOverview(): FleetOverviewResult {
  const stored = useBackendConfigStore((s) => s.backendBaseUrl);
  const backendBaseUrl = getEffectiveBackendBaseUrl(stored);
  const isConfigured = useBackendConfigStore((s) => s.isBackendConfigured());

  // FLEET-N1: single request for cluster list + per-cluster summaries +
  // aggregate totals — was GET /clusters + N×GET /clusters/{id}/summary.
  const overviewQuery = useQuery({
    queryKey: ['fleet', 'overview', backendBaseUrl],
    queryFn: () => getFleetOverview(backendBaseUrl),
    enabled: isConfigured,
    refetchInterval: FLEET_POLL_INTERVAL,
    staleTime: 15_000,
  });

  const backendClusters = overviewQuery.data?.clusters ?? [];
  const fleetClusters: FleetCluster[] = backendClusters.map(mergeCluster);
  const aggregates = computeAggregates(fleetClusters);

  // Only show loading skeleton on FIRST load, not on refetch — mirrors the
  // pre-fix isLoading semantics (true only while there is no data yet).
  const isLoading = overviewQuery.isLoading && !overviewQuery.data;

  return {
    clusters: fleetClusters,
    aggregates,
    isLoading,
    isError: overviewQuery.isError,
    error: overviewQuery.error as Error | null,
  };
}
