package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/ohare93/juggle/internal/claude"
	"github.com/spf13/cobra"
)

var setupAgentCmd = &cobra.Command{
	Use:   "setup-agent [agent-type]",
	Short: "Install juggler instructions for an AI coding agent",
	Long: `Install juggler workflow instructions into agent instruction files.

Supported agents: claude, cursor, aider

The command searches for existing instruction files in standard locations:
  - Claude: .claude/CLAUDE.md, CLAUDE.md, AGENTS.md
  - Cursor: .cursorrules, AGENTS.md
  - Aider: .aider.conf.yml, AGENTS.md

Examples:
  juggle setup-agent claude          # Install for Claude (same as setup-claude)
  juggle setup-agent cursor          # Install for Cursor AI
  juggle setup-agent aider           # Install for Aider
  juggle setup-agent claude --global # Install to global location
  juggle setup-agent --list          # List all supported agents`,
	Args: func(cmd *cobra.Command, args []string) error {
		// Allow no args if --list flag is provided
		if setupAgentOpts.listAgents {
			return nil
		}
		return cobra.ExactArgs(1)(cmd, args)
	},
	RunE: runSetupAgent,
}

var setupAgentOpts struct {
	global     bool
	dryRun     bool
	update     bool
	uninstall  bool
	force      bool
	listAgents bool
}

func init() {
	setupAgentCmd.Flags().BoolVar(&setupAgentOpts.global, "global", false, "Install to global location instead of local")
	setupAgentCmd.Flags().BoolVar(&setupAgentOpts.dryRun, "dry-run", false, "Show what would be added without actually installing")
	setupAgentCmd.Flags().BoolVar(&setupAgentOpts.update, "update", false, "Update existing instructions (removes and re-adds)")
	setupAgentCmd.Flags().BoolVar(&setupAgentOpts.uninstall, "uninstall", false, "Remove juggler instructions from file")
	setupAgentCmd.Flags().BoolVar(&setupAgentOpts.force, "force", false, "Don't prompt for confirmation")
	setupAgentCmd.Flags().BoolVar(&setupAgentOpts.listAgents, "list", false, "List all supported agent types")
}

func runSetupAgent(cmd *cobra.Command, args []string) error {
	// Handle --list flag
	if setupAgentOpts.listAgents {
		return listSupportedAgents()
	}

	agentType := args[0]

	// Validate agent type
	config, ok := claude.GetAgentConfig(agentType)
	if !ok {
		availableAgents := strings.Join(claude.ListSupportedAgents(), ", ")
		return fmt.Errorf("unsupported agent type: %s (supported: %s)", agentType, availableAgents)
	}

	opts := claude.InstallOptions{
		Global:    setupAgentOpts.global,
		Local:     !setupAgentOpts.global,
		DryRun:    setupAgentOpts.dryRun,
		Update:    setupAgentOpts.update,
		Uninstall: setupAgentOpts.uninstall,
		Force:     setupAgentOpts.force,
		AgentType: agentType,
	}

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

func listSupportedAgents() error {
	fmt.Println("Supported AI coding agents:")
	fmt.Println()

	agents := claude.ListSupportedAgents()
	sort.Strings(agents)

	for _, agentType := range agents {
		config, _ := claude.GetAgentConfig(agentType)
		fmt.Printf("  %s (%s)\n", agentType, config.Name)
		fmt.Printf("    File locations: %s\n", strings.Join(config.InstructionPaths, ", "))
	}

	fmt.Println()
	fmt.Println("Use 'juggle setup-agent <agent-type>' to install instructions for a specific agent")

	return nil
}

func handleAgentUninstall(targetPath string, hasInstructions bool, opts claude.InstallOptions, config claude.AgentConfig) error {
	if !hasInstructions {
		fmt.Println("No juggler instructions found in", targetPath)
		return nil
	}

	// Read current content
	content, err := claude.ReadFile(targetPath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Remove instructions
	newContent := claude.RemoveInstructions(content)

	if opts.DryRun {
		fmt.Printf("Dry run - would remove %s instructions from %s\n", config.Name, targetPath)
		return nil
	}

	// Confirm unless forced
	if !opts.Force {
		confirmed, err := ConfirmSingleKey(fmt.Sprintf("Remove %s instructions from %s?", config.Name, targetPath))
		if err != nil {
			return fmt.Errorf("operation cancelled")
		}
		if !confirmed {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	// Write updated content
	if err := claude.WriteFile(targetPath, newContent); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	fmt.Printf("✓ Removed %s instructions from %s\n", config.Name, targetPath)

	return nil
}

func handleAgentInstall(targetPath string, hasInstructions bool, opts claude.InstallOptions, config claude.AgentConfig) error {
	// Check if update is needed
	if hasInstructions && !opts.Update {
		fmt.Printf("%s instructions already exist in %s\n", config.Name, targetPath)
		fmt.Println("Use --update to replace them with the latest version")
		return nil
	}

	// Read current content (empty string if file doesn't exist)
	content, err := claude.ReadFile(targetPath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Generate new content
	var newContent string
	if hasInstructions {
		newContent = claude.UpdateInstructions(content, opts.Global)
	} else {
		if opts.Global {
			newContent = claude.AddGlobalInstructions(content)
		} else {
			newContent = claude.AddInstructions(content)
		}
	}

	// Show preview
	fmt.Printf("Juggler Setup for %s\n", config.Name)
	fmt.Println()

	scope := "local"
	if opts.Global {
		scope = "global"
	}

	action := "add"
	if hasInstructions {
		action = "update"
	}

	// Show file location info
	fileInfo := ""
	if !opts.Global {
		wd, _ := claude.GetProjectDir()
		if existing := claude.FindInstructionFileForAgent(wd, opts.AgentType); existing != "" {
			relPath := strings.TrimPrefix(existing, wd+"/")
			fileInfo = fmt.Sprintf(" (found existing file: %s)", relPath)
		}
	}

	fmt.Printf("This will %s %s instructions to: %s (%s)%s\n", action, config.Name, targetPath, scope, fileInfo)
	fmt.Println()

	// Choose the appropriate template for preview
	template := claude.InstructionsTemplate
	if opts.Global {
		template = claude.GlobalInstructionsTemplate
	}

	if opts.DryRun {
		fmt.Println("Dry run - would perform these actions:")
		fmt.Printf("- Install %s instructions to %s\n", config.Name, targetPath)
		fmt.Println()
		fmt.Println("Preview of instructions to be added:")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println(strings.TrimSpace(template))
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		return nil
	}

	// Show a preview snippet
	fmt.Println("Preview:")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	lines := strings.Split(strings.TrimSpace(template), "\n")
	previewLines := 15
	if len(lines) < previewLines {
		previewLines = len(lines)
	}
	for i := 0; i < previewLines; i++ {
		fmt.Println(lines[i])
	}
	if len(lines) > previewLines {
		fmt.Printf("... (%d more lines)\n", len(lines)-previewLines)
	}
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// Confirm unless forced
	if !opts.Force {
		confirmed, err := ConfirmSingleKey("Install these instructions?")
		if err != nil {
			return fmt.Errorf("operation cancelled")
		}
		if !confirmed {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	// Write file
	if err := claude.WriteFile(targetPath, newContent); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	verb := "Installed"
	if hasInstructions {
		verb = "Updated"
	}

	// Show success messages
	fmt.Println()
	if opts.Global {
		fmt.Printf("✓ Installed global %s instructions to %s\n", config.Name, targetPath)
	} else {
		fmt.Printf("✓ %s %s instructions to %s\n", verb, config.Name, targetPath)
	}

	fmt.Println()
	fmt.Printf("Juggler integration for %s complete!\n", config.Name)

	if opts.Global {
		fmt.Println()
		fmt.Println("Note: Global instructions point to project-specific files.")
		fmt.Println("Run 'juggle setup-agent", opts.AgentType, "' in each project for full integration.")
	}

	fmt.Println()
	fmt.Printf("⚠️  IMPORTANT: Restart %s for changes to take effect.\n", config.Name)

	return nil
}
