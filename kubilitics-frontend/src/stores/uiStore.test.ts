/**
 * Unit tests for src/stores/uiStore.ts
 *
 * Covers: sidebar collapse/expand, toggleSidebar, auto-collapse flag,
 * resource category toggling (single-open behavior), resources section open,
 * shell open/close, shell height.
 */
import { describe, it, expect, beforeEach } from 'vitest';
import { useUIStore } from './uiStore';

const PERSIST_KEY = 'kubilitics-sidebar-collapsed';

describe('uiStore', () => {
  beforeEach(() => {
    localStorage.removeItem(PERSIST_KEY);
    // Reset store to defaults
    useUIStore.setState({
      isSidebarCollapsed: false,
      isAutoCollapsed: false,
      expandedResourceCategories: ['workloads'],
      isResourcesSectionOpen: true,
      isShellOpen: false,
      shellHeightPx: 320,
    });
  });

  // ── Default state ──────────────────────────────────────────────────────────

  it('has correct default state', () => {
    const state = useUIStore.getState();
    expect(state.isSidebarCollapsed).toBe(false);
    expect(state.isAutoCollapsed).toBe(false);
    expect(state.expandedResourceCategories).toEqual(['workloads']);
    expect(state.isResourcesSectionOpen).toBe(true);
    expect(state.isShellOpen).toBe(false);
    expect(state.shellHeightPx).toBe(320);
  });

  // ── setSidebarCollapsed ────────────────────────────────────────────────────

  it('setSidebarCollapsed(true) collapses sidebar and clears auto flag', () => {
    useUIStore.getState().setAutoCollapsed(true);
    useUIStore.getState().setSidebarCollapsed(true);
    const state = useUIStore.getState();
    expect(state.isSidebarCollapsed).toBe(true);
    expect(state.isAutoCollapsed).toBe(false);
  });

  it('setSidebarCollapsed(false) expands sidebar', () => {
    useUIStore.getState().setSidebarCollapsed(true);
    useUIStore.getState().setSidebarCollapsed(false);
    expect(useUIStore.getState().isSidebarCollapsed).toBe(false);
  });

  // ── toggleSidebar ──────────────────────────────────────────────────────────

  it('toggleSidebar toggles from collapsed to expanded', () => {
    useUIStore.getState().setSidebarCollapsed(true);
    useUIStore.getState().toggleSidebar();
    expect(useUIStore.getState().isSidebarCollapsed).toBe(false);
  });

  it('toggleSidebar toggles from expanded to collapsed', () => {
    useUIStore.getState().toggleSidebar();
    expect(useUIStore.getState().isSidebarCollapsed).toBe(true);
  });

  it('toggleSidebar clears isAutoCollapsed', () => {
    useUIStore.setState({ isAutoCollapsed: true, isSidebarCollapsed: true });
    useUIStore.getState().toggleSidebar();
    expect(useUIStore.getState().isAutoCollapsed).toBe(false);
  });

  // ── setAutoCollapsed ───────────────────────────────────────────────────────

  it('setAutoCollapsed sets the auto-collapse flag', () => {
    useUIStore.getState().setAutoCollapsed(true);
    expect(useUIStore.getState().isAutoCollapsed).toBe(true);
    useUIStore.getState().setAutoCollapsed(false);
    expect(useUIStore.getState().isAutoCollapsed).toBe(false);
  });

  // ── toggleResourceCategory ─────────────────────────────────────────────────

  it('toggleResourceCategory adds a category (single-open: replaces existing)', () => {
    useUIStore.getState().toggleResourceCategory('networking');
    expect(useUIStore.getState().expandedResourceCategories).toEqual(['networking']);
  });

  it('toggleResourceCategory removes a category when already expanded', () => {
    useUIStore.getState().toggleResourceCategory('workloads');
    expect(useUIStore.getState().expandedResourceCategories).toEqual([]);
  });

  it('toggleResourceCategory enforces single-open behavior', () => {
    // Start with workloads expanded
    expect(useUIStore.getState().expandedResourceCategories).toEqual(['workloads']);
    // Toggle networking — should replace workloads, not append
    useUIStore.getState().toggleResourceCategory('networking');
    expect(useUIStore.getState().expandedResourceCategories).toEqual(['networking']);
    // Toggle storage — replaces networking
    useUIStore.getState().toggleResourceCategory('storage');
    expect(useUIStore.getState().expandedResourceCategories).toEqual(['storage']);
  });

  it('toggleResourceCategory re-expands a previously collapsed category', () => {
    useUIStore.getState().toggleResourceCategory('workloads'); // collapse
    expect(useUIStore.getState().expandedResourceCategories).toEqual([]);
    useUIStore.getState().toggleResourceCategory('workloads'); // re-expand
    expect(useUIStore.getState().expandedResourceCategories).toEqual(['workloads']);
  });

  // ── setResourcesSectionOpen ────────────────────────────────────────────────

  it('setResourcesSectionOpen toggles resources section', () => {
    useUIStore.getState().setResourcesSectionOpen(false);
    expect(useUIStore.getState().isResourcesSectionOpen).toBe(false);
    useUIStore.getState().setResourcesSectionOpen(true);
    expect(useUIStore.getState().isResourcesSectionOpen).toBe(true);
  });

  // ── setShellOpen ───────────────────────────────────────────────────────────

  it('setShellOpen opens and closes the shell panel', () => {
    useUIStore.getState().setShellOpen(true);
    expect(useUIStore.getState().isShellOpen).toBe(true);
    useUIStore.getState().setShellOpen(false);
    expect(useUIStore.getState().isShellOpen).toBe(false);
  });

  // ── setShellHeightPx ──────────────────────────────────────────────────────

  it('setShellHeightPx updates the shell height', () => {
    useUIStore.getState().setShellHeightPx(500);
    expect(useUIStore.getState().shellHeightPx).toBe(500);
  });

  it('setShellHeightPx accepts zero', () => {
    useUIStore.getState().setShellHeightPx(0);
    expect(useUIStore.getState().shellHeightPx).toBe(0);
  });

  // ── Persistence partialize ─────────────────────────────────────────────────

  it('persists only sidebar state, categories, and resources section', async () => {
    useUIStore.getState().setSidebarCollapsed(true);
    useUIStore.getState().setShellOpen(true);
    useUIStore.getState().setShellHeightPx(999);

    // Allow persist middleware to write
    await new Promise((r) => setTimeout(r, 20));

    const raw = localStorage.getItem(PERSIST_KEY);
    expect(raw).toBeTruthy();
    const parsed = JSON.parse(raw!);
    const state = parsed.state;

    // Persisted fields
    expect(state).toHaveProperty('isSidebarCollapsed');
    expect(state).toHaveProperty('expandedResourceCategories');
    expect(state).toHaveProperty('isResourcesSectionOpen');

    // Non-persisted fields should NOT appear
    expect(state).not.toHaveProperty('isShellOpen');
    expect(state).not.toHaveProperty('shellHeightPx');
    expect(state).not.toHaveProperty('isAutoCollapsed');
  });
});
