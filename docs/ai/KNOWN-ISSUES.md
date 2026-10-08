# Known Issues (Kubilitics)

Tracked, intentionally deferred items. Check here before "discovering"
these again.

## Deferred — needs its own task, not a quick fix

### Informer-cache coverage gap — partially fixed (3 kinds remain)
Backend informer cache (`resourceKindToStoreKey` in `informer.go`) tracked
27 resource kinds; Headlamp's comparable cache-invalidation allowlist
covers 46. **8 of the 11 missing kinds are now cached** (all GA/stable
APIs, zero new go.mod dependencies): `ResourceQuota`, `LimitRange`,
`EndpointSlice`, `Lease`, `VolumeAttachment`,
`MutatingWebhookConfiguration`, `ValidatingWebhookConfiguration` (same
`SharedInformerFactory` as everything else) and `CustomResourceDefinition`
(separate apiextensions clientset built from `client.Config` in
`NewInformerManager` — already a vendored dependency via
`internal/addon/helm/uninstall.go`'s existing use of the same clientset).
Regression tests: `internal/k8s/informer_coverage_gap_test.go`.

**Still deferred, genuinely needs its own task:**
- `APIService` — needs `k8s.io/kube-aggregator`, not currently a
  dependency.
- `VerticalPodAutoscaler` — `autoscaling.k8s.io` has no typed client in
  client-go; needs the VPA project's own generated client (also not
  vendored).
- `ResourceSlice` / `DeviceClass` (DRA) — client-go v0.35.1's typed
  informer targets `resource.k8s.io/v1` (GA), but the existing dynamic-
  client GVR fallback in `discovery.go` targets `v1alpha3`/`v1`
  inconsistently across the two kinds, which signals real version-skew
  risk: wiring the `v1` typed informer could permanently fail to sync
  (and silently disable the whole cache-sync gate — see `waitForSync`)
  on any cluster that doesn't yet serve DRA as GA. Needs explicit version
  negotiation before this is safe to add.

## Pre-existing, confirmed not regressions

- `ClusterPickerPage.test.tsx` ×2, `AddClusterDialog.test.tsx` ×1 — see
  `docs/ai/TESTING.md`. Verified pre-existing via baseline worktree
  comparison against `e1acb339`.
- `kubilitics-desktop/package.json`'s own `version` field has never been
  part of `scripts/bump-version.sh`'s 6 tracked files — stays at `1.0.0`,
  not a bug.

## Fixed, for reference (don't re-investigate)

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
