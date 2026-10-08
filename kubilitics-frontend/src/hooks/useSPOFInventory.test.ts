/**
 * Tests for useSPOFInventory — covers a reported bug: the SPOF inventory
 * page showed a hard error ("Dependency graph is still building") for up
 * to 60s on every freshly-connected cluster, even though that state is
 * transient and self-healing (the backend's graph engine starts lazily and
 * needs real time to finish its first build). No test existed for this
 * hook before this change.
 */
import React from 'react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { ReactNode } from 'react';
import { useSPOFInventory } from './useSPOFInventory';
import { BackendApiError } from '@/services/api/client';

let mockGetSPOFInventory: ReturnType<typeof vi.fn>;

vi.mock('@/services/api/spof', () => ({
  getSPOFInventory: (...args: unknown[]) => mockGetSPOFInventory(...args),
}));

vi.mock('@/hooks/useActiveClusterId', () => ({
  useActiveClusterId: () => 'c1',
}));

vi.mock('@/stores/backendConfigStore', () => ({
  useBackendConfigStore: (selector: (s: Record<string, unknown>) => unknown) =>
    selector({ isBackendConfigured: () => true, backendBaseUrl: 'http://localhost:8190' }),
  getEffectiveBackendBaseUrl: (stored: string) => stored ?? 'http://localhost:8190',
}));

function wrapper(qc: QueryClient) {
  return ({ children }: { children: ReactNode }) =>
    React.createElement(QueryClientProvider, { client: qc }, children);
}

describe('useSPOFInventory', () => {
  let qc: QueryClient;

  beforeEach(() => {
    qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    mockGetSPOFInventory = vi.fn();
  });

  it('treats the "still building" 503 as a soft state, not an error', async () => {
    mockGetSPOFInventory.mockRejectedValue(
      new BackendApiError('Dependency graph is still building', 503),
    );

    const { result } = renderHook(() => useSPOFInventory(), { wrapper: wrapper(qc) });

    await waitFor(() => expect(result.current.isGraphBuilding).toBe(true));
    expect(result.current.error).toBeNull();
  });

  it('surfaces a genuine error (not graph-building) as a real error', async () => {
    mockGetSPOFInventory.mockRejectedValue(
      new BackendApiError('Internal server error', 500),
    );

    const { result } = renderHook(() => useSPOFInventory(), { wrapper: wrapper(qc) });

    await waitFor(() => expect(result.current.error).not.toBeNull(), { timeout: 5_000 });
    expect(result.current.isGraphBuilding).toBe(false);
    expect(result.current.error?.message).toBe('Internal server error');
  });

  it('does not treat a 503 with a different message as graph-building', async () => {
    mockGetSPOFInventory.mockRejectedValue(
      new BackendApiError('Service temporarily unavailable', 503),
    );

    const { result } = renderHook(() => useSPOFInventory(), { wrapper: wrapper(qc) });

    await waitFor(() => expect(result.current.error).not.toBeNull(), { timeout: 5_000 });
    expect(result.current.isGraphBuilding).toBe(false);
  });

  it('clears the building state once data loads successfully', async () => {
    let callCount = 0;
    mockGetSPOFInventory.mockImplementation(() => {
      callCount++;
      if (callCount === 1) {
        return Promise.reject(new BackendApiError('Dependency graph is still building', 503));
      }
      return Promise.resolve({ items: [], total_spofs: 0, critical: 0, high: 0, medium: 0, low: 0 });
    });

    const { result } = renderHook(() => useSPOFInventory(), { wrapper: wrapper(qc) });

    await waitFor(() => expect(result.current.isGraphBuilding).toBe(true));

    await result.current.refetch();

    await waitFor(() => expect(result.current.isGraphBuilding).toBe(false));
    expect(result.current.error).toBeNull();
    expect(result.current.data?.total_spofs).toBe(0);
  });
});
