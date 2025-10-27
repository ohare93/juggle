package cli

import (
	"fmt"
	"sort"

	"github.com/ohare93/juggle/internal/session"
	"github.com/ohare93/juggle/internal/zellij"
	"github.com/spf13/cobra"
)

var nextCmd = &cobra.Command{
	Use:   "next",
	Short: "Determine and jump to the ball that needs attention most",
	Long: `Analyze all juggling balls and jump to the one that needs attention most.

Priority algorithm:
1. Balls that need catching (agent finished work, needs verification)
2. Balls that need throwing (agent needs direction/input)
3. Balls in-air but idle longest
4. Higher priority balls

By default, analyzes balls from all discovered projects. Use --local to restrict to current project only.

If running in Zellij, will automatically switch to the ball's tab.

Examples:
  juggle next           # Find next ball across all projects
  juggle next --local   # Find next ball in current project only`,
	RunE: runNext,
}

func runNext(cmd *cobra.Command, args []string) error {
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

	// Discover projects (respects --local flag)
	projects, err := DiscoverProjectsForCommand(config, store)
	if err != nil {
		return fmt.Errorf("failed to discover projects: %w", err)
	}

	// Load all juggling balls
	jugglingBalls, err := session.LoadJugglingBalls(projects)
	if err != nil {
		return fmt.Errorf("failed to load juggling balls: %w", err)
	}

	if len(jugglingBalls) == 0 {
		return fmt.Errorf("no balls currently being juggled")
	}

	nextBall := determineNextSession(jugglingBalls)

	fmt.Printf("→ Next ball: %s\n", nextBall.ID)
	fmt.Printf("  Project: %s\n", nextBall.WorkingDir)
	fmt.Printf("  Intent: %s\n", nextBall.Intent)
	if nextBall.JuggleState != nil {
		fmt.Printf("  Juggle State: %s\n", *nextBall.JuggleState)
	}
	if nextBall.StateMessage != "" {
		fmt.Printf("  Message: %s\n", nextBall.StateMessage)
	}
	fmt.Printf("  Priority: %s\n", nextBall.Priority)
	fmt.Printf("  Idle: %s\n", formatDuration(nextBall.IdleDuration()))

	// Try to jump if in Zellij
	zellijInfo, err := zellij.DetectInfo()
	if err == nil && zellijInfo.IsActive && nextBall.ZellijTab != "" {
		if err := zellij.GoToTab(nextBall.ZellijTab); err != nil {
			fmt.Printf("\nNote: Could not switch to tab: %v\n", err)
		} else {
			fmt.Printf("\n✓ Jumped to tab: %s\n", nextBall.ZellijTab)
		}
	}

	return nil
}

func determineNextSession(sessions []*session.Session) *session.Session {
	// Score each session
	type scored struct {
		sess  *session.Session
		score int
	}

	scoredSessions := make([]scored, 0, len(sessions))

	for _, sess := range sessions {
		s := scored{sess: sess, score: 0}

		// Priority based on juggle state
		if sess.JuggleState != nil {
			switch *sess.JuggleState {
			case session.JuggleNeedsCaught:
				// Highest priority: agent finished work, needs verification
				s.score += 1000
			case session.JuggleNeedsThrown:
				// Medium priority: agent needs direction/input
				s.score += 500
			case session.JuggleInAir:
				// Lower priority: agent is working
				s.score += 100
			}
		}

		// Priority weight
		s.score += sess.PriorityWeight() * 10

		// Idle time (older = higher score, max 100 points)
		idleHours := int(sess.IdleDuration().Hours())
		idleScore := idleHours * 2
		if idleScore > 100 {
			idleScore = 100
		}
		s.score += idleScore

		scoredSessions = append(scoredSessions, s)
	}

	// Sort by score descending
	sort.Slice(scoredSessions, func(i, j int) bool {
		return scoredSessions[i].score > scoredSessions[j].score
	})

	return scoredSessions[0].sess
}
