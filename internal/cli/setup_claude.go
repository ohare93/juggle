package cli

import (
	"fmt"

	"github.com/ohare93/juggle/internal/claude"
	"github.com/spf13/cobra"
)

var setupClaudeCmd = &cobra.Command{
	Use:   "setup-claude",
	Short: "Install juggler instructions for Claude Code agents",
	Long: `Install instructions to CLAUDE.md so Claude Code agents know how to use juggler.

This command adds comprehensive agent instructions to either:
  - .claude/CLAUDE.md (local, default)
  - ~/.claude/CLAUDE.md (global, with --global flag)

The instructions teach agents:
  - When and how to use juggler
  - The juggling state metaphor
  - Best practices for task management
  - Common commands and workflows`,
	RunE: runSetupClaude,
}

var setupClaudeOpts struct {
	global    bool
	dryRun    bool
	update    bool
	uninstall bool
	force     bool
}

func init() {
	setupClaudeCmd.Flags().BoolVar(&setupClaudeOpts.global, "global", false, "Install to global ~/.claude/CLAUDE.md instead of local .claude/CLAUDE.md")
	setupClaudeCmd.Flags().BoolVar(&setupClaudeOpts.dryRun, "dry-run", false, "Show what would be added without actually installing")
	setupClaudeCmd.Flags().BoolVar(&setupClaudeOpts.update, "update", false, "Update existing instructions (removes and re-adds)")
	setupClaudeCmd.Flags().BoolVar(&setupClaudeOpts.uninstall, "uninstall", false, "Remove juggler instructions from CLAUDE.md")
	setupClaudeCmd.Flags().BoolVar(&setupClaudeOpts.force, "force", false, "Don't prompt for confirmation")
}

func runSetupClaude(cmd *cobra.Command, args []string) error {
	// Delegate to setup-agent with "claude" type for backward compatibility
	opts := claude.InstallOptions{
		Global:    setupClaudeOpts.global,
		Local:     !setupClaudeOpts.global,
		DryRun:    setupClaudeOpts.dryRun,
		Update:    setupClaudeOpts.update,
		Uninstall: setupClaudeOpts.uninstall,
		Force:     setupClaudeOpts.force,
		AgentType: "claude",
	}

	config, _ := claude.GetAgentConfig("claude")

	// Get target path
	targetPath, err := claude.GetTargetPath(opts)
	if err != nil {
		return fmt.Errorf("failed to determine target path: %w", err)
	}

	// Check if instructions already exist
	hasInstructions, err := claude.HasInstructions(targetPath)
	if err != nil {
		return fmt.Errorf("failed to check for existing instructions: %w", err)
	}

	// Handle uninstall
	if opts.Uninstall {
		return handleAgentUninstall(targetPath, hasInstructions, opts, config)
	}

	// Handle install/update
	return handleAgentInstall(targetPath, hasInstructions, opts, config)
}

