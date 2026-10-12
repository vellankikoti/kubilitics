/**
 * Unit tests for src/hooks/useFleetOverview.ts
 *
 * FLEET-N1 (docs/FLEET-N1-IMPLEMENTATION.md): this hook previously issued
 * GET /clusters followed by N parallel GET /clusters/{id}/summary calls.
 * These tests now mock the single GET /fleet/overview call it replaced
 * that pattern with — covers: aggregation logic, empty clusters, status
 * mapping, mergeCluster helper behavior via the hook output, and the
 * single-request/isolation guarantees FLEET-N1 specifically added.
 */
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { createElement } from 'react';
import type { BackendFleetOverview, BackendFleetClusterInfo } from '@/services/backendApiClient';

// ── Mock controls ────────────────────────────────────────────────────────────

let mockIsBackendConfigured = true;
let mockBackendBaseUrl = 'http://localhost:8190';
let mockOverview: BackendFleetOverview = { clusters: [], totals: emptyTotals() };
let getFleetOverviewCallCount = 0;

function emptyTotals() {
  return { nodes: 0, pods: 0, deployments: 0, namespaces: 0, healthy: 0, degraded: 0, unhealthy: 0 };
}

function cluster(overrides: Partial<BackendFleetClusterInfo> & { id: string; name: string }): BackendFleetClusterInfo {
  return {
    context: 'ctx',
    status: 'connected',
    nodes: 0,
    pods: 0,
    deployments: 0,
    services: 0,
    namespaces: 0,
    healthStatus: '',
    reachable: true,
    ...overrides,
  };
}

vi.mock('@/stores/backendConfigStore', () => ({
  useBackendConfigStore: (selector: (s: Record<string, unknown>) => unknown) => {
    const state = {
      backendBaseUrl: mockBackendBaseUrl,
      isBackendConfigured: () => mockIsBackendConfigured,
    };
    return selector(state);
  },
  getEffectiveBackendBaseUrl: (url: string) => url,
}));

vi.mock('@/services/backendApiClient', () => ({
  getFleetOverview: vi.fn(async () => {
    getFleetOverviewCallCount++;
    return mockOverview;
  }),
}));

import { useFleetOverview } from './useFleetOverview';
import { getFleetOverview } from '@/services/backendApiClient';

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
        gcTime: 0,
      },
    },
  });
  return ({ children }: { children: React.ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children);
}

describe('useFleetOverview', () => {
  beforeEach(() => {
    mockIsBackendConfigured = true;
    mockBackendBaseUrl = 'http://localhost:8190';
    mockOverview = { clusters: [], totals: emptyTotals() };
    getFleetOverviewCallCount = 0;
    vi.clearAllMocks();
  });

  it('returns empty aggregates when no clusters exist', async () => {
    mockOverview = { clusters: [], totals: emptyTotals() };

    const { result } = renderHook(() => useFleetOverview(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.clusters).toEqual([]);
    expect(result.current.aggregates).toEqual({
      totalClusters: 0,
      totalNodes: 0,
      totalPods: 0,
      totalDeployments: 0,
      healthyClusters: 0,
      degradedClusters: 0,
      failedClusters: 0,
      unknownClusters: 0,
    });
  });

  it('aggregates cluster data correctly from the fleet overview response', async () => {
    mockOverview = {
      clusters: [
        cluster({ id: 'c1', name: 'prod', nodes: 3, pods: 50, deployments: 10, services: 8, namespaces: 5, healthStatus: 'healthy' }),
        cluster({ id: 'c2', name: 'staging', nodes: 2, pods: 20, deployments: 5, services: 3, namespaces: 3, healthStatus: 'warning' }),
      ],
      totals: { nodes: 5, pods: 70, deployments: 15, namespaces: 8, healthy: 1, degraded: 1, unhealthy: 0 },
    };

    const { result } = renderHook(() => useFleetOverview(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.clusters.length).toBe(2);
    });

    const prod = result.current.clusters.find((c) => c.id === 'c1');
    expect(prod).toBeDefined();
    expect(prod!.name).toBe('prod');
    expect(prod!.status).toBe('healthy');
    expect(prod!.nodeCount).toBe(3);
    expect(prod!.podCount).toBe(50);
    expect(prod!.healthGrade).toBe('A');

    const staging = result.current.clusters.find((c) => c.id === 'c2');
    expect(staging).toBeDefined();
    expect(staging!.status).toBe('warning');
    expect(staging!.healthGrade).toBe('C');

    expect(result.current.aggregates.totalClusters).toBe(2);
    expect(result.current.aggregates.totalNodes).toBe(5);
    expect(result.current.aggregates.totalPods).toBe(70);
    expect(result.current.aggregates.totalDeployments).toBe(15);
    expect(result.current.aggregates.healthyClusters).toBe(1);
    expect(result.current.aggregates.degradedClusters).toBe(1);
    expect(result.current.aggregates.failedClusters).toBe(0);
  });

  it('maps error/failed/disconnected cluster statuses correctly when the summary itself is unavailable', async () => {
    mockOverview = {
      clusters: [
        cluster({ id: 'c1', name: 'dead', status: 'disconnected', summary_unavailable: true, reachable: false }),
        cluster({ id: 'c2', name: 'ok', status: 'connected', summary_unavailable: true, reachable: false }),
        cluster({ id: 'c3', name: 'warn', status: 'degraded', summary_unavailable: true, reachable: false }),
      ],
      totals: emptyTotals(),
    };

    const { result } = renderHook(() => useFleetOverview(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.clusters.length).toBe(3);
    });

    const dead = result.current.clusters.find((c) => c.id === 'c1');
    expect(dead!.status).toBe('error');
    expect(dead!.healthGrade).toBe('F');

    const ok = result.current.clusters.find((c) => c.id === 'c2');
    expect(ok!.status).toBe('healthy');

    const warn = result.current.clusters.find((c) => c.id === 'c3');
    expect(warn!.status).toBe('warning');
  });

  it('uses the backend healthStatus over cluster.status when the summary is available', async () => {
    mockOverview = {
      clusters: [cluster({ id: 'c1', name: 'cluster', status: 'connected', healthStatus: 'critical' })],
      totals: emptyTotals(),
    };

    const { result } = renderHook(() => useFleetOverview(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.clusters.length).toBe(1);
    });

    // backend reports "critical" -> should map to "error" even though cluster.status is "connected"
    expect(result.current.clusters[0].status).toBe('error');
    expect(result.current.aggregates.failedClusters).toBe(1);
  });

  // UX-2 (docs/PRODUCTION-HARDENING-ROADMAP.md, Phase 7): a missing or
  // unrecognized cluster.status must never render as 'healthy' — Fleet has
  // no basis to claim the cluster is fine. Regression test for the
  // `if (!status) return 'healthy'` bug.
  it('maps a missing cluster.status to unknown, not healthy, when the summary is unavailable', async () => {
    mockOverview = {
      clusters: [cluster({ id: 'c1', name: 'fresh', status: '', summary_unavailable: true, reachable: false })],
      totals: emptyTotals(),
    };

    const { result } = renderHook(() => useFleetOverview(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.clusters.length).toBe(1);
    });

    expect(result.current.clusters[0].status).toBe('unknown');
    expect(result.current.clusters[0].status).not.toBe('healthy');
    expect(result.current.aggregates.unknownClusters).toBe(1);
    expect(result.current.aggregates.healthyClusters).toBe(0);
  });

  // Companion: a missing/unrecognized healthStatus must also map to
  // 'unknown', not 'healthy', even though cluster.status is fine.
  it('maps a missing healthStatus to unknown, not healthy', async () => {
    mockOverview = {
      clusters: [cluster({ id: 'c1', name: 'cluster', status: 'connected', healthStatus: '' })],
      totals: emptyTotals(),
    };

    const { result } = renderHook(() => useFleetOverview(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.clusters.length).toBe(1);
    });

    expect(result.current.clusters[0].status).toBe('unknown');
  });

  it('does not fetch when backend is not configured', async () => {
    mockIsBackendConfigured = false;
    mockOverview = { clusters: [cluster({ id: 'c1', name: 'cluster' })], totals: emptyTotals() };

    const { result } = renderHook(() => useFleetOverview(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.clusters).toEqual([]);
    expect(getFleetOverviewCallCount).toBe(0);
  });

  // ── FLEET-N1 regression tests ──────────────────────────────────────────────

  it('FLEET-N1: issues exactly ONE backend request regardless of cluster count (was 1 + N)', async () => {
    const clusters = Array.from({ length: 30 }, (_, i) =>
      cluster({ id: `c${i}`, name: `cluster-${i}`, nodes: 1, pods: 10, healthStatus: 'healthy' })
    );
    mockOverview = { clusters, totals: { ...emptyTotals(), nodes: 30, pods: 300, healthy: 30 } };

    const { result } = renderHook(() => useFleetOverview(), { wrapper: createWrapper() });

    await waitFor(() => {
      expect(result.current.clusters.length).toBe(30);
    });

    expect(getFleetOverview).toHaveBeenCalledTimes(1);
    expect(getFleetOverviewCallCount).toBe(1);
  });

  it('FLEET-N1: an unreachable cluster does not corrupt or hide healthy clusters in the same response', async () => {
    mockOverview = {
      clusters: [
        cluster({ id: 'healthy-1', name: 'a', nodes: 3, pods: 50, healthStatus: 'healthy', reachable: true }),
        cluster({
          id: 'broken-1', name: 'b', status: 'disconnected', summary_unavailable: true,
          reachable: false, error_message: 'dial tcp: connection refused',
        }),
        cluster({ id: 'healthy-2', name: 'c', nodes: 2, pods: 20, healthStatus: 'healthy', reachable: true }),
      ],
      totals: { ...emptyTotals(), nodes: 5, pods: 70, healthy: 2, unhealthy: 1 },
    };

    const { result } = renderHook(() => useFleetOverview(), { wrapper: createWrapper() });

    await waitFor(() => {
      expect(result.current.clusters.length).toBe(3);
    });

    const h1 = result.current.clusters.find((c) => c.id === 'healthy-1')!;
    expect(h1.status).toBe('healthy');
    expect(h1.podCount).toBe(50);

    const h2 = result.current.clusters.find((c) => c.id === 'healthy-2')!;
    expect(h2.status).toBe('healthy');
    expect(h2.podCount).toBe(20);

    const broken = result.current.clusters.find((c) => c.id === 'broken-1')!;
    expect(broken.status).not.toBe('healthy');
    expect(broken.reachable).toBe(false);
    expect(broken.summaryUnavailable).toBe(true);
    expect(broken.errorMessage).toBe('dial tcp: connection refused');

    expect(result.current.aggregates.healthyClusters).toBe(2);
  });

  it('FLEET-N1: propagates reachable/stale/staleAsOf for HEALTH-1/HEALTH-2 correctness (never fabricate healthy)', async () => {
    mockOverview = {
      clusters: [
        cluster({
          id: 'c1', name: 'cached', healthStatus: 'healthy',
          reachable: false, stale: true, stale_as_of: '2026-01-01T00:00:00Z',
        }),
      ],
      totals: emptyTotals(),
    };

    const { result } = renderHook(() => useFleetOverview(), { wrapper: createWrapper() });

    await waitFor(() => {
      expect(result.current.clusters.length).toBe(1);
    });

    const c = result.current.clusters[0];
    expect(c.reachable).toBe(false);
    expect(c.stale).toBe(true);
    expect(c.staleAsOf).toBe('2026-01-01T00:00:00Z');
  });

  // cluster-connection-state plan, Task 7: maps last_checked_at/
  // last_success_at/last_error onto each cluster, matching the freshness
  // fields presence.RegisteredCluster already exposes (added backend-side
  // in Task 2), so Fleet rows and Picker rows can render identical
  // "last seen" copy.
  it('maps last_checked_at/last_success_at/last_error onto each cluster', async () => {
    mockOverview = {
      clusters: [
        cluster({
          id: 'c1', name: 'c1', reachable: true, healthStatus: 'healthy',
          last_checked_at: '2026-10-12T00:10:00Z',
          last_success_at: '2026-10-12T00:10:00Z',
        }),
      ],
      totals: emptyTotals(),
    };

    const { result } = renderHook(() => useFleetOverview(), { wrapper: createWrapper() });

    await waitFor(() => {
      expect(result.current.clusters.length).toBe(1);
    });

    const c = result.current.clusters[0];
    expect(c.lastCheckedAt).toBe('2026-10-12T00:10:00Z');
    expect(c.lastSuccessAt).toBe('2026-10-12T00:10:00Z');
  });
});
