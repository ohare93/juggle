package session

import (
	"fmt"
	"os"
	"path/filepath"
)

// DiscoverProjects finds all directories containing .juggler folders
// DiscoverProjects finds all directories containing .juggler folders
func DiscoverProjects(config *Config) ([]string, error) {
	projects := make([]string, 0)

	for _, path := range config.SearchPaths {
		// Check if path exists and has .juggler directory
		jugglerPath := filepath.Join(path, ".juggler")
		if _, err := os.Stat(jugglerPath); err == nil {
			projects = append(projects, path)
		}
	}

	return projects, nil
}

// LoadAllBalls loads balls from all discovered projects
func LoadAllBalls(projectPaths []string) ([]*Session, error) {
	allBalls := make([]*Session, 0)

	for _, projectPath := range projectPaths {
		store, err := NewStore(projectPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to create store for %s: %v\n", projectPath, err)
			continue
		}

		balls, err := store.LoadBalls()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to load balls from %s: %v\n", projectPath, err)
			continue
		}

		allBalls = append(allBalls, balls...)
	}

	return allBalls, nil
}

// LoadInProgressBalls loads all in_progress balls from all projects
func LoadInProgressBalls(projectPaths []string) ([]*Session, error) {
	allBalls, err := LoadAllBalls(projectPaths)
	if err != nil {
		return nil, err
	}

	inProgress := make([]*Session, 0)
	for _, ball := range allBalls {
		if ball.State == StateInProgress {
			inProgress = append(inProgress, ball)
		}
	}

	return inProgress, nil
}

// LoadJugglingBalls loads all balls currently being juggled from all projects
// DEPRECATED: Use LoadInProgressBalls instead.
func LoadJugglingBalls(projectPaths []string) ([]*Session, error) {
	return LoadInProgressBalls(projectPaths)
}

// LoadPendingBalls loads all pending balls from all projects
func LoadPendingBalls(projectPaths []string) ([]*Session, error) {
	allBalls, err := LoadAllBalls(projectPaths)
	if err != nil {
		return nil, err
	}

	pending := make([]*Session, 0)
	for _, ball := range allBalls {
		if ball.State == StatePending {
			pending = append(pending, ball)
		}
	}

	return pending, nil
}

// LoadReadyBalls loads all ready balls from all projects
// DEPRECATED: Use LoadPendingBalls instead.
func LoadReadyBalls(projectPaths []string) ([]*Session, error) {
	return LoadPendingBalls(projectPaths)
}

// ProjectInfo holds information about a project and its balls
type ProjectInfo struct {
	Path            string
	Name            string
	TotalBalls      int
	PendingBalls    int
	InProgressBalls int
	BlockedBalls    int
	CompleteBalls   int

	// Legacy field aliases for backward compatibility
	JugglingBalls    int // Alias for InProgressBalls
	ReadyBalls       int // Alias for PendingBalls
	DroppedBalls     int // Alias for BlockedBalls
	NeedsThrownBalls int // Deprecated - always 0
	InAirBalls       int // Deprecated - always 0
	NeedsCaughtBalls int // Deprecated - always 0
}

// GetProjectsInfo returns information about all projects
func GetProjectsInfo(config *Config) ([]*ProjectInfo, error) {
	projectPaths, err := DiscoverProjects(config)
	if err != nil {
		return nil, err
	}

	infos := make([]*ProjectInfo, 0, len(projectPaths))

	for _, projectPath := range projectPaths {
		store, err := NewStore(projectPath)
		if err != nil {
			continue
		}

		balls, err := store.LoadBalls()
		if err != nil {
			continue
		}

		info := &ProjectInfo{
			Path:       projectPath,
			Name:       filepath.Base(projectPath),
			TotalBalls: len(balls),
		}

		for _, ball := range balls {
			switch ball.State {
			case StatePending:
				info.PendingBalls++
			case StateInProgress:
				info.InProgressBalls++
			case StateBlocked:
				info.BlockedBalls++
			case StateComplete:
				info.CompleteBalls++
			}
		}

		// Set legacy field aliases
		info.JugglingBalls = info.InProgressBalls
		info.ReadyBalls = info.PendingBalls
		info.DroppedBalls = info.BlockedBalls

		// Only include projects with balls
		if info.TotalBalls > 0 {
			infos = append(infos, info)
		}
	}

	return infos, nil
}
