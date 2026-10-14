/**
 * Task 13 (final crash sweep): same bug class as Pods.tsx — transformResource
 * accesses `resource.metadata.name/namespace/creationTimestamp` with no
 * optional chaining, plus a raw `(resource.metadata as unknown as Record<
 * string, unknown>).ownerReferences` cast on the same unguarded field. One
 * malformed list item (missing metadata) crashes the whole ReplicaSets page.
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, cleanup, configure } from '@testing-library/react';
import '@testing-library/jest-dom/vitest';
import React from 'react';
import { MemoryRouter } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { TooltipProvider } from '@/components/ui/tooltip';

configure({ asyncUtilTimeout: 3000 });
vi.setConfig({ testTimeout: 15000 });

vi.mock('framer-motion', () => ({
  AnimatePresence: ({ children }: { children?: React.ReactNode }) => <>{children}</>,
  motion: new Proxy({}, {
    get: (_target, prop) => {
      return React.forwardRef((props: Record<string, unknown>, ref: React.Ref<HTMLElement>) => {
        const { variants, initial, animate, whileHover, whileTap, whileInView, exit, layout, layoutId, transition, ...rest } = props;
        const Tag = String(prop) as keyof JSX.IntrinsicElements;
        return React.createElement(Tag, { ...rest, ref });
      });
    },
  }),
  useReducedMotion: () => true,
  useMotionValue: (initial: number) => ({ get: () => initial, set: () => {} }),
  useTransform: () => ({ on: () => (() => {}) }),
  useAnimation: () => ({ start: vi.fn(), stop: vi.fn() }),
  useInView: () => true,
  animate: () => ({ stop: vi.fn() }),
}));

vi.mock('@/stores/backendConfigStore', () => ({
  useBackendConfigStore: (selector?: (s: Record<string, unknown>) => unknown) => {
    const state: Record<string, unknown> = { backendBaseUrl: 'http://localhost:8190', isBackendConfigured: () => true };
    return selector ? selector(state) : state;
  },
  getEffectiveBackendBaseUrl: () => 'http://localhost:8190',
}));

vi.mock('@/stores/clusterPresenceStore', () => ({
  useActiveCluster: () => ({ id: 'test-cluster-id', name: 'test-cluster', serverUrl: 'https://test', provider: '' }),
  getActiveCluster: () => ({ id: 'test-cluster-id', name: 'test-cluster', serverUrl: 'https://test', provider: '' }),
}));

vi.mock('@/hooks/useActiveClusterId', () => ({ useActiveClusterId: () => 'test-cluster-id' }));
vi.mock('@/hooks/useConnectionStatus', () => ({ useConnectionStatus: () => ({ isConnected: true }) }));
vi.mock('@/hooks/useNamespacesFromCluster', () => ({ useNamespacesFromCluster: () => ({ data: ['default'], isLoading: false }) }));
vi.mock('@/hooks/useClusterSummary', () => ({
  useClusterSummaryWithProject: () => ({ data: undefined }),
  useClusterSummary: () => ({ data: undefined }),
}));

const wellFormedRS = {
  apiVersion: 'apps/v1',
  kind: 'ReplicaSet',
  metadata: { name: 'rs-a', namespace: 'default', uid: 'uid-a', creationTimestamp: new Date().toISOString() },
  spec: { replicas: 1 },
  status: { replicas: 1, readyReplicas: 1 },
};

const malformedRSNoMetadata = {
  apiVersion: 'apps/v1',
  kind: 'ReplicaSet',
  spec: { replicas: 0 },
  status: {},
};

const mockListResources = vi.fn(async () => ({ items: [wellFormedRS, malformedRSNoMetadata], metadata: { total: 2 } }));

class MockBackendApiError extends Error {
  status: number;
  constructor(message: string, status = 500) {
    super(message);
    this.status = status;
  }
}

vi.mock('@/services/backendApiClient', () => ({
  listResources: (...args: unknown[]) => mockListResources(...args),
  getResource: vi.fn(async () => ({})),
  deleteResource: vi.fn(async () => ({ ok: true })),
  patchResource: vi.fn(async () => ({ ok: true })),
  applyManifest: vi.fn(async () => ({ ok: true })),
  getPodLogsUrl: vi.fn(() => 'http://localhost/logs'),
  getCronJobJobs: vi.fn(async () => ({ items: [] })),
  CONFIRM_DESTRUCTIVE_HEADER: 'X-Confirm-Destructive',
  BackendApiError: MockBackendApiError,
  getPodMetrics: vi.fn(async () => ({})),
  postShellCommand: vi.fn(async () => ({ output: '' })),
  getMetricsSummary: vi.fn(async () => ({})),
}));

vi.mock('@/components/ui/sonner', () => ({ toast: { success: vi.fn(), error: vi.fn(), info: vi.fn() } }));

function createQueryClient() {
  return new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 }, mutations: { retry: false } } });
}

let ReplicaSetsPage: React.ComponentType;

beforeEach(async () => {
  vi.clearAllMocks();
  mockListResources.mockImplementation(async () => ({ items: [wellFormedRS, malformedRSNoMetadata], metadata: { total: 2 } }));
  ReplicaSetsPage = (await import('./ReplicaSets')).default;
});

afterEach(() => {
  cleanup();
  document.body.innerHTML = '';
});

describe('ReplicaSets page — crash risk from a list item missing metadata', () => {
  it('renders the well-formed replicaset without throwing, even when another item lacks metadata', async () => {
    const queryClient = createQueryClient();
    render(
      <QueryClientProvider client={queryClient}>
        <TooltipProvider>
          <MemoryRouter initialEntries={['/replicasets']}>
            <ReplicaSetsPage />
          </MemoryRouter>
        </TooltipProvider>
      </QueryClientProvider>,
    );

    await screen.findByText('rs-a');
  });
});
