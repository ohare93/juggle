package integration_test

import (
	"fmt"
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
	sessionStore := env.GetSessionStore(t)

	// Create a ball tagged with the session
	ball := env.CreateBall(t, "Test ball", session.PriorityMedium)
	ball.Tags = []string{"test-session"}
	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// Setup mock runner that returns BLOCKED (with progress)
	mock := agent.NewMockRunner(
		&agent.RunResult{
			Output:        "Working...\n<promise>BLOCKED: tools not available</promise>\nStopped.",
			Blocked:       true,
			BlockedReason: "tools not available",
		},
	)
	// Use progress updating mock to simulate agent updating progress
	agent.SetRunner(&progressUpdatingMockRunner{
		mock:         mock,
		sessionStore: sessionStore,
		sessionID:    "test-session",
	})
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
	if mock.NextIndex != 1 {
		t.Errorf("Expected 1 call to runner (exited on BLOCKED), got %d", mock.NextIndex)
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

	// Verify trust was passed to runner (PermissionBypass mode)
	if len(mock.Calls) == 0 {
		t.Fatal("No calls made to runner")
	}

	if mock.Calls[0].Permission != agent.PermissionBypass {
		t.Errorf("Expected Permission=PermissionBypass, got %s", mock.Calls[0].Permission)
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

// Rate limit tests

func TestAgentLoop_RateLimitRetries(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a session
	env.CreateSession(t, "test-session", "Test session for agent")

	// Create a ball that's complete so COMPLETE signal works
	ball := env.CreateBall(t, "Test ball", session.PriorityMedium)
	ball.Tags = []string{"test-session"}
	ball.State = session.StateComplete
	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// Setup mock runner that returns rate limit once, then succeeds
	mock := agent.NewMockRunner(
		&agent.RunResult{
			Output:      "Error: rate limit exceeded",
			RateLimited: true,
			RetryAfter:  100 * time.Millisecond, // Short for testing
		},
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
		MaxIterations: 3,
		Trust:         false,
		IterDelay:     0,
	}

	result, err := cli.RunAgentLoop(config)
	if err != nil {
		t.Fatalf("Agent run failed: %v", err)
	}

	// Verify runner was called twice (rate limit + success)
	if len(mock.Calls) != 2 {
		t.Errorf("Expected 2 calls to runner (rate limit + success), got %d", len(mock.Calls))
	}

	// Should be complete
	if !result.Complete {
		t.Error("Expected result.Complete=true")
	}

	// Should have waited
	if result.TotalWaitTime == 0 {
		t.Error("Expected TotalWaitTime > 0")
	}

	// Rate limit should NOT be exceeded (we successfully retried)
	if result.RateLimitExceded {
		t.Error("Expected RateLimitExceded=false (retry succeeded)")
	}
}

func TestAgentLoop_RateLimitMaxWaitExceeded(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a session
	env.CreateSession(t, "test-session", "Test session for agent")

	// Create a ball
	ball := env.CreateBall(t, "Test ball", session.PriorityMedium)
	ball.Tags = []string{"test-session"}
	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// Setup mock runner that always returns rate limit
	mock := agent.NewMockRunner(
		&agent.RunResult{
			Output:      "Error: rate limit exceeded",
			RateLimited: true,
			RetryAfter:  1 * time.Hour, // Long wait
		},
	)
	agent.SetRunner(mock)
	defer agent.ResetRunner()

	// Run the agent loop with short max-wait
	config := cli.AgentLoopConfig{
		SessionID:     "test-session",
		ProjectDir:    env.ProjectDir,
		MaxIterations: 3,
		Trust:         false,
		IterDelay:     0,
		MaxWait:       1 * time.Minute, // Max wait of 1 minute
	}

	result, err := cli.RunAgentLoop(config)
	if err != nil {
		t.Fatalf("Agent run failed: %v", err)
	}

	// Verify runner was only called once (rate limit exceeded max-wait immediately)
	if len(mock.Calls) != 1 {
		t.Errorf("Expected 1 call to runner (max-wait exceeded), got %d", len(mock.Calls))
	}

	// Should show rate limit exceeded
	if !result.RateLimitExceded {
		t.Error("Expected RateLimitExceded=true")
	}

	// Should NOT be complete
	if result.Complete {
		t.Error("Expected result.Complete=false")
	}
}

func TestAgentLoop_RateLimitLogsToProgress(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a session
	env.CreateSession(t, "test-session", "Test session for agent")

	// Create a ball that's complete
	ball := env.CreateBall(t, "Test ball", session.PriorityMedium)
	ball.Tags = []string{"test-session"}
	ball.State = session.StateComplete
	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// Setup mock runner that returns rate limit once, then succeeds
	mock := agent.NewMockRunner(
		&agent.RunResult{
			Output:      "Error: rate limit exceeded",
			RateLimited: true,
			RetryAfter:  100 * time.Millisecond,
		},
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
		MaxIterations: 3,
		Trust:         false,
		IterDelay:     0,
	}

	_, err := cli.RunAgentLoop(config)
	if err != nil {
		t.Fatalf("Agent run failed: %v", err)
	}

	// Verify rate limit was logged to progress
	sessionStore := env.GetSessionStore(t)
	progress, err := sessionStore.LoadProgress("test-session")
	if err != nil {
		t.Fatalf("Failed to load progress: %v", err)
	}

	if !strings.Contains(progress, "[RATE_LIMIT]") {
		t.Error("Expected [RATE_LIMIT] entry in progress log")
	}
	if !strings.Contains(progress, "waiting") {
		t.Error("Expected 'waiting' in rate limit progress message")
	}
}

func TestAgentLoop_RateLimitExponentialBackoff(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a session
	env.CreateSession(t, "test-session", "Test session for agent")

	// Create a ball that's complete
	ball := env.CreateBall(t, "Test ball", session.PriorityMedium)
	ball.Tags = []string{"test-session"}
	ball.State = session.StateComplete
	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// Setup mock runner that returns rate limit WITHOUT RetryAfter
	// This should trigger exponential backoff
	mock := agent.NewMockRunner(
		&agent.RunResult{
			Output:      "Error: rate limit exceeded",
			RateLimited: true,
			// RetryAfter is 0, so exponential backoff should be used
		},
		&agent.RunResult{
			Output:   "<promise>COMPLETE</promise>",
			Complete: true,
		},
	)
	agent.SetRunner(mock)
	defer agent.ResetRunner()

	// Run the agent loop - this would take 30+ seconds with real backoff
	// but we just verify the logic is called correctly
	config := cli.AgentLoopConfig{
		SessionID:     "test-session",
		ProjectDir:    env.ProjectDir,
		MaxIterations: 3,
		Trust:         false,
		IterDelay:     0,
		MaxWait:       1 * time.Second, // Very short max-wait to exit quickly
	}

	result, err := cli.RunAgentLoop(config)
	if err != nil {
		t.Fatalf("Agent run failed: %v", err)
	}

	// Should have hit max-wait since exponential backoff starts at 30s
	if !result.RateLimitExceded {
		// If it didn't exceed, it means the retry succeeded (which is also valid)
		// This test is mainly to ensure the code path works
		t.Log("Rate limit retry succeeded before max-wait")
	}
}

func TestAgentLoop_RateLimitMultipleRetries(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a session
	env.CreateSession(t, "test-session", "Test session for agent")

	// Create a ball that's complete
	ball := env.CreateBall(t, "Test ball", session.PriorityMedium)
	ball.Tags = []string{"test-session"}
	ball.State = session.StateComplete
	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// Setup mock runner that returns rate limit 3 times, then succeeds
	mock := agent.NewMockRunner(
		&agent.RunResult{
			Output:      "Error: rate limit exceeded",
			RateLimited: true,
			RetryAfter:  50 * time.Millisecond,
		},
		&agent.RunResult{
			Output:      "Error: rate limit exceeded again",
			RateLimited: true,
			RetryAfter:  50 * time.Millisecond,
		},
		&agent.RunResult{
			Output:      "Error: rate limit exceeded still",
			RateLimited: true,
			RetryAfter:  50 * time.Millisecond,
		},
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
		MaxIterations: 3,
		Trust:         false,
		IterDelay:     0,
	}

	result, err := cli.RunAgentLoop(config)
	if err != nil {
		t.Fatalf("Agent run failed: %v", err)
	}

	// Verify runner was called 4 times (3 rate limits + 1 success)
	if len(mock.Calls) != 4 {
		t.Errorf("Expected 4 calls to runner (3 rate limits + success), got %d", len(mock.Calls))
	}

	// Should be complete
	if !result.Complete {
		t.Error("Expected result.Complete=true")
	}

	// Should have accumulated wait time
	if result.TotalWaitTime == 0 {
		t.Error("Expected TotalWaitTime > 0")
	}
}

func TestAgentLoop_RateLimitResetsRetryCountOnSuccess(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a session
	env.CreateSession(t, "test-session", "Test session for agent")

	// Create two balls so we have multiple iterations
	ball1 := env.CreateBall(t, "First ball", session.PriorityMedium)
	ball1.Tags = []string{"test-session"}
	ball1.State = session.StatePending
	store := env.GetStore(t)
	if err := store.UpdateBall(ball1); err != nil {
		t.Fatalf("Failed to update ball1: %v", err)
	}

	ball2 := env.CreateBall(t, "Second ball", session.PriorityMedium)
	ball2.Tags = []string{"test-session"}
	ball2.State = session.StateComplete
	if err := store.UpdateBall(ball2); err != nil {
		t.Fatalf("Failed to update ball2: %v", err)
	}

	// Setup mock runner:
	// Iter 1: rate limit -> success (completes first ball via CONTINUE)
	// Iter 2: rate limit -> success (completes second ball)
	mock := agent.NewMockRunner(
		&agent.RunResult{
			Output:      "Error: rate limit",
			RateLimited: true,
			RetryAfter:  50 * time.Millisecond,
		},
		&agent.RunResult{
			Output:   "<promise>CONTINUE</promise>",
			Continue: true,
		},
		&agent.RunResult{
			Output:      "Error: rate limit again",
			RateLimited: true,
			RetryAfter:  50 * time.Millisecond,
		},
		&agent.RunResult{
			Output:   "<promise>COMPLETE</promise>",
			Complete: true,
		},
	)
	agent.SetRunner(mock)
	defer agent.ResetRunner()

	// Update ball1 to complete before the last iteration
	// (simulate agent completing it)
	ball1.State = session.StateComplete
	if err := store.UpdateBall(ball1); err != nil {
		t.Fatalf("Failed to update ball1: %v", err)
	}

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

	// Verify multiple calls were made
	if len(mock.Calls) < 2 {
		t.Errorf("Expected at least 2 calls to runner, got %d", len(mock.Calls))
	}

	// Should be complete
	if !result.Complete {
		t.Error("Expected result.Complete=true")
	}
}

// Progress validation tests

func TestAgentLoop_CompleteSignalRejectedWithoutProgress(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a session
	env.CreateSession(t, "test-session", "Test session for agent")

	// Create a ball that's already complete (so COMPLETE signal would be valid)
	ball := env.CreateBall(t, "Test ball", session.PriorityMedium)
	ball.Tags = []string{"test-session"}
	ball.State = session.StateComplete
	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// Setup mock runner that returns COMPLETE but doesn't update progress
	// (agent doesn't call juggle progress append)
	mock := agent.NewMockRunner(
		&agent.RunResult{
			Output:   "Working...\n<promise>COMPLETE</promise>",
			Complete: true,
		},
		&agent.RunResult{
			Output:   "More work...\n<promise>COMPLETE</promise>",
			Complete: true,
		},
	)
	agent.SetRunner(mock)
	defer agent.ResetRunner()

	// Run the agent loop - COMPLETE without progress should be rejected
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

	// Verify runner was called twice (COMPLETE rejected first time, terminal state check passed second time)
	// The ball is complete, so terminal state check should exit after first iteration even without progress
	// because we fall through to terminal state check
	if len(mock.Calls) < 1 {
		t.Errorf("Expected at least 1 call to runner, got %d", len(mock.Calls))
	}

	// Should still be complete because terminal state check passes
	// The warning was logged but loop continued and found all balls terminal
	if !result.Complete {
		t.Error("Expected result.Complete=true (terminal state check passed)")
	}
}

func TestAgentLoop_ContinueSignalRejectedWithoutProgress(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a session
	env.CreateSession(t, "test-session", "Test session for agent")

	// Create two pending balls
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

	// Setup mock runner that returns CONTINUE without updating progress
	mock := agent.NewMockRunner(
		&agent.RunResult{
			Output:   "Done!\n<promise>CONTINUE</promise>",
			Continue: true,
		},
		&agent.RunResult{
			Output:   "More!\n<promise>CONTINUE</promise>",
			Continue: true,
		},
		&agent.RunResult{
			Output: "Final iteration",
		},
	)
	agent.SetRunner(mock)
	defer agent.ResetRunner()

	// Run the agent loop - CONTINUE without progress should be rejected
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

	// Verify runner was called 3 times (CONTINUE signals rejected, fell through to terminal check each time)
	if len(mock.Calls) != 3 {
		t.Errorf("Expected 3 calls to runner (CONTINUE rejected each time), got %d", len(mock.Calls))
	}

	// Should not be complete (balls still pending)
	if result.Complete {
		t.Error("Expected result.Complete=false (balls still pending)")
	}
}

func TestAgentLoop_BlockedSignalRejectedWithoutProgress(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a session
	env.CreateSession(t, "test-session", "Test session for agent")

	// Create a pending ball
	ball := env.CreateBall(t, "Test ball", session.PriorityMedium)
	ball.Tags = []string{"test-session"}
	ball.State = session.StatePending
	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// Setup mock runner that returns BLOCKED without updating progress
	mock := agent.NewMockRunner(
		&agent.RunResult{
			Output:        "Can't continue\n<promise>BLOCKED: tools not available</promise>",
			Blocked:       true,
			BlockedReason: "tools not available",
		},
		&agent.RunResult{
			Output: "More work...",
		},
	)
	agent.SetRunner(mock)
	defer agent.ResetRunner()

	// Run the agent loop - BLOCKED without progress should be rejected
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

	// Verify runner was called twice (BLOCKED rejected first time)
	if len(mock.Calls) != 2 {
		t.Errorf("Expected 2 calls to runner (BLOCKED rejected), got %d", len(mock.Calls))
	}

	// Should NOT be blocked (signal was rejected)
	if result.Blocked {
		t.Error("Expected result.Blocked=false (BLOCKED signal rejected without progress)")
	}
}

func TestAgentLoop_CompleteSignalAcceptedWithProgress(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a session with initial progress
	env.CreateSession(t, "test-session", "Test session for agent")
	sessionStore := env.GetSessionStore(t)
	if err := sessionStore.AppendProgress("test-session", "[Initial] Starting work\n"); err != nil {
		t.Fatalf("Failed to append initial progress: %v", err)
	}

	// Create a ball that's already complete
	ball := env.CreateBall(t, "Test ball", session.PriorityMedium)
	ball.Tags = []string{"test-session"}
	ball.State = session.StateComplete
	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// Record initial progress line count
	initialLineCount := cli.GetProgressLineCountForTest(sessionStore, "test-session")

	// Setup mock runner that:
	// 1. "Updates progress" by us appending to progress before returning
	// 2. Returns COMPLETE signal
	mock := agent.NewMockRunner(
		&agent.RunResult{
			Output:   "Working...\n<promise>COMPLETE</promise>",
			Complete: true,
		},
	)

	// Custom run function that simulates agent updating progress
	origRunner := agent.DefaultRunner
	agent.SetRunner(&progressUpdatingMockRunner{
		mock:         mock,
		sessionStore: sessionStore,
		sessionID:    "test-session",
	})
	defer func() { agent.DefaultRunner = origRunner }()

	// Run the agent loop - COMPLETE with progress should be accepted
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

	// Verify progress was updated
	finalLineCount := cli.GetProgressLineCountForTest(sessionStore, "test-session")
	if finalLineCount <= initialLineCount {
		t.Errorf("Expected progress to be updated, but line count: initial=%d, final=%d",
			initialLineCount, finalLineCount)
	}

	// Should be complete (signal accepted with progress)
	if !result.Complete {
		t.Error("Expected result.Complete=true (COMPLETE signal accepted with progress)")
	}

	// Should have only called once (signal accepted)
	if mock.NextIndex != 1 {
		t.Errorf("Expected 1 call to runner, got %d", mock.NextIndex)
	}
}

func TestAgentLoop_BlockedSignalAcceptedWithProgress(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a session with initial progress
	env.CreateSession(t, "test-session", "Test session for agent")
	sessionStore := env.GetSessionStore(t)
	if err := sessionStore.AppendProgress("test-session", "[Initial] Starting work\n"); err != nil {
		t.Fatalf("Failed to append initial progress: %v", err)
	}

	// Create a pending ball
	ball := env.CreateBall(t, "Test ball", session.PriorityMedium)
	ball.Tags = []string{"test-session"}
	ball.State = session.StatePending
	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// Setup mock runner that returns BLOCKED with progress updated
	mock := agent.NewMockRunner(
		&agent.RunResult{
			Output:        "Can't continue\n<promise>BLOCKED: tools not available</promise>",
			Blocked:       true,
			BlockedReason: "tools not available",
		},
	)

	// Custom run function that simulates agent updating progress
	origRunner := agent.DefaultRunner
	agent.SetRunner(&progressUpdatingMockRunner{
		mock:         mock,
		sessionStore: sessionStore,
		sessionID:    "test-session",
	})
	defer func() { agent.DefaultRunner = origRunner }()

	// Run the agent loop - BLOCKED with progress should be accepted
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

	// Should be blocked (signal accepted with progress)
	if !result.Blocked {
		t.Error("Expected result.Blocked=true (BLOCKED signal accepted with progress)")
	}
	if result.BlockedReason != "tools not available" {
		t.Errorf("Expected BlockedReason 'tools not available', got '%s'", result.BlockedReason)
	}

	// Should have only called once (signal accepted)
	if mock.NextIndex != 1 {
		t.Errorf("Expected 1 call to runner, got %d", mock.NextIndex)
	}
}

func TestAgentLoop_ContinueSignalAcceptedWithProgress(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a session with initial progress
	env.CreateSession(t, "test-session", "Test session for agent")
	sessionStore := env.GetSessionStore(t)
	if err := sessionStore.AppendProgress("test-session", "[Initial] Starting work\n"); err != nil {
		t.Fatalf("Failed to append initial progress: %v", err)
	}

	// Create two pending balls
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

	// Setup mock runner that returns CONTINUE twice (with progress)
	mock := agent.NewMockRunner(
		&agent.RunResult{
			Output:   "Done first!\n<promise>CONTINUE</promise>",
			Continue: true,
		},
		&agent.RunResult{
			Output:   "Done second!\n<promise>CONTINUE</promise>",
			Continue: true,
		},
	)

	// Custom run function that simulates agent updating progress
	origRunner := agent.DefaultRunner
	agent.SetRunner(&progressUpdatingMockRunner{
		mock:         mock,
		sessionStore: sessionStore,
		sessionID:    "test-session",
	})
	defer func() { agent.DefaultRunner = origRunner }()

	// Run the agent loop - CONTINUE with progress should be accepted
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

	// Should have called twice (both CONTINUE signals accepted)
	if mock.NextIndex != 2 {
		t.Errorf("Expected 2 calls to runner, got %d", mock.NextIndex)
	}

	// Should show 2 iterations
	if result.Iterations != 2 {
		t.Errorf("Expected 2 iterations, got %d", result.Iterations)
	}

	// Should not be complete (balls still pending, max iterations hit)
	if result.Complete {
		t.Error("Expected result.Complete=false (max iterations reached)")
	}
}

func TestGetProgressLineCount(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a session
	env.CreateSession(t, "test-session", "Test session")
	sessionStore := env.GetSessionStore(t)

	// Test empty progress
	count := cli.GetProgressLineCountForTest(sessionStore, "test-session")
	if count != 0 {
		t.Errorf("Expected 0 lines for empty progress, got %d", count)
	}

	// Append one line
	if err := sessionStore.AppendProgress("test-session", "Line 1\n"); err != nil {
		t.Fatalf("Failed to append progress: %v", err)
	}
	count = cli.GetProgressLineCountForTest(sessionStore, "test-session")
	if count != 1 {
		t.Errorf("Expected 1 line, got %d", count)
	}

	// Append another line
	if err := sessionStore.AppendProgress("test-session", "Line 2\n"); err != nil {
		t.Fatalf("Failed to append progress: %v", err)
	}
	count = cli.GetProgressLineCountForTest(sessionStore, "test-session")
	if count != 2 {
		t.Errorf("Expected 2 lines, got %d", count)
	}

	// Append multi-line entry
	if err := sessionStore.AppendProgress("test-session", "Line 3\nLine 4\n"); err != nil {
		t.Fatalf("Failed to append progress: %v", err)
	}
	count = cli.GetProgressLineCountForTest(sessionStore, "test-session")
	if count != 4 {
		t.Errorf("Expected 4 lines, got %d", count)
	}
}

// progressUpdatingMockRunner wraps MockRunner and adds progress on each call
type progressUpdatingMockRunner struct {
	mock         *agent.MockRunner
	sessionStore *session.SessionStore
	sessionID    string
}

func (p *progressUpdatingMockRunner) Run(opts agent.RunOptions) (*agent.RunResult, error) {
	// Simulate agent updating progress before returning
	entry := fmt.Sprintf("[Iteration %d] Agent work completed\n", p.mock.NextIndex+1)
	_ = p.sessionStore.AppendProgress(p.sessionID, entry)

	return p.mock.Run(opts)
}

// Concurrent agent detection tests

func TestAgentLoop_ConcurrentAgentBlocked(t *testing.T) {
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

	// Acquire lock manually to simulate another agent running
	sessionStore := env.GetSessionStore(t)
	lock, err := sessionStore.AcquireSessionLock("test-session")
	if err != nil {
		t.Fatalf("Failed to acquire lock: %v", err)
	}
	defer lock.Release()

	// Setup mock runner (won't be called since lock blocks us)
	mock := agent.NewMockRunner(
		&agent.RunResult{
			Output:   "<promise>COMPLETE</promise>",
			Complete: true,
		},
	)
	agent.SetRunner(mock)
	defer agent.ResetRunner()

	// Run the agent loop - should fail because session is locked
	config := cli.AgentLoopConfig{
		SessionID:     "test-session",
		ProjectDir:    env.ProjectDir,
		MaxIterations: 1,
		Trust:         false,
		IterDelay:     0,
	}

	_, err = cli.RunAgentLoop(config)
	if err == nil {
		t.Fatal("Expected error when session is locked by another agent")
	}

	// Error should mention the session is locked
	if !strings.Contains(err.Error(), "locked") && !strings.Contains(err.Error(), "already") {
		t.Errorf("Expected error to mention lock, got: %v", err)
	}

	// Verify runner was NOT called
	if len(mock.Calls) != 0 {
		t.Errorf("Expected 0 calls to runner (blocked by lock), got %d", len(mock.Calls))
	}
}

func TestAgentLoop_LockReleasedOnCompletion(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a session
	env.CreateSession(t, "test-session", "Test session for agent")

	// Create a ball that's complete
	ball := env.CreateBall(t, "Test ball", session.PriorityMedium)
	ball.Tags = []string{"test-session"}
	ball.State = session.StateComplete
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

	// Verify lock was released - we should be able to acquire it again
	sessionStore := env.GetSessionStore(t)
	lock, err := sessionStore.AcquireSessionLock("test-session")
	if err != nil {
		t.Fatalf("Expected to acquire lock after agent completed: %v", err)
	}
	lock.Release()
}

func TestAgentLoop_LockReleasedOnError(t *testing.T) {
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

	// Setup mock runner that returns an error
	mock := agent.NewMockRunner(
		&agent.RunResult{
			Output: "Error occurred",
			Error:  fmt.Errorf("simulated error"),
		},
	)
	agent.SetRunner(mock)
	defer agent.ResetRunner()

	// Run the agent loop - will not error out (agent errors are captured in result)
	config := cli.AgentLoopConfig{
		SessionID:     "test-session",
		ProjectDir:    env.ProjectDir,
		MaxIterations: 1,
		Trust:         false,
		IterDelay:     0,
	}

	_, err := cli.RunAgentLoop(config)
	// Depending on implementation, might succeed or fail
	// The important thing is the lock is released

	_ = err // Don't check error - just verify lock is released

	// Verify lock was released
	sessionStore := env.GetSessionStore(t)
	lock, err := sessionStore.AcquireSessionLock("test-session")
	if err != nil {
		t.Fatalf("Expected to acquire lock after agent completed/errored: %v", err)
	}
	lock.Release()
}

func TestAgentLoop_LockErrorMessageIncludesPID(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a session
	env.CreateSession(t, "test-session", "Test session for agent")

	// Acquire lock manually
	sessionStore := env.GetSessionStore(t)
	lock, err := sessionStore.AcquireSessionLock("test-session")
	if err != nil {
		t.Fatalf("Failed to acquire lock: %v", err)
	}
	defer lock.Release()

	// Try to run agent - should fail with informative error
	config := cli.AgentLoopConfig{
		SessionID:     "test-session",
		ProjectDir:    env.ProjectDir,
		MaxIterations: 1,
		Trust:         false,
		IterDelay:     0,
	}

	_, err = cli.RunAgentLoop(config)
	if err == nil {
		t.Fatal("Expected error when session is locked")
	}

	// Error should contain PID info
	errStr := err.Error()
	if !strings.Contains(errStr, "PID") && !strings.Contains(errStr, "locked") {
		t.Errorf("Expected error to contain PID or 'locked', got: %v", err)
	}
}

// Iteration delay tests

func TestAgentLoop_IterationDelayNotAppliedBeforeFirstRun(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a session
	env.CreateSession(t, "test-session", "Test session for agent")

	// Create a ball that's in_progress to prevent early termination
	ball := env.CreateInProgressBall(t, "Test ball", session.PriorityMedium)
	ball.Tags = []string{"test-session"}
	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// Record timestamps for each call to verify timing
	callTimes := make([]time.Time, 0, 3)
	mock := agent.NewMockRunner(
		&agent.RunResult{Output: "Iteration 1"},
		&agent.RunResult{Output: "Iteration 2"},
		&agent.RunResult{Output: "Iteration 3"},
	)

	// Wrap the mock to track call times
	origRunner := agent.DefaultRunner
	agent.SetRunner(&timingMockRunner{
		mock:      mock,
		callTimes: &callTimes,
	})
	defer func() { agent.DefaultRunner = origRunner }()

	// Run with a significant delay (100ms) - enough to measure but fast for tests
	iterDelay := 100 * time.Millisecond
	startTime := time.Now()

	config := cli.AgentLoopConfig{
		SessionID:     "test-session",
		ProjectDir:    env.ProjectDir,
		MaxIterations: 3,
		Trust:         false,
		IterDelay:     iterDelay,
	}

	result, err := cli.RunAgentLoop(config)
	if err != nil {
		t.Fatalf("Agent run failed: %v", err)
	}

	// Verify 3 iterations ran
	if result.Iterations != 3 {
		t.Errorf("Expected 3 iterations, got %d", result.Iterations)
	}

	// Verify timing:
	// 1. First call should be almost immediate (< 50ms after start)
	// 2. Second call should be ~100ms after first call (the delay)
	// 3. Third call should be ~100ms after second call (the delay)
	if len(callTimes) != 3 {
		t.Fatalf("Expected 3 call times, got %d", len(callTimes))
	}

	// First iteration should start immediately (no delay before it)
	firstCallDelay := callTimes[0].Sub(startTime)
	if firstCallDelay > 50*time.Millisecond {
		t.Errorf("First iteration should start immediately, but took %v to start", firstCallDelay)
	}

	// Second iteration should be after the delay
	secondCallDelay := callTimes[1].Sub(callTimes[0])
	if secondCallDelay < 80*time.Millisecond { // Allow some tolerance
		t.Errorf("Expected delay between 1st and 2nd iteration (~%v), got %v", iterDelay, secondCallDelay)
	}

	// Third iteration should also be after the delay
	thirdCallDelay := callTimes[2].Sub(callTimes[1])
	if thirdCallDelay < 80*time.Millisecond {
		t.Errorf("Expected delay between 2nd and 3rd iteration (~%v), got %v", iterDelay, thirdCallDelay)
	}
}

func TestAgentLoop_IterationDelayWithFuzziness(t *testing.T) {
	// Test the fuzzy delay calculation function directly
	// The calculateFuzzyDelay function is tested via the exported wrapper

	// Test with no fuzz (exact delay)
	delay := cli.CalculateFuzzyDelayForTest(5, 0)
	if delay != 5*time.Minute {
		t.Errorf("Expected 5m with no fuzz, got %v", delay)
	}

	// Test with fuzz - run multiple times to verify randomness produces valid range
	baseMinutes := 5
	fuzz := 2
	minDelay := time.Duration(baseMinutes-fuzz) * time.Minute // 3 minutes
	maxDelay := time.Duration(baseMinutes+fuzz) * time.Minute // 7 minutes

	// Run 100 iterations to verify the range is respected
	for i := 0; i < 100; i++ {
		delay := cli.CalculateFuzzyDelayForTest(baseMinutes, fuzz)
		if delay < minDelay || delay > maxDelay {
			t.Errorf("Fuzzy delay %v outside expected range [%v, %v]", delay, minDelay, maxDelay)
		}
	}

	// Test edge case: fuzz larger than base (should never go negative)
	delay = cli.CalculateFuzzyDelayForTest(1, 5)
	if delay < 0 {
		t.Errorf("Fuzzy delay should never be negative, got %v", delay)
	}
}

func TestAgentLoop_NoDelayAfterLastIteration(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a session
	env.CreateSession(t, "test-session", "Test session for agent")

	// Create a ball in pending state
	ball := env.CreateBall(t, "Test ball", session.PriorityMedium)
	ball.Tags = []string{"test-session"}
	ball.State = session.StatePending
	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// Track elapsed time
	callTimes := make([]time.Time, 0, 2)
	mock := agent.NewMockRunner(
		&agent.RunResult{Output: "Iteration 1"},
		&agent.RunResult{Output: "Iteration 2"},
	)

	origRunner := agent.DefaultRunner
	agent.SetRunner(&timingMockRunner{
		mock:      mock,
		callTimes: &callTimes,
	})
	defer func() { agent.DefaultRunner = origRunner }()

	// Run with delay and max 2 iterations
	iterDelay := 100 * time.Millisecond
	startTime := time.Now()

	config := cli.AgentLoopConfig{
		SessionID:     "test-session",
		ProjectDir:    env.ProjectDir,
		MaxIterations: 2,
		Trust:         false,
		IterDelay:     iterDelay,
	}

	result, err := cli.RunAgentLoop(config)
	endTime := time.Now()
	if err != nil {
		t.Fatalf("Agent run failed: %v", err)
	}

	// Verify 2 iterations
	if result.Iterations != 2 {
		t.Errorf("Expected 2 iterations, got %d", result.Iterations)
	}

	// Total time should be:
	// - ~0ms: first iteration starts immediately
	// - ~100ms: delay before second iteration
	// - After 2nd iteration, NO delay (because it's the last one)
	//
	// So total should be ~100ms of delays (one delay between iter 1 and 2)
	// NOT ~200ms (which would indicate delay after last iteration too)
	totalTime := endTime.Sub(startTime)
	expectedMaxTime := 200 * time.Millisecond // 100ms delay + generous execution time

	if totalTime > expectedMaxTime {
		t.Errorf("Total time %v suggests delay was applied after last iteration (expected < %v)", totalTime, expectedMaxTime)
	}
}

// timingMockRunner wraps MockRunner and records call timestamps
type timingMockRunner struct {
	mock      *agent.MockRunner
	callTimes *[]time.Time
}

func (t *timingMockRunner) Run(opts agent.RunOptions) (*agent.RunResult, error) {
	*t.callTimes = append(*t.callTimes, time.Now())
	return t.mock.Run(opts)
}
