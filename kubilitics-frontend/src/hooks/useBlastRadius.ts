/**
 * Hook for fetching cluster-wide blast radius analysis.
 * Checks if graph engine is ready, fetches blast radius if so.
 * Never blocks UI — falls back gracefully if graph engine isn't available.
 */
import { useQuery } from '@tanstack/react-query';
import { getBlastRadius, getGraphStatus } from '@/services/api/blastRadius';
import { useActiveClusterId } from './useActiveClusterId';
import { useBackendConfigStore, getEffectiveBackendBaseUrl } from '@/stores/backendConfigStore';
import type { BlastRadiusResult, GraphStatus } from '@/services/api/types';

export interface UseBlastRadiusOptions {
  kind: string;
  namespace?: string | null;
  name?: string | null;
  enabled?: boolean;
  failureMode?: string;
}

export interface UseBlastRadiusReturn {
  data: BlastRadiusResult | undefined;
  graphStatus: GraphStatus | undefined;
  isLoading: boolean;
  isFetching: boolean;
  error: Error | null;
  /** True when graph engine is available and data is being fetched */
  isGraphReady: boolean;
}

export function useBlastRadius({
  kind,
  namespace,
  name,
  enabled = true,
  failureMode,
}: UseBlastRadiusOptions): UseBlastRadiusReturn {
  const clusterId = useActiveClusterId();
  const backendBaseUrl = useBackendConfigStore((s) => s.backendBaseUrl);
  const effectiveBaseUrl = getEffectiveBackendBaseUrl(backendBaseUrl);
  const isBackendConfigured = useBackendConfigStore((s) => s.isBackendConfigured());

  const normalizedNamespace = namespace ?? '';
  const normalizedName = name ?? '';
  const baseEnabled = enabled && !!clusterId && isBackendConfigured;

  // CRITICAL fix: this used to check graph status ONCE with no polling — "if
  // ready, proceed, if not, give up" (literally, per the old comment here).
  // The backend's ClusterGraphEngine is started lazily on first request and
  // its informer sync + first debounced rebuild take real wall-clock time
  // (seconds), so the very first status check on a freshly-connected
  // cluster almost always returns ready:false. With no polling, isGraphReady
  // stayed false for the rest of the session — the blast-radius query below
  // never ran again, and the UI silently degraded to a topology-only view
  // with no error message explaining why, which is exactly what was
  // reported as "a complete mess." Poll every 2s until ready, then stop.
  const {
    data: graphStatus,
    isLoading: isStatusLoading,
  } = useQuery<GraphStatus, Error>({
    queryKey: ['blast-radius-graph-status', clusterId],
    queryFn: () => getGraphStatus(effectiveBaseUrl, clusterId!),
    enabled: baseEnabled && !!clusterId,
    staleTime: 30_000,
    refetchInterval: (query) => (query.state.data?.ready ? false : 2_000),
    retry: 1,
    retryDelay: 1_000,
  });

  const graphReady = graphStatus?.ready === true;

  // Fetch blast radius only when graph is ready
  const blastEnabled = baseEnabled && graphReady && !!kind && !!normalizedName;

  const {
    data,
    isLoading: isBlastLoading,
    isFetching,
    error: blastError,
  } = useQuery<BlastRadiusResult, Error>({
    queryKey: ['blast-radius', clusterId, kind, normalizedNamespace, normalizedName, failureMode],
    queryFn: () => getBlastRadius(effectiveBaseUrl, clusterId!, normalizedNamespace, kind, normalizedName, failureMode),
    enabled: blastEnabled,
    staleTime: 60_000,
    retry: 1,
  });

  return {
    data,
    graphStatus,
    isLoading: isStatusLoading || (graphReady && isBlastLoading),
    isFetching,
    error: blastError ?? null,
    isGraphReady: graphReady,
  };
}
