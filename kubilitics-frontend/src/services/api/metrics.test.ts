/**
 * Tests for src/services/api/metrics.ts
 */
import { describe, it, expect, vi, beforeEach } from 'vitest';

vi.mock('@/lib/tauri', () => ({ isTauri: () => false }));
vi.mock('@/stores/authStore', () => ({
  useAuthStore: { getState: () => ({ logout: vi.fn() }) },
}));
vi.mock('@/stores/clusterStore', () => ({
  useClusterStore: { getState: () => ({ activeCluster: null, kubeconfigContent: null }) },
}));

import {
  getPodMetrics,
  getNodeMetrics,
  getDeploymentMetrics,
  getReplicaSetMetrics,
  getStatefulSetMetrics,
  getDaemonSetMetrics,
  getJobMetrics,
  getCronJobMetrics,
  getMetricsSummary,
  getMetricsHistory,
} from './metrics';
import { resetBackendCircuit, markBackendReady } from './client';

function mockFetch(body: unknown, status = 200) {
  globalThis.fetch = vi.fn().mockResolvedValue({
    ok: status >= 200 && status < 300,
    status,
    headers: new Headers(),
    text: () => Promise.resolve(typeof body === 'string' ? body : JSON.stringify(body)),
  } as unknown as Response);
}

beforeEach(() => {
  resetBackendCircuit();
  markBackendReady();
  vi.restoreAllMocks();
});

describe('getPodMetrics', () => {
  it('returns pod metrics data', async () => {
    const metrics = { cpu: '100m', memory: '256Mi', containers: [] };
    mockFetch(metrics);

    const result = await getPodMetrics('http://localhost:8190', 'c1', 'default', 'web-1');
    expect(result).toEqual(metrics);
  });

  it('builds correct URL with encoded segments', async () => {
    mockFetch({ cpu: '0', memory: '0', containers: [] });

    await getPodMetrics('http://localhost:8190', 'my cluster', 'kube system', 'pod/1');

    const calledUrl = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0][0] as string;
    expect(calledUrl).toContain('my%20cluster');
    expect(calledUrl).toContain('kube%20system');
    expect(calledUrl).toContain('pod%2F1');
  });
});

describe('getNodeMetrics', () => {
  it('returns node metrics data', async () => {
    const metrics = { cpu: '2000m', memory: '8Gi' };
    mockFetch(metrics);

    const result = await getNodeMetrics('http://localhost:8190', 'c1', 'node-1');
    expect(result).toEqual(metrics);
  });

  it('builds correct URL path', async () => {
    mockFetch({ cpu: '0', memory: '0' });

    await getNodeMetrics('http://localhost:8190', 'c1', 'worker-node-1');

    const calledUrl = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0][0] as string;
    expect(calledUrl).toContain('/metrics/nodes/worker-node-1');
  });
});

describe('getDeploymentMetrics', () => {
  it('returns deployment aggregated metrics', async () => {
    const metrics = { cpu: '500m', memory: '1Gi', pods: [] };
    mockFetch(metrics);

    const result = await getDeploymentMetrics('http://localhost:8190', 'c1', 'default', 'nginx');
    expect(result).toEqual(metrics);
  });

  it('builds correct URL path', async () => {
    mockFetch({ cpu: '0', memory: '0', pods: [] });

    await getDeploymentMetrics('http://localhost:8190', 'c1', 'prod', 'api-server');

    const calledUrl = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0][0] as string;
    expect(calledUrl).toContain('/metrics/prod/deployment/api-server');
  });
});

describe('workload metrics variants', () => {
  it('getReplicaSetMetrics uses /replicaset/ path', async () => {
    mockFetch({ cpu: '0', memory: '0', pods: [] });
    await getReplicaSetMetrics('http://localhost:8190', 'c1', 'ns', 'rs-1');
    const url = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0][0] as string;
    expect(url).toContain('/replicaset/rs-1');
  });

  it('getStatefulSetMetrics uses /statefulset/ path', async () => {
    mockFetch({ cpu: '0', memory: '0', pods: [] });
    await getStatefulSetMetrics('http://localhost:8190', 'c1', 'ns', 'sts-1');
    const url = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0][0] as string;
    expect(url).toContain('/statefulset/sts-1');
  });

  it('getDaemonSetMetrics uses /daemonset/ path', async () => {
    mockFetch({ cpu: '0', memory: '0', pods: [] });
    await getDaemonSetMetrics('http://localhost:8190', 'c1', 'ns', 'ds-1');
    const url = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0][0] as string;
    expect(url).toContain('/daemonset/ds-1');
  });

  it('getJobMetrics uses /job/ path', async () => {
    mockFetch({ cpu: '0', memory: '0', pods: [] });
    await getJobMetrics('http://localhost:8190', 'c1', 'ns', 'job-1');
    const url = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0][0] as string;
    expect(url).toContain('/job/job-1');
  });

  it('getCronJobMetrics uses /cronjob/ path', async () => {
    mockFetch({ cpu: '0', memory: '0', pods: [] });
    await getCronJobMetrics('http://localhost:8190', 'c1', 'ns', 'cj-1');
    const url = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0][0] as string;
    expect(url).toContain('/cronjob/cj-1');
  });
});

describe('getMetricsSummary', () => {
  it('returns summary data with query params', async () => {
    const summary = { cpu_usage: '500m', memory_usage: '2Gi', pod_count: 5 };
    mockFetch(summary);

    const result = await getMetricsSummary('http://localhost:8190', 'c1', {
      namespace: 'prod',
      resource_type: 'deployment',
      resource_name: 'api',
    });

    expect(result).toEqual(summary);
    const calledUrl = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0][0] as string;
    expect(calledUrl).toContain('namespace=prod');
    expect(calledUrl).toContain('resource_type=deployment');
    expect(calledUrl).toContain('resource_name=api');
  });

  it('omits namespace param when empty string', async () => {
    mockFetch({});

    await getMetricsSummary('http://localhost:8190', 'c1', {
      namespace: '',
      resource_type: 'node',
      resource_name: 'worker-1',
    });

    const calledUrl = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0][0] as string;
    expect(calledUrl).not.toContain('namespace=');
    expect(calledUrl).toContain('resource_type=node');
  });

  it('omits namespace param when undefined', async () => {
    mockFetch({});

    await getMetricsSummary('http://localhost:8190', 'c1', {
      resource_type: 'pod',
      resource_name: 'web-1',
    });

    const calledUrl = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0][0] as string;
    expect(calledUrl).not.toContain('namespace=');
  });
});

describe('getMetricsHistory', () => {
  it('returns history data with time range params', async () => {
    const history = { data_points: [{ timestamp: '2025-01-01T00:00:00Z', cpu: '100m', memory: '256Mi' }] };
    mockFetch(history);

    const result = await getMetricsHistory('http://localhost:8190', 'c1', {
      namespace: 'default',
      resource_type: 'pod',
      resource_name: 'web-1',
      duration: '1h',
    });

    expect(result).toEqual(history);
    const calledUrl = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0][0] as string;
    expect(calledUrl).toContain('duration=1h');
    expect(calledUrl).toContain('namespace=default');
    expect(calledUrl).toContain('/metrics/history');
  });

  it('omits duration when not provided', async () => {
    mockFetch({ data_points: [] });

    await getMetricsHistory('http://localhost:8190', 'c1', {
      resource_type: 'deployment',
      resource_name: 'api',
    });

    const calledUrl = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0][0] as string;
    expect(calledUrl).not.toContain('duration=');
  });

  it('omits namespace when not provided', async () => {
    mockFetch({ data_points: [] });

    await getMetricsHistory('http://localhost:8190', 'c1', {
      resource_type: 'node',
      resource_name: 'worker-1',
    });

    const calledUrl = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0][0] as string;
    expect(calledUrl).not.toContain('namespace=');
  });
});
