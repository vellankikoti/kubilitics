# Release Process (Kubilitics)

## Current state (keep this line updated)

```
CURRENT PUBLISHED RELEASE: v1.2.3 (2026-10-06, origin + upstream)
NEXT RELEASE: v1.2.4 (patch) — do not jump to v1.3.0 without an explicit,
  genuinely-new-capability reason approved by the project owner.
```

## Versioning policy

- **PATCH** (v1.2.x) for bug fixes, reliability hardening, CI fixes —
  the default for almost everything.
- **MINOR** (v1.x.0) only for a real new capability, not for "a campaign
  phase finished." Example candidate already scoped but NOT started:
  informer-cache expansion (12 missing resource kinds) — this is a
  genuine capability jump, correctly deferred to a future v1.3.0.
- Never invent a version number. Always sequential from the last
  **published** release (check `gh release list`, not local tags — a
  local tag can exist without a real release, see the v1.2.1 incident
  below).

## Known incident: v1.2.1 collision (2026-10-05)

A real, published-then-deleted v1.2.1 release existed in CI history
(`gh run list --workflow=release.yml`) but had no corresponding GitHub
Release or tag anymore. Always check `gh run list --workflow=release.yml`
before reusing a version number — `gh release list` alone can miss a
deleted release's history.

## The actual flow (proven 2x: v1.2.2, v1.2.3)

1. Consolidate completed, tested work onto a branch (e.g. `feat/stability`).
2. `./scripts/bump-version.sh X.Y.Z` — bumps all 6 version files in lockstep.
3. `./scripts/pre-release-check.sh X.Y.Z` — build/vet/test gate. Run fresh,
   never trust a prior run.
4. Commit the version bump, push the branch, open a PR to `main`.
5. **User reviews and merges the PR themselves** (or explicitly authorizes
   an admin merge over a known, documented, non-blocking CI failure).
6. Fast-forward local `main` to the merge commit: `git fetch origin main &&
   git checkout main && git merge --ff-only origin/main`.
7. **Real artifact build + install + test BEFORE tagging** (see
   `docs/ai/TESTING.md` for the exact flow) — write
   `docs/releases/vX.Y.Z-RELEASE-REPORT.md` with real evidence.
8. Create an annotated tag on the merge commit:
   `git tag -a vX.Y.Z -m "..."`. Verify `git rev-parse vX.Y.Z^{commit}`
   equals `git rev-parse HEAD`.
9. **Wait for explicit authorization**, then `git push origin vX.Y.Z` —
   this is what actually triggers `release.yml` (tag-push only; merging to
   `main` does NOT trigger a release build, by design — confirmed by
   reading the workflow triggers directly, not assumed).
10. Monitor `release.yml` (build, sign, notarize all 3 platforms) and
    `charts-publish.yml` (Helm) to completion via `gh run view` /
    `gh run list` — never claim success without checking.
11. Verify `gh release view vX.Y.Z` shows a real, non-draft,
    non-prerelease release with all expected signed artifacts.

## Known CI gotchas (don't rediscover these)

- **`release.yml` triggers on tag push only**, not on push to `main` and
  not on PR merge. This is intentional, not a bug — confirmed by reading
  the workflow's `on:` block directly.
- **`charts-publish.yml` also triggers on tag push.** If the tag already
  existed when a workflow bug gets fixed (e.g. a bad action version pin),
  re-running the original tag-triggered run replays the OLD broken
  workflow file (GitHub snapshots the workflow at trigger time). Use
  `workflow_dispatch -f tag=vX.Y.Z` to re-run against the CURRENT
  workflow file on `main` instead — or, if the gate that's failing checks
  check-runs against the tag's exact commit SHA, you may need to
  force-move the tag to the fixed commit (only ever on `upstream`/forks
  that haven't published yet — never on a repo with an already-published
  release for that tag).
- **macOS notarization failures are usually account-side, not code-side.**
  401 = stale app-specific password (regenerate at appleid.apple.com).
  403 "agreement missing or expired" = Apple Developer Program membership
  or a legal agreement needs re-acceptance by the account holder — not
  fixable by CI config changes.
- **Transient infra failures are real and need a retry, not a code
  investigation.** Example: `Desktop (windows)` failed on v1.2.3's first
  run with `HTTP 500` downloading the WiX toolset from GitHub — unrelated
  to any code change, fixed by `gh run rerun --failed`.
- **GitHub auto-disables scheduled (`cron`) workflows after ~60 days of
  repo inactivity**, and does NOT re-enable them automatically when
  activity resumes. Check `gh api repos/OWNER/REPO/actions/workflows
  --jq '.workflows[] | select(.state=="disabled_inactivity")'`
  periodically — this silently killed the weekly security scan for 2+
  months before it was caught.
- Origin (`vellankikoti/kubilitics`) and upstream (`kubilitics/kubilitics`
  org repo) are separate repos with **separate secrets** — an Apple
  password fix on one does not propagate to the other.

## What a release report must contain

See `docs/releases/v1.2.3-RELEASE-REPORT.md` for the template that's
actually been used twice successfully: Release/Theme, Production Changes
(with VERIFIED/CODE-PROVEN labels), Regression Evidence (real commands,
real output), Artifact (file, size, SHA256), Real Artifact Validation
(table of checks with VERIFIED/UNVERIFIED-TOOLING-GAP), Environment
Safety, Known Issues, Deferred Work, Security Status, Git state, Rollback.
