/**
 * Task 13 (final crash sweep): `useHealthScore` casts `overviewQuery.data.health`
 * via `as unknown as {...}` and immediately reads `.score`/`.grade`/`.status`/
 * `.breakdown` with no guard. `ClusterOverview.health` is typed required, so
 * tsc never flags this — same blind spot as the diagnoseWorkload P0 already
 * fixed this session. This proves whether a real-shaped backend `/overview`
 * response that omits `health` (plausible for a degraded/partial/stale
 * snapshot — nothing in the fetch path validates the field is present)
 * crashes the Dashboard's health score, one of the first things every user
 * sees.
 */
import { describe, it, expect, vi } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import React from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

vi.mock('@/stores/clusterPresenceStore', () => ({
  useActiveCluster: () => ({ id: 'test-cluster-id', name: 'test-cluster', serverUrl: 'https://test', provider: '' }),
}));

vi.mock('@/stores/backendConfigStore', () => ({
  useBackendConfigStore: (selector?: (s: Record<string, unknown>) => unknown) => {
    const state: Record<string, unknown> = { backendBaseUrl: 'http://localhost:8190', isBackendConfigured: () => true };
    return selector ? selector(state) : state;
  },
  getEffectiveBackendBaseUrl: () => 'http://localhost:8190',
}));

vi.mock('@/hooks/useActiveClusterId', () => ({ useActiveClusterId: () => 'test-cluster-id' }));

vi.mock('@/hooks/useClusterSummary', () => ({
  useClusterSummary: () => ({ data: undefined, isLoading: false, isError: false }),
}));

vi.mock('@/hooks/useKubernetes', () => ({
  useK8sResourceList: () => ({ data: undefined, isLoading: false, isError: false }),
}));

vi.mock('@/services/backendApiClient', () => ({
  getEvents: vi.fn(async () => ({ items: [] })),
}));

// Real-shaped backend overview response, missing only `health` — exactly
// the kind of partial/degraded 200 response a backend can plausibly send
// without the frontend's fetch layer rejecting it (no schema validation
// on the response body).
const overviewMissingHealth = {
  counts: { nodes: 1, pods: 2, namespaces: 1, deployments: 1 },
  pod_status: { running: 2, pending: 0, failed: 0, succeeded: 0 },
  alerts: { warnings: 0, critical: 0, top_3: [] },
};

vi.mock('@/hooks/useClusterOverview', () => ({
  useClusterOverview: () => ({ data: overviewMissingHealth, isLoading: false, isError: false }),
}));

function createWrapper() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } });
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  };
}

describe('useHealthScore — crash risk from a backend overview response missing `health`', () => {
  it('does not throw when overviewQuery.data has no `health` field', async () => {
    const { useHealthScore } = await import('./useHealthScore');
    const { result } = renderHook(() => useHealthScore(), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current).toBeDefined());
    // The real assertion is simply that rendering got here without throwing.
    expect(result.current.score).toBeDefined();
  });
});
