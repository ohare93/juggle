package cli

import (
	"fmt"
	"strings"

	"github.com/ohare93/juggle/internal/session"
	"github.com/spf13/cobra"
)

var (
	tagBallID string
)

var tagCmd = &cobra.Command{
	Use:     "tag",
	Aliases: []string{"tags"},
	Short: "Manage tags for a session",
	Long:  `Add, remove, and list tags for work sessions.`,
}

var tagAddCmd = &cobra.Command{
	Use:   "add <tag> [<tag2> <tag3> ...]",
	Short: "Add one or more tags to the current session",
	Long: `Add tags to a session. You can add multiple tags at once.

Examples:
  juggler tag add bug-fix
  juggler tag add security performance optimization
  juggler tag add --ball my-app-1 urgent`,
	Args: cobra.MinimumNArgs(1),
	RunE: runTagAdd,
}

var tagRmCmd = &cobra.Command{
	Use:   "rm <tag> [<tag2> <tag3> ...]",
	Short: "Remove tags from the current session",
	Long: `Remove one or more tags from a session.

Examples:
  juggler tag rm bug-fix
  juggler tag rm security performance`,
	Args: cobra.MinimumNArgs(1),
	RunE: runTagRm,
}

var tagListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all unique tags across all sessions",
	Long:  `Display all unique tags used across all sessions in all projects.`,
	RunE:  runTagList,
}

func init() {
	// Add --ball flag to tag subcommands
	tagAddCmd.Flags().StringVar(&tagBallID, "ball", "", "Target specific ball by ID")
	tagRmCmd.Flags().StringVar(&tagBallID, "ball", "", "Target specific ball by ID")

	tagCmd.AddCommand(tagAddCmd)
	tagCmd.AddCommand(tagRmCmd)
	tagCmd.AddCommand(tagListCmd)
}

// getCurrentBallForTag finds the appropriate ball to operate on
func getCurrentBallForTag(store *session.Store) (*session.Session, error) {
	// If explicit ball ID provided, use that
	if tagBallID != "" {
		ball, err := store.GetBallByID(tagBallID)
		if err != nil {
			return nil, fmt.Errorf("ball %s not found: %w", tagBallID, err)
		}
		return ball, nil
	}

	// Try to get most recently active juggling ball
	jugglingBalls, err := store.GetJugglingBalls()
	if err != nil || len(jugglingBalls) == 0 {
		return nil, fmt.Errorf("no juggling balls found. Use --ball <id> to specify which ball to tag")
	}
	ball := jugglingBalls[0] // Most recently active

	return ball, nil
}

func runTagAdd(cmd *cobra.Command, args []string) error {
	cwd, err := GetWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	store, err := NewStoreForCommand(cwd)
	if err != nil {
		return fmt.Errorf("failed to initialize store: %w", err)
	}

	ball, err := getCurrentBallForTag(store)
	if err != nil {
		return err
	}

	// Add all tags
	for _, tag := range args {
		ball.AddTag(strings.TrimSpace(tag))
	}

	if err := store.UpdateBall(ball); err != nil {
		return fmt.Errorf("failed to update ball: %w", err)
	}

	count := len(args)
	plural := "tag"
	if count > 1 {
		plural = "tags"
	}

	fmt.Printf("✓ Added %d %s to ball: %s\n", count, plural, ball.ID)
	fmt.Printf("  Current tags: %s\n", strings.Join(ball.Tags, ", "))

	return nil
}

func runTagRm(cmd *cobra.Command, args []string) error {
	cwd, err := GetWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	store, err := NewStoreForCommand(cwd)
	if err != nil {
		return fmt.Errorf("failed to initialize store: %w", err)
	}

	ball, err := getCurrentBallForTag(store)
	if err != nil {
		return err
	}

	// Remove tags
	tagsToRemove := make(map[string]bool)
	for _, tag := range args {
		tagsToRemove[strings.TrimSpace(tag)] = true
	}

	newTags := make([]string, 0)
	for _, tag := range ball.Tags {
		if !tagsToRemove[tag] {
			newTags = append(newTags, tag)
		}
	}

	removed := len(ball.Tags) - len(newTags)
	ball.Tags = newTags

	if err := store.UpdateBall(ball); err != nil {
		return fmt.Errorf("failed to update ball: %w", err)
	}

	plural := "tag"
	if removed != 1 {
		plural = "tags"
	}

	fmt.Printf("✓ Removed %d %s from ball: %s\n", removed, plural, ball.ID)
	if len(ball.Tags) > 0 {
		fmt.Printf("  Remaining tags: %s\n", strings.Join(ball.Tags, ", "))
	} else {
		fmt.Println("  No tags remaining")
	}

	return nil
}

func runTagList(cmd *cobra.Command, args []string) error {
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

	// Load all balls from all projects
	allBalls, err := session.LoadAllBalls(projects)
	if err != nil {
		return fmt.Errorf("failed to load balls: %w", err)
	}

	// Collect all unique tags
	tagCounts := make(map[string]int)
	for _, ball := range allBalls {
		for _, tag := range ball.Tags {
			tagCounts[tag]++
		}
	}

	if len(tagCounts) == 0 {
		fmt.Println("No tags found across all sessions.")
		fmt.Println("\nAdd tags with: juggler tag add <tag>")
		return nil
	}

	// Sort tags alphabetically
	tags := make([]string, 0, len(tagCounts))
	for tag := range tagCounts {
		tags = append(tags, tag)
	}

	// Simple bubble sort
	for i := 0; i < len(tags)-1; i++ {
		for j := i + 1; j < len(tags); j++ {
			if tags[i] > tags[j] {
				tags[i], tags[j] = tags[j], tags[i]
			}
		}
	}

	fmt.Printf("All tags (%d unique):\n\n", len(tags))
	for _, tag := range tags {
		count := tagCounts[tag]
		plural := "session"
		if count != 1 {
			plural = "sessions"
		}
		fmt.Printf("  %s (%d %s)\n", tag, count, plural)
	}

	return nil
}
