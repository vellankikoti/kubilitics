# Blast Radius Frontend v2 — Design Spec

**Date:** 2026-04-09
**Scope:** Sub-project 3 of 3 — Frontend Overhaul
**Depends on:** Backend Engine v2 (branch `blast-radius-v2`, complete)
**Status:** Approved for implementation

---

## Problem Statement

The current blast radius frontend displays 4 generic graph-metric cards (SPOF, Blast Radius %, Fan-In/Out, Cross-Namespace) and a criticality banner that don't reflect the new classification engine. The backend now returns composite sub-scores, impact classifications (broken/degraded/self-healing), natural language verdicts, and coverage levels — none of which are surfaced in the UI. Additionally, the TypeScript types use snake_case while the new backend returns camelCase JSON.

---

## Design Decisions

| Decision | Choice | Rationale |
|---|---|---|
| JSON field naming | camelCase throughout | Clean break — align TS types with new backend API |
| Header cards | Hybrid (score + badge + context) | Score visually dominant for power users, badge for quick scan, context for understanding |
| Banner | Headline only (score + level + dropdown + verdict) | Cards carry the detail; banner is the "headline" |
| Score tooltip | Hover tooltip (3-4 factors) + link to side sheet | Fast scanning + deep-dive without layout break |
| Coverage warning | Persistent amber banner + inline badge on Exposure card | Data integrity — never let users trust incomplete data without warning |
| Failure mode | Dropdown in banner, auto-detect default, re-fetches on change | Context-sensitive defaults with user override |

---

## Components

### 1. RiskIndicatorCards.tsx (Rewrite)

**Props:**
```typescript
interface RiskIndicatorCardsProps {
  subScores: SubScores;
  blastRadiusPercent: number;
  impactSummary: ImpactSummary;
  coverageLevel: string;
  onOpenDetail: (section: 'resilience' | 'exposure' | 'recovery' | 'impact') => void;
}
```

**Layout:** 4-card grid (`grid-cols-2 lg:grid-cols-4`), Framer Motion staggered entry.

**Each card structure:**
```
┌─────────────────────────────────────┐
│ LABEL (uppercase, 11px, muted)  [BADGE] │
│                                         │
│         SCORE (28px, bold, colored)     │
│                                         │
│ context line (11px, muted, 1 line max)  │
└─────────────────────────────────────┘
```

- **Score:** visually dominant, color-coded: green (>=70), yellow (40-69), red (<40)
- **Badge:** subtle pill, right-aligned, color-coded to match score tier
- **Context:** single crisp line, max ~40 chars

**Badge text mapping:**

| Card | >=70 | 40-69 | <40 |
|---|---|---|---|
| Resilience | STRONG (green) | MODERATE (yellow) | WEAK (red) |
| Recovery | FAST (green) | MODERATE (yellow) | SLOW (red) |
| Exposure | LOW (green) | MODERATE (yellow) | HIGH (red) |

Cluster Impact card uses blast radius percent:

| Condition | Badge |
|---|---|
| 0% | NONE (green) |
| <5% | LOW (green) |
| <20% | MODERATE (yellow) |
| >=20% | HIGH (red) |

**Context line examples:**
- Resilience: "3 replicas · HPA · PDB" or "1 replica · no HPA"
- Cluster Impact: "0 broken · 1 degraded" or "Self-healing"
- Exposure: "Not ingress · 2 consumers" or "Ingress-exposed · 5 consumers"
- Recovery: "Deployment · fast recovery" or "StatefulSet · ordered restart"

**Hover:** Each card triggers `ScoreTooltip` on hover.
**Click:** "View details" in tooltip calls `onOpenDetail(section)`.

**Exposure card special:** When `coverageLevel === "partial"`, show a small "Partial" badge (amber) next to the main badge, with a tooltip: "Consumer count may be incomplete — enable tracing for full accuracy."

---

### 2. CriticalityBanner.tsx (Rewrite)

**Props:**
```typescript
interface CriticalityBannerProps {
  criticalityScore: number;
  criticalityLevel: 'critical' | 'high' | 'medium' | 'low';
  verdict: string;
  targetName: string;
  failureMode: string;
  onFailureModeChange: (mode: string) => void;
}
```

**Layout:**
```
┌──────────────────────────────────────────────────────────┐
│ Impact Analysis for Pod/trace-demo-app      [pod-crash ▾] │
│ "No services lose functionality..."                   17  │
│                                                      LOW  │
└──────────────────────────────────────────────────────────┘
```

- **Left:** "Impact Analysis for `Kind/Name`" (bold) + verdict text (regular, truncated to 1-2 lines)
- **Right:** Large score number (36px, bold) + criticality level badge below
- **Top-right:** Failure mode dropdown (`<Select>` from shadcn/ui)

**Dropdown options:**
- Pod Crash (default for Pods)
- Workload Deletion (default for Deployments/StatefulSets)
- Namespace Deletion (default for Namespaces)

Changing the dropdown calls `onFailureModeChange` which triggers a re-fetch via `useBlastRadius`.

**Gradient backgrounds:** Same color mapping as current (critical=red, high=orange, medium=yellow, low=blue).

---

### 3. ScoreTooltip.tsx (New)

**Props:**
```typescript
interface ScoreTooltipProps {
  title: string;
  score: number;
  factors: ScoringFactor[];
  onViewDetails: () => void;
}
```

**Behavior:** Rendered inside a Radix `Tooltip` (from shadcn/ui). Shows on hover, hides on mouse-out.

**Layout:**
```
┌──────────────────────────────┐
│ Resilience: 50               │
│ ─────────────────────────    │
│ -20  2 replicas              │
│ -15  No autoscaler           │
│ -15  No disruption budget    │
│                              │
│ View details →               │
└──────────────────────────────┘
```

- **Header:** title + score
- **Factors:** max 4, each showing effect (+/-) and short note
- **Footer:** "View details →" link, calls `onViewDetails`
- **Constraints:** no scroll, max 4 factors (truncate with "+N more" if >4), max width 280px

---

### 4. ScoreDetailSheet.tsx (New)

**Props:**
```typescript
interface ScoreDetailSheetProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  initialSection?: 'resilience' | 'exposure' | 'recovery' | 'impact';
  result: BlastRadiusResult;
}
```

**Component:** Uses `Sheet` from shadcn/ui (slides in from right, 480px wide).

**Sections:**

1. **Header:** Overall score (large) + level badge + verdict text

2. **Sub-score sections** (4 collapsible, initially expand the `initialSection`):
   Each shows:
   - Section title + score + badge
   - Table of factors: Name | Value | Effect | Note
   - Source + confidence for Exposure section

3. **Affected Services:**
   List of ServiceImpact items with classification badges:
   - broken: red badge
   - degraded: yellow badge
   - self-healing: green badge
   Each shows: service name, remaining/total endpoints, threshold source

4. **Remediations:**
   Ordered by priority (critical → low). Each shows type, description, expected impact.

5. **Export button:** "Export Audit Trail (JSON)" — downloads the full audit trail. Triggers a re-fetch with `?audit=true` and saves the response as a `.json` file.

---

### 5. CoverageBanner.tsx (New)

**Props:**
```typescript
interface CoverageBannerProps {
  coverageLevel: string;
  coverageNote?: string;
}
```

**Behavior:** Only renders when `coverageLevel === "partial"`.

**Layout:** Thin amber bar (40px height) above the cards.
```
┌──────────────────────────────────────────────────────────┐
│ ⚠ Dependency coverage is partial — enable tracing for    │
│   full analysis                                          │
└──────────────────────────────────────────────────────────┘
```

- Background: `bg-amber-500/10` with `border-amber-500/20` border
- Icon: `AlertTriangle` from lucide-react (amber)
- Text: `coverageNote` from API, or default "Dependency coverage is partial — enable tracing for full analysis"
- **Non-dismissible** — always visible when partial

---

### 6. BlastRadiusTab.tsx (Modify)

**Changes:**
- Add `failureMode` state (auto-detected from `kind` prop, user-overridable via dropdown)
- Pass `failureMode` to `useBlastRadius`
- Add `detailSheetOpen` + `detailSheetSection` state for the side sheet
- Render new component hierarchy:
  ```
  CoverageBanner
  CriticalityBanner (with failure mode dropdown)
  RiskIndicatorCards (with tooltip + sheet trigger)
  ScoreDetailSheet (slide-in panel)
  SimulationControls + TopologyCanvas (unchanged)
  WaveBreakdown + RiskPanel (unchanged)
  ```

---

### 7. useBlastRadius.ts (Modify)

**Changes:**
- Accept `failureMode?: string` parameter
- Pass `?failure_mode=X` query param to API
- Update query key to include failure mode: `['blast-radius', clusterId, kind, namespace, name, failureMode]`

---

### 8. blastRadius.ts API Client (Modify)

**Changes:**
- `getBlastRadius` accepts optional `failureMode` and `audit` params
- Appends `?failure_mode=X` and `?audit=true` to URL when provided
- Remove snake_case → camelCase normalization (backend now returns camelCase natively)

---

### 9. types.ts (Modify)

Rewrite all blast radius interfaces with camelCase field names. Add new types:

```typescript
// New types
interface SubScores {
  resilience: SubScoreDetail;
  exposure: SubScoreDetail;
  recovery: SubScoreDetail;
  impact: SubScoreDetail;
}

interface SubScoreDetail {
  score: number;
  factors: ScoringFactor[];
  source?: string;      // "otel" | "k8s-native"
  confidence?: string;  // "high" | "low"
}

interface ScoringFactor {
  name: string;
  value: string;
  effect: number;
  note: string;
}

interface ScoreBreakdown {
  resilience: SubScoreDetail;
  exposure: SubScoreDetail;
  recovery: SubScoreDetail;
  impact: SubScoreDetail;
  overall: number;
  level: string;
}

interface ImpactSummary {
  brokenCount: number;
  degradedCount: number;
  selfHealingCount: number;
  totalWorkloads: number;
  capacityNotes: string[];
}

interface ServiceImpact {
  service: ResourceRef;
  classification: 'broken' | 'degraded' | 'self-healing';
  totalEndpoints: number;
  remainingEndpoints: number;
  threshold: number;
  thresholdSource: string;
  note: string;
}

interface IngressImpact {
  ingress: ResourceRef;
  classification: string;
  host: string;
  backendService: string;
  note: string;
}

interface ConsumerImpact {
  workload: ResourceRef;
  classification: string;
  dependsOn: string;
  note: string;
}

interface AuditTrail {
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

// Updated BlastRadiusResult (camelCase)
interface BlastRadiusResult {
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
  // Backward compat
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
```

---

## Files to Modify/Create

| File | Action |
|---|---|
| `src/services/api/types.ts` | Modify — rewrite blast radius interfaces to camelCase, add new types |
| `src/services/api/blastRadius.ts` | Modify — add failureMode + audit params |
| `src/hooks/useBlastRadius.ts` | Modify — accept failureMode, update query key |
| `src/components/blast-radius/RiskIndicatorCards.tsx` | Rewrite — 4 hybrid cards |
| `src/components/blast-radius/CriticalityBanner.tsx` | Rewrite — headline banner with dropdown |
| `src/components/blast-radius/ScoreTooltip.tsx` | New — hover tooltip for score factors |
| `src/components/blast-radius/ScoreDetailSheet.tsx` | New — side sheet for full breakdown |
| `src/components/blast-radius/CoverageBanner.tsx` | New — persistent amber bar for partial coverage |
| `src/components/resources/BlastRadiusTab.tsx` | Modify — wire new components, add failureMode state |

---

## Implementation Order

1. TypeScript types (`types.ts`) — foundation for everything
2. API client + hook updates (`blastRadius.ts`, `useBlastRadius.ts`)
3. CoverageBanner (new, simple)
4. CriticalityBanner (rewrite, includes failure mode dropdown)
5. RiskIndicatorCards (rewrite, core visual change)
6. ScoreTooltip (new, used by cards)
7. ScoreDetailSheet (new, largest new component)
8. BlastRadiusTab (wire everything together)

---

## Unchanged Components

These components continue to work with existing data from the API response:
- `SimulationControls.tsx` — uses waves data (still present in response)
- `SimulationEngine.ts` — uses waves data
- `WaveBreakdown.tsx` — uses waves data (field names will change to camelCase)
- `RiskPanel.tsx` — uses riskIndicators data
- `TopologyCanvas` — uses dependency chain
- `PreApplyPanel.tsx` — separate feature, unchanged

Note: WaveBreakdown and RiskPanel will need minor field name updates (snake_case → camelCase) but no structural changes.
