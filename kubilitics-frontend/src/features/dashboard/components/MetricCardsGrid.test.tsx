/**
 * Tests for MetricCardsGrid component.
 *
 * Verifies the 3x3 tile grid renders correct category labels, tile titles,
 * counts, and K8s icons.
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, cleanup } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import React from 'react';

/* ── Mocks ─────────────────────────────────────────────────────────────────── */

// Mock useResourceCounts — provide counts for all 9 tile types
const mockCounts: Record<string, number> = {
  nodes: 3,
  pods: 42,
  deployments: 12,
  services: 8,
  daemonsets: 2,
  namespaces: 5,
  configmaps: 20,
  secrets: 15,
  cronjobs: 4,
};

vi.mock('@/hooks/useResourceCounts', () => ({
  useResourceCounts: () => ({ counts: mockCounts }),
}));

// Mock useDashboardResourceHealth — return empty health for simplicity
vi.mock('@/hooks/useDashboardResourceHealth', () => ({
  useDashboardResourceHealth: () => ({ health: {}, isLoading: false }),
}));

// Mock useProjectStore — no active project
vi.mock('@/stores/projectStore', () => ({
  useProjectStore: (selector: (s: Record<string, unknown>) => unknown) => selector({ activeProject: null }),
}));

vi.mock('@/hooks/useActiveClusterId', () => ({ useActiveClusterId: () => 'c1' }));

const mockReachability = vi.fn();
vi.mock('@/hooks/useClusterReachability', () => ({
  useClusterReachability: () => mockReachability(),
}));

// Mock k8sIconMap — return a placeholder URL for every kind
vi.mock('@/topology/icons/k8sIconMap', () => ({
  default: new Proxy(
    {},
    { get: (_target, prop) => `/icons/${String(prop)}.svg` },
  ),
}));

// Mock cn utility
vi.mock('@/lib/utils', () => ({
  cn: (...args: unknown[]) => args.filter(Boolean).join(' '),
}));

import { MetricCardsGrid } from './MetricCardsGrid';

/* ── Helpers ───────────────────────────────────────────────────────────────── */

function renderGrid() {
  return render(
    <MemoryRouter>
      <MetricCardsGrid />
    </MemoryRouter>,
  );
}

/* ── Tests ──────────────────────────────────────────────────────────────────── */

describe('MetricCardsGrid', () => {
  afterEach(() => {
    cleanup();
  });

  beforeEach(() => {
    mockReachability.mockReturnValue({ status: 'connected', lastCheckedAt: 't', lastSuccessAt: 't' });
  });

  it('renders 3 category labels', () => {
    renderGrid();

    expect(screen.getByText('Infrastructure')).toBeTruthy();
    expect(screen.getByText('Networking')).toBeTruthy();
    expect(screen.getByText('Configuration')).toBeTruthy();
  });

  it('renders all 9 resource tile titles', () => {
    renderGrid();

    const expectedTitles = [
      'Nodes', 'Pods', 'Deployments',
      'Services', 'DaemonSets', 'Namespaces',
      'ConfigMaps', 'Secrets', 'CronJobs',
    ];
    for (const title of expectedTitles) {
      // Use getAllByText since accessible names on <a> links also contain the title text
      const matches = screen.getAllByText(title);
      expect(matches.length).toBeGreaterThanOrEqual(1);
    }
  });

  it('renders count values from useResourceCounts', () => {
    renderGrid();

    // Each count should appear in the document (use getAllByText to tolerate duplicates)
    for (const count of ['3', '42', '12', '8', '2', '5', '20', '15', '4']) {
      const matches = screen.getAllByText(count);
      expect(matches.length).toBeGreaterThanOrEqual(1);
    }
  });

  it('renders K8s icon images for each tile', () => {
    const { container } = renderGrid();

    const imgs = container.querySelectorAll('img');
    // There should be 9 icon images (one per tile)
    expect(imgs.length).toBe(9);

    // Each image src should point to our mocked icon path
    const srcs = Array.from(imgs).map((img) => img.getAttribute('src'));
    expect(srcs).toContain('/icons/node.svg');
    expect(srcs).toContain('/icons/pod.svg');
    expect(srcs).toContain('/icons/deployment.svg');
    expect(srcs).toContain('/icons/service.svg');
    expect(srcs).toContain('/icons/daemonset.svg');
    expect(srcs).toContain('/icons/namespace.svg');
    expect(srcs).toContain('/icons/configmap.svg');
    expect(srcs).toContain('/icons/secret.svg');
    expect(srcs).toContain('/icons/cronjob.svg');
  });

  it('renders 9 clickable links', () => {
    renderGrid();

    const links = screen.getAllByRole('link');
    expect(links).toHaveLength(9);

    // Check that href values map to the correct resource routes
    const hrefs = links.map((a) => (a as HTMLAnchorElement).getAttribute('href'));
    expect(hrefs).toContain('/nodes');
    expect(hrefs).toContain('/pods');
    expect(hrefs).toContain('/deployments');
    expect(hrefs).toContain('/services');
    expect(hrefs).toContain('/daemonsets');
    expect(hrefs).toContain('/namespaces');
    expect(hrefs).toContain('/configmaps');
    expect(hrefs).toContain('/secrets');
    expect(hrefs).toContain('/cronjobs');
  });

  it('shows "No data" legend when health data is empty', () => {
    renderGrid();

    // With empty health, each tile should show "No data"
    const noDataElements = screen.getAllByText('No data');
    expect(noDataElements).toHaveLength(9);
  });

  // stabilization Task 5: useResourceCounts/useDashboardResourceHealth both
  // disable their queries (enabled: isConnected) when the cluster is
  // unreachable, rather than erroring — so every tile silently fell back to
  // a bare "0" with no indication this is "couldn't check" rather than "the
  // cluster genuinely has zero of this resource." Same bug class as
  // ClusterHealthWidget/ClusterCapacity/DashboardHero, different manifestation.
  it('shows a clear unreachable message instead of fabricated zero counts when the cluster is unreachable', () => {
    mockReachability.mockReturnValue({
      status: 'unreachable',
      lastCheckedAt: '2026-10-13T00:00:00Z',
      lastSuccessAt: '2026-10-12T23:00:00Z',
      lastError: 'dial tcp: connection refused',
    });

    renderGrid();

    expect(screen.getByText(/cluster unreachable/i)).toBeInTheDocument();
    // None of the tile titles (and therefore none of the misleading "0"
    // counts) should render while unreachable.
    expect(screen.queryByText('Nodes')).not.toBeInTheDocument();
    expect(screen.queryByText('Pods')).not.toBeInTheDocument();
  });

  it('still renders the tile grid normally when connected', () => {
    renderGrid();

    expect(screen.queryByText(/cluster unreachable/i)).not.toBeInTheDocument();
    expect(screen.getByText('Nodes')).toBeInTheDocument();
  });
});
