/**
 * stabilization Task 4, Item 3 (UX-1 broader audit): the Health Dashboard
 * score/grade must not keep showing a stale, high-confidence score after
 * the cluster becomes unreachable mid-session — useClusterHealth has no
 * reachability awareness at all (no refetchInterval, no connection check),
 * so without this gate the last successfully-fetched score/level renders
 * forever with no staleness indication, exactly the class of bug fixed for
 * the Dashboard's ClusterHealthWidget in an earlier task.
 */
import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom/vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

vi.mock('react-router-dom', () => ({ useNavigate: () => vi.fn() }));
vi.mock('@/hooks/useActiveClusterId', () => ({ useActiveClusterId: () => 'c1' }));
vi.mock('@/stores/clusterPresenceStore', () => ({
  useActiveCluster: () => ({ id: 'c1', name: 'c1' }),
}));
vi.mock('@/hooks/useAIContext', () => ({ useAIContext: () => {} }));
vi.mock('@/hooks/useEventsIntelligence', () => ({
  useActiveInsights: () => ({ data: [] }),
  useDismissInsight: () => ({ mutate: vi.fn(), isPending: false }),
  useRecentChanges: () => ({ data: [], isLoading: false }),
}));

const mockClusterHealth = vi.fn();
vi.mock('@/hooks/useClusterHealth', () => ({
  useClusterHealth: () => mockClusterHealth(),
}));

const mockReachability = vi.fn();
vi.mock('@/hooks/useClusterReachability', () => ({
  useClusterReachability: () => mockReachability(),
}));

import HealthDashboard from './HealthDashboard';

function renderWithClient() {
  const qc = new QueryClient();
  return render(
    <QueryClientProvider client={qc}>
      <HealthDashboard />
    </QueryClientProvider>,
  );
}

describe('HealthDashboard — reachability gate (stabilization Task 4)', () => {
  it('does not show a healthy-looking score while the cluster is unreachable, even with stale cached data present', () => {
    mockClusterHealth.mockReturnValue({
      data: { score: 92, level: 'healthy', components: [] },
      isLoading: false,
      error: null,
    });
    mockReachability.mockReturnValue({
      status: 'unreachable',
      lastCheckedAt: '2026-10-13T00:00:00Z',
      lastSuccessAt: '2026-10-12T23:00:00Z',
      lastError: 'dial tcp: connection refused',
    });

    renderWithClient();

    expect(screen.getByText(/cluster unreachable/i)).toBeInTheDocument();
    // The stale score must not render as if it were current/trustworthy.
    expect(screen.queryByText('92')).not.toBeInTheDocument();
  });

  it('shows the score normally while the cluster is connected', () => {
    mockClusterHealth.mockReturnValue({
      data: { score: 92, level: 'healthy', components: [] },
      isLoading: false,
      error: null,
    });
    mockReachability.mockReturnValue({ status: 'connected', lastCheckedAt: 't', lastSuccessAt: 't' });

    renderWithClient();

    expect(screen.queryByText(/cluster unreachable/i)).not.toBeInTheDocument();
  });
});
