# Contributing to kcli

## Prerequisites

- Go 1.24+
- kubectl configured with at least one reachable context
- Unix-like shell (zsh/bash)

## Local workflow

```bash
cd kcli
go test ./...
./scripts/perf-check.sh
```

Before opening a PR:

1. Run unit tests (`go test ./...`).
2. Run performance gate (`./scripts/perf-check.sh`).
3. Validate CLI help and one live command against your cluster.

## Coding standards

- Prefer deterministic behavior and explicit errors.
- Keep kubectl parity changes backward compatible.
- Avoid placeholder logic for production paths.
- Add tests for behavioral fixes and regressions.

## Commit/PR checklist

1. Problem statement and scope are clear.
2. Implementation includes tests.
3. Performance-sensitive changes include measured impact.
4. User-facing docs updated (`README.md`, `COMMANDS.md`, or task docs).
