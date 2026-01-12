package integration_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ohare93/juggle/internal/session"
)

// TestUnarchiveCommand tests the unarchive command functionality
func TestUnarchiveCommand(t *testing.T) {
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{"UnarchiveWithDirectSyntax", testUnarchiveDirectSyntax},
		{"UnarchiveWithBallCommandSyntax", testUnarchiveBallCommandSyntax},
		{"UnarchiveBallNotInArchive", testUnarchiveBallNotInArchive},
		{"UnarchiveRestoresToReadyState", testUnarchiveRestoresToReadyState},
		{"UnarchiveRemovesFromArchive", testUnarchiveRemovesFromArchive},
		{"UnarchivePreservesMetadata", testUnarchivePreservesMetadata},
		{"UnarchiveWithShortID", testUnarchiveWithShortID},
		{"UnarchiveMultipleBallsInArchive", testUnarchiveMultipleBallsInArchive},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.test(t)
		})
	}
}

// testUnarchiveDirectSyntax tests: juggle unarchive <ball-id>
func testUnarchiveDirectSyntax(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	setupConfigWithTestProject(t, env)

	// 1. Create and complete a ball
	ball := env.CreateBall(t, "Test unarchive direct syntax", session.PriorityMedium)
	ballID := ball.ID

	// Mark as complete and archive
	ball.MarkComplete("All done")
	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}
	if err := store.ArchiveBall(ball); err != nil {
		t.Fatalf("Failed to archive ball: %v", err)
	}

	// 2. Verify it's archived
	archived, err := store.LoadArchivedBalls()
	if err != nil {
		t.Fatalf("Failed to load archived balls: %v", err)
	}
	if len(archived) != 1 {
		t.Fatalf("Expected 1 archived ball, got %d", len(archived))
	}
	if archived[0].ID != ballID {
		t.Fatalf("Expected ball ID %s, got %s", ballID, archived[0].ID)
	}

	// Verify it's not in active balls
	activeBalls, err := store.LoadBalls()
	if err != nil {
		t.Fatalf("Failed to load active balls: %v", err)
	}
	if len(activeBalls) != 0 {
		t.Fatalf("Expected 0 active balls, got %d", len(activeBalls))
	}

	// 3. Unarchive using direct syntax
	output := runJuggleCommand(t, env.ProjectDir, "unarchive", ballID)
	if !strings.Contains(output, "Unarchived") {
		t.Errorf("Expected 'Unarchived' in output, got: %s", output)
	}
	if !strings.Contains(output, ballID) {
		t.Errorf("Expected ball ID %s in output, got: %s", ballID, output)
	}

	// 4. Verify it's back in active balls as ready
	activeBalls, err = store.LoadBalls()
	if err != nil {
		t.Fatalf("Failed to load active balls after unarchive: %v", err)
	}
	if len(activeBalls) != 1 {
		t.Fatalf("Expected 1 active ball, got %d", len(activeBalls))
	}
	if activeBalls[0].ID != ballID {
		t.Fatalf("Expected ball ID %s, got %s", ballID, activeBalls[0].ID)
	}
	if activeBalls[0].State != session.StatePending {
		t.Errorf("Expected state %s, got %s", session.StatePending, activeBalls[0].State)
	}
	// JuggleState is a legacy field, no need to check it in new tests
	// Ball should be in pending state which means not blocked

	// 5. Verify it's removed from archive
	archived, err = store.LoadArchivedBalls()
	if err != nil {
		t.Fatalf("Failed to load archived balls after unarchive: %v", err)
	}
	if len(archived) != 0 {
		t.Errorf("Expected 0 archived balls, got %d", len(archived))
	}
}

// testUnarchiveBallCommandSyntax tests: juggle <ball-id> unarchive
func testUnarchiveBallCommandSyntax(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	setupConfigWithTestProject(t, env)

	// 1. Create and archive a ball
	ball := env.CreateBall(t, "Test ball command syntax", session.PriorityHigh)
	ballID := ball.ID

	ball.MarkComplete("Completed")
	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}
	if err := store.ArchiveBall(ball); err != nil {
		t.Fatalf("Failed to archive ball: %v", err)
	}

	// 2. Verify it's archived
	archived, err := store.LoadArchivedBalls()
	if err != nil {
		t.Fatalf("Failed to load archived balls: %v", err)
	}
	if len(archived) != 1 {
		t.Fatalf("Expected 1 archived ball, got %d", len(archived))
	}

	// 3. Unarchive using ball command syntax
	output := runJuggleCommand(t, env.ProjectDir, ballID, "unarchive")
	if !strings.Contains(output, "Unarchived") {
		t.Errorf("Expected 'Unarchived' in output, got: %s", output)
	}
	if !strings.Contains(output, ballID) {
		t.Errorf("Expected ball ID %s in output, got: %s", ballID, output)
	}

	// 4. Verify restoration
	activeBalls, err := store.LoadBalls()
	if err != nil {
		t.Fatalf("Failed to load active balls: %v", err)
	}
	if len(activeBalls) != 1 {
		t.Fatalf("Expected 1 active ball, got %d", len(activeBalls))
	}
	if activeBalls[0].ID != ballID {
		t.Errorf("Expected ball ID %s, got %s", ballID, activeBalls[0].ID)
	}
	if activeBalls[0].State != session.StatePending {
		t.Errorf("Expected state %s, got %s", session.StatePending, activeBalls[0].State)
	}

	// 5. Verify removal from archive
	archived, err = store.LoadArchivedBalls()
	if err != nil {
		t.Fatalf("Failed to load archived balls: %v", err)
	}
	if len(archived) != 0 {
		t.Errorf("Expected 0 archived balls, got %d", len(archived))
	}
}

// testUnarchiveBallNotInArchive tests error when ball doesn't exist in archive
func testUnarchiveBallNotInArchive(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	setupConfigWithTestProject(t, env)

	// Try to unarchive a nonexistent ball
	output, exitCode := runJuggleCommandWithError(t, env.ProjectDir, "unarchive", "nonexistent-1")
	if exitCode == 0 {
		t.Errorf("Expected non-zero exit code, got 0")
	}
	if !strings.Contains(output, "not found") {
		t.Errorf("Expected 'not found' in error output, got: %s", output)
	}
}

// testUnarchiveRestoresToReadyState tests that state is correctly set to ready
func testUnarchiveRestoresToReadyState(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	setupConfigWithTestProject(t, env)

	// Create ball with specific state progression
	ball := env.CreateBall(t, "State progression test", session.PriorityLow)
	ballID := ball.ID

	// Set to in_progress and then complete
	ball.SetState(session.StateInProgress)

	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// Mark complete and archive
	ball.MarkComplete("Final state")
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball to complete: %v", err)
	}
	if err := store.ArchiveBall(ball); err != nil {
		t.Fatalf("Failed to archive ball: %v", err)
	}

	// Verify archived state
	archived, err := store.LoadArchivedBalls()
	if err != nil {
		t.Fatalf("Failed to load archived balls: %v", err)
	}
	if len(archived) != 1 {
		t.Fatalf("Expected 1 archived ball, got %d", len(archived))
	}
	if archived[0].State != session.StateComplete {
		t.Errorf("Expected archived state %s, got %s", session.StateComplete, archived[0].State)
	}
	// CompletedAt is omitempty in JSON, so it may not round-trip if timezone/precision differs
	// Just verify the completion note for test setup validation
	if archived[0].CompletionNote != "Final state" {
		t.Errorf("Expected completion note 'Final state', got '%s'", archived[0].CompletionNote)
	}

	// Unarchive
	runJuggleCommand(t, env.ProjectDir, "unarchive", ballID)

	// Verify state is reset to ready
	activeBalls, err := store.LoadBalls()
	if err != nil {
		t.Fatalf("Failed to load active balls: %v", err)
	}
	if len(activeBalls) != 1 {
		t.Fatalf("Expected 1 active ball, got %d", len(activeBalls))
	}

	restoredBall := activeBalls[0]
	if restoredBall.State != session.StatePending {
		t.Errorf("Expected state %s, got %s", session.StatePending, restoredBall.State)
	}
	if restoredBall.BlockedReason != "" {
		t.Errorf("Expected empty BlockedReason, got '%s'", restoredBall.BlockedReason)
	}
	if restoredBall.CompletedAt != nil {
		t.Errorf("Expected nil CompletedAt, got %v", restoredBall.CompletedAt)
	}
	if restoredBall.CompletionNote != "" {
		t.Errorf("Expected empty CompletionNote, got '%s'", restoredBall.CompletionNote)
	}
}

// testUnarchiveRemovesFromArchive tests that unarchiving removes ball from archive
func testUnarchiveRemovesFromArchive(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	setupConfigWithTestProject(t, env)

	store := env.GetStore(t)

	// Create and archive multiple balls
	ball1 := env.CreateBall(t, "First ball", session.PriorityMedium)
	ball2 := env.CreateBall(t, "Second ball", session.PriorityMedium)
	ball3 := env.CreateBall(t, "Third ball", session.PriorityMedium)

	for _, ball := range []*session.Ball{ball1, ball2, ball3} {
		ball.MarkComplete("Done")
		if err := store.UpdateBall(ball); err != nil {
			t.Fatalf("Failed to update ball: %v", err)
		}
		if err := store.ArchiveBall(ball); err != nil {
			t.Fatalf("Failed to archive ball: %v", err)
		}
	}

	// Verify all three are archived
	archived, err := store.LoadArchivedBalls()
	if err != nil {
		t.Fatalf("Failed to load archived balls: %v", err)
	}
	if len(archived) != 3 {
		t.Fatalf("Expected 3 archived balls, got %d", len(archived))
	}

	// Unarchive the middle one
	runJuggleCommand(t, env.ProjectDir, "unarchive", ball2.ID)

	// Verify only ball2 is removed from archive
	archived, err = store.LoadArchivedBalls()
	if err != nil {
		t.Fatalf("Failed to load archived balls: %v", err)
	}
	if len(archived) != 2 {
		t.Fatalf("Expected 2 archived balls after unarchive, got %d", len(archived))
	}

	// Verify ball1 and ball3 are still archived
	archivedIDs := []string{archived[0].ID, archived[1].ID}
	hasB1, hasB3, hasB2 := false, false, false
	for _, id := range archivedIDs {
		if id == ball1.ID {
			hasB1 = true
		}
		if id == ball3.ID {
			hasB3 = true
		}
		if id == ball2.ID {
			hasB2 = true
		}
	}
	if !hasB1 || !hasB3 {
		t.Errorf("Expected ball1 and ball3 in archive, got IDs: %v", archivedIDs)
	}
	if hasB2 {
		t.Errorf("Expected ball2 NOT in archive, but it was found")
	}

	// Verify ball2 is in active balls
	activeBalls, err := store.LoadBalls()
	if err != nil {
		t.Fatalf("Failed to load active balls: %v", err)
	}
	if len(activeBalls) != 1 {
		t.Fatalf("Expected 1 active ball, got %d", len(activeBalls))
	}
	if activeBalls[0].ID != ball2.ID {
		t.Errorf("Expected active ball to be %s, got %s", ball2.ID, activeBalls[0].ID)
	}
}

// testUnarchivePreservesMetadata tests that todos, tags, priorities, timestamps are preserved
func testUnarchivePreservesMetadata(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	setupConfigWithTestProject(t, env)

	// Create ball with rich metadata
	ball := env.CreateBall(t, "Rich metadata ball", session.PriorityUrgent)
	ballID := ball.ID

	// Set acceptance criteria
	ball.SetAcceptanceCriteria([]string{"First criterion", "Second criterion"})

	// Add tags
	ball.Tags = []string{"feature", "backend", "urgent"}

	// Set timestamps
	startTime := time.Now().Add(-2 * time.Hour)
	ball.StartedAt = startTime
	ball.UpdateCount = 5

	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// Complete and archive (MarkComplete will update LastActivity)
	completedTime := time.Now()
	ball.MarkComplete("All features done")
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball to complete: %v", err)
	}
	if err := store.ArchiveBall(ball); err != nil {
		t.Fatalf("Failed to archive ball: %v", err)
	}

	// Unarchive
	runJuggleCommand(t, env.ProjectDir, "unarchive", ballID)

	// Verify metadata is preserved
	activeBalls, err := store.LoadBalls()
	if err != nil {
		t.Fatalf("Failed to load active balls: %v", err)
	}
	if len(activeBalls) != 1 {
		t.Fatalf("Expected 1 active ball, got %d", len(activeBalls))
	}

	restored := activeBalls[0]

	// Check basic fields
	if restored.ID != ballID {
		t.Errorf("Expected ID %s, got %s", ballID, restored.ID)
	}
	if restored.Title != "Rich metadata ball" {
		t.Errorf("Expected intent 'Rich metadata ball', got '%s'", restored.Title)
	}
	if restored.Priority != session.PriorityUrgent {
		t.Errorf("Expected priority %s, got %s", session.PriorityUrgent, restored.Priority)
	}

	// Check acceptance criteria are preserved
	if len(restored.AcceptanceCriteria) != 2 {
		t.Fatalf("Expected 2 acceptance criteria, got %d", len(restored.AcceptanceCriteria))
	}
	if restored.AcceptanceCriteria[0] != "First criterion" {
		t.Errorf("Expected first criterion 'First criterion', got '%s'", restored.AcceptanceCriteria[0])
	}
	if restored.AcceptanceCriteria[1] != "Second criterion" {
		t.Errorf("Expected second criterion 'Second criterion', got '%s'", restored.AcceptanceCriteria[1])
	}

	// Check tags are preserved
	if len(restored.Tags) != 3 {
		t.Fatalf("Expected 3 tags, got %d", len(restored.Tags))
	}
	expectedTags := map[string]bool{"feature": true, "backend": true, "urgent": true}
	for _, tag := range restored.Tags {
		if !expectedTags[tag] {
			t.Errorf("Unexpected tag: %s", tag)
		}
		delete(expectedTags, tag)
	}
	if len(expectedTags) > 0 {
		t.Errorf("Missing tags: %v", expectedTags)
	}

	// Check timestamps are preserved (compare Unix seconds to avoid nanosecond issues)
	if restored.StartedAt.Unix() != startTime.Unix() {
		t.Errorf("StartedAt not preserved: expected %v, got %v", startTime, restored.StartedAt)
	}
	// LastActivity should be around the completion time (MarkComplete calls UpdateActivity)
	// Allow for a few seconds of difference due to test execution time
	timeDiff := restored.LastActivity.Sub(completedTime).Abs()
	if timeDiff > 5*time.Second {
		t.Errorf("LastActivity not near completion time: expected around %v, got %v (diff: %v)", completedTime, restored.LastActivity, timeDiff)
	}
	if restored.UpdateCount != 5 {
		t.Errorf("Expected UpdateCount 5, got %d", restored.UpdateCount)
	}

	// Check working directory is set
	if restored.WorkingDir != env.ProjectDir {
		t.Errorf("Expected WorkingDir %s, got %s", env.ProjectDir, restored.WorkingDir)
	}
}

// testUnarchiveWithShortID tests unarchiving using short ID
func testUnarchiveWithShortID(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	setupConfigWithTestProject(t, env)

	// Create and archive a ball
	ball := env.CreateBall(t, "Short ID test", session.PriorityMedium)
	ballID := ball.ID
	shortID := ball.ShortID()

	ball.MarkComplete("Done")
	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}
	if err := store.ArchiveBall(ball); err != nil {
		t.Fatalf("Failed to archive ball: %v", err)
	}

	// Unarchive using short ID
	output := runJuggleCommand(t, env.ProjectDir, "unarchive", shortID)
	if !strings.Contains(output, "Unarchived") {
		t.Errorf("Expected 'Unarchived' in output, got: %s", output)
	}

	// Verify it's restored
	activeBalls, err := store.LoadBalls()
	if err != nil {
		t.Fatalf("Failed to load active balls: %v", err)
	}
	if len(activeBalls) != 1 {
		t.Fatalf("Expected 1 active ball, got %d", len(activeBalls))
	}
	if activeBalls[0].ID != ballID {
		t.Errorf("Expected ball ID %s, got %s", ballID, activeBalls[0].ID)
	}
}

// testUnarchiveMultipleBallsInArchive tests unarchiving when multiple balls are archived
func testUnarchiveMultipleBallsInArchive(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	setupConfigWithTestProject(t, env)

	store := env.GetStore(t)

	// Create and archive 5 balls
	balls := make([]*session.Ball, 5)
	for i := 0; i < 5; i++ {
		ball := env.CreateBall(t, fmt.Sprintf("Ball %d", i+1), session.PriorityMedium)
		ball.MarkComplete(fmt.Sprintf("Completed %d", i+1))
		if err := store.UpdateBall(ball); err != nil {
			t.Fatalf("Failed to update ball %d: %v", i+1, err)
		}
		if err := store.ArchiveBall(ball); err != nil {
			t.Fatalf("Failed to archive ball %d: %v", i+1, err)
		}
		balls[i] = ball
	}

	// Verify all 5 are archived
	archived, err := store.LoadArchivedBalls()
	if err != nil {
		t.Fatalf("Failed to load archived balls: %v", err)
	}
	if len(archived) != 5 {
		t.Fatalf("Expected 5 archived balls, got %d", len(archived))
	}

	// Unarchive ball 3 (middle one)
	targetBall := balls[2]
	runJuggleCommand(t, env.ProjectDir, "unarchive", targetBall.ID)

	// Verify 4 remain in archive
	archived, err = store.LoadArchivedBalls()
	if err != nil {
		t.Fatalf("Failed to load archived balls: %v", err)
	}
	if len(archived) != 4 {
		t.Fatalf("Expected 4 archived balls, got %d", len(archived))
	}

	// Verify target ball is in active
	activeBalls, err := store.LoadBalls()
	if err != nil {
		t.Fatalf("Failed to load active balls: %v", err)
	}
	if len(activeBalls) != 1 {
		t.Fatalf("Expected 1 active ball, got %d", len(activeBalls))
	}
	if activeBalls[0].ID != targetBall.ID {
		t.Errorf("Expected active ball %s, got %s", targetBall.ID, activeBalls[0].ID)
	}

	// Unarchive ball 1 and ball 5
	runJuggleCommand(t, env.ProjectDir, "unarchive", balls[0].ID)
	runJuggleCommand(t, env.ProjectDir, "unarchive", balls[4].ID)

	// Verify 2 remain in archive
	archived, err = store.LoadArchivedBalls()
	if err != nil {
		t.Fatalf("Failed to load archived balls: %v", err)
	}
	if len(archived) != 2 {
		t.Fatalf("Expected 2 archived balls, got %d", len(archived))
	}

	// Verify 3 are in active
	activeBalls, err = store.LoadBalls()
	if err != nil {
		t.Fatalf("Failed to load active balls: %v", err)
	}
	if len(activeBalls) != 3 {
		t.Fatalf("Expected 3 active balls, got %d", len(activeBalls))
	}
}

// runJuggleCommand runs a juggle command and returns its output
func runJuggleCommand(t *testing.T, workingDir string, args ...string) string {
	t.Helper()

	// Find juggle binary in the repo root
	// From workingDir (temp test dir), go up to repo root
	jugglerRoot := "/home/jmo/Development/juggler"
	juggleBinary := filepath.Join(jugglerRoot, "juggle")

	// Build juggle binary if it doesn't exist
	if _, err := os.Stat(juggleBinary); os.IsNotExist(err) {
		buildCmd := exec.Command("go", "build", "-o", "juggle", "./cmd/juggle")
		buildCmd.Dir = jugglerRoot
		if output, err := buildCmd.CombinedOutput(); err != nil {
			t.Fatalf("Failed to build juggle: %v\nOutput: %s", err, output)
		}
	}

	// Inject --config-home flag pointing to test config directory
	configHome := filepath.Join(workingDir, "..", "config")
	allArgs := append([]string{"--config-home", configHome}, args...)

	cmd := exec.Command(juggleBinary, allArgs...)
	cmd.Dir = workingDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Command failed: %v\nOutput: %s", err, output)
	}

	return string(output)
}

// runJuggleCommandWithError runs a command expecting an error and returns output and exit code
func runJuggleCommandWithError(t *testing.T, workingDir string, args ...string) (string, int) {
	t.Helper()

	jugglerRoot := "/home/jmo/Development/juggler"
	juggleBinary := filepath.Join(jugglerRoot, "juggle")

	// Build binary if needed
	if _, err := os.Stat(juggleBinary); os.IsNotExist(err) {
		buildCmd := exec.Command("go", "build", "-o", "juggle", "./cmd/juggle")
		buildCmd.Dir = jugglerRoot
		if output, err := buildCmd.CombinedOutput(); err != nil {
			t.Fatalf("Failed to build juggle: %v\nOutput: %s", err, output)
		}
	}

	// Inject --config-home flag pointing to test config directory
	configHome := filepath.Join(workingDir, "..", "config")
	allArgs := append([]string{"--config-home", configHome}, args...)

	cmd := exec.Command(juggleBinary, allArgs...)
	cmd.Dir = workingDir

	output, err := cmd.CombinedOutput()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}

	return strings.TrimSpace(string(output)), exitCode
}

// setupConfigWithTestProject creates a config that includes the test project directory
func setupConfigWithTestProject(t *testing.T, env *TestEnv) {
	t.Helper()

	// Create config file that includes our test project directory in search paths
	// DiscoverProjects checks if path/.juggler exists, so we need to add ProjectDir directly
	// Note: config file must be at configHome/.juggler/config.json
	jugglerConfigDir := filepath.Join(env.ConfigHome, ".juggler")
	if err := os.MkdirAll(jugglerConfigDir, 0755); err != nil {
		t.Fatalf("Failed to create juggler config dir: %v", err)
	}

	configPath := filepath.Join(jugglerConfigDir, "config.json")
	configContent := fmt.Sprintf(`{
  "search_paths": ["%s"]
}`, env.ProjectDir)

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}
}
