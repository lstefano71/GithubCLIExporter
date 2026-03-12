package main

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
	_ "modernc.org/sqlite"
)

// Regex to detect thinking blocks in assistant content.
var thinkingRE = regexp.MustCompile(`(?s)<(?:antml:)?thinking(?:_mode)?[^>]*>(.*?)</(?:antml:)?thinking(?:_mode)?>`)

// ParseSession parses all files in a session directory.
func ParseSession(sessionDir string) (*ParsedSession, error) {
	info, err := os.Stat(sessionDir)
	if err != nil || !info.IsDir() {
		return nil, fmt.Errorf("session directory not found: %s", sessionDir)
	}

	workspace := parseWorkspace(filepath.Join(sessionDir, "workspace.yaml"))
	events := parseEvents(filepath.Join(sessionDir, "events.jsonl"))
	todos, todoDeps := parseDB(filepath.Join(sessionDir, "session.db"))
	plan := readText(filepath.Join(sessionDir, "plan.md"))
	checkpoints := parseCheckpoints(filepath.Join(sessionDir, "checkpoints"))

	copilotVersion := ""
	var shutdownStats map[string]interface{}
	var errors []map[string]interface{}

	for _, ev := range events {
		switch ev.Type {
		case EventSessionStart:
			if v, ok := ev.Data["copilotVersion"].(string); ok {
				copilotVersion = v
			}
		case EventSessionShutdown:
			shutdownStats = ev.Data
		case EventSessionError:
			errors = append(errors, ev.Data)
		}
	}

	turns := buildTurns(events)

	return &ParsedSession{
		Workspace:      workspace,
		Events:         events,
		Turns:          turns,
		Todos:          todos,
		TodoDeps:       todoDeps,
		Checkpoints:    checkpoints,
		Plan:           plan,
		CopilotVersion: copilotVersion,
		ShutdownStats:  shutdownStats,
		Errors:         errors,
		SessionDir:     sessionDir,
	}, nil
}

func parseWorkspace(path string) WorkspaceMetadata {
	data, err := os.ReadFile(path)
	if err != nil {
		return WorkspaceMetadata{}
	}
	var ws WorkspaceMetadata
	if err := yaml.Unmarshal(data, &ws); err != nil {
		return WorkspaceMetadata{}
	}
	return ws
}

func parseEvents(path string) []Event {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var events []Event
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var raw map[string]interface{}
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			continue
		}

		ev := Event{
			Type:     strVal(raw, "type"),
			ID:       strVal(raw, "id"),
			RawTS:    strVal(raw, "timestamp"),
			ParentID: strVal(raw, "parentId"),
		}
		if d, ok := raw["data"].(map[string]interface{}); ok {
			ev.Data = d
		} else {
			ev.Data = make(map[string]interface{})
		}
		ev.Timestamp = ParseTS(ev.RawTS)
		events = append(events, ev)
	}
	return events
}

func parseDB(path string) ([]Todo, []TodoDep) {
	if _, err := os.Stat(path); err != nil {
		return nil, nil
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, nil
	}
	defer db.Close()

	var todos []Todo
	var deps []TodoDep

	if tableExists(db, "todos") {
		rows, err := db.Query("SELECT id, title, description, status FROM todos")
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var t Todo
				var desc, status sql.NullString
				if err := rows.Scan(&t.ID, &t.Title, &desc, &status); err == nil {
					t.Description = desc.String
					t.Status = status.String
					if t.Status == "" {
						t.Status = "pending"
					}
					todos = append(todos, t)
				}
			}
		}
	}

	if tableExists(db, "todo_deps") {
		rows, err := db.Query("SELECT todo_id, depends_on FROM todo_deps")
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var d TodoDep
				if err := rows.Scan(&d.TodoID, &d.DependsOn); err == nil {
					deps = append(deps, d)
				}
			}
		}
	}

	return todos, deps
}

func tableExists(db *sql.DB, name string) bool {
	var n string
	err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", name).Scan(&n)
	return err == nil
}

func parseCheckpoints(cpDir string) []Checkpoint {
	info, err := os.Stat(cpDir)
	if err != nil || !info.IsDir() {
		return nil
	}

	indexPath := filepath.Join(cpDir, "index.md")
	var checkpoints []Checkpoint

	if content, err := os.ReadFile(indexPath); err == nil {
		// Parse table rows: | idx | title | filename |
		re := regexp.MustCompile(`\|\s*(\d+)\s*\|\s*(.+?)\s*\|\s*(.+?)\s*\|`)
		matches := re.FindAllStringSubmatch(string(content), -1)
		for _, m := range matches {
			idx, _ := strconv.Atoi(m[1])
			title := strings.TrimSpace(m[2])
			filename := strings.TrimSpace(m[3])
			cpContent := readText(filepath.Join(cpDir, filename))
			checkpoints = append(checkpoints, Checkpoint{
				Index:    idx,
				Title:    title,
				Filename: filename,
				Content:  cpContent,
			})
		}
	} else {
		// Fallback: read all .md files
		entries, _ := os.ReadDir(cpDir)
		var mdFiles []string
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") && e.Name() != "index.md" {
				mdFiles = append(mdFiles, e.Name())
			}
		}
		sort.Strings(mdFiles)
		for i, name := range mdFiles {
			stem := strings.TrimSuffix(name, ".md")
			checkpoints = append(checkpoints, Checkpoint{
				Index:    i + 1,
				Title:    stem,
				Filename: name,
				Content:  readText(filepath.Join(cpDir, name)),
			})
		}
	}
	return checkpoints
}

func readText(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

// buildTurns groups events into conversation turns.
func buildTurns(events []Event) []ConversationTurn {
	// Index tool/subagent events by toolCallId
	toolStarts := map[string]Event{}
	toolCompletes := map[string]Event{}
	subagentStarts := map[string]Event{}
	subagentEnds := map[string]Event{}

	for _, ev := range events {
		tcID := strVal(ev.Data, "toolCallId")
		switch ev.Type {
		case EventToolExecutionStart:
			toolStarts[tcID] = ev
		case EventToolExecutionComplete:
			toolCompletes[tcID] = ev
		case EventSubagentStarted:
			subagentStarts[tcID] = ev
		case EventSubagentCompleted, EventSubagentFailed:
			subagentEnds[tcID] = ev
		}
	}

	var turns []ConversationTurn
	var currentMode string
	var cur *ConversationTurn

	for _, ev := range events {
		switch ev.Type {
		case EventSessionModeChanged:
			if m, ok := ev.Data["newMode"].(string); ok {
				currentMode = m
			}

		case EventUserMessage:
			content, _ := ev.Data["content"].(string)
			turns = append(turns, ConversationTurn{
				Role:      "user",
				Timestamp: ev.Timestamp,
				Content:   content,
				Mode:      currentMode,
			})

		case EventAssistantTurnStart:
			cur = &ConversationTurn{
				Role:      "assistant",
				Timestamp: ev.Timestamp,
				Mode:      currentMode,
			}

		case EventAssistantMessage:
			if cur == nil {
				cur = &ConversationTurn{
					Role:      "assistant",
					Timestamp: ev.Timestamp,
					Mode:      currentMode,
				}
			}

			rawContent, _ := ev.Data["content"].(string)

			// Extract thinking sections
			thinkingParts := thinkingRE.FindAllStringSubmatch(rawContent, -1)
			if len(thinkingParts) > 0 {
				for _, m := range thinkingParts {
					if cur.Thinking != "" {
						cur.Thinking += "\n"
					}
					cur.Thinking += m[1]
				}
				cleanContent := strings.TrimSpace(thinkingRE.ReplaceAllString(rawContent, ""))
				if cleanContent != "" {
					if cur.Content != "" {
						cur.Content += "\n\n" + cleanContent
					} else {
						cur.Content = cleanContent
					}
				}
			} else if rawContent != "" {
				if cur.Content != "" {
					cur.Content += "\n\n" + rawContent
				} else {
					cur.Content = rawContent
				}
			}

			// Process tool requests
			if toolReqs, ok := ev.Data["toolRequests"].([]interface{}); ok {
				for _, trRaw := range toolReqs {
					trMap, ok := trRaw.(map[string]interface{})
					if !ok {
						continue
					}
					tr := ToolRequest{
						ToolCallID: strVal(trMap, "toolCallId"),
						Name:       strVal(trMap, "name"),
					}
					if args, ok := trMap["arguments"].(map[string]interface{}); ok {
						tr.Arguments = args
					} else {
						tr.Arguments = make(map[string]interface{})
					}

					// Skip internal-only tools
					if tr.Name == "report_intent" {
						continue
					}

					tcID := tr.ToolCallID
					var result *ToolResult
					if _, hasStart := toolStarts[tcID]; hasStart {
						if _, hasComplete := toolCompletes[tcID]; hasComplete {
							r := ToolResultFromEvents(toolStarts[tcID], toolCompletes[tcID])
							result = &r
						}
					}

					desc, _ := tr.Arguments["description"].(string)

					// Check if this is a sub-agent call
					if saStart, isSA := subagentStarts[tcID]; isSA {
						sa := SubAgentRun{
							ToolCallID:  tcID,
							AgentName:   strVal(saStart.Data, "agentName"),
							DisplayName: strVal(saStart.Data, "agentDisplayName"),
							Description: strVal(saStart.Data, "agentDescription"),
							Success:     true,
						}
						if saEnd, hasEnd := subagentEnds[tcID]; hasEnd {
							sa.Success = saEnd.Type == EventSubagentCompleted
							if saEnd.Type == EventSubagentFailed {
								sa.Error, _ = saEnd.Data["error"].(string)
							}
						}
						cur.SubAgents = append(cur.SubAgents, sa)
					} else {
						cur.ToolCalls = append(cur.ToolCalls, ToolCall{
							Request:     tr,
							Result:      result,
							Description: desc,
						})
					}
				}
			}

		case EventAssistantTurnEnd:
			if cur != nil {
				turns = append(turns, *cur)
				cur = nil
			}
		}
	}

	// Flush any unfinished assistant turn
	if cur != nil {
		turns = append(turns, *cur)
	}

	return turns
}

// strVal safely extracts a string from a map.
func strVal(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
