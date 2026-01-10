package cli

import (
	"os"

	"github.com/ohare93/juggle/internal/session"
	"github.com/spf13/cobra"
)

var trackActivityCmd = &cobra.Command{
	Use:   "track-activity",
	Short: "Update last activity timestamp for current session",
	Long:  `Update the last activity timestamp. Called by Claude hooks.`,
	RunE:  runTrackActivity,
	Hidden: true, // Hide from help since it's mainly for hooks
}


// GetTrackActivityCmd returns the track-activity command for testing
func GetTrackActivityCmd() *cobra.Command {
	return trackActivityCmd
}

func runTrackActivity(cmd *cobra.Command, args []string) error {
	// Track activity for current directory's session
	cwd, err := session.GetCwd()
	if err != nil {
		// Silently ignore
		return nil
	}

	store, err := NewStoreForCommand(cwd)
	if err != nil {
		// Silently ignore if no .juggler directory
		return nil
	}

	// Get all balls in this project (not just juggling ones)
	allBalls, err := store.LoadBalls()
	if err != nil || len(allBalls) == 0 {
		// Silently ignore if no balls
		return nil
	}

	// Also get juggling balls for fallback resolution (steps 3 & 4)
	jugglingBalls, err := store.GetJugglingBalls()
	if err != nil {
		jugglingBalls = []*session.Session{}
	}

	// Resolution order:
	// 1. JUGGLER_CURRENT_BALL environment variable (explicit override) - search all balls
	// 2. If only one juggling ball, use it
	// 3. Most recently active juggling ball

	var ball *session.Session

	// 1. Check for explicit ball ID override (search all balls)
	if envBallID := os.Getenv("JUGGLER_CURRENT_BALL"); envBallID != "" {
		for _, b := range allBalls {
			if b.ID == envBallID {
				ball = b
				break
			}
		}
		if ball != nil {
			// Found via environment variable
			ball.UpdateActivity()
			ball.IncrementUpdateCount()
			return store.UpdateBall(ball)
		}
		// If env var set but ball not found, fall through to other methods
	}

	// If no juggling balls, silently ignore (nothing to track)
	if len(jugglingBalls) == 0 {
		return nil
	}

	// 2. If only one juggling ball, use it
	if len(jugglingBalls) == 1 {
		ball = jugglingBalls[0]
		ball.UpdateActivity()
		ball.IncrementUpdateCount()
		return store.UpdateBall(ball)
	}

	// 3. Fall back to most recently active juggling ball
	// (jugglingBalls is already sorted by most recent)
	ball = jugglingBalls[0]
	ball.UpdateActivity()
	ball.IncrementUpdateCount()

	return store.UpdateBall(ball)
}
