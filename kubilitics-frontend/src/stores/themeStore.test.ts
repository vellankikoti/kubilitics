/**
 * Unit tests for src/stores/themeStore.ts
 *
 * Covers: setTheme, toggleTheme, setResolvedTheme, default state.
 */
import { describe, it, expect, beforeEach, vi } from 'vitest';

vi.mock('@/lib/safeStorage', () => ({
  safeLocalStorage: {
    getItem: (_name: string) => null,
    setItem: (_name: string, _value: string) => {},
    removeItem: (_name: string) => {},
  },
}));

import { useThemeStore } from './themeStore';

describe('themeStore', () => {
  beforeEach(() => {
    useThemeStore.setState({
      theme: 'system',
      resolvedTheme: 'light',
    });
  });

  // ── Default state ──────────────────────────────────────────────────────────

  it('has correct default state', () => {
    const state = useThemeStore.getState();
    expect(state.theme).toBe('system');
    expect(state.resolvedTheme).toBe('light');
  });

  // ── setTheme ───────────────────────────────────────────────────────────────

  it('setTheme sets to dark', () => {
    useThemeStore.getState().setTheme('dark');
    expect(useThemeStore.getState().theme).toBe('dark');
  });

  it('setTheme sets to light', () => {
    useThemeStore.getState().setTheme('light');
    expect(useThemeStore.getState().theme).toBe('light');
  });

  it('setTheme sets to system', () => {
    useThemeStore.getState().setTheme('dark');
    useThemeStore.getState().setTheme('system');
    expect(useThemeStore.getState().theme).toBe('system');
  });

  // ── toggleTheme ────────────────────────────────────────────────────────────

  it('toggleTheme from light switches to dark', () => {
    useThemeStore.getState().setTheme('light');
    useThemeStore.getState().toggleTheme();
    expect(useThemeStore.getState().theme).toBe('dark');
  });

  it('toggleTheme from dark switches to light', () => {
    useThemeStore.getState().setTheme('dark');
    useThemeStore.getState().toggleTheme();
    expect(useThemeStore.getState().theme).toBe('light');
  });

  it('toggleTheme from system uses resolvedTheme to determine current', () => {
    // system with resolvedTheme = light -> toggle to dark
    useThemeStore.setState({ theme: 'system', resolvedTheme: 'light' });
    useThemeStore.getState().toggleTheme();
    expect(useThemeStore.getState().theme).toBe('dark');
  });

  it('toggleTheme from system with dark resolved switches to light', () => {
    useThemeStore.setState({ theme: 'system', resolvedTheme: 'dark' });
    useThemeStore.getState().toggleTheme();
    expect(useThemeStore.getState().theme).toBe('light');
  });

  // ── setResolvedTheme ───────────────────────────────────────────────────────

  it('setResolvedTheme updates resolvedTheme', () => {
    useThemeStore.getState().setResolvedTheme('dark');
    expect(useThemeStore.getState().resolvedTheme).toBe('dark');
    useThemeStore.getState().setResolvedTheme('light');
    expect(useThemeStore.getState().resolvedTheme).toBe('light');
  });
});
