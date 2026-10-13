import { useEffect, useRef } from 'react';
import type { PresenceSnapshot } from '@/types/resilient';
import { useClusterPresenceStore } from '@/stores/clusterPresenceStore';
import { apiUrl, eventSourceUrl } from '@/lib/backendUrl';

const PRESENCE_PATH = '/api/v1/presence';
const SSE_PATH = '/api/v1/presence/events';
// Final review, Critical #1 (cluster-connection-state plan): the SSE
// stream only carries discovery-source events (file/secret/manual
// add/remove), never a pure reachability flip — a cluster going down or
// recovering with the discovered-cluster set unchanged emits nothing.
// Without an independent poll, the store's reachable/last_checked_at
// fields are a one-time snapshot for the rest of the session. Polled at
// half the backend's own 60s reachability-refresh ticker
// (cmd/server/main.go's presence-periodic-refresh) so a change is caught
// within one backend tick on average, not up to two.
const REACHABILITY_POLL_MS = 30_000;

// useClusterPresence subscribes to the backend's presence layer:
//   1. On mount, fetch the current snapshot and apply it.
//   2. Open an EventSource to /api/v1/presence/events and apply deltas.
//
// On snapshot fetch failure we DO NOT apply an empty snapshot — doing
// that would create a false terminal state ("0 clusters, ready") that
// never re-fetches if the backend comes back up. Instead we retry with
// exponential backoff (1s → 2s → 4s → 8s → cap 15s). The route-level
// loading timeout (in PresenceEntryPoint / ProtectedRoute) owns the
// "give up eventually" UX; the hook keeps trying in the background so
// when the backend does come up, the store hot-patches to the real
// data and React re-renders to the right page.
export function useClusterPresence(): void {
  const applySnapshot = useClusterPresenceStore((s) => s.applySnapshot);
  const esRef = useRef<EventSource | null>(null);

  useEffect(() => {
    let cancelled = false;
    let fetchBackoff = 1000;
    let sseBackoff = 1000;
    let fetchTimer: ReturnType<typeof setTimeout> | null = null;

    async function fetchSnapshot(): Promise<boolean> {
      try {
        const r = await fetch(apiUrl(PRESENCE_PATH), { credentials: 'include' });
        if (!r.ok) throw new Error(`HTTP ${r.status}`);
        const snap = (await r.json()) as PresenceSnapshot;
        if (!cancelled) applySnapshot(snap);
        fetchBackoff = 1000; // reset on success
        return true;
      } catch (err) {
        console.warn('presence snapshot fetch failed — will retry:', err);
        return false;
      }
    }

    function scheduleFetchRetry() {
      if (cancelled) return;
      if (fetchTimer) clearTimeout(fetchTimer);
      fetchTimer = setTimeout(async () => {
        const ok = await fetchSnapshot();
        if (!ok && !cancelled) {
          fetchBackoff = Math.min(fetchBackoff * 2, 15_000);
          scheduleFetchRetry();
        }
      }, fetchBackoff);
    }

    function openStream() {
      if (cancelled) return;
      const es = new EventSource(eventSourceUrl(SSE_PATH), { withCredentials: true } as EventSourceInit);
      esRef.current = es;
      es.onmessage = async () => {
        // Simplest correct impl: re-fetch snapshot on any event.
        await fetchSnapshot();
      };
      es.onerror = () => {
        es.close();
        esRef.current = null;
        if (cancelled) return;
        setTimeout(() => {
          sseBackoff = Math.min(sseBackoff * 2, 30_000);
          openStream();
        }, sseBackoff);
      };
      es.onopen = () => { sseBackoff = 1000; };
    }

    (async () => {
      const ok = await fetchSnapshot();
      if (!ok && !cancelled) scheduleFetchRetry();
      openStream();
    })();

    const pollTimer = setInterval(() => {
      void fetchSnapshot();
    }, REACHABILITY_POLL_MS);

    return () => {
      cancelled = true;
      if (fetchTimer) clearTimeout(fetchTimer);
      clearInterval(pollTimer);
      esRef.current?.close();
    };
  }, [applySnapshot]);
}
