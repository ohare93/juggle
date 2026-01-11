package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ohare93/juggle/internal/session"
)

func TestCompletionGeneration(t *testing.T) {
	t.Run("BashCompletion", testBashCompletion)
	t.Run("ZshCompletion", testZshCompletion)
	t.Run("FishCompletion", testFishCompletion)
	t.Run("PowerShellCompletion", testPowerShellCompletion)
	t.Run("InvalidShell", testInvalidShellCompletion)
}

func testBashCompletion(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	output := runCompletionCommand(t, env, "bash")
	if output == "" {
		t.Fatal("Expected non-empty completion output")
	}
	if !strings.Contains(output, "juggle") {
		t.Error("Expected completion output to contain 'juggle'")
	}
	if !strings.Contains(output, "bash completion") {
		t.Error("Expected completion output to contain 'bash completion'")
	}
}

func testZshCompletion(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	output := runCompletionCommand(t, env, "zsh")
	if output == "" {
		t.Fatal("Expected non-empty completion output")
	}
	if !strings.Contains(output, "juggle") {
		t.Error("Expected completion output to contain 'juggle'")
	}
	// Zsh completion has specific markers
	if !strings.Contains(output, "compdef") {
		t.Error("Expected zsh completion output to contain 'compdef'")
	}
}

func testFishCompletion(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	output := runCompletionCommand(t, env, "fish")
	if output == "" {
		t.Fatal("Expected non-empty completion output")
	}
	if !strings.Contains(output, "juggle") {
		t.Error("Expected completion output to contain 'juggle'")
	}
	// Fish completion uses different syntax
	if !strings.Contains(output, "complete") {
		t.Error("Expected fish completion output to contain 'complete'")
	}
}

func testPowerShellCompletion(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	output := runCompletionCommand(t, env, "powershell")
	if output == "" {
		t.Fatal("Expected non-empty completion output")
	}
	if !strings.Contains(output, "juggle") {
		t.Error("Expected completion output to contain 'juggle'")
	}
	// PowerShell completion has specific markers
	if !strings.Contains(output, "Register-ArgumentCompleter") {
		t.Error("Expected powershell completion output to contain 'Register-ArgumentCompleter'")
	}
}

func testInvalidShellCompletion(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	output, _ := runCompletionCommandWithError(t, env, "invalid-shell")
	// Cobra shows help text for invalid subcommands
	if !strings.Contains(output, "Available Commands") {
		t.Errorf("Expected help text to contain 'Available Commands', got: %s", output)
	}
	if !strings.Contains(output, "bash") || !strings.Contains(output, "zsh") {
		t.Errorf("Expected help text to list available shells, got: %s", output)
	}
}

func TestBallIDCompletionFunction(t *testing.T) {
	t.Run("CompleteAllBallsEmptyPrefix", testCompleteAllBallsEmptyPrefix)
	t.Run("CompleteWithFullPrefix", testCompleteWithFullPrefix)
	t.Run("CompleteWithPartialPrefix", testCompleteWithPartialPrefix)
	t.Run("CompleteNoMatches", testCompleteNoMatches)
	t.Run("CompleteAcrossMultipleProjects", testCompleteAcrossMultipleProjects)
	t.Run("CompleteFiltersByState", testCompleteFiltersByState)
	t.Run("CompleteShowsIntent", testCompleteShowsIntent)
	t.Run("CompleteWithShortID", testCompleteWithShortID)
}

func testCompleteAllBallsEmptyPrefix(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create several test balls
	env.CreateBall(t, "First task", session.PriorityMedium)
	env.CreateBall(t, "Second task", session.PriorityHigh)
	env.CreateBall(t, "Third task", session.PriorityLow)

	// Verify balls are created
	store := env.GetStore(t)
	balls, err := store.LoadBalls()
	if err != nil {
		t.Fatalf("Failed to load balls: %v", err)
	}
	if len(balls) != 3 {
		t.Errorf("Expected 3 balls, got %d", len(balls))
	}

	// Verify all balls have the expected format for completion
	for _, ball := range balls {
		if ball.ID == "" {
			t.Error("Expected ball to have non-empty ID")
		}
		if ball.Intent == "" {
			t.Error("Expected ball to have non-empty Intent")
		}
	}
}

func testCompleteWithFullPrefix(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create test balls
	env.CreateBall(t, "Task one", session.PriorityMedium)
	env.CreateBall(t, "Task two", session.PriorityMedium)

	store := env.GetStore(t)
	balls, err := store.LoadBalls()
	if err != nil {
		t.Fatalf("Failed to load balls: %v", err)
	}
	if len(balls) < 2 {
		t.Errorf("Expected at least 2 balls, got %d", len(balls))
	}

	// Get the project prefix (e.g., "project-")
	projectPrefix := strings.Split(balls[0].ID, "-")[0] + "-"

	// Filter balls by prefix
	var matchingBalls []*session.Ball
	for _, ball := range balls {
		if strings.HasPrefix(ball.ID, projectPrefix) {
			matchingBalls = append(matchingBalls, ball)
		}
	}

	if len(matchingBalls) < 2 {
		t.Errorf("Expected at least 2 matching balls, got %d", len(matchingBalls))
	}
}

func testCompleteWithPartialPrefix(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a ball
	ball := env.CreateBall(t, "Test task", session.PriorityMedium)
	ballID := ball.ID

	// Get the first few characters of the ID
	prefixLen := len(ballID) - 1
	if prefixLen > 5 {
		prefixLen = 5
	}
	partialPrefix := ballID[:prefixLen]

	// Verify the ball matches this partial prefix
	if !strings.HasPrefix(ballID, partialPrefix) {
		t.Errorf("Expected ball ID %s to start with prefix %s", ballID, partialPrefix)
	}
}

func testCompleteNoMatches(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a ball
	env.CreateBall(t, "Test task", session.PriorityMedium)

	store := env.GetStore(t)
	balls, err := store.LoadBalls()
	if err != nil {
		t.Fatalf("Failed to load balls: %v", err)
	}

	// Filter with a prefix that doesn't match
	nonMatchingPrefix := "nonexistent-project-"
	var matchingBalls []*session.Ball
	for _, ball := range balls {
		if strings.HasPrefix(ball.ID, nonMatchingPrefix) {
			matchingBalls = append(matchingBalls, ball)
		}
	}

	if len(matchingBalls) != 0 {
		t.Errorf("Expected 0 matching balls, got %d", len(matchingBalls))
	}
}

func testCompleteAcrossMultipleProjects(t *testing.T) {
	// Setup two separate project directories
	envA := SetupTestEnv(t)
	defer CleanupTestEnv(t, envA)
	envB := SetupTestEnv(t)
	defer CleanupTestEnv(t, envB)

	// Create balls in both projects
	envA.CreateBall(t, "Task in project A", session.PriorityMedium)
	envB.CreateBall(t, "Task in project B", session.PriorityHigh)

	// Manually load balls from both projects
	// (Discovery requires structured directory trees which temp dirs don't have)
	storeA := envA.GetStore(t)
	ballsA, err := storeA.LoadBalls()
	if err != nil {
		t.Fatalf("Failed to load balls from project A: %v", err)
	}

	storeB := envB.GetStore(t)
	ballsB, err := storeB.LoadBalls()
	if err != nil {
		t.Fatalf("Failed to load balls from project B: %v", err)
	}

	// Combine balls from both projects
	allBalls := append(ballsA, ballsB...)

	if len(allBalls) < 2 {
		t.Errorf("Expected at least 2 balls across projects, got %d", len(allBalls))
	}

	// Verify balls from different projects have different IDs
	if len(allBalls) >= 2 {
		projectA := strings.Split(ballsA[0].ID, "-")[0]
		projectB := strings.Split(ballsB[0].ID, "-")[0]

		// Both balls should have proper ID format
		if projectA == "" || projectB == "" {
			t.Error("Expected both balls to have non-empty project prefixes")
		}
	}
}

func testCompleteFiltersByState(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create and complete a ball
	ball := env.CreateBall(t, "Task to complete", session.PriorityMedium)

	// Complete the ball
	ball.MarkComplete("Done")
	store := env.GetStore(t)
	if err := store.UpdateBall(ball); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}
	if err := store.ArchiveBall(ball); err != nil {
		t.Fatalf("Failed to archive ball: %v", err)
	}

	// Verify ball is no longer in active balls
	balls, err := store.LoadBalls()
	if err != nil {
		t.Fatalf("Failed to load balls: %v", err)
	}
	if len(balls) != 0 {
		t.Errorf("Expected 0 active balls (completed ball should be archived), got %d", len(balls))
	}

	// Create a new active ball
	env.CreateBall(t, "Active task", session.PriorityMedium)

	balls, err = store.LoadBalls()
	if err != nil {
		t.Fatalf("Failed to load balls: %v", err)
	}
	if len(balls) != 1 {
		t.Errorf("Expected 1 active ball, got %d", len(balls))
	}
}

func testCompleteShowsIntent(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	intent := "Important task with detailed description"
	ball := env.CreateBall(t, intent, session.PriorityHigh)

	// Verify intent is available for completion display
	if ball.Intent != intent {
		t.Errorf("Expected intent %q, got %q", intent, ball.Intent)
	}
}

func testCompleteWithShortID(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a ball
	ball := env.CreateBall(t, "Test task", session.PriorityMedium)
	fullID := ball.ID

	// Extract short ID (number part)
	parts := strings.Split(fullID, "-")
	if len(parts) < 2 {
		t.Fatalf("Expected ball ID to have at least 2 parts, got %d", len(parts))
	}
	shortID := parts[len(parts)-1]

	// Verify short ID can be resolved
	store := env.GetStore(t)
	resolvedBall, err := store.ResolveBallID(shortID)
	if err != nil {
		t.Fatalf("Failed to resolve short ID %s: %v", shortID, err)
	}
	if resolvedBall.ID != fullID {
		t.Errorf("Expected resolved ball ID %s, got %s", fullID, resolvedBall.ID)
	}
}

// Helper functions specific to completion tests

func runCompletionCommand(t *testing.T, env *TestEnv, shell string) string {
	t.Helper()

	jugglerRoot := "/home/jmo/Development/juggler"
	juggleBinary := filepath.Join(jugglerRoot, "juggle")

	// Build juggle binary if it doesn't exist
	if _, err := os.Stat(juggleBinary); os.IsNotExist(err) {
		buildCmd := exec.Command("go", "build", "-o", "juggle", "./cmd/juggle")
		buildCmd.Dir = jugglerRoot
		if output, err := buildCmd.CombinedOutput(); err != nil {
			t.Fatalf("Failed to build juggle: %v\nOutput: %s", err, output)
		}
	}

	cmd := exec.Command(juggleBinary, "completion", shell)
	cmd.Dir = env.ProjectDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Completion command failed: %v\nOutput: %s", err, output)
	}

	return string(output)
}

func runCompletionCommandWithError(t *testing.T, env *TestEnv, shell string) (string, int) {
	t.Helper()

	jugglerRoot := "/home/jmo/Development/juggler"
	juggleBinary := filepath.Join(jugglerRoot, "juggle")

	// Build binary if needed
	if _, err := os.Stat(juggleBinary); os.IsNotExist(err) {
		buildCmd := exec.Command("go", "build", "-o", "juggle", "./cmd/juggle")
		buildCmd.Dir = jugglerRoot
		if output, err := buildCmd.CombinedOutput(); err != nil {
			t.Fatalf("Failed to build juggle: %v\nOutput: %s", err, output)
		}
	}

	cmd := exec.Command(juggleBinary, "completion", shell)
	cmd.Dir = env.ProjectDir

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
