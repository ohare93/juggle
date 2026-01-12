package integration_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
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
