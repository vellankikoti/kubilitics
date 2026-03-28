/**
 * Unit tests for src/stores/projectStore.ts
 *
 * Covers: setActiveProject, clearActiveProject, default state.
 */
import { describe, it, expect, beforeEach } from 'vitest';
import { useProjectStore } from './projectStore';
import type { Project } from './projectStore';

const PERSIST_KEY = 'kubilitics-project-store';

const sampleProject: Project = {
  id: 'proj-1',
  name: 'Production',
  description: 'Production workloads',
  clusters: [
    { cluster_id: 'c1', namespaces: ['default', 'kube-system'] },
    { cluster_id: 'c2', namespaces: ['app-ns'] },
  ],
};

const anotherProject: Project = {
  id: 'proj-2',
  name: 'Staging',
  clusters: [],
};

describe('projectStore', () => {
  beforeEach(() => {
    localStorage.removeItem(PERSIST_KEY);
    useProjectStore.setState({
      activeProject: null,
      activeProjectId: null,
    });
  });

  // ── Default state ──────────────────────────────────────────────────────────

  it('has null defaults for activeProject and activeProjectId', () => {
    const state = useProjectStore.getState();
    expect(state.activeProject).toBeNull();
    expect(state.activeProjectId).toBeNull();
  });

  // ── setActiveProject ───────────────────────────────────────────────────────

  it('setActiveProject sets both activeProject and activeProjectId', () => {
    useProjectStore.getState().setActiveProject(sampleProject);
    const state = useProjectStore.getState();
    expect(state.activeProject).toEqual(sampleProject);
    expect(state.activeProjectId).toBe('proj-1');
  });

  it('setActiveProject with null clears both fields', () => {
    useProjectStore.getState().setActiveProject(sampleProject);
    useProjectStore.getState().setActiveProject(null);
    const state = useProjectStore.getState();
    expect(state.activeProject).toBeNull();
    expect(state.activeProjectId).toBeNull();
  });

  it('setActiveProject replaces existing project', () => {
    useProjectStore.getState().setActiveProject(sampleProject);
    useProjectStore.getState().setActiveProject(anotherProject);
    const state = useProjectStore.getState();
    expect(state.activeProject).toEqual(anotherProject);
    expect(state.activeProjectId).toBe('proj-2');
  });

  // ── clearActiveProject ─────────────────────────────────────────────────────

  it('clearActiveProject clears both activeProject and activeProjectId', () => {
    useProjectStore.getState().setActiveProject(sampleProject);
    useProjectStore.getState().clearActiveProject();
    const state = useProjectStore.getState();
    expect(state.activeProject).toBeNull();
    expect(state.activeProjectId).toBeNull();
  });

  it('clearActiveProject is idempotent on already-null state', () => {
    useProjectStore.getState().clearActiveProject();
    const state = useProjectStore.getState();
    expect(state.activeProject).toBeNull();
    expect(state.activeProjectId).toBeNull();
  });

  // ── Persistence ────────────────────────────────────────────────────────────

  it('persists the project data to localStorage', async () => {
    useProjectStore.getState().setActiveProject(sampleProject);

    // Allow async persist write
    await new Promise((r) => setTimeout(r, 20));

    const raw = localStorage.getItem(PERSIST_KEY);
    expect(raw).toBeTruthy();
    const parsed = JSON.parse(raw!);
    expect(parsed.state.activeProjectId).toBe('proj-1');
  });
});
