/**
 * cluster-connection-state plan, Task 7: Fleet rows show the same
 * "last checked" staleness copy the Picker shows, using the
 * lastCheckedAt/lastSuccessAt/lastError fields useFleetOverview now maps
 * (backend-side in Task 2, hook-side in this task's earlier steps).
 *
 * No prior test file existed for this page — created fresh, mocking this
 * component's 4 hook/store dependencies (useFleetOverview,
 * useBackendConfigStore, useActiveCluster/clusterPresenceStore,
 * useClusterOrganizationStore).
 */
import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom/vitest';
import { MemoryRouter } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { FleetCluster, FleetAggregates } from '@/hooks/useFleetOverview';

function emptyAggregates(): FleetAggregates {
  return {
    totalClusters: 0, totalNodes: 0, totalPods: 0, totalDeployments: 0,
    healthyClusters: 0, degradedClusters: 0, failedClusters: 0, unknownClusters: 0,
  };
}

const mockFleetOverview = vi.fn();
vi.mock('@/hooks/useFleetOverview', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/hooks/useFleetOverview')>();
  return { ...actual, useFleetOverview: () => mockFleetOverview() };
});

vi.mock('@/stores/backendConfigStore', () => ({
  useBackendConfigStore: (selector: (s: Record<string, unknown>) => unknown) =>
    selector({ backendBaseUrl: 'http://localhost:8190', isBackendConfigured: () => true }),
  getEffectiveBackendBaseUrl: (url: string) => url,
}));

vi.mock('@/stores/clusterPresenceStore', () => ({
  useActiveCluster: () => null,
  setActiveClusterBySessionId: vi.fn(),
}));

vi.mock('@/stores/clusterOrganizationStore', () => ({
  useClusterOrganizationStore: (selector: (s: Record<string, unknown>) => unknown) =>
    selector({
      favorites: [],
      envTags: {},
      groups: {},
      toggleFavorite: vi.fn(),
      setEnvTag: vi.fn(),
      addToGroup: vi.fn(),
      removeFromGroup: vi.fn(),
      addGroup: vi.fn(),
      removeGroup: vi.fn(),
    }),
  ENV_DOT_COLORS: {},
  ENV_LABELS: {},
  ENV_BADGE_CLASSES: {},
  GROUP_COLORS: ['bg-blue-500'],
}));

import FleetDashboard from './FleetDashboard';

function cluster(overrides: Partial<FleetCluster> & { id: string; name: string }): FleetCluster {
  return {
    context: 'ctx',
    status: 'healthy',
    nodeCount: 1,
    podCount: 1,
    healthScore: 0,
    healthGrade: 'A',
    deploymentCount: 0,
    serviceCount: 0,
    reachable: true,
    ...overrides,
  };
}

describe('FleetDashboard — staleness copy (cluster-connection-state)', () => {
  it('shows a "last checked" timestamp on an unreachable cluster row', () => {
    mockFleetOverview.mockReturnValue({
      clusters: [
        cluster({
          id: 'c1', name: 'unreachable-cluster', status: 'error', reachable: false,
          lastCheckedAt: '2026-10-12T00:10:00Z', lastError: 'dial tcp: connection refused',
        }),
      ],
      aggregates: { ...emptyAggregates(), totalClusters: 1, failedClusters: 1 },
      isLoading: false,
      isError: false,
      error: null,
    });

    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={queryClient}>
        <MemoryRouter><FleetDashboard /></MemoryRouter>
      </QueryClientProvider>
    );

    expect(screen.getByText(/last checked/i)).toBeInTheDocument();
  });
});
