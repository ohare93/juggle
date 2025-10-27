package claude

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// HookConfig represents a single hook configuration
type HookConfig struct {
	Command     string `json:"command"`
	Description string `json:"description"`
}

// HooksMap represents the hooks.json file structure
type HooksMap map[string]HookConfig

// GetHooksTemplate returns the hooks configuration as a JSON string
func GetHooksTemplate() (string, error) {
	hooks := HooksMap{
		"user-prompt-submit": {
			Command:     "juggle track-activity 2>/dev/null || true",
			Description: "Track activity on current ball",
		},
		"assistant-response-start": {
			Command:     "juggle reminder 2>/dev/null || true",
			Description: "Remind to check juggler state",
		},
	}

	jsonBytes, err := json.MarshalIndent(hooks, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal hooks template: %w", err)
	}

	return string(jsonBytes), nil
}

// InstallHooks writes hooks.json to the specified directory
// Creates .claude directory if needed
// Merges with existing hooks if any (doesn't overwrite other hooks)
func InstallHooks(projectDir string) error {
	claudeDir := filepath.Join(projectDir, ".claude")
	hooksPath := filepath.Join(claudeDir, "hooks.json")

	// Ensure .claude directory exists
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		return fmt.Errorf("failed to create .claude directory: %w", err)
	}

	// Get juggler hooks
	jugglerHooks := HooksMap{
		"user-prompt-submit": {
			Command:     "juggle track-activity 2>/dev/null || true",
			Description: "Track activity on current ball",
		},
		"assistant-response-start": {
			Command:     "juggle reminder 2>/dev/null || true",
			Description: "Remind to check juggler state",
		},
	}

	// Read existing hooks if file exists
	existingHooks := make(HooksMap)
	if FileExists(hooksPath) {
		data, err := os.ReadFile(hooksPath)
		if err != nil {
			return fmt.Errorf("failed to read existing hooks.json: %w", err)
		}

		if err := json.Unmarshal(data, &existingHooks); err != nil {
			return fmt.Errorf("failed to parse existing hooks.json: %w", err)
		}
	}

	// Merge hooks - juggler hooks take precedence
	for key, value := range jugglerHooks {
		existingHooks[key] = value
	}

	// Write merged hooks back
	jsonBytes, err := json.MarshalIndent(existingHooks, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal hooks: %w", err)
	}

	// Add trailing newline
	jsonBytes = append(jsonBytes, '\n')

	if err := os.WriteFile(hooksPath, jsonBytes, 0644); err != nil {
		return fmt.Errorf("failed to write hooks.json: %w", err)
	}

	return nil
}

// RemoveHooks removes juggler-specific hooks from hooks.json
// Preserves other hooks if they exist
func RemoveHooks(projectDir string) error {
	hooksPath := filepath.Join(projectDir, ".claude", "hooks.json")

	// If hooks.json doesn't exist, nothing to do
	if !FileExists(hooksPath) {
		return nil
	}

	// Read existing hooks
	data, err := os.ReadFile(hooksPath)
	if err != nil {
		return fmt.Errorf("failed to read hooks.json: %w", err)
	}

	existingHooks := make(HooksMap)
	if err := json.Unmarshal(data, &existingHooks); err != nil {
		return fmt.Errorf("failed to parse hooks.json: %w", err)
	}

	// Remove juggler hooks
	delete(existingHooks, "user-prompt-submit")
	delete(existingHooks, "assistant-response-start")

	// If no hooks remain, remove the file
	if len(existingHooks) == 0 {
		if err := os.Remove(hooksPath); err != nil {
			return fmt.Errorf("failed to remove hooks.json: %w", err)
		}
		return nil
	}

	// Write remaining hooks back
	jsonBytes, err := json.MarshalIndent(existingHooks, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal hooks: %w", err)
	}

	// Add trailing newline
	jsonBytes = append(jsonBytes, '\n')

	if err := os.WriteFile(hooksPath, jsonBytes, 0644); err != nil {
		return fmt.Errorf("failed to write hooks.json: %w", err)
	}

	return nil
}

// HooksInstalled checks if juggler hooks are present
func HooksInstalled(projectDir string) (bool, error) {
	hooksPath := filepath.Join(projectDir, ".claude", "hooks.json")

	// If file doesn't exist, hooks are not installed
	if !FileExists(hooksPath) {
		return false, nil
	}

	// Read hooks file
	data, err := os.ReadFile(hooksPath)
	if err != nil {
		return false, fmt.Errorf("failed to read hooks.json: %w", err)
	}

	hooks := make(HooksMap)
	if err := json.Unmarshal(data, &hooks); err != nil {
		return false, fmt.Errorf("failed to parse hooks.json: %w", err)
	}

	// Check if both juggler hooks are present with correct commands
	userPromptHook, hasUserPrompt := hooks["user-prompt-submit"]
	assistantHook, hasAssistant := hooks["assistant-response-start"]

	if !hasUserPrompt || !hasAssistant {
		return false, nil
	}

	// Verify the commands contain juggle
	if userPromptHook.Command != "juggle track-activity 2>/dev/null || true" {
		return false, nil
	}

	if assistantHook.Command != "juggle reminder 2>/dev/null || true" {
		return false, nil
	}

	return true, nil
}
