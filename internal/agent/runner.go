// Package agent provides the agent prompt template and runner interface
// for running AI agents with juggle.
package agent

import (
	"fmt"
	"os"
	"sync"

	"github.com/ohare93/juggle/internal/agent/provider"
)

// runnerMu protects access to DefaultRunner for concurrent safety.
var runnerMu sync.RWMutex

// RunMode defines how the agent should be executed
type RunMode = provider.RunMode

const (
	// ModeHeadless runs with captured output, no terminal interaction
	ModeHeadless = provider.ModeHeadless
	// ModeInteractive runs with terminal TUI, inherits stdin/stdout/stderr
	ModeInteractive = provider.ModeInteractive
)

// PermissionMode defines the agent's permission level
type PermissionMode = provider.PermissionMode

const (
	// PermissionAcceptEdits allows file edits with confirmation
	PermissionAcceptEdits = provider.PermissionAcceptEdits
	// PermissionPlan starts in plan/read-only mode
	PermissionPlan = provider.PermissionPlan
	// PermissionBypass bypasses all permission checks (dangerous)
	PermissionBypass = provider.PermissionBypass
)

// RunOptions configures how the agent is executed
type RunOptions = provider.RunOptions

// RunResult represents the outcome of a single agent run
type RunResult = provider.RunResult

// AutonomousSystemPrompt is appended to force autonomous operation in headless mode
const AutonomousSystemPrompt = provider.AutonomousSystemPrompt

// Runner defines the interface for running AI agents.
// Implementations must execute an agent with options and return the result.
type Runner interface {
	// Run executes the agent with the given options and returns the result.
	Run(opts RunOptions) (*RunResult, error)
}

// ProviderRunner wraps a provider.Provider to implement Runner
type ProviderRunner struct {
	Provider       provider.Provider
	ModelOverrides provider.ModelOverrides
}

// Run executes the agent using the configured provider
func (r *ProviderRunner) Run(opts RunOptions) (*RunResult, error) {
	p := r.Provider
	if p == nil {
		// Default to Claude provider if not configured
		p = provider.NewClaudeProvider()
	}

	// Apply model overrides if configured
	if opts.Model != "" && r.ModelOverrides != nil {
		originalModel := opts.Model
		opts.Model = provider.ApplyModelOverrides(opts.Model, r.ModelOverrides, p)
		if opts.Model != originalModel {
			fmt.Fprintf(os.Stderr, "Model override: %s → %s\n", originalModel, opts.Model)
		}
	}

	return p.Run(opts)
}

// DefaultRunner is the package-level runner used for agent operations.
// It uses Claude by default but can be configured to use other providers.
var DefaultRunner Runner = &ProviderRunner{
	Provider: provider.NewClaudeProvider(),
}

// SetRunner sets the package-level runner (for testing).
// This function is goroutine-safe.
func SetRunner(r Runner) {
	runnerMu.Lock()
	defer runnerMu.Unlock()
	DefaultRunner = r
}

// ResetRunner resets the runner to the default Claude provider.
// This function is goroutine-safe.
func ResetRunner() {
	runnerMu.Lock()
	defer runnerMu.Unlock()
	DefaultRunner = &ProviderRunner{
		Provider: provider.NewClaudeProvider(),
	}
}

// SetProvider sets the provider for the default runner.
// This is used to switch between Claude and OpenCode at runtime.
// This function is goroutine-safe.
func SetProvider(p provider.Provider) {
	runnerMu.Lock()
	defer runnerMu.Unlock()
	if pr, ok := DefaultRunner.(*ProviderRunner); ok {
		pr.Provider = p
	}
}

// SetModelOverrides sets the model overrides for the default runner.
// This function is goroutine-safe.
func SetModelOverrides(overrides map[string]string) {
	runnerMu.Lock()
	defer runnerMu.Unlock()
	if pr, ok := DefaultRunner.(*ProviderRunner); ok {
		pr.ModelOverrides = overrides
	}
}

// GetProvider returns the current provider from the default runner.
// Returns nil if the default runner is not a ProviderRunner.
// This function is goroutine-safe.
func GetProvider() provider.Provider {
	runnerMu.RLock()
	defer runnerMu.RUnlock()
	if pr, ok := DefaultRunner.(*ProviderRunner); ok {
		return pr.Provider
	}
	return nil
}

// ClaudeRunner is an alias for backward compatibility.
// Use ProviderRunner with provider.NewClaudeProvider() instead.
type ClaudeRunner = ProviderRunner

// MockRunner is a test implementation of Runner
type MockRunner struct {
	// Responses is a queue of results to return (FIFO)
	Responses []*RunResult
	// Calls records all calls made to Run
	Calls []RunOptions
	// NextIndex tracks which response to return next
	NextIndex int
}

// NewMockRunner creates a new MockRunner with the given responses
func NewMockRunner(responses ...*RunResult) *MockRunner {
	return &MockRunner{
		Responses: responses,
		Calls:     make([]RunOptions, 0),
	}
}

// Run records the call and returns the next queued response
func (m *MockRunner) Run(opts RunOptions) (*RunResult, error) {
	m.Calls = append(m.Calls, opts)

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
	m.Calls = make([]RunOptions, 0)
	m.NextIndex = 0
}

// SetResponses replaces the response queue
func (m *MockRunner) SetResponses(responses ...*RunResult) {
	m.Responses = responses
	m.NextIndex = 0
}
