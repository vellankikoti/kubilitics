/**
 * Independent re-verification (Task 12): DeploymentDetail.tsx had zero test
 * coverage before this (confirmed — no *Detail.test.tsx exists anywhere in
 * src/pages/). tsc flags 7 TS2339 errors on `.spec`/`.matchLabels` access
 * through ctx.resource (typed loosely). A prior code-reading pass traced
 * every access chain and found `?.` at every link from `.spec` onward, with
 * `ctx.resource` itself guaranteed to be at least `{}` (never null/undefined)
 * per useResourceDetail's `resource ?? ({} as T)` fallback. This test
 * actually renders the real page component with that exact `{}` shape
 * (the genuine disconnected-cluster runtime value) and confirms no throw,
 * rather than trusting the trace.
 */
import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom/vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import React from 'react';

let mockResource: Record<string, unknown> | undefined = {};
let mockIsConnected = false;

vi.mock('@/hooks/useK8sResourceDetail', () => ({
  useResourceDetail: () => ({
    resource: mockResource,
    isLoading: false,
    error: null,
    age: 'unknown',
    yaml: '',
    isConnected: mockIsConnected,
    refetch: vi.fn(),
  }),
  useResourceEvents: () => ({ events: [] }),
}));

vi.mock('@/hooks/useKubernetes', () => ({
  useDeleteK8sResource: () => ({ mutateAsync: vi.fn() }),
  useUpdateK8sResource: () => ({ mutateAsync: vi.fn() }),
  useK8sResource: () => ({ data: undefined, isLoading: false }),
  useK8sResourceList: () => ({ data: { items: [] }, isLoading: false }),
  usePatchK8sResource: () => ({ mutateAsync: vi.fn() }),
}));

vi.mock('@/hooks/useMutationPolling', () => ({
  useMutationPolling: () => ({ refetchInterval: false, isFastPolling: false, triggerFastPolling: vi.fn() }),
}));

vi.mock('@/hooks/useConnectionStatus', () => ({
  useConnectionStatus: () => ({ isConnected: mockIsConnected }),
}));

vi.mock('@/hooks/useActiveClusterId', () => ({
  useActiveClusterId: () => 'test-cluster-id',
}));

vi.mock('@/hooks/useAIContext', () => ({
  useAIContext: () => {},
}));

vi.mock('@/stores/clusterAppearance', () => ({
  getClusterAppearance: () => ({ color: '', environment: '', alias: '' }),
  setClusterAppearance: vi.fn(),
  getEnvBadgeLabel: () => null,
  getEnvBadgeClasses: () => '',
}));

vi.mock('@/stores/clusterPresenceStore', () => ({
  useActiveCluster: () => ({ id: 'test-cluster-id', name: 'test-cluster', serverUrl: 'https://test', provider: '' }),
  getActiveCluster: () => ({ id: 'test-cluster-id', name: 'test-cluster', serverUrl: 'https://test', provider: '' }),
  useClusterPresenceStore: () => ({ discovered: [], registered: [], connected: [], availableClusters: () => [] }),
  setActiveClusterBySessionId: vi.fn(),
  __resetForTest: vi.fn(),
}));

vi.mock('@/stores/backendConfigStore', () => ({
  useBackendConfigStore: (selector: (s: Record<string, unknown>) => unknown) =>
    selector({ isBackendConfigured: () => false, backendBaseUrl: 'http://localhost:8190' }),
  getEffectiveBackendBaseUrl: () => 'http://localhost:8190',
}));

vi.mock('@/components/layout/Breadcrumbs', () => ({
  Breadcrumbs: () => <div data-testid="breadcrumbs" />,
  useDetailBreadcrumbs: () => [{ label: 'Test' }],
}));

vi.mock('@/utils/resourceKindMapper', () => ({ normalizeKindForTopology: (kind: string) => kind }));
vi.mock('@/lib/exportUtils', () => ({ downloadResourceJson: vi.fn() }));
vi.mock('@/components/ui/sonner', () => ({ toast: { success: vi.fn(), error: vi.fn() } }));
vi.mock('@/lib/notificationFormatter', () => ({
  normalizeError: () => ({ title: 'Error', description: 'x', details: 'x' }),
  notifyError: vi.fn(),
  notifySuccess: vi.fn(),
}));
vi.mock('@/services/backendApiClient', () => ({
  BackendApiError: class BackendApiError extends Error {
    status: number;
    constructor(msg: string, status: number) { super(msg); this.status = status; this.name = 'BackendApiError'; }
  },
  getDeploymentRolloutHistory: vi.fn().mockResolvedValue({ revisions: [] }),
  postDeploymentRollback: vi.fn(),
  getResourceEvents: vi.fn().mockResolvedValue([]),
  getDeploymentMetrics: vi.fn().mockResolvedValue(null),
}));

// Render ALL custom tabs' content eagerly, same as GenericResourceDetail.test.tsx's
// template — ct.render(ctx) runs its synchronous body (the property-access chains
// under test) regardless of which tab is visually active.
vi.mock('@/components/resources', () => ({
  ResourceDetailLayout: ({ children, tabs, activeTab }: { children?: React.ReactNode; tabs?: { id: string; content?: React.ReactNode }[]; activeTab?: string }) => (
    <div data-testid="resource-detail-layout">
      {children}
      {tabs?.map((t) => <div key={t.id} data-testid={`tab-content-${t.id}`}>{t.content}</div>)}
    </div>
  ),
  YamlViewer: () => <div data-testid="yaml-viewer" />,
  EventsSection: () => <div data-testid="events-section" />,
  ActionsSection: () => <div data-testid="actions-section" />,
  DeleteConfirmDialog: () => <div data-testid="delete-dialog" />,
  ResourceTopologyView: () => <div data-testid="topology-view" />,
  ResourceComparisonView: () => <div data-testid="comparison-view" />,
  ContainersSection: ({ containers }: { containers?: unknown[] }) => <div data-testid="containers">{containers?.length ?? 0}</div>,
  LabelList: () => <div data-testid="label-list" />,
  AnnotationList: () => <div data-testid="annotation-list" />,
  MetricsDashboard: () => <div data-testid="metrics-dashboard" />,
  ScaleDialog: () => <div data-testid="scale-dialog" />,
  RolloutActionsDialog: () => <div data-testid="rollout-dialog" />,
  SectionCard: ({ children }: { children?: React.ReactNode }) => <div data-testid="section-card">{children}</div>,
  DetailRow: () => <div data-testid="detail-row" />,
  MultiPodLogViewer: () => <div data-testid="multi-pod-log-viewer" />,
  DetailPodTable: () => <div data-testid="detail-pod-table" />,
  WorkloadLogsTab: () => <div data-testid="workload-logs-tab" />,
  GenericResourceDetail: (props: { customTabs?: { id: string; render: (ctx: unknown) => React.ReactNode }[] }) => {
    const ctx = {
      resource: mockResource,
      name: (mockResource?.metadata as Record<string, unknown> | undefined)?.name ?? 'test-deployment',
      namespace: 'default',
      age: 'unknown',
      yaml: '',
      isConnected: mockIsConnected,
      refetch: () => {},
      clusterId: 'test-cluster-id',
      backendBaseUrl: '',
      isBackendConfigured: false,
    };
    // This is the exact line under test: GenericResourceDetail.tsx:700-705
    // calls ct.render(ctx) for every custom tab unconditionally on render.
    return (
      <div data-testid="generic-resource-detail">
        {props.customTabs?.map((ct) => (
          <div key={ct.id} data-testid={`custom-tab-${ct.id}`}>{ct.render(ctx)}</div>
        ))}
      </div>
    );
  },
}));

vi.mock('@/components/resources/BlastRadiusTab', () => ({ BlastRadiusTab: () => <div data-testid="blast-radius-tab" /> }));
vi.mock('@/components/resources/WorkloadTerminalTab', () => ({ WorkloadTerminalTab: () => <div data-testid="terminal-tab" /> }));
vi.mock('@/components/diagnose/DiagnosePanel', () => ({ DiagnosePanel: () => <div data-testid="diagnose-panel" /> }));

import DeploymentDetail from './DeploymentDetail';

function renderPage() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={['/deployments/default/my-deployment']}>
        <Routes>
          <Route path="/deployments/:namespace/:name" element={<DeploymentDetail />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>
  );
}

describe('DeploymentDetail — real render with the genuine disconnected/empty resource shape', () => {
  it('does not crash when resource is {} (the real useResourceDetail fallback, disconnected cluster)', () => {
    mockResource = {};
    mockIsConnected = false;
    expect(() => renderPage()).not.toThrow();
    expect(screen.getByTestId('generic-resource-detail')).toBeInTheDocument();
  });

  it('does not crash when resource has metadata but spec is entirely absent', () => {
    mockResource = { metadata: { name: 'my-deployment', namespace: 'default' } };
    mockIsConnected = true;
    expect(() => renderPage()).not.toThrow();
  });

  it('does not crash when resource.spec exists but selector/template are absent', () => {
    mockResource = { metadata: { name: 'my-deployment' }, spec: {} };
    mockIsConnected = true;
    expect(() => renderPage()).not.toThrow();
  });

  it('does not crash with a full realistic deployment shape (control case)', () => {
    mockResource = {
      metadata: { name: 'my-deployment', namespace: 'default' },
      spec: { replicas: 3, selector: { matchLabels: { app: 'my-app' } } },
      status: { readyReplicas: 3, availableReplicas: 3 },
    };
    mockIsConnected = true;
    expect(() => renderPage()).not.toThrow();
  });
});
