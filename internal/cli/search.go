package cli

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/ohare93/juggle/internal/session"
	"github.com/spf13/cobra"
)

var (
	searchTags     string
	searchStatus   string
	searchPriority string
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search for active balls by intent, tags, or other criteria",
	Long: `Search through active balls (excluding done) by matching against:
  - Intent text (case-insensitive)
  - Tags
  - Status
  - Priority

The query string will be matched against ball intents. Use flags for more specific filtering.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runSearch,
}

func init() {
	searchCmd.Flags().StringVar(&searchTags, "tags", "", "Filter by tags (comma-separated, OR logic)")
	searchCmd.Flags().StringVar(&searchStatus, "status", "", "Filter by status (planned|active|blocked|needs-review)")
	searchCmd.Flags().StringVar(&searchPriority, "priority", "", "Filter by priority (low|medium|high|urgent)")
}

func runSearch(cmd *cobra.Command, args []string) error {
	// Load config to discover projects
	config, err := LoadConfigForCommand()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Discover all projects
	projects, err := session.DiscoverProjects(config)
	if err != nil {
		return fmt.Errorf("failed to discover projects: %w", err)
	}

	if len(projects) == 0 {
		fmt.Println("No projects with .juggler directories found.")
		return nil
	}

	// Load all balls from all projects
	allBalls, err := session.LoadAllBalls(projects)
	if err != nil {
		return fmt.Errorf("failed to load balls: %w", err)
	}

	// Filter to non-complete balls
	activeBalls := make([]*session.Session, 0)
	for _, ball := range allBalls {
		if ball.ActiveState != session.ActiveComplete {
			activeBalls = append(activeBalls, ball)
		}
	}

	// Apply query filter if provided
	var query string
	if len(args) > 0 {
		query = strings.ToLower(args[0])
		filtered := make([]*session.Session, 0)
		for _, ball := range activeBalls {
			if strings.Contains(strings.ToLower(ball.Intent), query) {
				filtered = append(filtered, ball)
			}
		}
		activeBalls = filtered
	}

	// Apply tag filter if specified
	if searchTags != "" {
		tagList := strings.Split(searchTags, ",")
		for i := range tagList {
			tagList[i] = strings.TrimSpace(tagList[i])
		}

		filtered := make([]*session.Session, 0)
		for _, ball := range activeBalls {
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

	// Apply active state filter if specified
	if searchStatus != "" {
		validStates := map[string]bool{
			"ready":    true,
			"juggling": true,
			"dropped":  true,
			"complete": true,
		}
		if !validStates[searchStatus] {
			return fmt.Errorf("invalid active state: %s (must be ready|juggling|dropped|complete)", searchStatus)
		}

		filtered := make([]*session.Session, 0)
		for _, ball := range activeBalls {
			if string(ball.ActiveState) == searchStatus {
				filtered = append(filtered, ball)
			}
		}
		activeBalls = filtered
	}

	// Apply priority filter if specified
	if searchPriority != "" {
		if !session.ValidatePriority(searchPriority) {
			return fmt.Errorf("invalid priority: %s (must be low|medium|high|urgent)", searchPriority)
		}

		filtered := make([]*session.Session, 0)
		for _, ball := range activeBalls {
			if string(ball.Priority) == searchPriority {
				filtered = append(filtered, ball)
			}
		}
		activeBalls = filtered
	}

	if len(activeBalls) == 0 {
		fmt.Println("No balls found matching search criteria.")
		if query != "" {
			fmt.Printf("  Query: \"%s\"\n", query)
		}
		if searchTags != "" {
			fmt.Printf("  Tags: %s\n", searchTags)
		}
		if searchStatus != "" {
			fmt.Printf("  Status: %s\n", searchStatus)
		}
		if searchPriority != "" {
			fmt.Printf("  Priority: %s\n", searchPriority)
		}
		return nil
	}

	// Show search criteria
	fmt.Printf("Found %d ball(s)\n", len(activeBalls))
	if query != "" || searchTags != "" || searchStatus != "" || searchPriority != "" {
		fmt.Println("Search criteria:")
		if query != "" {
			fmt.Printf("  Query: \"%s\"\n", query)
		}
		if searchTags != "" {
			fmt.Printf("  Tags: %s\n", searchTags)
		}
		if searchStatus != "" {
			fmt.Printf("  Status: %s\n", searchStatus)
		}
		if searchPriority != "" {
			fmt.Printf("  Priority: %s\n", searchPriority)
		}
		fmt.Println()
	}

	// Display results
	renderSearchResults(activeBalls)

	return nil
}

func renderSearchResults(balls []*session.Session) {
	// Define styles
	headerStyle := StyleHeader.Padding(0, 1)

	// Use consistent styles from styles.go
	activeStyle := StyleInAir        // In-air (actively working)
	blockedStyle := StyleNeedsThrown // Needs-thrown (blocked/waiting)
	reviewStyle := StyleNeedsCaught  // Needs-caught (needs review)
	plannedStyle := StyleReady       // Ready (planned)

	// Table header
	fmt.Println(
		headerStyle.Render(padRight("ID", 25)) +
		headerStyle.Render(padRight("PROJECT", 30)) +
		headerStyle.Render(padRight("STATUS", 12)) +
		headerStyle.Render(padRight("PRIORITY", 10)) +
		headerStyle.Render(padRight("INTENT", 40)),
	)

	// Print each ball
	for _, ball := range balls {
		// Project name (truncated)
		projectName := ball.WorkingDir
		if len(projectName) > 28 {
			projectName = "..." + projectName[len(projectName)-25:]
		}
		projectCell := padRight(projectName, 30)

		// State with color
		var statusCell string
		var statusStyle lipgloss.Style
		stateStr := string(ball.ActiveState)
		if ball.JuggleState != nil {
			stateStr = stateStr + ":" + string(*ball.JuggleState)
		}
		
		switch ball.ActiveState {
		case session.ActiveJuggling:
			if ball.JuggleState != nil && *ball.JuggleState == session.JuggleNeedsCaught {
				statusStyle = reviewStyle
			} else if ball.JuggleState != nil && *ball.JuggleState == session.JuggleNeedsThrown {
				statusStyle = blockedStyle
			} else {
				statusStyle = activeStyle
			}
		case session.ActiveDropped:
			statusStyle = blockedStyle
		case session.ActiveReady:
			statusStyle = plannedStyle
		default:
			statusStyle = lipgloss.NewStyle()
		}
		
		// Pad first, then style
		statusCell = statusStyle.Render(padRight(stateStr, 12))

		// Priority
		priorityStr := string(ball.Priority)
		priorityCell := GetPriorityStyle(priorityStr).Render(padRight(priorityStr, 10))

		// Intent (truncated)
		intentCell := truncate(ball.Intent, 40)
		intentCell = padRight(intentCell, 40)

		fmt.Println(
			padRight(ball.ID, 25) + " " +
			projectCell + " " +
			statusCell + " " +
			priorityCell + " " +
			intentCell,
		)
	}
}
