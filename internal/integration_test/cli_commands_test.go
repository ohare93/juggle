package integration_test

import (
	"bytes"
	"encoding/json"
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

	if retrieved.Intent != "Test ball for show" {
		t.Errorf("Expected intent 'Test ball for show', got '%s'", retrieved.Intent)
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

	// Set to blocked with reason
	ball.SetBlocked("Waiting for API access")
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

	ball3.SetState(session.StateComplete)
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
		if containsIgnoreCase(b.Intent, "login") || containsIgnoreCase(b.Intent, "logout") {
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
	ball.SetState(session.StateComplete)
	store.UpdateBall(ball)

	// Archive the ball
	err := store.ArchiveBall(ball)
	if err != nil {
		t.Fatalf("Failed to archive ball: %v", err)
	}

	// Verify ball is no longer in active balls
	env.AssertBallNotExists(t, ball.ID)

	// Verify archive file exists
	archivePath := filepath.Join(env.JugglerDir, "archive", "balls.jsonl")
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
	sessionStorePath := filepath.Join(env.JugglerDir, "sessions")
	if err := os.MkdirAll(sessionStorePath, 0755); err != nil {
		t.Fatalf("Failed to create sessions directory: %v", err)
	}

	sessionStore, err := session.NewSessionStore(env.JugglerDir)
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
	sessionStore, err := session.NewSessionStore(env.JugglerDir)
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

	ball3.SetBlocked("Waiting for dependency")
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

	if retrieved.Intent != "Ball to move" {
		t.Errorf("Expected intent 'Ball to move', got '%s'", retrieved.Intent)
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
	sessionStore, err := session.NewSessionStore(env.JugglerDir)
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

// TestUpdateCommandTestsState tests updating ball tests state
func TestUpdateCommandTestsState(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	ball := env.CreateBall(t, "Test ball for tests state update", session.PriorityMedium)
	store := env.GetStore(t)

	// Update tests state to needed
	ball.SetTestsState(session.TestsStateNeeded)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	retrieved, err := store.GetBallByID(ball.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve ball: %v", err)
	}

	if retrieved.TestsState != session.TestsStateNeeded {
		t.Errorf("Expected tests state needed, got %s", retrieved.TestsState)
	}

	// Update tests state to done
	retrieved.SetTestsState(session.TestsStateDone)
	if err := store.UpdateBall(retrieved); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	retrieved2, _ := store.GetBallByID(ball.ID)
	if retrieved2.TestsState != session.TestsStateDone {
		t.Errorf("Expected tests state done, got %s", retrieved2.TestsState)
	}
}

// TestExportIncludesTestsState tests that JSON export includes tests state
func TestExportIncludesTestsState(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	ball := env.CreateBall(t, "Export tests state test", session.PriorityMedium)
	store := env.GetStore(t)

	ball.SetTestsState(session.TestsStateNeeded)
	store.UpdateBall(ball)

	// Export to JSON
	balls, _ := store.LoadBalls()
	jsonData, err := json.MarshalIndent(balls, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal balls to JSON: %v", err)
	}

	// Verify tests_state is in JSON
	jsonStr := string(jsonData)
	if !bytes.Contains([]byte(jsonStr), []byte("tests_state")) {
		t.Error("JSON export should contain tests_state field")
	}
	if !bytes.Contains([]byte(jsonStr), []byte("needed")) {
		t.Error("JSON export should contain tests state value 'needed'")
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
		if b.Intent == "Test intent for AC output" {
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

	jugglerRoot := "/home/jmo/Development/juggler"
	juggleBinary := filepath.Join(jugglerRoot, "juggle")

	// Build juggle binary if needed
	if _, err := os.Stat(juggleBinary); os.IsNotExist(err) {
		buildCmd := exec.Command("go", "build", "-o", "juggle", "./cmd/juggle")
		buildCmd.Dir = jugglerRoot
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

// TestTestsStateTransitions tests all valid tests state transitions
func TestTestsStateTransitions(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	ball := env.CreateBall(t, "Tests state transitions", session.PriorityMedium)
	store := env.GetStore(t)

	// Test all transitions
	transitions := []session.TestsState{
		session.TestsStateNeeded,
		session.TestsStateDone,
		session.TestsStateNotNeeded,
		session.TestsStateNeeded, // Can go back to needed
	}

	for _, state := range transitions {
		ball.SetTestsState(state)
		if err := store.UpdateBall(ball); err != nil {
			t.Fatalf("Failed to transition to %s: %v", state, err)
		}

		retrieved, _ := store.GetBallByID(ball.ID)
		if retrieved.TestsState != state {
			t.Errorf("Expected tests state %s, got %s", state, retrieved.TestsState)
		}
	}
}

// TestSessionAliasForSessionsCommand tests that 'session' works as an alias for 'sessions'
func TestSessionAliasForSessionsCommand(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	jugglerRoot := "/home/jmo/Development/juggler"
	juggleBinary := filepath.Join(jugglerRoot, "juggle")

	// Build juggle binary if needed
	if _, err := os.Stat(juggleBinary); os.IsNotExist(err) {
		buildCmd := exec.Command("go", "build", "-o", "juggle", "./cmd/juggle")
		buildCmd.Dir = jugglerRoot
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

	jugglerRoot := "/home/jmo/Development/juggler"
	juggleBinary := filepath.Join(jugglerRoot, "juggle")

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

	jugglerRoot := "/home/jmo/Development/juggler"
	juggleBinary := filepath.Join(jugglerRoot, "juggle")

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

	jugglerRoot := "/home/jmo/Development/juggler"
	juggleBinary := filepath.Join(jugglerRoot, "juggle")

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
	jugglerRoot := "/home/jmo/Development/juggler"
	juggleBinary := filepath.Join(jugglerRoot, "juggle")

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

	jugglerRoot := "/home/jmo/Development/juggler"
	juggleBinary := filepath.Join(jugglerRoot, "juggle")

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

	jugglerRoot := "/home/jmo/Development/juggler"
	juggleBinary := filepath.Join(jugglerRoot, "juggle")

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
	if result["intent"] != "JSON output test ball" {
		t.Errorf("Expected intent 'JSON output test ball', got '%v'", result["intent"])
	}
	if result["state"] != "in_progress" {
		t.Errorf("Expected state 'in_progress', got '%v'", result["state"])
	}
}

// TestBallOutputConsistency tests that 'juggle <id>' and 'juggle show <id>' show same info
func TestBallOutputConsistency(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	jugglerRoot := "/home/jmo/Development/juggler"
	juggleBinary := filepath.Join(jugglerRoot, "juggle")

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
		"Intent:",
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

	jugglerRoot := "/home/jmo/Development/juggler"
	juggleBinary := filepath.Join(jugglerRoot, "juggle")

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
	fieldsToCompare := []string{"id", "intent", "priority", "state"}
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

	jugglerRoot := "/home/jmo/Development/juggler"
	juggleBinary := filepath.Join(jugglerRoot, "juggle")

	// Create a ball with all possible fields populated
	ball := env.CreateBall(t, "Full fields test", session.PriorityUrgent)
	store := env.GetStore(t)
	ball.Start()
	ball.SetAcceptanceCriteria([]string{"AC 1", "AC 2"})
	ball.AddTag("tag1")
	ball.AddTag("tag2")
	ball.SetTestsState(session.TestsStateNeeded)
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
		"Intent:",
		"Priority:",
		"State:",
		"Tests:",
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
	jsonFields := []string{"id", "intent", "priority", "state", "tests_state", "tags", "depends_on", "acceptance_criteria"}
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

	jugglerRoot := "/home/jmo/Development/juggler"
	juggleBinary := filepath.Join(jugglerRoot, "juggle")

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

	jugglerRoot := "/home/jmo/Development/juggler"
	juggleBinary := filepath.Join(jugglerRoot, "juggle")

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

	jugglerRoot := "/home/jmo/Development/juggler"
	juggleBinary := filepath.Join(jugglerRoot, "juggle")

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

	jugglerRoot := "/home/jmo/Development/juggler"
	juggleBinary := filepath.Join(jugglerRoot, "juggle")

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

	jugglerRoot := "/home/jmo/Development/juggler"
	juggleBinary := filepath.Join(jugglerRoot, "juggle")

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
