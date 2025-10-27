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
				ReadyCount:     5,
				JugglingCount:  3,
				DroppedCount:   2,
				CompletedCount: 10,
			},
			expected: 50.0,
		},
		{
			name: "100% completion",
			metrics: &ProjectMetrics{
				ReadyCount:     0,
				JugglingCount:  0,
				DroppedCount:   0,
				CompletedCount: 10,
			},
			expected: 100.0,
		},
		{
			name: "0% completion",
			metrics: &ProjectMetrics{
				ReadyCount:     5,
				JugglingCount:  3,
				DroppedCount:   2,
				CompletedCount: 0,
			},
			expected: 0.0,
		},
		{
			name: "no balls",
			metrics: &ProjectMetrics{
				ReadyCount:     0,
				JugglingCount:  0,
				DroppedCount:   0,
				CompletedCount: 0,
			},
			expected: 0.0,
		},
		{
			name: "low completion 20%",
			metrics: &ProjectMetrics{
				ReadyCount:     20,
				JugglingCount:  10,
				DroppedCount:   10,
				CompletedCount: 10,
			},
			expected: 20.0,
		},
		{
			name: "high completion 80%",
			metrics: &ProjectMetrics{
				ReadyCount:     2,
				JugglingCount:  1,
				DroppedCount:   2,
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

	balls := []*session.Session{
		// Project A
		createTestBall(t, "project-a", "/path/to/a", session.ActiveReady, recentTime),
		createTestBall(t, "project-a", "/path/to/a", session.ActiveReady, staleTime),
		createTestBall(t, "project-a", "/path/to/a", session.ActiveJuggling, now),
		createTestBall(t, "project-a", "/path/to/a", session.ActiveComplete, now),
		createTestBall(t, "project-a", "/path/to/a", session.ActiveDropped, now),
		// Project B
		createTestBall(t, "project-b", "/path/to/b", session.ActiveReady, staleTime),
		createTestBall(t, "project-b", "/path/to/b", session.ActiveReady, staleTime),
		createTestBall(t, "project-b", "/path/to/b", session.ActiveComplete, now),
	}

	metricsMap := calculateProjectMetrics(balls)

	// Verify Project A
	metricsA := metricsMap["/path/to/a"]
	if metricsA == nil {
		t.Fatal("expected metrics for project A")
	}
	if metricsA.ReadyCount != 2 {
		t.Errorf("project A: expected 2 ready, got %d", metricsA.ReadyCount)
	}
	if metricsA.JugglingCount != 1 {
		t.Errorf("project A: expected 1 juggling, got %d", metricsA.JugglingCount)
	}
	if metricsA.CompletedCount != 1 {
		t.Errorf("project A: expected 1 completed, got %d", metricsA.CompletedCount)
	}
	if metricsA.DroppedCount != 1 {
		t.Errorf("project A: expected 1 dropped, got %d", metricsA.DroppedCount)
	}
	if metricsA.StaleReadyCount != 1 {
		t.Errorf("project A: expected 1 stale ready, got %d", metricsA.StaleReadyCount)
	}
	if !metricsA.HasCompletedBalls {
		t.Error("project A: expected HasCompletedBalls to be true")
	}

	// Verify Project B
	metricsB := metricsMap["/path/to/b"]
	if metricsB == nil {
		t.Fatal("expected metrics for project B")
	}
	if metricsB.ReadyCount != 2 {
		t.Errorf("project B: expected 2 ready, got %d", metricsB.ReadyCount)
	}
	if metricsB.CompletedCount != 1 {
		t.Errorf("project B: expected 1 completed, got %d", metricsB.CompletedCount)
	}
	if metricsB.StaleReadyCount != 2 {
		t.Errorf("project B: expected 2 stale ready, got %d", metricsB.StaleReadyCount)
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
					ReadyCount:        2,
					JugglingCount:     1,
					CompletedCount:    10,
					HasCompletedBalls: true,
					CompletionRatio:   76.9,
					StaleReadyCount:   0,
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
					ReadyCount:        10,
					JugglingCount:     5,
					CompletedCount:    3,
					HasCompletedBalls: true,
					CompletionRatio:   16.7,
					StaleReadyCount:   0,
				},
			},
			projectPaths:     []string{"/path/to/low"},
			expectedCount:    1,
			expectedContains: []string{"Low completion rate"},
		},
		{
			name: "stale ready balls",
			metricsMap: map[string]*ProjectMetrics{
				"/path/to/stale": {
					Name:              "stale",
					ReadyCount:        5,
					JugglingCount:     1,
					CompletedCount:    10,
					HasCompletedBalls: true,
					CompletionRatio:   62.5,
					StaleReadyCount:   3,
				},
			},
			projectPaths:     []string{"/path/to/stale"},
			expectedCount:    1,
			expectedContains: []string{"3 stale ready balls"},
		},
		{
			name: "multiple issues",
			metricsMap: map[string]*ProjectMetrics{
				"/path/to/problem": {
					Name:              "problem",
					ReadyCount:        15,
					JugglingCount:     10,
					DroppedCount:      8,
					CompletedCount:    2,
					HasCompletedBalls: true,
					CompletionRatio:   5.7,
					StaleReadyCount:   5,
				},
			},
			projectPaths:     []string{"/path/to/problem"},
			expectedCount:    3, // low completion, stale balls, high dropped
			expectedContains: []string{"Low completion rate", "5 stale ready balls", "dropped balls"},
		},
		{
			name: "many ready without completions",
			metricsMap: map[string]*ProjectMetrics{
				"/path/to/nostart": {
					Name:              "nostart",
					ReadyCount:        15,
					JugglingCount:     0,
					CompletedCount:    0,
					HasCompletedBalls: false,
					CompletionRatio:   0,
					StaleReadyCount:   0,
				},
			},
			projectPaths:     []string{"/path/to/nostart"},
			expectedCount:    1,
			expectedContains: []string{"Many ready balls but none completed"},
		},
		{
			name: "many juggling without completions",
			metricsMap: map[string]*ProjectMetrics{
				"/path/to/juggling": {
					Name:              "juggling",
					ReadyCount:        2,
					JugglingCount:     8,
					CompletedCount:    0,
					HasCompletedBalls: false,
					CompletionRatio:   0,
					StaleReadyCount:   0,
				},
			},
			projectPaths:     []string{"/path/to/juggling"},
			expectedCount:    1,
			expectedContains: []string{"Many balls juggling"},
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
				ReadyCount:        5,
				HasCompletedBalls: false,
			},
			expectedContain: "no completed balls yet",
		},
		{
			name: "no balls at all",
			metrics: &ProjectMetrics{
				ReadyCount:        0,
				JugglingCount:     0,
				DroppedCount:      0,
				CompletedCount:    0,
				HasCompletedBalls: false,
			},
			expectedContain: "no balls",
		},
		{
			name: "low completion with warning",
			metrics: &ProjectMetrics{
				ReadyCount:        20,
				CompletedCount:    5,
				HasCompletedBalls: true,
				CompletionRatio:   20.0,
			},
			expectedContain: "20%",
		},
		{
			name: "healthy completion no warning",
			metrics: &ProjectMetrics{
				ReadyCount:        5,
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

func TestStaleReadyDetection(t *testing.T) {
	now := time.Now()
	recentTime := now.Add(-10 * 24 * time.Hour)  // 10 days ago - not stale
	staleTime := now.Add(-35 * 24 * time.Hour)   // 35 days ago - stale
	veryStaleTime := now.Add(-90 * 24 * time.Hour) // 90 days ago - very stale

	balls := []*session.Session{
		createTestBall(t, "project", "/path/to/project", session.ActiveReady, recentTime),
		createTestBall(t, "project", "/path/to/project", session.ActiveReady, staleTime),
		createTestBall(t, "project", "/path/to/project", session.ActiveReady, veryStaleTime),
		createTestBall(t, "project", "/path/to/project", session.ActiveJuggling, staleTime), // Not counted as stale (not ready)
	}

	metricsMap := calculateProjectMetrics(balls)
	metrics := metricsMap["/path/to/project"]

	if metrics.ReadyCount != 3 {
		t.Errorf("expected 3 ready balls, got %d", metrics.ReadyCount)
	}

	if metrics.StaleReadyCount != 2 {
		t.Errorf("expected 2 stale ready balls, got %d", metrics.StaleReadyCount)
	}

	if len(metrics.StaleReadyBalls) != 2 {
		t.Errorf("expected 2 stale ready balls in array, got %d", len(metrics.StaleReadyBalls))
	}
}

// Helper functions

func createTestBall(t *testing.T, name, workingDir string, state session.ActiveState, startedAt time.Time) *session.Session {
	t.Helper()
	return &session.Session{
		ID:          name + "-1",
		WorkingDir:  workingDir,
		Intent:      "Test ball",
		Priority:    session.PriorityMedium,
		ActiveState: state,
		StartedAt:   startedAt,
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
