/**
 * DiffBreakdown — Collapsible bottom section with tabs for simulation diff details.
 *
 * Tabs: Removed | Modified | Added | Edges Lost
 * Each tab has a table: Kind | Namespace | Name | Score Change | Status badge
 *
 * Uses the existing Radix UI Tabs component.
 */
import { useState } from 'react';
import { ChevronDown, ChevronUp, Trash2, Pencil, Plus, Unlink } from 'lucide-react';
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs';
import { cn } from '@/lib/utils';
import type { SimulationResult, NodeInfo, NodeDiff, EdgeInfo } from '@/services/api/simulation';

interface DiffBreakdownProps {
  result: SimulationResult | null;
}

function StatusBadge({ status }: { status: string }) {
  const colorMap: Record<string, string> = {
    healthy: 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/50 dark:text-emerald-300',
    warning: 'bg-amber-100 text-amber-700 dark:bg-amber-900/50 dark:text-amber-300',
    error: 'bg-red-100 text-red-700 dark:bg-red-900/50 dark:text-red-300',
    critical: 'bg-red-100 text-red-700 dark:bg-red-900/50 dark:text-red-300',
    removed: 'bg-muted text-muted-foreground',
    unknown: 'bg-muted text-muted-foreground',
  };

  return (
    <span className={cn(
      "inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium",
      colorMap[status] ?? colorMap.unknown
    )}>
      {status}
    </span>
  );
}

function NodeInfoTable({ nodes, showScore = true }: { nodes: NodeInfo[]; showScore?: boolean }) {
  if (nodes.length === 0) {
    return (
      <p className="text-xs text-muted-foreground text-center py-4">
        None
      </p>
    );
  }

  return (
    <div className="overflow-x-auto">
      <table className="w-full text-xs">
        <thead>
          <tr className="border-b border-border">
            <th className="text-left py-1.5 px-2 font-medium text-muted-foreground">Kind</th>
            <th className="text-left py-1.5 px-2 font-medium text-muted-foreground">Namespace</th>
            <th className="text-left py-1.5 px-2 font-medium text-muted-foreground">Name</th>
            {showScore && (
              <th className="text-right py-1.5 px-2 font-medium text-muted-foreground">Score</th>
            )}
            <th className="text-left py-1.5 px-2 font-medium text-muted-foreground">Status</th>
          </tr>
        </thead>
        <tbody>
          {nodes.map((node) => (
            <tr key={node.key} className="border-b border-border/50 hover:bg-muted/50">
              <td className="py-1.5 px-2 text-foreground/80 font-medium">{node.kind}</td>
              <td className="py-1.5 px-2 text-muted-foreground">{node.namespace || '-'}</td>
              <td className="py-1.5 px-2 text-foreground">{node.name}</td>
              {showScore && (
                <td className="py-1.5 px-2 text-right text-muted-foreground">{node.health_score}</td>
              )}
              <td className="py-1.5 px-2">
                <StatusBadge status={node.status} />
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function NodeDiffTable({ nodes }: { nodes: NodeDiff[] }) {
  if (nodes.length === 0) {
    return (
      <p className="text-xs text-muted-foreground text-center py-4">
        None
      </p>
    );
  }

  return (
    <div className="overflow-x-auto">
      <table className="w-full text-xs">
        <thead>
          <tr className="border-b border-border">
            <th className="text-left py-1.5 px-2 font-medium text-muted-foreground">Kind</th>
            <th className="text-left py-1.5 px-2 font-medium text-muted-foreground">Namespace</th>
            <th className="text-left py-1.5 px-2 font-medium text-muted-foreground">Name</th>
            <th className="text-right py-1.5 px-2 font-medium text-muted-foreground">Score Change</th>
            <th className="text-left py-1.5 px-2 font-medium text-muted-foreground">Status</th>
          </tr>
        </thead>
        <tbody>
          {nodes.map((node) => {
            const delta = node.score_after - node.score_before;
            return (
              <tr key={node.key} className="border-b border-border/50 hover:bg-muted/50">
                <td className="py-1.5 px-2 text-foreground/80 font-medium">{node.kind}</td>
                <td className="py-1.5 px-2 text-muted-foreground">{node.namespace || '-'}</td>
                <td className="py-1.5 px-2 text-foreground">{node.name}</td>
                <td className="py-1.5 px-2 text-right">
                  <span className={cn(
                    "font-semibold",
                    delta > 0 ? "text-emerald-600 dark:text-emerald-400" : delta < 0 ? "text-red-600 dark:text-red-400" : "text-muted-foreground"
                  )}>
                    {node.score_before} -&gt; {node.score_after}
                  </span>
                </td>
                <td className="py-1.5 px-2">
                  <div className="flex items-center gap-1">
                    <StatusBadge status={node.status_before} />
                    <span className="text-muted-foreground">-&gt;</span>
                    <StatusBadge status={node.status_after} />
                  </div>
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}

function EdgeTable({ edges }: { edges: EdgeInfo[] }) {
  if (edges.length === 0) {
    return (
      <p className="text-xs text-muted-foreground text-center py-4">
        None
      </p>
    );
  }

  return (
    <div className="overflow-x-auto">
      <table className="w-full text-xs">
        <thead>
          <tr className="border-b border-border">
            <th className="text-left py-1.5 px-2 font-medium text-muted-foreground">Source</th>
            <th className="text-left py-1.5 px-2 font-medium text-muted-foreground">Target</th>
            <th className="text-left py-1.5 px-2 font-medium text-muted-foreground">Relationship</th>
          </tr>
        </thead>
        <tbody>
          {edges.map((edge, i) => (
            <tr key={`${edge.source}-${edge.target}-${i}`} className="border-b border-border/50 hover:bg-muted/50">
              <td className="py-1.5 px-2 text-foreground truncate max-w-[200px]">{edge.source}</td>
              <td className="py-1.5 px-2 text-foreground truncate max-w-[200px]">{edge.target}</td>
              <td className="py-1.5 px-2 text-muted-foreground">{edge.relationship}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

export default function DiffBreakdown({ result }: DiffBreakdownProps) {
  const [isExpanded, setIsExpanded] = useState(false);

  if (!result) return null;

  const removedCount = result.removed_nodes.length;
  const modifiedCount = result.modified_nodes.length;
  const addedCount = result.added_nodes.length;
  const edgesLostCount = result.lost_edges.length;
  const totalChanges = removedCount + modifiedCount + addedCount + edgesLostCount;

  if (totalChanges === 0) return null;

  return (
    <div className="border-t border-border bg-card">
      {/* Collapse toggle */}
      <button
        type="button"
        onClick={() => setIsExpanded(!isExpanded)}
        className="flex items-center justify-between w-full px-4 py-2 text-sm font-medium text-foreground hover:bg-muted/50 transition-colors"
      >
        <span>
          Diff Breakdown
          <span className="ml-2 text-xs text-muted-foreground">
            ({totalChanges} change{totalChanges !== 1 ? 's' : ''})
          </span>
        </span>
        {isExpanded ? (
          <ChevronDown className="h-4 w-4 text-muted-foreground" />
        ) : (
          <ChevronUp className="h-4 w-4 text-muted-foreground" />
        )}
      </button>

      {/* Expandable content */}
      {isExpanded && (
        <div className="px-4 pb-3">
          <Tabs defaultValue="removed">
            <TabsList>
              <TabsTrigger value="removed" className="gap-1.5">
                <Trash2 className="h-3 w-3" />
                Removed
                {removedCount > 0 && (
                  <span className="ml-1 rounded-full bg-red-100 dark:bg-red-900/50 text-red-700 dark:text-red-300 px-1.5 text-xs font-semibold">
                    {removedCount}
                  </span>
                )}
              </TabsTrigger>
              <TabsTrigger value="modified" className="gap-1.5">
                <Pencil className="h-3 w-3" />
                Modified
                {modifiedCount > 0 && (
                  <span className="ml-1 rounded-full bg-amber-100 dark:bg-amber-900/50 text-amber-700 dark:text-amber-300 px-1.5 text-xs font-semibold">
                    {modifiedCount}
                  </span>
                )}
              </TabsTrigger>
              <TabsTrigger value="added" className="gap-1.5">
                <Plus className="h-3 w-3" />
                Added
                {addedCount > 0 && (
                  <span className="ml-1 rounded-full bg-emerald-100 dark:bg-emerald-900/50 text-emerald-700 dark:text-emerald-300 px-1.5 text-xs font-semibold">
                    {addedCount}
                  </span>
                )}
              </TabsTrigger>
              <TabsTrigger value="edges" className="gap-1.5">
                <Unlink className="h-3 w-3" />
                Edges Lost
                {edgesLostCount > 0 && (
                  <span className="ml-1 rounded-full bg-muted text-muted-foreground px-1.5 text-xs font-semibold">
                    {edgesLostCount}
                  </span>
                )}
              </TabsTrigger>
            </TabsList>

            <TabsContent value="removed">
              <NodeInfoTable nodes={result.removed_nodes} />
            </TabsContent>
            <TabsContent value="modified">
              <NodeDiffTable nodes={result.modified_nodes} />
            </TabsContent>
            <TabsContent value="added">
              <NodeInfoTable nodes={result.added_nodes} />
            </TabsContent>
            <TabsContent value="edges">
              <EdgeTable edges={result.lost_edges} />
            </TabsContent>
          </Tabs>
        </div>
      )}
    </div>
  );
}
