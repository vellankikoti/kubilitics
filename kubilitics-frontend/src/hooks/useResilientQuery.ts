/**
 * useResilientQuery wraps react-query with the honest-degradation contract
 * established by 10848cf and generalized in
 * docs/architecture/2026-04-24-onboarding-v2-robustness-mega.md §5.
 *
 * Four explicit states every caller gets mapped onto the same shape:
 *
 *   healthy                    → data from backend,      isReachable=true,  isStale=false
 *   backend-stale              → data from backend,      isReachable=false, isStale=true
 *   session-cache-stale        → data from ref,          isReachable=false, isStale=true
 *   no-data (first-ever miss)  → data undefined,         isReachable=false, isStale=false
 *
 * Crucially: backend-stale data is displayed verbatim but is NOT promoted
 * into the hook's session cache. Otherwise a stale-of-stale snapshot could
 * resurrect itself as "the truth" on the next local cache hit.
 */
import { useQuery, type UseQueryOptions } from '@tanstack/react-query';
import { useRef } from 'react';
import type { ResilientResponse } from '@/types/resilient';

// Plain `(prev) => prev` placeholderData hands back ANY previous queryKey's
// data while a new one loads — including a different cluster's — the same
// cluster-switch stale-data bug useKubernetes.ts's keepPreviousDataSameCluster
// was built to fix. Scope it the same way: only reuse data when the
// queryKey's clusterId slot (index 2 — ['resilient', endpoint, clusterId])
// matches the current clusterId.
function placeholderDataSameCluster(currentClusterId: unknown) {
  return (previousData: unknown, previousQuery: { queryKey: readonly unknown[] } | undefined) => {
    if (!previousQuery) return undefined;
    return previousQuery.queryKey[2] === currentClusterId ? previousData : undefined;
  };
}

export interface ResilientQueryResult<T> {
  data: T | undefined;
  isLoading: boolean;
  isReachable: boolean;
  isStale: boolean;
  errorMessage: string | null;
  refetch: () => void;
}

interface Options<T> {
  /** Cache-bust on active-cluster switch. Included in the query key. */
  clusterId?: string;
  /** Polling interval; undefined → no polling. */
  refetchInterval?: number | false;
  /** Escape hatch — merged last so callers can override staleTime etc. */
  queryOptions?: Partial<UseQueryOptions<ResilientResponse<T>>>;
  /** Gate the query off until prerequisites are in place (e.g. have an active cluster). */
  enabled?: boolean;
}

export function useResilientQuery<T>(
  endpoint: string,
  options: Options<T> = {},
): ResilientQueryResult<T> {
  // Session cache survives re-renders; re-initialises on unmount. That's
  // on purpose — we do not want stale data bleeding across cluster swaps.
  const sessionCacheRef = useRef<T | undefined>(undefined);

  const query = useQuery<ResilientResponse<T>>({
    queryKey: ['resilient', endpoint, options.clusterId ?? ''],
    queryFn: async () => {
      const r = await fetch(endpoint, { credentials: 'include' });
      if (r.status >= 500) {
        // Real bug on the server — do not mask it with "unreachable".
        throw new Error(`HTTP ${r.status}`);
      }
      return (await r.json()) as ResilientResponse<T>;
    },
    // Hold the previous key's final response across key changes so the UI
    // doesn't flash to "no data" between queries — but only within the same
    // cluster; a real cluster switch must fall through to a loading state.
    placeholderData: placeholderDataSameCluster(options.clusterId ?? ''),
    staleTime: 30_000,
    retry: 1,
    refetchInterval: options.refetchInterval,
    enabled: options.enabled ?? endpoint !== '',
    ...options.queryOptions,
  });

  const env = query.data;

  // Promote to session cache ONLY on a live-healthy response.
  if (env && env.reachable && env.data !== undefined) {
    sessionCacheRef.current = env.data;
  }

  let data: T | undefined;
  let isStale = false;
  const isReachable = env?.reachable ?? false;
  let errorMessage: string | null = null;

  if (env) {
    if (env.reachable) {
      data = env.data;
      isStale = false;
    } else if (env.stale && env.data !== undefined) {
      // Backend stale — honest display, never promote.
      data = env.data;
      isStale = true;
      errorMessage = env.error_message ?? null;
    } else if (sessionCacheRef.current !== undefined) {
      data = sessionCacheRef.current;
      isStale = true;
      errorMessage = env.error_message ?? null;
    } else {
      data = undefined;
      errorMessage = env.error_message ?? null;
    }
  }

  return {
    data,
    isLoading: query.isLoading,
    isReachable,
    isStale,
    errorMessage,
    refetch: () => {
      void query.refetch();
    },
  };
}
