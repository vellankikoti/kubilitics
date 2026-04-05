/**
 * React Query hooks for log persistence & cross-pod search.
 * Pattern: matches useEventsIntelligence.ts
 */
import { useQuery } from '@tanstack/react-query';
import { useActiveClusterId } from './useActiveClusterId';
import { useBackendConfigStore, getEffectiveBackendBaseUrl } from '@/stores/backendConfigStore';
import {
  searchLogs,
  aggregateLogs,
} from '@/services/api/eventsIntelligence';
import type {
  StoredLog,
  LogSearchParams,
} from '@/services/api/eventsIntelligence';

function useBackendContext() {
  const clusterId = useActiveClusterId();
  const backendBaseUrl = useBackendConfigStore((s) => s.backendBaseUrl);
  const effectiveBaseUrl = getEffectiveBackendBaseUrl(backendBaseUrl);
  const isBackendConfigured = useBackendConfigStore((s) => s.isBackendConfigured());
  const enabled = !!clusterId && isBackendConfigured;
  return { clusterId, effectiveBaseUrl, enabled };
}

/**
 * Search stored logs with flexible filters.
 * Supports pod-level, namespace-level, owner-level, and full-text search.
 */
export function useLogSearch(params: LogSearchParams, options?: { enabled?: boolean }) {
  const { clusterId, effectiveBaseUrl, enabled } = useBackendContext();

  return useQuery<StoredLog[], Error>({
    queryKey: ['logs', 'search', clusterId, params],
    queryFn: () => searchLogs(effectiveBaseUrl, clusterId!, params),
    enabled: enabled && (options?.enabled !== false),
    staleTime: 15_000,
    refetchInterval: 30_000,
  });
}

/**
 * Aggregate logs across all pods of a workload (Deployment, StatefulSet, etc.).
 * Powers the "show me all ERROR logs across all checkout-api pods" use case.
 */
export function useAggregatedLogs(
  ownerKind: string,
  ownerName: string,
  namespace: string,
  options?: { level?: string; from?: number; to?: number; limit?: number; enabled?: boolean },
) {
  const { clusterId, effectiveBaseUrl, enabled } = useBackendContext();

  return useQuery<StoredLog[], Error>({
    queryKey: ['logs', 'aggregate', clusterId, ownerKind, ownerName, namespace, options],
    queryFn: () =>
      aggregateLogs(effectiveBaseUrl, clusterId!, ownerKind, ownerName, namespace, {
        level: options?.level,
        from: options?.from,
        to: options?.to,
        limit: options?.limit,
      }),
    enabled: enabled && !!ownerKind && !!ownerName && (options?.enabled !== false),
    staleTime: 15_000,
    refetchInterval: 30_000,
  });
}

// Re-export types for convenience
export type { StoredLog, LogSearchParams } from '@/services/api/eventsIntelligence';
