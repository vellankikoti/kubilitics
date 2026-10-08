# Decisions (Kubilitics)

Append-only log of non-obvious engineering/process decisions. Link from
here, don't duplicate the reasoning elsewhere.

## 2026-10-05/06 — Versioning policy reset

**Decision**: real baseline is the last *published* release (v1.2.2 at
the time), next release is v1.2.3 — sequential patch, not a minor/major
jump. Internal campaign phases (R1, R2, R3, "10K campaign", etc.) are
engineering checkpoints, not releases.

**Why**: the repo had accumulated v1.2.1/v1.2.2/v1.2.3/v1.3.0/v1.4.0 as
local-only artifacts from a per-phase versioning strategy that conflated
"an investigation finished" with "ship a release." None of those except
v1.2.0/v1.2.2 were ever genuinely published.

**How to apply**: before creating any new version, check `gh release
list` AND `gh run list --workflow=release.yml` (the latter catches a
real-but-deleted release that the former misses — see the v1.2.1
collision in `docs/ai/RELEASE-PROCESS.md`). Minor version bumps need an
explicit, approved, genuinely-new-capability reason, not a phase
boundary.

## 2026-10-05/06 — PR-based release flow, not direct push

**Decision**: changes land on a branch, a PR is opened, the repo owner
merges it themselves (or explicitly authorizes an admin-merge over a
documented non-blocking CI failure). Tagging/pushing the release tag
still requires separate explicit authorization after the PR merges.

**Why**: keeps a human in the loop at the actual "this goes to real
users" boundary, separate from the "this code is ready" boundary.

## 2026-10-06 — Release validation must include a real artifact test

**Decision**: a release isn't "stable" until the actual packaged
application has been built, installed (side-by-side, isolated), launched,
and smoke-tested against an isolated lab cluster — not just `go test`/
`npm test` passing. GUI-only validation gaps get reported as
`UNVERIFIED — TOOLING GAP`, never silently assumed to work.

**Why**: `go build`/`npm test` passing proves the code compiles and unit
tests pass, not that a real user can install and run the app. See
`docs/ai/TESTING.md` for the exact flow, proven twice (v1.2.2, v1.2.3).

## 2026-10-06 — `npm audit fix --force` is unsafe on this repo, don't use it

**Decision**: never run `npm audit --force` blindly; always pin an exact
target version for a dependency bump and test deliberately.

**Why**: the one real attempt (fixing the vitest/tinypool critical CVEs)
cascaded npm's resolver to vitest@5 unprompted and made the vulnerability
count *worse* (5 critical instead of 2). A careful, pinned attempt at
vitest@4.1.11 + vite@6.4.4 built cleanly but broke 107/108 test suites —
confirming the real fix needs a dedicated migration task, not a
dependency bump. See `docs/ai/KNOWN-ISSUES.md`.

## 2026-10-06 — Informer-cache expansion is a MINOR (v1.3.0) candidate, not a patch

**Decision**: expanding `resourceKindToStoreKey` coverage (27→~39 kinds)
is scoped but deliberately not started inside a patch release.

**Why**: real new backend surface — some kinds need different clientsets
than the existing typed `SharedInformerFactory` pattern, plus new
informer goroutines/memory per cluster need their own testing. Also
genuinely a capability improvement (faster + more reliable reads), which
is what MINOR versions are for per the versioning policy above — not
forced into a patch just because it was discovered during a patch-focused
session.

## 2026-10-06 — Claude Code environment setup (Phase 0/1)

**Decision**: built `docs/ai/` project knowledge layer + 3 Kubilitics-
specific skills (release, debugging, testing) + safety hooks for
destructive commands, rather than installing broad third-party
plugins/MCP servers speculatively.

**Why**: inventory showed 24 pre-installed AWS skills with zero relevance
to this Kubernetes/Go/React/Tauri stack (pure discovery overhead), a
broken GitHub MCP plugin duplicating the already-working `gh` CLI, and
OmniRoute installed but fully unconfigured (no database, no provider
keys, server not running) — confirmed via direct `omniroute status`/
`health` checks, not assumed. Kept the toolchain minimal rather than
installing speculative "might be useful" tooling.
