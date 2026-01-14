package cli

import (
	"fmt"
	"os"

	"github.com/ohare93/juggle/internal/session"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "juggle",
	Short: "Run AI agent loops with good UX",
	SilenceUsage: true,
	SilenceErrors: true,
	Long: `Juggle runs autonomous AI agent loops with good UX. Define tasks with
acceptance criteria, start the loop, and add/modify tasks while it runs.
No JSON editing - just TUI or CLI commands.

Quick start:
  juggle                   Launch the interactive terminal UI
  juggle plan              Define a new task interactively
  juggle agent run         Start the autonomous agent loop

Task operations:
  juggle <id>              Start a pending task / show details
  juggle <id> blocked "X"  Mark blocked with reason
  juggle <id> complete     Mark complete and archive
  juggle update <id> ...   Update task properties

Task states: pending → in_progress → complete (or blocked)`,
	RunE:                       runRootCommand,
	Args:                       cobra.ArbitraryArgs,
	DisableFlagParsing:         false,
	FParseErrWhitelist:         cobra.FParseErrWhitelist{UnknownFlags: true},
}

// GlobalOptions holds global configuration flags for testing and path overrides
type GlobalOptions struct {
	ConfigHome  string // Override for ~/.juggle directory
	ProjectDir  string // Override for current working directory
	JuggleDir   string // Override for .juggle directory name
	AllProjects bool   // Enable cross-project discovery (default is local only)
	JSONOutput  bool   // Output as JSON
}

// GlobalOpts holds the parsed global flags (exported for testing)
var GlobalOpts GlobalOptions

// GetWorkingDir returns the working directory, respecting the --project-dir override
func GetWorkingDir() (string, error) {
	if GlobalOpts.ProjectDir != "" {
		return GlobalOpts.ProjectDir, nil
	}
	return os.Getwd()
}

// GetStoreConfig returns StoreConfig based on global flags
func GetStoreConfig() session.StoreConfig {
	return session.StoreConfig{
		JuggleDirName: GlobalOpts.JuggleDir,
	}
}

// GetConfigOptions returns ConfigOptions based on global flags
func GetConfigOptions() session.ConfigOptions {
	opts := session.DefaultConfigOptions()
	if GlobalOpts.ConfigHome != "" {
		opts.ConfigHome = GlobalOpts.ConfigHome
	}
	if GlobalOpts.JuggleDir != "" {
		opts.JuggleDirName = GlobalOpts.JuggleDir
	}
	return opts
}

// NewStoreForCommand creates a Store with configuration from global flags
func NewStoreForCommand(projectDir string) (*session.Store, error) {
	return session.NewStoreWithConfig(projectDir, GetStoreConfig())
}

// LoadConfigForCommand loads Config with options from global flags
func LoadConfigForCommand() (*session.Config, error) {
	return session.LoadConfigWithOptions(GetConfigOptions())
}

// DiscoverProjectsForCommand discovers projects respecting the --all flag
// By default returns only current project directory (local only)
// If --all is set, discovers all projects from config search paths
func DiscoverProjectsForCommand(config *session.Config, store *session.Store) ([]string, error) {
	// --all enables cross-project discovery
	if GlobalOpts.AllProjects {
		return session.DiscoverProjects(config)
	}
	// Default: local only - return just the current project directory
	cwd, err := GetWorkingDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}
	return []string{cwd}, nil
}

// SetVersion sets the version string for the CLI
func SetVersion(v string) {
	rootCmd.Version = v
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

// BallsListOptions holds options for the balls list command
type BallsListOptions struct {
	ShowAll       bool // Show all balls including completed
	ShowCompleted bool // Show only completed balls
}

// BallsListOpts holds the parsed balls list flags
var BallsListOpts BallsListOptions

// ballsCmd lists all balls
var ballsCmd = &cobra.Command{
	Use:   "balls",
	Short: "List all balls (hides completed by default)",
	Long: `List all balls in the current project.

By default, only shows pending, in_progress, and blocked balls.
Use --all to include completed balls, or --completed to show only completed ones.

Examples:
  juggle balls              # Show active balls (pending, in_progress, blocked)
  juggle balls --all        # Show all balls including completed
  juggle balls --completed  # Show only completed balls`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return listAllBalls(cmd)
	},
}

// defaultHelpFunc stores cobra's default help function
var defaultHelpFunc func(*cobra.Command, []string)

// customHelpFunc returns a custom help function for root command
func customHelpFunc(cmd *cobra.Command, args []string) {
	// Only use custom help for root command; use default for others
	if cmd.Name() != "juggle" {
		defaultHelpFunc(cmd, args)
		return
	}

	// Print Long description which has getting started info
	fmt.Println(cmd.Long)
	fmt.Println()

	fmt.Println("Usage:")
	fmt.Println("  juggle [command]")
	fmt.Println("  juggle <ball-id> [operation]")
	fmt.Println()

	// Commands grouped by category
	fmt.Println("Common Commands:")
	fmt.Println("  plan           Plan a new ball interactively")
	fmt.Println("  balls          List all balls")
	fmt.Println("  sessions       Manage sessions (ball groupings)")
	fmt.Println("  tui            Launch interactive terminal UI")
	fmt.Println("  update <id>    Update a ball's properties")
	fmt.Println()

	fmt.Println("All Commands:")
	for _, c := range cmd.Commands() {
		if c.Name() != "help" && c.Name() != "completion" {
			fmt.Printf("  %-14s %s\n", c.Name(), c.Short)
		}
	}
	fmt.Println()

	fmt.Println("Flags:")
	fmt.Println("  -a, --all      Search across all projects")
	fmt.Println("  -h, --help     Help for juggle")
	fmt.Println("  -v, --version  Version for juggle")
	fmt.Println()
	fmt.Println("Use \"juggle [command] --help\" for more information about a command.")
}



func init() {
	// Add persistent global flags for testing and path overrides
	rootCmd.PersistentFlags().StringVar(&GlobalOpts.ConfigHome, "config-home", "", "Override ~/.juggle directory (for testing)")
	rootCmd.PersistentFlags().StringVar(&GlobalOpts.ProjectDir, "project-dir", "", "Override working directory (for testing)")
	rootCmd.PersistentFlags().StringVar(&GlobalOpts.JuggleDir, "juggle-dir", ".juggle", "Override .juggle directory name")
	rootCmd.PersistentFlags().BoolVarP(&GlobalOpts.AllProjects, "all", "a", false, "Search across all discovered projects")
	rootCmd.PersistentFlags().BoolVar(&GlobalOpts.JSONOutput, "json", false, "Output as JSON")

	// Set custom help function
	defaultHelpFunc = rootCmd.HelpFunc()
	rootCmd.SetHelpFunc(customHelpFunc)
	
	// Add flags for ballsCmd
	ballsCmd.Flags().BoolVar(&BallsListOpts.ShowAll, "all", false, "Show all balls including completed ones")
	ballsCmd.Flags().BoolVar(&BallsListOpts.ShowCompleted, "completed", false, "Show only completed balls")

	// Add commands
	rootCmd.AddCommand(ballsCmd)
	rootCmd.AddCommand(checkCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(nextCmd)
	rootCmd.AddCommand(showCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(projectsCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(historyCmd)
	rootCmd.AddCommand(exportCmd)
	rootCmd.AddCommand(planCmd)
	rootCmd.AddCommand(editCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(auditCmd)
	rootCmd.AddCommand(sessionsCmd)
	rootCmd.AddCommand(configCmd)
}
