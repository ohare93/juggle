package integration_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/ohare93/juggle/internal/session"
)

// TestBallFinalState_Plan tests that juggle plan creates balls with expected final state
func TestBallFinalState_Plan(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	setupConfigWithTestProject(t, env)

	// Create a ball using CLI plan command with non-interactive mode
	output := runJuggleCommand(t, env.ProjectDir, "plan",
		"Test ball creation",
		"--non-interactive",
		"-p", "high",
		"-c", "First criterion",
		"-c", "Second criterion",
		"--context", "Background context for this task",
		"--model-size", "medium",
		"-t", "feature,backend",
	)

	// Verify output indicates successful creation
	if !strings.Contains(output, "Planned ball added") {
		t.Errorf("Expected success message, got: %s", output)
	}

	// Extract ball ID from output
	lines := strings.Split(output, "\n")
	var ballID string
	for _, line := range lines {
		if strings.Contains(line, "Planned ball added:") {
			parts := strings.Split(line, ": ")
			if len(parts) >= 2 {
				ballID = strings.TrimSpace(parts[len(parts)-1])
			}
		}
	}
	if ballID == "" {
		t.Fatalf("Could not extract ball ID from output: %s", output)
	}

	// Get ball state via --json flag
	jsonOutput := runJuggleCommand(t, env.ProjectDir, "show", ballID, "--json")

	// Parse JSON to verify final state
	var ball session.Ball
	if err := json.Unmarshal([]byte(jsonOutput), &ball); err != nil {
		t.Fatalf("Failed to parse ball JSON: %v\nOutput: %s", err, jsonOutput)
	}

	// Verify all fields match expected values
	if ball.Title != "Test ball creation" {
		t.Errorf("Expected title 'Test ball creation', got '%s'", ball.Title)
	}

	if ball.Priority != session.PriorityHigh {
		t.Errorf("Expected priority 'high', got '%s'", ball.Priority)
	}

	if ball.State != session.StatePending {
		t.Errorf("Expected state 'pending', got '%s'", ball.State)
	}

	if ball.Context != "Background context for this task" {
		t.Errorf("Expected context 'Background context for this task', got '%s'", ball.Context)
	}

	if ball.ModelSize != session.ModelSizeMedium {
		t.Errorf("Expected model size 'medium', got '%s'", ball.ModelSize)
	}

	if len(ball.AcceptanceCriteria) != 2 {
		t.Errorf("Expected 2 acceptance criteria, got %d", len(ball.AcceptanceCriteria))
	} else {
		if ball.AcceptanceCriteria[0] != "First criterion" {
			t.Errorf("Expected first AC 'First criterion', got '%s'", ball.AcceptanceCriteria[0])
		}
		if ball.AcceptanceCriteria[1] != "Second criterion" {
			t.Errorf("Expected second AC 'Second criterion', got '%s'", ball.AcceptanceCriteria[1])
		}
	}

	// Verify tags (order may vary)
	expectedTags := map[string]bool{"feature": true, "backend": true}
	if len(ball.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d: %v", len(ball.Tags), ball.Tags)
	} else {
		for _, tag := range ball.Tags {
			if !expectedTags[tag] {
				t.Errorf("Unexpected tag: %s", tag)
			}
		}
	}
}

// TestBallFinalState_Update tests that juggle update modifies ball state correctly
func TestBallFinalState_Update(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	setupConfigWithTestProject(t, env)

	// Create a simple ball first
	output := runJuggleCommand(t, env.ProjectDir, "plan",
		"Original title",
		"--non-interactive",
		"-p", "low",
	)

	// Extract ball ID
	var ballID string
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Planned ball added:") {
			parts := strings.Split(line, ": ")
			if len(parts) >= 2 {
				ballID = strings.TrimSpace(parts[len(parts)-1])
			}
		}
	}
	if ballID == "" {
		t.Fatalf("Could not extract ball ID from output: %s", output)
	}

	// Update the ball with new values using the proper update command syntax
	runJuggleCommand(t, env.ProjectDir, "update", ballID,
		"--intent", "Updated title",
		"--priority", "urgent",
		"--tags", "updated-tag,another-tag",
	)

	// Verify final state via JSON
	jsonOutput := runJuggleCommand(t, env.ProjectDir, "show", ballID, "--json")

	var ball session.Ball
	if err := json.Unmarshal([]byte(jsonOutput), &ball); err != nil {
		t.Fatalf("Failed to parse ball JSON: %v\nOutput: %s", err, jsonOutput)
	}

	if ball.Title != "Updated title" {
		t.Errorf("Expected title 'Updated title', got '%s'", ball.Title)
	}

	if ball.Priority != session.PriorityUrgent {
		t.Errorf("Expected priority 'urgent', got '%s'", ball.Priority)
	}

	expectedTags := map[string]bool{"updated-tag": true, "another-tag": true}
	if len(ball.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d: %v", len(ball.Tags), ball.Tags)
	} else {
		for _, tag := range ball.Tags {
			if !expectedTags[tag] {
				t.Errorf("Unexpected tag: %s", tag)
			}
		}
	}
}

// TestBallFinalState_StateTransitions tests state transitions and their JSON output
func TestBallFinalState_StateTransitions(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	setupConfigWithTestProject(t, env)

	// Create a ball
	output := runJuggleCommand(t, env.ProjectDir, "plan",
		"State transition test",
		"--non-interactive",
	)

	// Extract ball ID
	var ballID string
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Planned ball added:") {
			parts := strings.Split(line, ": ")
			if len(parts) >= 2 {
				ballID = strings.TrimSpace(parts[len(parts)-1])
			}
		}
	}
	if ballID == "" {
		t.Fatalf("Could not extract ball ID from output: %s", output)
	}

	// Verify initial state is pending
	jsonOutput := runJuggleCommand(t, env.ProjectDir, "show", ballID, "--json")
	var ball session.Ball
	if err := json.Unmarshal([]byte(jsonOutput), &ball); err != nil {
		t.Fatalf("Failed to parse ball JSON: %v", err)
	}
	if ball.State != session.StatePending {
		t.Errorf("Expected initial state 'pending', got '%s'", ball.State)
	}

	// Transition to in_progress
	runJuggleCommand(t, env.ProjectDir, ballID, "in-progress")

	jsonOutput = runJuggleCommand(t, env.ProjectDir, "show", ballID, "--json")
	if err := json.Unmarshal([]byte(jsonOutput), &ball); err != nil {
		t.Fatalf("Failed to parse ball JSON: %v", err)
	}
	if ball.State != session.StateInProgress {
		t.Errorf("Expected state 'in_progress', got '%s'", ball.State)
	}

	// Transition to blocked with reason
	runJuggleCommand(t, env.ProjectDir, ballID, "blocked", "Waiting for API access")

	jsonOutput = runJuggleCommand(t, env.ProjectDir, "show", ballID, "--json")
	if err := json.Unmarshal([]byte(jsonOutput), &ball); err != nil {
		t.Fatalf("Failed to parse ball JSON: %v", err)
	}
	if ball.State != session.StateBlocked {
		t.Errorf("Expected state 'blocked', got '%s'", ball.State)
	}
	if ball.BlockedReason != "Waiting for API access" {
		t.Errorf("Expected blocked reason 'Waiting for API access', got '%s'", ball.BlockedReason)
	}
}

// TestBallFinalState_GlobalJSONFlag tests that --json global flag works with ball ID
func TestBallFinalState_GlobalJSONFlag(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	setupConfigWithTestProject(t, env)

	// Create a ball
	output := runJuggleCommand(t, env.ProjectDir, "plan",
		"Global JSON test",
		"--non-interactive",
	)

	// Extract ball ID
	var ballID string
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Planned ball added:") {
			parts := strings.Split(line, ": ")
			if len(parts) >= 2 {
				ballID = strings.TrimSpace(parts[len(parts)-1])
			}
		}
	}
	if ballID == "" {
		t.Fatalf("Could not extract ball ID from output: %s", output)
	}

	// Start the ball (transitions to in_progress)
	runJuggleCommand(t, env.ProjectDir, ballID)

	// Now use global --json flag with ball ID to get JSON output
	// When ball is in_progress, juggle <id> shows details
	jsonOutput := runJuggleCommand(t, env.ProjectDir, "--json", ballID)

	// Should be valid JSON
	var ball session.Ball
	if err := json.Unmarshal([]byte(jsonOutput), &ball); err != nil {
		t.Fatalf("Failed to parse ball JSON with global --json flag: %v\nOutput: %s", err, jsonOutput)
	}

	if ball.State != session.StateInProgress {
		t.Errorf("Expected state 'in_progress', got '%s'", ball.State)
	}
}

// TestBallFinalState_Dependencies tests that dependencies are captured in final state
func TestBallFinalState_Dependencies(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	setupConfigWithTestProject(t, env)

	// Create first ball (dependency target)
	output1 := runJuggleCommand(t, env.ProjectDir, "plan",
		"First ball",
		"--non-interactive",
	)

	// Extract first ball ID
	var ball1ID string
	lines := strings.Split(output1, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Planned ball added:") {
			parts := strings.Split(line, ": ")
			if len(parts) >= 2 {
				ball1ID = strings.TrimSpace(parts[len(parts)-1])
			}
		}
	}
	if ball1ID == "" {
		t.Fatalf("Could not extract first ball ID from output: %s", output1)
	}

	// Create second ball with dependency on first
	output2 := runJuggleCommand(t, env.ProjectDir, "plan",
		"Second ball",
		"--non-interactive",
		"--depends-on", ball1ID,
	)

	// Extract second ball ID
	var ball2ID string
	lines = strings.Split(output2, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Planned ball added:") {
			parts := strings.Split(line, ": ")
			if len(parts) >= 2 {
				ball2ID = strings.TrimSpace(parts[len(parts)-1])
			}
		}
	}
	if ball2ID == "" {
		t.Fatalf("Could not extract second ball ID from output: %s", output2)
	}

	// Verify dependency is captured in final state
	jsonOutput := runJuggleCommand(t, env.ProjectDir, "show", ball2ID, "--json")

	var ball session.Ball
	if err := json.Unmarshal([]byte(jsonOutput), &ball); err != nil {
		t.Fatalf("Failed to parse ball JSON: %v\nOutput: %s", err, jsonOutput)
	}

	if len(ball.DependsOn) != 1 {
		t.Errorf("Expected 1 dependency, got %d: %v", len(ball.DependsOn), ball.DependsOn)
	} else if ball.DependsOn[0] != ball1ID {
		t.Errorf("Expected dependency on '%s', got '%s'", ball1ID, ball.DependsOn[0])
	}
}
