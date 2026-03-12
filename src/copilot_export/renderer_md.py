"""Render a ParsedSession to Markdown."""

from __future__ import annotations

import json
from io import StringIO
from typing import Any

from .models import (
    ConversationTurn,
    ParsedSession,
    SubAgentRun,
    ToolCall,
)


def render_markdown(session: ParsedSession) -> str:
    """Render a parsed session as a Markdown document."""
    out = StringIO()
    w = out.write

    _render_header(w, session)
    _render_plan(w, session)
    _render_todos(w, session)
    _render_conversation(w, session)
    _render_checkpoints(w, session)
    _render_statistics(w, session)
    _render_errors(w, session)

    return out.getvalue()


# ---------------------------------------------------------------------------
# Sections
# ---------------------------------------------------------------------------

def _render_header(w, session: ParsedSession) -> None:
    ws = session.workspace
    title = ws.summary or ws.id or "Untitled Session"
    w(f"# Session: {title}\n\n")
    w("## Metadata\n\n")

    rows: list[tuple[str, str]] = []
    if ws.repository:
        rows.append(("Repository", ws.repository))
    rows.append(("Working Directory", ws.cwd))
    if ws.git_root and ws.git_root != ws.cwd:
        rows.append(("Git Root", ws.git_root))
    if session.copilot_version:
        rows.append(("Copilot Version", session.copilot_version))
    if ws.created_at:
        rows.append(("Started", ws.created_at.strftime("%Y-%m-%d %H:%M:%S UTC")))
    if ws.updated_at:
        rows.append(("Last Updated", ws.updated_at.strftime("%Y-%m-%d %H:%M:%S UTC")))
    if ws.id:
        rows.append(("Session ID", f"`{ws.id}`"))

    for label, value in rows:
        w(f"- **{label}**: {value}\n")
    w("\n")


def _render_plan(w, session: ParsedSession) -> None:
    if not session.plan:
        return
    w("## Session Plan\n\n")
    w(session.plan.strip())
    w("\n\n")


def _render_todos(w, session: ParsedSession) -> None:
    if not session.todos:
        return
    w("## Todos\n\n")
    w("| Status | Title | Description |\n")
    w("|--------|-------|-------------|\n")

    status_icons = {
        "done": "✅",
        "in_progress": "🔄",
        "pending": "⏳",
        "blocked": "🚫",
    }

    for todo in session.todos:
        icon = status_icons.get(todo.status, "❓")
        desc = todo.description.replace("\n", " ")
        if len(desc) > 120:
            desc = desc[:117] + "..."
        w(f"| {icon} {todo.status} | {todo.title} | {desc} |\n")
    w("\n")


def _render_conversation(w, session: ParsedSession) -> None:
    if not session.turns:
        return
    w("## Conversation\n\n")

    for turn in session.turns:
        if turn.role == "user":
            _render_user_turn(w, turn)
        else:
            _render_assistant_turn(w, turn)


def _render_user_turn(w, turn: ConversationTurn) -> None:
    ts = _fmt_ts(turn.timestamp)
    w(f"### 👤 User {ts}\n\n")
    w(turn.content.strip())
    w("\n\n")


def _render_assistant_turn(w, turn: ConversationTurn) -> None:
    ts = _fmt_ts(turn.timestamp)
    w(f"### 🤖 Assistant {ts}\n\n")

    # Thinking section (collapsible)
    if turn.thinking:
        w("<details><summary>💭 Thinking</summary>\n\n")
        w(turn.thinking.strip())
        w("\n\n</details>\n\n")

    # Main content
    if turn.content:
        w(turn.content.strip())
        w("\n\n")

    # Tool calls (collapsible)
    for tc in turn.tool_calls:
        _render_tool_call(w, tc)

    # Sub-agents (collapsible)
    for sa in turn.sub_agents:
        _render_subagent(w, sa)


def _render_tool_call(w, tc: ToolCall) -> None:
    name = tc.request.name
    desc = tc.description
    label = f"🔧 {name}"
    if desc:
        label += f" — {desc}"

    w(f"<details><summary>{label}</summary>\n\n")

    # Arguments
    args = tc.request.arguments
    if args:
        _render_tool_arguments(w, name, args)

    # Result
    if tc.result:
        if tc.result.success:
            w("**Result** (success):\n")
        else:
            w("**Result** (failed):\n")

        output = tc.result.detailed_content or tc.result.content
        if output:
            # Determine if the output looks like code
            if "\n" in output or len(output) > 200:
                w("```\n")
                w(output.strip())
                w("\n```\n")
            else:
                w(f"`{output.strip()}`\n")
    w("\n</details>\n\n")


def _render_tool_arguments(w, tool_name: str, args: dict[str, Any]) -> None:
    """Render tool arguments in a readable way based on tool type."""
    if tool_name == "powershell":
        cmd = args.get("command", "")
        if cmd:
            w(f"**Command**: `{cmd}`\n\n")
        mode = args.get("mode")
        if mode:
            w(f"**Mode**: {mode}\n\n")
    elif tool_name in ("view", "create", "edit"):
        path = args.get("path", "")
        if path:
            w(f"**Path**: `{path}`\n\n")
        if tool_name == "edit":
            old = args.get("old_str", "")
            new = args.get("new_str", "")
            if old:
                w("**Old**:\n```\n")
                w(old)
                w("\n```\n")
            if new:
                w("**New**:\n```\n")
                w(new)
                w("\n```\n")
        elif tool_name == "create":
            ft = args.get("file_text", "")
            if ft:
                w("**Content**:\n```\n")
                # Truncate very large file contents
                if len(ft) > 2000:
                    w(ft[:2000])
                    w(f"\n... ({len(ft)} chars total)")
                else:
                    w(ft)
                w("\n```\n")
    elif tool_name in ("grep", "glob"):
        pattern = args.get("pattern", "")
        if pattern:
            w(f"**Pattern**: `{pattern}`\n\n")
        path = args.get("path", "")
        if path:
            w(f"**Path**: `{path}`\n\n")
    elif tool_name == "sql":
        query = args.get("query", "")
        if query:
            w("**Query**:\n```sql\n")
            w(query)
            w("\n```\n")
    elif tool_name == "web_fetch":
        url = args.get("url", "")
        if url:
            w(f"**URL**: {url}\n\n")
    elif tool_name == "task":
        prompt = args.get("prompt", "")
        agent_type = args.get("agent_type", "")
        if agent_type:
            w(f"**Agent**: {agent_type}\n\n")
        if prompt:
            w(f"**Prompt**: {prompt[:500]}")
            if len(prompt) > 500:
                w(f"... ({len(prompt)} chars)")
            w("\n\n")
    else:
        # Generic: show as JSON
        args_str = json.dumps(args, indent=2, ensure_ascii=False)
        if len(args_str) > 1000:
            args_str = args_str[:1000] + "\n... (truncated)"
        w(f"**Arguments**:\n```json\n{args_str}\n```\n")


def _render_subagent(w, sa: SubAgentRun) -> None:
    status = "✅" if sa.success else "❌"
    w(f"<details><summary>🔍 Sub-agent: {sa.display_name or sa.agent_name} {status}</summary>\n\n")
    if sa.description:
        w(f"_{sa.description.strip()}_\n\n")
    if sa.error:
        w(f"**Error**: {sa.error}\n\n")
    w("</details>\n\n")


def _render_checkpoints(w, session: ParsedSession) -> None:
    if not session.checkpoints:
        return
    w("## Checkpoints\n\n")
    for cp in session.checkpoints:
        w(f"### Checkpoint {cp.index}: {cp.title}\n\n")
        if cp.content:
            w("<details><summary>View checkpoint content</summary>\n\n")
            w(cp.content.strip())
            w("\n\n</details>\n\n")


def _render_statistics(w, session: ParsedSession) -> None:
    stats = session.shutdown_stats
    if not stats:
        return
    w("## Session Statistics\n\n")

    total_requests = stats.get("totalPremiumRequests")
    if total_requests is not None:
        w(f"- **Premium Requests**: {total_requests}\n")

    api_ms = stats.get("totalApiDurationMs")
    if api_ms is not None:
        seconds = api_ms / 1000
        minutes = seconds / 60
        if minutes >= 1:
            w(f"- **Total API Duration**: {minutes:.1f} minutes\n")
        else:
            w(f"- **Total API Duration**: {seconds:.1f} seconds\n")

    changes = stats.get("codeChanges", {})
    if changes:
        added = changes.get("linesAdded", 0)
        removed = changes.get("linesRemoved", 0)
        files = changes.get("filesModified", [])
        w(f"- **Code Changes**: +{added} / -{removed} lines across {len(files)} files\n")

    shutdown_type = stats.get("shutdownType")
    if shutdown_type:
        w(f"- **Shutdown**: {shutdown_type}\n")

    w("\n")


def _render_errors(w, session: ParsedSession) -> None:
    if not session.errors:
        return
    w("## Session Errors\n\n")
    for err in session.errors:
        err_type = err.get("errorType", "unknown")
        msg = err.get("message", "")
        w(f"- ⚠️ **{err_type}**: {msg}\n")
    w("\n")


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def _fmt_ts(ts) -> str:
    if ts is None:
        return ""
    return f"({ts.strftime('%H:%M:%S')})"
