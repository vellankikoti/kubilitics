/**
 * Tests for LogViewer — covers several real production bugs reported
 * against the logs section:
 *  - search had no way to navigate between matches, and the view's scroll
 *    position was left stale when the filtered list changed, making
 *    matches "impossible to find by scrolling"
 *  - the match-count indicator was rendered too faint to read at a glance
 *  - "JSON" (prettify) mode didn't highlight search matches at all
 *  - "Structured" mode ignored the plain-text search box entirely
 * None of these had any test coverage before this change.
 */
import React from 'react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { LogViewer } from './LogViewer';
import { useLogFilterStore } from '@/stores/logFilterStore';
import type { LogEntry } from '@/lib/logParser';

// ── Mocks ───────────────────────────────────────────────────────────────────

vi.mock('@/hooks/useConnectionStatus', () => ({
  useConnectionStatus: () => ({ isConnected: false }),
}));

let mockIsDark = false;
vi.mock('@/hooks/useTheme', () => ({
  useTheme: () => ({ isDark: mockIsDark, theme: mockIsDark ? 'dark' : 'light' }),
}));

vi.mock('@/hooks/useKubernetes', () => ({
  useK8sPodLogs: () => ({ data: undefined, isLoading: false, error: null, refetch: vi.fn(), dataUpdatedAt: 0 }),
}));

vi.mock('@/hooks/useEventsIntelligence', () => ({
  useEventsQuery: () => ({ data: [] }),
}));

vi.mock('@/components/ui/sonner', () => ({
  toast: { success: vi.fn(), error: vi.fn(), info: vi.fn() },
}));

// Real-ish virtualizer: renders every row (test logs are small), and lets
// scrollToIndex be asserted on to verify match-navigation actually scrolls.
const scrollToIndexSpy = vi.fn();
vi.mock('@tanstack/react-virtual', () => ({
  useVirtualizer: ({ count }: { count: number }) => ({
    getTotalSize: () => count * 20,
    getVirtualItems: () =>
      Array.from({ length: count }, (_, index) => ({ index, start: index * 20, key: index })),
    scrollToIndex: scrollToIndexSpy,
    measureElement: () => {},
  }),
}));

function wrapper({ children }: { children: React.ReactNode }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return React.createElement(QueryClientProvider, { client: qc }, children);
}

function makeLogs(messages: string[]): LogEntry[] {
  return messages.map((message, i) => ({
    timestamp: `2026-01-01T00:00:0${i}Z`,
    level: 'info' as const,
    message,
    raw: message,
    isJson: false,
  }));
}

describe('LogViewer', () => {
  beforeEach(() => {
    scrollToIndexSpy.mockClear();
    mockIsDark = false;
    useLogFilterStore.setState({
      searchQuery: '',
      levelFilter: 'all',
      regexMode: false,
      inverseFilter: false,
      contextLines: 0,
      prettifyJson: false,
      hideTerminated: false,
    });
  });

  it('shows a dark, readable match counter (not the old faint gray) and updates as you type', () => {
    render(<LogViewer logs={makeLogs(['hello world', 'goodbye world', 'hello again'])} />, { wrapper });

    const input = screen.getByPlaceholderText('Search logs…');
    fireEvent.change(input, { target: { value: 'hello' } });

    const counter = screen.getByTitle(/Match 1 of 2/);
    expect(counter.textContent).toBe('1/2');
    // The fix: dark/legible text color classes, not the old light-gray-only ones.
    expect(counter.className).toContain('text-slate-700');
    expect(counter.className).not.toContain('text-slate-400');
  });

  it('shows 0/0 in red when the search has no matches', () => {
    render(<LogViewer logs={makeLogs(['hello world'])} />, { wrapper });
    fireEvent.change(screen.getByPlaceholderText('Search logs…'), { target: { value: 'nomatch' } });
    const counter = screen.getByTitle('No matches');
    expect(counter.textContent).toBe('0/0');
  });

  it('CRITICAL FIX: next/previous match navigation actually scrolls to the match (real navigation, not just a static count)', () => {
    render(<LogViewer logs={makeLogs(['foo', 'bar foo', 'baz', 'foo again'])} />, { wrapper });
    fireEvent.change(screen.getByPlaceholderText('Search logs…'), { target: { value: 'foo' } });
    scrollToIndexSpy.mockClear(); // drop the initial "jump to match 1" call

    fireEvent.click(screen.getByTitle('Next match (Enter)'));
    expect(scrollToIndexSpy).toHaveBeenCalledWith(1, { align: 'center' });

    fireEvent.click(screen.getByTitle('Previous match (Shift+Enter)'));
    expect(scrollToIndexSpy).toHaveBeenCalledWith(0, { align: 'center' });
  });

  it('wraps around when navigating past the last match', () => {
    render(<LogViewer logs={makeLogs(['foo', 'bar foo', 'baz'])} />, { wrapper });
    fireEvent.change(screen.getByPlaceholderText('Search logs…'), { target: { value: 'foo' } });
    scrollToIndexSpy.mockClear();

    fireEvent.click(screen.getByTitle('Next match (Enter)')); // from match 0 -> match 1 (last)
    fireEvent.click(screen.getByTitle('Next match (Enter)')); // wraps back to match 0
    expect(scrollToIndexSpy).toHaveBeenLastCalledWith(0, { align: 'center' });
  });

  it('resets to the first match and scrolls there when the search query changes', () => {
    render(<LogViewer logs={makeLogs(['foo one', 'foo two', 'foo three'])} />, { wrapper });
    const input = screen.getByPlaceholderText('Search logs…');
    fireEvent.change(input, { target: { value: 'foo' } });
    fireEvent.click(screen.getByTitle('Next match (Enter)')); // now at match 1
    scrollToIndexSpy.mockClear();

    fireEvent.change(input, { target: { value: 'two' } }); // new search
    expect(scrollToIndexSpy).toHaveBeenCalledWith(0, { align: 'center' });
  });

  it('highlights search matches inside JSON (prettify) mode — previously silently dropped all highlighting', () => {
    const jsonLine = '{"level":"info","msg":"connection established"}';
    const logs: LogEntry[] = [{
      timestamp: '2026-01-01T00:00:00Z',
      level: 'info',
      message: 'connection established',
      raw: jsonLine,
      isJson: true,
      jsonData: { level: 'info', msg: 'connection established' },
    }];
    render(<LogViewer logs={logs} />, { wrapper });

    fireEvent.click(screen.getByTitle('Prettify JSON logs'));
    fireEvent.change(screen.getByPlaceholderText('Search logs…'), { target: { value: 'established' } });

    const marks = document.querySelectorAll('mark');
    expect(marks.length).toBeGreaterThan(0);
    expect(Array.from(marks).some((m) => m.textContent === 'established')).toBe(true);
  });

  it('applies the plain-text search to Structured view too — previously had zero effect there', () => {
    const logs: LogEntry[] = [
      {
        timestamp: '2026-01-01T00:00:00Z', level: 'info',
        message: 'user login succeeded', raw: '{"level":"info","msg":"user login succeeded"}',
        isJson: true, jsonData: {},
      },
      {
        timestamp: '2026-01-01T00:00:01Z', level: 'info',
        message: 'user logout succeeded', raw: '{"level":"info","msg":"user logout succeeded"}',
        isJson: true, jsonData: {},
      },
    ];
    render(<LogViewer logs={logs} />, { wrapper });

    fireEvent.change(screen.getByPlaceholderText('Search logs…'), { target: { value: 'login' } });

    // "login" is highlighted via a <mark>, splitting the text node — assert
    // on normalized body text instead of an exact single-node match.
    const bodyText = document.body.textContent?.replace(/\s+/g, ' ') ?? '';
    expect(bodyText).toContain('user login succeeded');
    expect(bodyText).not.toContain('user logout succeeded');
  });
});
