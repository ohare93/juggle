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
	intentFlag      string
	descriptionFlag string
	priorityFlag    string
	tagsFlag        []string
	ballIDFlag      string
	sessionFlag     string
	modelSizeFlag   string
)

var startCmd = &cobra.Command{
	Use:   "start [intent]",
	Short: "Start tracking a new session",
	Long: `Start tracking a new work session in the current directory with the given intent.
If no intent is provided as an argument, you'll be prompted for it interactively.

To activate a planned session instead, use the --id flag.`,
	RunE: runStart,
}

func init() {
	startCmd.Flags().StringVarP(&intentFlag, "intent", "i", "", "What are you trying to accomplish?")
	startCmd.Flags().StringVarP(&descriptionFlag, "description", "d", "", "Additional context or details")
	startCmd.Flags().StringVarP(&priorityFlag, "priority", "p", "medium", "Priority: low, medium, high, urgent")
	startCmd.Flags().StringSliceVarP(&tagsFlag, "tags", "t", []string{}, "Tags for categorization")
	startCmd.Flags().StringVar(&ballIDFlag, "id", "", "ID of planned ball to activate")
	startCmd.Flags().StringVarP(&sessionFlag, "session", "s", "", "Session ID to link this ball to (adds session ID as tag)")
	startCmd.Flags().StringVarP(&modelSizeFlag, "model-size", "m", "", "Preferred LLM model size: small, medium, large (blank for default)")
}

func runStart(cmd *cobra.Command, args []string) error {
	// Get current working directory
	cwd, err := GetWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// If --id flag is provided, activate a planned ball
	if ballIDFlag != "" {
		store, err := NewStoreForCommand(cwd)
		if err != nil {
			return fmt.Errorf("failed to initialize store: %w", err)
		}

		ball, err := store.GetBallByID(ballIDFlag)
		if err != nil {
			return fmt.Errorf("failed to find ball %s: %w", ballIDFlag, err)
		}

		if ball.ActiveState != session.ActiveReady {
			return fmt.Errorf("ball %s is not in ready state (current state: %s)", ballIDFlag, ball.ActiveState)
		}

		// Transition to juggling
		ball.ActiveState = session.ActiveJuggling
		needsThrown := session.JuggleNeedsThrown
		ball.JuggleState = &needsThrown
		ball.UpdateActivity()

		if err := store.UpdateBall(ball); err != nil {
			return fmt.Errorf("failed to update ball: %w", err)
		}

		// Ensure project is in search paths for discovery
		_ = session.EnsureProjectInSearchPaths(cwd)

		fmt.Printf("✓ Started planned ball: %s\n", ball.ID)
		fmt.Printf("  Intent: %s\n", ball.Intent)
		fmt.Printf("  Priority: %s\n", ball.Priority)

		return nil
	}

	// Create a new ball
	store, err := NewStoreForCommand(cwd)
	if err != nil {
		return fmt.Errorf("failed to initialize store: %w", err)
	}

	// Get intent from: 1) first arg, 2) --intent flag, 3) prompt
	intent := ""
	if len(args) > 0 {
		intent = args[0]
	} else if intentFlag != "" {
		intent = intentFlag
	} else {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("What are you trying to accomplish? ")
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

	// Get description from: 1) --description flag, 2) prompt if not provided
	description := descriptionFlag
	if description == "" {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Add a description for context? (optional, press Enter to skip): ")
		input, err := reader.ReadString('\n')
		if err == nil {
			description = strings.TrimSpace(input)
		}
	}

	// Create the session
	sess, err := session.New(cwd, intent, session.Priority(priority))
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	// Set description if provided
	if description != "" {
		sess.SetDescription(description)
	}

	// Add tags if provided
	for _, tag := range tagsFlag {
		sess.AddTag(tag)
	}

	// Add session ID as tag if --session flag provided
	if sessionFlag != "" {
		sess.AddTag(sessionFlag)
	}

	// Set model size if provided
	if modelSizeFlag != "" {
		modelSize := session.ModelSize(modelSizeFlag)
		if modelSize != session.ModelSizeSmall && modelSize != session.ModelSizeMedium && modelSize != session.ModelSizeLarge {
			return fmt.Errorf("invalid model size %q, must be one of: small, medium, large", modelSizeFlag)
		}
		sess.ModelSize = modelSize
	}

	// Set to juggling/in-air since we're starting work NOW
	sess.ActiveState = session.ActiveJuggling
	inAir := session.JuggleInAir
	sess.JuggleState = &inAir

	// Save the session
	if err := store.AppendBall(sess); err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	// Ensure project is in search paths for discovery
	_ = session.EnsureProjectInSearchPaths(cwd)

	// Export session ID for hooks to use
	fmt.Printf("export JUGGLER_SESSION_ID=%s\n", sess.ID)
	fmt.Printf("\n✓ Started session: %s\n", sess.ID)
	fmt.Printf("  Intent: %s\n", sess.Intent)
	if sess.Description != "" {
		fmt.Printf("  Description: %s\n", sess.Description)
	}
	fmt.Printf("  Priority: %s\n", sess.Priority)
	if len(sess.Tags) > 0 {
		fmt.Printf("  Tags: %s\n", strings.Join(sess.Tags, ", "))
	}

	return nil
}
