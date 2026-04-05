/**
 * EventTimeline — Filter sidebar + event list for Timeline mode.
 */
import { useMemo, useCallback, useState, useEffect } from 'react';
import { motion } from 'framer-motion';
import { AlertTriangle, RotateCcw, Activity, Skull, Loader2, FilterX } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { cn } from '@/lib/utils';
import { useEventsStore } from '@/stores/eventsStore';
import { EventRow } from './EventRow';
import { ListPagination } from '@/components/list/ListPagination';
import type { WideEvent, EventQueryParams } from '@/services/api/eventsIntelligence';

/* ─── Time range presets ─────────────────────────────────────────────────── */

const TIME_RANGES: { label: string; value: string; ms: number }[] = [
  { label: '1h', value: '1h', ms: 3_600_000 },
  { label: '6h', value: '6h', ms: 21_600_000 },
  { label: '24h', value: '24h', ms: 86_400_000 },
  { label: '7d', value: '7d', ms: 604_800_000 },
];

const QUICK_FILTERS = [
  { label: 'Warnings', type: 'Warning', reason: '' },
  { label: 'OOMKills', type: '', reason: 'OOMKilling' },
  { label: 'Restarts', type: '', reason: 'BackOff' },
  { label: 'Failed', type: '', reason: 'Failed' },
];

const KINDS = ['', 'Pod', 'Deployment', 'ReplicaSet', 'StatefulSet', 'DaemonSet', 'Service', 'Node', 'Job', 'CronJob', 'Ingress'];
const TYPES = ['', 'Normal', 'Warning'];

/** Check whether any filter is active (non-default) */
function hasActiveFilters(store: { namespace: string; resourceKind: string; eventType: string; eventReason: string }) {
  return !!(store.namespace || store.resourceKind || store.eventType || store.eventReason);
}

/* ─── Component ──────────────────────────────────────────────────────────── */

const EVT_PAGE_SIZE = 25;

export function EventTimeline() {
  const store = useEventsStore();
  const [evtPage, setEvtPage] = useState(1);
  // SSE disabled — causes crashes in Tauri WebView. Events refresh via polling instead.
  const liveEvents: WideEvent[] = [];
  const isConnected = false;

  const timeRangeMs = TIME_RANGES.find((t) => t.value === store.timeRange)?.ms ?? 86_400_000;
  const from = Date.now() - timeRangeMs;

  const params: EventQueryParams = useMemo(
    () => ({
      from,
      to: Date.now(),
      namespace: store.namespace || undefined,
      kind: store.resourceKind || undefined,
      type: store.eventType || undefined,
      reason: store.eventReason || undefined,
      limit: 100,
      offset: 0,
    }),
    [from, store.namespace, store.resourceKind, store.eventType, store.eventReason],
  );

  const [historicalEvents, setHistoricalEvents] = useState<WideEvent[]>([]);
  const [initialLoading, setInitialLoading] = useState(true);
  const [isError, setIsError] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const isLoading = initialLoading && historicalEvents.length === 0;
  const isFetching = false;

  useEffect(() => {
    let cancelled = false;
    async function load() {
      try {
        const base = 'http://localhost:8190';
        const clustersRes = await fetch(`${base}/api/v1/clusters`);
        const clusters = await clustersRes.json();
        const connected = clusters.find((c: any) => c.status === 'connected');
        if (!connected) { setHistoricalEvents([]); setInitialLoading(false); return; }
        const qs = new URLSearchParams({
          from: String(params.from), to: String(params.to || Date.now()),
          limit: String(params.limit || 100), offset: '0',
        });
        if (params.namespace) qs.set('namespace', params.namespace);
        if (params.kind) qs.set('kind', params.kind);
        if (params.type) qs.set('type', params.type);
        if (params.reason) qs.set('reason', params.reason);
        const res = await fetch(`${base}/api/v1/clusters/${connected.id}/events-intelligence/query?${qs}`);
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        const data = await res.json();
        if (!cancelled) { setHistoricalEvents(Array.isArray(data) ? data : []); setInitialLoading(false); }
      } catch (err: any) {
        if (!cancelled) { setError(err); setIsError(true); setInitialLoading(false); }
      }
    }
    load();
    const interval = setInterval(load, 30000);
    return () => { cancelled = true; clearInterval(interval); };
  }, [params.from, params.to, params.namespace, params.kind, params.type, params.reason, params.limit, params.offset]);

  // Merge live + historical, deduplicate by event_id
  const mergedEvents = useMemo(() => {
    const map = new Map<string, (typeof liveEvents)[0]>();
    for (const e of liveEvents) map.set(e.event_id, e);
    for (const e of historicalEvents ?? []) {
      if (!map.has(e.event_id)) map.set(e.event_id, e);
    }
    return Array.from(map.values()).sort((a, b) => b.timestamp - a.timestamp);
  }, [liveEvents, historicalEvents]);

  // Extract unique namespaces from loaded events for the filter dropdown
  const uniqueNamespaces = useMemo(() => {
    const ns = new Set<string>();
    for (const e of historicalEvents) {
      if (e.resource_namespace) ns.add(e.resource_namespace);
    }
    return Array.from(ns).sort();
  }, [historicalEvents]);

  // Compute counts per filter value for display
  const filterCounts = useMemo(() => {
    const nsCounts: Record<string, number> = {};
    const kindCounts: Record<string, number> = {};
    const typeCounts: Record<string, number> = {};
    for (const e of mergedEvents) {
      if (e.resource_namespace) nsCounts[e.resource_namespace] = (nsCounts[e.resource_namespace] || 0) + 1;
      if (e.resource_kind) kindCounts[e.resource_kind] = (kindCounts[e.resource_kind] || 0) + 1;
      if (e.event_type) typeCounts[e.event_type] = (typeCounts[e.event_type] || 0) + 1;
    }
    return { nsCounts, kindCounts, typeCounts };
  }, [mergedEvents]);

  const filtersActive = hasActiveFilters(store);

  // Pagination
  const evtTotalPages = Math.max(1, Math.ceil(mergedEvents.length / EVT_PAGE_SIZE));
  const evtStartIdx = (evtPage - 1) * EVT_PAGE_SIZE;

  // Reset page when filters change
  useEffect(() => { setEvtPage(1); }, [store.namespace, store.resourceKind, store.eventType, store.eventReason, store.timeRange]);

  const handleQuickFilter = useCallback(
    (type: string, reason: string) => {
      store.setEventType(type);
      store.setEventReason(reason);
    },
    [store],
  );

  const handleViewContext = useCallback(
    (eventId: string) => {
      store.selectEvent(eventId);
    },
    [store],
  );

  return (
    <div className="grid grid-cols-1 lg:grid-cols-[260px_1fr] gap-4">
      {/* Filter sidebar */}
      <Card className="border-none soft-shadow glass-panel h-fit">
        <CardHeader className="pb-3">
          <CardTitle className="text-sm font-semibold flex items-center gap-2">
            <Activity className="h-4 w-4 text-primary" />
            Filters
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          {/* Time range */}
          <div className="space-y-1.5">
            <span className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/60">Time Range</span>
            <div className="flex gap-1">
              {TIME_RANGES.map((t) => (
                <Button
                  key={t.value}
                  variant={store.timeRange === t.value ? 'default' : 'outline'}
                  size="sm"
                  className={cn(
                    'h-6 text-[11px] flex-1 px-0 rounded-full',
                    store.timeRange === t.value && 'shadow-sm',
                  )}
                  onClick={() => store.setTimeRange(t.value)}
                >
                  {t.label}
                </Button>
              ))}
            </div>
          </div>

          {/* Namespace */}
          <FilterSelect
            label="Namespace"
            value={store.namespace}
            onChange={store.setNamespace}
            options={uniqueNamespaces}
            placeholder="All namespaces"
            counts={filterCounts.nsCounts}
          />

          {/* Kind */}
          <FilterSelect
            label="Resource Kind"
            value={store.resourceKind}
            onChange={store.setResourceKind}
            options={KINDS}
            placeholder="All kinds"
            counts={filterCounts.kindCounts}
          />

          {/* Type */}
          <FilterSelect
            label="Event Type"
            value={store.eventType}
            onChange={store.setEventType}
            options={TYPES}
            placeholder="All types"
            counts={filterCounts.typeCounts}
          />

          {/* Reset */}
          <Button
            variant={filtersActive ? 'outline' : 'ghost'}
            size="sm"
            className={cn(
              'w-full h-7 text-xs gap-1.5',
              filtersActive && 'border-amber-500/40 text-amber-600 dark:text-amber-400 hover:bg-amber-500/10',
            )}
            onClick={store.resetFilters}
          >
            {filtersActive ? <FilterX className="h-3 w-3" /> : <RotateCcw className="h-3 w-3" />}
            {filtersActive ? 'Clear Active Filters' : 'Reset Filters'}
          </Button>

          {/* Quick filters */}
          <div className="space-y-1.5 pt-2 border-t border-border/40">
            <span className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/60">Quick Filters</span>
            <div className="grid grid-cols-2 gap-1">
              {QUICK_FILTERS.map((qf) => {
                const isActive = (qf.type && store.eventType === qf.type) || (qf.reason && store.eventReason === qf.reason);
                return (
                  <Button
                    key={qf.label}
                    variant={isActive ? 'default' : 'outline'}
                    size="sm"
                    className={cn('h-7 text-xs', isActive && 'shadow-sm')}
                    onClick={() => handleQuickFilter(qf.type, qf.reason)}
                  >
                    {qf.label}
                  </Button>
                );
              })}
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Event list */}
      <Card className="border-none soft-shadow glass-panel">
        <CardHeader className="pb-2">
          <div className="flex items-center justify-between">
            <CardTitle className="text-sm font-semibold flex items-center gap-2">
              Events
              {mergedEvents.length > 0 && (
                <Badge variant="secondary" className="text-[10px] h-5">
                  {mergedEvents.length}
                </Badge>
              )}
              {isFetching && <Loader2 className="h-3 w-3 animate-spin text-muted-foreground" />}
            </CardTitle>
            <div className="flex items-center gap-2">
              {isConnected && (
                <Badge variant="secondary" className="bg-green-500/10 text-green-600 dark:text-green-400 border-green-500/20 text-[10px] h-5">
                  <span className="relative flex h-1.5 w-1.5 mr-1">
                    <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-500 opacity-60" />
                    <span className="relative inline-flex rounded-full h-1.5 w-1.5 bg-green-500" />
                  </span>
                  Live
                </Badge>
              )}
            </div>
          </div>
        </CardHeader>
        <CardContent className="p-0">
          {isError ? (
            <div className="flex flex-col items-center justify-center py-16 text-destructive">
              <AlertTriangle className="h-10 w-10 mb-3 opacity-50" />
              <p className="text-sm font-medium">Failed to load events</p>
              <p className="text-xs mt-1 text-muted-foreground">{error?.message || 'Unknown error'}</p>
            </div>
          ) : isLoading ? (
            <div className="flex items-center justify-center py-16">
              <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
            </div>
          ) : mergedEvents.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-16 text-muted-foreground">
              <Activity className="h-10 w-10 mb-3 opacity-30" />
              <p className="text-sm font-medium">No events match your filters</p>
              <p className="text-xs mt-1">Try adjusting the time range or clearing filters</p>
              {filtersActive && (
                <Button
                  variant="outline"
                  size="sm"
                  className="mt-4 h-8 text-xs gap-1.5"
                  onClick={store.resetFilters}
                >
                  <RotateCcw className="h-3 w-3" />
                  Reset Filters
                </Button>
              )}
            </div>
          ) : (
            <>
              <div className="max-h-[600px] overflow-y-auto">
                {mergedEvents.slice(evtStartIdx, evtStartIdx + EVT_PAGE_SIZE).map((event) => (
                  <EventRow
                    key={event.event_id}
                    event={event}
                    onViewContext={handleViewContext}
                  />
                ))}
              </div>
              {mergedEvents.length > EVT_PAGE_SIZE && (
                <div className="px-4 py-3 border-t border-border/40">
                  <ListPagination
                    hasPrev={evtPage > 1}
                    hasNext={evtPage < evtTotalPages}
                    onPrev={() => setEvtPage(p => Math.max(1, p - 1))}
                    onNext={() => setEvtPage(p => Math.min(evtTotalPages, p + 1))}
                    currentPage={evtPage}
                    totalPages={evtTotalPages}
                    onPageChange={setEvtPage}
                    rangeLabel={`${evtStartIdx + 1}–${Math.min(evtStartIdx + EVT_PAGE_SIZE, mergedEvents.length)} of ${mergedEvents.length}`}
                  />
                </div>
              )}
            </>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

/* ─── Filter Select ──────────────────────────────────────────────────────── */

function FilterSelect({
  label,
  value,
  onChange,
  options,
  placeholder,
  counts,
}: {
  label: string;
  value: string;
  onChange: (v: string) => void;
  options: string[];
  placeholder: string;
  counts?: Record<string, number>;
}) {
  return (
    <div className="space-y-1.5">
      <span className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/60">{label}</span>
      <select
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="w-full h-8 rounded-md border border-input bg-background px-2 text-xs ring-offset-background focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2"
      >
        <option value="">{placeholder}</option>
        {options.filter(Boolean).map((opt) => (
          <option key={opt} value={opt}>
            {opt}{counts?.[opt] != null ? ` (${counts[opt]})` : ''}
          </option>
        ))}
      </select>
    </div>
  );
}
