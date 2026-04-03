/**
 * SimulationPage — What-If Simulation Engine page.
 *
 * Three-panel layout:
 * - Left: ScenarioList (scenario cards)
 * - Center: Topology canvas with simulation diff overlay
 * - Right: ImpactSummary (health delta, SPOF changes, affected services)
 * - Bottom drawer: DiffBreakdown (tabs: Removed/Modified/Added/Edges Lost)
 */
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { toast } from 'sonner';
import { FlaskConical } from 'lucide-react';
import { TopologyCanvas } from '@/topology/TopologyCanvas';
import { useTopologyData } from '@/topology/hooks/useTopologyData';
import { useActiveClusterId } from '@/hooks/useActiveClusterId';
import { useSimulationStore } from '@/stores/simulationStore';
import { useRunSimulation } from '@/hooks/useSimulation';
import SimulationToolbar from '@/components/simulation/SimulationToolbar';
import ScenarioList from '@/components/simulation/ScenarioList';
import ImpactSummary from '@/components/simulation/ImpactSummary';
import DiffBreakdown from '@/components/simulation/DiffBreakdown';
import type { SimulationResult } from '@/services/api/simulation';

/** Build simulation diff overlay from result for TopologyCanvas */
function buildSimulationDiff(result: SimulationResult | null): {
  removed: Set<string>;
  added: Set<string>;
  modified: Set<string>;
  newSpofs: Set<string>;
} | null {
  if (!result) return null;
  const removed = new Set<string>();
  const added = new Set<string>();
  const modified = new Set<string>();
  const newSpofs = new Set<string>();

  for (const n of result.removed_nodes) {
    removed.add(n.key);
  }
  for (const n of result.added_nodes) {
    added.add(n.key);
  }
  for (const n of result.modified_nodes) {
    modified.add(n.key);
  }
  for (const spof of result.new_spofs) {
    newSpofs.add(spof.key);
  }

  return { removed, added, modified, newSpofs };
}

/** Build affected nodes set from result for canvas overlay */
function buildAffectedNodes(result: SimulationResult | null): Set<string> | null {
  if (!result) return null;
  const affected = new Set<string>();

  for (const n of result.removed_nodes) affected.add(n.key);
  for (const n of result.modified_nodes) affected.add(n.key);
  for (const n of result.added_nodes) affected.add(n.key);
  for (const spof of result.new_spofs) affected.add(spof.key);

  return affected.size > 0 ? affected : null;
}

export default function SimulationPage() {
  const clusterId = useActiveClusterId();
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null);

  const { scenarios, result, isRunning, error, autoRun, setResult, setRunning, setError } = useSimulationStore();
  const runMutation = useRunSimulation();

  // Load topology data (same as Topology page)
  const { topology, allNamespaces, isLoading: isTopoLoading } = useTopologyData({
    clusterId,
    viewMode: 'namespace',
    depth: 1,
    enabled: !!clusterId,
  });

  // Extract node names from topology for the toolbar selector
  const nodeNames = useMemo(() => {
    if (!topology?.nodes) return [];
    return topology.nodes
      .filter((n) => n.kind === 'Node')
      .map((n) => n.name)
      .sort();
  }, [topology]);

  // Extract resource keys (Kind/Namespace/Name) for the delete_resource selector
  const resourceKeys = useMemo(() => {
    if (!topology?.nodes) return [];
    return topology.nodes
      .filter((n) => n.kind !== 'Node' && n.kind !== 'Namespace')
      .map((n) => n.id || `${n.kind}/${n.namespace}/${n.name}`)
      .sort();
  }, [topology]);

  // Run simulation handler
  const handleRunSimulation = useCallback(async () => {
    if (!clusterId || scenarios.length === 0) return;

    setRunning(true);
    setError(null);

    try {
      const simResult = await runMutation.mutateAsync({
        clusterId,
        request: { scenarios },
      });
      setResult(simResult);
      toast.success('Simulation complete');
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Simulation failed';
      setError(message);
      toast.error(message);
    } finally {
      setRunning(false);
    }
  }, [clusterId, scenarios, runMutation, setResult, setRunning, setError]);

  // Auto-run when scenarios change and autoRun is enabled
  const prevScenariosLen = useRef(scenarios.length);
  useEffect(() => {
    if (autoRun && scenarios.length > 0 && scenarios.length !== prevScenariosLen.current) {
      handleRunSimulation();
    }
    prevScenariosLen.current = scenarios.length;
  }, [autoRun, scenarios.length, handleRunSimulation]);

  // Build overlay props for canvas
  const simulationDiff = useMemo(() => buildSimulationDiff(result), [result]);
  const simulationAffectedNodes = useMemo(() => buildAffectedNodes(result), [result]);

  // No cluster connected
  if (!clusterId) {
    return (
      <div className="flex flex-col items-center justify-center h-full text-center px-4">
        <FlaskConical className="h-12 w-12 text-slate-300 dark:text-slate-600 mb-4" />
        <h2 className="text-lg font-semibold text-slate-700 dark:text-slate-200 mb-2">
          What-If Simulation
        </h2>
        <p className="text-sm text-slate-500 dark:text-slate-400">
          Connect a cluster to start simulating failure scenarios.
        </p>
      </div>
    );
  }

  return (
    <div className="flex flex-col h-full overflow-hidden">
      {/* Toolbar */}
      <div className="flex-shrink-0 px-3 pt-3 pb-2">
        <SimulationToolbar
          onRunSimulation={handleRunSimulation}
          isRunning={isRunning}
          nodeNames={nodeNames}
          namespaces={allNamespaces}
          resourceKeys={resourceKeys}
        />
      </div>

      {/* Three-panel layout */}
      <div className="flex-1 flex min-h-0">
        {/* Left panel: Scenario List */}
        <div className="w-64 flex-shrink-0 border-r border-slate-200 dark:border-slate-700 bg-slate-50/50 dark:bg-slate-950/50 overflow-hidden">
          <div className="flex items-center gap-2 px-3 py-2 border-b border-slate-200 dark:border-slate-700">
            <FlaskConical className="h-4 w-4 text-violet-500" />
            <h2 className="text-xs font-semibold text-slate-700 dark:text-slate-200 uppercase tracking-wider">
              Scenarios
            </h2>
            {scenarios.length > 0 && (
              <span className="ml-auto text-xs text-slate-500 dark:text-slate-400">
                {scenarios.length}
              </span>
            )}
          </div>
          <ScenarioList />
        </div>

        {/* Center: Topology Canvas */}
        <div className="flex-1 relative min-w-0">
          {isTopoLoading && !topology ? (
            <div className="flex items-center justify-center h-full">
              <div className="flex flex-col items-center gap-2">
                <div className="h-8 w-8 rounded-full border-2 border-blue-500 border-t-transparent animate-spin" />
                <p className="text-sm text-slate-500 dark:text-slate-400">Loading topology...</p>
              </div>
            </div>
          ) : (
            <TopologyCanvas
              topology={topology}
              selectedNodeId={selectedNodeId}
              onSelectNode={setSelectedNodeId}
              simulationAffectedNodes={simulationAffectedNodes}
              simulationDiff={simulationDiff}
            />
          )}

          {/* Error banner */}
          {error && (
            <div className="absolute top-3 left-1/2 -translate-x-1/2 z-20 bg-red-50 dark:bg-red-900/50 border border-red-200 dark:border-red-800 text-red-700 dark:text-red-300 rounded-lg px-4 py-2 text-sm shadow-lg">
              {error}
            </div>
          )}
        </div>

        {/* Right panel: Impact Summary */}
        <div className="w-72 flex-shrink-0 border-l border-slate-200 dark:border-slate-700 bg-slate-50/50 dark:bg-slate-950/50 overflow-hidden">
          <div className="flex items-center gap-2 px-3 py-2 border-b border-slate-200 dark:border-slate-700">
            <h2 className="text-xs font-semibold text-slate-700 dark:text-slate-200 uppercase tracking-wider">
              Impact Analysis
            </h2>
          </div>
          <ImpactSummary result={result} />
        </div>
      </div>

      {/* Bottom drawer: Diff Breakdown */}
      <div className="flex-shrink-0">
        <DiffBreakdown result={result} />
      </div>
    </div>
  );
}
