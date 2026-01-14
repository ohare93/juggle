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
	store1, err := session.NewStoreWithConfig(project1, session.StoreConfig{JuggleDirName: ".juggle"})
	if err != nil {
		t.Fatalf("Failed to create store1: %v", err)
	}

	ball1 := &session.Ball{
		ID:           "test1-1",
		WorkingDir:   project1,
		Title:       "Ball in project 1",
		Priority:     session.PriorityMedium,
		State:        session.StateInProgress,
		StartedAt:    time.Now(),
		LastActivity: time.Now(),
	}
	if err := store1.Save(ball1); err != nil {
		t.Fatalf("Failed to save ball1: %v", err)
	}

	// Setup project 2
	store2, err := session.NewStoreWithConfig(project2, session.StoreConfig{JuggleDirName: ".juggle"})
	if err != nil {
		t.Fatalf("Failed to create store2: %v", err)
	}

	ball2 := &session.Ball{
		ID:           "test2-1",
		WorkingDir:   project2,
		Title:       "Ball in project 2",
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

	// Test default behavior (local only, no --all flag)
	t.Run("ExportLocal", func(t *testing.T) {
		cli.GlobalOpts.AllProjects = false
		cli.GlobalOpts.ProjectDir = project1

		projects, err := cli.DiscoverProjectsForCommand(config, store1)
		if err != nil {
			t.Fatalf("Failed to discover projects: %v", err)
		}

		if len(projects) != 1 {
			t.Errorf("Expected 1 project by default (local), got %d", len(projects))
		}

		allBalls, err := session.LoadAllBalls(projects)
		if err != nil {
			t.Fatalf("Failed to load balls: %v", err)
		}

		// Should only get ball from project1
		if len(allBalls) != 1 {
			t.Errorf("Expected 1 ball by default (local), got %d", len(allBalls))
		}
		if len(allBalls) > 0 && allBalls[0].ID != "test1-1" {
			t.Errorf("Expected ball 'test1-1', got '%s'", allBalls[0].ID)
		}
	})

	// Test with --all flag for cross-project discovery
	t.Run("ExportAllProjects", func(t *testing.T) {
		cli.GlobalOpts.AllProjects = true
		defer func() { cli.GlobalOpts.AllProjects = false }()

		projects, err := cli.DiscoverProjectsForCommand(config, store1)
		if err != nil {
			t.Fatalf("Failed to discover projects: %v", err)
		}

		if len(projects) != 2 {
			t.Errorf("Expected 2 projects with --all, got %d", len(projects))
		}

		allBalls, err := session.LoadAllBalls(projects)
		if err != nil {
			t.Fatalf("Failed to load balls: %v", err)
		}

		// Should get balls from both projects
		if len(allBalls) != 2 {
			t.Errorf("Expected 2 balls with --all, got %d", len(allBalls))
		}
	})
}

// TestExportBallIDs verifies --ball-ids filtering logic
func TestExportBallIDs(t *testing.T) {
	project := t.TempDir()

	store, err := session.NewStoreWithConfig(project, session.StoreConfig{JuggleDirName: ".juggle"})
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Create multiple balls using new state model
	balls := []*session.Ball{
		{
			ID:           "project-1",
			WorkingDir:   project,
			Title:       "Ball 1",
			Priority:     session.PriorityMedium,
			State:        session.StateInProgress,
			StartedAt:    time.Now(),
			LastActivity: time.Now(),
		},
		{
			ID:           "project-2",
			WorkingDir:   project,
			Title:       "Ball 2",
			Priority:     session.PriorityHigh,
			State:        session.StatePending,
			StartedAt:    time.Now(),
			LastActivity: time.Now(),
		},
		{
			ID:           "project-3",
			WorkingDir:   project,
			Title:       "Ball 3",
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
func filterBallsByIDs(balls []*session.Ball, ballIDsStr string) ([]*session.Ball, error) {
	idStrs := strings.Split(ballIDsStr, ",")
	requestedIDs := make([]string, 0, len(idStrs))
	for _, id := range idStrs {
		id = strings.TrimSpace(id)
		if id != "" {
			requestedIDs = append(requestedIDs, id)
		}
	}

	ballsByID := make(map[string]*session.Ball)
	ballsByShortID := make(map[string][]*session.Ball)

	for _, ball := range balls {
		ballsByID[ball.ID] = ball

		parts := strings.Split(ball.ID, "-")
		if len(parts) > 0 {
			shortID := parts[len(parts)-1]
			ballsByShortID[shortID] = append(ballsByShortID[shortID], ball)
		}
	}

	filteredBalls := make([]*session.Ball, 0)
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

	store, err := session.NewStoreWithConfig(project, session.StoreConfig{JuggleDirName: ".juggle"})
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Create balls in different states using new simplified state model
	balls := []*session.Ball{
		{
			ID:           "project-1",
			WorkingDir:   project,
			Title:       "Pending ball",
			Priority:     session.PriorityMedium,
			State:        session.StatePending,
			StartedAt:    time.Now(),
			LastActivity: time.Now(),
		},
		{
			ID:           "project-2",
			WorkingDir:   project,
			Title:       "In progress ball 1",
			Priority:     session.PriorityHigh,
			State:        session.StateInProgress,
			StartedAt:    time.Now(),
			LastActivity: time.Now(),
		},
		{
			ID:           "project-3",
			WorkingDir:   project,
			Title:       "In progress ball 2",
			Priority:     session.PriorityLow,
			State:        session.StateInProgress,
			StartedAt:    time.Now(),
			LastActivity: time.Now(),
		},
		{
			ID:            "project-4",
			WorkingDir:    project,
			Title:        "Blocked ball",
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
func filterBallsByState(balls []*session.Ball, stateStr string) ([]*session.Ball, error) {
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

	filteredBalls := make([]*session.Ball, 0)
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

	store, err := session.NewStoreWithConfig(project, session.StoreConfig{JuggleDirName: ".juggle"})
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Create active and completed balls using new state model
	completedTime := time.Now()
	balls := []*session.Ball{
		{
			ID:           "project-1",
			WorkingDir:   project,
			Title:       "Active ball",
			Priority:     session.PriorityMedium,
			State:        session.StateInProgress,
			StartedAt:    time.Now(),
			LastActivity: time.Now(),
		},
		{
			ID:             "project-2",
			WorkingDir:     project,
			Title:         "Completed ball",
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
		filtered := make([]*session.Ball, 0)
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

	store, err := session.NewStoreWithConfig(project, session.StoreConfig{JuggleDirName: ".juggle"})
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Create a ball with acceptance criteria using new state model
	ball := &session.Ball{
		ID:                 "project-1",
		WorkingDir:         project,
		Title:             "Ball with acceptance criteria",
		Priority:           session.PriorityHigh,
		State:              session.StateInProgress,
		StartedAt:          time.Now(),
		LastActivity:       time.Now(),
		Tags:               []string{"backend", "api"},
		AcceptanceCriteria: []string{"Criterion 1", "Criterion 2"},
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
	expectedColumns := []string{"ID", "Project", "Intent", "Priority", "State", "BlockedReason", "StartedAt", "CompletedAt", "LastActivity", "Tags", "AcceptanceCriteria", "CompletionNote"}
	if len(header) != len(expectedColumns) {
		t.Errorf("Expected %d columns, got %d", len(expectedColumns), len(header))
	}

	// Verify data row
	row := records[1]
	if row[0] != "project-1" {
		t.Errorf("Expected ID 'project-1', got '%s'", row[0])
	}
	if row[2] != "Ball with acceptance criteria" {
		t.Errorf("Expected intent 'Ball with acceptance criteria', got '%s'", row[2])
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
		t.Errorf("Expected AcceptanceCriteria count '2', got '%s'", row[10])
	}
}

// Helper function to export to CSV format using new state model
func exportToCSV(balls []*session.Ball) ([]byte, error) {
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
		"AcceptanceCriteria",
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

		row := []string{
			ball.ID,
			ball.WorkingDir,
			ball.Title,
			string(ball.Priority),
			string(ball.State),
			ball.BlockedReason,
			ball.StartedAt.Format("2006-01-02 15:04:05"),
			completedAt,
			ball.LastActivity.Format("2006-01-02 15:04:05"),
			tags,
			fmt.Sprintf("%d", len(ball.AcceptanceCriteria)),
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

	store, err := session.NewStoreWithConfig(project, session.StoreConfig{JuggleDirName: ".juggle"})
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	ball := &session.Ball{
		ID:           "project-1",
		WorkingDir:   project,
		Title:       "Test ball",
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
		Balls      []*session.Ball `json:"balls"`
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
func exportToJSON(balls []*session.Ball) ([]byte, error) {
	export := struct {
		ExportedAt string             `json:"exported_at"`
		TotalBalls int                `json:"total_balls"`
		Balls      []*session.Ball `json:"balls"`
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
	ballStore, err := session.NewStoreWithConfig(project, session.StoreConfig{JuggleDirName: ".juggle"})
	if err != nil {
		t.Fatalf("Failed to create ball store: %v", err)
	}

	balls := []*session.Ball{
		{
			ID:                 "project-1",
			WorkingDir:         project,
			Title:             "Implement feature A",
			Priority:           session.PriorityHigh,
			State:              session.StateInProgress,
			StartedAt:          time.Now(),
			LastActivity:       time.Now(),
			Tags:               []string{"test-feature", "backend"},
			AcceptanceCriteria: []string{"Design API completed", "Logic implemented and tested"},
		},
		{
			ID:            "project-2",
			WorkingDir:    project,
			Title:        "Implement feature B",
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
	if !strings.Contains(outputStr, "Title: Implement feature A") {
		t.Error("Missing intent for project-1")
	}
	if !strings.Contains(outputStr, "Design API completed") {
		t.Error("Missing acceptance criterion 1")
	}
	if !strings.Contains(outputStr, "Logic implemented and tested") {
		t.Error("Missing acceptance criterion 2")
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
func exportToRalph(projectDir, sessionID string, balls []*session.Ball) ([]byte, error) {
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
func writeBallForRalphTest(buf *strings.Builder, ball *session.Ball) {
	buf.WriteString("## " + ball.ID + " [" + string(ball.State) + "] (priority: " + string(ball.Priority) + ")\n")
	buf.WriteString("Title: " + ball.Title + "\n")
	if ball.State == session.StateBlocked && ball.BlockedReason != "" {
		buf.WriteString("Blocked: " + ball.BlockedReason + "\n")
	}
	if len(ball.AcceptanceCriteria) > 0 {
		buf.WriteString("Acceptance Criteria:\n")
		for i, ac := range ball.AcceptanceCriteria {
			buf.WriteString("  " + fmt.Sprintf("%d", i+1) + ". " + ac + "\n")
		}
	}
	if len(ball.Tags) > 0 {
		buf.WriteString("Tags: " + strings.Join(ball.Tags, ", ") + "\n")
	}
}

// TestExportSessionWithAllFlag verifies session-filtered export across projects
func TestExportSessionWithAllFlag(t *testing.T) {
	// Create two test projects
	project1 := t.TempDir()
	project2 := t.TempDir()

	// Setup project 1 with a ball tagged with a session
	store1, err := session.NewStoreWithConfig(project1, session.StoreConfig{JuggleDirName: ".juggle"})
	if err != nil {
		t.Fatalf("Failed to create store1: %v", err)
	}

	ball1 := &session.Ball{
		ID:           "project1-1",
		WorkingDir:   project1,
		Title:       "Feature A in project 1",
		Priority:     session.PriorityMedium,
		State:        session.StateInProgress,
		Tags:         []string{"shared-session"},
		StartedAt:    time.Now(),
		LastActivity: time.Now(),
	}
	if err := store1.Save(ball1); err != nil {
		t.Fatalf("Failed to save ball1: %v", err)
	}

	// Add a ball without the session tag
	ball1NoSession := &session.Ball{
		ID:           "project1-2",
		WorkingDir:   project1,
		Title:       "Unrelated feature in project 1",
		Priority:     session.PriorityLow,
		State:        session.StatePending,
		Tags:         []string{"other-tag"},
		StartedAt:    time.Now(),
		LastActivity: time.Now(),
	}
	if err := store1.Save(ball1NoSession); err != nil {
		t.Fatalf("Failed to save ball1NoSession: %v", err)
	}

	// Setup project 2 with a ball also tagged with the same session
	store2, err := session.NewStoreWithConfig(project2, session.StoreConfig{JuggleDirName: ".juggle"})
	if err != nil {
		t.Fatalf("Failed to create store2: %v", err)
	}

	ball2 := &session.Ball{
		ID:           "project2-1",
		WorkingDir:   project2,
		Title:       "Feature B in project 2",
		Priority:     session.PriorityHigh,
		State:        session.StateInProgress,
		Tags:         []string{"shared-session"},
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

	t.Run("SessionFilterWithAll_FindsBallsFromBothProjects", func(t *testing.T) {
		// With --all flag, should find balls from both projects with the session tag
		cli.GlobalOpts.AllProjects = true
		cli.GlobalOpts.ProjectDir = project1
		defer func() { cli.GlobalOpts.AllProjects = false }()

		projects, err := cli.DiscoverProjectsForCommand(config, store1)
		if err != nil {
			t.Fatalf("Failed to discover projects: %v", err)
		}

		if len(projects) != 2 {
			t.Errorf("Expected 2 projects with --all, got %d", len(projects))
		}

		allBalls, err := session.LoadAllBalls(projects)
		if err != nil {
			t.Fatalf("Failed to load balls: %v", err)
		}

		// Should have 3 balls total (2 from project1, 1 from project2)
		if len(allBalls) != 3 {
			t.Errorf("Expected 3 total balls with --all, got %d", len(allBalls))
		}

		// Filter by session tag (like export --session does)
		filteredBalls := make([]*session.Ball, 0)
		for _, ball := range allBalls {
			for _, tag := range ball.Tags {
				if tag == "shared-session" {
					filteredBalls = append(filteredBalls, ball)
					break
				}
			}
		}

		// Should have 2 balls with the session tag from both projects
		if len(filteredBalls) != 2 {
			t.Errorf("Expected 2 balls with session tag from both projects, got %d", len(filteredBalls))
		}

		// Verify we got balls from both projects
		foundProject1, foundProject2 := false, false
		for _, ball := range filteredBalls {
			if ball.WorkingDir == project1 {
				foundProject1 = true
			}
			if ball.WorkingDir == project2 {
				foundProject2 = true
			}
		}
		if !foundProject1 || !foundProject2 {
			t.Errorf("Expected balls from both projects, got: project1=%v, project2=%v", foundProject1, foundProject2)
		}
	})

	t.Run("SessionFilterWithoutAll_OnlyLocalBalls", func(t *testing.T) {
		// Without --all, should only find balls from current project
		cli.GlobalOpts.AllProjects = false
		cli.GlobalOpts.ProjectDir = project1

		projects, err := cli.DiscoverProjectsForCommand(config, store1)
		if err != nil {
			t.Fatalf("Failed to discover projects: %v", err)
		}

		// Should only return current project
		if len(projects) != 1 {
			t.Errorf("Expected 1 project (local), got %d", len(projects))
		}

		allBalls, err := session.LoadAllBalls(projects)
		if err != nil {
			t.Fatalf("Failed to load balls: %v", err)
		}

		// Should have 2 balls from project1 only
		if len(allBalls) != 2 {
			t.Errorf("Expected 2 balls from local project, got %d", len(allBalls))
		}

		// Filter by session tag
		filteredBalls := make([]*session.Ball, 0)
		for _, ball := range allBalls {
			for _, tag := range ball.Tags {
				if tag == "shared-session" {
					filteredBalls = append(filteredBalls, ball)
					break
				}
			}
		}

		// Should only have 1 ball with the session tag (from local project only)
		if len(filteredBalls) != 1 {
			t.Errorf("Expected 1 ball with session tag from local project, got %d", len(filteredBalls))
		}

		// Verify it's from project1
		if len(filteredBalls) > 0 && filteredBalls[0].WorkingDir != project1 {
			t.Errorf("Expected ball from project1, got ball from %s", filteredBalls[0].WorkingDir)
		}
	})
}

// TestExportSortsBallsWithInProgressFirst verifies that in_progress balls appear first in agent exports
func TestExportSortsBallsWithInProgressFirst(t *testing.T) {
	project := t.TempDir()

	// Create session and ball stores
	sessionStore, err := session.NewSessionStore(project)
	if err != nil {
		t.Fatalf("Failed to create session store: %v", err)
	}

	_, err = sessionStore.CreateSession("test-session", "Test session")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	ballStore, err := session.NewStoreWithConfig(project, session.StoreConfig{JuggleDirName: ".juggle"})
	if err != nil {
		t.Fatalf("Failed to create ball store: %v", err)
	}

	// Create balls with mixed states and priorities
	// The order they're created should NOT be the order in the export
	balls := []*session.Ball{
		{
			ID:           "project-1",
			WorkingDir:   project,
			Title:       "Pending urgent ball",
			Priority:     session.PriorityUrgent,
			State:        session.StatePending,
			Tags:         []string{"test-session"},
			StartedAt:    time.Now(),
			LastActivity: time.Now(),
		},
		{
			ID:           "project-2",
			WorkingDir:   project,
			Title:       "In progress low priority ball",
			Priority:     session.PriorityLow,
			State:        session.StateInProgress,
			Tags:         []string{"test-session"},
			StartedAt:    time.Now(),
			LastActivity: time.Now(),
		},
		{
			ID:           "project-3",
			WorkingDir:   project,
			Title:       "Blocked high priority ball",
			Priority:     session.PriorityHigh,
			State:        session.StateBlocked,
			Tags:         []string{"test-session"},
			StartedAt:    time.Now(),
			LastActivity: time.Now(),
		},
		{
			ID:           "project-4",
			WorkingDir:   project,
			Title:       "In progress high priority ball",
			Priority:     session.PriorityHigh,
			State:        session.StateInProgress,
			Tags:         []string{"test-session"},
			StartedAt:    time.Now(),
			LastActivity: time.Now(),
		},
		{
			ID:           "project-5",
			WorkingDir:   project,
			Title:       "Pending low priority ball",
			Priority:     session.PriorityLow,
			State:        session.StatePending,
			Tags:         []string{"test-session"},
			StartedAt:    time.Now(),
			LastActivity: time.Now(),
		},
	}

	for _, ball := range balls {
		if err := ballStore.Save(ball); err != nil {
			t.Fatalf("Failed to save ball %s: %v", ball.ID, err)
		}
	}

	// Use SortBallsForAgentExport to verify sorting
	sortedBalls := make([]*session.Ball, len(balls))
	copy(sortedBalls, balls)
	cli.SortBallsForAgentExport(sortedBalls)

	// Expected order:
	// 1. In progress high priority (project-4)
	// 2. In progress low priority (project-2)
	// 3. Pending urgent (project-1)
	// 4. Pending low (project-5)
	// 5. Blocked high (project-3)

	t.Run("InProgressBallsAppearFirst", func(t *testing.T) {
		if sortedBalls[0].State != session.StateInProgress {
			t.Errorf("Expected first ball to be in_progress, got %s", sortedBalls[0].State)
		}
		if sortedBalls[1].State != session.StateInProgress {
			t.Errorf("Expected second ball to be in_progress, got %s", sortedBalls[1].State)
		}
	})

	t.Run("InProgressBallsSortedByPriority", func(t *testing.T) {
		// First in_progress ball should be high priority
		if sortedBalls[0].ID != "project-4" {
			t.Errorf("Expected first ball to be project-4 (in_progress high), got %s", sortedBalls[0].ID)
		}
		// Second in_progress ball should be low priority
		if sortedBalls[1].ID != "project-2" {
			t.Errorf("Expected second ball to be project-2 (in_progress low), got %s", sortedBalls[1].ID)
		}
	})

	t.Run("PendingBallsFollowInProgress", func(t *testing.T) {
		if sortedBalls[2].State != session.StatePending {
			t.Errorf("Expected third ball to be pending, got %s", sortedBalls[2].State)
		}
		// Pending urgent should come before pending low
		if sortedBalls[2].ID != "project-1" {
			t.Errorf("Expected third ball to be project-1 (pending urgent), got %s", sortedBalls[2].ID)
		}
		if sortedBalls[3].ID != "project-5" {
			t.Errorf("Expected fourth ball to be project-5 (pending low), got %s", sortedBalls[3].ID)
		}
	})

	t.Run("BlockedBallsAppearLast", func(t *testing.T) {
		if sortedBalls[4].State != session.StateBlocked {
			t.Errorf("Expected last ball to be blocked, got %s", sortedBalls[4].State)
		}
		if sortedBalls[4].ID != "project-3" {
			t.Errorf("Expected last ball to be project-3 (blocked), got %s", sortedBalls[4].ID)
		}
	})
}

// TestExportSortsBallsWithDependencies verifies that dependency ordering is considered in agent exports
func TestExportSortsBallsWithDependencies(t *testing.T) {
	project := t.TempDir()

	// Create session and ball stores
	sessionStore, err := session.NewSessionStore(project)
	if err != nil {
		t.Fatalf("Failed to create session store: %v", err)
	}

	_, err = sessionStore.CreateSession("test-session", "Test session")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	ballStore, err := session.NewStoreWithConfig(project, session.StoreConfig{JuggleDirName: ".juggle"})
	if err != nil {
		t.Fatalf("Failed to create ball store: %v", err)
	}

	// Create balls where one has dependencies on another
	balls := []*session.Ball{
		{
			ID:           "project-1",
			WorkingDir:   project,
			Title:       "Independent ball",
			Priority:     session.PriorityMedium,
			State:        session.StatePending,
			Tags:         []string{"test-session"},
			DependsOn:    []string{}, // No dependencies
			StartedAt:    time.Now(),
			LastActivity: time.Now(),
		},
		{
			ID:           "project-2",
			WorkingDir:   project,
			Title:       "Dependent ball (depends on project-1)",
			Priority:     session.PriorityHigh, // Higher priority but has unsatisfied dep
			State:        session.StatePending,
			Tags:         []string{"test-session"},
			DependsOn:    []string{"project-1"}, // Depends on project-1
			StartedAt:    time.Now(),
			LastActivity: time.Now(),
		},
	}

	for _, ball := range balls {
		if err := ballStore.Save(ball); err != nil {
			t.Fatalf("Failed to save ball %s: %v", ball.ID, err)
		}
	}

	// Sort balls for agent export
	sortedBalls := make([]*session.Ball, len(balls))
	copy(sortedBalls, balls)
	cli.SortBallsForAgentExport(sortedBalls)

	t.Run("BallsWithSatisfiedDepsBeforeUnsatisfiedDeps", func(t *testing.T) {
		// project-1 (no deps, satisfied) should come before project-2 (unsatisfied dep)
		// even though project-2 has higher priority
		if sortedBalls[0].ID != "project-1" {
			t.Errorf("Expected ball without dependencies (project-1) first, got %s", sortedBalls[0].ID)
		}
		if sortedBalls[1].ID != "project-2" {
			t.Errorf("Expected ball with unsatisfied dependency (project-2) second, got %s", sortedBalls[1].ID)
		}
	})
}

// TestExportSortsBallsWithSatisfiedDependencies verifies balls with satisfied deps come first
func TestExportSortsBallsWithSatisfiedDependencies(t *testing.T) {
	project := t.TempDir()

	// Create session and ball stores
	sessionStore, err := session.NewSessionStore(project)
	if err != nil {
		t.Fatalf("Failed to create session store: %v", err)
	}

	_, err = sessionStore.CreateSession("test-session", "Test session")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	ballStore, err := session.NewStoreWithConfig(project, session.StoreConfig{JuggleDirName: ".juggle"})
	if err != nil {
		t.Fatalf("Failed to create ball store: %v", err)
	}

	// Create balls where dependency is already complete
	balls := []*session.Ball{
		{
			ID:           "project-1",
			WorkingDir:   project,
			Title:       "Completed base task",
			Priority:     session.PriorityMedium,
			State:        session.StateComplete, // Already complete
			Tags:         []string{"test-session"},
			StartedAt:    time.Now(),
			LastActivity: time.Now(),
		},
		{
			ID:           "project-2",
			WorkingDir:   project,
			Title:       "Task with satisfied dep (project-1 is complete)",
			Priority:     session.PriorityMedium,
			State:        session.StatePending,
			Tags:         []string{"test-session"},
			DependsOn:    []string{"project-1"}, // Depends on completed ball
			StartedAt:    time.Now(),
			LastActivity: time.Now(),
		},
		{
			ID:           "project-3",
			WorkingDir:   project,
			Title:       "Task with unsatisfied dep",
			Priority:     session.PriorityMedium,
			State:        session.StatePending,
			Tags:         []string{"test-session"},
			DependsOn:    []string{"project-99"}, // Depends on non-existent ball (treated as satisfied)
			StartedAt:    time.Now(),
			LastActivity: time.Now(),
		},
	}

	for _, ball := range balls {
		if err := ballStore.Save(ball); err != nil {
			t.Fatalf("Failed to save ball %s: %v", ball.ID, err)
		}
	}

	// Sort balls for agent export
	sortedBalls := make([]*session.Ball, len(balls))
	copy(sortedBalls, balls)
	cli.SortBallsForAgentExport(sortedBalls)

	t.Run("SatisfiedDepsGoFirst", func(t *testing.T) {
		// project-2 has satisfied dep (project-1 is complete)
		// project-3 has missing dep (treated as satisfied since not in set)
		// Both should be considered as having satisfied deps
		// Complete balls go last, so project-1 should be last
		foundComplete := false
		for i, ball := range sortedBalls {
			if ball.State == session.StateComplete {
				if i != len(sortedBalls)-1 {
					t.Errorf("Expected complete ball to be last, found at position %d", i)
				}
				foundComplete = true
			}
		}
		if !foundComplete {
			t.Error("Expected to find complete ball in sorted list")
		}
	})
}

// TestExportAgentIncludesInProgressBalls verifies that in_progress balls are included in agent export
func TestExportAgentIncludesInProgressBalls(t *testing.T) {
	project := t.TempDir()

	// Create session store
	sessionStore, err := session.NewSessionStore(project)
	if err != nil {
		t.Fatalf("Failed to create session store: %v", err)
	}

	_, err = sessionStore.CreateSession("test-session", "Test session for in_progress balls")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Create ball store
	ballStore, err := session.NewStoreWithConfig(project, session.StoreConfig{JuggleDirName: ".juggle"})
	if err != nil {
		t.Fatalf("Failed to create ball store: %v", err)
	}

	// Create an in_progress ball
	inProgressBall := &session.Ball{
		ID:                 "project-1",
		WorkingDir:         project,
		Title:             "In progress work item",
		Priority:           session.PriorityHigh,
		State:              session.StateInProgress,
		Tags:               []string{"test-session"},
		AcceptanceCriteria: []string{"AC 1", "AC 2"},
		StartedAt:          time.Now(),
		LastActivity:       time.Now(),
	}

	if err := ballStore.Save(inProgressBall); err != nil {
		t.Fatalf("Failed to save ball: %v", err)
	}

	// Load all balls and verify in_progress is included
	allBalls, err := session.LoadAllBalls([]string{project})
	if err != nil {
		t.Fatalf("Failed to load balls: %v", err)
	}

	// Filter by session
	sessionBalls := make([]*session.Ball, 0)
	for _, ball := range allBalls {
		for _, tag := range ball.Tags {
			if tag == "test-session" {
				sessionBalls = append(sessionBalls, ball)
				break
			}
		}
	}

	if len(sessionBalls) != 1 {
		t.Fatalf("Expected 1 ball, got %d", len(sessionBalls))
	}

	if sessionBalls[0].State != session.StateInProgress {
		t.Errorf("Expected ball to be in_progress, got %s", sessionBalls[0].State)
	}

	// Verify the ball appears in export output (using ralph format as proxy)
	output, err := exportToRalph(project, "test-session", sessionBalls)
	if err != nil {
		t.Fatalf("Failed to export: %v", err)
	}

	outputStr := string(output)

	// Verify in_progress ball appears in export
	if !strings.Contains(outputStr, "[in_progress]") {
		t.Error("Expected export to contain [in_progress] state marker")
	}

	if !strings.Contains(outputStr, "In progress work item") {
		t.Error("Expected export to contain in_progress ball intent")
	}
}

// TestExportAgentExcludesCompleteBallsByDefault verifies that complete balls are excluded from agent export by default
func TestExportAgentExcludesCompleteBallsByDefault(t *testing.T) {
	project := t.TempDir()

	// Create session store
	sessionStore, err := session.NewSessionStore(project)
	if err != nil {
		t.Fatalf("Failed to create session store: %v", err)
	}

	_, err = sessionStore.CreateSession("test-session", "Test session for complete balls filtering")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Create ball store
	ballStore, err := session.NewStoreWithConfig(project, session.StoreConfig{JuggleDirName: ".juggle"})
	if err != nil {
		t.Fatalf("Failed to create ball store: %v", err)
	}

	// Create an in_progress ball
	inProgressBall := &session.Ball{
		ID:                 "project-inprogress",
		WorkingDir:         project,
		Title:             "In progress work item",
		Priority:           session.PriorityHigh,
		State:              session.StateInProgress,
		Tags:               []string{"test-session"},
		AcceptanceCriteria: []string{"AC 1", "AC 2"},
		StartedAt:          time.Now(),
		LastActivity:       time.Now(),
	}

	// Create a pending ball
	pendingBall := &session.Ball{
		ID:                 "project-pending",
		WorkingDir:         project,
		Title:             "Pending work item",
		Priority:           session.PriorityMedium,
		State:              session.StatePending,
		Tags:               []string{"test-session"},
		AcceptanceCriteria: []string{"AC 3"},
		StartedAt:          time.Now(),
		LastActivity:       time.Now(),
	}

	// Create a complete ball
	completedTime := time.Now()
	completeBall := &session.Ball{
		ID:                 "project-complete",
		WorkingDir:         project,
		Title:             "Completed work item",
		Priority:           session.PriorityLow,
		State:              session.StateComplete,
		Tags:               []string{"test-session"},
		AcceptanceCriteria: []string{"AC 4"},
		StartedAt:          time.Now().Add(-1 * time.Hour),
		LastActivity:       time.Now().Add(-30 * time.Minute),
		CompletedAt:        &completedTime,
		CompletionNote:     "All done",
	}

	if err := ballStore.Save(inProgressBall); err != nil {
		t.Fatalf("Failed to save in_progress ball: %v", err)
	}
	if err := ballStore.Save(pendingBall); err != nil {
		t.Fatalf("Failed to save pending ball: %v", err)
	}
	if err := ballStore.Save(completeBall); err != nil {
		t.Fatalf("Failed to save complete ball: %v", err)
	}

	// Load all balls
	allBalls, err := session.LoadAllBalls([]string{project})
	if err != nil {
		t.Fatalf("Failed to load balls: %v", err)
	}

	// Filter by session
	sessionBalls := make([]*session.Ball, 0)
	for _, ball := range allBalls {
		for _, tag := range ball.Tags {
			if tag == "test-session" {
				sessionBalls = append(sessionBalls, ball)
				break
			}
		}
	}

	if len(sessionBalls) != 3 {
		t.Fatalf("Expected 3 balls, got %d", len(sessionBalls))
	}

	t.Run("ExcludeCompleteBallsByDefault", func(t *testing.T) {
		// Filter out complete balls (simulating default behavior)
		filtered := make([]*session.Ball, 0)
		for _, ball := range sessionBalls {
			if ball.State != session.StateComplete {
				filtered = append(filtered, ball)
			}
		}

		if len(filtered) != 2 {
			t.Errorf("Expected 2 non-complete balls, got %d", len(filtered))
		}

		// Export the filtered balls
		output, err := exportToRalph(project, "test-session", filtered)
		if err != nil {
			t.Fatalf("Failed to export: %v", err)
		}

		outputStr := string(output)

		// Should contain in_progress and pending balls
		if !strings.Contains(outputStr, "In progress work item") {
			t.Error("Expected export to contain in_progress ball")
		}
		if !strings.Contains(outputStr, "Pending work item") {
			t.Error("Expected export to contain pending ball")
		}

		// Should NOT contain complete ball
		if strings.Contains(outputStr, "Completed work item") {
			t.Error("Expected export to NOT contain complete ball by default")
		}
	})

	t.Run("IncludeCompleteBallsWithFlag", func(t *testing.T) {
		// With include-done flag, all balls should be included
		output, err := exportToRalph(project, "test-session", sessionBalls)
		if err != nil {
			t.Fatalf("Failed to export: %v", err)
		}

		outputStr := string(output)

		// Should contain all balls including complete
		if !strings.Contains(outputStr, "In progress work item") {
			t.Error("Expected export to contain in_progress ball")
		}
		if !strings.Contains(outputStr, "Pending work item") {
			t.Error("Expected export to contain pending ball")
		}
		if !strings.Contains(outputStr, "Completed work item") {
			t.Error("Expected export to contain complete ball with include-done flag")
		}
	})
}

// TestExportRalphExcludesCompleteBallsByDefault verifies that complete balls are excluded from ralph export by default
func TestExportRalphExcludesCompleteBallsByDefault(t *testing.T) {
	project := t.TempDir()

	// Create ball store
	ballStore, err := session.NewStoreWithConfig(project, session.StoreConfig{JuggleDirName: ".juggle"})
	if err != nil {
		t.Fatalf("Failed to create ball store: %v", err)
	}

	// Create balls with different states
	pendingBall := &session.Ball{
		ID:         "project-pending",
		WorkingDir: project,
		Title:     "Pending task for ralph",
		Priority:   session.PriorityMedium,
		State:      session.StatePending,
		Tags:       []string{"ralph-test"},
		StartedAt:  time.Now(),
	}

	completedTime := time.Now()
	completeBall := &session.Ball{
		ID:             "project-complete",
		WorkingDir:     project,
		Title:         "Completed task for ralph",
		Priority:       session.PriorityMedium,
		State:          session.StateComplete,
		Tags:           []string{"ralph-test"},
		StartedAt:      time.Now().Add(-1 * time.Hour),
		CompletedAt:    &completedTime,
		CompletionNote: "Done",
	}

	if err := ballStore.Save(pendingBall); err != nil {
		t.Fatalf("Failed to save pending ball: %v", err)
	}
	if err := ballStore.Save(completeBall); err != nil {
		t.Fatalf("Failed to save complete ball: %v", err)
	}

	// Load all balls
	allBalls, err := session.LoadAllBalls([]string{project})
	if err != nil {
		t.Fatalf("Failed to load balls: %v", err)
	}

	// Filter by session tag
	sessionBalls := make([]*session.Ball, 0)
	for _, ball := range allBalls {
		for _, tag := range ball.Tags {
			if tag == "ralph-test" {
				sessionBalls = append(sessionBalls, ball)
				break
			}
		}
	}

	if len(sessionBalls) != 2 {
		t.Fatalf("Expected 2 balls, got %d", len(sessionBalls))
	}

	// Filter out complete balls (default behavior)
	filtered := make([]*session.Ball, 0)
	for _, ball := range sessionBalls {
		if ball.State != session.StateComplete {
			filtered = append(filtered, ball)
		}
	}

	// Export filtered balls
	output, err := exportToRalph(project, "ralph-test", filtered)
	if err != nil {
		t.Fatalf("Failed to export: %v", err)
	}

	outputStr := string(output)

	// Should contain pending but not complete
	if !strings.Contains(outputStr, "Pending task for ralph") {
		t.Error("Expected export to contain pending ball")
	}
	if strings.Contains(outputStr, "Completed task for ralph") {
		t.Error("Expected export to NOT contain complete ball")
	}
}

// TestExportAgentIncludesSessionID verifies that the agent export includes the session ID
// This is critical for ensuring the agent knows which session it's working on and doesn't
// work on balls from other sessions.
func TestExportAgentIncludesSessionID(t *testing.T) {
	project := t.TempDir()

	// Create session store
	sessionStore, err := session.NewSessionStore(project)
	if err != nil {
		t.Fatalf("Failed to create session store: %v", err)
	}

	_, err = sessionStore.CreateSession("my-test-session", "Test session for session ID verification")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Create ball store
	ballStore, err := session.NewStoreWithConfig(project, session.StoreConfig{JuggleDirName: ".juggle"})
	if err != nil {
		t.Fatalf("Failed to create ball store: %v", err)
	}

	// Create a test ball tagged with the session
	ball := &session.Ball{
		ID:                 "project-1",
		WorkingDir:         project,
		Title:              "Test ball for session verification",
		Priority:           session.PriorityMedium,
		State:              session.StateInProgress,
		Tags:               []string{"my-test-session"},
		AcceptanceCriteria: []string{"AC 1"},
		StartedAt:          time.Now(),
		LastActivity:       time.Now(),
	}

	if err := ballStore.Save(ball); err != nil {
		t.Fatalf("Failed to save ball: %v", err)
	}

	// Export to agent format
	output, err := exportToAgent(project, "my-test-session", []*session.Ball{ball}, false, false)
	if err != nil {
		t.Fatalf("Failed to export: %v", err)
	}

	outputStr := string(output)

	// Verify <session> section exists with the correct session ID
	if !strings.Contains(outputStr, "<session>") {
		t.Error("Missing <session> opening tag in agent export")
	}
	if !strings.Contains(outputStr, "</session>") {
		t.Error("Missing </session> closing tag in agent export")
	}
	if !strings.Contains(outputStr, "my-test-session") {
		t.Error("Session ID 'my-test-session' not found in agent export")
	}

	// Verify session section appears between context and progress (structural validation)
	contextIdx := strings.Index(outputStr, "</context>")
	sessionIdx := strings.Index(outputStr, "<session>")
	progressIdx := strings.Index(outputStr, "<progress>")

	if contextIdx == -1 || sessionIdx == -1 || progressIdx == -1 {
		t.Error("Missing required sections in export")
	}

	if sessionIdx < contextIdx {
		t.Error("Expected <session> section to appear after </context>")
	}
	if sessionIdx > progressIdx {
		t.Error("Expected <session> section to appear before <progress>")
	}
}

// Helper function to export to agent format (mirrors cli.exportAgent)
func exportToAgent(projectDir, sessionID string, balls []*session.Ball, debug bool, singleBall bool) ([]byte, error) {
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

	// Load progress and limit to last 50 lines
	progress, _ := sessionStore.LoadProgress(sessionID)
	lines := strings.Split(progress, "\n")
	if len(lines) > 50 {
		progress = strings.Join(lines[len(lines)-50:], "\n")
	}

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

	// Write <session> section with the session ID
	buf.WriteString("<session>\n")
	buf.WriteString(sessionID)
	buf.WriteString("\n</session>\n\n")

	// Write <progress> section
	buf.WriteString("<progress>\n")
	if progress != "" {
		buf.WriteString(progress)
		if !strings.HasSuffix(progress, "\n") {
			buf.WriteString("\n")
		}
	}
	buf.WriteString("</progress>\n\n")

	// Write <balls> section
	buf.WriteString("<balls>\n")
	for i, ball := range balls {
		if i > 0 {
			buf.WriteString("\n")
		}
		writeBallForAgentTest(&buf, ball)
	}
	buf.WriteString("</balls>\n\n")

	// Write <instructions> section (simplified for testing)
	buf.WriteString("<instructions>\n")
	buf.WriteString("Agent instructions would go here.\n")
	buf.WriteString("</instructions>\n")

	return []byte(buf.String()), nil
}

// writeBallForAgentTest writes a single ball in agent format (test helper)
func writeBallForAgentTest(buf *strings.Builder, ball *session.Ball) {
	buf.WriteString("## " + ball.ID + " [" + string(ball.State) + "] (priority: " + string(ball.Priority) + ")\n")
	buf.WriteString("Title: " + ball.Title + "\n")
	if len(ball.AcceptanceCriteria) > 0 {
		buf.WriteString("Acceptance Criteria:\n")
		for i, ac := range ball.AcceptanceCriteria {
			buf.WriteString("  " + fmt.Sprintf("%d", i+1) + ". " + ac + "\n")
		}
	}
	if len(ball.Tags) > 0 {
		buf.WriteString("Tags: " + strings.Join(ball.Tags, ", ") + "\n")
	}
}
