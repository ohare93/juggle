package cli

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/ohare93/juggle/internal/session"
	"github.com/spf13/cobra"
)

var worktreeCmd = &cobra.Command{
	Use:     "worktree",
	Aliases: []string{"workspace"},
	Short:   "Manage git worktree links for parallel agent execution",
	Long: `Manage worktree links so multiple agents can run in parallel across
different git worktrees while sharing the same ball state.

The main repo stores a registry of linked worktrees, and each worktree
has a .juggle/link file pointing back to the main repo.

Commands must be run from the main repo directory.`,
}

var worktreeAddCmd = &cobra.Command{
	Use:   "add <worktree-path>",
	Short: "Register a git worktree with this repo",
	Long: `Register a git worktree so it shares this repo's .juggle/ state.

This creates:
  - An entry in .juggle/config.json listing the worktree
  - A .juggle/link file in the worktree pointing to this repo

After registration, running juggle commands in the worktree will
read and write to this repo's .juggle/ directory.

Example:
  # Create a worktree with git
  git worktree add ../my-feature-worktree feature-branch

  # Register it with juggle (run from main repo)
  juggle worktree add ../my-feature-worktree`,
	Args: cobra.ExactArgs(1),
	RunE: runWorktreeAdd,
}

var worktreeForgetCmd = &cobra.Command{
	Use:   "forget <worktree-path>",
	Short: "Unregister a worktree (does not delete it)",
	Long: `Remove a worktree from the registry and delete its .juggle/link file.

This does NOT delete the git worktree itself - use 'git worktree remove' for that.

Example:
  juggle worktree forget ../my-feature-worktree
  git worktree remove ../my-feature-worktree  # optional: remove worktree`,
	Args: cobra.ExactArgs(1),
	RunE: runWorktreeForget,
}

var worktreeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List registered worktrees",
	Long:  `Show all worktrees registered with this repo.`,
	Args:  cobra.NoArgs,
	RunE:  runWorktreeList,
}

var worktreeStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show worktree linking status for current directory",
	Long: `Check if the current directory is a worktree and show its linked main repo,
or if it's a main repo, show registered worktrees.`,
	Args: cobra.NoArgs,
	RunE: runWorktreeStatus,
}

var worktreeRunCmd = &cobra.Command{
	Use:   "run <command>",
	Short: "Run a command in all workspaces",
	Long: `Run a command in the main repo and all registered worktrees.

The command is executed sequentially in each workspace. Use --continue-on-error
to continue execution even if a command fails in one workspace.

Examples:
  juggle worktree run "devbox run build"
  juggle worktree run "go test ./..." --continue-on-error
  juggle worktree run "git status"`,
	Args: cobra.ExactArgs(1),
	RunE: runWorktreeRun,
}

var worktreeSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync .claude/settings.local.json across workspaces",
	Long: `Check for mismatches in .claude/settings.local.json between the main repo
and worktrees, and optionally create symlinks to keep them in sync.

The main repo's settings.local.json is treated as the source of truth.
Worktrees with different files will be prompted to symlink to the main repo's file.

Examples:
  juggle worktree sync              # Interactive mode
  juggle worktree sync --dry-run    # Show what would be done
  juggle worktree sync --yes        # Auto-confirm all symlinks`,
	Args: cobra.NoArgs,
	RunE: runWorktreeSync,
}

// Flags for run command
var worktreeRunContinueOnError bool

// Flags for sync command
var (
	worktreeSyncDryRun bool
	worktreeSyncYes    bool
)

func init() {
	// Add flags for run command
	worktreeRunCmd.Flags().BoolVar(&worktreeRunContinueOnError, "continue-on-error", false, "Continue running in other workspaces even if a command fails")

	// Add flags for sync command
	worktreeSyncCmd.Flags().BoolVar(&worktreeSyncDryRun, "dry-run", false, "Show what would be done without making changes")
	worktreeSyncCmd.Flags().BoolVarP(&worktreeSyncYes, "yes", "y", false, "Auto-confirm all symlink operations")

	worktreeCmd.AddCommand(worktreeAddCmd)
	worktreeCmd.AddCommand(worktreeForgetCmd)
	worktreeCmd.AddCommand(worktreeListCmd)
	worktreeCmd.AddCommand(worktreeStatusCmd)
	worktreeCmd.AddCommand(worktreeRunCmd)
	worktreeCmd.AddCommand(worktreeSyncCmd)
	rootCmd.AddCommand(worktreeCmd)
}

func runWorktreeAdd(cmd *cobra.Command, args []string) error {
	worktreePath := args[0]

	// Get absolute path for the worktree
	absWorktreePath, err := filepath.Abs(worktreePath)
	if err != nil {
		return fmt.Errorf("failed to resolve worktree path: %w", err)
	}

	// Get current directory as main repo
	mainDir, err := GetWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Check if we're running from a worktree (not allowed)
	isWt, err := session.IsWorktree(mainDir, GetStoreConfig().JuggleDirName)
	if err == nil && isWt {
		linkedMain, _ := session.GetLinkedMainRepo(mainDir, GetStoreConfig().JuggleDirName)
		return fmt.Errorf("cannot add worktrees from a worktree; run this command from the main repo: %s", linkedMain)
	}

	// Register the worktree
	if err := session.RegisterWorktree(mainDir, absWorktreePath, GetStoreConfig().JuggleDirName); err != nil {
		return err
	}

	fmt.Printf("Registered worktree: %s\n", absWorktreePath)
	fmt.Printf("  Link file created: %s/.juggle/link\n", absWorktreePath)
	fmt.Println("\nYou can now run juggle commands from the worktree.")
	return nil
}

func runWorktreeForget(cmd *cobra.Command, args []string) error {
	worktreePath := args[0]

	// Get absolute path for the worktree
	absWorktreePath, err := filepath.Abs(worktreePath)
	if err != nil {
		return fmt.Errorf("failed to resolve worktree path: %w", err)
	}

	// Get current directory as main repo
	mainDir, err := GetWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Check if we're running from a worktree (not allowed)
	isWt, err := session.IsWorktree(mainDir, GetStoreConfig().JuggleDirName)
	if err == nil && isWt {
		linkedMain, _ := session.GetLinkedMainRepo(mainDir, GetStoreConfig().JuggleDirName)
		return fmt.Errorf("cannot forget worktrees from a worktree; run this command from the main repo: %s", linkedMain)
	}

	// Forget the worktree
	if err := session.ForgetWorktree(mainDir, absWorktreePath, GetStoreConfig().JuggleDirName); err != nil {
		return err
	}

	fmt.Printf("Unregistered worktree: %s\n", absWorktreePath)
	fmt.Println("\nThe worktree directory still exists. To remove it:")
	fmt.Printf("  git worktree remove %s\n", worktreePath)
	return nil
}

func runWorktreeList(cmd *cobra.Command, args []string) error {
	mainDir, err := GetWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// If we're in a worktree, resolve to main repo
	storageDir, err := session.ResolveStorageDir(mainDir, GetStoreConfig().JuggleDirName)
	if err != nil {
		storageDir = mainDir
	}

	worktrees, err := session.ListWorktrees(storageDir, GetStoreConfig().JuggleDirName)
	if err != nil {
		return fmt.Errorf("failed to list worktrees: %w", err)
	}

	if len(worktrees) == 0 {
		fmt.Println("No worktrees registered.")
		fmt.Println("\nTo register a worktree:")
		fmt.Println("  juggle worktree add <path>")
		return nil
	}

	fmt.Printf("Registered worktrees for %s:\n\n", storageDir)
	for _, wt := range worktrees {
		status := "ok"
		if _, err := os.Stat(wt); os.IsNotExist(err) {
			status = "missing"
		}
		fmt.Printf("  %s [%s]\n", wt, status)
	}

	return nil
}

func runWorktreeStatus(cmd *cobra.Command, args []string) error {
	cwd, err := GetWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	juggleDirName := GetStoreConfig().JuggleDirName

	// Check if we're in a worktree
	isWt, err := session.IsWorktree(cwd, juggleDirName)
	if err != nil {
		return fmt.Errorf("failed to check worktree status: %w", err)
	}

	if isWt {
		linkedMain, err := session.GetLinkedMainRepo(cwd, juggleDirName)
		if err != nil {
			return fmt.Errorf("failed to get linked repo: %w", err)
		}
		fmt.Println("Current directory is a WORKTREE")
		fmt.Printf("  Linked to: %s\n", linkedMain)
		fmt.Printf("  Storage: %s/%s/\n", linkedMain, juggleDirName)
		return nil
	}

	// Check if we have a .juggle directory (main repo)
	jugglePath := filepath.Join(cwd, juggleDirName)
	if _, err := os.Stat(jugglePath); os.IsNotExist(err) {
		fmt.Println("Current directory is NOT a juggle project")
		fmt.Println("\nTo initialize:")
		fmt.Println("  juggle plan \"Your first task\"")
		return nil
	}

	// We're in a main repo
	fmt.Println("Current directory is a MAIN REPO")
	fmt.Printf("  Storage: %s/%s/\n", cwd, juggleDirName)

	worktrees, err := session.ListWorktrees(cwd, juggleDirName)
	if err != nil {
		return nil // Don't fail, just don't show worktrees
	}

	if len(worktrees) > 0 {
		fmt.Printf("\nRegistered worktrees: %d\n", len(worktrees))
		for _, wt := range worktrees {
			fmt.Printf("  %s\n", wt)
		}
	}

	return nil
}

// runWorktreeRun executes a command in all workspaces (main + worktrees)
func runWorktreeRun(cmd *cobra.Command, args []string) error {
	command := strings.TrimSpace(args[0])
	if command == "" {
		return fmt.Errorf("command cannot be empty")
	}

	// Get all workspaces
	workspaces, mainDir, err := getAllWorkspaces()
	if err != nil {
		return err
	}

	if len(workspaces) == 0 {
		return fmt.Errorf("no workspaces found")
	}

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	skipStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	succeeded := 0
	failed := 0
	skipped := 0

	for _, ws := range workspaces {
		// Validate workspace exists
		if _, err := os.Stat(ws); os.IsNotExist(err) {
			label := ws
			if ws == mainDir {
				label = ws + " (main)"
			}
			fmt.Printf("\n%s\n", skipStyle.Render("=== "+label+" === SKIPPED (directory not found)"))
			skipped++
			continue
		}

		// Print header
		label := ws
		if ws == mainDir {
			label = ws + " (main)"
		}
		fmt.Printf("\n%s\n", headerStyle.Render("=== "+label+" ==="))

		// Execute command in workspace
		shellCmd := exec.Command("sh", "-c", command)
		shellCmd.Dir = ws
		shellCmd.Stdout = os.Stdout
		shellCmd.Stderr = os.Stderr

		if err := shellCmd.Run(); err != nil {
			fmt.Printf("%s\n", errorStyle.Render(fmt.Sprintf("Command failed: %v", err)))
			failed++
			if !worktreeRunContinueOnError {
				return fmt.Errorf("command failed in %s: %w", ws, err)
			}
		} else {
			succeeded++
		}
	}

	// Print summary
	fmt.Println()
	total := succeeded + failed + skipped
	if failed > 0 || skipped > 0 {
		parts := []string{successStyle.Render(fmt.Sprintf("%d succeeded", succeeded))}
		if failed > 0 {
			parts = append(parts, errorStyle.Render(fmt.Sprintf("%d failed", failed)))
		}
		if skipped > 0 {
			parts = append(parts, skipStyle.Render(fmt.Sprintf("%d skipped", skipped)))
		}
		fmt.Printf("Done: %s\n", strings.Join(parts, ", "))
	} else {
		fmt.Printf("Done: %s\n", successStyle.Render(fmt.Sprintf("%d/%d succeeded", succeeded, total)))
	}

	return nil
}

// runWorktreeSync checks for settings.local.json mismatches and offers to fix them
func runWorktreeSync(cmd *cobra.Command, args []string) error {
	// Get all workspaces
	workspaces, mainDir, err := getAllWorkspaces()
	if err != nil {
		return err
	}

	if len(workspaces) == 0 {
		return fmt.Errorf("no workspaces found")
	}

	// Define the file to sync
	settingsFile := ".claude/settings.local.json"
	mainSettingsPath := filepath.Join(mainDir, settingsFile)

	// Check if main settings file exists
	if _, err := os.Lstat(mainSettingsPath); os.IsNotExist(err) {
		fmt.Printf("Main repo has no %s - nothing to sync.\n", settingsFile)
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to check main settings: %w", err)
	}

	// Get main file info for comparison
	mainContent, err := os.ReadFile(mainSettingsPath)
	if err != nil {
		return fmt.Errorf("failed to read main settings: %w", err)
	}

	mainPermCount := countPermissions(mainContent)

	// Styles
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15"))
	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	okStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	fmt.Println(headerStyle.Render("Checking " + settingsFile + ":"))
	fmt.Printf("  %s %s (%d permissions)\n", labelStyle.Render("Main:"), mainDir, mainPermCount)
	fmt.Println()

	// Track workspaces that need syncing
	type syncTarget struct {
		path   string
		status string // "differs", "missing"
	}
	var needsSync []syncTarget

	// Check each workspace (skip main)
	for _, ws := range workspaces {
		if ws == mainDir {
			continue
		}

		wsSettingsPath := filepath.Join(ws, settingsFile)
		wsName := filepath.Base(ws)

		// Check file status
		wsInfo, err := os.Lstat(wsSettingsPath)
		if os.IsNotExist(err) {
			fmt.Printf("  %s: %s\n", wsName, dimStyle.Render("- missing"))
			needsSync = append(needsSync, syncTarget{path: ws, status: "missing"})
			continue
		}
		if err != nil {
			fmt.Printf("  %s: error: %v\n", wsName, err)
			continue
		}

		// Check if it's already a symlink
		if wsInfo.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(wsSettingsPath)
			if err == nil && target == mainSettingsPath {
				fmt.Printf("  %s: %s\n", wsName, okStyle.Render("✓ symlinked to main"))
				continue
			}
			// Symlink to somewhere else
			fmt.Printf("  %s: %s (-> %s)\n", wsName, warnStyle.Render("⚠ symlinked elsewhere"), target)
			needsSync = append(needsSync, syncTarget{path: ws, status: "differs"})
			continue
		}

		// Read and compare content
		wsContent, err := os.ReadFile(wsSettingsPath)
		if err != nil {
			fmt.Printf("  %s: error reading: %v\n", wsName, err)
			continue
		}

		if bytes.Equal(mainContent, wsContent) {
			fmt.Printf("  %s: %s\n", wsName, okStyle.Render("✓ identical"))
			continue
		}

		// Content differs
		wsPermCount := countPermissions(wsContent)
		fmt.Printf("  %s: %s (%d permissions)\n", wsName, warnStyle.Render("⚠ differs"), wsPermCount)
		needsSync = append(needsSync, syncTarget{path: ws, status: "differs"})
	}

	// If nothing needs syncing, we're done
	if len(needsSync) == 0 {
		fmt.Println()
		fmt.Println(okStyle.Render("All workspaces are in sync."))
		return nil
	}

	// Dry run mode - just report
	if worktreeSyncDryRun {
		fmt.Println()
		fmt.Println("Dry run - would sync these workspaces:")
		for _, target := range needsSync {
			fmt.Printf("  %s (%s)\n", filepath.Base(target.path), target.status)
		}
		return nil
	}

	// Process each target
	fmt.Println()
	synced := 0
	autoConfirmAll := worktreeSyncYes

	for _, target := range needsSync {
		wsName := filepath.Base(target.path)
		wsSettingsPath := filepath.Join(target.path, settingsFile)

		// Prompt for confirmation (unless --yes or user selected 'all')
		if !autoConfirmAll {
			response, err := promptSyncAction(fmt.Sprintf("Symlink %s/%s to main?", wsName, settingsFile))
			if err != nil {
				return err
			}

			switch response {
			case "n":
				fmt.Println("  Skipped")
				continue
			case "a":
				autoConfirmAll = true
			case "q":
				fmt.Println("  Quit")
				return nil
			}
		}

		// Backup existing file if it exists and is not a symlink
		if info, err := os.Lstat(wsSettingsPath); err == nil {
			if info.Mode()&os.ModeSymlink == 0 {
				// Regular file - backup with timestamp if .bak already exists
				backupPath := wsSettingsPath + ".bak"
				if _, err := os.Stat(backupPath); err == nil {
					// .bak already exists, use timestamp
					backupPath = fmt.Sprintf("%s.bak.%d", wsSettingsPath, time.Now().Unix())
				}
				if err := os.Rename(wsSettingsPath, backupPath); err != nil {
					if !os.IsNotExist(err) { // Handle race condition gracefully
						fmt.Printf("  Failed to backup: %v\n", err)
						continue
					}
				} else {
					fmt.Printf("  Backed up to %s\n", filepath.Base(backupPath))
				}
			} else {
				// Remove existing symlink
				if err := os.Remove(wsSettingsPath); err != nil && !os.IsNotExist(err) {
					fmt.Printf("  Failed to remove existing symlink: %v\n", err)
					continue
				}
			}
		}

		// Ensure .claude directory exists
		claudeDir := filepath.Join(target.path, ".claude")
		if err := os.MkdirAll(claudeDir, 0755); err != nil {
			fmt.Printf("  Failed to create .claude directory: %v\n", err)
			continue
		}

		// Create symlink
		if err := os.Symlink(mainSettingsPath, wsSettingsPath); err != nil {
			fmt.Printf("  Failed to create symlink: %v\n", err)
			continue
		}

		fmt.Printf("  %s\n", okStyle.Render("Created symlink"))
		synced++
	}

	fmt.Println()
	fmt.Printf("Done: %d file(s) symlinked\n", synced)

	return nil
}

// getAllWorkspaces returns all workspace paths (main + worktrees) and the main directory
func getAllWorkspaces() ([]string, string, error) {
	cwd, err := GetWorkingDir()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get working directory: %w", err)
	}

	juggleDirName := GetStoreConfig().JuggleDirName

	// Resolve to main repo if we're in a worktree
	mainDir, err := session.ResolveStorageDir(cwd, juggleDirName)
	if err != nil {
		mainDir = cwd
	}

	// Get registered worktrees
	worktrees, err := session.ListWorktrees(mainDir, juggleDirName)
	if err != nil {
		worktrees = []string{}
	}

	// Build list: main first, then worktrees
	workspaces := make([]string, 0, 1+len(worktrees))
	workspaces = append(workspaces, mainDir)
	workspaces = append(workspaces, worktrees...)

	return workspaces, mainDir, nil
}

// countPermissions attempts to count permissions in a settings.local.json file
func countPermissions(content []byte) int {
	var settings struct {
		Permissions struct {
			Allow []interface{} `json:"allow"`
		} `json:"permissions"`
	}
	if err := json.Unmarshal(content, &settings); err != nil {
		return 0
	}
	return len(settings.Permissions.Allow)
}

// promptSyncAction prompts the user for sync confirmation
// Returns: "y" (yes), "n" (no), "a" (all), "q" (quit)
func promptSyncAction(prompt string) (string, error) {
	fmt.Printf("%s [y/n/a(ll)/q(uit)] ", prompt)

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	response := strings.ToLower(strings.TrimSpace(input))
	switch response {
	case "y", "yes":
		return "y", nil
	case "n", "no":
		return "n", nil
	case "a", "all":
		return "a", nil
	case "q", "quit":
		return "q", nil
	default:
		return "n", nil // Default to no
	}
}
