# kcli Documentation

kcli is the Kubilitics terminal core: a unified Kubernetes CLI with kubectl parity, context and namespace ergonomics, observability shortcuts, incident mode, and optional AI.

## User Documentation

| Document | Description |
|----------|-------------|
| [Getting Started](user/getting-started.md) | Quick start and first commands |
| [Installation](user/installation.md) | Install kcli on macOS, Linux, and Windows |
| [Command Reference](user/command-reference.md) | All commands, flags, and examples |
| [TUI Guide](user/tui-guide.md) | Interactive terminal UI (keybindings and views) |
| [AI Features](user/ai-guide.md) | Optional AI providers and commands |
| [Plugins](user/plugin-guide.md) | Install, develop, and run plugins |
| [Configuration](user/configuration.md) | Config file and `kcli config` reference |
| [Troubleshooting](user/troubleshooting.md) | Common issues and fixes |
| [FAQ](user/faq.md) | Frequently asked questions |

## Developer Documentation

| Document | Description |
|----------|-------------|
| [Architecture](developer/architecture.md) | Code layout, components, and data flow |
| [Contributing](developer/contributing.md) | How to contribute to kcli |
| [Plugin Development](developer/plugin-development.md) | Building plugins and manifest format |
| [Building from Source](developer/building.md) | Build and run from source |
| [Testing](developer/testing.md) | Running tests and writing new tests |

## Quick Links

- **Build:** `go build -o bin/kcli ./cmd/kcli`
- **Shell completion:** `kcli completion bash` \| `kcli completion zsh`
- **Config file:** `~/.kcli/config.yaml`
- **State file:** `~/.kcli/state.json`
- **Plugins directory:** `~/.kcli/plugins/`

## Searchable docs site

To build and serve a searchable documentation site locally:

```bash
pip install mkdocs mkdocs-material
cd kcli/docs
mkdocs serve
```

(MkDocs is run from `kcli/docs`; `docs_dir` is set to `.` so that the same directory is used as content.)

Then open http://127.0.0.1:8000. Use the search box for full-text search. To build static files only: `mkdocs build` (output in `site/`).
