/**
 * Strategy 3: Traffic-Weighted Betweenness Model
 *
 * Scores nodes by how many critical communication paths pass through them.
 * Uses a sampled betweenness approach: only computes shortest paths from
 * entry points (Ingress, LoadBalancer Services) to backends (StatefulSets,
 * PersistentVolumes) to stay within O(E * P) complexity.
 */
import type { TopologyGraph, TopologyNode, KubernetesKind } from '../types/topology.types';

/** Traffic weight by relationship type (how much real traffic this edge carries) */
const TRAFFIC_EDGE_WEIGHTS: Record<string, number> = {
  routes: 0.9,
  exposes: 0.8,
  selects: 0.7,
  runs: 0.6,
  owns: 0.3,
  scheduled_on: 0.2,
  mounts: 0.1,
  references: 0.1,
  configures: 0.1,
  contains: 0.05,
  stores: 0.1,
  backed_by: 0.5,
  permits: 0.1,
  limits: 0.05,
  manages: 0.1,
};

const ENTRY_KINDS: Set<KubernetesKind> = new Set(['Ingress']);
const BACKEND_KINDS: Set<KubernetesKind> = new Set([
  'StatefulSet', 'PersistentVolume', 'PersistentVolumeClaim',
]);

/**
 * Compute traffic-weighted betweenness centrality scores.
 */
export function computeTrafficBetweenness(graph: TopologyGraph): Map<string, number> {
  const scores = new Map<string, number>();
  for (const node of graph.nodes) {
    scores.set(node.id, 0);
  }

  if (graph.nodes.length === 0) return scores;

  // Identify entry points and backends
  const entryNodes = graph.nodes.filter(n => isEntryPoint(n, graph));
  const backendNodes = graph.nodes.filter(n => BACKEND_KINDS.has(n.kind));

  // If no clear entry/backend separation, fall back to degree-based heuristic
  if (entryNodes.length === 0 || backendNodes.length === 0) {
    return computeDegreeBetweenness(graph);
  }

  // Build weighted adjacency list
  const adj = new Map<string, Array<{ target: string; weight: number }>>();
  for (const node of graph.nodes) {
    adj.set(node.id, []);
  }
  for (const edge of graph.edges) {
    const weight = TRAFFIC_EDGE_WEIGHTS[edge.relationshipType] ?? 0.1;
    const confidence = edge.metadata?.confidence ?? 0.5;
    adj.get(edge.source)?.push({ target: edge.target, weight: weight * confidence });
    // Bidirectional for betweenness (traffic can flow either way)
    adj.get(edge.target)?.push({ target: edge.source, weight: weight * confidence * 0.5 });
  }

  // For each entry→backend pair, find shortest path and credit intermediate nodes
  for (const entry of entryNodes) {
    for (const backend of backendNodes) {
      if (entry.id === backend.id) continue;

      const path = bfsShortestPath(entry.id, backend.id, adj);
      if (!path || path.length < 3) continue;

      // Credit intermediate nodes (exclude source and target)
      for (let i = 1; i < path.length - 1; i++) {
        const current = scores.get(path[i]) ?? 0;
        scores.set(path[i], current + 1);
      }
    }
  }

  // Normalize to 0-100
  return normalizeScores(scores);
}

function isEntryPoint(node: TopologyNode, graph: TopologyGraph): boolean {
  if (ENTRY_KINDS.has(node.kind)) return true;
  // Services with type LoadBalancer or NodePort are also entry points
  if (node.kind === 'Service') {
    const inEdges = graph.edges.filter(e => e.target === node.id);
    const hasIngress = inEdges.some(e => {
      const sourceNode = graph.nodes.find(n => n.id === e.source);
      return sourceNode?.kind === 'Ingress';
    });
    if (hasIngress) return true;
  }
  return false;
}

/** BFS shortest path returning the node IDs along the path */
function bfsShortestPath(
  start: string,
  end: string,
  adj: Map<string, Array<{ target: string; weight: number }>>
): string[] | null {
  const visited = new Set<string>([start]);
  const parent = new Map<string, string>();
  const queue: string[] = [start];

  while (queue.length > 0) {
    const current = queue.shift()!;
    if (current === end) {
      // Reconstruct path
      const path: string[] = [end];
      let node = end;
      while (parent.has(node)) {
        node = parent.get(node)!;
        path.unshift(node);
      }
      return path;
    }

    const neighbors = adj.get(current) ?? [];
    // Sort by weight descending to prefer high-traffic edges
    neighbors.sort((a, b) => b.weight - a.weight);

    for (const { target } of neighbors) {
      if (!visited.has(target)) {
        visited.add(target);
        parent.set(target, current);
        queue.push(target);
      }
    }
  }

  return null; // No path found
}

/** Fallback: simple degree-based betweenness when entry/backend nodes aren't clear */
function computeDegreeBetweenness(graph: TopologyGraph): Map<string, number> {
  const scores = new Map<string, number>();
  const inDeg = new Map<string, number>();
  const outDeg = new Map<string, number>();

  for (const node of graph.nodes) {
    inDeg.set(node.id, 0);
    outDeg.set(node.id, 0);
  }

  for (const edge of graph.edges) {
    const weight = TRAFFIC_EDGE_WEIGHTS[edge.relationshipType] ?? 0.1;
    outDeg.set(edge.source, (outDeg.get(edge.source) ?? 0) + weight);
    inDeg.set(edge.target, (inDeg.get(edge.target) ?? 0) + weight);
  }

  // Nodes with both high in-degree AND out-degree are on many paths
  for (const node of graph.nodes) {
    const inScore = inDeg.get(node.id) ?? 0;
    const outScore = outDeg.get(node.id) ?? 0;
    // Geometric mean emphasizes nodes that are both receivers and senders
    scores.set(node.id, Math.sqrt(inScore * outScore));
  }

  return normalizeScores(scores);
}

function normalizeScores(scores: Map<string, number>): Map<string, number> {
  let max = 0;
  for (const s of scores.values()) {
    if (s > max) max = s;
  }
  if (max === 0) return scores;

  const result = new Map<string, number>();
  for (const [id, s] of scores) {
    result.set(id, (s / max) * 100);
  }
  return result;
}
