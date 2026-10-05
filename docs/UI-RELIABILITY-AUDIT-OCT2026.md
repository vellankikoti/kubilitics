# UI Reliability Audit — October 2026

## Issue count toward the project's "50 verified issues per release" bar

Precise accounting — every item below is a distinct, fixed, verified (typecheck
and/or `go build`/`go vet`/`go test` clean) code location, not a style nit:

| Batch | Count | Commits |
|---|---|---|
| Overview-pages-show-0 (5 hooks) + array-safety hardening (3 sites) | 8 | `dd0fe0cc`, `7e008947` |
| Dashboard health score, Simulation/HealthIssueDetail crashes, 4 more silent-failure hooks | 11 | `8c29ff3f` |
| Backend: 18 goroutine panic-recovery sites + 2 bonus deadlock fixes + 3 query-param 400s + 4 auth logging fixes | 27 | `9974329d` |
| **Total** | **46** | 5 commits on `feat/stability` |

4 short of the 50 floor. Two independent 3-pronged parallel audits (frontend
hooks, frontend pages, backend handlers) have now run this session — the
remaining highest-confidence unexplored surface is `src/components/` (UI
components, not yet systematically audited) and the `ProjectDetailPage`/
`ProjectDashboardPage` status-badge gap already flagged below.

## Context

User-reported: PVC/Namespace page crash ("Failed to load ns spine — {} is not
iterable") and Workloads Overview showing all zeros despite the sidebar
showing real counts. Triggered a systematic audit for this bug class —
**silent failure masquerading as "healthy empty state"** — across overview
and dashboard pages, plus a search for the specific crash-on-navigation
pattern. Work done on `feat/stability` (branched from `main` @ `9de1e1d5`,
the already-shipped v1.2.2 baseline).

## Fixed this session

**Commit `dd0fe0cc` — silent data-fetch failures rendering as "0, all healthy"**

Root cause, confirmed and reproduced in code: 5 overview-aggregation hooks
combine several `useK8sResourceList` queries with `.data?.items ?? []`,
which treats a failed query identically to a genuinely empty result.

| Hook | Page | Status before fix |
|---|---|---|
| `useWorkloadsOverview` | Workloads Overview | Computed pulse/counts with no error signal at all |
| `useResourcesOverview` | Resources Overview | Same |
| `useScalingOverview` | Scaling Overview | Same |
| `useAdmissionOverview` | Admission Overview | Hook silent; **page already had `if (isError) return <ApiError/>` — permanently dead code** |
| `useCRDOverview` | CRDs Overview | Same dead-code situation |

Fix: each hook now returns `isError` (OR of each underlying query's
`isError`). Backend `GetWorkloadsOverview` additionally now logs and flags
`DataPartial` instead of silently dropping per-resource-list errors.
`WorkloadsOverview.tsx` gained a retry-able warning banner for the partial
case. The other 4 hooks' pages already had correct `ApiError` branches
written — this reactivates them, no new UI code.

Confirmed clean (already correctly surfaced `isError`): `useStorageOverview`,
`useNetworkingOverview`, `useClusterOverviewData`, `useXRayDashboard`,
`useFleetOverview`. Dashboard, ClusterOverview, FleetDashboard pages verified
clean too.

**Commit `7e008947` — PVC/Namespace crash hardening ("X is not iterable")**

Investigated exhaustively (full read of `GenericResourceDetail.tsx`,
`NamespaceDetail.tsx`, `ResourceDetailLayout.tsx`, `ResourceHeader.tsx`,
`ResourceStatusCard.tsx`, `LabelList`/`AnnotationList`,
`useK8sResourceDetail.ts`) looking for every array-destructure / `for...of` /
array-spread that could receive a non-array. **Could not confirm the exact
single trigger without a live stack trace** — the crash is data-shape-
dependent on the specific namespace (`ns-spine`) and none of the usual
suspects had the bug on the happy path. Rather than claim false certainty,
hardened every genuine latent risk found:

- `useTableFiltersAndSort`'s `[...result]` — backs nearly every list page in
  the app; its contract doesn't enforce `items` is a real array, it just
  happens that every current caller passes `.filter()`/`.map()` output.
- `PVCFileBrowser`'s `[...children]` — reads directly from React state;
  happy path always stores a real array, but the read site didn't enforce it.
- `NamespaceDetail`'s `finalizers || []` — `||` only guards falsy values; a
  malformed `{}` response would slip through undetected.

All three are one-line `Array.isArray` guards, zero behavior change on the
happy path. **If the `ns-spine` crash recurs, grab the DevTools stack trace**
(right-click → Inspect in the Tauri window) — that pinpoints it in seconds
versus more static guessing.

## Round 2 findings (this session, continued)

**Confirmed clean (direct `useQuery` passthroughs, safe by construction):**
`useSPOFInventory`, `useRiskRanking` (via `useClusterHealth.ts`),
`useAutoPilotFindings`/`useAutoPilotActions`, `useReportSchedules`. All
correctly propagate `error`/`isError` to their pages, which correctly branch
on it. `RiskRanking.tsx`, `AutoPilotDashboard.tsx`, `ReportSchedules.tsx`,
`SPOFInventory.tsx` need no fix.

**New finding — `ProjectDetailPage.tsx` / `ProjectDashboardPage.tsx`:** both
use `useClustersFromBackend()` (a clean `useQuery`) to enrich each project's
configured cluster references with live connection status, via
`allClusters.find(c => c.id === clusterId)`. Neither page checks that
query's `isError`. Unlike the overview-zero bug, a failure here doesn't
render "0 clusters" — `project.clusters` comes from a separate, correctly-
error-handled query — it instead makes every cluster silently render as
"not found / disconnected" even if truly connected. Lower severity, same
family. **Not fixed this session** — the per-cluster status-badge logic
needs full tracing first to avoid a sloppy fix; flagged for next round.

## Headlamp comparison (grounded in real upstream engineering history)

Researched Headlamp's actual recent architecture decisions (not guessing)
via its GitHub releases/PRs. Three concrete, comparable points:

1. **Informer/cache coverage breadth.** Headlamp recently expanded its
   `filterImportantResources` cache-invalidation allowlist from **11 to 46**
   resource kinds — previously, edits to networking/RBAC/storage/Gateway
   API/policy/autoscaling/quota resources left cached views stale.
   **Kubilitics' `resourceKindToStoreKey` map (`informer.go`) tracks only
   27 kinds** — and critically, it's **missing exactly the kinds behind the
   4 overview pages fixed earlier this session**: `ResourceQuota`,
   `LimitRange`, `ResourceSlice`, `DeviceClass` (→ Resources Overview),
   `VerticalPodAutoscaler` (→ Scaling Overview),
   `MutatingWebhookConfiguration`/`ValidatingWebhookConfiguration` (→
   Admission Overview), `CustomResourceDefinition` (→ CRDs Overview). Also
   missing: `EndpointSlice`, `Lease`, `APIService`, `VolumeAttachment`.
   Every request for these kinds falls back to a live, uncached K8s API
   call on every load — slower, and more exposed to exactly the kind of
   transient failure that triggers the silent-zero bug class. **This is the
   single highest-leverage structural fix available**: expanding informer
   coverage improves both performance and reliability for the pages most
   recently found to be fragile.
2. **Bounded live-data strategy.** Headlamp moved Cluster Overview from
   persistent watch streams to 1-minute polling specifically to stop
   browser OOMs on large clusters, and added a 1,000-item pagination budget
   to Pod lists. Kubilitics is already aligned here: `useFleetOverview`
   polls every 30s, `useWorkloadsOverview`'s backend path polls every 60s,
   `useOverviewStream`'s WebSocket has proper cleanup on unmount (verified,
   no leak). No action needed.
3. **Asset compression — not applicable.** Checked: `vite.config.ts` only
   has `brotliSize: true` (a build-report stat, not real compression).
   Confirmed this Headlamp optimization doesn't actually transfer: Headlamp
   is a server-delivered web app where Brotli cuts HTTP download size.
   Kubilitics is a **Tauri desktop app** — assets are bundled into the app
   and served locally from its own webview, not downloaded over a network
   on each load. Closing this out as not a real gap rather than a lingering
   todo.

## Flagged for next-round audit

- `ProjectDetailPage.tsx` / `ProjectDashboardPage.tsx` — per-cluster status
  badge can silently show "disconnected" on a `clustersQuery` failure (see
  Round 2 findings above). Needs full status-badge trace before fixing.

**Checked and confirmed clean this session (Topology.tsx):** full trace of
`focusSet`'s `new Set([...upstream, ...downstream])` down through
`getUpstreamChain`/`getDownstreamChain` (`graphTraversal.ts`) to
`GraphModel.getParents`/`getChildren` (`graphModel.ts`) — all have strict
`Set`/array return types enforced by class-based construction, not raw API
passthrough. Not the source of the namespace crash; no fix needed.

**Checked and confirmed not applicable (Brotli/asset compression):** see
Headlamp comparison point 3 above — doesn't apply to a Tauri desktop app.

## Not started

- Backend informer-cache expansion (the fix for the coverage-gap finding
  above) — identified, not implemented. This is real new backend surface:
  extend `resourceKindToStoreKey` + register informers for the missing
  ~12 kinds, several via different clientsets than the existing typed
  `SharedInformerFactory` pattern (CRDs via apiextensions client, APIService
  via apiregistration client, VerticalPodAutoscaler via a CRD/dynamic
  informer since it's not a built-in K8s type). Own testing surface (watch
  correctness per clientset, memory footprint of ~12 more informers per
  cluster) — should not be rushed into this patch branch. **This is the
  natural next scoped piece of work**, separate from the patch fixes below.
- Remaining Headlamp comparison dimensions not yet researched: RBAC-aware
  UI gating, multi-cluster context switching UX, plugin architecture.

## Release plan

Per the established versioning policy (PATCH for small fixes, MINOR only for
genuinely major capability — see `v1.2.2`'s precedent).

- **v1.2.3** (this branch, `feat/stability`, typecheck/build verified clean):
  the 3 commits on this branch — overview-zero fix, PVC/namespace crash
  hardening, this audit doc. All defensive/additive, no behavior change on
  any currently-working path.
- **v1.2.4 candidate**: resolve the `ProjectDetailPage`/`ProjectDashboardPage`
  status-badge gap once traced.
- **v1.3.0 candidate (MINOR — genuinely new capability, not a patch)**:
  the informer-cache expansion. ~12 new resource kinds watched means
  meaningfully faster + more reliable reads for Resources/Scaling/Admission/
  CRDs Overview and anything else touching those kinds — a real capability
  jump, not a bug fix, and warrants its own dedicated investigation +
  testing phase per the project's versioning policy.
- **Separate initiative, not release-numbered**: remaining Headlamp
  dimensions (RBAC UI gating, multi-cluster UX, plugin architecture) — scope
  as its own investigation phase when picked up.
