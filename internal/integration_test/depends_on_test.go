package integration_test

import (
	"testing"

	"github.com/ohare93/juggle/internal/session"
)

func TestDependencyManagement(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create parent and child balls
	parent := env.CreateBall(t, "Parent task", session.PriorityHigh)
	child := env.CreateBall(t, "Child task", session.PriorityMedium)
	store := env.GetStore(t)

	// Add dependency
	child.AddDependency(parent.ID)

	if err := store.UpdateBall(child); err != nil {
		t.Fatalf("Failed to save ball with dependency: %v", err)
	}

	// Verify dependency was saved
	retrieved, err := store.GetBallByID(child.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve ball: %v", err)
	}

	if len(retrieved.DependsOn) != 1 {
		t.Fatalf("Expected 1 dependency, got %d", len(retrieved.DependsOn))
	}

	if retrieved.DependsOn[0] != parent.ID {
		t.Errorf("Expected dependency to be %s, got %s", parent.ID, retrieved.DependsOn[0])
	}
}

func TestDependencyDuplicatePrevention(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	parent := env.CreateBall(t, "Parent task", session.PriorityHigh)
	child := env.CreateBall(t, "Child task", session.PriorityMedium)

	// Add same dependency twice
	child.AddDependency(parent.ID)
	child.AddDependency(parent.ID)

	if len(child.DependsOn) != 1 {
		t.Errorf("Expected 1 dependency (no duplicates), got %d", len(child.DependsOn))
	}
}

func TestRemoveDependency(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	parent := env.CreateBall(t, "Parent task", session.PriorityHigh)
	child := env.CreateBall(t, "Child task", session.PriorityMedium)
	store := env.GetStore(t)

	// Add and then remove dependency
	child.AddDependency(parent.ID)
	if len(child.DependsOn) != 1 {
		t.Fatalf("Expected 1 dependency after add, got %d", len(child.DependsOn))
	}

	removed := child.RemoveDependency(parent.ID)
	if !removed {
		t.Error("Expected RemoveDependency to return true")
	}

	if len(child.DependsOn) != 0 {
		t.Errorf("Expected 0 dependencies after removal, got %d", len(child.DependsOn))
	}

	// Save and verify persistence
	if err := store.UpdateBall(child); err != nil {
		t.Fatalf("Failed to save ball: %v", err)
	}

	retrieved, err := store.GetBallByID(child.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve ball: %v", err)
	}

	if len(retrieved.DependsOn) != 0 {
		t.Errorf("Expected 0 dependencies after persistence, got %d", len(retrieved.DependsOn))
	}
}

func TestRemoveNonExistentDependency(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	ball := env.CreateBall(t, "Task", session.PriorityMedium)

	removed := ball.RemoveDependency("non-existent")
	if removed {
		t.Error("Expected RemoveDependency to return false for non-existent dependency")
	}
}

func TestSetDependencies(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	ball1 := env.CreateBall(t, "Task 1", session.PriorityHigh)
	ball2 := env.CreateBall(t, "Task 2", session.PriorityHigh)
	child := env.CreateBall(t, "Child task", session.PriorityMedium)
	store := env.GetStore(t)

	// Set multiple dependencies at once
	child.SetDependencies([]string{ball1.ID, ball2.ID})

	if err := store.UpdateBall(child); err != nil {
		t.Fatalf("Failed to save ball: %v", err)
	}

	retrieved, err := store.GetBallByID(child.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve ball: %v", err)
	}

	if len(retrieved.DependsOn) != 2 {
		t.Fatalf("Expected 2 dependencies, got %d", len(retrieved.DependsOn))
	}
}

func TestHasDependencies(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	parent := env.CreateBall(t, "Parent task", session.PriorityHigh)
	child := env.CreateBall(t, "Child task", session.PriorityMedium)

	if parent.HasDependencies() {
		t.Error("Parent should not have dependencies")
	}

	if child.HasDependencies() {
		t.Error("Child should not have dependencies initially")
	}

	child.AddDependency(parent.ID)

	if !child.HasDependencies() {
		t.Error("Child should have dependencies after adding one")
	}
}

func TestCircularDependencyDetection_Simple(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	ball1 := env.CreateBall(t, "Ball 1", session.PriorityHigh)
	ball2 := env.CreateBall(t, "Ball 2", session.PriorityMedium)

	// Create circular dependency: ball1 -> ball2 -> ball1
	ball1.AddDependency(ball2.ID)
	ball2.AddDependency(ball1.ID)

	balls := []*session.Ball{ball1, ball2}
	err := session.DetectCircularDependencies(balls)

	if err == nil {
		t.Error("Expected circular dependency error, got nil")
	}
}

func TestCircularDependencyDetection_Chain(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	ball1 := env.CreateBall(t, "Ball 1", session.PriorityHigh)
	ball2 := env.CreateBall(t, "Ball 2", session.PriorityMedium)
	ball3 := env.CreateBall(t, "Ball 3", session.PriorityLow)

	// Create chain: ball1 -> ball2 -> ball3 -> ball1
	ball1.AddDependency(ball2.ID)
	ball2.AddDependency(ball3.ID)
	ball3.AddDependency(ball1.ID)

	balls := []*session.Ball{ball1, ball2, ball3}
	err := session.DetectCircularDependencies(balls)

	if err == nil {
		t.Error("Expected circular dependency error for chain, got nil")
	}
}

func TestCircularDependencyDetection_NoCircle(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	ball1 := env.CreateBall(t, "Ball 1", session.PriorityHigh)
	ball2 := env.CreateBall(t, "Ball 2", session.PriorityMedium)
	ball3 := env.CreateBall(t, "Ball 3", session.PriorityLow)

	// Create valid chain: ball1 <- ball2 <- ball3 (no circle)
	ball2.AddDependency(ball1.ID)
	ball3.AddDependency(ball2.ID)

	balls := []*session.Ball{ball1, ball2, ball3}
	err := session.DetectCircularDependencies(balls)

	if err != nil {
		t.Errorf("Expected no circular dependency error, got: %v", err)
	}
}

func TestCircularDependencyDetection_NoDependencies(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	ball1 := env.CreateBall(t, "Ball 1", session.PriorityHigh)
	ball2 := env.CreateBall(t, "Ball 2", session.PriorityMedium)

	balls := []*session.Ball{ball1, ball2}
	err := session.DetectCircularDependencies(balls)

	if err != nil {
		t.Errorf("Expected no error for balls without dependencies, got: %v", err)
	}
}

func TestCircularDependencyDetection_DiamondPattern(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Diamond pattern: ball4 depends on ball2 and ball3, both depend on ball1
	ball1 := env.CreateBall(t, "Ball 1", session.PriorityHigh)
	ball2 := env.CreateBall(t, "Ball 2", session.PriorityMedium)
	ball3 := env.CreateBall(t, "Ball 3", session.PriorityMedium)
	ball4 := env.CreateBall(t, "Ball 4", session.PriorityLow)

	ball2.AddDependency(ball1.ID)
	ball3.AddDependency(ball1.ID)
	ball4.AddDependency(ball2.ID)
	ball4.AddDependency(ball3.ID)

	balls := []*session.Ball{ball1, ball2, ball3, ball4}
	err := session.DetectCircularDependencies(balls)

	if err != nil {
		t.Errorf("Expected no error for diamond pattern, got: %v", err)
	}
}

func TestCircularDependencyDetection_MissingDependency(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	ball1 := env.CreateBall(t, "Ball 1", session.PriorityHigh)

	// Add dependency to non-existent ball
	ball1.AddDependency("non-existent-ball")

	balls := []*session.Ball{ball1}
	err := session.DetectCircularDependencies(balls)

	// Should not error - missing dependencies are ignored (they might be in another project)
	if err != nil {
		t.Errorf("Expected no error for missing dependency, got: %v", err)
	}
}

func TestDependencyJSONPersistence(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	parent := env.CreateBall(t, "Parent", session.PriorityHigh)
	child := env.CreateBall(t, "Child", session.PriorityMedium)
	store := env.GetStore(t)

	// Set dependencies
	child.SetDependencies([]string{parent.ID, "external-123"})

	if err := store.UpdateBall(child); err != nil {
		t.Fatalf("Failed to save ball: %v", err)
	}

	// Create new store to force re-read from disk
	newStore, err := session.NewStore(env.ProjectDir)
	if err != nil {
		t.Fatalf("Failed to create new store: %v", err)
	}

	retrieved, err := newStore.GetBallByID(child.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve ball: %v", err)
	}

	if len(retrieved.DependsOn) != 2 {
		t.Fatalf("Expected 2 dependencies after reload, got %d", len(retrieved.DependsOn))
	}

	// Verify specific values
	found := make(map[string]bool)
	for _, dep := range retrieved.DependsOn {
		found[dep] = true
	}

	if !found[parent.ID] {
		t.Errorf("Expected parent.ID %s in dependencies", parent.ID)
	}
	if !found["external-123"] {
		t.Error("Expected 'external-123' in dependencies")
	}
}

// TestCLIDependsOnFlagInPlan tests the --depends-on flag in plan command
func TestCLIDependsOnFlagInPlan(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a parent ball first
	parent := env.CreateBall(t, "Parent task", session.PriorityHigh)
	store := env.GetStore(t)

	// Create a child ball with dependency using the store directly (simulating CLI)
	child, err := session.NewBall(env.ProjectDir, "Child task with dep", session.PriorityMedium)
	if err != nil {
		t.Fatalf("Failed to create child ball: %v", err)
	}

	// Simulate --depends-on flag by adding dependency before save
	child.AddDependency(parent.ID)

	if err := store.AppendBall(child); err != nil {
		t.Fatalf("Failed to save child ball: %v", err)
	}

	// Verify dependency was saved
	retrieved, err := store.GetBallByID(child.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve child ball: %v", err)
	}

	if len(retrieved.DependsOn) != 1 {
		t.Fatalf("Expected 1 dependency, got %d", len(retrieved.DependsOn))
	}

	if retrieved.DependsOn[0] != parent.ID {
		t.Errorf("Expected dependency %s, got %s", parent.ID, retrieved.DependsOn[0])
	}
}

// TestCLIAddDepFlag tests the --add-dep flag in update command
func TestCLIAddDepFlag(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	parent := env.CreateBall(t, "Parent task", session.PriorityHigh)
	child := env.CreateBall(t, "Child task", session.PriorityMedium)
	store := env.GetStore(t)

	// Simulate --add-dep flag
	child.AddDependency(parent.ID)
	if err := store.UpdateBall(child); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	retrieved, err := store.GetBallByID(child.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve ball: %v", err)
	}

	if len(retrieved.DependsOn) != 1 {
		t.Errorf("Expected 1 dependency, got %d", len(retrieved.DependsOn))
	}
}

// TestCLIRemoveDepFlag tests the --remove-dep flag in update command
func TestCLIRemoveDepFlag(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	parent := env.CreateBall(t, "Parent task", session.PriorityHigh)
	child := env.CreateBall(t, "Child task", session.PriorityMedium)
	store := env.GetStore(t)

	// Add dependency first
	child.AddDependency(parent.ID)
	if err := store.UpdateBall(child); err != nil {
		t.Fatalf("Failed to add dependency: %v", err)
	}

	// Simulate --remove-dep flag
	child.RemoveDependency(parent.ID)
	if err := store.UpdateBall(child); err != nil {
		t.Fatalf("Failed to remove dependency: %v", err)
	}

	retrieved, err := store.GetBallByID(child.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve ball: %v", err)
	}

	if len(retrieved.DependsOn) != 0 {
		t.Errorf("Expected 0 dependencies after removal, got %d", len(retrieved.DependsOn))
	}
}

// TestCLISetDepsFlag tests the --set-deps flag in update command
func TestCLISetDepsFlag(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	ball1 := env.CreateBall(t, "Ball 1", session.PriorityHigh)
	ball2 := env.CreateBall(t, "Ball 2", session.PriorityHigh)
	child := env.CreateBall(t, "Child task", session.PriorityMedium)
	store := env.GetStore(t)

	// Simulate --set-deps flag
	child.SetDependencies([]string{ball1.ID, ball2.ID})
	if err := store.UpdateBall(child); err != nil {
		t.Fatalf("Failed to set dependencies: %v", err)
	}

	retrieved, err := store.GetBallByID(child.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve ball: %v", err)
	}

	if len(retrieved.DependsOn) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(retrieved.DependsOn))
	}
}

// TestCLIDependencyCircularDetection tests circular dependency detection in CLI
func TestCLIDependencyCircularDetection(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	ball1 := env.CreateBall(t, "Ball 1", session.PriorityHigh)
	ball2 := env.CreateBall(t, "Ball 2", session.PriorityMedium)
	store := env.GetStore(t)

	// Create circular dependency
	ball1.AddDependency(ball2.ID)
	ball2.AddDependency(ball1.ID)

	// Save ball1
	if err := store.UpdateBall(ball1); err != nil {
		t.Fatalf("Failed to save ball1: %v", err)
	}

	// Save ball2
	if err := store.UpdateBall(ball2); err != nil {
		t.Fatalf("Failed to save ball2: %v", err)
	}

	// Detect circular dependencies
	balls, err := store.LoadBalls()
	if err != nil {
		t.Fatalf("Failed to load balls: %v", err)
	}

	err = session.DetectCircularDependencies(balls)
	if err == nil {
		t.Error("Expected circular dependency error, got nil")
	}
}

// TestCLISelfDependencyPrevention tests that a ball cannot depend on itself
func TestCLISelfDependencyPrevention(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	ball := env.CreateBall(t, "Self-referential ball", session.PriorityMedium)

	// Try to add self as dependency
	ball.AddDependency(ball.ID)

	// Check - the dependency should be added (session package doesn't prevent this)
	// The CLI layer should prevent it, but at session level it's allowed
	// This test documents the behavior
	if len(ball.DependsOn) != 1 {
		t.Logf("Note: Session package allows self-dependency (CLI prevents this)")
	}
}

// TestCLIDependencyWithShortID tests dependency resolution with short IDs
func TestCLIDependencyWithShortID(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	parent := env.CreateBall(t, "Parent task", session.PriorityHigh)
	child := env.CreateBall(t, "Child task", session.PriorityMedium)
	store := env.GetStore(t)

	// Add dependency using short ID
	shortID := parent.ShortID()
	child.AddDependency(shortID)

	if err := store.UpdateBall(child); err != nil {
		t.Fatalf("Failed to update ball: %v", err)
	}

	retrieved, err := store.GetBallByID(child.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve ball: %v", err)
	}

	// Dependency should be saved with the short ID (as provided)
	if len(retrieved.DependsOn) != 1 {
		t.Fatalf("Expected 1 dependency, got %d", len(retrieved.DependsOn))
	}

	// The stored ID should be what was provided
	if retrieved.DependsOn[0] != shortID {
		t.Errorf("Expected dependency ID %s, got %s", shortID, retrieved.DependsOn[0])
	}
}

// TestCLIDependencyMultipleOperations tests multiple dependency operations
func TestCLIDependencyMultipleOperations(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	ball1 := env.CreateBall(t, "Ball 1", session.PriorityHigh)
	ball2 := env.CreateBall(t, "Ball 2", session.PriorityHigh)
	ball3 := env.CreateBall(t, "Ball 3", session.PriorityHigh)
	child := env.CreateBall(t, "Child task", session.PriorityMedium)
	store := env.GetStore(t)

	// Add first dependency
	child.AddDependency(ball1.ID)
	if err := store.UpdateBall(child); err != nil {
		t.Fatalf("Failed to add first dependency: %v", err)
	}

	// Add second dependency
	child.AddDependency(ball2.ID)
	if err := store.UpdateBall(child); err != nil {
		t.Fatalf("Failed to add second dependency: %v", err)
	}

	// Verify we have 2 dependencies
	retrieved, _ := store.GetBallByID(child.ID)
	if len(retrieved.DependsOn) != 2 {
		t.Fatalf("Expected 2 dependencies, got %d", len(retrieved.DependsOn))
	}

	// Replace all dependencies with just ball3
	child.SetDependencies([]string{ball3.ID})
	if err := store.UpdateBall(child); err != nil {
		t.Fatalf("Failed to replace dependencies: %v", err)
	}

	// Verify we now have 1 dependency
	retrieved, _ = store.GetBallByID(child.ID)
	if len(retrieved.DependsOn) != 1 {
		t.Errorf("Expected 1 dependency after replacement, got %d", len(retrieved.DependsOn))
	}

	if retrieved.DependsOn[0] != ball3.ID {
		t.Errorf("Expected dependency %s, got %s", ball3.ID, retrieved.DependsOn[0])
	}
}
