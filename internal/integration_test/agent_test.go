package integration_test

import (
	"strings"
	"testing"

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

	// Create a ball tagged with the session
	ball := env.CreateBall(t, "Test ball", session.PriorityMedium)
	ball.Tags = []string{"test-session"}
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

	// Verify runner was only called once (loop exited on COMPLETE)
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
