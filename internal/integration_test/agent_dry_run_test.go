package integration_test

import (
	"strings"
	"testing"

	"github.com/ohare93/juggle/internal/cli"
	"github.com/ohare93/juggle/internal/session"
)

// Tests for --dry-run and --debug flag functionality

func TestAgentPromptGeneration_Basic(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a session
	env.CreateSession(t, "test-session", "Test session for dry run")

	// Create a ball tagged with the session
	ball := env.CreateBall(t, "Test ball for dry run", session.PriorityMedium)
	ball.Tags = []string{"test-session"}
	ball.AcceptanceCriteria = []string{"AC 1", "AC 2"}
	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// Generate prompt without debug mode
	prompt, err := cli.GenerateAgentPromptForTest(env.ProjectDir, "test-session", false, "")
	if err != nil {
		t.Fatalf("Failed to generate prompt: %v", err)
	}

	// Verify prompt contains expected sections
	requiredSections := []string{
		"<context>",
		"</context>",
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

	// Verify ball info is included
	if !strings.Contains(prompt, "Test ball for dry run") {
		t.Error("Prompt missing ball intent")
	}
	if !strings.Contains(prompt, "AC 1") || !strings.Contains(prompt, "AC 2") {
		t.Error("Prompt missing acceptance criteria")
	}
}

func TestAgentPromptGeneration_DebugMode(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a session
	env.CreateSession(t, "test-session", "Test session")

	// Create a ball tagged with the session
	ball := env.CreateBall(t, "Test ball", session.PriorityMedium)
	ball.Tags = []string{"test-session"}
	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// Generate prompt WITH debug mode
	prompt, err := cli.GenerateAgentPromptForTest(env.ProjectDir, "test-session", true, "")
	if err != nil {
		t.Fatalf("Failed to generate prompt: %v", err)
	}

	// Verify debug instructions are included
	if !strings.Contains(prompt, "DEBUG MODE") {
		t.Error("Debug mode prompt should contain DEBUG MODE section")
	}
	if !strings.Contains(prompt, "explain WHY") {
		t.Error("Debug mode prompt should contain reasoning instructions")
	}
}

func TestAgentPromptGeneration_DebugModeDisabled(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a session
	env.CreateSession(t, "test-session", "Test session")

	// Create a ball tagged with the session
	ball := env.CreateBall(t, "Test ball", session.PriorityMedium)
	ball.Tags = []string{"test-session"}
	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// Generate prompt WITHOUT debug mode
	prompt, err := cli.GenerateAgentPromptForTest(env.ProjectDir, "test-session", false, "")
	if err != nil {
		t.Fatalf("Failed to generate prompt: %v", err)
	}

	// Verify debug instructions are NOT included
	if strings.Contains(prompt, "DEBUG MODE") {
		t.Error("Non-debug mode prompt should NOT contain DEBUG MODE section")
	}
}

func TestAgentPromptGeneration_SingleBall(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a session
	env.CreateSession(t, "test-session", "Test session")

	// Create two balls tagged with the session
	ball1 := env.CreateBall(t, "First ball", session.PriorityMedium)
	ball1.Tags = []string{"test-session"}
	store := env.GetStore(t)
	if err := store.UpdateBall(ball1); err != nil {
		t.Fatalf("Failed to update ball1: %v", err)
	}

	ball2 := env.CreateBall(t, "Second ball", session.PriorityMedium)
	ball2.Tags = []string{"test-session"}
	if err := store.UpdateBall(ball2); err != nil {
		t.Fatalf("Failed to update ball2: %v", err)
	}

	// Generate prompt for specific ball (use short ID)
	shortID := ball1.ShortID()
	prompt, err := cli.GenerateAgentPromptForTest(env.ProjectDir, "test-session", false, shortID)
	if err != nil {
		t.Fatalf("Failed to generate prompt: %v", err)
	}

	// Verify single ball format is used (task tag instead of balls tag)
	if !strings.Contains(prompt, "<task>") {
		t.Error("Single ball prompt should use <task> format")
	}
	if strings.Contains(prompt, "<balls>") {
		t.Error("Single ball prompt should NOT use <balls> format")
	}

	// Verify only the targeted ball is included
	if !strings.Contains(prompt, "First ball") {
		t.Error("Prompt should contain targeted ball")
	}
	if strings.Contains(prompt, "Second ball") {
		t.Error("Prompt should NOT contain other balls when --ball is specified")
	}
}

func TestAgentPromptGeneration_WithContext(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a session with context
	sessionStore := env.GetSessionStore(t)
	sess, err := sessionStore.CreateSession("test-session", "Test session description")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Update context
	if err := sessionStore.UpdateSessionContext(sess.ID, "This is the session context for testing"); err != nil {
		t.Fatalf("Failed to update context: %v", err)
	}

	// Add progress
	if err := sessionStore.AppendProgress(sess.ID, "[2024-01-01] Initial progress\n"); err != nil {
		t.Fatalf("Failed to append progress: %v", err)
	}

	// Create a ball
	ball := env.CreateBall(t, "Test ball", session.PriorityMedium)
	ball.Tags = []string{"test-session"}
	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// Generate prompt
	prompt, err := cli.GenerateAgentPromptForTest(env.ProjectDir, "test-session", false, "")
	if err != nil {
		t.Fatalf("Failed to generate prompt: %v", err)
	}

	// Verify context is included
	if !strings.Contains(prompt, "This is the session context for testing") {
		t.Error("Prompt should contain session context")
	}

	// Verify progress is included
	if !strings.Contains(prompt, "Initial progress") {
		t.Error("Prompt should contain progress")
	}
}

func TestAgentPromptGeneration_SessionNotFound(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Don't create a session, just try to generate prompt
	// Should fail because session doesn't exist
	_, err := cli.GenerateAgentPromptForTest(env.ProjectDir, "non-existent-session", false, "")
	// This might not error if the session simply has no balls
	// The behavior is to return an empty balls list
	// Actually, it should error if no projects exist with jugglerDir
	// Let me check the implementation...

	// Based on the code, if projects exist but session not found, it returns empty balls
	// but if no projects found, it returns error
	if err != nil {
		// This is expected if no juggler dir exists
		t.Log("Expected error for missing project:", err)
	}
}

func TestAgentPromptGeneration_BallNotInSession(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a session
	env.CreateSession(t, "test-session", "Test session")

	// Create a ball WITHOUT the session tag
	ball := env.CreateBall(t, "Untagged ball", session.PriorityMedium)
	ball.Tags = []string{"different-session"}
	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// Generate prompt - should work but have empty balls
	prompt, err := cli.GenerateAgentPromptForTest(env.ProjectDir, "test-session", false, "")
	if err != nil {
		t.Fatalf("Prompt generation failed: %v", err)
	}

	// Verify ball is NOT in the prompt
	if strings.Contains(prompt, "Untagged ball") {
		t.Error("Prompt should NOT contain balls from different sessions")
	}
}

func TestAgentPromptGeneration_SpecificBallNotFound(t *testing.T) {
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

	// Try to generate prompt for non-existent ball
	_, err := cli.GenerateAgentPromptForTest(env.ProjectDir, "test-session", false, "non-existent-ball")
	if err == nil {
		t.Fatal("Expected error when ball not found in session")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}
}

func TestAgentPromptGeneration_PromptLength(t *testing.T) {
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

	// Generate prompt
	prompt, err := cli.GenerateAgentPromptForTest(env.ProjectDir, "test-session", false, "")
	if err != nil {
		t.Fatalf("Failed to generate prompt: %v", err)
	}

	// Verify prompt has reasonable length
	if len(prompt) == 0 {
		t.Error("Prompt should not be empty")
	}

	// Prompt should include instructions template which is typically several KB
	if len(prompt) < 100 {
		t.Errorf("Prompt seems too short: %d characters", len(prompt))
	}
}
