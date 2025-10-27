package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/ohare93/juggle/internal/session"
	"github.com/spf13/cobra"
)

var (
	historyTags     string
	historyPriority string
	historyLimit    int
	historyAfter    string
	historyBefore   string
	historySortBy   string
	historyStats    bool
)

var historyCmd = &cobra.Command{
	Use:   "history [query]",
	Short: "Query archived (done) balls",
	Long: `Search and view archived balls that have been marked as done.

By default shows the 20 most recently completed balls. Use flags to filter and search.

Examples:
  juggler history                    # Show 20 most recent completed balls
  juggler history bug                # Search for "bug" in intent
  juggler history --tags feature     # Show completed balls with "feature" tag
  juggler history --priority urgent  # Show completed urgent balls
  juggler history --stats            # Show archive statistics
  juggler history --after 2025-10-01 # Show balls completed after Oct 1, 2025`,
	Args: cobra.MaximumNArgs(1),
	RunE: runHistory,
}

func init() {
	historyCmd.Flags().StringVar(&historyTags, "tags", "", "Filter by tags (comma-separated, OR logic)")
	historyCmd.Flags().StringVar(&historyPriority, "priority", "", "Filter by priority (low|medium|high|urgent)")
	historyCmd.Flags().IntVar(&historyLimit, "limit", 20, "Maximum number of results (0 = no limit)")
	historyCmd.Flags().StringVar(&historyAfter, "after", "", "Show balls completed after date (YYYY-MM-DD)")
	historyCmd.Flags().StringVar(&historyBefore, "before", "", "Show balls completed before date (YYYY-MM-DD)")
	historyCmd.Flags().StringVar(&historySortBy, "sort", "completed-desc", "Sort by: completed-desc|completed-asc|priority")
	historyCmd.Flags().BoolVar(&historyStats, "stats", false, "Show archive statistics instead of listing balls")
}

func runHistory(cmd *cobra.Command, args []string) error {
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

	// Show stats if requested
	if historyStats {
		return showArchiveStats(projects)
	}

	// Build query
	query := session.ArchiveQuery{
		Limit:  historyLimit,
		SortBy: session.ArchiveSortBy(historySortBy),
	}

	if len(args) > 0 {
		query.Query = args[0]
	}

	// Parse tags
	if historyTags != "" {
		tagList := strings.Split(historyTags, ",")
		for i := range tagList {
			tagList[i] = strings.TrimSpace(tagList[i])
		}
		query.Tags = tagList
	}

	// Parse priority
	if historyPriority != "" {
		if !session.ValidatePriority(historyPriority) {
			return fmt.Errorf("invalid priority: %s (must be low|medium|high|urgent)", historyPriority)
		}
		query.Priority = session.Priority(historyPriority)
	}

	// Parse date filters
	if historyAfter != "" {
		t, err := time.Parse("2006-01-02", historyAfter)
		if err != nil {
			return fmt.Errorf("invalid date format for --after (use YYYY-MM-DD): %w", err)
		}
		query.CompletedAfter = &t
	}

	if historyBefore != "" {
		t, err := time.Parse("2006-01-02", historyBefore)
		if err != nil {
			return fmt.Errorf("invalid date format for --before (use YYYY-MM-DD): %w", err)
		}
		// Set to end of day
		endOfDay := t.Add(24*time.Hour - time.Second)
		query.CompletedBefore = &endOfDay
	}

	// Query archive
	balls, err := session.QueryArchive(projects, query)
	if err != nil {
		return fmt.Errorf("failed to query archive: %w", err)
	}

	if len(balls) == 0 {
		fmt.Println("No archived balls found matching criteria.")
		if query.Query != "" {
			fmt.Printf("  Query: \"%s\"\n", query.Query)
		}
		if len(query.Tags) > 0 {
			fmt.Printf("  Tags: %s\n", strings.Join(query.Tags, ", "))
		}
		if query.Priority != "" {
			fmt.Printf("  Priority: %s\n", query.Priority)
		}
		if query.CompletedAfter != nil {
			fmt.Printf("  After: %s\n", query.CompletedAfter.Format("2006-01-02"))
		}
		if query.CompletedBefore != nil {
			fmt.Printf("  Before: %s\n", query.CompletedBefore.Format("2006-01-02"))
		}
		return nil
	}

	// Show results
	fmt.Printf("Found %d archived ball(s)", len(balls))
	if query.Limit > 0 && len(balls) == query.Limit {
		fmt.Printf(" (limited to %d)", query.Limit)
	}
	fmt.Println()

	if query.Query != "" || len(query.Tags) > 0 || query.Priority != "" || query.CompletedAfter != nil || query.CompletedBefore != nil {
		fmt.Println("Filters:")
		if query.Query != "" {
			fmt.Printf("  Query: \"%s\"\n", query.Query)
		}
		if len(query.Tags) > 0 {
			fmt.Printf("  Tags: %s\n", strings.Join(query.Tags, ", "))
		}
		if query.Priority != "" {
			fmt.Printf("  Priority: %s\n", query.Priority)
		}
		if query.CompletedAfter != nil {
			fmt.Printf("  After: %s\n", query.CompletedAfter.Format("2006-01-02"))
		}
		if query.CompletedBefore != nil {
			fmt.Printf("  Before: %s\n", query.CompletedBefore.Format("2006-01-02"))
		}
		fmt.Println()
	}

	renderArchivedBalls(balls)

	return nil
}

func renderArchivedBalls(balls []*session.Session) {
	// Use consistent styles from styles.go
	headerStyle := StyleHeader.Padding(0, 1)
	doneStyle := StyleComplete

	// Table header
	fmt.Println(
		headerStyle.Render(padRight("ID", 25)) +
		headerStyle.Render(padRight("COMPLETED", 12)) +
		headerStyle.Render(padRight("DURATION", 10)) +
		headerStyle.Render(padRight("PRIORITY", 10)) +
		headerStyle.Render(padRight("INTENT", 40)),
	)

	// Print each ball
	for _, ball := range balls {
		// Completed date
		completedCell := "-"
		if ball.CompletedAt != nil {
			completedCell = ball.CompletedAt.Format("2006-01-02")
		}
		completedCell = padRight(completedCell, 12)

		// Duration
		durationCell := "-"
		if ball.CompletedAt != nil {
			duration := ball.CompletedAt.Sub(ball.StartedAt)
			durationCell = formatDuration(duration)
		}
		durationCell = padRight(durationCell, 10)

		// Priority
		priorityCell := padRight(string(ball.Priority), 10)

		// Intent (truncated)
		intentCell := truncate(ball.Intent, 40)
		intentCell = padRight(intentCell, 40)

		fmt.Println(
			doneStyle.Render(padRight(ball.ID, 25)) +
			doneStyle.Render(completedCell) +
			doneStyle.Render(durationCell) +
			doneStyle.Render(priorityCell) +
			doneStyle.Render(intentCell),
		)
	}
}

func showArchiveStats(projects []string) error {
	stats, err := session.GetArchiveStats(projects)
	if err != nil {
		return fmt.Errorf("failed to get archive stats: %w", err)
	}

	fmt.Printf("Archive Statistics\n\n")
	fmt.Printf("Total archived balls: %d\n\n", stats.TotalArchived)

	if stats.TotalArchived == 0 {
		fmt.Println("No archived balls yet.")
		return nil
	}

	// By priority
	fmt.Println("By Priority:")
	priorities := []session.Priority{
		session.PriorityUrgent,
		session.PriorityHigh,
		session.PriorityMedium,
		session.PriorityLow,
	}
	for _, p := range priorities {
		if count := stats.ByPriority[p]; count > 0 {
			fmt.Printf("  %s: %d\n", p, count)
		}
	}

	// By tag (top 10)
	if len(stats.ByTag) > 0 {
		fmt.Println("\nTop Tags:")
		type tagCount struct {
			tag   string
			count int
		}
		tags := make([]tagCount, 0, len(stats.ByTag))
		for tag, count := range stats.ByTag {
			tags = append(tags, tagCount{tag, count})
		}
		// Sort by count desc
		for i := 0; i < len(tags); i++ {
			for j := i + 1; j < len(tags); j++ {
				if tags[j].count > tags[i].count {
					tags[i], tags[j] = tags[j], tags[i]
				}
			}
		}
		// Show top 10
		limit := 10
		if len(tags) < limit {
			limit = len(tags)
		}
		for i := 0; i < limit; i++ {
			fmt.Printf("  %s: %d\n", tags[i].tag, tags[i].count)
		}
	}

	// Duration stats
	fmt.Println("\nDuration Statistics:")
	fmt.Printf("  Total time: %s\n", formatDuration(stats.TotalDuration))
	fmt.Printf("  Average: %s\n", formatDuration(stats.AverageDuration))
	fmt.Printf("  Shortest: %s\n", formatDuration(stats.ShortestDuration))
	fmt.Printf("  Longest: %s\n", formatDuration(stats.LongestDuration))

	return nil
}
