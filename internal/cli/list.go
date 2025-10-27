package cli

import (
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all sessions (alias for status)",
	Long:  `List all active sessions. This is an alias for the status command.`,
	RunE:  runStatus, // Reuse status command
}
