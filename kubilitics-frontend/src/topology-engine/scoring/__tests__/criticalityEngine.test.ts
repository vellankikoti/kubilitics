import { describe, it, expect } from 'vitest';
import type { TopologyGraph, TopologyNode, TopologyEdge } from '../../types/topology.types';
import { computeCriticality, getTopCritical, getCriticalityColor } from '../criticalityEngine';
import { computePageRank } from '../pageRankStrategy';
import { computeStructuralRisk } from '../structuralRiskStrategy';
import { computeTrafficBetweenness } from '../trafficBetweennessStrategy';

function makeNode(id: string, kind: TopologyNode['kind'], overrides: Partial<TopologyNode> = {}): TopologyNode {
  return {
    id,
    kind,
    namespace: 'default',
    name: id,
    apiVersion: 'v1',
    status: 'Running',
    label: id,
    metadata: { labels: {}, annotations: {}, createdAt: '2024-01-01', uid: id },
    computed: { health: 'healthy' },
    ...overrides,
  };
}

function makeEdge(source: string, target: string, type: string, confidence = 0.8): TopologyEdge {
  return {
    id: `${source}-${target}`,
    source,
    target,
    relationshipType: type as any,
    label: type,
    metadata: { derivation: 'test', confidence, sourceField: 'test' },
  };
}

function makeGraph(nodes: TopologyNode[], edges: TopologyEdge[]): TopologyGraph {
  return {
    schemaVersion: '1.0',
    nodes,
    edges,
    metadata: {
      clusterId: 'test',
      generatedAt: '2024-01-01',
      layoutSeed: '1',
      isComplete: true,
      warnings: [],
    },
  };
}

describe('criticalityEngine', () => {
  const graph = makeGraph(
    [
      makeNode('ingress-1', 'Ingress'),
      makeNode('svc-api', 'Service'),
      makeNode('deploy-api', 'Deployment', {
        computed: { health: 'healthy', replicas: { desired: 3, ready: 3, available: 3 } },
      }),
      makeNode('rs-api', 'ReplicaSet'),
      makeNode('pod-api-1', 'Pod'),
      makeNode('pod-api-2', 'Pod'),
      makeNode('svc-db', 'Service'),
      makeNode('sts-db', 'StatefulSet', {
        computed: { health: 'warning', replicas: { desired: 1, ready: 1, available: 1 } },
      }),
      makeNode('pod-db-1', 'Pod'),
      makeNode('cm-config', 'ConfigMap'),
    ],
    [
      makeEdge('ingress-1', 'svc-api', 'routes', 0.9),
      makeEdge('svc-api', 'pod-api-1', 'selects', 0.9),
      makeEdge('svc-api', 'pod-api-2', 'selects', 0.9),
      makeEdge('deploy-api', 'rs-api', 'owns', 1.0),
      makeEdge('rs-api', 'pod-api-1', 'owns', 1.0),
      makeEdge('rs-api', 'pod-api-2', 'owns', 1.0),
      makeEdge('svc-db', 'pod-db-1', 'selects', 0.9),
      makeEdge('sts-db', 'pod-db-1', 'owns', 1.0),
      makeEdge('pod-api-1', 'svc-db', 'references', 0.7),
      makeEdge('pod-api-2', 'svc-db', 'references', 0.7),
      makeEdge('pod-api-1', 'cm-config', 'mounts', 0.6),
      makeEdge('pod-api-2', 'cm-config', 'mounts', 0.6),
    ]
  );

  it('returns scores for all nodes', () => {
    const results = computeCriticality(graph);
    expect(results.length).toBe(graph.nodes.length);
    for (const r of results) {
      expect(r.score).toBeGreaterThanOrEqual(0);
      expect(r.score).toBeLessThanOrEqual(100);
      expect(['critical', 'high', 'moderate', 'low']).toContain(r.tier);
    }
  });

  it('scores ingress and service higher than leaf pods', () => {
    const results = computeCriticality(graph);
    const byId = new Map(results.map(r => [r.nodeId, r]));

    const ingressScore = byId.get('ingress-1')!.score;
    const svcApiScore = byId.get('svc-api')!.score;
    const podScore = byId.get('pod-api-1')!.score;

    // Ingress/Service should generally score higher than individual pods
    expect(ingressScore + svcApiScore).toBeGreaterThan(podScore * 2);
  });

  it('penalizes single-replica StatefulSet (structural risk)', () => {
    const riskScores = computeStructuralRisk(graph);
    const stsScore = riskScores.get('sts-db')!;
    const deployScore = riskScores.get('deploy-api')!;

    // StatefulSet with 1 replica + warning health should score very high
    expect(stsScore).toBeGreaterThan(deployScore);
  });

  it('fast mode skips blast radius and betweenness', () => {
    const full = computeCriticality(graph);
    const fast = computeCriticality(graph, { fastMode: true });

    expect(fast.length).toBe(full.length);
    for (const r of fast) {
      expect(r.strategies.blastRadius).toBe(0);
      expect(r.strategies.trafficBetweenness).toBe(0);
    }
  });

  it('handles empty graph', () => {
    const empty = makeGraph([], []);
    const results = computeCriticality(empty);
    expect(results).toEqual([]);
  });

  it('getTopCritical returns sorted subset', () => {
    const results = computeCriticality(graph);
    const top3 = getTopCritical(results, 3);
    expect(top3.length).toBe(3);
    expect(top3[0].score).toBeGreaterThanOrEqual(top3[1].score);
    expect(top3[1].score).toBeGreaterThanOrEqual(top3[2].score);
  });

  it('getCriticalityColor returns correct colors', () => {
    expect(getCriticalityColor('critical')).toBe('#E53935');
    expect(getCriticalityColor('high')).toBe('#FF9800');
    expect(getCriticalityColor('moderate')).toBe('#4CAF50');
    expect(getCriticalityColor('low')).toBe('#2196F3');
  });
});

describe('pageRank', () => {
  it('gives higher score to nodes with more dependents', () => {
    const graph = makeGraph(
      [
        makeNode('hub', 'Service'),
        makeNode('a', 'Pod'),
        makeNode('b', 'Pod'),
        makeNode('c', 'Pod'),
        makeNode('leaf', 'Pod'),
      ],
      [
        makeEdge('hub', 'a', 'selects'),
        makeEdge('hub', 'b', 'selects'),
        makeEdge('hub', 'c', 'selects'),
        makeEdge('a', 'leaf', 'references'),
      ]
    );

    const scores = computePageRank(graph);
    // Hub has most outgoing edges (reversed = most incoming in PageRank)
    // so nodes that hub points to should contribute back to hub
    // But actually in reversed graph, a/b/c point to hub, so hub gets high score
    expect(scores.get('hub')!).toBeGreaterThan(scores.get('leaf')!);
  });
});

describe('trafficBetweenness', () => {
  it('scores intermediate nodes on paths between ingress and backend', () => {
    const graph = makeGraph(
      [
        makeNode('ing', 'Ingress'),
        makeNode('svc', 'Service'),
        makeNode('deploy', 'Deployment'),
        makeNode('pv', 'PersistentVolume'),
      ],
      [
        makeEdge('ing', 'svc', 'routes'),
        makeEdge('svc', 'deploy', 'selects'),
        makeEdge('deploy', 'pv', 'mounts'),
      ]
    );

    const scores = computeTrafficBetweenness(graph);
    // svc and deploy are on the path between ingress and PV
    const svcScore = scores.get('svc')!;
    const deployScore = scores.get('deploy')!;
    // At least one intermediate node should have a score > 0
    expect(svcScore + deployScore).toBeGreaterThan(0);
  });
});
