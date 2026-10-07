/**
 * Batch 3 / Theme 3 #15: `topology` gets a new object reference on every
 * fetch/WebSocket update even when the node/edge ID set (the "shape") is
 * identical — only status/metrics changed. Previously this re-ran the full
 * layout algorithm (ELK or grid) every time. These tests cover the
 * shape-fingerprint gate that gets added: an unchanged shape reuses cached
 * positions (ELK not re-invoked), a changed shape (node/edge added/removed)
 * always re-runs layout. No test existed for this hook at all before this.
 */
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { useElkLayout } from './useElkLayout';
import type { TopologyResponse, TopologyNode, TopologyEdge } from '../types/topology';

const layoutMock = vi.fn(async (graph: { children: Array<{ id: string }> }) => ({
  children: graph.children.map((c, i) => ({ id: c.id, x: i * 100, y: 0 })),
}));

vi.mock('elkjs/lib/elk.bundled.js', () => ({
  default: class MockElk {
    layout = layoutMock;
  },
}));

function node(id: string): TopologyNode {
  return {
    id,
    kind: 'Pod',
    name: id,
    namespace: 'default',
    apiVersion: 'v1',
    category: 'workload',
    label: id,
    status: 'Running',
    layer: 0,
  };
}

function edge(id: string, source: string, target: string): TopologyEdge {
  return {
    id,
    source,
    target,
    relationshipType: 'owns',
    relationshipCategory: 'workload',
    label: 'owns',
    style: 'solid',
    healthy: true,
  };
}

function makeTopology(nodes: TopologyNode[], edges: TopologyEdge[]): TopologyResponse {
  return {
    metadata: { clusterId: 'c1', clusterName: 'c1', mode: 'namespace', resourceCount: nodes.length, edgeCount: edges.length, buildTimeMs: 0 },
    nodes,
    edges,
    groups: [],
  };
}

describe('useElkLayout shape-fingerprint gate', () => {
  beforeEach(() => {
    layoutMock.mockClear();
  });

  it('reuses cached positions (no ELK call) when the shape is unchanged across topology updates', async () => {
    const topoV1 = makeTopology([node('a'), node('b')], [edge('e1', 'a', 'b')]);
    const { result, rerender } = renderHook(
      ({ topology }: { topology: TopologyResponse }) => useElkLayout(topology, 'namespace'),
      { initialProps: { topology: topoV1 } },
    );

    await waitFor(() => expect(result.current.nodes.length).toBe(2));
    const callsAfterFirst = layoutMock.mock.calls.length;
    expect(callsAfterFirst).toBeGreaterThan(0);

    // Same shape (same node/edge IDs), but a NEW object reference and
    // different status on node "a" — simulates a WebSocket-driven refetch.
    const topoV2: TopologyResponse = {
      ...topoV1,
      nodes: [{ ...topoV1.nodes[0], status: 'Failed' }, topoV1.nodes[1]],
    };
    rerender({ topology: topoV2 });

    await waitFor(() => {
      expect(result.current.nodes.find((n) => n.id === 'a')?.data.status).toBe('error');
    });

    // ELK must NOT have been invoked again — the shape didn't change.
    expect(layoutMock.mock.calls.length).toBe(callsAfterFirst);
    // Positions must be identical to the first computation (reused from cache).
    const posA1 = result.current.nodes.find((n) => n.id === 'a')!.position;
    expect(posA1).toEqual({ x: 0, y: 0 });
  });

  it('re-runs layout when a node is added (shape changed)', async () => {
    const topoV1 = makeTopology([node('a'), node('b')], [edge('e1', 'a', 'b')]);
    const { result, rerender } = renderHook(
      ({ topology }: { topology: TopologyResponse }) => useElkLayout(topology, 'namespace'),
      { initialProps: { topology: topoV1 } },
    );
    await waitFor(() => expect(result.current.nodes.length).toBe(2));
    const callsAfterFirst = layoutMock.mock.calls.length;

    const topoV2 = makeTopology(
      [node('a'), node('b'), node('c')],
      [edge('e1', 'a', 'b'), edge('e2', 'b', 'c')],
    );
    rerender({ topology: topoV2 });

    await waitFor(() => expect(result.current.nodes.length).toBe(3));
    expect(layoutMock.mock.calls.length).toBeGreaterThan(callsAfterFirst);
  });

  it('re-runs layout when an edge is removed (shape changed, same node count)', async () => {
    const topoV1 = makeTopology(
      [node('a'), node('b'), node('c')],
      [edge('e1', 'a', 'b'), edge('e2', 'b', 'c')],
    );
    const { result, rerender } = renderHook(
      ({ topology }: { topology: TopologyResponse }) => useElkLayout(topology, 'namespace'),
      { initialProps: { topology: topoV1 } },
    );
    await waitFor(() => expect(result.current.nodes.length).toBe(3));
    const callsAfterFirst = layoutMock.mock.calls.length;

    const topoV2 = makeTopology(
      [node('a'), node('b'), node('c')],
      [edge('e1', 'a', 'b')], // e2 removed
    );
    rerender({ topology: topoV2 });

    await waitFor(() => {
      // isLayouting flips true->false around the recompute; just wait for
      // a subsequent layout call to have happened.
      expect(layoutMock.mock.calls.length).toBeGreaterThan(callsAfterFirst);
    });
  });

  it('re-runs layout when viewMode changes for the same topology shape', async () => {
    const topo = makeTopology([node('a'), node('b')], [edge('e1', 'a', 'b')]);
    const { result, rerender } = renderHook(
      ({ viewMode }: { viewMode: 'namespace' | 'cluster' }) => useElkLayout(topo, viewMode),
      { initialProps: { viewMode: 'namespace' as const } },
    );
    await waitFor(() => expect(result.current.nodes.length).toBe(2));
    const callsAfterFirst = layoutMock.mock.calls.length;

    rerender({ viewMode: 'cluster' });

    await waitFor(() => {
      expect(layoutMock.mock.calls.length).toBeGreaterThan(callsAfterFirst);
    });
  });
});
