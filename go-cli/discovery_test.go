package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// createFakeSession writes a minimal workspace.yaml and empty events.jsonl
// inside sessionsDir/<id> and returns the session directory path.
func createFakeSession(t *testing.T, dir, id, repo, summary string, createdAt time.Time) string {
	t.Helper()
	sessionDir := filepath.Join(dir, id)
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", sessionDir, err)
	}

	yaml := fmt.Sprintf(
		"id: %s\nrepository: %s\nsummary: %s\ncreated_at: %s\n",
		id, repo, summary, createdAt.UTC().Format(time.RFC3339),
	)
	if err := os.WriteFile(filepath.Join(sessionDir, "workspace.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatalf("WriteFile workspace.yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sessionDir, "events.jsonl"), []byte{}, 0o644); err != nil {
		t.Fatalf("WriteFile events.jsonl: %v", err)
	}
	return sessionDir
}

// ---------------------------------------------------------------------------
// GetSessionsDir
// ---------------------------------------------------------------------------

func TestGetSessionsDir_EnvOverride(t *testing.T) {
	t.Setenv("COPILOT_SESSIONS_DIR", "/custom/path/sessions")
	got := GetSessionsDir()
	if got != "/custom/path/sessions" {
		t.Errorf("expected /custom/path/sessions, got %s", got)
	}
}

func TestGetSessionsDir_Default(t *testing.T) {
	t.Setenv("COPILOT_SESSIONS_DIR", "")
	got := GetSessionsDir()
	suffix := filepath.Join(".copilot", "session-state")
	if !strings.HasSuffix(got, suffix) {
		t.Errorf("expected path ending with %s, got %s", suffix, got)
	}
}

// ---------------------------------------------------------------------------
// ScanSessions
// ---------------------------------------------------------------------------

func TestScanSessions(t *testing.T) {
	dir := t.TempDir()

	now := time.Now().UTC().Truncate(time.Second)
	t1 := now.Add(-3 * 24 * time.Hour) // 3 days ago
	t2 := now.Add(-1 * 24 * time.Hour) // 1 day ago
	t3 := now                           // now

	createFakeSession(t, dir, "aaa-oldest", "owner/repoA", "fix bug in parser", t1)
	createFakeSession(t, dir, "bbb-middle", "owner/repoB", "add tests for auth", t2)
	createFakeSession(t, dir, "ccc-newest", "owner/repoA", "refactor CLI export", t3)

	t.Run("NoFilters_AllReturnedNewestFirst", func(t *testing.T) {
		sessions := ScanSessions(dir, "", nil, "")
		if len(sessions) != 3 {
			t.Fatalf("expected 3 sessions, got %d", len(sessions))
		}
		if sessions[0].ID != "ccc-newest" {
			t.Errorf("expected newest first (ccc-newest), got %s", sessions[0].ID)
		}
		if sessions[1].ID != "bbb-middle" {
			t.Errorf("expected bbb-middle second, got %s", sessions[1].ID)
		}
		if sessions[2].ID != "aaa-oldest" {
			t.Errorf("expected aaa-oldest last, got %s", sessions[2].ID)
		}
	})

	t.Run("RepoFilter", func(t *testing.T) {
		sessions := ScanSessions(dir, "owner/repoA", nil, "")
		if len(sessions) != 2 {
			t.Fatalf("expected 2 sessions for repoA, got %d", len(sessions))
		}
		for _, s := range sessions {
			if s.Repository != "owner/repoA" {
				t.Errorf("unexpected repo %s", s.Repository)
			}
		}
	})

	t.Run("SinceFilter", func(t *testing.T) {
		cutoff := now.Add(-2 * 24 * time.Hour) // 2 days ago
		sessions := ScanSessions(dir, "", &cutoff, "")
		if len(sessions) != 2 {
			t.Fatalf("expected 2 sessions after cutoff, got %d", len(sessions))
		}
		for _, s := range sessions {
			if s.CreatedAt.Before(cutoff) {
				t.Errorf("session %s created at %v is before cutoff %v", s.ID, s.CreatedAt, cutoff)
			}
		}
	})

	t.Run("SearchFilter", func(t *testing.T) {
		sessions := ScanSessions(dir, "", nil, "auth")
		if len(sessions) != 1 {
			t.Fatalf("expected 1 session matching 'auth', got %d", len(sessions))
		}
		if sessions[0].ID != "bbb-middle" {
			t.Errorf("expected bbb-middle, got %s", sessions[0].ID)
		}
	})

	t.Run("SearchFilter_CaseInsensitive", func(t *testing.T) {
		sessions := ScanSessions(dir, "", nil, "CLI")
		if len(sessions) != 1 {
			t.Fatalf("expected 1 session matching 'CLI', got %d", len(sessions))
		}
		if sessions[0].ID != "ccc-newest" {
			t.Errorf("expected ccc-newest, got %s", sessions[0].ID)
		}
	})

	t.Run("EmptyDirectory", func(t *testing.T) {
		emptyDir := t.TempDir()
		sessions := ScanSessions(emptyDir, "", nil, "")
		if len(sessions) != 0 {
			t.Errorf("expected 0 sessions in empty dir, got %d", len(sessions))
		}
	})

	t.Run("NonExistentDirectory", func(t *testing.T) {
		sessions := ScanSessions(filepath.Join(dir, "nonexistent"), "", nil, "")
		if sessions != nil {
			t.Errorf("expected nil for nonexistent dir, got %v", sessions)
		}
	})
}

// ---------------------------------------------------------------------------
// ResolveSession
// ---------------------------------------------------------------------------

func TestResolveSession_DirectPath(t *testing.T) {
	dir := t.TempDir()
	sessionDir := filepath.Join(dir, "my-session")
	os.MkdirAll(sessionDir, 0o755)
	os.WriteFile(filepath.Join(sessionDir, "events.jsonl"), []byte{}, 0o644)

	got, err := ResolveSession(sessionDir, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != sessionDir {
		t.Errorf("expected %s, got %s", sessionDir, got)
	}
}

func TestResolveSession_DirectPath_MissingEvents(t *testing.T) {
	// A directory without events.jsonl should NOT resolve as direct path
	dir := t.TempDir()
	sessionDir := filepath.Join(dir, "no-events")
	os.MkdirAll(sessionDir, 0o755)

	_, err := ResolveSession(sessionDir, dir)
	if err == nil {
		t.Fatal("expected error for dir without events.jsonl")
	}
}

func TestResolveSession_ByIndex(t *testing.T) {
	dir := t.TempDir()
	now := time.Now().UTC().Truncate(time.Second)

	createFakeSession(t, dir, "sess-old", "owner/repo", "old session", now.Add(-2*time.Hour))
	createFakeSession(t, dir, "sess-new", "owner/repo", "new session", now)

	// Index 1 → newest session (sorted newest-first)
	got, err := ResolveSession("1", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := filepath.Join(dir, "sess-new")
	if got != expected {
		t.Errorf("expected %s, got %s", expected, got)
	}

	// Index 2 → second session
	got, err = ResolveSession("2", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected = filepath.Join(dir, "sess-old")
	if got != expected {
		t.Errorf("expected %s, got %s", expected, got)
	}
}

func TestResolveSession_ByIndex_OutOfRange(t *testing.T) {
	dir := t.TempDir()
	now := time.Now().UTC().Truncate(time.Second)
	createFakeSession(t, dir, "only-one", "owner/repo", "only session", now)

	_, err := ResolveSession("5", dir)
	if err == nil {
		t.Fatal("expected error for out-of-range index")
	}
	if !strings.Contains(err.Error(), "out of range") {
		t.Errorf("expected 'out of range' in error, got: %v", err)
	}
}

func TestResolveSession_ByFullUUID(t *testing.T) {
	dir := t.TempDir()
	uuid := "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
	now := time.Now().UTC().Truncate(time.Second)
	createFakeSession(t, dir, uuid, "owner/repo", "uuid session", now)

	got, err := ResolveSession(uuid, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := filepath.Join(dir, uuid)
	if got != expected {
		t.Errorf("expected %s, got %s", expected, got)
	}
}

func TestResolveSession_ByPrefix(t *testing.T) {
	dir := t.TempDir()
	now := time.Now().UTC().Truncate(time.Second)

	createFakeSession(t, dir, "abcd1234-0000-0000-0000-000000000001", "owner/repo", "s1", now)
	createFakeSession(t, dir, "efgh5678-0000-0000-0000-000000000002", "owner/repo", "s2", now)

	// Unambiguous prefix "abcd" → resolves
	got, err := ResolveSession("abcd", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := filepath.Join(dir, "abcd1234-0000-0000-0000-000000000001")
	if got != expected {
		t.Errorf("expected %s, got %s", expected, got)
	}
}

func TestResolveSession_AmbiguousPrefix(t *testing.T) {
	dir := t.TempDir()
	now := time.Now().UTC().Truncate(time.Second)

	createFakeSession(t, dir, "abcd1111-0000-0000-0000-000000000001", "owner/repo", "s1", now)
	createFakeSession(t, dir, "abcd2222-0000-0000-0000-000000000002", "owner/repo", "s2", now)

	_, err := ResolveSession("abcd", dir)
	if err == nil {
		t.Fatal("expected error for ambiguous prefix")
	}
	if !strings.Contains(err.Error(), "ambiguous") {
		t.Errorf("expected 'ambiguous' in error, got: %v", err)
	}
}

func TestResolveSession_NotFound(t *testing.T) {
	dir := t.TempDir()
	now := time.Now().UTC().Truncate(time.Second)
	createFakeSession(t, dir, "existing-session", "owner/repo", "exists", now)

	_, err := ResolveSession("nonexistent-xyz", dir)
	if err == nil {
		t.Fatal("expected error for non-matching specifier")
	}
	if !strings.Contains(err.Error(), "no session found") {
		t.Errorf("expected 'no session found' in error, got: %v", err)
	}
}
