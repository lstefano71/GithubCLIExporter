"""CLI interface for copilot-export."""

from __future__ import annotations

import argparse
import sys
from datetime import datetime
from pathlib import Path

from rich.console import Console
from rich.table import Table
from rich.prompt import IntPrompt

from .discovery import get_sessions_dir, resolve_session, scan_sessions
from .parser import parse_session
from .renderer_html import render_html
from .renderer_md import render_markdown

console = Console()


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        prog="copilot-export",
        description="Export Copilot CLI sessions to Markdown and HTML.",
    )
    parser.add_argument(
        "--sessions-dir",
        type=str,
        default=None,
        help="Override the sessions directory (default: ~/.copilot/session-state/)",
    )
    sub = parser.add_subparsers(dest="command")

    # --- list ---
    list_p = sub.add_parser("list", help="List available sessions")
    list_p.add_argument("--repo", type=str, help="Filter by repository (glob pattern)")
    list_p.add_argument("--since", type=str, help="Filter by date (YYYY-MM-DD)")
    list_p.add_argument("--search", type=str, help="Search in summary text")

    # --- export ---
    export_p = sub.add_parser("export", help="Export a session")
    export_p.add_argument(
        "session",
        nargs="?",
        default=None,
        help="Session specifier: index number, partial UUID, full UUID, or directory path",
    )
    export_p.add_argument(
        "--format",
        choices=["md", "html", "both"],
        default="both",
        help="Output format (default: both)",
    )
    export_p.add_argument(
        "--output",
        type=str,
        default=None,
        help="Output file path (without extension). Default: <session-dir>/export",
    )

    return parser


def run(args: argparse.Namespace) -> int:
    sessions_dir = Path(args.sessions_dir) if args.sessions_dir else get_sessions_dir()

    if args.command == "list":
        return _cmd_list(args, sessions_dir)
    elif args.command == "export":
        return _cmd_export(args, sessions_dir)
    else:
        # No command given — show help
        build_parser().print_help()
        return 0


def _cmd_list(args: argparse.Namespace, sessions_dir: Path) -> int:
    since = None
    if args.since:
        try:
            since = datetime.fromisoformat(args.since)
        except ValueError:
            console.print(f"[red]Invalid date format: {args.since}. Use YYYY-MM-DD.[/red]")
            return 1

    sessions = scan_sessions(
        sessions_dir=sessions_dir,
        repo_filter=args.repo,
        since=since,
        search=args.search,
    )

    if not sessions:
        console.print("[yellow]No sessions found.[/yellow]")
        return 0

    table = Table(title=f"Copilot CLI Sessions ({len(sessions)} found)")
    table.add_column("#", style="bold", width=4)
    table.add_column("Created", width=18)
    table.add_column("Repository", style="cyan", max_width=35)
    table.add_column("Summary", max_width=55)
    table.add_column("ID Prefix", style="dim", width=10)

    for i, s in enumerate(sessions, 1):
        created = s.created_at.strftime("%Y-%m-%d %H:%M") if s.created_at else "?"
        repo = s.repository or s.cwd or "?"
        summary = s.summary or ""
        if len(summary) > 55:
            summary = summary[:52] + "..."
        id_prefix = s.id[:8] if s.id else "?"
        table.add_row(str(i), created, repo, summary, id_prefix)

    console.print(table)
    return 0


def _cmd_export(args: argparse.Namespace, sessions_dir: Path) -> int:
    session_spec = args.session

    if session_spec is None:
        # Interactive selection
        session_spec = _interactive_select(sessions_dir)
        if session_spec is None:
            return 0

    try:
        session_path = resolve_session(session_spec, sessions_dir=sessions_dir)
    except (FileNotFoundError, ValueError) as e:
        console.print(f"[red]Error: {e}[/red]")
        return 1

    console.print(f"Parsing session: [cyan]{session_path}[/cyan]")
    session = parse_session(session_path)

    fmt = args.format
    output_base = args.output
    if not output_base:
        output_base = str(session_path / "export")

    if fmt in ("md", "both"):
        md_path = output_base + ".md"
        console.print(f"Rendering Markdown → [green]{md_path}[/green]")
        md_content = render_markdown(session)
        Path(md_path).write_text(md_content, encoding="utf-8")
        console.print(f"  ✅ {len(md_content):,} chars written")

    if fmt in ("html", "both"):
        html_path = output_base + ".html"
        console.print(f"Rendering HTML → [green]{html_path}[/green]")
        html_content = render_html(session)
        Path(html_path).write_text(html_content, encoding="utf-8")
        console.print(f"  ✅ {len(html_content):,} chars written")

    console.print("[bold green]Export complete![/bold green]")
    return 0


def _interactive_select(sessions_dir: Path) -> str | None:
    """Launch an interactive session picker."""
    sessions = scan_sessions(sessions_dir=sessions_dir)
    if not sessions:
        console.print("[yellow]No sessions found.[/yellow]")
        return None

    # Show table
    table = Table(title="Select a session to export")
    table.add_column("#", style="bold", width=4)
    table.add_column("Created", width=18)
    table.add_column("Repository", style="cyan", max_width=35)
    table.add_column("Summary", max_width=55)

    for i, s in enumerate(sessions, 1):
        created = s.created_at.strftime("%Y-%m-%d %H:%M") if s.created_at else "?"
        repo = s.repository or s.cwd or "?"
        summary = s.summary or ""
        if len(summary) > 55:
            summary = summary[:52] + "..."
        table.add_row(str(i), created, repo, summary)

    console.print(table)

    try:
        choice = IntPrompt.ask(
            "\nEnter session number (or 0 to cancel)",
            default=0,
            console=console,
        )
    except KeyboardInterrupt:
        return None

    if choice == 0 or choice < 1 or choice > len(sessions):
        return None

    return str(choice)
