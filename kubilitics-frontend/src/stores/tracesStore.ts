/**
 * Zustand store for Traces UI state.
 * Stores filters, selected trace, and active view mode.
 */
import { create } from 'zustand';

export type TracesMode = 'list' | 'map';

export interface TracesState {
  // Filters
  serviceFilter: string;
  statusFilter: string;
  minDuration: number | null;
  timeRange: string;

  // UI state
  mode: TracesMode;
  selectedTraceId: string | null;
  selectedSpanId: string | null;

  // Actions
  setServiceFilter: (s: string) => void;
  setStatusFilter: (s: string) => void;
  setMinDuration: (d: number | null) => void;
  setTimeRange: (r: string) => void;
  setMode: (m: TracesMode) => void;
  selectTrace: (id: string | null) => void;
  selectSpan: (id: string | null) => void;
  resetFilters: () => void;
}

const DEFAULT_FILTERS = {
  serviceFilter: '',
  statusFilter: '',
  minDuration: null as number | null,
  timeRange: '1h',
};

export const useTracesStore = create<TracesState>()((set) => ({
  ...DEFAULT_FILTERS,
  mode: 'list',
  selectedTraceId: null,
  selectedSpanId: null,

  setServiceFilter: (serviceFilter) => set({ serviceFilter }),
  setStatusFilter: (statusFilter) => set({ statusFilter }),
  setMinDuration: (minDuration) => set({ minDuration }),
  setTimeRange: (timeRange) => set({ timeRange }),
  setMode: (mode) => set({ mode }),
  selectTrace: (id) => set({ selectedTraceId: id, selectedSpanId: null }),
  selectSpan: (id) => set({ selectedSpanId: id }),
  resetFilters: () => set(DEFAULT_FILTERS),
}));
