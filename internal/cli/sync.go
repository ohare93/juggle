package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/ohare93/juggle/internal/session"
	"github.com/spf13/cobra"
)

var (
	syncWatch bool
)

// syncCmd is the parent command for sync operations
var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync external state into juggler",
	Long:  `Sync external state (like prd.json from ralph) into juggler balls.`,
}

// syncRalphCmd syncs prd.json status to balls
var syncRalphCmd = &cobra.Command{
	Use:   "ralph [prd.json]",
	Short: "Sync prd.json status to balls",
	Long: `Sync prd.json user story status to juggler balls for backward compatibility.

Reads a prd.json file and updates corresponding balls:
  - passes: true  → state: complete
  - passes: false → state: pending (or in_progress if has todos started)

Creates balls if they don't exist (matching by title/intent).

Examples:
  # Sync from default .agent/prd.json
  juggle sync ralph

  # Sync from specific file
  juggle sync ralph path/to/prd.json

  # Watch for changes and sync continuously
  juggle sync ralph --watch`,
	RunE: runSyncRalph,
}

func init() {
	syncRalphCmd.Flags().BoolVarP(&syncWatch, "watch", "w", false, "Watch for changes and sync continuously")
	syncCmd.AddCommand(syncRalphCmd)
	rootCmd.AddCommand(syncCmd)
}

// PRDFile represents the structure of a prd.json file
type PRDFile struct {
	Project     string      `json:"project"`
	BranchName  string      `json:"branchName"`
	Description string      `json:"description"`
	UserStories []UserStory `json:"userStories"`
}

// UserStory represents a user story in prd.json
type UserStory struct {
	ID                 string   `json:"id"`
	Title              string   `json:"title"`
	Description        string   `json:"description"`
	AcceptanceCriteria []string `json:"acceptanceCriteria"`
	Priority           int      `json:"priority"`
	Passes             bool     `json:"passes"`
}

func runSyncRalph(cmd *cobra.Command, args []string) error {
	// Get current directory
	cwd, err := GetWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Determine prd.json path
	var prdPath string
	if len(args) > 0 {
		prdPath = args[0]
		if !filepath.IsAbs(prdPath) {
			prdPath = filepath.Join(cwd, prdPath)
		}
	} else {
		// Default to .agent/prd.json
		prdPath = filepath.Join(cwd, ".agent", "prd.json")
	}

	// Check if file exists
	if _, err := os.Stat(prdPath); os.IsNotExist(err) {
		return fmt.Errorf("prd.json not found: %s", prdPath)
	}

	// If watch mode, set up file watcher
	if syncWatch {
		return watchAndSync(prdPath, cwd)
	}

	// Single sync
	return syncPRDFile(prdPath, cwd)
}

// syncPRDFile reads prd.json and syncs to balls
func syncPRDFile(prdPath, projectDir string) error {
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

	// Build lookup by title (intent)
	ballsByTitle := make(map[string]*session.Ball)
	for _, ball := range balls {
		ballsByTitle[ball.Intent] = ball
	}

	var created, updated, skipped int

	for _, story := range prd.UserStories {
		// Check if ball already exists (match by title)
		ball, exists := ballsByTitle[story.Title]

		if exists {
			// Update existing ball
			changed := false

			// Map passes to state
			newState := mapPassesToState(story.Passes, ball)
			if ball.State != newState {
				ball.State = newState
				changed = true
			}

			// Clear blocked reason if completing
			if newState == session.StateComplete && ball.BlockedReason != "" {
				ball.BlockedReason = ""
				changed = true
			}

			if changed {
				ball.UpdateActivity()
				if err := store.UpdateBall(ball); err != nil {
					fmt.Printf("Warning: failed to update ball %s: %v\n", ball.ID, err)
					continue
				}
				updated++
				fmt.Printf("Updated: %s → %s\n", story.ID, newState)
			} else {
				skipped++
			}
		} else {
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
			ball.State = mapPassesToState(story.Passes, ball)

			// Mark as complete if passes is true
			if story.Passes {
				now := time.Now()
				ball.CompletedAt = &now
			}

			// Add story ID as tag for reference
			ball.AddTag(story.ID)

			if err := store.AppendBall(ball); err != nil {
				fmt.Printf("Warning: failed to create ball for %s: %v\n", story.ID, err)
				continue
			}
			created++
			fmt.Printf("Created: %s → %s (%s)\n", story.ID, ball.ID, ball.State)

			// Add to lookup for subsequent stories
			ballsByTitle[story.Title] = ball
		}
	}

	fmt.Printf("\nSync complete: %d created, %d updated, %d unchanged\n", created, updated, skipped)
	return nil
}

// mapPassesToState maps prd.json passes field to ball state
func mapPassesToState(passes bool, ball *session.Ball) session.BallState {
	if passes {
		return session.StateComplete
	}

	// If ball exists and was in progress, preserve that state
	if ball != nil && ball.State == session.StateInProgress {
		return session.StateInProgress
	}

	// Default to pending
	return session.StatePending
}

// mapPriorityNumber maps numeric priority to Priority enum
// Lower numbers = higher priority
func mapPriorityNumber(p int) session.Priority {
	switch {
	case p <= 2:
		return session.PriorityUrgent
	case p <= 5:
		return session.PriorityHigh
	case p <= 10:
		return session.PriorityMedium
	default:
		return session.PriorityLow
	}
}

// watchAndSync watches prd.json for changes and syncs on each change
func watchAndSync(prdPath, projectDir string) error {
	// Initial sync
	if err := syncPRDFile(prdPath, projectDir); err != nil {
		return err
	}

	// Create file watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}
	defer watcher.Close()

	// Watch the directory containing prd.json (fsnotify watches directories)
	prdDir := filepath.Dir(prdPath)
	prdBase := filepath.Base(prdPath)
	if err := watcher.Add(prdDir); err != nil {
		return fmt.Errorf("failed to watch directory: %w", err)
	}

	fmt.Printf("\nWatching %s for changes...\n", prdPath)

	// Debounce timer to avoid multiple syncs for rapid changes
	var debounceTimer *time.Timer
	debounceDelay := 500 * time.Millisecond

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}

			// Only react to writes/creates on prd.json
			if filepath.Base(event.Name) != prdBase {
				continue
			}
			if event.Op&(fsnotify.Write|fsnotify.Create) == 0 {
				continue
			}

			// Debounce: reset timer on each event
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			debounceTimer = time.AfterFunc(debounceDelay, func() {
				fmt.Printf("\n[%s] Detected change, syncing...\n", time.Now().Format("15:04:05"))
				if err := syncPRDFile(prdPath, projectDir); err != nil {
					fmt.Printf("Sync error: %v\n", err)
				}
			})

		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			// Don't exit on transient errors, just log
			if !strings.Contains(err.Error(), "too many open files") {
				fmt.Printf("Watch error: %v\n", err)
			}
		}
	}
}
