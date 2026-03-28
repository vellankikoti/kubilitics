/**
 * Tests for useClusterWatcher — verifies notification behavior on state changes.
 *
 * Strategy: mock all external deps. Control data via a stateful mock of useQuery
 * that supports triggering re-renders with new data. The hook uses useEffect([data]),
 * so we need React to see a genuine data reference change between renders.
 */
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import React from 'react';

// ── Mock controls ───────────────────────────────────────────────────────────

const mockAddNotification = vi.fn();
let mockClusterId: string | null = 'test-cluster';

// State that tracks query data and a setState function to trigger re-renders
let setQueryData: ((d: unknown) => void) | null = null;

vi.mock('@/hooks/useActiveClusterId', () => ({
  useActiveClusterId: () => mockClusterId,
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

vi.mock('@/stores/notificationStore', () => ({
  useNotificationStore: (selector: (s: Record<string, unknown>) => unknown) => {
    const state = { addNotification: mockAddNotification };
    return selector(state);
  },
}));

vi.mock('@/services/backendApiClient', () => ({
  listResources: vi.fn().mockResolvedValue({ items: [] }),
}));

// Mock useQuery with useState so that setting data triggers a React re-render
// and the useEffect in useClusterWatcher fires with the new data reference.
vi.mock('@tanstack/react-query', () => ({
  useQuery: () => {
    const [data, _setData] = React.useState<unknown>(null);
    // Expose the setter so the test can trigger data changes
    setQueryData = _setData;
    return { data };
  },
}));

import { useClusterWatcher } from './useClusterWatcher';

// ── Helpers ─────────────────────────────────────────────────────────────────

function makePod(name: string, ns: string, waitingReason?: string, terminatedReason?: string) {
  const cs: { state: { waiting?: { reason: string }; terminated?: { reason: string } } }[] = [];
  if (waitingReason || terminatedReason) {
    const s: { state: { waiting?: { reason: string }; terminated?: { reason: string } } } = { state: {} };
    if (waitingReason) s.state.waiting = { reason: waitingReason };
    if (terminatedReason) s.state.terminated = { reason: terminatedReason };
    cs.push(s);
  }
  return {
    metadata: { name, namespace: ns },
    status: { phase: 'Running', containerStatuses: cs.length ? cs : undefined },
  };
}

function makeDeployment(name: string, ns: string, desired: number, available: number, unavailable: number) {
  return {
    metadata: { name, namespace: ns },
    spec: { replicas: desired },
    status: { availableReplicas: available, unavailableReplicas: unavailable },
  };
}

function makeData(overrides: { pods?: Record<string, unknown>[]; deployments?: Record<string, unknown>[]; nodes?: Record<string, unknown>[]; hpas?: Record<string, unknown>[] } = {}) {
  return {
    pods: overrides.pods ?? [],
    deployments: overrides.deployments ?? [],
    nodes: overrides.nodes ?? [],
    hpas: overrides.hpas ?? [],
  };
}

// ── Tests ───────────────────────────────────────────────────────────────────

describe('useClusterWatcher', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockClusterId = 'test-cluster';
    setQueryData = null;
  });

  it('does not emit notifications on first fetch', () => {
    renderHook(() => useClusterWatcher());

    // Simulate first query result with a problem pod
    act(() => {
      setQueryData!(makeData({
        pods: [makePod('crash-pod', 'default', 'CrashLoopBackOff')],
      }));
    });

    // First fetch should NOT trigger notifications
    expect(mockAddNotification).not.toHaveBeenCalled();
  });

  it('emits notification when pod crashes on second fetch (state change)', () => {
    renderHook(() => useClusterWatcher());

    // First poll: healthy pod
    act(() => {
      setQueryData!(makeData({
        pods: [makePod('my-pod', 'default')],
      }));
    });

    expect(mockAddNotification).not.toHaveBeenCalled();

    // Second poll: pod enters CrashLoopBackOff
    act(() => {
      setQueryData!(makeData({
        pods: [makePod('my-pod', 'default', 'CrashLoopBackOff')],
      }));
    });

    expect(mockAddNotification).toHaveBeenCalledTimes(1);
    expect(mockAddNotification).toHaveBeenCalledWith(
      expect.objectContaining({
        title: 'Pod CrashLoopBackOff',
        severity: 'error',
        category: 'cluster',
        resourceKind: 'Pod',
        resourceName: 'my-pod',
        namespace: 'default',
      }),
    );
  });

  it('does not re-emit for the same crash (dedup)', () => {
    renderHook(() => useClusterWatcher());

    // First poll: healthy
    act(() => {
      setQueryData!(makeData({
        pods: [makePod('my-pod', 'default')],
      }));
    });

    // Second poll: crash
    act(() => {
      setQueryData!(makeData({
        pods: [makePod('my-pod', 'default', 'CrashLoopBackOff')],
      }));
    });
    expect(mockAddNotification).toHaveBeenCalledTimes(1);

    // Third poll: same CrashLoopBackOff — no new notification
    act(() => {
      setQueryData!(makeData({
        pods: [makePod('my-pod', 'default', 'CrashLoopBackOff')],
      }));
    });

    expect(mockAddNotification).toHaveBeenCalledTimes(1);
  });

  it('emits notification when pod transitions to a different problem state', () => {
    renderHook(() => useClusterWatcher());

    // First poll: healthy
    act(() => {
      setQueryData!(makeData({ pods: [makePod('my-pod', 'default')] }));
    });

    // CrashLoopBackOff
    act(() => {
      setQueryData!(makeData({ pods: [makePod('my-pod', 'default', 'CrashLoopBackOff')] }));
    });
    expect(mockAddNotification).toHaveBeenCalledTimes(1);

    // OOMKilled (different fingerprint)
    act(() => {
      setQueryData!(makeData({ pods: [makePod('my-pod', 'default', undefined, 'OOMKilled')] }));
    });
    expect(mockAddNotification).toHaveBeenCalledTimes(2);
    expect(mockAddNotification).toHaveBeenLastCalledWith(
      expect.objectContaining({ title: 'Pod OOMKilled' }),
    );
  });

  it('emits notification for deployment degradation', () => {
    renderHook(() => useClusterWatcher());

    // First poll: healthy deployment
    act(() => {
      setQueryData!(makeData({
        deployments: [makeDeployment('my-deploy', 'default', 3, 3, 0)],
      }));
    });

    // Second poll: degraded
    act(() => {
      setQueryData!(makeData({
        deployments: [makeDeployment('my-deploy', 'default', 3, 1, 2)],
      }));
    });

    expect(mockAddNotification).toHaveBeenCalledTimes(1);
    expect(mockAddNotification).toHaveBeenCalledWith(
      expect.objectContaining({
        title: 'Deployment degraded',
        severity: 'warning',
        resourceKind: 'Deployment',
        resourceName: 'my-deploy',
      }),
    );
  });
});
