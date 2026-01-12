package integration_test

import (
	"strings"
	"testing"
	"time"

	"github.com/ohare93/juggle/internal/agent"
	"github.com/ohare93/juggle/internal/cli"
	"github.com/ohare93/juggle/internal/session"
)

// TestEnv helper to create a session for testing
func (env *TestEnv) CreateSession(t *testing.T, id, description string) *session.JuggleSession {
	t.Helper()

	sessionStore, err := session.NewSessionStoreWithConfig(env.ProjectDir, session.StoreConfig{
		JugglerDirName: ".juggler",
	})
	if err != nil {
		t.Fatalf("Failed to create session store: %v", err)
	}

	sess, err := sessionStore.CreateSession(id, description)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	return sess
}

// GetSessionStore returns a session store for the test environment
func (env *TestEnv) GetSessionStore(t *testing.T) *session.SessionStore {
	t.Helper()

	sessionStore, err := session.NewSessionStoreWithConfig(env.ProjectDir, session.StoreConfig{
		JugglerDirName: ".juggler",
	})
	if err != nil {
		t.Fatalf("Failed to create session store: %v", err)
	}

	return sessionStore
}

func TestAgentLoop_CompleteSignalExitsLoop(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a session
	env.CreateSession(t, "test-session", "Test session for agent")

	// Create a ball tagged with the session that is ALREADY COMPLETE
	// (COMPLETE signal only exits when all balls are in terminal state)
	ball := env.CreateBall(t, "Test ball", session.PriorityMedium)
	ball.Tags = []string{"test-session"}
	ball.State = session.StateComplete // Ball must be complete for COMPLETE signal to exit
	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// Setup mock runner that returns COMPLETE on first iteration
	mock := agent.NewMockRunner(
		&agent.RunResult{
			Output:   "Working...\n<promise>COMPLETE</promise>\nDone.",
			Complete: true,
		},
	)
	agent.SetRunner(mock)
	defer agent.ResetRunner()

	// Run the agent loop
	config := cli.AgentLoopConfig{
		SessionID:     "test-session",
		ProjectDir:    env.ProjectDir,
		MaxIterations: 5,
		Trust:         false,
		IterDelay:     0, // No delay for tests
	}

	result, err := cli.RunAgentLoop(config)
	if err != nil {
		t.Fatalf("Agent run failed: %v", err)
	}

	// Verify runner was only called once (loop exited on COMPLETE because ball is complete)
	if len(mock.Calls) != 1 {
		t.Errorf("Expected 1 call to runner (exited on COMPLETE), got %d", len(mock.Calls))
	}

	// Verify result shows complete
	if !result.Complete {
		t.Error("Expected result.Complete=true")
	}
	if result.Blocked {
		t.Error("Expected result.Blocked=false")
	}
}

func TestAgentLoop_BlockedSignalExitsWithReason(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a session
	env.CreateSession(t, "test-session", "Test session for agent")

	// Create a ball tagged with the session
	ball := env.CreateBall(t, "Test ball", session.PriorityMedium)
	ball.Tags = []string{"test-session"}
	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// Setup mock runner that returns BLOCKED
	mock := agent.NewMockRunner(
		&agent.RunResult{
			Output:        "Working...\n<promise>BLOCKED: tools not available</promise>\nStopped.",
			Blocked:       true,
			BlockedReason: "tools not available",
		},
	)
	agent.SetRunner(mock)
	defer agent.ResetRunner()

	// Run the agent loop
	config := cli.AgentLoopConfig{
		SessionID:     "test-session",
		ProjectDir:    env.ProjectDir,
		MaxIterations: 5,
		Trust:         false,
		IterDelay:     0,
	}

	result, err := cli.RunAgentLoop(config)
	if err != nil {
		t.Fatalf("Agent run failed: %v", err)
	}

	// Verify runner was only called once (loop exited on BLOCKED)
	if len(mock.Calls) != 1 {
		t.Errorf("Expected 1 call to runner (exited on BLOCKED), got %d", len(mock.Calls))
	}

	// Verify result shows blocked with reason
	if result.Complete {
		t.Error("Expected result.Complete=false")
	}
	if !result.Blocked {
		t.Error("Expected result.Blocked=true")
	}
	if result.BlockedReason != "tools not available" {
		t.Errorf("Expected BlockedReason 'tools not available', got '%s'", result.BlockedReason)
	}
}

func TestAgentLoop_MaxIterationsReached(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a session
	env.CreateSession(t, "test-session", "Test session for agent")

	// Create a ball tagged with the session (in_progress so it's not "complete")
	ball := env.CreateInProgressBall(t, "Test ball", session.PriorityMedium)
	ball.Tags = []string{"test-session"}
	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// Setup mock runner that never returns completion signals
	mock := agent.NewMockRunner(
		&agent.RunResult{Output: "Iteration 1"},
		&agent.RunResult{Output: "Iteration 2"},
		&agent.RunResult{Output: "Iteration 3"},
	)
	agent.SetRunner(mock)
	defer agent.ResetRunner()

	// Run with 3 max iterations
	config := cli.AgentLoopConfig{
		SessionID:     "test-session",
		ProjectDir:    env.ProjectDir,
		MaxIterations: 3,
		Trust:         false,
		IterDelay:     0,
	}

	result, err := cli.RunAgentLoop(config)
	if err != nil {
		t.Fatalf("Agent run failed: %v", err)
	}

	// Verify runner was called exactly 3 times
	if len(mock.Calls) != 3 {
		t.Errorf("Expected 3 calls to runner (max iterations), got %d", len(mock.Calls))
	}

	// Verify result shows neither complete nor blocked
	if result.Complete {
		t.Error("Expected result.Complete=false (max iterations, not complete)")
	}
	if result.Blocked {
		t.Error("Expected result.Blocked=false")
	}
	if result.Iterations != 3 {
		t.Errorf("Expected 3 iterations, got %d", result.Iterations)
	}
}

func TestAgentLoop_PromptContainsRequiredSections(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a session with context
	sessionStore := env.GetSessionStore(t)
	sess, err := sessionStore.CreateSession("test-session", "Test session description")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Add context to session
	if err := sessionStore.UpdateSessionContext(sess.ID, "This is session context for testing"); err != nil {
		t.Fatalf("Failed to update context: %v", err)
	}

	// Add progress
	if err := sessionStore.AppendProgress(sess.ID, "[2024-01-01] Progress entry\n"); err != nil {
		t.Fatalf("Failed to append progress: %v", err)
	}

	// Create a ball with acceptance criteria tagged with the session
	ball := env.CreateBall(t, "Implement feature X", session.PriorityHigh)
	ball.Tags = []string{"test-session"}
	ball.AcceptanceCriteria = []string{"First criterion", "Second criterion"}
	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// Setup mock runner to capture the prompt
	mock := agent.NewMockRunner(
		&agent.RunResult{
			Output:   "<promise>COMPLETE</promise>",
			Complete: true,
		},
	)
	agent.SetRunner(mock)
	defer agent.ResetRunner()

	// Run the agent loop
	config := cli.AgentLoopConfig{
		SessionID:     "test-session",
		ProjectDir:    env.ProjectDir,
		MaxIterations: 1,
		Trust:         false,
		IterDelay:     0,
	}

	_, err = cli.RunAgentLoop(config)
	if err != nil {
		t.Fatalf("Agent run failed: %v", err)
	}

	// Verify prompt was passed
	if len(mock.Calls) == 0 {
		t.Fatal("No calls made to runner")
	}

	prompt := mock.Calls[0].Prompt

	// Check for required sections
	requiredSections := []string{
		"<context>",
		"</context>",
		"<progress>",
		"</progress>",
		"<balls>",
		"</balls>",
		"<instructions>",
		"</instructions>",
	}

	for _, section := range requiredSections {
		if !strings.Contains(prompt, section) {
			t.Errorf("Prompt missing required section: %s", section)
		}
	}

	// Check that session context is included
	if !strings.Contains(prompt, "This is session context for testing") {
		t.Error("Prompt missing session context")
	}

	// Check that progress is included
	if !strings.Contains(prompt, "Progress entry") {
		t.Error("Prompt missing progress content")
	}

	// Check that ball is included
	if !strings.Contains(prompt, "Implement feature X") {
		t.Error("Prompt missing ball intent")
	}

	// Check that acceptance criteria are included
	if !strings.Contains(prompt, "First criterion") {
		t.Error("Prompt missing first acceptance criterion")
	}
	if !strings.Contains(prompt, "Second criterion") {
		t.Error("Prompt missing second acceptance criterion")
	}
}

func TestAgentLoop_EmptySessionHandling(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a session without any balls
	env.CreateSession(t, "empty-session", "Session with no balls")

	// Setup mock runner - should still be called
	mock := agent.NewMockRunner(
		&agent.RunResult{
			Output:   "<promise>COMPLETE</promise>",
			Complete: true,
		},
	)
	agent.SetRunner(mock)
	defer agent.ResetRunner()

	// Run the agent loop
	config := cli.AgentLoopConfig{
		SessionID:     "empty-session",
		ProjectDir:    env.ProjectDir,
		MaxIterations: 1,
		Trust:         false,
		IterDelay:     0,
	}

	_, err := cli.RunAgentLoop(config)
	if err != nil {
		t.Fatalf("Agent run failed: %v", err)
	}

	// Verify the prompt was still generated (empty balls list is valid)
	if len(mock.Calls) == 0 {
		t.Fatal("Expected at least one call to runner")
	}

	// Verify the prompt contains empty balls section
	prompt := mock.Calls[0].Prompt
	if !strings.Contains(prompt, "<balls>") {
		t.Error("Prompt missing balls section even for empty session")
	}
}

func TestAgentLoop_AllBallsCompleteExitsLoop(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a session
	env.CreateSession(t, "test-session", "Test session")

	// Create a ball that's already complete
	ball := env.CreateBall(t, "Already done", session.PriorityMedium)
	ball.Tags = []string{"test-session"}
	ball.State = session.StateComplete
	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// Setup mock runner - first call should check and see all complete
	mock := agent.NewMockRunner(
		&agent.RunResult{Output: "Checking..."}, // No COMPLETE signal
	)
	agent.SetRunner(mock)
	defer agent.ResetRunner()

	// Run the agent loop
	config := cli.AgentLoopConfig{
		SessionID:     "test-session",
		ProjectDir:    env.ProjectDir,
		MaxIterations: 5,
		Trust:         false,
		IterDelay:     0,
	}

	result, err := cli.RunAgentLoop(config)
	if err != nil {
		t.Fatalf("Agent run failed: %v", err)
	}

	// Should have exited after first iteration since all balls complete
	if len(mock.Calls) != 1 {
		t.Errorf("Expected 1 call (exited because all balls complete), got %d", len(mock.Calls))
	}

	// Should be marked complete
	if !result.Complete {
		t.Error("Expected result.Complete=true when all balls are complete")
	}
}

func TestAgentLoop_TrustFlagPassedToRunner(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a session
	env.CreateSession(t, "test-session", "Test session")

	// Create a ball
	ball := env.CreateBall(t, "Test ball", session.PriorityMedium)
	ball.Tags = []string{"test-session"}
	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// Setup mock runner
	mock := agent.NewMockRunner(
		&agent.RunResult{
			Output:   "<promise>COMPLETE</promise>",
			Complete: true,
		},
	)
	agent.SetRunner(mock)
	defer agent.ResetRunner()

	// Run with Trust=true
	config := cli.AgentLoopConfig{
		SessionID:     "test-session",
		ProjectDir:    env.ProjectDir,
		MaxIterations: 1,
		Trust:         true,
		IterDelay:     0,
	}

	_, err := cli.RunAgentLoop(config)
	if err != nil {
		t.Fatalf("Agent run failed: %v", err)
	}

	// Verify trust was passed to runner
	if len(mock.Calls) == 0 {
		t.Fatal("No calls made to runner")
	}

	if !mock.Calls[0].Trust {
		t.Error("Expected Trust=true to be passed to runner")
	}
}

func TestAgentLoop_SessionNotFound(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Don't create any session

	// Run the agent loop for non-existent session
	config := cli.AgentLoopConfig{
		SessionID:     "non-existent-session",
		ProjectDir:    env.ProjectDir,
		MaxIterations: 1,
		Trust:         false,
		IterDelay:     0,
	}

	_, err := cli.RunAgentLoop(config)
	if err == nil {
		t.Fatal("Expected error for non-existent session")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}
}

func TestAgentLoop_PrematureCOMPLETE_Continues(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a session
	env.CreateSession(t, "test-session", "Test session for agent")

	// Create TWO balls tagged with the session - one pending, one in_progress
	ball1 := env.CreateBall(t, "Pending ball", session.PriorityMedium)
	ball1.Tags = []string{"test-session"}
	ball1.State = session.StatePending
	store := env.GetStore(t)
	if err := store.UpdateBall(ball1); err != nil {
		t.Fatalf("Failed to update ball1: %v", err)
	}

	ball2 := env.CreateInProgressBall(t, "In progress ball", session.PriorityMedium)
	ball2.Tags = []string{"test-session"}
	if err := store.UpdateBall(ball2); err != nil {
		t.Fatalf("Failed to update ball2: %v", err)
	}

	// Setup mock runner that returns COMPLETE prematurely, then no signal
	// The first COMPLETE should be ignored because balls are still pending/in_progress
	mock := agent.NewMockRunner(
		&agent.RunResult{
			Output:   "Done!\n<promise>COMPLETE</promise>",
			Complete: true,
		},
		&agent.RunResult{
			Output: "Continuing work...",
		},
		&agent.RunResult{
			Output: "More work...",
		},
	)
	agent.SetRunner(mock)
	defer agent.ResetRunner()

	// Run the agent loop
	config := cli.AgentLoopConfig{
		SessionID:     "test-session",
		ProjectDir:    env.ProjectDir,
		MaxIterations: 3,
		Trust:         false,
		IterDelay:     0,
	}

	result, err := cli.RunAgentLoop(config)
	if err != nil {
		t.Fatalf("Agent run failed: %v", err)
	}

	// Verify runner was called 3 times (premature COMPLETE was ignored, hit max iterations)
	if len(mock.Calls) != 3 {
		t.Errorf("Expected 3 calls to runner (premature COMPLETE ignored), got %d", len(mock.Calls))
	}

	// Should NOT be marked complete since balls are still pending/in_progress
	if result.Complete {
		t.Error("Expected result.Complete=false (premature COMPLETE should be ignored)")
	}
}

func TestAgentLoop_AllBlockedOrComplete_ExitsLoop(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a session
	env.CreateSession(t, "test-session", "Test session for agent")

	// Create balls in terminal states: one complete, one blocked
	ball1 := env.CreateBall(t, "Complete ball", session.PriorityMedium)
	ball1.Tags = []string{"test-session"}
	ball1.State = session.StateComplete
	store := env.GetStore(t)
	if err := store.UpdateBall(ball1); err != nil {
		t.Fatalf("Failed to update ball1: %v", err)
	}

	ball2 := env.CreateBall(t, "Blocked ball", session.PriorityMedium)
	ball2.Tags = []string{"test-session"}
	ball2.State = session.StateBlocked
	ball2.BlockedReason = "Waiting on external dependency"
	if err := store.UpdateBall(ball2); err != nil {
		t.Fatalf("Failed to update ball2: %v", err)
	}

	// Setup mock runner that returns COMPLETE
	mock := agent.NewMockRunner(
		&agent.RunResult{
			Output:   "<promise>COMPLETE</promise>",
			Complete: true,
		},
	)
	agent.SetRunner(mock)
	defer agent.ResetRunner()

	// Run the agent loop
	config := cli.AgentLoopConfig{
		SessionID:     "test-session",
		ProjectDir:    env.ProjectDir,
		MaxIterations: 5,
		Trust:         false,
		IterDelay:     0,
	}

	result, err := cli.RunAgentLoop(config)
	if err != nil {
		t.Fatalf("Agent run failed: %v", err)
	}

	// Verify runner was only called once (all balls are in terminal state)
	if len(mock.Calls) != 1 {
		t.Errorf("Expected 1 call to runner (all terminal), got %d", len(mock.Calls))
	}

	// Should be marked complete
	if !result.Complete {
		t.Error("Expected result.Complete=true (all balls in terminal state)")
	}

	// Verify counts
	if result.BallsComplete != 1 {
		t.Errorf("Expected BallsComplete=1, got %d", result.BallsComplete)
	}
	if result.BallsBlocked != 1 {
		t.Errorf("Expected BallsBlocked=1, got %d", result.BallsBlocked)
	}
	if result.BallsTotal != 2 {
		t.Errorf("Expected BallsTotal=2, got %d", result.BallsTotal)
	}
}

func TestAgentLoop_TerminalStateExitsWithoutCOMPLETESignal(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a session
	env.CreateSession(t, "test-session", "Test session for agent")

	// Create a ball that's already blocked (terminal state)
	ball := env.CreateBall(t, "Blocked ball", session.PriorityMedium)
	ball.Tags = []string{"test-session"}
	ball.State = session.StateBlocked
	ball.BlockedReason = "External blocker"
	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// Setup mock runner that does NOT return COMPLETE signal
	mock := agent.NewMockRunner(
		&agent.RunResult{
			Output: "Checked the session, all blocked.",
		},
	)
	agent.SetRunner(mock)
	defer agent.ResetRunner()

	// Run the agent loop
	config := cli.AgentLoopConfig{
		SessionID:     "test-session",
		ProjectDir:    env.ProjectDir,
		MaxIterations: 5,
		Trust:         false,
		IterDelay:     0,
	}

	result, err := cli.RunAgentLoop(config)
	if err != nil {
		t.Fatalf("Agent run failed: %v", err)
	}

	// Should exit after first iteration because all balls are in terminal state
	// even without COMPLETE signal from agent
	if len(mock.Calls) != 1 {
		t.Errorf("Expected 1 call to runner (all terminal, no signal needed), got %d", len(mock.Calls))
	}

	// Should be marked complete
	if !result.Complete {
		t.Error("Expected result.Complete=true (all balls in terminal state)")
	}

	// All balls are blocked (not complete)
	if result.BallsBlocked != 1 {
		t.Errorf("Expected BallsBlocked=1, got %d", result.BallsBlocked)
	}
}

func TestAgentLoop_ContinueSignalContinuesLoop(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a session
	env.CreateSession(t, "test-session", "Test session for agent")

	// Create two pending balls tagged with the session
	ball1 := env.CreateBall(t, "First ball", session.PriorityMedium)
	ball1.Tags = []string{"test-session"}
	ball1.State = session.StatePending
	store := env.GetStore(t)
	if err := store.UpdateBall(ball1); err != nil {
		t.Fatalf("Failed to update ball1: %v", err)
	}

	ball2 := env.CreateBall(t, "Second ball", session.PriorityMedium)
	ball2.Tags = []string{"test-session"}
	ball2.State = session.StatePending
	if err := store.UpdateBall(ball2); err != nil {
		t.Fatalf("Failed to update ball2: %v", err)
	}

	// Setup mock runner that returns CONTINUE twice
	// CONTINUE causes the loop to skip terminal-state check and proceed to next iteration
	mock := agent.NewMockRunner(
		&agent.RunResult{
			Output:   "Completed first ball.\n<promise>CONTINUE</promise>",
			Continue: true,
		},
		&agent.RunResult{
			Output:   "Completed second ball.\n<promise>CONTINUE</promise>",
			Continue: true,
		},
	)
	agent.SetRunner(mock)
	defer agent.ResetRunner()

	// Run the agent loop with MaxIterations=2 so we hit max iterations
	// after the two CONTINUE signals
	config := cli.AgentLoopConfig{
		SessionID:     "test-session",
		ProjectDir:    env.ProjectDir,
		MaxIterations: 2,
		Trust:         false,
		IterDelay:     0,
	}

	result, err := cli.RunAgentLoop(config)
	if err != nil {
		t.Fatalf("Agent run failed: %v", err)
	}

	// Verify runner was called 2 times (CONTINUE, CONTINUE then max iterations)
	if len(mock.Calls) != 2 {
		t.Errorf("Expected 2 calls to runner (2 CONTINUE signals), got %d", len(mock.Calls))
	}

	// Result should show 2 iterations
	if result.Iterations != 2 {
		t.Errorf("Expected 2 iterations, got %d", result.Iterations)
	}

	// Should not be blocked (max iterations reached, not blocked)
	if result.Blocked {
		t.Error("Expected result.Blocked=false")
	}

	// Should not be complete (balls are still pending)
	if result.Complete {
		t.Error("Expected result.Complete=false (balls still pending)")
	}
}

func TestAgentLoop_TimeoutExitsLoop(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a session
	env.CreateSession(t, "test-session", "Test session for agent")

	// Create a ball tagged with the session
	ball := env.CreateBall(t, "Test ball", session.PriorityMedium)
	ball.Tags = []string{"test-session"}
	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// Setup mock runner that returns TIMED_OUT
	mock := agent.NewMockRunner(
		&agent.RunResult{
			Output:   "Working...",
			TimedOut: true,
		},
	)
	agent.SetRunner(mock)
	defer agent.ResetRunner()

	// Run the agent loop with timeout
	config := cli.AgentLoopConfig{
		SessionID:     "test-session",
		ProjectDir:    env.ProjectDir,
		MaxIterations: 5,
		Trust:         false,
		IterDelay:     0,
		Timeout:       5 * time.Minute,
	}

	result, err := cli.RunAgentLoop(config)
	if err != nil {
		t.Fatalf("Agent run failed: %v", err)
	}

	// Verify runner was only called once (loop exited on timeout)
	if len(mock.Calls) != 1 {
		t.Errorf("Expected 1 call to runner (exited on timeout), got %d", len(mock.Calls))
	}

	// Verify timeout was passed to runner
	if mock.Calls[0].Timeout != 5*time.Minute {
		t.Errorf("Expected timeout 5m passed to runner, got %v", mock.Calls[0].Timeout)
	}

	// Verify result shows timed out
	if !result.TimedOut {
		t.Error("Expected result.TimedOut=true")
	}
	if result.Complete {
		t.Error("Expected result.Complete=false")
	}
	if result.Blocked {
		t.Error("Expected result.Blocked=false")
	}
	if result.TimeoutMessage == "" {
		t.Error("Expected TimeoutMessage to be set")
	}
}

func TestAgentLoop_TimeoutLogsToProgress(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a session
	env.CreateSession(t, "test-session", "Test session for agent")

	// Create a ball tagged with the session
	ball := env.CreateBall(t, "Test ball", session.PriorityMedium)
	ball.Tags = []string{"test-session"}
	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// Setup mock runner that returns TIMED_OUT
	mock := agent.NewMockRunner(
		&agent.RunResult{
			Output:   "Working...",
			TimedOut: true,
		},
	)
	agent.SetRunner(mock)
	defer agent.ResetRunner()

	// Run the agent loop with timeout
	config := cli.AgentLoopConfig{
		SessionID:     "test-session",
		ProjectDir:    env.ProjectDir,
		MaxIterations: 5,
		Trust:         false,
		IterDelay:     0,
		Timeout:       5 * time.Minute,
	}

	_, err := cli.RunAgentLoop(config)
	if err != nil {
		t.Fatalf("Agent run failed: %v", err)
	}

	// Verify timeout was logged to progress
	sessionStore := env.GetSessionStore(t)
	progress, err := sessionStore.LoadProgress("test-session")
	if err != nil {
		t.Fatalf("Failed to load progress: %v", err)
	}

	if !strings.Contains(progress, "[TIMEOUT]") {
		t.Error("Expected [TIMEOUT] entry in progress log")
	}
	if !strings.Contains(progress, "Iteration 1 timed out") {
		t.Error("Expected timeout message to contain iteration info")
	}
}
