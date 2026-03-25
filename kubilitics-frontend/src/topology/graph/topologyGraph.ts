/**
 * Graphology data model layer for Kubernetes topology.
 *
 * Converts the backend TopologyResponse into a typed Graphology directed graph
 * suitable for ELK layout + Sigma.js rendering.
 */

import Graph from "graphology";
import type { TopologyNode, TopologyEdge, TopologyResponse } from "../types/topology";

// ---------------------------------------------------------------------------
// Attribute interfaces stored inside the Graphology graph
// ---------------------------------------------------------------------------

/** Per-node attributes stored in the Graphology graph. */
export interface K8sNodeAttributes {
  kind: string;
  name: string;
  namespace: string;
  status: string;
  category: string;
  label: string;
  /** Layer hint from the backend (used by ELK for rank ordering). */
  layer: number;
  /** Optional group id for namespace / logical grouping. */
  group?: string;
  /** Layout positions — written by the ELK layout pass. */
  x: number;
  y: number;
  /** Sigma node size (pixels). */
  size: number;
  /** Sigma node color (hex). */
  color: string;
  /** If true this node represents a collapsed PodGroup summary. */
  isPodGroup: boolean;
  /** Number of pods represented by a PodGroup node. */
  podGroupCount?: number;
  /** Extra operational data from the backend (metrics, IPs, etc.). */
  extra?: Record<string, unknown>;
}

/** Per-edge attributes stored in the Graphology graph. */
export interface K8sEdgeAttributes {
  relationshipType: string;
  relationshipCategory: string;
  label: string;
  color: string;
  size: number;
  /** Whether the edge represents a healthy relationship. */
  healthy: boolean;
  /** Visual style hint from the backend (e.g. "solid", "dashed"). */
  style: string;
  /** Whether the edge should be animated in the renderer. */
  animated: boolean;
}

// ---------------------------------------------------------------------------
// Color scheme — K8s official palette
// ---------------------------------------------------------------------------

const KIND_COLORS: Record<string, string> = {
  Pod: "#326CE5",
  PodGroup: "#326CE5",
  Deployment: "#1A73E8",
  ReplicaSet: "#1A73E8",
  StatefulSet: "#1A73E8",
  DaemonSet: "#1A73E8",
  Job: "#1A73E8",
  CronJob: "#1A73E8",
  Service: "#4CAF50",
  Ingress: "#4CAF50",
  NetworkPolicy: "#4CAF50",
  ConfigMap: "#FF9800",
  Secret: "#FF9800",
  Node: "#9C27B0",
  Namespace: "#607D8B",
  ServiceAccount: "#F44336",
  Role: "#F44336",
  RoleBinding: "#F44336",
  ClusterRole: "#F44336",
  ClusterRoleBinding: "#F44336",
  PersistentVolumeClaim: "#795548",
  PersistentVolume: "#795548",
  StorageClass: "#795548",
};

const DEFAULT_COLOR = "#78909C";

const EDGE_CATEGORY_COLORS: Record<string, string> = {
  ownership: "#1A73E8",
  networking: "#4CAF50",
  configuration: "#FF9800",
  storage: "#795548",
  rbac: "#F44336",
  scheduling: "#9C27B0",
};

const DEFAULT_EDGE_COLOR = "#90A4AE";

// ---------------------------------------------------------------------------
// Sizing helpers
// ---------------------------------------------------------------------------

/** Larger nodes for higher-level / more important resource kinds. */
const KIND_SIZE: Record<string, number> = {
  Namespace: 14,
  Node: 14,
  Deployment: 12,
  StatefulSet: 12,
  DaemonSet: 12,
  Service: 11,
  Ingress: 11,
  ReplicaSet: 9,
  Job: 9,
  CronJob: 9,
  Pod: 7,
  PodGroup: 10,
  ConfigMap: 6,
  Secret: 6,
  PersistentVolumeClaim: 8,
  PersistentVolume: 8,
  StorageClass: 8,
  ServiceAccount: 7,
  Role: 7,
  RoleBinding: 6,
  ClusterRole: 8,
  ClusterRoleBinding: 7,
};

const DEFAULT_SIZE = 7;

function nodeSize(kind: string): number {
  return KIND_SIZE[kind] ?? DEFAULT_SIZE;
}

function nodeColor(kind: string): string {
  return KIND_COLORS[kind] ?? DEFAULT_COLOR;
}

function edgeColor(category: string): string {
  return EDGE_CATEGORY_COLORS[category] ?? DEFAULT_EDGE_COLOR;
}

// ---------------------------------------------------------------------------
// Conversion helpers
// ---------------------------------------------------------------------------

function isPodGroup(node: TopologyNode): boolean {
  return node.kind === "PodGroup" || (node.group?.startsWith("podgroup:") ?? false);
}

/** Pack extra operational fields from the API node into a bag. */
function extractExtra(node: TopologyNode): Record<string, unknown> {
  const extra: Record<string, unknown> = {};
  if (node.metrics) extra.metrics = node.metrics;
  if (node.cost) extra.cost = node.cost;
  if (node.labels) extra.labels = node.labels;
  if (node.annotations) extra.annotations = node.annotations;
  if (node.createdAt) extra.createdAt = node.createdAt;
  if (node.podIP) extra.podIP = node.podIP;
  if (node.nodeName) extra.nodeName = node.nodeName;
  if (node.internalIP) extra.internalIP = node.internalIP;
  if (node.externalIP) extra.externalIP = node.externalIP;
  if (node.clusterIP) extra.clusterIP = node.clusterIP;
  if (node.serviceType) extra.serviceType = node.serviceType;
  if (node.containers != null) extra.containers = node.containers;
  if (node.statusReason) extra.statusReason = node.statusReason;
  if (node.apiVersion) extra.apiVersion = node.apiVersion;
  return extra;
}

// ---------------------------------------------------------------------------
// Build function
// ---------------------------------------------------------------------------

/**
 * Convert a backend TopologyResponse into a Graphology directed graph.
 *
 * - Nodes get color/size attributes based on their `kind`.
 * - PodGroup summary nodes are flagged so the renderer can draw them differently.
 * - Duplicate edges (same source + target + relationshipType) are skipped.
 * - Positions (x, y) default to 0 — the ELK layout pass will set them.
 */
export function buildTopologyGraph(
  apiResponse: TopologyResponse,
): Graph<K8sNodeAttributes, K8sEdgeAttributes> {
  const graph = new Graph<K8sNodeAttributes, K8sEdgeAttributes>({
    multi: false,
    type: "directed",
  });

  // ---- Nodes ----
  const nodeIds = new Set<string>();

  for (const node of apiResponse.nodes) {
    if (nodeIds.has(node.id)) continue; // guard against duplicate ids
    nodeIds.add(node.id);

    const podGroup = isPodGroup(node);
    const podGroupCount = podGroup ? (node.metrics?.podCount ?? 1) : undefined;

    const attrs: K8sNodeAttributes = {
      kind: node.kind,
      name: node.name,
      namespace: node.namespace,
      status: node.status,
      category: node.category,
      label: node.label,
      layer: node.layer,
      group: node.group,
      x: 0,
      y: 0,
      size: podGroup ? Math.min(nodeSize("PodGroup") + (podGroupCount ?? 0), 18) : nodeSize(node.kind),
      color: nodeColor(node.kind),
      isPodGroup: podGroup,
      podGroupCount,
      extra: extractExtra(node),
    };

    graph.addNode(node.id, attrs);
  }

  // ---- Edges ----
  const edgeKeys = new Set<string>();

  for (const edge of apiResponse.edges) {
    // Both endpoints must exist in the graph
    if (!graph.hasNode(edge.source) || !graph.hasNode(edge.target)) continue;

    // Deduplicate: same source + target + relationship type → skip
    const dedupeKey = `${edge.source}|${edge.target}|${edge.relationshipType}`;
    if (edgeKeys.has(dedupeKey)) continue;
    edgeKeys.add(dedupeKey);

    const attrs: K8sEdgeAttributes = {
      relationshipType: edge.relationshipType,
      relationshipCategory: edge.relationshipCategory,
      label: edge.label,
      color: edgeColor(edge.relationshipCategory),
      size: edge.healthy ? 1.5 : 2,
      healthy: edge.healthy,
      style: edge.style,
      animated: edge.animated ?? false,
    };

    // Use the backend-provided id if available, otherwise let graphology generate one.
    try {
      graph.addEdgeWithKey(edge.id, edge.source, edge.target, attrs);
    } catch {
      // Edge key collision (shouldn't happen after dedup, but be safe)
      graph.addEdge(edge.source, edge.target, attrs);
    }
  }

  return graph;
}

// ---------------------------------------------------------------------------
// Utility: update a single node's attributes after a WebSocket delta
// ---------------------------------------------------------------------------

export function patchNodeAttributes(
  graph: Graph<K8sNodeAttributes, K8sEdgeAttributes>,
  updated: TopologyNode,
): void {
  if (!graph.hasNode(updated.id)) return;

  graph.mergeNodeAttributes(updated.id, {
    status: updated.status,
    label: updated.label,
    extra: extractExtra(updated),
  });
}

// ---------------------------------------------------------------------------
// Utility: re-export color helpers for use in legends / tooltips
// ---------------------------------------------------------------------------

export { nodeColor as colorForKind, edgeColor as colorForEdgeCategory };
export { KIND_COLORS, EDGE_CATEGORY_COLORS };
