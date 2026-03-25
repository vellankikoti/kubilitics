/**
 * elkLayout.ts — ELK layout adapter for Graphology graphs.
 *
 * Converts a Graphology graph to ELK input format, runs the ELK layered
 * algorithm, and writes computed x/y positions back to graph node attributes.
 *
 * ELK settings are aligned with useElkLayout.ts (the React Flow layout hook).
 */

import Graph from "graphology";
import ELK from "elkjs/lib/elk.bundled";

// ─── Public Interface ────────────────────────────────────────────────────────

export interface LayoutOptions {
  /** Layout direction. Default: 'RIGHT' (left-to-right). */
  direction?: "RIGHT" | "DOWN";
  /** Spacing between sibling nodes in the same layer. Default: 50. */
  nodeSpacing?: number;
  /** Spacing between layers (ranks). Default: 150. */
  layerSpacing?: number;
}

// ─── K8s Layer Constraints ───────────────────────────────────────────────────
// Assigns ELK layer indices based on Kubernetes resource category so that
// infrastructure sits on the left and workloads on the right.

const KIND_TO_LAYER: Record<string, number> = {
  // Layer 0 — Infrastructure
  Namespace: 0,
  Node: 0,
  // Layer 1 — Networking
  Service: 1,
  Ingress: 1,
  // Layer 2 — Controllers
  Deployment: 2,
  StatefulSet: 2,
  DaemonSet: 2,
  CronJob: 2,
  // Layer 3 — Intermediary
  ReplicaSet: 3,
  Job: 3,
  // Layer 4 — Workload
  Pod: 4,
  PodGroup: 4,
};

function layerForKind(kind: string | undefined): number | undefined {
  if (!kind) return undefined;
  return KIND_TO_LAYER[kind];
}

// ─── ELK Types ───────────────────────────────────────────────────────────────

interface ElkNode {
  id: string;
  width: number;
  height: number;
  layoutOptions?: Record<string, string>;
}

interface ElkEdge {
  id: string;
  sources: string[];
  targets: string[];
}

interface ElkCompoundNode {
  id: string;
  layoutOptions: Record<string, string>;
  children: ElkNode[];
  edges: ElkEdge[];
}

interface ElkGraph {
  id: string;
  layoutOptions: Record<string, string>;
  children: Array<ElkNode | ElkCompoundNode>;
  edges: ElkEdge[];
}

interface ElkLayoutChild {
  id: string;
  x: number;
  y: number;
  children?: ElkLayoutChild[];
}

interface ElkLayoutResult {
  children?: ElkLayoutChild[];
}

// ─── Default Node Dimensions ─────────────────────────────────────────────────
// Match the "base" node size from designTokens.ts (320 x 130).

const DEFAULT_WIDTH = 320;
const DEFAULT_HEIGHT = 130;

// ─── ELK Instance Management ────────────────────────────────────────────────
// Try Web Worker first; fall back to main-thread ELK.

let elkSingleton: InstanceType<typeof ELK> | null = null;

function getElk(): InstanceType<typeof ELK> {
  if (elkSingleton) return elkSingleton;

  try {
    // elkjs/lib/elk.bundled includes a Web Worker shim by default.
    // If the worker cannot be created (e.g. SSR, restrictive CSP),
    // the bundled version still works synchronously on the main thread.
    elkSingleton = new ELK();
  } catch {
    // Fallback: create without worker support
    elkSingleton = new ELK();
  }

  return elkSingleton;
}

// ─── Core Layout Function ────────────────────────────────────────────────────

/**
 * Compute an ELK layered layout for a Graphology graph and write
 * the resulting x/y positions back onto each node's attributes.
 *
 * Node attributes read:
 *   - `width`  (number, optional — defaults to 320)
 *   - `height` (number, optional — defaults to 130)
 *   - `kind`   (string, optional — used for layer constraints)
 *   - `namespace` (string, optional — used for compound grouping)
 *
 * Node attributes written:
 *   - `x` (number)
 *   - `y` (number)
 */
export async function computeElkLayout(
  graph: Graph,
  options?: LayoutOptions,
): Promise<void> {
  if (graph.order === 0) return;

  const direction = options?.direction ?? "RIGHT";
  const nodeSpacing = options?.nodeSpacing ?? 50;
  const layerSpacing = options?.layerSpacing ?? 150;

  // ── Build ELK layout options (matching useElkLayout.ts) ──────────────────
  const layoutOptions: Record<string, string> = {
    "elk.algorithm": "layered",
    "elk.direction": direction,
    "elk.spacing.nodeNode": String(nodeSpacing),
    "elk.layered.spacing.nodeNodeBetweenLayers": String(layerSpacing),
    "elk.layered.spacing.edgeNodeBetweenLayers": "30",
    "elk.layered.crossingMinimization.strategy": "LAYER_SWEEP",
    "elk.layered.nodePlacement.strategy": "BRANDES_KOEPF",
    "elk.layered.thoroughness": "20",
    "elk.separateConnectedComponents": "true",
    "elk.spacing.componentComponent": "80",
    "elk.randomSeed": "42",
  };

  // ── Group nodes by namespace for compound layout ─────────────────────────
  const namespaceGroups = new Map<string, string[]>();
  const ungroupedNodeIds: string[] = [];

  graph.forEachNode((nodeId, attrs) => {
    const ns: string | undefined = attrs.namespace;
    if (ns) {
      if (!namespaceGroups.has(ns)) namespaceGroups.set(ns, []);
      namespaceGroups.get(ns)!.push(nodeId);
    } else {
      ungroupedNodeIds.push(nodeId);
    }
  });

  // ── Build ELK node for a Graphology node ─────────────────────────────────
  function toElkNode(nodeId: string): ElkNode {
    const attrs = graph.getNodeAttributes(nodeId);
    const w = (attrs.width as number) || DEFAULT_WIDTH;
    const h = (attrs.height as number) || DEFAULT_HEIGHT;
    const kind = attrs.kind as string | undefined;
    const layer = layerForKind(kind);

    const elkNode: ElkNode = { id: nodeId, width: w, height: h };

    if (layer !== undefined) {
      elkNode.layoutOptions = {
        "elk.layered.layerConstraint": String(layer),
      };
    }

    return elkNode;
  }

  // ── Collect edges ────────────────────────────────────────────────────────
  // Partition edges: intra-namespace (go inside compound node) vs
  // cross-namespace or ungrouped (go at root level).
  const rootEdges: ElkEdge[] = [];
  const nsEdges = new Map<string, ElkEdge[]>();

  // Build a nodeId → namespace lookup
  const nodeNs = new Map<string, string>();
  graph.forEachNode((nodeId, attrs) => {
    const ns = attrs.namespace as string | undefined;
    if (ns) nodeNs.set(nodeId, ns);
  });

  graph.forEachEdge((edgeId, _attrs, source, target) => {
    const elkEdge: ElkEdge = {
      id: edgeId,
      sources: [source],
      targets: [target],
    };

    const srcNs = nodeNs.get(source);
    const tgtNs = nodeNs.get(target);

    if (srcNs && tgtNs && srcNs === tgtNs) {
      // Intra-namespace edge — attach to the compound node
      if (!nsEdges.has(srcNs)) nsEdges.set(srcNs, []);
      nsEdges.get(srcNs)!.push(elkEdge);
    } else {
      rootEdges.push(elkEdge);
    }
  });

  // ── Assemble ELK graph ──────────────────────────────────────────────────
  const children: Array<ElkNode | ElkCompoundNode> = [];

  // Namespace compound nodes (groups)
  for (const [ns, nodeIds] of namespaceGroups) {
    // Only create a compound node if the namespace has more than one node.
    // A single-node namespace is better laid out at the root level.
    if (nodeIds.length === 1) {
      ungroupedNodeIds.push(nodeIds[0]);
      continue;
    }

    const compoundNode: ElkCompoundNode = {
      id: `ns:${ns}`,
      layoutOptions: {
        ...layoutOptions,
        // Tighter spacing inside namespace groups
        "elk.padding": "[top=40,left=20,bottom=20,right=20]",
      },
      children: nodeIds.map(toElkNode),
      edges: nsEdges.get(ns) ?? [],
    };

    children.push(compoundNode);
  }

  // Ungrouped nodes at root level
  for (const nodeId of ungroupedNodeIds) {
    children.push(toElkNode(nodeId));
  }

  const elkGraph: ElkGraph = {
    id: "root",
    layoutOptions,
    children,
    edges: rootEdges,
  };

  // ── Run ELK ─────────────────────────────────────────────────────────────
  const elk = getElk();
  let result: ElkLayoutResult;

  try {
    result = (await elk.layout(elkGraph)) as ElkLayoutResult;
  } catch (err) {
    // If ELK fails (e.g. Web Worker issue), try once more without the worker
    console.warn("[elkLayout] ELK layout failed, retrying:", err);
    const fallbackElk = new ELK();
    result = (await fallbackElk.layout(elkGraph)) as ElkLayoutResult;
  }

  // ── Write positions back to Graphology ──────────────────────────────────
  if (!result.children) return;

  for (const child of result.children) {
    if (child.id.startsWith("ns:") && child.children) {
      // Compound node — offset children by the parent's position
      const parentX = child.x ?? 0;
      const parentY = child.y ?? 0;

      for (const inner of child.children) {
        if (graph.hasNode(inner.id)) {
          graph.mergeNodeAttributes(inner.id, {
            x: parentX + (inner.x ?? 0),
            y: parentY + (inner.y ?? 0),
          });
        }
      }
    } else if (graph.hasNode(child.id)) {
      graph.mergeNodeAttributes(child.id, {
        x: child.x ?? 0,
        y: child.y ?? 0,
      });
    }
  }
}
