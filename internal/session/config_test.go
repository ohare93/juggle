package session

import (
	"os"
	"path/filepath"
	"testing"
)

// TestProjectConfig_SetDefaultAcceptanceCriteria tests setting repo-level ACs
func TestProjectConfig_SetDefaultAcceptanceCriteria(t *testing.T) {
	config := DefaultProjectConfig()

	criteria := []string{"Tests pass", "Build succeeds"}
	config.SetDefaultAcceptanceCriteria(criteria)

	if len(config.DefaultAcceptanceCriteria) != 2 {
		t.Errorf("expected 2 acceptance criteria, got %d", len(config.DefaultAcceptanceCriteria))
	}
	if config.DefaultAcceptanceCriteria[0] != "Tests pass" {
		t.Errorf("expected first criterion 'Tests pass', got '%s'", config.DefaultAcceptanceCriteria[0])
	}
}

// TestProjectConfig_HasDefaultAcceptanceCriteria tests the Has method
func TestProjectConfig_HasDefaultAcceptanceCriteria(t *testing.T) {
	config := DefaultProjectConfig()

	if config.HasDefaultAcceptanceCriteria() {
		t.Error("expected HasDefaultAcceptanceCriteria to return false for empty")
	}

	config.SetDefaultAcceptanceCriteria([]string{"Test"})

	if !config.HasDefaultAcceptanceCriteria() {
		t.Error("expected HasDefaultAcceptanceCriteria to return true after setting")
	}
}

// TestUpdateProjectAcceptanceCriteria tests updating and getting repo-level ACs
func TestUpdateProjectAcceptanceCriteria(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "juggler-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Ensure .juggler directory exists
	jugglerDir := filepath.Join(tmpDir, ".juggler")
	if err := os.MkdirAll(jugglerDir, 0755); err != nil {
		t.Fatalf("failed to create .juggler dir: %v", err)
	}

	// Update acceptance criteria
	criteria := []string{"Tests pass", "Build succeeds", "Documentation updated"}
	err = UpdateProjectAcceptanceCriteria(tmpDir, criteria)
	if err != nil {
		t.Fatalf("failed to update acceptance criteria: %v", err)
	}

	// Get and verify
	loaded, err := GetProjectAcceptanceCriteria(tmpDir)
	if err != nil {
		t.Fatalf("failed to get acceptance criteria: %v", err)
	}

	if len(loaded) != 3 {
		t.Errorf("expected 3 acceptance criteria, got %d", len(loaded))
	}
	if loaded[0] != "Tests pass" {
		t.Errorf("expected first criterion 'Tests pass', got '%s'", loaded[0])
	}
}

// TestGetProjectAcceptanceCriteria_Empty tests getting ACs when none exist
func TestGetProjectAcceptanceCriteria_Empty(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "juggler-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Ensure .juggler directory exists
	jugglerDir := filepath.Join(tmpDir, ".juggler")
	if err := os.MkdirAll(jugglerDir, 0755); err != nil {
		t.Fatalf("failed to create .juggler dir: %v", err)
	}

	// Get acceptance criteria (should be empty, not error)
	criteria, err := GetProjectAcceptanceCriteria(tmpDir)
	if err != nil {
		t.Fatalf("failed to get acceptance criteria: %v", err)
	}

	if len(criteria) != 0 {
		t.Errorf("expected 0 acceptance criteria, got %d", len(criteria))
	}
}

// TestProjectAcceptanceCriteria_Persistence tests ACs survive save/load
func TestProjectAcceptanceCriteria_Persistence(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "juggler-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Ensure .juggler directory exists
	jugglerDir := filepath.Join(tmpDir, ".juggler")
	if err := os.MkdirAll(jugglerDir, 0755); err != nil {
		t.Fatalf("failed to create .juggler dir: %v", err)
	}

	// Set acceptance criteria
	criteria := []string{"Run tests", "Check build"}
	if err := UpdateProjectAcceptanceCriteria(tmpDir, criteria); err != nil {
		t.Fatalf("failed to update ACs: %v", err)
	}

	// Load directly from config to verify persistence
	config, err := LoadProjectConfig(tmpDir)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if len(config.DefaultAcceptanceCriteria) != 2 {
		t.Errorf("expected 2 ACs after reload, got %d", len(config.DefaultAcceptanceCriteria))
	}
}

// TestProjectAcceptanceCriteria_Clear tests clearing all ACs
func TestProjectAcceptanceCriteria_Clear(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "juggler-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Ensure .juggler directory exists
	jugglerDir := filepath.Join(tmpDir, ".juggler")
	if err := os.MkdirAll(jugglerDir, 0755); err != nil {
		t.Fatalf("failed to create .juggler dir: %v", err)
	}

	// Set some criteria
	criteria := []string{"Test 1", "Test 2"}
	if err := UpdateProjectAcceptanceCriteria(tmpDir, criteria); err != nil {
		t.Fatalf("failed to update ACs: %v", err)
	}

	// Clear by setting empty
	if err := UpdateProjectAcceptanceCriteria(tmpDir, []string{}); err != nil {
		t.Fatalf("failed to clear ACs: %v", err)
	}

	// Verify cleared
	loaded, err := GetProjectAcceptanceCriteria(tmpDir)
	if err != nil {
		t.Fatalf("failed to get ACs: %v", err)
	}

	if len(loaded) != 0 {
		t.Errorf("expected 0 ACs after clear, got %d", len(loaded))
	}
}
