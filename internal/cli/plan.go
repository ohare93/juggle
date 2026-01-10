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

var acceptanceCriteriaFlag []string

func init() {
	planCmd.Flags().StringVarP(&intentFlag, "intent", "i", "", "What are you planning to work on?")
	planCmd.Flags().StringSliceVarP(&acceptanceCriteriaFlag, "ac", "a", []string{}, "Acceptance criteria (can be specified multiple times)")
	planCmd.Flags().StringVarP(&descriptionFlag, "description", "d", "", "DEPRECATED: Use -a/--ac instead. Sets first acceptance criterion.")
	planCmd.Flags().StringVarP(&priorityFlag, "priority", "p", "medium", "Priority: low, medium, high, urgent")
	planCmd.Flags().StringSliceVarP(&tagsFlag, "tags", "t", []string{}, "Tags for categorization")
	planCmd.Flags().StringVarP(&sessionFlag, "session", "s", "", "Session ID to link this ball to (adds session ID as tag)")
	planCmd.Flags().StringVarP(&modelSizeFlag, "model-size", "m", "", "Preferred LLM model size: small, medium, large (blank for default)")
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

	// Get acceptance criteria from: 1) --ac flags, 2) legacy --description flag, 3) prompt
	var acceptanceCriteria []string
	if len(acceptanceCriteriaFlag) > 0 {
		acceptanceCriteria = acceptanceCriteriaFlag
	} else if descriptionFlag != "" {
		// Legacy: treat description as first acceptance criterion
		acceptanceCriteria = []string{descriptionFlag}
	} else {
		// Prompt for acceptance criteria line by line
		reader := bufio.NewReader(os.Stdin)
		fmt.Println("Enter acceptance criteria (one per line, empty line to finish):")
		for {
			fmt.Print("  > ")
			input, err := reader.ReadString('\n')
			if err != nil {
				break
			}
			criterion := strings.TrimSpace(input)
			if criterion == "" {
				break
			}
			acceptanceCriteria = append(acceptanceCriteria, criterion)
		}
	}

	// Create the planned session
	ball, err := session.New(cwd, intent, session.Priority(priority))
	if err != nil {
		return fmt.Errorf("failed to create planned session: %w", err)
	}
	ball.State = session.StatePending

	// Set acceptance criteria if provided
	if len(acceptanceCriteria) > 0 {
		ball.SetAcceptanceCriteria(acceptanceCriteria)
	}

	// Add tags if provided
	for _, tag := range tagsFlag {
		ball.AddTag(tag)
	}

	// Add session ID as tag if --session flag provided
	if sessionFlag != "" {
		ball.AddTag(sessionFlag)
	}

	// Set model size if provided
	if modelSizeFlag != "" {
		modelSize := session.ModelSize(modelSizeFlag)
		if modelSize != session.ModelSizeSmall && modelSize != session.ModelSizeMedium && modelSize != session.ModelSizeLarge {
			return fmt.Errorf("invalid model size %q, must be one of: small, medium, large", modelSizeFlag)
		}
		ball.ModelSize = modelSize
	}

	// Save the ball
	if err := store.AppendBall(ball); err != nil {
		return fmt.Errorf("failed to save planned ball: %w", err)
	}

	// Ensure project is in search paths for discovery
	_ = session.EnsureProjectInSearchPaths(cwd)

	fmt.Printf("✓ Planned ball added: %s\n", ball.ID)
	fmt.Printf("  Intent: %s\n", ball.Intent)
	if len(ball.AcceptanceCriteria) > 0 {
		fmt.Printf("  Acceptance Criteria:\n")
		for i, ac := range ball.AcceptanceCriteria {
			fmt.Printf("    %d. %s\n", i+1, ac)
		}
	}
	fmt.Printf("  Priority: %s\n", ball.Priority)
	if len(ball.Tags) > 0 {
		fmt.Printf("  Tags: %s\n", strings.Join(ball.Tags, ", "))
	}
	fmt.Printf("\nStart working on this ball with: juggle %s in-progress\n", ball.ID)

	return nil
}
