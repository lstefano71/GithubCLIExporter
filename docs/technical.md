# Technical Notes

This document covers formats, rendering choices and implementation notes for the Go CLI exporter.

Exported formats

- Markdown (`.md`): single self-contained file with conversation view, collapsible tool calls and thinking sections.
- HTML (`.html`): single-file output with inlined CSS and optional dark/light theme toggle, table of contents, and syntax highlighting.

Session discovery

Sessions are stored under `~/.copilot/session-state/<uuid>/`. The CLI implements:

- scanning and summarizing sessions
- filtering by repository, date, and summary text
- selection by index number, partial UUID prefix, full UUID, or directory path
- interactive picker when no specifier is provided

Rendering

- Conversation is rendered chronologically; long thinking or tool outputs are wrapped in collapsible sections.
- Metadata (workspace, plan, todos, checkpoints) appears in dedicated sections.

Implementation

- Core CLI: `go-cli/main.go` — implements `list` and `export` commands; flags mirror the UX in the cheat sheet.
- Parsing and rendering: `parser.go`, `renderer_md.go`, `renderer_html.go` (in repo root /go-cli or top-level files depending on build target).

Notes on previous prototypes

An early proof-of-concept implementation existed during initial design; the maintained implementation is the Go CLI in `go-cli/`.

---

## Copilot CLI Session Format — Deep Dive

### Storage location

Sessions live under `~/.copilot/session-state/<uuid>/` where `<uuid>` is an opaque GUID. Multiple sessions may exist simultaneously; the directory name alone carries no human-readable information.

### Directory structure

```
<uuid>/
├── workspace.yaml          # Session metadata (see below)
├── events.jsonl            # Main event stream — one JSON object per line
├── session.db              # SQLite: todos and todo_deps tables
├── plan.md                 # Agent plan (present only when plan mode was used)
├── vscode.metadata.json    # VS Code metadata (typically empty {})
├── checkpoints/
│   ├── index.md            # TOC of all checkpoints
│   └── <slug>.md           # One file per checkpoint
├── research/               # Research artifacts (may be empty)
├── files/                  # Persistent session files
└── rewind-snapshots/
    ├── index.json          # Snapshot history — maps to git commits
    └── backups/            # File backups keyed by hash-timestamp
```

### `workspace.yaml`

Top-level session metadata. Example shape:

```yaml
id: "658af01e-f952-45f1-bc79-5d896d7c456b"
cwd: "D:/_Utenti/stf.APLITA/Source/Repos/MyProject"
git_root: "D:/_Utenti/stf.APLITA/Source/Repos/MyProject"
repository: "owner/MyProject"
summary: "Implement feature X and fix related tests"
summary_count: 3
created_at: "2026-03-12T14:00:00Z"
updated_at: "2026-03-12T16:22:00Z"
```

### `events.jsonl`

The main data source. Every line is a standalone JSON object with this outer envelope:

```json
{
  "type":      "<event-type>",
  "data":      { /* event-specific payload */ },
  "id":        "<uuid>",
  "timestamp": "2026-03-12T14:01:23.456Z",
  "parentId":  "<uuid> | null"
}
```

Events reference each other via `parentId`, forming a tree. A typical medium session has 1 000+ events and weighs ~3 MB.

#### Event types

**Session lifecycle**

| Type | Key `data` fields |
|---|---|
| `session.start` | `sessionId`, `version`, `producer`, `copilotVersion`, `startTime`, `context` (`cwd`, `gitRoot`, `repository`), `alreadyInUse` |
| `session.mode_changed` | `previousMode`, `newMode` — values: `interactive`, `autopilot`, `plan` |
| `session.plan_changed` | `operation` (e.g. `create`) |
| `session.compaction_start` | _(empty)_ |
| `session.compaction_complete` | `success`, `preCompactionTokens`, `preCompactionMessagesLength`, `summaryContent` |
| `session.error` | `errorType`, `message` |
| `session.shutdown` | `shutdownType`, `totalPremiumRequests`, `totalApiDurationMs`, `sessionStartTime`, `codeChanges` (`linesAdded`, `linesRemoved`, `filesModified`) |
| `session.task_complete` | `summary` |

**Conversation**

| Type | Key `data` fields |
|---|---|
| `user.message` | `content` (raw text), `transformedContent` (system-wrapped), `source`, `attachments`, `agentMode`, `interactionId` |
| `assistant.turn_start` | `turnId`, `interactionId` |
| `assistant.message` | `messageId`, `content` (text), `toolRequests` (array of `{toolCallId, name, arguments}`) |
| `assistant.turn_end` | `turnId` |

An `assistant.message` may carry text `content`, `toolRequests`, or both. In a typical session roughly half the assistant messages include tool requests.

The `content` field may contain thinking/reasoning sections wrapped in `<thinking>…</thinking>` tags — the exporter folds these into collapsible blocks.

**Tool execution**

| Type | Key `data` fields |
|---|---|
| `tool.execution_start` | `toolCallId`, `toolName`, `arguments` |
| `tool.execution_complete` | `toolCallId`, `success`, `result` (`content`, `detailedContent`), `model`, `toolTelemetry` |

The `result.content` field holds a plain-text summary; `result.detailedContent` typically contains a diff or verbose output. The exporter uses `detailedContent` when available.

**Sub-agents**

| Type | Key `data` fields |
|---|---|
| `subagent.started` | `toolCallId`, `agentName`, `agentDisplayName`, `agentDescription` |
| `subagent.completed` | `toolCallId`, `agentName`, `agentDisplayName` |
| `subagent.failed` | `toolCallId`, `agentName`, `agentDisplayName`, `error` |

Sub-agent events are nested under the `tool.execution_start`/`tool.execution_complete` pair for the `task` tool via `parentId`.

**Control**

| Type | Key `data` fields |
|---|---|
| `abort` | `reason` (e.g. `user initiated`) |

#### Tool names observed in the wild

`create`, `edit`, `exit_plan_mode`, `github-mcp-server-get_commit`, `glob`, `grep`, `list_powershell`, `powershell`, `read_powershell`, `report_intent`, `sql`, `task`, `task_complete`, `view`, `web_fetch`

### `session.db` — SQLite

Two tables:

**`todos`**

| Column | Type | Notes |
|---|---|---|
| `id` | TEXT | Unique slug (e.g. `"fix-parser"`) |
| `title` | TEXT | Short display title |
| `description` | TEXT | Full description |
| `status` | TEXT | `pending`, `in_progress`, `done` |
| `created_at` | TEXT | ISO 8601 |
| `updated_at` | TEXT | ISO 8601 |

**`todo_deps`**

| Column | Type | Notes |
|---|---|---|
| `todo_id` | TEXT | References `todos.id` |
| `depends_on` | TEXT | References `todos.id` |

### `checkpoints/`

Each checkpoint is a standalone Markdown file. `index.md` contains a table of contents. Checkpoint filenames follow a slug derived from the checkpoint summary. The exporter appends all checkpoint content as an appendix section.

### `rewind-snapshots/index.json`

Maps snapshot identifiers to git commit hashes and lists backed-up file paths. The `backups/` directory contains raw file content keyed by `<hash>-<timestamp>`. These are used for project state recovery and are not exported by the CLI.
