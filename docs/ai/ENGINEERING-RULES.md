# Engineering Rules (Kubilitics)

Permanent operating rules for AI-assisted engineering sessions on this
repository. Load this always; it's small by design.

## 1. Anti-loop

Never repeat a command without a new reason to expect a different result.
Never rebuild/retest without a changed input. If a command exceeds its
expected duration, stop and inspect — don't start a second copy, don't
assume it's hung, don't do unrelated work while waiting without checking.

## 2. Token economy

Before exploring: check this file, `docs/ai/`, `CLAUDE.md`, `git status`,
`git log`. Prefer targeted search over recursive reads. Don't reread
unchanged files. When you learn something non-obvious, write it to
`docs/ai/KNOWN-ISSUES.md` or `docs/ai/DECISIONS.md` so the next session
doesn't rediscover it.

## 3. Evidence labeling

Every claim gets one of: **VERIFIED** (directly tested) / **CODE-PROVEN**
(inspected but not run) / **UNVERIFIED** (couldn't test) / **PRE-EXISTING**
/ **DEFERRED** / **OUT OF SCOPE**. Never say "fixed," "released,"
"published," or "stable" without the evidence to back it.

Never upgrade LAB VERIFIED → REAL APP VERIFIED → REAL ENVIRONMENT
VERIFIED without actually doing the stronger check.

## 4. Recoverability

Before meaningful work: `git status --short`, `git branch --show-current`,
`git log -5 --oneline`. Never `git reset --hard` / `git clean -fd` /
`git checkout .` / `git restore .` without explicit authorization. Never
touch unrelated uncommitted work to simplify your own task.

## 5. Commit shape

Small, logical commits (implementation / tests / docs), not giant
campaigns. See `docs/ai/DECISIONS.md` for why the Oct 2026 consolidation
had to retroactively group ~135 files — don't repeat that pattern; commit
as you go.

## 6. Release discipline

See `docs/ai/RELEASE-PROCESS.md`. Patch/minor sequential versions only
(v1.2.2 → v1.2.3 → v1.2.4 …). Never invent a version jump without
explicit authorization. A documentation/investigation milestone is not a
release. Never create a tag, push a tag, or publish without being asked.

## 7. New P0/P1 found mid-task

Stop. Report: finding, evidence, severity, root cause (if known), proposed
fix, release impact. Do not silently expand scope or silently fix and
move on.

## 8. Stop conditions for investigations

Stop when: root cause confirmed, evidence contradicts the hypothesis,
reproduction fails repeatedly, the environment gets contaminated, a
command hangs, scope expands beyond the original question, or an
unrelated issue surfaces (log it, don't chase it mid-task).

## 9. Flow

`DISCOVER → PLAN → IMPLEMENT → TEST → COMMIT → ARTIFACT → REAL-APP TEST →
RELEASE → OBSERVE → NEXT RELEASE`. Not an endless
`DISCOVER → INVESTIGATE → INVESTIGATE → MORE DOCS → NEVER SHIP` spiral.
