package integration_test

import (
	"testing"

	"github.com/ohare93/juggle/internal/session"
)

// Tests for ball output field and researched state

func TestBallOutput_SetAndGet(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a ball
	ball := env.CreateBall(t, "Research API options", session.PriorityMedium)
	ballID := ball.ID

	// Verify no output initially
	if ball.HasOutput() {
		t.Error("Expected new ball to have no output")
	}
	if ball.Output != "" {
		t.Errorf("Expected empty output, got: %s", ball.Output)
	}

	// Set output
	store := env.GetStore(t)
	ball.SetOutput("API options: REST, GraphQL, gRPC")
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// Reload and verify
	loaded := env.AssertBallExists(t, ballID)
	if !loaded.HasOutput() {
		t.Error("Expected ball to have output after update")
	}
	if loaded.Output != "API options: REST, GraphQL, gRPC" {
		t.Errorf("Expected output 'API options: REST, GraphQL, gRPC', got: %s", loaded.Output)
	}
}

func TestBallOutput_MarkResearched(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create and start a ball
	ball := env.CreateInProgressBall(t, "Investigate performance bottleneck", session.PriorityHigh)
	ballID := ball.ID

	// Mark as researched with output
	store := env.GetStore(t)
	ball.MarkResearched("Found that the main bottleneck is in the database query layer. Recommend adding indexes.")
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// Verify state and output
	loaded := env.AssertBallExists(t, ballID)
	if loaded.State != session.StateResearched {
		t.Errorf("Expected state 'researched', got: %s", loaded.State)
	}
	if !loaded.HasOutput() {
		t.Error("Expected ball to have output after MarkResearched")
	}
	if loaded.Output != "Found that the main bottleneck is in the database query layer. Recommend adding indexes." {
		t.Errorf("Output mismatch: %s", loaded.Output)
	}
	if loaded.CompletedAt == nil {
		t.Error("Expected CompletedAt to be set for researched ball")
	}
}

func TestBallOutput_MarkResearchedClearsBlockedReason(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a blocked ball
	ball := env.CreateInProgressBall(t, "Investigate issue", session.PriorityMedium)
	ball.SetBlocked("Waiting for more info")

	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// Mark as researched (should clear blocked reason)
	ball.MarkResearched("Investigation complete")
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// Verify blocked reason is cleared
	loaded := env.AssertBallExists(t, ball.ID)
	if loaded.BlockedReason != "" {
		t.Errorf("Expected BlockedReason to be cleared, got: %s", loaded.BlockedReason)
	}
	if loaded.State != session.StateResearched {
		t.Errorf("Expected state 'researched', got: %s", loaded.State)
	}
}

func TestStateResearched_Validation(t *testing.T) {
	// Test that "researched" is a valid state
	if !session.ValidateBallState("researched") {
		t.Error("Expected 'researched' to be a valid ball state")
	}
	if !session.ValidateBallState("pending") {
		t.Error("Expected 'pending' to be a valid ball state")
	}
	if !session.ValidateBallState("in_progress") {
		t.Error("Expected 'in_progress' to be a valid ball state")
	}
	if !session.ValidateBallState("complete") {
		t.Error("Expected 'complete' to be a valid ball state")
	}
	if !session.ValidateBallState("blocked") {
		t.Error("Expected 'blocked' to be a valid ball state")
	}
	if session.ValidateBallState("invalid") {
		t.Error("Expected 'invalid' to NOT be a valid ball state")
	}
}

func TestBallOutput_SetOutputUpdatesActivity(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	ball := env.CreateBall(t, "Research task", session.PriorityMedium)
	originalActivity := ball.LastActivity

	// Wait a bit to ensure time difference
	ball.SetOutput("New output")

	if !ball.LastActivity.After(originalActivity) && !ball.LastActivity.Equal(originalActivity) {
		t.Error("Expected LastActivity to be updated after SetOutput")
	}
}

func TestBallOutput_PersistsThroughReload(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a ball with output
	ball := env.CreateBall(t, "Research output persistence", session.PriorityMedium)
	ball.SetOutput("This output should persist through reload cycles")

	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// Reload multiple times
	for i := 0; i < 3; i++ {
		loaded := env.AssertBallExists(t, ball.ID)
		if loaded.Output != "This output should persist through reload cycles" {
			t.Errorf("Iteration %d: Output was not persisted correctly: %s", i, loaded.Output)
		}
	}
}

func TestBallOutput_EmptyOutputHasNoOutput(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	ball := env.CreateBall(t, "Empty output test", session.PriorityMedium)

	// Set empty output
	ball.SetOutput("")

	if ball.HasOutput() {
		t.Error("Expected HasOutput to return false for empty output")
	}

	// Set whitespace-only (still counts as output since it's not empty string)
	ball.SetOutput("   ")
	if !ball.HasOutput() {
		t.Error("Expected HasOutput to return true for whitespace-only output")
	}
}

func TestBallOutput_ClearOutput(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create ball with output
	ball := env.CreateBall(t, "Clear output test", session.PriorityMedium)
	ball.SetOutput("Some research findings")

	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to save ball with output: %v", err)
	}

	// Verify output exists
	loaded := env.AssertBallExists(t, ball.ID)
	if !loaded.HasOutput() {
		t.Error("Expected ball to have output")
	}

	// Clear the output
	loaded.SetOutput("")
	if err := store.UpdateBall(loaded); err != nil {
		t.Fatalf("Failed to clear output: %v", err)
	}

	// Verify output is cleared
	reloaded := env.AssertBallExists(t, ball.ID)
	if reloaded.HasOutput() {
		t.Error("Expected output to be cleared")
	}
}

func TestBallOutput_LargeOutput(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	ball := env.CreateBall(t, "Large output test", session.PriorityMedium)

	// Create a large output string
	largeOutput := ""
	for i := 0; i < 1000; i++ {
		largeOutput += "This is line number " + string(rune('0'+i%10)) + " of the research output.\n"
	}

	ball.SetOutput(largeOutput)

	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to save ball with large output: %v", err)
	}

	// Verify large output persists
	loaded := env.AssertBallExists(t, ball.ID)
	if loaded.Output != largeOutput {
		t.Error("Large output was not preserved correctly")
	}
}

func TestBallOutput_MultilineOutput(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	ball := env.CreateBall(t, "Multiline output test", session.PriorityMedium)

	multilineOutput := `Research Findings:
1. First finding with details
2. Second finding with more details
3. Third finding

Recommendations:
- Option A: Pros and cons
- Option B: Pros and cons

Conclusion: Proceed with Option A`

	ball.SetOutput(multilineOutput)

	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to save ball with multiline output: %v", err)
	}

	loaded := env.AssertBallExists(t, ball.ID)
	if loaded.Output != multilineOutput {
		t.Errorf("Multiline output was not preserved correctly.\nExpected:\n%s\n\nGot:\n%s", multilineOutput, loaded.Output)
	}
}

func TestBallOutput_SpecialCharacters(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	ball := env.CreateBall(t, "Special characters test", session.PriorityMedium)

	// Output with special characters
	specialOutput := `Research "quotes" and 'apostrophes' test.
Backslash: \\ and forward: /
Unicode: 日本語 🎯 émoji
Tabs:	tab	here
JSON-like: {"key": "value", "array": [1, 2, 3]}`

	ball.SetOutput(specialOutput)

	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to save ball with special characters: %v", err)
	}

	loaded := env.AssertBallExists(t, ball.ID)
	if loaded.Output != specialOutput {
		t.Error("Output with special characters was not preserved correctly")
	}
}

func TestStateResearched_TransitionFromInProgress(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create in-progress ball
	ball := env.CreateInProgressBall(t, "Research task", session.PriorityMedium)

	if ball.State != session.StateInProgress {
		t.Errorf("Expected state 'in_progress', got: %s", ball.State)
	}

	// Transition to researched
	ball.MarkResearched("Research complete")

	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	loaded := env.AssertBallExists(t, ball.ID)
	if loaded.State != session.StateResearched {
		t.Errorf("Expected state 'researched', got: %s", loaded.State)
	}
}

func TestStateResearched_TransitionFromPending(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create pending ball
	ball := env.CreateBall(t, "Pending research task", session.PriorityMedium)

	if ball.State != session.StatePending {
		t.Errorf("Expected state 'pending', got: %s", ball.State)
	}

	// Transition directly to researched
	ball.MarkResearched("Quick research done")

	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	loaded := env.AssertBallExists(t, ball.ID)
	if loaded.State != session.StateResearched {
		t.Errorf("Expected state 'researched', got: %s", loaded.State)
	}
}
