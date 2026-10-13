/**
 * stabilization Task 4, Item 3 (UX-1 broader audit): DashboardHero is
 * rendered on the live Dashboard page directly alongside ClusterHealthWidget
 * (already fixed in an earlier task for this exact bug class) — it shows a
 * "Grade A" style score/status with zero reachability gating and never
 * checks the isLoading/isError signals useHealthScore already exposes.
 */
import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom/vitest';

const mockHealthScore = vi.fn();
vi.mock('@/hooks/useHealthScore', () => ({
  useHealthScore: () => mockHealthScore(),
}));

const mockLiveSignals = vi.fn();
vi.mock('@/hooks/useLiveSignals', () => ({
  useLiveSignals: () => mockLiveSignals(),
}));

vi.mock('@/hooks/useActiveClusterId', () => ({ useActiveClusterId: () => 'c1' }));

const mockReachability = vi.fn();
vi.mock('@/hooks/useClusterReachability', () => ({
  useClusterReachability: () => mockReachability(),
}));

import { DashboardHero } from './DashboardHero';

function setConnectedDefaults() {
  mockHealthScore.mockReturnValue({
    score: 92, grade: 'A', status: 'excellent',
    insight: 'All systems operating normally. No issues detected.',
    isLoading: false, isError: false,
  });
  mockLiveSignals.mockReturnValue({
    runningPods: 10, failedPods: 0, pendingPods: 0, totalNodes: 3, activeAlerts: 0, podRestarts: 0, isLoading: false,
  });
  mockReachability.mockReturnValue({ status: 'connected', lastCheckedAt: 't', lastSuccessAt: 't' });
}

describe('DashboardHero — reachability gate (stabilization Task 4)', () => {
  it('does not show a fabricated "Grade A" score while the cluster is unreachable, even with stale cached data present', () => {
    setConnectedDefaults();
    mockReachability.mockReturnValue({
      status: 'unreachable',
      lastCheckedAt: '2026-10-13T00:00:00Z',
      lastSuccessAt: '2026-10-12T23:00:00Z',
      lastError: 'dial tcp: connection refused',
    });

    render(<DashboardHero />);

    expect(screen.getByText(/cluster unreachable/i)).toBeInTheDocument();
    expect(screen.queryByText('92')).not.toBeInTheDocument();
    expect(screen.queryByText(/grade a/i)).not.toBeInTheDocument();
  });

  it('shows the score normally while the cluster is connected', () => {
    setConnectedDefaults();

    render(<DashboardHero />);

    expect(screen.queryByText(/cluster unreachable/i)).not.toBeInTheDocument();
    expect(screen.getByText('92')).toBeInTheDocument();
  });
});
