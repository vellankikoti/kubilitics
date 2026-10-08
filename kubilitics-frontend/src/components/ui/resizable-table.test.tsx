/**
 * Batch 3 / Theme 3 #19 + #21: ResizableTableProvider's context value was
 * rebuilt fresh on every render (no memoization) — fanning out to every
 * consumer across the 67+ pages using this component on any unrelated
 * re-render — and every setColumnWidth call (fired on every mousemove tick
 * during a drag) wrote to localStorage synchronously with no debounce.
 * No test existed for either behavior before this change.
 */
import React from 'react';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import {
  ResizableTableProvider,
  useResizableTable,
  type ResizableColumnConfig,
} from './resizable-table';

const columnConfig: ResizableColumnConfig[] = [
  { id: 'name', defaultWidth: 200 },
  { id: 'status', defaultWidth: 120 },
];

function wrapper(tableId: string, config: ResizableColumnConfig[]) {
  return ({ children }: { children: React.ReactNode }) =>
    React.createElement(ResizableTableProvider, { tableId, columnConfig: config }, children);
}

describe('ResizableTableProvider', () => {
  beforeEach(() => {
    localStorage.clear();
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('context value is referentially stable across an unrelated re-render', () => {
    const { result, rerender } = renderHook(() => useResizableTable(), {
      wrapper: wrapper('test-table', columnConfig),
    });
    const first = result.current;
    rerender();
    expect(result.current).toBe(first);
  });

  it('context value changes identity only when a width actually changes', () => {
    const { result } = renderHook(() => useResizableTable(), {
      wrapper: wrapper('test-table', columnConfig),
    });
    const first = result.current;
    act(() => {
      result.current.setColumnWidth('name', 250);
    });
    expect(result.current).not.toBe(first);
    expect(result.current.getWidth('name')).toBe(250);
  });

  it('debounces localStorage writes — no write before the debounce delay elapses', () => {
    const { result } = renderHook(() => useResizableTable(), {
      wrapper: wrapper('debounce-table', columnConfig),
    });

    act(() => {
      result.current.setColumnWidth('name', 300);
    });
    expect(localStorage.getItem('kubilitics-resizable-table-debounce-table')).toBeNull();

    act(() => {
      vi.advanceTimersByTime(299);
    });
    expect(localStorage.getItem('kubilitics-resizable-table-debounce-table')).toBeNull();

    act(() => {
      vi.advanceTimersByTime(1);
    });
    expect(localStorage.getItem('kubilitics-resizable-table-debounce-table')).not.toBeNull();
  });

  it('rapid successive width changes (simulating a drag) persist only once, with the final value', () => {
    const { result } = renderHook(() => useResizableTable(), {
      wrapper: wrapper('drag-table', columnConfig),
    });

    act(() => {
      result.current.setColumnWidth('name', 210);
      vi.advanceTimersByTime(50);
      result.current.setColumnWidth('name', 230);
      vi.advanceTimersByTime(50);
      result.current.setColumnWidth('name', 260);
    });

    // Still within the debounce window of the last call — nothing saved yet.
    expect(localStorage.getItem('kubilitics-resizable-table-drag-table')).toBeNull();

    act(() => {
      vi.advanceTimersByTime(300);
    });

    const saved = JSON.parse(localStorage.getItem('kubilitics-resizable-table-drag-table') ?? '{}');
    expect(saved.name).toBe(260);
  });

  it('getWidth falls back to defaultWidth when nothing is stored', () => {
    const { result } = renderHook(() => useResizableTable(), {
      wrapper: wrapper('fresh-table', columnConfig),
    });
    expect(result.current.getWidth('name')).toBe(200);
    expect(result.current.getWidth('status')).toBe(120);
  });

  it('setColumnWidth clamps to the configured min/max range', () => {
    const configWithMin: ResizableColumnConfig[] = [{ id: 'name', defaultWidth: 200, minWidth: 100 }];
    const { result } = renderHook(() => useResizableTable(), {
      wrapper: wrapper('clamp-table', configWithMin),
    });

    act(() => {
      result.current.setColumnWidth('name', 10); // below minWidth
    });
    expect(result.current.getWidth('name')).toBe(100);

    act(() => {
      result.current.setColumnWidth('name', 5000); // above MAX_WIDTH (800)
    });
    expect(result.current.getWidth('name')).toBe(800);
  });

  it('loads a previously persisted width on mount', () => {
    localStorage.setItem('kubilitics-resizable-table-persisted-table', JSON.stringify({ name: 333 }));
    const { result } = renderHook(() => useResizableTable(), {
      wrapper: wrapper('persisted-table', columnConfig),
    });
    expect(result.current.getWidth('name')).toBe(333);
  });
});
