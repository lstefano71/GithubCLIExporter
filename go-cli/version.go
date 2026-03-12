package main

import (
	"fmt"
	"runtime/debug"
)

// These variables are injected at build time via -ldflags:
//
//	go build -ldflags "-X main.version=1.0.0 -X main.commit=abc1234 -X main.date=2026-03-12T15:00:00Z"
//
// When not set (e.g. during local development or `go install`), init() falls
// back to Go's built-in VCS metadata from debug.ReadBuildInfo().
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func init() {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}

	// Use module version if available (set by `go install`)
	if version == "dev" && info.Main.Version != "" && info.Main.Version != "(devel)" {
		version = info.Main.Version
	}

	// Only populate from VCS info when ldflags didn't already set them
	commitFromVCS := false
	dirty := false

	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			if commit == "none" && len(s.Value) >= 12 {
				commit = s.Value[:12]
				commitFromVCS = true
			}
		case "vcs.time":
			if date == "unknown" {
				date = s.Value
			}
		case "vcs.modified":
			dirty = s.Value == "true"
		}
	}

	if dirty && commitFromVCS {
		commit += "-dirty"
	}
}

func shortVersion() string {
	return fmt.Sprintf("copilot-export %s", version)
}

func fullVersion() string {
	return fmt.Sprintf("copilot-export %s\n  commit: %s\n  built:  %s", version, commit, date)
}
