package integration_test

import (
	"testing"

	"github.com/ohare93/juggle/internal/cli"
	"github.com/ohare93/juggle/internal/session"
)

func TestLocalFlag(t *testing.T) {
	// Create two test projects
	project1 := t.TempDir()
	project2 := t.TempDir()

	// Setup project 1
	store1, err := session.NewStoreWithConfig(project1, session.StoreConfig{JugglerDirName: ".juggler"})
	if err != nil {
		t.Fatalf("Failed to create store1: %v", err)
	}

	ball1 := &session.Session{
		ID:         "test1-1",
		WorkingDir: project1,
		Intent:     "Ball in project 1",
		Priority:   session.PriorityMedium,
		State:      session.StateInProgress,
	}
	if err := store1.Save(ball1); err != nil {
		t.Fatalf("Failed to save ball1: %v", err)
	}

	// Setup project 2
	store2, err := session.NewStoreWithConfig(project2, session.StoreConfig{JugglerDirName: ".juggler"})
	if err != nil {
		t.Fatalf("Failed to create store2: %v", err)
	}

	ball2 := &session.Session{
		ID:         "test2-1",
		WorkingDir: project2,
		Intent:     "Ball in project 2",
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

	t.Run("DiscoverProjectsForCommand_WithoutLocal", func(t *testing.T) {
		cli.GlobalOpts.LocalOnly = false
		cli.GlobalOpts.ProjectDir = project1

		projects, err := cli.DiscoverProjectsForCommand(config, store1)
		if err != nil {
			t.Fatalf("Failed to discover projects: %v", err)
		}

		// Should discover both projects using config.SearchPaths
		if len(projects) != 2 {
			t.Errorf("Expected 2 projects without --local, got %d", len(projects))
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
	})

	t.Run("RootCommand_WithLocal", func(t *testing.T) {
		cli.GlobalOpts.LocalOnly = true
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
			t.Errorf("Expected 1 project with --local, got %d", len(projects))
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
			t.Errorf("Expected 1 ball with --local, got %d", len(balls))
		}

		if len(balls) > 0 && balls[0].WorkingDir != project1 {
			t.Errorf("Expected ball from project1, got ball from %s", balls[0].WorkingDir)
		}
	})

	t.Run("StatusCommand_LocalFlag", func(t *testing.T) {
		// Test with --local flag
		cli.GlobalOpts.LocalOnly = true
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
			t.Errorf("Status with --local should only show 1 project, got %d", len(projects))
		}
	})

	t.Run("NextCommand_LocalFlag", func(t *testing.T) {
		// Test with --local flag
		cli.GlobalOpts.LocalOnly = true
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
			t.Errorf("Next with --local should find 1 ball, got %d", len(balls))
		}

		if len(balls) > 0 && balls[0].WorkingDir != project1 {
			t.Errorf("Next with --local should find ball from project1, got ball from %s", balls[0].WorkingDir)
		}
	})

	t.Run("SearchCommand_LocalFlag", func(t *testing.T) {
		// Test with --local flag
		cli.GlobalOpts.LocalOnly = true
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
		activeBalls := make([]*session.Session, 0)
		for _, ball := range allBalls {
			if ball.ActiveState != session.ActiveComplete {
				activeBalls = append(activeBalls, ball)
			}
		}

		// Should only find ball from current project
		if len(activeBalls) != 1 {
			t.Errorf("Search with --local should find 1 ball, got %d", len(activeBalls))
		}

		if len(activeBalls) > 0 && activeBalls[0].WorkingDir != project1 {
			t.Errorf("Search with --local should find ball from project1, got ball from %s", activeBalls[0].WorkingDir)
		}
	})

	t.Run("LocalFlag_NoJugglerDirectory", func(t *testing.T) {
		// Test with --local when current directory has no .juggler
		noJugglerDir := t.TempDir()
		cli.GlobalOpts.LocalOnly = true
		cli.GlobalOpts.ProjectDir = noJugglerDir

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

		// Should return current directory even if no .juggler exists
		if len(projects) != 1 {
			t.Errorf("Expected 1 project (current dir), got %d", len(projects))
		}

		if len(projects) > 0 && projects[0] != noJugglerDir {
			t.Errorf("Expected current directory %s, got %s", noJugglerDir, projects[0])
		}
	})
}
