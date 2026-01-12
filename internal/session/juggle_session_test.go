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

func TestJuggleSession_SetDefaultModel(t *testing.T) {
	session := NewJuggleSession("test", "desc")
	originalUpdatedAt := session.UpdatedAt

	// Sleep briefly to ensure time difference
	time.Sleep(10 * time.Millisecond)

	session.SetDefaultModel(ModelSizeMedium)

	if session.DefaultModel != ModelSizeMedium {
		t.Errorf("expected DefaultModel 'medium', got '%s'", session.DefaultModel)
	}
	if !session.UpdatedAt.After(originalUpdatedAt) {
		t.Error("expected UpdatedAt to be updated")
	}
}

func TestValidateModelSize(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"", true},        // Blank is valid
		{"small", true},
		{"medium", true},
		{"large", true},
		{"invalid", false},
		{"SMALL", false},  // Case sensitive
		{"opus", false},   // Model name, not size
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := ValidateModelSize(tt.input); got != tt.expected {
				t.Errorf("ValidateModelSize(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestBall_SetModelSize(t *testing.T) {
	ball := &Ball{
		ID:       "test-1",
		Title:    "Test ball",
		Priority: PriorityMedium,
		State:    StatePending,
	}

	ball.SetModelSize(ModelSizeSmall)

	if ball.ModelSize != ModelSizeSmall {
		t.Errorf("expected ModelSize 'small', got '%s'", ball.ModelSize)
	}
}

func TestBall_ModelSize_JSON(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "juggler-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Create ball with model size
	ball, _ := NewBall(tmpDir, "Test ball with model", PriorityMedium)
	ball.SetModelSize(ModelSizeLarge)

	if err := store.AppendBall(ball); err != nil {
		t.Fatalf("failed to save ball: %v", err)
	}

	// Load balls back
	balls, err := store.LoadBalls()
	if err != nil {
		t.Fatalf("failed to load balls: %v", err)
	}

	if len(balls) != 1 {
		t.Fatalf("expected 1 ball, got %d", len(balls))
	}

	if balls[0].ModelSize != ModelSizeLarge {
		t.Errorf("expected ModelSize 'large' after reload, got '%s'", balls[0].ModelSize)
	}
}

func TestBall_ContextAndTitle_JSON(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "juggler-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Create ball with context and title
	ball, _ := NewBall(tmpDir, "Test ball title", PriorityMedium)
	ball.Context = "This is the context for the ball"

	if err := store.AppendBall(ball); err != nil {
		t.Fatalf("failed to save ball: %v", err)
	}

	// Load balls back
	balls, err := store.LoadBalls()
	if err != nil {
		t.Fatalf("failed to load balls: %v", err)
	}

	if len(balls) != 1 {
		t.Fatalf("expected 1 ball, got %d", len(balls))
	}

	if balls[0].Title != "Test ball title" {
		t.Errorf("expected Title 'Test ball title' after reload, got '%s'", balls[0].Title)
	}

	if balls[0].Context != "This is the context for the ball" {
		t.Errorf("expected Context 'This is the context for the ball' after reload, got '%s'", balls[0].Context)
	}
}

func TestBall_IntentToTitle_Migration(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "juggler-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Manually write a legacy ball with "intent" field instead of "title"
	legacyJSON := `{"id":"test-legacy-1","intent":"Legacy intent value","priority":"medium","state":"pending","started_at":"2024-01-01T00:00:00Z","last_activity":"2024-01-01T00:00:00Z","update_count":0}`
	ballsPath := filepath.Join(tmpDir, ".juggler", "balls.jsonl")
	if err := os.WriteFile(ballsPath, []byte(legacyJSON+"\n"), 0644); err != nil {
		t.Fatalf("failed to write legacy ball: %v", err)
	}

	// Load balls - should migrate intent to title
	balls, err := store.LoadBalls()
	if err != nil {
		t.Fatalf("failed to load balls: %v", err)
	}

	if len(balls) != 1 {
		t.Fatalf("expected 1 ball, got %d", len(balls))
	}

	if balls[0].Title != "Legacy intent value" {
		t.Errorf("expected Title 'Legacy intent value' after migration, got '%s'", balls[0].Title)
	}

	// Context should default to empty for migrated balls
	if balls[0].Context != "" {
		t.Errorf("expected Context to be empty for migrated ball, got '%s'", balls[0].Context)
	}
}

func TestBall_MigrationPreservesTitle(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "juggler-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Write a ball that has both intent and title - title should take precedence
	mixedJSON := `{"id":"test-mixed-1","intent":"Old intent","title":"New title","priority":"medium","state":"pending","started_at":"2024-01-01T00:00:00Z","last_activity":"2024-01-01T00:00:00Z","update_count":0}`
	ballsPath := filepath.Join(tmpDir, ".juggler", "balls.jsonl")
	if err := os.WriteFile(ballsPath, []byte(mixedJSON+"\n"), 0644); err != nil {
		t.Fatalf("failed to write mixed ball: %v", err)
	}

	// Load balls - title should be preferred over intent
	balls, err := store.LoadBalls()
	if err != nil {
		t.Fatalf("failed to load balls: %v", err)
	}

	if len(balls) != 1 {
		t.Fatalf("expected 1 ball, got %d", len(balls))
	}

	// Title should be the new value, not the old intent
	if balls[0].Title != "New title" {
		t.Errorf("expected Title 'New title' to be preserved, got '%s'", balls[0].Title)
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

// TestSessionStore_AppendProgress_AllMetaSession tests that _all virtual session
// creates directory and works for progress logging
func TestSessionStore_AppendProgress_AllMetaSession(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "juggler-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewSessionStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Append progress to "_all" virtual session - should work without creating session.json
	err = store.AppendProgress("_all", "First progress entry\n")
	if err != nil {
		t.Fatalf("failed to append progress to _all: %v", err)
	}

	// Verify directory was created
	allDir := filepath.Join(tmpDir, ".juggler", "sessions", "_all")
	if _, err := os.Stat(allDir); os.IsNotExist(err) {
		t.Error("expected _all session directory to be created")
	}

	// Verify progress.txt exists
	progressPath := filepath.Join(allDir, "progress.txt")
	if _, err := os.Stat(progressPath); os.IsNotExist(err) {
		t.Error("expected progress.txt to be created in _all directory")
	}

	// Append more progress
	err = store.AppendProgress("_all", "Second progress entry\n")
	if err != nil {
		t.Fatalf("failed to append second progress to _all: %v", err)
	}

	// Load progress
	progress, err := store.LoadProgress("_all")
	if err != nil {
		t.Fatalf("failed to load progress from _all: %v", err)
	}

	expected := "First progress entry\nSecond progress entry\n"
	if progress != expected {
		t.Errorf("expected progress '%s', got '%s'", expected, progress)
	}
}

// TestSessionStore_LoadProgress_AllMetaSession_Empty tests that loading from
// non-existent _all returns empty string (not error)
func TestSessionStore_LoadProgress_AllMetaSession_Empty(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "juggler-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewSessionStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Load progress from _all that doesn't exist yet - should return empty, not error
	progress, err := store.LoadProgress("_all")
	if err != nil {
		t.Fatalf("loading _all progress before it exists should not error: %v", err)
	}

	if progress != "" {
		t.Errorf("expected empty progress for non-existent _all, got '%s'", progress)
	}
}

// TestJuggleSession_SetAcceptanceCriteria tests setting session acceptance criteria
func TestJuggleSession_SetAcceptanceCriteria(t *testing.T) {
	session := NewJuggleSession("test", "desc")
	originalUpdatedAt := session.UpdatedAt

	// Sleep briefly to ensure time difference
	time.Sleep(10 * time.Millisecond)

	criteria := []string{"Tests pass", "Build succeeds"}
	session.SetAcceptanceCriteria(criteria)

	if len(session.AcceptanceCriteria) != 2 {
		t.Errorf("expected 2 acceptance criteria, got %d", len(session.AcceptanceCriteria))
	}
	if session.AcceptanceCriteria[0] != "Tests pass" {
		t.Errorf("expected first criterion 'Tests pass', got '%s'", session.AcceptanceCriteria[0])
	}
	if !session.UpdatedAt.After(originalUpdatedAt) {
		t.Error("expected UpdatedAt to be updated")
	}
}

// TestJuggleSession_AddAcceptanceCriterion tests adding individual criteria
func TestJuggleSession_AddAcceptanceCriterion(t *testing.T) {
	session := NewJuggleSession("test", "desc")

	session.AddAcceptanceCriterion("Tests pass")
	session.AddAcceptanceCriterion("Build succeeds")

	if len(session.AcceptanceCriteria) != 2 {
		t.Errorf("expected 2 acceptance criteria, got %d", len(session.AcceptanceCriteria))
	}
	if session.AcceptanceCriteria[1] != "Build succeeds" {
		t.Errorf("expected second criterion 'Build succeeds', got '%s'", session.AcceptanceCriteria[1])
	}
}

// TestJuggleSession_HasAcceptanceCriteria tests the HasAcceptanceCriteria method
func TestJuggleSession_HasAcceptanceCriteria(t *testing.T) {
	session := NewJuggleSession("test", "desc")

	if session.HasAcceptanceCriteria() {
		t.Error("expected HasAcceptanceCriteria to return false for empty criteria")
	}

	session.AddAcceptanceCriterion("Test criterion")

	if !session.HasAcceptanceCriteria() {
		t.Error("expected HasAcceptanceCriteria to return true after adding criterion")
	}
}

// TestSessionStore_UpdateSessionAcceptanceCriteria tests updating session ACs
func TestSessionStore_UpdateSessionAcceptanceCriteria(t *testing.T) {
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

	// Update acceptance criteria
	criteria := []string{"Tests pass", "Build succeeds", "Documentation updated"}
	err = store.UpdateSessionAcceptanceCriteria("my-session", criteria)
	if err != nil {
		t.Fatalf("failed to update acceptance criteria: %v", err)
	}

	// Load and verify
	session, err := store.LoadSession("my-session")
	if err != nil {
		t.Fatalf("failed to load session: %v", err)
	}

	if len(session.AcceptanceCriteria) != 3 {
		t.Errorf("expected 3 acceptance criteria, got %d", len(session.AcceptanceCriteria))
	}
	if session.AcceptanceCriteria[0] != "Tests pass" {
		t.Errorf("expected first criterion 'Tests pass', got '%s'", session.AcceptanceCriteria[0])
	}
}

// TestSessionStore_UpdateSessionAcceptanceCriteria_NotFound tests error handling
func TestSessionStore_UpdateSessionAcceptanceCriteria_NotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "juggler-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewSessionStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Try to update non-existent session
	err = store.UpdateSessionAcceptanceCriteria("nonexistent", []string{"criterion"})
	if err == nil {
		t.Error("expected error updating non-existent session")
	}
}

// TestSessionStore_UpdateSessionDefaultModel tests updating default model
func TestSessionStore_UpdateSessionDefaultModel(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "juggler-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewSessionStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Create a session
	_, err = store.CreateSession("test-model", "Test session")
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Update default model
	err = store.UpdateSessionDefaultModel("test-model", ModelSizeLarge)
	if err != nil {
		t.Fatalf("failed to update default model: %v", err)
	}

	// Verify the update
	session, err := store.LoadSession("test-model")
	if err != nil {
		t.Fatalf("failed to load session: %v", err)
	}

	if session.DefaultModel != ModelSizeLarge {
		t.Errorf("expected default model 'large', got '%s'", session.DefaultModel)
	}
}

// TestSessionStore_UpdateSessionDefaultModel_NotFound tests error handling
func TestSessionStore_UpdateSessionDefaultModel_NotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "juggler-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewSessionStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Try to update non-existent session
	err = store.UpdateSessionDefaultModel("nonexistent", ModelSizeMedium)
	if err == nil {
		t.Error("expected error updating non-existent session")
	}
}

// TestJuggleSession_AcceptanceCriteria_Persistence tests ACs survive JSON round-trip
func TestJuggleSession_AcceptanceCriteria_Persistence(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "juggler-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewSessionStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Create session with acceptance criteria
	_, err = store.CreateSession("my-session", "desc")
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	criteria := []string{"Run tests", "Check build", "Update docs"}
	if err := store.UpdateSessionAcceptanceCriteria("my-session", criteria); err != nil {
		t.Fatalf("failed to update ACs: %v", err)
	}

	// Create new store instance (simulates restart)
	store2, err := NewSessionStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create second store: %v", err)
	}

	// Load session with new store
	session, err := store2.LoadSession("my-session")
	if err != nil {
		t.Fatalf("failed to load session: %v", err)
	}

	if len(session.AcceptanceCriteria) != 3 {
		t.Errorf("expected 3 ACs after reload, got %d", len(session.AcceptanceCriteria))
	}
	if session.AcceptanceCriteria[2] != "Update docs" {
		t.Errorf("expected third criterion 'Update docs', got '%s'", session.AcceptanceCriteria[2])
	}
}

// TestSessionStore_AllMetaSession_NoSessionFile tests that _all doesn't create
// or require a session.json file (it's a virtual session for storage only)
func TestSessionStore_AllMetaSession_NoSessionFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "juggler-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewSessionStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Append progress to "_all"
	err = store.AppendProgress("_all", "Progress entry\n")
	if err != nil {
		t.Fatalf("failed to append progress: %v", err)
	}

	// Verify session.json does NOT exist (only progress.txt should)
	sessionPath := filepath.Join(tmpDir, ".juggler", "sessions", "_all", "session.json")
	if _, err := os.Stat(sessionPath); !os.IsNotExist(err) {
		t.Error("expected _all to NOT have session.json file")
	}

	// Verify progress.txt DOES exist
	progressPath := filepath.Join(tmpDir, ".juggler", "sessions", "_all", "progress.txt")
	if _, err := os.Stat(progressPath); os.IsNotExist(err) {
		t.Error("expected _all to have progress.txt file")
	}

	// ListSessions should NOT include _all (since it has no session.json)
	sessions, err := store.ListSessions()
	if err != nil {
		t.Fatalf("failed to list sessions: %v", err)
	}

	for _, s := range sessions {
		if s.ID == "_all" {
			t.Error("expected _all virtual session to NOT appear in ListSessions")
		}
	}
}

// TestComputeMinimalUniqueIDs tests the minimal unique ID computation
func TestComputeMinimalUniqueIDs(t *testing.T) {
	tests := []struct {
		name     string
		balls    []*Ball
		expected map[string]string
	}{
		{
			name:     "empty slice",
			balls:    []*Ball{},
			expected: map[string]string{},
		},
		{
			name: "single ball",
			balls: []*Ball{
				{ID: "project-01234abc"},
			},
			expected: map[string]string{
				"project-01234abc": "0",
			},
		},
		{
			name: "two balls with distinct first char",
			balls: []*Ball{
				{ID: "project-01234abc"},
				{ID: "project-56789def"},
			},
			expected: map[string]string{
				"project-01234abc": "0",
				"project-56789def": "5",
			},
		},
		{
			name: "two balls with same prefix - example from AC",
			balls: []*Ball{
				{ID: "project-1111222244"},
				{ID: "project-1122334455"},
			},
			expected: map[string]string{
				"project-1111222244": "111",
				"project-1122334455": "112",
			},
		},
		{
			name: "three balls with varying prefixes",
			balls: []*Ball{
				{ID: "project-abcd1111"},
				{ID: "project-abcd2222"},
				{ID: "project-bcde3333"},
			},
			expected: map[string]string{
				"project-abcd1111": "abcd1",
				"project-abcd2222": "abcd2",
				"project-bcde3333": "b",
			},
		},
		{
			name: "legacy numeric IDs",
			balls: []*Ball{
				{ID: "project-42"},
				{ID: "project-55"},
				{ID: "project-56"},
			},
			expected: map[string]string{
				"project-42": "4",
				"project-55": "55",
				"project-56": "56",
			},
		},
		{
			name: "one ID is prefix of another",
			balls: []*Ball{
				{ID: "project-abc"},
				{ID: "project-abcdef"},
			},
			expected: map[string]string{
				"project-abc":    "abc",
				"project-abcdef": "abcd",
			},
		},
		{
			name: "multiple similar prefixes",
			balls: []*Ball{
				{ID: "project-aaa111"},
				{ID: "project-aaa222"},
				{ID: "project-aab111"},
				{ID: "project-abb111"},
			},
			expected: map[string]string{
				"project-aaa111": "aaa1",
				"project-aaa222": "aaa2",
				"project-aab111": "aab",
				"project-abb111": "ab",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ComputeMinimalUniqueIDs(tt.balls)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d results, got %d", len(tt.expected), len(result))
			}
			for id, expectedMin := range tt.expected {
				if got := result[id]; got != expectedMin {
					t.Errorf("for ball %s: expected minimal ID '%s', got '%s'", id, expectedMin, got)
				}
			}
		})
	}
}

// TestResolveBallByPrefix tests prefix-based ball resolution
func TestResolveBallByPrefix(t *testing.T) {
	balls := []*Ball{
		{ID: "project-01234abc"},
		{ID: "project-1111222244"},
		{ID: "project-1122334455"},
		{ID: "project-56789def"},
	}

	tests := []struct {
		name        string
		prefix      string
		expectCount int
		expectID    string // Expected single match, empty if multiple or none
	}{
		{
			name:        "empty prefix returns nil",
			prefix:      "",
			expectCount: 0,
		},
		{
			name:        "exact short ID match",
			prefix:      "01234abc",
			expectCount: 1,
			expectID:    "project-01234abc",
		},
		{
			name:        "single char prefix - unique",
			prefix:      "0",
			expectCount: 1,
			expectID:    "project-01234abc",
		},
		{
			name:        "single char prefix - unique 5",
			prefix:      "5",
			expectCount: 1,
			expectID:    "project-56789def",
		},
		{
			name:        "prefix matches two balls",
			prefix:      "1",
			expectCount: 2,
		},
		{
			name:        "longer prefix distinguishes - 111",
			prefix:      "111",
			expectCount: 1,
			expectID:    "project-1111222244",
		},
		{
			name:        "longer prefix distinguishes - 112",
			prefix:      "112",
			expectCount: 1,
			expectID:    "project-1122334455",
		},
		{
			name:        "full short ID",
			prefix:      "56789def",
			expectCount: 1,
			expectID:    "project-56789def",
		},
		{
			name:        "full ball ID",
			prefix:      "project-56789def",
			expectCount: 1,
			expectID:    "project-56789def",
		},
		{
			name:        "non-matching prefix",
			prefix:      "xyz",
			expectCount: 0,
		},
		{
			name:        "case insensitive match - hex chars",
			prefix:      "0123",
			expectCount: 1,
			expectID:    "project-01234abc",
		},
		{
			name:        "case insensitive match - uppercase hex starting with 5",
			prefix:      "567",
			expectCount: 1,
			expectID:    "project-56789def",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := ResolveBallByPrefix(balls, tt.prefix)
			if len(matches) != tt.expectCount {
				t.Errorf("expected %d matches for prefix '%s', got %d", tt.expectCount, tt.prefix, len(matches))
			}
			if tt.expectID != "" && len(matches) == 1 {
				if matches[0].ID != tt.expectID {
					t.Errorf("expected ID '%s', got '%s'", tt.expectID, matches[0].ID)
				}
			}
		})
	}
}

// TestResolveBallByPrefix_EdgeCases tests edge cases for prefix resolution
func TestResolveBallByPrefix_EdgeCases(t *testing.T) {
	t.Run("nil balls slice", func(t *testing.T) {
		matches := ResolveBallByPrefix(nil, "abc")
		if len(matches) != 0 {
			t.Errorf("expected 0 matches for nil slice, got %d", len(matches))
		}
	})

	t.Run("exact match takes priority", func(t *testing.T) {
		balls := []*Ball{
			{ID: "project-abc"},
			{ID: "project-abcdef"},
		}
		// When searching for "abc", should get exact match first
		matches := ResolveBallByPrefix(balls, "abc")
		if len(matches) != 1 {
			t.Errorf("expected 1 match for exact short ID, got %d", len(matches))
		}
		if matches[0].ID != "project-abc" {
			t.Errorf("expected exact match 'project-abc', got '%s'", matches[0].ID)
		}
	})

	t.Run("full ID match", func(t *testing.T) {
		balls := []*Ball{
			{ID: "project-abc"},
			{ID: "project-abcdef"},
		}
		matches := ResolveBallByPrefix(balls, "project-abc")
		if len(matches) != 1 || matches[0].ID != "project-abc" {
			t.Errorf("expected exact full ID match, got %v", matches)
		}
	})
}

// TestMinimalUniqueIDsConsistency ensures IDs are actually unique
func TestMinimalUniqueIDsConsistency(t *testing.T) {
	balls := []*Ball{
		{ID: "proj-a1b2c3d4"},
		{ID: "proj-a1b2e5f6"},
		{ID: "proj-b3c4d5e6"},
		{ID: "proj-b3c4f7g8"},
	}

	minIDs := ComputeMinimalUniqueIDs(balls)

	// Verify all minimal IDs are unique
	seen := make(map[string]string)
	for id, minID := range minIDs {
		if existing, ok := seen[minID]; ok {
			t.Errorf("duplicate minimal ID '%s' for balls '%s' and '%s'", minID, existing, id)
		}
		seen[minID] = id
	}

	// Verify each ball can be resolved by its minimal ID
	for _, ball := range balls {
		minID := minIDs[ball.ID]
		matches := ResolveBallByPrefix(balls, minID)
		if len(matches) != 1 {
			t.Errorf("minimal ID '%s' for ball '%s' matched %d balls, expected 1", minID, ball.ID, len(matches))
		} else if matches[0].ID != ball.ID {
			t.Errorf("minimal ID '%s' resolved to '%s', expected '%s'", minID, matches[0].ID, ball.ID)
		}
	}
}

// TestLowerString tests the internal lowerString function
func TestLowerString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"ABC", "abc"},
		{"abc", "abc"},
		{"AbCdEf", "abcdef"},
		{"123ABC", "123abc"},
		{"", ""},
		{"HELLO-WORLD", "hello-world"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := lowerString(tt.input)
			if got != tt.expected {
				t.Errorf("lowerString(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// TestLoadAllSessions tests loading sessions from multiple projects
func TestLoadAllSessions(t *testing.T) {
	// Create two temp project directories
	project1, err := os.MkdirTemp("", "juggler-project1-*")
	if err != nil {
		t.Fatalf("failed to create project1 dir: %v", err)
	}
	defer os.RemoveAll(project1)

	project2, err := os.MkdirTemp("", "juggler-project2-*")
	if err != nil {
		t.Fatalf("failed to create project2 dir: %v", err)
	}
	defer os.RemoveAll(project2)

	// Create sessions in project1
	store1, err := NewSessionStore(project1)
	if err != nil {
		t.Fatalf("failed to create store1: %v", err)
	}
	_, err = store1.CreateSession("session-a", "Session A in project1")
	if err != nil {
		t.Fatalf("failed to create session-a: %v", err)
	}
	_, err = store1.CreateSession("session-b", "Session B in project1")
	if err != nil {
		t.Fatalf("failed to create session-b: %v", err)
	}

	// Create sessions in project2
	store2, err := NewSessionStore(project2)
	if err != nil {
		t.Fatalf("failed to create store2: %v", err)
	}
	_, err = store2.CreateSession("session-c", "Session C in project2")
	if err != nil {
		t.Fatalf("failed to create session-c: %v", err)
	}

	// Load all sessions from both projects
	sessions, err := LoadAllSessions([]string{project1, project2})
	if err != nil {
		t.Fatalf("failed to load all sessions: %v", err)
	}

	// Verify we got all 3 sessions
	if len(sessions) != 3 {
		t.Errorf("expected 3 sessions, got %d", len(sessions))
	}

	// Verify specific sessions exist
	sessionIDs := make(map[string]bool)
	for _, sess := range sessions {
		sessionIDs[sess.ID] = true
	}

	if !sessionIDs["session-a"] {
		t.Error("expected session-a to be in results")
	}
	if !sessionIDs["session-b"] {
		t.Error("expected session-b to be in results")
	}
	if !sessionIDs["session-c"] {
		t.Error("expected session-c to be in results")
	}
}

// TestLoadAllSessions_EmptyProject tests loading sessions when a project has no sessions
func TestLoadAllSessions_EmptyProject(t *testing.T) {
	// Create two temp project directories
	project1, err := os.MkdirTemp("", "juggler-project1-*")
	if err != nil {
		t.Fatalf("failed to create project1 dir: %v", err)
	}
	defer os.RemoveAll(project1)

	project2, err := os.MkdirTemp("", "juggler-project2-*")
	if err != nil {
		t.Fatalf("failed to create project2 dir: %v", err)
	}
	defer os.RemoveAll(project2)

	// Create session only in project1
	store1, err := NewSessionStore(project1)
	if err != nil {
		t.Fatalf("failed to create store1: %v", err)
	}
	_, err = store1.CreateSession("session-only", "Only session")
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Initialize project2 store but don't create sessions
	_, err = NewSessionStore(project2)
	if err != nil {
		t.Fatalf("failed to create store2: %v", err)
	}

	// Load all sessions
	sessions, err := LoadAllSessions([]string{project1, project2})
	if err != nil {
		t.Fatalf("failed to load all sessions: %v", err)
	}

	// Should only have 1 session
	if len(sessions) != 1 {
		t.Errorf("expected 1 session, got %d", len(sessions))
	}

	if len(sessions) > 0 && sessions[0].ID != "session-only" {
		t.Errorf("expected session ID 'session-only', got '%s'", sessions[0].ID)
	}
}

// TestLoadAllSessions_NoProjects tests loading sessions with no projects
func TestLoadAllSessions_NoProjects(t *testing.T) {
	sessions, err := LoadAllSessions([]string{})
	if err != nil {
		t.Fatalf("expected no error with empty project list, got: %v", err)
	}

	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions with no projects, got %d", len(sessions))
	}
}
