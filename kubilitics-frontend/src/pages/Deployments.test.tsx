/**
 * docs/ai/STABILIZATION-PLAN.md Phase 2 item 5: Deployments.tsx is one of the
 * two highest-traffic, highest-stakes list pages (deleting/scaling live
 * workloads) and had zero component test coverage before this file. Covers:
 *
 *  - handleDelete (single-deployment delete via the row's "Delete" menu item
 *    + DeleteConfirmDialog)
 *  - handleBulkDelete (multi-deployment delete via the floating BulkActionBar)
 *  - ScaleDialog (open, submit a new replica count, verify the patch call
 *    fires with the correct namespace/name/replicas, gated behind isConnected)
 *
 * Mocking strategy: real hooks (useK8sResourceList, useDeleteK8sResource,
 * usePatchK8sResource, etc.) run for real against a real QueryClient — only
 * the service-layer module (@/services/backendApiClient) is mocked.
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
  // when reduced-motion is on — avoids mocking the full motion-value API.
  useReducedMotion: () => true,
  useMotionValue: (initial: number) => ({ get: () => initial, set: () => {} }),
  useTransform: () => ({ on: () => (() => {}) }),
  useAnimation: () => ({ start: vi.fn(), stop: vi.fn() }),
  useInView: () => true,
  animate: () => ({ stop: vi.fn() }),
}));

// ── Backend config: always "configured" so the backend-list path is used ──
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

// isConnected is mockable per-test — needed for the "gated behind isConnected" assertions.
const mockUseConnectionStatus = vi.fn(() => ({ isConnected: true }));
vi.mock('@/hooks/useConnectionStatus', () => ({
  useConnectionStatus: () => mockUseConnectionStatus(),
}));

// ── Service layer boundary ──────────────────────────────────────────────────
function makeDeployment(name: string, replicas = 3) {
  return {
    apiVersion: 'apps/v1',
    kind: 'Deployment',
    metadata: {
      name,
      namespace: 'default',
      uid: `uid-${name}`,
      creationTimestamp: new Date().toISOString(),
      annotations: {},
    },
    spec: {
      replicas,
      strategy: { type: 'RollingUpdate', rollingUpdate: { maxSurge: '25%', maxUnavailable: '25%' } },
      template: { spec: { containers: [{ name: 'c1', image: 'nginx:latest' }] } },
    },
    status: { replicas, readyReplicas: replicas, updatedReplicas: replicas, availableReplicas: replicas, conditions: [] },
  };
}

const sampleDeployments = [makeDeployment('deploy-a'), makeDeployment('deploy-b')];

export const mockListResources = vi.fn(async () => ({
  items: sampleDeployments,
  metadata: { total: sampleDeployments.length },
}));
export const mockDeleteResource = vi.fn(async () => ({ ok: true }));
export const mockPatchResource = vi.fn(async () => ({ ok: true }));

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
  getDeploymentRolloutHistory: vi.fn(async () => ({ revisions: [] })),
  getEvents: vi.fn(async () => []),
  postDeploymentRollback: vi.fn(async () => ({ ok: true })),
  getMetricsSummary: vi.fn(async () => ({})),
}));

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

function renderDeployments() {
  const queryClient = createQueryClient();
  const buildTree = () => (
    <QueryClientProvider client={queryClient}>
      <TooltipProvider>
        <MemoryRouter initialEntries={['/deployments']}>
          <DeploymentsPage />
        </MemoryRouter>
      </TooltipProvider>
    </QueryClientProvider>
  );
  const result = render(buildTree());
  // Deployments reads useConnectionStatus() directly in its function body —
  // flipping the mock's return value alone doesn't cause a re-render. Calling
  // rerender() with a *freshly-built* element (not the cached one) is required:
  // React's bailout optimization skips re-invoking a subtree whose element is
  // referentially identical to last time, which would silently skip re-reading
  // the mock. A new element object each call avoids that bailout, forcing
  // Deployments' function body to re-run and pick up the new isConnected value
  // while preserving its component state (scaleDialog, selection, etc.).
  return { ...result, forceRerender: () => result.rerender(buildTree()) };
}

let DeploymentsPage: React.ComponentType;

/**
 * Opens the row-actions dropdown for the deployment with the given name and
 * returns its menu. Scoping by name (not array index) matters: both fixture
 * deployments are stamped with `new Date().toISOString()` at module-eval
 * time, which can tie or land in different milliseconds depending on
 * machine load — an index-based lookup (`getAllByRole(...)[0]`) could then
 * nondeterministically resolve to either row if the list's default sort key
 * depends on that timestamp, producing a real (CI-visible) flaky test.
 */
async function openRowActionsMenu(user: ReturnType<typeof userEvent.setup>, deploymentName: string) {
  const nameLink = screen.getByText(deploymentName);
  const row = nameLink.closest('tr');
  if (!row) throw new Error(`could not find table row for deployment ${deploymentName}`);
  await user.click(within(row).getByRole('button', { name: /deployment actions/i }));
}

beforeEach(async () => {
  vi.clearAllMocks();
  mockListResources.mockImplementation(async () => ({ items: sampleDeployments, metadata: { total: sampleDeployments.length } }));
  mockDeleteResource.mockImplementation(async () => ({ ok: true }));
  mockPatchResource.mockImplementation(async () => ({ ok: true }));
  mockUseConnectionStatus.mockReturnValue({ isConnected: true });
  DeploymentsPage = (await import('./Deployments')).default;
});

afterEach(() => {
  cleanup();
  document.body.innerHTML = '';
});

describe('Deployments page — single delete (handleDelete)', () => {
  it('does NOT call deleteResource until the confirmation dialog is confirmed', async () => {
    const user = userEvent.setup();
    renderDeployments();

    await screen.findByText('deploy-a');

    await openRowActionsMenu(user, 'deploy-a');
    await user.click(await screen.findByRole('menuitem', { name: /^delete$/i }));

    expect(await screen.findByText(/Delete Deployment\?/i)).toBeInTheDocument();
    expect(mockDeleteResource).not.toHaveBeenCalled();
  });

  it('calls deleteResource with the correct name/namespace once confirmed', async () => {
    const user = userEvent.setup();
    renderDeployments();

    await screen.findByText('deploy-a');

    await openRowActionsMenu(user, 'deploy-a');
    await user.click(await screen.findByRole('menuitem', { name: /^delete$/i }));
    await screen.findByText(/Delete Deployment\?/i);

    const input = await screen.findByPlaceholderText('deploy-a');
    await user.type(input, 'deploy-a');

    const alertDialog = screen.getByRole('alertdialog');
    const confirmBtn = within(alertDialog).getByRole('button', { name: /^delete$/i });
    expect(confirmBtn).not.toBeDisabled();
    await user.click(confirmBtn);

    await waitFor(() => expect(mockDeleteResource).toHaveBeenCalledTimes(1));
    const call = mockDeleteResource.mock.calls[0];
    // deleteResource(backendBaseUrl, clusterId, resourceType, namespace, name)
    expect(call[2]).toBe('deployments');
    expect(call[3]).toBe('default');
    expect(call[4]).toBe('deploy-a');
    expect(mockToastError).not.toHaveBeenCalled();
  });

  it('surfaces a delete API failure via toast.error instead of failing silently', async () => {
    mockDeleteResource.mockRejectedValueOnce(new Error('cluster unreachable'));
    const user = userEvent.setup();
    renderDeployments();

    await screen.findByText('deploy-a');

    await openRowActionsMenu(user, 'deploy-a');
    await user.click(await screen.findByRole('menuitem', { name: /^delete$/i }));
    await screen.findByText(/Delete Deployment\?/i);

    const input = await screen.findByPlaceholderText('deploy-a');
    await user.type(input, 'deploy-a');

    const alertDialog = screen.getByRole('alertdialog');
    await user.click(within(alertDialog).getByRole('button', { name: /^delete$/i }));

    await waitFor(() => expect(mockDeleteResource).toHaveBeenCalledTimes(1));
    await waitFor(() => expect(mockToastError).toHaveBeenCalled());
    expect(
      mockToastError.mock.calls.some((call) => /delete|not deleted/i.test(String(call[0]))),
    ).toBe(true);
  });

});

describe('Deployments page — bulk delete (handleBulkDelete via BulkActionBar)', () => {
  async function selectBothDeployments(user: ReturnType<typeof userEvent.setup>) {
    await screen.findByText('deploy-a');
    // First checkbox is "Select all deployments"; the rest are per-row.
    const checkboxes = screen.getAllByRole('checkbox');
    await user.click(checkboxes[1]);
    await user.click(checkboxes[2]);
  }

  it('does NOT call deleteResource until the BulkActionBar confirmation dialog is confirmed', async () => {
    const user = userEvent.setup();
    renderDeployments();
    await selectBothDeployments(user);

    const toolbar = await screen.findByRole('toolbar', { name: /bulk actions/i });
    expect(within(toolbar).getByText('deployments selected')).toBeInTheDocument();

    await user.click(within(toolbar).getByRole('button', { name: /^delete$/i }));

    expect(await screen.findByText(/Delete 2 deployments\?/i)).toBeInTheDocument();
    expect(mockDeleteResource).not.toHaveBeenCalled();
  });

  it('calls deleteResource once per selected deployment with correct namespace/name once confirmed', async () => {
    const user = userEvent.setup();
    renderDeployments();
    await selectBothDeployments(user);

    const toolbar = await screen.findByRole('toolbar', { name: /bulk actions/i });
    await user.click(within(toolbar).getByRole('button', { name: /^delete$/i }));
    await screen.findByText(/Delete 2 deployments\?/i);

    const dialog = screen.getByRole('dialog');
    await user.click(within(dialog).getByRole('button', { name: /^delete$/i }));

    await waitFor(() => expect(mockDeleteResource).toHaveBeenCalledTimes(2));
    const calledPairs = mockDeleteResource.mock.calls.map((c) => [c[3], c[4]]).sort();
    expect(calledPairs).toEqual([
      ['default', 'deploy-a'],
      ['default', 'deploy-b'],
    ]);

    await waitFor(() => expect(screen.queryByRole('toolbar', { name: /bulk actions/i })).not.toBeInTheDocument());
  });

  it('is gated behind isConnected — shows a toast and never calls deleteResource when disconnected', async () => {
    const user = userEvent.setup();
    const { forceRerender } = renderDeployments();
    await selectBothDeployments(user);

    // Disconnect *after* selecting (selection state lives in useMultiSelect,
    // independent of the item list, so it survives the item list clearing).
    // Deployments reads useConnectionStatus() in its function body, not via
    // a subscribed store, so force a re-render to pick up the new value.
    mockUseConnectionStatus.mockReturnValue({ isConnected: false });
    forceRerender();

    await screen.findByRole('toolbar', { name: /bulk actions/i });
    const deleteBtn = within(screen.getByRole('toolbar', { name: /bulk actions/i })).getByRole('button', { name: /^delete$/i });
    await user.click(deleteBtn);
    await screen.findByText(/Delete 2 deployments\?/i);
    const dialog = screen.getByRole('dialog');
    await user.click(within(dialog).getByRole('button', { name: /^delete$/i }));

    await waitFor(() => expect(mockToastError).toHaveBeenCalledWith('Connect cluster to delete deployments'));
    expect(mockDeleteResource).not.toHaveBeenCalled();
  });
});

describe('Deployments page — ScaleDialog', () => {
  it('opens from the row menu, submits a new replica count, and patches with correct namespace/name/replicas', async () => {
    const user = userEvent.setup();
    renderDeployments();

    await screen.findByText('deploy-a');

    await openRowActionsMenu(user, 'deploy-a');
    await user.click(await screen.findByRole('menuitem', { name: /^scale$/i }));

    // ScaleDialog opens showing current replica count (3, from makeDeployment default).
    expect(await screen.findByText('Scale Deployment')).toBeInTheDocument();
    const dialog = screen.getByRole('dialog', { name: /scale deployment/i });
    // Current + New both start at 3 (current replica count); the stepper
    // input is the least ambiguous way to assert the starting value.
    expect(within(dialog).getByRole('spinbutton')).toHaveValue(3);

    // Quick-scale to 5 replicas.
    await user.click(within(dialog).getByRole('button', { name: '5' }));

    await user.click(within(dialog).getByRole('button', { name: /^scale to 5$/i }));

    await waitFor(() => expect(mockPatchResource).toHaveBeenCalledTimes(1));
    const call = mockPatchResource.mock.calls[0];
    // patchResource(backendBaseUrl, clusterId, resourceType, namespace, name, patch)
    expect(call[2]).toBe('deployments');
    expect(call[3]).toBe('default');
    expect(call[4]).toBe('deploy-a');
    expect(call[5]).toEqual({ spec: { replicas: 5 } });

    await waitFor(() => expect(mockToastSuccess).toHaveBeenCalledWith('Scaled deploy-a to 5 replicas'));
  });

  it('is gated behind isConnected — shows a toast and never calls patchResource when disconnected', async () => {
    const user = userEvent.setup();
    const { forceRerender } = renderDeployments();

    await screen.findByText('deploy-a');

    await openRowActionsMenu(user, 'deploy-a');
    await user.click(await screen.findByRole('menuitem', { name: /^scale$/i }));
    await screen.findByText('Scale Deployment');

    // Disconnect while the dialog is open, then submit. scaleDialog.item is
    // local page state independent of the (now-empty) item list, so the
    // already-open ScaleDialog survives the re-render triggered below.
    mockUseConnectionStatus.mockReturnValue({ isConnected: false });
    forceRerender();

    const dialog = screen.getByRole('dialog', { name: /scale deployment/i });
    await user.click(within(dialog).getByRole('button', { name: '5' }));
    await user.click(within(dialog).getByRole('button', { name: /^scale to 5$/i }));

    await waitFor(() => expect(mockToastError).toHaveBeenCalledWith('Connect cluster to scale'));
    expect(mockPatchResource).not.toHaveBeenCalled();
  });
});
