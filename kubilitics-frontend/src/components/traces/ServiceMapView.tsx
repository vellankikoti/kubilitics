/**
 * ServiceMapView — Visual service dependency graph using @xyflow/react.
 * Nodes are services, edges are call relationships. Colors reflect error rate.
 */
import { useMemo, useCallback, useState, useEffect } from 'react';
import {
  ReactFlow,
  ReactFlowProvider,
  Background,
  BackgroundVariant,
  Handle,
  Position,
  type Node,
  type Edge,
  type NodeProps,
  type NodeTypes,
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import ELK from 'elkjs/lib/elk.bundled.js';
import { Card, CardContent } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { cn } from '@/lib/utils';
import { useServiceMap } from '@/hooks/useTraces';
import { useTracesStore } from '@/stores/tracesStore';
import type { ServiceNode as ServiceNodeData, ServiceEdge } from '@/services/api/traces';

/* ─── Helpers ──────────────────────────────────────────────────────────── */

function formatDuration(ns: number): string {
  const ms = ns / 1_000_000;
  if (ms < 1) return `${(ns / 1_000).toFixed(0)}us`;
  if (ms < 1000) return `${ms.toFixed(0)}ms`;
  return `${(ms / 1000).toFixed(1)}s`;
}

function errorRate(node: ServiceNodeData): number {
  if (node.span_count === 0) return 0;
  return node.error_count / node.span_count;
}

function nodeHealthColor(node: ServiceNodeData): {
  border: string;
  bg: string;
  text: string;
} {
  const rate = errorRate(node);
  if (rate >= 0.2) return { border: 'border-destructive', bg: 'bg-destructive/10', text: 'text-destructive' };
  if (rate >= 0.05) return { border: 'border-amber-500', bg: 'bg-amber-500/10', text: 'text-amber-600 dark:text-amber-400' };
  return { border: 'border-[hsl(var(--success))]', bg: 'bg-[hsl(var(--success))]/10', text: 'text-[hsl(var(--success))]' };
}

/* ─── Custom Node ──────────────────────────────────────────────────────── */

interface ServiceFlowNodeData {
  label: string;
  spanCount: number;
  errorCount: number;
  avgDurationNs: number;
  health: ReturnType<typeof nodeHealthColor>;
  [key: string]: unknown;
}

function ServiceFlowNode({ data }: NodeProps<Node<ServiceFlowNodeData>>) {
  return (
    <div
      className={cn(
        'rounded-xl border-2 px-4 py-3 bg-card shadow-md min-w-[140px] transition-shadow hover:shadow-lg',
        data.health.border,
      )}
    >
      <Handle type="target" position={Position.Top} className="!bg-muted-foreground !w-2 !h-2" />
      <div className="text-sm font-semibold truncate mb-1">{data.label}</div>
      <div className="flex items-center gap-3 text-[11px] text-muted-foreground">
        <span>{data.spanCount} spans</span>
        {data.errorCount > 0 && (
          <span className="text-destructive font-medium">{data.errorCount} err</span>
        )}
      </div>
      <div className="text-[11px] text-muted-foreground mt-0.5">
        avg {formatDuration(data.avgDurationNs)}
      </div>
      <Handle type="source" position={Position.Bottom} className="!bg-muted-foreground !w-2 !h-2" />
    </div>
  );
}

const nodeTypes: NodeTypes = {
  service: ServiceFlowNode,
};

/* ─── ELK Layout ───────────────────────────────────────────────────────── */

const elk = new ELK();

async function layoutGraph(
  nodes: Node<ServiceFlowNodeData>[],
  edges: Edge[],
): Promise<Node<ServiceFlowNodeData>[]> {
  const graph = {
    id: 'root',
    layoutOptions: {
      'elk.algorithm': 'layered',
      'elk.direction': 'DOWN',
      'elk.spacing.nodeNode': '60',
      'elk.layered.spacing.nodeNodeBetweenLayers': '80',
    },
    children: nodes.map((n) => ({
      id: n.id,
      width: 160,
      height: 80,
    })),
    edges: edges.map((e) => ({
      id: e.id,
      sources: [e.source],
      targets: [e.target],
    })),
  };

  const layouted = await elk.layout(graph);

  return nodes.map((node) => {
    const layoutNode = layouted.children?.find((n) => n.id === node.id);
    return {
      ...node,
      position: {
        x: layoutNode?.x ?? 0,
        y: layoutNode?.y ?? 0,
      },
    };
  });
}

/* ─── Component ────────────────────────────────────────────────────────── */

function ServiceMapInner() {
  const store = useTracesStore();

  // Direct fetch — same pattern as TraceList
  const [serviceMap, setServiceMap] = useState<any>(null);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const base = 'http://localhost:8190';
        const cl = await (await fetch(`${base}/api/v1/clusters`)).json();
        const c = cl.find((x: any) => x.status === 'connected');
        if (!c) { setIsLoading(false); return; }
        const res = await fetch(`${base}/api/v1/clusters/${c.id}/traces/services`);
        const data = await res.json();
        // Validate response has nodes array — API might return {error: ...}
        if (!cancelled) {
          if (data && Array.isArray(data.nodes)) {
            setServiceMap(data);
          } else {
            // Fallback: build service map from traces list
            const tracesRes = await fetch(`${base}/api/v1/clusters/${c.id}/traces?limit=100`);
            const traces = await tracesRes.json();
            if (Array.isArray(traces) && traces.length > 0) {
              const svcSet = new Map<string, { count: number; errors: number }>();
              for (const t of traces) {
                const svcs = Array.isArray(t.services) ? t.services : (typeof t.services === 'string' ? JSON.parse(t.services || '[]') : []);
                for (const s of svcs) {
                  const prev = svcSet.get(s) || { count: 0, errors: 0 };
                  svcSet.set(s, { count: prev.count + (t.span_count || 1), errors: prev.errors + (t.error_count || 0) });
                }
              }
              setServiceMap({
                nodes: [...svcSet.entries()].map(([name, v]) => ({ name, span_count: v.count, error_count: v.errors, avg_duration_ns: 0 })),
                edges: [],
              });
            }
          }
          setIsLoading(false);
        }
      } catch { if (!cancelled) setIsLoading(false); }
    })();
    return () => { cancelled = true; };
  }, []);

  const { initialNodes, initialEdges } = useMemo(() => {
    if (!serviceMap || !serviceMap.nodes) return { initialNodes: [], initialEdges: [] };

    const nodes: Node<ServiceFlowNodeData>[] = serviceMap.nodes.map((svc) => ({
      id: svc.name,
      type: 'service',
      position: { x: 0, y: 0 },
      data: {
        label: svc.name,
        spanCount: svc.span_count,
        errorCount: svc.error_count,
        avgDurationNs: svc.avg_duration_ns,
        health: nodeHealthColor(svc),
      },
    }));

    const edges: Edge[] = serviceMap.edges.map((e, i) => ({
      id: `e-${e.source}-${e.target}-${i}`,
      source: e.source,
      target: e.target,
      label: `${e.count}`,
      animated: true,
      style: { strokeWidth: Math.min(1 + Math.log2(e.count), 4) },
    }));

    return { initialNodes: nodes, initialEdges: edges };
  }, [serviceMap]);

  // Apply ELK layout
  const [layoutedNodes, setLayoutedNodes] = useState<Node<ServiceFlowNodeData>[]>([]);
  const [layoutReady, setLayoutReady] = useState(false);

  useMemo(() => {
    if (initialNodes.length === 0) {
      setLayoutedNodes([]);
      setLayoutReady(true);
      return;
    }
    setLayoutReady(false);
    layoutGraph(initialNodes, initialEdges).then((result) => {
      setLayoutedNodes(result);
      setLayoutReady(true);
    });
  }, [initialNodes, initialEdges]);

  if (isLoading) {
    return (
      <Card className="border-none soft-shadow glass-panel">
        <CardContent className="p-6">
          <div className="flex items-center justify-center h-[400px]">
            <Skeleton className="h-full w-full rounded-lg" />
          </div>
        </CardContent>
      </Card>
    );
  }

  if (!serviceMap || serviceMap.nodes.length === 0) {
    return (
      <Card className="border-none soft-shadow glass-panel">
        <CardContent className="p-6">
          <div className="flex items-center justify-center h-[400px] text-muted-foreground text-sm">
            No service map data available. Traces must be collected first.
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card className="border-none soft-shadow glass-panel">
      <CardContent className="p-0">
        <div className="h-[500px] rounded-lg overflow-hidden">
          {layoutReady && (
            <ReactFlow
              nodes={layoutedNodes}
              edges={initialEdges}
              nodeTypes={nodeTypes}
              fitView
              fitViewOptions={{ padding: 0.3 }}
              proOptions={{ hideAttribution: true }}
              minZoom={0.3}
              maxZoom={2}
            >
              <Background variant={BackgroundVariant.Dots} gap={16} size={1} className="!bg-transparent" />
            </ReactFlow>
          )}
        </div>
      </CardContent>
    </Card>
  );
}

export function ServiceMapView() {
  return (
    <ReactFlowProvider>
      <ServiceMapInner />
    </ReactFlowProvider>
  );
}

