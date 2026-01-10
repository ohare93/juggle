package cli

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ohare93/juggle/internal/session"
	"github.com/ohare93/juggle/internal/tui"
	"github.com/ohare93/juggle/internal/watcher"
	"github.com/spf13/cobra"
)

var tuiSplitView bool

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch interactive terminal UI",
	Long: `Launch an interactive terminal user interface for managing balls.

The TUI provides a full-screen interface with keyboard navigation,
filtering, and quick actions.

Use --split flag to launch the new three-panel split view:
  juggle tui --split

Use --local flag to restrict view to current project only:
  juggle --local tui

Navigation (split view):
  Tab/h/l    Switch between panels
  ↑/k        Move up within panel
  ↓/j        Move down within panel
  Enter      Select item / expand
  Esc        Go back / deselect

CRUD Operations (split view):
  a          Add new item (session/ball/todo)
  e          Edit selected item
  d          Delete selected item (with confirmation)
  Space      Toggle todo completion

State Management:
  s          Start ball
  c          Complete ball
  b          Block ball (prompts for reason)

Legacy Navigation:
  Tab        Cycle state (ready → juggling → complete → dropped → ready)
  r          Set ball to ready
  p          Cycle priority (low → medium → high → urgent → low)
  x          Delete ball (with confirmation)

Filters (toggleable):
  1          Show all states
  2          Toggle pending visibility
  3          Toggle in_progress visibility
  4          Toggle blocked visibility
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

	var model tui.Model

	if tuiSplitView {
		// Initialize split view with file watcher
		sessionStore, err := session.NewSessionStore(workingDir)
		if err != nil {
			return err
		}

		// Create file watcher
		w, err := watcher.New()
		if err != nil {
			return err
		}
		defer w.Close()

		// Watch the project directory
		if err := w.WatchProject(workingDir); err != nil {
			// Non-fatal: continue without watching if it fails
			w = nil
		} else {
			// Start the watcher
			w.Start()
		}

		model = tui.InitialSplitModelWithWatcher(store, sessionStore, config, GlobalOpts.LocalOnly, w)
	} else {
		// Initialize legacy TUI model
		model = tui.InitialModel(store, config, GlobalOpts.LocalOnly)
	}

	// Create program with alternate screen
	p := tea.NewProgram(model, tea.WithAltScreen())

	// Run
	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}

func init() {
	tuiCmd.Flags().BoolVarP(&tuiSplitView, "split", "s", false, "Use split-view layout with sessions, balls, and todos panels")
	rootCmd.AddCommand(tuiCmd)
}
