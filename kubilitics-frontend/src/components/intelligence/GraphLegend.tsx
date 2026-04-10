// src/components/intelligence/GraphLegend.tsx

type WorkspaceMode = 'live' | 'preview';

interface GraphLegendProps {
  mode: WorkspaceMode;
}

export function GraphLegend({ mode }: GraphLegendProps) {
  return (
    <div className="absolute bottom-16 left-4 z-20 bg-black/60 backdrop-blur-sm text-white/80 text-[10px] font-medium rounded-lg px-3 py-2 space-y-0.5 pointer-events-none">
      {mode === 'live' ? (
        <>
          <div className="flex items-center gap-1.5"><span className="h-2 w-2 rounded-full bg-red-500 inline-block" /> Failure / Focus</div>
          <div className="flex items-center gap-1.5"><span className="h-2 w-2 rounded-full bg-orange-500 inline-block" /> Direct impact</div>
          <div className="flex items-center gap-1.5"><span className="h-2 w-2 rounded-full bg-amber-400 inline-block" /> Cascading</div>
          <div className="flex items-center gap-1.5"><span className="h-2 w-2 rounded-full bg-slate-500 inline-block" /> Unaffected</div>
        </>
      ) : (
        <>
          <div className="flex items-center gap-1.5"><span className="h-2 w-2 rounded-full bg-green-500 inline-block" /> New resource</div>
          <div className="flex items-center gap-1.5"><span className="h-2 w-2 rounded-full bg-yellow-500 inline-block" /> Modified</div>
          <div className="flex items-center gap-1.5"><span className="h-2 w-2 rounded-full bg-red-500 inline-block" /> Risk / Broken</div>
          <div className="flex items-center gap-1.5"><span className="h-2 w-2 rounded-sm border border-dashed border-slate-400 inline-block" /> Deleted</div>
        </>
      )}
    </div>
  );
}
