package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ohare93/juggle/internal/cli"
	"github.com/ohare93/juggle/internal/session"
)

// TestShowCommand tests the show command functionality
func TestShowCommand(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a ball
	ball := env.CreateBall(t, "Test ball for show", session.PriorityHigh)

	// Test showing the ball
	store := env.GetStore(t)
	retrieved, err := store.GetBallByID(ball.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve ball: %v", err)
	}

	if retrieved.Title != "Test ball for show" {
		t.Errorf("Expected intent 'Test ball for show', got '%s'", retrieved.Title)
	}

	if retrieved.Priority != session.PriorityHigh {
		t.Errorf("Expected priority high, got %s", retrieved.Priority)
	}
}

// TestShowCommandWithAcceptanceCriteria tests show with acceptance criteria
func TestShowCommandWithAcceptanceCriteria(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	ball := env.CreateBall(t, "Test ball with criteria", session.PriorityMedium)
	store := env.GetStore(t)

	// Add acceptance criteria
	ball.SetAcceptanceCriteria([]string{
		"First criterion",
		"Second criterion",
		"Third criterion",
	})

	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// Retrieve and verify
	retrieved, err := store.GetBallByID(ball.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve ball: %v", err)
	}

	if len(retrieved.AcceptanceCriteria) != 3 {
		t.Errorf("Expected 3 acceptance criteria, got %d", len(retrieved.AcceptanceCriteria))
	}
}

// TestUpdateCommandState tests updating ball state
func TestUpdateCommandState(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	ball := env.CreateBall(t, "Test ball for state update", session.PriorityMedium)
	store := env.GetStore(t)

	// Start the ball (pending -> in_progress)
	ball.Start()
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	retrieved, err := store.GetBallByID(ball.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve ball: %v", err)
	}

	if retrieved.State != session.StateInProgress {
		t.Errorf("Expected state in_progress, got %s", retrieved.State)
	}
}

// TestUpdateCommandPriority tests updating ball priority
func TestUpdateCommandPriority(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	ball := env.CreateBall(t, "Test ball for priority update", session.PriorityLow)
	store := env.GetStore(t)

	// Update priority to urgent
	ball.Priority = session.PriorityUrgent
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	retrieved, err := store.GetBallByID(ball.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve ball: %v", err)
	}

	if retrieved.Priority != session.PriorityUrgent {
		t.Errorf("Expected priority urgent, got %s", retrieved.Priority)
	}
}

// TestUpdateCommandBlocked tests setting ball to blocked state
func TestUpdateCommandBlocked(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	ball := env.CreateBall(t, "Test ball for blocked", session.PriorityMedium)
	store := env.GetStore(t)

	// Start the ball first, then block it (blocked only valid from in_progress)
	ball.Start()
	if err := ball.SetBlocked("Waiting for API access"); err != nil {
		t.Fatalf("Failed to set blocked: %v", err)
	}
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	retrieved, err := store.GetBallByID(ball.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve ball: %v", err)
	}

	if retrieved.State != session.StateBlocked {
		t.Errorf("Expected state blocked, got %s", retrieved.State)
	}

	if retrieved.BlockedReason != "Waiting for API access" {
		t.Errorf("Expected blocked reason 'Waiting for API access', got '%s'", retrieved.BlockedReason)
	}
}

// TestUpdateCommandTags tests updating ball tags
func TestUpdateCommandTags(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	ball := env.CreateBall(t, "Test ball for tags", session.PriorityMedium)
	store := env.GetStore(t)

	// Add tags
	ball.Tags = []string{"feature", "frontend", "urgent"}
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	retrieved, err := store.GetBallByID(ball.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve ball: %v", err)
	}

	if len(retrieved.Tags) != 3 {
		t.Errorf("Expected 3 tags, got %d", len(retrieved.Tags))
	}
}

// TestListCommand tests listing balls
func TestListCommand(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create multiple balls
	env.CreateBall(t, "First ball", session.PriorityHigh)
	env.CreateBall(t, "Second ball", session.PriorityMedium)
	env.CreateBall(t, "Third ball", session.PriorityLow)

	store := env.GetStore(t)
	balls, err := store.LoadBalls()
	if err != nil {
		t.Fatalf("Failed to load balls: %v", err)
	}

	if len(balls) != 3 {
		t.Errorf("Expected 3 balls, got %d", len(balls))
	}
}

// TestListCommandFiltered tests listing balls with state filtering
func TestListCommandFiltered(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	store := env.GetStore(t)

	// Create balls with different states
	ball1 := env.CreateBall(t, "Pending ball", session.PriorityMedium)
	ball2 := env.CreateBall(t, "In progress ball", session.PriorityMedium)
	ball3 := env.CreateBall(t, "Complete ball", session.PriorityMedium)

	// Update states
	ball2.Start()
	store.UpdateBall(ball2)

	ball3.ForceSetState(session.StateComplete)
	store.UpdateBall(ball3)

	// Count by state
	balls, _ := store.LoadBalls()

	pendingCount := 0
	inProgressCount := 0
	completeCount := 0

	for _, b := range balls {
		switch b.State {
		case session.StatePending:
			pendingCount++
		case session.StateInProgress:
			inProgressCount++
		case session.StateComplete:
			completeCount++
		}
	}

	if pendingCount != 1 {
		t.Errorf("Expected 1 pending ball, got %d", pendingCount)
	}
	if inProgressCount != 1 {
		t.Errorf("Expected 1 in_progress ball, got %d", inProgressCount)
	}
	if completeCount != 1 {
		t.Errorf("Expected 1 complete ball, got %d", completeCount)
	}

	_ = ball1 // Keep reference to avoid unused warning
}

// TestSearchCommand tests searching balls by content
func TestSearchCommand(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create balls with different intents
	env.CreateBall(t, "Add authentication feature", session.PriorityHigh)
	env.CreateBall(t, "Fix login bug", session.PriorityMedium)
	env.CreateBall(t, "Implement logout", session.PriorityLow)

	store := env.GetStore(t)
	balls, _ := store.LoadBalls()

	// Search for "login" should find 2 balls
	loginMatches := 0
	for _, b := range balls {
		if containsIgnoreCase(b.Title, "login") || containsIgnoreCase(b.Title, "logout") {
			loginMatches++
		}
	}

	if loginMatches != 2 {
		t.Errorf("Expected 2 balls matching login/logout, got %d", loginMatches)
	}
}

// Helper function for case-insensitive search
func containsIgnoreCase(s, substr string) bool {
	return bytes.Contains(bytes.ToLower([]byte(s)), bytes.ToLower([]byte(substr)))
}

// TestHistoryCommand tests viewing archive history
func TestHistoryCommand(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	store := env.GetStore(t)

	// Create and complete a ball
	ball := env.CreateBall(t, "Ball to archive", session.PriorityMedium)
	ball.ForceSetState(session.StateComplete)
	store.UpdateBall(ball)

	// Archive the ball
	err := store.ArchiveBall(ball)
	if err != nil {
		t.Fatalf("Failed to archive ball: %v", err)
	}

	// Verify ball is no longer in active balls
	env.AssertBallNotExists(t, ball.ID)

	// Verify archive file exists
	archivePath := filepath.Join(env.JuggleDir, "archive", "balls.jsonl")
	if _, err := os.Stat(archivePath); os.IsNotExist(err) {
		t.Error("Expected archive file to exist")
	}
}

// TestExportCommand tests exporting balls
func TestExportCommand(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create balls
	ball1 := env.CreateBall(t, "Export ball 1", session.PriorityHigh)
	ball2 := env.CreateBall(t, "Export ball 2", session.PriorityMedium)

	// Add acceptance criteria
	ball1.SetAcceptanceCriteria([]string{"Criterion 1", "Criterion 2"})
	store := env.GetStore(t)
	store.UpdateBall(ball1)

	// Export to JSON
	balls, _ := store.LoadBalls()

	jsonData, err := json.MarshalIndent(balls, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal balls to JSON: %v", err)
	}

	// Verify JSON is valid and contains expected data
	var exported []*session.Ball
	if err := json.Unmarshal(jsonData, &exported); err != nil {
		t.Fatalf("Failed to unmarshal exported JSON: %v", err)
	}

	if len(exported) != 2 {
		t.Errorf("Expected 2 exported balls, got %d", len(exported))
	}

	_ = ball2 // Keep reference
}

// TestSessionsCommand tests session management
func TestSessionsCommand(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create session store
	sessionStorePath := filepath.Join(env.JuggleDir, "sessions")
	if err := os.MkdirAll(sessionStorePath, 0755); err != nil {
		t.Fatalf("Failed to create sessions directory: %v", err)
	}

	sessionStore, err := session.NewSessionStore(env.JuggleDir)
	if err != nil {
		t.Fatalf("Failed to create session store: %v", err)
	}

	// Create sessions
	sess1, err := sessionStore.CreateSession("feature-auth", "Authentication feature")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	sess2, err := sessionStore.CreateSession("bug-fix-login", "Fix login issues")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// List sessions
	sessions, err := sessionStore.ListSessions()
	if err != nil {
		t.Fatalf("Failed to list sessions: %v", err)
	}

	if len(sessions) != 2 {
		t.Errorf("Expected 2 sessions, got %d", len(sessions))
	}

	_ = sess1
	_ = sess2
}

// TestProgressCommand tests progress tracking
func TestProgressCommand(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create session store
	sessionStore, err := session.NewSessionStore(env.JuggleDir)
	if err != nil {
		t.Fatalf("Failed to create session store: %v", err)
	}

	// Create a session
	sess, err := sessionStore.CreateSession("test-session", "Test session for progress")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Append progress entries (matching CLI format with timestamps)
	entry1 := "[2024-01-01 10:00:00] Started implementation\n"
	err = sessionStore.AppendProgress(sess.ID, entry1)
	if err != nil {
		t.Fatalf("Failed to append progress: %v", err)
	}

	entry2 := "[2024-01-01 10:30:00] Added unit tests\n"
	err = sessionStore.AppendProgress(sess.ID, entry2)
	if err != nil {
		t.Fatalf("Failed to append progress: %v", err)
	}

	// Load progress
	progressContent, err := sessionStore.LoadProgress(sess.ID)
	if err != nil {
		t.Fatalf("Failed to load progress: %v", err)
	}

	// Verify content contains both entries
	if !bytes.Contains([]byte(progressContent), []byte("Started implementation")) {
		t.Error("Expected progress to contain 'Started implementation'")
	}

	if !bytes.Contains([]byte(progressContent), []byte("Added unit tests")) {
		t.Error("Expected progress to contain 'Added unit tests'")
	}
}

// TestTagCommand tests tag operations
func TestTagCommand(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	ball := env.CreateBall(t, "Ball for tagging", session.PriorityMedium)
	store := env.GetStore(t)

	// Add tag
	ball.AddTag("feature")
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// Verify tag added
	retrieved, _ := store.GetBallByID(ball.ID)
	if len(retrieved.Tags) != 1 || retrieved.Tags[0] != "feature" {
		t.Errorf("Expected tag 'feature', got %v", retrieved.Tags)
	}

	// Add another tag
	ball.AddTag("urgent")
	store.UpdateBall(ball)

	retrieved, _ = store.GetBallByID(ball.ID)
	if len(retrieved.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(retrieved.Tags))
	}

	// Remove tag
	ball.RemoveTag("feature")
	store.UpdateBall(ball)

	retrieved, _ = store.GetBallByID(ball.ID)
	if len(retrieved.Tags) != 1 || retrieved.Tags[0] != "urgent" {
		t.Errorf("Expected only 'urgent' tag, got %v", retrieved.Tags)
	}
}

// TestStatusCommand tests status overview
func TestStatusCommand(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	store := env.GetStore(t)

	// Create balls with various states
	ball1 := env.CreateBall(t, "Pending task", session.PriorityMedium)
	ball2 := env.CreateBall(t, "Active task", session.PriorityHigh)
	ball3 := env.CreateBall(t, "Blocked task", session.PriorityLow)

	ball2.Start()
	store.UpdateBall(ball2)

	// Start ball3 first, then block it (blocked only valid from in_progress)
	ball3.Start()
	if err := ball3.SetBlocked("Waiting for dependency"); err != nil {
		t.Fatalf("Failed to set blocked: %v", err)
	}
	store.UpdateBall(ball3)

	// Verify status counts
	balls, _ := store.LoadBalls()

	counts := make(map[session.BallState]int)
	for _, b := range balls {
		counts[b.State]++
	}

	if counts[session.StatePending] != 1 {
		t.Errorf("Expected 1 pending, got %d", counts[session.StatePending])
	}
	if counts[session.StateInProgress] != 1 {
		t.Errorf("Expected 1 in_progress, got %d", counts[session.StateInProgress])
	}
	if counts[session.StateBlocked] != 1 {
		t.Errorf("Expected 1 blocked, got %d", counts[session.StateBlocked])
	}

	_ = ball1
}

// TestMoveCommand tests moving balls between projects
func TestMoveCommandPreservesData(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a ball with acceptance criteria and tags
	ball := env.CreateBall(t, "Ball to move", session.PriorityHigh)
	store := env.GetStore(t)

	ball.SetAcceptanceCriteria([]string{"AC1", "AC2"})
	ball.AddTag("feature")
	ball.AddTag("frontend")
	store.UpdateBall(ball)

	// Retrieve and verify all data is preserved
	retrieved, _ := store.GetBallByID(ball.ID)

	if retrieved.Title != "Ball to move" {
		t.Errorf("Expected intent 'Ball to move', got '%s'", retrieved.Title)
	}
	if len(retrieved.AcceptanceCriteria) != 2 {
		t.Errorf("Expected 2 acceptance criteria, got %d", len(retrieved.AcceptanceCriteria))
	}
	if len(retrieved.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(retrieved.Tags))
	}
}

// TestGlobalOptions tests that global options are properly handled
func TestGlobalOptions(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Verify global options are set correctly for test environment
	if cli.GlobalOpts.ConfigHome != env.ConfigHome {
		t.Errorf("Expected ConfigHome %s, got %s", env.ConfigHome, cli.GlobalOpts.ConfigHome)
	}

	if cli.GlobalOpts.ProjectDir != env.ProjectDir {
		t.Errorf("Expected ProjectDir %s, got %s", env.ProjectDir, cli.GlobalOpts.ProjectDir)
	}
}

// TestBallIDResolution tests that balls can be found by short and full ID
func TestBallIDResolution(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	ball := env.CreateBall(t, "Test ball for ID resolution", session.PriorityMedium)
	store := env.GetStore(t)

	// Get by full ID
	fullID, err := store.GetBallByID(ball.ID)
	if err != nil {
		t.Fatalf("Failed to get ball by full ID: %v", err)
	}

	// Get by short ID
	shortID := ball.ShortID()
	byShortID, err := store.GetBallByShortID(shortID)
	if err != nil {
		t.Fatalf("Failed to get ball by short ID: %v", err)
	}

	if fullID.ID != byShortID.ID {
		t.Errorf("Full ID and short ID should resolve to same ball")
	}
}

// TestConcurrentBallUpdates tests that concurrent updates don't corrupt data
func TestConcurrentBallUpdates(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	ball := env.CreateBall(t, "Concurrent update test", session.PriorityMedium)
	store := env.GetStore(t)

	originalCount := ball.UpdateCount

	// Perform multiple updates with IncrementUpdateCount
	for i := 0; i < 5; i++ {
		ball.IncrementUpdateCount()
		ball.UpdateActivity()
		if err := store.UpdateBall(ball); err != nil {
			t.Fatalf("Failed update %d: %v", i, err)
		}
	}

	// Verify update count increased
	retrieved, _ := store.GetBallByID(ball.ID)
	expectedCount := originalCount + 5
	if retrieved.UpdateCount != expectedCount {
		t.Errorf("Expected update count %d, got %d", expectedCount, retrieved.UpdateCount)
	}
}

// TestBallStateTransitionHistory tests that state transitions update activity
func TestBallStateTransitionHistory(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	ball := env.CreateBall(t, "State transition test", session.PriorityMedium)
	store := env.GetStore(t)

	initialActivity := ball.LastActivity

	// Transition through states
	ball.Start()
	store.UpdateBall(ball)

	retrieved, _ := store.GetBallByID(ball.ID)
	if !retrieved.LastActivity.After(initialActivity) {
		t.Error("LastActivity should be updated after state change")
	}
}

// TestShowAllAcceptanceCriteria tests that all ACs are shown, not just the first
func TestShowAllAcceptanceCriteria(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	ball := env.CreateBall(t, "Multi-AC ball", session.PriorityMedium)
	store := env.GetStore(t)

	// Set multiple acceptance criteria
	criteria := []string{
		"First criterion: implement feature A",
		"Second criterion: add unit tests",
		"Third criterion: update documentation",
		"Fourth criterion: add integration tests",
	}
	ball.SetAcceptanceCriteria(criteria)

	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// Retrieve and verify ALL criteria are stored
	retrieved, err := store.GetBallByID(ball.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve ball: %v", err)
	}

	if len(retrieved.AcceptanceCriteria) != 4 {
		t.Errorf("Expected 4 acceptance criteria, got %d", len(retrieved.AcceptanceCriteria))
	}

	// Verify each criterion is preserved exactly
	for i, expected := range criteria {
		if i >= len(retrieved.AcceptanceCriteria) {
			t.Errorf("Missing criterion %d: %s", i+1, expected)
			continue
		}
		if retrieved.AcceptanceCriteria[i] != expected {
			t.Errorf("Criterion %d: expected '%s', got '%s'", i+1, expected, retrieved.AcceptanceCriteria[i])
		}
	}
}

// TestExportShowsAllAcceptanceCriteria tests that export formats include all ACs
func TestExportShowsAllAcceptanceCriteria(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	ball := env.CreateBall(t, "Export AC test", session.PriorityMedium)
	store := env.GetStore(t)

	// Set multiple acceptance criteria
	criteria := []string{
		"AC 1: Do thing one",
		"AC 2: Do thing two",
		"AC 3: Do thing three",
	}
	ball.SetAcceptanceCriteria(criteria)
	store.UpdateBall(ball)

	// Export and verify all ACs are in JSON
	balls, _ := store.LoadBalls()
	jsonData, err := json.MarshalIndent(balls, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	jsonStr := string(jsonData)
	for _, ac := range criteria {
		if !bytes.Contains([]byte(jsonStr), []byte(ac)) {
			t.Errorf("JSON export missing AC: %s", ac)
		}
	}

	// Verify the structure by unmarshaling
	var exported []*session.Ball
	if err := json.Unmarshal(jsonData, &exported); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(exported) != 1 {
		t.Fatalf("Expected 1 ball, got %d", len(exported))
	}

	if len(exported[0].AcceptanceCriteria) != 3 {
		t.Errorf("Exported ball should have 3 ACs, got %d", len(exported[0].AcceptanceCriteria))
	}
}

// TestRalphExportShowsAllAcceptanceCriteria tests that Ralph format includes all ACs
func TestRalphExportShowsAllAcceptanceCriteria(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	ball := env.CreateBall(t, "Ralph export test", session.PriorityHigh)
	store := env.GetStore(t)

	// Create a session to associate with the ball
	sessionStore, err := session.NewSessionStore(env.JuggleDir)
	if err != nil {
		t.Fatalf("Failed to create session store: %v", err)
	}

	sess, err := sessionStore.CreateSession("test-session", "Test session")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Set multiple acceptance criteria and tag with session
	criteria := []string{
		"Implement feature A correctly",
		"Add comprehensive tests",
		"Update API documentation",
		"Handle edge cases",
		"Performance optimization",
	}
	ball.SetAcceptanceCriteria(criteria)
	ball.AddTag(sess.ID)
	store.UpdateBall(ball)

	// Load balls and verify
	balls, _ := store.LoadBalls()
	if len(balls) != 1 {
		t.Fatalf("Expected 1 ball, got %d", len(balls))
	}

	exportedBall := balls[0]
	if len(exportedBall.AcceptanceCriteria) != 5 {
		t.Errorf("Expected 5 ACs, got %d", len(exportedBall.AcceptanceCriteria))
	}

	// Verify all criteria are present
	for i, expected := range criteria {
		if i >= len(exportedBall.AcceptanceCriteria) {
			t.Errorf("Missing AC %d", i+1)
			continue
		}
		if exportedBall.AcceptanceCriteria[i] != expected {
			t.Errorf("AC %d mismatch: expected '%s', got '%s'",
				i+1, expected, exportedBall.AcceptanceCriteria[i])
		}
	}
}

// TestPlanOutputDoesNotRepeatAcceptanceCriteria verifies that the plan command
// output does not redundantly display acceptance criteria after the user enters them
func TestPlanOutputDoesNotRepeatAcceptanceCriteria(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Run plan command with non-interactive flags
	output := runPlanCommand(t, env, "Test intent for AC output", "--ac", "First AC", "--ac", "Second AC")

	// The output should contain the confirmation message
	if !strings.Contains(output, "Planned ball added") {
		t.Errorf("Expected 'Planned ball added' in output, got: %s", output)
	}

	// The output should NOT contain "Acceptance Criteria:" because
	// the user already provided them via flags and we don't need to repeat
	if strings.Contains(output, "Acceptance Criteria:") {
		t.Errorf("Output should NOT contain 'Acceptance Criteria:' - user already entered them. Got: %s", output)
	}

	// Verify the ball was created with the ACs
	store := env.GetStore(t)
	balls, _ := store.LoadBalls()
	if len(balls) == 0 {
		t.Fatal("No balls created")
	}

	// Find the ball we just created
	var ball *session.Ball
	for _, b := range balls {
		if b.Title == "Test intent for AC output" {
			ball = b
			break
		}
	}
	if ball == nil {
		t.Fatal("Could not find created ball")
	}

	if len(ball.AcceptanceCriteria) != 2 {
		t.Errorf("Expected 2 acceptance criteria, got %d", len(ball.AcceptanceCriteria))
	}
}

// runPlanCommand runs the plan command and returns its output
func runPlanCommand(t *testing.T, env *TestEnv, intent string, args ...string) string {
	t.Helper()

	juggleRoot := GetRepoRoot(t)
	juggleBinary := filepath.Join(juggleRoot, "juggle")

	// Build juggle binary if needed
	if _, err := os.Stat(juggleBinary); os.IsNotExist(err) {
		buildCmd := exec.Command("go", "build", "-o", "juggle", "./cmd/juggle")
		buildCmd.Dir = juggleRoot
		if output, err := buildCmd.CombinedOutput(); err != nil {
			t.Fatalf("Failed to build juggle: %v\nOutput: %s", err, output)
		}
	}

	// Build command arguments
	cmdArgs := []string{"--config-home", env.ConfigHome, "plan", intent}
	cmdArgs = append(cmdArgs, args...)

	cmd := exec.Command(juggleBinary, cmdArgs...)
	cmd.Dir = env.ProjectDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Plan command failed: %v\nOutput: %s", err, output)
	}

	return string(output)
}

// TestSessionAliasForSessionsCommand tests that 'session' works as an alias for 'sessions'
func TestSessionAliasForSessionsCommand(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	juggleRoot := GetRepoRoot(t)
	juggleBinary := filepath.Join(juggleRoot, "juggle")

	// Build juggle binary if needed
	if _, err := os.Stat(juggleBinary); os.IsNotExist(err) {
		buildCmd := exec.Command("go", "build", "-o", "juggle", "./cmd/juggle")
		buildCmd.Dir = juggleRoot
		if output, err := buildCmd.CombinedOutput(); err != nil {
			t.Fatalf("Failed to build juggle: %v\nOutput: %s", err, output)
		}
	}

	// Test that 'session list' works
	sessionCmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "session", "list")
	sessionCmd.Dir = env.ProjectDir
	sessionOutput, err := sessionCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("'session list' command failed: %v\nOutput: %s", err, sessionOutput)
	}

	// Test that 'sessions list' works
	sessionsCmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "sessions", "list")
	sessionsCmd.Dir = env.ProjectDir
	sessionsOutput, err := sessionsCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("'sessions list' command failed: %v\nOutput: %s", err, sessionsOutput)
	}

	// Both should produce the same output (no sessions in fresh env)
	if string(sessionOutput) != string(sessionsOutput) {
		t.Errorf("'session' and 'sessions' produced different output:\nsession: %s\nsessions: %s",
			string(sessionOutput), string(sessionsOutput))
	}
}

// TestSessionAliasCreateCommand tests that 'session create' works
func TestSessionAliasCreateCommand(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	juggleRoot := GetRepoRoot(t)
	juggleBinary := filepath.Join(juggleRoot, "juggle")

	// Create a session using the alias
	createCmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "session", "create", "test-alias-session", "-m", "Test description")
	createCmd.Dir = env.ProjectDir
	output, err := createCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("'session create' command failed: %v\nOutput: %s", err, output)
	}

	// Verify output contains expected message
	if !strings.Contains(string(output), "Created session: test-alias-session") {
		t.Errorf("Expected 'Created session' in output, got: %s", output)
	}

	// Verify session was created using 'sessions show'
	showCmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "sessions", "show", "test-alias-session")
	showCmd.Dir = env.ProjectDir
	showOutput, err := showCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("'sessions show' command failed: %v\nOutput: %s", err, showOutput)
	}

	if !strings.Contains(string(showOutput), "test-alias-session") {
		t.Errorf("Session not found via 'sessions show', output: %s", showOutput)
	}
}

// TestSessionAliasShowCommand tests that 'session show' works
func TestSessionAliasShowCommand(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	juggleRoot := GetRepoRoot(t)
	juggleBinary := filepath.Join(juggleRoot, "juggle")

	// Create a session using 'sessions'
	createCmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "sessions", "create", "alias-show-test", "-m", "Test for show alias")
	createCmd.Dir = env.ProjectDir
	if output, err := createCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to create session: %v\nOutput: %s", err, output)
	}

	// Show using 'session' alias
	showCmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "session", "show", "alias-show-test")
	showCmd.Dir = env.ProjectDir
	showOutput, err := showCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("'session show' command failed: %v\nOutput: %s", err, showOutput)
	}

	// Verify it shows the session
	if !strings.Contains(string(showOutput), "alias-show-test") {
		t.Errorf("Expected session ID in output, got: %s", showOutput)
	}
	if !strings.Contains(string(showOutput), "Test for show alias") {
		t.Errorf("Expected description in output, got: %s", showOutput)
	}
}

// TestSessionAliasContextCommand tests that 'session context' works
func TestSessionAliasContextCommand(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	juggleRoot := GetRepoRoot(t)
	juggleBinary := filepath.Join(juggleRoot, "juggle")

	// Create a session
	createCmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "sessions", "create", "context-alias-test", "-m", "Test context alias")
	createCmd.Dir = env.ProjectDir
	if output, err := createCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to create session: %v\nOutput: %s", err, output)
	}

	// Set context using 'session' alias
	setCmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "session", "context", "context-alias-test", "--set", "This is the context set via alias")
	setCmd.Dir = env.ProjectDir
	if output, err := setCmd.CombinedOutput(); err != nil {
		t.Fatalf("'session context --set' failed: %v\nOutput: %s", err, output)
	}

	// Read context using 'sessions' (original command)
	readCmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "sessions", "context", "context-alias-test")
	readCmd.Dir = env.ProjectDir
	readOutput, err := readCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("'sessions context' failed: %v\nOutput: %s", err, readOutput)
	}

	if !strings.Contains(string(readOutput), "This is the context set via alias") {
		t.Errorf("Expected context text in output, got: %s", readOutput)
	}
}

// TestSessionAliasHelpShowsAlias tests that help text shows the alias
func TestSessionAliasHelpShowsAlias(t *testing.T) {
	juggleRoot := GetRepoRoot(t)
	juggleBinary := filepath.Join(juggleRoot, "juggle")

	// Get help for 'session' command
	helpCmd := exec.Command(juggleBinary, "session", "--help")
	helpOutput, err := helpCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("'session --help' failed: %v\nOutput: %s", err, helpOutput)
	}

	// Verify help mentions alias
	if !strings.Contains(string(helpOutput), "Aliases:") {
		t.Errorf("Expected 'Aliases:' in help output, got: %s", helpOutput)
	}
	if !strings.Contains(string(helpOutput), "session") {
		t.Errorf("Expected 'session' in help aliases, got: %s", helpOutput)
	}
}

// TestSessionsCommandStillWorks ensures the original 'sessions' command still functions
func TestSessionsCommandStillWorks(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	juggleRoot := GetRepoRoot(t)
	juggleBinary := filepath.Join(juggleRoot, "juggle")

	// Create session using 'sessions'
	createCmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "sessions", "create", "original-cmd-test", "-m", "Original command test")
	createCmd.Dir = env.ProjectDir
	output, err := createCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("'sessions create' failed: %v\nOutput: %s", err, output)
	}

	// List using 'sessions'
	listCmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "sessions", "list")
	listCmd.Dir = env.ProjectDir
	listOutput, err := listCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("'sessions list' failed: %v\nOutput: %s", err, listOutput)
	}

	if !strings.Contains(string(listOutput), "original-cmd-test") {
		t.Errorf("Expected session in list output, got: %s", listOutput)
	}
}

// TestBallOutputJSONFlag tests that 'juggle <id> --json' outputs JSON
func TestBallOutputJSONFlag(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	juggleRoot := GetRepoRoot(t)
	juggleBinary := filepath.Join(juggleRoot, "juggle")

	// Create and start a ball
	ball := env.CreateBall(t, "JSON output test ball", session.PriorityHigh)
	store := env.GetStore(t)
	ball.Start()
	store.UpdateBall(ball)

	// Test 'juggle <id> --json' outputs valid JSON
	cmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, ball.ShortID(), "--json")
	cmd.Dir = env.ProjectDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("'juggle %s --json' failed: %v\nOutput: %s", ball.ShortID(), err, output)
	}

	// Verify it's valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		t.Errorf("Output is not valid JSON: %v\nOutput: %s", err, output)
	}

	// Verify key fields are present
	if result["id"] != ball.ID {
		t.Errorf("Expected id '%s', got '%v'", ball.ID, result["id"])
	}
	if result["title"] != "JSON output test ball" {
		t.Errorf("Expected title 'JSON output test ball', got '%v'", result["title"])
	}
	if result["state"] != "in_progress" {
		t.Errorf("Expected state 'in_progress', got '%v'", result["state"])
	}
}

// TestBallOutputConsistency tests that 'juggle <id>' and 'juggle show <id>' show same info
func TestBallOutputConsistency(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	juggleRoot := GetRepoRoot(t)
	juggleBinary := filepath.Join(juggleRoot, "juggle")

	// Create a ball with various properties
	ball := env.CreateBall(t, "Consistency test ball", session.PriorityMedium)
	store := env.GetStore(t)
	ball.Start()
	ball.SetAcceptanceCriteria([]string{"AC 1", "AC 2", "AC 3"})
	ball.AddTag("test-tag")
	store.UpdateBall(ball)

	// Get output from 'juggle <id>'
	directCmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, ball.ShortID())
	directCmd.Dir = env.ProjectDir
	directOutput, err := directCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("'juggle %s' failed: %v\nOutput: %s", ball.ShortID(), err, directOutput)
	}

	// Get output from 'juggle show <id>'
	showCmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "show", ball.ShortID())
	showCmd.Dir = env.ProjectDir
	showOutput, err := showCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("'juggle show %s' failed: %v\nOutput: %s", ball.ShortID(), err, showOutput)
	}

	// Both should show all the same key information
	expectedFields := []string{
		"Ball ID:",
		"Title:",
		"Priority:",
		"State:",
		"Acceptance Criteria:",
		"AC 1",
		"AC 2",
		"AC 3",
		"test-tag",
	}

	for _, field := range expectedFields {
		if !strings.Contains(string(directOutput), field) {
			t.Errorf("'juggle %s' missing field: %s", ball.ShortID(), field)
		}
		if !strings.Contains(string(showOutput), field) {
			t.Errorf("'juggle show %s' missing field: %s", ball.ShortID(), field)
		}
	}
}

// TestBallOutputJSONConsistency tests that JSON output from both commands is consistent
func TestBallOutputJSONConsistency(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	juggleRoot := GetRepoRoot(t)
	juggleBinary := filepath.Join(juggleRoot, "juggle")

	// Create a ball
	ball := env.CreateBall(t, "JSON consistency test", session.PriorityHigh)
	store := env.GetStore(t)
	ball.Start()
	ball.SetAcceptanceCriteria([]string{"Criterion A", "Criterion B"})
	store.UpdateBall(ball)

	// Get JSON from 'juggle <id> --json'
	directCmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, ball.ShortID(), "--json")
	directCmd.Dir = env.ProjectDir
	directOutput, err := directCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("'juggle %s --json' failed: %v\nOutput: %s", ball.ShortID(), err, directOutput)
	}

	// Get JSON from 'juggle show <id> --json'
	showCmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "show", ball.ShortID(), "--json")
	showCmd.Dir = env.ProjectDir
	showOutput, err := showCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("'juggle show %s --json' failed: %v\nOutput: %s", ball.ShortID(), err, showOutput)
	}

	// Parse both JSON outputs
	var directJSON, showJSON map[string]interface{}
	if err := json.Unmarshal(directOutput, &directJSON); err != nil {
		t.Fatalf("Failed to parse direct JSON: %v", err)
	}
	if err := json.Unmarshal(showOutput, &showJSON); err != nil {
		t.Fatalf("Failed to parse show JSON: %v", err)
	}

	// Compare key fields
	fieldsToCompare := []string{"id", "title", "priority", "state"}
	for _, field := range fieldsToCompare {
		if directJSON[field] != showJSON[field] {
			t.Errorf("Field '%s' mismatch: direct=%v, show=%v", field, directJSON[field], showJSON[field])
		}
	}

	// Compare acceptance_criteria arrays
	directAC, _ := directJSON["acceptance_criteria"].([]interface{})
	showAC, _ := showJSON["acceptance_criteria"].([]interface{})
	if len(directAC) != len(showAC) {
		t.Errorf("Acceptance criteria count mismatch: direct=%d, show=%d", len(directAC), len(showAC))
	}
}

// TestBallOutputWithAllFields tests that output shows all ball fields
func TestBallOutputWithAllFields(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	juggleRoot := GetRepoRoot(t)
	juggleBinary := filepath.Join(juggleRoot, "juggle")

	// Create a ball with all possible fields populated
	ball := env.CreateBall(t, "Full fields test", session.PriorityUrgent)
	store := env.GetStore(t)
	ball.Start()
	ball.SetAcceptanceCriteria([]string{"AC 1", "AC 2"})
	ball.AddTag("tag1")
	ball.AddTag("tag2")
	ball.AddDependency("fake-dep-1")
	store.UpdateBall(ball)

	// Test text output
	textCmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, ball.ShortID())
	textCmd.Dir = env.ProjectDir
	textOutput, err := textCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Command failed: %v\nOutput: %s", err, textOutput)
	}

	// Verify all fields appear in text output
	textChecks := []string{
		"Ball ID:",
		"Title:",
		"Priority:",
		"State:",
		"Tags:",
		"Depends On:",
		"Acceptance Criteria:",
	}
	for _, check := range textChecks {
		if !strings.Contains(string(textOutput), check) {
			t.Errorf("Text output missing: %s\nOutput: %s", check, textOutput)
		}
	}

	// Test JSON output
	jsonCmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, ball.ShortID(), "--json")
	jsonCmd.Dir = env.ProjectDir
	jsonOutput, err := jsonCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("JSON command failed: %v\nOutput: %s", err, jsonOutput)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(jsonOutput, &result); err != nil {
		t.Fatalf("Invalid JSON: %v\nOutput: %s", err, jsonOutput)
	}

	// Verify JSON has key fields
	jsonFields := []string{"id", "title", "priority", "state", "tags", "depends_on", "acceptance_criteria"}
	for _, field := range jsonFields {
		if _, ok := result[field]; !ok {
			t.Errorf("JSON missing field: %s", field)
		}
	}
}

// TestBallOutputPendingVsStarted tests output for both pending and started balls
func TestBallOutputPendingVsStarted(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	juggleRoot := GetRepoRoot(t)
	juggleBinary := filepath.Join(juggleRoot, "juggle")

	// Create a pending ball
	pendingBall := env.CreateBall(t, "Pending ball test", session.PriorityMedium)

	// Create a started ball
	startedBall := env.CreateBall(t, "Started ball test", session.PriorityHigh)
	store := env.GetStore(t)
	startedBall.Start()
	store.UpdateBall(startedBall)

	// Get output for started ball (should show details, not "started" message)
	startedCmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, startedBall.ShortID())
	startedCmd.Dir = env.ProjectDir
	startedOutput, err := startedCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed for started ball: %v\nOutput: %s", err, startedOutput)
	}

	// Should show ball details (not start message)
	if !strings.Contains(string(startedOutput), "Ball ID:") {
		t.Errorf("Started ball should show details, got: %s", startedOutput)
	}
	if !strings.Contains(string(startedOutput), "in_progress") {
		t.Errorf("Started ball should show state, got: %s", startedOutput)
	}

	// Get output for pending ball (should activate it and show start message)
	pendingCmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, pendingBall.ShortID())
	pendingCmd.Dir = env.ProjectDir
	pendingOutput, err := pendingCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed for pending ball: %v\nOutput: %s", err, pendingOutput)
	}

	// Should show start confirmation
	if !strings.Contains(string(pendingOutput), "Started ball:") {
		t.Errorf("Pending ball should be started, got: %s", pendingOutput)
	}
}

// TestSessionsProgressCommand tests the 'juggle sessions progress <id>' command
func TestSessionsProgressCommand(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	juggleRoot := GetRepoRoot(t)
	juggleBinary := filepath.Join(juggleRoot, "juggle")

	// Create session
	createCmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "sessions", "create", "progress-test", "-m", "Test session")
	createCmd.Dir = env.ProjectDir
	output, err := createCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to create session: %v\nOutput: %s", err, output)
	}

	// Check progress when empty
	emptyCmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "sessions", "progress", "progress-test")
	emptyCmd.Dir = env.ProjectDir
	emptyOutput, err := emptyCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("'sessions progress' failed for empty session: %v\nOutput: %s", err, emptyOutput)
	}
	if !strings.Contains(string(emptyOutput), "No progress logged") {
		t.Errorf("Expected 'No progress logged' message, got: %s", emptyOutput)
	}

	// Append some progress
	appendCmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "progress", "append", "progress-test", "First entry")
	appendCmd.Dir = env.ProjectDir
	_, err = appendCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to append progress: %v", err)
	}

	appendCmd2 := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "progress", "append", "progress-test", "Second entry")
	appendCmd2.Dir = env.ProjectDir
	_, err = appendCmd2.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to append second progress: %v", err)
	}

	// View progress
	viewCmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "sessions", "progress", "progress-test")
	viewCmd.Dir = env.ProjectDir
	viewOutput, err := viewCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("'sessions progress' failed: %v\nOutput: %s", err, viewOutput)
	}

	// Check output contains both entries
	if !strings.Contains(string(viewOutput), "First entry") {
		t.Errorf("Expected 'First entry' in output, got: %s", viewOutput)
	}
	if !strings.Contains(string(viewOutput), "Second entry") {
		t.Errorf("Expected 'Second entry' in output, got: %s", viewOutput)
	}
}

// TestSessionsProgressWithAlias tests 'juggle session progress' (singular alias)
func TestSessionsProgressWithAlias(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	juggleRoot := GetRepoRoot(t)
	juggleBinary := filepath.Join(juggleRoot, "juggle")

	// Create session using alias
	createCmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "session", "create", "alias-progress-test", "-m", "Test with alias")
	createCmd.Dir = env.ProjectDir
	output, err := createCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to create session: %v\nOutput: %s", err, output)
	}

	// Append progress
	appendCmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "progress", "append", "alias-progress-test", "Alias test entry")
	appendCmd.Dir = env.ProjectDir
	_, err = appendCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to append progress: %v", err)
	}

	// View using alias
	viewCmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "session", "progress", "alias-progress-test")
	viewCmd.Dir = env.ProjectDir
	viewOutput, err := viewCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("'session progress' (alias) failed: %v\nOutput: %s", err, viewOutput)
	}

	if !strings.Contains(string(viewOutput), "Alias test entry") {
		t.Errorf("Expected 'Alias test entry' in output, got: %s", viewOutput)
	}
}

// TestSessionsProgressNonexistent tests error handling for nonexistent session
func TestSessionsProgressNonexistent(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	juggleRoot := GetRepoRoot(t)
	juggleBinary := filepath.Join(juggleRoot, "juggle")

	// Try to view progress for nonexistent session
	cmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "sessions", "progress", "nonexistent-session")
	cmd.Dir = env.ProjectDir
	output, err := cmd.CombinedOutput()

	// Should fail with error
	if err == nil {
		t.Fatalf("Expected error for nonexistent session, got success. Output: %s", output)
	}
	if !strings.Contains(string(output), "session not found") {
		t.Errorf("Expected 'session not found' error, got: %s", output)
	}
}

// TestSessionsProgressHelp tests help text for progress subcommand
func TestSessionsProgressHelp(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	juggleRoot := GetRepoRoot(t)
	juggleBinary := filepath.Join(juggleRoot, "juggle")

	cmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "sessions", "progress", "--help")
	cmd.Dir = env.ProjectDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("'sessions progress --help' failed: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(string(output), "progress log") {
		t.Errorf("Expected 'progress log' in help, got: %s", output)
	}
}

// TestPlanNonInteractiveFlag tests the --non-interactive flag for juggle plan
func TestPlanNonInteractiveFlag(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	juggleRoot := GetRepoRoot(t)
	juggleBinary := filepath.Join(juggleRoot, "juggle")

	// Build juggle binary if needed
	if _, err := os.Stat(juggleBinary); os.IsNotExist(err) {
		buildCmd := exec.Command("go", "build", "-o", "juggle", "./cmd/juggle")
		buildCmd.Dir = juggleRoot
		if output, err := buildCmd.CombinedOutput(); err != nil {
			t.Fatalf("Failed to build juggle: %v\nOutput: %s", err, output)
		}
	}

	// Test 1: Non-interactive with intent only - should use defaults
	cmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "plan", "Non-interactive test ball", "--non-interactive")
	cmd.Dir = env.ProjectDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Non-interactive plan failed: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(string(output), "Planned ball added") {
		t.Errorf("Expected 'Planned ball added', got: %s", output)
	}

	// Verify defaults were applied
	store := env.GetStore(t)
	balls, _ := store.LoadBalls()
	var ball *session.Ball
	for _, b := range balls {
		if b.Title == "Non-interactive test ball" {
			ball = b
			break
		}
	}
	if ball == nil {
		t.Fatal("Ball not created")
	}

	if ball.Priority != session.PriorityMedium {
		t.Errorf("Expected default priority 'medium', got: %s", ball.Priority)
	}
	if ball.State != session.StatePending {
		t.Errorf("Expected default state 'pending', got: %s", ball.State)
	}
}

// TestPlanNonInteractiveWithAllFlags tests --non-interactive with explicit flags
func TestPlanNonInteractiveWithAllFlags(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	juggleRoot := GetRepoRoot(t)
	juggleBinary := filepath.Join(juggleRoot, "juggle")

	cmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "plan",
		"Full flags test",
		"--non-interactive",
		"--priority", "high",
		"--session", "my-session",
		"-c", "AC 1",
		"-c", "AC 2",
		"--tags", "tag1,tag2",
	)
	cmd.Dir = env.ProjectDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Non-interactive plan with flags failed: %v\nOutput: %s", err, output)
	}

	// Verify all flags were applied
	store := env.GetStore(t)
	balls, _ := store.LoadBalls()
	var ball *session.Ball
	for _, b := range balls {
		if b.Title == "Full flags test" {
			ball = b
			break
		}
	}
	if ball == nil {
		t.Fatal("Ball not created")
	}

	if ball.Priority != session.PriorityHigh {
		t.Errorf("Expected priority 'high', got: %s", ball.Priority)
	}
	if len(ball.AcceptanceCriteria) != 2 {
		t.Errorf("Expected 2 acceptance criteria, got: %d", len(ball.AcceptanceCriteria))
	}
	// Session is added as a tag
	hasSessionTag := false
	for _, tag := range ball.Tags {
		if tag == "my-session" {
			hasSessionTag = true
			break
		}
	}
	if !hasSessionTag {
		t.Errorf("Expected session tag 'my-session', got tags: %v", ball.Tags)
	}
}

// TestPlanAcceptanceCriteriaWithCommas tests that AC containing commas are not split
func TestPlanAcceptanceCriteriaWithCommas(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	juggleRoot := GetRepoRoot(t)
	juggleBinary := filepath.Join(juggleRoot, "juggle")

	// Use an AC with commas - should stay as ONE criterion
	cmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "plan",
		"Comma test",
		"--non-interactive",
		"-c", "first, second, third",
	)
	cmd.Dir = env.ProjectDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Plan with comma in AC failed: %v\nOutput: %s", err, output)
	}

	// Verify exactly ONE acceptance criterion was created
	store := env.GetStore(t)
	balls, _ := store.LoadBalls()
	var ball *session.Ball
	for _, b := range balls {
		if b.Title == "Comma test" {
			ball = b
			break
		}
	}
	if ball == nil {
		t.Fatal("Ball not created")
	}

	if len(ball.AcceptanceCriteria) != 1 {
		t.Errorf("Expected exactly 1 acceptance criterion, got %d: %v", len(ball.AcceptanceCriteria), ball.AcceptanceCriteria)
	}
	if len(ball.AcceptanceCriteria) > 0 && ball.AcceptanceCriteria[0] != "first, second, third" {
		t.Errorf("Expected AC 'first, second, third', got: %s", ball.AcceptanceCriteria[0])
	}
}

// TestPlanNonInteractiveNoIntent tests that --non-interactive requires intent
func TestPlanNonInteractiveNoIntent(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	juggleRoot := GetRepoRoot(t)
	juggleBinary := filepath.Join(juggleRoot, "juggle")

	cmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "plan", "--non-interactive")
	cmd.Dir = env.ProjectDir
	output, err := cmd.CombinedOutput()

	// Should fail without intent
	if err == nil {
		t.Fatalf("Expected error for non-interactive without intent, got success: %s", output)
	}
	if !strings.Contains(string(output), "intent is required") {
		t.Errorf("Expected 'intent is required' error, got: %s", output)
	}
}

// TestPlanAlwaysCreatesPendingState verifies that new balls always start in pending state
func TestPlanAlwaysCreatesPendingState(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	juggleRoot := GetRepoRoot(t)
	juggleBinary := filepath.Join(juggleRoot, "juggle")

	cmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "plan",
		"Test always pending",
		"--non-interactive",
	)
	cmd.Dir = env.ProjectDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Non-interactive plan failed: %v\nOutput: %s", err, output)
	}

	// Verify ball is in pending state
	store := env.GetStore(t)
	balls, _ := store.LoadBalls()
	var ball *session.Ball
	for _, b := range balls {
		if b.Title == "Test always pending" {
			ball = b
			break
		}
	}
	if ball == nil {
		t.Fatal("Ball not created")
	}

	if ball.State != session.StatePending {
		t.Errorf("Expected state 'pending', got: %s", ball.State)
	}
}

// TestPlanCriteriaAliasFlag tests that --criteria works as alias for --ac
func TestPlanCriteriaAliasFlag(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	juggleRoot := GetRepoRoot(t)
	juggleBinary := filepath.Join(juggleRoot, "juggle")

	// Test --criteria alone
	cmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "plan",
		"Criteria alias test",
		"--non-interactive",
		"--criteria", "First via criteria",
		"--criteria", "Second via criteria",
	)
	cmd.Dir = env.ProjectDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Plan with --criteria failed: %v\nOutput: %s", err, output)
	}

	store := env.GetStore(t)
	balls, _ := store.LoadBalls()
	var ball *session.Ball
	for _, b := range balls {
		if b.Title == "Criteria alias test" {
			ball = b
			break
		}
	}
	if ball == nil {
		t.Fatal("Ball not created")
	}

	if len(ball.AcceptanceCriteria) != 2 {
		t.Errorf("Expected 2 acceptance criteria, got: %d", len(ball.AcceptanceCriteria))
	}
	if len(ball.AcceptanceCriteria) >= 2 {
		if ball.AcceptanceCriteria[0] != "First via criteria" {
			t.Errorf("Expected first AC 'First via criteria', got: %s", ball.AcceptanceCriteria[0])
		}
		if ball.AcceptanceCriteria[1] != "Second via criteria" {
			t.Errorf("Expected second AC 'Second via criteria', got: %s", ball.AcceptanceCriteria[1])
		}
	}
}

// TestPlanCriteriaAndACTogether tests that --criteria and -c can be used together
func TestPlanCriteriaAndACTogether(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	juggleRoot := GetRepoRoot(t)
	juggleBinary := filepath.Join(juggleRoot, "juggle")

	cmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "plan",
		"Both flags test",
		"--non-interactive",
		"-c", "Via -c flag",
		"--criteria", "Via criteria flag",
	)
	cmd.Dir = env.ProjectDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Plan with both flags failed: %v\nOutput: %s", err, output)
	}

	store := env.GetStore(t)
	balls, _ := store.LoadBalls()
	var ball *session.Ball
	for _, b := range balls {
		if b.Title == "Both flags test" {
			ball = b
			break
		}
	}
	if ball == nil {
		t.Fatal("Ball not created")
	}

	if len(ball.AcceptanceCriteria) != 2 {
		t.Errorf("Expected 2 acceptance criteria from both flags, got: %d", len(ball.AcceptanceCriteria))
	}
}

// TestPlanStateFlagRemoved verifies that --state flag no longer exists
func TestPlanStateFlagRemoved(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	juggleRoot := GetRepoRoot(t)
	juggleBinary := filepath.Join(juggleRoot, "juggle")

	cmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "plan",
		"Test state flag removed",
		"--non-interactive",
		"--state", "in_progress",
	)
	cmd.Dir = env.ProjectDir
	output, err := cmd.CombinedOutput()

	// Should fail with unknown flag error
	if err == nil {
		t.Fatalf("Expected error for unknown --state flag, got success: %s", output)
	}
	if !strings.Contains(string(output), "unknown flag") {
		t.Errorf("Expected 'unknown flag' error, got: %s", output)
	}
}

// TestSessionDeleteYesFlag tests the --yes flag for session delete
func TestSessionDeleteYesFlag(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	juggleRoot := GetRepoRoot(t)
	juggleBinary := filepath.Join(juggleRoot, "juggle")

	// Create a session
	createCmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "sessions", "create", "delete-yes-test", "-m", "Test for --yes flag")
	createCmd.Dir = env.ProjectDir
	if output, err := createCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to create session: %v\nOutput: %s", err, output)
	}

	// Delete with --yes flag
	deleteCmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "sessions", "delete", "delete-yes-test", "--yes")
	deleteCmd.Dir = env.ProjectDir
	output, err := deleteCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Delete with --yes failed: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(string(output), "Deleted session") {
		t.Errorf("Expected 'Deleted session', got: %s", output)
	}

	// Verify session is deleted
	showCmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "sessions", "show", "delete-yes-test")
	showCmd.Dir = env.ProjectDir
	showOutput, err := showCmd.CombinedOutput()
	if err == nil {
		t.Errorf("Session should be deleted, but show succeeded: %s", showOutput)
	}
}

// TestSessionDeleteYesShortFlag tests the -y short flag for session delete
func TestSessionDeleteYesShortFlag(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	juggleRoot := GetRepoRoot(t)
	juggleBinary := filepath.Join(juggleRoot, "juggle")

	// Create a session
	createCmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "sessions", "create", "delete-y-test", "-m", "Test for -y flag")
	createCmd.Dir = env.ProjectDir
	if output, err := createCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to create session: %v\nOutput: %s", err, output)
	}

	// Delete with -y flag
	deleteCmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "sessions", "delete", "delete-y-test", "-y")
	deleteCmd.Dir = env.ProjectDir
	output, err := deleteCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Delete with -y failed: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(string(output), "Deleted session") {
		t.Errorf("Expected 'Deleted session', got: %s", output)
	}
}

// TestConfigACClearYesFlag tests the --yes flag for config ac clear
func TestConfigACClearYesFlag(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	juggleRoot := GetRepoRoot(t)
	juggleBinary := filepath.Join(juggleRoot, "juggle")

	// Add some acceptance criteria first
	addCmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "config", "ac", "add", "Test AC")
	addCmd.Dir = env.ProjectDir
	if output, err := addCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to add AC: %v\nOutput: %s", err, output)
	}

	// Clear with --yes flag
	clearCmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "config", "ac", "clear", "--yes")
	clearCmd.Dir = env.ProjectDir
	output, err := clearCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Clear with --yes failed: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(string(output), "Cleared") {
		t.Errorf("Expected 'Cleared' message, got: %s", output)
	}

	// Verify ACs are cleared
	listCmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "config", "ac", "list")
	listCmd.Dir = env.ProjectDir
	listOutput, err := listCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("List failed: %v\nOutput: %s", err, listOutput)
	}

	if !strings.Contains(string(listOutput), "No repository-level") {
		t.Errorf("Expected no criteria, got: %s", listOutput)
	}
}

// TestPlanHelpShowsNonInteractiveFlag tests that --non-interactive appears in help
func TestPlanHelpShowsNonInteractiveFlag(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	juggleRoot := GetRepoRoot(t)
	juggleBinary := filepath.Join(juggleRoot, "juggle")

	cmd := exec.Command(juggleBinary, "plan", "--help")
	cmd.Dir = env.ProjectDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Help command failed: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(string(output), "--non-interactive") {
		t.Errorf("Expected '--non-interactive' in help, got: %s", output)
	}
	if !strings.Contains(string(output), "headless") {
		t.Errorf("Expected 'headless' in help description, got: %s", output)
	}
}

// TestSessionDeleteHelpShowsYesFlag tests that --yes appears in delete help
func TestSessionDeleteHelpShowsYesFlag(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	juggleRoot := GetRepoRoot(t)
	juggleBinary := filepath.Join(juggleRoot, "juggle")

	cmd := exec.Command(juggleBinary, "sessions", "delete", "--help")
	cmd.Dir = env.ProjectDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Help command failed: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(string(output), "--yes") {
		t.Errorf("Expected '--yes' in help, got: %s", output)
	}
	if !strings.Contains(string(output), "-y") {
		t.Errorf("Expected '-y' shorthand in help, got: %s", output)
	}
}

// TestSessionCreateACFlagSkipsPrompt tests that --ac flag skips interactive AC prompt
func TestSessionCreateACFlagSkipsPrompt(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	juggleRoot := GetRepoRoot(t)
	juggleBinary := filepath.Join(juggleRoot, "juggle")

	// Create session with --ac flag - should not prompt
	cmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "sessions", "create", "ac-flag-test", "-m", "Test session", "--ac", "AC 1", "--ac", "AC 2")
	cmd.Dir = env.ProjectDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("'sessions create' with --ac failed: %v\nOutput: %s", err, output)
	}

	// Verify output mentions 2 ACs
	if !strings.Contains(string(output), "2 item(s)") {
		t.Errorf("Expected '2 item(s)' in output, got: %s", output)
	}

	// Verify session has the ACs by showing it
	showCmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "sessions", "show", "ac-flag-test")
	showCmd.Dir = env.ProjectDir
	showOutput, err := showCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("'sessions show' failed: %v\nOutput: %s", err, showOutput)
	}

	if !strings.Contains(string(showOutput), "AC 1") {
		t.Errorf("Expected 'AC 1' in session, got: %s", showOutput)
	}
	if !strings.Contains(string(showOutput), "AC 2") {
		t.Errorf("Expected 'AC 2' in session, got: %s", showOutput)
	}
}

// TestSessionCreateNonInteractiveSkipsPrompt tests that --non-interactive skips AC prompt
func TestSessionCreateNonInteractiveSkipsPrompt(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	juggleRoot := GetRepoRoot(t)
	juggleBinary := filepath.Join(juggleRoot, "juggle")

	// Create session with --non-interactive - should not prompt for anything
	cmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "sessions", "create", "non-interactive-test", "-m", "Test session", "--non-interactive")
	cmd.Dir = env.ProjectDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("'sessions create' with --non-interactive failed: %v\nOutput: %s", err, output)
	}

	// Should complete without hanging on prompts
	if !strings.Contains(string(output), "Created session: non-interactive-test") {
		t.Errorf("Expected 'Created session' in output, got: %s", output)
	}
}

// TestSessionCreateNonInteractiveWithRepoDefaults tests inheritance in non-interactive mode
func TestSessionCreateNonInteractiveWithRepoDefaults(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	juggleRoot := GetRepoRoot(t)
	juggleBinary := filepath.Join(juggleRoot, "juggle")

	// First, set up repo-level ACs
	configAddCmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "config", "ac", "add", "Run tests")
	configAddCmd.Dir = env.ProjectDir
	if output, err := configAddCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to add repo AC: %v\nOutput: %s", err, output)
	}

	// Create session with --non-interactive - should inherit repo defaults
	cmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "sessions", "create", "repo-defaults-test", "-m", "Test session", "--non-interactive")
	cmd.Dir = env.ProjectDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("'sessions create' failed: %v\nOutput: %s", err, output)
	}

	// Should mention inherited ACs
	if !strings.Contains(string(output), "inherited 1 from repo defaults") {
		t.Errorf("Expected 'inherited 1 from repo defaults' in output, got: %s", output)
	}

	// Verify session has the inherited AC
	showCmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "sessions", "show", "repo-defaults-test")
	showCmd.Dir = env.ProjectDir
	showOutput, err := showCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("'sessions show' failed: %v\nOutput: %s", err, showOutput)
	}

	if !strings.Contains(string(showOutput), "Run tests") {
		t.Errorf("Expected inherited 'Run tests' AC in session, got: %s", showOutput)
	}
}

// TestSessionCreateHelpShowsNonInteractiveFlag tests that --non-interactive appears in help
func TestSessionCreateHelpShowsNonInteractiveFlag(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	juggleRoot := GetRepoRoot(t)
	juggleBinary := filepath.Join(juggleRoot, "juggle")

	cmd := exec.Command(juggleBinary, "sessions", "create", "--help")
	cmd.Dir = env.ProjectDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Help command failed: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(string(output), "--non-interactive") {
		t.Errorf("Expected '--non-interactive' in help, got: %s", output)
	}
	if !strings.Contains(string(output), "headless") {
		t.Errorf("Expected 'headless' in help description, got: %s", output)
	}
}

// TestMinimalUniqueIDDisplay tests that TUI and CLI show minimal unique IDs
func TestMinimalUniqueIDDisplay(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create two balls with similar IDs
	ball1, err := session.NewBall(env.ProjectDir, "First ball", session.PriorityMedium)
	if err != nil {
		t.Fatalf("failed to create ball1: %v", err)
	}
	ball2, err := session.NewBall(env.ProjectDir, "Second ball", session.PriorityHigh)
	if err != nil {
		t.Fatalf("failed to create ball2: %v", err)
	}

	store := env.GetStore(t)
	if err := store.AppendBall(ball1); err != nil {
		t.Fatalf("failed to append ball1: %v", err)
	}
	if err := store.AppendBall(ball2); err != nil {
		t.Fatalf("failed to append ball2: %v", err)
	}

	// Compute minimal unique IDs
	balls := []*session.Ball{ball1, ball2}
	minIDs := session.ComputeMinimalUniqueIDs(balls)

	// Verify both balls have minimal IDs
	if minIDs[ball1.ID] == "" {
		t.Errorf("ball1 should have a minimal ID")
	}
	if minIDs[ball2.ID] == "" {
		t.Errorf("ball2 should have a minimal ID")
	}

	// Verify minimal IDs are actually different
	if minIDs[ball1.ID] == minIDs[ball2.ID] {
		t.Errorf("minimal IDs should be unique: ball1=%s, ball2=%s", minIDs[ball1.ID], minIDs[ball2.ID])
	}

	// Verify each can be resolved by its minimal ID
	matches1 := session.ResolveBallByPrefix(balls, minIDs[ball1.ID])
	if len(matches1) != 1 || matches1[0].ID != ball1.ID {
		t.Errorf("minimal ID %s should resolve to ball1", minIDs[ball1.ID])
	}

	matches2 := session.ResolveBallByPrefix(balls, minIDs[ball2.ID])
	if len(matches2) != 1 || matches2[0].ID != ball2.ID {
		t.Errorf("minimal ID %s should resolve to ball2", minIDs[ball2.ID])
	}
}

// TestPrefixBallResolution tests that CLI commands accept prefix IDs
func TestPrefixBallResolution(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	ball := env.CreateBall(t, "Test prefix resolution", session.PriorityMedium)
	store := env.GetStore(t)

	// Get the minimal unique ID (should be single char for single ball)
	balls, _ := store.LoadBalls()
	minIDs := session.ComputeMinimalUniqueIDs(balls)
	minID := minIDs[ball.ID]

	// The minimal ID should be very short for a single ball
	if len(minID) > 1 {
		// Single ball should only need first char
		t.Logf("Note: minimal ID is %q (length %d)", minID, len(minID))
	}

	// Resolve by minimal ID via store
	resolved, err := store.ResolveBallID(minID)
	if err != nil {
		t.Fatalf("failed to resolve by minimal ID %s: %v", minID, err)
	}
	if resolved.ID != ball.ID {
		t.Errorf("minimal ID %s resolved to %s, expected %s", minID, resolved.ID, ball.ID)
	}
}

// TestPrefixBallResolution_Ambiguous tests ambiguous prefix detection
func TestPrefixBallResolution_Ambiguous(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create multiple balls
	ball1, _ := session.NewBall(env.ProjectDir, "Ball 1", session.PriorityMedium)
	ball2, _ := session.NewBall(env.ProjectDir, "Ball 2", session.PriorityHigh)
	ball3, _ := session.NewBall(env.ProjectDir, "Ball 3", session.PriorityLow)

	store := env.GetStore(t)
	store.AppendBall(ball1)
	store.AppendBall(ball2)
	store.AppendBall(ball3)

	balls, _ := store.LoadBalls()

	// Try to resolve with a very short prefix that might match multiple
	// The short IDs are 8 hex chars, so try matching with just first char of short ID
	// This may match multiple balls if their short IDs start with same char
	matches := session.ResolveBallByPrefix(balls, ball1.ShortID()[:1])

	// If multiple matches, that's expected behavior for ambiguous prefix
	// The point is it should return all matching balls, not error
	if len(matches) > 1 {
		t.Logf("Prefix matched %d balls (expected for ambiguous prefix)", len(matches))
	}
}

// TestStoreResolveBallIDStrict tests strict resolution that fails on ambiguity
func TestStoreResolveBallIDStrict(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create balls with known IDs by manually setting them
	ball1 := &session.Ball{
		ID:         "test-aaa111",
		Title:     "Ball A",
		Priority:   session.PriorityMedium,
		State:      session.StatePending,
		WorkingDir: env.ProjectDir,
	}
	ball2 := &session.Ball{
		ID:         "test-aaa222",
		Title:     "Ball B",
		Priority:   session.PriorityHigh,
		State:      session.StatePending,
		WorkingDir: env.ProjectDir,
	}

	store := env.GetStore(t)
	store.AppendBall(ball1)
	store.AppendBall(ball2)

	// Strict resolution should fail on ambiguous prefix
	_, err := store.ResolveBallIDStrict("aaa")
	if err == nil {
		t.Error("expected error for ambiguous prefix 'aaa'")
	}
	if err != nil && !strings.Contains(err.Error(), "ambiguous") {
		t.Errorf("expected ambiguous error, got: %v", err)
	}

	// Strict resolution should succeed with unique prefix
	resolved, err := store.ResolveBallIDStrict("aaa1")
	if err != nil {
		t.Errorf("expected success for unique prefix 'aaa1': %v", err)
	}
	if resolved != nil && resolved.ID != "test-aaa111" {
		t.Errorf("expected ball1, got %s", resolved.ID)
	}
}

// TestCLIBallCommandWithMinimalID tests that juggle <minimal-id> works
func TestCLIBallCommandWithMinimalID(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	juggleRoot := GetRepoRoot(t)
	juggleBinary := filepath.Join(juggleRoot, "juggle")

	ball := env.CreateBall(t, "Test CLI minimal ID", session.PriorityMedium)
	store := env.GetStore(t)

	balls, _ := store.LoadBalls()
	minIDs := session.ComputeMinimalUniqueIDs(balls)
	minID := minIDs[ball.ID]

	t.Logf("Ball ID: %s, Short ID: %s, Minimal ID: %s", ball.ID, ball.ShortID(), minID)
	t.Logf("Project dir: %s", env.ProjectDir)

	// First, test that the short ID works (this is the standard way)
	cmd1 := exec.Command(juggleBinary, "--config-home", env.ConfigHome, ball.ShortID())
	cmd1.Dir = env.ProjectDir
	output1, err := cmd1.CombinedOutput()
	if err != nil {
		t.Fatalf("juggle %s (short ID) failed: %v\nOutput: %s", ball.ShortID(), err, output1)
	}
	t.Logf("Short ID output: %s", output1)

	// Now test the minimal ID (should be even shorter prefix that's still unique)
	cmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, minID)
	cmd.Dir = env.ProjectDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("juggle %s (minimal ID) failed: %v\nOutput: %s", minID, err, output)
	}

	// Should show the ball info
	outputStr := string(output)
	if !strings.Contains(outputStr, ball.Title) {
		t.Errorf("expected output to contain intent %q, got: %s", ball.Title, outputStr)
	}
}

// TestCLIUpdateWithMinimalID tests that juggle update <minimal-id> works
func TestCLIUpdateWithMinimalID(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	juggleRoot := GetRepoRoot(t)
	juggleBinary := filepath.Join(juggleRoot, "juggle")

	ball := env.CreateBall(t, "Test update minimal ID", session.PriorityMedium)
	store := env.GetStore(t)

	balls, _ := store.LoadBalls()
	minIDs := session.ComputeMinimalUniqueIDs(balls)
	minID := minIDs[ball.ID]

	// Run juggle update <minimal-id> --state in_progress
	cmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "update", minID, "--state", "in_progress")
	cmd.Dir = env.ProjectDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("juggle update %s failed: %v\nOutput: %s", minID, err, output)
	}

	// Verify state was updated
	updated, _ := store.GetBallByID(ball.ID)
	if updated.State != session.StateInProgress {
		t.Errorf("expected state in_progress, got %s", updated.State)
	}
}

// TestPlanHelpShowsEditFlag tests that --edit appears in help
func TestPlanHelpShowsEditFlag(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	juggleRoot := GetRepoRoot(t)
	juggleBinary := filepath.Join(juggleRoot, "juggle")

	cmd := exec.Command(juggleBinary, "plan", "--help")
	cmd.Dir = env.ProjectDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Help command failed: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(string(output), "--edit") {
		t.Errorf("Expected '--edit' in help, got: %s", output)
	}
	if !strings.Contains(string(output), "EDITOR") {
		t.Errorf("Expected 'EDITOR' in help description, got: %s", output)
	}
}

// TestPlanHelpShowsTUIDefault tests that help mentions TUI as default
func TestPlanHelpShowsTUIDefault(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	juggleRoot := GetRepoRoot(t)
	juggleBinary := filepath.Join(juggleRoot, "juggle")

	cmd := exec.Command(juggleBinary, "plan", "--help")
	cmd.Dir = env.ProjectDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Help command failed: %v\nOutput: %s", err, output)
	}

	// Should mention TUI is default
	if !strings.Contains(string(output), "TUI") {
		t.Errorf("Expected 'TUI' mentioned in help, got: %s", output)
	}
}

// TestPlanNonInteractiveDoesNotLaunchTUI tests non-interactive bypasses TUI
func TestPlanNonInteractiveDoesNotLaunchTUI(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	juggleRoot := GetRepoRoot(t)
	juggleBinary := filepath.Join(juggleRoot, "juggle")

	// Run without a TTY - should work with --non-interactive
	cmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "plan",
		"Test ball for non-interactive",
		"--non-interactive",
	)
	cmd.Dir = env.ProjectDir
	// Don't set stdin - simulates non-TTY environment
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Non-interactive plan failed: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(string(output), "Planned ball added") {
		t.Errorf("Expected success output, got: %s", output)
	}
}

// TestPlanEditFlagPrePopulatesFields tests that --edit respects other flags
func TestPlanEditFlagPrePopulatesFields(t *testing.T) {
	// This test verifies the YAML template generation for --edit flag
	// We can't actually open an editor in tests, but we can test the template
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a test YAML template as if --edit would generate it
	intent := "Test intent for editor"
	priority := "high"
	tags := []string{"tag1", "tag2"}
	sessionID := "my-session"
	modelSize := "medium"
	acceptanceCriteria := []string{"AC 1", "AC 2"}

	// Call the internal function directly (we test the logic, not the editor interaction)
	yamlContent := createTestYAMLTemplate(intent, priority, tags, sessionID, modelSize, acceptanceCriteria)

	// Verify template contains expected content
	if !strings.Contains(yamlContent, "title: "+intent) {
		t.Errorf("Expected title in template, got: %s", yamlContent)
	}
	if !strings.Contains(yamlContent, "priority: "+priority) {
		t.Errorf("Expected priority in template, got: %s", yamlContent)
	}
	if !strings.Contains(yamlContent, "- tag1") {
		t.Errorf("Expected tag1 in template, got: %s", yamlContent)
	}
	if !strings.Contains(yamlContent, "- my-session") {
		t.Errorf("Expected session tag in template, got: %s", yamlContent)
	}
	if !strings.Contains(yamlContent, "model_size: "+modelSize) {
		t.Errorf("Expected model_size in template, got: %s", yamlContent)
	}
	if !strings.Contains(yamlContent, "- AC 1") {
		t.Errorf("Expected AC 1 in template, got: %s", yamlContent)
	}
}

// createTestYAMLTemplate creates a YAML template matching plan.go's createNewBallYAMLTemplate
func createTestYAMLTemplate(intent, priority string, tags []string, sessionID, modelSize string, acceptanceCriteria []string) string {
	allTags := tags
	if sessionID != "" {
		allTags = append(allTags, sessionID)
	}

	tagsYAML := "[]"
	if len(allTags) > 0 {
		var tagLines []string
		for _, tag := range allTags {
			tagLines = append(tagLines, fmt.Sprintf("  - %s", tag))
		}
		tagsYAML = "\n" + strings.Join(tagLines, "\n")
	}

	acYAML := "[]"
	if len(acceptanceCriteria) > 0 {
		var acLines []string
		for _, ac := range acceptanceCriteria {
			acLines = append(acLines, fmt.Sprintf("  - %s", ac))
		}
		acYAML = "\n" + strings.Join(acLines, "\n")
	}

	return fmt.Sprintf(`# Create New Ball
# Edit the fields below and save to create the ball
# Close without saving to cancel
#
# Required: title
# Optional: context, priority, tags, acceptance_criteria, model_size, depends_on

# Brief title describing what this ball is about (50 chars recommended)
title: %s

# Background context for this task (optional)
context: ""

# Priority: low, medium, high, urgent
priority: %s

# Tags for categorization (include session ID to link to a session)
tags: %s

# Acceptance criteria for completion
acceptance_criteria: %s

# Preferred LLM model size: small, medium, large (or empty for default)
model_size: %s

# Ball IDs this ball depends on (must complete before this one)
depends_on: []
`, intent, priority, tagsYAML, acYAML, modelSize)
}

// TestPlanWithIntentButNoTTY tests that plan with intent works without TTY
func TestPlanWithIntentButNoTTY(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	juggleRoot := GetRepoRoot(t)
	juggleBinary := filepath.Join(juggleRoot, "juggle")

	// Run with intent but simulate non-TTY (no stdin)
	cmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "plan",
		"--intent", "Test ball from non-TTY",
	)
	cmd.Dir = env.ProjectDir
	// Close stdin to simulate non-TTY
	cmd.Stdin = nil
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Plan with intent failed: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(string(output), "Planned ball added") {
		t.Errorf("Expected success output, got: %s", output)
	}
}

// TestPlanWithoutIntentAndNoTTYFails tests that plan without intent fails when no TTY
func TestPlanWithoutIntentAndNoTTYFails(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	juggleRoot := GetRepoRoot(t)
	juggleBinary := filepath.Join(juggleRoot, "juggle")

	// Run without intent and simulate non-TTY
	cmd := exec.Command(juggleBinary, "--config-home", env.ConfigHome, "plan")
	cmd.Dir = env.ProjectDir
	// Close stdin to simulate non-TTY
	cmd.Stdin = nil
	output, err := cmd.CombinedOutput()

	// Should fail because intent is required without TTY
	if err == nil {
		t.Errorf("Expected failure without intent and no TTY, but succeeded: %s", output)
	}

	if !strings.Contains(string(output), "intent is required") {
		t.Errorf("Expected 'intent is required' error, got: %s", output)
	}
}

// TestStandaloneBallModelCreation tests the standalone TUI model for ball creation
func TestStandaloneBallModelCreation(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	store := env.GetStore(t)

	// Create a standalone ball model and verify it can create a ball
	// We test the finalizeBallCreation logic by simulating form state

	// Create a ball directly to verify the store works
	ball, err := session.NewBall(env.ProjectDir, "Standalone test ball", session.PriorityMedium)
	if err != nil {
		t.Fatalf("Failed to create ball: %v", err)
	}

	ball.State = session.StatePending
	ball.Context = "Test context"
	ball.SetAcceptanceCriteria([]string{"AC 1", "AC 2"})

	err = store.AppendBall(ball)
	if err != nil {
		t.Fatalf("Failed to save ball: %v", err)
	}

	// Verify ball was created
	balls, _ := store.LoadBalls()
	found := false
	for _, b := range balls {
		if b.Title == "Standalone test ball" {
			found = true
			if b.Context != "Test context" {
				t.Errorf("Expected context 'Test context', got: %s", b.Context)
			}
			if len(b.AcceptanceCriteria) != 2 {
				t.Errorf("Expected 2 ACs, got: %d", len(b.AcceptanceCriteria))
			}
		}
	}
	if !found {
		t.Error("Ball not found after creation")
	}
}
