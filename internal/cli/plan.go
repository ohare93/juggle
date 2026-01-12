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

Non-interactive mode (for headless agents):
  juggle plan "Task intent" --non-interactive              # Uses defaults
  juggle plan "Task" -p high -c "AC1" --non-interactive    # With options

In non-interactive mode:
  - Intent is required (via args or --intent flag)
  - Priority defaults to 'medium' if not specified
  - State is always 'pending' (new balls start in pending state)
  - Tags, session, and acceptance criteria default to empty if not specified

Planned balls can be started later with: juggle <ball-id>`,
	RunE: runPlan,
}

var acceptanceCriteriaFlag []string
var dependsOnFlag []string
var nonInteractiveFlag bool

func init() {
	planCmd.Flags().StringVarP(&intentFlag, "intent", "i", "", "What are you planning to work on?")
	planCmd.Flags().StringSliceVarP(&acceptanceCriteriaFlag, "ac", "c", []string{}, "Acceptance criteria (can be specified multiple times)")
	planCmd.Flags().StringVarP(&descriptionFlag, "description", "d", "", "DEPRECATED: Use -c/--ac instead. Sets first acceptance criterion.")
	planCmd.Flags().StringVarP(&priorityFlag, "priority", "p", "", "Priority: low, medium, high, urgent (default: medium)")
	planCmd.Flags().StringSliceVarP(&tagsFlag, "tags", "t", []string{}, "Tags for categorization")
	planCmd.Flags().StringVarP(&sessionFlag, "session", "s", "", "Session ID to link this ball to (adds session ID as tag)")
	planCmd.Flags().StringVarP(&modelSizeFlag, "model-size", "m", "", "Preferred LLM model size: small, medium, large (blank for default)")
	planCmd.Flags().StringSliceVar(&dependsOnFlag, "depends-on", []string{}, "Ball IDs this ball depends on (can be specified multiple times)")
	planCmd.Flags().BoolVar(&nonInteractiveFlag, "non-interactive", false, "Skip interactive prompts, use defaults for unspecified fields (headless mode)")
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

	reader := bufio.NewReader(os.Stdin)

	// Get intent from: 1) positional args (joined), 2) --intent flag, 3) prompt
	intent := ""
	if len(args) > 0 {
		// Join all arguments to support multi-word intents without quotes
		intent = strings.Join(args, " ")
	} else if intentFlag != "" {
		intent = intentFlag
	} else if nonInteractiveFlag {
		return fmt.Errorf("intent is required in non-interactive mode (use positional args or --intent)")
	} else {
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

	// Get priority from: 1) --priority flag, 2) interactive selection, 3) default in non-interactive mode
	priority := priorityFlag
	if priority == "" {
		if nonInteractiveFlag {
			priority = "medium" // Default priority in non-interactive mode
		} else {
			priority = promptSelection(reader, "Priority", []string{"low", "medium", "high", "urgent"}, 1)
		}
	}
	if !session.ValidatePriority(priority) {
		return fmt.Errorf("invalid priority %q, must be one of: low, medium, high, urgent", priority)
	}

	// Get tags from: 1) --tags flag, 2) interactive prompt, 3) empty in non-interactive mode
	var tags []string
	if len(tagsFlag) > 0 {
		tags = tagsFlag
	} else if !nonInteractiveFlag {
		fmt.Print("Tags (comma-separated, empty for none): ")
		input, err := reader.ReadString('\n')
		if err == nil {
			input = strings.TrimSpace(input)
			if input != "" {
				for _, tag := range strings.Split(input, ",") {
					tag = strings.TrimSpace(tag)
					if tag != "" {
						tags = append(tags, tag)
					}
				}
			}
		}
	}
	// In non-interactive mode with no --tags flag, tags will be empty (no prompt)

	// Get session from: 1) --session flag, 2) interactive prompt, 3) empty in non-interactive mode
	sessionID := sessionFlag
	if sessionID == "" && !nonInteractiveFlag {
		fmt.Print("Session ID (empty for none): ")
		input, err := reader.ReadString('\n')
		if err == nil {
			sessionID = strings.TrimSpace(input)
		}
	}
	// In non-interactive mode with no --session flag, sessionID will be empty (no prompt)

	// Get acceptance criteria from: 1) --ac flags, 2) legacy --description flag, 3) prompt, 4) empty in non-interactive mode
	var acceptanceCriteria []string
	if len(acceptanceCriteriaFlag) > 0 {
		acceptanceCriteria = acceptanceCriteriaFlag
	} else if descriptionFlag != "" {
		// Legacy: treat description as first acceptance criterion
		acceptanceCriteria = []string{descriptionFlag}
	} else if !nonInteractiveFlag {
		// Prompt for acceptance criteria line by line
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
	// In non-interactive mode with no --ac flag, acceptanceCriteria will be empty (no prompt)

	// Create the planned ball
	ball, err := session.NewBall(cwd, intent, session.Priority(priority))
	if err != nil {
		return fmt.Errorf("failed to create planned ball: %w", err)
	}

	// New balls always start in pending state
	ball.State = session.StatePending

	// Set acceptance criteria if provided
	if len(acceptanceCriteria) > 0 {
		ball.SetAcceptanceCriteria(acceptanceCriteria)
	}

	// Add tags if provided
	for _, tag := range tags {
		ball.AddTag(tag)
	}

	// Add session ID as tag if provided
	if sessionID != "" {
		ball.AddTag(sessionID)
	}

	// Set model size if provided
	if modelSizeFlag != "" {
		modelSize := session.ModelSize(modelSizeFlag)
		if modelSize != session.ModelSizeSmall && modelSize != session.ModelSizeMedium && modelSize != session.ModelSizeLarge {
			return fmt.Errorf("invalid model size %q, must be one of: small, medium, large", modelSizeFlag)
		}
		ball.ModelSize = modelSize
	}

	// Set dependencies if provided
	if len(dependsOnFlag) > 0 {
		// Resolve dependency IDs (support both full and short IDs)
		resolvedDeps, err := resolveDependencyIDs(store, dependsOnFlag)
		if err != nil {
			return fmt.Errorf("failed to resolve dependencies: %w", err)
		}
		ball.SetDependencies(resolvedDeps)

		// Detect circular dependencies with this new ball
		balls, err := store.LoadBalls()
		if err != nil {
			return fmt.Errorf("failed to load balls for dependency check: %w", err)
		}
		allBalls := append(balls, ball)
		if err := session.DetectCircularDependencies(allBalls); err != nil {
			return fmt.Errorf("dependency error: %w", err)
		}
	}

	// Save the ball
	if err := store.AppendBall(ball); err != nil {
		return fmt.Errorf("failed to save planned ball: %w", err)
	}

	// Ensure project is in search paths for discovery
	_ = session.EnsureProjectInSearchPaths(cwd)

	fmt.Printf("✓ Planned ball added: %s\n", ball.ID)
	fmt.Printf("  Intent: %s\n", ball.Intent)
	fmt.Printf("  Priority: %s\n", ball.Priority)
	fmt.Printf("  State: %s\n", ball.State)
	if len(ball.Tags) > 0 {
		fmt.Printf("  Tags: %s\n", strings.Join(ball.Tags, ", "))
	}
	if ball.State == session.StatePending {
		fmt.Printf("\nStart working on this ball with: juggle %s in-progress\n", ball.ID)
	}

	return nil
}

// promptSelection prompts user to select from options using numbers
func promptSelection(reader *bufio.Reader, label string, options []string, defaultIdx int) string {
	fmt.Printf("%s:\n", label)
	for i, opt := range options {
		marker := " "
		if i == defaultIdx {
			marker = "*"
		}
		fmt.Printf("  %s %d. %s\n", marker, i+1, opt)
	}
	fmt.Printf("Enter number (default %d): ", defaultIdx+1)

	input, err := reader.ReadString('\n')
	if err != nil || strings.TrimSpace(input) == "" {
		return options[defaultIdx]
	}

	var idx int
	_, err = fmt.Sscanf(strings.TrimSpace(input), "%d", &idx)
	if err != nil || idx < 1 || idx > len(options) {
		return options[defaultIdx]
	}

	return options[idx-1]
}

// resolveDependencyIDs resolves ball IDs (full or short) to full ball IDs
func resolveDependencyIDs(store *session.Store, ids []string) ([]string, error) {
	balls, err := store.LoadBalls()
	if err != nil {
		return nil, err
	}

	resolved := make([]string, 0, len(ids))
	for _, id := range ids {
		found := false
		for _, ball := range balls {
			// Match full ID or short ID
			if ball.ID == id || ball.ShortID() == id || strings.HasPrefix(ball.ID, id) {
				resolved = append(resolved, ball.ID)
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("ball not found: %s", id)
		}
	}
	return resolved, nil
}
