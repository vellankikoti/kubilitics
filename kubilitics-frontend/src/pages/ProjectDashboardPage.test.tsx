/**
 * Tests for ProjectDashboardPage — covers a real bug: when the project's
 * clusters still resolve fine (from the project resource) but the separate
 * useClustersFromBackend() metadata fetch fails, handleConnectCluster used
 * to silently no-op (it gated on `allClusters.find(...)`, which is only
 * needed for display-name enrichment, not for the actual connect action).
 * The user clicked "Connect" and nothing happened, with no error shown.
 */
import React from 'react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';

const mockSetActiveClusterBySessionId = vi.fn();
const mockRefetch = vi.fn();

let mockClustersError = false;

vi.mock('@/hooks/useProjects', () => ({
  useProject: () => ({
    data: {
      id: 'proj-1',
      name: 'Test Project',
      description: '',
      clusters: [{ cluster_id: 'cluster-1', cluster_name: 'prod-cluster', cluster_status: 'connected' }],
      namespaces: [],
    },
    isLoading: false,
    error: null,
  }),
}));

vi.mock('@/stores/projectStore', () => ({
  useProjectStore: (selector: (s: Record<string, unknown>) => unknown) =>
    selector({ setActiveProject: vi.fn(), clearActiveProject: vi.fn() }),
}));

vi.mock('@/stores/backendConfigStore', () => ({
  useBackendConfigStore: (selector: (s: Record<string, unknown>) => unknown) =>
    selector({ isBackendConfigured: () => true }),
}));

vi.mock('@/stores/clusterPresenceStore', () => ({
  setActiveClusterBySessionId: (id: string) => mockSetActiveClusterBySessionId(id),
}));

vi.mock('@/stores/demoStore', () => ({
  useDemoStore: (selector: (s: Record<string, unknown>) => unknown) => selector({ setDemo: vi.fn() }),
}));

vi.mock('@/hooks/useActiveClusterId', () => ({
  useActiveClusterId: () => null, // forces the "select a cluster" screen
}));

vi.mock('@/hooks/useClustersFromBackend', () => ({
  useClustersFromBackend: () => ({
    data: mockClustersError ? undefined : [{ id: 'cluster-1', name: 'prod-cluster' }],
    isError: mockClustersError,
    refetch: mockRefetch,
  }),
}));

vi.mock('@/features/dashboard/components/DashboardLayout', () => ({
  DashboardLayout: () => null,
}));

import ProjectDashboardPage from './ProjectDashboardPage';

function renderPage() {
  return render(
    <MemoryRouter initialEntries={['/projects/proj-1/dashboard']}>
      <Routes>
        <Route path="/projects/:projectId/dashboard" element={<ProjectDashboardPage />} />
      </Routes>
    </MemoryRouter>
  );
}

describe('ProjectDashboardPage — connect action resilience', () => {
  beforeEach(() => {
    mockSetActiveClusterBySessionId.mockClear();
    mockRefetch.mockClear();
    mockClustersError = false;
  });

  it('connects using the project-known cluster id even when cluster metadata fetch failed', () => {
    mockClustersError = true;
    renderPage();

    // Falls back to the raw cluster_id as the button label since enrichment failed.
    const connectButton = screen.getByRole('button', { name: 'cluster-1' });
    fireEvent.click(connectButton);

    expect(mockSetActiveClusterBySessionId).toHaveBeenCalledWith('cluster-1');
  });

  it('shows a retryable warning when cluster metadata fails to load', () => {
    mockClustersError = true;
    renderPage();

    expect(screen.getByText(/Couldn't load live cluster names/i)).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Retry' }));
    expect(mockRefetch).toHaveBeenCalled();
  });

  it('does not show the warning when cluster metadata loads successfully', () => {
    mockClustersError = false;
    renderPage();

    expect(screen.queryByText(/Couldn't load live cluster names/i)).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'prod-cluster' })).toBeInTheDocument();
  });
});
