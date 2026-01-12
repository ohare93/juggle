package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var moveCmd = &cobra.Command{
	Use:   "move <ball-id> <target-project-path>",
	Short: "Move a ball to another project",
	Long: `Transfer a ball from its current project to another project.

The ball will be removed from the current project and added to the target project.
If a ball with the same ID exists in the target, you'll be prompted to provide a new ID.

The target path must be an existing juggler project (contain a .juggler directory).

Examples:
  juggle move juggler-5 ~/Development/other-project
  juggle move 5 ../sibling-project`,
	Args:              cobra.ExactArgs(2),
	ValidArgsFunction: CompleteBallIDs, // Complete for first argument only
	RunE:              runMove,
}

func init() {
	rootCmd.AddCommand(moveCmd)
}

func runMove(cmd *cobra.Command, args []string) error {
	ballID := args[0]
	targetPath := args[1]

	// Resolve target path to absolute
	targetPath, err := filepath.Abs(targetPath)
	if err != nil {
		return fmt.Errorf("invalid target path: %w", err)
	}

	// Verify target has .juggler directory
	jugglerDirName := GlobalOpts.JugglerDir
	if jugglerDirName == "" {
		jugglerDirName = ".juggler"
	}
	targetJugglerDir := filepath.Join(targetPath, jugglerDirName)
	if _, err := os.Stat(targetJugglerDir); os.IsNotExist(err) {
		return fmt.Errorf("target path is not a juggler project (no %s directory): %s", jugglerDirName, targetPath)
	}

	// Find ball across all projects
	ball, sourceStore, err := findBallByID(ballID)
	if err != nil {
		return fmt.Errorf("failed to find ball: %w", err)
	}

	// Check if trying to move to same project
	if ball.WorkingDir == targetPath {
		return fmt.Errorf("ball is already in the target project")
	}

	// Create store for target
	targetStore, err := NewStoreForCommand(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create store for target: %w", err)
	}

	// Check for ID conflict in target
	targetBalls, err := targetStore.LoadBalls()
	if err != nil {
		return fmt.Errorf("failed to load target balls: %w", err)
	}

	originalID := ball.ID
	for _, tb := range targetBalls {
		if tb.ID == ball.ID {
			// Prompt for new ID
			fmt.Printf("⚠️  Ball %s already exists in target project.\n", ball.ID)
			fmt.Printf("Please enter a new ID for the moved ball: ")
			reader := bufio.NewReader(os.Stdin)
			newID, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read input: %w", err)
			}
			newID = strings.TrimSpace(newID)
			if newID == "" {
				return fmt.Errorf("new ID cannot be empty")
			}

			// Validate new ID doesn't conflict
			for _, tb := range targetBalls {
				if tb.ID == newID {
					return fmt.Errorf("ID %s also exists in target project", newID)
				}
			}

			ball.ID = newID
			fmt.Printf("✓ Will use new ID: %s\n", newID)
			break
		}
	}

	// Update working directory
	ball.WorkingDir = targetPath

	// Show what's being moved
	fmt.Printf("\nMoving ball:\n")
	fmt.Printf("  From: %s\n", filepath.Base(sourceStore.ProjectDir()))
	fmt.Printf("  To:   %s\n", filepath.Base(targetPath))
	if ball.ID != originalID {
		fmt.Printf("  Old ID: %s\n", originalID)
		fmt.Printf("  New ID: %s\n", ball.ID)
	} else {
		fmt.Printf("  ID: %s\n", ball.ID)
	}
	fmt.Printf("  Title: %s\n", ball.Title)
	fmt.Println()

	// Append to target
	if err := targetStore.AppendBall(ball); err != nil {
		return fmt.Errorf("failed to add ball to target: %w", err)
	}

	// Delete from source
	if err := sourceStore.DeleteBall(originalID); err != nil {
		// Ball was added to target but we failed to remove from source
		// This is a problem but not critical - user can manually delete
		return fmt.Errorf("ball added to target but failed to remove from source (ID: %s): %w\nYou may need to manually delete from source", originalID, err)
	}

	fmt.Printf("✓ Successfully moved ball %s to %s\n", ball.ID, targetPath)
	return nil
}
