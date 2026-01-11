package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ohare93/juggle/internal/session"
	"github.com/spf13/cobra"
)

var (
	importSessionID string
)

// importCmd is the parent command for import operations
var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import external data into juggler",
	Long:  `Import external data (like prd.json from ralph) into juggler balls.`,
}

// importRalphCmd imports prd.json user stories as balls
var importRalphCmd = &cobra.Command{
	Use:   "ralph <prd.json>",
	Short: "Import prd.json user stories as balls",
	Long: `Import user stories from a prd.json file as juggler balls.

Creates balls from user stories with the following mappings:
  - title           → intent
  - acceptanceCriteria → acceptance_criteria
  - priority 1-2    → urgent
  - priority 3-5    → high
  - priority 6-10   → medium
  - priority 11+    → low
  - passes: true    → state: complete
  - passes: false   → state: pending

Skips stories that already exist (matching by title/intent).

Examples:
  # Import from prd.json and tag with session
  juggle import ralph prd.json --session my-feature

  # Import from .agent/prd.json
  juggle import ralph .agent/prd.json`,
	Args: cobra.ExactArgs(1),
	RunE: runImportRalph,
}

func init() {
	importRalphCmd.Flags().StringVarP(&importSessionID, "session", "s", "", "Session ID to tag imported balls with")
	importCmd.AddCommand(importRalphCmd)
	rootCmd.AddCommand(importCmd)
}

func runImportRalph(cmd *cobra.Command, args []string) error {
	prdPath := args[0]

	// Get current directory
	cwd, err := GetWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Make path absolute if relative
	if !filepath.IsAbs(prdPath) {
		prdPath = filepath.Join(cwd, prdPath)
	}

	// Check if file exists
	if _, err := os.Stat(prdPath); os.IsNotExist(err) {
		return fmt.Errorf("prd.json not found: %s", prdPath)
	}

	// Validate session exists if specified
	if importSessionID != "" {
		sessionStore, err := session.NewSessionStore(cwd)
		if err != nil {
			return fmt.Errorf("failed to create session store: %w", err)
		}
		if _, err := sessionStore.LoadSession(importSessionID); err != nil {
			return fmt.Errorf("session not found: %s", importSessionID)
		}
	}

	return importPRDFile(prdPath, cwd, importSessionID)
}

// importPRDFile reads prd.json and creates balls from user stories
func importPRDFile(prdPath, projectDir, sessionID string) error {
	// Read prd.json
	data, err := os.ReadFile(prdPath)
	if err != nil {
		return fmt.Errorf("failed to read prd.json: %w", err)
	}

	var prd PRDFile
	if err := json.Unmarshal(data, &prd); err != nil {
		return fmt.Errorf("failed to parse prd.json: %w", err)
	}

	// Create store for project
	store, err := NewStoreForCommand(projectDir)
	if err != nil {
		return fmt.Errorf("failed to create store: %w", err)
	}

	// Load existing balls
	balls, err := store.LoadBalls()
	if err != nil {
		return fmt.Errorf("failed to load balls: %w", err)
	}

	// Build lookup by title (intent) to check for existing balls
	existingTitles := make(map[string]bool)
	for _, ball := range balls {
		existingTitles[ball.Intent] = true
	}

	var imported, skipped int

	for _, story := range prd.UserStories {
		// Check if ball already exists (match by title)
		if existingTitles[story.Title] {
			fmt.Printf("Skipped: %s - \"%s\" (already exists)\n", story.ID, story.Title)
			skipped++
			continue
		}

		// Create new ball
		ball, err := session.NewBall(projectDir, story.Title, mapPriorityNumber(story.Priority))
		if err != nil {
			fmt.Printf("Warning: failed to create ball for %s: %v\n", story.ID, err)
			continue
		}

		// Set acceptance criteria
		if len(story.AcceptanceCriteria) > 0 {
			ball.SetAcceptanceCriteria(story.AcceptanceCriteria)
		}

		// Set state based on passes
		if story.Passes {
			ball.State = session.StateComplete
			now := time.Now()
			ball.CompletedAt = &now
		} else {
			ball.State = session.StatePending
		}

		// Add story ID as tag for reference
		ball.AddTag(story.ID)

		// Add session tag if specified
		if sessionID != "" {
			ball.AddTag(sessionID)
		}

		if err := store.AppendBall(ball); err != nil {
			fmt.Printf("Warning: failed to create ball for %s: %v\n", story.ID, err)
			continue
		}
		imported++
		fmt.Printf("Imported: %s → %s (%s)\n", story.ID, ball.ID, ball.State)

		// Add to lookup for subsequent stories
		existingTitles[story.Title] = true
	}

	fmt.Printf("\nImport complete: %d imported, %d skipped\n", imported, skipped)
	return nil
}
