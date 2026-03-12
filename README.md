# Copilot CLI Session Exporter

Export [GitHub Copilot CLI](https://githubnext.com/projects/copilot-cli) sessions to readable **Markdown** and **HTML** formats.

## Install

```bash
pip install -e .
```

## Usage

### List sessions

```bash
copilot-export list
copilot-export list --repo "*Leviathan*"
copilot-export list --search "fix bug"
copilot-export list --since 2026-03-10
```

### Export a session

```bash
# By partial UUID (git-style prefix matching)
copilot-export export 658a

# By index number from list output
copilot-export export 3

# By directory path
copilot-export export ./sessions/01

# Interactive picker (omit session specifier)
copilot-export export

# Choose format
copilot-export export 658a --format md
copilot-export export 658a --format html
copilot-export export 658a --format both   # default

# Custom output path
copilot-export export 658a --output ./my-export
```

### Environment variables

- `COPILOT_SESSIONS_DIR` — override the default sessions directory (`~/.copilot/session-state/`)

## Features

- **Session discovery**: List all sessions with summary, repository, timestamps
- **Flexible session selection**: partial UUID prefix, index number, full UUID, directory path, or interactive TUI
- **Markdown export**: Conversation view with collapsible tool calls, thinking sections, and sub-agent details
- **HTML export**: Self-contained single file with dark/light theme toggle, TOC, responsive layout
- **Full metadata**: Session plan, todos, checkpoints, statistics
