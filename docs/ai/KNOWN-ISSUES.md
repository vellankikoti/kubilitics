# Known Issues (Kubilitics)

Tracked, intentionally deferred items. Check here before "discovering"
these again.

## Deferred — needs its own task, not a quick fix

### Informer-cache coverage gap — CLOSED
Backend informer cache (`resourceKindToStoreKey` in `informer.go`) tracked
27 resource kinds; Headlamp's comparable cache-invalidation allowlist
covers 46. All 11 previously-missing kinds are now cached.

8 (all GA/stable APIs, zero new go.mod dependencies): `ResourceQuota`,
`LimitRange`, `EndpointSlice`, `Lease`, `VolumeAttachment`,
`MutatingWebhookConfiguration`, `ValidatingWebhookConfiguration` (same
`SharedInformerFactory` as everything else) and `CustomResourceDefinition`
(separate apiextensions clientset built from `client.Config` in
`NewInformerManager` — already a vendored dependency via
`internal/addon/helm/uninstall.go`'s existing use of the same clientset).
Regression tests: `internal/k8s/informer_coverage_gap_test.go`.

The final 3, previously deferred as "needs its own task" (Phase 4 of
`docs/ai/STABILIZATION-PLAN.md`), closed:
- `APIService` — added `k8s.io/kube-aggregator` (exact version match to
  the existing `k8s.io/*` v0.35.1 pins, no transitive bump). Always
  constructed: apiregistration.k8s.io is a core, always-present API group.
- `VerticalPodAutoscaler` — added
  `k8s.io/autoscaler/vertical-pod-autoscaler` v1.6.0 (also an exact
  client-go v0.35.1 match; v1.7+ requires client-go v0.36+). Gated on
  `clusterServesResources` discovery first — most clusters don't have the
  VPA CRDs installed, and building the informer anyway (without checking)
  was tried first and found to retry a 404 forever, spamming logs; fixed
  before landing.
- `ResourceSlice` / `DeviceClass` (DRA) — no new dependency; client-go
  v0.35.1 already has these informers. `discoverDRAVersion` negotiates
  the version (`v1`/`v1beta2`/`v1beta1`) a given cluster actually serves
  at startup instead of hardcoding one, avoiding the permanent-sync-
  failure risk this item was originally deferred over.

All three use a new generalized `auxInformerFactory` + `auxByKind`
mechanism (generalizing the original hand-written `crdFactory`/
`crdSynced` pair) so a kind's RBAC gap, missing CRD, or version mismatch
can only ever disable ITS OWN cache-first path, never the main ~34-kind
cache or another aux kind. Regression tests:
`internal/k8s/informer_aux_factories_test.go`.

## Pre-existing, confirmed not regressions

- **CORRECTION (2026-10-10):** `ClusterPickerPage.test.tsx` ×2 and
  `AddClusterDialog.test.tsx` ×1, previously filed here as "confirmed
  pre-existing, not a regression" (parallel-run flakiness), turned out to
  be three different deterministic bugs, not flakiness — fixed:
  - `ClusterPickerPage.test.tsx`'s reachability test used `getByLabelText`
    against an element with only a `title` attribute — that query never
    matches `title` at all, so the assertion was silently vacuous, not
    flaky. Fixed to `getByTitle`.
  - Its empty-state test asserted text ("No clusters found") the
    component has never rendered (actual: "No clusters detected") — a
    typo-level mismatch. It also asserted an empty-state "Add a cluster"
    dialog trigger that doesn't exist in the current two-pane layout;
    that half is a product decision (add the feature vs. drop the
    assertion), so it's split out and `it.skip`'d with an explanation
    rather than resolved unilaterally.
  - `AddClusterDialog.test.tsx` mocked `@/services/backendApiClient` (a
    re-export facade) while the component imports
    `addClusterWithUpload` from `@/services/api/clusters` directly —
    `vi.mock` intercepts by exact specifier, so the mock never took
    effect and every run hit a real (failing, no backend in jsdom) fetch.
  All four now pass deterministically; see
  `kubilitics-frontend/src/pages/ClusterPickerPage.test.tsx` and
  `.../src/components/cluster/AddClusterDialog.test.tsx`. If a test is
  filed here as "pre-existing flaky" again, verify it actually IS
  nondeterministic (reruns disagree) before accepting that label — a
  wrong query or wrong mock target reproduces as a 100%-consistent
  failure, which parallel-suite noise can look like at a glance.
- `kubilitics-desktop/package.json`'s own `version` field has never been
  part of `scripts/bump-version.sh`'s 6 tracked files — stays at `1.0.0`,
  not a bug.

## Fixed, for reference (don't re-investigate)

- **P0 — unreachable-cluster identity collision causing app-wide freeze
  (2026-10-10).** User-reported: clicking an inaccessible/unreachable
  cluster showed a stuck "loading" state, and the rest of the app
  (navigating to Fleet, any interaction) stopped responding. Reproduced
  live end-to-end: a real backend process + frontend dev server + a
  kubeconfig context pointed at a network-blackholed IP (TCP SYN never
  answered — the worst-case unreachable shape) + Playwright driving an
  actual click. Before the fix: clicking the cluster card never navigated
  within 30s. Root cause, found by tracing the actual identity data, not
  assumption: `addClusterWithSource` in `cluster_service.go` left
  `ServerURL` as `""` whenever the live connection test failed — it was
  only ever set inside the success branch. But
  `kubeconfig_source.go`'s `Enumerate()` reads the exact same
  `server:` field straight off the kubeconfig file (no network call) for
  the same cluster. Result: the moment any cluster is unreachable, it
  gets TWO different `identity.LogicalIdentity{Name, ServerURL}` keys
  system-wide — a real one from discovery, an empty one from the
  registered/manual record — so `discovery.Manager`'s dedup-by-key never
  merges them. The frontend renders the discovery-sourced entry (the one
  the user actually sees/clicks) with none of the enrichment that landed
  on the other, invisible entry — no `session_id`, no `kubeconfig_path`.
  Clicking it then re-POSTed `/api/v1/clusters` with
  `ClusterPickerPage.tsx`'s wrong fallback path
  (`c.kubeconfigPath ?? '~/.kube/config'`), producing the confusing stuck
  interaction. Fixed by `serverURLFromKubeconfig()` seeding `ServerURL`
  from the kubeconfig's declared field before the connection test runs,
  mirroring discovery's own parse, so the identity never depends on
  reachability. Confirmed fixed live: click now resolves in ~5s (the
  pre-existing bounded connection-test timeout), and Fleet
  navigation/page interactivity stay fully responsive throughout.
  Regression test:
  `TestClusterService_AddCluster_UnreachableClusterKeepsDeclaredServerURL`.
  **If a user reports "clicking a cluster freezes everything" again,
  check for a NEW instance of split identity first** — grep for any other
  place a `LogicalIdentity`-bearing struct gets built with a field that's
  only populated on a successful network call.

- Cluster detection/sync reliability (Headlamp comparison pass, 2026-10-10)
  — three root causes behind "cluster detection is unreliable and
  confusing," found by comparing Kubilitics' presence architecture against
  Headlamp's and verifying every claim against the actual code:
  1. `clusterSwitch.ts`'s invalidation bus (9 Zustand stores subscribed,
     fully tested) was never actually triggered — `emitClusterSwitch()`
     had zero production call sites, only test-file ones. Every
     cache-holding store silently kept the previous cluster's data after a
     switch. Fixed by emitting from inside
     `clusterPresenceStore.setActiveByLogicalIdentity()`, the one choke
     point every switch call site (`ClusterPickerPage`, `Header`,
     `useAutoConnect`, `Settings`, `FleetDashboard`, ...) funnels through.
  2. `discovery.Manager.Snapshot()`'s `Connected` list was hardcoded empty
     with a comment deferring it as needing a future `ConnectionManager` —
     but `SetReachabilityChecker`'s own doc comment already describes
     exactly that session tracking (ClusterService's live client
     registry). `Connected` is now derived as the subset of `Registered`
     the checker reports reachable.
  3. The real mechanism behind "recreating a kind cluster outside the app
     takes up to 60s to show up": each watch source (e.g.
     `KubeconfigFileSource`'s fsnotify handler) diffs its own prev/curr
     state independently of `Manager.discovered`, which only
     `Refresh()` rebuilds — previously called only on explicit
     add/remove or the 60s tick, never on a raw watch event. The
     frontend's SSE handler re-fetches `Snapshot()` the instant any event
     arrives, so it raced ahead of the next scheduled `Refresh()` and got
     stale data. `Manager.Events()` now calls `Refresh()` before
     forwarding each event.
  Also fixed an unrelated flaky `-race` failure in
  `manager_bounded_refresh_test.go`'s `hangingSource` test helper
  (check-then-close on a channel, racy under concurrent `Enumerate`
  calls — `sync.Once` instead), found while stress-testing the above.
  Kubilitics' identity model (name+serverURL logical identity surviving
  session-UUID churn) and SSE-push architecture were confirmed already
  sound and arguably ahead of Headlamp's polling model — the
  unreliability was wiring, not architecture.

- Vitest 2→4 / Vite 5→6 migration — cleared both critical CVEs
  (tinypool's prototype-pollution RCE; vitest's own UI-server file-read
  CVE — vitest 4 dropped tinypool as a dependency entirely). Done in an
  isolated worktree given the documented history of a prior attempt
  breaking 107/108 suites at the collection level; this attempt hit zero
  collection failures. Landed on `vite@6.4.4` + `vitest@4.1.11` +
  `@vitejs/plugin-react-swc@4.3.3` (the documented minimum-risk target —
  vitest 4 hard-requires vite ≥6 — not the latest vite 8/vitest 5, to
  keep the jump as small as the CVE fix needs). Two real issues surfaced
  and were fixed at the source, not papered over in tests:
  - `src/lib/topologyExport.test.ts` — vitest 4's mock "new" handling
    requires a `function`/`class` implementation (not an arrow function
    returning an object literal) for `new FileReader()` to correctly
    delegate to the mock factory.
  - `src/services/api/client.ts`'s `isRequestTimeout()` — `instanceof
    DOMException` is realm-fragile: Node's built-in
    `AbortSignal.timeout()` constructs its DOMException via an internal
    binding that isn't the same class reference as a jsdom test
    environment's `globalThis.DOMException`. Fixed by duck-typing on
    `.name === 'TimeoutError'` instead, which is also more robust in
    production (any multi-realm context, not just tests).

  `npm install` hit an unrelated `npm` 11.4.2 arborist bug (`Cannot read
  properties of null (reading 'edgesOut')`) on this specific optional-peer
  graph — worked around with `--legacy-peer-deps` for the one install
  command; the resulting lockfile installs fine with plain `npm ci`
  afterward. Verified: 965/968 tests passing (3 pre-existing failures,
  unchanged), typecheck clean, lint clean, both `vite build` and
  `TAURI_BUILD=true vite build` succeed, dev server boots, 0 critical
  `npm audit` findings (down from 2).
- ProjectDetailPage / ProjectDashboardPage status-badge gap — both pages
  now surface `useClustersFromBackend()`'s `isError` as a dismissible
  retry banner instead of silently discarding it. Root cause traced to
  the *callers*, not the hook (it already exposed `error`). Both pages
  also had the same real functional bug in their Connect handler: it
  gated the Connect action on finding the cluster in the enrichment list
  (`allClusters.find(...)`), so a metadata-fetch failure made Connect a
  silent no-op even though `clusterId` was already known from
  `project.clusters`. Fixed by connecting directly on the known id;
  enrichment data is now display-only for the success-toast name.
  **Caught two different ways**: `ProjectDashboardPage`'s fix was unit
  tested first (`src/pages/ProjectDashboardPage.test.tsx`); the identical
  bug in `ProjectDetailPage.handleConnect` was missed in the first pass
  (only its error *message* was improved, not the no-op itself) and only
  surfaced by live end-to-end testing — backend + frontend started
  against a real kind cluster, driven with Playwright, clusters API
  mocked to fail. Lesson: when the same bug class appears on two pages,
  verify the fix landed identically on both, not just the one with a
  test.
- 81 silent-data-failure / panic-recovery / resource-leak / cluster-switch-
  race fixes shipped in v1.2.3 — see
  `docs/releases/v1.2.3-RELEASE-REPORT.md` and
  `docs/UI-RELIABILITY-AUDIT-OCT2026.md` for full detail if the exact
  file/fix is needed again.
- Security audit (XSS, SQLi, path traversal, RBAC, secret-logging) —
  clean, confirmed via full adversarial pass, not unexamined.
- Concurrency audit (data races, unprotected shared state) — clean,
  confirmed via full trace of every goroutine-touching lifecycle manager.
