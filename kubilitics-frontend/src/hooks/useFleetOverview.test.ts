/**
 * Unit tests for src/hooks/useFleetOverview.ts
 *
 * Covers: aggregation logic, empty clusters, status mapping,
 * mergeCluster helper behavior via the hook output.
 */
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { createElement } from 'react';
import type { BackendCluster, BackendClusterSummary } from '@/services/backendApiClient';

// ── Mock controls ────────────────────────────────────────────────────────────

let mockIsBackendConfigured = true;
let mockBackendBaseUrl = 'http://localhost:8190';
let mockClusters: BackendCluster[] = [];
let mockSummaries: Record<string, BackendClusterSummary> = {};

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
  getClusters: vi.fn(async () => mockClusters),
  getClusterSummary: vi.fn(async (_baseUrl: string, clusterId: string) => {
    if (mockSummaries[clusterId]) return mockSummaries[clusterId];
    throw new Error(`Cluster ${clusterId} not found`);
  }),
}));

import { useFleetOverview } from './useFleetOverview';

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
    mockClusters = [];
    mockSummaries = {};
  });

  it('returns empty aggregates when no clusters exist', async () => {
    mockClusters = [];

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
    });
  });

  it('aggregates cluster data correctly with summaries', async () => {
    mockClusters = [
      {
        id: 'c1', name: 'prod', context: 'prod-ctx',
        status: 'connected', node_count: 3,
      } as BackendCluster,
      {
        id: 'c2', name: 'staging', context: 'staging-ctx',
        status: 'connected', node_count: 2,
      } as BackendCluster,
    ];

    mockSummaries = {
      c1: {
        id: 'c1', name: 'prod', node_count: 3, namespace_count: 5,
        pod_count: 50, deployment_count: 10, service_count: 8,
        health_status: 'healthy',
      },
      c2: {
        id: 'c2', name: 'staging', node_count: 2, namespace_count: 3,
        pod_count: 20, deployment_count: 5, service_count: 3,
        health_status: 'warning',
      },
    };

    const { result } = renderHook(() => useFleetOverview(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.clusters.length).toBe(2);
    });

    // Check individual cluster mapping
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

    // Check aggregates
    expect(result.current.aggregates.totalClusters).toBe(2);
    expect(result.current.aggregates.totalNodes).toBe(5);
    expect(result.current.aggregates.totalPods).toBe(70);
    expect(result.current.aggregates.totalDeployments).toBe(15);
    expect(result.current.aggregates.healthyClusters).toBe(1);
    expect(result.current.aggregates.degradedClusters).toBe(1);
    expect(result.current.aggregates.failedClusters).toBe(0);
  });

  it('maps error/failed/disconnected cluster statuses correctly', async () => {
    mockClusters = [
      { id: 'c1', name: 'dead', context: 'ctx', status: 'disconnected' } as BackendCluster,
      { id: 'c2', name: 'ok', context: 'ctx', status: 'connected' } as BackendCluster,
      { id: 'c3', name: 'warn', context: 'ctx', status: 'degraded' } as BackendCluster,
    ];
    // No summaries — status derived from cluster.status
    mockSummaries = {};

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

  it('uses summary health_status over cluster status when available', async () => {
    mockClusters = [
      { id: 'c1', name: 'cluster', context: 'ctx', status: 'connected' } as BackendCluster,
    ];
    mockSummaries = {
      c1: {
        id: 'c1', name: 'cluster', node_count: 1, namespace_count: 1,
        pod_count: 5, deployment_count: 2, service_count: 1,
        health_status: 'critical',
      },
    };

    const { result } = renderHook(() => useFleetOverview(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.clusters.length).toBe(1);
    });

    // summary says "critical" -> should map to "error" even though cluster.status is "connected"
    expect(result.current.clusters[0].status).toBe('error');
    expect(result.current.aggregates.failedClusters).toBe(1);
  });

  it('falls back to cluster node_count when summary is unavailable', async () => {
    mockClusters = [
      { id: 'c1', name: 'cluster', context: 'ctx', status: 'connected', node_count: 7 } as BackendCluster,
    ];
    mockSummaries = {};

    const { result } = renderHook(() => useFleetOverview(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.clusters.length).toBe(1);
    });

    expect(result.current.clusters[0].nodeCount).toBe(7);
    expect(result.current.clusters[0].podCount).toBe(0);
  });

  it('does not fetch when backend is not configured', async () => {
    mockIsBackendConfigured = false;
    mockClusters = [
      { id: 'c1', name: 'cluster', context: 'ctx' } as BackendCluster,
    ];

    const { result } = renderHook(() => useFleetOverview(), {
      wrapper: createWrapper(),
    });

    // Should not attempt to load
    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.clusters).toEqual([]);
  });
});
