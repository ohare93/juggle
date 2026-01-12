package cli

import (
	"os"
	"strings"

	"github.com/ohare93/juggle/internal/session"
	"github.com/spf13/cobra"
)

// CompleteBallIDs provides completion suggestions for ball IDs
// Includes balls from all discovered projects
func CompleteBallIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Get config and discover projects
	config, err := LoadConfigForCommand()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	// Get current directory for the store (needed for DiscoverProjectsForCommand)
	cwd, err := GetWorkingDir()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	store, err := NewStoreForCommand(cwd)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	projects, err := DiscoverProjectsForCommand(config, store)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	// Load all balls from all projects
	balls, err := session.LoadAllBalls(projects)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	// Use prefix matching and show minimal unique IDs as completions
	minimalIDs := session.ComputeMinimalUniqueIDs(balls)
	matches := session.ResolveBallByPrefix(balls, toComplete)

	var completions []string
	for _, ball := range matches {
		// Show minimal unique ID with intent as description
		minID := minimalIDs[ball.ID]
		completions = append(completions, minID+"\t"+ball.Title)
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}

// CompleteArchivedBallIDs provides completion for archived ball IDs
func CompleteArchivedBallIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Get config and discover projects
	config, err := LoadConfigForCommand()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	cwd, err := GetWorkingDir()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	store, err := NewStoreForCommand(cwd)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	projects, err := DiscoverProjectsForCommand(config, store)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	// Load archived balls
	archived, err := session.LoadArchivedBalls(projects)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	// Use prefix matching and show minimal unique IDs as completions
	minimalIDs := session.ComputeMinimalUniqueIDs(archived)
	matches := session.ResolveBallByPrefix(archived, toComplete)

	var completions []string
	for _, ball := range matches {
		// Show minimal unique ID with intent as description
		minID := minimalIDs[ball.ID]
		completions = append(completions, minID+"\t"+ball.Title)
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}

// CompletePriorities provides completion for priority values
func CompletePriorities(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	priorities := []string{"low", "medium", "high", "urgent"}
	return priorities, cobra.ShellCompDirectiveNoFileComp
}

// CompleteTags provides completion for existing tags
func CompleteTags(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	// Get store
	store, err := NewStoreForCommand(cwd)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	// Load balls
	balls, err := store.LoadBalls()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	// Collect unique tags
	tagSet := make(map[string]bool)
	for _, ball := range balls {
		for _, tag := range ball.Tags {
			tagSet[tag] = true
		}
	}

	// Convert to slice and filter by prefix
	var completions []string
	for tag := range tagSet {
		if strings.HasPrefix(tag, toComplete) {
			completions = append(completions, tag)
		}
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}
