// useClusterPresence.test.ts — exercises the snapshot-fetch + EventSource path.
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { useClusterPresence } from './useClusterPresence';
import { useClusterPresenceStore, __resetForTest } from '@/stores/clusterPresenceStore';

describe('useClusterPresence', () => {
  beforeEach(() => {
    __resetForTest();
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify({
      discovered: [{ identity: { name: 'a', serverUrl: 'https://a' }, source: 'kubeconfig' }],
      registered: [], connected: [], last_used: null,
    }), { status: 200 })));
    // Stub EventSource to a no-op open.
    class FakeES {
      url: string;
      onmessage: ((ev: MessageEvent) => unknown) | null = null;
      onerror: ((ev: Event) => unknown) | null = null;
      constructor(url: string) { this.url = url; }
      close() {}
    }
    vi.stubGlobal('EventSource', FakeES);
  });

  afterEach(() => vi.unstubAllGlobals());

  it('fetches initial snapshot on mount', async () => {
    renderHook(() => useClusterPresence());
    await waitFor(() => {
      expect(useClusterPresenceStore.getState().isReady).toBe(true);
      expect(useClusterPresenceStore.getState().discovered.length).toBe(1);
    });
  });

  // Final review, Critical #1 (cluster-connection-state plan): the SSE
  // stream only carries discovery-source events (kubeconfig file/secret
  // changes, manual add/remove) — a cluster's reachability flipping with
  // no change to the discovered-cluster set fires no SSE message, so
  // without a periodic re-fetch the frontend's reachability data is a
  // one-time snapshot that never updates for the rest of the session.
  // This locks in that a re-fetch happens on a timer, independent of SSE.
  it('re-fetches the snapshot periodically, not only on mount or SSE events', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({
      discovered: [{ identity: { name: 'a', serverUrl: 'https://a' }, source: 'kubeconfig' }],
      registered: [], connected: [], last_used: null,
    }), { status: 200 }));
    vi.stubGlobal('fetch', fetchMock);

    renderHook(() => useClusterPresence());
    await vi.waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1));

    await vi.advanceTimersByTimeAsync(30_000);
    expect(fetchMock.mock.calls.length).toBeGreaterThanOrEqual(2);

    vi.useRealTimers();
  });
});
