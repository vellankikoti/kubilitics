# Performance (Kubilitics)

Pointers, not a duplicate — the real investigation records live in
`docs/` (e.g. `RESOURCE-LIST-PAGINATION-N1-INVESTIGATION.md`,
`ENTERPRISE-PERFORMANCE-CAMPAIGN.md`) and in `docs/releases/*.md` for
what shipped. Read those for a specific past finding before
re-investigating.

## Established patterns, don't rediscover

- **Informer cache is the fast path.** `ListFromCache` reads are sub-ms;
  anything NOT in `resourceKindToStoreKey` (see `docs/ai/ARCHITECTURE.md`)
  falls back to a live K8s API call every time — slower and more exposed
  to transient failures.
- **Unbounded `ListOptions{}` calls are a repeat bug class.** Check for a
  missing `Limit:` on any new `.List(ctx, metav1.ListOptions{})` call,
  especially for cluster-wide Pods/Events on a function that has sibling
  calls which DO set a limit (copy-paste miss pattern — happened at least
  once, caught in `workloads.go`).
- **N+1 fan-out across clusters needs `errgroup.SetLimit()`.** Fleet/
  overview endpoints that loop over all registered clusters must bound
  concurrency, not fire unbounded goroutines per cluster.
- **Measure before fixing.** Use `buildTimeMs` (where present) as a
  structural proof of cache-hit vs. genuine-rebuild, not just latency
  feel. Don't claim a performance fix worked without a before/after
  number.

## Before/after discipline

Never say "faster" or "fixed the N+1" without an actual measurement in
the PR/commit message. The project's historical pattern (documented in
investigation docs) is real before/after numbers — e.g. "~105x speedup,"
"759x/337x improvement" — not qualitative claims.
