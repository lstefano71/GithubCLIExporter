package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// RenderMarkdown renders a parsed session as a Markdown document.
func RenderMarkdown(session *ParsedSession) string {
	var b strings.Builder
	renderMDFrontMatter(&b, session)
	renderMDHeader(&b, session)
	renderMDPlan(&b, session)
	renderMDTodos(&b, session)
	renderMDConversation(&b, session)
	renderMDCheckpoints(&b, session)
	renderMDStatistics(&b, session)
	renderMDErrors(&b, session)
	return b.String()
}

func renderMDFrontMatter(b *strings.Builder, s *ParsedSession) {
	b.WriteString("---\n")
	b.WriteString("generator: copilot-export\n")
	fmt.Fprintf(b, "generator_version: %s\n", version)

	ws := s.Workspace
	title := ws.Summary
	if title == "" {
		title = ws.ID
	}
	if title != "" {
		fmt.Fprintf(b, "title: %q\n", title)
	}
	if ws.CreatedAt != nil {
		fmt.Fprintf(b, "date: %q\n", ws.CreatedAt.Format(time.RFC3339))
	}
	if ws.Repository != "" {
		fmt.Fprintf(b, "repository: %q\n", ws.Repository)
	}
	if s.CopilotVersion != "" {
		fmt.Fprintf(b, "copilot_version: %q\n", s.CopilotVersion)
	}
	if ws.ID != "" {
		fmt.Fprintf(b, "session_id: %q\n", ws.ID)
	}
	b.WriteString("---\n\n")
}

func renderMDHeader(b *strings.Builder, s *ParsedSession) {
	ws := s.Workspace
	title := ws.Summary
	if title == "" {
		title = ws.ID
	}
	if title == "" {
		title = "Untitled Session"
	}
	fmt.Fprintf(b, "# Session: %s\n\n## Metadata\n\n", title)

	if ws.Repository != "" {
		fmt.Fprintf(b, "- **Repository**: %s\n", ws.Repository)
	}
	fmt.Fprintf(b, "- **Working Directory**: %s\n", ws.Cwd)
	if ws.GitRoot != "" && ws.GitRoot != ws.Cwd {
		fmt.Fprintf(b, "- **Git Root**: %s\n", ws.GitRoot)
	}
	if s.CopilotVersion != "" {
		fmt.Fprintf(b, "- **Copilot Version**: %s\n", s.CopilotVersion)
	}
	if ws.CreatedAt != nil {
		fmt.Fprintf(b, "- **Started**: %s\n", ws.CreatedAt.Format("2006-01-02 15:04:05 UTC"))
	}
	if ws.UpdatedAt != nil {
		fmt.Fprintf(b, "- **Last Updated**: %s\n", ws.UpdatedAt.Format("2006-01-02 15:04:05 UTC"))
	}
	if ws.ID != "" {
		fmt.Fprintf(b, "- **Session ID**: `%s`\n", ws.ID)
	}
	b.WriteString("\n")
}

func renderMDPlan(b *strings.Builder, s *ParsedSession) {
	if s.Plan == "" {
		return
	}
	b.WriteString("## Session Plan\n\n")
	b.WriteString(strings.TrimSpace(s.Plan))
	b.WriteString("\n\n")
}

func renderMDTodos(b *strings.Builder, s *ParsedSession) {
	if len(s.Todos) == 0 {
		return
	}
	b.WriteString("## Todos\n\n| Status | Title | Description |\n|--------|-------|-------------|\n")
	icons := map[string]string{"done": "✅", "in_progress": "🔄", "pending": "⏳", "blocked": "🚫"}
	for _, t := range s.Todos {
		icon := icons[t.Status]
		if icon == "" {
			icon = "❓"
		}
		desc := strings.ReplaceAll(t.Description, "\n", " ")
		if len(desc) > 120 {
			desc = desc[:117] + "..."
		}
		fmt.Fprintf(b, "| %s %s | %s | %s |\n", icon, t.Status, t.Title, desc)
	}
	b.WriteString("\n")
}

func renderMDConversation(b *strings.Builder, s *ParsedSession) {
	if len(s.Turns) == 0 {
		return
	}
	b.WriteString("## Conversation\n\n")
	for _, turn := range s.Turns {
		if turn.Role == "user" {
			renderMDUserTurn(b, turn)
		} else {
			renderMDAssistantTurn(b, turn)
		}
	}
}

func renderMDUserTurn(b *strings.Builder, t ConversationTurn) {
	fmt.Fprintf(b, "### 👤 User %s\n\n%s\n\n", fmtTS(t.Timestamp), strings.TrimSpace(t.Content))
}

func renderMDAssistantTurn(b *strings.Builder, t ConversationTurn) {
	fmt.Fprintf(b, "### 🤖 Assistant %s\n\n", fmtTS(t.Timestamp))
	if t.Thinking != "" {
		b.WriteString("<details><summary>💭 Thinking</summary>\n\n")
		b.WriteString(strings.TrimSpace(t.Thinking))
		b.WriteString("\n\n</details>\n\n")
	}
	if t.Content != "" {
		b.WriteString(strings.TrimSpace(t.Content))
		b.WriteString("\n\n")
	}
	for _, tc := range t.ToolCalls {
		renderMDToolCall(b, tc)
	}
	for _, sa := range t.SubAgents {
		renderMDSubAgent(b, sa)
	}
}

func renderMDToolCall(b *strings.Builder, tc ToolCall) {
	label := fmt.Sprintf("🔧 %s", tc.Request.Name)
	if tc.Description != "" {
		label += " — " + tc.Description
	}
	fmt.Fprintf(b, "<details><summary>%s</summary>\n\n", label)
	if len(tc.Request.Arguments) > 0 {
		WriteToolArgsMD(b, tc.Request.Name, tc.Request.Arguments)
	}
	if tc.Result != nil {
		if tc.Result.Success {
			b.WriteString("**Result** (success):\n")
		} else {
			b.WriteString("**Result** (failed):\n")
		}
		output := tc.Result.DetailedContent
		if output == "" {
			output = tc.Result.Content
		}
		if output != "" {
			output = strings.TrimSpace(output)
			if strings.Contains(output, "\n") || len(output) > 200 {
				fmt.Fprintf(b, "```\n%s\n```\n", output)
			} else {
				fmt.Fprintf(b, "`%s`\n", output)
			}
		}
	}
	b.WriteString("\n</details>\n\n")
}

func renderMDSubAgent(b *strings.Builder, sa SubAgentRun) {
	status := "✅"
	if !sa.Success {
		status = "❌"
	}
	name := sa.DisplayName
	if name == "" {
		name = sa.AgentName
	}
	fmt.Fprintf(b, "<details><summary>🔍 Sub-agent: %s %s</summary>\n\n", name, status)
	if sa.Description != "" {
		fmt.Fprintf(b, "_%s_\n\n", strings.TrimSpace(sa.Description))
	}
	if sa.Error != "" {
		fmt.Fprintf(b, "**Error**: %s\n\n", sa.Error)
	}
	b.WriteString("</details>\n\n")
}

func renderMDCheckpoints(b *strings.Builder, s *ParsedSession) {
	if len(s.Checkpoints) == 0 {
		return
	}
	b.WriteString("## Checkpoints\n\n")
	for _, cp := range s.Checkpoints {
		fmt.Fprintf(b, "### Checkpoint %d: %s\n\n", cp.Index, cp.Title)
		if cp.Content != "" {
			b.WriteString("<details><summary>View checkpoint content</summary>\n\n")
			b.WriteString(strings.TrimSpace(cp.Content))
			b.WriteString("\n\n</details>\n\n")
		}
	}
}

func renderMDStatistics(b *strings.Builder, s *ParsedSession) {
	stats := s.ShutdownStats
	if len(stats) == 0 {
		return
	}
	b.WriteString("## Session Statistics\n\n")
	if v, ok := stats["totalPremiumRequests"]; ok {
		fmt.Fprintf(b, "- **Premium Requests**: %v\n", v)
	}
	if v, ok := stats["totalApiDurationMs"]; ok {
		if ms, ok := toFloat64(v); ok {
			secs := ms / 1000
			mins := secs / 60
			if mins >= 1 {
				fmt.Fprintf(b, "- **Total API Duration**: %.1f minutes\n", mins)
			} else {
				fmt.Fprintf(b, "- **Total API Duration**: %.1f seconds\n", secs)
			}
		}
	}
	if changes, ok := stats["codeChanges"].(map[string]interface{}); ok {
		added, _ := toFloat64(changes["linesAdded"])
		removed, _ := toFloat64(changes["linesRemoved"])
		files, _ := changes["filesModified"].([]interface{})
		fmt.Fprintf(b, "- **Code Changes**: +%.0f / -%.0f lines across %d files\n", added, removed, len(files))
	}
	if st, ok := stats["shutdownType"].(string); ok {
		fmt.Fprintf(b, "- **Shutdown**: %s\n", st)
	}
	b.WriteString("\n")
}

func renderMDErrors(b *strings.Builder, s *ParsedSession) {
	if len(s.Errors) == 0 {
		return
	}
	b.WriteString("## Session Errors\n\n")
	for _, e := range s.Errors {
		errType := "unknown"
		if v, ok := e["errorType"].(string); ok {
			errType = v
		}
		msg, _ := e["message"].(string)
		fmt.Fprintf(b, "- ⚠️ **%s**: %s\n", errType, msg)
	}
	b.WriteString("\n")
}

func fmtTS(ts *time.Time) string {
	if ts == nil {
		return ""
	}
	return fmt.Sprintf("(%s)", ts.Format("15:04:05"))
}

func toFloat64(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	}
	return 0, false
}
