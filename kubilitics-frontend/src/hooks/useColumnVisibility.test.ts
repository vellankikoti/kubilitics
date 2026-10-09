/**
 * docs/ai/STABILIZATION-PLAN.md Phase 2 item 4: useColumnVisibility fans out
 * to 53 list pages and had zero test coverage.
 */
import { describe, expect, it, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useColumnVisibility } from './useColumnVisibility';

const columns = [
  { id: 'status', label: 'Status' },
  { id: 'age', label: 'Age' },
  { id: 'node', label: 'Node' },
];

beforeEach(() => {
  localStorage.clear();
});

describe('useColumnVisibility', () => {
  it('defaults to every column visible (alwaysVisible + all toggleable) with no stored state', () => {
    const { result } = renderHook(() => useColumnVisibility({ tableId: 'pods', columns }));
    expect(result.current.visibleColumns).toEqual(new Set(['name', 'status', 'age', 'node']));
    expect(result.current.isColumnVisible('name')).toBe(true);
    expect(result.current.isColumnVisible('status')).toBe(true);
  });

  it('hides a column and persists the change to localStorage', () => {
    const { result } = renderHook(() => useColumnVisibility({ tableId: 'pods', columns }));

    act(() => {
      result.current.setColumnVisible('age', false);
    });

    expect(result.current.isColumnVisible('age')).toBe(false);
    expect(result.current.isColumnVisible('status')).toBe(true);

    const stored = JSON.parse(localStorage.getItem('kubilitics-columns-pods') ?? '[]');
    expect(stored).not.toContain('age');
    expect(stored).toContain('status');
  });

  it('a fresh hook instance for the same tableId picks up the persisted hidden column', () => {
    const { result: first } = renderHook(() => useColumnVisibility({ tableId: 'pods', columns }));
    act(() => {
      first.current.setColumnVisible('node', false);
    });

    const { result: second } = renderHook(() => useColumnVisibility({ tableId: 'pods', columns }));
    expect(second.current.isColumnVisible('node')).toBe(false);
    expect(second.current.isColumnVisible('status')).toBe(true);
  });

  it('does not leak hidden-column state across different tableIds', () => {
    const { result: pods } = renderHook(() => useColumnVisibility({ tableId: 'pods', columns }));
    act(() => {
      pods.current.setColumnVisible('age', false);
    });

    const { result: deployments } = renderHook(() => useColumnVisibility({ tableId: 'deployments', columns }));
    expect(deployments.current.isColumnVisible('age')).toBe(true);
  });

  it('ignores a stored column id that no longer exists in the current column list', () => {
    localStorage.setItem('kubilitics-columns-pods', JSON.stringify(['status', 'stale-removed-column']));
    const { result } = renderHook(() => useColumnVisibility({ tableId: 'pods', columns }));
    // "status" survives (still a real column); the stale id must not appear
    // anywhere, and must not crash the hook.
    expect(result.current.isColumnVisible('status')).toBe(true);
    expect(result.current.visibleColumns.has('stale-removed-column')).toBe(false);
  });

  it('falls back to all-visible when the stored value is corrupt JSON', () => {
    localStorage.setItem('kubilitics-columns-pods', '{not valid json');
    const { result } = renderHook(() => useColumnVisibility({ tableId: 'pods', columns }));
    expect(result.current.visibleColumns).toEqual(new Set(['name', 'status', 'age', 'node']));
  });

  it('re-showing a hidden column updates both state and storage', () => {
    const { result } = renderHook(() => useColumnVisibility({ tableId: 'pods', columns }));
    act(() => {
      result.current.setColumnVisible('status', false);
    });
    expect(result.current.isColumnVisible('status')).toBe(false);

    act(() => {
      result.current.setColumnVisible('status', true);
    });
    expect(result.current.isColumnVisible('status')).toBe(true);
    const stored = JSON.parse(localStorage.getItem('kubilitics-columns-pods') ?? '[]');
    expect(stored).toContain('status');
  });
});
