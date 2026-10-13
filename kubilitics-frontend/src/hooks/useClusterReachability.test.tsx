import { describe, it, expect, beforeEach } from 'vitest';
import { renderHook } from '@testing-library/react';
import { useClusterReachability, useAllClusterReachability } from './useClusterReachability';
import { useClusterPresenceStore } from '@/stores/clusterPresenceStore';
import type { RegisteredCluster } from '@/types/resilient';

function registeredCluster(overrides: Partial<RegisteredCluster>): RegisteredCluster {
  return {
    identity: { name: 'test', serverUrl: 'https://127.0.0.1:1' },
    source: 'manual',
    registered_at: '2026-10-12T00:00:00Z',
    reachable: true,
    session_id: 'cluster-1',
    ...overrides,
  };
}

describe('useClusterReachability', () => {
  beforeEach(() => {
    useClusterPresenceStore.setState({
      discovered: [],
      registered: [],
      connected: [],
      activeLogicalIdentity: null,
      isReady: true,
    });
  });

  it('returns unknown when the cluster is not present at all', () => {
    const { result } = renderHook(() => useClusterReachability('cluster-1'));
    expect(result.current).toEqual({ status: 'unknown' });
  });

  it('returns connecting when registered but never checked', () => {
    useClusterPresenceStore.setState({
      registered: [registeredCluster({ session_id: 'cluster-1', last_checked_at: undefined })],
    });
    const { result } = renderHook(() => useClusterReachability('cluster-1'));
    expect(result.current).toEqual({ status: 'connecting' });
  });

  it('returns connected when reachable with a fresh check', () => {
    useClusterPresenceStore.setState({
      registered: [registeredCluster({
        session_id: 'cluster-1', reachable: true,
        last_checked_at: '2026-10-12T00:05:00Z', last_success_at: '2026-10-12T00:05:00Z',
      })],
    });
    const { result } = renderHook(() => useClusterReachability('cluster-1'));
    expect(result.current).toEqual({
      status: 'connected', lastCheckedAt: '2026-10-12T00:05:00Z', lastSuccessAt: '2026-10-12T00:05:00Z',
    });
  });

  it('returns unreachable when it was connected before but the last check failed', () => {
    useClusterPresenceStore.setState({
      registered: [registeredCluster({
        session_id: 'cluster-1', reachable: false,
        last_checked_at: '2026-10-12T00:10:00Z', last_success_at: '2026-10-12T00:05:00Z',
        last_error: 'dial tcp: connection refused',
      })],
    });
    const { result } = renderHook(() => useClusterReachability('cluster-1'));
    expect(result.current).toEqual({
      status: 'unreachable', lastCheckedAt: '2026-10-12T00:10:00Z',
      lastSuccessAt: '2026-10-12T00:05:00Z', lastError: 'dial tcp: connection refused',
    });
  });

  it('returns error when it has never once been reached', () => {
    useClusterPresenceStore.setState({
      registered: [registeredCluster({
        session_id: 'cluster-1', reachable: false,
        last_checked_at: '2026-10-12T00:10:00Z', last_success_at: undefined,
        last_error: 'auth error: invalid token',
      })],
    });
    const { result } = renderHook(() => useClusterReachability('cluster-1'));
    expect(result.current).toEqual({
      status: 'error', lastCheckedAt: '2026-10-12T00:10:00Z', lastError: 'auth error: invalid token',
    });
  });

  it('prefers the connected[] entry over registered[] when both exist (connected is a superset)', () => {
    useClusterPresenceStore.setState({
      registered: [registeredCluster({ session_id: 'cluster-1', reachable: false, last_checked_at: '2026-10-12T00:01:00Z' })],
      connected: [{
        ...registeredCluster({ session_id: 'cluster-1', reachable: true, last_checked_at: '2026-10-12T00:05:00Z', last_success_at: '2026-10-12T00:05:00Z' }),
        connected_at: '2026-10-12T00:05:00Z',
      }],
    });
    const { result } = renderHook(() => useClusterReachability('cluster-1'));
    expect(result.current.status).toBe('connected');
  });
});

describe('useAllClusterReachability', () => {
  beforeEach(() => {
    useClusterPresenceStore.setState({
      discovered: [],
      registered: [],
      connected: [],
      activeLogicalIdentity: null,
      isReady: true,
    });
  });

  it('does not infinite-loop re-rendering when store data is unchanged (regression)', () => {
    useClusterPresenceStore.setState({
      registered: [
        registeredCluster({ session_id: 'c1', reachable: true, last_checked_at: 't', last_success_at: 't' }),
        registeredCluster({ session_id: 'c2', reachable: false, last_checked_at: 't', last_success_at: null as unknown as string }),
      ],
    });
    // renderHook throws (React's "Maximum update depth exceeded") if the
    // selector allocates a new object on every call — this is the
    // assertion, not just a smoke check.
    expect(() => renderHook(() => useAllClusterReachability())).not.toThrow();
  });

  it('maps every registered cluster by session_id', () => {
    useClusterPresenceStore.setState({
      registered: [
        registeredCluster({ session_id: 'c1', reachable: true, last_checked_at: 't', last_success_at: 't' }),
        registeredCluster({ session_id: 'c2', reachable: false, last_checked_at: 't', last_success_at: undefined, last_error: 'boom' }),
      ],
    });
    const { result } = renderHook(() => useAllClusterReachability());
    expect(result.current.c1.status).toBe('connected');
    expect(result.current.c2.status).toBe('error');
  });
});
