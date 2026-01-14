package cli

import (
	"fmt"
	"sort"

	"github.com/ohare93/juggle/internal/session"
	"github.com/spf13/cobra"
)

var nextCmd = &cobra.Command{
	Use:   "next",
	Short: "Determine and jump to the ball that needs attention most",
	Long: `Analyze all in-progress balls and recommend the one that needs attention most.

Priority algorithm:
1. Higher priority balls score higher
2. Balls idle longer score higher

By default, analyzes balls from the current project only. Use --all to search across all discovered projects.

Examples:
  juggle next           # Find next ball in current project
  juggle next --all     # Find next ball across all projects`,
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

	// Discover projects (respects --all flag)
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
	fmt.Printf("  Title: %s\n", nextBall.Title)
	fmt.Printf("  State: %s\n", nextBall.State)
	if nextBall.BlockedReason != "" {
		fmt.Printf("  Blocked: %s\n", nextBall.BlockedReason)
	}
	fmt.Printf("  Priority: %s\n", nextBall.Priority)
	fmt.Printf("  Idle: %s\n", formatDuration(nextBall.IdleDuration()))

	return nil
}

func determineNextSession(sessions []*session.Ball) *session.Ball {
	// Score each session
	type scored struct {
		sess  *session.Ball
		score int
	}

	scoredSessions := make([]scored, 0, len(sessions))

	for _, sess := range sessions {
		s := scored{sess: sess, score: 0}

		// Priority weight (higher priority = higher score)
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
