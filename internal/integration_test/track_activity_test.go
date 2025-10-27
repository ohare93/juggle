package integration_test

import (
	"os"
	"testing"
	"time"

	"github.com/ohare93/juggle/internal/cli"
	"github.com/ohare93/juggle/internal/session"
)

// runTrackActivity is a test helper that invokes track-activity command logic
func runTrackActivity(env *TestEnv) error {
	// Save current directory
	originalDir, err := os.Getwd()
	if err != nil {
		return err
	}
	defer os.Chdir(originalDir)

	// Change to project directory so track-activity uses it
	if err := os.Chdir(env.ProjectDir); err != nil {
		return err
	}

	// Get the track-activity command and execute its RunE function
	cmd := cli.GetTrackActivityCmd()
	return cmd.RunE(cmd, []string{})
}

func TestTrackActivity_EnvironmentVariable(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create 3 juggling balls with different priorities
	ball1 := env.CreateJugglingBall(t, "Ball 1 - High Priority", session.PriorityHigh, session.JuggleInAir)
	ball2 := env.CreateJugglingBall(t, "Ball 2 - Medium Priority", session.PriorityMedium, session.JuggleInAir)
	ball3 := env.CreateJugglingBall(t, "Ball 3 - Low Priority", session.PriorityLow, session.JuggleInAir)

	// Make ball3 the most recently active by updating it
	time.Sleep(10 * time.Millisecond) // Ensure distinct timestamps
	store := env.GetStore(t)
	ball3.UpdateActivity()
	store.UpdateBall(ball3)

	// Set environment variable to override and select ball2
	env.SetEnvVar(t, "JUGGLER_CURRENT_BALL", ball2.ID)

	// Get initial update counts
	ball1InitialCount := env.GetBallUpdateCount(t, ball1.ID)
	ball2InitialCount := env.GetBallUpdateCount(t, ball2.ID)
	ball3InitialCount := env.GetBallUpdateCount(t, ball3.ID)

	// Run track-activity
	if err := runTrackActivity(env); err != nil {
		t.Fatalf("track-activity failed: %v", err)
	}

	// Verify only ball2 was updated (the one specified in env var)
	ball1AfterCount := env.GetBallUpdateCount(t, ball1.ID)
	ball2AfterCount := env.GetBallUpdateCount(t, ball2.ID)
	ball3AfterCount := env.GetBallUpdateCount(t, ball3.ID)

	if ball1AfterCount != ball1InitialCount {
		t.Errorf("Ball 1 should not have been updated: initial=%d, after=%d", ball1InitialCount, ball1AfterCount)
	}
	if ball2AfterCount <= ball2InitialCount {
		t.Errorf("Ball 2 should have been updated: initial=%d, after=%d", ball2InitialCount, ball2AfterCount)
	}
	if ball3AfterCount != ball3InitialCount {
		t.Errorf("Ball 3 should not have been updated: initial=%d, after=%d", ball3InitialCount, ball3AfterCount)
	}
}

func TestTrackActivity_EnvironmentVariable_InvalidID(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create 2 juggling balls
	ball1 := env.CreateJugglingBall(t, "Ball 1", session.PriorityHigh, session.JuggleInAir)
	ball2 := env.CreateJugglingBall(t, "Ball 2", session.PriorityMedium, session.JuggleInAir)

	// Make ball1 most recent
	time.Sleep(10 * time.Millisecond)
	store := env.GetStore(t)
	ball1.UpdateActivity()
	store.UpdateBall(ball1)

	// Set environment variable to invalid ball ID - should fall through to most recent (ball1)
	env.SetEnvVar(t, "JUGGLER_CURRENT_BALL", "nonexistent-ball-99")

	ball1InitialCount := env.GetBallUpdateCount(t, ball1.ID)
	ball2InitialCount := env.GetBallUpdateCount(t, ball2.ID)

	// Run track-activity
	if err := runTrackActivity(env); err != nil {
		t.Fatalf("track-activity failed: %v", err)
	}

	// Should fall through to most recent (ball1)
	ball1AfterCount := env.GetBallUpdateCount(t, ball1.ID)
	ball2AfterCount := env.GetBallUpdateCount(t, ball2.ID)

	if ball1AfterCount <= ball1InitialCount {
		t.Errorf("Ball 1 (most recent) should have been updated: initial=%d, after=%d", ball1InitialCount, ball1AfterCount)
	}
	if ball2AfterCount != ball2InitialCount {
		t.Errorf("Ball 2 should not have been updated: initial=%d, after=%d", ball2InitialCount, ball2AfterCount)
	}
}

func TestTrackActivity_SingleBall(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create exactly one juggling ball
	ball := env.CreateJugglingBall(t, "Only Ball", session.PriorityMedium, session.JuggleInAir)

	initialCount := env.GetBallUpdateCount(t, ball.ID)

	// Run track-activity (no env var, no Zellij, single ball)
	if err := runTrackActivity(env); err != nil {
		t.Fatalf("track-activity failed: %v", err)
	}

	afterCount := env.GetBallUpdateCount(t, ball.ID)

	if afterCount <= initialCount {
		t.Errorf("Single ball should have been updated: initial=%d, after=%d", initialCount, afterCount)
	}
}

func TestTrackActivity_MostRecentFallback(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create 3 juggling balls
	ball1 := env.CreateJugglingBall(t, "Ball 1", session.PriorityLow, session.JuggleInAir)
	ball2 := env.CreateJugglingBall(t, "Ball 2", session.PriorityMedium, session.JuggleInAir)
	ball3 := env.CreateJugglingBall(t, "Ball 3", session.PriorityHigh, session.JuggleInAir)

	// Update ball2 to be most recent
	store := env.GetStore(t)
	time.Sleep(10 * time.Millisecond)
	ball2.UpdateActivity()
	store.UpdateBall(ball2)

	ball1InitialCount := env.GetBallUpdateCount(t, ball1.ID)
	ball2InitialCount := env.GetBallUpdateCount(t, ball2.ID)
	ball3InitialCount := env.GetBallUpdateCount(t, ball3.ID)

	// Run track-activity (no env var, no Zellij, multiple balls -> use most recent)
	if err := runTrackActivity(env); err != nil {
		t.Fatalf("track-activity failed: %v", err)
	}

	ball1AfterCount := env.GetBallUpdateCount(t, ball1.ID)
	ball2AfterCount := env.GetBallUpdateCount(t, ball2.ID)
	ball3AfterCount := env.GetBallUpdateCount(t, ball3.ID)

	if ball1AfterCount != ball1InitialCount {
		t.Errorf("Ball 1 should not have been updated: initial=%d, after=%d", ball1InitialCount, ball1AfterCount)
	}
	if ball2AfterCount <= ball2InitialCount {
		t.Errorf("Ball 2 (most recent) should have been updated: initial=%d, after=%d", ball2InitialCount, ball2AfterCount)
	}
	if ball3AfterCount != ball3InitialCount {
		t.Errorf("Ball 3 should not have been updated: initial=%d, after=%d", ball3InitialCount, ball3AfterCount)
	}
}

func TestTrackActivity_NoJugglingBalls(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a ball but keep it in ready state (not juggling)
	sess := env.CreateSession(t, "Ready ball", session.PriorityMedium)
	env.AssertActiveState(t, sess.ID, session.ActiveReady)

	// Run track-activity - should silently succeed with no juggling balls
	if err := runTrackActivity(env); err != nil {
		t.Fatalf("track-activity should silently succeed with no juggling balls: %v", err)
	}

	// Verify the ready ball was not modified
	afterState := env.AssertSessionExists(t, sess.ID)
	if afterState.UpdateCount != 0 {
		t.Errorf("Ready ball should not have been updated, but UpdateCount is %d", afterState.UpdateCount)
	}
}

func TestTrackActivity_NoBallsAtAll(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Don't create any balls

	// Run track-activity - should silently succeed with no balls
	if err := runTrackActivity(env); err != nil {
		t.Fatalf("track-activity should silently succeed with no balls: %v", err)
	}

	// Verify no balls were created
	store := env.GetStore(t)
	balls, err := store.LoadBalls()
	if err != nil {
		t.Fatalf("Failed to load balls: %v", err)
	}
	if len(balls) != 0 {
		t.Errorf("Expected 0 balls, got %d", len(balls))
	}
}


func TestTrackActivity_ZellijSessionAndTab(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create balls with different Zellij session/tab info
	ball1 := env.CreateBallWithZellij(t, "Ball in session1/tab1", session.PriorityMedium, "session1", "tab1")
	ball2 := env.CreateBallWithZellij(t, "Ball in session1/tab2", session.PriorityMedium, "session1", "tab2")
	ball3 := env.CreateBallWithZellij(t, "Ball in session2/tab1", session.PriorityMedium, "session2", "tab1")

	// Set Zellij environment to match ball2 (session1/tab2)
	env.SetEnvVar(t, "ZELLIJ_SESSION_NAME", "session1")
	// Note: We can't easily mock zellij command output for tab name in integration tests
	// So we'll test the session-only matching fallback instead

	ball1InitialCount := env.GetBallUpdateCount(t, ball1.ID)
	ball2InitialCount := env.GetBallUpdateCount(t, ball2.ID)
	ball3InitialCount := env.GetBallUpdateCount(t, ball3.ID)

	// Run track-activity
	// With ZELLIJ_SESSION_NAME=session1, it should match ball1 or ball2 (both in session1)
	// Since we can't control tab detection in tests, it will match the first one it finds in session1
	if err := runTrackActivity(env); err != nil {
		t.Fatalf("track-activity failed: %v", err)
	}

	ball1AfterCount := env.GetBallUpdateCount(t, ball1.ID)
	ball2AfterCount := env.GetBallUpdateCount(t, ball2.ID)
	ball3AfterCount := env.GetBallUpdateCount(t, ball3.ID)

	// Either ball1 or ball2 should be updated (both are in session1)
	ball1Updated := ball1AfterCount > ball1InitialCount
	ball2Updated := ball2AfterCount > ball2InitialCount

	if !ball1Updated && !ball2Updated {
		t.Errorf("Expected ball1 or ball2 to be updated (both in session1), but neither was updated")
	}

	// ball3 should definitely NOT be updated (different session)
	if ball3AfterCount != ball3InitialCount {
		t.Errorf("Ball 3 (session2) should not have been updated: initial=%d, after=%d", ball3InitialCount, ball3AfterCount)
	}
}

func TestTrackActivity_ZellijSession_NoMatchingBall(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create balls with Zellij info
	ball1 := env.CreateBallWithZellij(t, "Ball in session1", session.PriorityMedium, "session1", "tab1")
	ball2 := env.CreateBallWithZellij(t, "Ball in session2", session.PriorityMedium, "session2", "tab2")

	// Make ball2 most recent
	time.Sleep(10 * time.Millisecond)
	store := env.GetStore(t)
	ball2.UpdateActivity()
	store.UpdateBall(ball2)

	// Set Zellij environment to session3 (no matching ball)
	env.SetEnvVar(t, "ZELLIJ_SESSION_NAME", "session3")

	ball1InitialCount := env.GetBallUpdateCount(t, ball1.ID)
	ball2InitialCount := env.GetBallUpdateCount(t, ball2.ID)

	// Run track-activity - should fall through to most recent (ball2)
	if err := runTrackActivity(env); err != nil {
		t.Fatalf("track-activity failed: %v", err)
	}

	ball1AfterCount := env.GetBallUpdateCount(t, ball1.ID)
	ball2AfterCount := env.GetBallUpdateCount(t, ball2.ID)

	if ball1AfterCount != ball1InitialCount {
		t.Errorf("Ball 1 should not have been updated: initial=%d, after=%d", ball1InitialCount, ball1AfterCount)
	}
	if ball2AfterCount <= ball2InitialCount {
		t.Errorf("Ball 2 (most recent) should have been updated after Zellij match failed: initial=%d, after=%d", ball2InitialCount, ball2AfterCount)
	}
}

func TestTrackActivity_PriorityOrder(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Test that environment variable takes precedence over everything
	ball1 := env.CreateBallWithZellij(t, "Ball with Zellij", session.PriorityHigh, "session1", "tab1")
	ball2 := env.CreateJugglingBall(t, "Ball without Zellij", session.PriorityMedium, session.JuggleInAir)

	// Make ball2 most recent
	time.Sleep(10 * time.Millisecond)
	store := env.GetStore(t)
	ball2.UpdateActivity()
	store.UpdateBall(ball2)

	// Set both Zellij env (which would match ball1) AND JUGGLER_CURRENT_BALL (which targets ball2)
	env.SetEnvVar(t, "ZELLIJ_SESSION_NAME", "session1")
	env.SetEnvVar(t, "JUGGLER_CURRENT_BALL", ball2.ID)

	ball1InitialCount := env.GetBallUpdateCount(t, ball1.ID)
	ball2InitialCount := env.GetBallUpdateCount(t, ball2.ID)

	// Run track-activity - env var should win
	if err := runTrackActivity(env); err != nil {
		t.Fatalf("track-activity failed: %v", err)
	}

	ball1AfterCount := env.GetBallUpdateCount(t, ball1.ID)
	ball2AfterCount := env.GetBallUpdateCount(t, ball2.ID)

	// JUGGLER_CURRENT_BALL should take precedence, so ball2 should be updated
	if ball1AfterCount != ball1InitialCount {
		t.Errorf("Ball 1 should not have been updated (JUGGLER_CURRENT_BALL overrides Zellij): initial=%d, after=%d", ball1InitialCount, ball1AfterCount)
	}
	if ball2AfterCount <= ball2InitialCount {
		t.Errorf("Ball 2 should have been updated (via JUGGLER_CURRENT_BALL): initial=%d, after=%d", ball2InitialCount, ball2AfterCount)
	}
}
