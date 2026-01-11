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

// CreateBall creates a new ball for testing
func (env *TestEnv) CreateBall(t *testing.T, intent string, priority session.Priority) *session.Ball {
	t.Helper()

	store, err := cli.NewStoreForCommand(env.ProjectDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	ball, err := session.NewBall(env.ProjectDir, intent, priority)
	if err != nil {
		t.Fatalf("Failed to create ball: %v", err)
	}

	if err := store.AppendBall(ball); err != nil {
		t.Fatalf("Failed to save ball: %v", err)
	}

	return ball
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

// AssertBallExists checks that a ball with the given ID exists
func (env *TestEnv) AssertBallExists(t *testing.T, ballID string) *session.Ball {
	t.Helper()

	store := env.GetStore(t)
	ball, err := store.GetBallByID(ballID)
	if err != nil {
		t.Fatalf("Expected ball %s to exist, but got error: %v", ballID, err)
	}

	return ball
}

// AssertState checks that a ball has the expected state
func (env *TestEnv) AssertState(t *testing.T, ballID string, expectedState session.BallState) {
	t.Helper()

	ball := env.AssertBallExists(t, ballID)
	if ball.State != expectedState {
		t.Fatalf("Expected ball %s to have state %s, but got %s", ballID, expectedState, ball.State)
	}
}

// AssertBallNotExists checks that a ball with the given ID does not exist in active balls
func (env *TestEnv) AssertBallNotExists(t *testing.T, ballID string) {
	t.Helper()

	store := env.GetStore(t)
	_, err := store.GetBallByID(ballID)
	if err == nil {
		t.Fatalf("Expected ball %s to not exist, but it was found", ballID)
	}
}

// AssertBallArchived checks that a ball is in the archive
func (env *TestEnv) AssertBallArchived(t *testing.T, ballID string) {
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

// CreateInProgressBall creates a new ball in in_progress state for testing
func (env *TestEnv) CreateInProgressBall(t *testing.T, intent string, priority session.Priority) *session.Ball {
	t.Helper()

	store := env.GetStore(t)

	ball, err := session.NewBall(env.ProjectDir, intent, priority)
	if err != nil {
		t.Fatalf("Failed to create ball: %v", err)
	}

	// Set to in_progress state
	ball.Start()

	if err := store.AppendBall(ball); err != nil {
		t.Fatalf("Failed to save in-progress ball: %v", err)
	}

	return ball
}

// GetBallUpdateCount retrieves the current update count for a ball
func (env *TestEnv) GetBallUpdateCount(t *testing.T, ballID string) int {
	t.Helper()

	ball := env.AssertBallExists(t, ballID)
	return ball.UpdateCount
}
