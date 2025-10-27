package cli

import (
	"os"

	"github.com/ohare93/juggle/internal/session"
	"github.com/ohare93/juggle/internal/zellij"
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

	// Get all juggling balls in this project
	jugglingBalls, err := store.GetJugglingBalls()
	if err != nil || len(jugglingBalls) == 0 {
		// Silently ignore if no juggling balls
		return nil
	}

	// Resolution order:
	// 1. JUGGLER_CURRENT_BALL environment variable (explicit override)
	// 2. Zellij session+tab matching
	// 3. If only one juggling ball, use it
	// 4. Most recently active juggling ball

	var ball *session.Session

	// 1. Check for explicit ball ID override
	if envBallID := os.Getenv("JUGGLER_CURRENT_BALL"); envBallID != "" {
		for _, b := range jugglingBalls {
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

	// 2. Try Zellij matching if in Zellij
	zellijInfo, err := zellij.DetectInfo()
	if err == nil && zellijInfo.IsActive && zellijInfo.SessionName != "" {
		// Try to match by session+tab
		for _, b := range jugglingBalls {
			if b.ZellijSession == zellijInfo.SessionName {
				// If tab name is available, match on both session and tab
				if zellijInfo.TabName != "" && b.ZellijTab != "" {
					if b.ZellijTab == zellijInfo.TabName {
						ball = b
						break
					}
				} else if b.ZellijTab == "" || zellijInfo.TabName == "" {
					// Match on session only if tab info not available
					ball = b
					break
				}
			}
		}
		if ball != nil {
			// Found via Zellij matching
			ball.UpdateActivity()
			ball.IncrementUpdateCount()
			return store.UpdateBall(ball)
		}
	}

	// 3. If only one juggling ball, use it
	if len(jugglingBalls) == 1 {
		ball = jugglingBalls[0]
		ball.UpdateActivity()
		ball.IncrementUpdateCount()
		return store.UpdateBall(ball)
	}

	// 4. Fall back to most recently active juggling ball
	// (jugglingBalls is already sorted by most recent)
	ball = jugglingBalls[0]
	ball.UpdateActivity()
	ball.IncrementUpdateCount()

	return store.UpdateBall(ball)
}
