"""Parse a Copilot CLI session directory into a ParsedSession."""

from __future__ import annotations

import json
import re
import sqlite3
from pathlib import Path
from typing import Any

import yaml

from .models import (
    Checkpoint,
    ConversationTurn,
    Event,
    EventType,
    ParsedSession,
    SubAgentRun,
    Todo,
    TodoDep,
    ToolCall,
    ToolRequest,
    ToolResult,
    WorkspaceMetadata,
)

# Regex to detect thinking blocks in assistant content
_THINKING_RE = re.compile(
    r"<(?:antml:)?thinking(?:_mode)?[^>]*>(.*?)</(?:antml:)?thinking(?:_mode)?>",
    re.DOTALL,
)


def parse_session(session_dir: str | Path) -> ParsedSession:
    """Parse all files in a session directory and return a structured ParsedSession."""
    root = Path(session_dir)
    if not root.is_dir():
        raise FileNotFoundError(f"Session directory not found: {root}")

    workspace = _parse_workspace(root / "workspace.yaml")
    events = _parse_events(root / "events.jsonl")
    todos, todo_deps = _parse_db(root / "session.db")
    plan = _read_text(root / "plan.md")
    checkpoints = _parse_checkpoints(root / "checkpoints")

    copilot_version = ""
    shutdown_stats: dict[str, Any] = {}
    errors: list[dict[str, Any]] = []

    for ev in events:
        if ev.type == EventType.SESSION_START:
            copilot_version = ev.data.get("copilotVersion", "")
        elif ev.type == EventType.SESSION_SHUTDOWN:
            shutdown_stats = ev.data
        elif ev.type == EventType.SESSION_ERROR:
            errors.append(ev.data)

    turns = _build_turns(events)

    return ParsedSession(
        workspace=workspace,
        events=events,
        turns=turns,
        todos=todos,
        todo_deps=todo_deps,
        checkpoints=checkpoints,
        plan=plan,
        copilot_version=copilot_version,
        shutdown_stats=shutdown_stats,
        errors=errors,
        session_dir=root,
    )


# ---------------------------------------------------------------------------
# File parsers
# ---------------------------------------------------------------------------

def _parse_workspace(path: Path) -> WorkspaceMetadata:
    if not path.exists():
        return WorkspaceMetadata(id="", cwd="")
    with open(path, "r", encoding="utf-8") as f:
        data = yaml.safe_load(f) or {}
    return WorkspaceMetadata.from_dict(data)


def _parse_events(path: Path) -> list[Event]:
    if not path.exists():
        return []
    events: list[Event] = []
    with open(path, "r", encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            try:
                obj = json.loads(line)
                events.append(Event.from_dict(obj))
            except json.JSONDecodeError:
                continue
    return events


def _parse_db(path: Path) -> tuple[list[Todo], list[TodoDep]]:
    if not path.exists():
        return [], []
    todos: list[Todo] = []
    deps: list[TodoDep] = []
    try:
        conn = sqlite3.connect(str(path))
        cursor = conn.cursor()

        # Check if tables exist
        cursor.execute(
            "SELECT name FROM sqlite_master WHERE type='table' AND name='todos'"
        )
        if cursor.fetchone():
            cursor.execute("SELECT id, title, description, status FROM todos")
            for row in cursor.fetchall():
                todos.append(Todo(id=row[0], title=row[1], description=row[2] or "", status=row[3] or "pending"))

        cursor.execute(
            "SELECT name FROM sqlite_master WHERE type='table' AND name='todo_deps'"
        )
        if cursor.fetchone():
            cursor.execute("SELECT todo_id, depends_on FROM todo_deps")
            for row in cursor.fetchall():
                deps.append(TodoDep(todo_id=row[0], depends_on=row[1]))

        conn.close()
    except sqlite3.Error:
        pass
    return todos, deps


def _parse_checkpoints(cp_dir: Path) -> list[Checkpoint]:
    if not cp_dir.is_dir():
        return []
    index_path = cp_dir / "index.md"
    checkpoints: list[Checkpoint] = []
    if index_path.exists():
        # Parse the index table to get checkpoint order and titles
        content = index_path.read_text(encoding="utf-8")
        # Matches table rows like: | 1 | Title | filename.md |
        for match in re.finditer(
            r"\|\s*(\d+)\s*\|\s*(.+?)\s*\|\s*(.+?)\s*\|", content
        ):
            idx = int(match.group(1))
            title = match.group(2).strip()
            filename = match.group(3).strip()
            cp_path = cp_dir / filename
            cp_content = ""
            if cp_path.exists():
                cp_content = cp_path.read_text(encoding="utf-8")
            checkpoints.append(
                Checkpoint(index=idx, title=title, filename=filename, content=cp_content)
            )
    else:
        # Fallback: read all .md files in order
        for i, md_file in enumerate(sorted(cp_dir.glob("*.md")), 1):
            if md_file.name == "index.md":
                continue
            checkpoints.append(
                Checkpoint(
                    index=i,
                    title=md_file.stem,
                    filename=md_file.name,
                    content=md_file.read_text(encoding="utf-8"),
                )
            )
    return checkpoints


def _read_text(path: Path) -> str:
    if not path.exists():
        return ""
    return path.read_text(encoding="utf-8")


# ---------------------------------------------------------------------------
# Build conversation turns from events
# ---------------------------------------------------------------------------

def _build_turns(events: list[Event]) -> list[ConversationTurn]:
    """Group events into a chronological list of conversation turns."""

    # Index tool execution events by toolCallId for quick lookup
    tool_starts: dict[str, Event] = {}
    tool_completes: dict[str, Event] = {}
    subagent_starts: dict[str, Event] = {}
    subagent_ends: dict[str, Event] = {}  # completed or failed

    for ev in events:
        if ev.type == EventType.TOOL_EXECUTION_START:
            tool_starts[ev.data.get("toolCallId", "")] = ev
        elif ev.type == EventType.TOOL_EXECUTION_COMPLETE:
            tool_completes[ev.data.get("toolCallId", "")] = ev
        elif ev.type == EventType.SUBAGENT_STARTED:
            subagent_starts[ev.data.get("toolCallId", "")] = ev
        elif ev.type in (EventType.SUBAGENT_COMPLETED, EventType.SUBAGENT_FAILED):
            subagent_ends[ev.data.get("toolCallId", "")] = ev

    turns: list[ConversationTurn] = []
    current_mode: str | None = None
    current_assistant_turn: ConversationTurn | None = None

    for ev in events:
        if ev.type == EventType.SESSION_MODE_CHANGED:
            current_mode = ev.data.get("newMode", current_mode)

        elif ev.type == EventType.USER_MESSAGE:
            turns.append(
                ConversationTurn(
                    role="user",
                    timestamp=ev.timestamp,
                    content=ev.data.get("content", ""),
                    mode=current_mode,
                )
            )

        elif ev.type == EventType.ASSISTANT_TURN_START:
            current_assistant_turn = ConversationTurn(
                role="assistant",
                timestamp=ev.timestamp,
                mode=current_mode,
            )

        elif ev.type == EventType.ASSISTANT_MESSAGE:
            if current_assistant_turn is None:
                current_assistant_turn = ConversationTurn(
                    role="assistant", timestamp=ev.timestamp, mode=current_mode
                )

            raw_content = ev.data.get("content", "")

            # Extract thinking sections
            thinking_parts = _THINKING_RE.findall(raw_content)
            if thinking_parts:
                current_assistant_turn.thinking += "\n".join(thinking_parts)
                # Remove thinking from displayed content
                clean_content = _THINKING_RE.sub("", raw_content).strip()
            else:
                clean_content = raw_content

            if clean_content:
                if current_assistant_turn.content:
                    current_assistant_turn.content += "\n\n" + clean_content
                else:
                    current_assistant_turn.content = clean_content

            # Process tool requests
            for tr_data in ev.data.get("toolRequests", []):
                tr = ToolRequest.from_dict(tr_data)

                # Skip internal-only tools
                if tr.name in ("report_intent",):
                    continue

                tc_id = tr.tool_call_id
                result = None
                if tc_id in tool_starts and tc_id in tool_completes:
                    result = ToolResult.from_execution_events(
                        tool_starts[tc_id], tool_completes[tc_id]
                    )

                description = tr.arguments.get("description", "")

                # Check if this is a sub-agent call
                if tc_id in subagent_starts:
                    sa_start = subagent_starts[tc_id]
                    sa_end = subagent_ends.get(tc_id)
                    sa = SubAgentRun(
                        tool_call_id=tc_id,
                        agent_name=sa_start.data.get("agentName", ""),
                        display_name=sa_start.data.get("agentDisplayName", ""),
                        description=sa_start.data.get("agentDescription", ""),
                        success=sa_end.type == EventType.SUBAGENT_COMPLETED if sa_end else True,
                        error=sa_end.data.get("error", "") if sa_end and sa_end.type == EventType.SUBAGENT_FAILED else "",
                    )
                    current_assistant_turn.sub_agents.append(sa)
                else:
                    current_assistant_turn.tool_calls.append(
                        ToolCall(request=tr, result=result, description=description)
                    )

        elif ev.type == EventType.ASSISTANT_TURN_END:
            if current_assistant_turn is not None:
                turns.append(current_assistant_turn)
                current_assistant_turn = None

    # Flush any unfinished assistant turn
    if current_assistant_turn is not None:
        turns.append(current_assistant_turn)

    return turns
