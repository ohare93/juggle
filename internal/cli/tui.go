package cli

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ohare93/juggle/internal/session"
	"github.com/ohare93/juggle/internal/tui"
	"github.com/spf13/cobra"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch interactive terminal UI",
	Long: `Launch an interactive terminal user interface for managing balls.

The TUI provides a full-screen interface with keyboard navigation,
filtering, and quick actions.

Use --local flag to restrict view to current project only:
  juggle --local tui

Navigation:
  ↑/k        Move up
  ↓/j        Move down
  Enter      View ball details
  b/Esc      Back to list (or exit from list)

State Management:
  Tab        Cycle state (ready → juggling → complete → dropped → ready)
  s          Start ball (ready → juggling:in-air)
  r          Set ball to ready
  c          Complete ball
  d          Drop ball

Ball Operations:
  x          Delete ball (with confirmation)
  p          Cycle priority (low → medium → high → urgent → low)

Filters (toggleable):
  1          Show all states
  2          Toggle ready visibility
  3          Toggle juggling visibility
  4          Toggle dropped visibility
  5          Toggle complete visibility

Other:
  R          Refresh/reload (shift+r)
  ?          Toggle help
  q          Quit`,
	RunE: runTUI,
}

func runTUI(cmd *cobra.Command, args []string) error {
	// Get working directory
	workingDir, err := os.Getwd()
	if err != nil {
		return err
	}

	// Load config
	config, err := session.LoadConfig()
	if err != nil {
		return err
	}

	// Create store
	store, err := session.NewStore(workingDir)
	if err != nil {
		return err
	}

	// Initialize TUI model
	model := tui.InitialModel(store, config, GlobalOpts.LocalOnly)

	// Create program with alternate screen
	p := tea.NewProgram(model, tea.WithAltScreen())

	// Run
	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}
