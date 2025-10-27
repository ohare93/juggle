package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/ohare93/juggle/internal/session"
	"github.com/spf13/cobra"
)

var (
	editIntent      string
	editDescription string
	editPriority    string
	editActiveState string
	editTags        string
)

var editCmd = &cobra.Command{
	Use:   "edit <ball-id>",
	Short: "Edit a ball's properties",
	Long: `Edit properties of a ball including intent, priority, active state, and tags.

When no flags are provided, enters interactive mode where you can edit all properties.

Examples:
  juggle edit my-app-1
  juggle edit my-app-1 --intent "New intent"
  juggle edit my-app-1 --priority urgent
  juggle edit my-app-1 --active-state dropped
  juggle edit my-app-1 --tags bug-fix,security`,
	Args: cobra.ExactArgs(1),
	RunE: runEdit,
}

func init() {
	editCmd.Flags().StringVar(&editIntent, "intent", "", "Update the ball intent")
	editCmd.Flags().StringVar(&editDescription, "description", "", "Update the ball description")
	editCmd.Flags().StringVar(&editPriority, "priority", "", "Update the priority (low|medium|high|urgent)")
	editCmd.Flags().StringVar(&editActiveState, "active-state", "", "Update the active state (ready|juggling|dropped|complete)")
	editCmd.Flags().StringVar(&editTags, "tags", "", "Update tags (comma-separated)")
}

func runEdit(cmd *cobra.Command, args []string) error {
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
	var foundStore *session.Store
	for _, projectPath := range projects {
		store, err := NewStoreForCommand(projectPath)
		if err != nil {
			continue
		}

		ball, err := store.GetBallByID(ballID)
		if err == nil && ball != nil {
			foundBall = ball
			foundStore = store
			break
		}
	}

	if foundBall == nil {
		return fmt.Errorf("ball %s not found in any project", ballID)
	}

	// If no flags provided, enter interactive mode
	if editIntent == "" && editDescription == "" && editPriority == "" && editActiveState == "" && editTags == "" {
		return runInteractiveEdit(foundBall, foundStore)
	}

	// Direct edit mode
	modified := false

	if editIntent != "" {
		foundBall.Intent = editIntent
		modified = true
		fmt.Printf("✓ Updated intent: %s\n", editIntent)
	}

	if editDescription != "" {
		foundBall.SetDescription(editDescription)
		modified = true
		fmt.Printf("✓ Updated description: %s\n", editDescription)
	}

	if editPriority != "" {
		if !session.ValidatePriority(editPriority) {
			return fmt.Errorf("invalid priority: %s (must be low|medium|high|urgent)", editPriority)
		}
		foundBall.Priority = session.Priority(editPriority)
		modified = true
		fmt.Printf("✓ Updated priority: %s\n", editPriority)
	}

	if editActiveState != "" {
		// Validate active state
		validStates := map[string]bool{
			"ready":     true,
			"juggling":  true,
			"dropped":   true,
			"complete":  true,
		}
		if !validStates[editActiveState] {
			return fmt.Errorf("invalid active state: %s (must be ready|juggling|dropped|complete)", editActiveState)
		}
		foundBall.ActiveState = session.ActiveState(editActiveState)
		modified = true
		fmt.Printf("✓ Updated active state: %s\n", editActiveState)
	}

	if editTags != "" {
		tags := strings.Split(editTags, ",")
		// Trim whitespace from each tag
		for i := range tags {
			tags[i] = strings.TrimSpace(tags[i])
		}
		foundBall.Tags = tags
		modified = true
		fmt.Printf("✓ Updated tags: %s\n", strings.Join(tags, ", "))
	}

	if modified {
		foundBall.UpdateActivity()
		if err := foundStore.UpdateBall(foundBall); err != nil {
			return fmt.Errorf("failed to update ball: %w", err)
		}
		fmt.Printf("\n✓ Ball %s updated successfully\n", ballID)
	}

	return nil
}

func runInteractiveEdit(ball *session.Session, store *session.Store) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("Editing ball: %s\n", ball.ID)
	fmt.Println("(Press Enter to keep current value)")
	fmt.Println()

	// Edit intent
	fmt.Printf("Intent [%s]: ", ball.Intent)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input != "" {
		ball.Intent = input
	}

	// Edit description
	currentDesc := ball.Description
	if currentDesc == "" {
		currentDesc = "none"
	}
	fmt.Printf("Description [%s]: ", currentDesc)
	input, _ = reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input != "" {
		ball.SetDescription(input)
	}

	// Edit priority
	fmt.Printf("Priority [%s] (low|medium|high|urgent): ", ball.Priority)
	input, _ = reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input != "" {
		if !session.ValidatePriority(input) {
			return fmt.Errorf("invalid priority: %s", input)
		}
		ball.Priority = session.Priority(input)
	}

	// Edit active state
	fmt.Printf("Active State [%s] (ready|juggling|dropped|complete): ", ball.ActiveState)
	input, _ = reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input != "" {
		validStates := map[string]bool{
			"ready":    true,
			"juggling": true,
			"dropped":  true,
			"complete": true,
		}
		if !validStates[input] {
			return fmt.Errorf("invalid active state: %s", input)
		}
		ball.ActiveState = session.ActiveState(input)
	}

	// Edit tags
	currentTags := strings.Join(ball.Tags, ", ")
	if currentTags == "" {
		currentTags = "none"
	}
	fmt.Printf("Tags [%s] (comma-separated): ", currentTags)
	input, _ = reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input != "" {
		tags := strings.Split(input, ",")
		// Trim whitespace from each tag
		for i := range tags {
			tags[i] = strings.TrimSpace(tags[i])
		}
		ball.Tags = tags
	}

	// Save changes
	ball.UpdateActivity()
	if err := store.UpdateBall(ball); err != nil {
		return fmt.Errorf("failed to update ball: %w", err)
	}

	fmt.Printf("\n✓ Ball %s updated successfully\n", ball.ID)
	fmt.Println("\nUpdated values:")
	fmt.Printf("  Intent: %s\n", ball.Intent)
	if ball.Description != "" {
		fmt.Printf("  Description: %s\n", ball.Description)
	}
	fmt.Printf("  Priority: %s\n", ball.Priority)
	fmt.Printf("  Active State: %s\n", ball.ActiveState)
	if len(ball.Tags) > 0 {
		fmt.Printf("  Tags: %s\n", strings.Join(ball.Tags, ", "))
	}

	return nil
}
