/**
 * Critical UX gap reported by a user: the Metrics tab's chart x-axis shows
 * only time-of-day ("03:45:00 PM") no matter the selected range. Fine up to
 * 6h (everything visible is "today"), but at 24h/7d multiple calendar days
 * collapse onto bare times with no way to tell which day a point is from.
 * These tests pin the fix: axis labels become date-aware past 6h, and a
 * date-range banner appears whenever the visible window spans >1 day.
 */
import { describe, it, expect, vi, beforeAll } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import '@testing-library/jest-dom/vitest';

// recharts' ResponsiveContainer measures its container via
// getBoundingClientRect/ResizeObserver; jsdom reports 0x0 for both by
// default, so without this recharts never renders axis ticks at all
// (a known recharts+jsdom limitation, not something our code controls).
beforeAll(() => {
  Object.defineProperty(HTMLElement.prototype, 'getBoundingClientRect', {
    configurable: true,
    value: () => ({ width: 800, height: 400, top: 0, left: 0, bottom: 400, right: 800, x: 0, y: 0, toJSON() {} }),
  });
  // recharts' ResponsiveContainer sets its own width/height state from the
  // ResizeObserver callback, not from an initial getBoundingClientRect read —
  // a no-op observe() leaves it stuck at 0x0 forever. Invoke the callback
  // once, synchronously, with the stubbed size above.
  global.ResizeObserver = class {
    constructor(private cb: ResizeObserverCallback) {}
    observe(target: Element) {
      this.cb(
        [{ target, contentRect: { width: 800, height: 400 } } as ResizeObserverEntry],
        this as unknown as ResizeObserver
      );
    }
    unobserve() {}
    disconnect() {}
  };
});
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { TooltipProvider } from '@/components/ui/tooltip';
import { formatAxisTick } from '@/lib/chartTime';

/**
 * recharts' SVG <Text> wraps a space-containing axis label (e.g. "09:15:00
 * PM") across multiple <tspan> lines in jsdom — real browsers have enough
 * measured width not to wrap it, but jsdom's getComputedTextLength isn't
 * reliable, so recharts' own wrap heuristic kicks in during tests. Match
 * against the parent <text> element's full (whitespace-normalized) content
 * instead of a single exact text node.
 */
function queryAxisLabel(expected: string): HTMLElement[] {
  const normalized = expected.replace(/\s+/g, '');
  return screen.getAllByText((_content, element) => {
    if (!element || element.tagName.toLowerCase() !== 'text') return false;
    return (element.textContent ?? '').replace(/\s+/g, '') === normalized;
  });
}

vi.mock('@/hooks/useActiveClusterId', () => ({
  useActiveClusterId: () => 'c1',
}));

vi.mock('@/stores/backendConfigStore', () => ({
  useBackendConfigStore: (sel: (s: { backendBaseUrl: string; isBackendConfigured: () => boolean }) => unknown) =>
    sel({ backendBaseUrl: 'http://backend', isBackendConfigured: () => true }),
  getEffectiveBackendBaseUrl: () => 'http://backend',
}));

// Two points one day apart so the default tab (15m) still renders real history
// (component requires >=2 points to use formatted timestamps instead of "now").
const DAY_MS = 24 * 60 * 60 * 1000;
const T0 = new Date('2026-10-08T10:00:00Z').getTime() / 1000;
const T1 = new Date('2026-10-14T15:45:00Z').getTime() / 1000;

// Stable (module-level, not re-created per call) mock payloads — a fresh
// object literal returned on every hook call busts useMemo's referential
// equality and causes an infinite render loop, since real useQuery data is
// reference-stable between renders unless an actual refetch occurs.
const HISTORY_RESPONSE = {
  points: [
    { ts: T0, cpu_milli: 40, memory_mib: 500, network_rx: 1000, network_tx: 2000 },
    { ts: T1, cpu_milli: 60, memory_mib: 600, network_rx: 2000, network_tx: 3000 },
  ],
  interval_sec: 30,
};

const SUMMARY_RESPONSE = {
  summary: {
    cluster_id: 'c1',
    namespace: 'default',
    resource_type: 'pod',
    resource_name: 'my-pod',
    total_cpu: '60m',
    total_memory: '600Mi',
    total_network_rx: 2000,
    total_network_tx: 3000,
    pod_count: 1,
    pods: [],
  },
};

vi.mock('@/hooks/useMetricsHistory', () => ({
  useMetricsHistory: () => ({ data: HISTORY_RESPONSE, refetch: vi.fn() }),
}));

vi.mock('@/hooks/useMetricsSummary', () => ({
  useMetricsSummary: () => ({ data: SUMMARY_RESPONSE, isLoading: false, error: null, refetch: vi.fn() }),
}));

import { MetricsDashboard } from './MetricsDashboard';

function renderWithQueryClient() {
  const client = new QueryClient();
  return render(
    <QueryClientProvider client={client}>
      <TooltipProvider>
        <MetricsDashboard resourceType="pod" resourceName="my-pod" namespace="default" />
      </TooltipProvider>
    </QueryClientProvider>
  );
}

describe('MetricsDashboard — range-aware chart time labels', () => {
  it('shows a time-only axis label on the default (short) range, no date', () => {
    renderWithQueryClient();
    const timeOnlyLabel = formatAxisTick(T1, '15m');
    expect(queryAxisLabel(timeOnlyLabel).length).toBeGreaterThan(0);
  });

  it('switches to a date-aware axis label after selecting 24h', () => {
    renderWithQueryClient();
    fireEvent.click(screen.getByRole('button', { name: '24h' }));
    const dateAwareLabel = formatAxisTick(T1, '24h');
    expect(queryAxisLabel(dateAwareLabel).length).toBeGreaterThan(0);
  });

  it('switches to a date-aware axis label after selecting 7d', () => {
    renderWithQueryClient();
    fireEvent.click(screen.getByRole('button', { name: '7d' }));
    const dateAwareLabel = formatAxisTick(T1, '168h');
    expect(queryAxisLabel(dateAwareLabel).length).toBeGreaterThan(0);
  });

  it('shows a visible date-range banner when the window spans multiple days', () => {
    renderWithQueryClient();
    // T0 and T1 are 6 days apart — the component's own points span >1 day
    // regardless of range selector, since both mocked points are always returned.
    expect(screen.getByText(/Oct 8.*–.*Oct 14, 2026/)).toBeInTheDocument();
  });
});

describe('MetricsDashboard — single-day range shows no banner', () => {
  it('does not render a date-range banner when all points fall on the same day', async () => {
    vi.resetModules();
    const sameDayHistory = {
      points: [
        { ts: T1 - DAY_MS / 1000 / 24, cpu_milli: 40, memory_mib: 500, network_rx: 1000, network_tx: 2000 },
        { ts: T1, cpu_milli: 60, memory_mib: 600, network_rx: 2000, network_tx: 3000 },
      ],
      interval_sec: 30,
    };
    vi.doMock('@/hooks/useMetricsHistory', () => ({
      useMetricsHistory: () => ({ data: sameDayHistory, refetch: vi.fn() }),
    }));
    const { MetricsDashboard: FreshMetricsDashboard } = await import('./MetricsDashboard');
    const client = new QueryClient();
    render(
      <QueryClientProvider client={client}>
        <TooltipProvider>
          <FreshMetricsDashboard resourceType="pod" resourceName="my-pod" namespace="default" />
        </TooltipProvider>
      </QueryClientProvider>
    );
    expect(screen.queryByText(/–.*2026/)).not.toBeInTheDocument();
  });
});
