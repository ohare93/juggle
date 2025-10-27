package cli

import (
	"fmt"
	"os"

	"github.com/ohare93/juggle/internal/session"
	"github.com/spf13/cobra"
)

var reminderCmd = &cobra.Command{
	Use:    "reminder",
	Short:  "Check if juggler state reminder should be shown",
	Long:   `Check if enough time has passed to show a reminder about checking juggler state. Called by Claude hooks.`,
	RunE:   runReminder,
	Hidden: true, // Hide from help since it's mainly for hooks
}

// GetReminderCmd returns the reminder command for testing
func GetReminderCmd() *cobra.Command {
	return reminderCmd
}

func runReminder(cmd *cobra.Command, args []string) error {
	// Get current working directory
	cwd, err := session.GetCwd()
	if err != nil {
		// Silently ignore - hook safety
		return nil
	}

	// Check if reminder should be shown
	shouldShow, err := session.ShouldShowReminder(cwd)
	if err != nil {
		// Silently ignore - hook safety
		return nil
	}

	// If reminder needed, print to stderr (so it doesn't interfere with hook output)
	if shouldShow {
		fmt.Fprintln(os.Stderr, "⚠️  Reminder: Check juggler state")
		fmt.Fprintln(os.Stderr, "Run: juggle")
	}

	// Always return nil - never error (hook safety)
	return nil
}
