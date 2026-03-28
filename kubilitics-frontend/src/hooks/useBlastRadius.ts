/**
 * Hook for fetching blast radius analysis from backend API.
 * Uses React Query with 60s staleTime. Falls back gracefully if API unavailable.
 */
import { useQuery } from '@tanstack/react-query';
import { getBlastRadius } from '@/services/backendApiClient';
import { useActiveClusterId } from './useActiveClusterId';
import { useBackendConfigStore, getEffectiveBackendBaseUrl } from '@/stores/backendConfigStore';
import type { BlastRadiusResult } from '@/services/api/types';

export interface UseBlastRadiusOptions {
  kind: string;
  namespace?: string | null;
  name?: string | null;
  enabled?: boolean;
}

export interface UseBlastRadiusReturn {
  data: BlastRadiusResult | undefined;
  isLoading: boolean;
  isFetching: boolean;
  error: Error | null;
  /** True when the API returned a 404 — the endpoint does not exist yet. */
  isUnavailable: boolean;
}

/**
 * Fetches blast radius analysis for a specific resource.
 * Falls back gracefully: if the API returns 404 the hook surfaces
 * `isUnavailable = true` so the component can keep using topology-derived data.
 */
export function useBlastRadius({
  kind,
  namespace,
  name,
  enabled = true,
}: UseBlastRadiusOptions): UseBlastRadiusReturn {
  const clusterId = useActiveClusterId();
  const backendBaseUrl = useBackendConfigStore((s) => s.backendBaseUrl);
  const effectiveBaseUrl = getEffectiveBackendBaseUrl(backendBaseUrl);
  const isBackendConfigured = useBackendConfigStore((s) => s.isBackendConfigured());

  const normalizedNamespace = namespace ?? '';
  const normalizedName = name ?? '';

  const queryEnabled =
    enabled &&
    !!clusterId &&
    isBackendConfigured &&
    !!kind &&
    !!normalizedName;

  const {
    data,
    isLoading,
    isFetching,
    error,
  } = useQuery<BlastRadiusResult, Error>({
    queryKey: ['blast-radius', clusterId, kind, normalizedNamespace, normalizedName],
    queryFn: async () => {
      if (!clusterId) throw new Error('Cluster not selected');
      if (!normalizedName) throw new Error('Resource name is required');
      return getBlastRadius(
        effectiveBaseUrl,
        clusterId,
        normalizedNamespace,
        kind,
        normalizedName,
      );
    },
    enabled: queryEnabled,
    staleTime: 60_000,
    retry: (failureCount, err) => {
      // Don't retry on 404 — the endpoint doesn't exist
      if (err && 'status' in err && (err as { status: number }).status === 404) return false;
      return failureCount < 2;
    },
    retryDelay: 1000,
  });

  const isUnavailable =
    !!error &&
    'status' in error &&
    (error as { status: number }).status === 404;

  return {
    data,
    isLoading,
    isFetching,
    error: isUnavailable ? null : (error ?? null),
    isUnavailable,
  };
}
