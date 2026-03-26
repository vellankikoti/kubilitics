/**
 * Strategy 1: Graph Centrality Model (Weighted PageRank)
 *
 * Adapts PageRank to score nodes by transitive structural importance.
 * Edges are reversed so importance flows from dependents upward to
 * the resources they depend on.
 */
import type { TopologyGraph } from '../types/topology.types';
import { RELATIONSHIP_WEIGHTS } from './types';

const DAMPING = 0.85;
const ITERATIONS = 20;

/**
 * Compute weighted PageRank scores for all nodes in the graph.
 * Returns a Map of nodeId → score normalized to 0–100.
 */
export function computePageRank(graph: TopologyGraph): Map<string, number> {
  const nodeIds = graph.nodes.map(n => n.id);
  const n = nodeIds.length;
  if (n === 0) return new Map();

  // Build reversed adjacency list with weights.
  // Original: parent → child. Reversed: child → parent.
  // This makes nodes that many important things depend on score higher.
  const inEdges = new Map<string, Array<{ from: string; weight: number }>>();
  const outWeightSum = new Map<string, number>();

  for (const id of nodeIds) {
    inEdges.set(id, []);
    outWeightSum.set(id, 0);
  }

  for (const edge of graph.edges) {
    const weight = RELATIONSHIP_WEIGHTS[edge.relationshipType] ?? 0.5;
    const confidence = edge.metadata?.confidence ?? 0.5;
    const edgeWeight = weight * confidence;

    // Reverse: target gets incoming edge from source (original direction was source→target)
    // After reversal for PageRank: source gets importance from target
    const reversed = inEdges.get(edge.source);
    if (reversed) {
      reversed.push({ from: edge.target, weight: edgeWeight });
    }
    outWeightSum.set(edge.target, (outWeightSum.get(edge.target) ?? 0) + edgeWeight);
  }

  // Initialize scores uniformly
  let scores = new Map<string, number>();
  const initial = 1.0 / n;
  for (const id of nodeIds) {
    scores.set(id, initial);
  }

  // Power iteration
  for (let iter = 0; iter < ITERATIONS; iter++) {
    const next = new Map<string, number>();
    for (const id of nodeIds) {
      let sum = 0;
      const incoming = inEdges.get(id)!;
      for (const { from, weight } of incoming) {
        const fromScore = scores.get(from) ?? 0;
        const fromOutWeight = outWeightSum.get(from) ?? 1;
        sum += (weight * fromScore) / (fromOutWeight || 1);
      }
      next.set(id, (1 - DAMPING) / n + DAMPING * sum);
    }
    scores = next;
  }

  // Normalize to 0–100
  let maxScore = 0;
  for (const s of scores.values()) {
    if (s > maxScore) maxScore = s;
  }

  const normalized = new Map<string, number>();
  if (maxScore === 0) {
    for (const id of nodeIds) normalized.set(id, 0);
  } else {
    for (const [id, s] of scores) {
      normalized.set(id, (s / maxScore) * 100);
    }
  }

  return normalized;
}
