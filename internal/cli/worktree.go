package cli

import (
	"fmt"
	"os"
	"path/filepath"

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

func init() {
	worktreeCmd.AddCommand(worktreeAddCmd)
	worktreeCmd.AddCommand(worktreeForgetCmd)
	worktreeCmd.AddCommand(worktreeListCmd)
	worktreeCmd.AddCommand(worktreeStatusCmd)
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
