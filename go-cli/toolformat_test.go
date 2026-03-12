package main

import (
	"strings"
	"testing"
)

func TestWriteToolArgsMD(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		args     map[string]interface{}
		contains []string
		absent   []string
	}{
		{
			name:     "powershell with command and mode",
			toolName: "powershell",
			args:     map[string]interface{}{"command": "go build ./...", "mode": "sync"},
			contains: []string{"**Command**: `go build ./...`", "**Mode**: sync"},
		},
		{
			name:     "view with path",
			toolName: "view",
			args:     map[string]interface{}{"path": "/src/main.go"},
			contains: []string{"**Path**: `/src/main.go`"},
		},
		{
			name:     "edit with path old_str new_str",
			toolName: "edit",
			args:     map[string]interface{}{"path": "/src/main.go", "old_str": "foo()", "new_str": "bar()"},
			contains: []string{"**Path**: `/src/main.go`", "**Old**:\n```\nfoo()", "**New**:\n```\nbar()"},
		},
		{
			name:     "create with short file_text",
			toolName: "create",
			args:     map[string]interface{}{"path": "/src/new.go", "file_text": "package main"},
			contains: []string{"**Path**: `/src/new.go`", "**Content**:\n```\npackage main\n```"},
			absent:   []string{"chars total)"},
		},
		{
			name:     "create with long file_text truncated",
			toolName: "create",
			args:     map[string]interface{}{"path": "/src/big.go", "file_text": strings.Repeat("x", 3000)},
			contains: []string{"**Path**: `/src/big.go`", "**Content**:", "(3000 chars total)"},
		},
		{
			name:     "grep with pattern and path",
			toolName: "grep",
			args:     map[string]interface{}{"pattern": "TODO", "path": "/src"},
			contains: []string{"**Pattern**: `TODO`", "**Path**: `/src`"},
		},
		{
			name:     "glob with pattern only",
			toolName: "glob",
			args:     map[string]interface{}{"pattern": "**/*.go"},
			contains: []string{"**Pattern**: `**/*.go`"},
			absent:   []string{"**Path**:"},
		},
		{
			name:     "sql with query",
			toolName: "sql",
			args:     map[string]interface{}{"query": "SELECT * FROM todos"},
			contains: []string{"**Query**:\n```sql\nSELECT * FROM todos\n```"},
		},
		{
			name:     "web_fetch with url",
			toolName: "web_fetch",
			args:     map[string]interface{}{"url": "https://example.com"},
			contains: []string{"**URL**: https://example.com"},
		},
		{
			name:     "task with short prompt",
			toolName: "task",
			args:     map[string]interface{}{"agent_type": "explore", "prompt": "Find all routes"},
			contains: []string{"**Agent**: explore", "**Prompt**: Find all routes"},
			absent:   []string{"chars)"},
		},
		{
			name:     "task with long prompt truncated",
			toolName: "task",
			args:     map[string]interface{}{"agent_type": "general-purpose", "prompt": strings.Repeat("a", 700)},
			contains: []string{"**Agent**: general-purpose", "**Prompt**: " + strings.Repeat("a", 500) + "...", "(700 chars)"},
		},
		{
			name:     "unknown tool shows JSON arguments",
			toolName: "custom_tool",
			args:     map[string]interface{}{"key1": "val1", "key2": "val2"},
			contains: []string{"**Arguments**:", "```json", "key1", "val1"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var b strings.Builder
			WriteToolArgsMD(&b, tc.toolName, tc.args)
			got := b.String()
			for _, want := range tc.contains {
				if !strings.Contains(got, want) {
					t.Errorf("output missing expected substring %q\ngot:\n%s", want, got)
				}
			}
			for _, unwanted := range tc.absent {
				if strings.Contains(got, unwanted) {
					t.Errorf("output should not contain %q\ngot:\n%s", unwanted, got)
				}
			}
		})
	}
}

func TestSval(t *testing.T) {
	tests := []struct {
		name string
		m    map[string]interface{}
		key  string
		want string
	}{
		{
			name: "key exists with string value",
			m:    map[string]interface{}{"name": "alice"},
			key:  "name",
			want: "alice",
		},
		{
			name: "key missing",
			m:    map[string]interface{}{"name": "alice"},
			key:  "age",
			want: "",
		},
		{
			name: "key exists but non-string value",
			m:    map[string]interface{}{"count": 42},
			key:  "count",
			want: "",
		},
		{
			name: "empty map",
			m:    map[string]interface{}{},
			key:  "anything",
			want: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := sval(tc.m, tc.key)
			if got != tc.want {
				t.Errorf("sval(%v, %q) = %q, want %q", tc.m, tc.key, got, tc.want)
			}
		})
	}
}
