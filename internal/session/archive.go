package session

import (
	"sort"
	"strings"
	"time"
)

// LoadArchivedBalls loads all archived balls from archive/balls.jsonl in all projects
func LoadArchivedBalls(projectPaths []string) ([]*Session, error) {
	archivedBalls := make([]*Session, 0)

	for _, projectPath := range projectPaths {
		store, err := NewStore(projectPath)
		if err != nil {
			continue // Skip projects we can't access
		}

		balls, err := store.LoadArchivedBalls()
		if err != nil {
			continue // Skip if archive can't be read
		}

		archivedBalls = append(archivedBalls, balls...)
	}

	return archivedBalls, nil
}

// ArchiveQuery represents search parameters for querying archived balls
type ArchiveQuery struct {
	// Text search (searches intent)
	Query string

	// Filter by tags (OR logic)
	Tags []string

	// Filter by priority
	Priority Priority

	// Date range filters
	CompletedAfter  *time.Time
	CompletedBefore *time.Time

	// Limit number of results (0 = no limit)
	Limit int

	// Sort order
	SortBy ArchiveSortBy
}

// ArchiveSortBy defines sort options for archived balls
type ArchiveSortBy string

const (
	SortByCompletedDesc ArchiveSortBy = "completed-desc" // Most recently completed first
	SortByCompletedAsc  ArchiveSortBy = "completed-asc"  // Oldest completed first
	SortByPriority      ArchiveSortBy = "priority"       // Highest priority first
)

// QueryArchive searches archived balls based on the given query
func QueryArchive(projectPaths []string, query ArchiveQuery) ([]*Session, error) {
	balls, err := LoadArchivedBalls(projectPaths)
	if err != nil {
		return nil, err
	}

	// Apply filters
	filtered := make([]*Session, 0)

	for _, ball := range balls {
		// Text search filter
		if query.Query != "" {
			if !strings.Contains(strings.ToLower(ball.Intent), strings.ToLower(query.Query)) {
				continue
			}
		}

		// Tag filter (OR logic)
		if len(query.Tags) > 0 {
			hasTag := false
			for _, filterTag := range query.Tags {
				for _, ballTag := range ball.Tags {
					if ballTag == filterTag {
						hasTag = true
						break
					}
				}
				if hasTag {
					break
				}
			}
			if !hasTag {
				continue
			}
		}

		// Priority filter
		if query.Priority != "" && ball.Priority != query.Priority {
			continue
		}

		// Completed after filter
		if query.CompletedAfter != nil && ball.CompletedAt != nil {
			if ball.CompletedAt.Before(*query.CompletedAfter) {
				continue
			}
		}

		// Completed before filter
		if query.CompletedBefore != nil && ball.CompletedAt != nil {
			if ball.CompletedAt.After(*query.CompletedBefore) {
				continue
			}
		}

		filtered = append(filtered, ball)
	}

	// Apply sorting
	switch query.SortBy {
	case SortByCompletedDesc:
		sort.Slice(filtered, func(i, j int) bool {
			// Nil-safe comparison
			if filtered[i].CompletedAt == nil {
				return false
			}
			if filtered[j].CompletedAt == nil {
				return true
			}
			return filtered[i].CompletedAt.After(*filtered[j].CompletedAt)
		})
	case SortByCompletedAsc:
		sort.Slice(filtered, func(i, j int) bool {
			if filtered[i].CompletedAt == nil {
				return true
			}
			if filtered[j].CompletedAt == nil {
				return false
			}
			return filtered[i].CompletedAt.Before(*filtered[j].CompletedAt)
		})
	case SortByPriority:
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].PriorityWeight() > filtered[j].PriorityWeight()
		})
	default:
		// Default: most recently completed first
		sort.Slice(filtered, func(i, j int) bool {
			if filtered[i].CompletedAt == nil {
				return false
			}
			if filtered[j].CompletedAt == nil {
				return true
			}
			return filtered[i].CompletedAt.After(*filtered[j].CompletedAt)
		})
	}

	// Apply limit
	if query.Limit > 0 && len(filtered) > query.Limit {
		filtered = filtered[:query.Limit]
	}

	return filtered, nil
}

// GetArchiveStats returns statistics about archived balls
func GetArchiveStats(projectPaths []string) (*ArchiveStats, error) {
	balls, err := LoadArchivedBalls(projectPaths)
	if err != nil {
		return nil, err
	}

	stats := &ArchiveStats{
		TotalArchived: len(balls),
		ByPriority:    make(map[Priority]int),
		ByTag:         make(map[string]int),
	}

	for _, ball := range balls {
		// Count by priority
		stats.ByPriority[ball.Priority]++

		// Count by tag
		for _, tag := range ball.Tags {
			stats.ByTag[tag]++
		}

		// Calculate duration if we have both start and completion times
		if ball.CompletedAt != nil {
			duration := ball.CompletedAt.Sub(ball.StartedAt)
			stats.TotalDuration += duration

			if stats.ShortestDuration == 0 || duration < stats.ShortestDuration {
				stats.ShortestDuration = duration
			}
			if duration > stats.LongestDuration {
				stats.LongestDuration = duration
			}
		}
	}

	if len(balls) > 0 {
		stats.AverageDuration = stats.TotalDuration / time.Duration(len(balls))
	}

	return stats, nil
}

// ArchiveStats holds statistics about archived balls
type ArchiveStats struct {
	TotalArchived     int
	ByPriority        map[Priority]int
	ByTag             map[string]int
	TotalDuration     time.Duration
	AverageDuration   time.Duration
	LongestDuration   time.Duration
	ShortestDuration  time.Duration
}
