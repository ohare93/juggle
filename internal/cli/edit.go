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
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: CompleteBallIDs,
	RunE:              runEdit,
}

func init() {
	editCmd.Flags().StringVar(&editIntent, "intent", "", "Update the ball intent")
	editCmd.Flags().StringVar(&editDescription, "description", "", "Update the ball description")
	editCmd.Flags().StringVar(&editPriority, "priority", "", "Update the priority (low|medium|high|urgent)")
	editCmd.Flags().StringVar(&editActiveState, "active-state", "", "Update the active state (ready|juggling|dropped|complete)")
	editCmd.Flags().StringVar(&editTags, "tags", "", "Update tags (comma-separated)")

	// Add completion for priority flag
	editCmd.RegisterFlagCompletionFunc("priority", CompletePriorities)
}

func runEdit(cmd *cobra.Command, args []string) error {
	ballID := args[0]

	// Use findBallByID which respects --all flag
	foundBall, foundStore, err := findBallByID(ballID)
	if err != nil {
		return err
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
		foundBall.SetAcceptanceCriteria([]string{editDescription})
		modified = true
		fmt.Printf("✓ Updated acceptance criteria: %s\n", editDescription)
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
		// Validate state - accept both new and legacy names
		stateMap := map[string]session.BallState{
			"pending":     session.StatePending,
			"in_progress": session.StateInProgress,
			"blocked":     session.StateBlocked,
			"complete":    session.StateComplete,
			// Legacy names
			"ready":    session.StatePending,
			"juggling": session.StateInProgress,
			"dropped":  session.StateBlocked,
		}
		newState, ok := stateMap[editActiveState]
		if !ok {
			return fmt.Errorf("invalid state: %s (must be pending|in_progress|blocked|complete)", editActiveState)
		}
		foundBall.SetState(newState)
		modified = true
		fmt.Printf("✓ Updated state: %s\n", foundBall.State)
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

func runInteractiveEdit(ball *session.Ball, store *session.Store) error {
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
		ball.SetState(session.BallState(input))
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
	if len(ball.Tags) > 0 {
		fmt.Printf("  Tags: %s\n", strings.Join(ball.Tags, ", "))
	}

	return nil
}
