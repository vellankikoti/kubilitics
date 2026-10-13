# Kubilitics — Known Issues (Reconciled)

**Last updated:** 2026-10-13
**Purpose:** Single source of truth for release readiness. Supersedes
status claims scattered across `ENTERPRISE-*`, `PRODUCTION-RELIABILITY-AUDIT.md`,
and `PRODUCTION-HARDENING-ROADMAP.md` where this document and those disagree —
this document wins on current status; those remain the historical record of
*how* earlier fixes were investigated and verified.

**Status legend** — used consistently below:
- **FIXED (merged)** — fix is on `main`, verified, evidence cited.
- **FIX READY (PR open, not merged)** — fix exists, verified (tests + independent
  review + CI), but has not been integrated. Do not describe as "fixed."
- **DEFERRED (tracked follow-up)** — real, confirmed defect; root cause and
  acceptance criteria defined; intentionally not fixed yet because the fix
  requires a redesign/migration broader than the mandate that found it.
- **BLOCKED** — fix cannot proceed without something outside engineering
  control (a credential, a build environment).
- **UNVERIFIED** — not reproduced/tested either way; explicitly not claimed
  as working or broken.

---

## 1. Release blockers (must resolve before next stable release)

| ID | Item | Status | Why it blocks |
|---|---|---|---|
| REL-1 | Phase 10 Release Gate (pre-existing, from `PRODUCTION-HARDENING-ROADMAP.md`) | **BLOCKED** on Phase 11 | Release Gate explicitly failed Criterion 1 ("no open P0/P1 finding, or UNVERIFIED with documented reasoning and sign-off") because of the WebSocket isolation gap. Phase 11 is defined but implementation has not started. This predates and is independent of this mandate's work — carried forward here so it isn't lost. |
| REL-2 | PR #126 (CVE + Dependabot dependency bumps) | **FIX READY, not merged** | CI green (13/13 checks), independently reviewed, delta re-verified after growing (go1.25 pin fix, otel bump, dompurify override). Awaiting your merge decision. |
| REL-3 | PR #127 (topology/blast-radius correctness fixes) | **FIX READY, not merged** | Independently reviewed APPROVE, full suite green, real-cluster verification done. Awaiting CI completion + your merge decision. |
| REL-4 | winget distribution channel | **BLOCKED** on credential | `WINGET_TOKEN` secret does not exist on `vellankikoti/kubilitics` (confirmed via `gh secret list`, not guessed). Needs a GitHub PAT with push access to `microsoft/winget-pkgs`, added by a human. Until it exists and a real publish+install is verified, winget must be treated as **unverified as a publishing path**, not merely "pending." |
| REL-5 | Windows and Linux desktop installer verification | **UNVERIFIED** | No Windows/Linux environment available in this working session. macOS is evidenced (see §2). Do not claim Windows/Linux installers work without a real install+launch test. |

**Explicit statement per your release gate:** Kubilitics is **not** release-ready as of this document. The missing `WINGET_TOKEN` alone does not invalidate every other channel, but REL-1 through REL-5 above each need an explicit resolution (merge, credential, or environment) before a stable release claim is accurate.

---

## 2. Distribution channels — current verified state

| Channel | State | Evidence |
|---|---|---|
| Homebrew tap (`vellankikoti/homebrew-kubilitics`) | **FIXED (merged)**, verified | Root cause: GitHub auto-disabled the tap's sync workflow after 90 days of inactivity (stuck at v1.2.0 while main shipped 3 releases ahead). Re-enabled, manually triggered, confirmed `Casks/kubilitics.rb` now bumped to v1.2.4 with SHA256 that was independently byte-for-byte verified against the real release DMG. **Residual risk (tracked, not yet fixed):** the underlying 60-90-day auto-disable behavior will recur without a keepalive mechanism — see FOLLOWUP-DIST-1 below. |
| winget | **BLOCKED** | See REL-4. Workflow code itself is correct — it has never silently failed, it has genuinely never run, by design (it no-ops when the secret is absent). |
| Helm chart | Not touched this mandate | Rated "partially verified" by the baseline investigation (Checkpoint A); unchanged here. |
| macOS desktop (signed/notarized .dmg) | Evidenced, not freshly installed | The existing local install's version matches, and its DMG's checksum was independently confirmed correct (see Homebrew row) — this is real evidence, not a fresh install+launch test. |
| Windows / Linux desktop installers | **UNVERIFIED** | See REL-5. |

### FOLLOWUP-DIST-1 — Homebrew tap workflow will auto-disable again
- **Repro/mechanism:** GitHub disables scheduled workflows after ~60-90 days of repository inactivity. This already happened once (stuck at v1.2.0 for ~3 months).
- **Impact:** Recurs indefinitely unless addressed; Homebrew users silently fall behind every time it fires.
- **Fix scope:** Add a low-cost keepalive (e.g., a monthly no-op commit, or convert part of the schedule to also perform a `workflow_dispatch` ping) so the workflow never goes 60+ days without activity.
- **Acceptance criteria:** Tap workflow remains `active` across a full quarter with no manual re-enable required.

---

## 3. Security — CVE and dependency disposition

### 3.1 Code-scanning CVEs (20 findings, original PR #126 scope)
**FIX READY (PR #126, not merged).** Verified: non-breaking bumps across 5 Go modules (`x/net`→0.61, `x/crypto`→0.58, `x/text`→0.43, `grpc`→1.84, `containerd`→1.7.36, `oras-go/v2`→2.6.3). Independent review confirmed each claimed CVE's advisory "fixed in" version is actually what landed (checked against osv.dev), and spot-checked grpc/containerd for real breaking-change risk (none found — containerd confirmed unused/transitive via `go mod why`).

### 3.2 containerd — 3 additional advisories, open, deferred
**DEFERRED — no non-breaking fix exists.**
- GO-2026-5064, GO-2026-5338, GO-2026-5622 (CRI checkpoint-restore smuggling/poisoning/symlink issues).
- Fixed only in containerd v2.x (v2.1.9+/2.2.5+/2.3.2+); no v1.7.x backport exists.
- **Risk assessment:** low in practice — containerd is a transitive dependency not imported by any kubilitics code (`go mod why` returns "main module does not need package"), and the vulnerable feature (CRI checkpoint/restore) is never exercised.
- **Acceptance criteria for closing:** either a v1.7.x backport appears upstream, or a deliberate, separately-reviewed major-version migration to containerd v2 is undertaken.

### 3.3 Dependabot alerts (76 total, surfaced after enabling Dependabot — it was off)
Full dedup against §3.1/§3.2 above:

| Disposition | Count | Detail |
|---|---|---|
| Already covered by PR #126 | 41 | Same packages/advisories as §3.1 (containerd, x/crypto, x/net, grpc, oras-go) at equal-or-lower fix thresholds. |
| **Fixed this pass** (PR #126) | 26 | OpenTelemetry (7 alerts, `go.opentelemetry.io/otel`→1.45.0, CVE-2026-81870) + DOMPurify (19 alerts, scoped `overrides` on monaco-editor's nested copy →3.4.17, verified not to touch the app's own separate top-level dompurify dependency). |
| Dev-only, no action needed | 4 | `braces` (no upstream fix exists at all; tailwindcss devDependency, build-time only), `postcss-selector-parser` (tailwindcss/typography devDependencies), `shell-quote` (2 alerts, Dependabot-scoped `development` on kubilitics-desktop), `uuid` (nested under Storybook addon, `development` scope). |
| **Deferred — tracked follow-up** | 4 | See FOLLOWUP-DEP-1/2/3 below, plus the 3 containerd advisories already covered in §3.2. |

### FOLLOWUP-DEP-1 — i18next-http-backend
- CVE-2026-105800 (low). Current 3.0.6 → fix needs 4.0.2 (major bump).
- **Acceptance criteria / safe path:** isolated bump commit + full frontend i18n test suite + manual language-switch smoke test, in its own reviewed PR.

### FOLLOWUP-DEP-2 — react-router
- CVE-2026-53666 + CVE-2026-53669 (medium). Current 6.30.6 → fix needs 7.18.0 (major, known breaking routing-API changes). **This is the one to prioritize among the three deferred dependency items** — it's the highest severity and sits in a direct, actively-used dependency (routing), not a transitive/build-only one.
- **Acceptance criteria / safe path:** dedicated migration PR following react-router's official v6→v7 guide, with full E2E route-coverage testing — not a drive-by bump.

### FOLLOWUP-DEP-3 — glib (Tauri Linux desktop build)
- GHSA-wrw7-89jp-8q8g (medium), `Cargo.lock`. Current 0.18.5 → fix needs 0.20.0+ (latest stable 0.22.10). gtk-rs ecosystem typically needs coordinated bumps across glib/gio/gdk/gtk.
- **Acceptance criteria / safe path:** requires a Linux build environment (unavailable in this session) to confirm the Tauri Linux target still compiles and launches after a coordinated bump.

---

## 4. Topology and blast-radius engine correctness

### 4.1 Corrections to the original baseline framing (do not reopen as bugs)
- **V2's "depth ≥ 3 is unbounded Full mode"** (collapsing requested depths 3/4/5 into one traversal) is **intentional design**, confirmed by a dedicated test (`TestViewFilter_Resource_FullModeDepthIsUnboundedAboveThree`). The only real defect was the REST layer echoing the raw out-of-range requested value into response metadata whose documented range is 0-3 — see §4.2.
- **V1's topology engine** (`internal/topology/relationships.go`) has **no live frontend caller** (confirmed by in-code comment and route audit). It is not an active source of user-facing V1/V2 disagreement today. The original three-engine framing overstated current risk; the real unreviewed duplication is between V2's BFS filter and the separate `internal/graph` BFS used for blast radius (flagged, not fixed — would require partial unification, out of scope for this mandate).

### 4.2 Fixed (PR #127, FIX READY — not merged)
- **Misleading hop-depth metadata**: API accepted `?depth=`/`?hops=` up to 5 and echoed the raw value in response metadata (documented range 0-3). Clamped; actual traversal behavior for depth≥3 is correctly unchanged. Regression-tested across `depth=4`, `depth=5`, `hops=4`, `hops=5`; verified as a real guard by reverting the fix and watching the test fail with the exact metadata leak.
- **Blast-radius confidence was cluster-global, not per-target**: two different resources in the same snapshot got identical confidence scores/notes regardless of individual PDB/HPA/trace coverage — a false-reassurance risk for an SRE querying an untraced, unprotected resource during a real incident. Fixed to scope PDB/HPA/trace signals to the actual target; endpoints-availability and "large cluster" signals correctly remain cluster-global (they describe data-source wiring, not per-target coverage). Verified: reverting the fix breaks the build against the new test (old signature can't satisfy it); independent reviewer additionally reasoned through a 3rd-target case beyond the happy path and found no residual leakage.
- Both fixes additionally verified end-to-end against a real `kind` cluster, not just unit fixtures (scratch DB, no production data, torn down afterward).

### FOLLOWUP-TOPO-1 — Blast-radius percentage inflates with fan-out
- **Status:** DEFERRED — confirmed real, pinned by a skipped regression test (`TestFollowupTopo1_BlastPercentDoesNotInflateFromFanOut`) that fails for real when un-skipped.
- **Repro:** 1 of 10 cluster workloads broken, with 1 service + 1 ingress + 3 OTel consumers pointing at it, scores **22% blast radius** instead of ~10%.
- **Root cause:** `computeBlastRadiusPercent` sums weighted impact once per `ServiceImpact`, once per `IngressImpact`, and once per `ConsumerImpact` for what can be the same underlying failure — one real failure counted up to 5x — then divides by total workload count (a denominator in a different unit than the numerator).
- **Impact:** Overstates blast-radius severity in proportion to how exposed/consumed a resource is, independent of how many distinct resources actually break.
- **Fix scope:** Redesign the percentage calculation to count distinct affected workloads once, regardless of how many taxonomies register the same underlying failure.
- **Acceptance criteria:** Overlapping paths and taxonomies must not inflate the distinct affected-workload percentage.

### FOLLOWUP-TOPO-2 — OTel trace matching has no namespace disambiguation
- **Status:** DEFERRED — confirmed real, pinned by a skipped regression test (`TestFollowupTopo2_OTelMatchingIsNamespaceScoped`).
- **Repro:** `prod/checkout` (never traced) and `staging/checkout` (traced) share identical confidence scores, and `prod/checkout`'s confidence note falsely cites trace evidence it doesn't have.
- **Root cause:** OTel service-map edge matching (in confidence scoring, traffic-hotspot amplification, and consumer-count logic) compares by bare resource name only (`strings.EqualFold`), with no namespace component.
- **Impact:** Two same-named resources in different namespaces contaminate each other's trace-derived confidence and traffic amplification — a real risk on any multi-tenant or shared-naming (e.g. staging/prod) cluster.
- **Fix scope:** OTel service-map edges need to carry or be resolved against a namespace; every name-match site needs to become namespace-aware. This is a change to the OTel ingestion/matching contract, not a local patch.
- **Acceptance criteria:** Identical names in different namespaces remain isolated unless an explicitly supported cross-namespace relationship establishes a connection.

---

## 5. Dashboard / Fleet / Picker cluster-connection-state (original reported bug)

**FIX READY (PR #119, not merged — predates this mandate).** Fixes the originally-reported Dashboard Health/Capacity contradiction (one card claiming "Grade A, 100%" while the sibling card said "No clusters connected," for the same unreachable cluster) plus two Critical issues (no periodic reachability refresh; fake-healthy states renderable via "connecting"/"unknown," not just "unreachable") found by independent review and fixed with TDD. Manual real-artifact smoke test (Phase 6 of that plan) remains **UNVERIFIED** — noted as a pending checkbox in the PR, not yet executed.

---

## 6. Carried forward from the pre-existing hardening effort (historical, not re-verified here)

`docs/PRODUCTION-HARDENING-ROADMAP.md` Phases 0-9 are marked complete as of 2026-10-02, with evidence in `docs/PRODUCTION-HARDENING-EXECUTION.md`. This mandate did not re-verify that work; it is referenced here only so the Release Gate status (§1, REL-1) is understood in context:
- Phases 1-9 fixed the majority of P0/P1 findings from `docs/PRODUCTION-RELIABILITY-AUDIT.md` (startup blocking, stale-healthy presence, count/metrics correctness, topology namespace scoping, blast-radius pagination/locking, frontend fabricated-health states, observability instrumentation, regression coverage).
- One finding, **CONTAM-1** (cross-cluster contamination risk), was deliberately left dormant/unfixed because the code path it lives in is currently inert — reactivating it without first closing the gap would turn a P1 into a P0. Still open, by design, carried forward here.
- **TOPOLOGY-3** was fixed. Graph correctness, relationship resolution, duplication, and large-cluster topology performance were explicitly left **UNVERIFIED** by that effort — this mandate's §4 above is the first real evidence-based work against that gap, though it only closed 2 of the issues found, not all of them.
- Phase 11 (WebSocket isolation) is the reason Phase 10's Release Gate is blocked — defined, not started, and outside this mandate's scope.

---

## 7. Summary: what's actually fixed vs. what's still open

**Fixed and merge-ready (needs your merge decision, not further work):**
- PR #119 — Dashboard/Fleet/Picker cluster-connection-state single source of truth.
- PR #126 — 20 code-scanning CVEs + 26 Dependabot alerts (otel, dompurify) + go1.25 CI fix.
- PR #127 — Blast-radius per-target confidence + topology depth-metadata clamp.

**Blocked on something outside engineering control:**
- winget (credential).
- Windows/Linux installer verification (environment).
- Phase 11 / Release Gate (scope not yet started — separate decision from this mandate).

**Deferred, tracked, with explicit acceptance criteria (not silently dropped):**
- FOLLOWUP-DIST-1 (Homebrew keepalive).
- FOLLOWUP-DEP-1/2/3 (i18next-http-backend, react-router, glib).
- 3 containerd advisories (§3.2).
- FOLLOWUP-TOPO-1/2 (blast% double-counting, OTel namespace collision).
- CONTAM-1 (carried forward, pre-existing).

**No unsupported claims:** nothing above is marked fixed without a cited test, command, or independent review; nothing is marked merged unless it actually is (currently: nothing in this mandate's scope is merged yet — all three PRs above are open).
