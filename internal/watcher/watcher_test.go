package watcher

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewWatcher(t *testing.T) {
	w, err := New()
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer w.Close()

	if w.Events == nil {
		t.Error("Events channel should not be nil")
	}
	if w.Errors == nil {
		t.Error("Errors channel should not be nil")
	}
}

func TestWatchProject(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()
	jugglerDir := filepath.Join(tmpDir, ".juggler")
	if err := os.MkdirAll(jugglerDir, 0755); err != nil {
		t.Fatalf("Failed to create juggler dir: %v", err)
	}

	// Create balls.jsonl
	ballsPath := filepath.Join(jugglerDir, "balls.jsonl")
	if err := os.WriteFile(ballsPath, []byte("{}"), 0644); err != nil {
		t.Fatalf("Failed to create balls.jsonl: %v", err)
	}

	w, err := New()
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer w.Close()

	if err := w.WatchProject(tmpDir); err != nil {
		t.Fatalf("Failed to watch project: %v", err)
	}
}

func TestWatchProject_NoJugglerDir(t *testing.T) {
	tmpDir := t.TempDir()

	w, err := New()
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer w.Close()

	err = w.WatchProject(tmpDir)
	if err == nil {
		t.Error("Expected error for missing .juggler directory")
	}
}

func TestClassifyEvent_BallsChanged(t *testing.T) {
	w, _ := New()
	defer w.Close()

	event := w.classifyEvent("/path/to/.juggler/balls.jsonl")
	if event == nil {
		t.Fatal("Expected event, got nil")
	}
	if event.Type != BallsChanged {
		t.Errorf("Expected BallsChanged, got %v", event.Type)
	}
}

func TestClassifyEvent_ProgressChanged(t *testing.T) {
	w, _ := New()
	defer w.Close()

	event := w.classifyEvent("/path/to/.juggler/sessions/my-session/progress.txt")
	if event == nil {
		t.Fatal("Expected event, got nil")
	}
	if event.Type != ProgressChanged {
		t.Errorf("Expected ProgressChanged, got %v", event.Type)
	}
	if event.SessionID != "my-session" {
		t.Errorf("Expected session ID 'my-session', got '%s'", event.SessionID)
	}
}

func TestClassifyEvent_SessionChanged(t *testing.T) {
	w, _ := New()
	defer w.Close()

	event := w.classifyEvent("/path/to/.juggler/sessions/my-session/session.json")
	if event == nil {
		t.Fatal("Expected event, got nil")
	}
	if event.Type != SessionChanged {
		t.Errorf("Expected SessionChanged, got %v", event.Type)
	}
	if event.SessionID != "my-session" {
		t.Errorf("Expected session ID 'my-session', got '%s'", event.SessionID)
	}
}

func TestClassifyEvent_Unknown(t *testing.T) {
	w, _ := New()
	defer w.Close()

	event := w.classifyEvent("/path/to/random/file.txt")
	if event != nil {
		t.Error("Expected nil for unknown file type")
	}
}

func TestWatcherBallsFileChange(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()
	jugglerDir := filepath.Join(tmpDir, ".juggler")
	if err := os.MkdirAll(jugglerDir, 0755); err != nil {
		t.Fatalf("Failed to create juggler dir: %v", err)
	}

	// Create balls.jsonl
	ballsPath := filepath.Join(jugglerDir, "balls.jsonl")
	if err := os.WriteFile(ballsPath, []byte("{}"), 0644); err != nil {
		t.Fatalf("Failed to create balls.jsonl: %v", err)
	}

	w, err := New()
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer w.Close()

	if err := w.WatchProject(tmpDir); err != nil {
		t.Fatalf("Failed to watch project: %v", err)
	}

	w.Start()

	// Give the watcher time to start
	time.Sleep(50 * time.Millisecond)

	// Write to balls.jsonl
	if err := os.WriteFile(ballsPath, []byte(`{"id": "test"}`), 0644); err != nil {
		t.Fatalf("Failed to write balls.jsonl: %v", err)
	}

	// Wait for event
	select {
	case event := <-w.Events:
		if event.Type != BallsChanged {
			t.Errorf("Expected BallsChanged event, got %v", event.Type)
		}
	case <-time.After(2 * time.Second):
		t.Error("Timed out waiting for event")
	}
}

func TestWatcherProgressFileChange(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()
	jugglerDir := filepath.Join(tmpDir, ".juggler")
	sessionsDir := filepath.Join(jugglerDir, "sessions", "test-session")
	if err := os.MkdirAll(sessionsDir, 0755); err != nil {
		t.Fatalf("Failed to create sessions dir: %v", err)
	}

	// Create progress.txt
	progressPath := filepath.Join(sessionsDir, "progress.txt")
	if err := os.WriteFile(progressPath, []byte("initial"), 0644); err != nil {
		t.Fatalf("Failed to create progress.txt: %v", err)
	}

	// Create balls.jsonl (required for WatchProject)
	ballsPath := filepath.Join(jugglerDir, "balls.jsonl")
	if err := os.WriteFile(ballsPath, []byte("{}"), 0644); err != nil {
		t.Fatalf("Failed to create balls.jsonl: %v", err)
	}

	w, err := New()
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer w.Close()

	if err := w.WatchProject(tmpDir); err != nil {
		t.Fatalf("Failed to watch project: %v", err)
	}

	w.Start()

	// Give the watcher time to start
	time.Sleep(50 * time.Millisecond)

	// Write to progress.txt
	if err := os.WriteFile(progressPath, []byte("updated progress"), 0644); err != nil {
		t.Fatalf("Failed to write progress.txt: %v", err)
	}

	// Wait for event
	select {
	case event := <-w.Events:
		if event.Type != ProgressChanged {
			t.Errorf("Expected ProgressChanged event, got %v", event.Type)
		}
		if event.SessionID != "test-session" {
			t.Errorf("Expected session ID 'test-session', got '%s'", event.SessionID)
		}
	case <-time.After(2 * time.Second):
		t.Error("Timed out waiting for event")
	}
}

func TestWatcherStop(t *testing.T) {
	w, err := New()
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}

	w.Start()

	if err := w.Stop(); err != nil {
		t.Errorf("Failed to stop watcher: %v", err)
	}

	// Stop should be idempotent
	if err := w.Stop(); err != nil {
		t.Errorf("Second stop should not error: %v", err)
	}
}
