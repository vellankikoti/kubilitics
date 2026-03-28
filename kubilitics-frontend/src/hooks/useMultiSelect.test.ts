/**
 * Tests for useMultiSelect hook — toggle, selectAll, clearSelection, isSelected, toggleRange, count.
 */
import { describe, it, expect } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useMultiSelect } from './useMultiSelect';

describe('useMultiSelect', () => {
  it('starts with empty selection', () => {
    const { result } = renderHook(() => useMultiSelect());
    expect(result.current.selectedIds.size).toBe(0);
    expect(result.current.count).toBe(0);
    expect(result.current.hasSelection).toBe(false);
  });

  // ── toggle ──────────────────────────────────────────────────────────────

  it('toggle adds an item to the selection', () => {
    const { result } = renderHook(() => useMultiSelect());

    act(() => {
      result.current.toggle('pod-1');
    });

    expect(result.current.isSelected('pod-1')).toBe(true);
    expect(result.current.count).toBe(1);
    expect(result.current.hasSelection).toBe(true);
  });

  it('toggle removes an item when called a second time', () => {
    const { result } = renderHook(() => useMultiSelect());

    act(() => {
      result.current.toggle('pod-1');
    });
    act(() => {
      result.current.toggle('pod-1');
    });

    expect(result.current.isSelected('pod-1')).toBe(false);
    expect(result.current.count).toBe(0);
  });

  it('toggle manages multiple items independently', () => {
    const { result } = renderHook(() => useMultiSelect());

    act(() => {
      result.current.toggle('a');
      result.current.toggle('b');
      result.current.toggle('c');
    });

    expect(result.current.count).toBe(3);

    act(() => {
      result.current.toggle('b');
    });

    expect(result.current.isSelected('a')).toBe(true);
    expect(result.current.isSelected('b')).toBe(false);
    expect(result.current.isSelected('c')).toBe(true);
    expect(result.current.count).toBe(2);
  });

  // ── selectAll ─────────────────────────────────────────────────────────────

  it('selectAll selects all provided IDs', () => {
    const { result } = renderHook(() => useMultiSelect());
    const ids = ['pod-1', 'pod-2', 'pod-3'];

    act(() => {
      result.current.selectAll(ids);
    });

    expect(result.current.count).toBe(3);
    expect(result.current.isSelected('pod-1')).toBe(true);
    expect(result.current.isSelected('pod-2')).toBe(true);
    expect(result.current.isSelected('pod-3')).toBe(true);
  });

  it('selectAll replaces any existing selection', () => {
    const { result } = renderHook(() => useMultiSelect());

    act(() => {
      result.current.toggle('old-item');
    });
    act(() => {
      result.current.selectAll(['new-1', 'new-2']);
    });

    expect(result.current.isSelected('old-item')).toBe(false);
    expect(result.current.count).toBe(2);
  });

  // ── clearSelection ────────────────────────────────────────────────────────

  it('clearSelection empties the selection', () => {
    const { result } = renderHook(() => useMultiSelect());

    act(() => {
      result.current.selectAll(['a', 'b', 'c']);
    });
    act(() => {
      result.current.clearSelection();
    });

    expect(result.current.count).toBe(0);
    expect(result.current.hasSelection).toBe(false);
  });

  // ── isSelected ────────────────────────────────────────────────────────────

  it('isSelected returns correct boolean', () => {
    const { result } = renderHook(() => useMultiSelect());

    act(() => {
      result.current.toggle('selected-item');
    });

    expect(result.current.isSelected('selected-item')).toBe(true);
    expect(result.current.isSelected('not-selected')).toBe(false);
  });

  // ── toggleRange ───────────────────────────────────────────────────────────

  it('toggleRange selects range between anchor and target', () => {
    const { result } = renderHook(() => useMultiSelect());
    const allIds = ['a', 'b', 'c', 'd', 'e'];

    // First toggle sets the anchor
    act(() => {
      result.current.toggle('b');
    });

    // toggleRange from anchor (b) to target (d) should select b, c, d
    act(() => {
      result.current.toggleRange('d', allIds);
    });

    expect(result.current.isSelected('a')).toBe(false);
    expect(result.current.isSelected('b')).toBe(true);
    expect(result.current.isSelected('c')).toBe(true);
    expect(result.current.isSelected('d')).toBe(true);
    expect(result.current.isSelected('e')).toBe(false);
  });

  it('toggleRange with no anchor just toggles the single item', () => {
    const { result } = renderHook(() => useMultiSelect());
    const allIds = ['a', 'b', 'c'];

    // clearSelection resets anchor to null
    act(() => {
      result.current.clearSelection();
    });

    act(() => {
      result.current.toggleRange('b', allIds);
    });

    expect(result.current.isSelected('b')).toBe(true);
    expect(result.current.count).toBe(1);
  });

  it('toggleRange works in reverse direction', () => {
    const { result } = renderHook(() => useMultiSelect());
    const allIds = ['a', 'b', 'c', 'd', 'e'];

    // Set anchor at 'd'
    act(() => {
      result.current.toggle('d');
    });

    // Range from d to b (reverse)
    act(() => {
      result.current.toggleRange('b', allIds);
    });

    expect(result.current.isSelected('a')).toBe(false);
    expect(result.current.isSelected('b')).toBe(true);
    expect(result.current.isSelected('c')).toBe(true);
    expect(result.current.isSelected('d')).toBe(true);
    expect(result.current.isSelected('e')).toBe(false);
  });

  // ── count ─────────────────────────────────────────────────────────────────

  it('count returns correct number of selected items', () => {
    const { result } = renderHook(() => useMultiSelect());

    expect(result.current.count).toBe(0);

    act(() => {
      result.current.selectAll(['x', 'y', 'z']);
    });
    expect(result.current.count).toBe(3);

    act(() => {
      result.current.toggle('x');
    });
    expect(result.current.count).toBe(2);
  });

  // ── isAllSelected / isSomeSelected ────────────────────────────────────────

  it('isAllSelected and isSomeSelected work correctly', () => {
    const { result } = renderHook(() => useMultiSelect());
    const allIds = ['a', 'b', 'c'];

    expect(result.current.isAllSelected(allIds)).toBe(false);
    expect(result.current.isSomeSelected(allIds)).toBe(false);

    act(() => {
      result.current.toggle('a');
    });

    expect(result.current.isAllSelected(allIds)).toBe(false);
    expect(result.current.isSomeSelected(allIds)).toBe(true);

    act(() => {
      result.current.selectAll(allIds);
    });

    expect(result.current.isAllSelected(allIds)).toBe(true);
    expect(result.current.isSomeSelected(allIds)).toBe(false);
  });
});
