"""Data models for Copilot CLI session events and metadata."""

from __future__ import annotations

from dataclasses import dataclass, field
from datetime import datetime
from enum import Enum
from pathlib import Path
from typing import Any


# ---------------------------------------------------------------------------
# workspace.yaml
# ---------------------------------------------------------------------------

@dataclass
class WorkspaceMetadata:
    id: str
    cwd: str
    git_root: str | None = None
    repository: str | None = None
    summary: str | None = None
    summary_count: int = 0
    created_at: datetime | None = None
    updated_at: datetime | None = None

    @classmethod
    def from_dict(cls, d: dict[str, Any]) -> WorkspaceMetadata:
        return cls(
            id=d.get("id", ""),
            cwd=d.get("cwd", ""),
            git_root=d.get("git_root"),
            repository=d.get("repository"),
            summary=d.get("summary"),
            summary_count=d.get("summary_count", 0),
            created_at=_parse_ts(d.get("created_at")),
            updated_at=_parse_ts(d.get("updated_at")),
        )


# ---------------------------------------------------------------------------
# Events
# ---------------------------------------------------------------------------

class EventType(str, Enum):
    SESSION_START = "session.start"
    SESSION_MODE_CHANGED = "session.mode_changed"
    SESSION_PLAN_CHANGED = "session.plan_changed"
    SESSION_COMPACTION_START = "session.compaction_start"
    SESSION_COMPACTION_COMPLETE = "session.compaction_complete"
    SESSION_ERROR = "session.error"
    SESSION_SHUTDOWN = "session.shutdown"
    SESSION_TASK_COMPLETE = "session.task_complete"
    USER_MESSAGE = "user.message"
    ASSISTANT_TURN_START = "assistant.turn_start"
    ASSISTANT_MESSAGE = "assistant.message"
    ASSISTANT_TURN_END = "assistant.turn_end"
    TOOL_EXECUTION_START = "tool.execution_start"
    TOOL_EXECUTION_COMPLETE = "tool.execution_complete"
    SUBAGENT_STARTED = "subagent.started"
    SUBAGENT_COMPLETED = "subagent.completed"
    SUBAGENT_FAILED = "subagent.failed"
    ABORT = "abort"


@dataclass
class Event:
    type: str
    data: dict[str, Any]
    id: str
    timestamp: datetime | None
    parent_id: str | None

    @classmethod
    def from_dict(cls, d: dict[str, Any]) -> Event:
        return cls(
            type=d.get("type", ""),
            data=d.get("data", {}),
            id=d.get("id", ""),
            timestamp=_parse_ts(d.get("timestamp")),
            parent_id=d.get("parentId"),
        )


# ---------------------------------------------------------------------------
# Tool calls & results (extracted from assistant.message / tool.execution_*)
# ---------------------------------------------------------------------------

@dataclass
class ToolRequest:
    tool_call_id: str
    name: str
    arguments: dict[str, Any] = field(default_factory=dict)

    @classmethod
    def from_dict(cls, d: dict[str, Any]) -> ToolRequest:
        return cls(
            tool_call_id=d.get("toolCallId", ""),
            name=d.get("name", ""),
            arguments=d.get("arguments", {}),
        )


@dataclass
class ToolResult:
    tool_call_id: str
    tool_name: str
    success: bool
    content: str = ""
    detailed_content: str = ""

    @classmethod
    def from_execution_events(
        cls, start: Event, complete: Event
    ) -> ToolResult:
        result = complete.data.get("result", {})
        return cls(
            tool_call_id=start.data.get("toolCallId", ""),
            tool_name=start.data.get("toolName", ""),
            success=complete.data.get("success", False),
            content=result.get("content", "") if isinstance(result, dict) else str(result),
            detailed_content=result.get("detailedContent", "") if isinstance(result, dict) else "",
        )


# ---------------------------------------------------------------------------
# Sub-agent
# ---------------------------------------------------------------------------

@dataclass
class SubAgentRun:
    tool_call_id: str
    agent_name: str
    display_name: str
    description: str = ""
    success: bool = True
    error: str = ""


# ---------------------------------------------------------------------------
# Conversation turn (high-level grouping)
# ---------------------------------------------------------------------------

@dataclass
class ConversationTurn:
    """A single turn in the conversation — either user or assistant."""
    role: str  # "user" or "assistant"
    timestamp: datetime | None = None
    content: str = ""
    thinking: str = ""
    tool_calls: list[ToolCall] = field(default_factory=list)
    sub_agents: list[SubAgentRun] = field(default_factory=list)
    mode: str | None = None  # current session mode when turn occurred


@dataclass
class ToolCall:
    """A tool invocation with its request and result."""
    request: ToolRequest
    result: ToolResult | None = None
    description: str = ""  # from tool arguments


# ---------------------------------------------------------------------------
# Todos (from session.db)
# ---------------------------------------------------------------------------

@dataclass
class Todo:
    id: str
    title: str
    description: str = ""
    status: str = "pending"


@dataclass
class TodoDep:
    todo_id: str
    depends_on: str


# ---------------------------------------------------------------------------
# Checkpoint
# ---------------------------------------------------------------------------

@dataclass
class Checkpoint:
    index: int
    title: str
    filename: str
    content: str = ""


# ---------------------------------------------------------------------------
# Parsed session (top-level container)
# ---------------------------------------------------------------------------

@dataclass
class ParsedSession:
    workspace: WorkspaceMetadata
    events: list[Event] = field(default_factory=list)
    turns: list[ConversationTurn] = field(default_factory=list)
    todos: list[Todo] = field(default_factory=list)
    todo_deps: list[TodoDep] = field(default_factory=list)
    checkpoints: list[Checkpoint] = field(default_factory=list)
    plan: str = ""
    copilot_version: str = ""
    shutdown_stats: dict[str, Any] = field(default_factory=dict)
    errors: list[dict[str, Any]] = field(default_factory=list)
    session_dir: Path | None = None


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def _parse_ts(value: Any) -> datetime | None:
    if value is None:
        return None
    if isinstance(value, datetime):
        return value
    try:
        s = str(value).rstrip("Z")
        return datetime.fromisoformat(s)
    except (ValueError, TypeError):
        return None
