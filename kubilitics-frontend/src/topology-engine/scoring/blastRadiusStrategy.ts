/**
 * Strategy 2: Blast Radius Dominance Model
 *
 * Scores each node by the size and severity of its downstream blast radius.
 * Leverages the existing computeBlastRadius() BFS engine but runs it across
 * eligible nodes and converts results into a 0–100 criticality score.
 */
import type { TopologyGraph, KubernetesKind } from '../types/topology.types';
import { computeBlastRadius } from '../utils/blastRadiusCompute';

/** Resource kinds worth computing blast radius for */
const ELIGIBLE_KINDS: Set<KubernetesKind> = new Set([
  'Service', 'Deployment', 'StatefulSet', 'DaemonSet',
  'Ingress', 'Node', 'PersistentVolume', 'CronJob',
  'Secret', 'ConfigMap',
]);

const ALPHA = 0.4; // weight for affected ratio
const BETA = 0.4;  // weight for average severity
const GAMMA = 0.2; // weight for SPOF bonus

/**
 * Compute blast radius dominance scores for all nodes.
 * Only runs full BFS for eligible resource kinds; others get scores
 * derived from their parent/owner's score.
 */
export function computeBlastRadiusScores(graph: TopologyGraph): Map<string, number> {
  const scores = new Map<string, number>();
  const totalNodes = graph.nodes.length;

  if (totalNodes === 0) return scores;

  // Phase 1: Compute blast radius for eligible nodes
  const eligibleNodes = graph.nodes.filter(n => ELIGIBLE_KINDS.has(n.kind));

  for (const node of eligibleNodes) {
    const result = computeBlastRadius(graph, node.id, {
      maxDepth: 4,
      includeDownstream: true,
      includeUpstream: false,
      propagationFactor: 0.7,
    });

    const affectedRatio = (result.affectedNodes.size / totalNodes) * 100;

    let totalSeverity = 0;
    result.severity.forEach(s => { totalSeverity += s; });
    const avgSeverity = result.affectedNodes.size > 0
      ? totalSeverity / result.affectedNodes.size
      : 0;

    const isSPOF = detectSPOF(node.id, graph);
    const spofBonus = isSPOF ? 25 : 0;

    const raw = ALPHA * affectedRatio + BETA * avgSeverity + GAMMA * spofBonus;
    scores.set(node.id, Math.min(100, raw));
  }

  // Phase 2: Non-eligible nodes inherit from their owner/parent
  for (const node of graph.nodes) {
    if (scores.has(node.id)) continue;

    // Find parent via ownership edges
    const ownerEdge = graph.edges.find(
      e => e.target === node.id && (e.relationshipType === 'owns' || e.relationshipType === 'selects')
    );

    if (ownerEdge && scores.has(ownerEdge.source)) {
      // Inherit 60% of parent's score
      scores.set(node.id, (scores.get(ownerEdge.source)! * 0.6));
    } else {
      scores.set(node.id, 0);
    }
  }

  // Normalize to 0-100
  return normalizeScores(scores);
}

function detectSPOF(nodeId: string, graph: TopologyGraph): boolean {
  const downstreamEdges = graph.edges.filter(e => e.source === nodeId);
  if (downstreamEdges.length < 3) return false;

  for (const edge of downstreamEdges) {
    const alternatives = graph.edges.filter(
      e => e.target === edge.target && e.source !== nodeId && e.relationshipType === edge.relationshipType
    );
    if (alternatives.length > 0) return false;
  }
  return true;
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
