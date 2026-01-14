package session

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestAcquireSessionLock_Success(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "lock-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create session store and session
	store, err := NewSessionStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	_, err = store.CreateSession("test-session", "Test session")
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Acquire lock
	lock, err := store.AcquireSessionLock("test-session")
	if err != nil {
		t.Fatalf("failed to acquire lock: %v", err)
	}
	defer lock.Release()

	// Verify lock file exists
	lockPath := filepath.Join(tmpDir, ".juggle", "sessions", "test-session", "agent.lock")
	if _, err := os.Stat(lockPath); os.IsNotExist(err) {
		t.Error("lock file should exist after acquiring lock")
	}

	// Verify lock info file exists
	lockInfoPath := filepath.Join(tmpDir, ".juggle", "sessions", "test-session", "agent.lock.info")
	if _, err := os.Stat(lockInfoPath); os.IsNotExist(err) {
		t.Error("lock info file should exist after acquiring lock")
	}

	// Verify lock info is written to the info file
	info, err := readLockInfo(lockInfoPath)
	if err != nil {
		t.Fatalf("failed to read lock info: %v", err)
	}

	if info.PID != os.Getpid() {
		t.Errorf("lock PID = %d, want %d", info.PID, os.Getpid())
	}

	if info.StartedAt.IsZero() {
		t.Error("lock StartedAt should not be zero")
	}
}

func TestAcquireSessionLock_AlreadyLocked(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "lock-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create session store and session
	store, err := NewSessionStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	_, err = store.CreateSession("test-session", "Test session")
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Acquire first lock
	lock1, err := store.AcquireSessionLock("test-session")
	if err != nil {
		t.Fatalf("failed to acquire first lock: %v", err)
	}
	defer lock1.Release()

	// Try to acquire second lock - should fail
	lock2, err := store.AcquireSessionLock("test-session")
	if err == nil {
		lock2.Release()
		t.Fatal("expected error when acquiring lock on already locked session")
	}

	// Error message should mention the session
	if err.Error() == "" {
		t.Error("error message should not be empty")
	}
}

func TestAcquireSessionLock_ReleaseAndReacquire(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "lock-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create session store and session
	store, err := NewSessionStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	_, err = store.CreateSession("test-session", "Test session")
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Acquire and release lock
	lock1, err := store.AcquireSessionLock("test-session")
	if err != nil {
		t.Fatalf("failed to acquire first lock: %v", err)
	}

	err = lock1.Release()
	if err != nil {
		t.Fatalf("failed to release lock: %v", err)
	}

	// Should be able to acquire again
	lock2, err := store.AcquireSessionLock("test-session")
	if err != nil {
		t.Fatalf("failed to acquire second lock after release: %v", err)
	}
	defer lock2.Release()
}

func TestAcquireSessionLock_NonexistentSession(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "lock-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create session store but no session
	store, err := NewSessionStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Try to acquire lock on nonexistent session
	_, err = store.AcquireSessionLock("nonexistent")
	if err == nil {
		t.Fatal("expected error when acquiring lock on nonexistent session")
	}
}

func TestIsLocked(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "lock-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create session store and session
	store, err := NewSessionStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	_, err = store.CreateSession("test-session", "Test session")
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Initially not locked
	locked, info := store.IsLocked("test-session")
	if locked {
		t.Error("session should not be locked initially")
	}
	if info != nil {
		t.Error("lock info should be nil when not locked")
	}

	// Acquire lock
	lock, err := store.AcquireSessionLock("test-session")
	if err != nil {
		t.Fatalf("failed to acquire lock: %v", err)
	}

	// Now should be locked
	locked, info = store.IsLocked("test-session")
	if !locked {
		t.Error("session should be locked after acquiring lock")
	}
	if info == nil {
		t.Error("lock info should not be nil when locked")
	}

	// Release lock
	lock.Release()

	// Should not be locked anymore
	locked, _ = store.IsLocked("test-session")
	if locked {
		t.Error("session should not be locked after release")
	}
}

func TestReleaseLock_Idempotent(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "lock-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create session store and session
	store, err := NewSessionStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	_, err = store.CreateSession("test-session", "Test session")
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Acquire lock
	lock, err := store.AcquireSessionLock("test-session")
	if err != nil {
		t.Fatalf("failed to acquire lock: %v", err)
	}

	// Release multiple times - should not error
	if err := lock.Release(); err != nil {
		t.Fatalf("first release failed: %v", err)
	}

	if err := lock.Release(); err != nil {
		t.Fatalf("second release should not error: %v", err)
	}
}

func TestConcurrentLockAttempts(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "lock-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create session store and session
	store, err := NewSessionStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	_, err = store.CreateSession("test-session", "Test session")
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Try to acquire locks concurrently
	numGoroutines := 10
	var wg sync.WaitGroup
	successCount := 0
	var mu sync.Mutex

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			lock, err := store.AcquireSessionLock("test-session")
			if err == nil {
				mu.Lock()
				successCount++
				mu.Unlock()
				// Hold lock briefly then release
				time.Sleep(10 * time.Millisecond)
				lock.Release()
			}
		}()
	}

	wg.Wait()

	// Only one should succeed at acquiring the initial lock
	// (others may succeed after releases, but that's the intended behavior)
	if successCount == 0 {
		t.Error("at least one goroutine should have acquired the lock")
	}
}

// TestLockFilesCleanedUpOnRelease verifies both lock file and info file are removed
// This is important for Windows compatibility where the lock file cannot be written
// to directly while locked, so we use a separate info file.
func TestLockFilesCleanedUpOnRelease(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "lock-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create session store and session
	store, err := NewSessionStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	_, err = store.CreateSession("test-session", "Test session")
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Acquire lock
	lock, err := store.AcquireSessionLock("test-session")
	if err != nil {
		t.Fatalf("failed to acquire lock: %v", err)
	}

	// Verify both files exist while locked
	lockPath := filepath.Join(tmpDir, ".juggle", "sessions", "test-session", "agent.lock")
	lockInfoPath := filepath.Join(tmpDir, ".juggle", "sessions", "test-session", "agent.lock.info")

	if _, err := os.Stat(lockPath); os.IsNotExist(err) {
		t.Error("lock file should exist while locked")
	}
	if _, err := os.Stat(lockInfoPath); os.IsNotExist(err) {
		t.Error("lock info file should exist while locked")
	}

	// Release lock
	if err := lock.Release(); err != nil {
		t.Fatalf("failed to release lock: %v", err)
	}

	// Verify both files are cleaned up
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Error("lock file should be removed after release")
	}
	if _, err := os.Stat(lockInfoPath); !os.IsNotExist(err) {
		t.Error("lock info file should be removed after release")
	}
}

func TestLockInfoContainsHostname(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "lock-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create session store and session
	store, err := NewSessionStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	_, err = store.CreateSession("test-session", "Test session")
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Acquire lock
	lock, err := store.AcquireSessionLock("test-session")
	if err != nil {
		t.Fatalf("failed to acquire lock: %v", err)
	}
	defer lock.Release()

	// Check that IsLocked returns hostname in info
	_, info := store.IsLocked("test-session")
	if info == nil {
		t.Fatal("expected lock info")
	}

	expectedHostname, _ := os.Hostname()
	if info.Hostname != expectedHostname {
		t.Errorf("hostname = %q, want %q", info.Hostname, expectedHostname)
	}
}
