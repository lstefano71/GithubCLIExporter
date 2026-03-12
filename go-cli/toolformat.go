package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

// WriteToolArgsMD writes tool arguments in Markdown format.
func WriteToolArgsMD(b *strings.Builder, toolName string, args map[string]interface{}) {
	writeToolArgs(b, toolName, args, false)
}

// WriteToolArgsHTML writes tool arguments in HTML-safe format (still Markdown, will be converted).
func WriteToolArgsHTML(b *strings.Builder, toolName string, args map[string]interface{}) {
	writeToolArgs(b, toolName, args, false)
}

func writeToolArgs(b *strings.Builder, toolName string, args map[string]interface{}, _ bool) {
	switch toolName {
	case "powershell":
		if cmd := sval(args, "command"); cmd != "" {
			fmt.Fprintf(b, "**Command**: `%s`\n\n", cmd)
		}
		if mode := sval(args, "mode"); mode != "" {
			fmt.Fprintf(b, "**Mode**: %s\n\n", mode)
		}

	case "view", "create", "edit":
		if p := sval(args, "path"); p != "" {
			fmt.Fprintf(b, "**Path**: `%s`\n\n", p)
		}
		if toolName == "edit" {
			if old := sval(args, "old_str"); old != "" {
				b.WriteString("**Old**:\n```\n")
				b.WriteString(old)
				b.WriteString("\n```\n")
			}
			if nw := sval(args, "new_str"); nw != "" {
				b.WriteString("**New**:\n```\n")
				b.WriteString(nw)
				b.WriteString("\n```\n")
			}
		} else if toolName == "create" {
			if ft := sval(args, "file_text"); ft != "" {
				b.WriteString("**Content**:\n```\n")
				if len(ft) > 2000 {
					b.WriteString(ft[:2000])
					fmt.Fprintf(b, "\n... (%d chars total)", len(ft))
				} else {
					b.WriteString(ft)
				}
				b.WriteString("\n```\n")
			}
		}

	case "grep", "glob":
		if pat := sval(args, "pattern"); pat != "" {
			fmt.Fprintf(b, "**Pattern**: `%s`\n\n", pat)
		}
		if p := sval(args, "path"); p != "" {
			fmt.Fprintf(b, "**Path**: `%s`\n\n", p)
		}

	case "sql":
		if q := sval(args, "query"); q != "" {
			b.WriteString("**Query**:\n```sql\n")
			b.WriteString(q)
			b.WriteString("\n```\n")
		}

	case "web_fetch":
		if u := sval(args, "url"); u != "" {
			fmt.Fprintf(b, "**URL**: %s\n\n", u)
		}

	case "task":
		if at := sval(args, "agent_type"); at != "" {
			fmt.Fprintf(b, "**Agent**: %s\n\n", at)
		}
		if p := sval(args, "prompt"); p != "" {
			if len(p) > 500 {
				fmt.Fprintf(b, "**Prompt**: %s... (%d chars)\n\n", p[:500], len(p))
			} else {
				fmt.Fprintf(b, "**Prompt**: %s\n\n", p)
			}
		}

	default:
		data, _ := json.MarshalIndent(args, "", "  ")
		s := string(data)
		if len(s) > 1000 {
			s = s[:1000] + "\n... (truncated)"
		}
		fmt.Fprintf(b, "**Arguments**:\n```json\n%s\n```\n", s)
	}
}

func sval(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
