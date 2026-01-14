package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	defaultConfigPath = ".juggle/config.json"

	// Default values for global configuration fields
	// These are documented here as the canonical source of defaults
	DefaultIterationDelayMinutes = 0  // No delay between agent iterations by default
	DefaultIterationDelayFuzz    = 0  // No variance in delay by default
	DefaultOverloadRetryMinutes  = 10 // Wait 10 minutes before retrying after 529 overload exhaustion
)

// Config holds global juggle configuration
// ConfigOptions holds configurable options for global config
type ConfigOptions struct {
	ConfigHome    string // Override for ~/.juggle directory
	JuggleDirName string // Name of the juggle directory (default: ".juggle")
}

// DefaultConfigOptions returns the default config options
func DefaultConfigOptions() ConfigOptions {
	home, _ := os.UserHomeDir()
	return ConfigOptions{
		ConfigHome:    home,
		JuggleDirName: ".juggle",
	}
}

type Config struct {
	SearchPaths []string `json:"search_paths"`
	// Agent iteration delay settings
	IterationDelayMinutes int `json:"iteration_delay_minutes,omitempty"` // Base delay between iterations in minutes
	IterationDelayFuzz    int `json:"iteration_delay_fuzz,omitempty"`    // Random +/- variance in minutes
	// Overload retry settings (for 529 errors after Claude's built-in retries exhaust)
	OverloadRetryMinutes int `json:"overload_retry_minutes,omitempty"` // Minutes to wait before retrying after 529 overload exhaustion
	// VCS settings
	VCS string `json:"vcs,omitempty"` // Version control system: "git" or "jj"

	// UnknownFields stores any fields from the config file that aren't recognized.
	// These are preserved when saving to avoid data loss.
	UnknownFields map[string]interface{} `json:"-"`
}

// knownConfigFields lists the field names we recognize in config JSON
var knownConfigFields = map[string]bool{
	"search_paths":            true,
	"iteration_delay_minutes": true,
	"iteration_delay_fuzz":    true,
	"overload_retry_minutes":  true,
	"vcs":                     true,
}

// UnmarshalJSON implements custom JSON unmarshaling to capture unknown fields
func (c *Config) UnmarshalJSON(data []byte) error {
	// First, unmarshal into a map to capture all fields
	var rawMap map[string]interface{}
	if err := json.Unmarshal(data, &rawMap); err != nil {
		return err
	}

	// Unmarshal known fields using a type alias to avoid recursion
	type configAlias Config
	var alias configAlias
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}

	// Copy known fields
	c.SearchPaths = alias.SearchPaths
	c.IterationDelayMinutes = alias.IterationDelayMinutes
	c.IterationDelayFuzz = alias.IterationDelayFuzz
	c.OverloadRetryMinutes = alias.OverloadRetryMinutes
	c.VCS = alias.VCS

	// Extract unknown fields
	c.UnknownFields = make(map[string]interface{})
	for key, value := range rawMap {
		if !knownConfigFields[key] {
			c.UnknownFields[key] = value
		}
	}

	return nil
}

// MarshalJSON implements custom JSON marshaling to preserve unknown fields
func (c *Config) MarshalJSON() ([]byte, error) {
	// Start with unknown fields
	result := make(map[string]interface{})
	for key, value := range c.UnknownFields {
		result[key] = value
	}

	// Add known fields (they take precedence over unknown fields with same name)
	result["search_paths"] = c.SearchPaths
	if c.IterationDelayMinutes != 0 {
		result["iteration_delay_minutes"] = c.IterationDelayMinutes
	}
	if c.IterationDelayFuzz != 0 {
		result["iteration_delay_fuzz"] = c.IterationDelayFuzz
	}
	if c.OverloadRetryMinutes != 0 {
		result["overload_retry_minutes"] = c.OverloadRetryMinutes
	}
	if c.VCS != "" {
		result["vcs"] = c.VCS
	}

	return json.Marshal(result)
}

// GetUnknownFields returns the list of unrecognized field names
func (c *Config) GetUnknownFields() []string {
	keys := make([]string, 0, len(c.UnknownFields))
	for key := range c.UnknownFields {
		keys = append(keys, key)
	}
	return keys
}

// DefaultConfig returns a configuration with common project locations
// DefaultConfig returns a configuration with empty search paths
// Projects are added automatically when balls are created
func DefaultConfig() *Config {
	return &Config{
		SearchPaths:           []string{},
		IterationDelayMinutes: DefaultIterationDelayMinutes,
		IterationDelayFuzz:    DefaultIterationDelayFuzz,
		OverloadRetryMinutes:  DefaultOverloadRetryMinutes,
		UnknownFields:         make(map[string]interface{}),
	}
}

// LoadConfig loads configuration from ~/.juggle/config.json
func LoadConfig() (*Config, error) {
	return LoadConfigWithOptions(DefaultConfigOptions())
}

// LoadConfigWithOptions loads configuration with custom options.
// After loading, the config is saved back to disk to ensure any default values
// are populated and the file reflects the full config structure.
func LoadConfigWithOptions(opts ConfigOptions) (*Config, error) {
	if opts.ConfigHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		opts.ConfigHome = home
	}

	configPath := filepath.Join(opts.ConfigHome, opts.JuggleDirName, "config.json")

	// If config doesn't exist, create with defaults
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		config := DefaultConfig()
		if err := config.SaveWithOptions(opts); err != nil {
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}
		return config, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Ensure UnknownFields map is initialized
	if config.UnknownFields == nil {
		config.UnknownFields = make(map[string]interface{})
	}

	// Save the config back to disk to persist any defaults that were applied.
	// This ensures the file always has all known fields with their current values.
	if err := config.SaveWithOptions(opts); err != nil {
		return nil, fmt.Errorf("failed to save config with defaults: %w", err)
	}

	return &config, nil
}

// Save persists the configuration to disk
func (c *Config) Save() error {
	return c.SaveWithOptions(DefaultConfigOptions())
}

// SaveWithOptions persists the configuration with custom options
func (c *Config) SaveWithOptions(opts ConfigOptions) error {
	if opts.ConfigHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		opts.ConfigHome = home
	}

	configPath := filepath.Join(opts.ConfigHome, opts.JuggleDirName, "config.json")
	configDir := filepath.Dir(configPath)

	// Ensure directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// AddSearchPath adds a new search path if it doesn't already exist
func (c *Config) AddSearchPath(path string) bool {
	for _, existing := range c.SearchPaths {
		if existing == path {
			return false // Already exists
		}
	}
	c.SearchPaths = append(c.SearchPaths, path)
	return true
}

// RemoveSearchPath removes a search path
func (c *Config) RemoveSearchPath(path string) bool {
	for i, existing := range c.SearchPaths {
		if existing == path {
			c.SearchPaths = append(c.SearchPaths[:i], c.SearchPaths[i+1:]...)
			return true
		}
	}
	return false
}

// SetIterationDelay sets the delay between agent iterations.
// delayMinutes is the base delay in minutes, fuzz is the +/- variance in minutes.
func (c *Config) SetIterationDelay(delayMinutes, fuzz int) {
	c.IterationDelayMinutes = delayMinutes
	c.IterationDelayFuzz = fuzz
}

// GetIterationDelay returns the delay settings (delayMinutes, fuzz).
// Returns (0, 0) if not configured.
func (c *Config) GetIterationDelay() (delayMinutes, fuzz int) {
	return c.IterationDelayMinutes, c.IterationDelayFuzz
}

// HasIterationDelay returns true if iteration delay is configured.
func (c *Config) HasIterationDelay() bool {
	return c.IterationDelayMinutes > 0
}

// ClearIterationDelay removes the iteration delay configuration.
func (c *Config) ClearIterationDelay() {
	c.IterationDelayMinutes = 0
	c.IterationDelayFuzz = 0
}

// SetVCS sets the global VCS preference.
// Valid values are "git", "jj", or "" (empty for auto-detect).
func (c *Config) SetVCS(vcs string) error {
	if vcs != "" && vcs != "git" && vcs != "jj" {
		return fmt.Errorf("invalid VCS type: %s (must be 'git' or 'jj')", vcs)
	}
	c.VCS = vcs
	return nil
}

// GetVCS returns the global VCS preference.
func (c *Config) GetVCS() string {
	return c.VCS
}

// ClearVCS removes the VCS preference, enabling auto-detection.
func (c *Config) ClearVCS() {
	c.VCS = ""
}

// EnsureProjectInSearchPaths ensures a project directory is in the search paths
// This is called when creating balls to automatically track the project
func EnsureProjectInSearchPaths(projectDir string) error {
	config, err := LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Add the path if not already present
	if config.AddSearchPath(projectDir) {
		if err := config.Save(); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
	}

	return nil
}

// ProjectConfig holds per-project configuration stored in .juggle/config.json
type ProjectConfig struct {
	DefaultAcceptanceCriteria []string `json:"default_acceptance_criteria,omitempty"` // Repo-level ACs applied to all sessions
	VCS                       string   `json:"vcs,omitempty"`                         // Version control system: "git" or "jj"
}

// DefaultProjectConfig returns a new project config with initial values
func DefaultProjectConfig() *ProjectConfig {
	return &ProjectConfig{}
}

// LoadProjectConfig loads the project configuration from projectDir/.juggle/config.json
func LoadProjectConfig(projectDir string) (*ProjectConfig, error) {
	configPath := filepath.Join(projectDir, projectStorePath, "config.json")

	// If config doesn't exist, create with defaults
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		config := DefaultProjectConfig()
		if err := SaveProjectConfig(projectDir, config); err != nil {
			return nil, fmt.Errorf("failed to create default project config: %w", err)
		}
		return config, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read project config file: %w", err)
	}

	var config ProjectConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal project config: %w", err)
	}

	return &config, nil
}

// SaveProjectConfig saves the project configuration to projectDir/.juggle/config.json
func SaveProjectConfig(projectDir string, config *ProjectConfig) error {
	configDir := filepath.Join(projectDir, projectStorePath)
	configPath := filepath.Join(configDir, "config.json")

	// Ensure directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create project config directory: %w", err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal project config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write project config file: %w", err)
	}

	return nil
}

// SetDefaultAcceptanceCriteria sets repo-level acceptance criteria
func (c *ProjectConfig) SetDefaultAcceptanceCriteria(criteria []string) {
	c.DefaultAcceptanceCriteria = criteria
}

// HasDefaultAcceptanceCriteria returns true if the project has default ACs
func (c *ProjectConfig) HasDefaultAcceptanceCriteria() bool {
	return len(c.DefaultAcceptanceCriteria) > 0
}

// SetVCS sets the project VCS preference.
// Valid values are "git", "jj", or "" (empty for inherit from global/auto-detect).
func (c *ProjectConfig) SetVCS(vcs string) error {
	if vcs != "" && vcs != "git" && vcs != "jj" {
		return fmt.Errorf("invalid VCS type: %s (must be 'git' or 'jj')", vcs)
	}
	c.VCS = vcs
	return nil
}

// GetVCS returns the project VCS preference.
func (c *ProjectConfig) GetVCS() string {
	return c.VCS
}

// ClearVCS removes the project VCS preference.
func (c *ProjectConfig) ClearVCS() {
	c.VCS = ""
}

// UpdateProjectAcceptanceCriteria updates the repo-level acceptance criteria
func UpdateProjectAcceptanceCriteria(projectDir string, criteria []string) error {
	config, err := LoadProjectConfig(projectDir)
	if err != nil {
		return err
	}

	config.SetDefaultAcceptanceCriteria(criteria)
	return SaveProjectConfig(projectDir, config)
}

// GetProjectAcceptanceCriteria returns the repo-level acceptance criteria
func GetProjectAcceptanceCriteria(projectDir string) ([]string, error) {
	config, err := LoadProjectConfig(projectDir)
	if err != nil {
		return nil, err
	}

	return config.DefaultAcceptanceCriteria, nil
}

// UpdateGlobalIterationDelay updates the iteration delay in global config
func UpdateGlobalIterationDelay(delayMinutes, fuzz int) error {
	return UpdateGlobalIterationDelayWithOptions(DefaultConfigOptions(), delayMinutes, fuzz)
}

// UpdateGlobalIterationDelayWithOptions updates the iteration delay with custom options
func UpdateGlobalIterationDelayWithOptions(opts ConfigOptions, delayMinutes, fuzz int) error {
	config, err := LoadConfigWithOptions(opts)
	if err != nil {
		return err
	}

	config.SetIterationDelay(delayMinutes, fuzz)
	return config.SaveWithOptions(opts)
}

// GetGlobalIterationDelay returns the iteration delay settings from global config
func GetGlobalIterationDelay() (delayMinutes, fuzz int, err error) {
	return GetGlobalIterationDelayWithOptions(DefaultConfigOptions())
}

// GetGlobalIterationDelayWithOptions returns the iteration delay with custom options
func GetGlobalIterationDelayWithOptions(opts ConfigOptions) (delayMinutes, fuzz int, err error) {
	config, err := LoadConfigWithOptions(opts)
	if err != nil {
		return 0, 0, err
	}

	delay, fuzz := config.GetIterationDelay()
	return delay, fuzz, nil
}

// ClearGlobalIterationDelay removes the iteration delay from global config
func ClearGlobalIterationDelay() error {
	return ClearGlobalIterationDelayWithOptions(DefaultConfigOptions())
}

// ClearGlobalIterationDelayWithOptions removes the iteration delay with custom options
func ClearGlobalIterationDelayWithOptions(opts ConfigOptions) error {
	config, err := LoadConfigWithOptions(opts)
	if err != nil {
		return err
	}

	config.ClearIterationDelay()
	return config.SaveWithOptions(opts)
}

// SetOverloadRetryMinutes sets how long to wait before retrying after 529 overload exhaustion.
func (c *Config) SetOverloadRetryMinutes(minutes int) {
	c.OverloadRetryMinutes = minutes
}

// GetOverloadRetryMinutes returns the overload retry interval in minutes.
// Returns the default (10) if not configured or set to 0.
func (c *Config) GetOverloadRetryMinutes() int {
	if c.OverloadRetryMinutes == 0 {
		return DefaultOverloadRetryMinutes
	}
	return c.OverloadRetryMinutes
}

// GetGlobalOverloadRetryMinutes returns the overload retry setting from global config
func GetGlobalOverloadRetryMinutes() (int, error) {
	return GetGlobalOverloadRetryMinutesWithOptions(DefaultConfigOptions())
}

// GetGlobalOverloadRetryMinutesWithOptions returns the overload retry setting with custom options
func GetGlobalOverloadRetryMinutesWithOptions(opts ConfigOptions) (int, error) {
	config, err := LoadConfigWithOptions(opts)
	if err != nil {
		return DefaultOverloadRetryMinutes, err
	}
	return config.GetOverloadRetryMinutes(), nil
}

// UpdateGlobalOverloadRetryMinutes updates the overload retry setting in global config
func UpdateGlobalOverloadRetryMinutes(minutes int) error {
	return UpdateGlobalOverloadRetryMinutesWithOptions(DefaultConfigOptions(), minutes)
}

// UpdateGlobalOverloadRetryMinutesWithOptions updates the overload retry setting with custom options
func UpdateGlobalOverloadRetryMinutesWithOptions(opts ConfigOptions, minutes int) error {
	config, err := LoadConfigWithOptions(opts)
	if err != nil {
		return err
	}

	config.SetOverloadRetryMinutes(minutes)
	return config.SaveWithOptions(opts)
}

// GetGlobalVCS returns the VCS setting from global config
func GetGlobalVCS() (string, error) {
	return GetGlobalVCSWithOptions(DefaultConfigOptions())
}

// GetGlobalVCSWithOptions returns the VCS setting with custom options
func GetGlobalVCSWithOptions(opts ConfigOptions) (string, error) {
	config, err := LoadConfigWithOptions(opts)
	if err != nil {
		return "", err
	}
	return config.GetVCS(), nil
}

// UpdateGlobalVCS updates the VCS setting in global config
func UpdateGlobalVCS(vcs string) error {
	return UpdateGlobalVCSWithOptions(DefaultConfigOptions(), vcs)
}

// UpdateGlobalVCSWithOptions updates the VCS setting with custom options
func UpdateGlobalVCSWithOptions(opts ConfigOptions, vcs string) error {
	config, err := LoadConfigWithOptions(opts)
	if err != nil {
		return err
	}

	if err := config.SetVCS(vcs); err != nil {
		return err
	}
	return config.SaveWithOptions(opts)
}

// ClearGlobalVCS clears the VCS setting from global config
func ClearGlobalVCS() error {
	return ClearGlobalVCSWithOptions(DefaultConfigOptions())
}

// ClearGlobalVCSWithOptions clears the VCS setting with custom options
func ClearGlobalVCSWithOptions(opts ConfigOptions) error {
	config, err := LoadConfigWithOptions(opts)
	if err != nil {
		return err
	}

	config.ClearVCS()
	return config.SaveWithOptions(opts)
}

// GetProjectVCS returns the VCS setting from project config
func GetProjectVCS(projectDir string) (string, error) {
	config, err := LoadProjectConfig(projectDir)
	if err != nil {
		return "", err
	}
	return config.GetVCS(), nil
}

// UpdateProjectVCS updates the VCS setting in project config
func UpdateProjectVCS(projectDir, vcs string) error {
	config, err := LoadProjectConfig(projectDir)
	if err != nil {
		return err
	}

	if err := config.SetVCS(vcs); err != nil {
		return err
	}
	return SaveProjectConfig(projectDir, config)
}

// ClearProjectVCS clears the VCS setting from project config
func ClearProjectVCS(projectDir string) error {
	config, err := LoadProjectConfig(projectDir)
	if err != nil {
		return err
	}

	config.ClearVCS()
	return SaveProjectConfig(projectDir, config)
}
