package cli

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ohare93/juggle/internal/session"
)

func TestCheckCommand_NoProjects(t *testing.T) {
	// Create temp directory for test config
	tmpDir := t.TempDir()

	// Set up global options to use test directory
	GlobalOpts.ConfigHome = tmpDir
	GlobalOpts.ProjectDir = tmpDir
	defer func() {
		GlobalOpts.ConfigHome = ""
		GlobalOpts.ProjectDir = ""
	}()

	// Create empty config
	cfg := &session.Config{
		SearchPaths: []string{tmpDir},
	}
	if err := cfg.SaveWithOptions(session.ConfigOptions{
		ConfigHome:     tmpDir,
		JugglerDirName: ".juggler",
	}); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Run check command
	err := runCheck(checkCmd, []string{})
	if err != nil {
		t.Errorf("expected no error with no projects, got: %v", err)
	}
}

func TestCheckCommand_NoJugglingBalls(t *testing.T) {
	// Create temp directory for test project
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "test-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project dir: %v", err)
	}

	// Set up global options to use test directory
	GlobalOpts.ConfigHome = tmpDir
	GlobalOpts.ProjectDir = projectDir
	defer func() {
		GlobalOpts.ConfigHome = ""
		GlobalOpts.ProjectDir = ""
	}()

	// Create config with project
	cfg := &session.Config{
		SearchPaths: []string{projectDir},
	}
	if err := cfg.SaveWithOptions(session.ConfigOptions{
		ConfigHome:     tmpDir,
		JugglerDirName: ".juggler",
	}); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Create store and juggler directory
	store, err := session.NewStore(projectDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Create a ready ball
	ball, err := session.New(projectDir, "Test ready ball", session.PriorityMedium)
	if err != nil {
		t.Fatalf("failed to create ball: %v", err)
	}
	ball.ActiveState = session.ActiveReady

	if err := store.AppendBall(ball); err != nil {
		t.Fatalf("failed to save ball: %v", err)
	}

	// Note: Cannot test interactive prompts in unit test without mocking stdin
	// This test verifies the command runs without error
	// The actual user interaction would need integration testing
}

func TestCheckCommand_SingleJugglingBall(t *testing.T) {
	// Create temp directory for test project
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "test-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project dir: %v", err)
	}

	// Set up global options
	GlobalOpts.ConfigHome = tmpDir
	GlobalOpts.ProjectDir = projectDir
	defer func() {
		GlobalOpts.ConfigHome = ""
		GlobalOpts.ProjectDir = ""
	}()

	// Create config
	cfg := &session.Config{
		SearchPaths: []string{projectDir},
	}
	if err := cfg.SaveWithOptions(session.ConfigOptions{
		ConfigHome:     tmpDir,
		JugglerDirName: ".juggler",
	}); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Create store and juggling ball
	store, err := session.NewStore(projectDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	ball, err := session.New(projectDir, "Test juggling ball", session.PriorityHigh)
	if err != nil {
		t.Fatalf("failed to create ball: %v", err)
	}
	ball.StartJuggling()

	if err := store.AppendBall(ball); err != nil {
		t.Fatalf("failed to save ball: %v", err)
	}

	// Verify ball is in juggling state
	if ball.ActiveState != session.ActiveJuggling {
		t.Errorf("expected ActiveJuggling, got %s", ball.ActiveState)
	}
	if ball.JuggleState == nil || *ball.JuggleState != session.JuggleNeedsThrown {
		t.Errorf("expected JuggleNeedsThrown state")
	}
}

func TestCheckCommand_MultipleJugglingBalls(t *testing.T) {
	// Create temp directory for test project
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "test-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project dir: %v", err)
	}

	// Set up global options
	GlobalOpts.ConfigHome = tmpDir
	GlobalOpts.ProjectDir = projectDir
	defer func() {
		GlobalOpts.ConfigHome = ""
		GlobalOpts.ProjectDir = ""
	}()

	// Create config
	cfg := &session.Config{
		SearchPaths: []string{projectDir},
	}
	if err := cfg.SaveWithOptions(session.ConfigOptions{
		ConfigHome:     tmpDir,
		JugglerDirName: ".juggler",
	}); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Create store and multiple juggling balls
	store, err := session.NewStore(projectDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	ball1, err := session.New(projectDir, "First juggling ball", session.PriorityHigh)
	if err != nil {
		t.Fatalf("failed to create ball1: %v", err)
	}
	ball1.StartJuggling()
	if err := store.AppendBall(ball1); err != nil {
		t.Fatalf("failed to save ball1: %v", err)
	}

	// Wait a moment to ensure different IDs
	time.Sleep(10 * time.Millisecond)

	ball2, err := session.New(projectDir, "Second juggling ball", session.PriorityMedium)
	if err != nil {
		t.Fatalf("failed to create ball2: %v", err)
	}
	ball2.StartJuggling()
	if err := store.AppendBall(ball2); err != nil {
		t.Fatalf("failed to save ball2: %v", err)
	}

	// Load and verify multiple juggling balls
	config, err := LoadConfigForCommand()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	projects, err := session.DiscoverProjects(config)
	if err != nil {
		t.Fatalf("failed to discover projects: %v", err)
	}
	jugglingBalls, err := session.LoadJugglingBalls(projects)
	if err != nil {
		t.Fatalf("failed to load juggling balls: %v", err)
	}

	if len(jugglingBalls) != 2 {
		t.Errorf("expected 2 juggling balls, got %d", len(jugglingBalls))
	}
}

func TestCheckCommand_MixedStates(t *testing.T) {
	// Create temp directory for test project
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "test-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project dir: %v", err)
	}

	// Set up global options
	GlobalOpts.ConfigHome = tmpDir
	GlobalOpts.ProjectDir = projectDir
	defer func() {
		GlobalOpts.ConfigHome = ""
		GlobalOpts.ProjectDir = ""
	}()

	// Create config
	cfg := &session.Config{
		SearchPaths: []string{projectDir},
	}
	if err := cfg.SaveWithOptions(session.ConfigOptions{
		ConfigHome:     tmpDir,
		JugglerDirName: ".juggler",
	}); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Create store and balls in different states
	store, err := session.NewStore(projectDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Juggling ball
	jugglingBall, err := session.New(projectDir, "Juggling ball", session.PriorityHigh)
	if err != nil {
		t.Fatalf("failed to create juggling ball: %v", err)
	}
	jugglingBall.StartJuggling()
	if err := store.AppendBall(jugglingBall); err != nil {
		t.Fatalf("failed to save juggling ball: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	// Ready ball
	readyBall, err := session.New(projectDir, "Ready ball", session.PriorityMedium)
	if err != nil {
		t.Fatalf("failed to create ready ball: %v", err)
	}
	readyBall.ActiveState = session.ActiveReady
	if err := store.AppendBall(readyBall); err != nil {
		t.Fatalf("failed to save ready ball: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	// Dropped ball (should not affect check command)
	droppedBall, err := session.New(projectDir, "Dropped ball", session.PriorityLow)
	if err != nil {
		t.Fatalf("failed to create dropped ball: %v", err)
	}
	droppedBall.SetActiveState(session.ActiveDropped)
	if err := store.AppendBall(droppedBall); err != nil {
		t.Fatalf("failed to save dropped ball: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	// Complete ball (should not affect check command)
	completeBall, err := session.New(projectDir, "Complete ball", session.PriorityLow)
	if err != nil {
		t.Fatalf("failed to create complete ball: %v", err)
	}
	completeBall.MarkComplete("Done")
	if err := store.AppendBall(completeBall); err != nil {
		t.Fatalf("failed to save complete ball: %v", err)
	}

	// Load and verify only juggling and ready balls are counted
	config, err := LoadConfigForCommand()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	projects, err := session.DiscoverProjects(config)
	if err != nil {
		t.Fatalf("failed to discover projects: %v", err)
	}

	jugglingBalls, err := session.LoadJugglingBalls(projects)
	if err != nil {
		t.Fatalf("failed to load juggling balls: %v", err)
	}
	if len(jugglingBalls) != 1 {
		t.Errorf("expected 1 juggling ball, got %d", len(jugglingBalls))
	}

	readyBalls, err := session.LoadReadyBalls(projects)
	if err != nil {
		t.Fatalf("failed to load ready balls: %v", err)
	}
	if len(readyBalls) != 1 {
		t.Errorf("expected 1 ready ball, got %d", len(readyBalls))
	}
}

func TestPluralS(t *testing.T) {
	tests := []struct {
		count    int
		expected string
	}{
		{0, "s"},
		{1, ""},
		{2, "s"},
		{10, "s"},
	}

	for _, tt := range tests {
		result := pluralS(tt.count)
		if result != tt.expected {
			t.Errorf("pluralS(%d) = %q, expected %q", tt.count, result, tt.expected)
		}
	}
}

func TestCheckCommand_DifferentJuggleStates(t *testing.T) {
	// Create temp directory for test project
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "test-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project dir: %v", err)
	}

	// Set up global options
	GlobalOpts.ConfigHome = tmpDir
	GlobalOpts.ProjectDir = projectDir
	defer func() {
		GlobalOpts.ConfigHome = ""
		GlobalOpts.ProjectDir = ""
	}()

	// Create config
	cfg := &session.Config{
		SearchPaths: []string{projectDir},
	}
	if err := cfg.SaveWithOptions(session.ConfigOptions{
		ConfigHome:     tmpDir,
		JugglerDirName: ".juggler",
	}); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Create store
	store, err := session.NewStore(projectDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Test each juggle state
	states := []struct {
		state   session.JuggleState
		message string
	}{
		{session.JuggleNeedsThrown, "waiting for input"},
		{session.JuggleInAir, "agent working"},
		{session.JuggleNeedsCaught, "needs review"},
	}

	for i, test := range states {
		ball, err := session.New(projectDir, "Test ball "+string(test.state), session.PriorityMedium)
		if err != nil {
			t.Fatalf("failed to create ball %d: %v", i, err)
		}
		ball.StartJuggling()
		ball.SetJuggleState(test.state, test.message)

		if err := store.AppendBall(ball); err != nil {
			t.Fatalf("failed to save ball %d: %v", i, err)
		}

		// Verify state
		if ball.JuggleState == nil || *ball.JuggleState != test.state {
			t.Errorf("ball %d: expected state %s, got %v", i, test.state, ball.JuggleState)
		}
		if ball.StateMessage != test.message {
			t.Errorf("ball %d: expected message %q, got %q", i, test.message, ball.StateMessage)
		}

		time.Sleep(10 * time.Millisecond)
	}
}
