/**
 * Tests for src/hooks/useResourceCounts.ts
 *
 * Covers: mock counts when disconnected, backend summary mapping,
 * direct K8s fallback, and the summaryMap field coverage.
 */
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook } from '@testing-library/react';

// ---- Mock controls ----
let mockIsConnected = false;
let mockIsBackendConfigured = false;
let mockCurrentClusterId: string | null = null;
let mockSummaryData: Record<string, number> | null = null;
let mockK8sData: Record<string, unknown> = {};

vi.mock('@/hooks/useConnectionStatus', () => ({
  useConnectionStatus: () => ({ isConnected: mockIsConnected }),
}));

vi.mock('@/stores/backendConfigStore', () => ({
  useBackendConfigStore: (selector: (s: Record<string, unknown>) => unknown) => {
    const state = {
      isBackendConfigured: () => mockIsBackendConfigured,
      currentClusterId: mockCurrentClusterId,
    };
    return selector(state);
  },
  getEffectiveBackendBaseUrl: () => 'http://localhost:8190',
}));

vi.mock('@/hooks/useClusterSummary', () => ({
  useClusterSummaryWithProject: (clusterId?: string) => ({
    data: clusterId ? mockSummaryData : null,
    isLoading: false,
  }),
}));

vi.mock('@/hooks/useKubernetes', () => ({
  useK8sResourceList: (resourceType: string, _ns: unknown, options: Record<string, unknown> | undefined) => {
    if (options?.enabled === false) {
      return { data: null, isLoading: false, isPlaceholderData: false };
    }
    const data = mockK8sData[resourceType];
    return {
      data: data ?? null,
      isLoading: false,
      isPlaceholderData: false,
    };
  },
}));

import { useResourceCounts, type ResourceCounts } from './useResourceCounts';

beforeEach(() => {
  mockIsConnected = false;
  mockIsBackendConfigured = false;
  mockCurrentClusterId = null;
  mockSummaryData = null;
  mockK8sData = {};
});

// ============================================================================
// Disconnected mode — returns zero counts (no fake data)
// ============================================================================

describe('useResourceCounts — disconnected', () => {
  it('returns zero counts when disconnected', () => {
    mockIsConnected = false;

    const { result } = renderHook(() => useResourceCounts());
    const { counts, isConnected } = result.current;

    expect(isConnected).toBe(false);
    expect(counts.pods).toBe(0);
    expect(counts.deployments).toBe(0);
    expect(counts.services).toBe(0);
    expect(counts.nodes).toBe(0);
    expect(counts.namespaces).toBe(0);
    expect(counts.jobs).toBe(0);
    expect(counts.cronjobs).toBe(0);
    expect(counts.secrets).toBe(0);
  });
});

// ============================================================================
// Backend summary mode
// ============================================================================

describe('useResourceCounts — backend summary', () => {
  it('uses backend summary when backend is configured', () => {
    mockIsConnected = true;
    mockIsBackendConfigured = true;
    mockCurrentClusterId = 'test-cluster';
    mockSummaryData = {
      pod_count: 10,
      deployment_count: 5,
      service_count: 3,
      node_count: 2,
      namespace_count: 4,
      statefulset_count: 1,
      replicaset_count: 6,
      daemonset_count: 2,
      job_count: 100,
      cronjob_count: 3,
      ingress_count: 7,
      ingressclass_count: 1,
      endpoint_count: 8,
      endpointslice_count: 9,
      networkpolicy_count: 2,
      configmap_count: 20,
      secret_count: 15,
      persistentvolume_count: 3,
      persistentvolumeclaim_count: 4,
      storageclass_count: 2,
      serviceaccount_count: 50,
      role_count: 10,
      clusterrole_count: 40,
      rolebinding_count: 12,
      clusterrolebinding_count: 30,
      hpa_count: 5,
      limitrange_count: 1,
      resourcequota_count: 2,
      poddisruptionbudget_count: 3,
      priorityclass_count: 2,
      customresourcedefinition_count: 8,
      mutatingwebhookconfiguration_count: 4,
      validatingwebhookconfiguration_count: 2,
    };

    const { result } = renderHook(() => useResourceCounts());
    const { counts, isConnected } = result.current;

    expect(isConnected).toBe(true);
    expect(counts.pods).toBe(10);
    expect(counts.deployments).toBe(5);
    expect(counts.services).toBe(3);
    expect(counts.nodes).toBe(2);
    expect(counts.namespaces).toBe(4);
    expect(counts.statefulsets).toBe(1);
    expect(counts.replicasets).toBe(6);
    expect(counts.daemonsets).toBe(2);
    expect(counts.jobs).toBe(100);
    expect(counts.cronjobs).toBe(3);
    expect(counts.ingresses).toBe(7);
    expect(counts.ingressclasses).toBe(1);
    expect(counts.endpoints).toBe(8);
    expect(counts.endpointslices).toBe(9);
    expect(counts.networkpolicies).toBe(2);
    expect(counts.configmaps).toBe(20);
    expect(counts.secrets).toBe(15);
    expect(counts.persistentvolumes).toBe(3);
    expect(counts.persistentvolumeclaims).toBe(4);
    expect(counts.storageclasses).toBe(2);
    expect(counts.serviceaccounts).toBe(50);
    expect(counts.roles).toBe(10);
    expect(counts.clusterroles).toBe(40);
    expect(counts.rolebindings).toBe(12);
    expect(counts.clusterrolebindings).toBe(30);
    expect(counts.horizontalpodautoscalers).toBe(5);
    expect(counts.limitranges).toBe(1);
    expect(counts.resourcequotas).toBe(2);
    expect(counts.poddisruptionbudgets).toBe(3);
    expect(counts.priorityclasses).toBe(2);
    expect(counts.customresourcedefinitions).toBe(8);
    expect(counts.mutatingwebhookconfigurations).toBe(4);
    expect(counts.validatingwebhookconfigurations).toBe(2);
  });

  it('returns 0 for types not in summary and without direct K8s data', () => {
    mockIsConnected = true;
    mockIsBackendConfigured = true;
    mockCurrentClusterId = 'test-cluster';
    // Minimal summary — omits many keys
    mockSummaryData = { pod_count: 5 };

    const { result } = renderHook(() => useResourceCounts());
    const { counts } = result.current;

    // pods mapped from summary
    expect(counts.pods).toBe(5);
    // Types not in summary and no direct K8s data fall to 0
    expect(counts.volumesnapshots).toBe(0);
    expect(counts.volumesnapshotclasses).toBe(0);
    expect(counts.apiservices).toBe(0);
    expect(counts.leases).toBe(0);
  });
});

// ============================================================================
// Direct K8s fallback mode
// ============================================================================

describe('useResourceCounts — direct K8s fallback', () => {
  it('uses item counts from direct K8s queries when no backend', () => {
    mockIsConnected = true;
    mockIsBackendConfigured = false;
    mockCurrentClusterId = null;

    // Simulate K8s list responses
    mockK8sData = {
      pods: { items: Array(7).fill({}), metadata: {} },
      deployments: { items: Array(3).fill({}), metadata: {} },
      services: { items: Array(2).fill({}), metadata: {} },
      nodes: { items: Array(1).fill({}), metadata: {} },
    };

    const { result } = renderHook(() => useResourceCounts());
    const { counts } = result.current;

    expect(counts.pods).toBe(7);
    expect(counts.deployments).toBe(3);
    expect(counts.services).toBe(2);
    expect(counts.nodes).toBe(1);
    // Types with no data return 0
    expect(counts.ingresses).toBe(0);
  });

  it('prefers metadata.total over items.length', () => {
    mockIsConnected = true;
    mockIsBackendConfigured = false;

    mockK8sData = {
      pods: { items: Array(5).fill({}), metadata: { total: 42 } },
    };

    const { result } = renderHook(() => useResourceCounts());
    expect(result.current.counts.pods).toBe(42);
  });

  it('returns isConnected true when connected', () => {
    mockIsConnected = true;
    mockIsBackendConfigured = false;

    const { result } = renderHook(() => useResourceCounts());
    expect(result.current.isConnected).toBe(true);
  });
});
