package integration_test

import (
	"strings"
	"testing"

	"github.com/ohare93/juggle/internal/session"
)

// These tests verify the underlying logic used by interactive commands
// without actually running the interactive prompts.

// TestPlanBallCreation tests the logic used by the plan command
func TestPlanBallCreation(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	store := env.GetStore(t)

	// Simulate what the plan command does internally
	ball, err := session.NewBall(env.ProjectDir, "User can authenticate with OAuth", session.PriorityHigh)
	if err != nil {
		t.Fatalf("Failed to create ball: %v", err)
	}

	// Set acceptance criteria as the plan command would
	ball.SetAcceptanceCriteria([]string{
		"User can click login button",
		"OAuth flow redirects to provider",
		"Token is stored securely",
	})

	if err := store.AppendBall(ball); err != nil {
		t.Fatalf("Failed to save ball: %v", err)
	}

	// Verify the ball was created correctly
	retrieved, err := store.GetBallByID(ball.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve ball: %v", err)
	}

	if retrieved.Title != "User can authenticate with OAuth" {
		t.Errorf("Unexpected intent: %s", retrieved.Title)
	}

	if len(retrieved.AcceptanceCriteria) != 3 {
		t.Errorf("Expected 3 acceptance criteria, got %d", len(retrieved.AcceptanceCriteria))
	}
}

// TestInteractiveUpdateLogic tests the logic used by interactive update
func TestInteractiveUpdateLogic(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	store := env.GetStore(t)

	// Create initial ball
	ball := env.CreateBall(t, "Original intent", session.PriorityLow)

	// Simulate interactive update - modify all fields
	ball.Title = "Updated intent"
	ball.Priority = session.PriorityHigh
	ball.SetAcceptanceCriteria([]string{"New criterion"})
	ball.Tags = []string{"updated", "interactive"}

	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// Verify all updates persisted
	retrieved, err := store.GetBallByID(ball.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve ball: %v", err)
	}

	if retrieved.Title != "Updated intent" {
		t.Errorf("Intent not updated: %s", retrieved.Title)
	}

	if retrieved.Priority != session.PriorityHigh {
		t.Errorf("Priority not updated: %s", retrieved.Priority)
	}

	if len(retrieved.AcceptanceCriteria) != 1 {
		t.Errorf("Acceptance criteria not updated")
	}

	if len(retrieved.Tags) != 2 {
		t.Errorf("Tags not updated")
	}
}

// TestConfirmDeleteLogic tests the logic used by delete confirmation
func TestConfirmDeleteLogic(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	store := env.GetStore(t)

	// Create a ball to delete
	ball := env.CreateBall(t, "Ball to delete", session.PriorityMedium)

	// Verify it exists
	_, err := store.GetBallByID(ball.ID)
	if err != nil {
		t.Fatalf("Ball should exist before deletion")
	}

	// Delete it (simulating user confirmed)
	if err := store.DeleteBall(ball.ID); err != nil {
		t.Fatalf("Failed to delete ball: %v", err)
	}

	// Verify it's gone
	_, err = store.GetBallByID(ball.ID)
	if err == nil {
		t.Error("Ball should not exist after deletion")
	}
}

// TestBlockWithReasonLogic tests the blocked state logic
func TestBlockWithReasonLogic(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	store := env.GetStore(t)

	ball := env.CreateBall(t, "Ball to block", session.PriorityMedium)

	// Simulate interactive block with reason
	reason := "Waiting for design approval from stakeholders"
	ball.SetBlocked(reason)

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

	if retrieved.BlockedReason != reason {
		t.Errorf("Blocked reason not set correctly: %s", retrieved.BlockedReason)
	}
}

// TestSessionCreationLogic tests session creation logic
func TestSessionCreationLogic(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	sessionStore, err := session.NewSessionStore(env.JugglerDir)
	if err != nil {
		t.Fatalf("Failed to create session store: %v", err)
	}

	// Simulate interactive session creation
	sess, err := sessionStore.CreateSession("feature-auth", "Implement user authentication system")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	if sess.ID != "feature-auth" {
		t.Errorf("Unexpected session ID: %s", sess.ID)
	}

	if sess.Description != "Implement user authentication system" {
		t.Errorf("Unexpected description: %s", sess.Description)
	}

	// Verify session persisted
	loaded, err := sessionStore.LoadSession("feature-auth")
	if err != nil {
		t.Fatalf("Failed to load session: %v", err)
	}

	if loaded.ID != sess.ID {
		t.Error("Loaded session doesn't match created session")
	}
}

// TestTaggingBallToSession tests the tagging logic
func TestTaggingBallToSession(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	store := env.GetStore(t)

	// Create a session
	sessionStore, err := session.NewSessionStore(env.JugglerDir)
	if err != nil {
		t.Fatalf("Failed to create session store: %v", err)
	}

	sess, err := sessionStore.CreateSession("my-session", "My session")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Create a ball
	ball := env.CreateBall(t, "Ball to tag", session.PriorityMedium)

	// Tag ball with session (simulating TUI tagging action)
	ball.AddTag(sess.ID)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// Verify tag was added
	retrieved, err := store.GetBallByID(ball.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve ball: %v", err)
	}

	hasTag := false
	for _, tag := range retrieved.Tags {
		if tag == sess.ID {
			hasTag = true
			break
		}
	}

	if !hasTag {
		t.Errorf("Ball should have session tag '%s', has: %v", sess.ID, retrieved.Tags)
	}
}

// TestValidatePriority tests priority validation
func TestValidatePriority(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"low", true},
		{"medium", true},
		{"high", true},
		{"urgent", true},
		{"LOW", false},  // Case sensitive
		{"URGENT", false},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := session.ValidatePriority(tt.input)
			if result != tt.valid {
				t.Errorf("ValidatePriority(%q) = %v, want %v", tt.input, result, tt.valid)
			}
		})
	}
}

// TestValidateBallState tests ball state validation
func TestValidateBallState(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"pending", true},
		{"in_progress", true},
		{"blocked", true},
		{"complete", true},
		{"PENDING", false}, // Case sensitive
		{"in-progress", false},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := session.ValidateBallState(tt.input)
			if result != tt.valid {
				t.Errorf("ValidateBallState(%q) = %v, want %v", tt.input, result, tt.valid)
			}
		})
	}
}

// TestInputParsing tests parsing of comma-separated tags
func TestInputParsing(t *testing.T) {
	// Simulate parsing comma-separated tags from input
	input := "feature, frontend, urgent, v2"

	tags := strings.Split(input, ",")
	for i := range tags {
		tags[i] = strings.TrimSpace(tags[i])
	}

	expected := []string{"feature", "frontend", "urgent", "v2"}

	if len(tags) != len(expected) {
		t.Errorf("Expected %d tags, got %d", len(expected), len(tags))
	}

	for i, tag := range tags {
		if tag != expected[i] {
			t.Errorf("Tag %d: expected %q, got %q", i, expected[i], tag)
		}
	}
}

// TestAcceptanceCriteriaParsing tests parsing multi-line acceptance criteria
func TestAcceptanceCriteriaParsing(t *testing.T) {
	// Simulate multi-line input for acceptance criteria
	input := `User can log in
User can log out
Session persists across page reloads
`

	lines := strings.Split(strings.TrimSpace(input), "\n")
	var criteria []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			criteria = append(criteria, line)
		}
	}

	if len(criteria) != 3 {
		t.Errorf("Expected 3 criteria, got %d", len(criteria))
	}

	if criteria[0] != "User can log in" {
		t.Errorf("First criterion incorrect: %s", criteria[0])
	}
}

// TestStartFromDifferentStates tests starting balls from various states
// Note: Start() only works from pending state by design
func TestStartFromDifferentStates(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	store := env.GetStore(t)

	tests := []struct {
		name          string
		initialState  session.BallState
		expectedState session.BallState
	}{
		// Start() only transitions from pending
		{"from_pending", session.StatePending, session.StateInProgress},
		// From other states, Start() is a no-op
		{"from_blocked", session.StateBlocked, session.StateBlocked},
		{"from_complete", session.StateComplete, session.StateComplete},
		{"from_in_progress", session.StateInProgress, session.StateInProgress},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ball := env.CreateBall(t, "Start test "+tt.name, session.PriorityMedium)

			// Set initial state
			ball.SetState(tt.initialState)
			store.UpdateBall(ball)

			// Start the ball
			ball.Start()
			store.UpdateBall(ball)

			retrieved, _ := store.GetBallByID(ball.ID)
			if retrieved.State != tt.expectedState {
				t.Errorf("Expected %s after start from %s, got %s", tt.expectedState, tt.initialState, retrieved.State)
			}
		})
	}
}

// TestCompleteFromDifferentStates tests completing balls from various states
func TestCompleteFromDifferentStates(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	store := env.GetStore(t)

	tests := []struct {
		name         string
		initialState session.BallState
	}{
		{"from_pending", session.StatePending},
		{"from_in_progress", session.StateInProgress},
		{"from_blocked", session.StateBlocked},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ball := env.CreateBall(t, "Complete test "+tt.name, session.PriorityMedium)

			// Set initial state
			ball.SetState(tt.initialState)
			store.UpdateBall(ball)

			// Complete the ball
			ball.SetState(session.StateComplete)
			store.UpdateBall(ball)

			retrieved, _ := store.GetBallByID(ball.ID)
			if retrieved.State != session.StateComplete {
				t.Errorf("Expected complete after completing from %s, got %s", tt.initialState, retrieved.State)
			}
		})
	}
}
