// Package agent provides the agent prompt template and runner interface
// for running AI agents with juggler.
package agent

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// RunResult represents the outcome of a single agent run
type RunResult struct {
	Output        string
	ExitCode      int
	Complete      bool
	Blocked       bool
	BlockedReason string
	Error         error
}

// Runner defines the interface for running AI agents.
// Implementations must execute an agent with a prompt and return the result.
type Runner interface {
	// Run executes the agent with the given prompt and returns the result.
	// The trust parameter controls permission levels (true = full permissions).
	Run(prompt string, trust bool) (*RunResult, error)
}

// DefaultRunner is the package-level runner used for agent operations.
// It can be replaced with a mock for testing via SetRunner.
var DefaultRunner Runner = &ClaudeRunner{}

// SetRunner sets the package-level runner (for testing).
func SetRunner(r Runner) {
	DefaultRunner = r
}

// ResetRunner resets the runner to the default ClaudeRunner.
func ResetRunner() {
	DefaultRunner = &ClaudeRunner{}
}

// autonomousSystemPrompt is appended to force autonomous operation
const autonomousSystemPrompt = `CRITICAL: You are an autonomous agent. DO NOT ask questions. DO NOT summarize. DO NOT wait for confirmation. START WORKING IMMEDIATELY. Execute the workflow in prompt.md without any preamble.`

// ClaudeRunner runs the Claude CLI as an AI agent
type ClaudeRunner struct{}

// Run executes Claude with the given prompt
func (r *ClaudeRunner) Run(prompt string, trust bool) (*RunResult, error) {
	result := &RunResult{}

	// Build command arguments
	args := []string{
		"--disable-slash-commands",
		"--append-system-prompt", autonomousSystemPrompt,
	}

	if trust {
		args = append(args, "--dangerously-skip-permissions")
	} else {
		// Default: accept edits permission mode
		args = append(args, "--permission-mode", "acceptEdits")
	}

	// Add prompt input flag
	args = append(args, "-p", "-")

	cmd := exec.Command("claude", args...)

	// Set up stdin with prompt
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	// Set up combined stdout/stderr capture
	var outputBuf strings.Builder

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start claude: %w", err)
	}

	// Write prompt to stdin
	go func() {
		defer stdin.Close()
		io.WriteString(stdin, prompt)
	}()

	// Stream output to console and capture
	go streamOutput(stdout, &outputBuf, os.Stdout)
	go streamOutput(stderr, &outputBuf, os.Stderr)

	// Wait for command to complete
	err = cmd.Wait()
	result.Output = outputBuf.String()

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		}
		result.Error = fmt.Errorf("claude exited with error: %w", err)
	}

	// Parse completion signals from output
	r.parseSignals(result)

	return result, nil
}

// parseSignals checks the output for COMPLETE/BLOCKED signals
func (r *ClaudeRunner) parseSignals(result *RunResult) {
	if strings.Contains(result.Output, "<promise>COMPLETE</promise>") {
		result.Complete = true
	}

	if idx := strings.Index(result.Output, "<promise>BLOCKED:"); idx != -1 {
		endIdx := strings.Index(result.Output[idx:], "</promise>")
		if endIdx != -1 {
			reason := strings.TrimSpace(result.Output[idx+len("<promise>BLOCKED:") : idx+endIdx])
			result.Blocked = true
			result.BlockedReason = reason
		}
	}
}

// streamOutput reads from reader and writes to both buffer and writer
func streamOutput(reader io.Reader, buf *strings.Builder, writer io.Writer) {
	scanner := bufio.NewScanner(reader)
	// Increase scanner buffer for long lines
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		buf.WriteString(line)
		buf.WriteString("\n")
		fmt.Fprintln(writer, line)
	}
}

// MockRunner is a test implementation of Runner
type MockRunner struct {
	// Responses is a queue of results to return (FIFO)
	Responses []*RunResult
	// Calls records all calls made to Run
	Calls []MockCall
	// NextIndex tracks which response to return next
	NextIndex int
}

// MockCall records the arguments of a single Run call
type MockCall struct {
	Prompt string
	Trust  bool
}

// NewMockRunner creates a new MockRunner with the given responses
func NewMockRunner(responses ...*RunResult) *MockRunner {
	return &MockRunner{
		Responses: responses,
		Calls:     make([]MockCall, 0),
	}
}

// Run records the call and returns the next queued response
func (m *MockRunner) Run(prompt string, trust bool) (*RunResult, error) {
	m.Calls = append(m.Calls, MockCall{Prompt: prompt, Trust: trust})

	if m.NextIndex >= len(m.Responses) {
		// Return a default blocked result if no more responses queued
		return &RunResult{
			Output:        "No more mock responses",
			Blocked:       true,
			BlockedReason: "MockRunner exhausted",
		}, nil
	}

	result := m.Responses[m.NextIndex]
	m.NextIndex++
	return result, nil
}

// Reset clears call history and resets response index
func (m *MockRunner) Reset() {
	m.Calls = make([]MockCall, 0)
	m.NextIndex = 0
}

// SetResponses replaces the response queue
func (m *MockRunner) SetResponses(responses ...*RunResult) {
	m.Responses = responses
	m.NextIndex = 0
}
