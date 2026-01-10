package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/ohare93/juggle/internal/session"
	"github.com/spf13/cobra"
)

var progressCmd = &cobra.Command{
	Use:   "progress",
	Short: "Manage session progress logs",
	Long:  `Commands for managing session progress logs (progress.txt files).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var progressAppendCmd = &cobra.Command{
	Use:   "append [session-id] <text>",
	Short: "Append a timestamped entry to session progress",
	Long: `Append a timestamped entry to a session's progress.txt file.

The session-id can be provided as the first argument, or via the
JUGGLER_SESSION_ID environment variable.

Creates progress.txt if it doesn't exist.

Examples:
  juggle progress append my-session "Completed user story US-001"
  JUGGLER_SESSION_ID=my-session juggle progress append "Fixed auth bug"`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runProgressAppend,
}

func init() {
	progressCmd.AddCommand(progressAppendCmd)
	rootCmd.AddCommand(progressCmd)
}

func runProgressAppend(cmd *cobra.Command, args []string) error {
	var sessionID, text string

	// Parse args: either (session-id, text) or just (text) with env var
	if len(args) == 2 {
		sessionID = args[0]
		text = args[1]
	} else {
		// Single arg - use env var for session ID
		sessionID = os.Getenv("JUGGLER_SESSION_ID")
		if sessionID == "" {
			return fmt.Errorf("session ID required: provide as first argument or set JUGGLER_SESSION_ID")
		}
		text = args[0]
	}

	cwd, err := GetWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	store, err := session.NewSessionStoreWithConfig(cwd, GetStoreConfig())
	if err != nil {
		return fmt.Errorf("failed to initialize session store: %w", err)
	}

	// Format timestamped entry
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	entry := fmt.Sprintf("[%s] %s\n", timestamp, text)

	// Append to progress file
	if err := store.AppendProgress(sessionID, entry); err != nil {
		return fmt.Errorf("failed to append progress: %w", err)
	}

	// Success message for agent confirmation
	fmt.Printf("Appended to session %s progress.txt\n", sessionID)
	return nil
}
