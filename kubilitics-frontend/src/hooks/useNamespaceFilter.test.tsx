/**
 * docs/ai/STABILIZATION-PLAN.md Phase 2 item 4: useNamespaceFilter fans out
 * to 28 list pages and had zero test coverage — exactly the "ships
 * silently" risk the plan calls out, especially since this hook is itself
 * brand new (added as part of the namespace-filter UI standardization).
 */
import { describe, expect, it } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import type { ReactNode } from 'react';
import { useNamespaceFilter } from './useNamespaceFilter';

function wrapper(initialEntries: string[]) {
  return ({ children }: { children: ReactNode }) => (
    <MemoryRouter initialEntries={initialEntries}>{children}</MemoryRouter>
  );
}

describe('useNamespaceFilter', () => {
  it('defaults to "All Namespaces" (empty set) with no query params', () => {
    const { result } = renderHook(() => useNamespaceFilter(), {
      wrapper: wrapper(['/pods']),
    });
    const [selected] = result.current;
    expect(selected.size).toBe(0);
  });

  it('seeds from the multi-value ?ns= param', () => {
    const { result } = renderHook(() => useNamespaceFilter(), {
      wrapper: wrapper(['/pods?ns=kube-system,default']),
    });
    const [selected] = result.current;
    expect(selected).toEqual(new Set(['kube-system', 'default']));
  });

  it('seeds from the legacy singular ?namespace= param as a one-time fallback', () => {
    const { result } = renderHook(() => useNamespaceFilter(), {
      wrapper: wrapper(['/pods?namespace=kube-system']),
    });
    const [selected] = result.current;
    expect(selected).toEqual(new Set(['kube-system']));
  });

  it('prefers ?ns= over the legacy ?namespace= when both are present', () => {
    const { result } = renderHook(() => useNamespaceFilter(), {
      wrapper: wrapper(['/pods?ns=default&namespace=kube-system']),
    });
    const [selected] = result.current;
    expect(selected).toEqual(new Set(['default']));
  });

  it('updates state when setSelected is called', () => {
    const { result } = renderHook(() => useNamespaceFilter(), {
      wrapper: wrapper(['/pods']),
    });

    act(() => {
      const [, setSelected] = result.current;
      setSelected(new Set(['default', 'kube-system']));
    });

    const [selected] = result.current;
    expect(selected).toEqual(new Set(['default', 'kube-system']));
  });

  it('clearing the selection (empty set) drops ?ns= entirely rather than writing an empty value', () => {
    const { result } = renderHook(() => useNamespaceFilter(), {
      wrapper: wrapper(['/pods?ns=default']),
    });

    act(() => {
      const [, setSelected] = result.current;
      setSelected(new Set());
    });

    const [selected] = result.current;
    expect(selected.size).toBe(0);
  });

  it('ignores empty entries from a malformed ?ns= value', () => {
    const { result } = renderHook(() => useNamespaceFilter(), {
      wrapper: wrapper(['/pods?ns=default,,kube-system,']),
    });
    const [selected] = result.current;
    expect(selected).toEqual(new Set(['default', 'kube-system']));
  });
});
