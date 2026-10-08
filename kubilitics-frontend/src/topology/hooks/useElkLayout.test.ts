/**
 * Batch 3 / Theme 3 #15/#16/#20: useElkLayout now runs layout inside a Web
 * Worker instead of the main thread (#16), gates relayout on an actual
 * shape change instead of every `topology` object-reference change (#15),
 * and terminates a superseded in-flight worker request instead of letting
 * stale work run to completion uselessly (#20 — the previous 5s UI
 * "timeout" was cosmetic and never actually cancelled anything).
 *
 * jsdom doesn't implement module Web Workers, so createElkLayoutWorker is
 * mocked with a fake Worker-shaped object (onmessage/postMessage/
 * terminate) that simulates the real worker's async request/response
 * protocol via a microtask, letting these tests exercise the hook's
 * request/cache/cancellation logic without a real worker thread.
 */
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { useElkLayout } from './useElkLayout';
import type { TopologyResponse, TopologyNode, TopologyEdge } from '../types/topology';
import type { LayoutRequestMessage, LayoutResponseMessage } from '../workers/elkLayout.worker';

interface FakeWorker {
  onmessage: ((ev: MessageEvent<LayoutResponseMessage>) => void) | null;
  onerror: (() => void) | null;
  postMessage: ReturnType<typeof vi.fn>;
  terminate: ReturnType<typeof vi.fn>;
}

let lastWorker: FakeWorker | null = null;
const createWorkerMock = vi.fn((): FakeWorker => {
  const worker: FakeWorker = {
    onmessage: null,
    onerror: null,
    postMessage: vi.fn((msg: LayoutRequestMessage) => {
      queueMicrotask(() => {
        const positions: Array<[string, { x: number; y: number }]> = msg.topology.nodes.map(
          (n, i) => [n.id, { x: i * 100, y: 0 }],
        );
        worker.onmessage?.({ data: { requestId: msg.requestId, positions } } as MessageEvent<LayoutResponseMessage>);
      });
    }),
    terminate: vi.fn(),
  };
  lastWorker = worker;
  return worker;
});

vi.mock('../workers/createElkLayoutWorker', () => ({
  createElkLayoutWorker: () => createWorkerMock(),
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

describe('useElkLayout', () => {
  beforeEach(() => {
    createWorkerMock.mockClear();
    lastWorker = null;
  });

  it('posts a message to the worker for the first layout and resolves positions', async () => {
    const topo = makeTopology([node('a'), node('b')], [edge('e1', 'a', 'b')]);
    const { result } = renderHook(() => useElkLayout(topo, 'namespace'));

    await waitFor(() => expect(result.current.nodes.length).toBe(2));
    expect(createWorkerMock).toHaveBeenCalledTimes(1);
    expect(result.current.nodes.find((n) => n.id === 'a')?.position).toEqual({ x: 0, y: 0 });
  });

  it('reuses cached positions (no worker message) when the shape is unchanged across topology updates', async () => {
    const topoV1 = makeTopology([node('a'), node('b')], [edge('e1', 'a', 'b')]);
    const { result, rerender } = renderHook(
      ({ topology }: { topology: TopologyResponse }) => useElkLayout(topology, 'namespace'),
      { initialProps: { topology: topoV1 } },
    );

    await waitFor(() => expect(result.current.nodes.length).toBe(2));
    const postCallsAfterFirst = lastWorker!.postMessage.mock.calls.length;
    expect(postCallsAfterFirst).toBeGreaterThan(0);

    // Same shape, new object reference, status changed — simulates a
    // WebSocket-driven refetch that didn't change the graph's topology.
    const topoV2: TopologyResponse = {
      ...topoV1,
      nodes: [{ ...topoV1.nodes[0], status: 'Failed' }, topoV1.nodes[1]],
    };
    rerender({ topology: topoV2 });

    await waitFor(() => {
      expect(result.current.nodes.find((n) => n.id === 'a')?.data.status).toBe('error');
    });

    // No new postMessage — the cache hit skipped the worker entirely.
    expect(lastWorker!.postMessage.mock.calls.length).toBe(postCallsAfterFirst);
    expect(result.current.nodes.find((n) => n.id === 'a')?.position).toEqual({ x: 0, y: 0 });
  });

  it('re-runs layout (new worker message) when a node is added', async () => {
    const topoV1 = makeTopology([node('a'), node('b')], [edge('e1', 'a', 'b')]);
    const { result, rerender } = renderHook(
      ({ topology }: { topology: TopologyResponse }) => useElkLayout(topology, 'namespace'),
      { initialProps: { topology: topoV1 } },
    );
    await waitFor(() => expect(result.current.nodes.length).toBe(2));

    const topoV2 = makeTopology(
      [node('a'), node('b'), node('c')],
      [edge('e1', 'a', 'b'), edge('e2', 'b', 'c')],
    );
    rerender({ topology: topoV2 });

    await waitFor(() => expect(result.current.nodes.length).toBe(3));
  });

  it('terminates a superseded worker request instead of letting it run to completion (real cancellation)', async () => {
    const topoV1 = makeTopology([node('a'), node('b')], [edge('e1', 'a', 'b')]);
    const { rerender } = renderHook(
      ({ topology }: { topology: TopologyResponse }) => useElkLayout(topology, 'namespace'),
      { initialProps: { topology: topoV1 } },
    );

    // Don't await the first layout settling — immediately supersede it with
    // a different shape while the first request is still "in flight"
    // (its queueMicrotask response hasn't fired yet).
    const firstWorker = lastWorker!;
    const topoV2 = makeTopology(
      [node('a'), node('b'), node('c')],
      [edge('e1', 'a', 'b'), edge('e2', 'b', 'c')],
    );
    rerender({ topology: topoV2 });

    expect(firstWorker.terminate).toHaveBeenCalledTimes(1);
    // A fresh worker must have been created for the new request.
    expect(createWorkerMock).toHaveBeenCalledTimes(2);
  });

  it('terminates and falls back to category grid if the worker never responds (hard timeout safety net)', async () => {
    vi.useFakeTimers();
    try {
      createWorkerMock.mockImplementationOnce(() => {
        const worker: FakeWorker = {
          onmessage: null,
          onerror: null,
          postMessage: vi.fn(), // never responds — simulates a hung/pathological computation
          terminate: vi.fn(),
        };
        lastWorker = worker;
        return worker;
      });

      const topo = makeTopology([node('a'), node('b')], [edge('e1', 'a', 'b')]);
      const { result } = renderHook(() => useElkLayout(topo, 'namespace'));

      await vi.advanceTimersByTimeAsync(30_000);

      expect(lastWorker!.terminate).toHaveBeenCalledTimes(1);
      vi.useRealTimers();
      await waitFor(() => expect(result.current.nodes.length).toBe(2));
      return;
    } finally {
      vi.useRealTimers();
    }
  });

  it('falls back to a category grid when the worker reports an error', async () => {
    createWorkerMock.mockImplementationOnce(() => {
      const worker: FakeWorker = {
        onmessage: null,
        onerror: null,
        postMessage: vi.fn((msg: LayoutRequestMessage) => {
          queueMicrotask(() => {
            worker.onmessage?.({ data: { requestId: msg.requestId, error: 'boom' } } as MessageEvent<LayoutResponseMessage>);
          });
        }),
        terminate: vi.fn(),
      };
      lastWorker = worker;
      return worker;
    });

    const topo = makeTopology([node('a'), node('b')], [edge('e1', 'a', 'b')]);
    const { result } = renderHook(() => useElkLayout(topo, 'namespace'));

    await waitFor(() => expect(result.current.nodes.length).toBe(2));
    // Category grid fallback still produces valid positions, not a crash.
    expect(result.current.nodes.every((n) => typeof n.position.x === 'number')).toBe(true);
  });
});
