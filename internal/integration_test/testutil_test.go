package integration_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ohare93/juggle/internal/cli"
	"github.com/ohare93/juggle/internal/session"
)

// TestEnv holds the test environment setup
type TestEnv struct {
	TempDir       string
	ProjectDir    string
	ConfigHome    string
	JugglerDir    string
	OriginalFlags cli.GlobalOptions
}

// SetupTestEnv creates an isolated test environment with temporary directories
func SetupTestEnv(t *testing.T) *TestEnv {
	t.Helper()

	// Create temporary directory for this test
	tempDir, err := os.MkdirTemp("", "juggler-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create subdirectories
	projectDir := filepath.Join(tempDir, "project")
	configHome := filepath.Join(tempDir, "config")
	jugglerDir := filepath.Join(projectDir, ".juggler")

	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("Failed to create project dir: %v", err)
	}
	if err := os.MkdirAll(configHome, 0755); err != nil {
		t.Fatalf("Failed to create config home: %v", err)
	}

	env := &TestEnv{
		TempDir:       tempDir,
		ProjectDir:    projectDir,
		ConfigHome:    configHome,
		JugglerDir:    jugglerDir,
		OriginalFlags: cli.GlobalOpts,
	}

	// Set global options for the test
	cli.GlobalOpts = cli.GlobalOptions{
		ConfigHome: configHome,
		ProjectDir: projectDir,
		JugglerDir: ".juggler",
	}

	return env
}

// CleanupTestEnv removes the test environment and restores original settings
func CleanupTestEnv(t *testing.T, env *TestEnv) {
	t.Helper()

	// Restore original flags
	cli.GlobalOpts = env.OriginalFlags

	// Clean up temp directory
	if err := os.RemoveAll(env.TempDir); err != nil {
		t.Logf("Warning: Failed to remove temp dir %s: %v", env.TempDir, err)
	}
}

// CreateSession creates a new session for testing
func (env *TestEnv) CreateSession(t *testing.T, intent string, priority session.Priority) *session.Session {
	t.Helper()

	store, err := cli.NewStoreForCommand(env.ProjectDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	sess, err := session.New(env.ProjectDir, intent, priority)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	if err := store.AppendBall(sess); err != nil {
		t.Fatalf("Failed to save session: %v", err)
	}

	return sess
}

// GetStore returns a store configured for the test environment
func (env *TestEnv) GetStore(t *testing.T) *session.Store {
	t.Helper()

	store, err := cli.NewStoreForCommand(env.ProjectDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	return store
}

// AssertSessionExists checks that a session with the given ID exists
func (env *TestEnv) AssertSessionExists(t *testing.T, sessionID string) *session.Session {
	t.Helper()

	store := env.GetStore(t)
	sess, err := store.GetBallByID(sessionID)
	if err != nil {
		t.Fatalf("Expected session %s to exist, but got error: %v", sessionID, err)
	}

	return sess
}

// AssertActiveState checks that a session has the expected active state
func (env *TestEnv) AssertActiveState(t *testing.T, sessionID string, expectedState session.ActiveState) {
	t.Helper()

	sess := env.AssertSessionExists(t, sessionID)
	if sess.ActiveState != expectedState {
		t.Fatalf("Expected session %s to have active state %s, but got %s", sessionID, expectedState, sess.ActiveState)
	}
}

// AssertJuggleState checks that a session has the expected juggle state
func (env *TestEnv) AssertJuggleState(t *testing.T, sessionID string, expectedState session.JuggleState) {
	t.Helper()

	sess := env.AssertSessionExists(t, sessionID)
	if sess.JuggleState == nil {
		t.Fatalf("Expected session %s to have juggle state %s, but got nil", sessionID, expectedState)
	}
	if *sess.JuggleState != expectedState {
		t.Fatalf("Expected session %s to have juggle state %s, but got %s", sessionID, expectedState, *sess.JuggleState)
	}
}

// AssertSessionNotExists checks that a session with the given ID does not exist in active sessions
func (env *TestEnv) AssertSessionNotExists(t *testing.T, sessionID string) {
	t.Helper()

	store := env.GetStore(t)
	_, err := store.GetBallByID(sessionID)
	if err == nil {
		t.Fatalf("Expected session %s to not exist, but it was found", sessionID)
	}
}

// AssertSessionArchived checks that a session is in the archive
func (env *TestEnv) AssertSessionArchived(t *testing.T, sessionID string) {
	t.Helper()

	archivePath := filepath.Join(env.JugglerDir, "archive")
	archiveFile := filepath.Join(archivePath, "balls.jsonl")

	if _, err := os.Stat(archiveFile); os.IsNotExist(err) {
		t.Fatalf("Expected archive file to exist at %s", archiveFile)
	}
}


// SetEnvVar sets an environment variable for the test and ensures cleanup
func (env *TestEnv) SetEnvVar(t *testing.T, key, value string) {
	t.Helper()

	// Store original value for cleanup
	originalValue, originalExists := os.LookupEnv(key)

	// Set the new value
	if err := os.Setenv(key, value); err != nil {
		t.Fatalf("Failed to set env var %s: %v", key, err)
	}

	// Register cleanup
	t.Cleanup(func() {
		if originalExists {
			os.Setenv(key, originalValue)
		} else {
			os.Unsetenv(key)
		}
	})
}

// ClearEnvVar clears an environment variable for the test and ensures cleanup
func (env *TestEnv) ClearEnvVar(t *testing.T, key string) {
	t.Helper()

	// Store original value for cleanup
	originalValue, originalExists := os.LookupEnv(key)

	// Clear the variable
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("Failed to unset env var %s: %v", key, err)
	}

	// Register cleanup
	t.Cleanup(func() {
		if originalExists {
			os.Setenv(key, originalValue)
		}
	})
}

// CreateJugglingBall creates a new session in juggling state for testing
func (env *TestEnv) CreateJugglingBall(t *testing.T, intent string, priority session.Priority, juggleState session.JuggleState) *session.Session {
	t.Helper()

	store := env.GetStore(t)

	sess, err := session.New(env.ProjectDir, intent, priority)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Set to juggling state
	sess.SetActiveState(session.ActiveJuggling)
	sess.SetJuggleState(juggleState, "")

	if err := store.AppendBall(sess); err != nil {
		t.Fatalf("Failed to save juggling session: %v", err)
	}

	return sess
}

// CreateBallWithZellij creates a new session with Zellij session and tab info
func (env *TestEnv) CreateBallWithZellij(t *testing.T, intent string, priority session.Priority, zellijSession, zellijTab string) *session.Session {
	t.Helper()

	store := env.GetStore(t)

	sess, err := session.New(env.ProjectDir, intent, priority)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Set Zellij info
	sess.SetZellijInfo(zellijSession, zellijTab)
	
	// Set to juggling state
	sess.SetActiveState(session.ActiveJuggling)
	inAir := session.JuggleInAir
	sess.JuggleState = &inAir

	if err := store.AppendBall(sess); err != nil {
		t.Fatalf("Failed to save session with Zellij info: %v", err)
	}

	return sess
}

// GetBallUpdateCount retrieves the current update count for a ball
func (env *TestEnv) GetBallUpdateCount(t *testing.T, ballID string) int {
	t.Helper()

	sess := env.AssertSessionExists(t, ballID)
	return sess.UpdateCount
}
