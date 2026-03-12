"""Session discovery — scan, list, and resolve Copilot CLI sessions."""

from __future__ import annotations

import fnmatch
import os
from datetime import datetime
from pathlib import Path

import yaml

from .models import WorkspaceMetadata

DEFAULT_SESSIONS_DIR = Path.home() / ".copilot" / "session-state"


def get_sessions_dir() -> Path:
    """Return the sessions directory, respecting env var override."""
    env = os.environ.get("COPILOT_SESSIONS_DIR")
    if env:
        return Path(env)
    return DEFAULT_SESSIONS_DIR


def scan_sessions(
    sessions_dir: Path | None = None,
    repo_filter: str | None = None,
    since: datetime | None = None,
    search: str | None = None,
) -> list[WorkspaceMetadata]:
    """Scan a sessions directory and return metadata for all valid sessions.

    Results are sorted newest-first by created_at.
    """
    root = sessions_dir or get_sessions_dir()
    if not root.is_dir():
        return []

    sessions: list[WorkspaceMetadata] = []
    for entry in root.iterdir():
        if not entry.is_dir():
            continue
        ws_path = entry / "workspace.yaml"
        if not ws_path.exists():
            continue
        try:
            with open(ws_path, "r", encoding="utf-8") as f:
                data = yaml.safe_load(f) or {}
            ws = WorkspaceMetadata.from_dict(data)
        except Exception:
            continue

        # Apply filters
        if repo_filter and ws.repository:
            if not fnmatch.fnmatch(ws.repository.lower(), repo_filter.lower()):
                continue
        elif repo_filter and not ws.repository:
            continue

        if since and ws.created_at and ws.created_at < since:
            continue

        if search:
            searchable = (ws.summary or "").lower()
            if search.lower() not in searchable:
                continue

        sessions.append(ws)

    # Sort newest-first
    sessions.sort(key=lambda s: s.created_at or datetime.min, reverse=True)
    return sessions


def resolve_session(
    specifier: str,
    sessions_dir: Path | None = None,
) -> Path:
    """Resolve a session specifier to a directory path.

    Specifier can be:
    - A directory path (returned as-is if valid)
    - A full UUID
    - A partial UUID prefix (git-style, must be unique)
    - An index number (1-based, from newest-first scan)
    """
    root = sessions_dir or get_sessions_dir()

    # 1. Check if it's a direct path
    as_path = Path(specifier)
    if as_path.is_dir() and (as_path / "events.jsonl").exists():
        return as_path

    # 2. Try as index number
    try:
        idx = int(specifier)
        sessions = scan_sessions(sessions_dir=root)
        if 1 <= idx <= len(sessions):
            session_id = sessions[idx - 1].id
            candidate = root / session_id
            if candidate.is_dir():
                return candidate
        raise ValueError(
            f"Index {idx} out of range (1–{len(sessions)})"
        )
    except ValueError as e:
        if "out of range" in str(e):
            raise
        pass  # Not a number, try UUID matching

    # 3. Try as full or partial UUID
    specifier_lower = specifier.lower().strip()
    if not root.is_dir():
        raise FileNotFoundError(f"Sessions directory not found: {root}")

    matches: list[Path] = []
    for entry in root.iterdir():
        if not entry.is_dir():
            continue
        if entry.name.lower() == specifier_lower:
            return entry  # Exact match
        if entry.name.lower().startswith(specifier_lower):
            matches.append(entry)

    if len(matches) == 1:
        return matches[0]
    elif len(matches) > 1:
        match_ids = [m.name for m in matches]
        raise ValueError(
            f"Ambiguous prefix '{specifier}' matches {len(matches)} sessions: "
            + ", ".join(match_ids[:5])
            + ("..." if len(match_ids) > 5 else "")
        )
    else:
        raise FileNotFoundError(
            f"No session found matching '{specifier}'"
        )
