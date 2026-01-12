package cli

import (
	"fmt"
	"strings"

	"github.com/ohare93/juggle/internal/session"
	"github.com/spf13/cobra"
)

var unarchiveCmd = &cobra.Command{
	Use:   "unarchive <ball-id>",
	Short: "Restore a completed ball from archive back to ready state",
	Long: `Unarchive takes a completed ball from the archive and restores it to ready state.

The ball will be added back to the active balls list and removed from the archive.

Examples:
  juggle unarchive juggler-5    # Restore ball juggler-5 from archive
  juggle juggler-5 unarchive    # Alternative syntax`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: CompleteArchivedBallIDs,
	RunE:              runUnarchive,
}

func init() {
	rootCmd.AddCommand(unarchiveCmd)
}

func runUnarchive(cmd *cobra.Command, args []string) error {
	ballID := args[0]

	// Find the ball in archives across all projects
	ball, store, err := findArchivedBallByID(ballID)
	if err != nil {
		return err
	}

	// Unarchive ball
	restoredBall, err := store.UnarchiveBall(ball.ID)
	if err != nil {
		return fmt.Errorf("failed to unarchive ball: %w", err)
	}

	// Show success message
	fmt.Printf("✓ Unarchived ball: %s\n", StyleHighlight.Render(restoredBall.ID))
	fmt.Printf("  State: %s\n", StylePending.Render(string(restoredBall.State)))
	fmt.Printf("  Intent: %s\n", restoredBall.Intent)

	return nil
}

// handleBallUnarchive handles the ball-specific unarchive command (juggle <ball-id> unarchive)
func handleBallUnarchive(ball *session.Ball, store *session.Store) error {
	// Check if ball is complete (in archive)
	if ball.State != session.StateComplete {
		return fmt.Errorf("ball %s is not archived (current state: %s)", ball.ID, ball.State)
	}

	// The ball parameter here is from archived balls
	// Call UnarchiveBall to restore it
	restoredBall, err := store.UnarchiveBall(ball.ID)
	if err != nil {
		return fmt.Errorf("failed to unarchive ball: %w", err)
	}

	// Show success message
	fmt.Printf("✓ Unarchived ball: %s\n", StyleHighlight.Render(restoredBall.ID))
	fmt.Printf("  State: %s\n", StylePending.Render(string(restoredBall.State)))
	fmt.Printf("  Intent: %s\n", restoredBall.Intent)

	return nil
}

// findArchivedBallByID searches for a ball by ID in archives (respects --all flag)
func findArchivedBallByID(ballID string) (*session.Ball, *session.Store, error) {
	config, err := LoadConfigForCommand()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load config: %w", err)
	}

	cwd, err := GetWorkingDir()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	store, err := NewStoreForCommand(cwd)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create store: %w", err)
	}

	// Discover projects (respects --all flag)
	projects, err := DiscoverProjectsForCommand(config, store)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to discover projects: %w", err)
	}

	archivedBalls, err := session.LoadArchivedBalls(projects)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load archived balls: %w", err)
	}

	// Use prefix matching
	matches := session.ResolveBallByPrefix(archivedBalls, ballID)
	if len(matches) == 0 {
		return nil, nil, fmt.Errorf("ball not found in archives: %s", ballID)
	}
	if len(matches) > 1 {
		matchingIDs := make([]string, len(matches))
		for i, m := range matches {
			matchingIDs[i] = m.ID
		}
		return nil, nil, fmt.Errorf("ambiguous ID '%s' matches %d archived balls: %s", ballID, len(matches), strings.Join(matchingIDs, ", "))
	}

	ball := matches[0]
	ballStore, err := NewStoreForCommand(ball.WorkingDir)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create store for ball: %w", err)
	}
	return ball, ballStore, nil
}
