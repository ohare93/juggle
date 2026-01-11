package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/ohare93/juggle/internal/session"
	"github.com/spf13/cobra"
)

var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Analyze project health and completion metrics",
	Long: `Audit provides insights into your juggling workflow:

- Completion ratios per project
- Stale pending balls (>30 days old)
- State distribution (pending/in_progress/blocked/completed)
- Actionable recommendations for improving workflow

Use this to identify:
- Projects with low completion rates
- Balls that have been pending but never started
- Patterns in blocked work`,
	RunE: runAudit,
}

// ProjectMetrics holds calculated metrics for a project
type ProjectMetrics struct {
	Path               string
	Name               string
	PendingCount       int
	InProgressCount    int
	BlockedCount       int
	CompletedCount     int
	CompletionRatio    float64
	StalePendingCount  int
	StalePendingBalls  []*session.Ball
	HasCompletedBalls  bool
}

const staleDays = 30

func runAudit(cmd *cobra.Command, args []string) error {
	// Get current directory
	cwd, err := GetWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// If current directory has .juggler, ensure it's tracked
	if _, err := os.Stat(filepath.Join(cwd, ".juggler")); err == nil {
		_ = session.EnsureProjectInSearchPaths(cwd)
	}

	// Load config to discover projects
	config, err := LoadConfigForCommand()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	store, err := NewStoreForCommand(cwd)
	if err != nil {
		return fmt.Errorf("failed to create store: %w", err)
	}

	// Discover projects (respects --all flag)
	projects, err := DiscoverProjectsForCommand(config, store)
	if err != nil {
		return fmt.Errorf("failed to discover projects: %w", err)
	}

	if len(projects) == 0 {
		fmt.Println("No projects with .juggler directories found.")
		fmt.Println("\nStart tracking work with: juggle start")
		return nil
	}

	// Load both active and archived balls from all projects
	activeBalls, err := session.LoadAllBalls(projects)
	if err != nil {
		return fmt.Errorf("failed to load active balls: %w", err)
	}

	archivedBalls, err := session.LoadArchivedBalls(projects)
	if err != nil {
		return fmt.Errorf("failed to load archived balls: %w", err)
	}

	// Combine active and archived balls for complete metrics
	allBalls := append(activeBalls, archivedBalls...)

	if len(allBalls) == 0 {
		fmt.Println("No balls found across all projects.")
		fmt.Println("\nStart tracking work with: juggle start")
		return nil
	}

	// Calculate metrics per project
	metricsMap := calculateProjectMetrics(allBalls)

	// Sort projects by path for consistent output
	projectPaths := make([]string, 0, len(metricsMap))
	for path := range metricsMap {
		projectPaths = append(projectPaths, path)
	}
	sort.Strings(projectPaths)

	// Render the audit report
	renderAuditReport(metricsMap, projectPaths)

	return nil
}

// calculateProjectMetrics computes all metrics for each project
func calculateProjectMetrics(balls []*session.Ball) map[string]*ProjectMetrics {
	metricsMap := make(map[string]*ProjectMetrics)
	staleThreshold := time.Now().Add(-staleDays * 24 * time.Hour)

	for _, ball := range balls {
		// Initialize project metrics if not exists
		if _, exists := metricsMap[ball.WorkingDir]; !exists {
			metricsMap[ball.WorkingDir] = &ProjectMetrics{
				Path:              ball.WorkingDir,
				Name:              ball.FolderName(),
				StalePendingBalls: make([]*session.Ball, 0),
			}
		}

		metrics := metricsMap[ball.WorkingDir]

		// Count by state
		switch ball.State {
		case session.StatePending:
			metrics.PendingCount++
			// Check if stale
			if ball.StartedAt.Before(staleThreshold) {
				metrics.StalePendingCount++
				metrics.StalePendingBalls = append(metrics.StalePendingBalls, ball)
			}
		case session.StateInProgress:
			metrics.InProgressCount++
		case session.StateBlocked:
			metrics.BlockedCount++
		case session.StateComplete:
			metrics.CompletedCount++
			metrics.HasCompletedBalls = true
		}
	}

	// Calculate completion ratios
	for _, metrics := range metricsMap {
		metrics.CompletionRatio = calculateCompletionRatio(metrics)
	}

	return metricsMap
}

// calculateCompletionRatio computes the completion percentage
// Formula: completed / (total_non_complete + completed) * 100
func calculateCompletionRatio(metrics *ProjectMetrics) float64 {
	totalNonComplete := metrics.PendingCount + metrics.InProgressCount + metrics.BlockedCount
	totalBalls := totalNonComplete + metrics.CompletedCount

	if totalBalls == 0 {
		return 0
	}

	return float64(metrics.CompletedCount) / float64(totalBalls) * 100
}

// renderAuditReport displays the audit results with styling
func renderAuditReport(metricsMap map[string]*ProjectMetrics, projectPaths []string) {
	// Header
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("12")). // Blue
		MarginBottom(1)

	fmt.Println(headerStyle.Render("🎯 Juggler Audit Report"))
	fmt.Println(headerStyle.Render("======================="))

	// Project details
	for _, path := range projectPaths {
		metrics := metricsMap[path]
		renderProjectMetrics(metrics)
	}

	// Overall recommendations
	renderRecommendations(metricsMap, projectPaths)
}

// renderProjectMetrics displays metrics for a single project
func renderProjectMetrics(metrics *ProjectMetrics) {
	projectStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("14")). // Cyan
		MarginTop(1)

	fmt.Println(projectStyle.Render(metrics.Path + ":"))

	// State counts
	fmt.Printf("  Pending: %d\n", metrics.PendingCount)
	fmt.Printf("  In Progress: %d\n", metrics.InProgressCount)
	fmt.Printf("  Blocked: %d\n", metrics.BlockedCount)
	fmt.Printf("  Completed: %d\n", metrics.CompletedCount)

	// Completion ratio with warning
	completionStr := formatCompletionRatio(metrics)
	fmt.Printf("  Completion ratio: %s\n", completionStr)

	// Stale pending balls
	if metrics.StalePendingCount > 0 {
		warningStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("11")) // Yellow
		staleMsg := fmt.Sprintf("%d (>%d days old)", metrics.StalePendingCount, staleDays)
		fmt.Printf("  Stale pending balls: %s\n", warningStyle.Render(staleMsg))
	}
}

// formatCompletionRatio formats the completion ratio with appropriate styling
func formatCompletionRatio(metrics *ProjectMetrics) string {
	totalBalls := metrics.PendingCount + metrics.InProgressCount + metrics.BlockedCount + metrics.CompletedCount
	if totalBalls == 0 {
		dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8")) // Gray
		return dimStyle.Render("N/A (no balls)")
	}

	if !metrics.HasCompletedBalls {
		dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8")) // Gray
		return dimStyle.Render("N/A (no completed balls yet)")
	}

	ratio := metrics.CompletionRatio
	ratioStr := fmt.Sprintf("%.0f%%", ratio)

	// Flag low completion ratios
	if ratio < 40 {
		warningStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("11")) // Yellow
		return ratioStr + " " + warningStyle.Render("⚠️  (low)")
	}

	return ratioStr
}

// renderRecommendations provides actionable insights
func renderRecommendations(metricsMap map[string]*ProjectMetrics, projectPaths []string) {
	recommendations := generateRecommendations(metricsMap, projectPaths)

	if len(recommendations) == 0 {
		successStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")). // Green
			MarginTop(1)
		fmt.Println()
		fmt.Println(successStyle.Render("✓ No issues detected - workflow looks healthy!"))
		return
	}

	// Recommendations header
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("11")). // Yellow
		MarginTop(1)

	fmt.Println()
	fmt.Println(headerStyle.Render("Recommendations:"))

	for _, rec := range recommendations {
		fmt.Printf("• %s: %s\n", rec.ProjectName, rec.Message)
	}
}

// Recommendation holds a single recommendation
type Recommendation struct {
	ProjectName string
	Message     string
	Priority    int // Higher = more important
}

// generateRecommendations creates actionable recommendations based on metrics
func generateRecommendations(metricsMap map[string]*ProjectMetrics, projectPaths []string) []Recommendation {
	recommendations := make([]Recommendation, 0)

	for _, path := range projectPaths {
		metrics := metricsMap[path]

		// Low completion rate
		if metrics.HasCompletedBalls && metrics.CompletionRatio < 40 {
			recommendations = append(recommendations, Recommendation{
				ProjectName: metrics.Name,
				Message:     "Low completion rate - focus on finishing started work",
				Priority:    3,
			})
		}

		// Stale pending balls
		if metrics.StalePendingCount > 0 {
			msg := fmt.Sprintf("%d stale pending balls - block or start them", metrics.StalePendingCount)
			recommendations = append(recommendations, Recommendation{
				ProjectName: metrics.Name,
				Message:     msg,
				Priority:    2,
			})
		}

		// High pending count without completions
		if metrics.PendingCount > 10 && !metrics.HasCompletedBalls {
			recommendations = append(recommendations, Recommendation{
				ProjectName: metrics.Name,
				Message:     "Many pending balls but none completed - start working through them",
				Priority:    2,
			})
		}

		// High blocked count
		if metrics.BlockedCount > 5 {
			recommendations = append(recommendations, Recommendation{
				ProjectName: metrics.Name,
				Message:     fmt.Sprintf("%d blocked balls - review why work is blocked", metrics.BlockedCount),
				Priority:    1,
			})
		}

		// Many in progress but none completed
		if metrics.InProgressCount > 5 && metrics.CompletedCount == 0 {
			recommendations = append(recommendations, Recommendation{
				ProjectName: metrics.Name,
				Message:     "Many balls in progress - consider completing some before starting more",
				Priority:    2,
			})
		}
	}

	// Sort by priority (highest first), then alphabetically
	sort.Slice(recommendations, func(i, j int) bool {
		if recommendations[i].Priority != recommendations[j].Priority {
			return recommendations[i].Priority > recommendations[j].Priority
		}
		return strings.Compare(recommendations[i].ProjectName, recommendations[j].ProjectName) < 0
	})

	return recommendations
}
