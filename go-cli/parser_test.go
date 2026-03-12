package main

import (
"os"
"path/filepath"
"strings"
"testing"
)

// ev builds a minimal Event for test convenience.
func ev(typ string, data map[string]interface{}) Event {
return Event{Type: typ, Data: data}
}

// ---------------------------------------------------------------------------
// 1. TestParseSession_FullSession
// ---------------------------------------------------------------------------

func TestParseSession_FullSession(t *testing.T) {
ps, err := ParseSession("testdata/full-session")
if err != nil {
t.Fatalf("ParseSession returned error: %v", err)
}
if ps.Workspace.ID != "test-session-001" {
t.Errorf("Workspace.ID = %q, want %q", ps.Workspace.ID, "test-session-001")
}
if ps.Workspace.Repository != "testuser/myapp" {
t.Errorf("Workspace.Repository = %q, want %q", ps.Workspace.Repository, "testuser/myapp")
}
if ps.Workspace.Summary != "Test session for unit tests" {
t.Errorf("Workspace.Summary = %q, want %q", ps.Workspace.Summary, "Test session for unit tests")
}
if ps.Workspace.Cwd != "/home/user/projects/myapp" {
t.Errorf("Workspace.Cwd = %q, want %q", ps.Workspace.Cwd, "/home/user/projects/myapp")
}
if ps.Workspace.CreatedAt == nil {
t.Error("Workspace.CreatedAt is nil")
}
if ps.Workspace.UpdatedAt == nil {
t.Error("Workspace.UpdatedAt is nil")
}
if len(ps.Events) == 0 {
t.Fatal("Events slice is empty")
}
if len(ps.Turns) == 0 {
t.Fatal("Turns slice is empty")
}
hasUser, hasAssistant := false, false
for _, turn := range ps.Turns {
if turn.Role == "user" {
hasUser = true
}
if turn.Role == "assistant" {
hasAssistant = true
}
}
if !hasUser {
t.Error("no user turn found")
}
if !hasAssistant {
t.Error("no assistant turn found")
}
if len(ps.Todos) == 0 {
t.Fatal("Todos is empty")
}
foundDone := false
for _, td := range ps.Todos {
if td.ID == "t1" && td.Status == "done" {
foundDone = true
}
}
if !foundDone {
t.Error("expected todo t1 with status done")
}
if len(ps.TodoDeps) == 0 {
t.Fatal("TodoDeps is empty")
}
if ps.Plan == "" {
t.Error("Plan is empty")
}
if len(ps.Checkpoints) == 0 {
t.Fatal("Checkpoints is empty")
}
if ps.Checkpoints[0].Title != "Initial review" {
t.Errorf("Checkpoints[0].Title = %q, want %q", ps.Checkpoints[0].Title, "Initial review")
}
if len(ps.Checkpoints) > 1 && ps.Checkpoints[1].Title != "Bug fix applied" {
t.Errorf("Checkpoints[1].Title = %q, want %q", ps.Checkpoints[1].Title, "Bug fix applied")
}
if ps.CopilotVersion != "1.0.3" {
t.Errorf("CopilotVersion = %q, want %q", ps.CopilotVersion, "1.0.3")
}
if ps.ShutdownStats == nil {
t.Error("ShutdownStats is nil")
}
if len(ps.Errors) == 0 {
t.Error("Errors slice is empty, expected session.error events")
}
if ps.SessionDir != "testdata/full-session" {
t.Errorf("SessionDir = %q, want %q", ps.SessionDir, "testdata/full-session")
}
}
// ---------------------------------------------------------------------------
// 2. TestParseSession_RealSession01
// ---------------------------------------------------------------------------

func TestParseSession_RealSession01(t *testing.T) {
dir := filepath.Join("..", "sessions", "01")
if _, err := os.Stat(dir); err != nil {
t.Skip("session 01 data not found")
}
ps, err := ParseSession(dir)
if err != nil {
t.Fatalf("ParseSession returned error: %v", err)
}
if ps.Workspace.ID == "" {
t.Error("Workspace.ID is empty")
}
if len(ps.Events) == 0 {
t.Error("Events is empty")
}
if len(ps.Turns) == 0 {
t.Error("Turns is empty")
}
if len(ps.Checkpoints) == 0 {
t.Error("Checkpoints is empty")
}
}

// ---------------------------------------------------------------------------
// 3. TestParseSession_EmptySession
// ---------------------------------------------------------------------------

func TestParseSession_EmptySession(t *testing.T) {
ps, err := ParseSession("testdata/empty-session")
if err != nil {
t.Fatalf("ParseSession returned error: %v", err)
}
if ps.Workspace.ID != "empty-session-001" {
t.Errorf("Workspace.ID = %q, want %q", ps.Workspace.ID, "empty-session-001")
}
if len(ps.Events) != 0 {
t.Errorf("Events len = %d, want 0", len(ps.Events))
}
if len(ps.Turns) != 0 {
t.Errorf("Turns len = %d, want 0", len(ps.Turns))
}
if len(ps.Todos) != 0 {
t.Errorf("Todos len = %d, want 0", len(ps.Todos))
}
}

// ---------------------------------------------------------------------------
// 4. TestParseSession_NonexistentDir
// ---------------------------------------------------------------------------

func TestParseSession_NonexistentDir(t *testing.T) {
_, err := ParseSession("testdata/does-not-exist")
if err == nil {
t.Fatal("expected error for nonexistent directory, got nil")
}
}

// ---------------------------------------------------------------------------
// 5. TestParseWorkspace
// ---------------------------------------------------------------------------

func TestParseWorkspace(t *testing.T) {
t.Run("valid YAML", func(t *testing.T) {
ws := parseWorkspace("testdata/full-session/workspace.yaml")
if ws.ID != "test-session-001" {
t.Errorf("ID = %q, want %q", ws.ID, "test-session-001")
}
if ws.Repository != "testuser/myapp" {
t.Errorf("Repository = %q, want %q", ws.Repository, "testuser/myapp")
}
if ws.Cwd != "/home/user/projects/myapp" {
t.Errorf("Cwd = %q, want %q", ws.Cwd, "/home/user/projects/myapp")
}
})
t.Run("missing file", func(t *testing.T) {
ws := parseWorkspace("testdata/nonexistent/workspace.yaml")
if ws.ID != "" {
t.Errorf("expected empty WorkspaceMetadata, got ID=%q", ws.ID)
}
})
t.Run("malformed YAML", func(t *testing.T) {
dir := t.TempDir()
path := filepath.Join(dir, "workspace.yaml")
if err := os.WriteFile(path, []byte(":::invalid\nyaml: [unterminated"), 0644); err != nil {
t.Fatal(err)
}
ws := parseWorkspace(path)
if ws.ID != "" {
t.Errorf("expected empty WorkspaceMetadata for malformed YAML, got ID=%q", ws.ID)
}
})
}
// ---------------------------------------------------------------------------
// 6. TestParseEvents
// ---------------------------------------------------------------------------

func TestParseEvents(t *testing.T) {
t.Run("full-session events", func(t *testing.T) {
events := parseEvents("testdata/full-session/events.jsonl")
if len(events) == 0 {
t.Fatal("expected non-empty events")
}
typeSet := map[string]bool{}
for _, e := range events {
typeSet[e.Type] = true
}
for _, want := range []string{
EventSessionStart, EventUserMessage, EventAssistantMessage,
EventToolExecutionStart, EventToolExecutionComplete, EventSessionShutdown,
} {
if !typeSet[want] {
t.Errorf("event type %q not found", want)
}
}
})
t.Run("empty file", func(t *testing.T) {
events := parseEvents("testdata/empty-session/events.jsonl")
if len(events) != 0 {
t.Errorf("expected 0 events, got %d", len(events))
}
})
t.Run("blank lines interspersed", func(t *testing.T) {
dir := t.TempDir()
path := filepath.Join(dir, "events.jsonl")
line1 := `{"type":"user.message","id":"1","timestamp":"2026-01-15T10:00:00Z","data":{"content":"hi"}}`
line2 := `{"type":"assistant.message","id":"2","timestamp":"2026-01-15T10:01:00Z","data":{"content":"hello"}}`
jsonl := "\n" + line1 + "\n\n" + line2 + "\n\n"
if err := os.WriteFile(path, []byte(jsonl), 0644); err != nil {
t.Fatal(err)
}
events := parseEvents(path)
if len(events) != 2 {
t.Errorf("expected 2 events, got %d", len(events))
}
})
t.Run("missing file", func(t *testing.T) {
events := parseEvents("testdata/nonexistent/events.jsonl")
if events != nil {
t.Errorf("expected nil for missing file, got %d events", len(events))
}
})
}