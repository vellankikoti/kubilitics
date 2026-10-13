/**
 * stabilization Task 4, Item 3 (UX-1 broader audit): HealthScoreCard is the
 * third component on the live Dashboard page found reading useHealthScore
 * directly with no reachability gating (same class of bug as
 * ClusterHealthWidget, already fixed, and DashboardHero, fixed earlier in
 * this same task).
 */
import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom/vitest';
import { TooltipProvider } from '@/components/ui/tooltip';

const mockHealthScore = vi.fn();
vi.mock('@/hooks/useHealthScore', () => ({
  useHealthScore: () => mockHealthScore(),
}));

vi.mock('@/hooks/useActiveClusterId', () => ({ useActiveClusterId: () => 'c1' }));

const mockReachability = vi.fn();
vi.mock('@/hooks/useClusterReachability', () => ({
  useClusterReachability: () => mockReachability(),
}));

import { HealthScoreCard } from './HealthScoreCard';

function setConnectedDefaults() {
  mockHealthScore.mockReturnValue({
    score: 92, grade: 'A', status: 'excellent',
    insight: 'All systems operating normally. No issues detected.',
    details: [], breakdown: { podHealth: 100, nodeHealth: 100, workloadHealth: 100, stability: 100, eventHealth: 100 },
    isLoading: false, isError: false,
  });
  mockReachability.mockReturnValue({ status: 'connected', lastCheckedAt: 't', lastSuccessAt: 't' });
}

describe('HealthScoreCard — reachability gate (stabilization Task 4)', () => {
  it('does not show a fabricated score while the cluster is unreachable, even with stale cached data present', () => {
    setConnectedDefaults();
    mockReachability.mockReturnValue({
      status: 'unreachable',
      lastCheckedAt: '2026-10-13T00:00:00Z',
      lastSuccessAt: '2026-10-12T23:00:00Z',
      lastError: 'dial tcp: connection refused',
    });

    render(<TooltipProvider><HealthScoreCard /></TooltipProvider>);

    expect(screen.getByText(/cluster unreachable/i)).toBeInTheDocument();
    expect(screen.queryByText('92')).not.toBeInTheDocument();
  });

  it('shows the score normally while the cluster is connected', () => {
    setConnectedDefaults();

    render(<TooltipProvider><HealthScoreCard /></TooltipProvider>);

    expect(screen.queryByText(/cluster unreachable/i)).not.toBeInTheDocument();
  });
});
