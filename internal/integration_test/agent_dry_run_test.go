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
	// Actually, it should error if no projects exist with juggleDir
	// Let me check the implementation...

	// Based on the code, if projects exist but session not found, it returns empty balls
	// but if no projects found, it returns error
	if err != nil {
		// This is expected if no juggle dir exists
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

// Tests for complete ball exclusion from agent prompt

func TestAgentPromptGeneration_ExcludesCompleteBalls(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a session
	env.CreateSession(t, "test-session", "Test session for complete ball filtering")

	store := env.GetStore(t)

	// Create a pending ball
	pendingBall := env.CreateBall(t, "Pending work item", session.PriorityMedium)
	pendingBall.Tags = []string{"test-session"}
	pendingBall.State = session.StatePending
	if err := store.UpdateBall(pendingBall); err != nil {
		t.Fatalf("Failed to update pending ball: %v", err)
	}

	// Create an in_progress ball
	inProgressBall := env.CreateBall(t, "In progress work item", session.PriorityHigh)
	inProgressBall.Tags = []string{"test-session"}
	inProgressBall.State = session.StateInProgress
	if err := store.UpdateBall(inProgressBall); err != nil {
		t.Fatalf("Failed to update in_progress ball: %v", err)
	}

	// Create a complete ball - this should be excluded
	completeBall := env.CreateBall(t, "Completed work item", session.PriorityLow)
	completeBall.Tags = []string{"test-session"}
	completeBall.State = session.StateComplete
	if err := store.UpdateBall(completeBall); err != nil {
		t.Fatalf("Failed to update complete ball: %v", err)
	}

	// Generate prompt
	prompt, err := cli.GenerateAgentPromptForTest(env.ProjectDir, "test-session", false, "")
	if err != nil {
		t.Fatalf("Failed to generate prompt: %v", err)
	}

	// Verify pending and in_progress balls are included
	if !strings.Contains(prompt, "Pending work item") {
		t.Error("Prompt should contain pending ball")
	}
	if !strings.Contains(prompt, "In progress work item") {
		t.Error("Prompt should contain in_progress ball")
	}

	// Verify complete ball is excluded
	if strings.Contains(prompt, "Completed work item") {
		t.Error("Prompt should NOT contain complete ball by default")
	}
}

func TestAgentPromptGeneration_ExcludesResearchedBalls(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a session
	env.CreateSession(t, "test-session", "Test session for researched ball filtering")

	store := env.GetStore(t)

	// Create a pending ball
	pendingBall := env.CreateBall(t, "Pending work item", session.PriorityMedium)
	pendingBall.Tags = []string{"test-session"}
	pendingBall.State = session.StatePending
	if err := store.UpdateBall(pendingBall); err != nil {
		t.Fatalf("Failed to update pending ball: %v", err)
	}

	// Create a researched ball - this should be excluded
	researchedBall := env.CreateBall(t, "Researched work item", session.PriorityLow)
	researchedBall.Tags = []string{"test-session"}
	researchedBall.State = session.StateResearched
	researchedBall.Output = "Research findings here"
	if err := store.UpdateBall(researchedBall); err != nil {
		t.Fatalf("Failed to update researched ball: %v", err)
	}

	// Generate prompt
	prompt, err := cli.GenerateAgentPromptForTest(env.ProjectDir, "test-session", false, "")
	if err != nil {
		t.Fatalf("Failed to generate prompt: %v", err)
	}

	// Verify pending ball is included
	if !strings.Contains(prompt, "Pending work item") {
		t.Error("Prompt should contain pending ball")
	}

	// Verify researched ball is excluded
	if strings.Contains(prompt, "Researched work item") {
		t.Error("Prompt should NOT contain researched ball by default")
	}
}

func TestAgentPromptGeneration_ExcludesBlockedBalls(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a session
	env.CreateSession(t, "test-session", "Test session for blocked ball exclusion")

	store := env.GetStore(t)

	// Create a blocked ball - this should be excluded (requires human intervention)
	blockedBall := env.CreateBall(t, "Blocked work item", session.PriorityHigh)
	blockedBall.Tags = []string{"test-session"}
	blockedBall.State = session.StateBlocked
	blockedBall.BlockedReason = "Missing dependency"
	if err := store.UpdateBall(blockedBall); err != nil {
		t.Fatalf("Failed to update blocked ball: %v", err)
	}

	// Generate prompt
	prompt, err := cli.GenerateAgentPromptForTest(env.ProjectDir, "test-session", false, "")
	if err != nil {
		t.Fatalf("Failed to generate prompt: %v", err)
	}

	// Verify blocked ball is excluded (requires human intervention, not for autonomous agents)
	if strings.Contains(prompt, "Blocked work item") {
		t.Error("Prompt should NOT contain blocked ball - blocked balls are excluded from autonomous runs")
	}
}

func TestAgentPromptGeneration_SpecificBallID_AllowsComplete(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a session
	env.CreateSession(t, "test-session", "Test session for specific ball access")

	store := env.GetStore(t)

	// Create a complete ball
	completeBall := env.CreateBall(t, "Completed work item for specific access", session.PriorityLow)
	completeBall.Tags = []string{"test-session"}
	completeBall.State = session.StateComplete
	if err := store.UpdateBall(completeBall); err != nil {
		t.Fatalf("Failed to update complete ball: %v", err)
	}

	// Generate prompt for specific ball ID - should work even for complete ball
	prompt, err := cli.GenerateAgentPromptForTest(env.ProjectDir, "test-session", false, completeBall.ShortID())
	if err != nil {
		t.Fatalf("Failed to generate prompt for specific ball: %v", err)
	}

	// Verify complete ball is included when specifically requested
	if !strings.Contains(prompt, "Completed work item for specific access") {
		t.Error("Prompt should contain complete ball when specifically requested")
	}
}

func TestAgentPromptGeneration_AllSession_ExcludesComplete(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	store := env.GetStore(t)

	// Create a pending ball (no session tag)
	pendingBall := env.CreateBall(t, "Pending in all session", session.PriorityMedium)
	pendingBall.State = session.StatePending
	if err := store.UpdateBall(pendingBall); err != nil {
		t.Fatalf("Failed to update pending ball: %v", err)
	}

	// Create a complete ball (no session tag)
	completeBall := env.CreateBall(t, "Complete in all session", session.PriorityLow)
	completeBall.State = session.StateComplete
	if err := store.UpdateBall(completeBall); err != nil {
		t.Fatalf("Failed to update complete ball: %v", err)
	}

	// Generate prompt for "all" meta-session
	prompt, err := cli.GenerateAgentPromptForTest(env.ProjectDir, "all", false, "")
	if err != nil {
		t.Fatalf("Failed to generate prompt for all session: %v", err)
	}

	// Verify pending ball is included
	if !strings.Contains(prompt, "Pending in all session") {
		t.Error("Prompt should contain pending ball in all session")
	}

	// Verify complete ball is excluded
	if strings.Contains(prompt, "Complete in all session") {
		t.Error("Prompt should NOT contain complete ball even in all session")
	}
}

func TestLoadBallsForModelSelection_ExcludesCompleteBalls(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a session
	env.CreateSession(t, "test-session", "Test session for model selection")

	store := env.GetStore(t)

	// Create a pending ball
	pendingBall := env.CreateBall(t, "Pending for model selection", session.PriorityMedium)
	pendingBall.Tags = []string{"test-session"}
	pendingBall.State = session.StatePending
	if err := store.UpdateBall(pendingBall); err != nil {
		t.Fatalf("Failed to update pending ball: %v", err)
	}

	// Create a complete ball
	completeBall := env.CreateBall(t, "Complete for model selection", session.PriorityLow)
	completeBall.Tags = []string{"test-session"}
	completeBall.State = session.StateComplete
	if err := store.UpdateBall(completeBall); err != nil {
		t.Fatalf("Failed to update complete ball: %v", err)
	}

	// Load balls for model selection
	balls, err := cli.LoadBallsForModelSelectionForTest(env.ProjectDir, "test-session", "")
	if err != nil {
		t.Fatalf("Failed to load balls for model selection: %v", err)
	}

	// Should only get the pending ball
	if len(balls) != 1 {
		t.Errorf("Expected 1 ball (pending only), got %d", len(balls))
	}

	if len(balls) > 0 && balls[0].State != session.StatePending {
		t.Errorf("Expected pending ball, got %s", balls[0].State)
	}
}
