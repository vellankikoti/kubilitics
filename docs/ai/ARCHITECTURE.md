# Architecture (Kubilitics)

High-level map. For Figma/design-token rules see the root `CLAUDE.md`; for
the code-review-graph MCP mandate see `kubilitics-backend/CLAUDE.md`.

## Repo shape

```
kubilitics/
  kubilitics-backend/    Go API server (REST, gRPC, K8s clients, informers)
  kubilitics-frontend/   React + TypeScript (Vite build, consumed by Tauri)
  kubilitics-desktop/    Tauri shell — bundles backend + ai-server + kcli
                         as externalBin sidecars, produces the installable app
  deploy/helm/           Helm chart for in-cluster deployment
  docs/                  Investigation records, release reports, ADRs
  scripts/               bump-version.sh, pre-release-check.sh, release.sh,
                         fetch-backend.sh / fetch-brain.sh / fetch-kcli.sh
```

The AI/brain (`kotg.ai`) and wire-contract (`kotg-schema`) are **separate
repos**, not in this monorepo — integrated via gRPC/HTTP client code in
`kubilitics-backend` (see `internal/ai` / `aiclient`).

## Backend (`kubilitics-backend/`)

- `internal/api/rest/` — HTTP handlers, one file per resource area
  (workloads.go, fleet.go, topology routes in handler.go, etc.). Every
  handler is registered with `h.wrapWithRBAC(handler, auth.RoleX)` —
  **a route registered without this wrapper and without its own inline
  claims check has zero authz enforcement.** (Verified clean as of
  Oct 2026 — all in-handler-check routes were traced and confirmed
  intentional, not gaps — but a *new* unwrapped route is a real risk.)
- `internal/k8s/informer.go` — `InformerManager`: the Lens/Headlamp
  caching model. Live K8s watch → in-memory store → sub-ms reads via
  `ListFromCache`. `resourceKindToStoreKey` is the authoritative list of
  cached kinds (currently 27 — see `docs/ai/KNOWN-ISSUES.md` for the
  coverage gap vs. Headlamp's 46).
- `internal/service/cluster_lifecycle.go` — `ClusterLifecycleManager`:
  lazy activation. A registered cluster's informers don't start until
  first real consumer use; idle clusters release after a 10-min TTL.
  Exactly-one-generation-per-activation guarantee, protected by a
  documented TOCTOU-safe locking pattern (see the file's own comments —
  this was specifically engineered to make the "TTL vs. active-request"
  race impossible, confirmed by a dedicated concurrency audit).
- `internal/graph/engine.go` — `EngineLifecycleManager`: same lazy-
  activation pattern for the Blast Radius `ClusterGraphEngine`.
- `internal/topology/` (v1) and `internal/topology/v2/` — resource
  relationship graph building. v2's `collector_k8s.go` has a
  `paginatedCollect[]` generic used across most resource-kind collectors.
- All 26 backend goroutines (`go func(){}()`) carry `defer recover()` as
  of v1.2.3 — **if you add a new one, it needs this too**, the pattern is
  `defer func() { if r := recover(); r != nil { slog.Default().Error(...)
  } }()` as the *first* defer in the goroutine body.
- DB layer: `internal/repository/` — both SQLite and Postgres backends
  behind the same interface; check both when fixing a repository bug (a
  real bug this session existed identically in both implementations).

## Frontend (`kubilitics-frontend/`)

- React 18 + TypeScript + Vite, React Query for all server state,
  Zustand (27 stores) for client state.
- `src/hooks/useKubernetes.ts` — `useK8sResourceList` /
  `usePaginatedResourceList`: the two most-used data hooks in the app.
  Query keys are cluster-scoped (`clusterId` included), and
  `placeholderData` is scoped to the same cluster via
  `keepPreviousDataSameCluster()` — **don't revert this to plain
  `keepPreviousData`**, that was the root cause of a real cluster-switch
  data-leak bug.
- `src/stores/clusterPresenceStore.ts` — the core cluster list/sidebar
  state, fed by Tauri backend snapshot events. Array fields use
  `Array.isArray()` guards, not `?? []` — a truthy non-array snapshot
  field previously could crash via `.map()`.
- Shared pattern across ~15+ hooks this session needed fixing: a hook
  aggregating multiple `useK8sResourceList`/`useQuery` calls must surface
  `isError` from EVERY underlying query — several pages had correctly-
  written `if (isError) return <ApiError/>` that was permanently dead
  because the hook never provided the signal. **When adding a new
  aggregation hook, always wire isError through.**

## Desktop (`kubilitics-desktop/`)

- Tauri v2. `beforeBuildCommand`/`beforeDevCommand` run
  `scripts/build-sidecars.sh`, which rebuilds `kubilitics-backend`,
  `kotg.ai` (brain), and fetches `kcli` fresh from source on every build
  — never ships stale Go binaries silently.
- Local dev/test builds are **unsigned** — Apple Developer ID + Tauri
  updater private key are CI-only secrets. `release.yml` (triggered by
  tag push only) does the real signed+notarized build.

## State management conventions

- Zustand: `create<State>()(persist((set) => ({...}), {name:
  'storage-key'}))`.
- React Query: `QueryClientProvider` in `App.tsx`; query keys always
  include `clusterId` for cluster-scoped data.

## Path aliases (frontend)

`@/*` → `src/*`, plus `@components/* @features/* @hooks/* @stores/*
@services/* @types/* @utils/* @lib/* @i18n/*`.
