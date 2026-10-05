/**
 * useClusterUtilization — computes cluster-wide CPU and Memory utilization
 * by fetching per-node metrics from the Metrics Server API and comparing
 * against each node's allocatable capacity.
 *
 * This supplements useClusterOverview whose backend response may not include
 * the `utilization` field even when the Metrics Server is installed.
 */
import { useQueries, useQuery } from '@tanstack/react-query';
import { useBackendConfigStore, getEffectiveBackendBaseUrl } from '@/stores/backendConfigStore';
import { getNodeMetrics, listResources } from '@/services/backendApiClient';
import { parseK8sCpuToMillicores, parseK8sQuantityToBytes } from '@/lib/k8sQuantity';

/* ─── Parsing helpers ───
 * METRICS-2 (docs/PRODUCTION-RELIABILITY-AUDIT.md): these previously
 * reimplemented unit parsing locally with `parseFloat(x) || 0` fallbacks,
 * silently turning a malformed quantity into 0 instead of null. Delegates to
 * the canonical parser; 0-on-failure is preserved here only for this
 * function's own `number` contract — the cluster-wide `metricsAvailable`
 * flag below (not a per-node fallback) is this hook's actual "unknown"
 * signal, and is unaffected by this change. */
function parseCpuMillicores(cpu: string): number {
  return parseK8sCpuToMillicores(cpu) ?? 0;
}

function parseMemoryBytes(mem: string): number {
  return parseK8sQuantityToBytes(mem) ?? 0;
}

interface NodeInfo {
  name: string;
  allocatableCpuMillicores: number;
  allocatableMemoryBytes: number;
}

export interface ClusterUtilization {
  cpuPercent: number;
  memoryPercent: number;
  cpuUsedMillicores: number;
  cpuTotalMillicores: number;
  memoryUsedBytes: number;
  memoryTotalBytes: number;
  nodeCount: number;
  metricsAvailable: boolean;
}

export function useClusterUtilization(clusterId: string | undefined) {
  const stored = useBackendConfigStore((s) => s.backendBaseUrl);
  const backendBaseUrl = getEffectiveBackendBaseUrl(stored);
  const isConfigured = useBackendConfigStore((s) => s.isBackendConfigured());

  // Step 1: Fetch node list to get names + capacity
  const nodesQuery = useQuery({
    queryKey: ['backend', 'resources', 'nodes', backendBaseUrl, clusterId],
    queryFn: async () => {
      const result = await listResources(backendBaseUrl, clusterId!, 'nodes');
      const items: NodeInfo[] = (result.items || []).map((node: Record<string, unknown>) => ({
        name: node.metadata?.name ?? '',
        allocatableCpuMillicores: parseCpuMillicores(node.status?.allocatable?.cpu ?? '0'),
        allocatableMemoryBytes: parseMemoryBytes(node.status?.allocatable?.memory ?? '0'),
      }));
      return items;
    },
    enabled: isConfigured && !!clusterId,
    staleTime: 60_000,
    refetchInterval: 60_000,
  });

  const nodes = nodesQuery.data ?? [];

  // Step 2: Fetch per-node metrics
  const metricsQueries = useQueries({
    queries: nodes.map((node) => ({
      queryKey: ['backend', 'nodeMetrics', backendBaseUrl, clusterId, node.name],
      queryFn: () => getNodeMetrics(backendBaseUrl, clusterId!, node.name),
      enabled: isConfigured && !!clusterId && nodes.length > 0,
      staleTime: 30_000,
      refetchInterval: 30_000,
      retry: 1,
    })),
  });

  // Step 3: Aggregate into cluster utilization
  const isLoading = nodesQuery.isLoading || metricsQueries.some((q) => q.isLoading);
  const anyMetricsSucceeded = metricsQueries.some((q) => q.isSuccess && q.data);

  let cpuUsedMillicores = 0;
  let cpuTotalMillicores = 0;
  let memoryUsedBytes = 0;
  let memoryTotalBytes = 0;

  nodes.forEach((node, i) => {
    cpuTotalMillicores += node.allocatableCpuMillicores;
    memoryTotalBytes += node.allocatableMemoryBytes;

    const metrics = metricsQueries[i]?.data;
    if (metrics) {
      cpuUsedMillicores += parseCpuMillicores(metrics.CPU ?? '');
      memoryUsedBytes += parseMemoryBytes(metrics.Memory ?? '');
    }
  });

  const cpuPercent = cpuTotalMillicores > 0 ? (cpuUsedMillicores / cpuTotalMillicores) * 100 : 0;
  const memoryPercent = memoryTotalBytes > 0 ? (memoryUsedBytes / memoryTotalBytes) * 100 : 0;

  const utilization: ClusterUtilization = {
    cpuPercent: Math.round(cpuPercent * 10) / 10,
    memoryPercent: Math.round(memoryPercent * 10) / 10,
    cpuUsedMillicores,
    cpuTotalMillicores,
    memoryUsedBytes,
    memoryTotalBytes,
    nodeCount: nodes.length,
    metricsAvailable: anyMetricsSucceeded,
  };

  // nodesQuery failing previously produced nodeCount:0, cpuPercent:0,
  // memoryPercent:0 — indistinguishable from a real, tiny, empty cluster.
  // metricsAvailable already covers the "metrics server has no data" case;
  // this covers the separate "we couldn't even list nodes" case.
  return { utilization, isLoading, isError: nodesQuery.isError };
}
