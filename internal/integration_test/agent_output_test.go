package integration_test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/ohare93/juggle/internal/agent"
	"github.com/ohare93/juggle/internal/cli"
	"github.com/ohare93/juggle/internal/session"
)

// captureOutput captures stdout during function execution and returns the output as a string.
// Uses defer to ensure os.Stdout is restored even if f() panics.
func captureOutput(f func()) string {
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		panic("Failed to create pipe for output capture: " + err.Error())
	}

	os.Stdout = w
	defer func() {
		os.Stdout = old
	}()

	f()

	w.Close()

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

// Separator constant used in agent output
const iterationSeparator = "════════════════════════════════════════════════════════════════════════════════"

// TestOutputFormatting_SingleIteration_Complete verifies output formatting for single iteration with COMPLETE signal
func TestOutputFormatting_SingleIteration_Complete(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	env.CreateSession(t, "test-session", "Test session")

	// Create a complete ball so COMPLETE signal exits
	ball := env.CreateBall(t, "Test ball", session.PriorityMedium)
	ball.Tags = []string{"test-session"}
	ball.State = session.StateComplete
	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	mock := agent.NewMockRunner(
		&agent.RunResult{
			Output:   "Working...\n<promise>COMPLETE</promise>\nDone.",
			Complete: true,
		},
	)
	agent.SetRunner(mock)
	defer agent.ResetRunner()

	config := cli.AgentLoopConfig{
		SessionID:     "test-session",
		ProjectDir:    env.ProjectDir,
		MaxIterations: 5,
		Trust:         false,
		IterDelay:     0,
	}

	output := captureOutput(func() {
		cli.RunAgentLoop(config)
	})

	// Verify iteration header appears
	if !strings.Contains(output, "Iteration 1/5") {
		t.Errorf("Expected iteration header 'Iteration 1/5' in output:\n%s", output)
	}

	// Verify no separator before first iteration (separator only between iterations)
	if strings.Contains(output, iterationSeparator) {
		t.Errorf("Expected no separator for single iteration, but found one in output:\n%s", output)
	}
}

// TestOutputFormatting_SingleIteration_Blocked verifies output formatting for single iteration with BLOCKED signal
func TestOutputFormatting_SingleIteration_Blocked(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	env.CreateSession(t, "test-session", "Test session")

	ball := env.CreateInProgressBall(t, "Test ball", session.PriorityMedium)
	ball.Tags = []string{"test-session"}
	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	mock := agent.NewMockRunner(
		&agent.RunResult{
			Output:        "Can't continue\n<promise>BLOCKED: waiting on approval</promise>",
			Blocked:       true,
			BlockedReason: "waiting on approval",
		},
	)
	agent.SetRunner(mock)
	defer agent.ResetRunner()

	config := cli.AgentLoopConfig{
		SessionID:     "test-session",
		ProjectDir:    env.ProjectDir,
		MaxIterations: 5,
		Trust:         false,
		IterDelay:     0,
	}

	output := captureOutput(func() {
		cli.RunAgentLoop(config)
	})

	// Verify iteration header appears
	if !strings.Contains(output, "Iteration 1/5") {
		t.Errorf("Expected iteration header in output:\n%s", output)
	}

	// Verify no separator (only one iteration)
	if strings.Contains(output, iterationSeparator) {
		t.Errorf("Expected no separator for single iteration, but found one")
	}
}

// TestOutputFormatting_MultipleIterations_Continue verifies output formatting for multiple iterations with CONTINUE signals
func TestOutputFormatting_MultipleIterations_Continue(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	env.CreateSession(t, "test-session", "Test session")

	// Create balls pre-marked as complete so COMPLETE signal will exit
	// (The test focuses on output formatting, not state validation)
	ball1 := env.CreateBall(t, "First ball", session.PriorityMedium)
	ball1.Tags = []string{"test-session"}
	ball1.State = session.StateComplete
	ball2 := env.CreateBall(t, "Second ball", session.PriorityMedium)
	ball2.Tags = []string{"test-session"}
	ball2.State = session.StateComplete
	store := env.GetStore(t)
	if err := store.UpdateBall(ball1); err != nil {
		t.Fatalf("Failed to update ball1: %v", err)
	}
	if err := store.UpdateBall(ball2); err != nil {
		t.Fatalf("Failed to update ball2: %v", err)
	}

	// First iteration: CONTINUE (agent says one ball done, continues to next)
	// Second iteration: COMPLETE (agent says all done, exits because balls are complete)
	mock := agent.NewMockRunner(
		&agent.RunResult{
			Output:   "Finished first ball\n<promise>CONTINUE</promise>",
			Continue: true,
		},
		&agent.RunResult{
			Output:   "All done\n<promise>COMPLETE</promise>",
			Complete: true,
		},
	)
	agent.SetRunner(mock)
	defer agent.ResetRunner()

	config := cli.AgentLoopConfig{
		SessionID:     "test-session",
		ProjectDir:    env.ProjectDir,
		MaxIterations: 5,
		Trust:         false,
		IterDelay:     0,
	}

	output := captureOutput(func() {
		cli.RunAgentLoop(config)
	})

	// Verify both iteration headers appear
	if !strings.Contains(output, "Iteration 1/5") {
		t.Errorf("Expected 'Iteration 1/5' in output:\n%s", output)
	}
	if !strings.Contains(output, "Iteration 2/5") {
		t.Errorf("Expected 'Iteration 2/5' in output:\n%s", output)
	}

	// Verify separator between iterations (exactly one separator for two iterations)
	separatorCount := strings.Count(output, iterationSeparator)
	if separatorCount != 1 {
		t.Errorf("Expected 1 separator between iterations, got %d in output:\n%s", separatorCount, output)
	}

	// Verify continuation message has spacing (blank line before it)
	if !strings.Contains(output, "\n\n✓ Agent completed a ball") {
		t.Errorf("Expected blank line before continuation message in output:\n%s", output)
	}
}

// TestOutputFormatting_MultipleIterations_MaxReached verifies output formatting when max iterations is reached
func TestOutputFormatting_MultipleIterations_MaxReached(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	env.CreateSession(t, "test-session", "Test session")

	// Ball stays in progress - never completes
	ball := env.CreateInProgressBall(t, "Ongoing ball", session.PriorityMedium)
	ball.Tags = []string{"test-session"}
	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// All iterations return no completion signals
	mock := agent.NewMockRunner(
		&agent.RunResult{Output: "Working iteration 1..."},
		&agent.RunResult{Output: "Working iteration 2..."},
		&agent.RunResult{Output: "Working iteration 3..."},
	)
	agent.SetRunner(mock)
	defer agent.ResetRunner()

	config := cli.AgentLoopConfig{
		SessionID:     "test-session",
		ProjectDir:    env.ProjectDir,
		MaxIterations: 3, // Only 3 iterations
		Trust:         false,
		IterDelay:     0,
	}

	output := captureOutput(func() {
		cli.RunAgentLoop(config)
	})

	// Verify all iteration headers appear
	if !strings.Contains(output, "Iteration 1/3") {
		t.Errorf("Expected 'Iteration 1/3' in output:\n%s", output)
	}
	if !strings.Contains(output, "Iteration 2/3") {
		t.Errorf("Expected 'Iteration 2/3' in output:\n%s", output)
	}
	if !strings.Contains(output, "Iteration 3/3") {
		t.Errorf("Expected 'Iteration 3/3' in output:\n%s", output)
	}

	// Verify correct number of separators (iterations - 1 = 2)
	separatorCount := strings.Count(output, iterationSeparator)
	if separatorCount != 2 {
		t.Errorf("Expected 2 separators for 3 iterations, got %d", separatorCount)
	}
}

// TestOutputFormatting_PrematureComplete verifies output formatting when COMPLETE is signaled prematurely
func TestOutputFormatting_PrematureComplete(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	env.CreateSession(t, "test-session", "Test session")

	// Ball stays in progress - COMPLETE signal should be ignored
	ball := env.CreateInProgressBall(t, "Incomplete ball", session.PriorityMedium)
	ball.Tags = []string{"test-session"}
	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// First iteration: COMPLETE (but ball not done - should warn and continue)
	// Second iteration: no signal, ball still not done
	mock := agent.NewMockRunner(
		&agent.RunResult{
			Output:   "Done!\n<promise>COMPLETE</promise>",
			Complete: true,
		},
		&agent.RunResult{
			Output: "Still working...",
		},
	)
	agent.SetRunner(mock)
	defer agent.ResetRunner()

	config := cli.AgentLoopConfig{
		SessionID:     "test-session",
		ProjectDir:    env.ProjectDir,
		MaxIterations: 2,
		Trust:         false,
		IterDelay:     0,
	}

	output := captureOutput(func() {
		cli.RunAgentLoop(config)
	})

	// Verify warning message appears with spacing
	if !strings.Contains(output, "⚠️  Agent signaled COMPLETE but only") {
		t.Errorf("Expected premature COMPLETE warning in output:\n%s", output)
	}

	// Verify blank line before warning
	if !strings.Contains(output, "\n\n⚠️  Agent signaled COMPLETE") {
		t.Errorf("Expected blank line before warning in output:\n%s", output)
	}

	// Verify second iteration runs (separator should appear)
	if !strings.Contains(output, "Iteration 2/2") {
		t.Errorf("Expected 'Iteration 2/2' after premature COMPLETE in output:\n%s", output)
	}
}

// TestOutputFormatting_RateLimit verifies output formatting during rate limiting
func TestOutputFormatting_RateLimit(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	env.CreateSession(t, "test-session", "Test session")

	ball := env.CreateBall(t, "Test ball", session.PriorityMedium)
	ball.Tags = []string{"test-session"}
	ball.State = session.StateComplete
	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// First call: rate limited, second call: success
	mock := agent.NewMockRunner(
		&agent.RunResult{
			RateLimited: true,
			RetryAfter:  0, // Immediate retry for test
		},
		&agent.RunResult{
			Output:   "Success\n<promise>COMPLETE</promise>",
			Complete: true,
		},
	)
	agent.SetRunner(mock)
	defer agent.ResetRunner()

	config := cli.AgentLoopConfig{
		SessionID:     "test-session",
		ProjectDir:    env.ProjectDir,
		MaxIterations: 5,
		Trust:         false,
		IterDelay:     0,
	}

	output := captureOutput(func() {
		cli.RunAgentLoop(config)
	})

	// Verify rate limit message appears
	if !strings.Contains(output, "⏳ Rate limited") {
		t.Errorf("Expected rate limit message in output:\n%s", output)
	}

	// Verify only ONE iteration header (rate limit retries same iteration)
	iterCount := strings.Count(output, "Iteration 1/5")
	if iterCount != 1 {
		t.Errorf("Expected exactly 1 'Iteration 1/5' header (rate limit retries same iteration), got %d", iterCount)
	}

	// Verify no separator (still iteration 1 after retry)
	if strings.Contains(output, iterationSeparator) {
		t.Errorf("Expected no separator when rate limited (same iteration), but found one")
	}
}

// TestOutputFormatting_Timeout verifies output formatting when iteration times out
func TestOutputFormatting_Timeout(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	env.CreateSession(t, "test-session", "Test session")

	ball := env.CreateInProgressBall(t, "Test ball", session.PriorityMedium)
	ball.Tags = []string{"test-session"}
	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	mock := agent.NewMockRunner(
		&agent.RunResult{
			TimedOut: true,
		},
	)
	agent.SetRunner(mock)
	defer agent.ResetRunner()

	config := cli.AgentLoopConfig{
		SessionID:     "test-session",
		ProjectDir:    env.ProjectDir,
		MaxIterations: 5,
		Trust:         false,
		IterDelay:     0,
	}

	output := captureOutput(func() {
		cli.RunAgentLoop(config)
	})

	// Verify iteration header appeared before timeout
	if !strings.Contains(output, "Iteration 1/5") {
		t.Errorf("Expected iteration header before timeout in output:\n%s", output)
	}

	// No separator should appear (only one iteration before timeout)
	if strings.Contains(output, iterationSeparator) {
		t.Errorf("Expected no separator for single iteration timeout")
	}
}

// TestOutputFormatting_Iterations_1 verifies output formatting with MaxIterations=1
func TestOutputFormatting_Iterations_1(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	env.CreateSession(t, "test-session", "Test session")

	ball := env.CreateInProgressBall(t, "Test ball", session.PriorityMedium)
	ball.Tags = []string{"test-session"}
	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	mock := agent.NewMockRunner(
		&agent.RunResult{Output: "Single iteration"},
	)
	agent.SetRunner(mock)
	defer agent.ResetRunner()

	config := cli.AgentLoopConfig{
		SessionID:     "test-session",
		ProjectDir:    env.ProjectDir,
		MaxIterations: 1,
		Trust:         false,
		IterDelay:     0,
	}

	output := captureOutput(func() {
		cli.RunAgentLoop(config)
	})

	// Verify header shows 1/1
	if !strings.Contains(output, "Iteration 1/1") {
		t.Errorf("Expected 'Iteration 1/1' in output:\n%s", output)
	}

	// No separator for single iteration
	if strings.Contains(output, iterationSeparator) {
		t.Errorf("Expected no separator for MaxIterations=1")
	}
}

// TestOutputFormatting_Iterations_3 verifies output formatting with 3 iterations using CONTINUE then COMPLETE
func TestOutputFormatting_Iterations_3(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	env.CreateSession(t, "test-session", "Test session")

	ball := env.CreateBall(t, "Test ball", session.PriorityMedium)
	ball.Tags = []string{"test-session"}
	ball.State = session.StateComplete // Pre-complete so COMPLETE signal exits
	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	mock := agent.NewMockRunner(
		&agent.RunResult{Output: "Working 1\n<promise>CONTINUE</promise>", Continue: true},
		&agent.RunResult{Output: "Working 2\n<promise>CONTINUE</promise>", Continue: true},
		&agent.RunResult{Output: "Done\n<promise>COMPLETE</promise>", Complete: true},
	)
	agent.SetRunner(mock)
	defer agent.ResetRunner()

	config := cli.AgentLoopConfig{
		SessionID:     "test-session",
		ProjectDir:    env.ProjectDir,
		MaxIterations: 5,
		Trust:         false,
		IterDelay:     0,
	}

	output := captureOutput(func() {
		cli.RunAgentLoop(config)
	})

	// Verify all 3 iteration headers
	if !strings.Contains(output, "Iteration 1/5") {
		t.Errorf("Expected 'Iteration 1/5'")
	}
	if !strings.Contains(output, "Iteration 2/5") {
		t.Errorf("Expected 'Iteration 2/5'")
	}
	if !strings.Contains(output, "Iteration 3/5") {
		t.Errorf("Expected 'Iteration 3/5'")
	}

	// Verify 2 separators (between iterations 1-2 and 2-3)
	separatorCount := strings.Count(output, iterationSeparator)
	if separatorCount != 2 {
		t.Errorf("Expected 2 separators for 3 iterations, got %d", separatorCount)
	}

	// Verify continuation messages
	continueCount := strings.Count(output, "✓ Agent completed a ball")
	if continueCount != 2 {
		t.Errorf("Expected 2 continuation messages, got %d", continueCount)
	}
}

// TestOutputFormatting_Iterations_10 verifies output formatting with 10 iterations
func TestOutputFormatting_Iterations_10(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	env.CreateSession(t, "test-session", "Test session")

	ball := env.CreateInProgressBall(t, "Test ball", session.PriorityMedium)
	ball.Tags = []string{"test-session"}
	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// Create 10 mock responses (no completion signals - will hit max iterations)
	responses := make([]*agent.RunResult, 10)
	for i := 0; i < 10; i++ {
		responses[i] = &agent.RunResult{Output: "Working..."}
	}
	mock := agent.NewMockRunner(responses...)
	agent.SetRunner(mock)
	defer agent.ResetRunner()

	config := cli.AgentLoopConfig{
		SessionID:     "test-session",
		ProjectDir:    env.ProjectDir,
		MaxIterations: 10,
		Trust:         false,
		IterDelay:     0,
	}

	output := captureOutput(func() {
		cli.RunAgentLoop(config)
	})

	// Verify all 10 iteration headers
	for i := 1; i <= 10; i++ {
		expected := fmt.Sprintf("Iteration %d/10", i)
		if !strings.Contains(output, expected) {
			t.Errorf("Expected '%s' in output", expected)
		}
	}

	// Verify 9 separators (between each pair of iterations)
	separatorCount := strings.Count(output, iterationSeparator)
	if separatorCount != 9 {
		t.Errorf("Expected 9 separators for 10 iterations, got %d", separatorCount)
	}
}
