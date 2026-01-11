package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var (
	deleteForce bool
)

var deleteCmd = &cobra.Command{
	Use:   "delete <ball-id>",
	Short: "Delete a ball permanently",
	Long: `Delete a ball permanently from storage.

This action cannot be undone. By default, you will be prompted to confirm
the deletion. Use --force to skip the confirmation prompt.

Examples:
  juggle delete my-app-1
  juggle delete my-app-1 --force`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: CompleteBallIDs,
	RunE:              runDelete,
}

func init() {
	deleteCmd.Flags().BoolVarP(&deleteForce, "force", "f", false, "Skip confirmation prompt")
}

func runDelete(cmd *cobra.Command, args []string) error {
	ballID := args[0]

	// Use findBallByID which respects --all flag
	foundBall, foundStore, err := findBallByID(ballID)
	if err != nil {
		return err
	}

	// Show ball information
	fmt.Printf("Ball to delete:\n")
	fmt.Printf("  ID: %s\n", foundBall.ID)
	fmt.Printf("  Intent: %s\n", foundBall.Intent)
	fmt.Printf("  Priority: %s\n", foundBall.Priority)
	fmt.Printf("  State: %s\n", foundBall.State)
	if len(foundBall.AcceptanceCriteria) > 0 {
		fmt.Printf("  Acceptance Criteria: %d items\n", len(foundBall.AcceptanceCriteria))
	}
	if len(foundBall.Tags) > 0 {
		fmt.Printf("  Tags: %s\n", strings.Join(foundBall.Tags, ", "))
	}
	fmt.Println()

	// Confirm deletion unless --force is used
	if !deleteForce {
		fmt.Print("Are you sure you want to delete this ball? This cannot be undone. ")
		confirmed, err := ConfirmSingleKey("")
		if err != nil {
			return fmt.Errorf("operation cancelled")
		}

		if !confirmed {
			fmt.Println("Deletion cancelled.")
			return nil
		}
	}

	// Delete the ball
	if err := foundStore.DeleteBall(foundBall.ID); err != nil {
		return fmt.Errorf("failed to delete ball: %w", err)
	}

	fmt.Printf("✓ Ball %s deleted successfully\n", ballID)
	return nil
}
