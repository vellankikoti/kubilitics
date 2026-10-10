# Stabilization Plan (Kubilitics)

Generated from a critical-review pass across security, desktop/Tauri
stability, test coverage, and scale/perf readiness (Oct 2026), on top of
everything already fixed this session (informer-cache correctness,
namespace-filter UI standardization across 29 pages, panic-recovery
coverage, CI gate fixes). Each review was a fresh, skeptical, read-only
pass — not a rehash of prior "confirmed clean" claims. Findings below are
verified by reading the actual code path, not inferred from docs/comments.

Organized into phases by urgency. **Phase 0 items are live, exploitable
issues in shipped code — they warrant attention independent of how the
rest of this plan gets sequenced.**

---

## Phase 0 — Security: fix before anything else ships

### 0.1 Secret redaction bypass via case-sensitivity mismatch (P0)
**`internal/pkg/redact/redact.go:26-29`** — `IsSecretKind()` is an exact-string
switch (`"Secret"`, `"secret"`, `"Secrets"`, `"secrets"`), but the actual
data-resolution paths are case-*insensitive*:
- `internal/k8s/informer.go:444` lowercases before cache lookup.
- `internal/k8s/discovery.go:260-264` uses `strings.EqualFold`.

**Exploit**: `GET /api/v1/clusters/{id}/resources/SECRETS` (or any case
variant outside the exact-match list, e.g. `SeCrEts`) passes validation,
hits the cache via lowercase lookup, returns real `Secret` objects —
but `IsSecretKind("SECRETS")` is `false`, so `redact.SecretData()` never
runs. **Full decoded Secret data returned to any role**, including
Viewer. Same bug independently reachable via `GetResource` and
`PatchResource`.

**Fix**: make `IsSecretKind` case-insensitive (lowercase-compare or
`strings.EqualFold`), not add more literals.

### 0.2 Events Intelligence + OTel trace APIs have zero RBAC (P0)
**`internal/events/api.go:92-128`** and **`internal/otel/api.go:29-47`**
register ~20 routes via plain `router.HandleFunc` — no `wrapWithRBAC`, no
role check, no per-cluster permission check anywhere in either file
(confirmed via grep, zero hits). Registered unconditionally in
`cmd/server/main.go` (lines ~933-934, ~988), on a **different code path**
than `handler.go`'s `SetupRoutes` (the function previously audited and
confirmed fully wrapped) — which is exactly why this was missed before.

**Exploit**: any authenticated user, any role, can hit
`GET /clusters/{otherClusterId}/events-intelligence/query`, `/incidents`,
`/insights/active`, `/logs/search`, `/state/at` (time-travel), traces
endpoints — **for a cluster they have no permission on**. Two mutating
routes (`POST /insights/{id}/dismiss`, `POST /events-intelligence/analyze`)
have no check at all. Cross-tenant data exfiltration.

**Fix**: wrap every route in both files with the same
`RequireRole`/per-cluster-permission pattern used in `handler.go`.

### 0.3 Container file browser is under-gated relative to its power (P1, bundle with 0.1/0.2)
**`internal/api/rest/handler.go:689-690`** — file-browser `ls`/`download`
require only `RoleViewer`, while exec/shell require `RoleOperator`. The
file browser can `cat` any in-container path, including
`/var/run/secrets/kubernetes.io/serviceaccount/token` or any
Secret-backed mounted volume — a Viewer can read a live SA token or
Secret content this way, bypassing the exact redaction the typed Secret
API applies. Raise to `RoleOperator`, matching exec's tier.

### 0.4 Scanner API: RBAC bypassed when claims is nil (P2, fix alongside above)
**`internal/api/rest/scanner_handler.go:60-66`** — `if claims != nil &&
claims.Role != "admin" ... { deny }` skips the check entirely when
`claims` is `nil` (auth-disabled/optional mode), allowing unauthenticated
`StartScan`. Also: host-path blocklist doesn't cover macOS `/Users/*`.
Fix: deny-by-default when `claims == nil`; expand/replace the blocklist
with an allowlist.

### 0.5 Dead code with no RBAC — delete or fix before it's wired in
**`internal/api/rest/schedule_handler.go`**'s `ScheduleHandler` (CRUD for
report schedules, zero RBAC checks) is never registered — not live, not
exploitable today. But it's a loaded gun: if someone wires it into
`main.go` later without noticing, that's an instant unauthenticated
mutating endpoint. Delete it, or fix its RBAC now while it's cheap.

---

## Phase 1 — Desktop/Tauri stability (P0s, affects every long-running session)

### 1.1 `restart_count` never resets — auto-recovery permanently dies after 3 lifetime crashes
**`kubilitics-desktop/src-tauri/src/sidecar.rs:59, 503-507`** — the
health-monitor's restart counter is cumulative for the entire app
session, never reset on a successful restart (manual or automatic).
After 3 backend crashes *ever* in one session (hours/days), auto-recovery
permanently disables itself — even if each crash was individually
recovered from. Fix: reset to 0 in `restart()` and on every successful
health-monitor-driven restart.

### 1.2 Main "Reconnect" banner doesn't actually restart a dead process
**`kubilitics-frontend/src/components/layout/BackendStatusBanner.tsx:51-57`**
— "Reconnect" only calls `resetBackendCircuit()` + retries the HTTP
fetch. If the backend process is genuinely dead (not just slow), every
click fails forever. The only real fix — `invoke('restart_sidecar')` —
exists only on the Settings page (`Settings.tsx:200-214`), which a user
hitting a dead backend has no reason to find. Combined with 1.1: after 3
lifetime crashes, a user with no Settings-page knowledge is stuck until
they quit and relaunch the whole app (full restart does work — it's just
non-obvious).

**Fix**: wire the main banner's "Reconnect" to try `restart_sidecar` when
plain retry fails N times, and emit a `backend-status: dead` event from
Rust when `MAX_RESTART_ATTEMPTS` is exhausted so the frontend can show
"auto-recovery exhausted, click to restart" instead of a generic
unreachable message.

### 1.3 No health-monitor loop for the AI/brain sidecar (P1)
Only `BackendManager` has a health-monitor loop; `BrainManager` has none.
If `kubilitics-ai-server` crashes post-start, nothing detects or restarts
it — chat stays broken until a manual restart or full app relaunch. Lower
severity (rest of the app stays usable) but an asymmetry worth closing —
mirror `BackendManager`'s loop.

**Confirmed solid, no action needed**: port-conflict fallback, updater
signature verification, sidecars built fresh from source on every release
(no stale-binary risk), OS-correct path handling via the `dirs` crate,
encrypted local secret store (deliberate choice over OS keychain),
unsigned-local vs. signed-CI separation.

---

## Phase 2 — Test coverage for the highest-leverage, highest-risk gaps

**Headline finding: zero automated test anywhere executes a destructive
action to completion.** E2E specs explicitly skip delete/restart
("avoid side effects"); the one test that opens a delete dialog clicks
Cancel, never Confirm. 153 of 156 frontend pages have no test file at all.

Priority order (highest leverage first — shared code or security surface,
not just raw file count):

1. **`internal/api/rest/resources.go`** — generic `DeleteResource`,
   `PatchResource`, `ApplyManifest` (the shared mutation path for every
   resource kind in the app). Zero test file. A GVK-parsing regression
   here breaks delete for one kind silently, with no signal.
2. **`internal/api/rest/auth.go`** (2119 lines) — `Login`, MFA check, rate
   limiter, session/device logging. Zero test file. This is the entire
   login/security surface.
3. **One real e2e test that completes a destructive action** against a
   disposable test resource — breaking the project-wide "never confirm
   destructive actions in tests" pattern. Even one is a meaningful signal
   floor-raise.
4. **Shared hooks with wide fan-out, zero tests**: `useTableFiltersAndSort`
   + `useColumnVisibility` (53 of ~156 pages each), `useNamespaceFilter`
   (28 pages, and brand new as of this session's work — exactly the
   "ships silently" scenario this gap describes).
5. **`Pods.tsx`/`Deployments.tsx`** `handleDelete`/`handleBulkDelete`/scale
   dialog component tests — the two highest-traffic pages with the
   highest-stakes actions (deleting/scaling live workloads).
6. Backend coverage is currently 25.5% (`internal/api/rest`) and 32.3%
   (`internal/service`) — both below a reasonable bar for
   mutation-handling packages. Notable zero-coverage files beyond the
   above: `exec.go`, `shell.go`/`shell_stream.go`, `portforward.go`,
   `oidc.go`/`saml.go`, `node_operations.go` (cordon/drain — a bad drain
   can cause an outage), `cronjobs.go`/`jobs.go` (trigger/retry).

---

## Phase 3 — Scale/perf verification (mostly "measure," not "fix")

The namespace-filter UI standardization itself did **not** introduce a
new perf cliff — client-side re-pagination already caps DOM rows
regardless of filtered-set size (verified: this is safer than it looked
at first glance, not a new gap).

**What's actually unverified**: the pre-existing `useK8sResourceList(kind,
undefined, { limit: 5000 })` fetch-all-then-filter-client-side pattern,
used by ~26 of the just-migrated list pages, has only ever been
benchmarked for Pods (up to ~11K pods in the lab). Deployments/Services/
ConfigMaps/Secrets at the documented 15,247-pod/11,102-workload scale
cluster were never measured — conservative extrapolation suggests
multi-second loads plus a background refetch every 5 minutes
(`refetchInterval` default) per open tab, but this is **estimated, not
measured**.

1. **DONE — measured, concern ruled out.** Benchmarked the backend list
   endpoint (`GET /clusters/{id}/resources/{kind}?limit=5000`, exactly
   what `useK8sResourceList` calls) against a real kind cluster seeded to
   the documented scale (15,256 pods, 5,002 deployments, 3,002 services,
   2,043 configmaps, 1,103 secrets, 35 namespaces — kind-nightshift-dev,
   rebuilt fresh for this test since its prior state had unrelated
   cert-rotation corruption). Three repeated requests per kind, all
   well under the extrapolated "multi-second" concern:
   - Pods: ~365ms · Deployments: ~372ms · Services: ~80ms ·
     ConfigMaps: ~29ms · Secrets: ~18ms
   The extrapolation was wrong — no server-side pagination migration is
   needed for these kinds at this scale. (Seeding used synthetic
   Pod/Deployment objects with replicas=0 and fake-but-real Node objects
   so nothing actually scheduled or ran real containers — see git history
   on this branch for the throwaway seeding tool, not kept in the repo.)
2. **DONE.** Debounced the `singleSelectedNamespace`-keyed side queries
   (Deployments' `eventsForScaleCount`, StatefulSets' PVC scoping) via
   `useDebouncedValue` — they no longer fire one request per namespace
   touched during rapid checkbox-clicking.
3. **P2, watch not fix**: EndpointSlice count scales with Service × pod
   fan-out and has no cache eviction/cap — worth a line item to check
   informer memory footprint at scale, not a confirmed problem.
4. **Confirmed non-issues**: `ResizableTable` has no virtualization, but
   all 26 migrated pages slice to `pageSize` (default 10) after
   filtering, so this doesn't matter in practice. `NamespaceFilter`'s own
   search/grouping is cheap even at 100+ namespaces (untested at that
   count, but the ops involved are trivially fast).

---

## Phase 4 — Already-tracked, deferred backend capability work

From `docs/ai/KNOWN-ISSUES.md` (unchanged by this review, included here
for one-stop phase planning):

- Informer-cache coverage: `APIService` (needs `k8s.io/kube-aggregator`,
  new dependency), `VerticalPodAutoscaler` (needs VPA's own generated
  client, new dependency), `ResourceSlice`/`DeviceClass` (DRA API
  version-skew risk — needs explicit version negotiation before it's
  safe to add as a typed informer).

---

## Suggested sequencing

Given the user will decide actual phase-by-phase execution, but as a
starting recommendation:

1. **Now**: Phase 0 (security) — these are live exploits, not
   hypothetical. 0.1 and 0.2 are small, surgical fixes (one function,
   two route-registration files) with low risk of collateral breakage.
2. **Next**: Phase 1 (desktop stability) — also small, surgical
   (reset a counter, wire one button, add one more health-monitor loop).
3. **Then**: Phase 2 items 1-3 (the highest-leverage test coverage) —
   larger effort, but closes the single biggest "ships silently" risk
   class across the whole app.
4. **Parallel/opportunistic**: Phase 3's benchmark (cheap to run, informs
   whether more pages need Pods-style server pagination) and the
   debounce fix (trivial).
5. **Scheduled separately**: Phase 4 (each item is its own dedicated task
   per the project's existing versioning policy — not patch-sized work).
