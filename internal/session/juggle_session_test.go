package session

import (
	"os"
	"path/filepath"
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
