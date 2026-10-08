/**
 * StructuredLogRow — renders a parsed JSON log line with collapsed/expanded states.
 * Collapsed: timestamp + level badge + message + top inline fields.
 * Expanded: full field grid with filter buttons.
 */
import { memo, useCallback } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { Copy, Search, ExternalLink } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';
import { toast } from '@/components/ui/sonner';
import type { ParsedLog } from '@/hooks/useLogParser';
import { HighlightedText } from '@/components/logs/HighlightedText';

/* ─── Props ───────────────────────────────────────────────────────────────── */

export interface StructuredLogRowProps {
  log: ParsedLog;
  isExpanded: boolean;
  onToggle: () => void;
  onFilterAdd: (field: string, value: string) => void;
  onNavigateToEvents?: (traceId: string) => void;
  /** Search regex from the toolbar's plain-text search box — previously
   * had zero effect in structured view; now highlights matches the same
   * way the plain log view does. */
  searchRegex?: RegExp | null;
}

/* ─── Helpers ─────────────────────────────────────────────────────────────── */

const LEVEL_STYLES: Record<string, string> = {
  ERROR: 'bg-red-500/20 text-red-600 dark:text-red-300 border-red-500/30',
  WARN: 'bg-amber-500/20 text-amber-600 dark:text-amber-300 border-amber-500/30',
  INFO: 'bg-blue-500/20 text-blue-600 dark:text-blue-300 border-blue-500/30',
  DEBUG: 'bg-slate-500/15 text-slate-500 dark:text-slate-400 border-slate-500/30',
};

const ROW_BG: Record<string, string> = {
  ERROR: 'bg-red-500/[0.04] dark:bg-red-500/[0.06]',
  WARN: 'bg-amber-500/[0.03] dark:bg-amber-500/[0.04]',
};

/** Fields that are "interesting" and should appear inline. */
const INTERESTING_PATTERNS = ['.id', '_id', '.status', '.duration', 'duration', '.error', '.code', 'status', 'code', 'latency', 'method', 'path', 'url'];

function isInteresting(key: string): boolean {
  const lower = key.toLowerCase();
  return INTERESTING_PATTERNS.some((p) => lower.includes(p));
}

function pickTopFields(fields: Record<string, unknown>, max: number): [string, unknown][] {
  const entries = Object.entries(fields);
  const interesting = entries.filter(([k]) => isInteresting(k));
  const rest = entries.filter(([k]) => !isInteresting(k));
  return [...interesting, ...rest].slice(0, max);
}

function flattenValue(val: unknown): string {
  if (val === null || val === undefined) return 'null';
  if (typeof val === 'object') return JSON.stringify(val);
  return String(val);
}

function formatTimestamp(ts: string | undefined): string {
  if (!ts) return '--:--:--';
  try {
    // Handle numeric timestamps (unix seconds or milliseconds)
    if (/^\d+$/.test(ts)) {
      const num = Number(ts);
      const date = new Date(num > 1e12 ? num : num * 1000);
      return date.toLocaleTimeString('en-US', { hour12: false, hour: '2-digit', minute: '2-digit', second: '2-digit' });
    }
    return new Date(ts).toLocaleTimeString('en-US', { hour12: false, hour: '2-digit', minute: '2-digit', second: '2-digit' });
  } catch {
    return ts.slice(11, 19) || '--:--:--';
  }
}

const TRACE_FIELDS = ['trace_id', 'traceId', 'request_id', 'requestId', 'correlation_id', 'correlationId'];

/* ─── Component ───────────────────────────────────────────────────────────── */

export const StructuredLogRow = memo(function StructuredLogRow({
  log,
  isExpanded,
  onToggle,
  onFilterAdd,
  onNavigateToEvents,
  searchRegex = null,
}: StructuredLogRowProps) {
  const levelStyle = LEVEL_STYLES[log.level ?? 'INFO'] ?? LEVEL_STYLES.INFO;
  const rowBg = ROW_BG[log.level ?? ''] ?? '';

  const handleCopyJson = useCallback(() => {
    navigator.clipboard.writeText(log.raw);
    toast.success('JSON copied');
  }, [log.raw]);

  const handleFieldClick = useCallback(
    (field: string, value: unknown) => {
      onFilterAdd(field, flattenValue(value));
    },
    [onFilterAdd],
  );

  const topFields = pickTopFields(log.fields, 3);

  return (
    <div className={cn('group border-b border-transparent hover:border-border/30 transition-colors', rowBg)}>
      {/* ── Collapsed row ──────────────────────────────────────────────── */}
      <div
        className="flex items-center gap-2 px-3 py-1 cursor-pointer select-none"
        onClick={onToggle}
        role="button"
        tabIndex={0}
        onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); onToggle(); } }}
      >
        {/* Timestamp */}
        <span className="shrink-0 tabular-nums text-[11px] text-muted-foreground/60 font-mono min-w-[56px]">
          {formatTimestamp(log.timestamp)}
        </span>

        {/* Level badge */}
        <Badge
          className={cn(
            'shrink-0 text-[10px] font-bold tracking-wider px-1.5 py-0 h-5 border',
            levelStyle,
          )}
        >
          {log.level ?? 'INFO'}
        </Badge>

        {/* Message */}
        <span className="flex-1 min-w-0 text-[12px] font-mono truncate text-foreground/90">
          <HighlightedText text={log.message ?? log.raw} regex={searchRegex} />
        </span>

        {/* Inline fields */}
        {topFields.map(([key, val]) => (
          <button
            key={key}
            className="shrink-0 text-[11px] font-mono px-1.5 py-0.5 rounded bg-muted/50 hover:bg-muted transition-colors max-w-[180px] truncate"
            onClick={(e) => {
              e.stopPropagation();
              handleFieldClick(key, val);
            }}
            title={`Filter: ${key}=${flattenValue(val)}`}
          >
            <span className="text-muted-foreground">{key}=</span>
            <span className="text-foreground">{flattenValue(val)}</span>
          </button>
        ))}
      </div>

      {/* ── Expanded detail ────────────────────────────────────────────── */}
      <AnimatePresence>
        {isExpanded && (
          <motion.div
            initial={{ height: 0, opacity: 0 }}
            animate={{ height: 'auto', opacity: 1 }}
            exit={{ height: 0, opacity: 0 }}
            transition={{ duration: 0.2, ease: 'easeOut' }}
            className="overflow-hidden"
          >
            <div className="px-6 py-3 border-t border-border/20 bg-muted/30">
              {/* Field grid */}
              <div className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1.5 text-[12px] font-mono max-w-2xl">
                {Object.entries(log.fields).map(([key, val]) => {
                  const isTrace = TRACE_FIELDS.includes(key);
                  return (
                    <div key={key} className="contents">
                      <span className="text-muted-foreground truncate">{key}</span>
                      <span className="flex items-center gap-1.5 min-w-0">
                        <span className="truncate text-foreground">{flattenValue(val)}</span>
                        <button
                          className="shrink-0 p-0.5 rounded hover:bg-muted transition-colors opacity-0 group-hover:opacity-100"
                          onClick={() => handleFieldClick(key, val)}
                          title={`Filter by ${key}`}
                        >
                          <Search className="h-3 w-3 text-muted-foreground" />
                        </button>
                        {isTrace && onNavigateToEvents && (
                          <button
                            className="shrink-0 p-0.5 rounded hover:bg-muted transition-colors text-primary/70 hover:text-primary"
                            onClick={() => onNavigateToEvents(flattenValue(val))}
                            title="View in Events"
                          >
                            <ExternalLink className="h-3 w-3" />
                          </button>
                        )}
                      </span>
                    </div>
                  );
                })}
              </div>

              {/* Actions */}
              <div className="flex items-center gap-2 mt-3 pt-2 border-t border-border/20">
                <Button variant="ghost" size="sm" className="h-7 text-xs gap-1.5" onClick={handleCopyJson}>
                  <Copy className="h-3 w-3" /> Copy JSON
                </Button>
              </div>
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
});
