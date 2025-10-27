package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/ohare93/juggle/internal/session"
	"github.com/spf13/cobra"
)

var planCmd = &cobra.Command{
	Use:   "plan [intent...]",
	Short: "Add a planned ball for future work",
	Long: `Add a planned ball to track future work you intend to do.

The intent can be provided as a positional argument (with or without quotes),
via the --intent flag, or interactively if neither is specified.

Examples:
  juggle plan "Fix the help text"        # With quotes
  juggle plan Fix the help text          # Without quotes (words joined)
  juggle plan --intent "Add new feature" # Using flag

Planned balls can be started later with: juggle <ball-id>`,
	RunE: runPlan,
}

func init() {
	planCmd.Flags().StringVarP(&intentFlag, "intent", "i", "", "What are you planning to work on?")
	planCmd.Flags().StringVarP(&descriptionFlag, "description", "d", "", "Additional context or details")
	planCmd.Flags().StringVarP(&priorityFlag, "priority", "p", "medium", "Priority: low, medium, high, urgent")
	planCmd.Flags().StringSliceVarP(&tagsFlag, "tags", "t", []string{}, "Tags for categorization")
}

func runPlan(cmd *cobra.Command, args []string) error {
	cwd, err := GetWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	store, err := NewStoreForCommand(cwd)
	if err != nil {
		return fmt.Errorf("failed to initialize store: %w", err)
	}

	// Get intent from: 1) positional args (joined), 2) --intent flag, 3) prompt
	intent := ""
	if len(args) > 0 {
		// Join all arguments to support multi-word intents without quotes
		intent = strings.Join(args, " ")
	} else if intentFlag != "" {
		intent = intentFlag
	} else {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("What do you plan to work on? ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}
		intent = strings.TrimSpace(input)
	}

	if intent == "" {
		return fmt.Errorf("intent is required")
	}

	// Validate and get priority
	priority := priorityFlag
	if !session.ValidatePriority(priority) {
		return fmt.Errorf("invalid priority %q, must be one of: low, medium, high, urgent", priority)
	}

	// Create the planned session
	ball, err := session.New(cwd, intent, session.Priority(priority))
	if err != nil {
		return fmt.Errorf("failed to create planned session: %w", err)
	}
	ball.ActiveState = session.ActiveReady

	// Set description if provided
	if descriptionFlag != "" {
		ball.SetDescription(descriptionFlag)
	}

	// Add tags if provided
	for _, tag := range tagsFlag {
		ball.AddTag(tag)
	}

	// Save the ball
	if err := store.AppendBall(ball); err != nil {
		return fmt.Errorf("failed to save planned ball: %w", err)
	}

	// Ensure project is in search paths for discovery
	_ = session.EnsureProjectInSearchPaths(cwd)

	fmt.Printf("✓ Planned ball added: %s\n", ball.ID)
	fmt.Printf("  Intent: %s\n", ball.Intent)
	if ball.Description != "" {
		fmt.Printf("  Description: %s\n", ball.Description)
	}
	fmt.Printf("  Priority: %s\n", ball.Priority)
	if len(ball.Tags) > 0 {
		fmt.Printf("  Tags: %s\n", strings.Join(ball.Tags, ", "))
	}
	fmt.Printf("\nStart working on this ball with: juggle %s in-air\n", ball.ID)

	return nil
}
