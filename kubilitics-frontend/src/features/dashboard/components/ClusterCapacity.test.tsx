/**
 * cluster-connection-state plan, Task 6: Cluster Capacity must show the
 * same "Cluster unreachable" message as Cluster Health for the same
 * cluster — the exact contradiction reported (Health: "Grade A, 100%"
 * next to Capacity: "No clusters connected", for one unreachable cluster).
 */
import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom/vitest';
import { TooltipProvider } from '@/components/ui/tooltip';

vi.mock('@/hooks/useActiveClusterId', () => ({
  useActiveClusterId: () => 'c1',
}));

const mockConnectionStatus = vi.fn();
vi.mock('@/hooks/useConnectionStatus', () => ({
  useConnectionStatus: () => mockConnectionStatus(),
}));

const mockOverview = vi.fn();
vi.mock('@/hooks/useClusterOverview', () => ({
  useClusterOverview: () => mockOverview(),
}));

const mockClusterUtilization = vi.fn();
vi.mock('@/hooks/useClusterUtilization', () => ({
  useClusterUtilization: () => mockClusterUtilization(),
}));

const mockK8sResourceList = vi.fn();
vi.mock('@/hooks/useKubernetes', () => ({
  useK8sResourceList: () => mockK8sResourceList(),
}));

const mockReachability = vi.fn();
vi.mock('@/hooks/useClusterReachability', () => ({
  useClusterReachability: () => mockReachability(),
}));

import { ClusterCapacity } from './ClusterCapacity';

function setConnectedDefaults() {
  mockConnectionStatus.mockReturnValue({ isConnected: true });
  mockOverview.mockReturnValue({ data: undefined });
  mockClusterUtilization.mockReturnValue({ utilization: { metricsAvailable: false, cpuTotalMillicores: 0 } });
  mockK8sResourceList.mockReturnValue({ data: undefined, isLoading: false });
  mockReachability.mockReturnValue({ status: 'connected', lastCheckedAt: 't', lastSuccessAt: 't' });
}

describe('ClusterCapacity — reachability gate (cluster-connection-state)', () => {
  it('shows the same "Cluster unreachable" message as the Health card when reachability says unreachable', () => {
    setConnectedDefaults();
    mockConnectionStatus.mockReturnValue({ isConnected: false });
    mockReachability.mockReturnValue({
      status: 'unreachable', lastCheckedAt: '2026-10-12T00:10:00Z',
      lastSuccessAt: '2026-10-12T00:05:00Z', lastError: 'dial tcp: connection refused',
    });

    render(<TooltipProvider><ClusterCapacity /></TooltipProvider>);

    expect(screen.getByText(/cluster unreachable/i)).toBeInTheDocument();
    expect(screen.queryByText(/no clusters connected/i)).not.toBeInTheDocument();
  });

  it('still shows the generic empty state when there is simply no cluster selected (not unreachable)', () => {
    setConnectedDefaults();
    mockReachability.mockReturnValue({ status: 'unknown' });

    render(<TooltipProvider><ClusterCapacity /></TooltipProvider>);

    expect(screen.queryByText(/cluster unreachable/i)).not.toBeInTheDocument();
  });
});
