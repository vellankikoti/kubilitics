/**
 * BlastRadiusTab — Dedicated impact analysis view for resource detail pages.
 *
 * Fetches blast radius data from the backend API via useBlastRadius.
 * Falls back to topology-derived criticality data when the API is unavailable.
 * Shows criticality scores, fan-in/fan-out, SPOF detection, dependency chain,
 * affected resources with links, and a failure simulation mode.
 */
import { useState, useMemo, useCallback, useRef } from "react";
import { Link } from "react-router-dom";
import {
  Zap,
  AlertTriangle,
  Shield,
  ArrowDownRight,
  ArrowUpRight,
  ExternalLink,
  ChevronRight,
} from "lucide-react";

import { useBlastRadius } from "@/hooks/useBlastRadius";
import { useResourceTopology } from "@/hooks/useResourceTopology";
import { useActiveClusterId } from "@/hooks/useActiveClusterId";
import { kindToRoutePath } from "@/utils/resourceKindMapper";
import { TopologyCanvas } from "@/topology/TopologyCanvas";
import { transformGraph } from "@/topology/utils/transformGraph";
import type { TopologyResponse, TopologyNode } from "@/topology/types/topology";
import type { ExportFormat } from "@/topology/TopologyCanvas";

export interface BlastRadiusTabProps {
  kind: string;
  namespace?: string;
  name?: string;
}

/** Criticality level to Tailwind color class mapping */
const LEVEL_COLORS: Record<string, string> = {
  critical: "bg-red-600",
  high: "bg-amber-500",
  medium: "bg-yellow-500",
  low: "bg-emerald-500",
};

const LEVEL_TEXT_COLORS: Record<string, string> = {
  critical: "text-red-600",
  high: "text-amber-500",
  medium: "text-yellow-500",
  low: "text-emerald-500",
};

/** Build a detail page path for a resource. */
function buildResourceDetailPath(resourceKind: string, resourceName: string, resourceNamespace?: string): string {
  const route = kindToRoutePath(resourceKind);
  if (resourceNamespace) {
    return `/${route}/${resourceNamespace}/${resourceName}`;
  }
  return `/${route}/${resourceName}`;
}

export function BlastRadiusTab({ kind, namespace, name }: BlastRadiusTabProps) {
  const clusterId = useActiveClusterId();

  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null);
  const [simulatedFailureNodeId, setSimulatedFailureNodeId] = useState<string | null>(null);

  const fitViewRef = useRef<(() => void) | null>(null);
  const exportRef = useRef<((format: ExportFormat, filename: string) => void) | null>(null);
  const centerOnNodeRef = useRef<((nodeId: string) => void) | null>(null);

  const canFetch = !!clusterId && !!kind && !!name;

  // ── Backend blast radius API ───────────────────────────────────────────
  const {
    data: blastRadiusData,
    isLoading: brLoading,
    error: brError,
    isUnavailable: brUnavailable,
  } = useBlastRadius({ kind, namespace, name, enabled: canFetch });

  // ── Topology graph (always fetched for the canvas + fallback data) ─────
  const { graph, isLoading: topoLoading, error: topoError } = useResourceTopology({
    kind,
    namespace,
    name,
    enabled: canFetch,
    depth: 3,
  });

  const isLoading = brLoading || topoLoading;
  const error = brError || topoError;

  // Transform engine graph to v2 format
  const topology = useMemo<TopologyResponse | null>(() => {
    if (!graph) return null;
    const response = transformGraph(graph, clusterId ?? undefined);
    response.metadata.mode = "resource";
    if (namespace) response.metadata.namespace = namespace;
    const focusId = response.nodes.find(
      (n) => n.kind === kind && n.name === name && (!namespace || n.namespace === namespace)
    )?.id;
    if (focusId) response.metadata.focusResource = focusId;
    return response;
  }, [graph, clusterId, namespace, kind, name]);

  // Find the focus node
  const focusNode = useMemo<TopologyNode | null>(() => {
    if (!topology) return null;
    return (
      topology.nodes.find(
        (n) => n.kind === kind && n.name === name && (!namespace || n.namespace === namespace)
      ) ?? null
    );
  }, [topology, kind, name, namespace]);

  // ── Resolve display values: prefer API data, fall back to topology criticality ──
  const useApi = !!blastRadiusData && !brUnavailable;
  const topoCriticality = focusNode?.criticality ?? null;

  const criticalityScore = useApi
    ? blastRadiusData!.criticalityScore
    : (topoCriticality?.score ?? 0);
  const level = useApi
    ? blastRadiusData!.level
    : (topoCriticality?.level ?? "low");
  const blastRadius = useApi
    ? blastRadiusData!.blastRadiusPercent
    : (topoCriticality?.blastRadius ?? 0);
  const fanIn = useApi
    ? blastRadiusData!.fanIn
    : (topoCriticality?.fanIn ?? 0);
  const fanOut = useApi
    ? blastRadiusData!.fanOut
    : (topoCriticality?.fanOut ?? 0);
  const isSPOF = useApi
    ? blastRadiusData!.isSPOF
    : (topoCriticality?.isSPOF ?? false);
  const affectedResources = useApi ? blastRadiusData!.affectedResources : [];
  const dependencyChain = useApi ? blastRadiusData!.dependencyChain : [];

  // Highlight the focus node
  const highlightNodeIds = useMemo(() => {
    return focusNode ? [focusNode.id] : [];
  }, [focusNode]);

  // BFS to compute affected nodes when simulating failure.
  const simulationAffectedNodes = useMemo(() => {
    if (!simulatedFailureNodeId || !topology) return null;
    const adj = new Map<string, Set<string>>();
    for (const edge of topology.edges) {
      if (!adj.has(edge.source)) adj.set(edge.source, new Set());
      if (!adj.has(edge.target)) adj.set(edge.target, new Set());
      adj.get(edge.source)!.add(edge.target);
      adj.get(edge.target)!.add(edge.source);
    }
    const affected = new Set<string>();
    affected.add(simulatedFailureNodeId);
    const queue = [simulatedFailureNodeId];
    while (queue.length > 0) {
      const current = queue.shift()!;
      for (const neighbor of adj.get(current) ?? []) {
        if (!affected.has(neighbor)) {
          affected.add(neighbor);
          queue.push(neighbor);
        }
      }
    }
    return affected;
  }, [simulatedFailureNodeId, topology]);

  const isSimulating = simulationAffectedNodes != null && simulationAffectedNodes.size > 0;
  const impactedCount = isSimulating ? simulationAffectedNodes!.size - 1 : 0;

  // Group affected nodes by kind for the simulation list
  const affectedByKind = useMemo(() => {
    if (!isSimulating || !topology) return new Map<string, TopologyNode[]>();
    const groups = new Map<string, TopologyNode[]>();
    for (const node of topology.nodes) {
      if (simulationAffectedNodes!.has(node.id) && node.id !== simulatedFailureNodeId) {
        const list = groups.get(node.kind) ?? [];
        list.push(node);
        groups.set(node.kind, list);
      }
    }
    return groups;
  }, [isSimulating, topology, simulationAffectedNodes, simulatedFailureNodeId]);

  const handleSimulateFailure = useCallback(() => {
    if (focusNode) {
      setSimulatedFailureNodeId(focusNode.id);
    }
  }, [focusNode]);

  const handleClearSimulation = useCallback(() => {
    setSimulatedFailureNodeId(null);
  }, []);

  // --- Render ---

  if (isLoading) {
    return (
      <div className="flex flex-col items-center justify-center min-h-[400px]">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-gray-200 border-t-blue-500 mb-4" />
        <p className="text-sm text-muted-foreground">Analyzing blast radius...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex flex-col items-center justify-center min-h-[400px] text-center p-6">
        <AlertTriangle className="h-12 w-12 text-destructive mb-4" />
        <p className="text-destructive font-medium mb-2">Failed to load blast radius data</p>
        <p className="text-sm text-muted-foreground">{error.message || String(error)}</p>
      </div>
    );
  }

  if (!topology || topology.nodes.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center min-h-[400px] text-center p-6">
        <Shield className="h-12 w-12 text-muted-foreground mb-4" />
        <p className="text-muted-foreground font-medium mb-2">No topology data available</p>
        <p className="text-sm text-muted-foreground">
          Cannot compute blast radius without connected resources.
        </p>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-4 p-4 w-full">
      {/* Score Cards Row */}
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
        {/* Criticality Score */}
        <div className="rounded-lg border border-gray-200 dark:border-slate-700 bg-white dark:bg-slate-900 p-4">
          <p className="text-xs font-medium text-gray-500 dark:text-slate-400 uppercase tracking-wider mb-1">
            Criticality Score
          </p>
          <div className="flex items-baseline gap-2">
            <p
              className={`text-2xl font-bold ${
                level === "critical"
                  ? "text-red-600"
                  : level === "high"
                  ? "text-amber-500"
                  : "text-gray-900 dark:text-gray-100"
              }`}
            >
              {criticalityScore}
            </p>
            <span className="text-xs text-gray-400">/100</span>
          </div>
        </div>

        {/* Blast Radius Percentage */}
        <div className="rounded-lg border border-gray-200 dark:border-slate-700 bg-white dark:bg-slate-900 p-4">
          <p className="text-xs font-medium text-gray-500 dark:text-slate-400 uppercase tracking-wider mb-1">
            Blast Radius
          </p>
          <div className="flex items-baseline gap-2">
            <p
              className={`text-2xl font-bold ${
                level === "critical" ? "text-red-600" : "text-gray-900 dark:text-gray-100"
              }`}
            >
              {blastRadius}%
            </p>
            <span
              className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-semibold text-white ${
                LEVEL_COLORS[level] ?? LEVEL_COLORS.low
              }`}
            >
              {level.charAt(0).toUpperCase() + level.slice(1)}
            </span>
          </div>
        </div>

        {/* Fan-in / Fan-out */}
        <div className="rounded-lg border border-gray-200 dark:border-slate-700 bg-white dark:bg-slate-900 p-4">
          <p className="text-xs font-medium text-gray-500 dark:text-slate-400 uppercase tracking-wider mb-1">
            Fan
          </p>
          <div className="flex items-center gap-3">
            <span className="flex items-center gap-1 text-sm font-medium text-gray-700 dark:text-gray-300">
              <ArrowDownRight className="h-3.5 w-3.5 text-blue-500" />
              IN: {fanIn}
            </span>
            <span className="flex items-center gap-1 text-sm font-medium text-gray-700 dark:text-gray-300">
              <ArrowUpRight className="h-3.5 w-3.5 text-orange-500" />
              OUT: {fanOut}
            </span>
          </div>
        </div>

        {/* SPOF Indicator */}
        <div className="rounded-lg border border-gray-200 dark:border-slate-700 bg-white dark:bg-slate-900 p-4">
          <p className="text-xs font-medium text-gray-500 dark:text-slate-400 uppercase tracking-wider mb-1">
            SPOF
          </p>
          {isSPOF ? (
            <span className="flex items-center gap-1.5 text-sm font-semibold text-amber-600 dark:text-amber-400">
              <AlertTriangle className="h-4 w-4" />
              Yes
            </span>
          ) : (
            <span className="flex items-center gap-1.5 text-sm font-medium text-emerald-600 dark:text-emerald-400">
              <Shield className="h-4 w-4" />
              No
            </span>
          )}
        </div>
      </div>

      {/* Simulation Controls */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          {!isSimulating ? (
            <button
              type="button"
              onClick={handleSimulateFailure}
              className="inline-flex items-center gap-2 rounded-lg border-2 border-red-300 dark:border-red-700 px-4 py-2 text-sm font-medium text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-950 transition-colors"
            >
              <Zap className="h-4 w-4" />
              Simulate Failure
            </button>
          ) : (
            <>
              <span
                className={`text-sm font-medium ${LEVEL_TEXT_COLORS[level] ?? "text-gray-600"}`}
              >
                {impactedCount} resource{impactedCount !== 1 ? "s" : ""} impacted
              </span>
              <button
                type="button"
                onClick={handleClearSimulation}
                className="inline-flex items-center gap-2 rounded-lg border border-gray-300 dark:border-slate-600 px-4 py-2 text-sm font-medium text-gray-600 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-slate-800 transition-colors"
              >
                Clear Simulation
              </button>
            </>
          )}
        </div>
        {brUnavailable && (
          <span className="text-xs text-gray-400 dark:text-slate-500 italic">
            Using topology-derived data (blast radius API not available)
          </span>
        )}
      </div>

      {/* Topology Canvas */}
      <div className="rounded-lg border border-gray-200 dark:border-slate-700 overflow-hidden" style={{ height: "500px" }}>
        <TopologyCanvas
          topology={topology}
          selectedNodeId={selectedNodeId}
          highlightNodeIds={highlightNodeIds}
          viewMode="resource"
          onSelectNode={setSelectedNodeId}
          fitViewRef={fitViewRef}
          exportRef={exportRef}
          centerOnNodeRef={centerOnNodeRef}
          simulationAffectedNodes={simulationAffectedNodes}
          simulatedFailureNodeId={simulatedFailureNodeId}
        />
      </div>

      {/* Affected Resources from API (always visible when API data available) */}
      {useApi && affectedResources.length > 0 && !isSimulating && (
        <div className="rounded-lg border border-gray-200 dark:border-slate-700 bg-white dark:bg-slate-900 p-4">
          <h4 className="text-sm font-semibold text-gray-900 dark:text-gray-100 mb-3">
            Affected Resources ({affectedResources.length})
          </h4>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2">
            {affectedResources.map((res) => (
              <Link
                key={`${res.kind}/${res.namespace}/${res.name}`}
                to={buildResourceDetailPath(res.kind, res.name, res.namespace)}
                className="flex items-center gap-2 rounded-md border border-gray-100 dark:border-slate-700 px-3 py-2 text-sm hover:bg-gray-50 dark:hover:bg-slate-800 transition-colors group"
              >
                <span className="text-xs font-medium text-gray-400 dark:text-slate-500 uppercase w-20 shrink-0 truncate">
                  {res.kind}
                </span>
                <span className="text-gray-700 dark:text-gray-300 truncate flex-1">
                  {res.name}
                </span>
                {res.impact === "direct" && (
                  <span className="text-xs text-red-500 font-medium shrink-0">direct</span>
                )}
                <ExternalLink className="h-3 w-3 text-gray-400 opacity-0 group-hover:opacity-100 transition-opacity shrink-0" />
              </Link>
            ))}
          </div>
        </div>
      )}

      {/* Dependency Chain from API */}
      {useApi && dependencyChain.length > 0 && (
        <div className="rounded-lg border border-gray-200 dark:border-slate-700 bg-white dark:bg-slate-900 p-4">
          <h4 className="text-sm font-semibold text-gray-900 dark:text-gray-100 mb-3">
            Dependency Chain
          </h4>
          <div className="flex flex-wrap items-center gap-1">
            {dependencyChain.map((entry, idx) => (
              <span key={idx} className="flex items-center gap-1">
                <span className="text-sm text-gray-700 dark:text-gray-300 bg-gray-100 dark:bg-slate-800 rounded px-2 py-0.5">
                  {entry}
                </span>
                {idx < dependencyChain.length - 1 && (
                  <ChevronRight className="h-3.5 w-3.5 text-gray-400 dark:text-slate-500 shrink-0" />
                )}
              </span>
            ))}
          </div>
        </div>
      )}

      {/* Affected Resources List (visible during simulation) */}
      {isSimulating && affectedByKind.size > 0 && (
        <div className="rounded-lg border border-gray-200 dark:border-slate-700 bg-white dark:bg-slate-900 p-4">
          <h4 className="text-sm font-semibold text-gray-900 dark:text-gray-100 mb-3">
            Affected Resources (Simulation)
          </h4>
          <div className="grid grid-cols-3 gap-4">
            {Array.from(affectedByKind.entries()).map(([resourceKind, nodes]) => (
              <div key={resourceKind}>
                <p className="text-xs font-medium text-gray-500 dark:text-slate-400 mb-1.5">
                  {resourceKind} ({nodes.length})
                </p>
                <ul className="space-y-0.5">
                  {nodes.map((node) => (
                    <li key={node.name}>
                      <Link
                        to={buildResourceDetailPath(node.kind, node.name, node.namespace)}
                        className="text-sm text-primary hover:underline truncate block"
                      >
                        {node.name}
                      </Link>
                    </li>
                  ))}
                </ul>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
