# Blast Radius Frontend v2 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the blast radius UI with new classification-aware components — hybrid score cards, headline banner with failure mode dropdown, score tooltips, detail side sheet, and coverage banner — consuming the v2 backend API with camelCase types.

**Architecture:** Update TypeScript interfaces to camelCase, modify API client + hook to support failure mode selection, then replace/create 5 UI components that consume the new sub-scores, impact classifications, and verdict data. Existing topology canvas, simulation engine, and wave breakdown remain unchanged (minor field renames only).

**Tech Stack:** React 18, TypeScript, Tailwind CSS, shadcn/ui (Tooltip, Sheet, Select, Badge, Collapsible), Framer Motion, Lucide icons, React Query.

**Spec:** `docs/superpowers/specs/2026-04-09-blast-radius-frontend-v2-design.md`

---

## File Structure

| File | Action | Responsibility |
|---|---|---|
| `src/services/api/types.ts` | Modify (lines 371-453) | Rewrite blast radius interfaces to camelCase, add 10 new types |
| `src/services/api/blastRadius.ts` | Modify | Add failureMode + audit query params |
| `src/hooks/useBlastRadius.ts` | Modify | Accept failureMode option, include in query key + API call |
| `src/components/blast-radius/CoverageBanner.tsx` | Create | Persistent amber bar when coverage is partial |
| `src/components/blast-radius/CriticalityBanner.tsx` | Rewrite | Headline banner: score + level + verdict + failure mode dropdown |
| `src/components/blast-radius/RiskIndicatorCards.tsx` | Rewrite | 4 hybrid cards: Resilience, Cluster Impact, Exposure, Recovery |
| `src/components/blast-radius/ScoreTooltip.tsx` | Create | Hover tooltip showing 3-4 scoring factors |
| `src/components/blast-radius/ScoreDetailSheet.tsx` | Create | Side sheet with full scoring breakdown + audit trail |
| `src/components/resources/BlastRadiusTab.tsx` | Modify | Wire new components, add failureMode state |
| `src/components/blast-radius/WaveBreakdown.tsx` | Modify | Rename snake_case field accesses to camelCase |
| `src/components/blast-radius/RiskPanel.tsx` | Modify | Rename snake_case field accesses to camelCase |

---

## Task 1: Update TypeScript Types

**Files:**
- Modify: `src/services/api/types.ts` (lines 371-453)

- [ ] **Step 1: Read the current blast radius types section**

Read `src/services/api/types.ts` lines 370-454 to see the exact current interfaces.

- [ ] **Step 2: Replace all blast radius interfaces with camelCase versions**

Replace the entire section from `ResourceRef` (line 371) through `BlastRadiusSummaryEntry` (line 453) with the new camelCase interfaces. Also add all new types.

The full replacement block (insert at the same location):

```typescript
// ---- Blast Radius Types (v2 — camelCase) ----

export interface ResourceRef {
  kind: string;
  name: string;
  namespace: string;
}

export interface ScoringFactor {
  name: string;
  value: string;
  effect: number;
  note: string;
}

export interface SubScoreDetail {
  score: number;
  factors: ScoringFactor[];
  source?: string;
  confidence?: string;
}

export interface SubScores {
  resilience: SubScoreDetail;
  exposure: SubScoreDetail;
  recovery: SubScoreDetail;
  impact: SubScoreDetail;
}

export interface ScoreBreakdown {
  resilience: SubScoreDetail;
  exposure: SubScoreDetail;
  recovery: SubScoreDetail;
  impact: SubScoreDetail;
  overall: number;
  level: string;
}

export interface ImpactSummary {
  brokenCount: number;
  degradedCount: number;
  selfHealingCount: number;
  totalWorkloads: number;
  capacityNotes: string[];
}

export interface ServiceImpact {
  service: ResourceRef;
  classification: 'broken' | 'degraded' | 'self-healing';
  totalEndpoints: number;
  remainingEndpoints: number;
  threshold: number;
  thresholdSource: string;
  note: string;
}

export interface IngressImpact {
  ingress: ResourceRef;
  classification: string;
  host: string;
  backendService: string;
  note: string;
}

export interface ConsumerImpact {
  workload: ResourceRef;
  classification: string;
  dependsOn: string;
  note: string;
}

export interface ServiceImpactAudit {
  service: string;
  totalEndpoints: number;
  lostEndpoints: number;
  remainingPercent: number;
  threshold: number;
  thresholdSource: string;
  classification: string;
}

export interface AuditTrail {
  timestamp: string;
  targetResource: ResourceRef;
  failureMode: string;
  graphStalenessMs: number;
  traceDataAgeMs?: number;
  lostPods: ResourceRef[];
  serviceImpacts: ServiceImpactAudit[];
  ingressImpacts: IngressImpact[];
  consumerImpacts?: ConsumerImpact[];
  scoreBreakdown: ScoreBreakdown;
  clusterWorkloadCount: number;
  coverageLevel: string;
}

export interface Remediation {
  type: string;
  description: string;
  priority: 'critical' | 'high' | 'medium' | 'low';
  impact: string;
}

export interface BlastRadiusResult {
  targetResource: ResourceRef;
  failureMode: string;
  blastRadiusPercent: number;
  criticalityScore: number;
  criticalityLevel: 'critical' | 'high' | 'medium' | 'low';
  subScores: SubScores;
  impactSummary: ImpactSummary;
  affectedServices: ServiceImpact[];
  affectedIngresses?: IngressImpact[];
  affectedConsumers?: ConsumerImpact[];
  scoreBreakdown: ScoreBreakdown;
  verdict: string;
  auditTrail?: AuditTrail;
  coverageLevel: string;
  coverageNote?: string;
  replicaCount: number;
  isSPOF: boolean;
  hasHPA: boolean;
  hasPDB: boolean;
  isIngressExposed: boolean;
  ingressHosts: string[];
  remediations: Remediation[];
  fanIn: number;
  fanOut: number;
  totalAffected: number;
  affectedNamespaces: number;
  waves: BlastWave[];
  dependencyChain: BlastDependencyEdge[];
  riskIndicators: RiskIndicator[];
  graphNodeCount: number;
  graphEdgeCount: number;
  graphStalenessMs: number;
}

export interface BlastWave {
  depth: number;
  resources: AffectedResource[];
}

export interface AffectedResource {
  kind: string;
  name: string;
  namespace: string;
  impact: 'direct' | 'transitive';
  waveDepth: number;
  failurePath: PathHop[];
}

export interface PathHop {
  from: ResourceRef;
  to: ResourceRef;
  edgeType: string;
  detail: string;
}

export interface RiskIndicator {
  severity: 'critical' | 'warning' | 'info';
  title: string;
  detail: string;
}

export interface BlastDependencyEdge {
  source: ResourceRef;
  target: ResourceRef;
  type: string;
  detail?: string;
}

export interface GraphStatus {
  ready: boolean;
  nodeCount: number;
  edgeCount: number;
  namespaceCount: number;
  lastRebuildMs: number;
  stalenessMs: number;
  rebuildCount: number;
  error?: string;
}

export interface BlastRadiusSummaryEntry {
  resource: ResourceRef;
  criticalityScore: number;
  criticalityLevel: string;
  blastRadiusPercent: number;
  fanIn: number;
  isSPOF: boolean;
  affectedNamespaces: number;
}
```

- [ ] **Step 3: Fix all TypeScript compile errors from the rename**

Run: `cd /Users/koti/myFuture/Kubernetes/kubilitics/kubilitics-frontend && npx tsc --noEmit 2>&1 | head -50`

This will show all files that reference the old snake_case field names. Fix each one:
- `blastRadius.ts` — update field access in normalization
- `useBlastRadius.ts` — update GraphStatus field access
- `BlastRadiusTab.tsx` — update all `data.field_name` → `data.fieldName`
- `WaveBreakdown.tsx` — update `wave_depth` → `waveDepth`, `failure_path` → `failurePath`
- `RiskPanel.tsx` — update field names
- `CriticalityBanner.tsx` — update field names
- `RiskIndicatorCards.tsx` — update field names

Fix ALL errors until `tsc --noEmit` passes clean.

- [ ] **Step 4: Verify the app builds**

Run: `cd /Users/koti/myFuture/Kubernetes/kubilitics/kubilitics-frontend && npm run build 2>&1 | tail -10`
Expected: Build succeeds

- [ ] **Step 5: Commit**

```bash
git add -A && git commit -m "refactor(blast-radius): migrate all types to camelCase — align with v2 backend API"
```

---

## Task 2: Update API Client + Hook

**Files:**
- Modify: `src/services/api/blastRadius.ts`
- Modify: `src/hooks/useBlastRadius.ts`

- [ ] **Step 1: Add failureMode and audit params to getBlastRadius**

In `src/services/api/blastRadius.ts`, update the `getBlastRadius` function to accept optional params:

```typescript
export async function getBlastRadius(
  baseUrl: string,
  clusterId: string,
  namespace: string,
  kind: string,
  name: string,
  failureMode?: string,
  audit?: boolean,
): Promise<BlastRadiusResult> {
  const ns = namespace || '-';
  const url = `${baseUrl}/api/v1/clusters/${encodeURIComponent(clusterId)}/blast-radius/${encodeURIComponent(ns)}/${encodeURIComponent(kind)}/${encodeURIComponent(name)}`;
  
  const params = new URLSearchParams();
  if (failureMode) params.set('failure_mode', failureMode);
  if (audit) params.set('audit', 'true');
  const queryString = params.toString();
  
  const res = await fetch(queryString ? `${url}?${queryString}` : url);
  if (!res.ok) throw new Error(`Blast radius request failed: ${res.status}`);
  const data: BlastRadiusResult = await res.json();
  
  // Defensive normalization for nil slices from Go
  data.waves = data.waves || [];
  data.dependencyChain = data.dependencyChain || [];
  data.riskIndicators = data.riskIndicators || [];
  data.ingressHosts = data.ingressHosts || [];
  data.affectedServices = data.affectedServices || [];
  data.remediations = data.remediations || [];
  data.impactSummary = data.impactSummary || { brokenCount: 0, degradedCount: 0, selfHealingCount: 0, totalWorkloads: 0, capacityNotes: [] };
  data.impactSummary.capacityNotes = data.impactSummary.capacityNotes || [];
  for (const w of data.waves) {
    w.resources = w.resources || [];
    for (const r of w.resources) {
      r.failurePath = r.failurePath || [];
    }
  }
  
  return data;
}
```

- [ ] **Step 2: Add failureMode to useBlastRadius hook**

In `src/hooks/useBlastRadius.ts`, update the options interface and pass failureMode through:

```typescript
export interface UseBlastRadiusOptions {
  kind: string;
  namespace?: string | null;
  name?: string | null;
  enabled?: boolean;
  failureMode?: string;
}
```

Update the hook to destructure `failureMode` from options, include it in the query key, and pass it to `getBlastRadius`:

- Query key: `['blast-radius', clusterId, kind, normalizedNamespace, normalizedName, failureMode]`
- API call: `getBlastRadius(baseUrl, clusterId, ns, kind, name, failureMode)`

- [ ] **Step 3: Verify build**

Run: `cd /Users/koti/myFuture/Kubernetes/kubilitics/kubilitics-frontend && npm run build 2>&1 | tail -5`
Expected: Build succeeds

- [ ] **Step 4: Commit**

```bash
git add src/services/api/blastRadius.ts src/hooks/useBlastRadius.ts
git commit -m "feat(blast-radius): add failure mode + audit params to API client and hook"
```

---

## Task 3: CoverageBanner Component

**Files:**
- Create: `src/components/blast-radius/CoverageBanner.tsx`

- [ ] **Step 1: Create the component**

```tsx
import { AlertTriangle } from 'lucide-react';

interface CoverageBannerProps {
  coverageLevel: string;
  coverageNote?: string;
}

export function CoverageBanner({ coverageLevel, coverageNote }: CoverageBannerProps) {
  if (coverageLevel !== 'partial') return null;

  return (
    <div className="flex items-center gap-2 px-4 py-2 rounded-lg bg-amber-500/10 border border-amber-500/20 text-amber-200 text-sm">
      <AlertTriangle className="h-4 w-4 shrink-0 text-amber-500" />
      <span>
        {coverageNote || 'Dependency coverage is partial — enable tracing for full analysis'}
      </span>
    </div>
  );
}
```

- [ ] **Step 2: Verify build**

Run: `cd /Users/koti/myFuture/Kubernetes/kubilitics/kubilitics-frontend && npm run build 2>&1 | tail -5`
Expected: Build succeeds

- [ ] **Step 3: Commit**

```bash
git add src/components/blast-radius/CoverageBanner.tsx
git commit -m "feat(blast-radius): add CoverageBanner — persistent amber bar for partial coverage"
```

---

## Task 4: CriticalityBanner Rewrite

**Files:**
- Rewrite: `src/components/blast-radius/CriticalityBanner.tsx`

- [ ] **Step 1: Read current file**

Read `src/components/blast-radius/CriticalityBanner.tsx` to understand current structure.

- [ ] **Step 2: Rewrite with new design**

Replace the entire file content with:

```tsx
import { cn } from '@/lib/utils';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';

const gradientMap: Record<string, string> = {
  critical: 'from-red-600 to-red-900',
  high: 'from-orange-500 to-orange-800',
  medium: 'from-yellow-500 to-yellow-700',
  low: 'from-blue-500 to-blue-700',
};

export interface CriticalityBannerProps {
  criticalityScore: number;
  criticalityLevel: 'critical' | 'high' | 'medium' | 'low';
  verdict: string;
  targetName: string;
  failureMode: string;
  onFailureModeChange: (mode: string) => void;
}

export function CriticalityBanner({
  criticalityScore,
  criticalityLevel,
  verdict,
  targetName,
  failureMode,
  onFailureModeChange,
}: CriticalityBannerProps) {
  const gradient = gradientMap[criticalityLevel] || gradientMap.low;

  return (
    <div className={cn('relative rounded-xl p-5 bg-gradient-to-r text-white overflow-hidden', gradient)}>
      <div className="flex items-start justify-between gap-4">
        {/* Left: target + verdict */}
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-3 mb-1">
            <h3 className="text-sm font-medium text-white/80">
              Impact Analysis for <span className="font-bold text-white">{targetName}</span>
            </h3>
          </div>
          <p className="text-sm text-white/70 line-clamp-2 mt-1">{verdict}</p>
        </div>

        {/* Right: score + dropdown */}
        <div className="flex flex-col items-end gap-2 shrink-0">
          <Select value={failureMode} onValueChange={onFailureModeChange}>
            <SelectTrigger className="h-7 w-[160px] text-xs bg-white/10 border-white/20 text-white">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="pod-crash">Pod Crash</SelectItem>
              <SelectItem value="workload-deletion">Workload Deletion</SelectItem>
              <SelectItem value="namespace-deletion">Namespace Deletion</SelectItem>
            </SelectContent>
          </Select>
          <div className="text-right">
            <div className="text-4xl font-bold leading-none">{Math.round(criticalityScore)}</div>
            <div className="text-xs font-semibold uppercase tracking-wider mt-1 text-white/80">
              {criticalityLevel}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
```

- [ ] **Step 3: Verify build**

Run: `cd /Users/koti/myFuture/Kubernetes/kubilitics/kubilitics-frontend && npm run build 2>&1 | tail -5`
Expected: Build succeeds

- [ ] **Step 4: Commit**

```bash
git add src/components/blast-radius/CriticalityBanner.tsx
git commit -m "feat(blast-radius): rewrite CriticalityBanner — headline with failure mode dropdown"
```

---

## Task 5: ScoreTooltip Component

**Files:**
- Create: `src/components/blast-radius/ScoreTooltip.tsx`

- [ ] **Step 1: Create the component**

```tsx
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip';
import { cn } from '@/lib/utils';
import type { ScoringFactor } from '@/services/api/types';

interface ScoreTooltipProps {
  title: string;
  score: number;
  factors: ScoringFactor[];
  onViewDetails: () => void;
  children: React.ReactNode;
}

export function ScoreTooltip({ title, score, factors, onViewDetails, children }: ScoreTooltipProps) {
  const displayFactors = factors.slice(0, 4);
  const remaining = factors.length - displayFactors.length;

  return (
    <TooltipProvider delayDuration={200}>
      <Tooltip>
        <TooltipTrigger asChild>{children}</TooltipTrigger>
        <TooltipContent side="bottom" className="w-[280px] p-3" sideOffset={8}>
          <div className="flex items-center justify-between mb-2">
            <span className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">{title}</span>
            <span className="text-sm font-bold">{score}</span>
          </div>
          <div className="border-t border-border pt-2 space-y-1.5">
            {displayFactors.map((f, i) => (
              <div key={i} className="flex items-center gap-2 text-xs">
                <span className={cn(
                  'font-mono w-8 text-right shrink-0',
                  f.effect > 0 ? 'text-green-500' : f.effect < 0 ? 'text-red-400' : 'text-muted-foreground'
                )}>
                  {f.effect > 0 ? '+' : ''}{Math.round(f.effect)}
                </span>
                <span className="text-muted-foreground truncate">{f.note}</span>
              </div>
            ))}
            {remaining > 0 && (
              <div className="text-xs text-muted-foreground">+{remaining} more</div>
            )}
          </div>
          <button
            onClick={(e) => { e.stopPropagation(); onViewDetails(); }}
            className="mt-2 text-xs text-primary hover:underline"
          >
            View details →
          </button>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}
```

- [ ] **Step 2: Verify build**

Run: `cd /Users/koti/myFuture/Kubernetes/kubilitics/kubilitics-frontend && npm run build 2>&1 | tail -5`

- [ ] **Step 3: Commit**

```bash
git add src/components/blast-radius/ScoreTooltip.tsx
git commit -m "feat(blast-radius): add ScoreTooltip — hover tooltip for scoring factors"
```

---

## Task 6: RiskIndicatorCards Rewrite

**Files:**
- Rewrite: `src/components/blast-radius/RiskIndicatorCards.tsx`

- [ ] **Step 1: Read current file**

Read the current RiskIndicatorCards.tsx.

- [ ] **Step 2: Rewrite with hybrid card design**

Replace the entire file:

```tsx
import { motion } from 'framer-motion';
import { cn } from '@/lib/utils';
import { ScoreTooltip } from './ScoreTooltip';
import type { SubScores, ImpactSummary } from '@/services/api/types';

export interface RiskIndicatorCardsProps {
  subScores: SubScores;
  blastRadiusPercent: number;
  impactSummary: ImpactSummary;
  coverageLevel: string;
  onOpenDetail: (section: 'resilience' | 'exposure' | 'recovery' | 'impact') => void;
}

function scoreColor(score: number): string {
  if (score >= 70) return 'text-green-500';
  if (score >= 40) return 'text-yellow-500';
  return 'text-red-500';
}

function badgeStyle(score: number): { bg: string; text: string; label: string } {
  if (score >= 70) return { bg: 'bg-green-500/10', text: 'text-green-500', label: '' };
  if (score >= 40) return { bg: 'bg-yellow-500/10', text: 'text-yellow-500', label: '' };
  return { bg: 'bg-red-500/10', text: 'text-red-500', label: '' };
}

function resilienceBadge(score: number) {
  const s = badgeStyle(score);
  s.label = score >= 70 ? 'STRONG' : score >= 40 ? 'MODERATE' : 'WEAK';
  return s;
}

function impactBadge(pct: number) {
  if (pct === 0) return { bg: 'bg-green-500/10', text: 'text-green-500', label: 'NONE' };
  if (pct < 5) return { bg: 'bg-green-500/10', text: 'text-green-500', label: 'LOW' };
  if (pct < 20) return { bg: 'bg-yellow-500/10', text: 'text-yellow-500', label: 'MODERATE' };
  return { bg: 'bg-red-500/10', text: 'text-red-500', label: 'HIGH' };
}

function exposureBadge(score: number) {
  const s = badgeStyle(score);
  // Exposure is inverted: high score = bad
  if (score < 20) { s.label = 'LOW'; s.bg = 'bg-green-500/10'; s.text = 'text-green-500'; }
  else if (score <= 50) { s.label = 'MODERATE'; s.bg = 'bg-yellow-500/10'; s.text = 'text-yellow-500'; }
  else { s.label = 'HIGH'; s.bg = 'bg-red-500/10'; s.text = 'text-red-500'; }
  return s;
}

function recoveryBadge(score: number) {
  const s = badgeStyle(score);
  s.label = score >= 70 ? 'FAST' : score >= 40 ? 'MODERATE' : 'SLOW';
  return s;
}

function impactColor(pct: number): string {
  if (pct === 0) return 'text-green-500';
  if (pct < 5) return 'text-green-500';
  if (pct < 20) return 'text-yellow-500';
  return 'text-red-500';
}

function contextLine(factors: { name: string; value: string; note: string }[]): string {
  return factors
    .slice(0, 3)
    .map(f => f.note)
    .join(' · ')
    .slice(0, 45);
}

export function RiskIndicatorCards({
  subScores,
  blastRadiusPercent,
  impactSummary,
  coverageLevel,
  onOpenDetail,
}: RiskIndicatorCardsProps) {
  const cards = [
    {
      key: 'resilience' as const,
      label: 'Resilience',
      score: subScores.resilience.score,
      displayValue: String(subScores.resilience.score),
      color: scoreColor(subScores.resilience.score),
      badge: resilienceBadge(subScores.resilience.score),
      context: contextLine(subScores.resilience.factors),
      factors: subScores.resilience.factors,
      extraBadge: null as string | null,
    },
    {
      key: 'impact' as const,
      label: 'Cluster Impact',
      score: Math.round(blastRadiusPercent),
      displayValue: `${blastRadiusPercent.toFixed(1)}%`,
      color: impactColor(blastRadiusPercent),
      badge: impactBadge(blastRadiusPercent),
      context: impactSummary.brokenCount === 0 && impactSummary.degradedCount === 0
        ? 'Self-healing'
        : `${impactSummary.brokenCount} broken · ${impactSummary.degradedCount} degraded`,
      factors: subScores.impact.factors,
      extraBadge: null,
    },
    {
      key: 'exposure' as const,
      label: 'Exposure',
      score: subScores.exposure.score,
      displayValue: String(subScores.exposure.score),
      color: subScores.exposure.score < 20 ? 'text-green-500' : subScores.exposure.score <= 50 ? 'text-yellow-500' : 'text-red-500',
      badge: exposureBadge(subScores.exposure.score),
      context: contextLine(subScores.exposure.factors),
      factors: subScores.exposure.factors,
      extraBadge: coverageLevel === 'partial' ? 'Partial' : null,
    },
    {
      key: 'recovery' as const,
      label: 'Recovery',
      score: subScores.recovery.score,
      displayValue: String(subScores.recovery.score),
      color: scoreColor(subScores.recovery.score),
      badge: recoveryBadge(subScores.recovery.score),
      context: contextLine(subScores.recovery.factors),
      factors: subScores.recovery.factors,
      extraBadge: null,
    },
  ];

  return (
    <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
      {cards.map((card, index) => (
        <motion.div
          key={card.key}
          initial={{ opacity: 0, y: 12 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.35, ease: 'easeOut', delay: index * 0.05 }}
        >
          <ScoreTooltip
            title={card.label}
            score={card.score}
            factors={card.factors}
            onViewDetails={() => onOpenDetail(card.key)}
          >
            <div className="border-none soft-shadow glass-panel rounded-xl p-4 cursor-pointer hover:shadow-md transition-shadow">
              <div className="flex items-center justify-between mb-1">
                <span className="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
                  {card.label}
                </span>
                <div className="flex items-center gap-1.5">
                  {card.extraBadge && (
                    <span className="px-1.5 py-0.5 rounded-full text-[10px] font-semibold bg-amber-500/10 text-amber-500">
                      {card.extraBadge}
                    </span>
                  )}
                  <span className={cn('px-2 py-0.5 rounded-full text-[10px] font-semibold', card.badge.bg, card.badge.text)}>
                    {card.badge.label}
                  </span>
                </div>
              </div>
              <div className={cn('text-[28px] font-bold leading-none my-2', card.color)}>
                {card.displayValue}
              </div>
              <div className="text-[11px] text-muted-foreground truncate">
                {card.context}
              </div>
            </div>
          </ScoreTooltip>
        </motion.div>
      ))}
    </div>
  );
}
```

- [ ] **Step 3: Verify build**

Run: `cd /Users/koti/myFuture/Kubernetes/kubilitics/kubilitics-frontend && npm run build 2>&1 | tail -5`

- [ ] **Step 4: Commit**

```bash
git add src/components/blast-radius/RiskIndicatorCards.tsx
git commit -m "feat(blast-radius): rewrite RiskIndicatorCards — hybrid score+badge+context cards"
```

---

## Task 7: ScoreDetailSheet Component

**Files:**
- Create: `src/components/blast-radius/ScoreDetailSheet.tsx`

- [ ] **Step 1: Create the component**

```tsx
import { Sheet, SheetContent, SheetHeader, SheetTitle } from '@/components/ui/sheet';
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { ChevronDown, Download } from 'lucide-react';
import { cn } from '@/lib/utils';
import type { BlastRadiusResult, SubScoreDetail, ServiceImpact, Remediation } from '@/services/api/types';
import { useState } from 'react';

interface ScoreDetailSheetProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  initialSection?: 'resilience' | 'exposure' | 'recovery' | 'impact';
  result: BlastRadiusResult;
}

function classificationBadge(cls: string) {
  switch (cls) {
    case 'broken': return <Badge variant="destructive" className="text-[10px]">Broken</Badge>;
    case 'degraded': return <Badge className="text-[10px] bg-yellow-500/10 text-yellow-500 border-yellow-500/20">Degraded</Badge>;
    default: return <Badge className="text-[10px] bg-green-500/10 text-green-500 border-green-500/20">Self-healing</Badge>;
  }
}

function SubScoreSection({ title, detail, defaultOpen }: { title: string; detail: SubScoreDetail; defaultOpen: boolean }) {
  const [open, setOpen] = useState(defaultOpen);
  return (
    <Collapsible open={open} onOpenChange={setOpen}>
      <CollapsibleTrigger className="flex items-center justify-between w-full py-2 px-3 rounded-lg hover:bg-muted/50">
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium">{title}</span>
          <span className="text-sm font-bold">{detail.score}</span>
          {detail.source && (
            <span className="text-[10px] text-muted-foreground">via {detail.source}</span>
          )}
        </div>
        <ChevronDown className={cn('h-4 w-4 transition-transform', open && 'rotate-180')} />
      </CollapsibleTrigger>
      <CollapsibleContent>
        <div className="px-3 pb-3">
          <table className="w-full text-xs">
            <thead>
              <tr className="text-muted-foreground">
                <th className="text-left py-1 font-medium">Factor</th>
                <th className="text-left py-1 font-medium">Value</th>
                <th className="text-right py-1 font-medium">Effect</th>
              </tr>
            </thead>
            <tbody>
              {detail.factors.map((f, i) => (
                <tr key={i} className="border-t border-border/50">
                  <td className="py-1.5 text-muted-foreground">{f.note}</td>
                  <td className="py-1.5">{f.value}</td>
                  <td className={cn('py-1.5 text-right font-mono',
                    f.effect > 0 ? 'text-green-500' : f.effect < 0 ? 'text-red-400' : 'text-muted-foreground'
                  )}>
                    {f.effect > 0 ? '+' : ''}{Math.round(f.effect)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          {detail.confidence && (
            <div className="mt-2 text-[10px] text-muted-foreground">
              Confidence: {detail.confidence}
            </div>
          )}
        </div>
      </CollapsibleContent>
    </Collapsible>
  );
}

export function ScoreDetailSheet({ open, onOpenChange, initialSection, result }: ScoreDetailSheetProps) {
  const handleExport = async () => {
    // Re-fetch with audit=true to get full audit trail
    try {
      const baseUrl = (window as any).__KUBILITICS_API_BASE__ || '';
      const clusterId = result.targetResource.namespace; // TODO: get from context
      const auditResult = await fetch(
        `${baseUrl}/api/v1/clusters/${encodeURIComponent(clusterId)}/blast-radius/${encodeURIComponent(result.targetResource.namespace)}/${encodeURIComponent(result.targetResource.kind)}/${encodeURIComponent(result.targetResource.name)}?audit=true`
      ).then(r => r.json()).catch(() => result);
      
      const blob = new Blob([JSON.stringify(auditResult, null, 2)], { type: 'application/json' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `blast-radius-audit-${result.targetResource.kind}-${result.targetResource.name}.json`;
      a.click();
      URL.revokeObjectURL(url);
    } catch {
      // Fallback: export current data without audit trail
      const blob = new Blob([JSON.stringify(result, null, 2)], { type: 'application/json' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `blast-radius-${result.targetResource.kind}-${result.targetResource.name}.json`;
      a.click();
      URL.revokeObjectURL(url);
    }
  };

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className="w-[480px] sm:max-w-[480px] overflow-y-auto">
        <SheetHeader>
          <SheetTitle className="flex items-center justify-between">
            <span>Score Breakdown</span>
            <Badge variant={result.criticalityLevel === 'critical' ? 'destructive' : 'secondary'}>
              {Math.round(result.criticalityScore)} {result.criticalityLevel.toUpperCase()}
            </Badge>
          </SheetTitle>
          <p className="text-sm text-muted-foreground">{result.verdict}</p>
        </SheetHeader>

        <div className="mt-6 space-y-2">
          <SubScoreSection title="Resilience" detail={result.subScores.resilience} defaultOpen={initialSection === 'resilience'} />
          <SubScoreSection title="Exposure" detail={result.subScores.exposure} defaultOpen={initialSection === 'exposure'} />
          <SubScoreSection title="Recovery" detail={result.subScores.recovery} defaultOpen={initialSection === 'recovery'} />
          <SubScoreSection title="Impact" detail={result.subScores.impact} defaultOpen={initialSection === 'impact'} />
        </div>

        {result.affectedServices.length > 0 && (
          <div className="mt-6">
            <h4 className="text-sm font-semibold mb-2">Affected Services</h4>
            <div className="space-y-2">
              {result.affectedServices.map((si: ServiceImpact, i: number) => (
                <div key={i} className="flex items-center justify-between text-xs p-2 rounded-lg bg-muted/30">
                  <div>
                    <span className="font-medium">{si.service.name}</span>
                    <span className="text-muted-foreground ml-2">
                      {si.remainingEndpoints}/{si.totalEndpoints} endpoints
                    </span>
                  </div>
                  {classificationBadge(si.classification)}
                </div>
              ))}
            </div>
          </div>
        )}

        {result.remediations.length > 0 && (
          <div className="mt-6">
            <h4 className="text-sm font-semibold mb-2">Remediations</h4>
            <div className="space-y-2">
              {result.remediations.map((r: Remediation, i: number) => (
                <div key={i} className="text-xs p-2 rounded-lg bg-muted/30">
                  <div className="flex items-center gap-2">
                    <Badge variant="outline" className="text-[10px]">{r.priority}</Badge>
                    <span>{r.description}</span>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

        <div className="mt-6">
          <Button variant="outline" size="sm" onClick={handleExport} className="w-full">
            <Download className="h-4 w-4 mr-2" />
            Export Audit Trail (JSON)
          </Button>
        </div>
      </SheetContent>
    </Sheet>
  );
}
```

- [ ] **Step 2: Verify build**

Run: `cd /Users/koti/myFuture/Kubernetes/kubilitics/kubilitics-frontend && npm run build 2>&1 | tail -5`

- [ ] **Step 3: Commit**

```bash
git add src/components/blast-radius/ScoreDetailSheet.tsx
git commit -m "feat(blast-radius): add ScoreDetailSheet — full breakdown side panel with export"
```

---

## Task 8: Wire Everything into BlastRadiusTab

**Files:**
- Modify: `src/components/resources/BlastRadiusTab.tsx`

- [ ] **Step 1: Read current file to understand structure**

Read the full BlastRadiusTab.tsx, focusing on:
- Imports (top)
- State declarations (lines 54-68)
- Where CriticalityBanner and RiskIndicatorCards are rendered (lines 383-404)
- Return structure

- [ ] **Step 2: Add new imports**

Add at the top of the file:

```typescript
import { CoverageBanner } from '@/components/blast-radius/CoverageBanner';
import { ScoreDetailSheet } from '@/components/blast-radius/ScoreDetailSheet';
import { useState } from 'react';  // if not already imported
```

- [ ] **Step 3: Add failure mode state**

After the existing state declarations (around line 63), add:

```typescript
const [failureMode, setFailureMode] = useState<string>(() => {
  switch (kind.toLowerCase()) {
    case 'pod': return 'pod-crash';
    case 'namespace': return 'namespace-deletion';
    default: return 'workload-deletion';
  }
});
const [detailSheetOpen, setDetailSheetOpen] = useState(false);
const [detailSheetSection, setDetailSheetSection] = useState<'resilience' | 'exposure' | 'recovery' | 'impact'>('resilience');
```

- [ ] **Step 4: Pass failureMode to useBlastRadius**

Update the `useBlastRadius` call to include `failureMode`:

```typescript
const { data: blastData, ... } = useBlastRadius({
  kind,
  namespace,
  name,
  failureMode,
  enabled: true,
});
```

- [ ] **Step 5: Update the render section**

Replace the CriticalityBanner and RiskIndicatorCards rendering (lines ~383-404) with:

```tsx
{/* Coverage Banner */}
{blastData && (
  <CoverageBanner
    coverageLevel={blastData.coverageLevel}
    coverageNote={blastData.coverageNote}
  />
)}

{/* Criticality Banner */}
{blastData && (
  <CriticalityBanner
    criticalityScore={blastData.criticalityScore}
    criticalityLevel={blastData.criticalityLevel}
    verdict={blastData.verdict}
    targetName={`${kind}/${name}`}
    failureMode={failureMode}
    onFailureModeChange={setFailureMode}
  />
)}

{/* Risk Indicator Cards */}
{blastData && (
  <RiskIndicatorCards
    subScores={blastData.subScores}
    blastRadiusPercent={blastData.blastRadiusPercent}
    impactSummary={blastData.impactSummary}
    coverageLevel={blastData.coverageLevel}
    onOpenDetail={(section) => {
      setDetailSheetSection(section);
      setDetailSheetOpen(true);
    }}
  />
)}

{/* Score Detail Sheet */}
{blastData && (
  <ScoreDetailSheet
    open={detailSheetOpen}
    onOpenChange={setDetailSheetOpen}
    initialSection={detailSheetSection}
    result={blastData}
  />
)}
```

- [ ] **Step 6: Update WaveBreakdown and RiskPanel field names**

In the same file, update any remaining snake_case field accesses:
- `blastData.risk_indicators` → `blastData.riskIndicators`
- Any other snake_case references

- [ ] **Step 7: Verify build**

Run: `cd /Users/koti/myFuture/Kubernetes/kubilitics/kubilitics-frontend && npm run build 2>&1 | tail -10`
Expected: Build succeeds

- [ ] **Step 8: Commit**

```bash
git add src/components/resources/BlastRadiusTab.tsx
git commit -m "feat(blast-radius): wire v2 components into BlastRadiusTab — coverage, banner, cards, detail sheet"
```

---

## Task 9: Update WaveBreakdown + RiskPanel Field Names

**Files:**
- Modify: `src/components/blast-radius/WaveBreakdown.tsx`
- Modify: `src/components/blast-radius/RiskPanel.tsx`

- [ ] **Step 1: Read WaveBreakdown.tsx and fix field names**

Search for `wave_depth`, `failure_path`, `edge_type` and rename to `waveDepth`, `failurePath`, `edgeType`.

- [ ] **Step 2: Read RiskPanel.tsx and fix field names**

Search for any snake_case field access and rename to camelCase.

- [ ] **Step 3: Verify full build**

Run: `cd /Users/koti/myFuture/Kubernetes/kubilitics/kubilitics-frontend && npm run build 2>&1 | tail -5`
Expected: Build succeeds with zero errors

- [ ] **Step 4: Commit**

```bash
git add src/components/blast-radius/WaveBreakdown.tsx src/components/blast-radius/RiskPanel.tsx
git commit -m "refactor(blast-radius): rename snake_case to camelCase in WaveBreakdown and RiskPanel"
```

---

## Summary

| Task | Component | Complexity |
|---|---|---|
| 1 | TypeScript types (camelCase migration) | Medium (many files touched) |
| 2 | API client + hook (failure mode) | Small |
| 3 | CoverageBanner | Small |
| 4 | CriticalityBanner rewrite | Small |
| 5 | ScoreTooltip | Small |
| 6 | RiskIndicatorCards rewrite | Medium (core visual) |
| 7 | ScoreDetailSheet | Medium (largest new component) |
| 8 | BlastRadiusTab wiring | Medium (integration) |
| 9 | WaveBreakdown + RiskPanel field renames | Small |

Total: 9 tasks. Tasks 1-2 must be sequential (types before API). Tasks 3-7 are independent (new/rewritten components). Task 8 depends on all prior tasks. Task 9 can happen alongside 8.
