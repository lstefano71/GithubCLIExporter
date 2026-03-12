package main

import (
	"strings"
	"testing"
)

func TestShortVersion(t *testing.T) {
	result := shortVersion()

	if !strings.Contains(result, "copilot-export") {
		t.Errorf("shortVersion() = %q, want it to contain %q", result, "copilot-export")
	}
	if !strings.Contains(result, version) {
		t.Errorf("shortVersion() = %q, want it to contain version %q", result, version)
	}
}

func TestFullVersion(t *testing.T) {
	result := fullVersion()

	for _, want := range []string{"copilot-export", "commit:", "built:", version, commit, date} {
		if !strings.Contains(result, want) {
			t.Errorf("fullVersion() = %q, want it to contain %q", result, want)
		}
	}
}
