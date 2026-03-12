package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

func main() {
	var sessionsDir string

	rootCmd := &cobra.Command{
		Use:     "copilot-export",
		Short:   "Export Copilot CLI sessions to Markdown and HTML",
		Version: version,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			fmt.Println(shortVersion())
		},
	}
	rootCmd.SetVersionTemplate(fullVersion() + "\n")
	rootCmd.PersistentFlags().StringVar(&sessionsDir, "sessions-dir", "", "Override sessions directory (default: ~/.copilot/session-state/)")

	// ---- list ----
	var repoFilter, sinceStr, searchStr string
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List available sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := sessionsDir
			if dir == "" {
				dir = GetSessionsDir()
			}
			var since *time.Time
			if sinceStr != "" {
				t, err := time.Parse("2006-01-02", sinceStr)
				if err != nil {
					return fmt.Errorf("invalid date format: %s (use YYYY-MM-DD)", sinceStr)
				}
				since = &t
			}
			sessions := ScanSessions(dir, repoFilter, since, searchStr)
			if len(sessions) == 0 {
				fmt.Println("No sessions found.")
				return nil
			}
			fmt.Printf("Copilot CLI Sessions (%d found)\n\n", len(sessions))
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "#\tCreated\tRepository\tSummary\tID Prefix")
			fmt.Fprintln(w, "─\t───────\t──────────\t───────\t─────────")
			for i, s := range sessions {
				created := "?"
				if s.CreatedAt != nil {
					created = s.CreatedAt.Format("2006-01-02 15:04")
				}
				repo := s.Repository
				if repo == "" {
					repo = s.Cwd
				}
				if repo == "" {
					repo = "?"
				}
				summary := s.Summary
				if len(summary) > 55 {
					summary = summary[:52] + "..."
				}
				idPrefix := "?"
				if len(s.ID) >= 8 {
					idPrefix = s.ID[:8]
				}
				fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n", i+1, created, repo, summary, idPrefix)
			}
			w.Flush()
			return nil
		},
	}
	listCmd.Flags().StringVar(&repoFilter, "repo", "", "Filter by repository (glob pattern)")
	listCmd.Flags().StringVar(&sinceStr, "since", "", "Filter by date (YYYY-MM-DD)")
	listCmd.Flags().StringVar(&searchStr, "search", "", "Search in summary text")

	// ---- export ----
	var formatStr, outputBase string
	exportCmd := &cobra.Command{
		Use:   "export [session]",
		Short: "Export a session to Markdown and/or HTML",
		Long: `Export a session to Markdown and/or HTML.

Session can be specified as:
  - An index number from 'list' output (e.g., 3)
  - A partial UUID prefix (e.g., 658a)
  - A full UUID
  - A directory path`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := sessionsDir
			if dir == "" {
				dir = GetSessionsDir()
			}

			var specifier string
			if len(args) > 0 {
				specifier = args[0]
			} else {
				// Interactive selection
				spec, err := interactiveSelect(dir)
				if err != nil {
					return err
				}
				if spec == "" {
					return nil
				}
				specifier = spec
			}

			sessionPath, err := ResolveSession(specifier, dir)
			if err != nil {
				return err
			}
			fmt.Printf("Parsing session: %s\n", sessionPath)

			session, err := ParseSession(sessionPath)
			if err != nil {
				return fmt.Errorf("failed to parse session: %w", err)
			}

			base := outputBase
			if base == "" {
				slug := slugify(session.Workspace.Summary)
				if slug == "" {
					slug = slugify(session.Workspace.ID)
				}
				if slug == "" {
					slug = "session"
				}
				exportsDir := filepath.Join(".", "exports")
				os.MkdirAll(exportsDir, 0o755)
				base = filepath.Join(exportsDir, slug)
			}

			if formatStr == "md" || formatStr == "both" {
				mdPath := base + ".md"
				fmt.Printf("Rendering Markdown → %s\n", mdPath)
				content := RenderMarkdown(session)
				if err := os.WriteFile(mdPath, []byte(content), 0o644); err != nil {
					return fmt.Errorf("failed to write %s: %w", mdPath, err)
				}
				fmt.Printf("  ✅ %d chars written\n", len(content))
			}
			if formatStr == "html" || formatStr == "both" {
				htmlPath := base + ".html"
				fmt.Printf("Rendering HTML → %s\n", htmlPath)
				content := RenderHTML(session)
				if err := os.WriteFile(htmlPath, []byte(content), 0o644); err != nil {
					return fmt.Errorf("failed to write %s: %w", htmlPath, err)
				}
				fmt.Printf("  ✅ %d chars written\n", len(content))
			}

			fmt.Println("Export complete!")
			return nil
		},
	}
	exportCmd.Flags().StringVar(&formatStr, "format", "both", "Output format: md, html, or both")
	exportCmd.Flags().StringVar(&outputBase, "output", "", "Output file path (without extension)")

	rootCmd.AddCommand(listCmd, exportCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func interactiveSelect(sessionsDir string) (string, error) {
	sessions := ScanSessions(sessionsDir, "", nil, "")
	if len(sessions) == 0 {
		fmt.Println("No sessions found.")
		return "", nil
	}

	fmt.Printf("Select a session to export (%d found)\n\n", len(sessions))
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "#\tCreated\tRepository\tSummary")
	fmt.Fprintln(w, "─\t───────\t──────────\t───────")
	for i, s := range sessions {
		created := "?"
		if s.CreatedAt != nil {
			created = s.CreatedAt.Format("2006-01-02 15:04")
		}
		repo := s.Repository
		if repo == "" {
			repo = s.Cwd
		}
		if repo == "" {
			repo = "?"
		}
		summary := s.Summary
		if len(summary) > 55 {
			summary = summary[:52] + "..."
		}
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", i+1, created, repo, summary)
	}
	w.Flush()

	fmt.Print("\nEnter session number (or 0 to cancel): ")
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" || line == "0" {
		return "", nil
	}
	return line, nil
}

var slugRe1 = regexp.MustCompile(`[^\w\s-]`)
var slugRe2 = regexp.MustCompile(`[\s_]+`)
var slugRe3 = regexp.MustCompile(`-+`)

func slugify(text string) string {
	text = strings.ToLower(strings.TrimSpace(text))
	text = slugRe1.ReplaceAllString(text, "")
	text = slugRe2.ReplaceAllString(text, "-")
	text = slugRe3.ReplaceAllString(text, "-")
	text = strings.Trim(text, "-")
	if len(text) > 80 {
		text = text[:80]
	}
	if text == "" {
		return "session"
	}
	return text
}
