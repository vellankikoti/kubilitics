/**
 * IncidentView — incident narrative cards.
 * Uses direct fetch() to avoid React Query cluster ID issues.
 */
import { useState, useCallback, useEffect } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import {
  AlertCircle,
  AlertTriangle,
  CheckCircle2,
  ChevronRight,
  Clock,
  Activity,
  Loader2,
  Shield,
} from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';
import { getBackendBase } from '@/lib/backendUrl';
import { useEventsStore } from '@/stores/eventsStore';
import type { Incident } from '@/services/api/eventsIntelligence';

/* ─── Helpers ────────────────────────────────────────────────────────────── */

const SEVERITY_STYLES: Record<string, { bg: string; text: string; icon: typeof AlertCircle }> = {
  critical: { bg: 'bg-red-500/15', text: 'text-red-600 dark:text-red-400', icon: AlertCircle },
  high: { bg: 'bg-amber-500/15', text: 'text-amber-600 dark:text-amber-400', icon: AlertTriangle },
  medium: { bg: 'bg-yellow-500/15', text: 'text-yellow-600 dark:text-yellow-400', icon: AlertTriangle },
  low: { bg: 'bg-green-500/15', text: 'text-green-600 dark:text-green-400', icon: CheckCircle2 },
};

const STATUS_STYLES: Record<string, string> = {
  active: 'bg-red-500/15 text-red-600 dark:text-red-400 border-red-500/20',
  resolved: 'bg-green-500/15 text-green-600 dark:text-green-400 border-green-500/20',
  investigating: 'bg-amber-500/15 text-amber-600 dark:text-amber-400 border-amber-500/20',
};

function formatDuration(ms: number): string {
  if (ms < 60_000) return `${Math.round(ms / 1000)}s`;
  if (ms < 3_600_000) return `${Math.round(ms / 60_000)}m`;
  return `${(ms / 3_600_000).toFixed(1)}h`;
}

function formatTime(ts: number): string {
  return new Date(ts).toLocaleString(undefined, {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

/* ─── Component ──────────────────────────────────────────────────────────── */

export function IncidentView() {
  const store = useEventsStore();
  const [incidents, setIncidents] = useState<Incident[] | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    async function load() {
      try {
        const base = getBackendBase();
        const clustersRes = await fetch(`${base}/api/v1/clusters`);
        const clusters: Array<{ id: string; status: string }> = await clustersRes.json();
        const connected = clusters.find((c) => c.status === 'connected');
        if (!connected) {
          if (!cancelled) { setIncidents([]); setIsLoading(false); }
          return;
        }
        const res = await fetch(`${base}/api/v1/clusters/${connected.id}/incidents`);
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        const data = await res.json();
        if (!cancelled) {
          setIncidents(Array.isArray(data) ? data : []);
          setIsLoading(false);
        }
      } catch (err: unknown) {
        if (!cancelled) {
          console.error('[IncidentView] fetch error:', err);
          setError(err instanceof Error ? err.message : 'Failed to fetch');
          setIncidents([]);
          setIsLoading(false);
        }
      }
    }
    load();
    return () => { cancelled = true; };
  }, []);

  const viewIncidentEvents = useCallback(
    (incident: Incident) => {
      store.setNamespace(incident.namespace);
      store.setMode('timeline');
    },
    [store],
  );

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-16">
        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (error) {
    return (
      <Card className="border-none soft-shadow glass-panel">
        <CardContent className="flex flex-col items-center justify-center py-16 text-muted-foreground">
          <AlertCircle className="h-10 w-10 mb-3 opacity-30 text-destructive" />
          <p className="text-sm font-medium text-destructive">Failed to load incidents</p>
          <p className="text-xs mt-1">{error}</p>
        </CardContent>
      </Card>
    );
  }

  if (!incidents || incidents.length === 0) {
    return (
      <Card className="border-none soft-shadow glass-panel">
        <CardContent className="flex flex-col items-center justify-center py-16 text-muted-foreground">
          <Shield className="h-10 w-10 mb-3 opacity-30" />
          <p className="text-sm font-medium">No incidents detected</p>
          <p className="text-xs mt-1 text-center max-w-md">
            Your cluster is operating normally. Incidents are auto-detected when health score drops significantly or multiple warnings occur.
          </p>
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="space-y-3">
      {incidents.map((incident, i) => (
        <IncidentCard
          key={incident.incident_id}
          incident={incident}
          index={i}
          onViewEvents={() => viewIncidentEvents(incident)}
        />
      ))}
    </div>
  );
}

/* ─── Incident Card ──────────────────────────────────────────────────────── */

function IncidentCard({
  incident,
  index,
  onViewEvents,
}: {
  incident: Incident;
  index: number;
  onViewEvents: () => void;
}) {
  const [expanded, setExpanded] = useState(false);
  const severity = SEVERITY_STYLES[incident.severity] ?? SEVERITY_STYLES.medium;
  const SeverityIcon = severity.icon;
  const statusStyle = STATUS_STYLES[incident.status] ?? STATUS_STYLES.investigating;
  const duration = incident.ended_at
    ? incident.ended_at - incident.started_at
    : Date.now() - incident.started_at;
  const healthDelta =
    incident.health_before != null && incident.health_after != null
      ? Math.round(incident.health_after - incident.health_before)
      : null;

  return (
    <motion.div
      initial={{ opacity: 0, y: 12 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.25, delay: index * 0.05 }}
    >
      <Card className="border-none soft-shadow glass-panel">
        <button
          onClick={() => setExpanded(!expanded)}
          className="w-full text-left"
        >
          <CardHeader className="pb-2">
            <div className="flex items-start gap-3">
              <div className={cn('p-2 rounded-lg', severity.bg)}>
                <SeverityIcon className={cn('h-5 w-5', severity.text)} />
              </div>
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2 flex-wrap">
                  <CardTitle className="text-sm font-semibold">
                    {incident.root_cause_kind}/{incident.root_cause_name}
                  </CardTitle>
                  <Badge variant="outline" className={cn('text-[10px] px-1.5 py-0 h-5', statusStyle)}>
                    {incident.status}
                  </Badge>
                  <Badge variant="outline" className={cn('text-[10px] px-1.5 py-0 h-5', severity.bg, severity.text)}>
                    {incident.severity}
                  </Badge>
                </div>
                <div className="flex items-center gap-3 mt-1 text-xs text-muted-foreground">
                  <span className="flex items-center gap-1">
                    <Clock className="h-3 w-3" />
                    {formatTime(incident.started_at)}
                  </span>
                  <span>Duration: {formatDuration(duration)}</span>
                  <span>{incident.namespace}</span>
                  {healthDelta != null && (
                    <span className={cn('font-medium', healthDelta < 0 ? 'text-red-500' : 'text-green-500')}>
                      Health: {healthDelta > 0 ? '+' : ''}{healthDelta}
                    </span>
                  )}
                </div>
              </div>
              <ChevronRight
                className={cn(
                  'h-4 w-4 text-muted-foreground/50 transition-transform shrink-0 mt-1',
                  expanded && 'rotate-90',
                )}
              />
            </div>
          </CardHeader>
        </button>

        <AnimatePresence>
          {expanded && (
            <motion.div
              initial={{ height: 0, opacity: 0 }}
              animate={{ height: 'auto', opacity: 1 }}
              exit={{ height: 0, opacity: 0 }}
              transition={{ duration: 0.2 }}
              className="overflow-hidden"
            >
              <CardContent className="pt-0 space-y-4">
                {/* Root cause summary */}
                <p className="text-sm text-muted-foreground">{incident.root_cause_summary}</p>

                {/* Health metrics */}
                <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
                  {incident.health_before != null && (
                    <MetricBox label="Health Before" value={Math.round(incident.health_before)} />
                  )}
                  {incident.health_lowest != null && (
                    <MetricBox label="Health Lowest" value={Math.round(incident.health_lowest)} className="text-red-500" />
                  )}
                  {incident.health_after != null && (
                    <MetricBox label="Health After" value={Math.round(incident.health_after)} />
                  )}
                  {healthDelta != null && (
                    <MetricBox
                      label="Health Delta"
                      value={healthDelta}
                      prefix={healthDelta > 0 ? '+' : ''}
                      className={healthDelta < 0 ? 'text-red-500' : 'text-green-500'}
                    />
                  )}
                </div>

                {/* TTD / TTR */}
                <div className="flex gap-4 text-xs">
                  {incident.ttd != null && (
                    <span className="text-muted-foreground">
                      Time to Detect: <span className="font-medium text-foreground">{formatDuration(incident.ttd)}</span>
                    </span>
                  )}
                  {incident.ttr != null && (
                    <span className="text-muted-foreground">
                      Time to Resolve: <span className="font-medium text-foreground">{formatDuration(incident.ttr)}</span>
                    </span>
                  )}
                </div>

                {/* Actions */}
                <Button variant="outline" size="sm" className="h-7 text-xs gap-1.5" onClick={onViewEvents}>
                  <Activity className="h-3 w-3" />
                  View Events
                </Button>
              </CardContent>
            </motion.div>
          )}
        </AnimatePresence>
      </Card>
    </motion.div>
  );
}

function MetricBox({
  label,
  value,
  prefix = '',
  className,
}: {
  label: string;
  value: number;
  prefix?: string;
  className?: string;
}) {
  return (
    <div className="bg-muted/50 rounded-lg p-2.5 text-center">
      <p className="text-[10px] text-muted-foreground uppercase tracking-wider">{label}</p>
      <p className={cn('text-lg font-bold font-mono mt-0.5', className)}>
        {prefix}{value}
      </p>
    </div>
  );
}
