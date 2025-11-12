package integration_test

import (
	"encoding/csv"
	"encoding/json"
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
		ActiveState:  session.ActiveJuggling,
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
		ActiveState:  session.ActiveJuggling,
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

	// Create multiple balls
	balls := []*session.Session{
		{
			ID:           "project-1",
			WorkingDir:   project,
			Intent:       "Ball 1",
			Priority:     session.PriorityMedium,
			ActiveState:  session.ActiveJuggling,
			StartedAt:    time.Now(),
			LastActivity: time.Now(),
		},
		{
			ID:           "project-2",
			WorkingDir:   project,
			Intent:       "Ball 2",
			Priority:     session.PriorityHigh,
			ActiveState:  session.ActiveReady,
			StartedAt:    time.Now(),
			LastActivity: time.Now(),
		},
		{
			ID:           "project-3",
			WorkingDir:   project,
			Intent:       "Ball 3",
			Priority:     session.PriorityLow,
			ActiveState:  session.ActiveJuggling,
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

	// Create balls in different states
	inAir := session.JuggleInAir
	needsCaught := session.JuggleNeedsCaught

	balls := []*session.Session{
		{
			ID:           "project-1",
			WorkingDir:   project,
			Intent:       "Ready ball",
			Priority:     session.PriorityMedium,
			ActiveState:  session.ActiveReady,
			StartedAt:    time.Now(),
			LastActivity: time.Now(),
		},
		{
			ID:           "project-2",
			WorkingDir:   project,
			Intent:       "Juggling in-air",
			Priority:     session.PriorityHigh,
			ActiveState:  session.ActiveJuggling,
			JuggleState:  &inAir,
			StartedAt:    time.Now(),
			LastActivity: time.Now(),
		},
		{
			ID:           "project-3",
			WorkingDir:   project,
			Intent:       "Juggling needs-caught",
			Priority:     session.PriorityLow,
			ActiveState:  session.ActiveJuggling,
			JuggleState:  &needsCaught,
			StartedAt:    time.Now(),
			LastActivity: time.Now(),
		},
		{
			ID:           "project-4",
			WorkingDir:   project,
			Intent:       "Dropped ball",
			Priority:     session.PriorityMedium,
			ActiveState:  session.ActiveDropped,
			StartedAt:    time.Now(),
			LastActivity: time.Now(),
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

	// Test filtering by active state only
	t.Run("FilterByJuggling", func(t *testing.T) {
		filtered, err := filterBallsByState(allBalls, "juggling")
		if err != nil {
			t.Fatalf("Failed to filter balls: %v", err)
		}

		if len(filtered) != 2 {
			t.Errorf("Expected 2 juggling balls, got %d", len(filtered))
		}
	})

	// Test filtering by specific juggle state
	t.Run("FilterByInAir", func(t *testing.T) {
		filtered, err := filterBallsByState(allBalls, "juggling:in-air")
		if err != nil {
			t.Fatalf("Failed to filter balls: %v", err)
		}

		if len(filtered) != 1 {
			t.Errorf("Expected 1 in-air ball, got %d", len(filtered))
		}
		if len(filtered) > 0 && filtered[0].ID != "project-2" {
			t.Errorf("Expected ball 'project-2', got '%s'", filtered[0].ID)
		}
	})

	// Test filtering by multiple states
	t.Run("FilterByMultipleStates", func(t *testing.T) {
		filtered, err := filterBallsByState(allBalls, "ready,dropped")
		if err != nil {
			t.Fatalf("Failed to filter balls: %v", err)
		}

		if len(filtered) != 2 {
			t.Errorf("Expected 2 balls (ready+dropped), got %d", len(filtered))
		}
	})

	// Test invalid state
	t.Run("InvalidState", func(t *testing.T) {
		_, err := filterBallsByState(allBalls, "invalid-state")
		if err == nil {
			t.Errorf("Expected error for invalid state, got none")
		}
	})

	// Test invalid juggle state format
	t.Run("InvalidJuggleStateFormat", func(t *testing.T) {
		_, err := filterBallsByState(allBalls, "ready:in-air")
		if err == nil {
			t.Errorf("Expected error for juggle state with non-juggling active state, got none")
		}
	})
}

// Helper function to test state filtering logic
func filterBallsByState(balls []*session.Session, stateStr string) ([]*session.Session, error) {
	stateStrs := strings.Split(stateStr, ",")
	type stateFilter struct {
		activeState session.ActiveState
		juggleState string
	}
	stateFilters := make([]stateFilter, 0, len(stateStrs))

	for _, s := range stateStrs {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}

		parts := strings.Split(s, ":")
		activeState := parts[0]
		var juggleState string
		if len(parts) > 1 {
			juggleState = parts[1]
		}

		if !isValidActiveState(activeState) {
			return nil, &InvalidStateError{State: activeState}
		}

		if juggleState != "" {
			if activeState != "juggling" {
				return nil, &InvalidJuggleStateFormatError{StateStr: s}
			}
			if !isValidJuggleState(juggleState) {
				return nil, &InvalidJuggleStateError{State: juggleState}
			}
		}

		stateFilters = append(stateFilters, stateFilter{
			activeState: session.ActiveState(activeState),
			juggleState: juggleState,
		})
	}

	if len(stateFilters) == 0 {
		return balls, nil
	}

	filteredBalls := make([]*session.Session, 0)
	for _, ball := range balls {
		for _, filter := range stateFilters {
			if matchesStateFilter(ball, filter) {
				filteredBalls = append(filteredBalls, ball)
				break
			}
		}
	}

	return filteredBalls, nil
}

func matchesStateFilter(ball *session.Session, filter struct {
	activeState session.ActiveState
	juggleState string
}) bool {
	if ball.ActiveState != filter.activeState {
		return false
	}

	if filter.juggleState == "" {
		return true
	}

	if ball.JuggleState == nil {
		return false
	}

	return string(*ball.JuggleState) == filter.juggleState
}

func isValidActiveState(state string) bool {
	return state == "ready" || state == "juggling" || state == "dropped" || state == "complete"
}

func isValidJuggleState(state string) bool {
	return state == "needs-thrown" || state == "in-air" || state == "needs-caught"
}

type InvalidStateError struct {
	State string
}

func (e *InvalidStateError) Error() string {
	return "invalid active state: " + e.State + " (must be ready, juggling, dropped, or complete)"
}

type InvalidJuggleStateError struct {
	State string
}

func (e *InvalidJuggleStateError) Error() string {
	return "invalid juggle state: " + e.State + " (must be needs-thrown, in-air, or needs-caught)"
}

type InvalidJuggleStateFormatError struct {
	StateStr string
}

func (e *InvalidJuggleStateFormatError) Error() string {
	return "juggle state can only be specified with 'juggling' active state: " + e.StateStr
}

// TestExportIncludeDone verifies --include-done filtering logic
func TestExportIncludeDone(t *testing.T) {
	project := t.TempDir()

	store, err := session.NewStoreWithConfig(project, session.StoreConfig{JugglerDirName: ".juggler"})
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Create active and completed balls
	completedTime := time.Now()
	balls := []*session.Session{
		{
			ID:           "project-1",
			WorkingDir:   project,
			Intent:       "Active ball",
			Priority:     session.PriorityMedium,
			ActiveState:  session.ActiveJuggling,
			StartedAt:    time.Now(),
			LastActivity: time.Now(),
		},
		{
			ID:             "project-2",
			WorkingDir:     project,
			Intent:         "Completed ball",
			Priority:       session.PriorityHigh,
			ActiveState:    session.ActiveComplete,
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
			if ball.ActiveState != session.ActiveComplete {
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

	// Create a ball with todos
	inAir := session.JuggleInAir
	ball := &session.Session{
		ID:           "project-1",
		WorkingDir:   project,
		Intent:       "Ball with todos",
		Priority:     session.PriorityHigh,
		ActiveState:  session.ActiveJuggling,
		JuggleState:  &inAir,
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

	// Verify header
	header := records[0]
	expectedColumns := []string{"ID", "Project", "Intent", "Priority", "ActiveState", "JuggleState", "StartedAt", "CompletedAt", "LastActivity", "Tags", "TodosTotal", "TodosCompleted", "CompletionNote"}
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
	if row[5] != "in-air" {
		t.Errorf("Expected juggle state 'in-air', got '%s'", row[5])
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

// Helper function to export to CSV format
func exportToCSV(balls []*session.Session) ([]byte, error) {
	var buf strings.Builder
	writer := csv.NewWriter(&buf)

	// Write header
	header := []string{
		"ID",
		"Project",
		"Intent",
		"Priority",
		"ActiveState",
		"JuggleState",
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

		juggleState := ""
		if ball.JuggleState != nil {
			juggleState = string(*ball.JuggleState)
		}
		row := []string{
			ball.ID,
			ball.WorkingDir,
			ball.Intent,
			string(ball.Priority),
			string(ball.ActiveState),
			juggleState,
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
		ActiveState:  session.ActiveReady,
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
