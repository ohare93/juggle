# Agent Testing Design - AgentRunner Interface

This document describes the testing architecture for the agent loop functionality, using interface abstraction to enable deterministic, fast tests without spawning real Claude processes.

## Core Interface

```go
// internal/agent/runner.go

package agent

import (
    "context"
    "time"
)

// AgentRunner abstracts the agent execution for testability
type AgentRunner interface {
    // Run executes the agent with the given prompt and returns output
    Run(ctx context.Context, prompt string, opts RunOptions) (RunResult, error)
}

type RunOptions struct {
    Trust   bool          // Skip permission prompts
    Timeout time.Duration // Per-iteration timeout
    Model   string        // Model override (optional)
}

type RunResult struct {
    Output        string     // Full stdout/stderr
    ExitCode      int        // Process exit code
    Signal        SignalType // COMPLETE, BLOCKED, or NONE
    BlockedReason string     // If Signal == BLOCKED
}

type SignalType int

const (
    SignalNone SignalType = iota
    SignalComplete
    SignalBlocked
)

// ParseSignal extracts completion signal from output
func ParseSignal(output string) (SignalType, string) {
    if strings.Contains(output, "<promise>COMPLETE</promise>") {
        return SignalComplete, ""
    }
    if idx := strings.Index(output, "<promise>BLOCKED:"); idx != -1 {
        // Extract reason between "BLOCKED:" and "</promise>"
        start := idx + len("<promise>BLOCKED:")
        end := strings.Index(output[start:], "</promise>")
        if end != -1 {
            reason := strings.TrimSpace(output[start : start+end])
            return SignalBlocked, reason
        }
    }
    return SignalNone, ""
}
```

## Production Implementation

```go
// internal/agent/claude_runner.go

package agent

import (
    "bytes"
    "context"
    "io"
    "os"
    "os/exec"
)

type ClaudeRunner struct {
    Command string // Default: "claude"
}

func NewClaudeRunner() *ClaudeRunner {
    return &ClaudeRunner{Command: "claude"}
}

func (r *ClaudeRunner) Run(ctx context.Context, prompt string, opts RunOptions) (RunResult, error) {
    args := []string{
        "--disable-slash-commands",
        "--setting-sources", "",
        "--append-system-prompt", autonomousSystemPrompt,
        "-p", "-",
    }

    if opts.Trust {
        args = append(args, "--dangerously-skip-permissions")
    } else {
        args = append(args, "--permission-mode", "acceptEdits")
    }

    if opts.Model != "" {
        args = append(args, "--model", opts.Model)
    }

    cmd := exec.CommandContext(ctx, r.Command, args...)
    cmd.Stdin = strings.NewReader(prompt)

    var output bytes.Buffer
    cmd.Stdout = io.MultiWriter(os.Stdout, &output)
    cmd.Stderr = io.MultiWriter(os.Stderr, &output)

    err := cmd.Run()
    exitCode := 0
    if exitErr, ok := err.(*exec.ExitError); ok {
        exitCode = exitErr.ExitCode()
    }

    signal, reason := ParseSignal(output.String())

    return RunResult{
        Output:        output.String(),
        ExitCode:      exitCode,
        Signal:        signal,
        BlockedReason: reason,
    }, nil // Return nil error even on non-zero exit - agent may have output
}

const autonomousSystemPrompt = `CRITICAL: You are an autonomous agent.
DO NOT ask questions. DO NOT summarize. DO NOT wait for confirmation.
START WORKING IMMEDIATELY. Execute the workflow in prompt.md without any preamble.`
```

## Mock Implementation

```go
// internal/agent/mock_runner.go

package agent

import (
    "context"
    "fmt"
)

// MockRunner provides scripted responses for testing
type MockRunner struct {
    Responses []RunResult // Scripted responses per iteration
    Calls     []MockCall  // Captured calls for assertion
    CallIndex int

    // Optional: custom handler for dynamic responses
    Handler func(prompt string, opts RunOptions) (RunResult, error)
}

type MockCall struct {
    Prompt  string
    Options RunOptions
}

func NewMockRunner(responses ...RunResult) *MockRunner {
    return &MockRunner{Responses: responses}
}

func (r *MockRunner) Run(ctx context.Context, prompt string, opts RunOptions) (RunResult, error) {
    r.Calls = append(r.Calls, MockCall{Prompt: prompt, Options: opts})

    // Check context cancellation
    if ctx.Err() != nil {
        return RunResult{}, ctx.Err()
    }

    // Use custom handler if set
    if r.Handler != nil {
        return r.Handler(prompt, opts)
    }

    // Return scripted response
    if r.CallIndex >= len(r.Responses) {
        // Default: signal complete if no more responses
        return RunResult{Signal: SignalComplete, Output: "<promise>COMPLETE</promise>"}, nil
    }

    result := r.Responses[r.CallIndex]
    r.CallIndex++
    return result, nil
}

// Reset clears call history for reuse
func (r *MockRunner) Reset() {
    r.Calls = nil
    r.CallIndex = 0
}

// AssertCallCount verifies expected number of iterations
func (r *MockRunner) AssertCallCount(t *testing.T, expected int) {
    t.Helper()
    if len(r.Calls) != expected {
        t.Errorf("expected %d calls, got %d", expected, len(r.Calls))
    }
}

// AssertPromptContains verifies prompt content
func (r *MockRunner) AssertPromptContains(t *testing.T, callIndex int, substr string) {
    t.Helper()
    if callIndex >= len(r.Calls) {
        t.Fatalf("call index %d out of range (got %d calls)", callIndex, len(r.Calls))
    }
    if !strings.Contains(r.Calls[callIndex].Prompt, substr) {
        t.Errorf("call %d prompt does not contain %q", callIndex, substr)
    }
}
```

## CLI Wiring

```go
// internal/cli/agent.go (modified)

package cli

import (
    "github.com/ohare93/juggle/internal/agent"
    // ...
)

// Package-level variable for dependency injection
var AgentRunner agent.AgentRunner = agent.NewClaudeRunner()

func runAgentRun(cmd *cobra.Command, args []string) error {
    sessionID := args[0]

    // ... existing setup code ...

    for i := 0; i < maxIterations; i++ {
        prompt, err := generateAgentPrompt(sessionID)
        if err != nil {
            return err
        }

        ctx := cmd.Context()
        if timeout > 0 {
            var cancel context.CancelFunc
            ctx, cancel = context.WithTimeout(ctx, timeout)
            defer cancel()
        }

        result, err := AgentRunner.Run(ctx, prompt, agent.RunOptions{
            Trust:   trustFlag,
            Timeout: timeout,
        })
        if err != nil {
            return fmt.Errorf("agent execution failed: %w", err)
        }

        // Handle signals
        switch result.Signal {
        case agent.SignalComplete:
            fmt.Println("✓ Agent signaled COMPLETE")
            return nil
        case agent.SignalBlocked:
            fmt.Printf("✗ Agent blocked: %s\n", result.BlockedReason)
            return fmt.Errorf("agent blocked: %s", result.BlockedReason)
        }

        // Check if all balls complete
        if allBallsComplete(sessionID) {
            fmt.Println("✓ All session balls complete")
            return nil
        }

        fmt.Printf("Progress: iteration %d/%d complete\n", i+1, maxIterations)
    }

    fmt.Printf("Reached max iterations (%d)\n", maxIterations)
    return nil
}
```

## Test Examples

### Basic Completion Test

```go
// internal/integration_test/agent_test.go

func TestAgentRun_CompletesOnSignal(t *testing.T) {
    env := SetupTestEnv(t)
    defer CleanupTestEnv(t, env)

    // Setup: session with one ball
    setupSessionWithBalls(t, env, "test-session", 1)

    // Install mock runner
    originalRunner := cli.AgentRunner
    defer func() { cli.AgentRunner = originalRunner }()

    mock := agent.NewMockRunner(
        agent.RunResult{
            Signal: agent.SignalComplete,
            Output: "Working... <promise>COMPLETE</promise>",
        },
    )
    cli.AgentRunner = mock

    // Execute
    err := runAgentCommand(t, "test-session", "--iterations", "5")

    // Assert
    require.NoError(t, err)
    mock.AssertCallCount(t, 1) // Stopped after first iteration
}
```

### Blocked Signal Test

```go
func TestAgentRun_ExitsOnBlocked(t *testing.T) {
    env := SetupTestEnv(t)
    defer CleanupTestEnv(t, env)

    setupSessionWithBalls(t, env, "blocked-test", 2)

    originalRunner := cli.AgentRunner
    defer func() { cli.AgentRunner = originalRunner }()

    cli.AgentRunner = agent.NewMockRunner(
        agent.RunResult{Signal: agent.SignalNone, Output: "Starting..."},
        agent.RunResult{
            Signal:        agent.SignalBlocked,
            BlockedReason: "npm not found",
            Output:        "<promise>BLOCKED: npm not found</promise>",
        },
    )

    err := runAgentCommand(t, "blocked-test", "--iterations", "10")

    require.Error(t, err)
    require.Contains(t, err.Error(), "npm not found")
    require.Len(t, cli.AgentRunner.(*agent.MockRunner).Calls, 2)
}
```

### Multi-Iteration Test

```go
func TestAgentRun_IteratesUntilComplete(t *testing.T) {
    env := SetupTestEnv(t)
    defer CleanupTestEnv(t, env)

    setupSessionWithBalls(t, env, "multi-test", 3)

    originalRunner := cli.AgentRunner
    defer func() { cli.AgentRunner = originalRunner }()

    cli.AgentRunner = agent.NewMockRunner(
        agent.RunResult{Signal: agent.SignalNone, Output: "Ball 1 done"},
        agent.RunResult{Signal: agent.SignalNone, Output: "Ball 2 done"},
        agent.RunResult{Signal: agent.SignalNone, Output: "Ball 3 done"},
        agent.RunResult{Signal: agent.SignalComplete, Output: "<promise>COMPLETE</promise>"},
    )

    err := runAgentCommand(t, "multi-test", "--iterations", "10")

    require.NoError(t, err)
    require.Len(t, cli.AgentRunner.(*agent.MockRunner).Calls, 4)
}
```

### Max Iterations Test

```go
func TestAgentRun_RespectsMaxIterations(t *testing.T) {
    env := SetupTestEnv(t)
    defer CleanupTestEnv(t, env)

    setupSessionWithBalls(t, env, "max-test", 1)

    originalRunner := cli.AgentRunner
    defer func() { cli.AgentRunner = originalRunner }()

    // Never signals complete
    cli.AgentRunner = agent.NewMockRunner(
        agent.RunResult{Signal: agent.SignalNone, Output: "Working..."},
        agent.RunResult{Signal: agent.SignalNone, Output: "Still working..."},
        agent.RunResult{Signal: agent.SignalNone, Output: "More work..."},
    )

    err := runAgentCommand(t, "max-test", "--iterations", "3")

    require.NoError(t, err) // Max iterations is not an error
    require.Len(t, cli.AgentRunner.(*agent.MockRunner).Calls, 3)
}
```

### Prompt Content Verification

```go
func TestAgentRun_PromptContainsRequiredSections(t *testing.T) {
    env := SetupTestEnv(t)
    defer CleanupTestEnv(t, env)

    // Setup session with context and progress
    sessionStore := env.GetSessionStore(t)
    sessionStore.CreateSession("prompt-test", "Epic description")
    sessionStore.UpdateContext("prompt-test", "Custom context\nMultiple lines")
    sessionStore.AppendProgress("prompt-test", "Prior work done")

    store := env.GetStore(t)
    ball := &session.Session{
        ID:                 "juggler-test-1",
        Intent:             "Build feature X",
        AcceptanceCriteria: []string{"Criterion A", "Criterion B"},
        Priority:           session.PriorityHigh,
        State:              session.JugglePending,
        Tags:               []string{"prompt-test"},
    }
    store.AppendBall(ball)

    originalRunner := cli.AgentRunner
    defer func() { cli.AgentRunner = originalRunner }()

    mock := agent.NewMockRunner(
        agent.RunResult{Signal: agent.SignalComplete},
    )
    cli.AgentRunner = mock

    runAgentCommand(t, "prompt-test", "--iterations", "1")

    prompt := mock.Calls[0].Prompt

    // Verify all sections present
    assert.Contains(t, prompt, "<context>")
    assert.Contains(t, prompt, "Epic description")
    assert.Contains(t, prompt, "Custom context")
    assert.Contains(t, prompt, "<progress>")
    assert.Contains(t, prompt, "Prior work done")
    assert.Contains(t, prompt, "<balls>")
    assert.Contains(t, prompt, "Build feature X")
    assert.Contains(t, prompt, "Criterion A")
    assert.Contains(t, prompt, "<instructions>")
    assert.Contains(t, prompt, "ONE BALL PER ITERATION")
    assert.Contains(t, prompt, "<promise>COMPLETE</promise>")
}
```

### Empty Session Test

```go
func TestAgentRun_EmptySessionExitsImmediately(t *testing.T) {
    env := SetupTestEnv(t)
    defer CleanupTestEnv(t, env)

    sessionStore := env.GetSessionStore(t)
    sessionStore.CreateSession("empty-test", "No balls")

    originalRunner := cli.AgentRunner
    defer func() { cli.AgentRunner = originalRunner }()

    mock := agent.NewMockRunner()
    cli.AgentRunner = mock

    err := runAgentCommand(t, "empty-test", "--iterations", "5")

    // Should exit without calling agent
    require.NoError(t, err)
    require.Len(t, mock.Calls, 0, "Agent should not be called for empty session")
}
```

### Trust Flag Test

```go
func TestAgentRun_TrustFlagPassedToRunner(t *testing.T) {
    env := SetupTestEnv(t)
    defer CleanupTestEnv(t, env)

    setupSessionWithBalls(t, env, "trust-test", 1)

    originalRunner := cli.AgentRunner
    defer func() { cli.AgentRunner = originalRunner }()

    mock := agent.NewMockRunner(
        agent.RunResult{Signal: agent.SignalComplete},
    )
    cli.AgentRunner = mock

    // Without --trust
    runAgentCommand(t, "trust-test", "--iterations", "1")
    assert.False(t, mock.Calls[0].Options.Trust)

    mock.Reset()

    // With --trust
    runAgentCommand(t, "trust-test", "--iterations", "1", "--trust")
    assert.True(t, mock.Calls[0].Options.Trust)
}
```

## Test Helpers

```go
// internal/integration_test/agent_helpers_test.go

func setupSessionWithBalls(t *testing.T, env *TestEnv, sessionID string, count int) {
    t.Helper()

    sessionStore := env.GetSessionStore(t)
    sessionStore.CreateSession(sessionID, fmt.Sprintf("Test session with %d balls", count))

    store := env.GetStore(t)
    for i := 1; i <= count; i++ {
        ball := &session.Session{
            ID:       fmt.Sprintf("test-%s-%d", sessionID, i),
            Intent:   fmt.Sprintf("Task %d for %s", i, sessionID),
            Priority: session.PriorityMedium,
            State:    session.JugglePending,
            Tags:     []string{sessionID},
        }
        store.AppendBall(ball)
    }
}

func runAgentCommand(t *testing.T, args ...string) error {
    t.Helper()

    fullArgs := append([]string{"agent", "run"}, args...)
    cmd := cli.NewRootCmd()
    cmd.SetArgs(fullArgs)
    return cmd.Execute()
}
```

## Summary

This design provides:

1. **Clean separation** - Interface abstracts subprocess execution
2. **Fast tests** - No real Claude processes spawned
3. **Deterministic** - Scripted responses ensure predictable behavior
4. **Flexible** - MockRunner supports both scripted and dynamic responses
5. **Observable** - Captured calls enable prompt verification
6. **Safe defaults** - Mock signals COMPLETE if responses exhausted

Implementation balls: `juggler-81` (interface refactor) and `juggler-82` (integration tests).
