package cli

import (
	"bufio"
	"encoding/json"
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
	updateOutput      string
	updateModelSize   string
	updateJSONFlag    bool
	updateAddDep      []string
	updateRemoveDep   []string
	updateSetDeps     []string
)

var updateCmd = &cobra.Command{
	Use:   "update <ball-id>",
	Short: "Update a ball's properties",
	Long: `Update properties of a ball including intent, priority, state, acceptance criteria, tags, dependencies, and output.

When no flags are provided, enters interactive mode where you can edit all properties.

Examples:
  juggle update my-app-1
  juggle update my-app-1 --intent "New intent"
  juggle update my-app-1 --priority urgent
  juggle update my-app-1 --state in_progress
  juggle update my-app-1 --state blocked --reason "Waiting for API"
  juggle update my-app-1 --state researched --output "Investigation results..."
  juggle update my-app-1 --criteria "User can log in" --criteria "Session persists"
  juggle update my-app-1 --tags bug-fix,security
  juggle update my-app-1 --output "Research findings: ..."
  juggle update my-app-1 --model-size small
  juggle update my-app-1 --add-dep other-ball-5
  juggle update my-app-1 --remove-dep other-ball-3
  juggle update my-app-1 --set-deps ball-1,ball-2`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: CompleteBallIDs,
	RunE:              runUpdate,
}

func init() {
	updateCmd.Flags().StringVar(&updateIntent, "intent", "", "Update the ball intent")
	updateCmd.Flags().StringVar(&updatePriority, "priority", "", "Update the priority (low|medium|high|urgent)")
	updateCmd.Flags().StringVar(&updateState, "state", "", "Update the state (pending|in_progress|blocked|complete|researched)")
	updateCmd.Flags().StringArrayVar(&updateCriteria, "criteria", nil, "Set acceptance criteria (can be specified multiple times)")
	updateCmd.Flags().StringVar(&updateTags, "tags", "", "Update tags (comma-separated)")
	updateCmd.Flags().StringVar(&updateBlockReason, "reason", "", "Blocked reason (required when setting state to blocked)")
	updateCmd.Flags().StringVar(&updateOutput, "output", "", "Set research output/results")
	updateCmd.Flags().StringVar(&updateModelSize, "model-size", "", "Set preferred model size (small|medium|large)")
	updateCmd.Flags().BoolVar(&updateJSONFlag, "json", false, "Output updated ball as JSON")
	updateCmd.Flags().StringSliceVar(&updateAddDep, "add-dep", nil, "Add dependency (ball ID, can be specified multiple times)")
	updateCmd.Flags().StringSliceVar(&updateRemoveDep, "remove-dep", nil, "Remove dependency (ball ID, can be specified multiple times)")
	updateCmd.Flags().StringSliceVar(&updateSetDeps, "set-deps", nil, "Replace all dependencies (comma-separated ball IDs)")

	// Add completion for flags
	updateCmd.RegisterFlagCompletionFunc("priority", CompletePriorities)
	updateCmd.RegisterFlagCompletionFunc("state", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"pending", "in_progress", "blocked", "complete", "researched"}, cobra.ShellCompDirectiveNoFileComp
	})
	updateCmd.RegisterFlagCompletionFunc("model-size", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"small", "medium", "large"}, cobra.ShellCompDirectiveNoFileComp
	})
}

func runUpdate(cmd *cobra.Command, args []string) error {
	ballID := args[0]

	// Use findBallByID which respects --all flag
	foundBall, foundStore, err := findBallByID(ballID)
	if err != nil {
		if updateJSONFlag {
			return printJSONError(err)
		}
		return err
	}

	// If no flags provided (except --json), enter interactive mode
	if updateIntent == "" && updatePriority == "" && updateState == "" && updateCriteria == nil && updateTags == "" && updateOutput == "" && updateModelSize == "" && updateAddDep == nil && updateRemoveDep == nil && updateSetDeps == nil && !updateJSONFlag {
		return runInteractiveUpdate(foundBall, foundStore)
	}

	// Direct update mode
	modified := false

	if updateIntent != "" {
		foundBall.Title = updateIntent
		modified = true
		if !updateJSONFlag {
			fmt.Printf("✓ Updated title: %s\n", updateIntent)
		}
	}

	if updatePriority != "" {
		if !session.ValidatePriority(updatePriority) {
			err := fmt.Errorf("invalid priority: %s (must be low|medium|high|urgent)", updatePriority)
			if updateJSONFlag {
				return printJSONError(err)
			}
			return err
		}
		foundBall.Priority = session.Priority(updatePriority)
		modified = true
		if !updateJSONFlag {
			fmt.Printf("✓ Updated priority: %s\n", updatePriority)
		}
	}

	if updateState != "" {
		// Validate state
		stateMap := map[string]session.BallState{
			"pending":     session.StatePending,
			"in_progress": session.StateInProgress,
			"blocked":     session.StateBlocked,
			"complete":    session.StateComplete,
			"researched":  session.StateResearched,
		}
		newState, ok := stateMap[updateState]
		if !ok {
			err := fmt.Errorf("invalid state: %s (must be pending|in_progress|blocked|complete|researched)", updateState)
			if updateJSONFlag {
				return printJSONError(err)
			}
			return err
		}

		// If setting to blocked, require a reason
		if newState == session.StateBlocked {
			if updateBlockReason == "" {
				err := fmt.Errorf("blocked reason required: use --reason flag when setting state to blocked")
				if updateJSONFlag {
					return printJSONError(err)
				}
				return err
			}
			if err := foundBall.SetBlocked(updateBlockReason); err != nil {
				if updateJSONFlag {
					return printJSONError(err)
				}
				return err
			}
			if !updateJSONFlag {
				fmt.Printf("✓ Updated state: blocked (reason: %s)\n", updateBlockReason)
			}
		} else if newState == session.StateResearched {
			// For researched state, use output if provided, or use existing output
			output := updateOutput
			if output == "" {
				output = foundBall.Output
			}
			foundBall.MarkResearched(output)
			if !updateJSONFlag {
				fmt.Printf("✓ Updated state: researched\n")
			}
		} else {
			if err := foundBall.SetState(newState); err != nil {
				return err
			}
			if !updateJSONFlag {
				fmt.Printf("✓ Updated state: %s\n", foundBall.State)
			}
		}
		modified = true
	}

	if updateCriteria != nil {
		foundBall.SetAcceptanceCriteria(updateCriteria)
		modified = true
		if !updateJSONFlag {
			fmt.Printf("✓ Updated acceptance criteria (%d items)\n", len(updateCriteria))
		}
	}

	if updateTags != "" {
		tags := strings.Split(updateTags, ",")
		// Trim whitespace from each tag
		for i := range tags {
			tags[i] = strings.TrimSpace(tags[i])
		}
		foundBall.Tags = tags
		modified = true
		if !updateJSONFlag {
			fmt.Printf("✓ Updated tags: %s\n", strings.Join(tags, ", "))
		}
	}

	if updateModelSize != "" {
		if !session.ValidateModelSize(updateModelSize) {
			err := fmt.Errorf("invalid model size: %s (must be small|medium|large)", updateModelSize)
			if updateJSONFlag {
				return printJSONError(err)
			}
			return err
		}
		foundBall.SetModelSize(session.ModelSize(updateModelSize))
		modified = true
		if !updateJSONFlag {
			fmt.Printf("✓ Updated model size: %s\n", updateModelSize)
		}
	}

	// Handle output separately (not tied to researched state)
	if updateOutput != "" && updateState != "researched" {
		foundBall.SetOutput(updateOutput)
		modified = true
		if !updateJSONFlag {
			fmt.Printf("✓ Updated output (%d characters)\n", len(updateOutput))
		}
	}

	// Handle dependency modifications
	depsModified := false
	if updateSetDeps != nil {
		// Replace all dependencies
		resolvedDeps, err := resolveDependencyIDsForUpdate(foundStore, updateSetDeps, foundBall.ID)
		if err != nil {
			if updateJSONFlag {
				return printJSONError(err)
			}
			return err
		}
		foundBall.SetDependencies(resolvedDeps)
		depsModified = true
		modified = true
		if !updateJSONFlag {
			fmt.Printf("✓ Set dependencies: %s\n", strings.Join(resolvedDeps, ", "))
		}
	}

	if updateAddDep != nil {
		// Add dependencies
		resolvedDeps, err := resolveDependencyIDsForUpdate(foundStore, updateAddDep, foundBall.ID)
		if err != nil {
			if updateJSONFlag {
				return printJSONError(err)
			}
			return err
		}
		for _, dep := range resolvedDeps {
			foundBall.AddDependency(dep)
		}
		depsModified = true
		modified = true
		if !updateJSONFlag {
			fmt.Printf("✓ Added dependencies: %s\n", strings.Join(resolvedDeps, ", "))
		}
	}

	if updateRemoveDep != nil {
		// Remove dependencies
		resolvedDeps, err := resolveDependencyIDsForUpdate(foundStore, updateRemoveDep, foundBall.ID)
		if err != nil {
			if updateJSONFlag {
				return printJSONError(err)
			}
			return err
		}
		for _, dep := range resolvedDeps {
			if foundBall.RemoveDependency(dep) {
				if !updateJSONFlag {
					fmt.Printf("✓ Removed dependency: %s\n", dep)
				}
			}
		}
		depsModified = true
		modified = true
	}

	// Detect circular dependencies after any dependency modification
	if depsModified {
		balls, err := foundStore.LoadBalls()
		if err != nil {
			if updateJSONFlag {
				return printJSONError(err)
			}
			return fmt.Errorf("failed to load balls for dependency check: %w", err)
		}
		// Replace the ball in the list with the modified version
		for i, b := range balls {
			if b.ID == foundBall.ID {
				balls[i] = foundBall
				break
			}
		}
		if err := session.DetectCircularDependencies(balls); err != nil {
			if updateJSONFlag {
				return printJSONError(err)
			}
			return fmt.Errorf("dependency error: %w", err)
		}
	}

	if modified {
		foundBall.UpdateActivity()
		if err := foundStore.UpdateBall(foundBall); err != nil {
			if updateJSONFlag {
				return printJSONError(err)
			}
			return fmt.Errorf("failed to update ball: %w", err)
		}
		if updateJSONFlag {
			return printBallJSON(foundBall)
		}
		fmt.Printf("\n✓ Ball %s updated successfully\n", ballID)
	} else if updateJSONFlag {
		// Even with no modifications, output the ball in JSON mode
		return printBallJSON(foundBall)
	}

	return nil
}

// printUpdateJSON outputs the updated ball as JSON (uses show.go's helper)
func printUpdateJSON(ball *session.Ball) error {
	data, err := json.MarshalIndent(ball, "", "  ")
	if err != nil {
		return printJSONError(err)
	}
	fmt.Println(string(data))
	return nil
}

func runInteractiveUpdate(ball *session.Ball, store *session.Store) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("Updating ball: %s\n", ball.ID)
	fmt.Println("(Press Enter to keep current value)")
	fmt.Println()

	// Edit intent
	fmt.Printf("Intent [%s]: ", ball.Title)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input != "" {
		ball.Title = input
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
	fmt.Printf("State [%s] (pending|in_progress|blocked|complete|researched): ", ball.State)
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
				if err := ball.SetBlocked(reason); err != nil {
					return err
				}
			} else if ball.BlockedReason == "" {
				return fmt.Errorf("blocked reason required when setting state to blocked")
			} else {
				if err := ball.SetState(newState); err != nil {
					return err
				}
			}
		} else if newState == session.StateResearched {
			fmt.Printf("Research output [%s]: ", truncateForDisplay(ball.Output, 50))
			output, _ := reader.ReadString('\n')
			output = strings.TrimSpace(output)
			if output != "" {
				ball.MarkResearched(output)
			} else if ball.Output != "" {
				ball.MarkResearched(ball.Output)
			} else {
				ball.MarkResearched("")
			}
		} else {
			if err := ball.SetState(newState); err != nil {
				return err
			}
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

	// Edit output
	currentOutput := ball.Output
	if currentOutput == "" {
		currentOutput = "none"
	} else if len(currentOutput) > 50 {
		currentOutput = currentOutput[:50] + "..."
	}
	fmt.Printf("Output [%s] (enter new text, '-' to keep, 'clear' to remove): ", currentOutput)
	input, _ = reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input != "" && input != "-" {
		if input == "clear" {
			ball.SetOutput("")
		} else {
			ball.SetOutput(input)
		}
	}

	// Edit model size
	currentModelSize := string(ball.ModelSize)
	if currentModelSize == "" {
		currentModelSize = "unset"
	}
	fmt.Printf("Model Size [%s] (small|medium|large): ", currentModelSize)
	input, _ = reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input != "" {
		if !session.ValidateModelSize(input) {
			return fmt.Errorf("invalid model size: %s", input)
		}
		ball.SetModelSize(session.ModelSize(input))
	}

	// Save changes
	ball.UpdateActivity()
	if err := store.UpdateBall(ball); err != nil {
		return fmt.Errorf("failed to update ball: %w", err)
	}

	fmt.Printf("\n✓ Ball %s updated successfully\n", ball.ID)
	fmt.Println("\nUpdated values:")
	fmt.Printf("  Title: %s\n", ball.Title)
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
	if ball.Output != "" {
		fmt.Printf("  Output: %d characters\n", len(ball.Output))
	}
	if ball.ModelSize != "" {
		fmt.Printf("  Model Size: %s\n", ball.ModelSize)
	}

	return nil
}

// truncateForDisplay truncates a string to the given length with ellipsis
func truncateForDisplay(s string, maxLen int) string {
	if s == "" {
		return "none"
	}
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// resolveDependencyIDsForUpdate resolves ball IDs (full or short) to full ball IDs
// excludeID is the ID of the ball being updated, to prevent self-dependency
func resolveDependencyIDsForUpdate(store *session.Store, ids []string, excludeID string) ([]string, error) {
	balls, err := store.LoadBalls()
	if err != nil {
		return nil, err
	}

	resolved := make([]string, 0, len(ids))
	for _, id := range ids {
		// Use prefix matching
		matches := session.ResolveBallByPrefix(balls, id)
		if len(matches) == 0 {
			return nil, fmt.Errorf("ball not found: %s", id)
		}
		if len(matches) > 1 {
			matchingIDs := make([]string, len(matches))
			for i, m := range matches {
				matchingIDs[i] = m.ID
			}
			return nil, fmt.Errorf("ambiguous ID '%s' matches %d balls: %s", id, len(matches), strings.Join(matchingIDs, ", "))
		}
		ball := matches[0]
		if ball.ID == excludeID {
			return nil, fmt.Errorf("cannot add self as dependency: %s", id)
		}
		resolved = append(resolved, ball.ID)
	}
	return resolved, nil
}
