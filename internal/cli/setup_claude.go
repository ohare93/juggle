package cli

import (
	"fmt"
	"strings"

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
	global       bool
	dryRun       bool
	update       bool
	uninstall    bool
	force        bool
	installHooks bool
}

func init() {
	setupClaudeCmd.Flags().BoolVar(&setupClaudeOpts.global, "global", false, "Install to global ~/.claude/CLAUDE.md instead of local .claude/CLAUDE.md")
	setupClaudeCmd.Flags().BoolVar(&setupClaudeOpts.dryRun, "dry-run", false, "Show what would be added without actually installing")
	setupClaudeCmd.Flags().BoolVar(&setupClaudeOpts.update, "update", false, "Update existing instructions (removes and re-adds)")
	setupClaudeCmd.Flags().BoolVar(&setupClaudeOpts.uninstall, "uninstall", false, "Remove juggler instructions from CLAUDE.md")
	setupClaudeCmd.Flags().BoolVar(&setupClaudeOpts.force, "force", false, "Don't prompt for confirmation")
	setupClaudeCmd.Flags().BoolVar(&setupClaudeOpts.installHooks, "install-hooks", false, "Also install hooks for activity tracking")
}

func runSetupClaude(cmd *cobra.Command, args []string) error {
	opts := claude.InstallOptions{
		Global:    setupClaudeOpts.global,
		Local:     !setupClaudeOpts.global,
		DryRun:    setupClaudeOpts.dryRun,
		Update:    setupClaudeOpts.update,
		Uninstall: setupClaudeOpts.uninstall,
		Force:     setupClaudeOpts.force,
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
		return handleUninstall(targetPath, hasInstructions, opts)
	}

	// Handle install/update
	return handleInstall(targetPath, hasInstructions, opts)
}

func handleUninstall(targetPath string, hasInstructions bool, opts claude.InstallOptions) error {
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

	// Check if hooks exist (only for local installations)
	var hooksExist bool
	var projectDir string
	if !opts.Global {
		projectDir, err = claude.GetProjectDir()
		if err == nil {
			hooksExist, _ = claude.HooksInstalled(projectDir)
		}
	}

	if opts.DryRun {
		fmt.Println("Dry run - would perform these actions:")
		fmt.Println("- Remove juggler instructions from", targetPath)
		if hooksExist {
			fmt.Println("- Remove hooks from .claude/hooks.json")
		}
		return nil
	}

	// Confirm unless forced
	if !opts.Force {
		fmt.Printf("Remove juggler instructions from %s? [y/N]: ", targetPath)
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	// Write updated content
	if err := claude.WriteFile(targetPath, newContent); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	fmt.Println("✓ Removed juggler instructions from", targetPath)

	// Remove hooks if they exist
	if hooksExist && projectDir != "" {
		if err := claude.RemoveHooks(projectDir); err != nil {
			// Don't fail entire operation if hooks removal fails
			fmt.Println("⚠️  Warning: Failed to remove hooks:", err)
		} else {
			fmt.Println("✓ Removed hooks from .claude/hooks.json")
		}
	}

	return nil
}

func handleInstall(targetPath string, hasInstructions bool, opts claude.InstallOptions) error {
	// Check if update is needed
	if hasInstructions && !opts.Update {
		fmt.Printf("Juggler instructions already exist in %s\n", targetPath)
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
	fmt.Println("Juggler Setup for Claude Code")
	fmt.Println()
	
	scope := "local"
	if opts.Global {
		scope = "global"
	}
	
	action := "add"
	if hasInstructions {
		action = "update"
	}
	
	fmt.Printf("This will %s agent instructions to: %s (%s)\n", action, targetPath, scope)
	fmt.Println()

	// Choose the appropriate template for preview
	template := claude.InstructionsTemplate
	if opts.Global {
		template = claude.GlobalInstructionsTemplate
	}

	if opts.DryRun {
		fmt.Println("Dry run - would perform these actions:")
		fmt.Println("- Install juggler instructions to", targetPath)
		if setupClaudeOpts.installHooks && !opts.Global {
			fmt.Println("- Install hooks to .claude/hooks.json")
		}
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
		fmt.Printf("Install these instructions? [Y/n]: ")
		var response string
		fmt.Scanln(&response)
		response = strings.ToLower(strings.TrimSpace(response))
		if response != "" && response != "y" && response != "yes" {
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

	// Install hooks if requested (only for local installations)
	var hooksInstalled bool
	if setupClaudeOpts.installHooks && !opts.Global {
		projectDir, err := claude.GetProjectDir()
		if err != nil {
			fmt.Println("⚠️  Warning: Failed to get project directory for hooks:", err)
		} else {
			if err := claude.InstallHooks(projectDir); err != nil {
				// Don't fail entire operation if hooks installation fails
				fmt.Println("⚠️  Warning: Failed to install hooks:", err)
			} else {
				hooksInstalled = true
			}
		}
	}

	// Show success messages
	fmt.Println()
	if opts.Global {
		fmt.Println("✓ Installed global juggler instructions to", targetPath)
	} else {
		fmt.Println("✓", verb, "juggler instructions to", targetPath)
	}

	if hooksInstalled {
		fmt.Println("✓ Installed hooks to .claude/hooks.json")
	}

	fmt.Println()

	// Show appropriate completion message
	if opts.Global {
		fmt.Println("Juggler integration complete!")
		fmt.Println()
		fmt.Println("Note: Global instructions point to project-specific CLAUDE.md files.")
		fmt.Println("Run 'juggle setup-claude --install-hooks' in each project for full integration.")
	} else {
		fmt.Println("Juggler integration complete!")
	}

	fmt.Println()
	fmt.Println("⚠️  IMPORTANT: Restart Claude Code for changes to take effect.")

	return nil
}
