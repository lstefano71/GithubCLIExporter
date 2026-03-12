package main

import (
	"strings"
	"testing"
)

func TestSlugify(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"normal text", "Hello World", "hello-world"},
		{"special characters", "Fix bug #123!", "fix-bug-123"},
		{"leading/trailing spaces", "  hello  ", "hello"},
		{"underscores", "hello_world", "hello-world"},
		{"multiple dashes and spaces", "hello   ---   world", "hello-world"},
		{"very long text", strings.Repeat("a", 100), strings.Repeat("a", 80)},
		{"empty string", "", "session"},
		{"whitespace only", "   ", "session"},
		{"already clean", "hello-world", "hello-world"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := slugify(tt.input)
			if got != tt.expected {
				t.Errorf("slugify(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
