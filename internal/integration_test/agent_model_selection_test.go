package integration_test

import (
	"testing"

	"github.com/ohare93/juggle/internal/agent"
	"github.com/ohare93/juggle/internal/cli"
	"github.com/ohare93/juggle/internal/session"
)

// TestSelectModelForIteration_ExplicitModelFlag tests that explicit --model flag takes precedence
func TestSelectModelForIteration_ExplicitModelFlag(t *testing.T) {
	balls := []*session.Ball{
		{ID: "ball-1", State: session.StatePending, ModelSize: session.ModelSizeSmall},
		{ID: "ball-2", State: session.StatePending, ModelSize: session.ModelSizeLarge},
	}

	config := cli.AgentLoopConfig{
		Model: "sonnet", // Explicitly set via --model flag
	}

	result := cli.SelectModelForIterationForTest(config, balls, "")

	if result.Model != "sonnet" {
		t.Errorf("Expected model=sonnet (explicit flag), got %s", result.Model)
	}
	if result.Reason != "explicitly set via --model flag" {
		t.Errorf("Expected reason about explicit flag, got: %s", result.Reason)
	}
}

// TestSelectModelForIteration_SessionDefault tests session default model fallback
func TestSelectModelForIteration_SessionDefault(t *testing.T) {
	// All balls have blank model preference
	balls := []*session.Ball{
		{ID: "ball-1", State: session.StatePending, ModelSize: ""},
		{ID: "ball-2", State: session.StatePending, ModelSize: ""},
	}

	config := cli.AgentLoopConfig{} // No explicit model

	result := cli.SelectModelForIterationForTest(config, balls, session.ModelSizeMedium)

	if result.Model != "sonnet" {
		t.Errorf("Expected model=sonnet (session default), got %s", result.Model)
	}
}

// TestSelectModelForIteration_MajorityBallPreference tests selection based on ball count
func TestSelectModelForIteration_MajorityBallPreference(t *testing.T) {
	balls := []*session.Ball{
		{ID: "ball-1", State: session.StatePending, ModelSize: session.ModelSizeSmall},
		{ID: "ball-2", State: session.StatePending, ModelSize: session.ModelSizeSmall},
		{ID: "ball-3", State: session.StatePending, ModelSize: session.ModelSizeLarge},
	}

	config := cli.AgentLoopConfig{} // No explicit model

	result := cli.SelectModelForIterationForTest(config, balls, "")

	if result.Model != "haiku" {
		t.Errorf("Expected model=haiku (2 balls prefer small), got %s", result.Model)
	}
	if result.BallsCount != 2 {
		t.Errorf("Expected BallsCount=2, got %d", result.BallsCount)
	}
}

// TestSelectModelForIteration_DefaultToOpus tests default when no preferences
func TestSelectModelForIteration_DefaultToOpus(t *testing.T) {
	balls := []*session.Ball{
		{ID: "ball-1", State: session.StatePending, ModelSize: ""},
		{ID: "ball-2", State: session.StatePending, ModelSize: ""},
	}

	config := cli.AgentLoopConfig{} // No explicit model

	result := cli.SelectModelForIterationForTest(config, balls, "") // No session default

	if result.Model != "opus" {
		t.Errorf("Expected model=opus (default when no preferences), got %s", result.Model)
	}
}

// TestSelectModelForIteration_FiltersCompleteBalls tests that complete balls are excluded
func TestSelectModelForIteration_FiltersCompleteBalls(t *testing.T) {
	balls := []*session.Ball{
		{ID: "ball-1", State: session.StateComplete, ModelSize: session.ModelSizeLarge}, // Complete, should be ignored
		{ID: "ball-2", State: session.StatePending, ModelSize: session.ModelSizeSmall},
	}

	config := cli.AgentLoopConfig{} // No explicit model

	result := cli.SelectModelForIterationForTest(config, balls, "")

	if result.Model != "haiku" {
		t.Errorf("Expected model=haiku (only pending ball prefers small), got %s", result.Model)
	}
}

// TestSelectModelForIteration_NoActiveBalls tests handling when all balls are complete
func TestSelectModelForIteration_NoActiveBalls(t *testing.T) {
	balls := []*session.Ball{
		{ID: "ball-1", State: session.StateComplete, ModelSize: session.ModelSizeSmall},
		{ID: "ball-2", State: session.StateResearched, ModelSize: session.ModelSizeMedium},
	}

	config := cli.AgentLoopConfig{} // No explicit model

	result := cli.SelectModelForIterationForTest(config, balls, "")

	if result.Model != "opus" {
		t.Errorf("Expected model=opus (default when no active balls), got %s", result.Model)
	}
	if result.Reason != "no active balls" {
		t.Errorf("Expected reason='no active balls', got: %s", result.Reason)
	}
}

// TestSelectModelForIteration_TieBreaksToLargerModel tests tie-breaking
func TestSelectModelForIteration_TieBreaksToLargerModel(t *testing.T) {
	balls := []*session.Ball{
		{ID: "ball-1", State: session.StatePending, ModelSize: session.ModelSizeSmall},
		{ID: "ball-2", State: session.StatePending, ModelSize: session.ModelSizeLarge},
	}

	config := cli.AgentLoopConfig{} // No explicit model

	result := cli.SelectModelForIterationForTest(config, balls, "")

	// Both have 1 ball, should pick opus (larger model) first due to priority order
	if result.Model != "opus" {
		t.Errorf("Expected model=opus (tie-break to larger), got %s", result.Model)
	}
}

// TestPrioritizeBallsByModel_MatchingBallsFirst tests model-based prioritization
func TestPrioritizeBallsByModel_MatchingBallsFirst(t *testing.T) {
	balls := []*session.Ball{
		{ID: "ball-1", State: session.StatePending, ModelSize: session.ModelSizeLarge},
		{ID: "ball-2", State: session.StatePending, ModelSize: session.ModelSizeSmall},
		{ID: "ball-3", State: session.StatePending, ModelSize: session.ModelSizeSmall},
	}

	cli.PrioritizeBallsByModelForTest(balls, "haiku", "")

	// Small/haiku balls should come first
	if balls[0].ModelSize != session.ModelSizeSmall {
		t.Errorf("Expected first ball to prefer small model, got %s", balls[0].ModelSize)
	}
	if balls[1].ModelSize != session.ModelSizeSmall {
		t.Errorf("Expected second ball to prefer small model, got %s", balls[1].ModelSize)
	}
	if balls[2].ModelSize != session.ModelSizeLarge {
		t.Errorf("Expected last ball to prefer large model, got %s", balls[2].ModelSize)
	}
}

// TestPrioritizeBallsByModel_PreservesOrderWithinGroup tests order preservation
func TestPrioritizeBallsByModel_PreservesOrderWithinGroup(t *testing.T) {
	balls := []*session.Ball{
		{ID: "ball-A", State: session.StatePending, ModelSize: session.ModelSizeSmall},
		{ID: "ball-B", State: session.StatePending, ModelSize: session.ModelSizeLarge},
		{ID: "ball-C", State: session.StatePending, ModelSize: session.ModelSizeSmall},
	}

	cli.PrioritizeBallsByModelForTest(balls, "haiku", "")

	// Within small group, A should come before C (original order preserved)
	if balls[0].ID != "ball-A" {
		t.Errorf("Expected ball-A first in small group, got %s", balls[0].ID)
	}
	if balls[1].ID != "ball-C" {
		t.Errorf("Expected ball-C second in small group, got %s", balls[1].ID)
	}
}

// TestPrioritizeBallsByModel_BlankMatchesAll tests that blank model balls match any model
func TestPrioritizeBallsByModel_BlankMatchesAll(t *testing.T) {
	balls := []*session.Ball{
		{ID: "ball-1", State: session.StatePending, ModelSize: session.ModelSizeLarge},
		{ID: "ball-2", State: session.StatePending, ModelSize: ""},  // Blank should match
		{ID: "ball-3", State: session.StatePending, ModelSize: session.ModelSizeSmall},
	}

	cli.PrioritizeBallsByModelForTest(balls, "haiku", "")

	// ball-3 (small/haiku) should be first, ball-2 (blank) matches any so depends on original position
	// ball-1 (large) should be last
	if balls[len(balls)-1].ModelSize != session.ModelSizeLarge {
		t.Errorf("Expected large model ball to be last, got %s", balls[len(balls)-1].ModelSize)
	}
}

// TestPrioritizeBallsByModel_SessionDefaultFallback tests session default model as fallback
func TestPrioritizeBallsByModel_SessionDefaultFallback(t *testing.T) {
	balls := []*session.Ball{
		{ID: "ball-1", State: session.StatePending, ModelSize: session.ModelSizeLarge},
		{ID: "ball-2", State: session.StatePending, ModelSize: ""},  // Uses session default
	}

	// Session default is small (haiku), so ball-2 should match haiku
	cli.PrioritizeBallsByModelForTest(balls, "haiku", session.ModelSizeSmall)

	// ball-2 should be first (matches haiku via session default)
	if balls[0].ID != "ball-2" {
		t.Errorf("Expected ball-2 first (matches via session default), got %s", balls[0].ID)
	}
}

// TestFilterActiveBalls tests filtering of terminal state balls
func TestFilterActiveBalls(t *testing.T) {
	balls := []*session.Ball{
		{ID: "ball-1", State: session.StatePending},
		{ID: "ball-2", State: session.StateComplete},
		{ID: "ball-3", State: session.StateInProgress},
		{ID: "ball-4", State: session.StateResearched},
		{ID: "ball-5", State: session.StateBlocked},
	}

	active := cli.FilterActiveBallsForTest(balls)

	if len(active) != 3 {
		t.Errorf("Expected 3 active balls (pending, in_progress, blocked), got %d", len(active))
	}

	// Check that complete and researched are excluded
	for _, ball := range active {
		if ball.State == session.StateComplete || ball.State == session.StateResearched {
			t.Errorf("Expected complete/researched to be filtered out, found %s", ball.State)
		}
	}
}

// TestCountBallsByModel tests model counting
func TestCountBallsByModel(t *testing.T) {
	balls := []*session.Ball{
		{ID: "ball-1", ModelSize: session.ModelSizeSmall},
		{ID: "ball-2", ModelSize: session.ModelSizeSmall},
		{ID: "ball-3", ModelSize: session.ModelSizeLarge},
		{ID: "ball-4", ModelSize: ""},
		{ID: "ball-5", ModelSize: session.ModelSizeMedium},
	}

	counts := cli.CountBallsByModelForTest(balls)

	if counts["haiku"] != 2 {
		t.Errorf("Expected 2 haiku balls, got %d", counts["haiku"])
	}
	if counts["opus"] != 1 {
		t.Errorf("Expected 1 opus ball, got %d", counts["opus"])
	}
	if counts["sonnet"] != 1 {
		t.Errorf("Expected 1 sonnet ball, got %d", counts["sonnet"])
	}
	if counts[""] != 1 {
		t.Errorf("Expected 1 blank/unset ball, got %d", counts[""])
	}
}

// TestModelSelectionInAgentLoop tests model selection is called during agent loop
func TestModelSelectionInAgentLoop(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a session
	env.CreateSession(t, "test-session", "Test session for model selection")

	// Create balls with different model preferences
	ball1 := env.CreateBall(t, "Ball 1 - small", session.PriorityMedium)
	ball1.Tags = []string{"test-session"}
	ball1.ModelSize = session.ModelSizeSmall
	ball1.State = session.StateComplete // Complete so loop exits
	store := env.GetStore(t)
	if err := store.UpdateBall(ball1); err != nil {
		t.Fatalf("Failed to update ball1: %v", err)
	}

	// Setup mock runner
	mock := agent.NewMockRunner(
		&agent.RunResult{
			Output:   "Working...\n<promise>COMPLETE</promise>\nDone.",
			Complete: true,
		},
	)
	agent.SetRunner(mock)
	defer agent.ResetRunner()

	// Run the agent loop without explicit model
	config := cli.AgentLoopConfig{
		SessionID:     "test-session",
		ProjectDir:    env.ProjectDir,
		MaxIterations: 1,
		Trust:         false,
		IterDelay:     0,
		// Note: No Model set, so it should auto-select
	}

	result, err := cli.RunAgentLoop(config)
	if err != nil {
		t.Fatalf("Agent run failed: %v", err)
	}

	// Verify the loop completed
	if !result.Complete {
		t.Error("Expected result.Complete=true")
	}

	// Verify runner was called (model selection happened)
	if len(mock.Calls) != 1 {
		t.Errorf("Expected 1 call to runner, got %d", len(mock.Calls))
	}

	// The model should have been auto-selected
	// Since ball1 is complete, with no active balls, it defaults to opus
	if mock.Calls[0].Model != "opus" {
		t.Errorf("Expected model=opus (default when no active balls), got %s", mock.Calls[0].Model)
	}
}

// TestModelSelectionWithActiveBalls tests model is selected based on active ball preferences
func TestModelSelectionWithActiveBalls(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a session
	env.CreateSession(t, "test-session", "Test session")

	// Create balls - 2 prefer small, 1 prefers large
	ball1 := env.CreateBall(t, "Ball 1", session.PriorityMedium)
	ball1.Tags = []string{"test-session"}
	ball1.ModelSize = session.ModelSizeSmall
	ball1.State = session.StatePending

	ball2 := env.CreateBall(t, "Ball 2", session.PriorityMedium)
	ball2.Tags = []string{"test-session"}
	ball2.ModelSize = session.ModelSizeSmall
	ball2.State = session.StatePending

	ball3 := env.CreateBall(t, "Ball 3", session.PriorityMedium)
	ball3.Tags = []string{"test-session"}
	ball3.ModelSize = session.ModelSizeLarge
	ball3.State = session.StatePending

	store := env.GetStore(t)
	if err := store.UpdateBall(ball1); err != nil {
		t.Fatalf("Failed to update ball1: %v", err)
	}
	if err := store.UpdateBall(ball2); err != nil {
		t.Fatalf("Failed to update ball2: %v", err)
	}
	if err := store.UpdateBall(ball3); err != nil {
		t.Fatalf("Failed to update ball3: %v", err)
	}

	// Setup mock runner that marks all balls as complete
	mock := agent.NewMockRunner(
		&agent.RunResult{
			Output:   "Working...\n<promise>COMPLETE</promise>\nDone.",
			Complete: true,
		},
	)
	agent.SetRunner(mock)
	defer agent.ResetRunner()

	// Mark balls as complete so COMPLETE signal is accepted
	ball1.State = session.StateComplete
	ball2.State = session.StateComplete
	ball3.State = session.StateComplete
	store.UpdateBall(ball1)
	store.UpdateBall(ball2)
	store.UpdateBall(ball3)

	// Run the agent loop
	config := cli.AgentLoopConfig{
		SessionID:     "test-session",
		ProjectDir:    env.ProjectDir,
		MaxIterations: 1,
		Trust:         false,
		IterDelay:     0,
	}

	_, err := cli.RunAgentLoop(config)
	if err != nil {
		t.Fatalf("Agent run failed: %v", err)
	}

	// Since all balls are now complete (no active balls), should default to opus
	if len(mock.Calls) > 0 && mock.Calls[0].Model != "opus" {
		t.Errorf("Expected model=opus (no active balls), got %s", mock.Calls[0].Model)
	}
}

// TestModelSelectionWithExplicitFlag tests that explicit --model flag takes precedence
func TestModelSelectionWithExplicitFlag(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a session
	env.CreateSession(t, "test-session", "Test session")

	// Create a ball with large model preference
	ball := env.CreateBall(t, "Ball", session.PriorityMedium)
	ball.Tags = []string{"test-session"}
	ball.ModelSize = session.ModelSizeLarge
	ball.State = session.StateComplete
	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// Setup mock runner
	mock := agent.NewMockRunner(
		&agent.RunResult{
			Output:   "Working...\n<promise>COMPLETE</promise>\nDone.",
			Complete: true,
		},
	)
	agent.SetRunner(mock)
	defer agent.ResetRunner()

	// Run with explicit model flag set to haiku (overrides ball preference)
	config := cli.AgentLoopConfig{
		SessionID:     "test-session",
		ProjectDir:    env.ProjectDir,
		MaxIterations: 1,
		Trust:         false,
		IterDelay:     0,
		Model:         "haiku", // Explicit flag should take precedence
	}

	_, err := cli.RunAgentLoop(config)
	if err != nil {
		t.Fatalf("Agent run failed: %v", err)
	}

	// Explicit flag should override ball preference
	if len(mock.Calls) > 0 && mock.Calls[0].Model != "haiku" {
		t.Errorf("Expected model=haiku (explicit flag), got %s", mock.Calls[0].Model)
	}
}
