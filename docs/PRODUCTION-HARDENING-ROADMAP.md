# Kubilitics Production Hardening Roadmap

**Companion document to:** `docs/PRODUCTION-RELIABILITY-AUDIT.md` — read that first. Every phase below references finding IDs from that audit.

**Status: Phase 0, 1, 2, 3, 4, 5, 6, 7, 8, and 9 complete (2026-10-02). Phase 11 complete (fixed 2026-10-05 in commit `0a7761a0`, independently re-verified 2026-10-13 against 8 deterministic isolation tests + `go test -race` — see that commit and `internal/api/websocket/contam1_isolation_test.go`/`informer_broadcast_test.go`).** Phase 10's Release Gate Criterion 1 (no open P0/P1 finding) is therefore satisfied as far as CONTAM-1 is concerned; Phase 10 itself has not been formally re-run end-to-end against the full P0/P1 inventory. Phases execute one at a time, in order, each with its own scope confirmation, tests, and before/after evidence — never silently batched.

---

## PHASE 0 — Baseline

**Goal:** Reproduce and measure the failures this audit traced statically, against a real cluster (or a realistic multi-context kubeconfig with deliberately-unreachable entries), to convert UNVERIFIED/HIGH-confidence findings into CONFIRMED and replace configured-value guesses with real measurements.

**Scope:**
- Reproduce STARTUP-1 with N unreachable kubeconfig contexts; measure actual wall-clock time to first HTTP response.
- Reproduce COUNTS-1's `DeletedFinalStateUnknown` path against a real informer under relist/churn conditions (the current evidence is a scratch repro of the control-flow, not a live-cluster observation).
- Reproduce TOPOLOGY-1/2 against a cluster large enough that a full build exceeds 8s.
- Attempt to reproduce the exact reported "32588369311109.9 GiB / 0% used" value, or confirm it cannot be reproduced with current inputs and the structural risk (METRICS-2) stands as the documented cause class instead.
- Resolve the two explicitly UNVERIFIED caveats from the audit: FLEET-1's "registered-but-now-slow cluster" scenario, and BLASTRADIUS-3's lock-contention impact.

**Gate:** Every P0/P1 finding is either reproduced with a measurement, or explicitly re-labeled UNVERIFIED with a stated reason it couldn't be reproduced in this environment. No phase 1 work starts until this baseline exists.

---

## PHASE 1 — Reliability Foundation ✅ COMPLETE (2026-10-02)

**Status: DONE.** See `docs/PRODUCTION-HARDENING-EXECUTION.md` for full evidence. 6 of 7 findings fixed; LOADING-4 explicitly BLOCKED (the naive fix would have broken every connected cluster's informer watch connections — documented, not forced through).

**Goal:** No critical user operation can hang indefinitely. This is Root Cause A from the audit — timeout/cancellation enforcement becomes structural instead of opt-in.

**Findings addressed:** LOADING-1, LOADING-2, LOADING-3, LOADING-4, LOADING-5, TOPOLOGY-2 (timeout-mismatch half), LIFECYCLE-2 (timeout half)

**Scope:**
- Add a hard client-side timeout (`AbortSignal.timeout()`) inside `backendRequest`/`backendRequestText` in `kubilitics-frontend/src/services/api/client.ts`, matching the existing `getHealth()` pattern. This is the single highest-leverage change in the whole audit.
- Thread React Query's per-query `signal` into `backendRequest` calls so unmount-cancellation works, independent of the hang-timeout fix.
- Fix `GetClusterOverview`'s fallback path (`overview.go:50-66`) to go through the existing `withTimeout()`-wrapped client methods instead of raw `clientset` calls.
- Set `rest.Config.Timeout` at K8s client construction time (`internal/k8s/client.go`) as defense-in-depth alongside the existing per-call wrapper.
- Bound `WaitForCacheSync` with a timeout + retry/backoff, and surface permanent sync failure as a visible per-cluster signal instead of silent fallback.
- Reconcile the topology client/server timeout mismatch (8s client vs. 30s server) and make the client's give-up a real cancellation via `AbortController`.
- Fix the cluster-removal dialog's asymmetric `onError`/`onSuccess` cleanup in `Settings.tsx`.

**Gate:** No critical user operation (resource list/detail load, dashboard overview, topology fetch, cluster removal) can hang past its configured timeout without surfacing an explicit error state to the user. Demonstrate this against a deliberately-hung mock backend for each of the 5 flows above.

---

## PHASE 2 — Cluster Lifecycle ✅ COMPLETE (2026-10-02)

**Status: DONE.** See `docs/PRODUCTION-HARDENING-EXECUTION.md` ("Phase 2 — Cluster Lifecycle: Execution Record") for full evidence. Summary: STARTUP-1, HEALTH-1, HEALTH-2 fixed; LIFECYCLE-1 locked in with a new regression test; CONTAM-1 explicitly deferred (dead/inert code, resuming it is a larger unit of work than this phase's scope, documented rather than left ambiguous).

**Goal:** Unavailable clusters cannot block healthy clusters; removal works even when the remote cluster is offline; health represents a real, freshness-tracked check.

**Findings addressed:** STARTUP-1, HEALTH-1, HEALTH-2, CONTAM-1, (LIFECYCLE-1 — already holds, lock in with a test)

**Scope:**
- Parallelize `LoadClustersFromRepo` (`cluster_service.go:706-753`) using the same bounded `errgroup` fan-out pattern already correctly implemented in `fleet.go`'s `GetFleetOverview`. Consider also binding the HTTP listener before cluster connectivity resolves, matching the Headlamp/Lens-style async-connect pattern frontend comments already reference.
- Wire `discovery.Manager.Snapshot()`'s `Reachable` field to a real check (reuse `ClusterService.Status` or a lightweight preflight) instead of the current hardcoded `true`.
- Add `LastCheckedAt`/`LastSuccessAt`/`Latency`/`LastError` fields to the presence wire type; populate from the real check; render relative staleness in the Fleet/cluster-picker UI instead of a flat boolean.
- Before resuming any WebSocket resource/topology broadcast work: wire `ServeWS` to read `cluster_id` (or require a subscribe message) and tag every `BroadcastResourceEvent` call with the real originating cluster ID. If this work is not prioritized this phase, explicitly mark CONTAM-1 as "deferred, feature remains inert" rather than silently leaving it ambiguous.
- Add the regression test locking in LIFECYCLE-1's already-correct guarantee (removal succeeds without a remote dial).

**Gate:** Demonstrate: (a) N unreachable clusters in kubeconfig do not add more than a small constant delay to app startup; (b) removing a cluster while its remote API server is unreachable completes immediately and is idempotent; (c) a cluster that goes offline or is removed stops showing as "Reachable"/"Healthy" within one check interval, with a visible last-checked timestamp.

---

## PHASE 3 — Data Correctness ✅ COMPLETE (2026-10-02)

**Status: DONE.** See `docs/PRODUCTION-HARDENING-EXECUTION.md` ("Phase 3 — Data Correctness: Execution Record") for full evidence. Summary: COUNTS-1 fixed (tombstone unwrap + periodic self-heal reconciliation); METRICS-1 fixed (replaced hardcoded-format parsers with `resource.ParseQuantity`); METRICS-2 fixed for 6 of 8 audited files (2 were already correctly guarded on re-verification and left untouched) via a new canonical `k8sQuantity.ts` utility; COUNTS-2 and METRICS-3 locked in with regression tests (already correct).

**Goal:** Metrics are correct, unit-safe, and traceable to source; invalid values are rejected rather than silently displayed.

**Findings addressed:** COUNTS-1, METRICS-1, METRICS-2, (COUNTS-2, METRICS-3 — already correct, lock in with tests)

**Scope:**
- Fix `updatePodStatus` (`overview_cache.go:292-366`) to either recompute from `informer.GetStore("Pod").List()` like every sibling counter, or add periodic full reconciliation against the store; unwrap `cache.DeletedFinalStateUnknown` before the type assertion regardless.
- Replace `parseMemoryToMi`/`parseCPUToMilli` (`internal/metrics/aggregate.go`) with `resource.ParseQuantity(...).Value()`/`.MilliValue()`, matching the correct pattern already used in `grpc/service.go`.
- Introduce one shared, tested `parseK8sQuantityToBytes(string): number | null` utility on the frontend; migrate all ~10 existing hand-rolled parsers (`useClusterUtilization.ts`, `ClusterCapacity.tsx`, `Nodes.tsx`, `NodeDetail.tsx`, `PodDetail.tsx`, `MetricsDashboard.tsx`, `lib/utils.ts`) onto it. Add explicit finite/bounds validation immediately before any percentage or format call, rendering "unknown" instead of a silently-wrong number.
- Add the regression tests locking in COUNTS-2 and METRICS-3's already-correct behavior.

**Gate:** Demonstrate pod count stability under a simulated relist/missed-delete sequence (should stay equal to actual store count); demonstrate all memory-displaying widgets agree on one cluster's total given the same input; demonstrate a malformed/Ki-or-Gi-suffixed quantity renders "unknown," never a NaN-derived or 1024x-wrong number.

---

## PHASE 4 — Startup & Fleet ✅ COMPLETE (2026-10-02)

**Status: DONE.** See `docs/PRODUCTION-HARDENING-EXECUTION.md` ("Phase 4 — Startup & Fleet: Execution Record") for full evidence. Summary: STARTUP-1 completed — `LoadClustersFromRepo`'s connection-testing now runs in the background, decoupled from the HTTP listener bind (measured: listener ready in ~75ms regardless of persisted-cluster count/availability, vs. 18s+ for a realistic 100-cluster/15-hanging mix under Phase 2's fix alone). FLEET-1's UNVERIFIED caveat resolved — `GetClusterSummary` now bounded by `client.WithTimeout()`. **New finding discovered and documented (not fixed, correctly out of this phase's scope):** the Fleet Dashboard frontend never adopted the backend's `/fleet/overview` aggregate endpoint — it does its own N+1 client-side fan-out, which is the most likely real explanation for the Fleet UX complaint. Recommended as a follow-up.

**Goal:** Application becomes usable without waiting for unavailable/slow clusters; Fleet communicates per-cluster state clearly under partial failure.

**Findings addressed:** STARTUP-1 (completion/verification), FLEET-1 (lock in + resolve the one UNVERIFIED caveat)

**Scope:**
- Verify the Phase 2 startup-parallelization fix against the Phase 0 baseline measurement; confirm the app reaches a usable state in roughly constant time regardless of cluster count/availability.
- Resolve FLEET-1's UNVERIFIED caveat: confirm whether a registered-but-newly-slow cluster has an effective per-call deadline in `GetClusterSummary`'s K8s calls, and add one explicitly if the shared client timeout config can't be relied on to always be set.
- Establish real startup-time performance budgets from Phase 0 baseline data (not guessed numbers).

**Gate:** Measured startup-to-usable time meets the budget established in Phase 0, regardless of the number or availability of configured clusters, confirmed on a kubeconfig with several deliberately-unreachable contexts.

---

## PHASE 5 — Topology ✅ COMPLETE (2026-10-02)

**Status: DONE.** See `docs/PRODUCTION-HARDENING-EXECUTION.md` ("Phase 5 — Topology: Execution Record") for full evidence. Summary: TOPOLOGY-1 fixed — the namespace selector now genuinely scopes the backend request for the common single-namespace case (verified: cache is correctly isolated by cluster+namespace, so this is contamination-safe). TOPOLOGY-3 fixed via the roadmap's own comment-correction fallback (porting V2's pod-aggregation into V1 would require a V1 schema change, judged disproportionate to this phase). Graph correctness, relationship resolution, duplication, and large-cluster performance (Steps 4-7, 10-11) were explicitly left UNVERIFIED-this-phase — unmodified code, outside both findings' scope.

**Goal:** Topology either returns valid, partial, empty, or explicit timeout/error results — never infinite loading.

**Findings addressed:** TOPOLOGY-1, TOPOLOGY-3, (TOPOLOGY-2's timeout-mismatch half already fixed in Phase 1)

**Scope:**
- Thread the selected namespace from `useTopologyData` through `useClusterTopology` to `getTopology`'s backend call, so server-side discovery is actually scoped (closes the gap where "Empty set = All Namespaces = 735 resources = system freeze" fires on every load regardless of UI selection).
- Resolve the V1/V2 topology divergence: either port V2's pod-aggregation into V1 (the page actually in use), or migrate the Topology page onto V2 — and correct the misleading comment either way.
- Re-verify TOPOLOGY-2 is fully closed once namespace scoping reduces how often large builds are attempted.

**Gate:** Opening the Topology view for a single namespace on a large cluster completes in a time proportional to that namespace's resource count, not the whole cluster's; a deliberately oversized namespace/cluster produces an explicit partial-result or timeout state, never an un-resolving spinner.

---

## PHASE 6 — Blast Radius ✅ COMPLETE (2026-10-02)

**Status: DONE.** See `docs/PRODUCTION-HARDENING-EXECUTION.md` ("Phase 6 — Blast Radius: Execution Record") for full evidence. Summary: BLASTRADIUS-1 fixed via partial-sourcing — `GetResourceTopology` now reuses the running `ClusterGraphEngine`'s informer cache for the 16/34 resource types it tracks (zero live API calls for those), falling back to a live fetch only for the ~18 types the engine doesn't cover or when no engine is running yet; full type coverage preserved in both cases (a literal full-replacement fix was rejected — the engine's informer set doesn't cover RBAC/Nodes/Namespaces/storage/events, so fully sourcing from it would have silently dropped those types, violating this phase's own "never silently incomplete" goal). BLASTRADIUS-2 fixed — all 34 resource types (not just Pods/Events) now paginate to completion via a shared generic helper, with a 25,000-item safety cap per type that reuses the existing `FailedResources` signal if hit. BLASTRADIUS-3 fixed — `getOrStartGraphEngine`'s global mutex no longer spans client resolution/engine start; replaced with a per-clusterID `singleflight.Group` so one cluster's cold start cannot block another's lookups. All three verified via revert-and-reconfirm (6 new tests, each shown to fail against the pre-fix code).

**Goal:** Blast Radius returns a useful result or an explicit failure/partial result — never infinite loading, never silently incomplete.

**Findings addressed:** BLASTRADIUS-1, BLASTRADIUS-2, BLASTRADIUS-3

**Scope:**
- Source `GetResourceTopology`'s resource bundle from the already-running `ClusterGraphEngine`/informer lister caches instead of re-fetching ~28 resource types live from the K8s API on every cache miss.
- Extract the pods/events pagination-loop pattern into a shared paginated collector and apply it to all 28 resource types in `collector_k8s.go`, or explicitly surface a truncation warning when a list is capped.
- Shard or release-before-call the global mutex in `getOrStartGraphEngine` (`blast_radius.go:134-175`) so one cluster's cold-start client resolution cannot block another cluster's blast-radius/graph-status requests.

**Gate:** Opening Blast Radius for a resource on a large cluster completes without redundant live API fan-out; a cluster with >500 of any resource type still produces a complete (or explicitly-flagged-incomplete) blast radius graph; two different clusters' blast-radius requests do not block each other.

---

## PHASE 7 — Frontend UX Reliability ✅ COMPLETE (2026-10-02)

**Status: DONE.** See `docs/PRODUCTION-HARDENING-EXECUTION.md` ("Phase 7 — Frontend UX Reliability: Execution Record") for full evidence. Summary: UX-2 fixed — Fleet and the Header kubeconfig dropdown no longer default a missing/unrecognized cluster status to "healthy" (previously a crash risk: `FleetDashboard`'s `statusConfig` had no `unknown` key, so a cluster with unresolvable health would throw on render); Fleet's and the cluster picker's staleness indicators now use a consistent, always-visible amber treatment instead of Fleet having none and the picker being hover-only. UX-3 investigated and found ALREADY FIXED at the primary touchpoints (Dashboard's `ClusterCapacity` widget and `MetricsDashboard`'s empty state already distinguish "unknown" from "0" correctly) — not modified. UX-1 partially fixed: one confirmed, representative instance (`ClusterHealthWidget` showing a fabricated "At Risk, 0%" verdict during loading/error) fixed with full test coverage; the broader 154-page audit was judged disproportionate to this phase and left explicitly UNVERIFIED, not silently assumed complete.

**Goal:** Standardize loading / empty / error / timeout / partial / stale states so users can understand failures without developer tools.

**Scope (to be refined after Phases 1-6 land, since several of those phases will already produce new error/timeout states that need a consistent presentation layer):**
- Audit `AsyncSection`/`PageSkeleton`/`DetailPageSkeleton` usage across pages for consistent handling of the new timeout/abort errors introduced in Phase 1.
- Ensure the staleness indicators added in Phase 2 (HEALTH-2) have a consistent visual treatment wherever cluster health is shown (Fleet, cluster picker, sidebar).
- Ensure the "unknown" metric state introduced in Phase 3 has a consistent visual treatment distinct from "0."

**Gate:** A sampled set of critical pages (Dashboard, Fleet, Topology, Blast Radius, resource list/detail) each visibly and distinctly communicate loading / empty / error / timeout / partial / stale without requiring the browser devtools to diagnose.

---

## PHASE 8 — Observability ✅ COMPLETE (2026-10-02)

**Status: DONE.** See `docs/PRODUCTION-HARDENING-EXECUTION.md` ("Phase 8 — Observability: Execution Record") for full evidence. Summary: OBS-1 fixed — every `context.DeadlineExceeded` site in `internal/api/rest` (9 of 10 found; the 10th, `kcli.go`, was already fully instrumented) now logs a structured warning (request_id/cluster_id/operation/path) and increments a new `kubilitics_request_timeouts_total{operation,cluster_id}` counter via a shared `respondTimeout` choke point, with zero change to response behavior. OBS-2 fixed — `Client.updateHealth` (the single choke point behind `TestConnection`/`GetClusterInfo`/`ListResources`/`GetResource`/`DeleteResource`/`PatchResource`, the same calls that maintain HEALTH-2's `LastCheckedAt` freshness field) now feeds `kubilitics_cluster_health_check_duration_seconds{cluster_id}` and `..._failures_total{cluster_id}`. OBS-3 investigated and found ALREADY FIXED — `request_id`/`cluster_id` were already on every request's structured log line globally (not just the four named hot paths), via pre-existing `middleware.RequestID`/`middleware.StructuredLog` wired at the router level — not modified.

**Goal:** Major production failures are diagnosable.

**Scope (to be scoped in detail once Phases 1-6 reveal which signals were actually missing during remediation):**
- Add structured logging/metrics around the timeout boundaries introduced in Phase 1 (did a request time out, where, how often).
- Add a per-cluster health-check latency/failure counter fed by the Phase 2 freshness fields.
- Add request IDs and cluster IDs to backend log lines on the hot paths touched by this audit (overview, topology, blast-radius, cluster lifecycle).

**Gate:** Given a synthetic failure in any of Phases 1-6's scope, an operator can identify the failing component and cluster from logs/metrics alone, without reading source code.

---

## PHASE 9 — Regression / Failure Testing ✅ COMPLETE (2026-10-02)

**Status: DONE.** See `docs/PRODUCTION-HARDENING-EXECUTION.md` ("Phase 9 — Regression / Failure Testing: Execution Record") for full evidence. Summary: audited all 19 individual items in the audit's "Test Coverage Gaps" section against the actual test suites Phases 1-8 produced — 17 were already closed (each phase's own regression-test requirement had substantially overlapped this list already), confirmed by reading the actual tests, not assumed. Of the 2 genuine gaps found: METRICS-2's scientific-notation sub-gap fixed (2 new tests on the canonical parser, confirmed correct-but-previously-unverified behavior); CONTAM-1's "no test exercises SetupInformerHandlers" gap closed with a characterization test that proves the dormant contamination behavior is now machine-checked (CONTAM-1 itself remains deliberately unfixed/dormant, unchanged from Phase 2). One remaining gap (TOPOLOGY-3, P2) documented as out of the Gate's P0/P1 requirement, not silently dropped.

**Goal:** Automate the test coverage gaps identified in the audit.

**Scope:** Implement the specific missing tests enumerated in the audit's "Test Coverage Gaps" section — one test (or test suite) per finding, covering: healthy clusters, unreachable clusters, slow clusters, auth/RBAC failures, metrics unavailable, API timeout, cluster removal/re-add, restart, large clusters, large topology/resource counts, `DeletedFinalStateUnknown` delete events, and namespace-scoped topology requests.

**Gate:** Every P0/P1 finding in the audit has at least one automated regression test that would fail on the original code and pass after the corresponding phase's fix.

---

## PHASE 10 — Release Gate

**Goal:** Objective, evidence-based release criteria.

**Criteria:**
- No P0/P1 finding from `docs/PRODUCTION-RELIABILITY-AUDIT.md` remains open, OR is explicitly re-classified UNVERIFIED with documented reasoning and user sign-off.
- All Phase 9 regression tests pass.
- Phase 0/4 performance budgets pass on a kubeconfig with multiple deliberately-unreachable contexts.
- No known data-correctness issue (COUNTS-1, METRICS-1/2 class) remains unresolved.
- No infinite-loading path remains in: Dashboard load, resource list/detail load, Topology load, Blast Radius load, cluster add/remove.

---

## PHASE 11 — WebSocket Isolation & Release Unblock

**Status: COMPLETE (fixed 2026-10-05, commit `0a7761a0` "fix(reliability): bound blocking I/O, fix cross-cluster WebSocket isolation"; independently re-verified 2026-10-13).** Authorized 2026-10-02 as a roadmap amendment, after Phase 10's Release Gate ran and was explicitly BLOCKED on Criterion 1 ("no open P0/P1 finding, or UNVERIFIED with documented reasoning and user sign-off"). See `docs/PRODUCTION-HARDENING-EXECUTION.md` ("Phase 10 — Release Gate: Execution Record") for the original blocking evidence. The fix landed 3 days after this phase was authorized but this document was never updated to reflect it until the 2026-10-13 re-verification — a reminder that a phase isn't actually closed until its status line is.

**2026-10-13 re-verification evidence:** all 8 of this phase's numbered acceptance tests below exist and pass in `internal/api/websocket/contam1_isolation_test.go` (7 tests) plus `informer_broadcast_test.go` (end-to-end proof through the still-dormant publisher path). `go test -race ./internal/api/websocket/...` — exit 0, zero race warnings. Full backend suite (`go build`, `go vet`, `go test ./...`) — 54 packages, zero failures. Dormancy confirmed preserved: `cmd/server/main.go` still constructs the WebSocket handler with `informerMgr: nil` and never calls `SetupInformerHandlers()`/`SetClusterID()` — no production activation occurred.

**Finding:** CONTAM-1 — WebSocket resource/topology broadcast pipeline is dead code and lacks real cluster scoping.

**Historical context (preserved, not rewritten):** CONTAM-1 was originally assigned to **Phase 2 — Cluster Lifecycle**, which explicitly deferred it rather than fixing it: *"CONTAM-1 explicitly deferred (dead/inert code, resuming it is a larger unit of work than this phase's scope, documented rather than left ambiguous)"* (Phase 2's summary, above). Phase 9 later added a characterization test (`TestSetupInformerHandlers_BroadcastsWithEmptyClusterID`) that machine-checks the gap's existence without fixing it. Phase 10's Release Gate then ran Criterion 1 against the full P0/P1 inventory, found CONTAM-1 confirmed-open, and — per explicit user instruction — blocked rather than signing off on shipping with it open. No phase between Phase 2 and Phase 10 was ever assigned to fix it; Phase 11 is the first phase with that mandate.

**Severity:** P1 while dormant (current state — the pipeline is not wired into any active production route). Would become a critical cross-cluster isolation issue — one tenant's cluster A events reaching a subscriber scoped to cluster B — if the pipeline is ever activated without first closing this gap.

**Objective:** Resolve CONTAM-1 safely and provide machine-checkable evidence that cluster isolation is preserved across the entire WebSocket broadcast pipeline, sufficient to unblock Phase 10's Criterion 1.

**Scope — may modify ONLY the code and tests required to resolve CONTAM-1:**
- Trace the complete WebSocket lifecycle end to end: connection → authentication/identity (if applicable) → cluster selection → subscription registration → informer/event source → broadcast → Hub filtering → client delivery.
- Identify the authoritative source of cluster identity for each broadcast event.
- Ensure every broadcast event (not just ones originating from `SetupInformerHandlers`) carries the correct, real cluster identity through to `Hub.BroadcastResourceEvent`.
- Ensure `Hub`'s filtering logic uses that identity correctly for every delivery path (no bypass when identity happens to be present vs. absent).
- Ensure a missing/empty cluster identity **fails closed** (does not broadcast across cluster boundaries) rather than failing open (today's behavior — see Phase 9's characterization test).
- Preserve the pipeline's current, intentional dormant/inert production state — **do not activate it** merely to demonstrate the fix; prove correctness via direct, deterministic tests against the Hub/subscription/broadcast code paths instead.
- Reuse the existing Hub/subscription architecture (`Client.clustersSubs`, `Client.AcceptsCluster`, `Hub.broadcast`/`BroadcastResourceEvent`) — no new WebSocket architecture, no authentication/authorization redesign.
- Do not fix unrelated topology, Fleet, UX, metrics, or observability issues, and do not revisit TOPOLOGY-3 or any other finding not named here.

**Security/isolation acceptance — all of the following require deterministic, automated, `go test -race`-clean tests:**
1. Same-cluster delivery: a cluster A event reaches a cluster A subscriber.
2. Cross-cluster isolation: a cluster A event does NOT reach a cluster B subscriber.
3. Reverse isolation: a cluster B event does NOT reach a cluster A subscriber.
4. Multiple subscribers: all cluster A subscribers correctly receive cluster A events.
5. Missing cluster identity: an event with empty/missing cluster identity must not cross cluster boundaries (fail closed, not fail open).
6. Concurrent clusters: events from A and B occurring concurrently remain correctly isolated.
7. Subscribe/unsubscribe lifecycle: an unsubscribed/removed client receives no subsequent events.
8. Race safety: the affected packages pass `go test -race` with no new races introduced.

**Regression requirements:**
- At least one regression test must demonstrate the pre-fix vulnerable behavior (or otherwise structurally prove the pre-fix defect) before the fix is applied.
- After the fix: the 8 isolation tests above pass, `go test -race` is clean for the affected backend packages, the full backend suite passes, and the full Phase 1-10 regression suite remains green (per this engagement's standing regression-protection discipline).
- Cross-cluster isolation must never be claimed without an explicit, passing test proving it.

**Release-gate dependency:** Phase 10's Criterion 1 may only be considered satisfied once: CONTAM-1 is fixed, all 8 isolation tests pass, race testing passes, and the Phase 1-10 regression suite remains green. Compiling cleanly is not sufficient evidence on its own.

**Gate:** Every one of the 8 security/isolation test cases above exists, passes, and is `-race`-clean; the full backend and frontend regression suites from Phases 1-10 remain green; Phase 10's Criterion 1 is re-run and passes without requiring a sign-off exception.

---

## Execution Discipline

Per the governing brief: **execute one phase at a time.** For each phase:
1. Explain exact scope (the bullets above are a starting point, not a substitute for re-confirming scope against the then-current code).
2. Implement the smallest safe change that satisfies the phase's findings.
3. Add the regression test(s) for that phase before/alongside the fix.
4. Run targeted tests, then integration/E2E tests where applicable.
5. Measure before/after against the Phase 0 baseline where a performance claim is being made.
6. Review logs for regressions.
7. Update `docs/PRODUCTION-RELIABILITY-AUDIT.md` (mark findings fixed, update confidence) and this roadmap (mark phase complete) with evidence.
8. Report evidence to the user and stop — do not silently proceed to the next phase.

**Current status: awaiting review and approval of the audit and this roadmap. No implementation has begun.**
