/**
 * Tests for useClusterUtilization — covers a real production bug: the
 * dashboard showed "753%" memory utilization instead of a sane value.
 * cpuPercent/memoryPercent had no upper clamp (every sibling percent
 * computation in the codebase, e.g. ClusterCapacity.tsx's pct() helper,
 * already clamps to [0,100]). Usage can transiently read above allocatable
 * capacity from the metrics-server sample; the fix clamps the displayed
 * percentage regardless of why the raw ratio exceeds 100%. No test existed
 * for this hook at all before this.
 */
import React from 'react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { ReactNode } from 'react';

let mockIsBackendConfigured = true;
let mockNodeItems: Array<Record<string, unknown>> = [];
let mockNodeMetrics: Record<string, { CPU?: string; Memory?: string }> = {};

vi.mock('@/stores/backendConfigStore', () => ({
  useBackendConfigStore: (selector: (s: Record<string, unknown>) => unknown) =>
    selector({ isBackendConfigured: () => mockIsBackendConfigured, backendBaseUrl: 'http://localhost:8190' }),
  getEffectiveBackendBaseUrl: (stored: string) => stored ?? 'http://localhost:8190',
}));

vi.mock('@/services/backendApiClient', () => ({
  listResources: async () => ({ items: mockNodeItems }),
  getNodeMetrics: async (_baseUrl: string, _clusterId: string, nodeName: string) =>
    mockNodeMetrics[nodeName] ?? {},
}));

function wrapper(qc: QueryClient) {
  return ({ children }: { children: ReactNode }) =>
    React.createElement(QueryClientProvider, { client: qc }, children);
}

function node(name: string, allocatableMemory: string, allocatableCpu = '4') {
  return {
    metadata: { name },
    status: { allocatable: { memory: allocatableMemory, cpu: allocatableCpu } },
  };
}

describe('useClusterUtilization', () => {
  let qc: QueryClient;

  beforeEach(async () => {
    qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    mockIsBackendConfigured = true;
    mockNodeItems = [];
    mockNodeMetrics = {};
  });

  it('clamps memoryPercent to 100 when usage exceeds reported capacity', async () => {
    mockNodeItems = [node('node-1', '1Gi')];
    // A metrics-server sample reporting more usage than allocatable
    // capacity — whatever upstream cause, the display must still clamp.
    mockNodeMetrics = { 'node-1': { Memory: '8Gi', CPU: '100m' } };

    const { useClusterUtilization } = await import('./useClusterUtilization');
    const { result } = renderHook(() => useClusterUtilization('c1'), { wrapper: wrapper(qc) });

    await waitFor(() => expect(result.current.utilization.metricsAvailable).toBe(true));
    expect(result.current.utilization.memoryPercent).toBeLessThanOrEqual(100);
    expect(result.current.utilization.memoryPercent).toBe(100);
  });

  it('clamps cpuPercent to 100 when usage exceeds reported capacity', async () => {
    mockNodeItems = [node('node-1', '1Gi', '1')];
    mockNodeMetrics = { 'node-1': { CPU: '5000m', Memory: '100Mi' } };

    const { useClusterUtilization } = await import('./useClusterUtilization');
    const { result } = renderHook(() => useClusterUtilization('c1'), { wrapper: wrapper(qc) });

    await waitFor(() => expect(result.current.utilization.metricsAvailable).toBe(true));
    expect(result.current.utilization.cpuPercent).toBeLessThanOrEqual(100);
    expect(result.current.utilization.cpuPercent).toBe(100);
  });

  it('computes a normal in-range percentage correctly (no regression from clamping)', async () => {
    mockNodeItems = [node('node-1', '10Gi')];
    mockNodeMetrics = { 'node-1': { Memory: '5Gi', CPU: '500m' } };

    const { useClusterUtilization } = await import('./useClusterUtilization');
    const { result } = renderHook(() => useClusterUtilization('c1'), { wrapper: wrapper(qc) });

    await waitFor(() => expect(result.current.utilization.metricsAvailable).toBe(true));
    expect(result.current.utilization.memoryPercent).toBe(50);
  });

  it('never returns a negative percentage', async () => {
    mockNodeItems = [node('node-1', '0')];
    mockNodeMetrics = { 'node-1': { Memory: '1Gi', CPU: '0' } };

    const { useClusterUtilization } = await import('./useClusterUtilization');
    const { result } = renderHook(() => useClusterUtilization('c1'), { wrapper: wrapper(qc) });

    await waitFor(() => expect(result.current.utilization.metricsAvailable).toBe(true));
    expect(result.current.utilization.memoryPercent).toBeGreaterThanOrEqual(0);
  });
});
