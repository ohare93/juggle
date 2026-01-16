package cli

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/ohare93/juggle/internal/session"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var sessionsCmd = &cobra.Command{
	Use:     "sessions",
	Aliases: []string{"session"},
	Short:   "Manage juggle sessions (ball groupings by tag)",
	Long: `Manage juggle sessions that group balls by tag.

Sessions provide context and progress tracking for related balls.
A session's ID serves as the tag that links balls to that session.

Commands:
  sessions create <id> [-m description]  Create a new session
  sessions list                          List all sessions
  sessions show <id>                     Show session details
  sessions edit <id>                     Edit session properties (opens in editor)
  sessions context <id> [--edit]         View or edit session context
  sessions progress <id>                 View session progress log
  sessions progress clear <id>           Clear session progress log
  sessions delete <id>                   Delete a session

Alias: 'session' can be used instead of 'sessions'`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var (
	sessionDescriptionFlag      string
	sessionContextFlag          string
	sessionEditFlag             bool
	sessionSetFlag              string
	sessionACFlag               []string // Acceptance criteria for session
	sessionYesFlag              bool     // Skip confirmation for delete
	sessionNonInteractiveFlag   bool     // Skip interactive prompts
)

var sessionsCreateCmd = &cobra.Command{
	Use:   "create <id>",
	Short: "Create a new session",
	Long: `Create a new session with the given ID.

The session ID will also be used as a tag to link balls to this session.
Sessions are stored in .juggle/sessions/<id>/session.json with a
corresponding progress.txt file for agent memory.`,
	Args: cobra.ExactArgs(1),
	RunE: runSessionsCreate,
}

var sessionsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all sessions",
	RunE:  runSessionsList,
}

var sessionsShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show session details",
	Args:  cobra.ExactArgs(1),
	RunE:  runSessionsShow,
}

var sessionsContextCmd = &cobra.Command{
	Use:   "context <id>",
	Short: "View or edit session context",
	Long: `View or edit the context for a session.

Without flags, displays the current context.
With --edit, opens the context in $EDITOR for editing.
With --set "text", sets the context directly (agent-friendly).`,
	Args: cobra.ExactArgs(1),
	RunE: runSessionsContext,
}

var sessionsDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a session",
	Long: `Delete a session and its associated files.

This removes the session directory including session.json and progress.txt.
Balls tagged with this session ID are not affected.

Use --yes (-y) to skip the confirmation prompt (for headless/automated use).`,
	Args: cobra.ExactArgs(1),
	RunE: runSessionsDelete,
}

var sessionsProgressCmd = &cobra.Command{
	Use:   "progress <id>",
	Short: "View session progress log",
	Long: `View the progress log (progress.txt) for a session.

Shows timestamped entries that track the session's history and agent activity.`,
	Args: cobra.ExactArgs(1),
	RunE: runSessionsProgress,
}

var sessionsProgressClearCmd = &cobra.Command{
	Use:   "clear <id>",
	Short: "Clear session progress log",
	Long: `Clear the progress log (progress.txt) for a session.

This truncates the progress file to empty, removing all logged history.
Use --yes (-y) to skip the confirmation prompt (for headless/automated use).`,
	Args: cobra.ExactArgs(1),
	RunE: runSessionsProgressClear,
}

var sessionProgressClearYesFlag bool

var sessionsEditCmd = &cobra.Command{
	Use:   "edit <id>",
	Short: "Edit a session's properties",
	Long: `Edit properties of a session including description, context, acceptance criteria, and default model.

Without flags, opens the session in $EDITOR for editing.
With flags, updates the specified properties directly.

Examples:
  juggle sessions edit my-session                    # Open in editor
  juggle sessions edit my-session -m "New description"
  juggle sessions edit my-session --ac "AC1" --ac "AC2"
  juggle sessions edit my-session --default-model medium`,
	Args: cobra.ExactArgs(1),
	RunE: runSessionsEdit,
}

// Edit command flags (separate from create flags to avoid conflicts)
var (
	sessionEditDescriptionFlag   string
	sessionEditContextSetFlag    string
	sessionEditACFlag            []string
	sessionEditDefaultModelFlag  string
	sessionEditACAppendFlag      []string
	sessionEditACRemoveFlag      []string
)

func init() {
	// Add flags for create command
	sessionsCreateCmd.Flags().StringVarP(&sessionDescriptionFlag, "message", "m", "", "Session description")
	sessionsCreateCmd.Flags().StringVar(&sessionContextFlag, "context", "", "Initial session context (agent-friendly)")
	sessionsCreateCmd.Flags().StringSliceVar(&sessionACFlag, "ac", []string{}, "Session-level acceptance criteria (can be specified multiple times)")
	sessionsCreateCmd.Flags().BoolVar(&sessionNonInteractiveFlag, "non-interactive", false, "Skip interactive prompts (for headless mode)")
	sessionsContextCmd.Flags().BoolVar(&sessionEditFlag, "edit", false, "Open context in $EDITOR")
	sessionsContextCmd.Flags().StringVar(&sessionSetFlag, "set", "", "Set context directly (agent-friendly)")
	sessionsDeleteCmd.Flags().BoolVarP(&sessionYesFlag, "yes", "y", false, "Skip confirmation prompt (for headless mode)")
	sessionsProgressClearCmd.Flags().BoolVarP(&sessionProgressClearYesFlag, "yes", "y", false, "Skip confirmation prompt (for headless mode)")

	// Add flags for edit command
	sessionsEditCmd.Flags().StringVarP(&sessionEditDescriptionFlag, "message", "m", "", "Update session description")
	sessionsEditCmd.Flags().StringVar(&sessionEditContextSetFlag, "context", "", "Set session context directly")
	sessionsEditCmd.Flags().StringSliceVar(&sessionEditACFlag, "ac", []string{}, "Replace acceptance criteria (can be specified multiple times)")
	sessionsEditCmd.Flags().StringSliceVar(&sessionEditACAppendFlag, "ac-append", []string{}, "Append acceptance criteria (can be specified multiple times)")
	sessionsEditCmd.Flags().StringSliceVar(&sessionEditACRemoveFlag, "ac-remove", []string{}, "Remove acceptance criteria by text (can be specified multiple times)")
	sessionsEditCmd.Flags().StringVar(&sessionEditDefaultModelFlag, "default-model", "", "Set default model size (small|medium|large)")

	// Add subcommands
	sessionsCmd.AddCommand(sessionsCreateCmd)
	sessionsCmd.AddCommand(sessionsListCmd)
	sessionsCmd.AddCommand(sessionsShowCmd)
	sessionsCmd.AddCommand(sessionsContextCmd)
	sessionsCmd.AddCommand(sessionsDeleteCmd)
	sessionsCmd.AddCommand(sessionsProgressCmd)
	sessionsCmd.AddCommand(sessionsEditCmd)

	// Add progress subcommands
	sessionsProgressCmd.AddCommand(sessionsProgressClearCmd)
}

func runSessionsCreate(cmd *cobra.Command, args []string) error {
	id := args[0]
	description := sessionDescriptionFlag

	cwd, err := GetWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	store, err := session.NewSessionStoreWithConfig(cwd, GetStoreConfig())
	if err != nil {
		return fmt.Errorf("failed to initialize session store: %w", err)
	}

	sess, err := store.CreateSession(id, description)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	// Set context if provided
	if sessionContextFlag != "" {
		if err := store.UpdateSessionContext(id, sessionContextFlag); err != nil {
			return fmt.Errorf("failed to set context: %w", err)
		}
	}

	// Get repo-level defaults for reference
	repoACs, _ := session.GetProjectAcceptanceCriteria(cwd)
	inheritedCount := len(repoACs)

	// Set acceptance criteria if provided via flag
	var acceptanceCriteria []string
	if len(sessionACFlag) > 0 {
		acceptanceCriteria = sessionACFlag
	} else if !sessionNonInteractiveFlag && term.IsTerminal(int(os.Stdin.Fd())) {
		// Interactive mode: ask if user wants to add ACs (only in TTY)
		if inheritedCount > 0 {
			fmt.Printf("This session will inherit %d acceptance criteria from repo defaults.\n", inheritedCount)
		}
		confirmed, err := ConfirmSingleKey("Would you like to add acceptance criteria?")
		if err != nil {
			return fmt.Errorf("failed to read confirmation: %w", err)
		}
		if confirmed {
			// Prompt for acceptance criteria line by line (same UX as ball creation)
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
	}

	// Apply acceptance criteria: provided ACs take precedence, otherwise inherit defaults
	if len(acceptanceCriteria) > 0 {
		if err := store.UpdateSessionAcceptanceCriteria(id, acceptanceCriteria); err != nil {
			return fmt.Errorf("failed to set acceptance criteria: %w", err)
		}
	} else if inheritedCount > 0 {
		// No ACs provided interactively, inherit from repo-level defaults
		if err := store.UpdateSessionAcceptanceCriteria(id, repoACs); err != nil {
			return fmt.Errorf("failed to set default acceptance criteria: %w", err)
		}
	}

	fmt.Printf("Created session: %s\n", sess.ID)
	if description != "" {
		fmt.Printf("  Description: %s\n", description)
	}
	if sessionContextFlag != "" {
		fmt.Printf("  Context: (set)\n")
	}
	if len(acceptanceCriteria) > 0 {
		fmt.Printf("  Acceptance criteria: %d item(s)\n", len(acceptanceCriteria))
	} else if inheritedCount > 0 {
		fmt.Printf("  Acceptance criteria: (inherited %d from repo defaults)\n", inheritedCount)
	}
	fmt.Printf("  Path: .juggle/sessions/%s/\n", id)

	return nil
}

func runSessionsList(cmd *cobra.Command, args []string) error {
	cwd, err := GetWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	store, err := session.NewSessionStoreWithConfig(cwd, GetStoreConfig())
	if err != nil {
		return fmt.Errorf("failed to initialize session store: %w", err)
	}

	sessions, err := store.ListSessions()
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}

	if len(sessions) == 0 {
		fmt.Println("No sessions found.")
		fmt.Println("\nCreate a session with: juggle sessions create <id> -m \"description\"")
		return nil
	}

	// Load balls to get counts per session (session ID = tag)
	ballStore, err := NewStoreForCommand(cwd)
	if err != nil {
		return fmt.Errorf("failed to initialize ball store: %w", err)
	}

	balls, err := ballStore.LoadBalls()
	if err != nil {
		balls = []*session.Ball{} // Continue even if no balls
	}

	// Count balls per session (by tag)
	ballCounts := make(map[string]int)
	for _, ball := range balls {
		for _, tag := range ball.Tags {
			ballCounts[tag]++
		}
	}

	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	valueStyle := lipgloss.NewStyle()

	fmt.Printf("Sessions (%d):\n\n", len(sessions))
	for _, sess := range sessions {
		ballCount := ballCounts[sess.ID]
		ballCountStr := fmt.Sprintf("%d ball(s)", ballCount)

		fmt.Printf("%s %s\n", labelStyle.Render(sess.ID+":"), valueStyle.Render(sess.Description))
		fmt.Printf("  Balls: %s | Created: %s\n", ballCountStr, sess.CreatedAt.Format("2006-01-02"))
		fmt.Println()
	}

	return nil
}

func runSessionsShow(cmd *cobra.Command, args []string) error {
	id := args[0]

	cwd, err := GetWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	store, err := session.NewSessionStoreWithConfig(cwd, GetStoreConfig())
	if err != nil {
		return fmt.Errorf("failed to initialize session store: %w", err)
	}

	sess, err := store.LoadSession(id)
	if err != nil {
		return fmt.Errorf("failed to load session: %w", err)
	}

	// Load progress
	progress, err := store.LoadProgress(id)
	if err != nil {
		progress = ""
	}

	// Load balls with this tag
	ballStore, err := NewStoreForCommand(cwd)
	if err != nil {
		return fmt.Errorf("failed to initialize ball store: %w", err)
	}

	allBalls, err := ballStore.LoadBalls()
	if err != nil {
		allBalls = []*session.Ball{}
	}

	// Filter balls by tag matching session ID
	var sessionBalls []*session.Ball
	for _, ball := range allBalls {
		for _, tag := range ball.Tags {
			if tag == id {
				sessionBalls = append(sessionBalls, ball)
				break
			}
		}
	}

	// Render session details
	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	valueStyle := lipgloss.NewStyle()
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15"))

	fmt.Println(headerStyle.Render("Session: " + sess.ID))
	fmt.Println()

	if sess.Description != "" {
		fmt.Println(labelStyle.Render("Description:"), valueStyle.Render(sess.Description))
	}
	fmt.Println(labelStyle.Render("Created:"), valueStyle.Render(sess.CreatedAt.Format(time.RFC3339)))
	fmt.Println(labelStyle.Render("Updated:"), valueStyle.Render(sess.UpdatedAt.Format(time.RFC3339)))

	// Acceptance criteria section
	fmt.Println()
	fmt.Printf("%s (%d)\n", labelStyle.Render("Acceptance Criteria:"), len(sess.AcceptanceCriteria))
	if len(sess.AcceptanceCriteria) > 0 {
		for i, ac := range sess.AcceptanceCriteria {
			fmt.Printf("  %d. %s\n", i+1, ac)
		}
	} else {
		fmt.Println("  (no session-level acceptance criteria)")
	}

	// Context section
	fmt.Println()
	fmt.Println(labelStyle.Render("Context:"))
	if sess.Context != "" {
		// Indent context for readability
		lines := strings.Split(sess.Context, "\n")
		for _, line := range lines {
			fmt.Printf("  %s\n", line)
		}
	} else {
		fmt.Println("  (no context set)")
	}

	// Balls section
	fmt.Println()
	fmt.Printf("%s (%d)\n", labelStyle.Render("Balls:"), len(sessionBalls))
	if len(sessionBalls) > 0 {
		for _, ball := range sessionBalls {
			stateStyle := lipgloss.NewStyle()
			switch ball.State {
			case session.StateInProgress:
				stateStyle = stateStyle.Foreground(lipgloss.Color("10"))
			case session.StatePending:
				stateStyle = stateStyle.Foreground(lipgloss.Color("14"))
			case session.StateBlocked:
				stateStyle = stateStyle.Foreground(lipgloss.Color("11"))
			case session.StateComplete:
				stateStyle = stateStyle.Foreground(lipgloss.Color("8"))
			}
			fmt.Printf("  - %s [%s] %s\n", ball.ID, stateStyle.Render(string(ball.State)), ball.Title)
		}
	} else {
		fmt.Println("  (no balls linked to this session)")
	}

	// Progress section
	fmt.Println()
	fmt.Println(labelStyle.Render("Progress:"))
	if progress != "" {
		lines := strings.Split(progress, "\n")
		for _, line := range lines {
			fmt.Printf("  %s\n", line)
		}
	} else {
		fmt.Println("  (no progress logged)")
	}

	return nil
}

func runSessionsContext(cmd *cobra.Command, args []string) error {
	id := args[0]

	cwd, err := GetWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	store, err := session.NewSessionStoreWithConfig(cwd, GetStoreConfig())
	if err != nil {
		return fmt.Errorf("failed to initialize session store: %w", err)
	}

	// Verify session exists
	_, err = store.LoadSession(id)
	if err != nil {
		return fmt.Errorf("failed to load session: %w", err)
	}

	// Handle --set flag (agent-friendly)
	if sessionSetFlag != "" {
		if err := store.UpdateSessionContext(id, sessionSetFlag); err != nil {
			return fmt.Errorf("failed to update context: %w", err)
		}
		fmt.Printf("Updated context for session: %s\n", id)
		return nil
	}

	if sessionEditFlag {
		// Open in editor
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "vi" // Default fallback
		}

		sess, err := store.LoadSession(id)
		if err != nil {
			return fmt.Errorf("failed to load session: %w", err)
		}

		// Create temp file with current context
		tmpFile, err := os.CreateTemp("", "juggle-context-*.md")
		if err != nil {
			return fmt.Errorf("failed to create temp file: %w", err)
		}
		tmpPath := tmpFile.Name()
		defer os.Remove(tmpPath)

		if _, err := tmpFile.WriteString(sess.Context); err != nil {
			tmpFile.Close()
			return fmt.Errorf("failed to write temp file: %w", err)
		}
		tmpFile.Close()

		// Open editor
		editorCmd := exec.Command(editor, tmpPath)
		editorCmd.Stdin = os.Stdin
		editorCmd.Stdout = os.Stdout
		editorCmd.Stderr = os.Stderr

		if err := editorCmd.Run(); err != nil {
			return fmt.Errorf("editor failed: %w", err)
		}

		// Read edited content
		newContext, err := os.ReadFile(tmpPath)
		if err != nil {
			return fmt.Errorf("failed to read edited content: %w", err)
		}

		// Update session context
		if err := store.UpdateSessionContext(id, string(newContext)); err != nil {
			return fmt.Errorf("failed to update context: %w", err)
		}

		fmt.Printf("Updated context for session: %s\n", id)
		return nil
	}

	// Just display context
	sess, err := store.LoadSession(id)
	if err != nil {
		return fmt.Errorf("failed to load session: %w", err)
	}

	if sess.Context == "" {
		fmt.Println("No context set for session:", id)
		fmt.Println("\nSet context with: juggle sessions context", id, "--set \"text\"")
		return nil
	}

	fmt.Println(sess.Context)
	return nil
}

func runSessionsDelete(cmd *cobra.Command, args []string) error {
	id := args[0]

	cwd, err := GetWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	store, err := session.NewSessionStoreWithConfig(cwd, GetStoreConfig())
	if err != nil {
		return fmt.Errorf("failed to initialize session store: %w", err)
	}

	// Verify session exists
	if _, err := store.LoadSession(id); err != nil {
		return fmt.Errorf("session not found: %s", id)
	}

	// Confirm deletion (skip with --yes flag)
	if !sessionYesFlag {
		confirmed, err := ConfirmSingleKey(fmt.Sprintf("Delete session '%s'? This will remove the session directory and all its contents.", id))
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	if err := store.DeleteSession(id); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	fmt.Printf("Deleted session: %s\n", id)
	return nil
}

func runSessionsProgress(cmd *cobra.Command, args []string) error {
	id := args[0]

	cwd, err := GetWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	store, err := session.NewSessionStoreWithConfig(cwd, GetStoreConfig())
	if err != nil {
		return fmt.Errorf("failed to initialize session store: %w", err)
	}

	// Verify session exists
	if _, err := store.LoadSession(id); err != nil {
		return fmt.Errorf("session not found: %s", id)
	}

	// Load progress
	progress, err := store.LoadProgress(id)
	if err != nil {
		return fmt.Errorf("failed to load progress: %w", err)
	}

	if progress == "" {
		fmt.Println("No progress logged for session:", id)
		fmt.Println("\nAppend progress with: juggle progress append", id, "\"text\"")
		return nil
	}

	fmt.Print(progress)
	return nil
}

func runSessionsProgressClear(cmd *cobra.Command, args []string) error {
	id := args[0]

	cwd, err := GetWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	store, err := session.NewSessionStoreWithConfig(cwd, GetStoreConfig())
	if err != nil {
		return fmt.Errorf("failed to initialize session store: %w", err)
	}

	// Verify session exists (skip for _all virtual session)
	if id != "_all" && id != "all" {
		if _, err := store.LoadSession(id); err != nil {
			return fmt.Errorf("session not found: %s", id)
		}
	}

	// Normalize "all" to "_all" for consistency
	clearID := id
	if id == "all" {
		clearID = "_all"
	}

	// Confirm clearing (skip with --yes flag)
	if !sessionProgressClearYesFlag {
		confirmed, err := ConfirmSingleKey(fmt.Sprintf("Clear progress for session '%s'? This cannot be undone.", id))
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	if err := store.ClearProgress(clearID); err != nil {
		return fmt.Errorf("failed to clear progress: %w", err)
	}

	fmt.Printf("Cleared progress for session: %s\n", id)
	return nil
}

func runSessionsEdit(cmd *cobra.Command, args []string) error {
	id := args[0]

	cwd, err := GetWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	store, err := session.NewSessionStoreWithConfig(cwd, GetStoreConfig())
	if err != nil {
		return fmt.Errorf("failed to initialize session store: %w", err)
	}

	// Load session to verify it exists
	sess, err := store.LoadSession(id)
	if err != nil {
		return fmt.Errorf("session not found: %s", id)
	}

	// Check if any flags are provided
	hasFlags := sessionEditDescriptionFlag != "" ||
		sessionEditContextSetFlag != "" ||
		len(sessionEditACFlag) > 0 ||
		len(sessionEditACAppendFlag) > 0 ||
		len(sessionEditACRemoveFlag) > 0 ||
		sessionEditDefaultModelFlag != ""

	// If no flags provided, open in editor
	if !hasFlags {
		return runSessionsEditInEditor(store, sess)
	}

	// Direct edit mode with flags
	modified := false

	if sessionEditDescriptionFlag != "" {
		if err := store.UpdateSessionDescription(id, sessionEditDescriptionFlag); err != nil {
			return fmt.Errorf("failed to update description: %w", err)
		}
		fmt.Printf("✓ Updated description: %s\n", sessionEditDescriptionFlag)
		modified = true
	}

	if sessionEditContextSetFlag != "" {
		if err := store.UpdateSessionContext(id, sessionEditContextSetFlag); err != nil {
			return fmt.Errorf("failed to update context: %w", err)
		}
		fmt.Printf("✓ Updated context\n")
		modified = true
	}

	// Handle acceptance criteria modifications
	if len(sessionEditACFlag) > 0 {
		// Replace all ACs
		if err := store.UpdateSessionAcceptanceCriteria(id, sessionEditACFlag); err != nil {
			return fmt.Errorf("failed to update acceptance criteria: %w", err)
		}
		fmt.Printf("✓ Replaced acceptance criteria (%d items)\n", len(sessionEditACFlag))
		modified = true
	} else if len(sessionEditACAppendFlag) > 0 || len(sessionEditACRemoveFlag) > 0 {
		// Reload session to get current ACs
		sess, err = store.LoadSession(id)
		if err != nil {
			return fmt.Errorf("failed to reload session: %w", err)
		}
		acs := make([]string, len(sess.AcceptanceCriteria))
		copy(acs, sess.AcceptanceCriteria)

		// Append new ACs
		if len(sessionEditACAppendFlag) > 0 {
			acs = append(acs, sessionEditACAppendFlag...)
			fmt.Printf("✓ Appended %d acceptance criteria\n", len(sessionEditACAppendFlag))
		}

		// Remove ACs by text match
		if len(sessionEditACRemoveFlag) > 0 {
			removeSet := make(map[string]bool)
			for _, r := range sessionEditACRemoveFlag {
				removeSet[r] = true
			}
			filtered := make([]string, 0, len(acs))
			removed := 0
			for _, ac := range acs {
				if !removeSet[ac] {
					filtered = append(filtered, ac)
				} else {
					removed++
				}
			}
			acs = filtered
			fmt.Printf("✓ Removed %d acceptance criteria\n", removed)
		}

		if err := store.UpdateSessionAcceptanceCriteria(id, acs); err != nil {
			return fmt.Errorf("failed to update acceptance criteria: %w", err)
		}
		modified = true
	}

	if sessionEditDefaultModelFlag != "" {
		ms := session.ModelSize(sessionEditDefaultModelFlag)
		if ms != session.ModelSizeSmall && ms != session.ModelSizeMedium && ms != session.ModelSizeLarge && ms != session.ModelSizeBlank {
			return fmt.Errorf("invalid model size %q, must be one of: small, medium, large (or empty to clear)", sessionEditDefaultModelFlag)
		}
		if err := store.UpdateSessionDefaultModel(id, ms); err != nil {
			return fmt.Errorf("failed to update default model: %w", err)
		}
		if ms == session.ModelSizeBlank {
			fmt.Printf("✓ Cleared default model\n")
		} else {
			fmt.Printf("✓ Updated default model: %s\n", sessionEditDefaultModelFlag)
		}
		modified = true
	}

	if modified {
		fmt.Printf("\n✓ Session %s updated successfully\n", id)
	}

	return nil
}

func runSessionsEditInEditor(store *session.SessionStore, sess *session.JuggleSession) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi" // Default fallback
	}

	// Create a temporary file with session data in editable format
	tmpFile, err := os.CreateTemp("", "juggle-session-*.yaml")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	// Write session data in a human-readable format
	var content strings.Builder
	content.WriteString("# Session: " + sess.ID + "\n")
	content.WriteString("# Edit the values below and save. Lines starting with # are ignored.\n")
	content.WriteString("# Delete a line to keep the original value.\n\n")

	content.WriteString("description: " + sess.Description + "\n\n")

	content.WriteString("# Model size: small, medium, large (or empty for default)\n")
	content.WriteString("default_model: " + string(sess.DefaultModel) + "\n\n")

	content.WriteString("# Acceptance criteria (one per line after 'acceptance_criteria:')\n")
	content.WriteString("acceptance_criteria:\n")
	for _, ac := range sess.AcceptanceCriteria {
		content.WriteString("  - " + ac + "\n")
	}
	if len(sess.AcceptanceCriteria) == 0 {
		content.WriteString("  # - Add criteria here\n")
	}
	content.WriteString("\n")

	content.WriteString("# Context (multi-line, everything after 'context:' until end of file)\n")
	content.WriteString("context: |\n")
	if sess.Context != "" {
		for _, line := range strings.Split(sess.Context, "\n") {
			content.WriteString("  " + line + "\n")
		}
	} else {
		content.WriteString("  \n")
	}

	if _, err := tmpFile.WriteString(content.String()); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	tmpFile.Close()

	// Open editor
	editorCmd := exec.Command(editor, tmpPath)
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr

	if err := editorCmd.Run(); err != nil {
		return fmt.Errorf("editor failed: %w", err)
	}

	// Read and parse edited content
	editedContent, err := os.ReadFile(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to read edited content: %w", err)
	}

	// Parse the edited content
	newDesc, newModel, newACs, newContext, err := parseEditedSession(string(editedContent))
	if err != nil {
		return fmt.Errorf("failed to parse edited content: %w", err)
	}

	// Apply changes
	modified := false

	if newDesc != sess.Description {
		if err := store.UpdateSessionDescription(sess.ID, newDesc); err != nil {
			return fmt.Errorf("failed to update description: %w", err)
		}
		fmt.Printf("✓ Updated description\n")
		modified = true
	}

	if newModel != sess.DefaultModel {
		if err := store.UpdateSessionDefaultModel(sess.ID, newModel); err != nil {
			return fmt.Errorf("failed to update default model: %w", err)
		}
		fmt.Printf("✓ Updated default model: %s\n", newModel)
		modified = true
	}

	if !stringSliceEqual(newACs, sess.AcceptanceCriteria) {
		if err := store.UpdateSessionAcceptanceCriteria(sess.ID, newACs); err != nil {
			return fmt.Errorf("failed to update acceptance criteria: %w", err)
		}
		fmt.Printf("✓ Updated acceptance criteria (%d items)\n", len(newACs))
		modified = true
	}

	if newContext != sess.Context {
		if err := store.UpdateSessionContext(sess.ID, newContext); err != nil {
			return fmt.Errorf("failed to update context: %w", err)
		}
		fmt.Printf("✓ Updated context\n")
		modified = true
	}

	if modified {
		fmt.Printf("\n✓ Session %s updated successfully\n", sess.ID)
	} else {
		fmt.Println("No changes made.")
	}

	return nil
}

// parseEditedSession parses the edited session file content
func parseEditedSession(content string) (description string, model session.ModelSize, acs []string, context string, err error) {
	lines := strings.Split(content, "\n")

	inACs := false
	inContext := false
	var contextLines []string

	for _, line := range lines {
		// Skip comments (but not in context section)
		if !inContext && strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}

		if inContext {
			// Context continues until end of file
			// Remove the 2-space indent if present
			if strings.HasPrefix(line, "  ") {
				contextLines = append(contextLines, line[2:])
			} else if line == "" {
				contextLines = append(contextLines, "")
			} else {
				// Line without indent, still include it
				contextLines = append(contextLines, line)
			}
			continue
		}

		if inACs {
			trimmed := strings.TrimSpace(line)
			// Check if this is a new field or still in ACs
			if strings.HasPrefix(trimmed, "- ") {
				ac := strings.TrimPrefix(trimmed, "- ")
				if ac != "" && !strings.HasPrefix(ac, "#") {
					acs = append(acs, ac)
				}
			} else if strings.Contains(line, ":") && !strings.HasPrefix(trimmed, "-") && !strings.HasPrefix(trimmed, "#") {
				// New field, exit ACs mode
				inACs = false
			} else {
				continue // empty line or continuation
			}
		}

		if !inACs && !inContext {
			// Parse key: value lines
			if strings.HasPrefix(line, "description:") {
				description = strings.TrimSpace(strings.TrimPrefix(line, "description:"))
			} else if strings.HasPrefix(line, "default_model:") {
				modelStr := strings.TrimSpace(strings.TrimPrefix(line, "default_model:"))
				model = session.ModelSize(modelStr)
			} else if strings.HasPrefix(line, "acceptance_criteria:") {
				inACs = true
			} else if strings.HasPrefix(line, "context:") {
				remainder := strings.TrimSpace(strings.TrimPrefix(line, "context:"))
				if remainder == "|" || remainder == "" {
					inContext = true
				} else {
					// Single-line context
					context = remainder
				}
			}
		}
	}

	// Join context lines, trimming trailing empty lines
	if len(contextLines) > 0 {
		// Trim trailing empty lines
		for len(contextLines) > 0 && contextLines[len(contextLines)-1] == "" {
			contextLines = contextLines[:len(contextLines)-1]
		}
		context = strings.Join(contextLines, "\n")
	}

	return description, model, acs, context, nil
}

// stringSliceEqual compares two string slices for equality
func stringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
