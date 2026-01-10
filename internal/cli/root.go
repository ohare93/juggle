package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/ohare93/juggle/internal/session"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "juggle",
	Short: "Manage multiple tasks being juggled simultaneously",
	SilenceUsage: true,
	SilenceErrors: true,
	Long: `Juggle helps you manage multiple concurrent tasks (balls) with AI agents.
Track which balls are in-air (being worked on), need-thrown (need direction),
or need-caught (need verification).

Getting started:
- Create and start juggling immediately: juggle start
- Plan for later: juggle plan, then juggle <ball-id> to activate
- See what's juggling: juggle (no args)
- See only local project balls: juggle --local

The juggling metaphor:
- ready: Ball is ready to start juggling
- juggling: Ball is currently being juggled
  - needs-thrown: Needs your direction/input
  - in-air: Agent is actively working
  - needs-caught: Agent finished, needs your verification
- dropped: Ball was explicitly dropped
- complete: Ball successfully caught and complete`,
	RunE:                       runRootCommand,
	Args:                       cobra.ArbitraryArgs,
	DisableFlagParsing:         false,
	FParseErrWhitelist:         cobra.FParseErrWhitelist{UnknownFlags: true},
}

// GlobalOptions holds global configuration flags for testing and path overrides
type GlobalOptions struct {
	ConfigHome string // Override for ~/.juggler directory
	ProjectDir string // Override for current working directory
	JugglerDir string // Override for .juggler directory name
	LocalOnly  bool   // Restrict operations to current project only
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
		JugglerDirName: GlobalOpts.JugglerDir,
	}
}

// GetConfigOptions returns ConfigOptions based on global flags
func GetConfigOptions() session.ConfigOptions {
	opts := session.DefaultConfigOptions()
	if GlobalOpts.ConfigHome != "" {
		opts.ConfigHome = GlobalOpts.ConfigHome
	}
	if GlobalOpts.JugglerDir != "" {
		opts.JugglerDirName = GlobalOpts.JugglerDir
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

// DiscoverProjectsForCommand discovers projects respecting the --local flag
// If --local is set, returns only current project directory
// Otherwise discovers all projects from config search paths
func DiscoverProjectsForCommand(config *session.Config, store *session.Store) ([]string, error) {
	if GlobalOpts.LocalOnly {
		// Local only - return just the current project directory
		cwd, err := GetWorkingDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get current directory: %w", err)
		}
		return []string{cwd}, nil
	}
	// Cross-project - discover all
	return session.DiscoverProjects(config)
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

// ballsCmd lists all balls
var ballsCmd = &cobra.Command{
	Use:   "balls",
	Short: "List all balls regardless of state",
	RunE: func(cmd *cobra.Command, args []string) error {
		return listAllBalls(cmd)
	},
}

// defaultHelpFunc stores cobra's default help function
var defaultHelpFunc func(*cobra.Command, []string)

// customHelpFunc returns a custom help function for root command
func customHelpFunc(cmd *cobra.Command, args []string) {
	// Only use custom help for root command; use default for others
	if cmd.Name() != "juggler" {
		defaultHelpFunc(cmd, args)
		return
	}
	
	// Custom help output with better session command formatting
	fmt.Println(cmd.Long)
	fmt.Println()
	
	fmt.Println("Usage:")
	fmt.Println("  juggler [command]")
	fmt.Println()
	
	// Session Commands section with special formatting
	fmt.Println("Session Commands:")
	fmt.Println("  juggler session [subcommand]  (aliases: ball, project, repo, current)")
	fmt.Println()
	fmt.Println("    When called without subcommands, shows current session details.")
	fmt.Println("    Session operations can be called directly or via 'session' command:")
	fmt.Println()
	fmt.Println("      block (b)          Mark current session as blocked")
	fmt.Println("      unblock (ub)       Clear blocker from current session")
	fmt.Println("      done (d, complete) Mark current session as complete")
	fmt.Println("      todo (t, todos)    Manage todos for a session")
	fmt.Println("      tag (tags)         Manage tags for a session")
	fmt.Println()
	fmt.Println("    Examples: juggler block, juggler session block, juggler ball done")
	fmt.Println()
	
	// All Commands
	fmt.Println("All Commands:")
	for _, c := range cmd.Commands() {
		if c.Name() != "help" && c.Name() != "completion" {
			aliases := ""
			if len(c.Aliases) > 0 {
				aliases = " (" + strings.Join(c.Aliases, ", ") + ")"
			}
			fmt.Printf("  %-20s %s\n", c.Name()+aliases, c.Short)
		}
	}
	fmt.Println()
	
	fmt.Println("Flags:")
	fmt.Println("  -h, --help   help for juggler")
	fmt.Println()
	fmt.Println("Use \"juggler [command] --help\" for more information about a command.")
}



// customSessionHelpFunc returns a custom help function for session command
func customSessionHelpFunc(cmd *cobra.Command, args []string) {
	fmt.Println(cmd.Long)
	fmt.Println()
	
	fmt.Println("Usage:")
	fmt.Println("  juggler session [subcommand]  (aliases: ball, project, repo, current)")
	fmt.Println()
	
	fmt.Println("When called without subcommands, shows current session details.")
	fmt.Println()
	
	fmt.Println("Available Subcommands:")
	fmt.Println("  block (b)          Mark current session as blocked")
	fmt.Println("  unblock (ub)       Clear blocker from current session") 
	fmt.Println("  done (d, complete) Mark current session as complete")
	fmt.Println("  todo (t, todos)    Manage todos for a session")
	fmt.Println("  tag (tags)         Manage tags for a session")
	fmt.Println()
	
	fmt.Println("Note: All subcommands can also be called directly:")
	fmt.Println("  juggler block     = juggler session block")
	fmt.Println("  juggler ball done = juggler session done")
	fmt.Println()
	
	fmt.Println("Flags:")
	fmt.Println("  -h, --help   help for session")
	fmt.Println()
	fmt.Println("Use \"juggler session [subcommand] --help\" for more information about a subcommand.")
}

func init() {
	// Add persistent global flags for testing and path overrides
	rootCmd.PersistentFlags().StringVar(&GlobalOpts.ConfigHome, "config-home", "", "Override ~/.juggler directory (for testing)")
	rootCmd.PersistentFlags().StringVar(&GlobalOpts.ProjectDir, "project-dir", "", "Override working directory (for testing)")
	rootCmd.PersistentFlags().StringVar(&GlobalOpts.JugglerDir, "juggler-dir", ".juggler", "Override .juggler directory name")
	rootCmd.PersistentFlags().BoolVar(&GlobalOpts.LocalOnly, "local", false, "Restrict operations to current project only")

	// Set custom help function
	defaultHelpFunc = rootCmd.HelpFunc()
	rootCmd.SetHelpFunc(customHelpFunc)
	
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
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(setupAgentCmd)
	rootCmd.AddCommand(setupClaudeCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(auditCmd)
	rootCmd.AddCommand(reminderCmd)
	rootCmd.AddCommand(trackActivityCmd)
}
