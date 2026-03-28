/**
 * Tests for clusterOrganizationStore — groups, favorites, env tags, fuzzy search, persistence.
 */
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { useClusterOrganizationStore, fuzzyMatch } from './clusterOrganizationStore';

const PERSIST_KEY = 'kubilitics-cluster-organization';

function resetStore() {
  const s = useClusterOrganizationStore.getState();
  // Remove all groups
  for (const id of Object.keys(s.groups)) s.removeGroup(id);
  // Remove all favorites
  for (const fav of [...s.favorites]) s.toggleFavorite(fav);
  // Remove all env tags
  for (const cid of Object.keys(s.envTags)) s.setEnvTag(cid, null);
}

describe('clusterOrganizationStore', () => {
  beforeEach(() => {
    localStorage.removeItem(PERSIST_KEY);
    resetStore();
  });

  afterEach(() => {
    resetStore();
  });

  // ── Groups ────────────────────────────────────────────────────────────────

  it('addGroup creates a group with name and color', () => {
    const store = useClusterOrganizationStore.getState();
    store.addGroup('g1', 'Team Alpha', '#3b82f6');

    const updated = useClusterOrganizationStore.getState();
    expect(updated.groups['g1']).toBeDefined();
    expect(updated.groups['g1'].name).toBe('Team Alpha');
    expect(updated.groups['g1'].color).toBe('#3b82f6');
    expect(updated.groups['g1'].clusterIds).toEqual([]);
  });

  it('removeGroup deletes a group', () => {
    const store = useClusterOrganizationStore.getState();
    store.addGroup('g1', 'Team Alpha', '#3b82f6');
    store.removeGroup('g1');

    const updated = useClusterOrganizationStore.getState();
    expect(updated.groups['g1']).toBeUndefined();
  });

  it('renameGroup changes the group name', () => {
    const store = useClusterOrganizationStore.getState();
    store.addGroup('g1', 'Team Alpha', '#3b82f6');
    store.renameGroup('g1', 'Team Beta');

    const updated = useClusterOrganizationStore.getState();
    expect(updated.groups['g1'].name).toBe('Team Beta');
    expect(updated.groups['g1'].color).toBe('#3b82f6');
  });

  it('addToGroup and removeFromGroup manage cluster membership', () => {
    const store = useClusterOrganizationStore.getState();
    store.addGroup('g1', 'Group', '#fff');
    store.addToGroup('g1', 'cluster-1');
    store.addToGroup('g1', 'cluster-2');

    let updated = useClusterOrganizationStore.getState();
    expect(updated.groups['g1'].clusterIds).toEqual(['cluster-1', 'cluster-2']);

    // Duplicate add should not create a second entry
    store.addToGroup('g1', 'cluster-1');
    updated = useClusterOrganizationStore.getState();
    expect(updated.groups['g1'].clusterIds).toEqual(['cluster-1', 'cluster-2']);

    store.removeFromGroup('g1', 'cluster-1');
    updated = useClusterOrganizationStore.getState();
    expect(updated.groups['g1'].clusterIds).toEqual(['cluster-2']);
  });

  // ── Favorites ─────────────────────────────────────────────────────────────

  it('toggleFavorite adds a cluster to favorites', () => {
    const store = useClusterOrganizationStore.getState();
    store.toggleFavorite('cluster-a');

    const updated = useClusterOrganizationStore.getState();
    expect(updated.favorites).toContain('cluster-a');
    expect(updated.isFavorite('cluster-a')).toBe(true);
  });

  it('toggleFavorite removes a cluster from favorites on second call', () => {
    const store = useClusterOrganizationStore.getState();
    store.toggleFavorite('cluster-a');
    store.toggleFavorite('cluster-a');

    const updated = useClusterOrganizationStore.getState();
    expect(updated.favorites).not.toContain('cluster-a');
    expect(updated.isFavorite('cluster-a')).toBe(false);
  });

  // ── Environment tags ──────────────────────────────────────────────────────

  it('setEnvTag assigns an environment tag to a cluster', () => {
    const store = useClusterOrganizationStore.getState();
    store.setEnvTag('cluster-x', 'production');

    const updated = useClusterOrganizationStore.getState();
    expect(updated.envTags['cluster-x']).toBe('production');
    expect(updated.getEnvTag('cluster-x')).toBe('production');
  });

  it('setEnvTag with null removes the tag', () => {
    const store = useClusterOrganizationStore.getState();
    store.setEnvTag('cluster-x', 'staging');
    store.setEnvTag('cluster-x', null);

    const updated = useClusterOrganizationStore.getState();
    expect(updated.envTags['cluster-x']).toBeUndefined();
    expect(updated.getEnvTag('cluster-x')).toBeUndefined();
  });

  // ── Persistence ───────────────────────────────────────────────────────────

  it('persists state to localStorage', async () => {
    const store = useClusterOrganizationStore.getState();
    store.addGroup('g1', 'Persisted Group', '#ff0000');
    store.toggleFavorite('fav-cluster');
    store.setEnvTag('env-cluster', 'development');

    // Zustand persist writes asynchronously
    await new Promise((r) => setTimeout(r, 50));

    const raw = localStorage.getItem(PERSIST_KEY);
    expect(raw).toBeTruthy();

    const parsed = JSON.parse(raw!);
    const state = parsed.state;

    expect(state.groups['g1']).toBeDefined();
    expect(state.groups['g1'].name).toBe('Persisted Group');
    expect(state.favorites).toContain('fav-cluster');
    expect(state.envTags['env-cluster']).toBe('development');
  });
});

// ── fuzzyMatch ──────────────────────────────────────────────────────────────

describe('fuzzyMatch', () => {
  it('empty query matches everything with score 1', () => {
    const result = fuzzyMatch('', 'anything');
    expect(result.matches).toBe(true);
    expect(result.score).toBe(1);
  });

  it('exact substring match returns score 2', () => {
    const result = fuzzyMatch('prod', 'us-east-production');
    expect(result.matches).toBe(true);
    expect(result.score).toBe(2);
  });

  it('case-insensitive exact substring match returns score 2', () => {
    const result = fuzzyMatch('PROD', 'us-east-production');
    expect(result.matches).toBe(true);
    expect(result.score).toBe(2);
  });

  it('fuzzy character-by-character match succeeds when chars appear in order', () => {
    // "upe" appears in "us-production-east" as u...p...e (non-contiguous)
    const result = fuzzyMatch('upe', 'us-production-east');
    expect(result.matches).toBe(true);
    expect(result.score).toBeGreaterThan(0);
  });

  it('returns no match when characters do not appear in order', () => {
    const result = fuzzyMatch('xyz', 'production');
    expect(result.matches).toBe(false);
    expect(result.score).toBeLessThan(3); // fewer than all 3 matched
  });

  it('consecutive characters yield higher score than spread-out ones', () => {
    // "prod" in "xprodyz" = substring match, score 2
    // "pyz" in "xprodyz" = fuzzy match: p(1)+y(1)+z(2)=4... but let's use a case
    // where we can compare two fuzzy (non-substring) matches.
    // In "abcdefghij": "abc" is a substring (score 2), "adj" is fuzzy
    // Better: compare two pure fuzzy matches with different consecutiveness.
    // "abcxyz": "acy" = a(1), c reset, skip b, c(1), ... actually let's pick clear examples.

    // Both are fuzzy (non-substring) matches in a longer string
    // "a_b_c_d_e" for query "abcde": a(1) _ skip, b(1) _ skip, c(1) _ skip, d(1) _ skip, e(1) = score 5
    // vs "abcde_xyz" for query "abcde": substring match, score 2
    // We want two fuzzy matches with different consecutiveness:
    // target = "a__bc__de__f"
    // query1 = "bcde" -> b(1)c(2)..skip..d(1)e(2) = 6  (two pairs of consecutive)
    // query2 = "bdf"  -> b(1)..skip..d(1)..skip..f(1) = 3  (all spread out)
    const moreConsecutive = fuzzyMatch('bcde', 'a__bc__de__f');
    const lessConsecutive = fuzzyMatch('bdf', 'a__bc__de__f');

    expect(moreConsecutive.matches).toBe(true);
    expect(lessConsecutive.matches).toBe(true);
    expect(moreConsecutive.score).toBeGreaterThan(lessConsecutive.score);
  });
});
