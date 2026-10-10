import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { useClusterPresenceStore, __resetForTest } from './clusterPresenceStore';
import { onClusterSwitch, clearClusterSwitchListeners, __lastCluster } from './clusterSwitch';

describe('clusterPresenceStore', () => {
  beforeEach(() => { __resetForTest(); });

  it('initial state is empty and not ready', () => {
    const s = useClusterPresenceStore.getState();
    expect(s.discovered).toEqual([]);
    expect(s.registered).toEqual([]);
    expect(s.connected).toEqual([]);
    expect(s.activeLogicalIdentity).toBeNull();
    expect(s.isReady).toBe(false);
  });

  it('applySnapshot populates state + marks ready', () => {
    useClusterPresenceStore.getState().applySnapshot({
      discovered: [{ identity: { name: 'a', serverUrl: 'https://a' }, source: 'kubeconfig' }],
      registered: [],
      connected: [],
      last_used: { name: 'a', serverUrl: 'https://a' },
    });
    const s = useClusterPresenceStore.getState();
    expect(s.discovered.length).toBe(1);
    expect(s.isReady).toBe(true);
    expect(s.activeLogicalIdentity?.name).toBe('a');
  });

  it('setActiveByLogicalIdentity persists to localStorage', () => {
    const id = { name: 'prod', serverUrl: 'https://prod' };
    useClusterPresenceStore.getState().setActiveByLogicalIdentity(id);
    const raw = localStorage.getItem('kubilitics.presence.lastActive');
    expect(JSON.parse(raw!)).toEqual(id);
  });

  describe('setActiveByLogicalIdentity emits a cluster-switch event', () => {
    beforeEach(() => {
      clearClusterSwitchListeners();
    });

    afterEach(() => {
      clearClusterSwitchListeners();
    });

    it('notifies subscribers on the clusterSwitch bus so cached stores invalidate', () => {
      const seen: string[] = [];
      onClusterSwitch((id) => seen.push(id));

      useClusterPresenceStore.getState().setActiveByLogicalIdentity({
        name: 'prod',
        serverUrl: 'https://prod',
      });

      expect(seen).toHaveLength(1);
      expect(seen[0]).toBe(__lastCluster());
    });

    it('does not re-emit when switching to the same logical identity twice', () => {
      const seen: string[] = [];
      const id = { name: 'staging', serverUrl: 'https://staging' };
      useClusterPresenceStore.getState().setActiveByLogicalIdentity(id);
      onClusterSwitch((cid) => seen.push(cid));

      useClusterPresenceStore.getState().setActiveByLogicalIdentity({ ...id });

      expect(seen).toHaveLength(0);
    });

    it('emits again when switching to a genuinely different cluster', () => {
      const seen: string[] = [];
      onClusterSwitch((cid) => seen.push(cid));

      useClusterPresenceStore.getState().setActiveByLogicalIdentity({
        name: 'a',
        serverUrl: 'https://a',
      });
      useClusterPresenceStore.getState().setActiveByLogicalIdentity({
        name: 'b',
        serverUrl: 'https://b',
      });

      expect(seen).toHaveLength(2);
      expect(seen[0]).not.toBe(seen[1]);
    });
  });

  it('activeCluster derives from connected using logical identity', () => {
    useClusterPresenceStore.getState().applySnapshot({
      discovered: [],
      registered: [],
      connected: [{
        identity: { name: 'prod', serverUrl: 'https://prod' },
        source: 'kubeconfig',
        registered_at: '', reachable: true, connected_at: '',
        session_id: 'uuid-prod',
      }],
      last_used: { name: 'prod', serverUrl: 'https://prod' },
    });
    const s = useClusterPresenceStore.getState();
    expect(s.activeCluster()?.identity.name).toBe('prod');
  });

  it('session_id + provider round-trip through applySnapshot', () => {
    useClusterPresenceStore.getState().applySnapshot({
      discovered: [{
        identity: { name: 'prod', serverUrl: 'https://prod' },
        source: 'manual',
        session_id: 'uuid-prod',
        provider: 'eks',
      }],
      registered: [{
        identity: { name: 'prod', serverUrl: 'https://prod' },
        source: 'manual',
        registered_at: '2026-04-24T00:00:00Z',
        reachable: true,
        session_id: 'uuid-prod',
        provider: 'eks',
      }],
      connected: [],
    });
    const s = useClusterPresenceStore.getState();
    expect(s.registered[0].session_id).toBe('uuid-prod');
    expect(s.registered[0].provider).toBe('eks');
    expect(s.discovered[0].session_id).toBe('uuid-prod');
  });
});
