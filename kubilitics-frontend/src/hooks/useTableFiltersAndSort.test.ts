/**
 * docs/ai/STABILIZATION-PLAN.md Phase 2 item 4: useTableFiltersAndSort fans
 * out to 53 list pages and had zero test coverage — the single
 * highest-fan-out shared hook in the app with no automated signal at all.
 */
import { describe, expect, it } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import {
  useTableFiltersAndSort,
  mapClientSortToServerSort,
  type ColumnConfig,
} from './useTableFiltersAndSort';

interface Item {
  name: string;
  status: string;
  age: string;
  replicas: number;
}

const items: Item[] = [
  { name: 'charlie', status: 'Running', age: '5m', replicas: 3 },
  { name: 'alpha', status: 'Pending', age: '2h', replicas: 1 },
  { name: 'bravo', status: 'Running', age: '3d', replicas: 2 },
];

const columns: ColumnConfig<Item>[] = [
  { columnId: 'name', getValue: (i) => i.name, sortable: true, filterable: false },
  { columnId: 'status', getValue: (i) => i.status, sortable: true, filterable: true },
  { columnId: 'age', getValue: (i) => i.age, sortable: true, filterable: false },
  { columnId: 'replicas', getValue: (i) => i.replicas, sortable: true, filterable: false },
];

describe('useTableFiltersAndSort', () => {
  it('returns items unchanged with no filter or sort applied', () => {
    const { result } = renderHook(() => useTableFiltersAndSort(items, { columns }));
    expect(result.current.filteredAndSortedItems).toEqual(items);
    expect(result.current.hasActiveFilters).toBe(false);
  });

  it('guards against a non-array items value instead of crashing', () => {
    const { result } = renderHook(() =>
      // Simulates a not-yet-loaded or malformed API payload reaching the hook.
      useTableFiltersAndSort(undefined as unknown as Item[], { columns }),
    );
    expect(result.current.filteredAndSortedItems).toEqual([]);
  });

  it('filters items by an allowed-value set on a filterable column', () => {
    const { result } = renderHook(() => useTableFiltersAndSort(items, { columns }));

    act(() => {
      result.current.setColumnFilter('status', new Set(['Running']));
    });

    expect(result.current.filteredAndSortedItems.map((i) => i.name)).toEqual(['charlie', 'bravo']);
    expect(result.current.hasActiveFilters).toBe(true);
  });

  it('clearAllFilters resets every column filter', () => {
    const { result } = renderHook(() => useTableFiltersAndSort(items, { columns }));
    act(() => {
      result.current.setColumnFilter('status', new Set(['Running']));
    });
    expect(result.current.hasActiveFilters).toBe(true);

    act(() => {
      result.current.clearAllFilters();
    });
    expect(result.current.hasActiveFilters).toBe(false);
    expect(result.current.filteredAndSortedItems).toEqual(items);
  });

  it('setColumnFilter(id, null) clears just that column', () => {
    const { result } = renderHook(() => useTableFiltersAndSort(items, { columns }));
    act(() => {
      result.current.setColumnFilter('status', new Set(['Running']));
    });
    act(() => {
      result.current.setColumnFilter('status', null);
    });
    expect(result.current.hasActiveFilters).toBe(false);
  });

  it('sorts numerically by default for a numeric column', () => {
    const { result } = renderHook(() => useTableFiltersAndSort(items, { columns }));
    act(() => {
      result.current.setSort('replicas', 'asc');
    });
    expect(result.current.filteredAndSortedItems.map((i) => i.replicas)).toEqual([1, 2, 3]);

    act(() => {
      result.current.setSort('replicas', 'desc');
    });
    expect(result.current.filteredAndSortedItems.map((i) => i.replicas)).toEqual([3, 2, 1]);
  });

  it('sorts age-like strings by actual duration, not lexical order', () => {
    // Lexically "2h" < "3d" < "5m", but the real durations are 5m < 2h < 3d.
    const { result } = renderHook(() => useTableFiltersAndSort(items, { columns }));
    act(() => {
      result.current.setSort('age', 'asc');
    });
    expect(result.current.filteredAndSortedItems.map((i) => i.age)).toEqual(['5m', '2h', '3d']);
  });

  it('clicking the same sort column twice toggles order', () => {
    const { result } = renderHook(() => useTableFiltersAndSort(items, { columns }));
    act(() => {
      result.current.setSort('name'); // first click: defaults to asc
    });
    expect(result.current.sortOrder).toBe('asc');

    act(() => {
      result.current.setSort('name'); // second click, same key: toggles
    });
    expect(result.current.sortOrder).toBe('desc');
  });

  it('switching to a different sort column resets order to asc', () => {
    const { result } = renderHook(() => useTableFiltersAndSort(items, { columns }));
    act(() => {
      result.current.setSort('name', 'desc');
    });
    act(() => {
      result.current.setSort('replicas');
    });
    expect(result.current.sortKey).toBe('replicas');
    expect(result.current.sortOrder).toBe('asc');
  });

  it('computes distinct values and counts per filterable column', () => {
    const { result } = renderHook(() => useTableFiltersAndSort(items, { columns }));
    expect(result.current.distinctValuesByColumn.status).toEqual(['Pending', 'Running']);
    expect(result.current.valueCountsByColumn.status).toEqual([
      { value: 'Pending', count: 1 },
      { value: 'Running', count: 2 },
    ]);
  });

  it('applies an initial defaultSortKey/defaultSortOrder from config', () => {
    const { result } = renderHook(() =>
      useTableFiltersAndSort(items, { columns, defaultSortKey: 'name', defaultSortOrder: 'desc' }),
    );
    expect(result.current.sortKey).toBe('name');
    expect(result.current.sortOrder).toBe('desc');
    expect(result.current.filteredAndSortedItems.map((i) => i.name)).toEqual(['charlie', 'bravo', 'alpha']);
  });
});

describe('mapClientSortToServerSort', () => {
  it('passes through name/namespace as-is', () => {
    expect(mapClientSortToServerSort('name', 'asc')).toEqual({ sortBy: 'name', sortOrder: 'asc' });
    expect(mapClientSortToServerSort('namespace', 'desc')).toEqual({ sortBy: 'namespace', sortOrder: 'desc' });
  });

  it('inverts order for the age column (client asc = newest-first)', () => {
    expect(mapClientSortToServerSort('age', 'asc')).toEqual({ sortBy: 'creationTimestamp', sortOrder: 'desc' });
    expect(mapClientSortToServerSort('age', 'desc')).toEqual({ sortBy: 'creationTimestamp', sortOrder: 'asc' });
  });

  it('falls back to creationTimestamp desc for a column the server cannot sort', () => {
    expect(mapClientSortToServerSort('cpu', 'asc')).toEqual({ sortBy: 'creationTimestamp', sortOrder: 'desc' });
    expect(mapClientSortToServerSort(null, 'asc')).toEqual({ sortBy: 'creationTimestamp', sortOrder: 'desc' });
  });

  it('applies a per-page override before falling through to the default switch', () => {
    expect(
      mapClientSortToServerSort('ready', 'asc', { ready: 'status.readyReplicas' }),
    ).toEqual({ sortBy: 'status.readyReplicas', sortOrder: 'asc' });
  });
});
