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
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export balls to JSON or CSV",
	Long: `Export session data to JSON or CSV format for analysis or backup.

By default exports all active balls (excluding done) across all discovered projects.
Use --local to restrict to current project only.

Filters are applied in order:
1. --ball-ids (if specified, only these balls)
2. --filter-state (if specified, only balls in these states)
3. --include-done (if false, excludes completed balls)

Examples:
  # Export all active balls across all projects
  juggler export --format json --output balls.json

  # Export only current project balls
  juggler export --local --format csv

  # Export specific balls by ID (supports full or short IDs)
  juggler export --ball-ids "juggler-5,48" --format json

  # Export only juggling balls
  juggler export --filter-state juggling --format json

  # Export only in-air balls (requires ActiveState:JuggleState format)
  juggler export --filter-state "juggling:in-air" --format json

  # Combine filters: export local ready and juggling balls
  juggler export --local --filter-state "ready,juggling" --format csv`,
	RunE: runExport,
}

func init() {
	exportCmd.Flags().StringVar(&exportFormat, "format", "json", "Export format: json or csv")
	exportCmd.Flags().StringVar(&exportOutput, "output", "", "Output file path (default: stdout)")
	exportCmd.Flags().BoolVar(&exportIncludeDone, "include-done", false, "Include archived (done) balls in export")
	exportCmd.Flags().StringVar(&exportBallIDs, "ball-ids", "", "Filter by specific ball IDs (comma-separated, supports full or short IDs)")
	exportCmd.Flags().StringVar(&exportFilterState, "filter-state", "", "Filter by states (comma-separated: ready, juggling, juggling:in-air, etc.)")
}

func runExport(cmd *cobra.Command, args []string) error {
	// Validate format
	if exportFormat != "json" && exportFormat != "csv" {
		return fmt.Errorf("invalid format: %s (must be json or csv)", exportFormat)
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

	// Apply filters in order: ball-ids → filter-state → include-done
	balls := allBalls

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

	// Filter 3: --include-done (always applied)
	if !exportIncludeDone {
		filteredBalls := make([]*session.Session, 0)
		for _, ball := range balls {
			if ball.ActiveState != session.ActiveComplete {
				filteredBalls = append(filteredBalls, ball)
			}
		}
		balls = filteredBalls
	}

	if len(balls) == 0 {
		return fmt.Errorf("no balls to export")
	}

	// Export based on format
	var output []byte
	switch exportFormat {
	case "json":
		output, err = exportJSON(balls)
	case "csv":
		output, err = exportCSV(balls)
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
// Supports: "ready", "juggling", "juggling:in-air", "juggling:needs-thrown", etc.
func filterByState(balls []*session.Session, stateStr string) ([]*session.Session, error) {
	// Parse comma-separated list
	stateStrs := strings.Split(stateStr, ",")
	stateFilters := make([]stateFilter, 0, len(stateStrs))

	for _, s := range stateStrs {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}

		// Parse state (may include juggle state)
		parts := strings.Split(s, ":")
		activeState := parts[0]
		var juggleState string
		if len(parts) > 1 {
			juggleState = parts[1]
		}

		// Validate active state
		if !isValidActiveState(activeState) {
			return nil, fmt.Errorf("invalid active state: %s (must be ready, juggling, dropped, or complete)", activeState)
		}

		// Validate juggle state if present
		if juggleState != "" {
			if activeState != "juggling" {
				return nil, fmt.Errorf("juggle state can only be specified with 'juggling' active state: %s", s)
			}
			if !isValidJuggleState(juggleState) {
				return nil, fmt.Errorf("invalid juggle state: %s (must be needs-thrown, in-air, or needs-caught)", juggleState)
			}
		}

		stateFilters = append(stateFilters, stateFilter{
			activeState: session.ActiveState(activeState),
			juggleState: juggleState,
		})
	}

	if len(stateFilters) == 0 {
		return balls, nil
	}

	// Filter balls
	filteredBalls := make([]*session.Session, 0)
	for _, ball := range balls {
		for _, filter := range stateFilters {
			if matchesStateFilter(ball, filter) {
				filteredBalls = append(filteredBalls, ball)
				break
			}
		}
	}

	return filteredBalls, nil
}

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
