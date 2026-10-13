/**
 * Cluster Health — gauge, breakdown, insight. Uses overview API when available; else useHealthScore.
 */
import React from "react";
import { Link } from "react-router-dom";
import { PieChart, Pie, Cell, ResponsiveContainer, Label } from "recharts";
import { Info, CheckCircle2, AlertTriangle, AlertCircle, ChevronRight, Boxes, Server, Layers, RefreshCw, ShieldAlert } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Progress } from "@/components/ui/progress";
import { useClusterOverview } from "@/hooks/useClusterOverview";
import { useHealthScore } from "@/hooks/useHealthScore";
import { useClusterReachability } from "@/hooks/useClusterReachability";
import { useActiveCluster } from "@/stores/clusterPresenceStore";
import { useBackendConfigStore } from "@/stores/backendConfigStore";
import { EmptyNoClusters } from "@/components/ui/empty-state";
import { cn } from "@/lib/utils";

import { useActiveClusterId } from '@/hooks/useActiveClusterId';
const STATUS_CONFIG: Record<string, { badge: string; icon: typeof CheckCircle2 }> = {
  excellent: { badge: "bg-success/10 border-success/20 text-success", icon: CheckCircle2 },
  healthy: { badge: "bg-success/10 border-success/20 text-success", icon: CheckCircle2 },
  good: { badge: "bg-success/10 border-success/20 text-success", icon: CheckCircle2 },
  fair: { badge: "bg-amber-100 dark:bg-amber-900/30 border-amber-200 text-amber-700 dark:text-amber-400", icon: AlertTriangle },
  degraded: { badge: "bg-amber-100 dark:bg-amber-900/30 border-amber-200 text-amber-700 dark:text-amber-400", icon: AlertTriangle },
  poor: { badge: "bg-amber-100 dark:bg-amber-900/30 border-amber-200 text-amber-700 dark:text-amber-400", icon: AlertTriangle },
  unhealthy: { badge: "bg-rose-100 dark:bg-rose-900/30 border-rose-200 text-rose-700 dark:text-rose-400", icon: AlertCircle },
  critical: { badge: "bg-rose-100 dark:bg-rose-900/30 border-rose-200 text-rose-700 dark:text-rose-400", icon: AlertCircle },
};

const BREAKDOWN_LABELS: Record<string, string> = {
  nodeHealth: "Nodes",
  workloadHealth: "Workloads",
  podHealth: "Pods",
  stability: "Stability",
  eventHealth: "Events",
};

export const ClusterHealthWidget = () => {
  const activeCluster = useActiveCluster();
  const currentClusterId = useActiveClusterId();
  const isBackendConfigured = useBackendConfigStore((s) => s.isBackendConfigured());
  const clusterId = currentClusterId ?? undefined;

  const overview = useClusterOverview(clusterId);
  const healthScore = useHealthScore();
  const reachability = useClusterReachability(clusterId);
  // Final review, Critical #2: real data (a health score) may only ever
  // render when reachability is positively 'connected' — never merely
  // "not confirmed unreachable". 'connecting'/'unknown' previously fell
  // through to the real-score branch, so a cluster the backend has never
  // actually reached (no client ever constructed, reachable:false with
  // no last_checked_at) still showed a cached "Grade A" from
  // useHealthScore's own optimistic empty-data default — the same
  // fabricated-healthy bug this whole plan exists to kill, just reached
  // via a different state than 'unreachable'.
  const isUnreachable = reachability.status === 'unreachable' || reachability.status === 'error';
  const isConnecting = reachability.status === 'connecting' || reachability.status === 'unknown';
  const isConnected = reachability.status === 'connected';

  // UX-1 (docs/PRODUCTION-HARDENING-ROADMAP.md, Phase 7): this widget
  // previously never checked overview/healthScore loading or error state at
  // all — on first load (before any data arrives) it rendered healthScore's
  // default "score 0, grade F, critical" exactly as if the cluster were
  // actually unhealthy, and a fetch failure looked identical to "0 issues."
  // healthScore.isLoading/isError already aggregate across this widget's own
  // overview query and the direct-K8s fallback (see useHealthScore.ts).
  const isLoading = healthScore.isLoading;
  const isError = healthScore.isError;

  const hasOverview = isBackendConfigured && clusterId && overview.data?.health;
  const score = hasOverview ? overview.data!.health.score : healthScore.score;
  const grade = hasOverview ? overview.data!.health.grade : healthScore.grade;
  const status = hasOverview ? overview.data!.health.status : healthScore.status;
  const { breakdown, insight } = healthScore;

  const config = STATUS_CONFIG[status] ?? STATUS_CONFIG.critical;
  const StatusIcon = config.icon;

  const data = [
    { name: "Healthy", value: score, color: "url(#colorGradient)" },
    { name: "Issues", value: 100 - score, color: "hsl(var(--muted))" },
  ];

  const statusLabel =
    ["excellent", "good", "healthy"].includes(status) ? "Good State" :
    ["fair", "degraded"].includes(status) ? "Needs Attention" :
    "At Risk";
  const hasIdentity = !!(activeCluster || clusterId);
  const hasData = hasIdentity && isConnected;

  return (
    <Card className="h-full min-h-[28rem] border-none soft-shadow glass-panel relative overflow-hidden group flex flex-col">
      <div className="absolute top-0 left-0 right-0 h-1 bg-gradient-to-r from-primary via-blue-500 to-cyan-500" />
      <div className="absolute top-0 right-0 w-48 h-48 bg-primary/5 rounded-full blur-3xl -translate-y-1/2 translate-x-1/2 pointer-events-none" />

      <CardHeader className="pb-2 pt-5 px-6 relative z-10 shrink-0">
        <div className="flex items-center justify-between gap-2">
          <h2 className="text-lg font-bold text-foreground">
            Cluster Health
          </h2>
          <div className="flex items-center gap-2">
            {/* UX-1: the status badge must not claim a health verdict ("At
                Risk" etc.) while that verdict is still loading or could not
                be determined — same reasoning as the CardContent gating below. */}
            {!isLoading && !isError && hasData && (
              <div className={cn("inline-flex items-center gap-1.5 px-2 py-1 rounded-full border text-xs font-semibold shadow-sm backdrop-blur-sm", config.badge)}>
                <StatusIcon className="w-3 h-3" />
                <span>{statusLabel}</span>
              </div>
            )}
            <Tooltip>
              <TooltipTrigger>
                <Info className="w-3.5 h-3.5 text-muted-foreground hover:text-primary transition-colors shrink-0" />
              </TooltipTrigger>
              <TooltipContent>Overall cluster health based on node status, pod availability, and event severity.</TooltipContent>
            </Tooltip>
          </div>
        </div>
      </CardHeader>

      <CardContent className="flex-1 flex flex-col pt-2 pb-6 px-6 relative z-10">
        {isUnreachable ? (
          <div className="flex-1 flex flex-col items-center justify-center gap-2 text-center" role="status">
            <AlertTriangle className="w-8 h-8 text-amber-500" />
            <p className="text-sm font-medium text-foreground">Cluster unreachable</p>
            <p className="text-xs text-muted-foreground max-w-[22rem]">
              {reachability.status === 'unreachable'
                ? `Last connected ${reachability.lastSuccessAt ? new Date(reachability.lastSuccessAt).toLocaleString() : 'previously'}.`
                : 'This cluster has never been successfully reached.'}
            </p>
          </div>
        ) : isConnecting && hasIdentity ? (
          <div className="flex-1 flex flex-col items-center justify-center gap-3" role="status" aria-label="Connecting to cluster">
            <div className="h-40 w-40 rounded-full border-4 border-muted animate-pulse" />
            <p className="text-xs text-muted-foreground">Connecting to cluster…</p>
          </div>
        ) : !hasData ? (
          <div className="flex-1 flex items-center justify-center">
            <EmptyNoClusters
              size="sm"
              primaryAction={{ label: "Connect Cluster", href: "/clusters?addCluster=true" }}
            />
          </div>
        ) : isLoading ? (
          <div className="flex-1 flex flex-col items-center justify-center gap-3" role="status" aria-label="Loading cluster health">
            <div className="h-40 w-40 rounded-full border-4 border-muted animate-pulse" />
            <p className="text-xs text-muted-foreground">Loading cluster health…</p>
          </div>
        ) : isError ? (
          <div className="flex-1 flex flex-col items-center justify-center gap-2 text-center" role="alert">
            <AlertCircle className="w-8 h-8 text-destructive" />
            <p className="text-sm font-medium text-foreground">Unable to load cluster health</p>
            <p className="text-xs text-muted-foreground max-w-[22rem]">
              The health check request failed or timed out. This does not mean the cluster is unhealthy — it means health data could not be retrieved right now.
            </p>
          </div>
        ) : (
          <>
            <div className="flex items-center justify-center shrink-0" style={{ height: "160px" }}>
              <ResponsiveContainer width="100%" height="100%">
                <PieChart aria-label="Cluster health score chart">
                  <defs>
                    <linearGradient id="colorGradient" x1="0" y1="0" x2="1" y2="1">
                      <stop offset="0%" stopColor="hsl(var(--primary))" />
                      <stop offset="100%" stopColor="#00E5FF" />
                    </linearGradient>
                  </defs>
                  <Pie
                    data={data}
                    cx="50%"
                    cy="50%"
                    innerRadius={48}
                    outerRadius={64}
                    startAngle={90}
                    endAngle={-270}
                    dataKey="value"
                    stroke="none"
                    cornerRadius={10}
                    paddingAngle={4}
                  >
                    {data.map((entry, index) => (
                      <Cell key={`cell-${index}`} fill={entry.color} aria-label={`${entry.name}: ${entry.value}%`} />
                    ))}
                    <Label
                      content={({ viewBox }) => {
                        if (viewBox && "cx" in viewBox && "cy" in viewBox) {
                          return (
                            <text x={viewBox.cx} y={viewBox.cy} textAnchor="middle" dominantBaseline="middle">
                              <tspan x={viewBox.cx} y={(viewBox.cy || 0) - 4} className="fill-foreground text-3xl font-black tracking-tighter">
                                {score}
                              </tspan>
                              <tspan x={viewBox.cx} y={(viewBox.cy || 0) + 14} className="fill-muted-foreground text-[10px] font-bold uppercase tracking-[0.2em]">
                                Grade {grade}
                              </tspan>
                            </text>
                          );
                        }
                        return null;
                      }}
                    />
                  </Pie>
                </PieChart>
              </ResponsiveContainer>
            </div>

            {/* Breakdown — 5 factors (matches Resource Allocation vertical style) */}
            <div className="space-y-2 pt-3 mt-auto border-t border-border/50">
              <p className="text-[10px] font-semibold text-muted-foreground uppercase tracking-wider">Health factors</p>
              <div className="space-y-3">
                {(["nodeHealth", "workloadHealth", "podHealth", "stability", "eventHealth"] as const).map((key) => {
                  const val = Math.min(100, Math.max(0, breakdown[key]));
                  const Icon = key === "podHealth" ? Boxes : key === "nodeHealth" ? Server : key === "workloadHealth" ? Layers : key === "stability" ? RefreshCw : ShieldAlert;
                  return (
                    <div key={key} className="space-y-1">
                      <div className="flex justify-between text-xs">
                        <span className="text-muted-foreground flex items-center gap-1.5">
                          <Icon className="w-3 h-3" /> {BREAKDOWN_LABELS[key]}
                        </span>
                        <span className="font-semibold tabular-nums">{val}%</span>
                      </div>
                      <Progress
                        value={val}
                        className="h-1.5 bg-muted/50"
                        indicatorClassName={cn(
                          val >= 80 ? "bg-emerald-500" : val >= 60 ? "bg-amber-500" : "bg-rose-500"
                        )}
                      />
                    </div>
                  );
                })}
              </div>
            </div>

            {/* Insight */}
            {insight && (
              <div className="mt-4 pt-3 border-t border-border/50">
                <p className="text-xs text-muted-foreground leading-relaxed line-clamp-2">{insight}</p>
                {(["fair", "degraded", "poor", "unhealthy", "critical"].includes(status)) && (
                  <Link
                    to="/events"
                    className="inline-flex items-center gap-1 mt-2 text-xs font-medium text-primary hover:underline"
                  >
                    View events
                    <ChevronRight className="w-3 h-3" />
                  </Link>
                )}
              </div>
            )}
          </>
        )}
      </CardContent>
    </Card>
  );
};
