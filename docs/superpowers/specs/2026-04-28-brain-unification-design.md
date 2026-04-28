# Brain Unification (Phase 2 #1) — Design

**Status:** Spec
**Date:** 2026-04-28
**Author:** Phase 2 #1 brainstorm session
**Prereqs:** Phase 1 LLM-as-translator architecture (merged dfda2f50), Phase 2 #2-#6 (merged 9f5ad750 / 5e78cff4 / 4a8c6e1b / 83e96a23 / 1be01cf3)
**Successor:** A future "extract to standalone repo `kotg-llm-translator`" project, when API stabilises and a third consumer appears.

---

## Problem

Two Go modules — `vellankikoti/kubilitics/brain` (desktop chat) and `kubilitics/kotg-toolserver` (in-cluster MCP) — duplicate ~17 files of the LLM-as-translator architecture I shipped in Phase 1+2.

Today they are kept in sync by manual `sed`-port (every PR I land in brain, I copy-with-module-path-rewrite to toolserver). That worked for 5 Phase 2 PRs but is unsustainable:

- Two real conflicts surfaced during Phase 2 progressive merges (`registry.go` and `derive.go` when #98 inspect_* expansion and #101 LogBlock both touched the same lines from different angles).
- Every shaper, every renderer, every prompt tweak needs to be done twice — and the two copies *can* drift, even when they shouldn't.
- The Phase 1 spec's "single chokepoint" guarantee is weakened: there are now two `BuildDeterministicResponse` functions, and the architecture-test that asserts "only this function emits render_block events" runs in two places independently.

## Empirical scope

Of 21 LLM-as-translator files I shipped:

- **17 are byte-identical mod module path** between brain and toolserver
- **4 differ** only because of incomplete merge propagation (`registry.go`, `derive.go`, `shapers/pods.go`, `llm/types/tool_execution.go`) — those reconcile to identical once Phase 2 PRs all settle

Pre-existing drift (NOT my concern, NOT in this spec):
- `internal/llm/budget` — 5 brain files vs 3 toolserver files
- `internal/llm/provider/openai` — 8 brain files vs 3 toolserver files

These are deferred to a future "Phase 3 provider/budget reconciliation" project. Doing them in this PR series would force product policy decisions (which provider abstraction wins, which budget enforcement is canonical) that aren't mine to make.

## Goals

1. **Zero sed-port from now on.** Any future change to the translator lives in one place; both consumers pick it up via `go.mod` pin.
2. **Architectural invariants enforced once.** The "single chokepoint emits render_block", "type fence on summary.Generate", "shaper coverage" tests run in the shared package, not in each consumer.
3. **No regression in either consumer.** Brain's chat and toolserver's MCP path produce identical render_block events to today.
4. **Designed-for-extraction.** The shared module lives inside the kubilitics monorepo for now (D-strict per brainstorm Q2), but is structured so that promoting it to a standalone repo (`kotg-llm-translator`) later is mechanical: just move the directory, update go.mod paths.

## Non-goals

- Unifying `internal/llm/budget` or `internal/llm/provider/openai`. Out of scope; deferred to a separate spec.
- Unifying the 8 provider tool loops (`provider/{anthropic,openai,custom,ollama}/tool_loop.go`). They're full of provider-specific quirks (Anthropic ContentBlocks, OpenAI tool_calls, Ollama streaming format). Lifting them is a separate Phase 3+ project. They keep calling `translator.WrapExecutorForRender` exactly as today; only the wrap function moves.
- Changing wire format. `kotg-schema` proto is unchanged. Frontend is unchanged. Backend WS handler is unchanged.
- Performance work. The translator code is already fast enough; this is a structural change only.

## Architecture

### Where the code lives — D-strict (brainstorm Q2)

A new internal monorepo module at:

```
kubilitics/llm/
  contract/
  translator/
```

`brain` and `kotg-toolserver` both import from `github.com/vellankikoti/kubilitics/llm/...`. During development this is a `replace` directive in their go.mod (same pattern Phase 1 used while consuming the unreleased kotg-schema). After the unification PR series merges, the consumers pin to a real git SHA / tag.

**Why D over A (new repo) or C (kotg.ai monorepo):** A is too early — release ceremony per change would slow Phase 3+ iteration. C inverts dependency direction (org repo `kubilitics/kotg.ai` becoming upstream of `vellankikoti/kubilitics`) which is a governance problem. D keeps friction low while preserving extraction optionality.

### Package layout — C-disciplined (brainstorm Q3)

```
kubilitics/llm/
  go.mod                              # github.com/vellankikoti/kubilitics/llm
  go.sum

  contract/                           # Pure types. Zero behaviour. Zero non-stdlib deps.
    event.go                            AgentStreamEvent, RenderBlockEvent, ToolEvent
    config.go                           AgentConfig (NO env-var reading)
    derived.go                          DerivedSummary (the type fence)
    render_types.go                     Class, RenderType, ToolBehavior
    contract_test.go                    json-tag snapshot tests (catches accidental wire breaks)

  translator/                         # The 5 primitives + thin orchestration.
    registry.go                         Lookup() — the rollout knob
    chokepoint.go                       BuildDeterministicResponse — single emitter
    derive.go                           derive() — counts-only summary derivation
    wrap.go                             WrapExecutorForRender — the runtime fence
    run.go                              Run() — thin orchestration (≤80 lines, CI-enforced)
    *_test.go                           registry/chokepoint/derive/wrap unit tests
    hallucination_probes_test.go        the 30-probe suite

    summary/                          # Sub-package: isolates the LLM-call seam.
      summary.go                        Generate, defaultDeterministicFormatter
      llmcompleter.go                   NewLLMCompleter, SetCompleter, SwapLLMCompleterForTest
      *_test.go

    shapers/                          # Per-tool transforms.
      pods.go                           list_resources, get_resource, list_pods, get_pod_yaml
      logs.go                           get_logs
      inspect.go                        27 inspect_<kind> registrations
      *_test.go
      fixtures/                         shared fixture JSONs
```

### Design constraints (CI-enforced where possible)

1. **`contract/` has zero non-stdlib imports.**
   *Enforcement:* `depguard` rule denies any non-stdlib import in `contract/**/*.go`.
   *Test:* `TestContractStdlibOnly` parses `go list -m all` for the contract package and asserts the import set is a subset of the stdlib package list.

2. **`translator/` imports `contract/` and stdlib only — no `internal/llm`, no provider packages, no audit/budget.**
   *Enforcement:* `depguard` rule denies imports of `kubilitics/brain/internal/llm`, `kotg-toolserver/internal/llm`, `kubilitics/brain/internal/audit`, etc.

3. **`Run()` is ≤ 80 lines AND has no business-logic branching.** A line-count cap is a proxy; the real rule is structural. Both are CI-enforced.
   *Test 1 (LOC):* `TestRunStaysThin` reads `run.go`, counts non-blank non-comment lines, fails on >80.
   *Test 2 (no branching):* `TestRunOnlyComposes` parses `run.go` AST and asserts:
     - No `if`/`switch` whose condition references provider names (`anthropic`, `openai`, `ollama`, `custom`), audit/budget identifiers, or any name matching `(?i)policy|strategy|mode|enable|disable`.
     - No call to anything outside the `translator/` and `contract/` packages.
     - The function body must be a straight pipeline: `reg := …; wrapped := …; ev := …; return …`.
   *Override:* a `// orchestration-cap-override: <reason>` magic comment suppresses the LOC test only; the no-branching test cannot be suppressed. Any override is logged in CI summary and is an automatic code-review flag.

4. **Single emitter invariant carries over.** `BuildDeterministicResponse` is still the only constructor of `render_block` events; `TestSingleEmitter` walks the package call graph (using `golang.org/x/tools/go/packages`) and asserts no other function literal has the shape `Event{Kind: "render_block", ...}`.

5. **Type fence on `summary.Generate` carries over.** `Generate(ctx, derived.DerivedSummary) (string, error)` — the signature compile-time forbids passing arbitrary tool data into the LLM. Test pins the signature.

6. **Shaper coverage carries over.** `TestEveryDeterministicToolHasShaper` lives in the unified `translator` package now (one place, one truth).

### Event stream is part of the contract (not translator detail)

The single most dangerous remaining drift vector. `Run()` could be byte-identical and behaviour could still diverge if the event sequence isn't pinned. Lifting this out of "implementation behaviour" and into "contract" closes that hole.

**Pinned in `contract/event.go` as documentation + enforced by `contract/event_test.go`:**

| Property | Rule |
|---|---|
| Event types | Exactly the 5 fields on `AgentStreamEvent` today: `TextToken`, `ToolEvent`, `RenderBlock`, `Done`, `Err`. New event variants require a contract bump (see versioning). |
| Per-event invariant | Exactly one of `{TextToken≠"", ToolEvent≠nil, RenderBlock≠nil, Done==true, Err≠nil}` is set per event. Any combination of two is a contract violation. |
| Sequence start | The first event a consumer sees on a fresh stream is **NOT** required to be a sentinel. Streams begin lazy. (This matches today's behaviour; pinning it.) |
| Tool lifecycle | Every `ToolEvent.Phase == "calling"` MUST be followed (eventually, on the same channel) by either `Phase == "result"` OR `Phase == "error"` with the same `CallID`. No orphan calling-events. |
| Render-block lifecycle | `RenderBlock` events are emitted by the executor wrapper *between* the tool's `calling` and the LLM's next response token. They are independent of the tool's `result` event — both flow. |
| Terminal | After `Done == true` OR `Err != nil`, the channel MUST be closed. Consumers MUST treat `Done` and `Err` as mutually exclusive terminals. |
| Ordering | Events from a single agentic turn are FIFO from producer's perspective. The buffer is 64; producers MUST NOT skip events on backpressure (would silently drop a render_block). If the channel is full, the producer blocks. |

`contract/event_test.go` includes:
- `TestEventOnlyOneFieldSet` — fuzz: build random `AgentStreamEvent` values, assert exactly-one-of holds for each.
- `TestEventTerminalClosesChannel` — fake producer + consumer, assert `Done`/`Err` is the last value before close.
- `TestToolLifecycleHasMatchingResult` — golden trace of an agentic turn, asserts every `calling` has a matching `result` or `error`.

These tests live in `contract/`, not `translator/`. The contract is what they test.

### Versioning discipline (even before standalone-repo extraction)

Once `kotg-toolserver` (separate Go module) imports from `vellankikoti/kubilitics/llm/...`, every change in the shared module is a cross-module change. "Just merge to main" silently changes downstream behaviour on next `go mod tidy`. The discipline below applies from PR 1 onward.

**Change classification (every PR to `kubilitics/llm/` declares one of):**

| Class | Definition | Examples | Required |
|---|---|---|---|
| `patch` | Internal refactor, no observable behaviour change. | Rename a private helper. Reorder map keys. Split a file. | Tests stay green. CHANGELOG entry under "internal". |
| `minor` | Additive only. New event-stream fields are NOT minor (see major). | New `RenderType` constant. New shaper for a new tool. New `Run()` option that defaults to today's behaviour. | CHANGELOG "added" entry. Bench baseline re-snap if perf-relevant. |
| `major` | Behaviour change visible to consumers. | Event ordering changes. New required field on `AgentStreamEvent`. Removed function. Default value of any `AgentConfig` field changes. Default prompt of `summary.Generate` changes. | CHANGELOG "changed" entry. Explicit "consumer impact" section listing brain + toolserver call sites that need review. Both consumers' SHAs called out. |

**Pinning:**
- `kotg-toolserver`'s `go.mod` MUST pin to a commit SHA (or tagged version), never `main`.
- `brain` consumes from the local replace during dev; CI builds use the same SHA pin as toolserver.
- A `LLM_TRANSLATOR_REQUIRED_SHA` env var is checked in both consumer CI suites; if the consumer's `go.mod` pin doesn't match, CI fails. (Prevents "I forgot to bump" silent skew.)

**File:** `kubilitics/llm/CHANGELOG.md` — kept up to date in the same PR as the change. Empty PRs (docs-only) declared `patch` and noted as such.

### Backpressure model — who owns the channel

The "no-skip-on-backpressure" rule above guarantees correctness (no silently-dropped render_blocks) but introduces a liveness risk: a slow consumer can stall the LLM loop because the translator's send blocks on a full channel. This is the most common cause of "works in dev, hangs in prod" in streaming systems. Lock responsibility explicitly:

| Layer | Responsibility |
|---|---|
| Translator (producer) | MUST NOT drop events. MAY block indefinitely on `evtCh <- event` when the channel is full. MUST honour `ctx.Done()` while blocked (use `select` with `case <-ctx.Done(): return`). |
| Consumer (brain runtime, toolserver runtime) | MUST drain the event channel from a dedicated goroutine. MUST NOT read events synchronously on the same goroutine that called `translator.Run()` or invoked the LLM. Slow consumers MAY exert backpressure (the translator will block, which is correct), but MUST NOT block forever — request-level timeout via `ctx` is the consumer's responsibility. |
| Channel buffer | Recommended size: **64** (matches today's value across all 8 provider tool loops; empirically sufficient for normal-cluster turn sizes including a 100-pod table + LLM tokens). Consumers MAY override if they have measured reason; translator does not enforce a specific size. |

*Test in `contract/event_test.go`:* `TestProducerHonoursContextCancel` — fake producer with a 1-element channel, fake consumer that doesn't drain, parent context cancelled mid-stream; assert the producer goroutine exits within 100ms (not deadlocks).

This rule is what prevents the classic streaming bug where a UI hang silently freezes the whole agentic loop.

### Tool execution boundary — who owns retries + error normalization

`WrapExecutorForRender` treats the inner `ToolExecutor` as a black box. To keep that boundary clean across both consumers, lock the responsibility split explicitly:

| Behaviour | Owner | Why |
|---|---|---|
| Tool call retry on transient failure | **Consumer's executor** (NOT translator) | Retry policy is per-deployment (in-cluster vs desktop have different network reliability budgets). Translator stays deterministic. |
| Error normalization (HTTP 5xx → user-facing message) | **Consumer's executor** | The translator sees only `(string, error)`; structured error decoration belongs to the layer that knows the deployment context. |
| Emitting `ToolEvent{Phase:"calling"}` | **Translator** | Always emitted before invoking the inner executor. |
| Emitting `ToolEvent{Phase:"result"}` on success | **Translator** | Always emitted after a non-error executor return. |
| Emitting `ToolEvent{Phase:"error"}` on failure | **Translator** | Always emitted after an error return — including normalized errors that the executor returned as `(result, nil)` with structured error inside `result`. (The executor decides; the translator just reports what it got.) |
| Recovering from executor `panic()` | **Consumer's executor** | The translator does NOT install a recover() around tool calls. A panicking executor is a consumer bug; surfacing it is more honest than silently swallowing. |
| Per-tool timeout | **Consumer's executor** | Translator passes ctx through; per-tool deadlines are the consumer's concern. |

**Translator hard-rules (enforced by `wrap_test.go`):**
- The translator MUST NOT call the inner executor more than once per `WrapExecutorForRender → Execute` invocation. Test asserts call count == 1 for both happy path and error path.
- The translator MUST NOT modify the executor's return value before emitting events (no message rewrites, no error wrapping). Test asserts the bytes flowing into `BuildDeterministicResponse` are byte-equal to what the inner executor returned.

This split keeps:
- **Translator** = deterministic and decision-free.
- **Consumer** = policy owner (retry strategy, error UX, panic handling, timeouts).

### Production-mindset invariants

These three properties are easy to assume and silently lose. Encoding them as tests now prevents the class of bug that surfaces only at scale.

**1. Context propagation is end-to-end.**
The translator's `ctx` MUST be passed through to every blocking call:
- The inner `ToolExecutor.Execute(ctx, …)`.
- The summary completer (`summary.Generate(ctx, …)`, which calls the LLM).
- Every channel send (`select { case ch <- ev: case <-ctx.Done(): return }`).

No `context.Background()` in `translator/`. No `context.TODO()`. Enforced by `TestContextPropagationEndToEnd` in `translator/`:
- Cancel a parent ctx mid-stream.
- Assert the inner executor's ctx-cancel signal fires within 50ms.
- Assert the event channel closes within 100ms of parent cancel.
- Assert no orphaned goroutines remain (see invariant 2).

**2. Zero goroutine leaks.**
Every goroutine the translator spawns MUST exit on either `ctx.Done()` OR channel close — no blocking-forever paths. Enforced by `TestNoGoroutineLeak` using `runtime.NumGoroutine()` delta:
- Snapshot goroutine count.
- Run `Run()` to completion (and separately, with cancellation).
- Allow up to 100ms for graceful shutdown.
- Assert `NumGoroutine()` returns to baseline ±0.

This catches the classic "channel send blocks forever because nobody is reading" + "goroutine that owned the producer never exits" combo.

**3. Translator is deterministic.**
Given the same `(input, LLM responses, executor outputs)` triple, the translator MUST produce the same event sequence byte-for-byte. This means:
- **No `time.Now()` in `translator/` business logic.** Timestamps belong to the consumer if needed; the translator carries opaque payloads. (`humanAge()` in `shapers/` is the one exception — it operates on a passed-in `time.Time` from the tool result, not on wall-clock-now. CI-enforced via `TestNoWallClockInTranslator` that greps for `time.Now\(\)` outside fixture/test files.)
- **No `math/rand` or `crypto/rand`.** Tool-call IDs come from the LLM provider; render block IDs come from the consumer if at all.
- **No global mutable state** other than the `summary.llmCompleter` seam (which is test-swappable AND production-swap-once-at-startup-via-`SetCompleter`; both modes are deterministic per-call).
- The `summary.SwapLLMCompleterForTest` seam is the only sanctioned mutation point and is documented as such.

Determinism is not just a code-quality nicety here — it's what makes the byte-equal e2e parity gate (acceptance #3) meaningful. Without it, "same render_block events" becomes "events that look similar to a human", which is unfalsifiable.

**Why these matter together:** the three properties compound — they unlock replay-based debugging (re-run a failed turn from a recorded LLM trace and see identical output), audit trail integrity (the contract event sequence IS the audit record), and future trace-based simulation (e.g., for the brain-test suite to replay production turns offline). Most teams discover they wanted these only after a production incident; encoding them upfront is cheap.

### Consumer-side changes

Per consumer (brain + toolserver), the migration is:

- Delete `internal/derived/`, `internal/render/`, `internal/llm/summary/`, `internal/llm/executor_wrapper*.go`, `internal/llm/fault_injection_test.go`, `internal/llm/types/render_event_test.go`. Their content now lives in the shared module.
- The `internal/llm/types/tool_execution.go` shrinks: only the consumer-specific bits stay (`DefaultAgentConfig` with env-var override for brain, plain default for toolserver). Common types (`AgentStreamEvent`, `RenderBlockEvent`, `ToolEvent`, `ToolExecutor`, `AgentConfig`) move to `contract/`. The local file becomes a thin re-export or a constructor-only file.
- The 4 `internal/llm/provider/*/tool_loop.go` files stay where they are. The line `executor = llm.WrapExecutorForRender(executor, evtCh, cfg.Namespace)` becomes `executor = translator.WrapExecutorForRender(executor, evtCh, cfg.Namespace)`. Same call shape, new import.
- `cmd/server/main.go` continues to call `summary.SetCompleter(summary.NewLLMCompleter(bridge))` (Phase 2 #5 wiring). Path becomes `translator/summary` instead of `internal/llm/summary`.
- Frontend is untouched. The wire envelope (`render_block` JSON shape) is unchanged because the contract types are byte-equivalent to today's.

### Migration sequencing — multi-PR plan

Designed so each PR is independently reversible and the codebase compiles + tests pass after every step. **No "big bang"; no "everything at once."**

- **PR 1 — Create the shared module skeleton.** Adds `kubilitics/llm/contract/` and `kubilitics/llm/translator/` with copies of the current brain code. Both consumers continue to use their own copies. The shared module compiles + has its own tests passing, but is not yet imported. Reversible: delete the directory.
- **PR 2 — Brain switches over.** Brain's `internal/derived`, `internal/render`, `internal/llm/summary`, `internal/llm/executor_wrapper.go`, and the type-fence bits of `internal/llm/types` are deleted. All call sites in brain re-point to the shared module via go.mod replace + pseudo-version pin. All brain tests pass. Reversible: revert the PR.
- **PR 3 — Toolserver switches over.** Same as PR 2 but for `kotg-toolserver`. Brain stays on its existing pin; toolserver pins to the same SHA. Both consumer test suites green.
- **PR 4 — Drop go.mod replace, pin to released SHA.** Once PR 1 is squash-merged on `vellankikoti/kubilitics` main, both consumers replace their `replace` directive with a real `go.mod` pin to the merge SHA. Pure mechanical, mostly checksum updates. Reversible: revert the go.mod swap.

After PR 4 lands, manual sed-port is dead. Future Phase 3+ changes happen once in `kubilitics/llm/`.

### Failure modes + rollback

- **PR 1 reveals a hidden import cycle.** The brain's `internal/render` currently imports `internal/llm/summary`; in the shared module both move together so this resolves. Tested by the build itself.
- **PR 2 fails CI on brain.** Revert the brain switchover; toolserver is untouched; sed-port resumes for any in-flight Phase 3 work until the issue is fixed.
- **A shaper used by toolserver but not brain (or vice versa) is discovered.** The shared `Shapers` map covers all tools that exist in *either* registry today. If toolserver's MCP grows a new tool that brain doesn't know about, it registers it in its own `cmd/server/main.go` via a `translator.RegisterShaper(name, fn)` extension hook. (This hook does NOT exist today; we add it in PR 1 because it's the only structural concession to the consumer drift problem.)
- **Performance regression.** The translator code is already in a hot path; `go test -bench` baselines run in PR 1 and re-run after each consumer switchover. Threshold: no >5% latency regression on `BenchmarkBuildDeterministicResponse_ListPods13`.

### What this does NOT change

- `kotg-schema` is unchanged. The translator is a Go-runtime layer above the wire schema; the wire schema is independently versioned.
- The consumer-side `internal/llm/provider/*/tool_loop.go` files keep their provider-specific event shapes intact. Lifting them is a future project.
- The Phase 2 #5 LLM-summary wiring stays the same: `summary.SetCompleter(summary.NewLLMCompleter(bridge))` in each consumer's `main.go`, just with the new import path.

## Acceptance gates

The PR series is done when ALL hold:

1. `go test ./...` passes in `kubilitics/llm/`, `kubilitics/brain/`, and `kotg-toolserver/`.
2. The 30 hallucination probes (`TestHallucinationProbes`) run in the shared module exactly once, not in each consumer.
3. `cargo tauri dev` against docker-desktop produces identical render_block events for the canonical 6-probe e2e (list_pods kube-system, list_pods default, get_resource coredns YAML, list_pods nonexistent-ns, summarize cluster health, list+followup-restarts) as today.
4. The 27 inspect_* tools, get_logs (LogBlock), and get_resource (yaml_block) all hit the structured-render path 5/5 times in stochastic e2e probes.
5. `grep -r "BuildDeterministicResponse" brain/ kotg-toolserver/` returns ZERO matches outside import statements (the function lives only in the shared module).
6. `wc -l brain/internal/llm/executor_wrapper.go kotg-toolserver/internal/llm/executor_wrapper.go` returns "no such file or directory" — the duplicates are gone.

## Open questions deferred to plan-time

- **Exact go.mod module path:** `github.com/vellankikoti/kubilitics/llm` is the obvious choice. Alternative: `github.com/vellankikoti/kubilitics-llm` as a sibling module. Will pick at plan-time based on directory layout convenience.
- **Which test framework for the architecture tests** (`TestRunStaysThin`, `TestSingleEmitter`)?  Stdlib `testing` + `go/parser` is enough; not pulling in a third-party AST library.
- **Bench baseline file location:** likely `kubilitics/llm/translator/bench_baseline.json` so CI compares against a checked-in number.

These are mechanical and don't change the architecture; deferred so the spec stays focused on shape.
