package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ohare93/juggle/internal/claude"
)

func TestSetupClaudeWithHooks_Fresh(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()

	// Change to temp directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	// Set up flags
	setupClaudeOpts.global = false
	setupClaudeOpts.dryRun = false
	setupClaudeOpts.update = false
	setupClaudeOpts.uninstall = false
	setupClaudeOpts.force = true
	setupClaudeOpts.installHooks = true
	defer func() {
		setupClaudeOpts = struct {
			global       bool
			dryRun       bool
			update       bool
			uninstall    bool
			force        bool
			installHooks bool
		}{}
	}()

	// Run setup command
	if err := runSetupClaude(nil, nil); err != nil {
		t.Fatalf("runSetupClaude() error = %v", err)
	}

	// Verify CLAUDE.md was created
	claudePath := filepath.Join(tempDir, ".claude", "CLAUDE.md")
	if _, err := os.Stat(claudePath); os.IsNotExist(err) {
		t.Fatal("CLAUDE.md was not created")
	}

	// Verify hooks.json was created
	hooksPath := filepath.Join(tempDir, ".claude", "hooks.json")
	if _, err := os.Stat(hooksPath); os.IsNotExist(err) {
		t.Fatal("hooks.json was not created")
	}

	// Verify hooks are installed correctly
	installed, err := claude.HooksInstalled(tempDir)
	if err != nil {
		t.Fatalf("HooksInstalled() error = %v", err)
	}
	if !installed {
		t.Error("Hooks were not installed correctly")
	}
}

func TestSetupClaudeWithHooks_UpdateExisting(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()

	// Change to temp directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	// Install initial version without hooks
	setupClaudeOpts.global = false
	setupClaudeOpts.dryRun = false
	setupClaudeOpts.update = false
	setupClaudeOpts.uninstall = false
	setupClaudeOpts.force = true
	setupClaudeOpts.installHooks = false
	defer func() {
		setupClaudeOpts = struct {
			global       bool
			dryRun       bool
			update       bool
			uninstall    bool
			force        bool
			installHooks bool
		}{}
	}()

	if err := runSetupClaude(nil, nil); err != nil {
		t.Fatalf("Initial setup error = %v", err)
	}

	// Verify hooks don't exist yet
	hooksPath := filepath.Join(tempDir, ".claude", "hooks.json")
	if _, err := os.Stat(hooksPath); !os.IsNotExist(err) {
		t.Error("hooks.json should not exist yet")
	}

	// Update with hooks
	setupClaudeOpts.update = true
	setupClaudeOpts.installHooks = true

	if err := runSetupClaude(nil, nil); err != nil {
		t.Fatalf("Update with hooks error = %v", err)
	}

	// Verify hooks were installed
	installed, err := claude.HooksInstalled(tempDir)
	if err != nil {
		t.Fatalf("HooksInstalled() error = %v", err)
	}
	if !installed {
		t.Error("Hooks were not installed during update")
	}
}

func TestSetupClaudeWithHooks_GlobalSkipsHooks(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()

	// Change to temp directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	// Set up flags for global install with hooks flag
	setupClaudeOpts.global = true
	setupClaudeOpts.dryRun = false
	setupClaudeOpts.update = false
	setupClaudeOpts.uninstall = false
	setupClaudeOpts.force = true
	setupClaudeOpts.installHooks = true // This should be ignored for global
	defer func() {
		setupClaudeOpts = struct {
			global       bool
			dryRun       bool
			update       bool
			uninstall    bool
			force        bool
			installHooks bool
		}{}
	}()

	// Run setup command
	if err := runSetupClaude(nil, nil); err != nil {
		t.Fatalf("runSetupClaude() error = %v", err)
	}

	// Verify hooks were NOT created in project directory
	hooksPath := filepath.Join(tempDir, ".claude", "hooks.json")
	if _, err := os.Stat(hooksPath); !os.IsNotExist(err) {
		t.Error("hooks.json should not be created for global installation")
	}
}

func TestSetupClaudeWithHooks_DryRun(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()

	// Change to temp directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	// Set up flags for dry run with hooks
	setupClaudeOpts.global = false
	setupClaudeOpts.dryRun = true
	setupClaudeOpts.update = false
	setupClaudeOpts.uninstall = false
	setupClaudeOpts.force = true
	setupClaudeOpts.installHooks = true
	defer func() {
		setupClaudeOpts = struct {
			global       bool
			dryRun       bool
			update       bool
			uninstall    bool
			force        bool
			installHooks bool
		}{}
	}()

	// Run setup command
	if err := runSetupClaude(nil, nil); err != nil {
		t.Fatalf("runSetupClaude() error = %v", err)
	}

	// Verify nothing was created
	claudePath := filepath.Join(tempDir, ".claude", "CLAUDE.md")
	if _, err := os.Stat(claudePath); !os.IsNotExist(err) {
		t.Error("CLAUDE.md should not be created in dry-run mode")
	}

	hooksPath := filepath.Join(tempDir, ".claude", "hooks.json")
	if _, err := os.Stat(hooksPath); !os.IsNotExist(err) {
		t.Error("hooks.json should not be created in dry-run mode")
	}
}

func TestSetupClaudeUninstall_WithHooks(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()

	// Change to temp directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	// Install with hooks
	setupClaudeOpts.global = false
	setupClaudeOpts.dryRun = false
	setupClaudeOpts.update = false
	setupClaudeOpts.uninstall = false
	setupClaudeOpts.force = true
	setupClaudeOpts.installHooks = true
	defer func() {
		setupClaudeOpts = struct {
			global       bool
			dryRun       bool
			update       bool
			uninstall    bool
			force        bool
			installHooks bool
		}{}
	}()

	if err := runSetupClaude(nil, nil); err != nil {
		t.Fatalf("Initial setup error = %v", err)
	}

	// Verify both exist
	claudePath := filepath.Join(tempDir, ".claude", "CLAUDE.md")
	hooksPath := filepath.Join(tempDir, ".claude", "hooks.json")

	if _, err := os.Stat(claudePath); os.IsNotExist(err) {
		t.Fatal("CLAUDE.md was not created")
	}
	if _, err := os.Stat(hooksPath); os.IsNotExist(err) {
		t.Fatal("hooks.json was not created")
	}

	// Uninstall
	setupClaudeOpts.uninstall = true
	setupClaudeOpts.installHooks = false

	if err := runSetupClaude(nil, nil); err != nil {
		t.Fatalf("Uninstall error = %v", err)
	}

	// Verify CLAUDE.md still exists but instructions are removed
	content, err := os.ReadFile(claudePath)
	if err != nil {
		t.Fatalf("Failed to read CLAUDE.md: %v", err)
	}
	if strings.Contains(string(content), claude.InstructionsMarkerStart) {
		t.Error("Instructions were not removed from CLAUDE.md")
	}

	// Verify hooks.json was removed (since only juggler hooks existed)
	if _, err := os.Stat(hooksPath); !os.IsNotExist(err) {
		t.Error("hooks.json should have been removed")
	}
}

func TestSetupClaudeUninstall_PreservesOtherHooks(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()

	// Change to temp directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	// Install with hooks
	setupClaudeOpts.global = false
	setupClaudeOpts.dryRun = false
	setupClaudeOpts.update = false
	setupClaudeOpts.uninstall = false
	setupClaudeOpts.force = true
	setupClaudeOpts.installHooks = true
	defer func() {
		setupClaudeOpts = struct {
			global       bool
			dryRun       bool
			update       bool
			uninstall    bool
			force        bool
			installHooks bool
		}{}
	}()

	if err := runSetupClaude(nil, nil); err != nil {
		t.Fatalf("Initial setup error = %v", err)
	}

	// Add a custom hook
	hooksPath := filepath.Join(tempDir, ".claude", "hooks.json")
	customHook := `{
  "user-prompt-submit": {
    "command": "juggle track-activity 2>/dev/null || true",
    "description": "Track activity on current ball"
  },
  "assistant-response-start": {
    "command": "juggle reminder 2>/dev/null || true",
    "description": "Remind to check juggler state"
  },
  "custom-hook": {
    "command": "echo 'custom'",
    "description": "Custom hook"
  }
}
`
	if err := os.WriteFile(hooksPath, []byte(customHook), 0644); err != nil {
		t.Fatalf("Failed to add custom hook: %v", err)
	}

	// Uninstall
	setupClaudeOpts.uninstall = true
	setupClaudeOpts.installHooks = false

	if err := runSetupClaude(nil, nil); err != nil {
		t.Fatalf("Uninstall error = %v", err)
	}

	// Verify hooks.json still exists
	if _, err := os.Stat(hooksPath); os.IsNotExist(err) {
		t.Error("hooks.json should be preserved when custom hooks exist")
	}

	// Verify custom hook is still there
	content, err := os.ReadFile(hooksPath)
	if err != nil {
		t.Fatalf("Failed to read hooks.json: %v", err)
	}
	if !strings.Contains(string(content), "custom-hook") {
		t.Error("Custom hook was removed but should be preserved")
	}

	// Verify juggler hooks are gone
	if strings.Contains(string(content), "juggle track-activity") {
		t.Error("Juggler hooks were not removed")
	}
}

func TestSetupClaudeUninstall_DryRun(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()

	// Change to temp directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	// Install with hooks
	setupClaudeOpts.global = false
	setupClaudeOpts.dryRun = false
	setupClaudeOpts.update = false
	setupClaudeOpts.uninstall = false
	setupClaudeOpts.force = true
	setupClaudeOpts.installHooks = true
	defer func() {
		setupClaudeOpts = struct {
			global       bool
			dryRun       bool
			update       bool
			uninstall    bool
			force        bool
			installHooks bool
		}{}
	}()

	if err := runSetupClaude(nil, nil); err != nil {
		t.Fatalf("Initial setup error = %v", err)
	}

	// Dry run uninstall
	setupClaudeOpts.dryRun = true
	setupClaudeOpts.uninstall = true
	setupClaudeOpts.installHooks = false

	if err := runSetupClaude(nil, nil); err != nil {
		t.Fatalf("Uninstall dry-run error = %v", err)
	}

	// Verify nothing was removed
	claudePath := filepath.Join(tempDir, ".claude", "CLAUDE.md")
	content, err := os.ReadFile(claudePath)
	if err != nil {
		t.Fatalf("Failed to read CLAUDE.md: %v", err)
	}
	if !strings.Contains(string(content), claude.InstructionsMarkerStart) {
		t.Error("Instructions were removed in dry-run mode")
	}

	installed, _ := claude.HooksInstalled(tempDir)
	if !installed {
		t.Error("Hooks were removed in dry-run mode")
	}
}
