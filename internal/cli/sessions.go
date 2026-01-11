package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/ohare93/juggle/internal/session"
	"github.com/spf13/cobra"
)

var sessionsCmd = &cobra.Command{
	Use:   "sessions",
	Short: "Manage juggle sessions (ball groupings by tag)",
	Long: `Manage juggle sessions that group balls by tag.

Sessions provide context and progress tracking for related balls.
A session's ID serves as the tag that links balls to that session.

Commands:
  sessions create <id> [-m description]  Create a new session
  sessions list                          List all sessions
  sessions show <id>                     Show session details
  sessions context <id> [--edit]         View or edit session context
  sessions delete <id>                   Delete a session`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var (
	sessionDescriptionFlag string
	sessionContextFlag     string
	sessionEditFlag        bool
	sessionSetFlag         string
)

var sessionsCreateCmd = &cobra.Command{
	Use:   "create <id>",
	Short: "Create a new session",
	Long: `Create a new session with the given ID.

The session ID will also be used as a tag to link balls to this session.
Sessions are stored in .juggler/sessions/<id>/session.json with a
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
Balls tagged with this session ID are not affected.`,
	Args: cobra.ExactArgs(1),
	RunE: runSessionsDelete,
}

func init() {
	// Add flags
	sessionsCreateCmd.Flags().StringVarP(&sessionDescriptionFlag, "message", "m", "", "Session description")
	sessionsCreateCmd.Flags().StringVar(&sessionContextFlag, "context", "", "Initial session context (agent-friendly)")
	sessionsContextCmd.Flags().BoolVar(&sessionEditFlag, "edit", false, "Open context in $EDITOR")
	sessionsContextCmd.Flags().StringVar(&sessionSetFlag, "set", "", "Set context directly (agent-friendly)")

	// Add subcommands
	sessionsCmd.AddCommand(sessionsCreateCmd)
	sessionsCmd.AddCommand(sessionsListCmd)
	sessionsCmd.AddCommand(sessionsShowCmd)
	sessionsCmd.AddCommand(sessionsContextCmd)
	sessionsCmd.AddCommand(sessionsDeleteCmd)
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

	fmt.Printf("Created session: %s\n", sess.ID)
	if description != "" {
		fmt.Printf("  Description: %s\n", description)
	}
	if sessionContextFlag != "" {
		fmt.Printf("  Context: (set)\n")
	}
	fmt.Printf("  Path: .juggler/sessions/%s/\n", id)

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
			fmt.Printf("  - %s [%s] %s\n", ball.ID, stateStyle.Render(string(ball.State)), ball.Intent)
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

	// Confirm deletion
	confirmed, err := ConfirmSingleKey(fmt.Sprintf("Delete session '%s'? This will remove the session directory and all its contents.", id))
	if err != nil {
		return err
	}
	if !confirmed {
		fmt.Println("Cancelled.")
		return nil
	}

	if err := store.DeleteSession(id); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	fmt.Printf("Deleted session: %s\n", id)
	return nil
}
