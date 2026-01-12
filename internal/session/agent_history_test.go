package session

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewAgentRunRecord(t *testing.T) {
	startTime := time.Now()
	record := NewAgentRunRecord("test-session", "/test/project", startTime)

	if record.ID == "" {
		t.Error("Expected ID to be set")
	}
	if record.SessionID != "test-session" {
		t.Errorf("Expected session ID 'test-session', got '%s'", record.SessionID)
	}
	if record.ProjectDir != "/test/project" {
		t.Errorf("Expected project dir '/test/project', got '%s'", record.ProjectDir)
	}
	if !record.StartedAt.Equal(startTime) {
		t.Errorf("Expected start time %v, got %v", startTime, record.StartedAt)
	}
}

func TestAgentRunRecord_SetComplete(t *testing.T) {
	record := NewAgentRunRecord("test", "/test", time.Now())
	record.SetComplete(5, 10, 2, 12)

	if record.Result != "complete" {
		t.Errorf("Expected result 'complete', got '%s'", record.Result)
	}
	if record.Iterations != 5 {
		t.Errorf("Expected 5 iterations, got %d", record.Iterations)
	}
	if record.BallsComplete != 10 {
		t.Errorf("Expected 10 balls complete, got %d", record.BallsComplete)
	}
	if record.BallsBlocked != 2 {
		t.Errorf("Expected 2 balls blocked, got %d", record.BallsBlocked)
	}
	if record.BallsTotal != 12 {
		t.Errorf("Expected 12 total balls, got %d", record.BallsTotal)
	}
	if record.EndedAt.IsZero() {
		t.Error("Expected EndedAt to be set")
	}
}

func TestAgentRunRecord_SetBlocked(t *testing.T) {
	record := NewAgentRunRecord("test", "/test", time.Now())
	record.SetBlocked(3, "API key missing", 5, 1, 10)

	if record.Result != "blocked" {
		t.Errorf("Expected result 'blocked', got '%s'", record.Result)
	}
	if record.BlockedReason != "API key missing" {
		t.Errorf("Expected blocked reason 'API key missing', got '%s'", record.BlockedReason)
	}
	if record.Iterations != 3 {
		t.Errorf("Expected 3 iterations, got %d", record.Iterations)
	}
}

func TestAgentRunRecord_SetTimeout(t *testing.T) {
	record := NewAgentRunRecord("test", "/test", time.Now())
	record.SetTimeout(2, "Iteration 2 timed out after 5m", 3, 0, 8)

	if record.Result != "timeout" {
		t.Errorf("Expected result 'timeout', got '%s'", record.Result)
	}
	if record.TimeoutMessage != "Iteration 2 timed out after 5m" {
		t.Errorf("Expected timeout message, got '%s'", record.TimeoutMessage)
	}
}

func TestAgentRunRecord_SetMaxIterations(t *testing.T) {
	record := NewAgentRunRecord("test", "/test", time.Now())
	record.SetMaxIterations(10, 8, 0, 10)

	if record.Result != "max_iterations" {
		t.Errorf("Expected result 'max_iterations', got '%s'", record.Result)
	}
}

func TestAgentRunRecord_SetRateLimitExceeded(t *testing.T) {
	record := NewAgentRunRecord("test", "/test", time.Now())
	record.SetRateLimitExceeded(4, 10*time.Minute, 6, 1, 12)

	if record.Result != "rate_limit" {
		t.Errorf("Expected result 'rate_limit', got '%s'", record.Result)
	}
	if record.TotalWaitTime != 10*time.Minute {
		t.Errorf("Expected 10m wait time, got %v", record.TotalWaitTime)
	}
}

func TestAgentRunRecord_SetCancelled(t *testing.T) {
	record := NewAgentRunRecord("test", "/test", time.Now())
	record.SetCancelled(1, 0, 0, 5)

	if record.Result != "cancelled" {
		t.Errorf("Expected result 'cancelled', got '%s'", record.Result)
	}
}

func TestAgentRunRecord_SetError(t *testing.T) {
	record := NewAgentRunRecord("test", "/test", time.Now())
	record.SetError(1, "command failed", 0, 0, 5)

	if record.Result != "error" {
		t.Errorf("Expected result 'error', got '%s'", record.Result)
	}
	if record.ErrorMessage != "command failed" {
		t.Errorf("Expected error message 'command failed', got '%s'", record.ErrorMessage)
	}
}

func TestAgentRunRecord_Duration(t *testing.T) {
	startTime := time.Now().Add(-5 * time.Minute)
	record := NewAgentRunRecord("test", "/test", startTime)
	record.EndedAt = time.Now()

	duration := record.Duration()
	if duration < 4*time.Minute || duration > 6*time.Minute {
		t.Errorf("Expected duration around 5 minutes, got %v", duration)
	}
}

func TestAgentHistoryStore_AppendAndLoad(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "juggler-history-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create store
	store, err := NewAgentHistoryStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create history store: %v", err)
	}

	// Create and append a record
	record1 := NewAgentRunRecord("session1", tmpDir, time.Now().Add(-2*time.Hour))
	record1.SetComplete(5, 10, 2, 12)
	record1.OutputFile = "/path/to/output1.txt"

	if err := store.AppendRecord(record1); err != nil {
		t.Fatalf("Failed to append record: %v", err)
	}

	// Append another record
	record2 := NewAgentRunRecord("session2", tmpDir, time.Now().Add(-1*time.Hour))
	record2.SetBlocked(3, "API error", 5, 1, 10)
	record2.OutputFile = "/path/to/output2.txt"

	if err := store.AppendRecord(record2); err != nil {
		t.Fatalf("Failed to append second record: %v", err)
	}

	// Load history
	history, err := store.LoadHistory()
	if err != nil {
		t.Fatalf("Failed to load history: %v", err)
	}

	if len(history) != 2 {
		t.Fatalf("Expected 2 records, got %d", len(history))
	}

	// Verify order (most recent first)
	if history[0].SessionID != "session2" {
		t.Errorf("Expected first record to be session2 (most recent), got %s", history[0].SessionID)
	}
	if history[1].SessionID != "session1" {
		t.Errorf("Expected second record to be session1 (older), got %s", history[1].SessionID)
	}
}

func TestAgentHistoryStore_LoadHistoryBySession(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "juggler-history-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewAgentHistoryStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create history store: %v", err)
	}

	// Add records for different sessions
	record1 := NewAgentRunRecord("session1", tmpDir, time.Now().Add(-3*time.Hour))
	record1.SetComplete(5, 10, 0, 10)
	store.AppendRecord(record1)

	record2 := NewAgentRunRecord("session2", tmpDir, time.Now().Add(-2*time.Hour))
	record2.SetComplete(3, 5, 0, 5)
	store.AppendRecord(record2)

	record3 := NewAgentRunRecord("session1", tmpDir, time.Now().Add(-1*time.Hour))
	record3.SetBlocked(2, "blocked", 3, 1, 5)
	store.AppendRecord(record3)

	// Load by session
	session1History, err := store.LoadHistoryBySession("session1")
	if err != nil {
		t.Fatalf("Failed to load history by session: %v", err)
	}

	if len(session1History) != 2 {
		t.Fatalf("Expected 2 records for session1, got %d", len(session1History))
	}

	for _, r := range session1History {
		if r.SessionID != "session1" {
			t.Errorf("Expected all records to be session1, got %s", r.SessionID)
		}
	}
}

func TestAgentHistoryStore_LoadRecentHistory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "juggler-history-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewAgentHistoryStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create history store: %v", err)
	}

	// Add several records
	for i := 0; i < 10; i++ {
		record := NewAgentRunRecord("session", tmpDir, time.Now().Add(-time.Duration(i)*time.Hour))
		record.SetComplete(1, 1, 0, 1)
		store.AppendRecord(record)
	}

	// Load only 5 most recent
	history, err := store.LoadRecentHistory(5)
	if err != nil {
		t.Fatalf("Failed to load recent history: %v", err)
	}

	if len(history) != 5 {
		t.Fatalf("Expected 5 records, got %d", len(history))
	}
}

func TestAgentHistoryStore_EmptyHistory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "juggler-history-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewAgentHistoryStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create history store: %v", err)
	}

	// Load without any records
	history, err := store.LoadHistory()
	if err != nil {
		t.Fatalf("Failed to load empty history: %v", err)
	}

	if len(history) != 0 {
		t.Errorf("Expected empty history, got %d records", len(history))
	}
}

func TestAgentHistoryStore_HistoryFilePath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "juggler-history-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewAgentHistoryStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create history store: %v", err)
	}

	expectedPath := filepath.Join(tmpDir, ".juggler", "agent_history.jsonl")
	actualPath := store.historyFilePath()

	if actualPath != expectedPath {
		t.Errorf("Expected path '%s', got '%s'", expectedPath, actualPath)
	}
}
