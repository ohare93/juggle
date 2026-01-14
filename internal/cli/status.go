package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/ohare93/juggle/internal/session"
	"github.com/spf13/cobra"
)

var (
	filterTags     string
	filterPriority string
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show all active sessions",
	Long: `Display a table of all active sessions with their current status.

By default, shows balls from the current project only. Use --all to show balls from all discovered projects.

Examples:
  juggle status                    # Show current project only
  juggle status --all              # Show all discovered projects
  juggle status --tags feature     # Filter by tags
  juggle status --priority high    # Filter by priority`,
	RunE:  runStatus,
}

func init() {
	statusCmd.Flags().StringVar(&filterTags, "tags", "", "Filter by tags (comma-separated, OR logic)")
	statusCmd.Flags().StringVar(&filterPriority, "priority", "", "Filter by priority (low|medium|high|urgent)")
}

func runStatus(cmd *cobra.Command, args []string) error {
	// Get current directory
	cwd, err := GetWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// If current directory has .juggle, ensure it's tracked
	if _, err := os.Stat(filepath.Join(cwd, ".juggle")); err == nil {
		_ = session.EnsureProjectInSearchPaths(cwd)
	}

	// Load config to discover projects
	config, err := LoadConfigForCommand()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	store, err := NewStoreForCommand(cwd)
	if err != nil {
		return fmt.Errorf("failed to create store: %w", err)
	}

	// Discover projects (respects --all flag)
	projects, err := DiscoverProjectsForCommand(config, store)
	if err != nil {
		return fmt.Errorf("failed to discover projects: %w", err)
	}

	if len(projects) == 0 {
		fmt.Println("No projects with .juggle directories found.")
		fmt.Println("\nStart a new session with: juggle start")
		return nil
	}

	allBalls, err := session.LoadAllBalls(projects)
	if err != nil {
		return fmt.Errorf("failed to load balls: %w", err)
	}

	// Filter to non-complete balls
	activeBalls := make([]*session.Ball, 0)
	for _, ball := range allBalls {
		if ball.State != session.StateComplete {
			activeBalls = append(activeBalls, ball)
		}
	}

	// Apply tag filter if specified
	if filterTags != "" {
		tagList := strings.Split(filterTags, ",")
		// Trim whitespace from each tag
		for i := range tagList {
			tagList[i] = strings.TrimSpace(tagList[i])
		}

		filtered := make([]*session.Ball, 0)
		for _, ball := range activeBalls {
			// Check if ball has any of the specified tags (OR logic)
			hasTag := false
			for _, filterTag := range tagList {
				for _, ballTag := range ball.Tags {
					if ballTag == filterTag {
						hasTag = true
						break
					}
				}
				if hasTag {
					break
				}
			}
			if hasTag {
				filtered = append(filtered, ball)
			}
		}
		activeBalls = filtered
	}

	// Apply priority filter if specified
	if filterPriority != "" {
		if !session.ValidatePriority(filterPriority) {
			return fmt.Errorf("invalid priority: %s (must be low|medium|high|urgent)", filterPriority)
		}

		filtered := make([]*session.Ball, 0)
		for _, ball := range activeBalls {
			if string(ball.Priority) == filterPriority {
				filtered = append(filtered, ball)
			}
		}
		activeBalls = filtered
	}

	if len(activeBalls) == 0 {
		if filterTags != "" || filterPriority != "" {
			fmt.Println("No balls match the specified filters.")
			if filterTags != "" {
				fmt.Printf("  Tags: %s\n", filterTags)
			}
			if filterPriority != "" {
				fmt.Printf("  Priority: %s\n", filterPriority)
			}
		} else {
			fmt.Println("No active balls found.")
			fmt.Println("\nStart a new session with: juggle start")
			fmt.Println("Or plan future work with: juggle plan")
		}
		return nil
	}

	// Show active filters
	if filterTags != "" || filterPriority != "" {
		fmt.Println("Active filters:")
		if filterTags != "" {
			fmt.Printf("  Tags: %s\n", filterTags)
		}
		if filterPriority != "" {
			fmt.Printf("  Priority: %s\n", filterPriority)
		}
		fmt.Println()
	}

	// Group balls by project
	ballsByProject := make(map[string][]*session.Ball)
	for _, ball := range activeBalls {
		ballsByProject[ball.WorkingDir] = append(ballsByProject[ball.WorkingDir], ball)
	}

	// Current working directory already retrieved above for highlighting
	
	// Try to identify current ball (most recently active non-done, non-planned ball in cwd)
	var currentBallID string
	if cwdBalls, ok := ballsByProject[cwd]; ok && len(cwdBalls) > 0 {
		// Filter in-progress balls in current project
		activeBalls := make([]*session.Ball, 0)
		for _, ball := range cwdBalls {
			if ball.State == session.StateInProgress {
				activeBalls = append(activeBalls, ball)
			}
		}
		
		// Get most recently active
		if len(activeBalls) > 0 {
			sort.Slice(activeBalls, func(i, j int) bool {
				return activeBalls[i].LastActivity.After(activeBalls[j].LastActivity)
			})
			currentBallID = activeBalls[0].ID
		}
	}

	// Render grouped by project
	renderGroupedSessions(ballsByProject, cwd, currentBallID)

	return nil
}


func renderGroupedSessions(ballsByProject map[string][]*session.Ball, cwd string, currentBallID string) {
	// Use consistent styles from styles.go
	headerStyle := StyleHeader
	activeStyle := StyleInProgress // In-progress (actively working)
	blockedStyle := StyleBlocked   // Blocked
	plannedStyle := StylePending   // Pending (planned)

	// Get sorted project names
	projectNames := make([]string, 0, len(ballsByProject))
	for projectPath := range ballsByProject {
		projectNames = append(projectNames, projectPath)
	}
	sort.Strings(projectNames)

	// For each project
	for _, projectPath := range projectNames {
		balls := ballsByProject[projectPath]

		// Project header
		projectName := projectPath
		if projectPath == cwd {
			projectName = projectName + " (current)"
		}
		fmt.Printf("\n%s\n", lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")).Render(projectName))

		// Table header
		fmt.Println(
			headerStyle.Render(padRight("ID", 25)) +
			headerStyle.Render(padRight("STATUS", 12)) +
			headerStyle.Render(padRight("PRIORITY", 10)) +
			headerStyle.Render(padRight("CRITERIA", 10)) +
			headerStyle.Render(padRight("INTENT", 40)),
		)

		// Sort balls by status priority: in_progress > blocked > pending
		sort.Slice(balls, func(i, j int) bool {
			stateOrder := map[session.BallState]int{
				session.StateInProgress: 0,
				session.StateBlocked:    1,
				session.StatePending:    2,
			}
			// Sort by state
			if stateOrder[balls[i].State] != stateOrder[balls[j].State] {
				return stateOrder[balls[i].State] < stateOrder[balls[j].State]
			}
			return false
		})

		// Print each ball
		for _, ball := range balls {
			// State with color
			var statusCell string
			var statusStyle lipgloss.Style
			stateStr := string(ball.State)

			switch ball.State {
			case session.StateInProgress:
				statusStyle = activeStyle
			case session.StateBlocked:
				statusStyle = blockedStyle
			case session.StatePending:
				statusStyle = plannedStyle
			default:
				statusStyle = lipgloss.NewStyle()
			}
			
			// Pad first, then style
			statusCell = statusStyle.Render(padRight(stateStr, 12))

			// Priority
			priorityStr := string(ball.Priority)
			priorityCell := GetPriorityStyle(priorityStr).Render(padRight(priorityStr, 10))

			// Acceptance Criteria
			criteriaCell := "-"
			if len(ball.AcceptanceCriteria) > 0 {
				criteriaCell = fmt.Sprintf("%d", len(ball.AcceptanceCriteria))
			}
			criteriaCell = padRight(criteriaCell, 10)

			// Intent (truncated)
			intentCell := truncate(ball.Title, 40)
			intentCell = padRight(intentCell, 40)

			// Highlight current ball with arrow
			ballIDCell := ball.ID
			if ball.ID == currentBallID {
				ballIDCell = "→ " + ball.ID
			}
			ballIDCell = padRight(ballIDCell, 25)

			fmt.Println(
				ballIDCell + " " +
				statusCell + " " +
				priorityCell + " " +
				criteriaCell + " " +
				intentCell,
			)

		}
	}

	fmt.Println()
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return "< 1m"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		hours := int(d.Hours())
		minutes := int(d.Minutes()) % 60
		if minutes == 0 {
			return fmt.Sprintf("%dh", hours)
		}
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	if hours == 0 {
		return fmt.Sprintf("%dd", days)
	}
	return fmt.Sprintf("%dd %dh", days, hours)
}
