package session

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"
)

func TestNewJuggleSession(t *testing.T) {
	session := NewJuggleSession("test-session", "Test description")

	if session.ID != "test-session" {
		t.Errorf("expected ID 'test-session', got '%s'", session.ID)
	}
	if session.Description != "Test description" {
		t.Errorf("expected Description 'Test description', got '%s'", session.Description)
	}
	if session.Context != "" {
		t.Errorf("expected empty Context, got '%s'", session.Context)
	}
	if session.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
	if session.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set")
	}
}

func TestJuggleSession_SetContext(t *testing.T) {
	session := NewJuggleSession("test", "desc")
	originalUpdatedAt := session.UpdatedAt

	// Sleep briefly to ensure time difference
	time.Sleep(10 * time.Millisecond)

	session.SetContext("New context")

	if session.Context != "New context" {
		t.Errorf("expected Context 'New context', got '%s'", session.Context)
	}
	if !session.UpdatedAt.After(originalUpdatedAt) {
		t.Error("expected UpdatedAt to be updated")
	}
}

func TestJuggleSession_SetDescription(t *testing.T) {
	session := NewJuggleSession("test", "original")
	originalUpdatedAt := session.UpdatedAt

	// Sleep briefly to ensure time difference
	time.Sleep(10 * time.Millisecond)

	session.SetDescription("updated description")

	if session.Description != "updated description" {
		t.Errorf("expected Description 'updated description', got '%s'", session.Description)
	}
	if !session.UpdatedAt.After(originalUpdatedAt) {
		t.Error("expected UpdatedAt to be updated")
	}
}

func TestSessionStore_CreateAndLoadSession(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "juggler-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewSessionStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Create session
	session, err := store.CreateSession("my-session", "My session description")
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	if session.ID != "my-session" {
		t.Errorf("expected ID 'my-session', got '%s'", session.ID)
	}

	// Verify directory was created
	sessionDir := filepath.Join(tmpDir, ".juggler", "sessions", "my-session")
	if _, err := os.Stat(sessionDir); os.IsNotExist(err) {
		t.Error("expected session directory to be created")
	}

	// Verify session.json exists
	sessionFile := filepath.Join(sessionDir, "session.json")
	if _, err := os.Stat(sessionFile); os.IsNotExist(err) {
		t.Error("expected session.json to be created")
	}

	// Verify progress.txt exists
	progressFile := filepath.Join(sessionDir, "progress.txt")
	if _, err := os.Stat(progressFile); os.IsNotExist(err) {
		t.Error("expected progress.txt to be created")
	}

	// Load session back
	loaded, err := store.LoadSession("my-session")
	if err != nil {
		t.Fatalf("failed to load session: %v", err)
	}

	if loaded.ID != session.ID {
		t.Errorf("expected ID '%s', got '%s'", session.ID, loaded.ID)
	}
	if loaded.Description != session.Description {
		t.Errorf("expected Description '%s', got '%s'", session.Description, loaded.Description)
	}
}

func TestSessionStore_CreateSession_AlreadyExists(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "juggler-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewSessionStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Create session
	_, err = store.CreateSession("my-session", "First")
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Try to create again
	_, err = store.CreateSession("my-session", "Second")
	if err == nil {
		t.Error("expected error when creating duplicate session")
	}
}

func TestSessionStore_ListSessions(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "juggler-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewSessionStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// List empty
	sessions, err := store.ListSessions()
	if err != nil {
		t.Fatalf("failed to list sessions: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions, got %d", len(sessions))
	}

	// Create sessions
	store.CreateSession("session-1", "First")
	store.CreateSession("session-2", "Second")
	store.CreateSession("session-3", "Third")

	// List again
	sessions, err = store.ListSessions()
	if err != nil {
		t.Fatalf("failed to list sessions: %v", err)
	}
	if len(sessions) != 3 {
		t.Errorf("expected 3 sessions, got %d", len(sessions))
	}
}

func TestSessionStore_UpdateSessionContext(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "juggler-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewSessionStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Create session
	_, err = store.CreateSession("my-session", "desc")
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Update context
	err = store.UpdateSessionContext("my-session", "New context content")
	if err != nil {
		t.Fatalf("failed to update context: %v", err)
	}

	// Load and verify
	session, err := store.LoadSession("my-session")
	if err != nil {
		t.Fatalf("failed to load session: %v", err)
	}

	if session.Context != "New context content" {
		t.Errorf("expected Context 'New context content', got '%s'", session.Context)
	}
}

func TestSessionStore_DeleteSession(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "juggler-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewSessionStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Create session
	_, err = store.CreateSession("my-session", "desc")
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Delete session
	err = store.DeleteSession("my-session")
	if err != nil {
		t.Fatalf("failed to delete session: %v", err)
	}

	// Verify it's gone
	_, err = store.LoadSession("my-session")
	if err == nil {
		t.Error("expected error loading deleted session")
	}

	// Verify directory is gone
	sessionDir := filepath.Join(tmpDir, ".juggler", "sessions", "my-session")
	if _, err := os.Stat(sessionDir); !os.IsNotExist(err) {
		t.Error("expected session directory to be deleted")
	}
}

func TestSessionStore_DeleteSession_NotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "juggler-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewSessionStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Try to delete non-existent session
	err = store.DeleteSession("nonexistent")
	if err == nil {
		t.Error("expected error deleting non-existent session")
	}
}

func TestSessionStore_AppendAndLoadProgress(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "juggler-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewSessionStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Create session
	_, err = store.CreateSession("my-session", "desc")
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Append progress
	err = store.AppendProgress("my-session", "First line\n")
	if err != nil {
		t.Fatalf("failed to append progress: %v", err)
	}

	err = store.AppendProgress("my-session", "Second line\n")
	if err != nil {
		t.Fatalf("failed to append progress: %v", err)
	}

	// Load progress
	progress, err := store.LoadProgress("my-session")
	if err != nil {
		t.Fatalf("failed to load progress: %v", err)
	}

	expected := "First line\nSecond line\n"
	if progress != expected {
		t.Errorf("expected progress '%s', got '%s'", expected, progress)
	}
}

func TestSessionStore_AppendProgress_SessionNotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "juggler-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewSessionStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Try to append to non-existent session
	err = store.AppendProgress("nonexistent", "content")
	if err == nil {
		t.Error("expected error appending to non-existent session")
	}
}

func TestSessionStore_LoadSession_NotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "juggler-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewSessionStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	_, err = store.LoadSession("nonexistent")
	if err == nil {
		t.Error("expected error loading non-existent session")
	}
}

func TestLoadBallsBySession(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "juggler-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a ball store
	ballStore, err := NewStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create ball store: %v", err)
	}

	// Create balls with different tags (simulating session membership)
	ball1, _ := NewBall(tmpDir, "Ball 1 - belongs to session-a", PriorityMedium)
	ball1.AddTag("session-a")
	if err := ballStore.AppendBall(ball1); err != nil {
		t.Fatalf("failed to save ball1: %v", err)
	}

	ball2, _ := NewBall(tmpDir, "Ball 2 - belongs to session-a and session-b", PriorityMedium)
	ball2.AddTag("session-a")
	ball2.AddTag("session-b")
	if err := ballStore.AppendBall(ball2); err != nil {
		t.Fatalf("failed to save ball2: %v", err)
	}

	ball3, _ := NewBall(tmpDir, "Ball 3 - belongs to session-b", PriorityMedium)
	ball3.AddTag("session-b")
	if err := ballStore.AppendBall(ball3); err != nil {
		t.Fatalf("failed to save ball3: %v", err)
	}

	ball4, _ := NewBall(tmpDir, "Ball 4 - no session", PriorityMedium)
	if err := ballStore.AppendBall(ball4); err != nil {
		t.Fatalf("failed to save ball4: %v", err)
	}

	projectPaths := []string{tmpDir}

	// Test session-a: should have ball1 and ball2
	sessionABalls, err := LoadBallsBySession(projectPaths, "session-a")
	if err != nil {
		t.Fatalf("failed to load balls for session-a: %v", err)
	}
	if len(sessionABalls) != 2 {
		t.Errorf("expected 2 balls for session-a, got %d", len(sessionABalls))
	}

	// Test session-b: should have ball2 and ball3
	sessionBBalls, err := LoadBallsBySession(projectPaths, "session-b")
	if err != nil {
		t.Fatalf("failed to load balls for session-b: %v", err)
	}
	if len(sessionBBalls) != 2 {
		t.Errorf("expected 2 balls for session-b, got %d", len(sessionBBalls))
	}

	// Test non-existent session: should return empty
	noSessionBalls, err := LoadBallsBySession(projectPaths, "session-c")
	if err != nil {
		t.Fatalf("failed to load balls for session-c: %v", err)
	}
	if len(noSessionBalls) != 0 {
		t.Errorf("expected 0 balls for session-c, got %d", len(noSessionBalls))
	}
}

func TestLoadBallsBySession_MultipleSessions(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "juggler-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a ball store
	ballStore, err := NewStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create ball store: %v", err)
	}

	// Create a ball that belongs to multiple sessions
	ball, _ := NewBall(tmpDir, "Multi-session ball", PriorityMedium)
	ball.AddTag("session-1")
	ball.AddTag("session-2")
	ball.AddTag("session-3")
	if err := ballStore.AppendBall(ball); err != nil {
		t.Fatalf("failed to save ball: %v", err)
	}

	projectPaths := []string{tmpDir}

	// Verify ball appears in all three sessions
	for _, sessionID := range []string{"session-1", "session-2", "session-3"} {
		balls, err := LoadBallsBySession(projectPaths, sessionID)
		if err != nil {
			t.Fatalf("failed to load balls for %s: %v", sessionID, err)
		}
		if len(balls) != 1 {
			t.Errorf("expected 1 ball for %s, got %d", sessionID, len(balls))
		}
		if len(balls) > 0 && balls[0].ID != ball.ID {
			t.Errorf("expected ball ID %s for %s, got %s", ball.ID, sessionID, balls[0].ID)
		}
	}
}

// TestBallIDFormat tests that new ball IDs use UUID-based format
func TestBallIDFormat(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "juggler-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a ball
	ball, err := NewBall(tmpDir, "Test ball", PriorityMedium)
	if err != nil {
		t.Fatalf("failed to create ball: %v", err)
	}

	// UUID-based format: <project>-<8-char-hex>
	// e.g., "juggler-test-a1b2c3d4"
	uuidPattern := regexp.MustCompile(`^.+-[0-9a-f]{8}$`)
	if !uuidPattern.MatchString(ball.ID) {
		t.Errorf("expected UUID-based ID format (project-8hexchars), got '%s'", ball.ID)
	}
}

// TestBallIDUniqueness tests that multiple balls get unique IDs
func TestBallIDUniqueness(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "juggler-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create multiple balls
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		ball, err := NewBall(tmpDir, "Test ball", PriorityMedium)
		if err != nil {
			t.Fatalf("failed to create ball %d: %v", i, err)
		}
		if ids[ball.ID] {
			t.Errorf("duplicate ID generated: %s", ball.ID)
		}
		ids[ball.ID] = true
	}
}

// TestLegacyBallIDCompatibility tests that legacy numeric IDs are still supported
func TestLegacyBallIDCompatibility(t *testing.T) {
	// Test ShortID works with legacy numeric format
	legacyBall := &Ball{ID: "myproject-42"}
	if legacyBall.ShortID() != "42" {
		t.Errorf("expected ShortID '42' for legacy format, got '%s'", legacyBall.ShortID())
	}

	// Test ShortID works with new UUID format
	uuidBall := &Ball{ID: "myproject-a1b2c3d4"}
	if uuidBall.ShortID() != "a1b2c3d4" {
		t.Errorf("expected ShortID 'a1b2c3d4' for UUID format, got '%s'", uuidBall.ShortID())
	}

	// Test FolderName still works
	legacyBall.WorkingDir = "/home/user/myproject"
	if legacyBall.FolderName() != "myproject" {
		t.Errorf("expected FolderName 'myproject', got '%s'", legacyBall.FolderName())
	}
}

// TestBallIDPrefixMatchesProjectDir tests that ball ID prefix matches the project directory name
func TestBallIDPrefixMatchesProjectDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "juggler-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	ball, err := NewBall(tmpDir, "Test ball", PriorityMedium)
	if err != nil {
		t.Fatalf("failed to create ball: %v", err)
	}

	// The ID should start with the base name of the temp dir
	baseName := filepath.Base(tmpDir)
	expectedPrefix := baseName + "-"
	if len(ball.ID) < len(expectedPrefix) || ball.ID[:len(expectedPrefix)] != expectedPrefix {
		t.Errorf("expected ID to start with '%s', got '%s'", expectedPrefix, ball.ID)
	}
}
