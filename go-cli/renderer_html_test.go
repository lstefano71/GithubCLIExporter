package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Helper utilities
// ---------------------------------------------------------------------------

func mustContain(t *testing.T, output, substr string) {
	t.Helper()
	if !strings.Contains(output, substr) {
		t.Errorf("output should contain %q but does not", substr)
	}
}

func mustNotContain(t *testing.T, output, substr string) {
	t.Helper()
	if strings.Contains(output, substr) {
		t.Errorf("output should NOT contain %q but does", substr)
	}
}

// ---------------------------------------------------------------------------
// TestEsc
// ---------------------------------------------------------------------------

func TestEsc(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"angle brackets", "<div>", "&lt;div&gt;"},
		{"ampersand", "a&b", "a&amp;b"},
		{"double quotes", `say "hello"`, "say &#34;hello&#34;"},
		{"single quotes", "it's", "it&#39;s"},
		{"normal text unchanged", "hello world", "hello world"},
		{"empty string", "", ""},
		{"mixed special chars", `<a href="x">&`, `&lt;a href=&#34;x&#34;&gt;&amp;`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := esc(tc.input)
			if got != tc.want {
				t.Errorf("esc(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestMdToHTML
// ---------------------------------------------------------------------------

func TestMdToHTML(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		contain string
	}{
		{"bold text produces strong tag", "**bold**", "<strong>bold</strong>"},
		{"backtick produces code tag", "`code`", "<code>code</code>"},
		{"empty string produces empty output", "", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := mdToHTML(tc.input)
			if tc.contain == "" {
				if strings.TrimSpace(got) != "" {
					t.Errorf("mdToHTML(%q) = %q, want empty", tc.input, got)
				}
				return
			}
			if !strings.Contains(got, tc.contain) {
				t.Errorf("mdToHTML(%q) = %q, want it to contain %q", tc.input, got, tc.contain)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestFmtTSShort
// ---------------------------------------------------------------------------

func TestFmtTSShort(t *testing.T) {
	t.Run("nil returns empty string", func(t *testing.T) {
		got := fmtTSShort(nil)
		if got != "" {
			t.Errorf("fmtTSShort(nil) = %q, want \"\"", got)
		}
	})
	t.Run("non-nil returns HH:MM format", func(t *testing.T) {
		ts := time.Date(2025, 6, 15, 14, 30, 0, 0, time.UTC)
		got := fmtTSShort(&ts)
		if got != "14:30" {
			t.Errorf("fmtTSShort(%v) = %q, want \"14:30\"", ts, got)
		}
	})
}

// ---------------------------------------------------------------------------
// TestPreview
// ---------------------------------------------------------------------------

func TestPreview(t *testing.T) {
	tests := []struct {
		name   string
		text   string
		maxLen int
		want   string
	}{
		{"empty returns empty", "", 80, ""},
		{"short text returns same", "hello", 80, "hello"},
		{"long text truncated at maxLen", "abcdefghij", 5, "abcde"},
		{"multi-line newlines replaced with spaces", "line1\nline2\nline3", 80, "line1 line2 line3"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := preview(tc.text, tc.maxLen)
			if got != tc.want {
				t.Errorf("preview(%q, %d) = %q, want %q", tc.text, tc.maxLen, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestRenderHTML_Structure
// ---------------------------------------------------------------------------

func TestRenderHTML_Structure(t *testing.T) {
	session := &ParsedSession{
		Workspace: WorkspaceMetadata{
			Summary: "My Test Session",
		},
	}
	out := RenderHTML(session)

	if !strings.HasPrefix(out, "<!DOCTYPE html>") {
		t.Error("output should start with <!DOCTYPE html>")
	}
	if !strings.HasSuffix(out, "</html>") {
		t.Error("output should end with </html>")
	}

	for _, tag := range []string{"<html", "<head>", "<body>", "</head>", "</body>"} {
		mustContain(t, out, tag)
	}

	mustContain(t, out, "<title>My Test Session</title>")
	mustContain(t, out, `<meta name="generator"`)
	mustContain(t, out, "copilot-export v")
	mustContain(t, out, "<style>")
	mustContain(t, out, "<script>")
	mustContain(t, out, "theme-toggle")
}

// ---------------------------------------------------------------------------
// TestRenderHTML_FullSession
// ---------------------------------------------------------------------------

func buildFullHTMLSession() *ParsedSession {
	ts := time.Date(2025, 7, 1, 10, 0, 0, 0, time.UTC)
	return &ParsedSession{
		Workspace: WorkspaceMetadata{
			ID:         "session-abc",
			Cwd:        "/home/user/project",
			Repository: "owner/repo",
			Summary:    "Full session test",
			CreatedAt:  &ts,
			UpdatedAt:  &ts,
		},
		CopilotVersion: "1.0.0",
		Plan:           "## Plan\n- Step 1\n- Step 2",
		Todos: []Todo{
			{ID: "t1", Title: "Task one", Description: "Do something", Status: "done"},
			{ID: "t2", Title: "Task two", Description: "Do more", Status: "pending"},
		},
		Turns: []ConversationTurn{
			{
				Role:      "user",
				Timestamp: &ts,
				Content:   "Please help me with this project",
			},
			{
				Role:      "assistant",
				Timestamp: &ts,
				Content:   "Sure, I can help!",
				ToolCalls: []ToolCall{
					{
						Request: ToolRequest{
							ToolCallID: "tc1",
							Name:       "powershell",
							Arguments: map[string]interface{}{
								"command": "ls -la",
							},
						},
						Result: &ToolResult{
							ToolCallID: "tc1",
							ToolName:   "powershell",
							Success:    true,
							Content:    "file1.go\nfile2.go",
						},
						Description: "List files",
					},
				},
				SubAgents: []SubAgentRun{
					{
						ToolCallID:  "sa1",
						AgentName:   "explore",
						DisplayName: "Explore codebase",
						Description: "Search for patterns",
						Success:     true,
					},
				},
			},
		},
		Checkpoints: []Checkpoint{
			{Index: 1, Title: "Initial checkpoint", Content: "Checkpoint content here"},
		},
		ShutdownStats: map[string]interface{}{
			"totalPremiumRequests": 5,
		},
		Errors: []map[string]interface{}{
			{"errorType": "timeout", "message": "Request timed out"},
		},
	}
}

func TestRenderHTML_FullSession(t *testing.T) {
	session := buildFullHTMLSession()
	out := RenderHTML(session)

	sections := []string{
		`id="metadata"`,
		`id="plan"`,
		`id="todos"`,
		`id="conversation"`,
		`id="checkpoints"`,
		`id="statistics"`,
		`id="errors"`,
	}
	for _, s := range sections {
		mustContain(t, out, s)
	}

	mustContain(t, out, "👤 User")
	mustContain(t, out, "🤖 Assistant")

	// Tool call details
	mustContain(t, out, "<details>")
	mustContain(t, out, "🔧 powershell")
	mustContain(t, out, "List files")

	// Sub-agent details
	mustContain(t, out, "🔍 Sub-agent: Explore codebase")

	// Footer
	mustContain(t, out, "export-footer")
	mustContain(t, out, "Generated by copilot-export v")
}

// ---------------------------------------------------------------------------
// TestRenderHTML_EmptySession
// ---------------------------------------------------------------------------

func TestRenderHTML_EmptySession(t *testing.T) {
	session := &ParsedSession{}
	out := RenderHTML(session)

	if !strings.HasPrefix(out, "<!DOCTYPE html>") {
		t.Error("output should start with <!DOCTYPE html>")
	}
	mustContain(t, out, "<title>Session Export</title>")
	mustNotContain(t, out, `id="conversation"`)
}

// ---------------------------------------------------------------------------
// TestRenderHTML_TodoIcons
// ---------------------------------------------------------------------------

func TestRenderHTML_TodoIcons(t *testing.T) {
	session := &ParsedSession{
		Workspace: WorkspaceMetadata{Summary: "Todo icons test"},
		Todos: []Todo{
			{ID: "1", Title: "Done task", Status: "done"},
			{ID: "2", Title: "In progress task", Status: "in_progress"},
			{ID: "3", Title: "Pending task", Status: "pending"},
			{ID: "4", Title: "Blocked task", Status: "blocked"},
			{ID: "5", Title: "Unknown task", Status: "something_else"},
		},
	}
	out := RenderHTML(session)

	icons := []struct {
		icon   string
		status string
	}{
		{"✅", "done"},
		{"🔄", "in_progress"},
		{"⏳", "pending"},
		{"🚫", "blocked"},
		{"❓", "something_else"},
	}
	for _, ic := range icons {
		if !strings.Contains(out, ic.icon) {
			t.Errorf("output should contain icon %s for status %q", ic.icon, ic.status)
		}
	}
}

// ---------------------------------------------------------------------------
// TestRenderHTML_Integration
// ---------------------------------------------------------------------------

func TestRenderHTML_Integration(t *testing.T) {
	sessionDir := filepath.Join("..", "sessions", "01")
	if _, err := os.Stat(sessionDir); os.IsNotExist(err) {
		t.Skipf("session directory %s not found, skipping integration test", sessionDir)
	}

	session, err := ParseSession(sessionDir)
	if err != nil {
		t.Fatalf("ParseSession(%s) error: %v", sessionDir, err)
	}

	out := RenderHTML(session)

	if out == "" {
		t.Fatal("RenderHTML returned empty string")
	}
	if !strings.HasPrefix(out, "<!DOCTYPE html>") {
		t.Error("output should start with <!DOCTYPE html>")
	}
	if !strings.HasSuffix(out, "</html>") {
		t.Error("output should end with </html>")
	}

	expectedSections := []string{
		`id="metadata"`,
		`id="conversation"`,
	}
	for _, s := range expectedSections {
		mustContain(t, out, s)
	}
}

// ---------------------------------------------------------------------------
// TestRenderHTML_HtmlEscapingInContent
// ---------------------------------------------------------------------------

func TestRenderHTML_HtmlEscapingInContent(t *testing.T) {
	ts := time.Date(2025, 7, 1, 10, 0, 0, 0, time.UTC)
	session := &ParsedSession{
		Workspace: WorkspaceMetadata{Summary: "Escape test"},
		Turns: []ConversationTurn{
			{
				Role:      "user",
				Timestamp: &ts,
				Content:   "Use <script>alert('xss')</script> in code",
			},
		},
	}
	out := RenderHTML(session)

	// The raw HTML tags must NOT appear unescaped in the output outside of
	// the markdown-rendered turn-content div. The preview attribute must be
	// properly escaped.
	mustContain(t, out, "&lt;script&gt;")
	mustContain(t, out, "&lt;/script&gt;")
}
