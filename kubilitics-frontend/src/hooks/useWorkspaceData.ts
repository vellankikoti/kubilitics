/**
 * Orchestration hook for the Intelligence Workspace.
 * Combines topology, blast radius, and pre-apply preview data into one surface.
 */
import { useState, useMemo, useCallback } from 'react';
import { useBlastRadius } from '@/hooks/useBlastRadius';
import { useResourceTopology } from '@/hooks/useResourceTopology';
import { usePreApplyBlastRadius } from '@/hooks/usePreApplyBlastRadius';
import type { TopologyGraph } from '@/topology/graph';
import type { PreviewResult } from '@/services/api/preview';
import type { BlastRadiusResult } from '@/services/api/types';

export type WorkspaceMode = 'live' | 'preview';

export interface WorkspaceData {
  // Mode
  mode: WorkspaceMode;
  setMode: (mode: WorkspaceMode) => void;

  // Topology
  graph: TopologyGraph | undefined;
  topoLoading: boolean;
  topoError: Error | null;

  // Blast radius (live mode)
  blastData: BlastRadiusResult | undefined;
  blastLoading: boolean;
  blastError: Error | null;
  isGraphReady: boolean;
  failureMode: string;
  setFailureMode: (mode: string) => void;

  // Preview (change mode)
  previewData: PreviewResult | undefined;
  previewLoading: boolean;
  previewError: Error | null;
  analyzeManifest: (yaml: string) => void;
  clearPreview: () => void;
  manifestFilename: string | null;
  setManifestFilename: (name: string | null) => void;

  // Depth control
  depth: number;
  setDepth: (d: number) => void;

  // Diff overlay for preview mode (maps to TopologyCanvas simulationDiff prop)
  previewDiff: {
    removed: Set<string>;
    added: Set<string>;
    modified: Set<string>;
    newSpofs: Set<string>;
  } | null;
}

function getDefaultFailureMode(kind: string): string {
  switch (kind.toLowerCase()) {
    case 'pod': return 'pod-crash';
    case 'namespace': return 'namespace-deletion';
    default: return 'workload-deletion';
  }
}

function buildPreviewDiff(preview: PreviewResult | undefined) {
  if (!preview) return null;
  const added = new Set<string>();
  const modified = new Set<string>();
  const removed = new Set<string>();
  const newSpofs = new Set<string>();

  for (const r of preview.affected_resources) {
    const key = `${r.kind}/${r.namespace}/${r.name}`;
    switch (r.impact) {
      case 'created': added.add(key); break;
      case 'modified': modified.add(key); break;
      case 'deleted': removed.add(key); break;
    }
  }
  for (const s of preview.new_spofs) {
    newSpofs.add(`${s.kind}/${s.namespace}/${s.name}`);
  }

  return { added, modified, removed, newSpofs };
}

export function useWorkspaceData(
  kind: string,
  namespace: string,
  name: string,
  enabled = true,
): WorkspaceData {
  const [mode, setMode] = useState<WorkspaceMode>('live');
  const [failureMode, setFailureMode] = useState(() => getDefaultFailureMode(kind));
  const [depth, setDepth] = useState(2);
  const [manifestFilename, setManifestFilename] = useState<string | null>(null);

  // Topology graph
  const { graph, isLoading: topoLoading, error: topoError } = useResourceTopology({
    kind,
    namespace,
    name,
    enabled,
    depth,
  });

  // Blast radius (always fetched; consumer uses live mode to decide visibility)
  const {
    data: blastData,
    isLoading: blastLoading,
    error: blastError,
    isGraphReady,
  } = useBlastRadius({ kind, namespace, name, enabled, failureMode });

  // Pre-apply change preview
  const {
    analyze,
    data: previewData,
    isLoading: previewLoading,
    error: previewError,
    reset: resetPreview,
  } = usePreApplyBlastRadius();

  const analyzeManifest = useCallback((yaml: string) => {
    analyze(yaml);
    setMode('preview');
  }, [analyze]);

  const clearPreview = useCallback(() => {
    resetPreview();
    setMode('live');
    setManifestFilename(null);
  }, [resetPreview]);

  const previewDiff = useMemo(() => buildPreviewDiff(previewData), [previewData]);

  return {
    mode,
    setMode,
    graph,
    topoLoading,
    topoError,
    blastData,
    blastLoading,
    blastError,
    isGraphReady,
    failureMode,
    setFailureMode,
    previewData,
    previewLoading,
    previewError,
    analyzeManifest,
    clearPreview,
    manifestFilename,
    setManifestFilename,
    depth,
    setDepth,
    previewDiff,
  };
}
