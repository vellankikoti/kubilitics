# Contributing to kcli

Thank you for considering contributing. This page outlines how to get started.

## Prerequisites

- **Go 1.21+**
- **kubectl** on your PATH (for testing cluster commands)
- A **Kubernetes cluster** or **kind/minikube** (optional but useful for integration tests)

## Getting the code

```bash
git clone https://github.com/kubilitics/kcli.git
cd kcli
go mod download
```

## Building and running

```bash
go build -o bin/kcli ./cmd/kcli
./bin/kcli version
./bin/kcli --help
```

See [Building from Source](building.md) for more detail.

## Running tests

```bash
go test ./...
```

See [Testing](testing.md) for test layout and how to run specific suites (unit, integration, alpha smoke, performance).

## Code style

- Follow standard Go conventions (format with `gofmt` or `goimports`).
- Keep packages focused: `internal/cli` for commands, `internal/runner` for kubectl execution, `internal/config` for config, etc.
- Prefer small, testable functions; avoid global state beyond the app struct and config/state files.
- Add tests for new behavior when possible (unit tests in `*_test.go` next to the code).

## Submitting changes

1. **Fork** the repository and create a branch from `main` (or the current development branch).
2. **Implement** your change with tests if applicable.
3. **Run** `go test ./...` and any project scripts (e.g. `./scripts/alpha-smoke.sh`, `./scripts/perf-check.sh`) that are relevant.
4. **Commit** with clear messages (e.g. "Add --output json to kcli events").
5. **Open a pull request** describing the change and referencing any related issues.

## Areas where help is welcome

- **Documentation** — Improving user and developer docs in `docs/`.
- **Tests** — More unit and integration coverage, especially for observability, incident, and AI paths.
- **Plugins** — Official plugins, marketplace metadata, and plugin SDK examples.
- **Performance** — Startup time, completion latency, TUI refresh with large lists.
- **Platform** — Packaging (Homebrew, apt, Chocolatey, etc.) and CI for multiple OS/arch.

## Reporting bugs

Open an issue with:

- kcli version (`kcli version`)
- OS and arch
- Steps to reproduce
- Expected vs actual behavior
- Relevant config (redact secrets) or command lines

## Feature requests

Open an issue describing the use case and proposed behavior. Discussion may lead to a design doc or task breakdown before implementation.

## License and conduct

Contributions are subject to the project’s license (see repository). Be respectful and professional in all interactions.
