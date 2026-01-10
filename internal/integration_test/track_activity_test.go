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

	// Run track-activity (no env var, single ball)
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

	// Run track-activity (no env var, multiple balls -> use most recent)
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


func TestTrackActivity_PriorityOrder(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Test that environment variable takes precedence over everything
	ball1 := env.CreateJugglingBall(t, "Ball 1", session.PriorityHigh, session.JuggleInAir)
	ball2 := env.CreateJugglingBall(t, "Ball 2", session.PriorityMedium, session.JuggleInAir)

	// Make ball2 most recent
	time.Sleep(10 * time.Millisecond)
	store := env.GetStore(t)
	ball2.UpdateActivity()
	store.UpdateBall(ball2)

	// Set JUGGLER_CURRENT_BALL to target ball2
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
		t.Errorf("Ball 1 should not have been updated (JUGGLER_CURRENT_BALL targets ball2): initial=%d, after=%d", ball1InitialCount, ball1AfterCount)
	}
	if ball2AfterCount <= ball2InitialCount {
		t.Errorf("Ball 2 should have been updated (via JUGGLER_CURRENT_BALL): initial=%d, after=%d", ball2InitialCount, ball2AfterCount)
	}
}
