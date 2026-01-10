package integration_test

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ohare93/juggle/internal/cli"
	"github.com/ohare93/juggle/internal/session"
)

// TestExportLocal verifies --local flag restricts export to current project
func TestExportLocal(t *testing.T) {
	// Create two test projects
	project1 := t.TempDir()
	project2 := t.TempDir()

	// Setup project 1
	store1, err := session.NewStoreWithConfig(project1, session.StoreConfig{JugglerDirName: ".juggler"})
	if err != nil {
		t.Fatalf("Failed to create store1: %v", err)
	}

	ball1 := &session.Session{
		ID:           "test1-1",
		WorkingDir:   project1,
		Intent:       "Ball in project 1",
		Priority:     session.PriorityMedium,
		State:        session.StateInProgress,
		StartedAt:    time.Now(),
		LastActivity: time.Now(),
	}
	if err := store1.Save(ball1); err != nil {
		t.Fatalf("Failed to save ball1: %v", err)
	}

	// Setup project 2
	store2, err := session.NewStoreWithConfig(project2, session.StoreConfig{JugglerDirName: ".juggler"})
	if err != nil {
		t.Fatalf("Failed to create store2: %v", err)
	}

	ball2 := &session.Session{
		ID:           "test2-1",
		WorkingDir:   project2,
		Intent:       "Ball in project 2",
		Priority:     session.PriorityHigh,
		State:        session.StateInProgress,
		StartedAt:    time.Now(),
		LastActivity: time.Now(),
	}
	if err := store2.Save(ball2); err != nil {
		t.Fatalf("Failed to save ball2: %v", err)
	}

	// Create a config with both projects in search paths
	config := &session.Config{
		SearchPaths: []string{project1, project2},
	}

	// Test with --local flag
	t.Run("ExportLocal", func(t *testing.T) {
		cli.GlobalOpts.LocalOnly = true
		cli.GlobalOpts.ProjectDir = project1

		projects, err := cli.DiscoverProjectsForCommand(config, store1)
		if err != nil {
			t.Fatalf("Failed to discover projects: %v", err)
		}

		if len(projects) != 1 {
			t.Errorf("Expected 1 project with --local, got %d", len(projects))
		}

		allBalls, err := session.LoadAllBalls(projects)
		if err != nil {
			t.Fatalf("Failed to load balls: %v", err)
		}

		// Should only get ball from project1
		if len(allBalls) != 1 {
			t.Errorf("Expected 1 ball with --local, got %d", len(allBalls))
		}
		if len(allBalls) > 0 && allBalls[0].ID != "test1-1" {
			t.Errorf("Expected ball 'test1-1', got '%s'", allBalls[0].ID)
		}
	})

	// Test without --local flag
	t.Run("ExportAllProjects", func(t *testing.T) {
		cli.GlobalOpts.LocalOnly = false

		projects, err := cli.DiscoverProjectsForCommand(config, store1)
		if err != nil {
			t.Fatalf("Failed to discover projects: %v", err)
		}

		if len(projects) != 2 {
			t.Errorf("Expected 2 projects without --local, got %d", len(projects))
		}

		allBalls, err := session.LoadAllBalls(projects)
		if err != nil {
			t.Fatalf("Failed to load balls: %v", err)
		}

		// Should get balls from both projects
		if len(allBalls) != 2 {
			t.Errorf("Expected 2 balls without --local, got %d", len(allBalls))
		}
	})
}

// TestExportBallIDs verifies --ball-ids filtering logic
func TestExportBallIDs(t *testing.T) {
	project := t.TempDir()

	store, err := session.NewStoreWithConfig(project, session.StoreConfig{JugglerDirName: ".juggler"})
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Create multiple balls using new state model
	balls := []*session.Session{
		{
			ID:           "project-1",
			WorkingDir:   project,
			Intent:       "Ball 1",
			Priority:     session.PriorityMedium,
			State:        session.StateInProgress,
			StartedAt:    time.Now(),
			LastActivity: time.Now(),
		},
		{
			ID:           "project-2",
			WorkingDir:   project,
			Intent:       "Ball 2",
			Priority:     session.PriorityHigh,
			State:        session.StatePending,
			StartedAt:    time.Now(),
			LastActivity: time.Now(),
		},
		{
			ID:           "project-3",
			WorkingDir:   project,
			Intent:       "Ball 3",
			Priority:     session.PriorityLow,
			State:        session.StateInProgress,
			StartedAt:    time.Now(),
			LastActivity: time.Now(),
		},
	}

	for _, ball := range balls {
		if err := store.Save(ball); err != nil {
			t.Fatalf("Failed to save ball %s: %v", ball.ID, err)
		}
	}

	// Test filtering by full ID
	t.Run("FilterByFullID", func(t *testing.T) {
		projects := []string{project}
		allBalls, err := session.LoadAllBalls(projects)
		if err != nil {
			t.Fatalf("Failed to load balls: %v", err)
		}

		// Simulate filterByBallIDs for "project-2"
		filtered, err := filterBallsByIDs(allBalls, "project-2")
		if err != nil {
			t.Fatalf("Failed to filter balls: %v", err)
		}

		if len(filtered) != 1 {
			t.Errorf("Expected 1 ball, got %d", len(filtered))
		}
		if len(filtered) > 0 && filtered[0].ID != "project-2" {
			t.Errorf("Expected ball 'project-2', got '%s'", filtered[0].ID)
		}
	})

	// Test filtering by short IDs
	t.Run("FilterByShortIDs", func(t *testing.T) {
		projects := []string{project}
		allBalls, err := session.LoadAllBalls(projects)
		if err != nil {
			t.Fatalf("Failed to load balls: %v", err)
		}

		// Simulate filterByBallIDs for "1,3"
		filtered, err := filterBallsByIDs(allBalls, "1,3")
		if err != nil {
			t.Fatalf("Failed to filter balls: %v", err)
		}

		if len(filtered) != 2 {
			t.Errorf("Expected 2 balls, got %d", len(filtered))
		}

		foundBall1, foundBall3 := false, false
		for _, ball := range filtered {
			if ball.ID == "project-1" {
				foundBall1 = true
			}
			if ball.ID == "project-3" {
				foundBall3 = true
			}
		}
		if !foundBall1 || !foundBall3 {
			t.Errorf("Expected balls 1 and 3, got %v", filtered)
		}
	})

	// Test invalid ball ID
	t.Run("InvalidBallID", func(t *testing.T) {
		projects := []string{project}
		allBalls, err := session.LoadAllBalls(projects)
		if err != nil {
			t.Fatalf("Failed to load balls: %v", err)
		}

		_, err = filterBallsByIDs(allBalls, "project-999")
		if err == nil {
			t.Errorf("Expected error for invalid ball ID, got none")
		}
	})
}

// Helper function to test ball ID filtering logic
func filterBallsByIDs(balls []*session.Session, ballIDsStr string) ([]*session.Session, error) {
	idStrs := strings.Split(ballIDsStr, ",")
	requestedIDs := make([]string, 0, len(idStrs))
	for _, id := range idStrs {
		id = strings.TrimSpace(id)
		if id != "" {
			requestedIDs = append(requestedIDs, id)
		}
	}

	ballsByID := make(map[string]*session.Session)
	ballsByShortID := make(map[string][]*session.Session)

	for _, ball := range balls {
		ballsByID[ball.ID] = ball

		parts := strings.Split(ball.ID, "-")
		if len(parts) > 0 {
			shortID := parts[len(parts)-1]
			ballsByShortID[shortID] = append(ballsByShortID[shortID], ball)
		}
	}

	filteredBalls := make([]*session.Session, 0)
	seenBalls := make(map[string]bool)

	for _, requestedID := range requestedIDs {
		if ball, exists := ballsByID[requestedID]; exists {
			if !seenBalls[ball.ID] {
				filteredBalls = append(filteredBalls, ball)
				seenBalls[ball.ID] = true
			}
			continue
		}

		if matches, exists := ballsByShortID[requestedID]; exists {
			if len(matches) == 1 {
				ball := matches[0]
				if !seenBalls[ball.ID] {
					filteredBalls = append(filteredBalls, ball)
					seenBalls[ball.ID] = true
				}
			} else if len(matches) > 1 {
				matchingIDs := make([]string, len(matches))
				for i, m := range matches {
					matchingIDs[i] = m.ID
				}
				return nil, &AmbiguousShortIDError{ShortID: requestedID, Matches: matchingIDs}
			}
			continue
		}

		return nil, &BallNotFoundError{BallID: requestedID}
	}

	return filteredBalls, nil
}

type AmbiguousShortIDError struct {
	ShortID string
	Matches []string
}

func (e *AmbiguousShortIDError) Error() string {
	return "ambiguous short ID '" + e.ShortID + "' matches multiple balls: " + strings.Join(e.Matches, ", ")
}

type BallNotFoundError struct {
	BallID string
}

func (e *BallNotFoundError) Error() string {
	return "ball ID not found: " + e.BallID
}

// TestExportFilterState verifies --filter-state filtering logic
func TestExportFilterState(t *testing.T) {
	project := t.TempDir()

	store, err := session.NewStoreWithConfig(project, session.StoreConfig{JugglerDirName: ".juggler"})
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Create balls in different states using new simplified state model
	balls := []*session.Session{
		{
			ID:           "project-1",
			WorkingDir:   project,
			Intent:       "Pending ball",
			Priority:     session.PriorityMedium,
			State:        session.StatePending,
			StartedAt:    time.Now(),
			LastActivity: time.Now(),
		},
		{
			ID:           "project-2",
			WorkingDir:   project,
			Intent:       "In progress ball 1",
			Priority:     session.PriorityHigh,
			State:        session.StateInProgress,
			StartedAt:    time.Now(),
			LastActivity: time.Now(),
		},
		{
			ID:           "project-3",
			WorkingDir:   project,
			Intent:       "In progress ball 2",
			Priority:     session.PriorityLow,
			State:        session.StateInProgress,
			StartedAt:    time.Now(),
			LastActivity: time.Now(),
		},
		{
			ID:            "project-4",
			WorkingDir:    project,
			Intent:        "Blocked ball",
			Priority:      session.PriorityMedium,
			State:         session.StateBlocked,
			BlockedReason: "waiting for input",
			StartedAt:     time.Now(),
			LastActivity:  time.Now(),
		},
	}

	for _, ball := range balls {
		if err := store.Save(ball); err != nil {
			t.Fatalf("Failed to save ball %s: %v", ball.ID, err)
		}
	}

	projects := []string{project}
	allBalls, err := session.LoadAllBalls(projects)
	if err != nil {
		t.Fatalf("Failed to load balls: %v", err)
	}

	// Test filtering by in_progress state
	t.Run("FilterByInProgress", func(t *testing.T) {
		filtered, err := filterBallsByState(allBalls, "in_progress")
		if err != nil {
			t.Fatalf("Failed to filter balls: %v", err)
		}

		if len(filtered) != 2 {
			t.Errorf("Expected 2 in_progress balls, got %d", len(filtered))
		}
	})

	// Test filtering by pending state
	t.Run("FilterByPending", func(t *testing.T) {
		filtered, err := filterBallsByState(allBalls, "pending")
		if err != nil {
			t.Fatalf("Failed to filter balls: %v", err)
		}

		if len(filtered) != 1 {
			t.Errorf("Expected 1 pending ball, got %d", len(filtered))
		}
		if len(filtered) > 0 && filtered[0].ID != "project-1" {
			t.Errorf("Expected ball 'project-1', got '%s'", filtered[0].ID)
		}
	})

	// Test filtering by multiple states
	t.Run("FilterByMultipleStates", func(t *testing.T) {
		filtered, err := filterBallsByState(allBalls, "pending,blocked")
		if err != nil {
			t.Fatalf("Failed to filter balls: %v", err)
		}

		if len(filtered) != 2 {
			t.Errorf("Expected 2 balls (pending+blocked), got %d", len(filtered))
		}
	})

	// Test invalid state
	t.Run("InvalidState", func(t *testing.T) {
		_, err := filterBallsByState(allBalls, "invalid-state")
		if err == nil {
			t.Errorf("Expected error for invalid state, got none")
		}
	})
}

// Helper function to test state filtering logic using new simplified state model
func filterBallsByState(balls []*session.Session, stateStr string) ([]*session.Session, error) {
	stateStrs := strings.Split(stateStr, ",")
	stateFilters := make([]session.BallState, 0, len(stateStrs))

	for _, s := range stateStrs {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}

		if !isValidState(s) {
			return nil, &InvalidStateError{State: s}
		}

		stateFilters = append(stateFilters, session.BallState(s))
	}

	if len(stateFilters) == 0 {
		return balls, nil
	}

	filteredBalls := make([]*session.Session, 0)
	for _, ball := range balls {
		for _, filter := range stateFilters {
			if ball.State == filter {
				filteredBalls = append(filteredBalls, ball)
				break
			}
		}
	}

	return filteredBalls, nil
}

func isValidState(state string) bool {
	return state == "pending" || state == "in_progress" || state == "blocked" || state == "complete"
}

type InvalidStateError struct {
	State string
}

func (e *InvalidStateError) Error() string {
	return "invalid state: " + e.State + " (must be pending, in_progress, blocked, or complete)"
}

// TestExportIncludeDone verifies --include-done filtering logic
func TestExportIncludeDone(t *testing.T) {
	project := t.TempDir()

	store, err := session.NewStoreWithConfig(project, session.StoreConfig{JugglerDirName: ".juggler"})
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Create active and completed balls using new state model
	completedTime := time.Now()
	balls := []*session.Session{
		{
			ID:           "project-1",
			WorkingDir:   project,
			Intent:       "Active ball",
			Priority:     session.PriorityMedium,
			State:        session.StateInProgress,
			StartedAt:    time.Now(),
			LastActivity: time.Now(),
		},
		{
			ID:             "project-2",
			WorkingDir:     project,
			Intent:         "Completed ball",
			Priority:       session.PriorityHigh,
			State:          session.StateComplete,
			StartedAt:      time.Now().Add(-1 * time.Hour),
			LastActivity:   time.Now().Add(-30 * time.Minute),
			CompletedAt:    &completedTime,
			CompletionNote: "All done",
		},
	}

	for _, ball := range balls {
		if err := store.Save(ball); err != nil {
			t.Fatalf("Failed to save ball: %v", err)
		}
	}

	projects := []string{project}
	allBalls, err := session.LoadAllBalls(projects)
	if err != nil {
		t.Fatalf("Failed to load balls: %v", err)
	}

	// Test without --include-done (exclude complete)
	t.Run("ExcludeCompleted", func(t *testing.T) {
		filtered := make([]*session.Session, 0)
		for _, ball := range allBalls {
			if ball.State != session.StateComplete {
				filtered = append(filtered, ball)
			}
		}

		if len(filtered) != 1 {
			t.Errorf("Expected 1 active ball, got %d", len(filtered))
		}
	})

	// Test with --include-done (include complete)
	t.Run("IncludeCompleted", func(t *testing.T) {
		if len(allBalls) != 2 {
			t.Errorf("Expected 2 balls (active + complete), got %d", len(allBalls))
		}
	})
}

// TestExportCSVFormat verifies CSV export format
func TestExportCSVFormat(t *testing.T) {
	project := t.TempDir()

	store, err := session.NewStoreWithConfig(project, session.StoreConfig{JugglerDirName: ".juggler"})
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Create a ball with todos using new state model
	ball := &session.Session{
		ID:           "project-1",
		WorkingDir:   project,
		Intent:       "Ball with todos",
		Priority:     session.PriorityHigh,
		State:        session.StateInProgress,
		StartedAt:    time.Now(),
		LastActivity: time.Now(),
		Tags:         []string{"backend", "api"},
		Todos: []session.Todo{
			{Text: "Todo 1", Done: true},
			{Text: "Todo 2", Done: false},
		},
	}

	if err := store.Save(ball); err != nil {
		t.Fatalf("Failed to save ball: %v", err)
	}

	projects := []string{project}
	allBalls, err := session.LoadAllBalls(projects)
	if err != nil {
		t.Fatalf("Failed to load balls: %v", err)
	}

	// Simulate CSV export
	outputFile := filepath.Join(t.TempDir(), "export.csv")
	csvData, err := exportToCSV(allBalls)
	if err != nil {
		t.Fatalf("Failed to export to CSV: %v", err)
	}

	if err := os.WriteFile(outputFile, csvData, 0644); err != nil {
		t.Fatalf("Failed to write CSV file: %v", err)
	}

	// Read and parse CSV
	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read CSV file: %v", err)
	}

	reader := csv.NewReader(strings.NewReader(string(data)))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to parse CSV: %v", err)
	}

	if len(records) != 2 {
		t.Fatalf("Expected 2 records (header + 1 ball), got %d", len(records))
	}

	// Verify header (updated for new state model)
	header := records[0]
	expectedColumns := []string{"ID", "Project", "Intent", "Priority", "State", "BlockedReason", "StartedAt", "CompletedAt", "LastActivity", "Tags", "TodosTotal", "TodosCompleted", "CompletionNote"}
	if len(header) != len(expectedColumns) {
		t.Errorf("Expected %d columns, got %d", len(expectedColumns), len(header))
	}

	// Verify data row
	row := records[1]
	if row[0] != "project-1" {
		t.Errorf("Expected ID 'project-1', got '%s'", row[0])
	}
	if row[2] != "Ball with todos" {
		t.Errorf("Expected intent 'Ball with todos', got '%s'", row[2])
	}
	if row[3] != "high" {
		t.Errorf("Expected priority 'high', got '%s'", row[3])
	}
	if row[4] != "in_progress" {
		t.Errorf("Expected state 'in_progress', got '%s'", row[4])
	}
	if row[9] != "backend;api" {
		t.Errorf("Expected tags 'backend;api', got '%s'", row[9])
	}
	if row[10] != "2" {
		t.Errorf("Expected TodosTotal '2', got '%s'", row[10])
	}
	if row[11] != "1" {
		t.Errorf("Expected TodosCompleted '1', got '%s'", row[11])
	}
}

// Helper function to export to CSV format using new state model
func exportToCSV(balls []*session.Session) ([]byte, error) {
	var buf strings.Builder
	writer := csv.NewWriter(&buf)

	// Write header
	header := []string{
		"ID",
		"Project",
		"Intent",
		"Priority",
		"State",
		"BlockedReason",
		"StartedAt",
		"CompletedAt",
		"LastActivity",
		"Tags",
		"TodosTotal",
		"TodosCompleted",
		"CompletionNote",
	}
	if err := writer.Write(header); err != nil {
		return nil, err
	}

	// Write rows
	for _, ball := range balls {
		completedAt := ""
		if ball.CompletedAt != nil {
			completedAt = ball.CompletedAt.Format("2006-01-02 15:04:05")
		}

		tags := strings.Join(ball.Tags, ";")

		total, completed := ball.TodoStats()

		row := []string{
			ball.ID,
			ball.WorkingDir,
			ball.Intent,
			string(ball.Priority),
			string(ball.State),
			ball.BlockedReason,
			ball.StartedAt.Format("2006-01-02 15:04:05"),
			completedAt,
			ball.LastActivity.Format("2006-01-02 15:04:05"),
			tags,
			string(rune('0' + total)),
			string(rune('0' + completed)),
			ball.CompletionNote,
		}

		if err := writer.Write(row); err != nil {
			return nil, err
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}

	return []byte(buf.String()), nil
}

// TestExportJSONFormat verifies JSON export format
func TestExportJSONFormat(t *testing.T) {
	project := t.TempDir()

	store, err := session.NewStoreWithConfig(project, session.StoreConfig{JugglerDirName: ".juggler"})
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	ball := &session.Session{
		ID:           "project-1",
		WorkingDir:   project,
		Intent:       "Test ball",
		Priority:     session.PriorityMedium,
		State:        session.StatePending,
		StartedAt:    time.Now(),
		LastActivity: time.Now(),
	}

	if err := store.Save(ball); err != nil {
		t.Fatalf("Failed to save ball: %v", err)
	}

	projects := []string{project}
	allBalls, err := session.LoadAllBalls(projects)
	if err != nil {
		t.Fatalf("Failed to load balls: %v", err)
	}

	// Simulate JSON export
	jsonData, err := exportToJSON(allBalls)
	if err != nil {
		t.Fatalf("Failed to export to JSON: %v", err)
	}

	// Verify JSON structure
	var result struct {
		TotalBalls int                `json:"total_balls"`
		Balls      []*session.Session `json:"balls"`
	}
	if err := json.Unmarshal(jsonData, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if result.TotalBalls != 1 {
		t.Errorf("Expected 1 ball in JSON, got %d", result.TotalBalls)
	}

	if len(result.Balls) != 1 {
		t.Fatalf("Expected 1 ball in array, got %d", len(result.Balls))
	}

	if result.Balls[0].ID != "project-1" {
		t.Errorf("Expected ball ID 'project-1', got '%s'", result.Balls[0].ID)
	}
}

// Helper function to export to JSON format
func exportToJSON(balls []*session.Session) ([]byte, error) {
	export := struct {
		ExportedAt string             `json:"exported_at"`
		TotalBalls int                `json:"total_balls"`
		Balls      []*session.Session `json:"balls"`
	}{
		ExportedAt: time.Now().Format(time.RFC3339),
		TotalBalls: len(balls),
		Balls:      balls,
	}

	data, err := json.MarshalIndent(export, "", "  ")
	if err != nil {
		return nil, err
	}

	return data, nil
}

// TestExportRalphFormat verifies Ralph agent export format
func TestExportRalphFormat(t *testing.T) {
	project := t.TempDir()

	// Create session store and a session
	sessionStore, err := session.NewSessionStore(project)
	if err != nil {
		t.Fatalf("Failed to create session store: %v", err)
	}

	// Create a session with context
	sess, err := sessionStore.CreateSession("test-feature", "Implement test feature")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Update context
	if err := sessionStore.UpdateSessionContext("test-feature", "This is the context for the test feature.\nIt includes important information."); err != nil {
		t.Fatalf("Failed to update session context: %v", err)
	}

	// Append progress
	if err := sessionStore.AppendProgress("test-feature", "Started work on the feature.\nCompleted initial setup.\n"); err != nil {
		t.Fatalf("Failed to append progress: %v", err)
	}

	// Create ball store and balls with the session tag
	ballStore, err := session.NewStoreWithConfig(project, session.StoreConfig{JugglerDirName: ".juggler"})
	if err != nil {
		t.Fatalf("Failed to create ball store: %v", err)
	}

	balls := []*session.Session{
		{
			ID:           "project-1",
			WorkingDir:   project,
			Intent:       "Implement feature A",
			Description:  "The first part of the feature",
			Priority:     session.PriorityHigh,
			State:        session.StateInProgress,
			StartedAt:    time.Now(),
			LastActivity: time.Now(),
			Tags:         []string{"test-feature", "backend"},
			Todos: []session.Todo{
				{Text: "Design API", Done: true},
				{Text: "Implement logic", Done: false},
			},
		},
		{
			ID:            "project-2",
			WorkingDir:    project,
			Intent:        "Implement feature B",
			Priority:      session.PriorityMedium,
			State:         session.StateBlocked,
			BlockedReason: "waiting for API spec",
			StartedAt:     time.Now(),
			LastActivity:  time.Now(),
			Tags:          []string{"test-feature"},
		},
	}

	for _, ball := range balls {
		if err := ballStore.Save(ball); err != nil {
			t.Fatalf("Failed to save ball %s: %v", ball.ID, err)
		}
	}

	// Export to Ralph format
	output, err := exportToRalph(project, "test-feature", balls)
	if err != nil {
		t.Fatalf("Failed to export to Ralph format: %v", err)
	}

	outputStr := string(output)

	// Verify <context> section
	if !strings.Contains(outputStr, "<context>") {
		t.Error("Missing <context> tag")
	}
	if !strings.Contains(outputStr, "</context>") {
		t.Error("Missing </context> tag")
	}
	if !strings.Contains(outputStr, "# Implement test feature") {
		t.Error("Missing session description in context")
	}
	if !strings.Contains(outputStr, "This is the context for the test feature") {
		t.Error("Missing session context content")
	}

	// Verify <progress> section
	if !strings.Contains(outputStr, "<progress>") {
		t.Error("Missing <progress> tag")
	}
	if !strings.Contains(outputStr, "</progress>") {
		t.Error("Missing </progress> tag")
	}
	if !strings.Contains(outputStr, "Started work on the feature") {
		t.Error("Missing progress content")
	}

	// Verify <tasks> section
	if !strings.Contains(outputStr, "<tasks>") {
		t.Error("Missing <tasks> tag")
	}
	if !strings.Contains(outputStr, "</tasks>") {
		t.Error("Missing </tasks> tag")
	}

	// Verify task content
	if !strings.Contains(outputStr, "## project-1 [in_progress] (priority: high)") {
		t.Error("Missing task header for project-1")
	}
	if !strings.Contains(outputStr, "Intent: Implement feature A") {
		t.Error("Missing intent for project-1")
	}
	if !strings.Contains(outputStr, "[x] Design API") {
		t.Error("Missing completed todo")
	}
	if !strings.Contains(outputStr, "[ ] Implement logic") {
		t.Error("Missing incomplete todo")
	}

	// Verify blocked ball
	if !strings.Contains(outputStr, "## project-2 [blocked] (priority: medium)") {
		t.Error("Missing task header for project-2")
	}
	if !strings.Contains(outputStr, "Blocked: waiting for API spec") {
		t.Error("Missing blocked reason")
	}

	_ = sess // Use the session variable
}

// Helper function to export to Ralph format
func exportToRalph(projectDir, sessionID string, balls []*session.Session) ([]byte, error) {
	var buf strings.Builder

	// Load session store to get context and progress
	sessionStore, err := session.NewSessionStore(projectDir)
	if err != nil {
		return nil, err
	}

	// Try to load the session
	juggleSession, err := sessionStore.LoadSession(sessionID)
	if err != nil {
		juggleSession = &session.JuggleSession{
			ID:          sessionID,
			Description: "",
			Context:     "",
		}
	}

	// Load progress
	progress, _ := sessionStore.LoadProgress(sessionID)

	// Write <context> section
	buf.WriteString("<context>\n")
	if juggleSession.Description != "" {
		buf.WriteString("# " + juggleSession.Description + "\n\n")
	}
	if juggleSession.Context != "" {
		buf.WriteString(juggleSession.Context)
		if !strings.HasSuffix(juggleSession.Context, "\n") {
			buf.WriteString("\n")
		}
	}
	buf.WriteString("</context>\n\n")

	// Write <progress> section
	buf.WriteString("<progress>\n")
	if progress != "" {
		buf.WriteString(progress)
		if !strings.HasSuffix(progress, "\n") {
			buf.WriteString("\n")
		}
	}
	buf.WriteString("</progress>\n\n")

	// Write <tasks> section
	buf.WriteString("<tasks>\n")
	for i, ball := range balls {
		if i > 0 {
			buf.WriteString("\n")
		}
		writeBallForRalphTest(&buf, ball)
	}
	buf.WriteString("</tasks>\n")

	return []byte(buf.String()), nil
}

// writeBallForRalphTest writes a single ball in Ralph format (test helper)
func writeBallForRalphTest(buf *strings.Builder, ball *session.Session) {
	buf.WriteString("## " + ball.ID + " [" + string(ball.State) + "] (priority: " + string(ball.Priority) + ")\n")
	buf.WriteString("Intent: " + ball.Intent + "\n")
	if ball.Description != "" {
		buf.WriteString("Description: " + ball.Description + "\n")
	}
	if ball.State == session.StateBlocked && ball.BlockedReason != "" {
		buf.WriteString("Blocked: " + ball.BlockedReason + "\n")
	}
	if len(ball.Todos) > 0 {
		buf.WriteString("Todos:\n")
		for i, todo := range ball.Todos {
			checkbox := "[ ]"
			if todo.Done {
				checkbox = "[x]"
			}
			buf.WriteString("  " + fmt.Sprintf("%d", i+1) + ". " + checkbox + " " + todo.Text + "\n")
			if todo.Description != "" {
				buf.WriteString("     " + todo.Description + "\n")
			}
		}
	}
	if len(ball.Tags) > 0 {
		buf.WriteString("Tags: " + strings.Join(ball.Tags, ", ") + "\n")
	}
}
