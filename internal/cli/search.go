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
	searchState    string
	searchPriority string
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search for active balls by intent, tags, or other criteria",
	Long: `Search through active balls (excluding complete) by matching against:
  - Intent text (case-insensitive)
  - Tags
  - State
  - Priority

The query string will be matched against ball intents. Use flags for more specific filtering.

By default, searches the current project only. Use --all to search across all discovered projects.

Examples:
  juggle search bug                    # Search current project for "bug"
  juggle search --all feature          # Search all projects for "feature"
  juggle search --tags backend         # Search by tags
  juggle search --state blocked        # Search by state
  juggle search --priority high        # Search by priority`,
	Args: cobra.MaximumNArgs(1),
	RunE: runSearch,
}

func init() {
	searchCmd.Flags().StringVar(&searchTags, "tags", "", "Filter by tags (comma-separated, OR logic)")
	searchCmd.Flags().StringVar(&searchState, "state", "", "Filter by state (pending|in_progress|blocked|complete)")
	searchCmd.Flags().StringVar(&searchPriority, "priority", "", "Filter by priority (low|medium|high|urgent)")
}

func runSearch(cmd *cobra.Command, args []string) error {
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
		return nil
	}

	// Load all balls from discovered projects
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

	// Apply query filter if provided
	var query string
	if len(args) > 0 {
		query = strings.ToLower(args[0])
		filtered := make([]*session.Ball, 0)
		for _, ball := range activeBalls {
			if strings.Contains(strings.ToLower(ball.Title), query) {
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

		filtered := make([]*session.Ball, 0)
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

	// Apply state filter if specified
	if searchState != "" {
		if !session.ValidateBallState(searchState) {
			return fmt.Errorf("invalid state: %s (must be pending|in_progress|blocked|complete)", searchState)
		}

		filtered := make([]*session.Ball, 0)
		for _, ball := range activeBalls {
			if string(ball.State) == searchState {
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

		filtered := make([]*session.Ball, 0)
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
		if searchState != "" {
			fmt.Printf("  State: %s\n", searchState)
		}
		if searchPriority != "" {
			fmt.Printf("  Priority: %s\n", searchPriority)
		}
		return nil
	}

	// Show search criteria
	fmt.Printf("Found %d ball(s)\n", len(activeBalls))
	if query != "" || searchTags != "" || searchState != "" || searchPriority != "" {
		fmt.Println("Search criteria:")
		if query != "" {
			fmt.Printf("  Query: \"%s\"\n", query)
		}
		if searchTags != "" {
			fmt.Printf("  Tags: %s\n", searchTags)
		}
		if searchState != "" {
			fmt.Printf("  State: %s\n", searchState)
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

func renderSearchResults(balls []*session.Ball) {
	// Define styles
	headerStyle := StyleHeader.Padding(0, 1)

	// Use consistent styles from styles.go
	activeStyle := StyleInProgress // In-progress (actively working)
	blockedStyle := StyleBlocked   // Blocked
	plannedStyle := StylePending   // Pending (planned)

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

		// Intent (truncated)
		intentCell := truncate(ball.Title, 40)
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
