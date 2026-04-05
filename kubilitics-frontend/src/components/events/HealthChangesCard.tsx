/**
 * HealthChangesCard — "What Changed" card showing recent changes
 * that impacted cluster health. Used on the HealthDashboard.
 */
import { GitCommit } from 'lucide-react';
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Skeleton } from '@/components/ui/skeleton';
import { cn } from '@/lib/utils';
import { useRecentChanges } from '@/hooks/useEventsIntelligence';
import type { Change } from '@/services/api/eventsIntelligence';

/* ─── Helpers ──────────────────────────────────────────────────────────────── */

function relativeTime(ts: number): string {
  const diff = Date.now() - ts;
  if (diff < 60_000) return `${Math.floor(diff / 1_000)}s ago`;
  if (diff < 3_600_000) return `${Math.floor(diff / 60_000)}m ago`;
  if (diff < 86_400_000) return `${Math.floor(diff / 3_600_000)}h ago`;
  return `${Math.floor(diff / 86_400_000)}d ago`;
}

const CHANGE_TYPE_STYLES: Record<string, string> = {
  rollout: 'bg-blue-500/15 text-blue-600 dark:text-blue-400 border-blue-500/20',
  scale: 'bg-purple-500/15 text-purple-600 dark:text-purple-400 border-purple-500/20',
  config: 'bg-amber-500/15 text-amber-600 dark:text-amber-400 border-amber-500/20',
  update: 'bg-cyan-500/15 text-cyan-600 dark:text-cyan-400 border-cyan-500/20',
};

/* ─── Change Row ───────────────────────────────────────────────────────────── */

function ChangeRow({ change }: { change: Change }) {
  const typeStyle =
    CHANGE_TYPE_STYLES[change.change_type] ??
    'bg-muted text-muted-foreground border-border';

  const healthDelta = change.health_impact;

  return (
    <div className="flex items-center gap-2.5 py-2.5">
      {/* Change type badge */}
      <Badge
        variant="outline"
        className={cn('text-[10px] px-1.5 py-0 h-5 shrink-0 font-medium', typeStyle)}
      >
        {change.change_type}
      </Badge>

      {/* Resource name */}
      <div className="flex-1 min-w-0">
        <span className="text-[13px] font-medium text-foreground truncate block">
          {change.resource_name}
        </span>
        <span className="text-[11px] text-muted-foreground">
          {change.resource_kind}
          {change.resource_namespace ? ` / ${change.resource_namespace}` : ''}
          {' \u00b7 '}
          {relativeTime(change.timestamp)}
        </span>
      </div>

      {/* Health impact delta */}
      {healthDelta != null && healthDelta !== 0 && (
        <Badge
          variant="outline"
          className={cn(
            'text-[10px] px-1.5 py-0 h-5 shrink-0 font-mono font-bold',
            healthDelta < 0
              ? 'text-red-500 border-red-500/20'
              : 'text-green-500 border-green-500/20',
          )}
        >
          {healthDelta > 0 ? '+' : ''}
          {Math.round(healthDelta)}
        </Badge>
      )}
    </div>
  );
}

/* ─── Main Component ───────────────────────────────────────────────────────── */

export function HealthChangesCard() {
  const { data: changes, isLoading } = useRecentChanges(5);

  // Filter to only changes with health_impact
  const impactfulChanges = (changes ?? []).filter(
    (c) => c.health_impact != null && c.health_impact !== 0,
  );

  return (
    <Card className="border-none soft-shadow glass-panel">
      <CardHeader className="pb-3">
        <div className="flex items-center gap-2">
          <GitCommit className="h-4 w-4 text-muted-foreground" />
          <CardTitle className="text-sm font-semibold">What Changed</CardTitle>
          {impactfulChanges.length > 0 && (
            <Badge
              variant="secondary"
              className="text-[10px] h-5 px-1.5 font-medium"
            >
              {impactfulChanges.length}
            </Badge>
          )}
        </div>
      </CardHeader>
      <CardContent className="pt-0">
        {isLoading ? (
          <div className="space-y-3">
            {Array.from({ length: 3 }).map((_, i) => (
              <div key={i} className="flex items-center gap-2.5 py-2">
                <Skeleton className="h-5 w-14" />
                <div className="flex-1 space-y-1">
                  <Skeleton className="h-3 w-3/4" />
                  <Skeleton className="h-2.5 w-1/2" />
                </div>
                <Skeleton className="h-5 w-8" />
              </div>
            ))}
          </div>
        ) : impactfulChanges.length === 0 ? (
          <p className="text-sm text-muted-foreground py-3 text-center">
            No recent health-impacting changes
          </p>
        ) : (
          <div className="divide-y divide-border/30">
            {impactfulChanges.map((change) => (
              <ChangeRow key={change.change_id} change={change} />
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
