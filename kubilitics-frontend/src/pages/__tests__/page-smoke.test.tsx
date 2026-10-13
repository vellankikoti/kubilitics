/**
 * Smoke tests for major Kubilitics frontend pages.
 *
 * Goal: verify each page renders without crashing and shows basic content.
 * All external hooks/stores are mocked to avoid network calls.
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, cleanup, fireEvent, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom/vitest';
import React from 'react';
import { MemoryRouter } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

// ---------------------------------------------------------------------------
// Shared mocks — keep minimal, just enough to prevent crashes
// ---------------------------------------------------------------------------

// Mock framer-motion to avoid animation issues in tests
vi.mock('framer-motion', () => ({
  motion: new Proxy({}, {
    get: (_target, prop) => {
      // Return a forwardRef component for any HTML element (div, section, etc.)
      return React.forwardRef((props: Record<string, unknown>, ref: React.Ref<HTMLElement>) => {
        const { variants, initial, animate, whileHover, whileTap, whileInView, exit, layout, layoutId, transition, ...rest } = props;
        const Tag = String(prop) as keyof JSX.IntrinsicElements;
        return React.createElement(Tag, { ...rest, ref });
      });
    },
  }),
  AnimatePresence: ({ children }: { children?: React.ReactNode }) => <>{children}</>,
  useAnimation: () => ({ start: vi.fn(), stop: vi.fn() }),
  useInView: () => true,
}));

// Backend config store
vi.mock('@/stores/backendConfigStore', () => ({
  useBackendConfigStore: (selector?: (s: Record<string, unknown>) => unknown) => {
    const state: Record<string, unknown> = {
      backendBaseUrl: 'http://localhost:8190',
      currentClusterId: 'test-cluster-id',
      setBackendBaseUrl: vi.fn(),
      setCurrentClusterId: vi.fn(),
      isBackendConfigured: () => true,
    };
    return selector ? selector(state) : state;
  },
  getEffectiveBackendBaseUrl: () => 'http://localhost:8190',
}));

// Phase 7: clusterStore is deleted. The cluster-appearance helpers that used
// to live there now live in @/stores/clusterAppearance; mock that instead.
vi.mock('@/stores/clusterAppearance', () => ({
  getClusterAppearance: () => ({ color: '', environment: '', alias: '' }),
  setClusterAppearance: vi.fn(),
  getEnvBadgeLabel: () => null,
  getEnvBadgeClasses: () => '',
}));

vi.mock('@/stores/clusterPresenceStore', () => ({
  useActiveCluster: () => ({
    id: 'test-cluster-id',
    name: 'test-cluster',
    serverUrl: 'https://test',
    provider: '',
  }),
  getActiveCluster: () => ({
    id: 'test-cluster-id',
    name: 'test-cluster',
    serverUrl: 'https://test',
    provider: '',
  }),
  useClusterPresenceStore: (selector?: (s: Record<string, unknown>) => unknown) => {
    const state: Record<string, unknown> = {
      discovered: [],
      registered: [],
      connected: [],
      availableClusters: () => [],
      activeLogicalIdentity: { name: 'test-cluster', serverUrl: 'https://test' },
      isReady: true,
      setActiveByLogicalIdentity: vi.fn(),
      applySnapshot: vi.fn(),
      activeCluster: () => null,
    };
    return selector ? selector(state) : state;
  },
  setActiveClusterBySessionId: vi.fn(),
  __resetForTest: vi.fn(),
}));

// Theme store
vi.mock('@/stores/themeStore', () => ({
  useThemeStore: (selector?: (s: Record<string, unknown>) => unknown) => {
    const state: Record<string, unknown> = {
      theme: 'system' as const,
      setTheme: vi.fn(),
    };
    return selector ? selector(state) : state;
  },
}));

// Cluster organization store (FleetDashboard)
vi.mock('@/stores/clusterOrganizationStore', () => ({
  useClusterOrganizationStore: (selector?: (s: Record<string, unknown>) => unknown) => {
    const state: Record<string, unknown> = {
      // Real store (src/stores/clusterOrganizationStore.ts) types this as
      // string[] — ClusterCard calls favorites.includes(id), an Array
      // method. A Set here was never exercised because no prior test
      // rendered a populated cluster list (ClusterCard only mounts per
      // cluster), so the mismatch was latent until Phase 7's UX-2 test did.
      favorites: [] as string[],
      envTags: {} as Record<string, string>,
      groups: {} as Record<string, unknown>,
      toggleFavorite: vi.fn(),
      setEnvTag: vi.fn(),
      addToGroup: vi.fn(),
      removeFromGroup: vi.fn(),
      addGroup: vi.fn(),
    };
    return selector ? selector(state) : state;
  },
  ENV_DOT_COLORS: {},
  ENV_LABELS: {},
  ENV_BADGE_CLASSES: {},
  GROUP_COLORS: ['#3b82f6'],
}));

// Connection status
vi.mock('@/hooks/useConnectionStatus', () => ({
  useConnectionStatus: () => ({ isConnected: true }),
}));

// Resource counts
vi.mock('@/hooks/useResourceCounts', () => ({
  useResourceCounts: () => ({
    counts: { pods: 10, deployments: 5, services: 3, nodes: 2 },
    isLoading: false,
    isInitialLoad: false,
    isConnected: true,
  }),
}));

// Cluster overview (Dashboard)
vi.mock('@/hooks/useClusterOverview', () => ({
  useClusterOverview: () => ({
    data: { health: 'healthy', nodeCount: 2, podCount: 10 },
    isLoading: false,
    error: null,
    isError: false,
  }),
}));

// Fleet overview (FleetDashboard)
vi.mock('@/hooks/useFleetOverview', () => ({
  useFleetOverview: () => ({
    clusters: [],
    aggregates: { totalNodes: 0, totalPods: 0, healthyClusters: 0, warningClusters: 0, errorClusters: 0 },
    isLoading: false,
    isError: false,
    error: null,
  }),
}));

// Active cluster ID
vi.mock('@/hooks/useActiveClusterId', () => ({
  useActiveClusterId: () => 'test-cluster-id',
}));

// Namespaces
vi.mock('@/hooks/useNamespacesFromCluster', () => ({
  useNamespacesFromCluster: () => ({
    namespaces: ['default', 'kube-system'],
    isLoading: false,
  }),
}));

// K8s resource list (RBACAnalyzer)
vi.mock('@/hooks/useKubernetes', () => ({
  useK8sResourceList: () => ({
    data: { items: [] },
    isLoading: false,
    isError: false,
    error: null,
  }),
}));

// Backend circuit breaker
vi.mock('@/hooks/useBackendCircuitOpen', () => ({
  useBackendCircuitOpen: () => false,
}));

// Clusters from backend (Settings) — mockUseClustersFromBackend is a vi.fn()
// so individual tests (e.g. the LIFECYCLE-2 regression test below) can
// override its return value, while every other test keeps the empty-list
// default via resetClustersFromBackendMock() in beforeEach.
export const mockUseClustersFromBackend = vi.fn(() => ({ data: [] as unknown[], isLoading: false }));
vi.mock('@/hooks/useClustersFromBackend', () => ({
  useClustersFromBackend: () => mockUseClustersFromBackend(),
}));

// Backend API client
export const mockDeleteCluster = vi.fn().mockResolvedValue(undefined);
vi.mock('@/services/backendApiClient', () => ({
  getHealth: vi.fn().mockResolvedValue({ status: 'ok' }),
  deleteCluster: (...args: unknown[]) => mockDeleteCluster(...args),
  getProjects: vi.fn().mockResolvedValue([]),
  deleteProject: vi.fn().mockResolvedValue(undefined),
  searchResources: vi.fn().mockResolvedValue([]),
}));

// Resources API
vi.mock('@/services/api/resources', () => ({
  applyManifest: vi.fn().mockResolvedValue({ ok: true }),
}));

// Toast notifications
vi.mock('@/components/ui/sonner', () => ({
  toast: { success: vi.fn(), error: vi.fn(), info: vi.fn() },
}));

// Tauri detection
vi.mock('@/lib/tauri', () => ({
  isTauri: () => false,
}));

// Backend constants
vi.mock('@/lib/backendConstants', () => ({
  DEFAULT_BACKEND_BASE_URL: 'http://localhost:8190',
  isLocalHostname: () => true,
}));

// Backend cluster adapter
vi.mock('@/lib/backendClusterAdapter', () => ({
  backendClusterToCluster: (c: unknown) => c,
}));

// Dashboard tour (Dashboard page)
vi.mock('@/components/onboarding/DashboardTour', () => ({
  DashboardTour: () => null,
  useDashboardTour: () => ({ showTour: false, completeTour: vi.fn(), skipTour: vi.fn() }),
}));

// Dashboard sub-components — stub them out to isolate page-level rendering
vi.mock('@/components/dashboard/LiveSignalStrip', () => ({
  LiveSignalStrip: () => <div data-testid="live-signal-strip">LiveSignalStrip</div>,
}));
vi.mock('@/components/dashboard/DashboardHero', () => ({
  DashboardHero: () => <div data-testid="dashboard-hero">DashboardHero</div>,
}));
vi.mock('@/components/dashboard/IntelligencePanel', () => ({
  IntelligencePanel: () => <div data-testid="intelligence-panel">IntelligencePanel</div>,
}));
vi.mock('@/features/dashboard/components/ClusterOverviewPanel', () => ({
  ClusterOverviewPanel: () => <div data-testid="cluster-overview-panel">ClusterOverviewPanel</div>,
}));
vi.mock('@/components/dashboard/ActivityFeed', () => ({
  ActivityFeed: () => <div data-testid="activity-feed">ActivityFeed</div>,
}));
vi.mock('@/components/dashboard/WorkloadCapacitySnapshot', () => ({
  WorkloadCapacitySnapshot: () => <div data-testid="workload-capacity">WorkloadCapacitySnapshot</div>,
}));
vi.mock('@/components/dashboard/HealthScoreCard', () => ({
  HealthScoreCard: () => <div data-testid="health-score-card">HealthScoreCard</div>,
}));
vi.mock('@/components/dashboard/ClusterDetailsPanel', () => ({
  ClusterDetailsPanel: () => <div data-testid="cluster-details">ClusterDetailsPanel</div>,
}));

// Code editor (ResourceTemplates)
vi.mock('@/components/editor/CodeEditor', () => ({
  CodeEditor: () => <div data-testid="code-editor">CodeEditor</div>,
}));

// Project components (Settings)
vi.mock('@/components/projects/CreateProjectDialog', () => ({
  CreateProjectDialog: () => null,
}));
vi.mock('@/components/projects/ProjectCard', () => ({
  ProjectCard: () => <div>ProjectCard</div>,
}));
vi.mock('@/components/projects/ProjectSettingsDialog', () => ({
  ProjectSettingsDialog: () => null,
}));
vi.mock('@/components/settings/ClusterAppearance', () => ({
  ClusterAppearanceSettings: () => <div data-testid="cluster-appearance">ClusterAppearance</div>,
}));

// StatusBadge (FleetDashboard) — FleetDashboard actually uses the
// variant/label "legacy API" (see status-badge.tsx), not status/children;
// the mock previously ignored both, so the badge always rendered empty
// regardless of what status was passed — silently swallowing any label text
// a test might assert on.
vi.mock('@/components/ui/status-badge', () => ({
  StatusBadge: ({ status, children, label }: { status?: string; children?: React.ReactNode; label?: string }) => (
    <span data-testid="status-badge" role="status" aria-label={label ?? status}>{children || label || status}</span>
  ),
}));

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

function createQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
      mutations: { retry: false },
    },
  });
}

function renderPage(ui: React.ReactElement, { route = '/' } = {}) {
  const queryClient = createQueryClient();
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[route]}>
        {ui}
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

// Polyfill window.matchMedia for jsdom (used by Settings page)
beforeEach(() => {
  Object.defineProperty(window, 'matchMedia', {
    writable: true,
    value: vi.fn().mockImplementation((query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })),
  });
});

afterEach(() => {
  cleanup();
});

// ---------------------------------------------------------------------------
// Page smoke tests
// ---------------------------------------------------------------------------

describe('Page smoke tests', () => {
  describe('FleetDashboard', () => {
    it('renders without crashing and shows Fleet Overview heading', async () => {
      const FleetDashboard = (await import('@/pages/FleetDashboard')).default;
      const { container } = renderPage(<FleetDashboard />, { route: '/fleet' });
      expect(container).toBeTruthy();
      expect(screen.getByText('Fleet Overview')).toBeInTheDocument();
    });

    it('shows empty state when no clusters', async () => {
      const FleetDashboard = (await import('@/pages/FleetDashboard')).default;
      renderPage(<FleetDashboard />, { route: '/fleet' });
      // With 0 clusters, should show the connect prompt
      expect(screen.getByText(/Connect your first cluster/)).toBeInTheDocument();
    });

    // UX-2 (docs/PRODUCTION-HARDENING-ROADMAP.md, Phase 7): a cluster with
    // status 'unknown' previously had no entry in FleetDashboard's
    // statusConfig map, so `cfg.ringClass` on the resulting `undefined`
    // would throw during render — the whole Fleet page would crash the
    // moment one cluster's health couldn't be determined. Regression test:
    // render must succeed and show the distinct "Unknown" label.
    it('renders an unknown-status cluster without crashing, labeled distinctly from Healthy', async () => {
      const fleetHook = (await import('@/hooks/useFleetOverview')) as unknown as Record<string, unknown>;
      const origHook = fleetHook.useFleetOverview;
      fleetHook.useFleetOverview = () => ({
        clusters: [{
          id: 'c1', name: 'mystery-cluster', context: 'ctx',
          status: 'unknown', nodeCount: 0, podCount: 0, healthScore: 0,
          healthGrade: '?', deploymentCount: 0, serviceCount: 0,
        }],
        aggregates: {
          totalClusters: 1, totalNodes: 0, totalPods: 0, totalDeployments: 0,
          healthyClusters: 0, degradedClusters: 0, failedClusters: 0, unknownClusters: 1,
        },
        isLoading: false,
        isError: false,
        error: null,
      });

      const FleetDashboard = (await import('@/pages/FleetDashboard')).default;
      expect(() => renderPage(<FleetDashboard />, { route: '/fleet' })).not.toThrow();
      expect(screen.getByText('mystery-cluster')).toBeInTheDocument();
      // "Healthy" also appears as a static stat-card label (0 Healthy) even
      // when no cluster is healthy, so assert on the cluster's own status
      // badge specifically rather than absence of the word anywhere on the page.
      expect(screen.getByTestId('status-badge')).toHaveTextContent('Unknown');
      expect(screen.getByTestId('status-badge')).not.toHaveTextContent('Healthy');

      fleetHook.useFleetOverview = origHook;
    });
  });

  describe('ResourceTemplates', () => {
    it('renders without crashing and shows heading', async () => {
      const ResourceTemplates = (await import('@/pages/ResourceTemplates')).default;
      const { container } = renderPage(<ResourceTemplates />);
      expect(container).toBeTruthy();
      expect(screen.getByText('Resource Templates')).toBeInTheDocument();
    });

    it('shows search input', async () => {
      const ResourceTemplates = (await import('@/pages/ResourceTemplates')).default;
      renderPage(<ResourceTemplates />);
      expect(screen.getByPlaceholderText('Search templates...')).toBeInTheDocument();
    });
  });

  describe('RBACAnalyzer', () => {
    it('renders without crashing and shows RBAC Analyzer heading', async () => {
      const RBACAnalyzer = (await import('@/pages/RBACAnalyzer')).default;
      const { container } = renderPage(<RBACAnalyzer />);
      expect(container).toBeTruthy();
      expect(screen.getByText('RBAC Analyzer')).toBeInTheDocument();
    });

    it('shows tab navigation', async () => {
      const RBACAnalyzer = (await import('@/pages/RBACAnalyzer')).default;
      renderPage(<RBACAnalyzer />);
      // RBACAnalyzer has tabs — "Permission Matrix" may appear in both tab trigger and content
      expect(screen.getAllByText('Permission Matrix').length).toBeGreaterThan(0);
    });
  });

  describe('Settings', () => {
    it('renders without crashing and shows Settings heading', async () => {
      const Settings = (await import('@/pages/Settings')).default;
      const { container } = renderPage(<Settings />);
      expect(container).toBeTruthy();
      expect(screen.getByText('Settings')).toBeInTheDocument();
    });

    // LIFECYCLE-2 (docs/PRODUCTION-RELIABILITY-AUDIT.md): the delete-cluster
    // confirmation dialog's onError previously never cleared clusterToRemove,
    // so a failed delete left the dialog visibly open ("stuck") instead of
    // surfacing the error and letting the user retry.
    describe('delete cluster — error path (LIFECYCLE-2)', () => {
      afterEach(() => {
        mockUseClustersFromBackend.mockReturnValue({ data: [], isLoading: false });
        mockDeleteCluster.mockReset().mockResolvedValue(undefined);
      });

      it('closes the confirmation dialog when deleteCluster rejects', async () => {
        mockUseClustersFromBackend.mockReturnValue({
          data: [{ id: 'other-cluster', name: 'other-cluster', context: 'ctx', status: 'connected' }],
          isLoading: false,
        });
        mockDeleteCluster.mockRejectedValue(new Error('backend unreachable'));

        const Settings = (await import('@/pages/Settings')).default;
        const { container } = renderPage(<Settings />);

        // The delete trigger is an icon-only button (lucide Trash2, no
        // accessible text) — locate it by the icon's lucide class.
        const trashButton = container
          .querySelector('svg.lucide-trash2')
          ?.closest('button');
        expect(trashButton).toBeTruthy();
        fireEvent.click(trashButton!);

        const confirmButton = await screen.findByRole('button', { name: /remove cluster/i });
        fireEvent.click(confirmButton);

        // Dialog should close (confirm button unmounts) once the rejected
        // mutation's onError clears clusterToRemove — not stay stuck open.
        await waitFor(() => {
          expect(screen.queryByRole('button', { name: /remove cluster/i })).not.toBeInTheDocument();
        });
      });
    });
  });
});
