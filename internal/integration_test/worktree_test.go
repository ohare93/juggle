package integration_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ohare93/juggle/internal/session"
)

func TestWorktreeRegistration(t *testing.T) {
	env := SetupTestEnv(t)

	// Create the .juggle directory in the project (via GetStore which creates it)
	_ = env.GetStore(t)

	// Create a "worktree" directory (simulated - doesn't need to be a real git worktree)
	worktreePath := filepath.Join(env.TempDir, "worktree-1")
	if err := os.MkdirAll(worktreePath, 0755); err != nil {
		t.Fatalf("Failed to create worktree dir: %v", err)
	}

	t.Run("RegisterWorktree", func(t *testing.T) {
		err := session.RegisterWorktree(env.ProjectDir, worktreePath, ".juggle")
		if err != nil {
			t.Fatalf("RegisterWorktree failed: %v", err)
		}

		// Verify link file was created in worktree
		linkPath := filepath.Join(worktreePath, ".juggle", "link")
		linkData, err := os.ReadFile(linkPath)
		if err != nil {
			t.Fatalf("Failed to read link file: %v", err)
		}

		linkedPath := strings.TrimSpace(string(linkData))
		if linkedPath != env.ProjectDir {
			t.Errorf("Link file points to %q, expected %q", linkedPath, env.ProjectDir)
		}

		// Verify worktree is in config
		worktrees, err := session.ListWorktrees(env.ProjectDir, ".juggle")
		if err != nil {
			t.Fatalf("ListWorktrees failed: %v", err)
		}

		found := false
		for _, wt := range worktrees {
			if wt == worktreePath {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Worktree %s not found in list: %v", worktreePath, worktrees)
		}
	})

	t.Run("IsWorktree", func(t *testing.T) {
		// Main repo should not be a worktree
		isWt, err := session.IsWorktree(env.ProjectDir, ".juggle")
		if err != nil {
			t.Fatalf("IsWorktree failed: %v", err)
		}
		if isWt {
			t.Error("Main repo should not be detected as worktree")
		}

		// Worktree directory should be a worktree
		isWt, err = session.IsWorktree(worktreePath, ".juggle")
		if err != nil {
			t.Fatalf("IsWorktree failed: %v", err)
		}
		if !isWt {
			t.Error("Worktree should be detected as worktree")
		}
	})

	t.Run("ResolveStorageDir", func(t *testing.T) {
		// From main repo, should resolve to itself
		resolved, err := session.ResolveStorageDir(env.ProjectDir, ".juggle")
		if err != nil {
			t.Fatalf("ResolveStorageDir failed: %v", err)
		}
		if resolved != env.ProjectDir {
			t.Errorf("Main repo resolved to %q, expected %q", resolved, env.ProjectDir)
		}

		// From worktree, should resolve to main repo
		resolved, err = session.ResolveStorageDir(worktreePath, ".juggle")
		if err != nil {
			t.Fatalf("ResolveStorageDir failed: %v", err)
		}
		if resolved != env.ProjectDir {
			t.Errorf("Worktree resolved to %q, expected %q", resolved, env.ProjectDir)
		}
	})

	t.Run("ForgetWorktree", func(t *testing.T) {
		err := session.ForgetWorktree(env.ProjectDir, worktreePath, ".juggle")
		if err != nil {
			t.Fatalf("ForgetWorktree failed: %v", err)
		}

		// Verify link file was removed
		linkPath := filepath.Join(worktreePath, ".juggle", "link")
		if _, err := os.Stat(linkPath); !os.IsNotExist(err) {
			t.Error("Link file should be removed after forget")
		}

		// Verify worktree is no longer in config
		worktrees, err := session.ListWorktrees(env.ProjectDir, ".juggle")
		if err != nil {
			t.Fatalf("ListWorktrees failed: %v", err)
		}

		for _, wt := range worktrees {
			if wt == worktreePath {
				t.Error("Worktree should not be in list after forget")
			}
		}
	})
}

func TestWorktreeBallSharing(t *testing.T) {
	env := SetupTestEnv(t)

	// Create the .juggle directory in the project (via GetStore which creates it)
	_ = env.GetStore(t)

	// Create worktree directory
	worktreePath := filepath.Join(env.TempDir, "worktree-share")
	if err := os.MkdirAll(worktreePath, 0755); err != nil {
		t.Fatalf("Failed to create worktree dir: %v", err)
	}

	// Register worktree
	if err := session.RegisterWorktree(env.ProjectDir, worktreePath, ".juggle"); err != nil {
		t.Fatalf("RegisterWorktree failed: %v", err)
	}

	t.Run("BallsCreatedInMainVisibleFromWorktree", func(t *testing.T) {
		// Create ball from main repo
		mainStore, err := session.NewStoreWithConfig(env.ProjectDir, session.StoreConfig{JuggleDirName: ".juggle"})
		if err != nil {
			t.Fatalf("Failed to create main store: %v", err)
		}

		ball, err := session.NewBall(env.ProjectDir, "Test from main", session.PriorityMedium)
		if err != nil {
			t.Fatalf("Failed to create ball: %v", err)
		}
		if err := mainStore.AppendBall(ball); err != nil {
			t.Fatalf("Failed to append ball: %v", err)
		}

		// Load balls from worktree - should see the ball
		wtStore, err := session.NewStoreWithConfig(worktreePath, session.StoreConfig{JuggleDirName: ".juggle"})
		if err != nil {
			t.Fatalf("Failed to create worktree store: %v", err)
		}

		balls, err := wtStore.LoadBalls()
		if err != nil {
			t.Fatalf("Failed to load balls: %v", err)
		}

		if len(balls) != 1 {
			t.Errorf("Expected 1 ball from worktree, got %d", len(balls))
		}

		if len(balls) > 0 && balls[0].Title != "Test from main" {
			t.Errorf("Expected ball title 'Test from main', got %q", balls[0].Title)
		}
	})

	t.Run("BallsCreatedInWorktreeVisibleFromMain", func(t *testing.T) {
		// Create ball from worktree
		wtStore, err := session.NewStoreWithConfig(worktreePath, session.StoreConfig{JuggleDirName: ".juggle"})
		if err != nil {
			t.Fatalf("Failed to create worktree store: %v", err)
		}

		ball, err := session.NewBall(worktreePath, "Test from worktree", session.PriorityHigh)
		if err != nil {
			t.Fatalf("Failed to create ball: %v", err)
		}
		if err := wtStore.AppendBall(ball); err != nil {
			t.Fatalf("Failed to append ball: %v", err)
		}

		// Load balls from main repo - should see both balls
		mainStore, err := session.NewStoreWithConfig(env.ProjectDir, session.StoreConfig{JuggleDirName: ".juggle"})
		if err != nil {
			t.Fatalf("Failed to create main store: %v", err)
		}

		balls, err := mainStore.LoadBalls()
		if err != nil {
			t.Fatalf("Failed to load balls: %v", err)
		}

		if len(balls) != 2 {
			t.Errorf("Expected 2 balls from main, got %d", len(balls))
		}

		// Find the worktree ball
		var wtBall *session.Ball
		for _, b := range balls {
			if b.Title == "Test from worktree" {
				wtBall = b
				break
			}
		}

		if wtBall == nil {
			t.Error("Worktree ball not found in main repo")
		}
	})

	t.Run("WorkingDirReflectsWhereStoreWasCreated", func(t *testing.T) {
		// Load balls from worktree - WorkingDir should be worktree path
		wtStore, err := session.NewStoreWithConfig(worktreePath, session.StoreConfig{JuggleDirName: ".juggle"})
		if err != nil {
			t.Fatalf("Failed to create worktree store: %v", err)
		}

		balls, err := wtStore.LoadBalls()
		if err != nil {
			t.Fatalf("Failed to load balls: %v", err)
		}

		// Note: WorkingDir is set to store's projectDir, which is the worktree path
		// But since we're loading from worktree store which resolves to main for storage,
		// the WorkingDir will be... let me check the implementation.
		// Actually looking at the code, LoadBalls sets WorkingDir to s.projectDir,
		// and store.projectDir is set to the original input (not storageDir).
		// Wait, no - I kept projectDir as the original input in Store,
		// but in SessionStore I set it to storageDir. Let me check.

		// Actually looking at my implementation again:
		// Store: projectDir = original input, storage paths use storageDir
		// SessionStore: projectDir = storageDir

		// So for Store, balls loaded will have WorkingDir = original input = worktree path
		// This is correct behavior!

		// Actually wait, I need to re-read my Store implementation.
		// I see that I did NOT change the Store to keep the original projectDir.
		// Let me check the actual code...

		// Looking at the edit I made to store.go, I only changed the storage paths
		// to use storageDir, but the Store struct still has projectDir = original input.
		// So LoadBalls will set WorkingDir = projectDir = worktree path. Good!

		for _, ball := range balls {
			// WorkingDir should be the main repo path since that's where the Store
			// was created (after resolution). Actually no - let me trace through:
			// 1. NewStoreWithConfig(worktreePath, ...)
			// 2. projectDir = worktreePath
			// 3. storageDir = main repo (from ResolveStorageDir)
			// 4. store.projectDir = projectDir (worktreePath)
			// 5. LoadBalls sets ball.WorkingDir = s.projectDir = worktreePath

			// So balls loaded from worktree store should have WorkingDir = worktreePath
			if ball.WorkingDir != worktreePath {
				t.Errorf("Ball WorkingDir = %q, expected %q", ball.WorkingDir, worktreePath)
			}
		}
	})
}

func TestWorktreeBallIDUsesMainRepoName(t *testing.T) {
	env := SetupTestEnv(t)

	// Create the .juggle directory in the project
	_ = env.GetStore(t)

	// Create worktree with a distinct name
	worktreePath := filepath.Join(env.TempDir, "some-worktree-name")
	if err := os.MkdirAll(worktreePath, 0755); err != nil {
		t.Fatalf("Failed to create worktree dir: %v", err)
	}

	// Register worktree
	if err := session.RegisterWorktree(env.ProjectDir, worktreePath, ".juggle"); err != nil {
		t.Fatalf("RegisterWorktree failed: %v", err)
	}

	// Create a ball from the worktree
	ball, err := session.NewBall(worktreePath, "Test ball from worktree", session.PriorityMedium)
	if err != nil {
		t.Fatalf("Failed to create ball: %v", err)
	}

	// The ball ID should use the main repo's folder name ("project"), not the worktree name
	mainRepoName := filepath.Base(env.ProjectDir)
	worktreeName := filepath.Base(worktreePath)

	if strings.HasPrefix(ball.ID, worktreeName+"-") {
		t.Errorf("Ball ID %q should NOT start with worktree name %q", ball.ID, worktreeName)
	}

	if !strings.HasPrefix(ball.ID, mainRepoName+"-") {
		t.Errorf("Ball ID %q should start with main repo name %q", ball.ID, mainRepoName)
	}

	// But WorkingDir should still be the worktree path (where agent works)
	if ball.WorkingDir != worktreePath {
		t.Errorf("Ball WorkingDir = %q, expected worktree path %q", ball.WorkingDir, worktreePath)
	}
}

func TestWorkspaceAlias(t *testing.T) {
	env := SetupTestEnv(t)

	// Create the .juggle directory in the project
	_ = env.GetStore(t)

	// Create a worktree directory
	worktreePath := filepath.Join(env.TempDir, "worktree-alias-test")
	if err := os.MkdirAll(worktreePath, 0755); err != nil {
		t.Fatalf("Failed to create worktree dir: %v", err)
	}

	t.Run("WorkspaceAddSameAsWorktreeAdd", func(t *testing.T) {
		// Register worktree using session API (simulating CLI behavior)
		err := session.RegisterWorktree(env.ProjectDir, worktreePath, ".juggle")
		if err != nil {
			t.Fatalf("RegisterWorktree failed: %v", err)
		}

		// Verify worktree was registered
		worktrees, err := session.ListWorktrees(env.ProjectDir, ".juggle")
		if err != nil {
			t.Fatalf("ListWorktrees failed: %v", err)
		}

		found := false
		for _, wt := range worktrees {
			if wt == worktreePath {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Worktree %s not found in list: %v", worktreePath, worktrees)
		}
	})

	t.Run("WorkspaceListSameAsWorktreeList", func(t *testing.T) {
		worktrees, err := session.ListWorktrees(env.ProjectDir, ".juggle")
		if err != nil {
			t.Fatalf("ListWorktrees failed: %v", err)
		}

		if len(worktrees) == 0 {
			t.Error("Expected at least one worktree")
		}
	})

	t.Run("WorkspaceStatusSameAsWorktreeStatus", func(t *testing.T) {
		// Main repo should not be a worktree
		isWt, err := session.IsWorktree(env.ProjectDir, ".juggle")
		if err != nil {
			t.Fatalf("IsWorktree failed: %v", err)
		}
		if isWt {
			t.Error("Main repo should not be detected as worktree")
		}

		// Worktree directory should be a worktree
		isWt, err = session.IsWorktree(worktreePath, ".juggle")
		if err != nil {
			t.Fatalf("IsWorktree failed: %v", err)
		}
		if !isWt {
			t.Error("Worktree should be detected as worktree")
		}
	})

	t.Run("WorkspaceForgetSameAsWorktreeForget", func(t *testing.T) {
		err := session.ForgetWorktree(env.ProjectDir, worktreePath, ".juggle")
		if err != nil {
			t.Fatalf("ForgetWorktree failed: %v", err)
		}

		// Verify worktree was removed
		worktrees, err := session.ListWorktrees(env.ProjectDir, ".juggle")
		if err != nil {
			t.Fatalf("ListWorktrees failed: %v", err)
		}

		for _, wt := range worktrees {
			if wt == worktreePath {
				t.Error("Worktree should not be in list after forget")
			}
		}
	})
}

func TestWorktreeValidation(t *testing.T) {
	env := SetupTestEnv(t)

	// Create the .juggle directory in the project (via GetStore which creates it)
	_ = env.GetStore(t)

	t.Run("CannotRegisterMainAsWorktree", func(t *testing.T) {
		err := session.RegisterWorktree(env.ProjectDir, env.ProjectDir, ".juggle")
		if err == nil {
			t.Error("Should not be able to register main repo as its own worktree")
		}
	})

	t.Run("CannotRegisterNonexistentPath", func(t *testing.T) {
		err := session.RegisterWorktree(env.ProjectDir, "/nonexistent/path", ".juggle")
		if err == nil {
			t.Error("Should not be able to register nonexistent path")
		}
	})

	t.Run("CannotRegisterSameWorktreeTwice", func(t *testing.T) {
		worktreePath := filepath.Join(env.TempDir, "worktree-dup")
		if err := os.MkdirAll(worktreePath, 0755); err != nil {
			t.Fatalf("Failed to create worktree dir: %v", err)
		}

		// First registration should succeed
		if err := session.RegisterWorktree(env.ProjectDir, worktreePath, ".juggle"); err != nil {
			t.Fatalf("First registration failed: %v", err)
		}

		// Second registration should fail
		err := session.RegisterWorktree(env.ProjectDir, worktreePath, ".juggle")
		if err == nil {
			t.Error("Should not be able to register same worktree twice")
		}
	})

	t.Run("ForgetNonregisteredWorktree", func(t *testing.T) {
		err := session.ForgetWorktree(env.ProjectDir, "/some/random/path", ".juggle")
		if err == nil {
			t.Error("Should get error when forgetting unregistered worktree")
		}
	})
}
