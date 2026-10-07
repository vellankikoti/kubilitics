/**
 * Tests for useBlastRadius — covers a reported bug: the graph-status check
 * ran exactly once with no polling ("if ready, proceed, if not, give up",
 * per the old code comment), so isGraphReady stayed false for the rest of
 * the session on any cluster whose graph engine wasn't already warm —
 * the blast-radius query never got a second chance to run, and the UI
 * silently showed a topology-only view with no explanation. No test
 * existed for this hook before this change.
 */
import React from 'react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { ReactNode } from 'react';
import { useBlastRadius } from './useBlastRadius';

let mockGetGraphStatus: ReturnType<typeof vi.fn>;
let mockGetBlastRadius: ReturnType<typeof vi.fn>;

vi.mock('@/services/api/blastRadius', () => ({
  getGraphStatus: (...args: unknown[]) => mockGetGraphStatus(...args),
  getBlastRadius: (...args: unknown[]) => mockGetBlastRadius(...args),
}));

vi.mock('@/hooks/useActiveClusterId', () => ({
  useActiveClusterId: () => 'c1',
}));

vi.mock('@/stores/backendConfigStore', () => ({
  useBackendConfigStore: (selector: (s: Record<string, unknown>) => unknown) =>
    selector({ isBackendConfigured: () => true, backendBaseUrl: 'http://localhost:8190' }),
  getEffectiveBackendBaseUrl: (stored: string) => stored ?? 'http://localhost:8190',
}));

function wrapper(qc: QueryClient) {
  return ({ children }: { children: ReactNode }) =>
    React.createElement(QueryClientProvider, { client: qc }, children);
}

describe('useBlastRadius', () => {
  let qc: QueryClient;

  beforeEach(() => {
    qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    mockGetGraphStatus = vi.fn();
    mockGetBlastRadius = vi.fn().mockResolvedValue({ waves: [], criticalityScore: 0 });
  });

  it('CRITICAL FIX: recovers automatically once the graph engine finishes its cold start, instead of giving up forever', async () => {
    let call = 0;
    mockGetGraphStatus.mockImplementation(() => {
      call++;
      // First call (and the cluster's graph engine is still cold-starting):
      // not ready yet — this is the state that used to be permanent.
      return Promise.resolve({
        ready: call > 1,
        nodeCount: call > 1 ? 50 : 0,
        edgeCount: 0,
        namespaceCount: 0,
        lastRebuildMs: 0,
        stalenessMs: 0,
        rebuildCount: call > 1 ? 1 : 0,
      });
    });

    const { result } = renderHook(
      () => useBlastRadius({ kind: 'Pod', namespace: 'default', name: 'my-pod' }),
      { wrapper: wrapper(qc) },
    );

    await waitFor(() => expect(result.current.isGraphReady).toBe(false));
    expect(mockGetBlastRadius).not.toHaveBeenCalled();

    // Without polling, this would never happen — isGraphReady would stay
    // false forever. Force the query's refetchInterval to fire.
    await waitFor(() => expect(call).toBeGreaterThan(1), { timeout: 5_000 });
    await waitFor(() => expect(result.current.isGraphReady).toBe(true), { timeout: 5_000 });

    await waitFor(() => expect(mockGetBlastRadius).toHaveBeenCalled());
  });

  it('stops polling graph-status once ready (no wasted requests)', async () => {
    mockGetGraphStatus.mockResolvedValue({
      ready: true, nodeCount: 10, edgeCount: 5, namespaceCount: 1,
      lastRebuildMs: 100, stalenessMs: 0, rebuildCount: 1,
    });

    const { result } = renderHook(
      () => useBlastRadius({ kind: 'Pod', namespace: 'default', name: 'my-pod' }),
      { wrapper: wrapper(qc) },
    );

    await waitFor(() => expect(result.current.isGraphReady).toBe(true));
    const callsAtReady = mockGetGraphStatus.mock.calls.length;

    // Give any erroneous polling a window to fire, then confirm it didn't.
    await new Promise((r) => setTimeout(r, 300));
    expect(mockGetGraphStatus.mock.calls.length).toBe(callsAtReady);
  });
});
