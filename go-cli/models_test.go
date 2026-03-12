package main

import (
	"testing"
	"time"
)

func TestParseTS(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantNil bool
		wantUTC time.Time // ignored when wantNil is true
	}{
		{
			name:    "RFC3339Nano",
			input:   "2026-01-15T10:30:00.123456789Z",
			wantNil: false,
			wantUTC: time.Date(2026, 1, 15, 10, 30, 0, 123456789, time.UTC),
		},
		{
			name:    "RFC3339",
			input:   "2026-01-15T10:30:00Z",
			wantNil: false,
			wantUTC: time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC),
		},
		{
			name:    "bare datetime with T separator",
			input:   "2026-01-15T10:30:00",
			wantNil: false,
			wantUTC: time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC),
		},
		{
			name:    "space-separated datetime",
			input:   "2026-01-15 10:30:00",
			wantNil: false,
			wantUTC: time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC),
		},
		{
			name:    "empty string returns nil",
			input:   "",
			wantNil: true,
		},
		{
			name:    "garbage string returns nil",
			input:   "not-a-date",
			wantNil: true,
		},
		{
			name:    "partial date returns nil",
			input:   "2026-01-15",
			wantNil: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseTS(tc.input)
			if tc.wantNil {
				if got != nil {
					t.Fatalf("ParseTS(%q) = %v, want nil", tc.input, got)
				}
				return
			}
			if got == nil {
				t.Fatalf("ParseTS(%q) = nil, want %v", tc.input, tc.wantUTC)
			}
			if !got.Equal(tc.wantUTC) {
				t.Errorf("ParseTS(%q) = %v, want %v", tc.input, *got, tc.wantUTC)
			}
		})
	}
}

func TestToolResultFromEvents(t *testing.T) {
	tests := []struct {
		name     string
		start    Event
		complete Event
		want     ToolResult
	}{
		{
			name: "all fields populated",
			start: Event{
				Type: EventToolExecutionStart,
				Data: map[string]interface{}{
					"toolCallId": "call-123",
					"toolName":   "readFile",
				},
			},
			complete: Event{
				Type: EventToolExecutionComplete,
				Data: map[string]interface{}{
					"success": true,
					"result": map[string]interface{}{
						"content":         "file contents here",
						"detailedContent": "full detailed output",
					},
				},
			},
			want: ToolResult{
				ToolCallID:      "call-123",
				ToolName:        "readFile",
				Success:         true,
				Content:         "file contents here",
				DetailedContent: "full detailed output",
			},
		},
		{
			name: "missing result map in complete event",
			start: Event{
				Data: map[string]interface{}{
					"toolCallId": "call-456",
					"toolName":   "grep",
				},
			},
			complete: Event{
				Data: map[string]interface{}{
					"success": true,
				},
			},
			want: ToolResult{
				ToolCallID:      "call-456",
				ToolName:        "grep",
				Success:         true,
				Content:         "",
				DetailedContent: "",
			},
		},
		{
			name: "missing content and detailedContent in result",
			start: Event{
				Data: map[string]interface{}{
					"toolCallId": "call-789",
					"toolName":   "bash",
				},
			},
			complete: Event{
				Data: map[string]interface{}{
					"success": true,
					"result":  map[string]interface{}{},
				},
			},
			want: ToolResult{
				ToolCallID:      "call-789",
				ToolName:        "bash",
				Success:         true,
				Content:         "",
				DetailedContent: "",
			},
		},
		{
			name: "success false",
			start: Event{
				Data: map[string]interface{}{
					"toolCallId": "call-fail",
					"toolName":   "edit",
				},
			},
			complete: Event{
				Data: map[string]interface{}{
					"success": false,
					"result": map[string]interface{}{
						"content": "error: file not found",
					},
				},
			},
			want: ToolResult{
				ToolCallID:      "call-fail",
				ToolName:        "edit",
				Success:         false,
				Content:         "error: file not found",
				DetailedContent: "",
			},
		},
		{
			name:     "empty Data maps",
			start:    Event{Data: map[string]interface{}{}},
			complete: Event{Data: map[string]interface{}{}},
			want: ToolResult{
				ToolCallID:      "",
				ToolName:        "",
				Success:         false,
				Content:         "",
				DetailedContent: "",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ToolResultFromEvents(tc.start, tc.complete)
			if got.ToolCallID != tc.want.ToolCallID {
				t.Errorf("ToolCallID = %q, want %q", got.ToolCallID, tc.want.ToolCallID)
			}
			if got.ToolName != tc.want.ToolName {
				t.Errorf("ToolName = %q, want %q", got.ToolName, tc.want.ToolName)
			}
			if got.Success != tc.want.Success {
				t.Errorf("Success = %v, want %v", got.Success, tc.want.Success)
			}
			if got.Content != tc.want.Content {
				t.Errorf("Content = %q, want %q", got.Content, tc.want.Content)
			}
			if got.DetailedContent != tc.want.DetailedContent {
				t.Errorf("DetailedContent = %q, want %q", got.DetailedContent, tc.want.DetailedContent)
			}
		})
	}
}
