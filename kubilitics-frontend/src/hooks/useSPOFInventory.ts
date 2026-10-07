/**
 * Hook for fetching cluster-wide SPOF (Single Point of Failure) inventory.
 * Follows the same pattern as useBlastRadius: useActiveClusterId + backendConfigStore + react-query.
 */
import { useQuery } from '@tanstack/react-query';
import { getSPOFInventory } from '@/services/api/spof';
import type { SPOFInventory } from '@/services/api/spof';
import { useActiveClusterId } from './useActiveClusterId';
import { useBackendConfigStore, getEffectiveBackendBaseUrl } from '@/stores/backendConfigStore';
import { BackendApiError } from '@/services/api/client';

export interface UseSPOFInventoryFilters {
  namespace?: string;
  kind?: string;
  severity?: string;
}

export interface UseSPOFInventoryReturn {
  data: SPOFInventory | undefined;
  isLoading: boolean;
  isFetching: boolean;
  error: Error | null;
  /** True while the backend's dependency graph is still doing its initial
   * build (503 "Dependency graph is still building") — a transient,
   * expected state on a freshly-connected cluster, not a real failure. */
  isGraphBuilding: boolean;
  refetch: () => void;
}

/** The backend's GetSPOFInventory returns a plain 503 with this exact
 * message while the graph engine's first build is still in flight
 * (spof_handler.go). Matching on it lets the hook treat "still building"
 * as a soft, self-healing state instead of a hard failure. */
function isGraphBuildingError(error: unknown): boolean {
  return error instanceof BackendApiError
    && error.status === 503
    && error.message.toLowerCase().includes('still building');
}

export function useSPOFInventory(
  filters?: UseSPOFInventoryFilters,
): UseSPOFInventoryReturn {
  const clusterId = useActiveClusterId();
  const backendBaseUrl = useBackendConfigStore((s) => s.backendBaseUrl);
  const effectiveBaseUrl = getEffectiveBackendBaseUrl(backendBaseUrl);
  const isBackendConfigured = useBackendConfigStore((s) => s.isBackendConfigured());

  const enabled = !!clusterId && isBackendConfigured;

  const {
    data,
    isLoading,
    isFetching,
    error,
    refetch,
  } = useQuery<SPOFInventory, Error>({
    queryKey: ['spof-inventory', clusterId, filters?.namespace, filters?.kind, filters?.severity],
    queryFn: () => getSPOFInventory(effectiveBaseUrl, clusterId!, filters),
    enabled,
    staleTime: 30_000,
    // Previously a flat 60s interval regardless of state — on a freshly-
    // connected cluster the first request's 503 "still building" would sit
    // as a visible error banner for up to a minute before self-healing.
    // Poll fast (2s) specifically while building, fall back to the
    // original 60s cadence once data has loaded successfully at least once.
    refetchInterval: (query) => (isGraphBuildingError(query.state.error) ? 2_000 : 60_000),
    retry: (failureCount, err) => !isGraphBuildingError(err) && failureCount < 1,
  });

  const building = isGraphBuildingError(error);

  return {
    data,
    isLoading,
    isFetching,
    // The building state is expected and self-healing — don't surface it
    // as an error the page has to render a failure banner for.
    error: building ? null : (error ?? null),
    isGraphBuilding: building,
    refetch,
  };
}
