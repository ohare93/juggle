package integration_test

import (
	"strings"
	"testing"

	"github.com/ohare93/juggle/internal/agent"
	"github.com/ohare93/juggle/internal/cli"
	"github.com/ohare93/juggle/internal/session"
)

// TestAgentRefine_SessionFilter tests that balls are filtered by session tag
func TestAgentRefine_SessionFilter(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a session
	env.CreateSession(t, "test-session", "Test session for refinement")

	// Create balls: one with session tag, one without
	ball1 := env.CreateBall(t, "Ball in session", session.PriorityMedium)
	ball1.Tags = []string{"test-session"}
	store := env.GetStore(t)
	if err := store.UpdateBall(ball1); err != nil {
		t.Fatalf("Failed to update ball1: %v", err)
	}

	ball2 := env.CreateBall(t, "Ball not in session", session.PriorityMedium)
	ball2.Tags = []string{"other-session"}
	if err := store.UpdateBall(ball2); err != nil {
		t.Fatalf("Failed to update ball2: %v", err)
	}

	// Setup mock runner to capture the prompt
	mock := agent.NewMockRunner(
		&agent.RunResult{Output: "Done"},
	)
	agent.SetRunner(mock)
	defer agent.ResetRunner()

	// Load balls for refine with session filter
	balls, err := cli.LoadBallsForRefineForTest(env.ProjectDir, "test-session")
	if err != nil {
		t.Fatalf("Failed to load balls: %v", err)
	}

	// Verify only the session ball is included
	if len(balls) != 1 {
		t.Errorf("Expected 1 ball, got %d", len(balls))
	}
	if len(balls) > 0 && balls[0].Title != "Ball in session" {
		t.Errorf("Expected ball 'Ball in session', got '%s'", balls[0].Title)
	}
}

// TestAgentRefine_DefaultLoadsCurrentRepo tests default loads from current repo only
func TestAgentRefine_DefaultLoadsCurrentRepo(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create balls without session tags
	ball := env.CreateBall(t, "Test ball", session.PriorityMedium)
	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// Load balls without session filter (default behavior)
	balls, err := cli.LoadBallsForRefineForTest(env.ProjectDir, "")
	if err != nil {
		t.Fatalf("Failed to load balls: %v", err)
	}

	// Verify ball is included
	if len(balls) < 1 {
		t.Errorf("Expected at least 1 ball, got %d", len(balls))
	}
}

// TestAgentRefine_PromptContainsBalls tests that the prompt includes ball info
func TestAgentRefine_PromptContainsBalls(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a ball with acceptance criteria
	ball := env.CreateBall(t, "Feature to implement", session.PriorityHigh)
	ball.AcceptanceCriteria = []string{"First criterion", "Second criterion"}
	ball.Tags = []string{"test-feature"}
	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// Generate refine prompt
	balls := []*session.Ball{ball}
	prompt, err := cli.GenerateRefinePromptForTest(env.ProjectDir, "test-feature", balls)
	if err != nil {
		t.Fatalf("Failed to generate prompt: %v", err)
	}

	// Verify prompt contains ball info
	if !strings.Contains(prompt, "Feature to implement") {
		t.Error("Prompt missing ball intent")
	}
	if !strings.Contains(prompt, "First criterion") {
		t.Error("Prompt missing first acceptance criterion")
	}
	if !strings.Contains(prompt, "Second criterion") {
		t.Error("Prompt missing second acceptance criterion")
	}
	if !strings.Contains(prompt, ball.ID) {
		t.Error("Prompt missing ball ID")
	}
	if !strings.Contains(prompt, "high") {
		t.Error("Prompt missing priority")
	}
}

// TestAgentRefine_PromptContainsInstructions tests that the prompt includes refinement template
func TestAgentRefine_PromptContainsInstructions(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	ball := env.CreateBall(t, "Test ball", session.PriorityMedium)
	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// Generate refine prompt
	balls := []*session.Ball{ball}
	prompt, err := cli.GenerateRefinePromptForTest(env.ProjectDir, "", balls)
	if err != nil {
		t.Fatalf("Failed to generate prompt: %v", err)
	}

	// Verify prompt contains instruction sections
	if !strings.Contains(prompt, "<instructions>") {
		t.Error("Prompt missing <instructions> tag")
	}
	if !strings.Contains(prompt, "</instructions>") {
		t.Error("Prompt missing </instructions> tag")
	}
	if !strings.Contains(prompt, "<balls>") {
		t.Error("Prompt missing <balls> tag")
	}
	if !strings.Contains(prompt, "</balls>") {
		t.Error("Prompt missing </balls> tag")
	}
	// Check for refinement template content
	if !strings.Contains(prompt, "Acceptance Criteria Quality") {
		t.Error("Prompt missing refinement instructions")
	}
	if !strings.Contains(prompt, "juggle update") {
		t.Error("Prompt missing CLI command examples")
	}
}

// TestAgentRefine_UsesPlanMode tests that the command uses plan mode
func TestAgentRefine_UsesPlanMode(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a ball
	ball := env.CreateBall(t, "Test ball", session.PriorityMedium)
	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// Setup mock runner to capture options
	mock := agent.NewMockRunner(
		&agent.RunResult{Output: "Done"},
	)
	agent.SetRunner(mock)
	defer agent.ResetRunner()

	// Run the refine command via exported function
	err := cli.RunAgentRefineForTest(env.ProjectDir, "")
	if err != nil {
		t.Fatalf("Agent refine failed: %v", err)
	}

	// Verify plan mode was used
	if len(mock.Calls) == 0 {
		t.Fatal("No calls made to runner")
	}

	if mock.Calls[0].Permission != agent.PermissionPlan {
		t.Errorf("Expected Permission=PermissionPlan, got %s", mock.Calls[0].Permission)
	}
}

// TestAgentRefine_UsesInteractiveMode tests that the command uses interactive mode
func TestAgentRefine_UsesInteractiveMode(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a ball
	ball := env.CreateBall(t, "Test ball", session.PriorityMedium)
	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// Setup mock runner
	mock := agent.NewMockRunner(
		&agent.RunResult{Output: "Done"},
	)
	agent.SetRunner(mock)
	defer agent.ResetRunner()

	// Run the refine command
	err := cli.RunAgentRefineForTest(env.ProjectDir, "")
	if err != nil {
		t.Fatalf("Agent refine failed: %v", err)
	}

	// Verify interactive mode was used
	if len(mock.Calls) == 0 {
		t.Fatal("No calls made to runner")
	}

	if mock.Calls[0].Mode != agent.ModeInteractive {
		t.Errorf("Expected Mode=ModeInteractive, got %s", mock.Calls[0].Mode)
	}
}

// TestAgentRefine_ExcludesCompletedBalls tests that completed balls are excluded by default
func TestAgentRefine_ExcludesCompletedBalls(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a completed ball and a pending ball
	completedBall := env.CreateBall(t, "Completed ball", session.PriorityMedium)
	completedBall.State = session.StateComplete
	store := env.GetStore(t)
	if err := store.UpdateBall(completedBall); err != nil {
		t.Fatalf("Failed to update completed ball: %v", err)
	}

	pendingBall := env.CreateBall(t, "Pending ball", session.PriorityMedium)
	if err := store.UpdateBall(pendingBall); err != nil {
		t.Fatalf("Failed to update pending ball: %v", err)
	}

	// Load balls without session filter
	balls, err := cli.LoadBallsForRefineForTest(env.ProjectDir, "")
	if err != nil {
		t.Fatalf("Failed to load balls: %v", err)
	}

	// Verify only non-complete balls are included
	for _, ball := range balls {
		if ball.State == session.StateComplete {
			t.Errorf("Completed ball should be excluded: %s", ball.Title)
		}
	}
}

// TestAgentRefine_NoBallsMessage tests that a message is shown when no balls exist
func TestAgentRefine_NoBallsMessage(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Don't create any balls

	// Setup mock runner (shouldn't be called)
	mock := agent.NewMockRunner()
	agent.SetRunner(mock)
	defer agent.ResetRunner()

	// Run the refine command - should not error but also not call runner
	err := cli.RunAgentRefineForTest(env.ProjectDir, "")
	if err != nil {
		t.Fatalf("Agent refine failed: %v", err)
	}

	// Verify runner was NOT called (no balls to refine)
	if len(mock.Calls) != 0 {
		t.Errorf("Expected no calls to runner (no balls), got %d", len(mock.Calls))
	}
}

// TestAgentRefine_SessionIncludesCompletedBalls tests that session filter includes all balls
func TestAgentRefine_SessionIncludesCompletedBalls(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create session
	env.CreateSession(t, "test-session", "Test session")

	// Create a completed ball in the session
	completedBall := env.CreateBall(t, "Completed ball", session.PriorityMedium)
	completedBall.State = session.StateComplete
	completedBall.Tags = []string{"test-session"}
	store := env.GetStore(t)
	if err := store.UpdateBall(completedBall); err != nil {
		t.Fatalf("Failed to update completed ball: %v", err)
	}

	// Load balls with session filter
	balls, err := cli.LoadBallsForRefineForTest(env.ProjectDir, "test-session")
	if err != nil {
		t.Fatalf("Failed to load balls: %v", err)
	}

	// When filtering by session, all balls (including completed) should be included
	if len(balls) != 1 {
		t.Errorf("Expected 1 ball (session includes completed), got %d", len(balls))
	}
}

// TestAgentRefine_PromptShowsMissingACs tests that balls without ACs are flagged
func TestAgentRefine_PromptShowsMissingACs(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a ball without acceptance criteria
	ball := env.CreateBall(t, "Ball without ACs", session.PriorityMedium)
	ball.AcceptanceCriteria = []string{} // Empty ACs
	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// Generate refine prompt
	balls := []*session.Ball{ball}
	prompt, err := cli.GenerateRefinePromptForTest(env.ProjectDir, "", balls)
	if err != nil {
		t.Fatalf("Failed to generate prompt: %v", err)
	}

	// Verify prompt flags missing ACs
	if !strings.Contains(prompt, "none - needs definition") {
		t.Error("Prompt should flag missing acceptance criteria")
	}
}

// TestAgentRefine_AllMetaSession tests that "all" acts as a meta-session for all repo balls
func TestAgentRefine_AllMetaSession(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create balls with different session tags
	ball1 := env.CreateBall(t, "Ball in session A", session.PriorityMedium)
	ball1.Tags = []string{"session-a"}
	store := env.GetStore(t)
	if err := store.UpdateBall(ball1); err != nil {
		t.Fatalf("Failed to update ball1: %v", err)
	}

	ball2 := env.CreateBall(t, "Ball in session B", session.PriorityMedium)
	ball2.Tags = []string{"session-b"}
	if err := store.UpdateBall(ball2); err != nil {
		t.Fatalf("Failed to update ball2: %v", err)
	}

	ball3 := env.CreateBall(t, "Ball with no session", session.PriorityMedium)
	// No tags
	if err := store.UpdateBall(ball3); err != nil {
		t.Fatalf("Failed to update ball3: %v", err)
	}

	// Load balls with "all" as session - should get all non-complete balls
	balls, err := cli.LoadBallsForRefineForTest(env.ProjectDir, "all")
	if err != nil {
		t.Fatalf("Failed to load balls: %v", err)
	}

	// Should have all 3 balls
	if len(balls) != 3 {
		t.Errorf("Expected 3 balls with 'all' meta-session, got %d", len(balls))
	}
}

// TestAgentRefine_PromptContainsSkillDirective tests that the prompt instructs use of the juggler skill
func TestAgentRefine_PromptContainsSkillDirective(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	ball := env.CreateBall(t, "Test ball", session.PriorityMedium)
	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	// Generate refine prompt
	balls := []*session.Ball{ball}
	prompt, err := cli.GenerateRefinePromptForTest(env.ProjectDir, "", balls)
	if err != nil {
		t.Fatalf("Failed to generate prompt: %v", err)
	}

	// Verify prompt contains directive to use Skill tool with juggler
	if !strings.Contains(prompt, "Skill tool") {
		t.Error("Prompt missing 'Skill tool' reference")
	}
	if !strings.Contains(prompt, `skill="juggler"`) {
		t.Error("Prompt missing 'skill=\"juggler\"' directive")
	}
	// Verify the instructions mention juggle CLI commands (not raw JSON editing)
	if !strings.Contains(prompt, "juggle update") {
		t.Error("Prompt missing 'juggle update' command reference")
	}
	if !strings.Contains(prompt, "juggle plan") {
		t.Error("Prompt missing 'juggle plan' command reference")
	}
}

// TestAgentRefine_AllExcludesCompletedBalls tests that "all" still excludes completed balls
func TestAgentRefine_AllExcludesCompletedBalls(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a completed ball
	completedBall := env.CreateBall(t, "Completed ball", session.PriorityMedium)
	completedBall.State = session.StateComplete
	store := env.GetStore(t)
	if err := store.UpdateBall(completedBall); err != nil {
		t.Fatalf("Failed to update completed ball: %v", err)
	}

	// Create a pending ball
	pendingBall := env.CreateBall(t, "Pending ball", session.PriorityMedium)
	if err := store.UpdateBall(pendingBall); err != nil {
		t.Fatalf("Failed to update pending ball: %v", err)
	}

	// Load balls with "all" - should exclude completed
	balls, err := cli.LoadBallsForRefineForTest(env.ProjectDir, "all")
	if err != nil {
		t.Fatalf("Failed to load balls: %v", err)
	}

	// Should have only the pending ball
	if len(balls) != 1 {
		t.Errorf("Expected 1 ball (excluding completed), got %d", len(balls))
	}
	if len(balls) > 0 && balls[0].Title != "Pending ball" {
		t.Errorf("Expected 'Pending ball', got '%s'", balls[0].Title)
	}
}
