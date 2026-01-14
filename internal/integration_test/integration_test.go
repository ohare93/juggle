package integration_test

import (
	"testing"

	"github.com/ohare93/juggle/internal/session"
)

func TestBasicSessionLifecycle(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a new session
	sess := env.CreateBall(t, "Test integration feature", session.PriorityHigh)

	// Verify it exists
	env.AssertBallExists(t, sess.ID)
	env.AssertState(t, sess.ID, session.StatePending)
}

func TestMultipleSessions(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create multiple sessions
	sess1 := env.CreateBall(t, "First session", session.PriorityHigh)
	sess2 := env.CreateBall(t, "Second session", session.PriorityMedium)
	sess3 := env.CreateBall(t, "Third session", session.PriorityLow)

	// Verify all exist
	env.AssertBallExists(t, sess1.ID)
	env.AssertBallExists(t, sess2.ID)
	env.AssertBallExists(t, sess3.ID)

	// Get all sessions
	store := env.GetStore(t)
	sessions, err := store.LoadBalls()
	if err != nil {
		t.Fatalf("Failed to load all sessions: %v", err)
	}

	if len(sessions) != 3 {
		t.Fatalf("Expected 3 sessions, got %d", len(sessions))
	}
}

func TestAcceptanceCriteriaManagement(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	sess := env.CreateBall(t, "Session with acceptance criteria", session.PriorityHigh)
	store := env.GetStore(t)

	// Set acceptance criteria
	sess.SetAcceptanceCriteria([]string{"First criterion", "Second criterion"})

	if err := store.UpdateBall(sess); err != nil {
		t.Fatalf("Failed to save session with acceptance criteria: %v", err)
	}

	// Retrieve and verify
	retrieved, err := store.GetBallByID(sess.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve session: %v", err)
	}

	if len(retrieved.AcceptanceCriteria) != 2 {
		t.Fatalf("Expected 2 acceptance criteria, got %d", len(retrieved.AcceptanceCriteria))
	}
}

func TestTagManagement(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	sess := env.CreateBall(t, "Session with tags", session.PriorityMedium)
	store := env.GetStore(t)

	// Add tags
	sess.AddTag("feature")
	sess.AddTag("urgent")
	sess.AddTag("backend")

	if err := store.UpdateBall(sess); err != nil {
		t.Fatalf("Failed to save session with tags: %v", err)
	}

	// Verify tags were saved
	retrieved, err := store.GetBallByID(sess.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve session: %v", err)
	}

	if len(retrieved.Tags) != 3 {
		t.Fatalf("Expected 3 tags, got %d", len(retrieved.Tags))
	}
}

func TestConfigOperations(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Load config (should create default if not exists)
	config, err := session.LoadConfigWithOptions(session.ConfigOptions{
		ConfigHome:     env.ConfigHome,
		JuggleDirName: ".juggle",
	})
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Modify and save
	config.SearchPaths = []string{"/test/path1", "/test/path2"}
	if err := config.SaveWithOptions(session.ConfigOptions{
		ConfigHome:     env.ConfigHome,
		JuggleDirName: ".juggle",
	}); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Reload and verify
	reloaded, err := session.LoadConfigWithOptions(session.ConfigOptions{
		ConfigHome:     env.ConfigHome,
		JuggleDirName: ".juggle",
	})
	if err != nil {
		t.Fatalf("Failed to reload config: %v", err)
	}

	if len(reloaded.SearchPaths) != 2 {
		t.Fatalf("Expected 2 search paths, got %d", len(reloaded.SearchPaths))
	}
}

func TestShortID(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	sess := env.CreateBall(t, "Short ID test", session.PriorityMedium)
	store := env.GetStore(t)

	// Test retrieving by short ID
	shortID := sess.ShortID()
	retrieved, err := store.GetBallByShortID(shortID)
	if err != nil {
		t.Fatalf("Failed to get session by short ID %s: %v", shortID, err)
	}

	if retrieved.ID != sess.ID {
		t.Fatalf("Expected session ID %s, got %s", sess.ID, retrieved.ID)
	}
}

func TestPriorityWeight(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	priorities := []session.Priority{
		session.PriorityLow,
		session.PriorityMedium,
		session.PriorityHigh,
		session.PriorityUrgent,
	}

	expectedWeights := []int{1, 2, 3, 4}

	for i, priority := range priorities {
		sess := env.CreateBall(t, "Priority test", priority)
		weight := sess.PriorityWeight()

		if weight != expectedWeights[i] {
			t.Fatalf("Expected weight %d for priority %s, got %d",
				expectedWeights[i], priority, weight)
		}
	}
}

func TestDeleteBall(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create multiple sessions
	sess1 := env.CreateBall(t, "Session to keep", session.PriorityHigh)
	sess2 := env.CreateBall(t, "Session to delete", session.PriorityMedium)
	sess3 := env.CreateBall(t, "Another session to keep", session.PriorityLow)

	// Verify all exist
	env.AssertBallExists(t, sess1.ID)
	env.AssertBallExists(t, sess2.ID)
	env.AssertBallExists(t, sess3.ID)

	store := env.GetStore(t)

	// Delete the middle session
	if err := store.DeleteBall(sess2.ID); err != nil {
		t.Fatalf("Failed to delete ball: %v", err)
	}

	// Verify sess2 is gone
	_, err := store.GetBallByID(sess2.ID)
	if err == nil {
		t.Fatalf("Expected error when getting deleted ball, got nil")
	}

	// Verify sess1 and sess3 still exist
	env.AssertBallExists(t, sess1.ID)
	env.AssertBallExists(t, sess3.ID)

	// Verify only 2 sessions remain
	sessions, err := store.LoadBalls()
	if err != nil {
		t.Fatalf("Failed to load all sessions: %v", err)
	}

	if len(sessions) != 2 {
		t.Fatalf("Expected 2 sessions after deletion, got %d", len(sessions))
	}
}

func TestDeleteNonExistentBall(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	sess := env.CreateBall(t, "Test session", session.PriorityMedium)
	store := env.GetStore(t)

	// Delete should succeed even if ball doesn't exist (it's already gone)
	if err := store.DeleteBall("non-existent-id"); err != nil {
		t.Fatalf("Expected no error when deleting non-existent ball, got: %v", err)
	}

	// Original session should still exist
	env.AssertBallExists(t, sess.ID)
}

func TestPendingBallStateTransition(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a session in pending state
	sess := env.CreateBall(t, "Test pending to in_progress transition", session.PriorityMedium)

	// Verify initial state is pending
	env.AssertState(t, sess.ID, session.StatePending)

	// Start the ball (transition to in_progress)
	sess.Start()

	store := env.GetStore(t)
	if err := store.UpdateBall(sess); err != nil {
		t.Fatalf("Failed to save session: %v", err)
	}

	// Verify final state is in_progress
	env.AssertState(t, sess.ID, session.StateInProgress)
}

