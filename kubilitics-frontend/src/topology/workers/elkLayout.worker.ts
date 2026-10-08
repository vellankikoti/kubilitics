/**
 * Web Worker entry point for topology layout (Batch 3 / Theme 3 #16).
 * Runs ELK (and the grid/hybrid fallback algorithms) off the main thread so
 * a large graph's layout computation never blocks UI interaction. Owns its
 * own ElkConstructor instance for its whole lifetime — reused across
 * requests, matching the previous main-thread behavior of lazily loading
 * elkjs once and reusing the instance.
 */
import ElkConstructor from "elkjs/lib/elk.bundled.js";
import { computeGraphPositions, type ValidEdge } from "../layout/layoutEngine";
import type { TopologyResponse, ViewMode } from "../types/topology";

export interface LayoutRequestMessage {
  requestId: number;
  topology: TopologyResponse;
  viewMode: ViewMode;
  validEdges: ValidEdge[];
  nodeWidth: number;
  nodeHeight: number;
  focusNodeId?: string;
}

export interface LayoutResponseMessage {
  requestId: number;
  positions?: Array<[string, { x: number; y: number }]>;
  error?: string;
}

let elkInstance: InstanceType<typeof ElkConstructor> | null = null;

self.onmessage = async (ev: MessageEvent<LayoutRequestMessage>) => {
  const { requestId, topology, viewMode, validEdges, nodeWidth, nodeHeight, focusNodeId } = ev.data;
  try {
    if (!elkInstance) elkInstance = new ElkConstructor();
    const positions = await computeGraphPositions(
      topology, viewMode, elkInstance, validEdges, nodeWidth, nodeHeight, focusNodeId,
    );
    const response: LayoutResponseMessage = { requestId, positions: Array.from(positions.entries()) };
    (self as unknown as Worker).postMessage(response);
  } catch (err) {
    const response: LayoutResponseMessage = {
      requestId,
      error: err instanceof Error ? err.message : String(err),
    };
    (self as unknown as Worker).postMessage(response);
  }
};
