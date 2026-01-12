package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	defaultConfigPath = ".juggler/config.json"
)

// Config holds global juggler configuration
// ConfigOptions holds configurable options for global config
type ConfigOptions struct {
	ConfigHome     string // Override for ~/.juggler directory
	JugglerDirName string // Name of the juggler directory (default: ".juggler")
}

// DefaultConfigOptions returns the default config options
func DefaultConfigOptions() ConfigOptions {
	home, _ := os.UserHomeDir()
	return ConfigOptions{
		ConfigHome:     home,
		JugglerDirName: ".juggler",
	}
}

type Config struct {
	SearchPaths []string `json:"search_paths"`
}

// DefaultConfig returns a configuration with common project locations
// DefaultConfig returns a configuration with empty search paths
// Projects are added automatically when balls are created
func DefaultConfig() *Config {
	return &Config{
		SearchPaths: []string{},
	}
}

// LoadConfig loads configuration from ~/.juggler/config.json
func LoadConfig() (*Config, error) {
	return LoadConfigWithOptions(DefaultConfigOptions())
}

// LoadConfigWithOptions loads configuration with custom options
func LoadConfigWithOptions(opts ConfigOptions) (*Config, error) {
	if opts.ConfigHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		opts.ConfigHome = home
	}

	configPath := filepath.Join(opts.ConfigHome, opts.JugglerDirName, "config.json")

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

	configPath := filepath.Join(opts.ConfigHome, opts.JugglerDirName, "config.json")
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

// ProjectConfig holds per-project configuration stored in .juggler/config.json
// Note: NextBallCount is deprecated but kept for backward compatibility with existing config files
type ProjectConfig struct {
	NextBallCount int `json:"next_ball_count,omitempty"` // Deprecated: balls now use UUID-based IDs
}

// DefaultProjectConfig returns a new project config with initial values
func DefaultProjectConfig() *ProjectConfig {
	return &ProjectConfig{}
}

// LoadProjectConfig loads the project configuration from projectDir/.juggler/config.json
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

// SaveProjectConfig saves the project configuration to projectDir/.juggler/config.json
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

// GetAndIncrementBallCount is deprecated - kept for backward compatibility only.
// Ball IDs now use UUID-based generation instead of sequential counters.
// This function is no longer used but remains for any external callers.
func GetAndIncrementBallCount(projectDir string) (int, error) {
	config, err := LoadProjectConfig(projectDir)
	if err != nil {
		return 0, err
	}

	currentCount := config.NextBallCount
	if currentCount == 0 {
		currentCount = 1
	}
	config.NextBallCount = currentCount + 1

	if err := SaveProjectConfig(projectDir, config); err != nil {
		return 0, fmt.Errorf("failed to increment ball count: %w", err)
	}

	return currentCount, nil
}
