package main

import (
	"time"
)

// EventType constants matching the 17 Copilot CLI event types.
const (
	EventSessionStart            = "session.start"
	EventSessionModeChanged      = "session.mode_changed"
	EventSessionPlanChanged      = "session.plan_changed"
	EventSessionCompactionStart  = "session.compaction_start"
	EventSessionCompactionEnd    = "session.compaction_complete"
	EventSessionError            = "session.error"
	EventSessionShutdown         = "session.shutdown"
	EventSessionTaskComplete     = "session.task_complete"
	EventUserMessage             = "user.message"
	EventAssistantTurnStart      = "assistant.turn_start"
	EventAssistantMessage        = "assistant.message"
	EventAssistantTurnEnd        = "assistant.turn_end"
	EventToolExecutionStart      = "tool.execution_start"
	EventToolExecutionComplete   = "tool.execution_complete"
	EventSubagentStarted         = "subagent.started"
	EventSubagentCompleted       = "subagent.completed"
	EventSubagentFailed          = "subagent.failed"
	EventAbort                   = "abort"
)

// WorkspaceMetadata from workspace.yaml.
type WorkspaceMetadata struct {
	ID           string     `yaml:"id"`
	Cwd          string     `yaml:"cwd"`
	GitRoot      string     `yaml:"git_root"`
	Repository   string     `yaml:"repository"`
	Summary      string     `yaml:"summary"`
	SummaryCount int        `yaml:"summary_count"`
	CreatedAt    *time.Time `yaml:"created_at"`
	UpdatedAt    *time.Time `yaml:"updated_at"`
}

// Event is a single entry from events.jsonl.
type Event struct {
	Type      string                 `json:"type"`
	Data      map[string]interface{} `json:"data"`
	ID        string                 `json:"id"`
	Timestamp *time.Time             `json:"-"`
	RawTS     string                 `json:"timestamp"`
	ParentID  string                 `json:"parentId"`
}

// ToolRequest from an assistant.message toolRequests array entry.
type ToolRequest struct {
	ToolCallID string                 `json:"toolCallId"`
	Name       string                 `json:"name"`
	Arguments  map[string]interface{} `json:"arguments"`
}

// ToolResult assembled from tool.execution_start + tool.execution_complete.
type ToolResult struct {
	ToolCallID      string
	ToolName        string
	Success         bool
	Content         string
	DetailedContent string
}

// SubAgentRun represents a sub-agent lifecycle.
type SubAgentRun struct {
	ToolCallID  string
	AgentName   string
	DisplayName string
	Description string
	Success     bool
	Error       string
}

// ToolCall pairs a request with its result.
type ToolCall struct {
	Request     ToolRequest
	Result      *ToolResult
	Description string
}

// ConversationTurn is a high-level grouping: one user or assistant turn.
type ConversationTurn struct {
	Role      string // "user" or "assistant"
	Timestamp *time.Time
	Content   string
	Thinking  string
	ToolCalls []ToolCall
	SubAgents []SubAgentRun
	Mode      string
}

// Todo from session.db.
type Todo struct {
	ID          string
	Title       string
	Description string
	Status      string
}

// TodoDep from session.db.
type TodoDep struct {
	TodoID    string
	DependsOn string
}

// Checkpoint from checkpoints/ directory.
type Checkpoint struct {
	Index    int
	Title    string
	Filename string
	Content  string
}

// ParsedSession is the top-level container for all session data.
type ParsedSession struct {
	Workspace      WorkspaceMetadata
	Events         []Event
	Turns          []ConversationTurn
	Todos          []Todo
	TodoDeps       []TodoDep
	Checkpoints    []Checkpoint
	Plan           string
	CopilotVersion string
	ShutdownStats  map[string]interface{}
	Errors         []map[string]interface{}
	SessionDir     string
}

// ParseTS parses a timestamp string, handling ISO 8601 with and without timezone.
func ParseTS(s string) *time.Time {
	if s == "" {
		return nil
	}
	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return &t
		}
	}
	return nil
}

// ToolResultFromEvents builds a ToolResult from start + complete event pair.
func ToolResultFromEvents(start, complete Event) ToolResult {
	result, _ := complete.Data["result"].(map[string]interface{})
	content := ""
	detailed := ""
	if result != nil {
		if c, ok := result["content"].(string); ok {
			content = c
		}
		if d, ok := result["detailedContent"].(string); ok {
			detailed = d
		}
	}

	success, _ := complete.Data["success"].(bool)
	toolCallID, _ := start.Data["toolCallId"].(string)
	toolName, _ := start.Data["toolName"].(string)

	return ToolResult{
		ToolCallID:      toolCallID,
		ToolName:        toolName,
		Success:         success,
		Content:         content,
		DetailedContent: detailed,
	}
}
