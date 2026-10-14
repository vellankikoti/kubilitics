/**
 * Shared timestamp formatting for metrics charts (MetricsDashboard and any
 * future chart consumer). Replaces two independent inline `toLocaleTimeString`
 * calls that both dropped the date unconditionally — fine up to 6h, actively
 * misleading at 24h/7d where multiple calendar days collapse onto bare times
 * like "03:45:00 PM" with no way to tell which day a point belongs to.
 */

/** Ranges whose axis ticks need a date, not just a time, to stay unambiguous. */
const DATE_AWARE_RANGES = new Set(['24h', '168h']);

/**
 * Axis tick label for a chart point. Time-only below 24h (everything in view
 * is "today"); date+time once the range can plausibly span multiple days.
 */
export function formatAxisTick(tsSeconds: number, rangeValue: string): string {
  const date = new Date(tsSeconds * 1000);
  if (DATE_AWARE_RANGES.has(rangeValue)) {
    return date.toLocaleString(undefined, {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  }
  return date.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit', second: '2-digit' });
}

/**
 * Tooltip label for a chart point: always the full absolute date and time,
 * regardless of selected range, so hovering never leaves the day ambiguous.
 */
export function formatTooltipTimestamp(tsSeconds: number): string {
  const date = new Date(tsSeconds * 1000);
  return date.toLocaleString(undefined, {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  });
}

/**
 * "Oct 8 – Oct 14, 2026"-style banner text for display under a chart title
 * when its visible points span more than one calendar day — oriented without
 * needing to hover. Returns null when everything falls on a single day (or
 * there are fewer than 2 points), so callers can omit the banner entirely.
 */
export function getDateRangeLabel(points: { ts: number }[]): string | null {
  if (points.length < 2) return null;
  const first = new Date(points[0].ts * 1000);
  const last = new Date(points[points.length - 1].ts * 1000);
  const sameDay =
    first.getFullYear() === last.getFullYear() &&
    first.getMonth() === last.getMonth() &&
    first.getDate() === last.getDate();
  if (sameDay) return null;
  const fmt = (d: Date) => d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
  const year = last.getFullYear();
  return `${fmt(first)} – ${fmt(last)}, ${year}`;
}
