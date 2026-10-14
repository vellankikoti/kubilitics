/**
 * Task 13 (final crash sweep): `transformResource` in Pods.tsx accesses
 * `resource.metadata.name` / `.namespace` / `.creationTimestamp` with no
 * optional chaining (`resource.metadata as unknown as Record<string,
 * unknown>).deletionTimestamp` on the very next line over, inconsistently,
 * DOES guard one field — `resource.metadata?.uid` — right next to the
 * unguarded ones, which is exactly the kind of inconsistency that preceded
 * the two real P0s already found this session (file-browser response-shape
 * crash, diagnoseWorkload unguarded `.metadata` access). `KubernetesResource`
 * declares `metadata` as required, so tsc never flags this — same blind spot.
 *
 * This test proves whether a real backend list response containing one
 * well-formed pod and one pod missing `metadata` entirely (a realistic
 * backend data-integrity gap, not a hypothetical) crashes the whole Pods
 * list page, using the SAME real render harness as Pods.test.tsx (real
 * hooks, only the service layer mocked).
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
    const state: Record<string, unknown> = {
      backendBaseUrl: 'http://localhost:8190',
      isBackendConfigured: () => true,
    };
    return selector ? selector(state) : state;
  },
  getEffectiveBackendBaseUrl: () => 'http://localhost:8190',
}));

vi.mock('@/stores/clusterPresenceStore', () => ({
  useActiveCluster: () => ({ id: 'test-cluster-id', name: 'test-cluster', serverUrl: 'https://test', provider: '' }),
  getActiveCluster: () => ({ id: 'test-cluster-id', name: 'test-cluster', serverUrl: 'https://test', provider: '' }),
}));

vi.mock('@/hooks/useActiveClusterId', () => ({
  useActiveClusterId: () => 'test-cluster-id',
}));

vi.mock('@/hooks/useConnectionStatus', () => ({
  useConnectionStatus: () => ({ isConnected: true }),
}));

vi.mock('@/hooks/useNamespacesFromCluster', () => ({
  useNamespacesFromCluster: () => ({ data: ['default'], isLoading: false }),
}));

vi.mock('@/hooks/useClusterSummary', () => ({
  useClusterSummaryWithProject: () => ({ data: undefined }),
  useClusterSummary: () => ({ data: undefined }),
}));

// One well-formed pod, one realistically malformed: metadata entirely
// absent. Real-world analog: a backend data-integrity gap or a CRD-adjacent
// resource whose metadata didn't round-trip — not a contrived shape.
const wellFormedPod = {
  apiVersion: 'v1',
  kind: 'Pod',
  metadata: { name: 'pod-a', namespace: 'default', uid: 'uid-a', creationTimestamp: new Date().toISOString() },
  spec: { containers: [{ name: 'c1', image: 'nginx:latest' }] },
  status: { phase: 'Running', containerStatuses: [{ name: 'c1', ready: true, restartCount: 0, state: { running: {} } }] },
};

const malformedPodNoMetadata = {
  apiVersion: 'v1',
  kind: 'Pod',
  // metadata intentionally omitted
  spec: { containers: [{ name: 'c1', image: 'nginx:latest' }] },
  status: { phase: 'Running', containerStatuses: [] },
};

const mockListResources = vi.fn(async () => ({
  items: [wellFormedPod, malformedPodNoMetadata],
  metadata: { total: 2 },
}));

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
  getPodMetrics: vi.fn(async () => ({ CPU: '10m', Memory: '20Mi' })),
  postShellCommand: vi.fn(async () => ({ output: '' })),
  getMetricsSummary: vi.fn(async () => ({})),
}));

vi.mock('@/components/ui/sonner', () => ({
  toast: { success: vi.fn(), error: vi.fn(), info: vi.fn() },
}));

function createQueryClient() {
  return new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 }, mutations: { retry: false } } });
}

let PodsPage: React.ComponentType;

beforeEach(async () => {
  vi.clearAllMocks();
  mockListResources.mockImplementation(async () => ({ items: [wellFormedPod, malformedPodNoMetadata], metadata: { total: 2 } }));
  PodsPage = (await import('./Pods')).default;
});

afterEach(() => {
  cleanup();
  document.body.innerHTML = '';
});

describe('Pods page — crash risk from a list item missing metadata', () => {
  it('renders the well-formed pod without throwing, even when another item in the same list lacks metadata', async () => {
    const queryClient = createQueryClient();
    render(
      <QueryClientProvider client={queryClient}>
        <TooltipProvider>
          <MemoryRouter initialEntries={['/pods']}>
            <PodsPage />
          </MemoryRouter>
        </TooltipProvider>
      </QueryClientProvider>,
    );

    await screen.findByText('pod-a');
  });
});
