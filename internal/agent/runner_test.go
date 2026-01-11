package agent

import (
	"testing"
)

func TestMockRunner_Run(t *testing.T) {
	t.Run("returns queued responses in order", func(t *testing.T) {
		mock := NewMockRunner(
			&RunResult{Output: "first", Complete: true},
			&RunResult{Output: "second", Blocked: true, BlockedReason: "test reason"},
		)

		// First call
		result, err := mock.Run("prompt1", false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Output != "first" {
			t.Errorf("expected output 'first', got '%s'", result.Output)
		}
		if !result.Complete {
			t.Error("expected Complete=true")
		}

		// Second call
		result, err = mock.Run("prompt2", true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Output != "second" {
			t.Errorf("expected output 'second', got '%s'", result.Output)
		}
		if !result.Blocked {
			t.Error("expected Blocked=true")
		}
		if result.BlockedReason != "test reason" {
			t.Errorf("expected BlockedReason 'test reason', got '%s'", result.BlockedReason)
		}
	})

	t.Run("records all calls", func(t *testing.T) {
		mock := NewMockRunner(&RunResult{Output: "ok"})

		mock.Run("prompt1", false)
		mock.Run("prompt2", true)
		mock.Run("prompt3", false)

		if len(mock.Calls) != 3 {
			t.Fatalf("expected 3 calls, got %d", len(mock.Calls))
		}

		if mock.Calls[0].Prompt != "prompt1" {
			t.Errorf("expected first prompt 'prompt1', got '%s'", mock.Calls[0].Prompt)
		}
		if mock.Calls[0].Trust != false {
			t.Error("expected first call Trust=false")
		}

		if mock.Calls[1].Prompt != "prompt2" {
			t.Errorf("expected second prompt 'prompt2', got '%s'", mock.Calls[1].Prompt)
		}
		if mock.Calls[1].Trust != true {
			t.Error("expected second call Trust=true")
		}
	})

	t.Run("returns default blocked when exhausted", func(t *testing.T) {
		mock := NewMockRunner(&RunResult{Output: "only one"})

		// First call succeeds
		mock.Run("first", false)

		// Second call should return default blocked
		result, err := mock.Run("second", false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Blocked {
			t.Error("expected Blocked=true when exhausted")
		}
		if result.BlockedReason != "MockRunner exhausted" {
			t.Errorf("expected BlockedReason 'MockRunner exhausted', got '%s'", result.BlockedReason)
		}
	})

	t.Run("Reset clears calls and index", func(t *testing.T) {
		mock := NewMockRunner(
			&RunResult{Output: "first"},
			&RunResult{Output: "second"},
		)

		mock.Run("prompt1", false)
		mock.Run("prompt2", false)

		mock.Reset()

		if len(mock.Calls) != 0 {
			t.Errorf("expected 0 calls after reset, got %d", len(mock.Calls))
		}
		if mock.NextIndex != 0 {
			t.Errorf("expected NextIndex=0 after reset, got %d", mock.NextIndex)
		}

		// Should return first response again
		result, _ := mock.Run("new prompt", false)
		if result.Output != "first" {
			t.Errorf("expected 'first' after reset, got '%s'", result.Output)
		}
	})

	t.Run("SetResponses replaces queue", func(t *testing.T) {
		mock := NewMockRunner(&RunResult{Output: "old"})

		mock.Run("prompt", false) // Consume old response

		mock.SetResponses(&RunResult{Output: "new1"}, &RunResult{Output: "new2"})

		result, _ := mock.Run("prompt", false)
		if result.Output != "new1" {
			t.Errorf("expected 'new1', got '%s'", result.Output)
		}

		result, _ = mock.Run("prompt", false)
		if result.Output != "new2" {
			t.Errorf("expected 'new2', got '%s'", result.Output)
		}
	})
}

func TestClaudeRunner_parseSignals(t *testing.T) {
	runner := &ClaudeRunner{}

	t.Run("detects COMPLETE signal", func(t *testing.T) {
		result := &RunResult{
			Output: "Some output...\n<promise>COMPLETE</promise>\nMore output...",
		}

		runner.parseSignals(result)

		if !result.Complete {
			t.Error("expected Complete=true")
		}
		if result.Blocked {
			t.Error("expected Blocked=false")
		}
	})

	t.Run("detects BLOCKED signal with reason", func(t *testing.T) {
		result := &RunResult{
			Output: "Some output...\n<promise>BLOCKED: tools not available</promise>\nMore output...",
		}

		runner.parseSignals(result)

		if result.Complete {
			t.Error("expected Complete=false")
		}
		if !result.Blocked {
			t.Error("expected Blocked=true")
		}
		if result.BlockedReason != "tools not available" {
			t.Errorf("expected BlockedReason 'tools not available', got '%s'", result.BlockedReason)
		}
	})

	t.Run("no signals detected in normal output", func(t *testing.T) {
		result := &RunResult{
			Output: "Just some normal output without any signals",
		}

		runner.parseSignals(result)

		if result.Complete {
			t.Error("expected Complete=false")
		}
		if result.Blocked {
			t.Error("expected Blocked=false")
		}
	})

	t.Run("handles empty output", func(t *testing.T) {
		result := &RunResult{
			Output: "",
		}

		runner.parseSignals(result)

		if result.Complete {
			t.Error("expected Complete=false")
		}
		if result.Blocked {
			t.Error("expected Blocked=false")
		}
	})
}

func TestDefaultRunner(t *testing.T) {
	t.Run("DefaultRunner is ClaudeRunner by default", func(t *testing.T) {
		ResetRunner()
		_, ok := DefaultRunner.(*ClaudeRunner)
		if !ok {
			t.Error("expected DefaultRunner to be *ClaudeRunner")
		}
	})

	t.Run("SetRunner changes DefaultRunner", func(t *testing.T) {
		mock := NewMockRunner(&RunResult{Output: "mock"})
		SetRunner(mock)

		_, ok := DefaultRunner.(*MockRunner)
		if !ok {
			t.Error("expected DefaultRunner to be *MockRunner after SetRunner")
		}

		// Clean up
		ResetRunner()
	})

	t.Run("ResetRunner restores ClaudeRunner", func(t *testing.T) {
		mock := NewMockRunner(&RunResult{Output: "mock"})
		SetRunner(mock)
		ResetRunner()

		_, ok := DefaultRunner.(*ClaudeRunner)
		if !ok {
			t.Error("expected DefaultRunner to be *ClaudeRunner after ResetRunner")
		}
	})
}
