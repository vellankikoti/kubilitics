/**
 * cluster-connection-state plan, Task 6 Step 5 — the exact bug reported
 * via screenshot: Dashboard's Cluster Health card showed "Grade A, 100%,
 * Good State" while Cluster Capacity showed "No clusters connected", for
 * the same unreachable cluster, on the same screen. Both cards now read
 * useClusterReachability as their single input — this locks them to never
 * disagree again.
 */
import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom/vitest';
import { TooltipProvider } from '@/components/ui/tooltip';

vi.mock('@/stores/clusterPresenceStore', () => ({
  useActiveCluster: () => ({ id: 'c1', name: 'test-cluster' }),
}));
vi.mock('@/hooks/useActiveClusterId', () => ({
  useActiveClusterId: () => 'c1',
}));
vi.mock('@/stores/backendConfigStore', () => ({
  useBackendConfigStore: (selector: (s: Record<string, unknown>) => unknown) =>
    selector({ isBackendConfigured: () => true }),
}));
vi.mock('@/hooks/useClusterOverview', () => ({
  useClusterOverview: () => ({ data: undefined, isLoading: false, isError: false }),
}));
vi.mock('@/hooks/useHealthScore', () => ({
  useHealthScore: () => ({
    score: 100, grade: 'A', status: 'excellent',
    breakdown: { podHealth: 100, nodeHealth: 100, workloadHealth: 100, stability: 100, eventHealth: 100 },
    details: [], insight: '', isLoading: false, isError: false,
  }),
}));
vi.mock('@/hooks/useConnectionStatus', () => ({
  useConnectionStatus: () => ({ isConnected: false }),
}));
vi.mock('@/hooks/useClusterUtilization', () => ({
  useClusterUtilization: () => ({ utilization: { metricsAvailable: false, cpuTotalMillicores: 0 } }),
}));
vi.mock('@/hooks/useKubernetes', () => ({
  useK8sResourceList: () => ({ data: undefined, isLoading: false }),
}));

const mockReachability = vi.fn();
vi.mock('@/hooks/useClusterReachability', () => ({
  useClusterReachability: () => mockReachability(),
}));

import { ClusterHealthWidget } from '../ClusterHealthWidget';
import { ClusterCapacity } from '../ClusterCapacity';

describe('Health and Capacity cards on an unreachable cluster', () => {
  it('both show "Cluster unreachable" — never one healthy and one empty', () => {
    mockReachability.mockReturnValue({
      status: 'unreachable', lastCheckedAt: '2026-10-12T00:10:00Z',
      lastSuccessAt: '2026-10-12T00:05:00Z', lastError: 'dial tcp: connection refused',
    });

    const { unmount } = render(<TooltipProvider><ClusterHealthWidget /></TooltipProvider>);
    expect(screen.getByText(/cluster unreachable/i)).toBeInTheDocument();
    unmount();

    render(<TooltipProvider><ClusterCapacity /></TooltipProvider>);
    expect(screen.getByText(/cluster unreachable/i)).toBeInTheDocument();
  });
});
