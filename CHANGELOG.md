# Changelog

All notable changes to Kubilitics will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v0.1.0] - 2026-03-28

First public release of Kubilitics. Clean version reset with full security audit,
release pipeline hardening, and production readiness.

### Added

**Desktop App (Tauri 2.0)**
- macOS (universal), Windows (x64), Linux (x64) desktop application
- Go backend sidecar with automatic health monitoring and restart
- Auto-detect kubeconfig from `~/.kube/config` with multi-context support
- AES-256-GCM encrypted kubeconfig storage
- Check-for-updates via GitHub Releases API

**Kubernetes Dashboard**
- Real-time resource monitoring via WebSocket informer streams
- 51 resource detail pages with unified SectionCard design
- Multi-cluster management with fleet overview
- Topology visualization (React Flow + ELK layout) with export
- In-browser terminal with kcli (kubectl wrapper) integration
- Port forwarding, log viewer, pod exec

**Backend (Go)**
- REST API + WebSocket with RBAC (Admin/Operator/Viewer)
- JWT auth, API key auth, MFA/TOTP, OIDC/SAML SSO support
- Rate limiting (token bucket per IP), circuit breaker
- SQLite (desktop) / PostgreSQL (in-cluster) database
- Kubernetes Secret data redaction in API responses
- Version reporting via `/health` endpoint

**In-Cluster Deployment**
- Helm chart with NetworkPolicies, security contexts, RBAC
- cert-manager integration for TLS
- Multi-arch Docker images (amd64, arm64)
- Production values file with all security hardening enabled

**CI/CD Pipeline**
- Version consistency check across 6 files
- CI gate before release (verifies all checks passed)
- Trivy scanning for both backend and frontend Docker images
- govulncheck, npm audit, Semgrep, Gitleaks, Kubescape
- Dependabot for Go, npm, Cargo, and GitHub Actions
- Checksums generation for all release artifacts

### Security

- Backend binds to `127.0.0.1` by default (desktop), `0.0.0.0` for in-cluster
- Kubernetes tokens and kubeconfig credentials excluded from localStorage persistence
- Zustand store migration cleans stale credentials from previous versions
- Content Security Policy without `unsafe-eval`
- Security headers (CSP, HSTS with preload, X-Frame-Options, etc.)
- WebSocket Origin validation with auth fallback
- Secure temp file creation (`os.CreateTemp`) prevents symlink attacks
- Docker entrypoint sanitizes environment variables before JS injection
- nginx proxy port corrected (was 819, now 8190)

### Architecture

```
Kubilitics
├── kubilitics-desktop  (Tauri 2.0 host, Rust)
├── kubilitics-frontend (React + TypeScript + Vite SPA)
├── kubilitics-backend  (Go REST API + WebSocket, SQLite/PostgreSQL, port 8190)
└── kcli                (kubectl wrapper CLI, Go — separate repo)
```

---

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for how to contribute to Kubilitics.

## License

Apache 2.0 - See [LICENSE](LICENSE) for details.
