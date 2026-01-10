package cli

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/ohare93/juggle/internal/session"
	"github.com/spf13/cobra"
)

var (
	exportFormat      string
	exportOutput      string
	exportIncludeDone bool
	exportBallIDs     string
	exportFilterState string
	exportSession     string
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export balls to JSON, CSV, or Ralph format",
	Long: `Export session data to JSON, CSV, or Ralph format for analysis or agent use.

By default exports all active balls (excluding done) across all discovered projects.
Use --local to restrict to current project only.

Filters are applied in order:
1. --session (if specified, exports only balls with matching session tag)
2. --ball-ids (if specified, only these balls)
3. --filter-state (if specified, only balls in these states)
4. --include-done (if false, excludes completed balls)

The Ralph format (--format ralph) is designed for agent loops and includes:
- <context> section from the session's context
- <progress> section from the session's progress.txt
- <tasks> section with balls, their state, priority, and todos

Examples:
  # Export all active balls across all projects
  juggler export --format json --output balls.json

  # Export only current project balls
  juggler export --local --format csv

  # Export session in Ralph format for agent use
  juggler export --session my-feature --format ralph

  # Export specific balls by ID (supports full or short IDs)
  juggler export --ball-ids "juggler-5,48" --format json

  # Export only in_progress balls
  juggler export --filter-state in_progress --format json

  # Combine filters: export local pending and in_progress balls
  juggler export --local --filter-state "pending,in_progress" --format csv`,
	RunE: runExport,
}

func init() {
	exportCmd.Flags().StringVar(&exportFormat, "format", "json", "Export format: json, csv, or ralph")
	exportCmd.Flags().StringVar(&exportOutput, "output", "", "Output file path (default: stdout)")
	exportCmd.Flags().BoolVar(&exportIncludeDone, "include-done", false, "Include archived (done) balls in export")
	exportCmd.Flags().StringVar(&exportBallIDs, "ball-ids", "", "Filter by specific ball IDs (comma-separated, supports full or short IDs)")
	exportCmd.Flags().StringVar(&exportFilterState, "filter-state", "", "Filter by states (comma-separated: pending, in_progress, blocked, complete)")
	exportCmd.Flags().StringVar(&exportSession, "session", "", "Export balls from a specific session (for ralph format, includes context and progress)")
}

func runExport(cmd *cobra.Command, args []string) error {
	// Validate format
	if exportFormat != "json" && exportFormat != "csv" && exportFormat != "ralph" {
		return fmt.Errorf("invalid format: %s (must be json, csv, or ralph)", exportFormat)
	}

	// Ralph format requires --session
	if exportFormat == "ralph" && exportSession == "" {
		return fmt.Errorf("ralph format requires --session flag")
	}

	// Get current directory
	cwd, err := GetWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Load config to discover projects
	config, err := LoadConfigForCommand()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create store for current directory (needed for DiscoverProjectsForCommand)
	store, err := NewStoreForCommand(cwd)
	if err != nil {
		return fmt.Errorf("failed to create store: %w", err)
	}

	// Discover projects (respects --local flag)
	projects, err := DiscoverProjectsForCommand(config, store)
	if err != nil {
		return fmt.Errorf("failed to discover projects: %w", err)
	}

	if len(projects) == 0 {
		return fmt.Errorf("no projects with .juggler directories found")
	}

	// Load all balls from discovered projects
	allBalls, err := session.LoadAllBalls(projects)
	if err != nil {
		return fmt.Errorf("failed to load balls: %w", err)
	}

	// Apply filters in order: session → ball-ids → filter-state → include-done
	balls := allBalls

	// Filter 0: --session (if specified, filter by session tag)
	if exportSession != "" {
		filteredBalls := make([]*session.Session, 0)
		for _, ball := range balls {
			for _, tag := range ball.Tags {
				if tag == exportSession {
					filteredBalls = append(filteredBalls, ball)
					break
				}
			}
		}
		balls = filteredBalls
	}

	// Filter 1: --ball-ids (if specified)
	if exportBallIDs != "" {
		balls, err = filterByBallIDs(balls, exportBallIDs, projects)
		if err != nil {
			return err
		}
	}

	// Filter 2: --filter-state (if specified)
	if exportFilterState != "" {
		balls, err = filterByState(balls, exportFilterState)
		if err != nil {
			return err
		}
	}

	// Filter 3: --include-done (always applied, except for ralph format which always includes all)
	if !exportIncludeDone && exportFormat != "ralph" {
		filteredBalls := make([]*session.Session, 0)
		for _, ball := range balls {
			if ball.State != session.StateComplete {
				filteredBalls = append(filteredBalls, ball)
			}
		}
		balls = filteredBalls
	}

	// For ralph format, we allow empty balls (session might just have context)
	if len(balls) == 0 && exportFormat != "ralph" {
		return fmt.Errorf("no balls to export")
	}

	// Export based on format
	var output []byte
	switch exportFormat {
	case "json":
		output, err = exportJSON(balls)
	case "csv":
		output, err = exportCSV(balls)
	case "ralph":
		output, err = exportRalph(cwd, exportSession, balls)
	}

	if err != nil {
		return fmt.Errorf("failed to export: %w", err)
	}

	// Write to file or stdout
	if exportOutput != "" {
		if err := os.WriteFile(exportOutput, output, 0644); err != nil {
			return fmt.Errorf("failed to write to file: %w", err)
		}
		fmt.Printf("✓ Exported %d ball(s) to %s\n", len(balls), exportOutput)
	} else {
		fmt.Print(string(output))
	}

	return nil
}

// filterByBallIDs filters balls by specific IDs (supports full and short IDs)
func filterByBallIDs(balls []*session.Session, ballIDsStr string, projects []string) ([]*session.Session, error) {
	// Parse comma-separated list
	idStrs := strings.Split(ballIDsStr, ",")
	requestedIDs := make([]string, 0, len(idStrs))
	for _, id := range idStrs {
		id = strings.TrimSpace(id)
		if id != "" {
			requestedIDs = append(requestedIDs, id)
		}
	}

	if len(requestedIDs) == 0 {
		return balls, nil
	}

	// Build a map of all balls for quick lookup
	ballsByID := make(map[string]*session.Session)
	ballsByShortID := make(map[string][]*session.Session)

	for _, ball := range balls {
		ballsByID[ball.ID] = ball

		// Extract short ID (number after last dash)
		parts := strings.Split(ball.ID, "-")
		if len(parts) > 0 {
			shortID := parts[len(parts)-1]
			ballsByShortID[shortID] = append(ballsByShortID[shortID], ball)
		}
	}

	// Resolve each requested ID
	filteredBalls := make([]*session.Session, 0)
	seenBalls := make(map[string]bool)

	for _, requestedID := range requestedIDs {
		// Try full ID first
		if ball, exists := ballsByID[requestedID]; exists {
			if !seenBalls[ball.ID] {
				filteredBalls = append(filteredBalls, ball)
				seenBalls[ball.ID] = true
			}
			continue
		}

		// Try short ID
		if matches, exists := ballsByShortID[requestedID]; exists {
			if len(matches) == 1 {
				ball := matches[0]
				if !seenBalls[ball.ID] {
					filteredBalls = append(filteredBalls, ball)
					seenBalls[ball.ID] = true
				}
			} else if len(matches) > 1 {
				// Ambiguous short ID
				matchingIDs := make([]string, len(matches))
				for i, m := range matches {
					matchingIDs[i] = m.ID
				}
				return nil, fmt.Errorf("ambiguous short ID '%s' matches multiple balls: %s", requestedID, strings.Join(matchingIDs, ", "))
			}
			continue
		}

		// ID not found
		return nil, fmt.Errorf("ball ID not found: %s", requestedID)
	}

	return filteredBalls, nil
}

// filterByState filters balls by state(s)
// Supports new states: "pending", "in_progress", "blocked", "complete"
// Also supports legacy states for backward compatibility: "ready", "juggling", "dropped"
func filterByState(balls []*session.Session, stateStr string) ([]*session.Session, error) {
	// Parse comma-separated list
	stateStrs := strings.Split(stateStr, ",")
	stateFilters := make([]session.BallState, 0, len(stateStrs))

	for _, s := range stateStrs {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}

		// Map legacy states to new states
		var ballState session.BallState
		switch s {
		case "pending":
			ballState = session.StatePending
		case "in_progress":
			ballState = session.StateInProgress
		case "blocked":
			ballState = session.StateBlocked
		case "complete":
			ballState = session.StateComplete
		// Legacy state mappings
		case "ready":
			ballState = session.StatePending
		case "juggling":
			ballState = session.StateInProgress
		case "dropped":
			ballState = session.StateBlocked
		default:
			// Check if it's a legacy format with juggle state (e.g., "juggling:in-air")
			if strings.Contains(s, ":") {
				parts := strings.Split(s, ":")
				if parts[0] == "juggling" {
					ballState = session.StateInProgress
				} else {
					return nil, fmt.Errorf("invalid state: %s (must be pending, in_progress, blocked, or complete)", s)
				}
			} else {
				return nil, fmt.Errorf("invalid state: %s (must be pending, in_progress, blocked, or complete)", s)
			}
		}

		stateFilters = append(stateFilters, ballState)
	}

	if len(stateFilters) == 0 {
		return balls, nil
	}

	// Filter balls
	filteredBalls := make([]*session.Session, 0)
	for _, ball := range balls {
		for _, filter := range stateFilters {
			if ball.State == filter {
				filteredBalls = append(filteredBalls, ball)
				break
			}
		}
	}

	return filteredBalls, nil
}

// Legacy types and functions kept for backward compatibility
type stateFilter struct {
	activeState session.ActiveState
	juggleState string // empty means any juggle state
}

func matchesStateFilter(ball *session.Session, filter stateFilter) bool {
	if ball.ActiveState != filter.activeState {
		return false
	}

	if filter.juggleState == "" {
		return true
	}

	if ball.JuggleState == nil {
		return false
	}

	return string(*ball.JuggleState) == filter.juggleState
}

func isValidActiveState(state string) bool {
	return state == "ready" || state == "juggling" || state == "dropped" || state == "complete"
}

func isValidJuggleState(state string) bool {
	return state == "needs-thrown" || state == "in-air" || state == "needs-caught"
}

func exportJSON(balls []*session.Session) ([]byte, error) {
	// Create enhanced ball exports with computed fields
	type ballExport struct {
		*session.Session
		TodosCompleted int `json:"todos_completed"`
		TodosTotal     int `json:"todos_total"`
	}

	enhancedBalls := make([]*ballExport, len(balls))
	for i, ball := range balls {
		total, completed := ball.TodoStats()
		enhancedBalls[i] = &ballExport{
			Session:        ball,
			TodosCompleted: completed,
			TodosTotal:     total,
		}
	}

	// Create export structure
	export := struct {
		ExportedAt string        `json:"exported_at"`
		TotalBalls int           `json:"total_balls"`
		Balls      []*ballExport `json:"balls"`
	}{
		ExportedAt: fmt.Sprintf("%d", 1),
		TotalBalls: len(balls),
		Balls:      enhancedBalls,
	}

	data, err := json.MarshalIndent(export, "", "  ")
	if err != nil {
		return nil, err
	}

	return data, nil
}

func exportCSV(balls []*session.Session) ([]byte, error) {
	var buf strings.Builder
	writer := csv.NewWriter(&buf)

	// Write header
	header := []string{
		"ID",
		"Project",
		"Intent",
		"Description",
		"Priority",
		"ActiveState",
		"JuggleState",
		"StartedAt",
		"CompletedAt",
		"LastActivity",
		"Tags",
		"TodosTotal",
		"TodosCompleted",
		"CompletionNote",
	}
	if err := writer.Write(header); err != nil {
		return nil, err
	}

	// Write rows
	for _, ball := range balls {
		completedAt := ""
		if ball.CompletedAt != nil {
			completedAt = ball.CompletedAt.Format("2006-01-02 15:04:05")
		}

		tags := strings.Join(ball.Tags, ";")

		total, completed := ball.TodoStats()

		juggleState := ""
		if ball.JuggleState != nil {
			juggleState = string(*ball.JuggleState)
		}
		row := []string{
			ball.ID,
			ball.WorkingDir,
			ball.Intent,
			ball.Description,
			string(ball.Priority),
			string(ball.ActiveState),
			juggleState,
			ball.StartedAt.Format("2006-01-02 15:04:05"),
			completedAt,
			ball.LastActivity.Format("2006-01-02 15:04:05"),
			tags,
			fmt.Sprintf("%d", total),
			fmt.Sprintf("%d", completed),
			ball.CompletionNote,
		}

		if err := writer.Write(row); err != nil {
			return nil, err
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}

	return []byte(buf.String()), nil
}

// exportRalph exports session data in Ralph agent format
// Format:
// <context>
// [session context]
// </context>
//
// <progress>
// [progress.txt content]
// </progress>
//
// <tasks>
// [balls with todos]
// </tasks>
func exportRalph(projectDir, sessionID string, balls []*session.Session) ([]byte, error) {
	var buf strings.Builder

	// Load session store to get context and progress
	sessionStore, err := session.NewSessionStore(projectDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create session store: %w", err)
	}

	// Try to load the session
	juggleSession, err := sessionStore.LoadSession(sessionID)
	if err != nil {
		// Session might not exist yet, that's okay for Ralph export
		// We'll just have empty context
		juggleSession = &session.JuggleSession{
			ID:          sessionID,
			Description: "",
			Context:     "",
		}
	}

	// Load progress
	progress, _ := sessionStore.LoadProgress(sessionID) // Ignore error, empty progress is fine

	// Write <context> section
	buf.WriteString("<context>\n")
	if juggleSession.Description != "" {
		buf.WriteString("# " + juggleSession.Description + "\n\n")
	}
	if juggleSession.Context != "" {
		buf.WriteString(juggleSession.Context)
		if !strings.HasSuffix(juggleSession.Context, "\n") {
			buf.WriteString("\n")
		}
	}
	buf.WriteString("</context>\n\n")

	// Write <progress> section
	buf.WriteString("<progress>\n")
	if progress != "" {
		buf.WriteString(progress)
		if !strings.HasSuffix(progress, "\n") {
			buf.WriteString("\n")
		}
	}
	buf.WriteString("</progress>\n\n")

	// Write <tasks> section
	buf.WriteString("<tasks>\n")
	for i, ball := range balls {
		if i > 0 {
			buf.WriteString("\n")
		}
		writeBallForRalph(&buf, ball)
	}
	buf.WriteString("</tasks>\n")

	return []byte(buf.String()), nil
}

// writeBallForRalph writes a single ball in Ralph format
func writeBallForRalph(buf *strings.Builder, ball *session.Session) {
	// Task header with ID, state, and priority
	header := fmt.Sprintf("## %s [%s] (priority: %s)", ball.ID, ball.State, ball.Priority)
	if ball.ModelSize != "" {
		header += fmt.Sprintf(" (model: %s)", ball.ModelSize)
	}
	buf.WriteString(header + "\n")

	// Intent
	buf.WriteString(fmt.Sprintf("Intent: %s\n", ball.Intent))

	// Description if present
	if ball.Description != "" {
		buf.WriteString(fmt.Sprintf("Description: %s\n", ball.Description))
	}

	// Blocked reason if blocked
	if ball.State == session.StateBlocked && ball.BlockedReason != "" {
		buf.WriteString(fmt.Sprintf("Blocked: %s\n", ball.BlockedReason))
	}

	// Todos
	if len(ball.Todos) > 0 {
		buf.WriteString("Todos:\n")
		for i, todo := range ball.Todos {
			checkbox := "[ ]"
			if todo.Done {
				checkbox = "[x]"
			}
			buf.WriteString(fmt.Sprintf("  %d. %s %s\n", i+1, checkbox, todo.Text))
			if todo.Description != "" {
				buf.WriteString(fmt.Sprintf("     %s\n", todo.Description))
			}
		}
	}

	// Tags
	if len(ball.Tags) > 0 {
		buf.WriteString(fmt.Sprintf("Tags: %s\n", strings.Join(ball.Tags, ", ")))
	}
}
