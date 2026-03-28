/**
 * Tests for MultiPodLogViewer — pod selector rendering, empty state, streaming indicator.
 */
import React from 'react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MultiPodLogViewer, type PodTarget } from './MultiPodLogViewer';

// ── Mocks ───────────────────────────────────────────────────────────────────

vi.mock('@/hooks/useConnectionStatus', () => ({
  useConnectionStatus: () => ({ isConnected: true }),
}));

vi.mock('@/hooks/useTheme', () => ({
  useTheme: () => ({ isDark: true, theme: 'dark' }),
}));

vi.mock('@/stores/backendConfigStore', () => ({
  useBackendConfigStore: (selector: (s: Record<string, unknown>) => unknown) => {
    const state = {
      backendBaseUrl: 'http://localhost:8190',
      isBackendConfigured: () => true,
    };
    return selector(state);
  },
  getEffectiveBackendBaseUrl: () => 'http://localhost:8190',
}));

vi.mock('@/hooks/useActiveClusterId', () => ({
  useActiveClusterId: () => 'test-cluster',
}));

vi.mock('@/services/backendApiClient', () => ({
  getPodLogsUrl: () => 'http://localhost:8190/api/pods/logs',
}));

vi.mock('@/components/ui/sonner', () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

vi.mock('@tanstack/react-virtual', () => ({
  useVirtualizer: () => ({
    getTotalSize: () => 0,
    getVirtualItems: () => [],
    scrollToIndex: vi.fn(),
  }),
}));

// Mock fetch to prevent network calls
const mockFetch = vi.fn().mockResolvedValue({
  ok: false,
  status: 500,
  body: null,
});
vi.stubGlobal('fetch', mockFetch);

// Suppress console.warn from the stream failure logs
vi.spyOn(console, 'warn').mockImplementation(() => {});

// ── Tests ───────────────────────────────────────────────────────────────────

const testPods: PodTarget[] = [
  { name: 'my-app-abc-x1y2z', namespace: 'default', containers: ['main'] },
  { name: 'my-app-abc-a3b4c', namespace: 'default', containers: ['main'] },
  { name: 'my-worker-def-m5n6', namespace: 'production', containers: ['worker'] },
];

describe('MultiPodLogViewer', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders pod selector with pod names', () => {
    render(<MultiPodLogViewer pods={testPods} />);

    // The component renders short pod names (last segment after dash)
    expect(screen.getByText('x1y2z')).toBeDefined();
    expect(screen.getByText('a3b4c')).toBeDefined();
    expect(screen.getByText('m5n6')).toBeDefined();
  });

  it('shows pod count badge', () => {
    render(<MultiPodLogViewer pods={testPods} />);

    // The pod sources section shows "3/3 pods" — may appear multiple times
    const badges = screen.getAllByText('3/3 pods');
    expect(badges.length).toBeGreaterThanOrEqual(1);
  });

  it('shows "Waiting for log data" when no logs are present', () => {
    render(<MultiPodLogViewer pods={testPods} />);

    // When allLogs is empty and pods are provided, it shows the waiting message
    const waitingElements = screen.getAllByText(/Waiting for log data/);
    expect(waitingElements.length).toBeGreaterThanOrEqual(1);
  });

  it('renders streaming indicator when connected and streaming', () => {
    render(<MultiPodLogViewer pods={testPods} />);

    // The streaming badge shows "Streaming" text (may appear multiple times)
    const streamingElements = screen.getAllByText('Streaming');
    expect(streamingElements.length).toBeGreaterThanOrEqual(1);
  });

  it('shows "No pods selected for streaming" when pods array is empty', () => {
    render(<MultiPodLogViewer pods={[]} />);

    expect(screen.getByText('No pods selected for streaming')).toBeDefined();
  });

  it('renders Select All / Deselect All button in pod picker', () => {
    render(<MultiPodLogViewer pods={testPods} />);

    // When all pods are selected (initial state), the button says "Deselect All"
    const deselectBtns = screen.getAllByText('Deselect All');
    expect(deselectBtns.length).toBeGreaterThanOrEqual(1);
  });

  it('renders the Pod Sources label', () => {
    render(<MultiPodLogViewer pods={testPods} />);

    const labels = screen.getAllByText('Pod Sources');
    expect(labels.length).toBeGreaterThanOrEqual(1);
  });

  it('renders level filter pills', () => {
    render(<MultiPodLogViewer pods={testPods} />);

    // Each pill text may appear multiple times in the DOM
    expect(screen.getAllByText('All').length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText('Info').length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText('Warn').length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText('Error').length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText('Debug').length).toBeGreaterThanOrEqual(1);
  });
});
