# Kubilitics: Operational Intelligence Platform — Design Spec

**Date:** 2026-04-02
**Status:** Approved
**Source:** KOBS Strategic Intelligence Report (April 2026)
**Scope:** Full 12-week roadmap — 15 tasks across 3 phases, 2 parallel execution tracks

---

## 1. Thesis

Kubilitics pivots from "Kubernetes dashboard" to "Operational Intelligence Platform." The dependency graph is the product. The intelligence layer is the moat. The dashboard is just the interface.

**One-liner:** "Every K8s tool shows you what exists. Kubilitics shows you what will break, what depends on what, and how to make your clusters structurally resilient — before incidents happen."

## 2. Three Pillars

Every feature serves exactly one pillar. If it doesn't fit, it doesn't ship.

| Pillar | Engine | Purpose |
|--------|--------|---------|
| Structural Intelligence | Topology Engine | Understand cluster as a structural system |
| Change Intelligence | Blast Radius Engine | Predict impact of any mutation before execution |
| Resilience Intelligence | Reports & Scores | Deliver executive-level operational risk insights |

## 3. Target Personas

- **K8s Architect** — manages 5-50 clusters, needs structural dependency understanding
- **Platform Engineering Lead** — builds IDPs, needs change risk assessment before approving PRs
- **SRE Manager** — accountable for uptime SLAs, needs to know fragility before incidents
- **Engineering VP** — needs quarterly resilience reports showing SPOFs and remediation plans

## 4. Execution Architecture: Two Parallel Tracks

```
TRACK 1: CORE ENGINE                    TRACK 2: INTELLIGENCE LAYER
─────────────────────                   ──────────────────────────

T1: Fix 6 topology bugs (8-10d) ──┐
                                  ├──→ T6: Structural health scores (7d)
T2: Rewrite blast radius (5-7d)  │     T7: SPOF inventory dashboard (5d)
T3: Fix cross-NS viz (3d)        │     T8: Namespace risk ranking (4d)
T4: Fix blast % denominator (2d) │         │
T5: Add remediation suggestions (3d)       │
         │                                 │
         ▼                                 ▼
T10: Pre-apply blast radius (7d)  T9: Topology diffing (10d)
         │                                 │
         └────────────┬───────────────────┘
                      ▼
              T11: Resilience report (7d)
              T12: Scheduled reports (5d)
              T13: CNCF submission (5d)
              T14: Compliance mapping (7d)
              T15: Public launch (5d)
```

**Dependency rules:**
- Track 2 starts after T1 (topology bugs fixed) — health scores need trustworthy graph
- T9 (topology diffing) needs T1 complete + snapshot infrastructure
- T10 (pre-apply) needs T2+T4 (accurate blast radius scoring)
- T11-T15 need both tracks converged — reports aggregate health scores + blast radius data
- kcli is out of scope for this plan (separate component)

---

## 5. Phase 1: Fix the Foundation (Track 1, Tasks 1-5)

### T1: Fix 6 Topology Bugs (8-10 days)

**Pillar:** Structural Intelligence

5 distinct fixes (bugs 1 and 6 share root cause):

| Bug | Location | Fix |
|-----|----------|-----|
| Cache key missing mode/depth | `internal/pkg/topologycache/cache.go:34-39` | Migrate V1 callers to V2 cache key pattern (`clusterID\|mode\|namespace\|depth`), or deprecate V1 cache entirely |
| No active cache invalidation | `topologycache/cache.go` — TTL-only, 30s | Hook informer event handlers to call `InvalidateForCluster()` on resource add/update/delete |
| Expand ignores depth boundaries | `builder/depth_filter.go:103-143` | Add parent chain resolution when expanding at depth 0 — include parents of newly visible neighbors |
| Dynamic hub threshold instability | `filter.go:280-316` — 5% formula | Add hysteresis band: flip to hub at 5% threshold, un-flip at 3% to prevent oscillation near boundary |
| Sibling filtering breaks cross-deps | `filter.go:215` | Allow traversal through siblings when they're on the critical path (Pod A -> ConfigMap -> Pod B must not be broken) |
| Mode-depth cache inconsistency | V1 cache serves stale entries across modes | Same root cause as bug #1 — fixed by cache key fix |

**Files touched:**
- `internal/pkg/topologycache/cache.go`
- `internal/topology/v2/builder/depth_filter.go`
- `internal/topology/v2/filter.go`
- `internal/service/topology_service.go` (cache call sites)

**Validation:** Existing tests in `filter_test.go` must pass. Add regression tests for each bug: mode switch returns correct topology, expand at depth 0 includes parents, hub threshold doesn't oscillate, sibling filter preserves cross-dep chains.

### T2: Rewrite Blast Radius Scoring (5-7 days)

**Pillar:** Change Intelligence

Current scoring in `internal/graph/scoring.go:21-66` is additive and doesn't model failure modes. A non-SPOF pod in a 3-replica Deployment scores 46 (MEDIUM) due to graph centrality alone.

**New scoring model — three failure modes:**

| Mode | Description | Replica Factor |
|------|-------------|----------------|
| `pod-crash` | Single pod goes away | `impact = baseScore * (1 / replicas)` — near-zero if replicas > 1 |
| `workload-deletion` | Entire Deployment/StatefulSet/DaemonSet removed | `impact = baseScore * 1.0` — full impact |
| `namespace-deletion` | Entire namespace removed | `impact = sum(workload-deletion for all workloads in NS)` — catastrophic |

**Base score calculation (unchanged components):**
```
baseScore = 0
baseScore += min(pageRank * 30, 30)       // Graph centrality
baseScore += min(fanIn * 3, 20)            // Direct dependents
if crossNsCount > 1:
    baseScore += min(crossNsCount * 2.5, 10)
if isDataStore: baseScore += 15
if isIngressExposed: baseScore += 10
if isSPOF: baseScore += 10
if !hasHPA: baseScore += 5
if !hasPDB: baseScore += 5
```

**Final score:** `score = baseScore * replicaFactor` (capped at 100)

**Recalibrated thresholds:**
- LOW: < 20
- MEDIUM: 20-45
- HIGH: 45-70
- CRITICAL: > 70

**Result:** A pod-crash in a 3-replica Deployment with baseScore 46 → `46 * (1/3) = 15` → LOW. Correct.

**Files touched:**
- `internal/graph/scoring.go` (main rewrite)
- `internal/graph/builder.go` (pass failure mode)
- `internal/models/blast_radius.go` (add FailureMode field)
- `internal/graph/snapshot.go` (update level thresholds)

**Validation:** Update `scoring_test.go` with cases: pod-crash in 3-replica = LOW, workload-deletion of same = MEDIUM/HIGH, namespace-deletion = CRITICAL.

### T3: Fix Cross-Namespace Visualization (3 days)

**Pillar:** Change Intelligence

Current state: blast radius header claims "3 namespaces affected" but the topology graph only shows resources in the target's namespace.

**Fix:** In `BlastRadiusTab.tsx:106-143`, when injecting blast-affected resources into the topology graph, include resources from ALL affected namespaces (not just the target namespace). Add namespace group nodes for cross-namespace resources so the visual clearly shows the cascade crossing namespace boundaries.

**Files touched:**
- `kubilitics-frontend/src/components/blast-radius/BlastRadiusTab.tsx`
- `kubilitics-frontend/src/components/blast-radius/WaveBreakdown.tsx` (add namespace column)

### T4: Fix Blast % Denominator (2 days)

**Pillar:** Change Intelligence

Current: `totalAffected / s.TotalWorkloads` where TotalWorkloads = all cluster workloads. Misleading — includes unrelated resources.

**Fix:** Change denominator to reachable subgraph size: `totalAffected / len(reachableFromTarget)`. This answers "what percentage of related resources are impacted" instead of "what percentage of the entire cluster is impacted."

**Files touched:**
- `internal/graph/snapshot.go:213-218`
- `internal/graph/builder.go` (compute reachable subgraph size during graph build)

### T5: Add Remediation Suggestions (3 days)

**Pillar:** Change Intelligence

Rule-based remediation engine attached to blast radius results.

**Rules:**
| Condition | Remediation | Priority |
|-----------|-------------|----------|
| `replicas == 1` | "Increase replicas to at least 3 for redundancy" | HIGH |
| `!hasPDB` | "Add PodDisruptionBudget to protect against voluntary disruptions" | HIGH |
| `!hasHPA` | "Add HorizontalPodAutoscaler for elastic scaling" | MEDIUM |
| `isSPOF && fanIn > 5` | "Critical SPOF — mounted by {fanIn} workloads. Add redundancy immediately" | CRITICAL |
| `crossNsCount > 2` | "Cross-namespace dependency hub — consider namespace-local copies" | MEDIUM |

**Data model:**
```go
type Remediation struct {
    Type        string // "add-pdb", "increase-replicas", "add-hpa", etc.
    Description string // Human-readable suggestion
    Priority    string // "critical", "high", "medium", "low"
    Impact      string // "Reduces blast radius by ~X%"
}
```

**Files touched:**
- New: `internal/graph/remediation.go`
- Modified: `internal/models/blast_radius.go` (add `Remediations []Remediation`)
- Modified: `internal/graph/snapshot.go` (call remediation engine during blast radius computation)

---

## 6. Phase 2: Build the Intelligence Layer (Tasks 6-10)

### T6: Structural Health Scores (7 days)

**Pillar:** Resilience Intelligence

New package: `internal/intelligence/health/`

**Composite score per namespace (0-100, higher = healthier):**

| Component | Weight | Calculation |
|-----------|--------|-------------|
| SPOF density | 25% | `1 - (spofCount / totalWorkloads)` |
| PDB coverage | 20% | `workloadsWithPDB / totalWorkloads` |
| HPA coverage | 15% | `workloadsWithHPA / scalableWorkloads` |
| Redundancy ratio | 20% | `avg(currentReplicas / max(specReplicas, 2))` across workloads — capped at 1.0, where 2 is the minimum safe replica count |
| Dependency depth | 10% | Inverse of max critical path length in dependency graph |
| Cross-NS risk | 10% | Inverse of cross-namespace dependency density |

**Scopes:**
- Per-namespace: weighted sum of components
- Per-cluster: weighted average of namespace scores (weighted by workload count)
- Per-fleet: weighted average of cluster scores

**API:** `GET /api/v1/clusters/{id}/health` → `HealthReport{Score, Level, Components[], Namespaces[]}`

**Frontend:** Health badge on cluster overview + namespace list. Color coding: green (80+), yellow (50-79), orange (25-49), red (<25).

**Dependencies:** Runs on existing topology graph from T1 fixes. No new informers needed.

**Files:**
- New: `internal/intelligence/health/scorer.go`, `health/models.go`
- New: `internal/api/rest/health_handler.go`
- New: `kubilitics-frontend/src/pages/HealthDashboard.tsx`
- Modified: `kubilitics-frontend/src/pages/ClusterOverview.tsx` (add health badge)

### T7: SPOF Inventory Dashboard (5 days)

**Pillar:** Resilience Intelligence

**Backend:** `GET /api/v1/clusters/{id}/spofs` — queries existing `IsSPOF` flag from criticality scorer, enriches with:
- SPOF reason (single replica, no PDB, sole consumer of ConfigMap/Secret, etc.)
- Blast radius score if this SPOF fails (from T2 rewritten scoring)
- Remediation recommendation (from T5)

**Frontend:** New page `/clusters/{id}/spof-inventory`:
- Sortable/filterable table: Resource | Kind | Namespace | SPOF Reason | Blast Radius | Remediation
- Filters: namespace, kind, severity
- Each row links to blast radius detail view
- Summary header: "X SPOFs found, Y critical, Z high"

**Files:**
- New: `internal/intelligence/spof/inventory.go`
- New: `internal/api/rest/spof_handler.go`
- New: `kubilitics-frontend/src/pages/SPOFInventory.tsx`

### T8: Namespace Risk Ranking (4 days)

**Pillar:** Resilience Intelligence

**Backend:** `GET /api/v1/clusters/{id}/risk-ranking` — leverages T6 health scores, inverts to risk:
- `riskScore = 100 - healthScore`
- Enriched with: SPOF count, avg blast radius, cross-NS dependency count, resource saturation %

**Frontend:** Sortable scorecard page `/clusters/{id}/risk-ranking`:
- Each namespace row expandable to show contributing factors
- Color-coded risk bars
- "Platform lead's view" — designed for prioritizing hardening work

**Files:**
- New: `internal/api/rest/risk_handler.go`
- New: `kubilitics-frontend/src/pages/RiskRanking.tsx`

### T9: Topology Diffing (10 days)

**Pillar:** Structural Intelligence

Largest task. Two sub-components:

**Snapshot storage:**
- New database table: `topology_snapshots(id, cluster_id, namespace, snapshot_json, created_at)`
- Migration in `kubilitics-backend/migrations/`
- Daily cron job serializes current topology graph to snapshot
- On-demand snapshots via `POST /api/v1/clusters/{id}/topology/snapshot`
- Retention: 90 days default, configurable via env var
- Snapshot format: `{nodes: [], edges: [], metadata: {healthScore, spofCount, timestamp}}`

**Diff engine:** New package `internal/intelligence/diff/`
- Compare two snapshots: nodes added/removed, edges added/removed, weight changes
- Output: `TopologyDiff{AddedNodes[], RemovedNodes[], AddedEdges[], RemovedEdges[], ChangedEdges[], Summary}`
- Summary: natural language — "5 new dependencies added, 2 SPOFs introduced, 1 cross-namespace edge removed"

**API:** `GET /api/v1/clusters/{id}/topology/diff?from=2026-03-01&to=2026-03-08`

**Frontend:** New tab on topology page:
- Timeline slider to select two dates
- Overlay diff on graph: green nodes/edges = added, red = removed, yellow = changed
- Summary panel with natural-language changelog
- "What changed this week" default view

**Files:**
- New: `internal/intelligence/diff/engine.go`, `diff/models.go`, `diff/snapshot_store.go`
- New: `internal/api/rest/diff_handler.go`
- New migration: `migrations/*/add_topology_snapshots.sql`
- New: `kubilitics-frontend/src/topology/TopologyDiffTab.tsx`
- Modified: `kubilitics-frontend/src/topology/TopologyPage.tsx` (add diff tab)

### T10: Pre-Apply Blast Radius (7 days)

**Pillar:** Change Intelligence

Web UI only (kcli out of scope).

**Backend:** `POST /api/v1/clusters/{id}/blast-radius/preview`
- Accepts: manifest YAML in request body
- Pipeline:
  1. `kubectl apply --dry-run=server -f -` against cluster
  2. Parse dry-run result to identify resources that would be created/modified/deleted
  3. Build hypothetical graph with proposed changes applied
  4. Diff against current graph
  5. Run blast radius scoring (T2) on changed/removed resources
  6. Return aggregate impact report

**Response:** `PreApplyResult{AffectedResources[], BlastRadius, HealthScoreDelta, NewSPOFs[], RemovedSPOFs[], Warnings[]}`

**Frontend:** "What-If" panel accessible from cluster overview:
- Drag-and-drop YAML file or paste manifest
- "Analyze Impact" button
- Shows: before/after health score, new SPOFs introduced, resources affected, blast radius visualization
- "Apply" button (with confirmation) if user proceeds

**Files:**
- New: `internal/intelligence/preview/engine.go`, `preview/models.go`
- New: `internal/api/rest/preview_handler.go`
- New: `kubilitics-frontend/src/components/blast-radius/PreApplyPanel.tsx`
- Modified: `kubilitics-frontend/src/pages/ClusterOverview.tsx` (add What-If entry point)

---

## 7. Phase 3: Executive Layer + CNCF (Tasks 11-15)

### T11: Auto-Generated Resilience Report (7 days)

**Pillar:** Resilience Intelligence

**Backend:** `POST /api/v1/clusters/{id}/reports/resilience` → generates PDF or DOCX

**Report sections:**
1. Executive summary — cluster health score, trend vs last report
2. SPOF inventory — table from T7 with blast radius and remediations
3. Namespace risk ranking — from T8, top 10 riskiest namespaces
4. Dependency map — topology graph exported as SVG (existing `export_service.go`)
5. Blast radius heatmap — top 10 highest-impact resources
6. Topology drift summary — from T9, structural changes since last report
7. Recommendations — prioritized remediation list from T5

**Tech:** Server-side PDF via Go library (`go-pdf` or `chromedp` HTML-to-PDF). DOCX via `unioffice`. Leverages existing export infrastructure in `export_service.go`.

**Frontend:** "Generate Report" button on cluster overview. Format selector (PDF/DOCX). Progress indicator. Download on completion.

**Files:**
- New: `internal/intelligence/reports/resilience.go`, `reports/pdf.go`, `reports/docx.go`
- New: `internal/api/rest/reports_handler.go`
- New: `kubilitics-frontend/src/components/reports/GenerateReportButton.tsx`

### T12: Scheduled Reports (5 days)

**Pillar:** Resilience Intelligence

**Backend:**
- New table: `report_schedules(id, cluster_id, frequency, format, webhook_url, next_run, created_at)`
- Frequencies: weekly, biweekly, monthly
- Delivery: webhook URL (Slack, Teams, email relay) with report as attachment
- Cron runner in backend — checks `next_run`, generates report via T11, delivers, updates `next_run`

**Frontend:** Settings page — create/edit/delete schedules. Preview next delivery date. Delivery history log.

**Files:**
- New migration: `migrations/*/add_report_schedules.sql`
- New: `internal/intelligence/reports/scheduler.go`
- New: `kubilitics-frontend/src/pages/ReportSchedules.tsx`

### T13: CNCF Landscape Submission (5 days)

Process and documentation only — no application code:
- Clean up README for OSS presentation
- Add CONTRIBUTING.md, CODE_OF_CONDUCT.md, GOVERNANCE.md
- Verify Apache 2.0 license is prominent
- Submit application to CNCF Landscape under "Observability & Analysis"
- Set up community channels (Discord/Slack, GitHub Discussions)

### T14: Compliance Mapping (7 days)

**Pillar:** Resilience Intelligence

New package: `internal/intelligence/compliance/`

**CIS Kubernetes Benchmark mappings:**
| Structural Finding | CIS Control | Status Logic |
|--------------------|-------------|--------------|
| SPOF with no PDB | CIS 5.2.1 | FAIL if any workload lacks PDB |
| No resource limits | CIS 5.1.1 | FAIL if any container missing limits |
| No network policy | CIS 5.3.2 | FAIL if namespace has no NetworkPolicy |
| Privileged containers | CIS 5.2.5 | FAIL if any pod runs privileged |
| No RBAC | CIS 5.1.3 | WARN if permissive ClusterRoleBindings |

**Data model:**
```go
type ComplianceResult struct {
    Framework         string              // "cis-1.8", "soc2", "hipaa"
    Controls          []ControlResult
    PassCount         int
    FailCount         int
    WarnCount         int
}
type ControlResult struct {
    ControlID         string              // "CIS-5.2.1"
    Description       string
    Status            string              // "pass", "fail", "warn"
    AffectedResources []ResourceRef
    Remediation       string
}
```

**API:** `GET /api/v1/clusters/{id}/compliance?framework=cis-1.8`

**Frontend:** Compliance dashboard — controls listed with pass/fail/warning, drill-down to affected resources. Exportable as section in resilience report (T11).

**Files:**
- New: `internal/intelligence/compliance/engine.go`, `compliance/cis.go`, `compliance/models.go`
- New: `internal/api/rest/compliance_handler.go`
- New: `kubilitics-frontend/src/pages/ComplianceDashboard.tsx`

### T15: Public Launch (5 days)

Marketing and community — no application code:
- Blog post: "Introducing Kubilitics — Operational Intelligence for Kubernetes"
- Product Hunt launch
- Hacker News Show HN post
- KubeCon CFP submission
- Social media targeting K8s architects and platform engineers

---

## 8. What to STOP (Scope Discipline)

Per the strategic report, the following are explicitly out of scope:

- **No new resource type pages** — 140+ pages are enough
- **No addon marketplace work** — focus on intelligence features
- **No i18n** — English-only until product-market fit
- **No desktop app polish** — web app is the enterprise product
- **No cost dashboard** — Kubecost/CAST AI own this space
- **No kcli changes** — separate component, separate plan

---

## 9. New Package Structure

```
internal/
  intelligence/          # NEW — all intelligence features
    health/              # T6: Structural health scores
      scorer.go
      models.go
    spof/                # T7: SPOF inventory
      inventory.go
    diff/                # T9: Topology diffing
      engine.go
      models.go
      snapshot_store.go
    preview/             # T10: Pre-apply blast radius
      engine.go
      models.go
    reports/             # T11-T12: Resilience reports + scheduling
      resilience.go
      pdf.go
      docx.go
      scheduler.go
    compliance/          # T14: Compliance mapping
      engine.go
      cis.go
      models.go
  graph/                 # EXISTING — modified
    scoring.go           # T2: Rewritten scoring model
    remediation.go       # T5: NEW — remediation engine
    builder.go           # T2, T4: Modified
    snapshot.go          # T2, T4: Modified
  topology/v2/           # EXISTING — modified
    builder/
      depth_filter.go    # T1: Expand depth fix
    filter.go            # T1: Hub threshold, sibling filter fixes
  pkg/topologycache/
    cache.go             # T1: Cache key fix or deprecation
```

---

## 10. API Surface Summary

| Endpoint | Method | Task | Description |
|----------|--------|------|-------------|
| `/api/v1/clusters/{id}/health` | GET | T6 | Structural health scores |
| `/api/v1/clusters/{id}/spofs` | GET | T7 | SPOF inventory |
| `/api/v1/clusters/{id}/risk-ranking` | GET | T8 | Namespace risk ranking |
| `/api/v1/clusters/{id}/topology/snapshot` | POST | T9 | Create topology snapshot |
| `/api/v1/clusters/{id}/topology/diff` | GET | T9 | Compare two snapshots |
| `/api/v1/clusters/{id}/blast-radius/preview` | POST | T10 | Pre-apply blast radius |
| `/api/v1/clusters/{id}/reports/resilience` | POST | T11 | Generate resilience report |
| `/api/v1/clusters/{id}/reports/schedules` | CRUD | T12 | Report schedule management |
| `/api/v1/clusters/{id}/compliance` | GET | T14 | Compliance check |

---

## 11. Testing Strategy

Each task includes:
- **Unit tests** for new packages (Go: `*_test.go` alongside source)
- **Regression tests** for bug fixes (T1: one test per topology bug)
- **Integration tests** for API endpoints (extend `tests/integration/`)
- **Frontend component tests** (Vitest for new pages/components)

No E2E tests for intelligence features in Phase 1-2 — too dependent on cluster state. E2E added in Phase 3 with fixture clusters.
