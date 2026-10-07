# Testing (Kubilitics)

## Commands (run fresh, never trust a prior run)

```bash
# Backend
cd kubilitics-backend
go build ./...                 # expect: ~10-30s clean build, exit 0
go vet ./...                   # expect: ~10-30s, no output = clean
go test ./...                  # expect: 1-3 min, "ok" per package
go test -race ./...            # expect: 2-5 min, 0 data races
golangci-lint run --path-mode=abs --timeout=8m ./...   # expect: "0 issues"

# Frontend
cd kubilitics-frontend
npx tsc --noEmit                # expect: ~30-60s, 0 errors
npx vitest run                  # expect: ~35-50s, see baseline below
npm run build                   # expect: ~1-2min, real dist/ output
npm audit --audit-level=critical  # expect: exit 0 (see known gap below)
```

## Known pre-existing frontend test baseline (as of v1.2.3)

**928/931 passing.** The 3 failures are confirmed pre-existing — verified
directly by running the identical tests against an isolated worktree at
the pre-session baseline commit (`e1acb339`), not assumed:

- `ClusterPickerPage.test.tsx` — 2 failures (label/text-content mismatches
  between test expectations and component copy, not functional bugs)
- `AddClusterDialog.test.tsx` — 1 failure (same class)

**Do not try to silently fix these without being asked** — they're
tracked, not blocking, and "fixing" test expectations to match component
copy (or vice versa) is a real decision, not a trivial patch.

## Known gap: npm audit critical CVEs (dev-only, confirmed non-blocking)

`tinypool`/`vitest` critical CVEs, present via `devDependencies.vitest`
(not `dependencies` — confirmed zero production exposure via dependency-
path analysis: only `src/test-setup.ts` imports `vitest` anywhere in
`src/`). Real fix requires **vitest 2→4 AND vite 5→6+ together** (vitest
4's peer dependency, not optional) — confirmed via a real, reverted
attempt that broke **107 of 108 test suites** at the collection/config
level (not a few failing assertions — systemic). This is a dedicated
migration task, not a quick dependency bump. See
`docs/ai/KNOWN-ISSUES.md`.

**If you're tempted to run `npm audit fix --force`: don't.** It cascaded
to vitest@5 unprompted and made the vulnerability count WORSE (5 critical
instead of 2) in the one real attempt made. Always pin an exact target
version and test deliberately in an isolated worktree, never let npm
auto-resolve a major bump on a repo this size.

## Real artifact validation flow (required before tagging a release)

1. `cd kubilitics-desktop && npm run build` — rebuilds Go sidecars from
   current HEAD + frontend bundle + Rust release build + DMG/app bundle.
   Expect ~5-8 min total (Rust compile is the long pole, ~2-3 min alone).
2. `shasum -a 256` the DMG, record it.
3. Copy the `.app` to an isolated dir (e.g. `/tmp/kubilitics-vX.Y.Z-lab/`)
   — **never overwrite `/Applications/Kubilitics.app`** without explicit
   authorization.
4. Write an isolated kubeconfig pointing at a **dedicated lab/kind
   cluster** — **never point at `nightshift-dev`** (the real cluster) or
   any cluster you don't own for this test.
5. `env HOME=<isolated-home> open <isolated-app-path>`.
6. Hit the backend's REST API directly (`curl 127.0.0.1:<port>/healthz`,
   `/api/v1/clusters`, etc.) to smoke-test — GUI interaction itself is
   **UNVERIFIED — TOOLING GAP**, always say so explicitly, never claim
   GUI validation that didn't happen.
7. Exercise the specific functionality the release actually changed, not
   just a generic smoke test.
8. Kill, confirm clean process exit, relaunch, repeat the critical checks.
9. Before AND after the whole test: confirm the real cluster's pod count
   is unchanged (`kubectl --context <real-context> get pods -A --no-headers
   | wc -l`) and the real `/Applications/Kubilitics.app` PID is unchanged
   throughout.

## Race detection notes

`go test -race` found and the repo fixed a real data race in
`mockClusterRepoForValidation.List()` (test mock returning shared
pointers across concurrent calls — production's real SQLite repo always
deserializes fresh objects, so this was a test-infra bug, not a
production bug). If `-race` ever flags something in production code
(not a mock), treat it as P1 — don't downgrade severity because it's
"only" a race.
