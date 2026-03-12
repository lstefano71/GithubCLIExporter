"""Render a ParsedSession to a self-contained HTML file."""

from __future__ import annotations

import html
import json
import re
from io import StringIO
from typing import Any

from .models import (
    ConversationTurn,
    ParsedSession,
    SubAgentRun,
    ToolCall,
)

# ---------------------------------------------------------------------------
# CSS
# ---------------------------------------------------------------------------

_CSS = """
:root {
  --bg: #ffffff; --fg: #1a1a2e; --bg2: #f5f5f7; --border: #d1d5db;
  --accent: #2563eb; --accent2: #7c3aed; --success: #16a34a; --error: #dc2626;
  --code-bg: #f1f5f9; --code-fg: #334155; --user-bg: #eff6ff; --assistant-bg: #f9fafb;
  --summary-bg: #f8fafc; --shadow: rgba(0,0,0,0.05);
}
[data-theme="dark"] {
  --bg: #0f172a; --fg: #e2e8f0; --bg2: #1e293b; --border: #334155;
  --accent: #60a5fa; --accent2: #a78bfa; --success: #4ade80; --error: #f87171;
  --code-bg: #1e293b; --code-fg: #e2e8f0; --user-bg: #1e293b; --assistant-bg: #0f172a;
  --summary-bg: #1e293b; --shadow: rgba(0,0,0,0.3);
}
*, *::before, *::after { box-sizing: border-box; }
body {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  background: var(--bg); color: var(--fg); margin: 0; padding: 0;
  line-height: 1.6; font-size: 15px;
}
.container { max-width: 960px; margin: 0 auto; padding: 2rem 1.5rem; }
h1 { font-size: 1.8rem; border-bottom: 2px solid var(--accent); padding-bottom: 0.5rem; margin-top: 0; }
h2 { font-size: 1.4rem; color: var(--accent); margin-top: 2.5rem; border-bottom: 1px solid var(--border); padding-bottom: 0.3rem; }
h3 { font-size: 1.1rem; margin-top: 1.5rem; }
a { color: var(--accent); }
.meta-list { list-style: none; padding: 0; }
.meta-list li { padding: 0.2rem 0; }
.meta-list strong { min-width: 140px; display: inline-block; }
pre, code { font-family: 'Cascadia Code', 'Fira Code', 'JetBrains Mono', monospace; font-size: 0.9em; }
code { background: var(--code-bg); color: var(--code-fg); padding: 0.15em 0.35em; border-radius: 4px; }
pre { background: var(--code-bg); color: var(--code-fg); padding: 1rem; border-radius: 8px; overflow-x: auto; border: 1px solid var(--border); }
pre code { background: none; padding: 0; }
table { border-collapse: collapse; width: 100%; margin: 1rem 0; }
th, td { border: 1px solid var(--border); padding: 0.5rem 0.75rem; text-align: left; }
th { background: var(--bg2); font-weight: 600; }
.turn { margin: 1.5rem 0; padding: 1rem 1.25rem; border-radius: 10px; border: 1px solid var(--border); }
.turn-user { background: var(--user-bg); border-left: 4px solid var(--accent); }
.turn-assistant { background: var(--assistant-bg); border-left: 4px solid var(--accent2); }
.turn-header { font-weight: 600; font-size: 0.95rem; margin-bottom: 0.5rem; opacity: 0.8; }
.turn-content { white-space: pre-wrap; word-wrap: break-word; }
details { margin: 0.75rem 0; border: 1px solid var(--border); border-radius: 8px; overflow: hidden; }
details > summary {
  padding: 0.6rem 1rem; background: var(--summary-bg); cursor: pointer;
  font-weight: 500; user-select: none; list-style: none;
}
details > summary::-webkit-details-marker { display: none; }
details > summary::before { content: '▶ '; font-size: 0.75em; transition: transform 0.2s; display: inline-block; }
details[open] > summary::before { transform: rotate(90deg); }
details > .detail-content { padding: 0.75rem 1rem; }
.tool-label { color: var(--accent2); }
.thinking-label { color: #b45309; }
.subagent-label { color: var(--accent); }
.success { color: var(--success); }
.error { color: var(--error); }
.stat-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 1rem; margin: 1rem 0; }
.stat-card {
  background: var(--bg2); border: 1px solid var(--border); border-radius: 8px;
  padding: 1rem; text-align: center;
}
.stat-card .stat-value { font-size: 1.5rem; font-weight: 700; color: var(--accent); }
.stat-card .stat-label { font-size: 0.85rem; opacity: 0.7; }
.theme-toggle {
  position: fixed; top: 1rem; right: 1rem; z-index: 100;
  background: var(--bg2); border: 1px solid var(--border); border-radius: 8px;
  padding: 0.5rem 0.75rem; cursor: pointer; font-size: 1.2rem;
}
.toc { background: var(--bg2); border: 1px solid var(--border); border-radius: 8px; padding: 1rem 1.5rem; margin: 1rem 0; }
.toc ul { list-style: none; padding-left: 1rem; }
.toc > ul { padding-left: 0; }
.toc a { text-decoration: none; }
.toc a:hover { text-decoration: underline; }
@media (max-width: 640px) {
  .container { padding: 1rem; }
  h1 { font-size: 1.4rem; }
  pre { font-size: 0.8em; }
}
"""

_JS = """
(function() {
  const btn = document.getElementById('theme-toggle');
  const html = document.documentElement;
  const saved = localStorage.getItem('theme');
  if (saved) html.setAttribute('data-theme', saved);
  btn.addEventListener('click', function() {
    const current = html.getAttribute('data-theme');
    const next = current === 'dark' ? 'light' : 'dark';
    html.setAttribute('data-theme', next);
    localStorage.setItem('theme', next);
    btn.textContent = next === 'dark' ? '☀️' : '🌙';
  });
})();
"""


def render_html(session: ParsedSession) -> str:
    """Render a parsed session as a self-contained HTML document."""
    out = StringIO()
    w = out.write

    title = html.escape(session.workspace.summary or session.workspace.id or "Session Export")

    w("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n")
    w(f"<meta charset=\"utf-8\">\n<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n")
    w(f"<title>{title}</title>\n")
    w(f"<style>{_CSS}</style>\n")
    w("</head>\n<body>\n")
    w("<button id=\"theme-toggle\" class=\"theme-toggle\">🌙</button>\n")
    w("<div class=\"container\">\n")

    _html_header(w, session)
    _html_toc(w, session)
    _html_plan(w, session)
    _html_todos(w, session)
    _html_conversation(w, session)
    _html_checkpoints(w, session)
    _html_statistics(w, session)
    _html_errors(w, session)

    w("</div>\n")
    w(f"<script>{_JS}</script>\n")
    w("</body>\n</html>")

    return out.getvalue()


# ---------------------------------------------------------------------------
# Sections
# ---------------------------------------------------------------------------

def _html_header(w, session: ParsedSession) -> None:
    ws = session.workspace
    title = html.escape(ws.summary or ws.id or "Untitled Session")
    w(f"<h1 id=\"top\">{title}</h1>\n")
    w("<h2 id=\"metadata\">Metadata</h2>\n<ul class=\"meta-list\">\n")

    if ws.repository:
        w(f"<li><strong>Repository:</strong> {_esc(ws.repository)}</li>\n")
    w(f"<li><strong>Working Directory:</strong> {_esc(ws.cwd)}</li>\n")
    if ws.git_root and ws.git_root != ws.cwd:
        w(f"<li><strong>Git Root:</strong> {_esc(ws.git_root)}</li>\n")
    if session.copilot_version:
        w(f"<li><strong>Copilot Version:</strong> {_esc(session.copilot_version)}</li>\n")
    if ws.created_at:
        w(f"<li><strong>Started:</strong> {ws.created_at.strftime('%Y-%m-%d %H:%M:%S UTC')}</li>\n")
    if ws.updated_at:
        w(f"<li><strong>Last Updated:</strong> {ws.updated_at.strftime('%Y-%m-%d %H:%M:%S UTC')}</li>\n")
    if ws.id:
        w(f"<li><strong>Session ID:</strong> <code>{_esc(ws.id)}</code></li>\n")
    w("</ul>\n")


def _html_toc(w, session: ParsedSession) -> None:
    w("<div class=\"toc\">\n<strong>Table of Contents</strong>\n<ul>\n")
    w("<li><a href=\"#metadata\">Metadata</a></li>\n")
    if session.plan:
        w("<li><a href=\"#plan\">Session Plan</a></li>\n")
    if session.todos:
        w("<li><a href=\"#todos\">Todos</a></li>\n")
    if session.turns:
        w("<li><a href=\"#conversation\">Conversation</a></li>\n")
    if session.checkpoints:
        w("<li><a href=\"#checkpoints\">Checkpoints</a></li>\n")
    if session.shutdown_stats:
        w("<li><a href=\"#statistics\">Session Statistics</a></li>\n")
    if session.errors:
        w("<li><a href=\"#errors\">Session Errors</a></li>\n")
    w("</ul>\n</div>\n")


def _html_plan(w, session: ParsedSession) -> None:
    if not session.plan:
        return
    w("<h2 id=\"plan\">Session Plan</h2>\n")
    w(f"<pre><code>{_esc(session.plan.strip())}</code></pre>\n")


def _html_todos(w, session: ParsedSession) -> None:
    if not session.todos:
        return
    w("<h2 id=\"todos\">Todos</h2>\n")
    w("<table>\n<thead><tr><th>Status</th><th>Title</th><th>Description</th></tr></thead>\n<tbody>\n")
    icons = {"done": "✅", "in_progress": "🔄", "pending": "⏳", "blocked": "🚫"}
    for todo in session.todos:
        icon = icons.get(todo.status, "❓")
        desc = todo.description.replace("\n", " ")
        if len(desc) > 150:
            desc = desc[:147] + "..."
        w(f"<tr><td>{icon} {_esc(todo.status)}</td><td>{_esc(todo.title)}</td><td>{_esc(desc)}</td></tr>\n")
    w("</tbody>\n</table>\n")


def _html_conversation(w, session: ParsedSession) -> None:
    if not session.turns:
        return
    w("<h2 id=\"conversation\">Conversation</h2>\n")
    for i, turn in enumerate(session.turns):
        if turn.role == "user":
            _html_user_turn(w, turn, i)
        else:
            _html_assistant_turn(w, turn, i)


def _html_user_turn(w, turn: ConversationTurn, idx: int) -> None:
    ts = _fmt_ts(turn.timestamp)
    w(f"<div class=\"turn turn-user\" id=\"turn-{idx}\">\n")
    w(f"<div class=\"turn-header\">👤 User {ts}</div>\n")
    w(f"<div class=\"turn-content\">{_esc(turn.content.strip())}</div>\n")
    w("</div>\n")


def _html_assistant_turn(w, turn: ConversationTurn, idx: int) -> None:
    ts = _fmt_ts(turn.timestamp)
    w(f"<div class=\"turn turn-assistant\" id=\"turn-{idx}\">\n")
    w(f"<div class=\"turn-header\">🤖 Assistant {ts}</div>\n")

    if turn.thinking:
        w("<details>\n<summary><span class=\"thinking-label\">💭 Thinking</span></summary>\n")
        w(f"<div class=\"detail-content\"><pre>{_esc(turn.thinking.strip())}</pre></div>\n")
        w("</details>\n")

    if turn.content:
        w(f"<div class=\"turn-content\">{_esc(turn.content.strip())}</div>\n")

    for tc in turn.tool_calls:
        _html_tool_call(w, tc)

    for sa in turn.sub_agents:
        _html_subagent(w, sa)

    w("</div>\n")


def _html_tool_call(w, tc: ToolCall) -> None:
    name = _esc(tc.request.name)
    desc = _esc(tc.description) if tc.description else ""
    label = f"<span class=\"tool-label\">🔧 {name}</span>"
    if desc:
        label += f" — {desc}"

    w(f"<details>\n<summary>{label}</summary>\n<div class=\"detail-content\">\n")

    args = tc.request.arguments
    if args:
        _html_tool_arguments(w, tc.request.name, args)

    if tc.result:
        status_class = "success" if tc.result.success else "error"
        status_text = "success" if tc.result.success else "failed"
        w(f"<p><strong>Result</strong> (<span class=\"{status_class}\">{status_text}</span>):</p>\n")
        output = tc.result.detailed_content or tc.result.content
        if output:
            w(f"<pre><code>{_esc(output.strip())}</code></pre>\n")

    w("</div>\n</details>\n")


def _html_tool_arguments(w, tool_name: str, args: dict[str, Any]) -> None:
    if tool_name == "powershell":
        cmd = args.get("command", "")
        if cmd:
            w(f"<p><strong>Command:</strong> <code>{_esc(cmd)}</code></p>\n")
    elif tool_name in ("view", "create", "edit"):
        path = args.get("path", "")
        if path:
            w(f"<p><strong>Path:</strong> <code>{_esc(path)}</code></p>\n")
        if tool_name == "edit":
            old = args.get("old_str", "")
            new = args.get("new_str", "")
            if old:
                w(f"<p><strong>Old:</strong></p><pre><code>{_esc(old)}</code></pre>\n")
            if new:
                w(f"<p><strong>New:</strong></p><pre><code>{_esc(new)}</code></pre>\n")
        elif tool_name == "create":
            ft = args.get("file_text", "")
            if ft:
                display = ft[:2000] + f"\n... ({len(ft)} chars total)" if len(ft) > 2000 else ft
                w(f"<p><strong>Content:</strong></p><pre><code>{_esc(display)}</code></pre>\n")
    elif tool_name in ("grep", "glob"):
        pattern = args.get("pattern", "")
        if pattern:
            w(f"<p><strong>Pattern:</strong> <code>{_esc(pattern)}</code></p>\n")
    elif tool_name == "sql":
        query = args.get("query", "")
        if query:
            w(f"<p><strong>Query:</strong></p><pre><code>{_esc(query)}</code></pre>\n")
    elif tool_name == "web_fetch":
        url = args.get("url", "")
        if url:
            w(f"<p><strong>URL:</strong> <a href=\"{_esc(url)}\">{_esc(url)}</a></p>\n")
    elif tool_name == "task":
        agent = args.get("agent_type", "")
        prompt = args.get("prompt", "")
        if agent:
            w(f"<p><strong>Agent:</strong> {_esc(agent)}</p>\n")
        if prompt:
            display = prompt[:500] + f"... ({len(prompt)} chars)" if len(prompt) > 500 else prompt
            w(f"<p><strong>Prompt:</strong> {_esc(display)}</p>\n")
    else:
        args_str = json.dumps(args, indent=2, ensure_ascii=False)
        if len(args_str) > 1000:
            args_str = args_str[:1000] + "\n... (truncated)"
        w(f"<p><strong>Arguments:</strong></p><pre><code>{_esc(args_str)}</code></pre>\n")


def _html_subagent(w, sa: SubAgentRun) -> None:
    status = "<span class=\"success\">✅</span>" if sa.success else "<span class=\"error\">❌</span>"
    name = _esc(sa.display_name or sa.agent_name)
    w(f"<details>\n<summary><span class=\"subagent-label\">🔍 Sub-agent: {name}</span> {status}</summary>\n")
    w("<div class=\"detail-content\">\n")
    if sa.description:
        w(f"<p><em>{_esc(sa.description.strip())}</em></p>\n")
    if sa.error:
        w(f"<p class=\"error\"><strong>Error:</strong> {_esc(sa.error)}</p>\n")
    w("</div>\n</details>\n")


def _html_checkpoints(w, session: ParsedSession) -> None:
    if not session.checkpoints:
        return
    w("<h2 id=\"checkpoints\">Checkpoints</h2>\n")
    for cp in session.checkpoints:
        w(f"<h3>Checkpoint {cp.index}: {_esc(cp.title)}</h3>\n")
        if cp.content:
            w("<details>\n<summary>View checkpoint content</summary>\n")
            w(f"<div class=\"detail-content\"><pre>{_esc(cp.content.strip())}</pre></div>\n")
            w("</details>\n")


def _html_statistics(w, session: ParsedSession) -> None:
    stats = session.shutdown_stats
    if not stats:
        return
    w("<h2 id=\"statistics\">Session Statistics</h2>\n")
    w("<div class=\"stat-grid\">\n")

    total_requests = stats.get("totalPremiumRequests")
    if total_requests is not None:
        w(f"<div class=\"stat-card\"><div class=\"stat-value\">{total_requests}</div><div class=\"stat-label\">Premium Requests</div></div>\n")

    api_ms = stats.get("totalApiDurationMs")
    if api_ms is not None:
        minutes = api_ms / 1000 / 60
        if minutes >= 1:
            w(f"<div class=\"stat-card\"><div class=\"stat-value\">{minutes:.1f}m</div><div class=\"stat-label\">API Duration</div></div>\n")
        else:
            seconds = api_ms / 1000
            w(f"<div class=\"stat-card\"><div class=\"stat-value\">{seconds:.1f}s</div><div class=\"stat-label\">API Duration</div></div>\n")

    changes = stats.get("codeChanges", {})
    if changes:
        added = changes.get("linesAdded", 0)
        removed = changes.get("linesRemoved", 0)
        files = changes.get("filesModified", [])
        w(f"<div class=\"stat-card\"><div class=\"stat-value\">+{added} / -{removed}</div><div class=\"stat-label\">Lines Changed ({len(files)} files)</div></div>\n")

    w("</div>\n")


def _html_errors(w, session: ParsedSession) -> None:
    if not session.errors:
        return
    w("<h2 id=\"errors\">Session Errors</h2>\n<ul>\n")
    for err in session.errors:
        err_type = _esc(err.get("errorType", "unknown"))
        msg = _esc(err.get("message", ""))
        w(f"<li>⚠️ <strong>{err_type}</strong>: {msg}</li>\n")
    w("</ul>\n")


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def _esc(s: str) -> str:
    return html.escape(str(s))


def _fmt_ts(ts) -> str:
    if ts is None:
        return ""
    return f"({ts.strftime('%H:%M:%S')})"
