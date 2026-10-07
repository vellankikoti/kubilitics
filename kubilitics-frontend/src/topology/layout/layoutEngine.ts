/**
 * Pure topology-layout computation, extracted from useElkLayout.ts so it
 * can run inside a Web Worker (Batch 3 / Theme 3 #16) instead of blocking
 * the main thread. No DOM, no React — only plain data in, plain data out.
 * The ELK instance itself is passed in rather than constructed here so the
 * worker (which owns ELK's lifecycle) and any future caller share the same
 * entry point.
 */
import type { TopologyResponse, ViewMode, TopologyNode } from "../types/topology";

export type ValidEdge = {
  id: string;
  source: string;
  target: string;
  label: string;
  detail?: string;
  relationshipCategory?: string;
  healthy?: boolean;
};

export interface ElkLike {
  layout(graph: ElkGraph): Promise<ElkLayoutResult>;
}

// ─── Layer Constraints ─────────────────────────────────────────────────────
// Map backend `layer` field (0-5) to ELK layer constraint properties.

function getLayerConstraint(layer: number | undefined): Record<string, string> {
  if (layer == null) return {};
  return { "elk.layered.layerConstraint": String(layer) };
}

// ─── ELK Configuration ─────────────────────────────────────────────────────

const ELK_LAYERED_BASE: Record<string, string> = {
  "elk.algorithm": "layered",
  "elk.direction": "RIGHT",
  "elk.spacing.nodeNode": "40",
  "elk.layered.spacing.nodeNodeBetweenLayers": "140",
  "elk.layered.spacing.edgeNodeBetweenLayers": "30",
  "elk.layered.crossingMinimization.strategy": "LAYER_SWEEP",
  "elk.layered.nodePlacement.strategy": "BRANDES_KOEPF",
  "elk.layered.thoroughness": "20",
  "elk.separateConnectedComponents": "true",
  "elk.spacing.componentComponent": "80",
};

export const ELK_OPTIONS: Record<ViewMode, Record<string, string>> = {
  namespace: { ...ELK_LAYERED_BASE },
  cluster:   { ...ELK_LAYERED_BASE, "elk.spacing.nodeNode": "70", "elk.layered.spacing.nodeNodeBetweenLayers": "160" },
  rbac:      { ...ELK_LAYERED_BASE, "elk.spacing.nodeNode": "70" },
  traffic:   { ...ELK_LAYERED_BASE },
  resource:  {
    ...ELK_LAYERED_BASE,
    "elk.layered.thoroughness": "40",
    "elk.layered.nodePlacement.strategy": "NETWORK_SIMPLEX",
    "elk.spacing.nodeNode": "50",
    "elk.layered.spacing.nodeNodeBetweenLayers": "160",
    "elk.layered.crossingMinimization.strategy": "LAYER_SWEEP",
    "elk.layered.compaction.connectedComponents": "true",
    "elk.separateConnectedComponents": "false",
  },
};

// ─── Types ──────────────────────────────────────────────────────────────────

export interface ElkGraph {
  id: string;
  layoutOptions: Record<string, string>;
  children: Array<{ id: string; width: number; height: number; layoutOptions?: Record<string, string> }>;
  edges: Array<{ id: string; sources: string[]; targets: string[] }>;
}

export interface ElkLayoutResult {
  children?: Array<{ id: string; x: number; y: number }>;
}

// ─── K8s Category Order (left-to-right dependency flow) ─────────────────────

const CATEGORY_ORDER: Record<string, number> = {
  cluster:    0,
  scheduling: 1,
  compute:    2,
  workload:   2,
  networking: 3,
  config:     4,
  storage:    5,
  security:   6,
  rbac:       6,
  scaling:    7,
  custom:     8,
};

function categoryOrder(cat: string): number {
  return CATEGORY_ORDER[cat] ?? 8;
}

// ─── Connected Component Detection ─────────────────────────────────────────

interface ConnectedComponent {
  nodeIds: Set<string>;
  edgeIds: Set<string>;
}

function findConnectedComponents(
  nodeIds: string[],
  edges: Array<{ id: string; source: string; target: string }>
): ConnectedComponent[] {
  const adj = new Map<string, Set<string>>();
  const edgeMap = new Map<string, Array<{ id: string; source: string; target: string }>>();

  for (const nid of nodeIds) {
    adj.set(nid, new Set());
  }
  for (const e of edges) {
    adj.get(e.source)?.add(e.target);
    adj.get(e.target)?.add(e.source);
    if (!edgeMap.has(e.source)) edgeMap.set(e.source, []);
    if (!edgeMap.has(e.target)) edgeMap.set(e.target, []);
    edgeMap.get(e.source)!.push(e);
    edgeMap.get(e.target)!.push(e);
  }

  const visited = new Set<string>();
  const components: ConnectedComponent[] = [];

  for (const nid of nodeIds) {
    if (visited.has(nid)) continue;

    const component: ConnectedComponent = { nodeIds: new Set(), edgeIds: new Set() };
    const queue = [nid];
    visited.add(nid);

    while (queue.length > 0) {
      const current = queue.pop()!;
      component.nodeIds.add(current);

      for (const e of edgeMap.get(current) ?? []) {
        component.edgeIds.add(e.id);
      }

      for (const neighbor of adj.get(current) ?? []) {
        if (!visited.has(neighbor)) {
          visited.add(neighbor);
          queue.push(neighbor);
        }
      }
    }

    components.push(component);
  }

  return components;
}

// ─── Category-grouped Grid Layout ───────────────────────────────────────────

export function categoryGridLayout(
  topology: TopologyResponse
): Map<string, { x: number; y: number }> {
  const positions = new Map<string, { x: number; y: number }>();
  const nodeW = 300;
  const nodeH = 150;
  const groupGapX = 160;

  const groups = new Map<string, TopologyNode[]>();
  for (const n of topology.nodes) {
    const cat = n.category || "custom";
    if (!groups.has(cat)) groups.set(cat, []);
    groups.get(cat)!.push(n);
  }

  const sortedCategories = Array.from(groups.entries())
    .sort((a, b) => categoryOrder(a[0]) - categoryOrder(b[0]));

  let columnX = 0;

  for (const [, nodes] of sortedCategories) {
    const subCols = Math.max(2, Math.min(8, Math.ceil(Math.sqrt(nodes.length * 1.5))));

    nodes.forEach((n, idx) => {
      const subCol = idx % subCols;
      const row = Math.floor(idx / subCols);
      positions.set(n.id, {
        x: columnX + subCol * nodeW,
        y: row * nodeH,
      });
    });

    const colWidth = subCols * nodeW;
    columnX += colWidth + groupGapX;
  }

  return positions;
}

// ─── Focus-Node Anchoring ────────────────────────────────────────────────────

function anchorOnFocusNode(
  positions: Map<string, { x: number; y: number }>,
  focusNodeId: string | undefined,
  nodeWidth: number,
  nodeHeight: number
): void {
  if (!focusNodeId || !positions.has(focusNodeId)) return;
  const fp = positions.get(focusNodeId)!;
  const offsetX = fp.x + nodeWidth / 2;
  const offsetY = fp.y + nodeHeight / 2;
  for (const [id, pos] of positions) {
    positions.set(id, { x: pos.x - offsetX, y: pos.y - offsetY });
  }
}

// ─── Hybrid Layout: ELK for components, grid for arrangement ────────────────

async function hybridLayout(
  topology: TopologyResponse,
  elkInstance: ElkLike,
  viewMode: ViewMode,
  validEdges: ValidEdge[],
  nodeWidth: number,
  nodeHeight: number,
  focusNodeId?: string
): Promise<Map<string, { x: number; y: number }>> {
  const positions = new Map<string, { x: number; y: number }>();
  const elkOptions = ELK_OPTIONS[viewMode];

  const nodeLayerMap = new Map<string, number>();
  for (const n of topology.nodes) {
    if (n.layer != null) nodeLayerMap.set(n.id, n.layer);
  }

  const nodeIds = topology.nodes.map((n) => n.id);
  const components = findConnectedComponents(nodeIds, validEdges);

  const connectedComponents = components.filter((c) => c.nodeIds.size >= 2);
  const isolatedNodeIds = new Set<string>();
  for (const c of components) {
    if (c.nodeIds.size === 1) {
      for (const nid of c.nodeIds) isolatedNodeIds.add(nid);
    }
  }

  const componentBounds: Array<{
    positions: Map<string, { x: number; y: number }>;
    width: number;
    height: number;
  }> = [];

  for (const comp of connectedComponents) {
    const compNodeIds = Array.from(comp.nodeIds);
    const compEdges = validEdges.filter((e) => comp.edgeIds.has(e.id));

    const elkGraph: ElkGraph = {
      id: `comp-${compNodeIds[0]}`,
      layoutOptions: {
        ...elkOptions,
        "elk.randomSeed": "42",
      },
      children: compNodeIds.map((nid) => ({
        id: nid,
        width: nodeWidth,
        height: nodeHeight,
        layoutOptions: getLayerConstraint(nodeLayerMap.get(nid)),
      })),
      edges: compEdges.map((e) => ({
        id: e.id,
        sources: [e.source],
        targets: [e.target],
      })),
    };

    try {
      const result = await elkInstance.layout(elkGraph);
      const compPositions = new Map<string, { x: number; y: number }>();
      let maxX = 0, maxY = 0;

      for (const child of result.children ?? []) {
        compPositions.set(child.id, { x: child.x, y: child.y });
        maxX = Math.max(maxX, child.x + nodeWidth);
        maxY = Math.max(maxY, child.y + nodeHeight);
      }

      componentBounds.push({
        positions: compPositions,
        width: maxX,
        height: maxY,
      });
    } catch {
      const compPositions = new Map<string, { x: number; y: number }>();
      const cols = Math.max(2, Math.ceil(Math.sqrt(compNodeIds.length)));
      compNodeIds.forEach((nid, idx) => {
        compPositions.set(nid, {
          x: (idx % cols) * 300,
          y: Math.floor(idx / cols) * 150,
        });
      });
      componentBounds.push({
        positions: compPositions,
        width: cols * 300,
        height: Math.ceil(compNodeIds.length / cols) * 150,
      });
    }
  }

  componentBounds.sort((a, b) => b.positions.size - a.positions.size);

  const MAX_ROW_WIDTH = Math.max(3000, Math.sqrt(topology.nodes.length) * 400);
  const COMPONENT_GAP = 120;

  let currentX = 0;
  let currentY = 0;
  let rowMaxHeight = 0;

  for (const comp of componentBounds) {
    if (currentX > 0 && currentX + comp.width > MAX_ROW_WIDTH) {
      currentX = 0;
      currentY += rowMaxHeight + COMPONENT_GAP;
      rowMaxHeight = 0;
    }

    for (const [nid, pos] of comp.positions) {
      positions.set(nid, {
        x: currentX + pos.x,
        y: currentY + pos.y,
      });
    }

    currentX += comp.width + COMPONENT_GAP;
    rowMaxHeight = Math.max(rowMaxHeight, comp.height);
  }

  if (isolatedNodeIds.size > 0) {
    const isolatedY = componentBounds.length > 0
      ? currentY + rowMaxHeight + COMPONENT_GAP * 2
      : 0;

    const nodeMap = new Map<string, TopologyNode>();
    for (const n of topology.nodes) nodeMap.set(n.id, n);

    const catGroups = new Map<string, string[]>();
    for (const nid of isolatedNodeIds) {
      const cat = nodeMap.get(nid)?.category || "custom";
      if (!catGroups.has(cat)) catGroups.set(cat, []);
      catGroups.get(cat)!.push(nid);
    }

    const sortedCats = Array.from(catGroups.entries())
      .sort((a, b) => categoryOrder(a[0]) - categoryOrder(b[0]));

    const totalIsolated = isolatedNodeIds.size;
    const targetCols = Math.max(4, Math.min(12, Math.ceil(Math.sqrt(totalIsolated * 2))));
    const isoX = 0;
    let isoY = isolatedY;

    for (const [, nids] of sortedCats) {
      const cols = Math.max(2, Math.min(targetCols, Math.ceil(Math.sqrt(nids.length * 2.5))));
      nids.forEach((nid, idx) => {
        positions.set(nid, {
          x: isoX + (idx % cols) * 300,
          y: isoY + Math.floor(idx / cols) * 150,
        });
      });
      const rows = Math.ceil(nids.length / cols);
      isoY += rows * 150 + 60;
    }
  }

  anchorOnFocusNode(positions, focusNodeId, nodeWidth, nodeHeight);

  return positions;
}

// ─── Top-level algorithm selection ───────────────────────────────────────────
// STRATEGY (smart algorithm selection):
// 1. LARGE GRAPH (300+ nodes): category grid — ELK is too slow/poor here.
// 2. DENSE GRAPH (density >= 2.0): full ELK layered.
// 3. HYBRID (default): ELK per connected component + grid arrangement.
// 4. No ELK instance available: category grid fallback.

export async function computeGraphPositions(
  topology: TopologyResponse,
  viewMode: ViewMode,
  elkInstance: ElkLike | null,
  validEdges: ValidEdge[],
  nodeWidth: number,
  nodeHeight: number,
  focusNodeId?: string
): Promise<Map<string, { x: number; y: number }>> {
  const nodeCount = topology.nodes.length;
  const density = validEdges.length / Math.max(1, nodeCount);

  if (nodeCount > 300) {
    return categoryGridLayout(topology);
  }

  if (elkInstance && density >= 2.0) {
    const elkOptions = ELK_OPTIONS[viewMode];
    const elkGraph: ElkGraph = {
      id: "root",
      layoutOptions: {
        ...elkOptions,
        "elk.randomSeed": "42",
        "elk.aspectRatio": String(Math.max(1.2, Math.min(2.5, nodeCount / 20))),
      },
      children: topology.nodes.map((n) => ({
        id: n.id,
        width: nodeWidth,
        height: nodeHeight,
      })),
      edges: validEdges.map((e) => ({
        id: e.id,
        sources: [e.source],
        targets: [e.target],
      })),
    };

    const result = await elkInstance.layout(elkGraph);
    const positions = new Map<string, { x: number; y: number }>();
    for (const child of result.children ?? []) {
      positions.set(child.id, { x: child.x, y: child.y });
    }
    anchorOnFocusNode(positions, focusNodeId, nodeWidth, nodeHeight);
    return positions;
  }

  if (elkInstance) {
    return hybridLayout(topology, elkInstance, viewMode, validEdges, nodeWidth, nodeHeight, focusNodeId);
  }

  return categoryGridLayout(topology);
}
