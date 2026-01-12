package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/ohare93/juggle/internal/session"
	"github.com/spf13/cobra"
)

var (
	syncWatch     bool
	syncWriteBack bool
	syncCheck     bool
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
  juggle sync ralph --watch

  # Write ball state back to prd.json
  juggle sync ralph --write-back

  # Check for conflicts without syncing
  juggle sync ralph --check`,
	RunE: runSyncRalph,
}

func init() {
	syncRalphCmd.Flags().BoolVarP(&syncWatch, "watch", "w", false, "Watch for changes and sync continuously")
	syncRalphCmd.Flags().BoolVar(&syncWriteBack, "write-back", false, "Write ball state back to prd.json")
	syncRalphCmd.Flags().BoolVar(&syncCheck, "check", false, "Check for conflicts without syncing")
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

// SyncConflict represents a conflict between prd.json and a ball
type SyncConflict struct {
	StoryID   string
	BallID    string
	Title     string
	FieldName string
	PRDValue  string
	BallValue string
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

	// If check mode, detect conflicts only
	if syncCheck {
		return checkConflicts(prdPath, cwd)
	}

	// If write-back mode, sync balls → prd.json
	if syncWriteBack {
		return writeToPRD(prdPath, cwd)
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

// mapStateToPasses maps ball state to prd.json passes field
func mapStateToPasses(state session.BallState) bool {
	return state == session.StateComplete || state == session.StateResearched
}

// mapPriorityToNumber maps Priority enum to numeric priority
// Lower numbers = higher priority
func mapPriorityToNumber(p session.Priority) int {
	switch p {
	case session.PriorityUrgent:
		return 1
	case session.PriorityHigh:
		return 4
	case session.PriorityMedium:
		return 7
	default:
		return 15
	}
}

// writeToPRD writes ball state back to prd.json
func writeToPRD(prdPath, projectDir string) error {
	// Read existing prd.json
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

	var updated, unchanged int

	// Update user stories based on ball state
	for i := range prd.UserStories {
		story := &prd.UserStories[i]

		// Find matching ball by title
		ball, exists := ballsByTitle[story.Title]
		if !exists {
			// No matching ball - skip
			unchanged++
			continue
		}

		// Map ball state to passes
		newPasses := mapStateToPasses(ball.State)
		if story.Passes != newPasses {
			story.Passes = newPasses
			updated++
			fmt.Printf("Updated: %s → passes: %t\n", story.ID, newPasses)
		} else {
			unchanged++
		}

		// Optionally update priority if it differs significantly
		newPriority := mapPriorityToNumber(ball.Priority)
		if story.Priority != newPriority {
			story.Priority = newPriority
		}

		// Update acceptance criteria if they've changed
		if len(ball.AcceptanceCriteria) > 0 {
			story.AcceptanceCriteria = ball.AcceptanceCriteria
		}
	}

	// Write updated prd.json
	updatedData, err := json.MarshalIndent(prd, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal prd.json: %w", err)
	}

	if err := os.WriteFile(prdPath, updatedData, 0644); err != nil {
		return fmt.Errorf("failed to write prd.json: %w", err)
	}

	fmt.Printf("\nWrite-back complete: %d updated, %d unchanged\n", updated, unchanged)
	return nil
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

// checkConflicts detects and reports conflicts between prd.json and balls
func checkConflicts(prdPath, projectDir string) error {
	conflicts, err := detectConflicts(prdPath, projectDir)
	if err != nil {
		return err
	}

	if len(conflicts) == 0 {
		fmt.Println("No conflicts detected. prd.json and balls are in sync.")
		return nil
	}

	fmt.Printf("Found %d conflict(s):\n\n", len(conflicts))

	// Group conflicts by story
	byStory := make(map[string][]SyncConflict)
	for _, c := range conflicts {
		byStory[c.StoryID] = append(byStory[c.StoryID], c)
	}

	// Sort story IDs for deterministic output order
	storyIDs := make([]string, 0, len(byStory))
	for storyID := range byStory {
		storyIDs = append(storyIDs, storyID)
	}
	sort.Strings(storyIDs)

	for _, storyID := range storyIDs {
		storyConflicts := byStory[storyID]
		first := storyConflicts[0]
		fmt.Printf("─── %s: %s ───\n", storyID, first.Title)
		fmt.Printf("Ball: %s\n\n", first.BallID)

		for _, c := range storyConflicts {
			fmt.Printf("  Field: %s\n", c.FieldName)
			fmt.Printf("    prd.json: %s\n", c.PRDValue)
			fmt.Printf("    ball:     %s\n\n", c.BallValue)
		}
	}

	fmt.Println("To resolve:")
	fmt.Println("  - Use 'juggle sync ralph' to apply prd.json values to balls")
	fmt.Println("  - Use 'juggle sync ralph --write-back' to apply ball values to prd.json")

	return nil
}

// detectConflicts finds differences between prd.json and matching balls
func detectConflicts(prdPath, projectDir string) ([]SyncConflict, error) {
	// Read prd.json
	data, err := os.ReadFile(prdPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read prd.json: %w", err)
	}

	var prd PRDFile
	if err := json.Unmarshal(data, &prd); err != nil {
		return nil, fmt.Errorf("failed to parse prd.json: %w", err)
	}

	// Create store for project
	store, err := NewStoreForCommand(projectDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create store: %w", err)
	}

	// Load existing balls
	balls, err := store.LoadBalls()
	if err != nil {
		return nil, fmt.Errorf("failed to load balls: %w", err)
	}

	// Build lookup by title (intent)
	ballsByTitle := make(map[string]*session.Ball)
	for _, ball := range balls {
		ballsByTitle[ball.Intent] = ball
	}

	var conflicts []SyncConflict

	for _, story := range prd.UserStories {
		ball, exists := ballsByTitle[story.Title]
		if !exists {
			// No matching ball - not a conflict, just new story
			continue
		}

		// Check state conflict
		prdState := mapPassesToState(story.Passes, nil)
		if ball.State != prdState {
			// Only report conflict if both sides have meaningful state changes
			// i.e., if prd says complete but ball is not complete, or vice versa
			isRealConflict := false
			if story.Passes && ball.State != session.StateComplete && ball.State != session.StateResearched {
				isRealConflict = true
			} else if !story.Passes && (ball.State == session.StateComplete || ball.State == session.StateResearched) {
				isRealConflict = true
			}

			if isRealConflict {
				conflicts = append(conflicts, SyncConflict{
					StoryID:   story.ID,
					BallID:    ball.ID,
					Title:     story.Title,
					FieldName: "state/passes",
					PRDValue:  fmt.Sprintf("passes=%t → %s", story.Passes, prdState),
					BallValue: string(ball.State),
				})
			}
		}

		// Check priority conflict
		prdPriority := mapPriorityNumber(story.Priority)
		if ball.Priority != prdPriority {
			conflicts = append(conflicts, SyncConflict{
				StoryID:   story.ID,
				BallID:    ball.ID,
				Title:     story.Title,
				FieldName: "priority",
				PRDValue:  fmt.Sprintf("%d (%s)", story.Priority, prdPriority),
				BallValue: string(ball.Priority),
			})
		}

		// Check acceptance criteria conflict
		if !stringSlicesEqual(story.AcceptanceCriteria, ball.AcceptanceCriteria) {
			conflicts = append(conflicts, SyncConflict{
				StoryID:   story.ID,
				BallID:    ball.ID,
				Title:     story.Title,
				FieldName: "acceptance_criteria",
				PRDValue:  formatACList(story.AcceptanceCriteria),
				BallValue: formatACList(ball.AcceptanceCriteria),
			})
		}
	}

	return conflicts, nil
}

// stringSlicesEqual compares two string slices for equality
func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// formatACList formats acceptance criteria for display
func formatACList(acs []string) string {
	if len(acs) == 0 {
		return "(none)"
	}
	if len(acs) == 1 {
		return acs[0]
	}
	return fmt.Sprintf("%d items: %s", len(acs), truncateStr(acs[0], 30))
}

// truncateStr shortens a string to maxLen characters
func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
