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
	updateIntent      string
	updatePriority    string
	updateState       string
	updateCriteria    []string
	updateTags        string
	updateBlockReason string
)

var updateCmd = &cobra.Command{
	Use:   "update <ball-id>",
	Short: "Update a ball's properties",
	Long: `Update properties of a ball including intent, priority, state, acceptance criteria, and tags.

When no flags are provided, enters interactive mode where you can edit all properties.

Examples:
  juggle update my-app-1
  juggle update my-app-1 --intent "New intent"
  juggle update my-app-1 --priority urgent
  juggle update my-app-1 --state in_progress
  juggle update my-app-1 --state blocked --reason "Waiting for API"
  juggle update my-app-1 --criteria "User can log in" --criteria "Session persists"
  juggle update my-app-1 --tags bug-fix,security`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: CompleteBallIDs,
	RunE:              runUpdate,
}

func init() {
	updateCmd.Flags().StringVar(&updateIntent, "intent", "", "Update the ball intent")
	updateCmd.Flags().StringVar(&updatePriority, "priority", "", "Update the priority (low|medium|high|urgent)")
	updateCmd.Flags().StringVar(&updateState, "state", "", "Update the state (pending|in_progress|blocked|complete)")
	updateCmd.Flags().StringArrayVar(&updateCriteria, "criteria", nil, "Set acceptance criteria (can be specified multiple times)")
	updateCmd.Flags().StringVar(&updateTags, "tags", "", "Update tags (comma-separated)")
	updateCmd.Flags().StringVar(&updateBlockReason, "reason", "", "Blocked reason (required when setting state to blocked)")

	// Add completion for flags
	updateCmd.RegisterFlagCompletionFunc("priority", CompletePriorities)
	updateCmd.RegisterFlagCompletionFunc("state", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"pending", "in_progress", "blocked", "complete"}, cobra.ShellCompDirectiveNoFileComp
	})
}

func runUpdate(cmd *cobra.Command, args []string) error {
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
	if updateIntent == "" && updatePriority == "" && updateState == "" && updateCriteria == nil && updateTags == "" {
		return runInteractiveUpdate(foundBall, foundStore)
	}

	// Direct update mode
	modified := false

	if updateIntent != "" {
		foundBall.Intent = updateIntent
		modified = true
		fmt.Printf("✓ Updated intent: %s\n", updateIntent)
	}

	if updatePriority != "" {
		if !session.ValidatePriority(updatePriority) {
			return fmt.Errorf("invalid priority: %s (must be low|medium|high|urgent)", updatePriority)
		}
		foundBall.Priority = session.Priority(updatePriority)
		modified = true
		fmt.Printf("✓ Updated priority: %s\n", updatePriority)
	}

	if updateState != "" {
		// Validate state
		stateMap := map[string]session.BallState{
			"pending":     session.StatePending,
			"in_progress": session.StateInProgress,
			"blocked":     session.StateBlocked,
			"complete":    session.StateComplete,
		}
		newState, ok := stateMap[updateState]
		if !ok {
			return fmt.Errorf("invalid state: %s (must be pending|in_progress|blocked|complete)", updateState)
		}

		// If setting to blocked, require a reason
		if newState == session.StateBlocked {
			if updateBlockReason == "" {
				return fmt.Errorf("blocked reason required: use --reason flag when setting state to blocked")
			}
			foundBall.SetBlocked(updateBlockReason)
			fmt.Printf("✓ Updated state: blocked (reason: %s)\n", updateBlockReason)
		} else {
			foundBall.SetState(newState)
			fmt.Printf("✓ Updated state: %s\n", foundBall.State)
		}
		modified = true
	}

	if updateCriteria != nil {
		foundBall.SetAcceptanceCriteria(updateCriteria)
		modified = true
		fmt.Printf("✓ Updated acceptance criteria (%d items)\n", len(updateCriteria))
	}

	if updateTags != "" {
		tags := strings.Split(updateTags, ",")
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

func runInteractiveUpdate(ball *session.Session, store *session.Store) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("Updating ball: %s\n", ball.ID)
	fmt.Println("(Press Enter to keep current value)")
	fmt.Println()

	// Edit intent
	fmt.Printf("Intent [%s]: ", ball.Intent)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input != "" {
		ball.Intent = input
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

	// Edit state
	fmt.Printf("State [%s] (pending|in_progress|blocked|complete): ", ball.State)
	input, _ = reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input != "" {
		if !session.ValidateBallState(input) {
			return fmt.Errorf("invalid state: %s", input)
		}
		newState := session.BallState(input)
		if newState == session.StateBlocked {
			fmt.Printf("Blocked reason [%s]: ", ball.BlockedReason)
			reason, _ := reader.ReadString('\n')
			reason = strings.TrimSpace(reason)
			if reason != "" {
				ball.SetBlocked(reason)
			} else if ball.BlockedReason == "" {
				return fmt.Errorf("blocked reason required when setting state to blocked")
			} else {
				ball.SetState(newState)
			}
		} else {
			ball.SetState(newState)
		}
	}

	// Edit acceptance criteria
	fmt.Printf("Acceptance Criteria (current: %d items)\n", len(ball.AcceptanceCriteria))
	fmt.Println("Enter new criteria line by line (empty line to finish, '-' to keep existing):")
	var newCriteria []string
	for {
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input == "" {
			break
		}
		if input == "-" {
			newCriteria = ball.AcceptanceCriteria
			break
		}
		newCriteria = append(newCriteria, input)
	}
	if len(newCriteria) > 0 {
		ball.SetAcceptanceCriteria(newCriteria)
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
	fmt.Printf("  Priority: %s\n", ball.Priority)
	fmt.Printf("  State: %s\n", ball.State)
	if ball.BlockedReason != "" {
		fmt.Printf("  Blocked Reason: %s\n", ball.BlockedReason)
	}
	if len(ball.AcceptanceCriteria) > 0 {
		fmt.Printf("  Acceptance Criteria: %d items\n", len(ball.AcceptanceCriteria))
	}
	if len(ball.Tags) > 0 {
		fmt.Printf("  Tags: %s\n", strings.Join(ball.Tags, ", "))
	}

	return nil
}
