/**
 * EventStatsBar — bottom stats bar showing key event metrics.
 * Uses direct fetch() to avoid React Query cluster ID issues.
 */
import { useState, useEffect } from 'react';
import { Activity, AlertTriangle, HeartPulse, Flame, Loader2 } from 'lucide-react';
import { Card } from '@/components/ui/card';
import { cn } from '@/lib/utils';
import type { EventStats } from '@/services/api/eventsIntelligence';
import { getBackendBase } from '@/lib/backendUrl';

export function EventStatsBar() {
  const [stats, setStats] = useState<EventStats | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    async function load() {
      try {
        const base = getBackendBase();
        const clustersRes = await fetch(`${base}/api/v1/clusters`);
        const clusters: Array<{ id: string; status: string }> = await clustersRes.json();
        const connected = clusters.find((c) => c.status === 'connected');
        if (!connected) {
          if (!cancelled) { setStats(null); setIsLoading(false); }
          return;
        }
        const res = await fetch(`${base}/api/v1/clusters/${connected.id}/events-intelligence/stats`);
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        const data = await res.json();
        if (!cancelled) {
          // Normalize maps
          data.by_type = data.by_type ?? {};
          data.by_severity = data.by_severity ?? {};
          data.by_reason = data.by_reason ?? {};
          setStats(data);
          setIsLoading(false);
        }
      } catch (err: unknown) {
        if (!cancelled) {
          console.error('[EventStatsBar] fetch error:', err);
          setStats(null);
          setIsLoading(false);
        }
      }
    }
    load();
    const interval = setInterval(load, 30_000); // refresh every 30 seconds
    return () => { cancelled = true; clearInterval(interval); };
  }, []);

  if (isLoading) {
    return (
      <Card className="border-none soft-shadow glass-panel p-3">
        <div className="flex items-center justify-center">
          <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
        </div>
      </Card>
    );
  }

  if (!stats) return null;

  const warnings = stats.by_type?.['Warning'] ?? 0;
  const normals = stats.by_type?.['Normal'] ?? 0;

  // Top reason
  const topReason = stats.by_reason
    ? Object.entries(stats.by_reason).sort(([, a], [, b]) => b - a)[0]
    : null;

  return (
    <Card className="border-none soft-shadow glass-panel">
      <div className="flex items-center justify-between px-4 py-3 gap-6 flex-wrap text-xs">
        <StatItem
          icon={Activity}
          label="Total Events (24h)"
          value={String(stats.total_events ?? 0)}
          iconClassName="text-blue-500"
        />
        <StatItem
          icon={AlertTriangle}
          label="Warnings"
          value={String(warnings)}
          iconClassName="text-amber-500"
        />
        <StatItem
          icon={HeartPulse}
          label="Normal"
          value={String(normals)}
          iconClassName="text-green-500"
        />
        {topReason && (
          <StatItem
            icon={Flame}
            label="Top Reason"
            value={`${topReason[0]} (${topReason[1]})`}
            iconClassName="text-purple-500"
          />
        )}
      </div>
    </Card>
  );
}

function StatItem({
  icon: Icon,
  label,
  value,
  iconClassName,
}: {
  icon: typeof Activity;
  label: string;
  value: string;
  iconClassName?: string;
}) {
  return (
    <div className="flex items-center gap-2">
      <Icon className={cn('h-4 w-4 shrink-0', iconClassName)} />
      <div>
        <p className="text-[10px] text-muted-foreground uppercase tracking-wider">{label}</p>
        <p className="text-sm font-semibold">{value}</p>
      </div>
    </div>
  );
}
