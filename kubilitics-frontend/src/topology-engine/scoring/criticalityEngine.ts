/**
 * Strategy 5: Unified Criticality Index (Hybrid Ensemble)
 *
 * Combines all four strategies into a single weighted score.
 * This is the main entry point for criticality scoring.
 */
import type { TopologyGraph } from '../types/topology.types';
import type {
  CriticalityResult,
  CriticalityTier,
  EnsembleWeights,
} from './types';
import { DEFAULT_ENSEMBLE_WEIGHTS } from './types';
import { computePageRank } from './pageRankStrategy';
import { computeBlastRadiusScores } from './blastRadiusStrategy';
import { computeTrafficBetweenness } from './trafficBetweennessStrategy';
import { computeStructuralRisk } from './structuralRiskStrategy';

export interface CriticalityEngineOptions {
  weights?: Partial<EnsembleWeights>;
  /** Skip expensive strategies for faster results */
  fastMode?: boolean;
}

/**
 * Compute criticality scores for all nodes in the graph.
 *
 * In fast mode, only PageRank and StructuralRisk are computed (skips
 * blast radius BFS and betweenness path-finding).
 */
export function computeCriticality(
  graph: TopologyGraph,
  options: CriticalityEngineOptions = {}
): CriticalityResult[] {
  const weights: EnsembleWeights = {
    ...DEFAULT_ENSEMBLE_WEIGHTS,
    ...options.weights,
  };

  if (graph.nodes.length === 0) return [];

  // Phase 1: Fast strategies (always computed)
  const pageRankScores = computePageRank(graph);
  const structuralRiskScores = computeStructuralRisk(graph);

  // Phase 2: Expensive strategies (skipped in fast mode)
  let blastRadiusScores: Map<string, number>;
  let betweennessScores: Map<string, number>;

  if (options.fastMode) {
    // In fast mode, redistribute weights to available strategies
    const fastWeights = {
      pageRank: weights.pageRank + weights.blastRadius * 0.5 + weights.trafficBetweenness * 0.5,
      structuralRisk: weights.structuralRisk + weights.blastRadius * 0.5 + weights.trafficBetweenness * 0.5,
    };

    blastRadiusScores = new Map();
    betweennessScores = new Map();

    return graph.nodes.map(node => {
      const pr = pageRankScores.get(node.id) ?? 0;
      const sr = structuralRiskScores.get(node.id) ?? 0;
      const score = fastWeights.pageRank * pr + fastWeights.structuralRisk * sr;
      const clampedScore = Math.min(100, Math.max(0, score));

      return {
        nodeId: node.id,
        score: Math.round(clampedScore * 10) / 10,
        tier: getTier(clampedScore),
        strategies: {
          pageRank: Math.round(pr * 10) / 10,
          blastRadius: 0,
          trafficBetweenness: 0,
          structuralRisk: Math.round(sr * 10) / 10,
        },
      };
    });
  }

  blastRadiusScores = computeBlastRadiusScores(graph);
  betweennessScores = computeTrafficBetweenness(graph);

  // Phase 3: Combine into unified score
  return graph.nodes.map(node => {
    const pr = pageRankScores.get(node.id) ?? 0;
    const br = blastRadiusScores.get(node.id) ?? 0;
    const tb = betweennessScores.get(node.id) ?? 0;
    const sr = structuralRiskScores.get(node.id) ?? 0;

    const score =
      weights.pageRank * pr +
      weights.blastRadius * br +
      weights.trafficBetweenness * tb +
      weights.structuralRisk * sr;

    const clampedScore = Math.min(100, Math.max(0, score));

    return {
      nodeId: node.id,
      score: Math.round(clampedScore * 10) / 10,
      tier: getTier(clampedScore),
      strategies: {
        pageRank: Math.round(pr * 10) / 10,
        blastRadius: Math.round(br * 10) / 10,
        trafficBetweenness: Math.round(tb * 10) / 10,
        structuralRisk: Math.round(sr * 10) / 10,
      },
    };
  });
}

function getTier(score: number): CriticalityTier {
  if (score >= 80) return 'critical';
  if (score >= 50) return 'high';
  if (score >= 20) return 'moderate';
  return 'low';
}

/** Get color for criticality tier (matches existing overlay pattern) */
export function getCriticalityColor(tier: CriticalityTier): string {
  switch (tier) {
    case 'critical': return '#E53935';
    case 'high': return '#FF9800';
    case 'moderate': return '#4CAF50';
    case 'low': return '#2196F3';
  }
}

/** Get sorted results (most critical first) */
export function getTopCritical(results: CriticalityResult[], limit = 10): CriticalityResult[] {
  return [...results].sort((a, b) => b.score - a.score).slice(0, limit);
}
