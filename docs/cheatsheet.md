# Copilot Export — Cheat Sheet

Quick examples for the Go CLI (`copilot-export`).

List sessions

```bash
copilot-export list
copilot-export list --repo "*MyRepo*"
copilot-export list --search "fix bug"
copilot-export list --since 2026-03-10
```

Export sessions

```bash
copilot-export export 658a              # partial UUID prefix
copilot-export export 3                 # index from `list`
copilot-export export ./sessions/01     # path
copilot-export export                    # interactive picker
copilot-export export 658a --format md
copilot-export export 658a --format html
copilot-export export 658a --format both  # default
copilot-export export 658a --output ./my-export
```

Flags summary

- `--sessions-dir` — override sessions directory (default: `~/.copilot/session-state/`)
- `list` flags: `--repo`, `--since`, `--search`
- `export` flags: `--format` (`md|html|both`), `--output`

Notes

- If `export` is called without a specifier, an interactive picker lists sessions for selection.
- The tool supports partial UUID prefix matching (git-style) and index selection from `list` output.
