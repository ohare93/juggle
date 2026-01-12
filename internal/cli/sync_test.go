package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ohare93/juggle/internal/session"
)

func TestSyncRalph(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Create .juggler directory
	jugglerDir := filepath.Join(tmpDir, ".juggler")
	if err := os.MkdirAll(jugglerDir, 0755); err != nil {
		t.Fatalf("failed to create .juggler dir: %v", err)
	}

	// Create archive directory
	archiveDir := filepath.Join(jugglerDir, "archive")
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		t.Fatalf("failed to create archive dir: %v", err)
	}

	// Create test prd.json
	prdFile := PRDFile{
		Project:     "TestProject",
		BranchName:  "feature/test",
		Description: "Test project",
		UserStories: []UserStory{
			{
				ID:                 "US-001",
				Title:              "First Story",
				Description:        "First story description",
				AcceptanceCriteria: []string{"Criterion 1", "Criterion 2"},
				Priority:           1,
				Passes:             false,
			},
			{
				ID:                 "US-002",
				Title:              "Second Story",
				Description:        "Second story description",
				AcceptanceCriteria: []string{"Criterion A"},
				Priority:           5,
				Passes:             true,
			},
		},
	}

	prdData, err := json.MarshalIndent(prdFile, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal prd: %v", err)
	}

	prdPath := filepath.Join(tmpDir, "prd.json")
	if err := os.WriteFile(prdPath, prdData, 0644); err != nil {
		t.Fatalf("failed to write prd.json: %v", err)
	}

	// Run sync
	if err := syncPRDFile(prdPath, tmpDir); err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	// Verify balls were created
	store, err := session.NewStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	balls, err := store.LoadBalls()
	if err != nil {
		t.Fatalf("failed to load balls: %v", err)
	}

	if len(balls) != 2 {
		t.Fatalf("expected 2 balls, got %d", len(balls))
	}

	// Find balls by intent
	var firstBall, secondBall *session.Ball
	for _, ball := range balls {
		if ball.Intent == "First Story" {
			firstBall = ball
		} else if ball.Intent == "Second Story" {
			secondBall = ball
		}
	}

	if firstBall == nil {
		t.Fatal("first ball not found")
	}
	if secondBall == nil {
		t.Fatal("second ball not found")
	}

	// Verify first ball (passes: false -> pending)
	if firstBall.State != session.StatePending {
		t.Errorf("expected first ball state pending, got %s", firstBall.State)
	}
	if len(firstBall.AcceptanceCriteria) != 2 {
		t.Errorf("expected 2 acceptance criteria, got %d", len(firstBall.AcceptanceCriteria))
	}
	if firstBall.Priority != session.PriorityUrgent {
		t.Errorf("expected priority urgent (p1), got %s", firstBall.Priority)
	}

	// Verify second ball (passes: true -> complete)
	if secondBall.State != session.StateComplete {
		t.Errorf("expected second ball state complete, got %s", secondBall.State)
	}
	if secondBall.Priority != session.PriorityHigh {
		t.Errorf("expected priority high (p5), got %s", secondBall.Priority)
	}

	// Verify tags contain story ID
	hasTag := false
	for _, tag := range firstBall.Tags {
		if tag == "US-001" {
			hasTag = true
			break
		}
	}
	if !hasTag {
		t.Errorf("first ball should have US-001 tag, got %v", firstBall.Tags)
	}
}

func TestSyncRalphUpdate(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Create .juggler directory
	jugglerDir := filepath.Join(tmpDir, ".juggler")
	if err := os.MkdirAll(jugglerDir, 0755); err != nil {
		t.Fatalf("failed to create .juggler dir: %v", err)
	}

	// Create archive directory
	archiveDir := filepath.Join(jugglerDir, "archive")
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		t.Fatalf("failed to create archive dir: %v", err)
	}

	// Create initial prd.json with passes: false
	prdFile := PRDFile{
		Project: "TestProject",
		UserStories: []UserStory{
			{
				ID:       "US-001",
				Title:    "Test Story",
				Priority: 5,
				Passes:   false,
			},
		},
	}

	prdData, _ := json.MarshalIndent(prdFile, "", "  ")
	prdPath := filepath.Join(tmpDir, "prd.json")
	os.WriteFile(prdPath, prdData, 0644)

	// First sync - should create ball as pending
	if err := syncPRDFile(prdPath, tmpDir); err != nil {
		t.Fatalf("first sync failed: %v", err)
	}

	store, _ := session.NewStore(tmpDir)
	balls, _ := store.LoadBalls()
	if balls[0].State != session.StatePending {
		t.Errorf("expected pending, got %s", balls[0].State)
	}

	// Update prd.json to passes: true
	prdFile.UserStories[0].Passes = true
	prdData, _ = json.MarshalIndent(prdFile, "", "  ")
	os.WriteFile(prdPath, prdData, 0644)

	// Second sync - should update ball to complete
	if err := syncPRDFile(prdPath, tmpDir); err != nil {
		t.Fatalf("second sync failed: %v", err)
	}

	balls, _ = store.LoadBalls()
	if balls[0].State != session.StateComplete {
		t.Errorf("expected complete, got %s", balls[0].State)
	}
}

func TestMapPriorityNumber(t *testing.T) {
	tests := []struct {
		priority int
		expected session.Priority
	}{
		{1, session.PriorityUrgent},
		{2, session.PriorityUrgent},
		{3, session.PriorityHigh},
		{5, session.PriorityHigh},
		{6, session.PriorityMedium},
		{10, session.PriorityMedium},
		{11, session.PriorityLow},
		{100, session.PriorityLow},
	}

	for _, tt := range tests {
		got := mapPriorityNumber(tt.priority)
		if got != tt.expected {
			t.Errorf("mapPriorityNumber(%d) = %s, want %s", tt.priority, got, tt.expected)
		}
	}
}

func TestMapPassesToState(t *testing.T) {
	// Test passes: true -> complete
	if state := mapPassesToState(true, nil); state != session.StateComplete {
		t.Errorf("expected complete for passes=true, got %s", state)
	}

	// Test passes: false with no ball -> pending
	if state := mapPassesToState(false, nil); state != session.StatePending {
		t.Errorf("expected pending for passes=false with nil ball, got %s", state)
	}

	// Test passes: false with pending ball -> pending
	pendingBall := &session.Ball{
		State: session.StatePending,
	}
	if state := mapPassesToState(false, pendingBall); state != session.StatePending {
		t.Errorf("expected pending for passes=false with pending ball, got %s", state)
	}

	// Test passes: false with in_progress ball -> in_progress (preserves state)
	inProgressBall := &session.Ball{
		State: session.StateInProgress,
	}
	if state := mapPassesToState(false, inProgressBall); state != session.StateInProgress {
		t.Errorf("expected in_progress for passes=false with in_progress ball, got %s", state)
	}
}

func TestMapStateToPasses(t *testing.T) {
	tests := []struct {
		state    session.BallState
		expected bool
	}{
		{session.StateComplete, true},
		{session.StateResearched, true},
		{session.StatePending, false},
		{session.StateInProgress, false},
		{session.StateBlocked, false},
	}

	for _, tt := range tests {
		got := mapStateToPasses(tt.state)
		if got != tt.expected {
			t.Errorf("mapStateToPasses(%s) = %t, want %t", tt.state, got, tt.expected)
		}
	}
}

func TestMapPriorityToNumber(t *testing.T) {
	tests := []struct {
		priority session.Priority
		expected int
	}{
		{session.PriorityUrgent, 1},
		{session.PriorityHigh, 4},
		{session.PriorityMedium, 7},
		{session.PriorityLow, 15},
	}

	for _, tt := range tests {
		got := mapPriorityToNumber(tt.priority)
		if got != tt.expected {
			t.Errorf("mapPriorityToNumber(%s) = %d, want %d", tt.priority, got, tt.expected)
		}
	}
}

func TestWriteToPRD(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Create .juggler directory
	jugglerDir := filepath.Join(tmpDir, ".juggler")
	if err := os.MkdirAll(jugglerDir, 0755); err != nil {
		t.Fatalf("failed to create .juggler dir: %v", err)
	}

	// Create archive directory
	archiveDir := filepath.Join(jugglerDir, "archive")
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		t.Fatalf("failed to create archive dir: %v", err)
	}

	// Create prd.json with two stories (both passes: false)
	prdFile := PRDFile{
		Project: "TestProject",
		UserStories: []UserStory{
			{
				ID:       "US-001",
				Title:    "First Story",
				Priority: 5,
				Passes:   false,
			},
			{
				ID:       "US-002",
				Title:    "Second Story",
				Priority: 10,
				Passes:   false,
			},
		},
	}

	prdData, _ := json.MarshalIndent(prdFile, "", "  ")
	prdPath := filepath.Join(tmpDir, "prd.json")
	os.WriteFile(prdPath, prdData, 0644)

	// Initial sync to create balls
	if err := syncPRDFile(prdPath, tmpDir); err != nil {
		t.Fatalf("initial sync failed: %v", err)
	}

	// Mark first ball as complete in juggler
	store, _ := session.NewStore(tmpDir)
	balls, _ := store.LoadBalls()

	var firstBall *session.Ball
	for _, ball := range balls {
		if ball.Intent == "First Story" {
			firstBall = ball
			break
		}
	}
	if firstBall == nil {
		t.Fatal("first ball not found")
	}

	firstBall.State = session.StateComplete
	if err := store.UpdateBall(firstBall); err != nil {
		t.Fatalf("failed to update ball: %v", err)
	}

	// Run write-back
	if err := writeToPRD(prdPath, tmpDir); err != nil {
		t.Fatalf("write-back failed: %v", err)
	}

	// Read updated prd.json
	updatedData, _ := os.ReadFile(prdPath)
	var updatedPRD PRDFile
	json.Unmarshal(updatedData, &updatedPRD)

	// Verify first story is now passes: true
	if !updatedPRD.UserStories[0].Passes {
		t.Errorf("expected first story passes=true after write-back, got false")
	}

	// Verify second story is still passes: false
	if updatedPRD.UserStories[1].Passes {
		t.Errorf("expected second story passes=false, got true")
	}
}

func TestWriteToPRDResearchedState(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Create .juggler directory
	jugglerDir := filepath.Join(tmpDir, ".juggler")
	if err := os.MkdirAll(jugglerDir, 0755); err != nil {
		t.Fatalf("failed to create .juggler dir: %v", err)
	}

	// Create archive directory
	archiveDir := filepath.Join(jugglerDir, "archive")
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		t.Fatalf("failed to create archive dir: %v", err)
	}

	// Create prd.json
	prdFile := PRDFile{
		Project: "TestProject",
		UserStories: []UserStory{
			{
				ID:       "US-001",
				Title:    "Research Task",
				Priority: 5,
				Passes:   false,
			},
		},
	}

	prdData, _ := json.MarshalIndent(prdFile, "", "  ")
	prdPath := filepath.Join(tmpDir, "prd.json")
	os.WriteFile(prdPath, prdData, 0644)

	// Initial sync to create ball
	if err := syncPRDFile(prdPath, tmpDir); err != nil {
		t.Fatalf("initial sync failed: %v", err)
	}

	// Mark ball as researched (completed without code changes)
	store, _ := session.NewStore(tmpDir)
	balls, _ := store.LoadBalls()

	balls[0].State = session.StateResearched
	balls[0].Output = "Research findings: ..."
	if err := store.UpdateBall(balls[0]); err != nil {
		t.Fatalf("failed to update ball: %v", err)
	}

	// Run write-back
	if err := writeToPRD(prdPath, tmpDir); err != nil {
		t.Fatalf("write-back failed: %v", err)
	}

	// Read updated prd.json - researched should map to passes: true
	updatedData, _ := os.ReadFile(prdPath)
	var updatedPRD PRDFile
	json.Unmarshal(updatedData, &updatedPRD)

	if !updatedPRD.UserStories[0].Passes {
		t.Errorf("expected researched state to map to passes=true, got false")
	}
}

func TestWriteToPRDUpdatesAcceptanceCriteria(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Create .juggler directory
	jugglerDir := filepath.Join(tmpDir, ".juggler")
	if err := os.MkdirAll(jugglerDir, 0755); err != nil {
		t.Fatalf("failed to create .juggler dir: %v", err)
	}

	// Create archive directory
	archiveDir := filepath.Join(jugglerDir, "archive")
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		t.Fatalf("failed to create archive dir: %v", err)
	}

	// Create prd.json with one AC
	prdFile := PRDFile{
		Project: "TestProject",
		UserStories: []UserStory{
			{
				ID:                 "US-001",
				Title:              "Test Story",
				AcceptanceCriteria: []string{"Original AC"},
				Priority:           5,
				Passes:             false,
			},
		},
	}

	prdData, _ := json.MarshalIndent(prdFile, "", "  ")
	prdPath := filepath.Join(tmpDir, "prd.json")
	os.WriteFile(prdPath, prdData, 0644)

	// Initial sync to create ball
	if err := syncPRDFile(prdPath, tmpDir); err != nil {
		t.Fatalf("initial sync failed: %v", err)
	}

	// Update ball with new acceptance criteria
	store, _ := session.NewStore(tmpDir)
	balls, _ := store.LoadBalls()

	balls[0].AcceptanceCriteria = []string{"New AC 1", "New AC 2"}
	if err := store.UpdateBall(balls[0]); err != nil {
		t.Fatalf("failed to update ball: %v", err)
	}

	// Run write-back
	if err := writeToPRD(prdPath, tmpDir); err != nil {
		t.Fatalf("write-back failed: %v", err)
	}

	// Read updated prd.json
	updatedData, _ := os.ReadFile(prdPath)
	var updatedPRD PRDFile
	json.Unmarshal(updatedData, &updatedPRD)

	// Verify acceptance criteria were updated
	if len(updatedPRD.UserStories[0].AcceptanceCriteria) != 2 {
		t.Errorf("expected 2 ACs after write-back, got %d", len(updatedPRD.UserStories[0].AcceptanceCriteria))
	}
	if updatedPRD.UserStories[0].AcceptanceCriteria[0] != "New AC 1" {
		t.Errorf("expected first AC to be 'New AC 1', got %s", updatedPRD.UserStories[0].AcceptanceCriteria[0])
	}
}

func TestWriteToPRDNoMatchingBall(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Create .juggler directory
	jugglerDir := filepath.Join(tmpDir, ".juggler")
	if err := os.MkdirAll(jugglerDir, 0755); err != nil {
		t.Fatalf("failed to create .juggler dir: %v", err)
	}

	// Create archive directory
	archiveDir := filepath.Join(jugglerDir, "archive")
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		t.Fatalf("failed to create archive dir: %v", err)
	}

	// Create prd.json with story that has no matching ball
	prdFile := PRDFile{
		Project: "TestProject",
		UserStories: []UserStory{
			{
				ID:       "US-001",
				Title:    "Story Without Ball",
				Priority: 5,
				Passes:   false,
			},
		},
	}

	prdData, _ := json.MarshalIndent(prdFile, "", "  ")
	prdPath := filepath.Join(tmpDir, "prd.json")
	os.WriteFile(prdPath, prdData, 0644)

	// Don't create any balls - just run write-back
	if err := writeToPRD(prdPath, tmpDir); err != nil {
		t.Fatalf("write-back failed: %v", err)
	}

	// Read prd.json - should be unchanged
	updatedData, _ := os.ReadFile(prdPath)
	var updatedPRD PRDFile
	json.Unmarshal(updatedData, &updatedPRD)

	// Verify story is still passes: false (no matching ball to change it)
	if updatedPRD.UserStories[0].Passes {
		t.Errorf("expected story without ball to remain passes=false, got true")
	}
}
