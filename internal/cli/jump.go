package cli

import (
	"fmt"

	"github.com/ohare93/juggle/internal/session"
	"github.com/ohare93/juggle/internal/zellij"
	"github.com/spf13/cobra"
)

var jumpCmd = &cobra.Command{
	Use:   "jump <session-id>",
	Short: "Jump to a session's Zellij tab",
	Long: `Switch to the Zellij tab associated with the given session.
Requires Zellij to be running and the session to have Zellij info.`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: CompleteBallIDs,
	RunE:              runJump,
}

func runJump(cmd *cobra.Command, args []string) error {
	if !zellij.IsInstalled() {
		return fmt.Errorf("Zellij is not installed")
	}

	zellijInfo, err := zellij.DetectInfo()
	if err != nil || !zellijInfo.IsActive {
		return fmt.Errorf("not running in Zellij")
	}

	ballID := args[0]

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

	// Search for the ball across all projects
	var foundBall *session.Session
	for _, projectPath := range projects {
		store, err := NewStoreForCommand(projectPath)
		if err != nil {
			continue
		}

		ball, err := store.GetBallByID(ballID)
		if err == nil && ball != nil {
			foundBall = ball
			break
		}
	}

	if foundBall == nil {
		return fmt.Errorf("ball %s not found in any project", ballID)
	}

	if foundBall.ZellijTab == "" {
		return fmt.Errorf("ball has no Zellij tab information")
	}

	if err := zellij.GoToTab(foundBall.ZellijTab); err != nil {
		return fmt.Errorf("failed to switch tabs: %w", err)
	}

	fmt.Printf("✓ Jumped to %s (tab: %s)\n", foundBall.ID, foundBall.ZellijTab)
	return nil
}
