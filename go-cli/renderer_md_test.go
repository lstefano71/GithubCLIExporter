package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// fmtTS
// ---------------------------------------------------------------------------

func TestFmtTS(t *testing.T) {
	t.Run("nil returns empty string", func(t *testing.T) {
		if got := fmtTS(nil); got != "" {
			t.Errorf("fmtTS(nil) = %q, want empty string", got)
		}
	})
	t.Run("non-nil returns (HH:MM:SS)", func(t *testing.T) {
		ts := time.Date(2025, 6, 15, 14, 30, 45, 0, time.UTC)
		got := fmtTS(&ts)
		want := "(14:30:45)"
		if got != want {
			t.Errorf("fmtTS(%v) = %q, want %q", ts, got, want)
		}
	})
}

// ---------------------------------------------------------------------------
// toFloat64
// ---------------------------------------------------------------------------

func TestToFloat64(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		wantVal float64
		wantOK  bool
	}{
		{"float64", float64(3.14), 3.14, true},
		{"int", int(42), 42, true},
		{"int64", int64(99), 99, true},
		{"json.Number", json.Number("123.456"), 123.456, true},
		{"string returns false", "hello", 0, false},
		{"nil returns false", nil, 0, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			val, ok := toFloat64(tc.input)
			if ok != tc.wantOK {
				t.Fatalf("toFloat64(%v) ok = %v, want %v", tc.input, ok, tc.wantOK)
			}
			if ok && val != tc.wantVal {
				t.Errorf("toFloat64(%v) = %v, want %v", tc.input, val, tc.wantVal)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func timePtr(t time.Time) *time.Time { return &t }

func buildFullSession() *ParsedSession {
	created := time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)
	updated := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
	userTS := time.Date(2025, 6, 15, 10, 5, 0, 0, time.UTC)
	assistTS := time.Date(2025, 6, 15, 10, 6, 0, 0, time.UTC)

	return &ParsedSession{
		Workspace: WorkspaceMetadata{
			Repository: "owner/repo",
			Cwd:        "/home/user/project",
			Summary:    "Full Test Session",
			ID:         "sess-001",
			CreatedAt:  &created,
			UpdatedAt:  &updated,
		},
		CopilotVersion: "1.2.3",
		Plan:           "Step 1: do things\nStep 2: profit",
		Todos: []Todo{
			{ID: "t1", Title: "Setup", Description: "Setup the project", Status: "done"},
			{ID: "t2", Title: "Implement", Description: "Write the code", Status: "pending"},
			{ID: "t3", Title: "Deploy", Description: "Ship it", Status: "in_progress"},
		},
		Turns: []ConversationTurn{
			{
				Role:      "user",
				Timestamp: &userTS,
				Content:   "Hello, build me a project.",
			},
			{
				Role:      "assistant",
				Timestamp: &assistTS,
				Content:   "Sure, I will build it now.",
				Thinking:  "Let me think about the architecture...",
				ToolCalls: []ToolCall{
					{
						Request: ToolRequest{
							ToolCallID: "tc-1",
							Name:       "powershell",
							Arguments:  map[string]interface{}{"command": "echo hello"},
						},
						Result: &ToolResult{
							ToolCallID: "tc-1",
							ToolName:   "powershell",
							Success:    true,
							Content:    "hello",
						},
						Description: "Run echo",
					},
				},
				SubAgents: []SubAgentRun{
					{
						ToolCallID:  "sa-1",
						AgentName:   "explore",
						DisplayName: "Explorer Agent",
						Description: "Searching the codebase",
						Success:     true,
					},
				},
			},
		},
		Checkpoints: []Checkpoint{
			{Index: 1, Title: "Initial commit", Content: "Checkpoint content here"},
		},
		ShutdownStats: map[string]interface{}{
			"totalPremiumRequests": float64(10),
			"totalApiDurationMs":   float64(45000),
			"codeChanges": map[string]interface{}{
				"linesAdded":    float64(100),
				"linesRemoved":  float64(20),
				"filesModified": []interface{}{"a.go", "b.go"},
			},
			"shutdownType": "normal",
		},
		Errors: []map[string]interface{}{
			{"errorType": "timeout", "message": "API call timed out"},
		},
	}
}

// ---------------------------------------------------------------------------
// TestRenderMarkdown_FullSession
// ---------------------------------------------------------------------------

func TestRenderMarkdown_FullSession(t *testing.T) {
	out := RenderMarkdown(buildFullSession())

	required := []string{
		"---\n",
		"generator: copilot-export",
		"generator_version:",
		"title: \"Full Test Session\"",
		"repository: \"owner/repo\"",
		"copilot_version: \"1.2.3\"",
		"session_id: \"sess-001\"",
		"# Session: Full Test Session",
		"## Metadata",
		"**Repository**: owner/repo",
		"**Copilot Version**: 1.2.3",
		"**Session ID**: `sess-001`",
		"## Session Plan",
		"Step 1: do things",
		"## Todos",
		"✅ done",
		"⏳ pending",
		"🔄 in_progress",
		"## Conversation",
		"### 👤 User",
		"Hello, build me a project.",
		"### 🤖 Assistant",
		"💭 Thinking",
		"🔧 powershell",
		"🔍 Sub-agent:",
		"## Checkpoints",
		"Checkpoint 1: Initial commit",
		"## Session Statistics",
		"**Premium Requests**: 10",
		"**Total API Duration**:",
		"**Code Changes**:",
		"**Shutdown**: normal",
		"## Session Errors",
		"⚠️ **timeout**: API call timed out",
	}

	for _, s := range required {
		if !strings.Contains(out, s) {
			t.Errorf("output missing %q", s)
		}
	}
}

// ---------------------------------------------------------------------------
// TestRenderMarkdown_EmptySession
// ---------------------------------------------------------------------------

func TestRenderMarkdown_EmptySession(t *testing.T) {
	s := &ParsedSession{
		Workspace: WorkspaceMetadata{Summary: "Empty"},
	}
	out := RenderMarkdown(s)

	if !strings.Contains(out, "---\n") {
		t.Error("missing front matter delimiters")
	}
	if !strings.Contains(out, "generator: copilot-export") {
		t.Error("missing generator field in front matter")
	}
	if !strings.Contains(out, "# Session: Empty") {
		t.Error("missing title")
	}
	// These sections should be absent when there is no data.
	for _, absent := range []string{"## Conversation", "## Todos", "## Session Plan", "## Checkpoints", "## Session Statistics", "## Session Errors"} {
		if strings.Contains(out, absent) {
			t.Errorf("unexpected section %q in empty session output", absent)
		}
	}
}

// ---------------------------------------------------------------------------
// TestRenderMarkdown_UserTurn
// ---------------------------------------------------------------------------

func TestRenderMarkdown_UserTurn(t *testing.T) {
	ts := time.Date(2025, 1, 1, 8, 0, 0, 0, time.UTC)
	s := &ParsedSession{
		Workspace: WorkspaceMetadata{Summary: "User Turn"},
		Turns: []ConversationTurn{
			{Role: "user", Timestamp: &ts, Content: "Please help me."},
		},
	}
	out := RenderMarkdown(s)

	if !strings.Contains(out, "### 👤 User") {
		t.Error("missing user heading")
	}
	if !strings.Contains(out, "Please help me.") {
		t.Error("missing user message content")
	}
	if !strings.Contains(out, "(08:00:00)") {
		t.Error("missing timestamp")
	}
}

// ---------------------------------------------------------------------------
// TestRenderMarkdown_AssistantTurnWithThinking
// ---------------------------------------------------------------------------

func TestRenderMarkdown_AssistantTurnWithThinking(t *testing.T) {
	ts := time.Date(2025, 1, 1, 9, 0, 0, 0, time.UTC)
	s := &ParsedSession{
		Workspace: WorkspaceMetadata{Summary: "Thinking Test"},
		Turns: []ConversationTurn{
			{
				Role:      "assistant",
				Timestamp: &ts,
				Content:   "Here is the answer.",
				Thinking:  "Deep reasoning about the problem...",
			},
		},
	}
	out := RenderMarkdown(s)

	if !strings.Contains(out, "💭 Thinking") {
		t.Error("missing thinking summary label")
	}
	if !strings.Contains(out, "Deep reasoning about the problem...") {
		t.Error("missing thinking content")
	}
	if !strings.Contains(out, "<details>") {
		t.Error("thinking should be wrapped in <details>")
	}
}

// ---------------------------------------------------------------------------
// TestRenderMarkdown_ToolCall
// ---------------------------------------------------------------------------

func TestRenderMarkdown_ToolCall(t *testing.T) {
	s := &ParsedSession{
		Workspace: WorkspaceMetadata{Summary: "ToolCall Test"},
		Turns: []ConversationTurn{
			{
				Role: "assistant",
				ToolCalls: []ToolCall{
					{
						Request: ToolRequest{
							ToolCallID: "tc-42",
							Name:       "powershell",
							Arguments:  map[string]interface{}{"command": "ls -la"},
						},
						Result: &ToolResult{
							ToolCallID: "tc-42",
							ToolName:   "powershell",
							Success:    true,
							Content:    "file1.txt\nfile2.txt",
						},
					},
				},
			},
		},
	}
	out := RenderMarkdown(s)

	if !strings.Contains(out, "🔧 powershell") {
		t.Error("missing tool call label")
	}
	if !strings.Contains(out, "file1.txt") {
		t.Error("missing tool result content")
	}
	if !strings.Contains(out, "**Result** (success)") {
		t.Error("missing success indicator")
	}
}

// ---------------------------------------------------------------------------
// TestRenderMarkdown_SubAgent
// ---------------------------------------------------------------------------

func TestRenderMarkdown_SubAgent(t *testing.T) {
	s := &ParsedSession{
		Workspace: WorkspaceMetadata{Summary: "SubAgent Test"},
		Turns: []ConversationTurn{
			{
				Role: "assistant",
				SubAgents: []SubAgentRun{
					{
						ToolCallID:  "sa-99",
						AgentName:   "explore",
						DisplayName: "Explorer",
						Description: "Looking for files",
						Success:     true,
					},
				},
			},
		},
	}
	out := RenderMarkdown(s)

	if !strings.Contains(out, "🔍 Sub-agent:") {
		t.Error("missing sub-agent label")
	}
	if !strings.Contains(out, "Explorer") {
		t.Error("missing sub-agent display name")
	}
	if !strings.Contains(out, "Looking for files") {
		t.Error("missing sub-agent description")
	}
}

// ---------------------------------------------------------------------------
// TestRenderMarkdown_Statistics
// ---------------------------------------------------------------------------

func TestRenderMarkdown_Statistics(t *testing.T) {
	s := &ParsedSession{
		Workspace: WorkspaceMetadata{Summary: "Stats Test"},
		ShutdownStats: map[string]interface{}{
			"totalPremiumRequests": float64(5),
			"totalApiDurationMs":   float64(120000),
			"shutdownType":         "normal",
		},
	}
	out := RenderMarkdown(s)

	if !strings.Contains(out, "## Session Statistics") {
		t.Error("missing statistics section")
	}
	if !strings.Contains(out, "**Premium Requests**: 5") {
		t.Error("missing premium requests value")
	}
	if !strings.Contains(out, "**Total API Duration**:") {
		t.Error("missing API duration")
	}
	// 120000 ms = 120 s = 2.0 minutes
	if !strings.Contains(out, "2.0 minutes") {
		t.Error("expected duration in minutes for >= 60s")
	}
	if !strings.Contains(out, "**Shutdown**: normal") {
		t.Error("missing shutdown type")
	}
}

// ---------------------------------------------------------------------------
// TestRenderMarkdown_Integration
// ---------------------------------------------------------------------------

func TestRenderMarkdown_Integration(t *testing.T) {
	sessionDir := filepath.Join("..", "sessions", "01")
	if _, err := os.Stat(sessionDir); os.IsNotExist(err) {
		t.Skipf("session data not found at %s, skipping integration test", sessionDir)
	}

	session, err := ParseSession(sessionDir)
	if err != nil {
		t.Fatalf("ParseSession(%s) error: %v", sessionDir, err)
	}

	out := RenderMarkdown(session)
	if len(out) == 0 {
		t.Fatal("RenderMarkdown produced empty output")
	}

	expected := []string{
		"# Session:",
		"## Metadata",
		"## Conversation",
	}
	for _, s := range expected {
		if !strings.Contains(out, s) {
			t.Errorf("integration output missing %q", s)
		}
	}
}
