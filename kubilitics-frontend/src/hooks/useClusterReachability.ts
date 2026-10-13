// THE ONLY HOOK ANY UI MAY GATE ON for "is this cluster reachable right
// now." Wraps clusterPresenceStore — does not duplicate its data in a
// second store. See docs/superpowers/specs/2026-10-12-cluster-connection-state-design.md
// for the precedence table this implements.
import { useMemo } from 'react';
import { useShallow } from 'zustand/react/shallow';
import { useClusterPresenceStore } from '@/stores/clusterPresenceStore';
import type { RegisteredCluster, ConnectedCluster } from '@/types/resilient';

export type ClusterReachability =
  | { status: 'unknown' }
  | { status: 'connecting' }
  | { status: 'connected'; lastCheckedAt: string; lastSuccessAt: string }
  | { status: 'unreachable'; lastCheckedAt: string; lastSuccessAt: string | null; lastError: string }
  | { status: 'error'; lastCheckedAt: string; lastError: string };

function deriveReachability(
  entry: RegisteredCluster | ConnectedCluster | undefined
): ClusterReachability {
  if (!entry) return { status: 'unknown' };
  if (!entry.last_checked_at) return { status: 'connecting' };

  if (!entry.reachable) {
    if (!entry.last_success_at) {
      return { status: 'error', lastCheckedAt: entry.last_checked_at, lastError: entry.last_error ?? '' };
    }
    return {
      status: 'unreachable',
      lastCheckedAt: entry.last_checked_at,
      lastSuccessAt: entry.last_success_at,
      lastError: entry.last_error ?? '',
    };
  }

  return {
    status: 'connected',
    lastCheckedAt: entry.last_checked_at,
    lastSuccessAt: entry.last_success_at ?? entry.last_checked_at,
  };
}

export function useClusterReachability(clusterId: string | undefined): ClusterReachability {
  return useClusterPresenceStore(
    useShallow((s) => {
      if (!clusterId) return { status: 'unknown' } as const;
      const connected = s.connected.find((c) => c.session_id === clusterId);
      const registered = s.registered.find((c) => c.session_id === clusterId);
      return deriveReachability(connected ?? registered);
    })
  );
}

export function useAllClusterReachability(): Record<string, ClusterReachability> {
  // Select the store's own arrays (stable references, only change on a
  // real store update) via useShallow, then derive the per-cluster map in
  // a separate useMemo. Building the Record directly inside the selector
  // would allocate a fresh object on every call — one level too deep for
  // useShallow's single-level comparison to catch — causing an infinite
  // render loop (found while running this task's own tests).
  const { registered, connected } = useClusterPresenceStore(
    useShallow((s) => ({ registered: s.registered, connected: s.connected }))
  );
  return useMemo(() => {
    const result: Record<string, ClusterReachability> = {};
    for (const c of registered) {
      const connectedMatch = connected.find((cc) => cc.session_id === c.session_id);
      result[c.session_id] = deriveReachability(connectedMatch ?? c);
    }
    return result;
  }, [registered, connected]);
}
