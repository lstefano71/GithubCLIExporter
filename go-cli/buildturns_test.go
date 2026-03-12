package main

import (
"os"
"path/filepath"
"testing"
"time"
)

// helper for timestamps
func ts(s string) *time.Time {
t, _ := time.Parse(time.RFC3339, s)
return &t
}

// ---------------------------------------------------------------------------
// 1. TestBuildTurns_UserMessage
// ---------------------------------------------------------------------------

func TestBuildTurns_UserMessage(t *testing.T) {
now := ts("2026-01-15T10:00:00Z")
events := []Event{
{Type: EventUserMessage, Data: map[string]interface{}{"content": "hello"}, ID: "evt-1", Timestamp: now},
}
turns := buildTurns(events)
if len(turns) != 1 {
t.Fatalf("expected 1 turn, got %d", len(turns))
}
if turns[0].Role != "user" {
t.Errorf("Role = %q, want %q", turns[0].Role, "user")
}
if turns[0].Content != "hello" {
t.Errorf("Content = %q, want %q", turns[0].Content, "hello")
}
}

// ---------------------------------------------------------------------------
// 2. TestBuildTurns_AssistantTurn
// ---------------------------------------------------------------------------

func TestBuildTurns_AssistantTurn(t *testing.T) {
now := ts("2026-01-15T10:00:00Z")
events := []Event{
{Type: EventAssistantTurnStart, Data: map[string]interface{}{}, ID: "evt-1", Timestamp: now},
{Type: EventAssistantMessage, Data: map[string]interface{}{"content": "I can help"}, ID: "evt-2", Timestamp: now},
{Type: EventAssistantTurnEnd, Data: map[string]interface{}{}, ID: "evt-3", Timestamp: now},
}
turns := buildTurns(events)
if len(turns) != 1 {
t.Fatalf("expected 1 turn, got %d", len(turns))
}
if turns[0].Role != "assistant" {
t.Errorf("Role = %q, want %q", turns[0].Role, "assistant")
}
if turns[0].Content != "I can help" {
t.Errorf("Content = %q, want %q", turns[0].Content, "I can help")
}
}

// ---------------------------------------------------------------------------
// 3. TestBuildTurns_ModeChange
// ---------------------------------------------------------------------------

func TestBuildTurns_ModeChange(t *testing.T) {
now := ts("2026-01-15T10:00:00Z")
events := []Event{
{Type: EventSessionModeChanged, Data: map[string]interface{}{"newMode": "autopilot"}, ID: "evt-1", Timestamp: now},
{Type: EventUserMessage, Data: map[string]interface{}{"content": "go ahead"}, ID: "evt-2", Timestamp: now},
}
turns := buildTurns(events)
if len(turns) != 1 {
t.Fatalf("expected 1 turn, got %d", len(turns))
}
if turns[0].Mode != "autopilot" {
t.Errorf("Mode = %q, want %q", turns[0].Mode, "autopilot")
}
}

// ---------------------------------------------------------------------------
// 4. TestBuildTurns_UnfinishedAssistant
// ---------------------------------------------------------------------------

func TestBuildTurns_UnfinishedAssistant(t *testing.T) {
now := ts("2026-01-15T10:00:00Z")
events := []Event{
{Type: EventAssistantTurnStart, Data: map[string]interface{}{}, ID: "evt-1", Timestamp: now},
{Type: EventAssistantMessage, Data: map[string]interface{}{"content": "partial"}, ID: "evt-2", Timestamp: now},
}
turns := buildTurns(events)
if len(turns) != 1 {
t.Fatalf("expected 1 turn (unfinished), got %d", len(turns))
}
if turns[0].Content != "partial" {
t.Errorf("Content = %q, want %q", turns[0].Content, "partial")
}
}

// ---------------------------------------------------------------------------
// 5. TestBuildTurns_ThinkingContent
// ---------------------------------------------------------------------------

func TestBuildTurns_ThinkingContent(t *testing.T) {
now := ts("2026-01-15T10:00:00Z")
rawContent := "<antml:thinking>deep thought</antml:thinking>visible text"
events := []Event{
{Type: EventAssistantTurnStart, Data: map[string]interface{}{}, ID: "evt-1", Timestamp: now},
{Type: EventAssistantMessage, Data: map[string]interface{}{"content": rawContent}, ID: "evt-2", Timestamp: now},
{Type: EventAssistantTurnEnd, Data: map[string]interface{}{}, ID: "evt-3", Timestamp: now},
}
turns := buildTurns(events)
if len(turns) != 1 {
t.Fatalf("expected 1 turn, got %d", len(turns))
}
if turns[0].Thinking != "deep thought" {
t.Errorf("Thinking = %q, want %q", turns[0].Thinking, "deep thought")
}
if turns[0].Content != "visible text" {
t.Errorf("Content = %q, want %q", turns[0].Content, "visible text")
}
}
// ---------------------------------------------------------------------------
// 6. TestBuildTurns_ToolCalls
// ---------------------------------------------------------------------------

func TestBuildTurns_ToolCalls(t *testing.T) {
now := ts("2026-01-15T10:00:00Z")
events := []Event{
{Type: EventAssistantTurnStart, Data: map[string]interface{}{}, ID: "evt-1", Timestamp: now},
{
Type: EventAssistantMessage,
Data: map[string]interface{}{
"content": "running command",
"toolRequests": []interface{}{
map[string]interface{}{
"toolCallId": "tc-1",
"name":       "powershell",
"arguments": map[string]interface{}{
"command":     "echo hello",
"description": "say hello",
},
},
},
},
ID:        "evt-2",
Timestamp: now,
},
{
Type: EventToolExecutionStart,
Data: map[string]interface{}{
"toolCallId": "tc-1",
"toolName":   "powershell",
},
ID:        "evt-3",
Timestamp: now,
},
{
Type: EventToolExecutionComplete,
Data: map[string]interface{}{
"toolCallId": "tc-1",
"success":    true,
"result": map[string]interface{}{
"content": "hello\n",
},
},
ID:        "evt-4",
Timestamp: now,
},
{Type: EventAssistantTurnEnd, Data: map[string]interface{}{}, ID: "evt-5", Timestamp: now},
}
turns := buildTurns(events)
if len(turns) != 1 {
t.Fatalf("expected 1 turn, got %d", len(turns))
}
if len(turns[0].ToolCalls) != 1 {
t.Fatalf("expected 1 ToolCall, got %d", len(turns[0].ToolCalls))
}
tc := turns[0].ToolCalls[0]
if tc.Request.Name != "powershell" {
t.Errorf("Request.Name = %q, want %q", tc.Request.Name, "powershell")
}
if tc.Result == nil {
t.Fatal("Result is nil")
}
if !tc.Result.Success {
t.Error("Result.Success = false, want true")
}
}

// ---------------------------------------------------------------------------
// 7. TestBuildTurns_ReportIntentSkipped
// ---------------------------------------------------------------------------

func TestBuildTurns_ReportIntentSkipped(t *testing.T) {
now := ts("2026-01-15T10:00:00Z")
events := []Event{
{Type: EventAssistantTurnStart, Data: map[string]interface{}{}, ID: "evt-1", Timestamp: now},
{
Type: EventAssistantMessage,
Data: map[string]interface{}{
"content": "working",
"toolRequests": []interface{}{
map[string]interface{}{
"toolCallId": "tc-ri",
"name":       "report_intent",
"arguments": map[string]interface{}{
"intent": "Exploring codebase",
},
},
},
},
ID:        "evt-2",
Timestamp: now,
},
{Type: EventAssistantTurnEnd, Data: map[string]interface{}{}, ID: "evt-3", Timestamp: now},
}
turns := buildTurns(events)
if len(turns) != 1 {
t.Fatalf("expected 1 turn, got %d", len(turns))
}
if len(turns[0].ToolCalls) != 0 {
t.Errorf("expected 0 ToolCalls (report_intent skipped), got %d", len(turns[0].ToolCalls))
}
}

// ---------------------------------------------------------------------------
// 8. TestBuildTurns_SubAgents
// ---------------------------------------------------------------------------

func TestBuildTurns_SubAgents(t *testing.T) {
now := ts("2026-01-15T10:00:00Z")
events := []Event{
{Type: EventAssistantTurnStart, Data: map[string]interface{}{}, ID: "evt-1", Timestamp: now},
{
Type: EventAssistantMessage,
Data: map[string]interface{}{
"content": "launching agent",
"toolRequests": []interface{}{
map[string]interface{}{
"toolCallId": "tc-sa",
"name":       "task",
"arguments":  map[string]interface{}{},
},
},
},
ID:        "evt-2",
Timestamp: now,
},
{
Type: EventSubagentStarted,
Data: map[string]interface{}{
"toolCallId":       "tc-sa",
"agentName":        "explore",
"agentDisplayName": "Explore",
"agentDescription": "desc",
},
ID:        "evt-3",
Timestamp: now,
},
{
Type: EventSubagentCompleted,
Data: map[string]interface{}{
"toolCallId": "tc-sa",
},
ID:        "evt-4",
Timestamp: now,
},
{Type: EventAssistantTurnEnd, Data: map[string]interface{}{}, ID: "evt-5", Timestamp: now},
}
turns := buildTurns(events)
if len(turns) != 1 {
t.Fatalf("expected 1 turn, got %d", len(turns))
}
if len(turns[0].SubAgents) != 1 {
t.Fatalf("expected 1 SubAgent, got %d", len(turns[0].SubAgents))
}
sa := turns[0].SubAgents[0]
if !sa.Success {
t.Error("SubAgent.Success = false, want true")
}
if sa.AgentName != "explore" {
t.Errorf("AgentName = %q, want %q", sa.AgentName, "explore")
}
}

// ---------------------------------------------------------------------------
// 9. TestBuildTurns_SubAgentFailed
// ---------------------------------------------------------------------------

func TestBuildTurns_SubAgentFailed(t *testing.T) {
now := ts("2026-01-15T10:00:00Z")
events := []Event{
{Type: EventAssistantTurnStart, Data: map[string]interface{}{}, ID: "evt-1", Timestamp: now},
{
Type: EventAssistantMessage,
Data: map[string]interface{}{
"content": "launching agent",
"toolRequests": []interface{}{
map[string]interface{}{
"toolCallId": "tc-sa",
"name":       "task",
"arguments":  map[string]interface{}{},
},
},
},
ID:        "evt-2",
Timestamp: now,
},
{
Type: EventSubagentStarted,
Data: map[string]interface{}{
"toolCallId":       "tc-sa",
"agentName":        "explore",
"agentDisplayName": "Explore",
"agentDescription": "desc",
},
ID:        "evt-3",
Timestamp: now,
},
{
Type: EventSubagentFailed,
Data: map[string]interface{}{
"toolCallId": "tc-sa",
"error":      "timeout exceeded",
},
ID:        "evt-4",
Timestamp: now,
},
{Type: EventAssistantTurnEnd, Data: map[string]interface{}{}, ID: "evt-5", Timestamp: now},
}
turns := buildTurns(events)
if len(turns) != 1 {
t.Fatalf("expected 1 turn, got %d", len(turns))
}
if len(turns[0].SubAgents) != 1 {
t.Fatalf("expected 1 SubAgent, got %d", len(turns[0].SubAgents))
}
sa := turns[0].SubAgents[0]
if sa.Success {
t.Error("SubAgent.Success = true, want false")
}
if sa.Error != "timeout exceeded" {
t.Errorf("SubAgent.Error = %q, want %q", sa.Error, "timeout exceeded")
}
}

// ---------------------------------------------------------------------------
// 10. TestParseCheckpoints
// ---------------------------------------------------------------------------

func TestParseCheckpoints(t *testing.T) {
t.Run("with index.md", func(t *testing.T) {
cps := parseCheckpoints("testdata/full-session/checkpoints")
if len(cps) != 2 {
t.Fatalf("expected 2 checkpoints, got %d", len(cps))
}
if cps[0].Title != "Initial review" {
t.Errorf("cps[0].Title = %q, want %q", cps[0].Title, "Initial review")
}
if cps[1].Title != "Bug fix applied" {
t.Errorf("cps[1].Title = %q, want %q", cps[1].Title, "Bug fix applied")
}
})

t.Run("fallback no index.md", func(t *testing.T) {
dir := t.TempDir()
os.WriteFile(filepath.Join(dir, "alpha.md"), []byte("alpha content"), 0644)
os.WriteFile(filepath.Join(dir, "beta.md"), []byte("beta content"), 0644)
cps := parseCheckpoints(dir)
if len(cps) != 2 {
t.Fatalf("expected 2 checkpoints, got %d", len(cps))
}
if cps[0].Filename != "alpha.md" {
t.Errorf("cps[0].Filename = %q, want %q", cps[0].Filename, "alpha.md")
}
if cps[1].Filename != "beta.md" {
t.Errorf("cps[1].Filename = %q, want %q", cps[1].Filename, "beta.md")
}
})

t.Run("nonexistent dir", func(t *testing.T) {
cps := parseCheckpoints("testdata/does-not-exist/checkpoints")
if cps != nil {
t.Errorf("expected nil, got %d checkpoints", len(cps))
}
})
}

// ---------------------------------------------------------------------------
// 11. TestParseDB
// ---------------------------------------------------------------------------

func TestParseDB(t *testing.T) {
t.Run("full-session db", func(t *testing.T) {
todos, deps := parseDB("testdata/full-session/session.db")
if len(todos) != 3 {
t.Fatalf("expected 3 todos, got %d", len(todos))
}
if len(deps) != 2 {
t.Fatalf("expected 2 deps, got %d", len(deps))
}
})

t.Run("nonexistent file", func(t *testing.T) {
todos, deps := parseDB("testdata/nonexistent/session.db")
if todos != nil {
t.Errorf("expected nil todos, got %d", len(todos))
}
if deps != nil {
t.Errorf("expected nil deps, got %d", len(deps))
}
})
}

// ---------------------------------------------------------------------------
// 12. TestStrVal
// ---------------------------------------------------------------------------

func TestStrVal(t *testing.T) {
m := map[string]interface{}{
"name": "alice",
"age":  42,
}

t.Run("key present", func(t *testing.T) {
if got := strVal(m, "name"); got != "alice" {
t.Errorf("strVal(m, 'name') = %q, want %q", got, "alice")
}
})

t.Run("key missing", func(t *testing.T) {
if got := strVal(m, "missing"); got != "" {
t.Errorf("strVal(m, 'missing') = %q, want %q", got, "")
}
})

t.Run("key with int value", func(t *testing.T) {
if got := strVal(m, "age"); got != "" {
t.Errorf("strVal(m, 'age') = %q, want %q (int should return empty)", got, "")
}
})
}