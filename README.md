# Copilot CLI Session Exporter

Export [GitHub Copilot CLI](https://githubnext.com/projects/copilot-cli) sessions to readable **Markdown** and **HTML** formats.

## Install

```bash
pip install -e .
```

Alternatively you can run the Python CLI from source without installing:

```bash
# from the repository root
python -m venv .venv        # optional: create a virtual environment
# Windows
.\.venv\Scripts\activate
# macOS / Linux
source .venv/bin/activate

pip install -r requirements.txt || true  # optional: install deps if present
python -m copilot_export list
```

You can also use the installed console script after `pip install -e .`:

```bash
copilot-export list
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

## Go CLI

This repository also contains a Go implementation of the CLI located in the `go-cli` directory. The Go CLI provides the same `list` and `export` commands and is distributed as a standalone binary.

Prerequisites:

- Install Go (see https://go.dev). The module specifies `go 1.25.7`; any recent Go 1.25+ installation should work.

Build (Unix / macOS):

```bash
cd go-cli
go build -o copilot-export .
# run locally
./copilot-export list
./copilot-export export 658a --format both
```

Build (Windows PowerShell):

```powershell
cd go-cli
go build -o copilot-export.exe .
.\copilot-export.exe list
.\copilot-export.exe export 658a --format both
```

Install to your GOPATH/GOBIN (optional):

```bash
cd go-cli
go install
# this installs the binary to $GOBIN (or GOPATH/bin)
copilot-export list
```

Notes:

- Both CLIs read the same Copilot session state directory by default (`~/.copilot/session-state/`). Use the `--sessions-dir` flag or set `COPILOT_SESSIONS_DIR` (Python) to point to a custom location.
- The Python package exposes the `copilot-export` console script (defined in the package entry points) and can be run via `python -m copilot_export` without installation.
