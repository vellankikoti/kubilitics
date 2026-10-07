import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { Node, Edge } from "@xyflow/react";
import type { BaseNodeData } from "../nodes/BaseNode";
import type { LabeledEdgeData } from "../edges/LabeledEdge";
import type { TopologyResponse, ViewMode } from "../types/topology";
import { getNodeDims } from "../constants/designTokens";
import { categoryGridLayout } from "../layout/layoutEngine";
import type { LayoutRequestMessage, LayoutResponseMessage } from "../workers/elkLayout.worker";
import { createElkLayoutWorker } from "../workers/createElkLayoutWorker";

// Safety net against a genuinely pathological/hung layout computation
// (distinct from TopologyCanvas's 5s "taking too long" UI message, which is
// deliberately much shorter so the user sees feedback quickly on a merely
// slow — not stuck — large graph). If the worker hasn't responded within
// this window, the request is actually terminated (not just reported as
// slow) and falls back to the category grid.
const LAYOUT_HARD_TIMEOUT_MS = 30_000;

/**
 * useElkLayout — Topology layout engine.
 *
 * The actual algorithm (ELK layered / hybrid / category grid — see
 * ../layout/layoutEngine.ts for the strategy writeup) runs inside a Web
 * Worker (Batch 3 / Theme 3 #16) so a large graph's layout never blocks
 * the main thread or UI interaction. This hook owns:
 *   - the worker's lifecycle (one worker per mount; terminated and replaced
 *     if a newer layout request supersedes one still in flight — this is
 *     what makes the 5s "taking too long" UI state in TopologyCanvas.tsx
 *     correspond to real, reclaimed work instead of a stuck background
 *     computation that runs to completion anyway — Theme 3 #20)
 *   - a shape-fingerprint cache so a `topology` update that doesn't change
 *     the node/edge ID set (only e.g. a status field) reuses the last
 *     computed positions instead of re-invoking the worker at all
 *     (Theme 3 #15)
 */

export function useElkLayout(
  topology: TopologyResponse | null,
  viewMode: ViewMode = "namespace",
  nodeType: string = "base"
) {
  // IMPORTANT: Layout computation always uses "base" dimensions so that:
  // 1. Switching semantic zoom (compact/base/expanded) never triggers re-layout
  // 2. Export mode (which locks nodeType to "base") never triggers re-layout
  // 3. Node spacing is always computed for the largest card size (base)
  // The `nodeType` parameter only controls visual rendering via the useMemo below.
  const [positionedNodes, setPositionedNodes] = useState<
    Array<{ id: string; x: number; y: number; data: BaseNodeData }>
  >([]);
  const [layoutEdges, setLayoutEdges] = useState<Edge<LabeledEdgeData>[]>([]);
  const [isLayouting, setIsLayouting] = useState(false);
  const workerRef = useRef<Worker | null>(null);
  const layoutGenRef = useRef(0);
  const pendingRef = useRef<Map<number, {
    resolve: (positions: Map<string, { x: number; y: number }>) => void;
    reject: (err: Error) => void;
  }>>(new Map());

  // Shape-fingerprint gate: `topology` gets a new object reference on every
  // fetch/WebSocket update even when the node/edge ID set is identical (only
  // status/metrics/labels changed) — this previously re-ran the full layout
  // algorithm (ELK or grid) on every such update. Caching the last computed
  // positions keyed by the shape (which nodes/edges actually participate in
  // layout, plus viewMode since that changes ELK_OPTIONS) lets an unchanged
  // shape reuse cached positions and skip straight to re-rendering with
  // fresh node data — a real relayout still runs whenever the shape changes.
  const lastShapeFingerprintRef = useRef<string | null>(null);
  const lastPositionsRef = useRef<Map<string, { x: number; y: number }> | null>(null);

  const terminateWorker = useCallback((reason: string) => {
    if (workerRef.current) {
      workerRef.current.terminate();
      workerRef.current = null;
    }
    // Reject any requests still awaiting a response from the killed worker
    // so their callers' promises settle instead of hanging forever.
    for (const { reject } of pendingRef.current.values()) {
      reject(new Error(reason));
    }
    pendingRef.current.clear();
  }, []);

  const getOrCreateWorker = useCallback((): Worker => {
    if (workerRef.current) return workerRef.current;
    const worker = createElkLayoutWorker();
    worker.onmessage = (ev: MessageEvent<LayoutResponseMessage>) => {
      const { requestId, positions, error } = ev.data;
      const pending = pendingRef.current.get(requestId);
      if (!pending) return; // superseded/already settled
      pendingRef.current.delete(requestId);
      if (error) {
        pending.reject(new Error(error));
      } else {
        pending.resolve(new Map(positions ?? []));
      }
    };
    worker.onerror = () => {
      // The worker itself crashed (not a caught computation error, which
      // comes through onmessage above). Reject everything in flight and
      // let the next layout request spin up a fresh worker.
      for (const { reject } of pendingRef.current.values()) {
        reject(new Error("layout worker crashed"));
      }
      pendingRef.current.clear();
      workerRef.current = null;
    };
    workerRef.current = worker;
    return worker;
  }, []);

  useEffect(() => {
    return () => {
      terminateWorker("component unmounted");
    };
  }, [terminateWorker]);

  // Main layout computation
  const computeLayout = useCallback(async () => {
    if (!topology?.nodes?.length) {
      setPositionedNodes([]);
      setLayoutEdges([]);
      return;
    }

    const nodeIds = new Set(topology.nodes.map((n) => n.id));
    const validEdges = topology.edges.filter(
      (e) => nodeIds.has(e.source) && nodeIds.has(e.target)
    );

    const shapeFingerprint =
      viewMode + "|" +
      Array.from(nodeIds).sort().join(",") + "|" +
      validEdges.map((e) => e.id).sort().join(",");
    const cachedPositions =
      shapeFingerprint === lastShapeFingerprintRef.current ? lastPositionsRef.current : null;

    const gen = ++layoutGenRef.current;

    // A previous request is still in flight — it's about to be superseded.
    // Terminate its worker now (reclaiming the CPU it was burning) instead
    // of letting it run to completion only to be discarded by the
    // staleness check below. Skipped entirely on a cache hit, since that
    // path never touches the worker.
    if (!cachedPositions && pendingRef.current.size > 0) {
      terminateWorker("superseded by a newer layout request");
    }

    if (!cachedPositions) setIsLayouting(true);

    const dims = getNodeDims("base");
    const focusNodeId = viewMode === "resource"
      ? topology.metadata?.focusResource
      : undefined;

    try {
      let positions: Map<string, { x: number; y: number }>;

      if (cachedPositions) {
        positions = cachedPositions;
      } else {
        const worker = getOrCreateWorker();
        positions = await new Promise<Map<string, { x: number; y: number }>>((resolve, reject) => {
          const hardTimeoutId = setTimeout(() => {
            terminateWorker(`layout computation exceeded ${LAYOUT_HARD_TIMEOUT_MS}ms — treating as stuck`);
          }, LAYOUT_HARD_TIMEOUT_MS);
          pendingRef.current.set(gen, {
            resolve: (p) => { clearTimeout(hardTimeoutId); resolve(p); },
            reject: (err) => { clearTimeout(hardTimeoutId); reject(err); },
          });
          const message: LayoutRequestMessage = {
            requestId: gen,
            topology,
            viewMode,
            validEdges,
            nodeWidth: dims.width,
            nodeHeight: dims.height,
            focusNodeId,
          };
          worker.postMessage(message);
        });
      }

      // Stale check — a newer computeLayout call started (and already
      // terminated our worker) while we were awaiting this one.
      if (gen !== layoutGenRef.current) return;

      lastShapeFingerprintRef.current = shapeFingerprint;
      lastPositionsRef.current = positions;

      const positioned = topology.nodes.map((tn) => {
        const pos = positions.get(tn.id) ?? { x: 0, y: 0 };
        return {
          id: tn.id,
          x: pos.x,
          y: pos.y,
          data: {
            kind: tn.kind,
            name: tn.name,
            namespace: tn.namespace || undefined,
            category: tn.category,
            status: mapStatus(tn.status),
            statusReason: tn.statusReason ?? tn.status,
            metrics: tn.metrics,
            labels: tn.labels,
            createdAt: tn.createdAt,
          } as BaseNodeData,
        };
      });

      const edges: Edge<LabeledEdgeData>[] = validEdges.map((e) => ({
        id: e.id,
        source: e.source,
        target: e.target,
        type: "labeled",
        animated: e.animated ?? false,
        data: {
          label: e.label,
          detail: e.detail,
          relationshipCategory: e.relationshipCategory,
          relationshipType: e.relationshipType,
          healthy: e.healthy,
        },
      }));

      setPositionedNodes(positioned);
      setLayoutEdges(edges);
    } catch (err) {
      console.warn("[useElkLayout] Layout failed, using category grid:", err);
      if (gen !== layoutGenRef.current) return;

      const positions = categoryGridLayout(topology);
      lastShapeFingerprintRef.current = shapeFingerprint;
      lastPositionsRef.current = positions;

      const positioned = topology.nodes.map((tn) => {
        const pos = positions.get(tn.id) ?? { x: 0, y: 0 };
        return {
          id: tn.id,
          x: pos.x,
          y: pos.y,
          data: {
            kind: tn.kind,
            name: tn.name,
            namespace: tn.namespace || undefined,
            category: tn.category,
            status: mapStatus(tn.status),
            statusReason: tn.statusReason ?? tn.status,
          } as BaseNodeData,
        };
      });
      setPositionedNodes(positioned);
      setLayoutEdges(
        topology.edges
          .filter((e) => nodeIds.has(e.source) && nodeIds.has(e.target))
          .map((e) => ({
            id: e.id,
            source: e.source,
            target: e.target,
            type: "labeled",
            data: {
              label: e.label,
              detail: e.detail,
              relationshipCategory: e.relationshipCategory,
              relationshipType: e.relationshipType,
              healthy: e.healthy,
            },
          }))
      );
    } finally {
      if (gen === layoutGenRef.current) setIsLayouting(false);
    }
  // NOTE: nodeType intentionally excluded — layout always uses "base" dims.
  // nodeType only affects the visual rendering (useMemo below), not layout positions.
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [topology, viewMode, terminateWorker, getOrCreateWorker]);

  useEffect(() => {
    computeLayout();
  }, [computeLayout]);

  // Apply nodeType (semantic zoom) without re-layout
  const nodes: Node<BaseNodeData>[] = useMemo(
    () =>
      positionedNodes.map((pn) => ({
        id: pn.id,
        type: nodeType,
        position: { x: pn.x, y: pn.y },
        data: pn.data,
      })),
    [positionedNodes, nodeType]
  );

  return { nodes, edges: layoutEdges, isLayouting };
}

function mapStatus(
  status: string
): "healthy" | "warning" | "error" | "unknown" {
  if (
    ["healthy", "Running", "Ready", "Bound", "Available", "Completed", "Active"].includes(status)
  )
    return "healthy";
  if (["Pending", "warning", "PartiallyAvailable"].includes(status))
    return "warning";
  if (["Failed", "error", "NotReady", "Lost", "CrashLoopBackOff"].includes(status))
    return "error";
  return "unknown";
}
