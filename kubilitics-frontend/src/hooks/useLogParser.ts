/**
 * Hook that processes raw log lines into structured data.
 * Detects JSON-structured logs, extracts fields, and provides facet metadata.
 */
import { useMemo, useRef } from 'react';

/* ─── Types ───────────────────────────────────────────────────────────────── */

export interface ParsedLog {
  raw: string;
  index: number;
  isStructured: boolean;
  timestamp?: string;
  level?: string; // ERROR, WARN, INFO, DEBUG
  message?: string;
  fields: Record<string, unknown>;
}

export interface FieldInfo {
  name: string;
  uniqueValues: number;
  topValues: { value: string; count: number }[];
}

interface UseLogParserResult {
  parsedLogs: ParsedLog[];
  detectedFields: FieldInfo[];
  isStructured: boolean;
}

/* ─── Field extraction helpers ────────────────────────────────────────────── */

const TIMESTAMP_KEYS = ['timestamp', 'time', 'ts', '@timestamp'];
const LEVEL_KEYS = ['level', 'severity', 'lvl'];
const MESSAGE_KEYS = ['msg', 'message', 'text'];

function normalizeLevel(raw: unknown): string | undefined {
  if (typeof raw !== 'string') return undefined;
  const upper = raw.toUpperCase();
  switch (upper) {
    case 'ERROR':
    case 'ERR':
    case 'FATAL':
    case 'CRITICAL':
      return 'ERROR';
    case 'WARN':
    case 'WARNING':
      return 'WARN';
    case 'INFO':
    case 'INFORMATION':
      return 'INFO';
    case 'DEBUG':
    case 'TRACE':
      return 'DEBUG';
    default:
      return upper;
  }
}

function findField(obj: Record<string, unknown>, candidates: string[]): unknown | undefined {
  for (const key of candidates) {
    if (key in obj) return obj[key];
  }
  return undefined;
}

function parseSingleLine(line: string, index: number): ParsedLog {
  const trimmed = line.trim();
  if (!trimmed) {
    return { raw: line, index, isStructured: false, fields: {} };
  }

  // Attempt JSON parse
  if (trimmed.startsWith('{')) {
    try {
      const parsed = JSON.parse(trimmed) as Record<string, unknown>;

      const tsRaw = findField(parsed, TIMESTAMP_KEYS);
      const timestamp = typeof tsRaw === 'string' || typeof tsRaw === 'number'
        ? String(tsRaw)
        : undefined;

      const level = normalizeLevel(findField(parsed, LEVEL_KEYS));
      const msgRaw = findField(parsed, MESSAGE_KEYS);
      const message = typeof msgRaw === 'string' ? msgRaw : undefined;

      // Collect remaining fields (exclude timestamp/level/message keys)
      const excludeKeys = new Set([...TIMESTAMP_KEYS, ...LEVEL_KEYS, ...MESSAGE_KEYS]);
      const fields: Record<string, unknown> = {};
      for (const [k, v] of Object.entries(parsed)) {
        if (!excludeKeys.has(k)) {
          fields[k] = v;
        }
      }

      // Consider it structured if it has at least level or message
      const isStructured = !!(level || message);

      return { raw: line, index, isStructured, timestamp, level, message, fields };
    } catch {
      // Not valid JSON
    }
  }

  return { raw: line, index, isStructured: false, fields: {} };
}

/* ─── Field aggregation ───────────────────────────────────────────────────── */

function flattenValue(val: unknown): string {
  if (val === null || val === undefined) return 'null';
  if (typeof val === 'object') return JSON.stringify(val);
  return String(val);
}

function buildFieldInfo(parsedLogs: ParsedLog[]): FieldInfo[] {
  const fieldMap = new Map<string, Map<string, number>>();

  for (const log of parsedLogs) {
    if (!log.isStructured) continue;
    for (const [key, val] of Object.entries(log.fields)) {
      let valueMap = fieldMap.get(key);
      if (!valueMap) {
        valueMap = new Map<string, number>();
        fieldMap.set(key, valueMap);
      }
      const strVal = flattenValue(val);
      valueMap.set(strVal, (valueMap.get(strVal) ?? 0) + 1);
    }

    // Also include level as a pseudo-field
    if (log.level) {
      let valueMap = fieldMap.get('level');
      if (!valueMap) {
        valueMap = new Map<string, number>();
        fieldMap.set('level', valueMap);
      }
      valueMap.set(log.level, (valueMap.get(log.level) ?? 0) + 1);
    }
  }

  const fields: FieldInfo[] = [];
  for (const [name, valueMap] of fieldMap) {
    const sorted = Array.from(valueMap.entries())
      .sort((a, b) => b[1] - a[1])
      .slice(0, 5)
      .map(([value, count]) => ({ value, count }));

    fields.push({
      name,
      uniqueValues: valueMap.size,
      topValues: sorted,
    });
  }

  // Sort by unique values ascending (fewer = more useful for faceting)
  fields.sort((a, b) => a.uniqueValues - b.uniqueValues);

  return fields;
}

/* ─── Hook ────────────────────────────────────────────────────────────────── */

export function useLogParser(rawLines: string[]): UseLogParserResult {
  // Cache to avoid re-parsing unchanged lines
  const cacheRef = useRef<{ lines: string[]; parsed: ParsedLog[] }>({
    lines: [],
    parsed: [],
  });

  const parsedLogs = useMemo(() => {
    const cache = cacheRef.current;

    // If exact same reference, return cached
    if (rawLines === cache.lines) return cache.parsed;

    // Incremental: reuse already-parsed lines from the start
    const result: ParsedLog[] = [];
    const parseLimit = rawLines.length; // parse all (visible window handled by virtualizer)

    for (let i = 0; i < parseLimit; i++) {
      if (i < cache.parsed.length && rawLines[i] === cache.lines[i]) {
        // Reuse cached parse, but update index
        result.push(cache.parsed[i]);
      } else {
        result.push(parseSingleLine(rawLines[i], i));
      }
    }

    cache.lines = rawLines;
    cache.parsed = result;
    return result;
  }, [rawLines]);

  const isStructured = useMemo(() => {
    if (parsedLogs.length === 0) return false;
    const structuredCount = parsedLogs.filter((l) => l.isStructured).length;
    return structuredCount / parsedLogs.length > 0.5;
  }, [parsedLogs]);

  const detectedFields = useMemo(() => {
    if (!isStructured) return [];
    return buildFieldInfo(parsedLogs);
  }, [parsedLogs, isStructured]);

  return { parsedLogs, detectedFields, isStructured };
}
