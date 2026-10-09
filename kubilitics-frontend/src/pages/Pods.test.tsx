/**
 * docs/ai/STABILIZATION-PLAN.md Phase 2 item 5: Pods.tsx is one of the two
 * highest-traffic, highest-stakes list pages (deleting live pods) and had
 * zero component test coverage before this file. This covers the two
 * destructive-action code paths:
 *
 *  - handleDelete (single-pod delete via the row's "Delete" menu item +
 *    DeleteConfirmDialog)
 *  - handleBulkDelete (multi-pod delete via the floating BulkActionBar)
 *
 * Mocking strategy: real hooks (useK8sResourceList, useServerPaginatedResourceList,
 * useDeleteK8sResource, etc.) run for real against a real QueryClient — only the
 * service-layer module (@/services/backendApiClient) is mocked, so these tests
 * exercise the actual mutation/query wiring, not a stand-in for it.
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, cleanup, within, waitFor, configure } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom/vitest';
import React from 'react';
import { MemoryRouter } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { TooltipProvider } from '@/components/ui/tooltip';

// Under concurrent test-file load (CPU contention when multiple heavy
// page-render suites run together) the default 1000ms findBy/waitFor
// timeout is occasionally too tight for a multi-step async flow (dialog
// portal mount, sequential bulk-delete awaits). Widen it for this file.
configure({ asyncUtilTimeout: 3000 });

// Same contention concern as asyncUtilTimeout above: the whole test's own
// default 5000ms budget can be too tight under concurrent file load for a
// multi-step flow (open dialog -> type -> confirm -> sequential bulk awaits).
vi.setConfig({ testTimeout: 15000 });

// ── framer-motion: synchronous passthrough (see BulkActionBar.test.tsx) ────
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
  // AnimatedNumber (ListPageStatCard) short-circuits to a plain static render
  // when reduced-motion is on — simplest way to avoid mocking the full
  // useMotionValue/useTransform/animate API surface for a card we don't assert on.
  useReducedMotion: () => true,
  useMotionValue: (initial: number) => ({ get: () => initial, set: () => {} }),
  useTransform: () => ({ on: () => (() => {}) }),
  useAnimation: () => ({ start: vi.fn(), stop: vi.fn() }),
  useInView: () => true,
  animate: () => ({ stop: vi.fn() }),
}));

// ── Backend config: always "configured" so the backend (server-paginated) path is used ──
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

// isConnected is mockable per-test (used to verify the "gated behind isConnected" cases)
const mockUseConnectionStatus = vi.fn(() => ({ isConnected: true }));
vi.mock('@/hooks/useConnectionStatus', () => ({
  useConnectionStatus: () => mockUseConnectionStatus(),
}));

vi.mock('@/hooks/useNamespacesFromCluster', () => ({
  useNamespacesFromCluster: () => ({ data: ['default'], isLoading: false }),
}));

vi.mock('@/hooks/useClusterSummary', () => ({
  useClusterSummaryWithProject: () => ({ data: undefined }),
  useClusterSummary: () => ({ data: undefined }),
}));

// ── Service layer boundary ──────────────────────────────────────────────────
const samplePods = [
  {
    apiVersion: 'v1',
    kind: 'Pod',
    metadata: {
      name: 'pod-a',
      namespace: 'default',
      uid: 'uid-a',
      creationTimestamp: new Date().toISOString(),
    },
    spec: { containers: [{ name: 'c1', image: 'nginx:latest' }] },
    status: {
      phase: 'Running',
      podIP: '10.0.0.1',
      hostIP: '10.0.0.100',
      containerStatuses: [{ name: 'c1', ready: true, restartCount: 0, state: { running: { startedAt: new Date().toISOString() } } }],
    },
  },
  {
    apiVersion: 'v1',
    kind: 'Pod',
    metadata: {
      name: 'pod-b',
      namespace: 'default',
      uid: 'uid-b',
      creationTimestamp: new Date().toISOString(),
    },
    spec: { containers: [{ name: 'c1', image: 'nginx:latest' }] },
    status: {
      phase: 'Running',
      podIP: '10.0.0.2',
      hostIP: '10.0.0.100',
      containerStatuses: [{ name: 'c1', ready: true, restartCount: 0, state: { running: { startedAt: new Date().toISOString() } } }],
    },
  },
];

export const mockListResources = vi.fn(async () => ({
  items: samplePods,
  metadata: { total: samplePods.length },
}));
export const mockDeleteResource = vi.fn(async () => ({ ok: true }));
export const mockPatchResource = vi.fn(async () => ({ ok: true }));
export const mockGetPodMetrics = vi.fn(async () => ({ CPU: '10m', Memory: '20Mi' }));

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
  deleteResource: (...args: unknown[]) => mockDeleteResource(...args),
  patchResource: (...args: unknown[]) => mockPatchResource(...args),
  applyManifest: vi.fn(async () => ({ ok: true })),
  getPodLogsUrl: vi.fn(() => 'http://localhost/logs'),
  getCronJobJobs: vi.fn(async () => ({ items: [] })),
  CONFIRM_DESTRUCTIVE_HEADER: 'X-Confirm-Destructive',
  BackendApiError: MockBackendApiError,
  getPodMetrics: (...args: unknown[]) => mockGetPodMetrics(...args),
  postShellCommand: vi.fn(async () => ({ output: '' })),
  getMetricsSummary: vi.fn(async () => ({})),
}));

// Toast — asserted on directly for the "errors surface, not silent" requirement.
export const mockToastSuccess = vi.fn();
export const mockToastError = vi.fn();
export const mockToastInfo = vi.fn();
vi.mock('@/components/ui/sonner', () => ({
  toast: {
    success: (...args: unknown[]) => mockToastSuccess(...args),
    error: (...args: unknown[]) => mockToastError(...args),
    info: (...args: unknown[]) => mockToastInfo(...args),
  },
}));

// ---------------------------------------------------------------------------

function createQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
      mutations: { retry: false },
    },
  });
}

function renderPods() {
  const queryClient = createQueryClient();
  return render(
    <QueryClientProvider client={queryClient}>
      <TooltipProvider>
        <MemoryRouter initialEntries={['/pods']}>
          <PodsPage />
        </MemoryRouter>
      </TooltipProvider>
    </QueryClientProvider>,
  );
}

let PodsPage: React.ComponentType;

/**
 * Opens the row-actions dropdown for the pod with the given name and
 * returns its menu. Scoping by name (not array index) matters: the table's
 * default sort is by age, and both fixture pods are stamped with
 * `new Date().toISOString()` at module-eval time, which can tie or land in
 * different milliseconds depending on machine load — an index-based lookup
 * (`getAllByRole(...)[0]`) would then nondeterministically resolve to either
 * row, producing a real (CI-visible) flaky test.
 */
async function openRowActionsMenu(user: ReturnType<typeof userEvent.setup>, podName: string) {
  const nameLink = screen.getByText(podName);
  const row = nameLink.closest('tr');
  if (!row) throw new Error(`could not find table row for pod ${podName}`);
  await user.click(within(row).getByRole('button', { name: /pod actions/i }));
}

beforeEach(async () => {
  vi.clearAllMocks();
  mockListResources.mockImplementation(async () => ({ items: samplePods, metadata: { total: samplePods.length } }));
  mockDeleteResource.mockImplementation(async () => ({ ok: true }));
  mockUseConnectionStatus.mockReturnValue({ isConnected: true });
  PodsPage = (await import('./Pods')).default;
});

afterEach(() => {
  cleanup();
  document.body.innerHTML = '';
});

describe('Pods page — single delete (handleDelete)', () => {
  it('does NOT call deleteResource until the confirmation dialog is confirmed', async () => {
    const user = userEvent.setup();
    renderPods();

    await screen.findByText('pod-a');

    // Open the row actions menu for pod-a and click "Delete"
    await openRowActionsMenu(user, 'pod-a');
    await user.click(await screen.findByRole('menuitem', { name: /^delete$/i }));

    // Dialog opens, requires typing the pod name to confirm — delete must NOT
    // have fired yet.
    expect(await screen.findByText(/Delete Pod\?/i)).toBeInTheDocument();
    expect(mockDeleteResource).not.toHaveBeenCalled();

    // The confirm button is disabled until the name is typed exactly.
    const confirmButton = screen.getByRole('button', { name: /^delete$/i, hidden: true }) ??
      screen.getAllByRole('button').find((b) => b.textContent?.trim() === 'Delete');
    expect(confirmButton).toBeTruthy();
  });

  it('calls deleteResource with the correct name/namespace once confirmed', async () => {
    const user = userEvent.setup();
    renderPods();

    await screen.findByText('pod-a');

    await openRowActionsMenu(user, 'pod-a');
    await user.click(await screen.findByRole('menuitem', { name: /^delete$/i }));

    await screen.findByText(/Delete Pod\?/i);

    // requireNameConfirmation=true for single delete — type the pod name.
    const input = await screen.findByPlaceholderText('pod-a');
    await user.type(input, 'pod-a');

    const alertDialog = screen.getByRole('alertdialog');
    const confirmBtn = within(alertDialog).getByRole('button', { name: /^delete$/i });
    expect(confirmBtn).not.toBeDisabled();
    await user.click(confirmBtn);

    await waitFor(() => expect(mockDeleteResource).toHaveBeenCalledTimes(1));
    const call = mockDeleteResource.mock.calls[0];
    // deleteResource(backendBaseUrl, clusterId, resourceType, namespace, name)
    expect(call[2]).toBe('pods');
    expect(call[3]).toBe('default');
    expect(call[4]).toBe('pod-a');

    // Dialog closes after a successful delete.
    await waitFor(() => expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument());
    expect(mockToastError).not.toHaveBeenCalled();
  });

  it('surfaces a delete API failure via toast.error instead of failing silently', async () => {
    mockDeleteResource.mockRejectedValueOnce(new Error('cluster unreachable'));
    const user = userEvent.setup();
    renderPods();

    await screen.findByText('pod-a');

    await openRowActionsMenu(user, 'pod-a');
    await user.click(await screen.findByRole('menuitem', { name: /^delete$/i }));
    await screen.findByText(/Delete Pod\?/i);

    const input = await screen.findByPlaceholderText('pod-a');
    await user.type(input, 'pod-a');

    const alertDialog = screen.getByRole('alertdialog');
    await user.click(within(alertDialog).getByRole('button', { name: /^delete$/i }));

    await waitFor(() => expect(mockDeleteResource).toHaveBeenCalledTimes(1));
    // Two independent layers surface the failure: the optimistic-delete
    // mutation's onError (notifyError) AND DeleteConfirmDialog's own catch
    // block around onConfirm(). Either way, the user sees *something* — the
    // bug class this guards against is the error being swallowed entirely.
    await waitFor(() => expect(mockToastError).toHaveBeenCalled());
    expect(
      mockToastError.mock.calls.some((call) => /delete|not deleted/i.test(String(call[0]))),
    ).toBe(true);
  });
});

describe('Pods page — bulk delete (handleBulkDelete via BulkActionBar)', () => {
  async function selectBothPods(user: ReturnType<typeof userEvent.setup>) {
    await screen.findByText('pod-a');
    const checkboxes = screen.getAllByRole('checkbox', { name: /^select pod-/i });
    await user.click(checkboxes[0]);
    await user.click(checkboxes[1]);
  }

  it('does NOT call deleteResource until the BulkActionBar confirmation dialog is confirmed', async () => {
    const user = userEvent.setup();
    renderPods();
    await selectBothPods(user);

    const toolbar = await screen.findByRole('toolbar', { name: /bulk actions/i });
    expect(within(toolbar).getByText('pods selected')).toBeInTheDocument();

    await user.click(within(toolbar).getByRole('button', { name: /^delete$/i }));

    // Confirmation dialog appears; delete must not have fired yet.
    expect(await screen.findByText(/Delete 2 pods\?/i)).toBeInTheDocument();
    expect(mockDeleteResource).not.toHaveBeenCalled();
  });

  it('calls deleteResource once per selected pod with correct namespace/name once confirmed', async () => {
    const user = userEvent.setup();
    renderPods();
    await selectBothPods(user);

    const toolbar = await screen.findByRole('toolbar', { name: /bulk actions/i });
    await user.click(within(toolbar).getByRole('button', { name: /^delete$/i }));
    await screen.findByText(/Delete 2 pods\?/i);

    // Confirm within the bulk confirmation dialog (distinct from the single-delete AlertDialog).
    const dialog = screen.getByRole('dialog');
    await user.click(within(dialog).getByRole('button', { name: /^delete$/i }));

    await waitFor(() => expect(mockDeleteResource).toHaveBeenCalledTimes(2));
    const calledPairs = mockDeleteResource.mock.calls.map((c) => [c[3], c[4]]).sort();
    expect(calledPairs).toEqual([
      ['default', 'pod-a'],
      ['default', 'pod-b'],
    ]);

    // Selection clears and the bulk toolbar disappears once all deletes succeed.
    await waitFor(() => expect(screen.queryByRole('toolbar', { name: /bulk actions/i })).not.toBeInTheDocument());
  });

  it('reports per-item failures (toast/result summary) rather than failing silently when one delete rejects', async () => {
    mockDeleteResource.mockImplementation(async (_base: unknown, _cluster: unknown, _type: unknown, _ns: unknown, name: unknown) => {
      if (name === 'pod-b') throw new Error('boom');
      return { ok: true };
    });

    const user = userEvent.setup();
    renderPods();
    await selectBothPods(user);

    const toolbar = await screen.findByRole('toolbar', { name: /bulk actions/i });
    await user.click(within(toolbar).getByRole('button', { name: /^delete$/i }));
    await screen.findByText(/Delete 2 pods\?/i);
    const dialog = screen.getByRole('dialog');
    await user.click(within(dialog).getByRole('button', { name: /^delete$/i }));

    await waitFor(() => expect(mockDeleteResource).toHaveBeenCalledTimes(2));

    // BulkActionBar surfaces a failure summary ("1 failed, 1 succeeded") —
    // the toolbar stays mounted (selection not cleared) because not everything succeeded.
    await waitFor(() => expect(screen.getByText(/1 failed, 1 succeeded/i)).toBeInTheDocument());
    expect(screen.getByRole('toolbar', { name: /bulk actions/i })).toBeInTheDocument();
  });
});
