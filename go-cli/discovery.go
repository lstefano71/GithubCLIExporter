package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// GetSessionsDir returns the sessions directory, respecting env override.
func GetSessionsDir() string {
	if env := os.Getenv("COPILOT_SESSIONS_DIR"); env != "" {
		return env
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".copilot", "session-state")
}

// ScanSessions scans a sessions directory and returns metadata sorted newest-first.
func ScanSessions(sessionsDir, repoFilter string, since *time.Time, search string) []WorkspaceMetadata {
	if sessionsDir == "" {
		sessionsDir = GetSessionsDir()
	}
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		return nil
	}

	var sessions []WorkspaceMetadata
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		wsPath := filepath.Join(sessionsDir, entry.Name(), "workspace.yaml")
		data, err := os.ReadFile(wsPath)
		if err != nil {
			continue
		}
		var ws WorkspaceMetadata
		if err := yaml.Unmarshal(data, &ws); err != nil {
			continue
		}

		// Apply repo filter (case-insensitive glob match)
		if repoFilter != "" {
			if ws.Repository == "" {
				continue
			}
			matched, _ := filepath.Match(strings.ToLower(repoFilter), strings.ToLower(ws.Repository))
			if !matched {
				continue
			}
		}

		// Apply since filter
		if since != nil && ws.CreatedAt != nil && ws.CreatedAt.Before(*since) {
			continue
		}

		// Apply search filter (substring in summary)
		if search != "" {
			if !strings.Contains(strings.ToLower(ws.Summary), strings.ToLower(search)) {
				continue
			}
		}

		sessions = append(sessions, ws)
	}

	// Sort newest-first
	sort.Slice(sessions, func(i, j int) bool {
		ti := sessions[i].CreatedAt
		tj := sessions[j].CreatedAt
		if ti == nil && tj == nil {
			return false
		}
		if ti == nil {
			return false
		}
		if tj == nil {
			return true
		}
		return ti.After(*tj)
	})

	return sessions
}

// ResolveSession resolves a session specifier to a directory path.
func ResolveSession(specifier, sessionsDir string) (string, error) {
	if sessionsDir == "" {
		sessionsDir = GetSessionsDir()
	}

	// 1. Check if it's a direct path
	if info, err := os.Stat(specifier); err == nil && info.IsDir() {
		evPath := filepath.Join(specifier, "events.jsonl")
		if _, err := os.Stat(evPath); err == nil {
			return specifier, nil
		}
	}

	// 2. Try as index number
	if idx, err := strconv.Atoi(specifier); err == nil {
		sessions := ScanSessions(sessionsDir, "", nil, "")
		if idx >= 1 && idx <= len(sessions) {
			candidate := filepath.Join(sessionsDir, sessions[idx-1].ID)
			if info, err := os.Stat(candidate); err == nil && info.IsDir() {
				return candidate, nil
			}
		}
		return "", fmt.Errorf("index %d out of range (1–%d)", idx, len(sessions))
	}

	// 3. Try as full or partial UUID
	specLower := strings.ToLower(strings.TrimSpace(specifier))
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		return "", fmt.Errorf("sessions directory not found: %s", sessionsDir)
	}

	var matches []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		nameLower := strings.ToLower(name)
		if nameLower == specLower {
			return filepath.Join(sessionsDir, name), nil // exact match
		}
		if strings.HasPrefix(nameLower, specLower) {
			matches = append(matches, name)
		}
	}

	if len(matches) == 1 {
		return filepath.Join(sessionsDir, matches[0]), nil
	}
	if len(matches) > 1 {
		shown := matches
		suffix := ""
		if len(shown) > 5 {
			shown = shown[:5]
			suffix = "..."
		}
		return "", fmt.Errorf("ambiguous prefix '%s' matches %d sessions: %s%s",
			specifier, len(matches), strings.Join(shown, ", "), suffix)
	}
	return "", fmt.Errorf("no session found matching '%s'", specifier)
}
