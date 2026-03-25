/**
 * SigmaTopologyCanvas — WebGL-powered topology renderer using Sigma.js v3
 *
 * Replaces React Flow for large graphs (1000+ nodes). Sigma renders nodes and
 * edges via WebGL while this React wrapper manages the lifecycle, events, and
 * visual state (hover highlights, selection, K8s-themed colors).
 *
 * The incoming Graphology `graph` must have x/y positions already set (e.g. by
 * ELK layout). Node attributes expected:
 *   { x, y, kind, name, category, status, label }
 * Edge attributes expected:
 *   { relationshipCategory, relationshipType }
 */

import { useCallback, useEffect, useRef, useState } from "react";
import Graph from "graphology";
import type { Attributes } from "graphology-types";
import {
  getCategoryColor,
  getEdgeColor,
  mapStatusKey,
  STATUS_COLORS,
} from "../constants/designTokens";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface SigmaTopologyCanvasProps {
  /** Graphology graph with x/y positions already set by ELK */
  graph: Graph;
  /** Fired when the user clicks a node */
  onNodeClick?: (nodeId: string) => void;
  /** Fired when the user hovers a node (null = left all nodes) */
  onNodeHover?: (nodeId: string | null) => void;
  /** Currently selected node — highlighted with its neighbors */
  selectedNode?: string | null;
  /** Extra CSS class on the container div */
  className?: string;
}

/** Sigma node display attributes we write into the graph */
interface NodeDisplayAttrs extends Attributes {
  x: number;
  y: number;
  size: number;
  color: string;
  label: string;
  type: string;
  highlighted?: boolean;
  hidden?: boolean;
  zIndex?: number;
}

/** Sigma edge display attributes */
interface EdgeDisplayAttrs extends Attributes {
  size: number;
  color: string;
  type: string;
  hidden?: boolean;
}

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

/** Node size range — mapped from connection count */
const MIN_NODE_SIZE = 4;
const MAX_NODE_SIZE = 18;
const DEFAULT_NODE_SIZE = 6;

/** Dimmed opacity for nodes/edges not in the highlight set */
const DIM_COLOR = "rgba(150, 150, 150, 0.15)";

/** Highlight ring color */
const SELECTED_RING_COLOR = "#2563EB";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/** Compute node size based on degree (edges connected) */
function sizeFromDegree(degree: number, maxDegree: number): number {
  if (maxDegree <= 1) return DEFAULT_NODE_SIZE;
  const ratio = degree / maxDegree;
  return MIN_NODE_SIZE + ratio * (MAX_NODE_SIZE - MIN_NODE_SIZE);
}

/** Map a K8s kind string to a category accent color */
function colorForKind(category: string | undefined, status: string | undefined): string {
  // If the resource is in error/warning, tint the node accordingly
  if (status) {
    const sKey = mapStatusKey(status);
    if (sKey === "error") return STATUS_COLORS.error;
    if (sKey === "warning") return STATUS_COLORS.warning;
  }
  return getCategoryColor(category ?? "custom").accent;
}

/** Map edge relationship category to a color hex */
function colorForEdge(relationshipCategory: string | undefined): string {
  return getEdgeColor(relationshipCategory);
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function SigmaTopologyCanvas({
  graph,
  onNodeClick,
  onNodeHover,
  selectedNode,
  className,
}: SigmaTopologyCanvasProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  // Keep a stable ref to the Sigma instance so we can interact with it
  // without re-mounting.
  const rendererRef = useRef<InstanceType<typeof import("sigma").Sigma> | null>(null);
  const [hoveredNode, setHoveredNode] = useState<string | null>(null);

  // ------- Prepare graph display attributes --------------------------------
  // We clone + mutate node/edge attrs for Sigma display. We do this outside
  // the effect so the graph reference itself can be used as the dep key.
  const prepareGraph = useCallback(
    (g: Graph) => {
      // Find max degree for sizing
      let maxDegree = 1;
      g.forEachNode((node) => {
        const deg = g.degree(node);
        if (deg > maxDegree) maxDegree = deg;
      });

      g.forEachNode((node, attrs) => {
        const degree = g.degree(node);
        g.mergeNodeAttributes(node, {
          size: sizeFromDegree(degree, maxDegree),
          color: colorForKind(attrs.category as string | undefined, attrs.status as string | undefined),
          label: (attrs.name as string) || (attrs.label as string) || node,
          type: "circle",
        } satisfies Partial<NodeDisplayAttrs>);
      });

      g.forEachEdge((_edge, attrs) => {
        const relCat = attrs.relationshipCategory as string | undefined;
        g.mergeEdgeAttributes(_edge, {
          size: 1,
          color: colorForEdge(relCat),
          type: "line",
        } satisfies Partial<EdgeDisplayAttrs>);
      });
    },
    [],
  );

  // ------- Mount / unmount Sigma renderer ----------------------------------
  useEffect(() => {
    const container = containerRef.current;
    if (!container || !graph || graph.order === 0) return;

    let renderer: InstanceType<typeof import("sigma").Sigma> | null = null;
    let cancelled = false;

    // Dynamic import avoids SSR/Node issues with WebGL
    (async () => {
      const { Sigma } = await import("sigma");
      if (cancelled) return;

      // Prepare display attributes on the graph
      prepareGraph(graph);

      renderer = new Sigma(graph, container, {
        // Performance
        renderEdgeLabels: false,
        labelRenderedSizeThreshold: 8,

        // Camera bounds
        minCameraRatio: 0.01,
        maxCameraRatio: 10,

        // Defaults
        defaultNodeColor: "#6366F1",
        defaultEdgeColor: "#94a3b8",
        defaultNodeType: "circle",
        defaultEdgeType: "line",

        // Labels
        labelFont: "Inter, system-ui, sans-serif",
        labelSize: 12,
        labelColor: { color: "#1e293b" },

        // Interaction
        zIndex: true,
        stagePadding: 40,
        labelDensity: 0.5,
        labelGridCellSize: 120,

        // Node / edge reducers for hover + selection highlighting
        nodeReducer: (node, data) => {
          const res = { ...data };
          const active = hoveredNodeRef.current ?? selectedNodeRef.current;
          if (active) {
            if (node === active || neighborSetRef.current.has(node)) {
              res.highlighted = true;
              res.zIndex = 1;
            } else {
              res.color = DIM_COLOR;
              res.highlighted = false;
              res.zIndex = 0;
              res.label = "";
            }
          }
          // Selected node gets a special color boost
          if (node === selectedNodeRef.current) {
            res.color = SELECTED_RING_COLOR;
            res.highlighted = true;
            res.zIndex = 2;
          }
          return res;
        },
        edgeReducer: (edge, data) => {
          const res = { ...data };
          const active = hoveredNodeRef.current ?? selectedNodeRef.current;
          if (active) {
            const src = graph.source(edge);
            const tgt = graph.target(edge);
            if (src === active || tgt === active) {
              res.hidden = false;
              res.zIndex = 1;
            } else {
              res.color = DIM_COLOR;
              res.hidden = false;
              res.zIndex = 0;
            }
          }
          return res;
        },
      });

      rendererRef.current = renderer;

      // ---- Events ----------------------------------------------------------

      renderer.on("clickNode", ({ node }) => {
        onNodeClick?.(node);
      });

      renderer.on("enterNode", ({ node }) => {
        setHoveredNode(node);
        onNodeHover?.(node);
        container.style.cursor = "pointer";
        renderer?.refresh();
      });

      renderer.on("leaveNode", () => {
        setHoveredNode(null);
        onNodeHover?.(null);
        container.style.cursor = "default";
        renderer?.refresh();
      });

      // ---- Camera: fit to view on mount ------------------------------------
      const camera = renderer.getCamera();
      camera.animatedReset({ duration: 300 });
    })();

    return () => {
      cancelled = true;
      if (renderer) {
        renderer.kill();
      }
      rendererRef.current = null;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [graph]);

  // ------- Stable refs for reducer closures --------------------------------
  // The reducers are set once during Sigma construction, so they need refs
  // to read the latest hovered/selected state.
  const hoveredNodeRef = useRef<string | null>(null);
  const selectedNodeRef = useRef<string | null>(null);
  const neighborSetRef = useRef<Set<string>>(new Set());

  // Keep refs in sync
  useEffect(() => {
    hoveredNodeRef.current = hoveredNode;
    updateNeighborSet();
  }, [hoveredNode]);

  useEffect(() => {
    selectedNodeRef.current = selectedNode ?? null;
    updateNeighborSet();
  }, [selectedNode]);

  /** Recompute the neighbor set for the active (hovered or selected) node */
  const updateNeighborSet = useCallback(() => {
    const active = hoveredNodeRef.current ?? selectedNodeRef.current;
    if (!active || !graph || !graph.hasNode(active)) {
      neighborSetRef.current = new Set();
    } else {
      neighborSetRef.current = new Set(graph.neighbors(active));
    }
    // Trigger Sigma refresh to apply reducer changes
    rendererRef.current?.refresh();
  }, [graph]);

  return (
    <div
      ref={containerRef}
      className={className}
      style={{
        width: "100%",
        height: "100%",
        position: "relative",
        overflow: "hidden",
      }}
      data-testid="sigma-topology-canvas"
    />
  );
}

export default SigmaTopologyCanvas;
