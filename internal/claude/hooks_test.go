package claude

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestGetHooksTemplate(t *testing.T) {
	template, err := GetHooksTemplate()
	if err != nil {
		t.Fatalf("GetHooksTemplate() error = %v", err)
	}

	// Verify it's valid JSON
	var hooks HooksMap
	if err := json.Unmarshal([]byte(template), &hooks); err != nil {
		t.Fatalf("Template is not valid JSON: %v", err)
	}

	// Verify required hooks are present
	expectedHooks := []string{"user-prompt-submit", "assistant-response-start"}
	for _, hookName := range expectedHooks {
		hook, exists := hooks[hookName]
		if !exists {
			t.Errorf("Missing expected hook: %s", hookName)
		}

		if hook.Command == "" {
			t.Errorf("Hook %s has empty command", hookName)
		}

		if hook.Description == "" {
			t.Errorf("Hook %s has empty description", hookName)
		}

		// Verify safe command pattern (should contain the safety suffix)
		safetySuffix := "2>/dev/null || true"
		if len(hook.Command) < len(safetySuffix) || hook.Command[len(hook.Command)-len(safetySuffix):] != safetySuffix {
			t.Errorf("Hook %s command doesn't end with safety pattern: %s", hookName, hook.Command)
		}
	}

	// Verify specific hooks
	userPromptHook := hooks["user-prompt-submit"]
	if userPromptHook.Command != "juggle track-activity 2>/dev/null || true" {
		t.Errorf("user-prompt-submit command = %q, want %q",
			userPromptHook.Command, "juggle track-activity 2>/dev/null || true")
	}

	assistantHook := hooks["assistant-response-start"]
	if assistantHook.Command != "juggle reminder 2>/dev/null || true" {
		t.Errorf("assistant-response-start command = %q, want %q",
			assistantHook.Command, "juggle reminder 2>/dev/null || true")
	}
}

func TestInstallHooks_Fresh(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()

	// Install hooks
	if err := InstallHooks(tempDir); err != nil {
		t.Fatalf("InstallHooks() error = %v", err)
	}

	// Verify .claude directory was created
	claudeDir := filepath.Join(tempDir, ".claude")
	if info, err := os.Stat(claudeDir); err != nil {
		t.Fatalf("Failed to stat .claude directory: %v", err)
	} else if !info.IsDir() {
		t.Fatal(".claude is not a directory")
	}

	// Verify hooks.json was created
	hooksPath := filepath.Join(claudeDir, "hooks.json")
	data, err := os.ReadFile(hooksPath)
	if err != nil {
		t.Fatalf("Failed to read hooks.json: %v", err)
	}

	// Verify content
	var hooks HooksMap
	if err := json.Unmarshal(data, &hooks); err != nil {
		t.Fatalf("Failed to parse hooks.json: %v", err)
	}

	if len(hooks) != 2 {
		t.Errorf("Expected 2 hooks, got %d", len(hooks))
	}

	// Verify file permissions
	info, _ := os.Stat(hooksPath)
	if info.Mode().Perm() != 0644 {
		t.Errorf("hooks.json permissions = %o, want 0644", info.Mode().Perm())
	}

	// Verify JSON formatting (pretty-printed with 2 spaces)
	var prettyJSON map[string]interface{}
	json.Unmarshal(data, &prettyJSON)
	expectedJSON, _ := json.MarshalIndent(prettyJSON, "", "  ")

	// Trim trailing newline for comparison
	dataStr := string(data)
	if dataStr[len(dataStr)-1] == '\n' {
		dataStr = dataStr[:len(dataStr)-1]
	}

	if dataStr != string(expectedJSON) {
		t.Error("JSON is not properly formatted with 2-space indentation")
	}
}

func TestInstallHooks_Merge(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()
	claudeDir := filepath.Join(tempDir, ".claude")
	hooksPath := filepath.Join(claudeDir, "hooks.json")

	// Create .claude directory
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("Failed to create .claude directory: %v", err)
	}

	// Write existing hooks
	existingHooks := HooksMap{
		"custom-hook": {
			Command:     "echo 'custom'",
			Description: "Custom hook",
		},
		"another-hook": {
			Command:     "echo 'another'",
			Description: "Another hook",
		},
	}
	existingJSON, _ := json.MarshalIndent(existingHooks, "", "  ")
	if err := os.WriteFile(hooksPath, existingJSON, 0644); err != nil {
		t.Fatalf("Failed to write existing hooks: %v", err)
	}

	// Install juggler hooks
	if err := InstallHooks(tempDir); err != nil {
		t.Fatalf("InstallHooks() error = %v", err)
	}

	// Read merged hooks
	data, err := os.ReadFile(hooksPath)
	if err != nil {
		t.Fatalf("Failed to read hooks.json: %v", err)
	}

	var hooks HooksMap
	if err := json.Unmarshal(data, &hooks); err != nil {
		t.Fatalf("Failed to parse hooks.json: %v", err)
	}

	// Verify we have all hooks
	if len(hooks) != 4 {
		t.Errorf("Expected 4 hooks (2 existing + 2 juggler), got %d", len(hooks))
	}

	// Verify existing hooks are preserved
	if _, exists := hooks["custom-hook"]; !exists {
		t.Error("custom-hook was not preserved")
	}
	if _, exists := hooks["another-hook"]; !exists {
		t.Error("another-hook was not preserved")
	}

	// Verify juggler hooks are present
	if _, exists := hooks["user-prompt-submit"]; !exists {
		t.Error("user-prompt-submit was not added")
	}
	if _, exists := hooks["assistant-response-start"]; !exists {
		t.Error("assistant-response-start was not added")
	}
}

func TestInstallHooks_Update(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()
	claudeDir := filepath.Join(tempDir, ".claude")
	hooksPath := filepath.Join(claudeDir, "hooks.json")

	// Create .claude directory
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("Failed to create .claude directory: %v", err)
	}

	// Write existing hooks with old juggler hooks
	existingHooks := HooksMap{
		"user-prompt-submit": {
			Command:     "old-command",
			Description: "Old description",
		},
		"custom-hook": {
			Command:     "echo 'custom'",
			Description: "Custom hook",
		},
	}
	existingJSON, _ := json.MarshalIndent(existingHooks, "", "  ")
	if err := os.WriteFile(hooksPath, existingJSON, 0644); err != nil {
		t.Fatalf("Failed to write existing hooks: %v", err)
	}

	// Install/update juggler hooks
	if err := InstallHooks(tempDir); err != nil {
		t.Fatalf("InstallHooks() error = %v", err)
	}

	// Read updated hooks
	data, err := os.ReadFile(hooksPath)
	if err != nil {
		t.Fatalf("Failed to read hooks.json: %v", err)
	}

	var hooks HooksMap
	if err := json.Unmarshal(data, &hooks); err != nil {
		t.Fatalf("Failed to parse hooks.json: %v", err)
	}

	// Verify juggler hook was updated
	userPromptHook := hooks["user-prompt-submit"]
	if userPromptHook.Command != "juggle track-activity 2>/dev/null || true" {
		t.Errorf("user-prompt-submit not updated, got %q", userPromptHook.Command)
	}

	// Verify custom hook was preserved
	if _, exists := hooks["custom-hook"]; !exists {
		t.Error("custom-hook was not preserved during update")
	}
}

func TestRemoveHooks_All(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()
	claudeDir := filepath.Join(tempDir, ".claude")
	hooksPath := filepath.Join(claudeDir, "hooks.json")

	// Install hooks first
	if err := InstallHooks(tempDir); err != nil {
		t.Fatalf("InstallHooks() error = %v", err)
	}

	// Remove hooks
	if err := RemoveHooks(tempDir); err != nil {
		t.Fatalf("RemoveHooks() error = %v", err)
	}

	// Verify hooks.json was removed (since only juggler hooks existed)
	if _, err := os.Stat(hooksPath); !os.IsNotExist(err) {
		t.Error("hooks.json should have been removed when no other hooks exist")
	}
}

func TestRemoveHooks_PreserveOthers(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()
	claudeDir := filepath.Join(tempDir, ".claude")
	hooksPath := filepath.Join(claudeDir, "hooks.json")

	// Install hooks first
	if err := InstallHooks(tempDir); err != nil {
		t.Fatalf("InstallHooks() error = %v", err)
	}

	// Add a custom hook
	data, _ := os.ReadFile(hooksPath)
	var hooks HooksMap
	json.Unmarshal(data, &hooks)
	hooks["custom-hook"] = HookConfig{
		Command:     "echo 'custom'",
		Description: "Custom hook",
	}
	updatedJSON, _ := json.MarshalIndent(hooks, "", "  ")
	os.WriteFile(hooksPath, updatedJSON, 0644)

	// Remove juggler hooks
	if err := RemoveHooks(tempDir); err != nil {
		t.Fatalf("RemoveHooks() error = %v", err)
	}

	// Verify hooks.json still exists
	data, err := os.ReadFile(hooksPath)
	if err != nil {
		t.Fatalf("hooks.json was removed but should have been preserved: %v", err)
	}

	// Verify only custom hook remains
	var remainingHooks HooksMap
	if err := json.Unmarshal(data, &remainingHooks); err != nil {
		t.Fatalf("Failed to parse hooks.json: %v", err)
	}

	if len(remainingHooks) != 1 {
		t.Errorf("Expected 1 hook, got %d", len(remainingHooks))
	}

	if _, exists := remainingHooks["custom-hook"]; !exists {
		t.Error("custom-hook was not preserved")
	}

	// Verify juggler hooks were removed
	if _, exists := remainingHooks["user-prompt-submit"]; exists {
		t.Error("user-prompt-submit was not removed")
	}
	if _, exists := remainingHooks["assistant-response-start"]; exists {
		t.Error("assistant-response-start was not removed")
	}
}

func TestRemoveHooks_NoFile(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()

	// Try to remove hooks when file doesn't exist (should not error)
	if err := RemoveHooks(tempDir); err != nil {
		t.Fatalf("RemoveHooks() error = %v, want nil", err)
	}
}

func TestHooksInstalled_True(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()

	// Install hooks
	if err := InstallHooks(tempDir); err != nil {
		t.Fatalf("InstallHooks() error = %v", err)
	}

	// Check if installed
	installed, err := HooksInstalled(tempDir)
	if err != nil {
		t.Fatalf("HooksInstalled() error = %v", err)
	}

	if !installed {
		t.Error("HooksInstalled() = false, want true")
	}
}

func TestHooksInstalled_False_NoFile(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()

	// Check if installed (file doesn't exist)
	installed, err := HooksInstalled(tempDir)
	if err != nil {
		t.Fatalf("HooksInstalled() error = %v", err)
	}

	if installed {
		t.Error("HooksInstalled() = true, want false")
	}
}

func TestHooksInstalled_False_MissingHook(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()
	claudeDir := filepath.Join(tempDir, ".claude")
	hooksPath := filepath.Join(claudeDir, "hooks.json")

	// Create .claude directory
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("Failed to create .claude directory: %v", err)
	}

	// Write incomplete hooks (missing one)
	incompleteHooks := HooksMap{
		"user-prompt-submit": {
			Command:     "juggle track-activity 2>/dev/null || true",
			Description: "Track activity on current ball",
		},
	}
	incompleteJSON, _ := json.MarshalIndent(incompleteHooks, "", "  ")
	if err := os.WriteFile(hooksPath, incompleteJSON, 0644); err != nil {
		t.Fatalf("Failed to write incomplete hooks: %v", err)
	}

	// Check if installed
	installed, err := HooksInstalled(tempDir)
	if err != nil {
		t.Fatalf("HooksInstalled() error = %v", err)
	}

	if installed {
		t.Error("HooksInstalled() = true, want false (missing hook)")
	}
}

func TestHooksInstalled_False_WrongCommand(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()
	claudeDir := filepath.Join(tempDir, ".claude")
	hooksPath := filepath.Join(claudeDir, "hooks.json")

	// Create .claude directory
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("Failed to create .claude directory: %v", err)
	}

	// Write hooks with wrong command
	wrongHooks := HooksMap{
		"user-prompt-submit": {
			Command:     "wrong-command",
			Description: "Track activity on current ball",
		},
		"assistant-response-start": {
			Command:     "juggle reminder 2>/dev/null || true",
			Description: "Remind to check juggler state",
		},
	}
	wrongJSON, _ := json.MarshalIndent(wrongHooks, "", "  ")
	if err := os.WriteFile(hooksPath, wrongJSON, 0644); err != nil {
		t.Fatalf("Failed to write wrong hooks: %v", err)
	}

	// Check if installed
	installed, err := HooksInstalled(tempDir)
	if err != nil {
		t.Fatalf("HooksInstalled() error = %v", err)
	}

	if installed {
		t.Error("HooksInstalled() = true, want false (wrong command)")
	}
}

func TestHooksInstalled_WithOtherHooks(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()

	// Install hooks
	if err := InstallHooks(tempDir); err != nil {
		t.Fatalf("InstallHooks() error = %v", err)
	}

	// Add custom hook
	hooksPath := filepath.Join(tempDir, ".claude", "hooks.json")
	data, _ := os.ReadFile(hooksPath)
	var hooks HooksMap
	json.Unmarshal(data, &hooks)
	hooks["custom-hook"] = HookConfig{
		Command:     "echo 'custom'",
		Description: "Custom hook",
	}
	updatedJSON, _ := json.MarshalIndent(hooks, "", "  ")
	os.WriteFile(hooksPath, updatedJSON, 0644)

	// Check if installed (should still be true despite other hooks)
	installed, err := HooksInstalled(tempDir)
	if err != nil {
		t.Fatalf("HooksInstalled() error = %v", err)
	}

	if !installed {
		t.Error("HooksInstalled() = false, want true (should ignore other hooks)")
	}
}
