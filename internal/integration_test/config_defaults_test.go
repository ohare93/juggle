package integration_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/ohare93/juggle/internal/session"
)

// TestConfigDefaults_SparseConfig tests that loading a sparse config populates defaults
func TestConfigDefaults_SparseConfig(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a sparse config with only search_paths (missing delay fields)
	configPath := filepath.Join(env.ConfigHome, ".juggler", "config.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	sparseConfig := map[string]interface{}{
		"search_paths": []string{"/some/path"},
	}
	data, _ := json.MarshalIndent(sparseConfig, "", "  ")
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("Failed to write sparse config: %v", err)
	}

	// Load the config (should trigger default population)
	opts := session.ConfigOptions{
		ConfigHome:     env.ConfigHome,
		JugglerDirName: ".juggler",
	}
	config, err := session.LoadConfigWithOptions(opts)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify search_paths was preserved
	if len(config.SearchPaths) != 1 || config.SearchPaths[0] != "/some/path" {
		t.Errorf("Expected search_paths to be preserved, got: %v", config.SearchPaths)
	}

	// Verify defaults were applied (0 is the default for delay fields)
	if config.IterationDelayMinutes != session.DefaultIterationDelayMinutes {
		t.Errorf("Expected IterationDelayMinutes to be %d, got: %d",
			session.DefaultIterationDelayMinutes, config.IterationDelayMinutes)
	}
	if config.IterationDelayFuzz != session.DefaultIterationDelayFuzz {
		t.Errorf("Expected IterationDelayFuzz to be %d, got: %d",
			session.DefaultIterationDelayFuzz, config.IterationDelayFuzz)
	}

	// Verify the file was updated by re-reading it directly
	updatedData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read updated config: %v", err)
	}

	var updatedConfig map[string]interface{}
	if err := json.Unmarshal(updatedData, &updatedConfig); err != nil {
		t.Fatalf("Failed to parse updated config: %v", err)
	}

	// The file should now have search_paths field
	if _, ok := updatedConfig["search_paths"]; !ok {
		t.Error("Expected updated config to have 'search_paths' field")
	}
}

// TestConfigDefaults_UnknownFieldsPreserved tests that unknown fields are preserved
func TestConfigDefaults_UnknownFieldsPreserved(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a config with unknown fields
	configPath := filepath.Join(env.ConfigHome, ".juggler", "config.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	configWithUnknown := map[string]interface{}{
		"search_paths":    []string{},
		"foo_bar":         "some_value",
		"unknown_setting": 42,
		"nested_unknown": map[string]interface{}{
			"key": "value",
		},
	}
	data, _ := json.MarshalIndent(configWithUnknown, "", "  ")
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Load the config
	opts := session.ConfigOptions{
		ConfigHome:     env.ConfigHome,
		JugglerDirName: ".juggler",
	}
	config, err := session.LoadConfigWithOptions(opts)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify unknown fields are tracked
	unknownFields := config.GetUnknownFields()
	sort.Strings(unknownFields)

	expectedUnknown := []string{"foo_bar", "nested_unknown", "unknown_setting"}
	if len(unknownFields) != len(expectedUnknown) {
		t.Errorf("Expected %d unknown fields, got %d: %v", len(expectedUnknown), len(unknownFields), unknownFields)
	}

	for i, field := range expectedUnknown {
		if i >= len(unknownFields) || unknownFields[i] != field {
			t.Errorf("Expected unknown field '%s' at position %d", field, i)
		}
	}

	// Verify the file still has unknown fields after load (which saves)
	updatedData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read updated config: %v", err)
	}

	var updatedConfig map[string]interface{}
	if err := json.Unmarshal(updatedData, &updatedConfig); err != nil {
		t.Fatalf("Failed to parse updated config: %v", err)
	}

	// Check unknown fields are preserved
	if _, ok := updatedConfig["foo_bar"]; !ok {
		t.Error("Expected 'foo_bar' to be preserved in config file")
	}
	if _, ok := updatedConfig["unknown_setting"]; !ok {
		t.Error("Expected 'unknown_setting' to be preserved in config file")
	}
	if _, ok := updatedConfig["nested_unknown"]; !ok {
		t.Error("Expected 'nested_unknown' to be preserved in config file")
	}
}

// TestConfigDefaults_EmptyConfigPopulated tests creating config from scratch
func TestConfigDefaults_EmptyConfigPopulated(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Don't create any config file - let LoadConfigWithOptions create defaults
	opts := session.ConfigOptions{
		ConfigHome:     env.ConfigHome,
		JugglerDirName: ".juggler",
	}

	// Ensure the config file doesn't exist
	configPath := filepath.Join(env.ConfigHome, ".juggler", "config.json")
	if _, err := os.Stat(configPath); err == nil {
		os.Remove(configPath)
	}

	// Load config (should create default)
	config, err := session.LoadConfigWithOptions(opts)
	if err != nil {
		t.Fatalf("Failed to load/create config: %v", err)
	}

	// Verify it was created with defaults
	if config.SearchPaths == nil {
		t.Error("Expected SearchPaths to be initialized")
	}
	if config.IterationDelayMinutes != session.DefaultIterationDelayMinutes {
		t.Errorf("Expected default IterationDelayMinutes")
	}

	// Verify file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Expected config file to be created")
	}
}

// TestConfigDefaults_UnknownFieldsGetterEmpty tests GetUnknownFields on config without unknowns
func TestConfigDefaults_UnknownFieldsGetterEmpty(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a valid config without unknown fields
	configPath := filepath.Join(env.ConfigHome, ".juggler", "config.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	validConfig := map[string]interface{}{
		"search_paths":            []string{"/path1", "/path2"},
		"iteration_delay_minutes": 5,
		"iteration_delay_fuzz":    2,
	}
	data, _ := json.MarshalIndent(validConfig, "", "  ")
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Load config
	opts := session.ConfigOptions{
		ConfigHome:     env.ConfigHome,
		JugglerDirName: ".juggler",
	}
	config, err := session.LoadConfigWithOptions(opts)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// GetUnknownFields should return empty slice
	unknownFields := config.GetUnknownFields()
	if len(unknownFields) != 0 {
		t.Errorf("Expected no unknown fields, got: %v", unknownFields)
	}

	// Verify known values were loaded correctly
	if config.IterationDelayMinutes != 5 {
		t.Errorf("Expected IterationDelayMinutes=5, got: %d", config.IterationDelayMinutes)
	}
	if config.IterationDelayFuzz != 2 {
		t.Errorf("Expected IterationDelayFuzz=2, got: %d", config.IterationDelayFuzz)
	}
	if len(config.SearchPaths) != 2 {
		t.Errorf("Expected 2 search paths, got: %d", len(config.SearchPaths))
	}
}

// TestConfigDefaults_SavePreservesUnknown tests that saving config preserves unknown fields
func TestConfigDefaults_SavePreservesUnknown(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a config with unknown fields
	configPath := filepath.Join(env.ConfigHome, ".juggler", "config.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	configWithUnknown := map[string]interface{}{
		"search_paths":   []string{},
		"custom_setting": "preserve_me",
	}
	data, _ := json.MarshalIndent(configWithUnknown, "", "  ")
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Load config
	opts := session.ConfigOptions{
		ConfigHome:     env.ConfigHome,
		JugglerDirName: ".juggler",
	}
	config, err := session.LoadConfigWithOptions(opts)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Modify a known field
	config.AddSearchPath("/new/path")

	// Save the config
	if err := config.SaveWithOptions(opts); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Re-read and verify unknown field is still there
	updatedData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read updated config: %v", err)
	}

	var updatedConfig map[string]interface{}
	if err := json.Unmarshal(updatedData, &updatedConfig); err != nil {
		t.Fatalf("Failed to parse updated config: %v", err)
	}

	// Custom setting should be preserved
	if val, ok := updatedConfig["custom_setting"]; !ok || val != "preserve_me" {
		t.Error("Expected 'custom_setting' to be preserved with value 'preserve_me'")
	}

	// New search path should be added
	paths, ok := updatedConfig["search_paths"].([]interface{})
	if !ok || len(paths) != 1 || paths[0] != "/new/path" {
		t.Errorf("Expected search_paths to contain '/new/path', got: %v", updatedConfig["search_paths"])
	}
}
