/**
 * Unit tests for src/hooks/useBlastRadius.ts
 *
 * Covers: successful data return, 404 isUnavailable detection,
 * error handling, disabled when missing params.
 */
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { createElement } from 'react';
import type { BlastRadiusResult } from '@/services/api/types';

// ── Mock controls ────────────────────────────────────────────────────────────

let mockClusterId: string | null = 'cluster-1';
let mockBackendBaseUrl = 'http://localhost:8190';
let mockIsBackendConfigured = true;
let mockGetBlastRadiusResult: BlastRadiusResult | Error = {
  criticalityScore: 75,
  level: 'high',
  blastRadiusPercent: 42,
  fanIn: 3,
  fanOut: 5,
  isSPOF: false,
  affectedResources: [],
  dependencyChain: [],
};

vi.mock('./useActiveClusterId', () => ({
  useActiveClusterId: () => mockClusterId,
}));

vi.mock('@/stores/backendConfigStore', () => ({
  useBackendConfigStore: (selector: (s: Record<string, unknown>) => unknown) => {
    const state = {
      backendBaseUrl: mockBackendBaseUrl,
      isBackendConfigured: () => mockIsBackendConfigured,
    };
    return selector(state);
  },
  getEffectiveBackendBaseUrl: (url: string) => url,
}));

vi.mock('@/services/backendApiClient', () => ({
  getBlastRadius: vi.fn(async () => {
    if (mockGetBlastRadiusResult instanceof Error) {
      throw mockGetBlastRadiusResult;
    }
    return mockGetBlastRadiusResult;
  }),
}));

import { useBlastRadius } from './useBlastRadius';

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        gcTime: 0,
      },
    },
  });
  return ({ children }: { children: React.ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children);
}

describe('useBlastRadius', () => {
  beforeEach(() => {
    mockClusterId = 'cluster-1';
    mockBackendBaseUrl = 'http://localhost:8190';
    mockIsBackendConfigured = true;
    mockGetBlastRadiusResult = {
      criticalityScore: 75,
      level: 'high',
      blastRadiusPercent: 42,
      fanIn: 3,
      fanOut: 5,
      isSPOF: false,
      affectedResources: [],
      dependencyChain: [],
    };
  });

  it('returns data when API is available', async () => {
    const { result } = renderHook(
      () => useBlastRadius({ kind: 'Deployment', namespace: 'default', name: 'nginx' }),
      { wrapper: createWrapper() },
    );

    await waitFor(() => {
      expect(result.current.data).toBeDefined();
    });

    expect(result.current.data?.criticalityScore).toBe(75);
    expect(result.current.data?.level).toBe('high');
    expect(result.current.isUnavailable).toBe(false);
    expect(result.current.error).toBeNull();
  });

  it('returns isUnavailable = true when API returns 404', async () => {
    const notFoundError = new Error('Not Found') as Error & { status: number };
    notFoundError.status = 404;
    mockGetBlastRadiusResult = notFoundError;

    const { result } = renderHook(
      () => useBlastRadius({ kind: 'Deployment', namespace: 'default', name: 'nginx' }),
      { wrapper: createWrapper() },
    );

    await waitFor(() => {
      expect(result.current.isUnavailable).toBe(true);
    });

    expect(result.current.data).toBeUndefined();
    // error should be null when isUnavailable (the hook nullifies it)
    expect(result.current.error).toBeNull();
  });

  it('returns error for non-404 failures', async () => {
    const serverError = new Error('Internal Server Error') as Error & { status: number };
    serverError.status = 500;
    mockGetBlastRadiusResult = serverError;

    const { result } = renderHook(
      () => useBlastRadius({ kind: 'Deployment', namespace: 'default', name: 'nginx' }),
      { wrapper: createWrapper() },
    );

    // The hook has a custom retry function that retries up to 2 times for non-404 errors.
    // Wait long enough for retries to exhaust (retryDelay: 1000ms, 2 retries).
    await waitFor(
      () => {
        expect(result.current.error).toBeTruthy();
      },
      { timeout: 10000 },
    );

    expect(result.current.isUnavailable).toBe(false);
    expect(result.current.error?.message).toBe('Internal Server Error');
  });

  it('does not fetch when enabled = false', () => {
    const { result } = renderHook(
      () => useBlastRadius({ kind: 'Deployment', namespace: 'default', name: 'nginx', enabled: false }),
      { wrapper: createWrapper() },
    );

    expect(result.current.isLoading).toBe(false);
    expect(result.current.data).toBeUndefined();
  });

  it('does not fetch when clusterId is null', () => {
    mockClusterId = null;

    const { result } = renderHook(
      () => useBlastRadius({ kind: 'Deployment', namespace: 'default', name: 'nginx' }),
      { wrapper: createWrapper() },
    );

    expect(result.current.isLoading).toBe(false);
    expect(result.current.data).toBeUndefined();
  });

  it('does not fetch when name is empty', () => {
    const { result } = renderHook(
      () => useBlastRadius({ kind: 'Deployment', namespace: 'default', name: '' }),
      { wrapper: createWrapper() },
    );

    expect(result.current.isLoading).toBe(false);
    expect(result.current.data).toBeUndefined();
  });

  it('does not fetch when backend is not configured', () => {
    mockIsBackendConfigured = false;

    const { result } = renderHook(
      () => useBlastRadius({ kind: 'Deployment', namespace: 'default', name: 'nginx' }),
      { wrapper: createWrapper() },
    );

    expect(result.current.isLoading).toBe(false);
    expect(result.current.data).toBeUndefined();
  });

  it('handles null namespace gracefully', async () => {
    const { result } = renderHook(
      () => useBlastRadius({ kind: 'Node', namespace: null, name: 'worker-1' }),
      { wrapper: createWrapper() },
    );

    await waitFor(() => {
      expect(result.current.data).toBeDefined();
    });

    expect(result.current.data?.criticalityScore).toBe(75);
  });
});
