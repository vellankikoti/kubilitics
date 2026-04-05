/**
 * RecentEventsCard — compact card showing the last 8 cluster events.
 * Used on the Dashboard for at-a-glance cluster activity.
 */
import { Link } from 'react-router-dom';
import { Activity, ArrowRight } from 'lucide-react';
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Skeleton } from '@/components/ui/skeleton';
import { cn } from '@/lib/utils';
import { useEventsQuery } from '@/hooks/useEventsIntelligence';
import type { WideEvent } from '@/services/api/eventsIntelligence';

/* ─── Helpers ──────────────────────────────────────────────────────────────── */

function relativeTime(ts: number): string {
  const diff = Date.now() - ts;
  if (diff < 60_000) return `${Math.floor(diff / 1_000)}s ago`;
  if (diff < 3_600_000) return `${Math.floor(diff / 60_000)}m ago`;
  if (diff < 86_400_000) return `${Math.floor(diff / 3_600_000)}h ago`;
  return `${Math.floor(diff / 86_400_000)}d ago`;
}

const TYPE_DOT: Record<string, string> = {
  Warning: 'bg-amber-500',
  Normal: 'bg-blue-500',
};

/* ─── Compact Row ──────────────────────────────────────────────────────────── */

function CompactEventRow({ event }: { event: WideEvent }) {
  const dotColor = TYPE_DOT[event.event_type] ?? 'bg-muted-foreground';

  return (
    <div className="flex items-center gap-2.5 py-1.5 px-1 group">
      {/* Timestamp */}
      <span className="text-[11px] text-muted-foreground font-mono w-14 shrink-0 tabular-nums">
        {relativeTime(event.timestamp)}
      </span>

      {/* Type dot */}
      <span className={cn('h-1.5 w-1.5 rounded-full shrink-0', dotColor)} />

      {/* Resource name */}
      <span className="text-[13px] font-medium text-foreground truncate flex-1 min-w-0">
        {event.resource_name}
      </span>

      {/* Reason */}
      <span className="text-[11px] text-muted-foreground truncate max-w-[120px] hidden sm:inline">
        {event.reason}
      </span>
    </div>
  );
}

/* ─── Loading Skeleton ─────────────────────────────────────────────────────── */

function EventsSkeleton() {
  return (
    <div className="space-y-2">
      {Array.from({ length: 6 }).map((_, i) => (
        <div key={i} className="flex items-center gap-2.5 py-1.5 px-1">
          <Skeleton className="h-3 w-12" />
          <Skeleton className="h-1.5 w-1.5 rounded-full" />
          <Skeleton className="h-3 flex-1" />
          <Skeleton className="h-3 w-16 hidden sm:block" />
        </div>
      ))}
    </div>
  );
}

/* ─── Main Component ───────────────────────────────────────────────────────── */

export function RecentEventsCard() {
  const { data: events, isLoading } = useEventsQuery({ limit: 8 });

  return (
    <Card className="border-none soft-shadow glass-panel card-accent">
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Activity className="h-4 w-4 text-muted-foreground" />
            <CardTitle className="text-sm font-semibold">Recent Events</CardTitle>
            {events && events.length > 0 && (
              <Badge
                variant="secondary"
                className="text-[10px] h-5 px-1.5 font-medium"
              >
                {events.length}
              </Badge>
            )}
          </div>
          <Link
            to="/events-intelligence"
            className="flex items-center gap-1 text-[11px] font-medium text-primary hover:text-primary/80 transition-colors"
          >
            View All
            <ArrowRight className="h-3 w-3" />
          </Link>
        </div>
      </CardHeader>
      <CardContent className="pt-0">
        {isLoading ? (
          <EventsSkeleton />
        ) : !events || events.length === 0 ? (
          <p className="text-sm text-muted-foreground py-4 text-center">
            No recent events
          </p>
        ) : (
          <div className="divide-y divide-border/30">
            {events.map((event) => (
              <CompactEventRow key={event.event_id} event={event} />
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
