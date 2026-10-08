/**
 * EventStatsBar — bottom stats bar showing key event metrics.
 */
import { Activity, AlertTriangle, HeartPulse, Flame, Loader2, AlertCircle } from 'lucide-react';
import { Card } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';
import { useEventStats } from '@/hooks/useEventsIntelligence';

export function EventStatsBar() {
  const { data: stats, isLoading, isError, refetch } = useEventStats();

  if (isLoading) {
    return (
      <Card className="border-none soft-shadow glass-panel p-3">
        <div className="flex items-center justify-center">
          <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
        </div>
      </Card>
    );
  }

  if (isError) {
    return (
      <Card className="border-none soft-shadow glass-panel p-3">
        <div className="flex items-center justify-between gap-3 text-xs text-muted-foreground">
          <span className="flex items-center gap-2">
            <AlertCircle className="h-3.5 w-3.5 text-destructive" />
            Couldn&apos;t load event stats.
          </span>
          <Button variant="outline" size="sm" className="h-6 text-xs" onClick={() => refetch()}>
            Retry
          </Button>
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
