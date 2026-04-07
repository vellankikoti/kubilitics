/**
 * EventAnalyze — query builder + horizontal bar chart for aggregate analysis.
 * Uses direct fetch() to avoid React Query cluster ID issues.
 */
import { useState, useCallback, useEffect } from 'react';
import { BarChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer, Cell } from 'recharts';
import { Play, Loader2, Zap } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';
import { useEventsStore } from '@/stores/eventsStore';
import type { AnalyzeResult } from '@/services/api/eventsIntelligence';
import { getBackendBase } from '@/lib/backendUrl';

/* ─── Presets ────────────────────────────────────────────────────────────── */

const GROUP_BY_OPTIONS = [
  { label: 'Reason', value: 'reason' },
  { label: 'Namespace', value: 'resource_namespace' },
  { label: 'Resource Kind', value: 'resource_kind' },
  { label: 'Node', value: 'node_name' },
  { label: 'Severity', value: 'severity' },
  { label: 'Event Type', value: 'event_type' },
  { label: 'Source', value: 'source_component' },
  { label: 'Owner Kind', value: 'owner_kind' },
];

const TIME_PRESETS: { label: string; value: string }[] = [
  { label: '1h', value: '1h' },
  { label: '6h', value: '6h' },
  { label: '24h', value: '24h' },
  { label: '7d', value: '7d' },
];

const PRESET_QUERIES: { label: string; group_by: string; time_range: string }[] = [
  { label: 'Events by Namespace', group_by: 'resource_namespace', time_range: '24h' },
  { label: 'Events by Reason', group_by: 'reason', time_range: '24h' },
  { label: 'Events by Kind', group_by: 'resource_kind', time_range: '24h' },
  { label: 'Warnings by Node', group_by: 'node_name', time_range: '7d' },
];

const BAR_COLORS = [
  'hsl(221.2 83.2% 53.3%)',
  'hsl(263 70% 50%)',
  'hsl(142 76% 36%)',
  'hsl(38 92% 50%)',
  'hsl(346 77% 50%)',
  'hsl(199 89% 48%)',
];

/* ─── Component ──────────────────────────────────────────────────────────── */

export function EventAnalyze() {
  const store = useEventsStore();
  const [groupBy, setGroupBy] = useState('reason');
  const [timeRange, setTimeRange] = useState('24h');
  const [results, setResults] = useState<AnalyzeResult[] | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const runQuery = useCallback(async (gb?: string, tr?: string) => {
    const effectiveGroupBy = gb ?? groupBy;
    const effectiveTimeRange = tr ?? timeRange;
    setIsLoading(true);
    setError(null);
    try {
      const base = getBackendBase();
      const clustersRes = await fetch(`${base}/api/v1/clusters`);
      const clusters: Array<{ id: string; status: string }> = await clustersRes.json();
      const connected = clusters.find((c) => c.status === 'connected');
      if (!connected) {
        setResults([]);
        setIsLoading(false);
        return;
      }
      const res = await fetch(`${base}/api/v1/clusters/${connected.id}/events-intelligence/analyze`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          group_by: effectiveGroupBy,
          time_range: effectiveTimeRange,
          namespace: store.namespace || undefined,
          top_n: 20,
        }),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json();
      setResults(Array.isArray(data) ? data : []);
    } catch (err: unknown) {
      console.error('[EventAnalyze] fetch error:', err);
      setError(err instanceof Error ? err.message : 'Failed to fetch');
      setResults([]);
    } finally {
      setIsLoading(false);
    }
  }, [groupBy, timeRange, store.namespace]);

  // Auto-run default query on mount
  useEffect(() => {
    runQuery('reason', '24h');
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const runPreset = useCallback(
    (preset: typeof PRESET_QUERIES[0]) => {
      setGroupBy(preset.group_by);
      setTimeRange(preset.time_range);
      runQuery(preset.group_by, preset.time_range);
    },
    [runQuery],
  );

  const handleBarClick = useCallback(
    (data: { group_key: string }) => {
      if (groupBy === 'resource_namespace') {
        store.setNamespace(data.group_key);
      } else if (groupBy === 'resource_kind') {
        store.setResourceKind(data.group_key);
      } else if (groupBy === 'reason') {
        store.setEventReason(data.group_key);
      } else if (groupBy === 'event_type') {
        store.setEventType(data.group_key);
      }
      store.setMode('timeline');
    },
    [groupBy, store],
  );

  return (
    <div className="space-y-4">
      {/* Query builder */}
      <Card className="border-none soft-shadow glass-panel">
        <CardHeader className="pb-3">
          <CardTitle className="text-sm font-semibold flex items-center gap-2">
            <Zap className="h-4 w-4 text-primary" />
            Analyze Events
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex flex-wrap items-end gap-3">
            {/* Group by */}
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground">Group by</label>
              <select
                value={groupBy}
                onChange={(e) => setGroupBy(e.target.value)}
                className="h-9 rounded-md border border-input bg-background px-3 text-sm ring-offset-background focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2"
              >
                {GROUP_BY_OPTIONS.map((o) => (
                  <option key={o.value} value={o.value}>
                    {o.label}
                  </option>
                ))}
              </select>
            </div>

            {/* Time range */}
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground">Time Range</label>
              <div className="flex gap-1">
                {TIME_PRESETS.map((t) => (
                  <Button
                    key={t.value}
                    variant={timeRange === t.value ? 'default' : 'outline'}
                    size="sm"
                    className="h-9 text-xs"
                    onClick={() => setTimeRange(t.value)}
                  >
                    {t.label}
                  </Button>
                ))}
              </div>
            </div>

            {/* Run */}
            <Button onClick={() => runQuery()} className="h-9 gap-2" disabled={isLoading}>
              {isLoading ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <Play className="h-4 w-4" />
              )}
              Run Query
            </Button>
          </div>

          {/* Preset queries */}
          <div className="flex flex-wrap gap-2">
            {PRESET_QUERIES.map((p) => (
              <Button
                key={p.label}
                variant="outline"
                size="sm"
                className="h-7 text-xs"
                onClick={() => runPreset(p)}
              >
                {p.label}
              </Button>
            ))}
          </div>
        </CardContent>
      </Card>

      {/* Error state */}
      {error && (
        <Card className="border-none soft-shadow glass-panel">
          <CardContent className="py-4">
            <p className="text-sm text-destructive">{error}</p>
          </CardContent>
        </Card>
      )}

      {/* Results chart */}
      {results && results.length > 0 && (
        <Card className="border-none soft-shadow glass-panel">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-semibold">
              Results ({results.length} groups)
            </CardTitle>
          </CardHeader>
          <CardContent>
            <ResponsiveContainer width="100%" height={Math.max(200, results.length * 36)}>
              <BarChart
                data={results}
                layout="vertical"
                margin={{ left: 120, right: 16, top: 8, bottom: 8 }}
              >
                <XAxis type="number" tick={{ fontSize: 11 }} />
                <YAxis
                  type="category"
                  dataKey="group_key"
                  tick={{ fontSize: 11 }}
                  width={110}
                />
                <Tooltip
                  contentStyle={{
                    borderRadius: '8px',
                    border: 'none',
                    boxShadow: '0 4px 12px rgba(0,0,0,0.15)',
                    fontSize: '12px',
                  }}
                  formatter={(value: number) => [value, 'Count']}
                />
                <Bar
                  dataKey="count"
                  radius={[0, 4, 4, 0]}
                  cursor="pointer"
                  onClick={(data) => handleBarClick(data)}
                >
                  {results.map((_, i) => (
                    <Cell key={i} fill={BAR_COLORS[i % BAR_COLORS.length]} />
                  ))}
                </Bar>
              </BarChart>
            </ResponsiveContainer>

            {/* Results table */}
            <div className="mt-4 rounded-lg border border-border overflow-hidden">
              <table className="w-full text-sm">
                <thead>
                  <tr className="bg-muted/50">
                    <th className="text-left px-3 py-2 text-xs font-medium text-muted-foreground">#</th>
                    <th className="text-left px-3 py-2 text-xs font-medium text-muted-foreground">
                      {GROUP_BY_OPTIONS.find((o) => o.value === groupBy)?.label ?? 'Group'}
                    </th>
                    <th className="text-right px-3 py-2 text-xs font-medium text-muted-foreground">Count</th>
                    <th className="text-right px-3 py-2 text-xs font-medium text-muted-foreground">First Seen</th>
                    <th className="text-right px-3 py-2 text-xs font-medium text-muted-foreground">Last Seen</th>
                  </tr>
                </thead>
                <tbody>
                  {results.map((row, i) => (
                    <tr
                      key={row.group_key}
                      className="border-t border-border hover:bg-muted/30 cursor-pointer transition-colors"
                      onClick={() => handleBarClick(row)}
                    >
                      <td className="px-3 py-2 text-xs text-muted-foreground">{i + 1}</td>
                      <td className="px-3 py-2 font-medium">{row.group_key}</td>
                      <td className="px-3 py-2 text-right font-mono">{row.count}</td>
                      <td className="px-3 py-2 text-right text-xs text-muted-foreground">
                        {row.first_seen ? new Date(row.first_seen).toLocaleString(undefined, {
                          month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit',
                        }) : '-'}
                      </td>
                      <td className="px-3 py-2 text-right text-xs text-muted-foreground">
                        {row.last_seen ? new Date(row.last_seen).toLocaleString(undefined, {
                          month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit',
                        }) : '-'}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Empty state */}
      {results && results.length === 0 && !error && (
        <Card className="border-none soft-shadow glass-panel">
          <CardContent className="flex flex-col items-center justify-center py-16 text-muted-foreground">
            <Zap className="h-10 w-10 mb-3 opacity-30" />
            <p className="text-sm font-medium">No results</p>
            <p className="text-xs mt-1">Try a different group-by dimension or time range</p>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
