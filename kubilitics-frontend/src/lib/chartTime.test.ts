import { describe, it, expect } from 'vitest';
import { formatAxisTick, formatTooltipTimestamp, getDateRangeLabel } from './chartTime';

// All timestamps below are fixed epoch seconds so assertions are timezone-stable
// within a single run (Date#toLocale* uses the host's local timezone either way,
// so we assert on *shape*/distinguishing substrings rather than exact strings
// where locale output could vary by CI timezone).

describe('formatAxisTick', () => {
  it('shows time only (no date) for short ranges — 5m/15m/30m/1h/6h', () => {
    const ts = new Date('2026-10-14T15:45:12Z').getTime() / 1000;
    for (const range of ['5m', '15m', '30m', '1h', '6h']) {
      const label = formatAxisTick(ts, range);
      // No day-of-month digit pair followed by a month-ish abbreviation pattern
      expect(label).not.toMatch(/\b(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\b/);
    }
  });

  it('includes the date for long ranges — 24h and 7d (168h)', () => {
    const ts = new Date('2026-10-14T15:45:12Z').getTime() / 1000;
    for (const range of ['24h', '168h']) {
      const label = formatAxisTick(ts, range);
      expect(label).toMatch(/\b(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\b/);
    }
  });
});

describe('formatTooltipTimestamp', () => {
  it('always includes both a date and a time, regardless of range', () => {
    const ts = new Date('2026-10-14T15:45:12Z').getTime() / 1000;
    const label = formatTooltipTimestamp(ts);
    expect(label).toMatch(/\b(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\b/);
    // a time component: digits, colon, digits
    expect(label).toMatch(/\d{1,2}:\d{2}/);
  });
});

describe('getDateRangeLabel', () => {
  it('returns null when all points fall on the same calendar day', () => {
    const base = new Date('2026-10-14T10:00:00Z').getTime() / 1000;
    const points = [base, base + 60, base + 3600].map((ts) => ({ ts }));
    expect(getDateRangeLabel(points)).toBeNull();
  });

  it('returns a "start – end" label when points span multiple calendar days', () => {
    const start = new Date('2026-10-08T10:00:00Z').getTime() / 1000;
    const end = new Date('2026-10-14T10:00:00Z').getTime() / 1000;
    const label = getDateRangeLabel([{ ts: start }, { ts: end }]);
    expect(label).not.toBeNull();
    expect(label).toMatch(/\b(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\b/);
  });

  it('returns null for an empty or single-point list', () => {
    expect(getDateRangeLabel([])).toBeNull();
    expect(getDateRangeLabel([{ ts: Date.now() / 1000 }])).toBeNull();
  });
});
