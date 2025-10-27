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

// LoadActiveBalls loads all active balls (active, blocked, needs-review) from all projects
// LoadJugglingBalls loads all balls currently being juggled from all projects
func LoadJugglingBalls(projectPaths []string) ([]*Session, error) {
	allBalls, err := LoadAllBalls(projectPaths)
	if err != nil {
		return nil, err
	}

	juggling := make([]*Session, 0)
	for _, ball := range allBalls {
		if ball.ActiveState == ActiveJuggling {
			juggling = append(juggling, ball)
		}
	}

	return juggling, nil
}

// LoadPlannedBalls loads all planned balls from all projects
// LoadReadyBalls loads all ready balls from all projects
func LoadReadyBalls(projectPaths []string) ([]*Session, error) {
	allBalls, err := LoadAllBalls(projectPaths)
	if err != nil {
		return nil, err
	}

	readyBalls := make([]*Session, 0)
	for _, ball := range allBalls {
		if ball.ActiveState == ActiveReady {
			readyBalls = append(readyBalls, ball)
		}
	}

	return readyBalls, nil
}

// ProjectInfo holds information about a project and its balls
type ProjectInfo struct {
	Path              string
	Name              string
	TotalBalls        int
	JugglingBalls     int
	ReadyBalls        int
	DroppedBalls      int
	CompleteBalls     int
	NeedsThrownBalls  int
	InAirBalls        int
	NeedsCaughtBalls  int
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
			Path: projectPath,
			Name: filepath.Base(projectPath),
			TotalBalls: len(balls),
		}

		for _, ball := range balls {
			switch ball.ActiveState {
			case ActiveReady:
				info.ReadyBalls++
			case ActiveJuggling:
				info.JugglingBalls++
				// Also count juggle states
				if ball.JuggleState != nil {
					switch *ball.JuggleState {
					case JuggleNeedsThrown:
						info.NeedsThrownBalls++
					case JuggleInAir:
						info.InAirBalls++
					case JuggleNeedsCaught:
						info.NeedsCaughtBalls++
					}
				}
			case ActiveDropped:
				info.DroppedBalls++
			case ActiveComplete:
				info.CompleteBalls++
			}
		}

		// Only include projects with balls
		if info.TotalBalls > 0 {
			infos = append(infos, info)
		}
	}

	return infos, nil
}
