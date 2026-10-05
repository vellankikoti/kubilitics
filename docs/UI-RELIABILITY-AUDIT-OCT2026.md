# UI Reliability Audit — October 2026

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

## Flagged for next-round audit (not yet investigated this session)

Found via grep for the same "0 `isError` usages despite aggregating live
data" signature. Not confirmed bugs — just unaudited:

- `ProjectDetailPage.tsx`, `ProjectDashboardPage.tsx` — project rollup pages
- `SPOFInventory.tsx`, `RiskRanking.tsx` — Blast Radius intelligence pages
- `AutoPilotDashboard.tsx`
- `ReportSchedules.tsx`
- `Topology.tsx` — appeared in the original "Failed to load" grep; worth
  checking given its complexity and the namespace-crash's topology-adjacency

## Not started

- Systematic Headlamp pattern comparison (informer-cache-first data model,
  error UX conventions, perf characteristics) — the user's "100x better in
  every aspect" ask. This is a multi-session effort; recommend scoping it as
  its own investigation phase rather than folding into this patch.

## Release plan

Per the established versioning policy (PATCH for small fixes, MINOR only for
genuinely major capability — see `v1.2.2`'s precedent): this session's two
commits are a coherent, tested, low-risk patch.

- **v1.2.3** (this branch, ready once typecheck/build verified): the 2
  commits above. Both are defensive/additive — no behavior change on any
  currently-working path, only activates previously-dead error handling and
  adds guards against a crash class.
- **Next**: finish auditing the "flagged for next-round" pages above. If
  they turn up the same bug class, bundle into v1.2.4. If something
  genuinely architectural turns up (e.g. a real root cause for `ns-spine`
  via live repro), that may warrant its own release depending on blast
  radius.
- **Later / separate initiative**: Headlamp-grade stability/performance
  pass — scope this as its own multi-session campaign with its own
  checkpoint structure, not bundled into patch releases.
