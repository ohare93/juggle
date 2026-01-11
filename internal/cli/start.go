package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ohare93/juggle/internal/session"
	"github.com/ohare93/juggle/internal/tui"
	"github.com/ohare93/juggle/internal/watcher"
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
	Use:   "start [session-id | intent]",
	Short: "Start a session or create a new ball",
	Long: `Start a session or create a new work item.

Session-focused mode (recommended):
  juggle start <session-id>    Start work on all pending balls in the session
  juggle start                 Show session selector TUI

Ball-focused mode (legacy):
  juggle start "my intent"     Create and start a new ball with the given intent
  juggle start --id <ball-id>  Activate a specific planned ball

When starting a session:
- All pending balls in the session are set to in_progress
- Session context is displayed
- Progress file is accessible for the session

Examples:
  juggle start feature-auth    Start the feature-auth session
  juggle start                 Select a session interactively
  juggle start "Fix bug #123"  Create a new ball (legacy mode)`,
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

	store, err := NewStoreForCommand(cwd)
	if err != nil {
		return fmt.Errorf("failed to initialize store: %w", err)
	}

	sessionStore, err := session.NewSessionStoreWithConfig(cwd, GetStoreConfig())
	if err != nil {
		return fmt.Errorf("failed to initialize session store: %w", err)
	}

	// If --id flag is provided, activate a planned ball (legacy mode)
	if ballIDFlag != "" {
		return startPlannedBall(store, cwd, ballIDFlag)
	}

	// If argument provided, check if it's a session ID first
	if len(args) > 0 {
		arg := args[0]

		// Try to load as session
		juggleSession, err := sessionStore.LoadSession(arg)
		if err == nil {
			// Found a session - start it
			return startSession(store, sessionStore, cwd, juggleSession)
		}

		// Not a session ID - treat as intent for legacy ball creation
		return createAndStartBall(store, cwd, arg)
	}

	// No arguments - show session selector TUI
	return showSessionSelector(store, sessionStore, cwd)
}

// startPlannedBall activates a planned ball by ID (legacy --id flag)
func startPlannedBall(store *session.Store, cwd, ballID string) error {
	ball, err := store.GetBallByID(ballID)
	if err != nil {
		return fmt.Errorf("failed to find ball %s: %w", ballID, err)
	}

	if ball.State != session.StatePending {
		return fmt.Errorf("ball %s is not in pending state (current state: %s)", ballID, ball.State)
	}

	// Transition to in_progress
	ball.State = session.StateInProgress
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

// startSession starts all pending balls in a session and displays context
func startSession(store *session.Store, sessionStore *session.SessionStore, cwd string, juggleSession *session.JuggleSession) error {
	config, err := LoadConfigForCommand()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get all balls for this session
	projectPaths, err := DiscoverProjectsForCommand(config, store)
	if err != nil {
		return fmt.Errorf("failed to discover projects: %w", err)
	}

	sessionBalls, err := session.LoadBallsBySession(projectPaths, juggleSession.ID)
	if err != nil {
		return fmt.Errorf("failed to load balls for session: %w", err)
	}

	// Start all pending balls in the session
	startedCount := 0
	for _, ball := range sessionBalls {
		if ball.State == session.StatePending {
			ball.State = session.StateInProgress
			ball.UpdateActivity()

			// Get the store for this ball's project directory
			ballStore, err := session.NewStoreWithConfig(ball.WorkingDir, GetStoreConfig())
			if err != nil {
				fmt.Printf("  Warning: failed to update ball %s: %v\n", ball.ID, err)
				continue
			}

			if err := ballStore.UpdateBall(ball); err != nil {
				fmt.Printf("  Warning: failed to update ball %s: %v\n", ball.ID, err)
				continue
			}
			startedCount++
		}
	}

	// Ensure project is in search paths
	_ = session.EnsureProjectInSearchPaths(cwd)

	// Display session info
	fmt.Printf("✓ Started session: %s\n", juggleSession.ID)
	if juggleSession.Description != "" {
		fmt.Printf("  Description: %s\n", juggleSession.Description)
	}
	fmt.Printf("  Balls started: %d\n", startedCount)
	fmt.Printf("  Total balls: %d\n", len(sessionBalls))

	// Display session context if available
	if juggleSession.Context != "" {
		fmt.Printf("\n--- Session Context ---\n%s\n", juggleSession.Context)
	}

	// Show path to progress file
	progressPath := sessionStore.ProjectDir() + "/.juggler/sessions/" + juggleSession.ID + "/progress.txt"
	fmt.Printf("\n  Progress file: %s\n", progressPath)

	// Export session ID for hooks
	fmt.Printf("\nexport JUGGLER_SESSION_ID=%s\n", juggleSession.ID)

	return nil
}

// showSessionSelector launches a TUI to select a session
func showSessionSelector(store *session.Store, sessionStore *session.SessionStore, cwd string) error {
	config, err := LoadConfigForCommand()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	sessions, err := sessionStore.ListSessions()
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}

	if len(sessions) == 0 {
		fmt.Println("No sessions found. Create one with: juggle sessions create <id>")
		fmt.Println("Or start a new ball with: juggle start \"my intent\"")
		return nil
	}

	// Create a file watcher for live updates
	w, err := watcher.New()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}
	defer w.Close()

	// Watch the project directory
	if err := w.WatchProject(cwd); err != nil {
		// Non-fatal: continue without watching if it fails
		w = nil
	} else {
		// Start the watcher
		w.Start()
	}

	// Initialize the TUI in split view mode
	model := tui.InitialSplitModelWithWatcher(store, sessionStore, config, GlobalOpts.LocalOnly, w, "")

	// Run the TUI
	p := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	// Check if a session was selected
	if m, ok := finalModel.(tui.Model); ok {
		if m.SelectedSessionID() != "" {
			// User selected a session - start it
			selectedSession, err := sessionStore.LoadSession(m.SelectedSessionID())
			if err != nil {
				return fmt.Errorf("failed to load selected session: %w", err)
			}
			return startSession(store, sessionStore, cwd, selectedSession)
		}
	}

	return nil
}

// createAndStartBall creates a new ball with the given intent (legacy mode)
func createAndStartBall(store *session.Store, cwd, intent string) error {
	// Validate and get priority
	priority := priorityFlag
	if !session.ValidatePriority(priority) {
		return fmt.Errorf("invalid priority %q, must be one of: low, medium, high, urgent", priority)
	}

	// Get description from flag or prompt
	description := descriptionFlag
	if description == "" {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Add a description for context? (optional, press Enter to skip): ")
		input, err := reader.ReadString('\n')
		if err == nil {
			description = strings.TrimSpace(input)
		}
	}

	// Create the ball
	ball, err := session.NewBall(cwd, intent, session.Priority(priority))
	if err != nil {
		return fmt.Errorf("failed to create ball: %w", err)
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

	// Set to in_progress since we're starting work NOW
	ball.State = session.StateInProgress

	// Save the ball
	if err := store.AppendBall(ball); err != nil {
		return fmt.Errorf("failed to save ball: %w", err)
	}

	// Ensure project is in search paths for discovery
	_ = session.EnsureProjectInSearchPaths(cwd)

	// Export ball ID for hooks to use
	fmt.Printf("export JUGGLER_CURRENT_BALL=%s\n", ball.ID)
	fmt.Printf("\n✓ Started ball: %s\n", ball.ID)
	fmt.Printf("  Intent: %s\n", ball.Intent)
	fmt.Printf("  Priority: %s\n", ball.Priority)
	if len(ball.Tags) > 0 {
		fmt.Printf("  Tags: %s\n", strings.Join(ball.Tags, ", "))
	}

	return nil
}
