/**
 * Strategy 4: Structural Risk Score (Resource-Aware Composite)
 *
 * Scores each node based on resource type, redundancy, health, and connectivity.
 * O(V) computation — no graph traversal needed. Uses only node metadata and degree.
 */
import type { TopologyGraph, TopologyNode, HealthStatus } from '../types/topology.types';
import { RESOURCE_TYPE_WEIGHTS } from './types';

const HEALTH_MULTIPLIERS: Record<HealthStatus, number> = {
  healthy: 1.0,
  warning: 1.3,
  critical: 1.8,
  unknown: 1.2,
};

/**
 * Compute structural risk scores for all nodes.
 */
export function computeStructuralRisk(graph: TopologyGraph): Map<string, number> {
  const scores = new Map<string, number>();

  if (graph.nodes.length === 0) return scores;

  // Pre-compute degrees
  const inDeg = new Map<string, number>();
  const outDeg = new Map<string, number>();
  for (const node of graph.nodes) {
    inDeg.set(node.id, 0);
    outDeg.set(node.id, 0);
  }
  for (const edge of graph.edges) {
    inDeg.set(edge.target, (inDeg.get(edge.target) ?? 0) + 1);
    outDeg.set(edge.source, (outDeg.get(edge.source) ?? 0) + 1);
  }

  for (const node of graph.nodes) {
    const typeWeight = RESOURCE_TYPE_WEIGHTS[node.kind] ?? 5;
    const redundancyPenalty = getRedundancyPenalty(node);
    const healthMultiplier = HEALTH_MULTIPLIERS[node.computed?.health ?? 'unknown'];
    const connectivity = getConnectivityFactor(
      inDeg.get(node.id) ?? 0,
      outDeg.get(node.id) ?? 0
    );

    const raw = typeWeight * redundancyPenalty * healthMultiplier * connectivity;
    scores.set(node.id, raw);
  }

  // Normalize to 0-100
  return normalizeScores(scores);
}

function getRedundancyPenalty(node: TopologyNode): number {
  const replicas = node.computed?.replicas;
  if (!replicas) return 1.0;

  const desired = replicas.desired;
  if (desired <= 1) return 2.0;
  if (desired === 2) return 1.5;
  return 1.0;
}

function getConnectivityFactor(inDeg: number, outDeg: number): number {
  // Range: 1.0 to 2.0
  return 1.0 + Math.min(1.0, (inDeg + outDeg) / 20);
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
