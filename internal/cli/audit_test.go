package cli

import (
	"testing"
	"time"

	"github.com/ohare93/juggle/internal/session"
)

func TestCalculateCompletionRatio(t *testing.T) {
	tests := []struct {
		name     string
		metrics  *ProjectMetrics
		expected float64
	}{
		{
			name: "50% completion",
			metrics: &ProjectMetrics{
				PendingCount:     5,
				InProgressCount:  3,
				BlockedCount:   2,
				CompletedCount: 10,
			},
			expected: 50.0,
		},
		{
			name: "100% completion",
			metrics: &ProjectMetrics{
				PendingCount:     0,
				InProgressCount:  0,
				BlockedCount:   0,
				CompletedCount: 10,
			},
			expected: 100.0,
		},
		{
			name: "0% completion",
			metrics: &ProjectMetrics{
				PendingCount:     5,
				InProgressCount:  3,
				BlockedCount:   2,
				CompletedCount: 0,
			},
			expected: 0.0,
		},
		{
			name: "no balls",
			metrics: &ProjectMetrics{
				PendingCount:     0,
				InProgressCount:  0,
				BlockedCount:   0,
				CompletedCount: 0,
			},
			expected: 0.0,
		},
		{
			name: "low completion 20%",
			metrics: &ProjectMetrics{
				PendingCount:     20,
				InProgressCount:  10,
				BlockedCount:   10,
				CompletedCount: 10,
			},
			expected: 20.0,
		},
		{
			name: "high completion 80%",
			metrics: &ProjectMetrics{
				PendingCount:     2,
				InProgressCount:  1,
				BlockedCount:   2,
				CompletedCount: 20,
			},
			expected: 80.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateCompletionRatio(tt.metrics)
			if result != tt.expected {
				t.Errorf("expected %.2f, got %.2f", tt.expected, result)
			}
		})
	}
}

func TestCalculateProjectMetrics(t *testing.T) {
	now := time.Now()
	staleTime := now.Add(-35 * 24 * time.Hour) // 35 days ago
	recentTime := now.Add(-10 * 24 * time.Hour) // 10 days ago

	balls := []*session.Ball{
		// Project A
		createTestBall(t, "project-a", "/path/to/a", session.StatePending, recentTime),
		createTestBall(t, "project-a", "/path/to/a", session.StatePending, staleTime),
		createTestBall(t, "project-a", "/path/to/a", session.StateInProgress, now),
		createTestBall(t, "project-a", "/path/to/a", session.StateComplete, now),
		createTestBall(t, "project-a", "/path/to/a", session.StateBlocked, now),
		// Project B
		createTestBall(t, "project-b", "/path/to/b", session.StatePending, staleTime),
		createTestBall(t, "project-b", "/path/to/b", session.StatePending, staleTime),
		createTestBall(t, "project-b", "/path/to/b", session.StateComplete, now),
	}

	metricsMap := calculateProjectMetrics(balls)

	// Verify Project A
	metricsA := metricsMap["/path/to/a"]
	if metricsA == nil {
		t.Fatal("expected metrics for project A")
	}
	if metricsA.PendingCount != 2 {
		t.Errorf("project A: expected 2 pending, got %d", metricsA.PendingCount)
	}
	if metricsA.InProgressCount != 1 {
		t.Errorf("project A: expected 1 in-progress, got %d", metricsA.InProgressCount)
	}
	if metricsA.CompletedCount != 1 {
		t.Errorf("project A: expected 1 completed, got %d", metricsA.CompletedCount)
	}
	if metricsA.BlockedCount != 1 {
		t.Errorf("project A: expected 1 blocked, got %d", metricsA.BlockedCount)
	}
	if metricsA.StalePendingCount != 1 {
		t.Errorf("project A: expected 1 stale pending, got %d", metricsA.StalePendingCount)
	}
	if !metricsA.HasCompletedBalls {
		t.Error("project A: expected HasCompletedBalls to be true")
	}

	// Verify Project B
	metricsB := metricsMap["/path/to/b"]
	if metricsB == nil {
		t.Fatal("expected metrics for project B")
	}
	if metricsB.PendingCount != 2 {
		t.Errorf("project B: expected 2 pending, got %d", metricsB.PendingCount)
	}
	if metricsB.CompletedCount != 1 {
		t.Errorf("project B: expected 1 completed, got %d", metricsB.CompletedCount)
	}
	if metricsB.StalePendingCount != 2 {
		t.Errorf("project B: expected 2 stale pending, got %d", metricsB.StalePendingCount)
	}

	// Verify completion ratios
	expectedRatioA := 20.0 // 1 complete out of 5 total
	if metricsA.CompletionRatio != expectedRatioA {
		t.Errorf("project A: expected ratio %.2f, got %.2f", expectedRatioA, metricsA.CompletionRatio)
	}

	// Project B: 1 complete out of 3 total (rounded to ~33.33%)
	if metricsB.CompletionRatio < 33.0 || metricsB.CompletionRatio > 34.0 {
		t.Errorf("project B: expected ratio around 33.33, got %.2f", metricsB.CompletionRatio)
	}
}

func TestGenerateRecommendations(t *testing.T) {
	tests := []struct {
		name             string
		metricsMap       map[string]*ProjectMetrics
		projectPaths     []string
		expectedCount    int
		expectedContains []string
	}{
		{
			name: "healthy project - no recommendations",
			metricsMap: map[string]*ProjectMetrics{
				"/path/to/healthy": {
					Name:              "healthy",
					PendingCount:        2,
					InProgressCount:     1,
					CompletedCount:    10,
					HasCompletedBalls: true,
					CompletionRatio:   76.9,
					StalePendingCount:   0,
				},
			},
			projectPaths:     []string{"/path/to/healthy"},
			expectedCount:    0,
			expectedContains: []string{},
		},
		{
			name: "low completion rate",
			metricsMap: map[string]*ProjectMetrics{
				"/path/to/low": {
					Name:              "low",
					PendingCount:        10,
					InProgressCount:     5,
					CompletedCount:    3,
					HasCompletedBalls: true,
					CompletionRatio:   16.7,
					StalePendingCount:   0,
				},
			},
			projectPaths:     []string{"/path/to/low"},
			expectedCount:    1,
			expectedContains: []string{"Low completion rate"},
		},
		{
			name: "stale pending balls",
			metricsMap: map[string]*ProjectMetrics{
				"/path/to/stale": {
					Name:              "stale",
					PendingCount:        5,
					InProgressCount:     1,
					CompletedCount:    10,
					HasCompletedBalls: true,
					CompletionRatio:   62.5,
					StalePendingCount:   3,
				},
			},
			projectPaths:     []string{"/path/to/stale"},
			expectedCount:    1,
			expectedContains: []string{"3 stale pending balls"},
		},
		{
			name: "multiple issues",
			metricsMap: map[string]*ProjectMetrics{
				"/path/to/problem": {
					Name:              "problem",
					PendingCount:        15,
					InProgressCount:     10,
					BlockedCount:      8,
					CompletedCount:    2,
					HasCompletedBalls: true,
					CompletionRatio:   5.7,
					StalePendingCount:   5,
				},
			},
			projectPaths:     []string{"/path/to/problem"},
			expectedCount:    3, // low completion, stale balls, high blocked
			expectedContains: []string{"Low completion rate", "5 stale pending balls", "blocked balls"},
		},
		{
			name: "many pending without completions",
			metricsMap: map[string]*ProjectMetrics{
				"/path/to/nostart": {
					Name:              "nostart",
					PendingCount:        15,
					InProgressCount:     0,
					CompletedCount:    0,
					HasCompletedBalls: false,
					CompletionRatio:   0,
					StalePendingCount:   0,
				},
			},
			projectPaths:     []string{"/path/to/nostart"},
			expectedCount:    1,
			expectedContains: []string{"Many pending balls but none completed"},
		},
		{
			name: "many in-progress without completions",
			metricsMap: map[string]*ProjectMetrics{
				"/path/to/in-progress": {
					Name:              "in-progress",
					PendingCount:        2,
					InProgressCount:     8,
					CompletedCount:    0,
					HasCompletedBalls: false,
					CompletionRatio:   0,
					StalePendingCount:   0,
				},
			},
			projectPaths:     []string{"/path/to/in-progress"},
			expectedCount:    1,
			expectedContains: []string{"Many balls in progress"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recommendations := generateRecommendations(tt.metricsMap, tt.projectPaths)

			if len(recommendations) != tt.expectedCount {
				t.Errorf("expected %d recommendations, got %d", tt.expectedCount, len(recommendations))
			}

			// Check that expected strings are present
			for _, expected := range tt.expectedContains {
				found := false
				for _, rec := range recommendations {
					if containsString(rec.Message, expected) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected recommendation containing '%s', not found", expected)
				}
			}
		})
	}
}

func TestFormatCompletionRatio(t *testing.T) {
	tests := []struct {
		name            string
		metrics         *ProjectMetrics
		expectedContain string
	}{
		{
			name: "no completed balls",
			metrics: &ProjectMetrics{
				PendingCount:        5,
				HasCompletedBalls: false,
			},
			expectedContain: "no completed balls yet",
		},
		{
			name: "no balls at all",
			metrics: &ProjectMetrics{
				PendingCount:        0,
				InProgressCount:     0,
				BlockedCount:      0,
				CompletedCount:    0,
				HasCompletedBalls: false,
			},
			expectedContain: "no balls",
		},
		{
			name: "low completion with warning",
			metrics: &ProjectMetrics{
				PendingCount:        20,
				CompletedCount:    5,
				HasCompletedBalls: true,
				CompletionRatio:   20.0,
			},
			expectedContain: "20%",
		},
		{
			name: "healthy completion no warning",
			metrics: &ProjectMetrics{
				PendingCount:        5,
				CompletedCount:    20,
				HasCompletedBalls: true,
				CompletionRatio:   80.0,
			},
			expectedContain: "80%",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatCompletionRatio(tt.metrics)
			if !containsString(result, tt.expectedContain) {
				t.Errorf("expected result to contain '%s', got '%s'", tt.expectedContain, result)
			}
		})
	}
}

func TestStalePendingDetection(t *testing.T) {
	now := time.Now()
	recentTime := now.Add(-10 * 24 * time.Hour)  // 10 days ago - not stale
	staleTime := now.Add(-35 * 24 * time.Hour)   // 35 days ago - stale
	veryStaleTime := now.Add(-90 * 24 * time.Hour) // 90 days ago - very stale

	balls := []*session.Ball{
		createTestBall(t, "project", "/path/to/project", session.StatePending, recentTime),
		createTestBall(t, "project", "/path/to/project", session.StatePending, staleTime),
		createTestBall(t, "project", "/path/to/project", session.StatePending, veryStaleTime),
		createTestBall(t, "project", "/path/to/project", session.StateInProgress, staleTime), // Not counted as stale (not pending)
	}

	metricsMap := calculateProjectMetrics(balls)
	metrics := metricsMap["/path/to/project"]

	if metrics.PendingCount != 3 {
		t.Errorf("expected 3 pending balls, got %d", metrics.PendingCount)
	}

	if metrics.StalePendingCount != 2 {
		t.Errorf("expected 2 stale pending balls, got %d", metrics.StalePendingCount)
	}

	if len(metrics.StalePendingBalls) != 2 {
		t.Errorf("expected 2 stale pending balls in array, got %d", len(metrics.StalePendingBalls))
	}
}

// Helper functions

func createTestBall(t *testing.T, name, workingDir string, state session.BallState, startedAt time.Time) *session.Ball {
	t.Helper()
	return &session.Ball{
		ID:         name + "-1",
		WorkingDir: workingDir,
		Intent:     "Test ball",
		Priority:   session.PriorityMedium,
		State:      state,
		StartedAt:  startedAt,
	}
}

func containsString(s, substr string) bool {
	// Strip ANSI codes for comparison (simple approach)
	cleaned := s
	// Remove common lipgloss ANSI sequences
	for i := 0; i < len(cleaned); i++ {
		if cleaned[i] == '\x1b' {
			// Find the end of the ANSI sequence (typically ends with 'm')
			end := i
			for end < len(cleaned) && cleaned[end] != 'm' {
				end++
			}
			if end < len(cleaned) {
				cleaned = cleaned[:i] + cleaned[end+1:]
			}
		}
	}
	return containsSubstring(cleaned, substr)
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && hasSubstring(s, substr)
}

func hasSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
