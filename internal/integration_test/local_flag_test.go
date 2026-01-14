package integration_test

import (
	"testing"

	"github.com/ohare93/juggle/internal/cli"
	"github.com/ohare93/juggle/internal/session"
)

func TestLocalOnlyBehavior(t *testing.T) {
	// Create two test projects
	project1 := t.TempDir()
	project2 := t.TempDir()

	// Setup project 1
	store1, err := session.NewStoreWithConfig(project1, session.StoreConfig{JuggleDirName: ".juggle"})
	if err != nil {
		t.Fatalf("Failed to create store1: %v", err)
	}

	ball1 := &session.Ball{
		ID:         "test1-1",
		WorkingDir: project1,
		Title:     "Ball in project 1",
		Priority:   session.PriorityMedium,
		State:      session.StateInProgress,
	}
	if err := store1.Save(ball1); err != nil {
		t.Fatalf("Failed to save ball1: %v", err)
	}

	// Setup project 2
	store2, err := session.NewStoreWithConfig(project2, session.StoreConfig{JuggleDirName: ".juggle"})
	if err != nil {
		t.Fatalf("Failed to create store2: %v", err)
	}

	ball2 := &session.Ball{
		ID:         "test2-1",
		WorkingDir: project2,
		Title:     "Ball in project 2",
		Priority:   session.PriorityMedium,
		State:      session.StateInProgress,
	}
	if err := store2.Save(ball2); err != nil {
		t.Fatalf("Failed to save ball2: %v", err)
	}

	// Create a config with both projects in search paths
	config := &session.Config{
		SearchPaths: []string{project1, project2},
	}

	t.Run("DiscoverProjectsForCommand_DefaultLocal", func(t *testing.T) {
		// Default is local only (no --all flag)
		cli.GlobalOpts.AllProjects = false
		cli.GlobalOpts.ProjectDir = project1

		projects, err := cli.DiscoverProjectsForCommand(config, store1)
		if err != nil {
			t.Fatalf("Failed to discover projects: %v", err)
		}

		// Should only return current project by default (local only)
		if len(projects) != 1 {
			t.Errorf("Expected 1 project by default (local only), got %d", len(projects))
		}

		if len(projects) > 0 && projects[0] != project1 {
			t.Errorf("Expected current project %s, got %s", project1, projects[0])
		}
	})

	t.Run("DiscoverProjectsForCommand_WithAll", func(t *testing.T) {
		// Use --all to enable cross-project discovery
		cli.GlobalOpts.AllProjects = true
		cli.GlobalOpts.ProjectDir = project1

		projects, err := cli.DiscoverProjectsForCommand(config, store1)
		if err != nil {
			t.Fatalf("Failed to discover projects: %v", err)
		}

		// Should discover both projects with --all
		if len(projects) != 2 {
			t.Errorf("Expected 2 projects with --all, got %d", len(projects))
		}

		// Verify we get both project paths
		foundProj1, foundProj2 := false, false
		for _, p := range projects {
			if p == project1 {
				foundProj1 = true
			}
			if p == project2 {
				foundProj2 = true
			}
		}
		if !foundProj1 || !foundProj2 {
			t.Errorf("Expected to find both projects, got: %v", projects)
		}

		// Reset for next test
		cli.GlobalOpts.AllProjects = false
	})

	t.Run("RootCommand_DefaultLocal", func(t *testing.T) {
		cli.GlobalOpts.AllProjects = false
		cli.GlobalOpts.ProjectDir = project1

		config, err := cli.LoadConfigForCommand()
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		cwd, _ := cli.GetWorkingDir()
		store, _ := cli.NewStoreForCommand(cwd)
		projects, err := cli.DiscoverProjectsForCommand(config, store)
		if err != nil {
			t.Fatalf("Failed to discover projects: %v", err)
		}

		// Should only return current project
		if len(projects) != 1 {
			t.Errorf("Expected 1 project by default (local), got %d", len(projects))
		}

		if len(projects) > 0 && projects[0] != project1 {
			t.Errorf("Expected current project %s, got %s", project1, projects[0])
		}

		balls, err := session.LoadJugglingBalls(projects)
		if err != nil {
			t.Fatalf("Failed to load balls: %v", err)
		}

		// Should only find ball from project1
		if len(balls) != 1 {
			t.Errorf("Expected 1 ball by default (local), got %d", len(balls))
		}

		if len(balls) > 0 && balls[0].WorkingDir != project1 {
			t.Errorf("Expected ball from project1, got ball from %s", balls[0].WorkingDir)
		}
	})

	t.Run("StatusCommand_DefaultLocal", func(t *testing.T) {
		// Test by default (local) flag
		cli.GlobalOpts.AllProjects = false
		cli.GlobalOpts.ProjectDir = project1

		config, err := cli.LoadConfigForCommand()
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		cwd, _ := cli.GetWorkingDir()
		store, _ := cli.NewStoreForCommand(cwd)
		projects, err := cli.DiscoverProjectsForCommand(config, store)
		if err != nil {
			t.Fatalf("Failed to discover projects: %v", err)
		}

		if len(projects) != 1 {
			t.Errorf("Status by default (local) should only show 1 project, got %d", len(projects))
		}
	})

	t.Run("NextCommand_DefaultLocal", func(t *testing.T) {
		// Test by default (local) flag
		cli.GlobalOpts.AllProjects = false
		cli.GlobalOpts.ProjectDir = project1

		config, err := cli.LoadConfigForCommand()
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		cwd, _ := cli.GetWorkingDir()
		store, _ := cli.NewStoreForCommand(cwd)
		projects, err := cli.DiscoverProjectsForCommand(config, store)
		if err != nil {
			t.Fatalf("Failed to discover projects: %v", err)
		}

		balls, err := session.LoadJugglingBalls(projects)
		if err != nil {
			t.Fatalf("Failed to load balls: %v", err)
		}

		// Should only find ball from current project
		if len(balls) != 1 {
			t.Errorf("Next by default (local) should find 1 ball, got %d", len(balls))
		}

		if len(balls) > 0 && balls[0].WorkingDir != project1 {
			t.Errorf("Next by default (local) should find ball from project1, got ball from %s", balls[0].WorkingDir)
		}
	})

	t.Run("SearchCommand_DefaultLocal", func(t *testing.T) {
		// Test by default (local) flag
		cli.GlobalOpts.AllProjects = false
		cli.GlobalOpts.ProjectDir = project1

		config, err := cli.LoadConfigForCommand()
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		cwd, _ := cli.GetWorkingDir()
		store, _ := cli.NewStoreForCommand(cwd)
		projects, err := cli.DiscoverProjectsForCommand(config, store)
		if err != nil {
			t.Fatalf("Failed to discover projects: %v", err)
		}

		allBalls, err := session.LoadAllBalls(projects)
		if err != nil {
			t.Fatalf("Failed to load balls: %v", err)
		}

		// Filter to non-complete balls
		activeBalls := make([]*session.Ball, 0)
		for _, ball := range allBalls {
			if ball.State != session.StateComplete {
				activeBalls = append(activeBalls, ball)
			}
		}

		// Should only find ball from current project
		if len(activeBalls) != 1 {
			t.Errorf("Search by default (local) should find 1 ball, got %d", len(activeBalls))
		}

		if len(activeBalls) > 0 && activeBalls[0].WorkingDir != project1 {
			t.Errorf("Search by default (local) should find ball from project1, got ball from %s", activeBalls[0].WorkingDir)
		}
	})

	t.Run("DefaultLocal_NoJuggleDirectory", func(t *testing.T) {
		// Test by default (local) when current directory has no .juggle
		noJuggleDir := t.TempDir()
		cli.GlobalOpts.AllProjects = false
		cli.GlobalOpts.ProjectDir = noJuggleDir

		config, err := cli.LoadConfigForCommand()
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		cwd, _ := cli.GetWorkingDir()
		store, _ := cli.NewStoreForCommand(cwd)
		projects, err := cli.DiscoverProjectsForCommand(config, store)
		if err != nil {
			t.Fatalf("Failed to discover projects: %v", err)
		}

		// Should return current directory even if no .juggle exists
		if len(projects) != 1 {
			t.Errorf("Expected 1 project (current dir), got %d", len(projects))
		}

		if len(projects) > 0 && projects[0] != noJuggleDir {
			t.Errorf("Expected current directory %s, got %s", noJuggleDir, projects[0])
		}
	})

	t.Run("FindBallByID_LocalOnly", func(t *testing.T) {
		// Reset global opts
		cli.GlobalOpts.AllProjects = false
		cli.GlobalOpts.ProjectDir = project1

		// When searching for a ball from project2 while in project1 (local only mode),
		// the ball should not be found and should suggest using --all
		config, err := cli.LoadConfigForCommand()
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		cwd, _ := cli.GetWorkingDir()
		store, _ := cli.NewStoreForCommand(cwd)

		projects, err := cli.DiscoverProjectsForCommand(config, store)
		if err != nil {
			t.Fatalf("Failed to discover projects: %v", err)
		}

		// Load balls from discovered projects (local only)
		allBalls, err := session.LoadAllBalls(projects)
		if err != nil {
			t.Fatalf("Failed to load balls: %v", err)
		}

		// Should only have ball1 from project1
		if len(allBalls) != 1 {
			t.Errorf("Expected 1 ball in local mode, got %d", len(allBalls))
		}

		// ball2 (test2-1) should not be found
		found := false
		for _, ball := range allBalls {
			if ball.ID == "test2-1" {
				found = true
				break
			}
		}
		if found {
			t.Error("Ball from project2 should not be found in local mode")
		}

		// ball1 (test1-1) should be found
		found = false
		for _, ball := range allBalls {
			if ball.ID == "test1-1" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Ball from project1 should be found in local mode")
		}
	})

	t.Run("FindBallByID_WithAll", func(t *testing.T) {
		// With --all flag, balls from both projects should be discoverable
		cli.GlobalOpts.AllProjects = true
		cli.GlobalOpts.ProjectDir = project1

		projects, err := cli.DiscoverProjectsForCommand(config, store1)
		if err != nil {
			t.Fatalf("Failed to discover projects: %v", err)
		}

		allBalls, err := session.LoadAllBalls(projects)
		if err != nil {
			t.Fatalf("Failed to load balls: %v", err)
		}

		// Should have both balls
		if len(allBalls) != 2 {
			t.Errorf("Expected 2 balls with --all, got %d", len(allBalls))
		}

		// Both balls should be found
		foundBall1, foundBall2 := false, false
		for _, ball := range allBalls {
			if ball.ID == "test1-1" {
				foundBall1 = true
			}
			if ball.ID == "test2-1" {
				foundBall2 = true
			}
		}
		if !foundBall1 || !foundBall2 {
			t.Error("Both balls should be found with --all flag")
		}

		// Reset
		cli.GlobalOpts.AllProjects = false
	})
}
