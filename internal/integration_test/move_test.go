package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ohare93/juggle/internal/session"
)

func TestMoveCommand(t *testing.T) {
	t.Run("MoveToAnotherProject", testMoveToAnotherProject)
	t.Run("MoveUpdatesWorkingDir", testMoveUpdatesWorkingDir)
	t.Run("MoveNonexistentBall", testMoveNonexistentBall)
	t.Run("MoveToNonJuggleProject", testMoveToNonJuggleProject)
	t.Run("MovePreservesMetadata", testMovePreservesMetadata)
	t.Run("MoveRemovesFromSource", testMoveRemovesFromSource)
	t.Run("MoveWithShortID", testMoveWithShortID)
	t.Run("MoveToSameProject", testMoveToSameProject)
}

func testMoveToAnotherProject(t *testing.T) {
	// Setup two separate test environments
	envA := SetupTestEnv(t)
	defer CleanupTestEnv(t, envA)
	envB := SetupTestEnv(t)
	defer CleanupTestEnv(t, envB)

	// Initialize .juggle directory in project B by getting store
	_ = envB.GetStore(t)

	// Setup config with both projects so discovery works
	setupConfigWithProjects(t, envA.ConfigHome, envA.ProjectDir, envB.ProjectDir)

	// Create ball in project A
	ball := envA.CreateBall(t, "Move me", session.PriorityMedium)
	ballID := ball.ID

	// Move to project B
	output := runMoveCommand(t, envA.ProjectDir, ballID, envB.ProjectDir)
	if !strings.Contains(output, "Successfully moved") {
		t.Errorf("Expected success message, got: %s", output)
	}

	// Verify removed from A
	storeA := envA.GetStore(t)
	ballsA, err := storeA.LoadBalls()
	if err != nil {
		t.Fatalf("Failed to load balls from A: %v", err)
	}
	if len(ballsA) != 0 {
		t.Errorf("Expected 0 balls in project A, got %d", len(ballsA))
	}

	// Verify added to B
	storeB := envB.GetStore(t)
	ballsB, err := storeB.LoadBalls()
	if err != nil {
		t.Fatalf("Failed to load balls from B: %v", err)
	}
	if len(ballsB) != 1 {
		t.Errorf("Expected 1 ball in project B, got %d", len(ballsB))
	}
	if len(ballsB) > 0 && ballsB[0].ID != ballID {
		t.Errorf("Expected ball ID %s, got %s", ballID, ballsB[0].ID)
	}
}

func testMoveUpdatesWorkingDir(t *testing.T) {
	envA := SetupTestEnv(t)
	defer CleanupTestEnv(t, envA)
	envB := SetupTestEnv(t)
	defer CleanupTestEnv(t, envB)

	// Initialize .juggle directory in project B
	_ = envB.GetStore(t)

	// Setup config with both projects
	setupConfigWithProjects(t, envA.ConfigHome, envA.ProjectDir, envB.ProjectDir)

	// Create ball in project A
	ball := envA.CreateBall(t, "Test", session.PriorityMedium)
	ballID := ball.ID

	// Move to project B
	runMoveCommand(t, envA.ProjectDir, ballID, envB.ProjectDir)

	// Load ball from B and verify working directory
	storeB := envB.GetStore(t)
	ballsB, err := storeB.LoadBalls()
	if err != nil {
		t.Fatalf("Failed to load balls from B: %v", err)
	}
	if len(ballsB) != 1 {
		t.Fatalf("Expected 1 ball, got %d", len(ballsB))
	}

	if ballsB[0].WorkingDir != envB.ProjectDir {
		t.Errorf("Expected working dir %s, got %s", envB.ProjectDir, ballsB[0].WorkingDir)
	}
}

func testMoveNonexistentBall(t *testing.T) {
	envA := SetupTestEnv(t)
	defer CleanupTestEnv(t, envA)
	envB := SetupTestEnv(t)
	defer CleanupTestEnv(t, envB)

	// Initialize .juggle directories
	_ = envA.GetStore(t)
	_ = envB.GetStore(t)

	// Setup config with both projects
	setupConfigWithProjects(t, envA.ConfigHome, envA.ProjectDir, envB.ProjectDir)

	output, exitCode := runMoveCommandWithError(t, envA.ProjectDir, "nonexistent-1", envB.ProjectDir)
	if exitCode == 0 {
		t.Error("Expected non-zero exit code for nonexistent ball")
	}
	if !strings.Contains(output, "not found") && !strings.Contains(output, "failed to find") {
		t.Errorf("Expected 'not found' in error, got: %s", output)
	}
}

func testMoveToNonJuggleProject(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	ball := env.CreateBall(t, "Test", session.PriorityMedium)

	// Create temp dir without .juggle
	nonJuggleDir, err := os.MkdirTemp("", "non-juggle-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(nonJuggleDir)

	output, exitCode := runMoveCommandWithError(t, env.ProjectDir, ball.ID, nonJuggleDir)
	if exitCode == 0 {
		t.Error("Expected non-zero exit code for non-juggle project")
	}
	if !strings.Contains(output, "not a juggle project") {
		t.Errorf("Expected 'not a juggle project' in error, got: %s", output)
	}
}

func testMovePreservesMetadata(t *testing.T) {
	envA := SetupTestEnv(t)
	defer CleanupTestEnv(t, envA)
	envB := SetupTestEnv(t)
	defer CleanupTestEnv(t, envB)

	// Initialize .juggle directory in project B
	_ = envB.GetStore(t)

	// Setup config with both projects
	setupConfigWithProjects(t, envA.ConfigHome, envA.ProjectDir, envB.ProjectDir)

	// Create ball with metadata
	ball := envA.CreateBall(t, "Complex task", session.PriorityHigh)
	ball.SetAcceptanceCriteria([]string{"First criterion", "Second criterion"})
	ball.AddTag("feature")
	ball.AddTag("urgent")

	storeA := envA.GetStore(t)
	if err := storeA.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}
	ballID := ball.ID

	// Move to project B
	runMoveCommand(t, envA.ProjectDir, ballID, envB.ProjectDir)

	// Load ball from B and verify metadata
	storeB := envB.GetStore(t)
	movedBall, err := storeB.GetBallByID(ballID)
	if err != nil {
		t.Fatalf("Failed to load moved ball: %v", err)
	}

	// Verify intent
	if movedBall.Title != "Complex task" {
		t.Errorf("Expected intent 'Complex task', got %s", movedBall.Title)
	}

	// Verify priority
	if movedBall.Priority != session.PriorityHigh {
		t.Errorf("Expected priority %s, got %s", session.PriorityHigh, movedBall.Priority)
	}

	// Verify acceptance criteria
	if len(movedBall.AcceptanceCriteria) != 2 {
		t.Errorf("Expected 2 acceptance criteria, got %d", len(movedBall.AcceptanceCriteria))
	}

	// Verify tags
	if len(movedBall.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(movedBall.Tags))
	}
}

func testMoveRemovesFromSource(t *testing.T) {
	envA := SetupTestEnv(t)
	defer CleanupTestEnv(t, envA)
	envB := SetupTestEnv(t)
	defer CleanupTestEnv(t, envB)

	// Initialize .juggle directory in project B
	_ = envB.GetStore(t)

	// Setup config with both projects
	setupConfigWithProjects(t, envA.ConfigHome, envA.ProjectDir, envB.ProjectDir)

	// Create two balls in project A
	ball1 := envA.CreateBall(t, "Ball to keep", session.PriorityMedium)
	ball2 := envA.CreateBall(t, "Ball to move", session.PriorityHigh)

	// Verify both exist in A
	storeA := envA.GetStore(t)
	ballsA, err := storeA.LoadBalls()
	if err != nil {
		t.Fatalf("Failed to load balls: %v", err)
	}
	if len(ballsA) != 2 {
		t.Fatalf("Expected 2 balls in A, got %d", len(ballsA))
	}

	// Move ball2 to project B
	runMoveCommand(t, envA.ProjectDir, ball2.ID, envB.ProjectDir)

	// Verify ball2 is gone from A but ball1 remains
	ballsA, err = storeA.LoadBalls()
	if err != nil {
		t.Fatalf("Failed to load balls: %v", err)
	}
	if len(ballsA) != 1 {
		t.Errorf("Expected 1 ball remaining in A, got %d", len(ballsA))
	}
	if len(ballsA) > 0 && ballsA[0].ID != ball1.ID {
		t.Errorf("Expected remaining ball to be %s, got %s", ball1.ID, ballsA[0].ID)
	}

	// Verify ball2 is in B
	storeB := envB.GetStore(t)
	ballsB, err := storeB.LoadBalls()
	if err != nil {
		t.Fatalf("Failed to load balls from B: %v", err)
	}
	if len(ballsB) != 1 {
		t.Errorf("Expected 1 ball in B, got %d", len(ballsB))
	}
	if len(ballsB) > 0 && ballsB[0].ID != ball2.ID {
		t.Errorf("Expected moved ball to be %s, got %s", ball2.ID, ballsB[0].ID)
	}
}

func testMoveWithShortID(t *testing.T) {
	envA := SetupTestEnv(t)
	defer CleanupTestEnv(t, envA)
	envB := SetupTestEnv(t)
	defer CleanupTestEnv(t, envB)

	// Initialize .juggle directory in project B
	_ = envB.GetStore(t)

	// Setup config with both projects
	setupConfigWithProjects(t, envA.ConfigHome, envA.ProjectDir, envB.ProjectDir)

	// Create ball in project A
	ball := envA.CreateBall(t, "Test", session.PriorityMedium)
	fullID := ball.ID

	// Extract short ID
	parts := strings.Split(fullID, "-")
	if len(parts) < 2 {
		t.Fatalf("Expected ball ID to have at least 2 parts, got %d", len(parts))
	}
	shortID := parts[len(parts)-1]

	// Move using short ID
	output := runMoveCommand(t, envA.ProjectDir, shortID, envB.ProjectDir)
	if !strings.Contains(output, "Successfully moved") {
		t.Errorf("Expected success message, got: %s", output)
	}

	// Verify ball moved
	storeB := envB.GetStore(t)
	ballsB, err := storeB.LoadBalls()
	if err != nil {
		t.Fatalf("Failed to load balls from B: %v", err)
	}
	if len(ballsB) != 1 {
		t.Errorf("Expected 1 ball in B, got %d", len(ballsB))
	}
	if len(ballsB) > 0 && ballsB[0].ID != fullID {
		t.Errorf("Expected ball ID %s, got %s", fullID, ballsB[0].ID)
	}
}

func testMoveToSameProject(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Setup config with project
	setupConfigWithProjects(t, env.ConfigHome, env.ProjectDir)

	ball := env.CreateBall(t, "Test", session.PriorityMedium)

	// Try to move to same project
	output, exitCode := runMoveCommandWithError(t, env.ProjectDir, ball.ID, env.ProjectDir)
	if exitCode == 0 {
		t.Error("Expected non-zero exit code when moving to same project")
	}
	if !strings.Contains(output, "already in the target project") {
		t.Errorf("Expected 'already in the target project' in error, got: %s", output)
	}
}

// Helper functions for move tests

// setupConfigWithProjects creates a config file with the given projects in search paths
func setupConfigWithProjects(t *testing.T, configHome string, projectDirs ...string) {
	t.Helper()

	// Create config with project directories in search paths
	config := &session.Config{
		SearchPaths: projectDirs,
	}

	if err := config.SaveWithOptions(session.ConfigOptions{
		ConfigHome:     configHome,
		JuggleDirName: ".juggle",
	}); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}
}

func runMoveCommand(t *testing.T, workingDir string, ballID string, targetDir string) string {
	t.Helper()

	juggleRoot := GetRepoRoot(t)
	juggleBinary := filepath.Join(juggleRoot, "juggle")

	// Build juggle binary if it doesn't exist
	if _, err := os.Stat(juggleBinary); os.IsNotExist(err) {
		buildCmd := exec.Command("go", "build", "-o", "juggle", "./cmd/juggle")
		buildCmd.Dir = juggleRoot
		if output, err := buildCmd.CombinedOutput(); err != nil {
			t.Fatalf("Failed to build juggle: %v\nOutput: %s", err, output)
		}
	}

	// Inject --config-home flag pointing to test config directory
	configHome := filepath.Join(workingDir, "..", "config")
	cmd := exec.Command(juggleBinary, "--config-home", configHome, "move", ballID, targetDir)
	cmd.Dir = workingDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Move command failed: %v\nOutput: %s", err, output)
	}

	return strings.TrimSpace(string(output))
}

func runMoveCommandWithError(t *testing.T, workingDir string, ballID string, targetDir string) (string, int) {
	t.Helper()

	juggleRoot := GetRepoRoot(t)
	juggleBinary := filepath.Join(juggleRoot, "juggle")

	// Build binary if needed
	if _, err := os.Stat(juggleBinary); os.IsNotExist(err) {
		buildCmd := exec.Command("go", "build", "-o", "juggle", "./cmd/juggle")
		buildCmd.Dir = juggleRoot
		if output, err := buildCmd.CombinedOutput(); err != nil {
			t.Fatalf("Failed to build juggle: %v\nOutput: %s", err, output)
		}
	}

	// Inject --config-home flag pointing to test config directory
	configHome := filepath.Join(workingDir, "..", "config")
	cmd := exec.Command(juggleBinary, "--config-home", configHome, "move", ballID, targetDir)
	cmd.Dir = workingDir

	output, err := cmd.CombinedOutput()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}

	return strings.TrimSpace(string(output)), exitCode
}
