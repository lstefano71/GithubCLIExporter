# Copilot CLI Session Exporter

[![Release](https://img.shields.io/github/v/release/lstefano71/GithubCLIExporter?style=flat-square)](https://github.com/lstefano71/GithubCLIExporter/releases/latest)

Export GitHub Copilot CLI sessions to readable **Markdown** and **HTML** formats.

This repository contains the primary, maintained implementation in Go (located in `go-cli/`). The Go CLI is distributed as a standalone binary that provides the `list` and `export` commands.

**Quick links:**

- **Cheat sheet:** [docs/cheatsheet.md](docs/cheatsheet.md)
- **Build & install:** [docs/build.md](docs/build.md)
- **Technical docs:** [docs/technical.md](docs/technical.md)
- **Releases:** https://github.com/lstefano71/GithubCLIExporter/releases/latest

## Quick Start (cheat sheet)

List available sessions (scans `~/.copilot/session-state/` by default):

```bash
copilot-export list
copilot-export list --repo "*MyRepo*"
copilot-export list --search "fix bug"
copilot-export list --since 2026-03-10
```

Export a session (specifier can be index, partial UUID, full UUID, or path):

```bash
copilot-export export 658a                 # partial UUID
copilot-export export 3                    # index from `list`
copilot-export export ./sessions/01        # path
copilot-export export                       # interactive picker
copilot-export export 658a --format md
copilot-export export 658a --format html
copilot-export export 658a --format both   # default
copilot-export export 658a --output ./my-export
```

Common flags:

- `--sessions-dir` — override sessions directory (default: `~/.copilot/session-state/`)
- `list` flags: `--repo`, `--since`, `--search`
- `export` flags: `--format` (`md|html|both`), `--output`

For the full cheat sheet and examples see [docs/cheatsheet.md](docs/cheatsheet.md).

## Build & Install

See [docs/build.md](docs/build.md) for platform-specific build and install instructions for the Go CLI.

## Technical Docs

Implementation details, export format, and rendering notes are in [docs/technical.md](docs/technical.md).

## Releases & Versioning

Pre-built binaries are published on the Releases page: https://github.com/lstefano71/GithubCLIExporter/releases

### Version info in the binary

The Go CLI embeds version information and exposes it via `copilot-export --version`.

---
If you previously used an earlier prototype of this project, that proof-of-concept has been superseded by the Go CLI implementation. See the technical docs for migration notes.
