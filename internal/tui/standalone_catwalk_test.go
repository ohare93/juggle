package tui

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/knz/catwalk"
	"github.com/ohare93/juggle/internal/session"
)

// createTestStandaloneBallModel creates a StandaloneBallModel for testing.
// Uses a temp directory for the store to avoid nil pointer issues.
func createTestStandaloneBallModel(t *testing.T) StandaloneBallModel {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "juggle-tui-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	store, err := session.NewStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Create text input for title field
	ti := textinput.New()
	ti.CharLimit = 256
	ti.Width = 60
	ti.Placeholder = "What is this ball about? (50 char recommended)"
	ti.Blur()

	// Create textarea for context field
	ta := textarea.New()
	ta.Placeholder = "Background context for this task"
	ta.CharLimit = 2000
	ta.SetWidth(60)
	ta.SetHeight(1)
	ta.ShowLineNumbers = false
	ta.Focus()

	return StandaloneBallModel{
		store:               store,
		textInput:           ti,
		contextInput:        ta,
		pendingBallPriority: 1, // Default to medium
		fileAutocomplete:    NewAutocompleteState(store.ProjectDir()),
		width:               80,
		height:              24,
	}
}

// TestStandaloneBallForm tests the standalone ball creation form using catwalk.
// Run with -rewrite to update golden files.
func TestStandaloneBallForm(t *testing.T) {
	model := createTestStandaloneBallModel(t)
	catwalk.RunModel(t, "testdata/standalone_ball_form", model)
}

// TestStandaloneBallFormWithData tests the form with pre-populated data.
func TestStandaloneBallFormWithData(t *testing.T) {
	model := createTestStandaloneBallModel(t)
	model.pendingBallIntent = "Test task intent"
	model.pendingBallContext = "Some context for the task"
	model.contextInput.SetValue("Some context for the task")
	model.pendingBallTags = "feature, backend"
	model.pendingAcceptanceCriteria = []string{
		"First criterion",
		"Second criterion",
	}
	catwalk.RunModel(t, "testdata/standalone_ball_form_with_data", model)
}

// TestStandaloneBallFormNavigation tests navigating through form fields.
func TestStandaloneBallFormNavigation(t *testing.T) {
	model := createTestStandaloneBallModel(t)
	catwalk.RunModel(t, "testdata/standalone_ball_form_navigation", model)
}

// TestStandaloneBallFormLongContext tests that long context text wraps correctly.
func TestStandaloneBallFormLongContext(t *testing.T) {
	model := createTestStandaloneBallModel(t)
	longContext := "One Two Three Four Five Six Seven Eight Nine Ten Eleven Twelve"
	model.pendingBallContext = longContext
	model.contextInput.SetValue(longContext)
	// Move focus away from context field so we see the wrapped display
	model.pendingBallFormField = 1 // fieldIntent
	model.contextInput.Blur()
	model.textInput.Focus()
	catwalk.RunModel(t, "testdata/standalone_ball_form_long_context", model)
}

// TestStandaloneBallFormLongContextEditing tests long context while editing (field focused).
func TestStandaloneBallFormLongContextEditing(t *testing.T) {
	model := createTestStandaloneBallModel(t)
	longContext := "One Two Three Four Five Six Seven Eight Nine Ten Eleven Twelve"
	model.PrePopulate("Test intent", longContext, nil, "", "medium", "", nil, nil)
	catwalk.RunModel(t, "testdata/standalone_ball_form_long_context_editing", model)
}

// TestStandaloneBallFormVeryLongContext tests with even longer context (3+ lines).
func TestStandaloneBallFormVeryLongContext(t *testing.T) {
	model := createTestStandaloneBallModel(t)
	longContext := "One Two Three Four Five Six Seven Eight Nine Ten Eleven Twelve Thirteen Fourteen Fifteen Sixteen Seventeen Eighteen Nineteen Twenty TwentyOne TwentyTwo TwentyThree TwentyFour TwentyFive"
	model.PrePopulate("Test intent", longContext, nil, "", "medium", "", nil, nil)
	catwalk.RunModel(t, "testdata/standalone_ball_form_very_long_context", model)
}

// TestStandaloneBallFormRealConstructor tests using the actual NewStandaloneBallModel constructor.
func TestStandaloneBallFormRealConstructor(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "juggle-tui-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	store, err := session.NewStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	model := NewStandaloneBallModel(store, nil)
	longContext := "One Two Three Four Five Six Seven Eight Nine Ten Eleven Twelve"
	model.PrePopulate("Test intent", longContext, nil, "", "medium", "", nil, nil)
	catwalk.RunModel(t, "testdata/standalone_ball_form_real_constructor", model)
}

// TestStandaloneBallFormTypingLongContext tests typing long text into context field.
func TestStandaloneBallFormTypingLongContext(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "juggle-tui-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	store, err := session.NewStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	model := NewStandaloneBallModel(store, nil)
	catwalk.RunModel(t, "testdata/standalone_ball_form_typing_long_context", model)
}

// TestStandaloneTitlePlaceholderFromContext tests that title placeholder updates as context changes.
// AC1: Title placeholder shows first ~50 chars of context
// AC2: Placeholder updates as context changes
// AC3: Empty context shows default placeholder
func TestStandaloneTitlePlaceholderFromContext(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "juggle-tui-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	store, err := session.NewStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	model := NewStandaloneBallModel(store, nil)
	catwalk.RunModel(t, "testdata/standalone_title_placeholder_from_context", model)
}

// createTestSplitViewModel creates a Model configured for split view testing.
func createTestSplitViewModel(t *testing.T) Model {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "juggle-tui-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	store, err := session.NewStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	sessionStore, err := session.NewSessionStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create session store: %v", err)
	}

	config := &session.Config{
		SearchPaths: []string{tmpDir},
	}

	ti := textinput.New()
	ti.CharLimit = 256
	ti.Width = 60

	ta := textarea.New()
	ta.CharLimit = 2000
	ta.SetWidth(60)
	ta.SetHeight(1)

	m := Model{
		store:        store,
		sessionStore: sessionStore,
		config:       config,
		localOnly:    true,
		balls:        make([]*session.Ball, 0),
		filteredBalls: make([]*session.Ball, 0),
		sessions:     make([]*session.JuggleSession, 0),
		activePanel:  SessionsPanel,
		sessionCursor: 0,
		mode:         splitView,
		filterStates: map[string]bool{
			"pending":     true,
			"in_progress": true,
			"blocked":     true,
			"complete":    false,
		},
		textInput:            ti,
		contextInput:         ta,
		width:                80,
		height:               24,
		showPriorityColumn:   true,
		showTagsColumn:       true,
		agentStatus:          AgentStatus{},
		pendingKeySequence:   "",
		activityLog:          make([]ActivityEntry, 0),
	}

	// Set fixed time for deterministic tests
	fixedTime := time.Date(2025, 1, 13, 16, 41, 11, 0, time.UTC)
	m.nowFunc = func() time.Time {
		return fixedTime
	}

	return m
}

// TestSessionsPanelEmpty tests the sessions panel with no sessions.
func TestSessionsPanelEmpty(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.sessions = make([]*session.JuggleSession, 0) // Empty sessions list
	catwalk.RunModel(t, "testdata/sessions_panel_empty", model)
}

// TestSessionsPanelWithSessions tests the sessions panel with multiple sessions.
func TestSessionsPanelWithSessions(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.sessions = []*session.JuggleSession{
		{ID: "session-1", Description: "Backend work"},
		{ID: "session-2", Description: "Frontend tasks"},
		{ID: "session-3", Description: "DevOps improvements"},
	}
	// filterSessions() prepends pseudo-sessions, so real sessions start at index 2
	model.sessionCursor = 2
	model.selectedSession = model.sessions[0]
	catwalk.RunModel(t, "testdata/sessions_panel_with_sessions", model)
}

// TestSessionsPanelWithSelection tests the sessions panel with a selected session.
func TestSessionsPanelWithSelection(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.sessions = []*session.JuggleSession{
		{ID: "session-1", Description: "Backend work"},
		{ID: "session-2", Description: "Frontend tasks"},
		{ID: "session-3", Description: "DevOps improvements"},
	}
	// filterSessions() prepends pseudo-sessions, so second real session is at index 3
	model.sessionCursor = 3
	model.selectedSession = model.sessions[1]
	catwalk.RunModel(t, "testdata/sessions_panel_with_selection", model)
}

// TestSessionsPanelWithPseudoSessions tests rendering of pseudo-sessions (All, Untagged).
func TestSessionsPanelWithPseudoSessions(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.sessions = []*session.JuggleSession{
		{ID: PseudoSessionAll, Description: "All balls"},
		{ID: PseudoSessionUntagged, Description: "Untagged balls"},
		{ID: "session-1", Description: "Backend work"},
	}
	model.sessionCursor = 0
	model.selectedSession = model.sessions[0]
	catwalk.RunModel(t, "testdata/sessions_panel_with_pseudo_sessions", model)
}

// TestSessionsPanelWithAgentRunning tests sessions panel with an active agent indicator.
func TestSessionsPanelWithAgentRunning(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.sessions = []*session.JuggleSession{
		{ID: "session-1", Description: "Backend work"},
		{ID: "session-2", Description: "Frontend tasks"},
		{ID: "session-3", Description: "DevOps improvements"},
	}
	model.sessionCursor = 1
	model.selectedSession = model.sessions[1]
	model.agentStatus = AgentStatus{
		Running:   true,
		SessionID: "session-2",
	}
	catwalk.RunModel(t, "testdata/sessions_panel_with_agent_running", model)
}

// TestSessionSelectorEmpty tests the session selector renders correctly when empty.
func TestSessionSelectorEmpty(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.mode = sessionSelectorView
	model.sessionSelectItems = make([]*session.JuggleSession, 0)
	model.sessionSelectIndex = 0
	catwalk.RunModel(t, "testdata/session_selector_empty", model)
}

// TestSessionSelectorWithSessions tests the session selector with multiple sessions.
func TestSessionSelectorWithSessions(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.mode = sessionSelectorView
	sessions := []*session.JuggleSession{
		{ID: "session-1", Description: "Backend work"},
		{ID: "session-2", Description: "Frontend tasks"},
		{ID: "session-3", Description: "DevOps improvements"},
	}
	model.sessionSelectItems = sessions
	model.sessionSelectIndex = 0
	catwalk.RunModel(t, "testdata/session_selector_with_sessions", model)
}

// TestDependencySelectorEmpty tests the dependency selector with no available balls.
func TestDependencySelectorEmpty(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.mode = dependencySelectorView
	model.dependencySelectBalls = make([]*session.Ball, 0)
	model.dependencySelectIndex = 0
	model.dependencySelectActive = make(map[string]bool)
	catwalk.RunModel(t, "testdata/dependency_selector_empty", model)
}

// TestDependencySelectorWithBalls tests the dependency selector with multiple balls.
func TestDependencySelectorWithBalls(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.mode = dependencySelectorView
	balls := []*session.Ball{
		{ID: "juggle-1", Title: "First pending task", State: session.StatePending, Priority: session.PriorityMedium},
		{ID: "juggle-2", Title: "Second in progress task", State: session.StateInProgress, Priority: session.PriorityHigh},
		{ID: "juggle-3", Title: "Third blocked task", State: session.StateBlocked, Priority: session.PriorityLow},
	}
	model.dependencySelectBalls = balls
	model.dependencySelectIndex = 0
	model.dependencySelectActive = make(map[string]bool)
	catwalk.RunModel(t, "testdata/dependency_selector_with_balls", model)
}

// TestDependencySelectorWithSelection tests the dependency selector with some balls selected.
func TestDependencySelectorWithSelection(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.mode = dependencySelectorView
	balls := []*session.Ball{
		{ID: "juggle-1", Title: "First pending task", State: session.StatePending, Priority: session.PriorityMedium},
		{ID: "juggle-2", Title: "Second in progress task", State: session.StateInProgress, Priority: session.PriorityHigh},
		{ID: "juggle-3", Title: "Third blocked task", State: session.StateBlocked, Priority: session.PriorityLow},
	}
	selectedActive := make(map[string]bool)
	selectedActive["juggle-1"] = true
	selectedActive["juggle-3"] = true

	model.dependencySelectBalls = balls
	model.dependencySelectIndex = 1
	model.dependencySelectActive = selectedActive
	catwalk.RunModel(t, "testdata/dependency_selector_with_selection", model)
}

// TestBallsPanelEmpty tests the balls panel with no balls.
func TestBallsPanelEmpty(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.activePanel = BallsPanel
	model.balls = make([]*session.Ball, 0)
	model.filteredBalls = make([]*session.Ball, 0)
	catwalk.RunModel(t, "testdata/balls_panel_empty", model)
}

// TestBallsPanelWithBalls tests the balls panel with balls in different states.
func TestBallsPanelWithBalls(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.activePanel = BallsPanel
	model.balls = []*session.Ball{
		{
			ID:       "juggle-1",
			Title:    "First pending task",
			State:    session.StatePending,
			Priority: session.PriorityLow,
		},
		{
			ID:       "juggle-2",
			Title:    "Second in progress task",
			State:    session.StateInProgress,
			Priority: session.PriorityMedium,
		},
		{
			ID:            "juggle-3",
			Title:         "Third blocked task",
			State:         session.StateBlocked,
			BlockedReason: "Waiting for dependencies",
			Priority:      session.PriorityHigh,
		},
		{
			ID:       "juggle-4",
			Title:    "Fourth complete task",
			State:    session.StateComplete,
			Priority: session.PriorityUrgent,
		},
	}
	model.filteredBalls = model.balls
	model.cursor = 0
	model.selectedBall = model.balls[0]
	catwalk.RunModel(t, "testdata/balls_panel_with_balls", model)
}

// TestBallsPanelWithPriorityColumn tests the balls panel with priority column visible.
func TestBallsPanelWithPriorityColumn(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.activePanel = BallsPanel
	model.showPriorityColumn = true
	model.balls = []*session.Ball{
		{ID: "juggle-1", Title: "Low priority task", State: session.StatePending, Priority: session.PriorityLow},
		{ID: "juggle-2", Title: "Medium priority task", State: session.StateInProgress, Priority: session.PriorityMedium},
		{ID: "juggle-3", Title: "High priority task", State: session.StateBlocked, Priority: session.PriorityHigh},
		{ID: "juggle-4", Title: "Urgent task", State: session.StateComplete, Priority: session.PriorityUrgent},
	}
	model.filteredBalls = model.balls
	model.cursor = 1
	model.selectedBall = model.balls[1]
	catwalk.RunModel(t, "testdata/balls_panel_with_priority_column", model)
}

// TestBallsPanelWithTagsColumn tests the balls panel with tags column visible.
func TestBallsPanelWithTagsColumn(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.activePanel = BallsPanel
	model.showTagsColumn = true
	model.balls = []*session.Ball{
		{ID: "juggle-1", Title: "Tagged task one", State: session.StatePending, Tags: []string{"feature", "backend"}},
		{ID: "juggle-2", Title: "Tagged task two", State: session.StateInProgress, Tags: []string{"bugfix", "ui"}},
		{ID: "juggle-3", Title: "Task with many tags", State: session.StateBlocked, Tags: []string{"docs", "refactor", "testing"}},
	}
	model.filteredBalls = model.balls
	model.cursor = 2
	model.selectedBall = model.balls[2]
	catwalk.RunModel(t, "testdata/balls_panel_with_tags_column", model)
}

// TestBallsPanelWithTestsColumn tests the balls panel with tests column visible.
func TestBallsPanelWithTestsColumn(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.activePanel = BallsPanel
	model.showTestsColumn = true
	model.balls = []*session.Ball{
		{
			ID:                   "juggle-1",
			Title:                "Task with no tests",
			State:                session.StatePending,
			AcceptanceCriteria:   []string{},
		},
		{
			ID:                   "juggle-2",
			Title:                "Task with some tests",
			State:                session.StateInProgress,
			AcceptanceCriteria:   []string{"AC 1: Test X", "AC 2: Test Y"},
		},
		{
			ID:                   "juggle-3",
			Title:                "Task with many tests",
			State:                session.StateBlocked,
			AcceptanceCriteria:   []string{"AC 1", "AC 2", "AC 3", "AC 4", "AC 5"},
		},
	}
	model.filteredBalls = model.balls
	model.cursor = 1
	model.selectedBall = model.balls[1]
	catwalk.RunModel(t, "testdata/balls_panel_with_tests_column", model)
}

// TestBallsPanelMultipleColumns tests the balls panel with all columns visible.
func TestBallsPanelMultipleColumns(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.activePanel = BallsPanel
	model.showPriorityColumn = true
	model.showTagsColumn = true
	model.showTestsColumn = true
	model.balls = []*session.Ball{
		{
			ID:                   "juggle-1",
			Title:                "First complex task",
			State:                session.StatePending,
			Priority:             session.PriorityHigh,
			Tags:                 []string{"feature", "backend"},
			AcceptanceCriteria:   []string{"AC 1", "AC 2"},
		},
		{
			ID:                   "juggle-2",
			Title:                "Second complex task",
			State:                session.StateInProgress,
			Priority:             session.PriorityMedium,
			Tags:                 []string{"bugfix"},
			AcceptanceCriteria:   []string{"AC 1"},
		},
	}
	model.filteredBalls = model.balls
	model.cursor = 0
	model.selectedBall = model.balls[0]
	catwalk.RunModel(t, "testdata/balls_panel_with_multiple_columns", model)
}

// TestBallsPanelSortByID tests the balls panel sorted by ID ascending.
func TestBallsPanelSortByID(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.activePanel = BallsPanel
	model.sortOrder = SortByIDASC
	model.balls = []*session.Ball{
		{ID: "juggle-3", Title: "Third task", State: session.StatePending},
		{ID: "juggle-1", Title: "First task", State: session.StateInProgress},
		{ID: "juggle-2", Title: "Second task", State: session.StateBlocked},
	}
	model.filteredBalls = model.balls
	model.cursor = 0
	model.selectedBall = model.balls[0]
	catwalk.RunModel(t, "testdata/balls_panel_sort_by_id", model)
}

// TestBallsPanelSortByPriority tests the balls panel sorted by priority.
func TestBallsPanelSortByPriority(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.activePanel = BallsPanel
	model.sortOrder = SortByPriority
	model.showPriorityColumn = true
	model.balls = []*session.Ball{
		{ID: "juggle-1", Title: "Low priority", State: session.StatePending, Priority: session.PriorityLow},
		{ID: "juggle-4", Title: "Urgent task", State: session.StateInProgress, Priority: session.PriorityUrgent},
		{ID: "juggle-2", Title: "Medium priority", State: session.StateBlocked, Priority: session.PriorityMedium},
		{ID: "juggle-3", Title: "High priority", State: session.StateComplete, Priority: session.PriorityHigh},
	}
	model.filteredBalls = model.balls
	model.cursor = 0
	model.selectedBall = model.balls[0]
	catwalk.RunModel(t, "testdata/balls_panel_sort_by_priority", model)
}

// TestBallsPanelWithBlockedReason tests displaying balls with blocked reasons.
func TestBallsPanelWithBlockedReason(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.activePanel = BallsPanel
	model.balls = []*session.Ball{
		{
			ID:            "juggle-1",
			Title:         "Blocked by dependency",
			State:         session.StateBlocked,
			BlockedReason: "Waiting for juggle-2 to complete",
		},
		{
			ID:            "juggle-2",
			Title:         "Blocked by external event",
			State:         session.StateBlocked,
			BlockedReason: "Awaiting API credentials from DevOps",
		},
		{
			ID:       "juggle-3",
			Title:    "In progress task",
			State:    session.StateInProgress,
		},
	}
	model.filteredBalls = model.balls
	model.cursor = 0
	model.selectedBall = model.balls[0]
	catwalk.RunModel(t, "testdata/balls_panel_with_blocked_reason", model)
}

// TestBallsPanelWithDependencies tests displaying balls with dependency indicators.
func TestBallsPanelWithDependencies(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.activePanel = BallsPanel
	model.balls = []*session.Ball{
		{
			ID:       "juggle-1",
			Title:    "Task with no deps",
			State:    session.StatePending,
		},
		{
			ID:        "juggle-2",
			Title:     "Task with dependencies",
			State:     session.StatePending,
			DependsOn: []string{"juggle-1"},
		},
		{
			ID:        "juggle-3",
			Title:     "Task with multiple deps",
			State:     session.StateInProgress,
			DependsOn: []string{"juggle-1", "juggle-2"},
		},
	}
	model.filteredBalls = model.balls
	model.cursor = 2
	model.selectedBall = model.balls[2]
	catwalk.RunModel(t, "testdata/balls_panel_with_dependencies", model)
}

// TestBallsPanelScroll tests scrolling behavior with many balls.
func TestBallsPanelScroll(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.activePanel = BallsPanel
	// Create 20 balls to test scrolling
	for i := 1; i <= 20; i++ {
		idx := i - 1
		priority := session.PriorityLow
		if idx%4 == 1 {
			priority = session.PriorityMedium
		} else if idx%4 == 2 {
			priority = session.PriorityHigh
		} else if idx%4 == 3 {
			priority = session.PriorityUrgent
		}
		model.balls = append(model.balls, &session.Ball{
			ID:       formatBallID(i),
			Title:    formatBallTitle(i),
			State:    session.StatePending,
			Priority: priority,
		})
	}
	model.filteredBalls = model.balls
	model.cursor = 15 // Scroll to middle
	model.selectedBall = model.balls[15]
	model.showPriorityColumn = true
	catwalk.RunModel(t, "testdata/balls_panel_scroll", model)
}

// TestActivityLogViewEmpty tests the activity log view with no entries.
func TestActivityLogViewEmpty(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.activePanel = ActivityPanel
	model.bottomPaneMode = BottomPaneActivity
	model.activityLog = make([]ActivityEntry, 0)
	model.balls = append(model.balls, &session.Ball{
		ID:    "juggle-1",
		Title: "Sample ball",
		State: session.StatePending,
	})
	model.filteredBalls = model.balls
	catwalk.RunModel(t, "testdata/activity_log_view_empty", model)
}

// TestActivityLogViewWithEntries tests the activity log view with multiple entries.
func TestActivityLogViewWithEntries(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.activePanel = ActivityPanel
	model.bottomPaneMode = BottomPaneActivity

	fixedTime := time.Date(2025, 1, 13, 16, 41, 11, 0, time.UTC)
	model.activityLog = []ActivityEntry{
		{Time: fixedTime.Add(0), Message: "Balls loaded"},
		{Time: fixedTime.Add(1 * time.Second), Message: "Sessions loaded"},
		{Time: fixedTime.Add(2 * time.Second), Message: "Ball juggle-1 selected"},
		{Time: fixedTime.Add(3 * time.Second), Message: "Activity log refreshed"},
	}

	model.balls = append(model.balls, &session.Ball{
		ID:    "juggle-1",
		Title: "Sample ball",
		State: session.StatePending,
	})
	model.filteredBalls = model.balls
	catwalk.RunModel(t, "testdata/activity_log_view_with_entries", model)
}

// TestBallDetailView tests the ball detail view rendering.
func TestBallDetailView(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.activePanel = ActivityPanel
	model.bottomPaneMode = BottomPaneDetail

	model.balls = append(model.balls, &session.Ball{
		ID:        "juggle-1",
		Title:     "Fix authentication bug",
		State:     session.StateInProgress,
		Priority:  session.PriorityHigh,
		Context:   "Users are unable to login with SSO",
		Tags:      []string{"bug", "authentication"},
		DependsOn: []string{},
	})
	model.filteredBalls = model.balls
	model.cursor = 0
	model.selectedBall = model.balls[0]
	catwalk.RunModel(t, "testdata/ball_detail_view", model)
}

// TestSplitBottomPane tests the split bottom pane with activity and details side by side.
func TestSplitBottomPane(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.activePanel = BallsPanel
	model.bottomPaneMode = BottomPaneSplit

	fixedTime := time.Date(2025, 1, 13, 16, 41, 11, 0, time.UTC)
	model.activityLog = []ActivityEntry{
		{Time: fixedTime.Add(0), Message: "Balls loaded"},
		{Time: fixedTime.Add(1 * time.Second), Message: "Sessions loaded"},
	}

	model.balls = append(model.balls, &session.Ball{
		ID:        "juggle-1",
		Title:     "Add test coverage",
		State:     session.StateInProgress,
		Priority:  session.PriorityMedium,
		Context:   "Need to improve test coverage to 80%",
		Tags:      []string{"testing", "refactor"},
		DependsOn: []string{},
	})
	model.filteredBalls = model.balls
	model.cursor = 0
	model.selectedBall = model.balls[0]
	catwalk.RunModel(t, "testdata/split_bottom_pane", model)
}

// TestCyclingBottomPaneModes tests cycling through bottom pane display modes.
func TestCyclingBottomPaneModes(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.activePanel = ActivityPanel
	model.bottomPaneMode = BottomPaneActivity

	fixedTime := time.Date(2025, 1, 13, 16, 41, 11, 0, time.UTC)
	model.activityLog = []ActivityEntry{
		{Time: fixedTime.Add(0), Message: "Mode cycle test started"},
	}

	model.balls = append(model.balls, &session.Ball{
		ID:    "juggle-1",
		Title: "Test ball",
		State: session.StatePending,
	})
	model.filteredBalls = model.balls
	catwalk.RunModel(t, "testdata/cycling_bottom_pane_modes", model)
}

// TestActivityLogScrolling tests scrolling behavior in the activity log.
func TestActivityLogScrolling(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.activePanel = ActivityPanel
	model.bottomPaneMode = BottomPaneActivity

	fixedTime := time.Date(2025, 1, 13, 16, 41, 11, 0, time.UTC)
	// Create 20 activity entries to test scrolling
	for i := 0; i < 20; i++ {
		model.activityLog = append(model.activityLog, ActivityEntry{
			Time:    fixedTime.Add(time.Duration(i) * time.Second),
			Message: fmt.Sprintf("Activity entry %d", i+1),
		})
	}

	model.balls = append(model.balls, &session.Ball{
		ID:    "juggle-1",
		Title: "Test ball",
		State: session.StatePending,
	})
	model.filteredBalls = model.balls
	model.activityLogOffset = 10 // Scroll to middle
	catwalk.RunModel(t, "testdata/activity_log_scrolling", model)
}

// TestDetailViewScrolling tests scrolling behavior in the detail view.
func TestDetailViewScrolling(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.activePanel = ActivityPanel
	model.bottomPaneMode = BottomPaneDetail

	longContext := "This is a very long context that should demonstrate scrolling in the detail view panel. " +
		"It contains multiple lines of text to show how the detail view handles content that is too large " +
		"to fit in the available space. The detail panel should be able to scroll through this content " +
		"to show all the information about the selected ball. This is important for displaying comprehensive " +
		"context and acceptance criteria that may be quite lengthy."

	model.balls = append(model.balls, &session.Ball{
		ID:        "juggle-1",
		Title:     "Long context ball",
		State:     session.StateInProgress,
		Priority:  session.PriorityHigh,
		Context:   longContext,
		Tags:      []string{"feature", "documentation"},
		DependsOn: []string{},
	})
	model.filteredBalls = model.balls
	model.cursor = 0
	model.selectedBall = model.balls[0]
	model.detailScrollOffset = 5
	catwalk.RunModel(t, "testdata/detail_view_scrolling", model)
}

// TestAgentOutputPanelVisible tests the agent output panel when visible but not expanded.
func TestAgentOutputPanelVisible(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.activePanel = BallsPanel
	model.bottomPaneMode = BottomPaneActivity
	model.agentOutputVisible = true
	model.agentOutputExpanded = false

	fixedTime := time.Date(2025, 1, 13, 17, 11, 14, 0, time.UTC)
	model.agentOutput = []AgentOutputEntry{
		{Time: fixedTime, Line: "Starting agent...", IsError: false},
		{Time: fixedTime, Line: "Agent running", IsError: false},
	}

	model.balls = append(model.balls, &session.Ball{
		ID:    "juggle-1",
		Title: "Test ball",
		State: session.StatePending,
	})
	model.filteredBalls = model.balls
	catwalk.RunModel(t, "testdata/agent_output_panel_visible", model)
}

// TestAgentOutputPanelExpanded tests the agent output panel when expanded.
func TestAgentOutputPanelExpanded(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.activePanel = BallsPanel
	model.bottomPaneMode = BottomPaneActivity
	model.agentOutputVisible = true
	model.agentOutputExpanded = true

	fixedTime := time.Date(2025, 1, 13, 16, 41, 11, 0, time.UTC)
	// Create multiple lines of agent output
	for i := 0; i < 10; i++ {
		model.agentOutput = append(model.agentOutput, AgentOutputEntry{
			Time:    fixedTime.Add(time.Duration(i) * time.Second),
			Line:    fmt.Sprintf("Agent output line %d", i+1),
			IsError: i%3 == 2, // Make every third line an error
		})
	}

	model.balls = append(model.balls, &session.Ball{
		ID:    "juggle-1",
		Title: "Test ball",
		State: session.StatePending,
	})
	model.filteredBalls = model.balls
	catwalk.RunModel(t, "testdata/agent_output_panel_expanded", model)
}

// TestConfirmDeleteListView tests the delete confirmation dialog in list view.
func TestConfirmDeleteListView(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "juggle-tui-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	store, err := session.NewStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Create balls
	balls := []*session.Ball{
		{ID: "juggle-1", Title: "Important task", State: session.StatePending, Priority: session.PriorityHigh},
		{ID: "juggle-2", Title: "Another task", State: session.StateInProgress, Priority: session.PriorityMedium},
	}
	for _, ball := range balls {
		if err := store.AppendBall(ball); err != nil {
			t.Fatalf("failed to append ball: %v", err)
		}
	}

	sessionStore, err := session.NewSessionStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create session store: %v", err)
	}

	config := &session.Config{
		SearchPaths: []string{tmpDir},
	}

	ti := textinput.New()
	ti.CharLimit = 256
	ti.Width = 60

	ta := textarea.New()
	ta.CharLimit = 2000
	ta.SetWidth(60)
	ta.SetHeight(1)

	model := Model{
		store:         store,
		sessionStore:  sessionStore,
		config:        config,
		localOnly:     true,
		balls:         balls,
		filteredBalls: balls,
		sessions:      make([]*session.JuggleSession, 0),
		activePanel:   BallsPanel,
		mode:          confirmDeleteView,
		cursor:        0,
		filterStates: map[string]bool{
			"pending":     true,
			"in_progress": true,
			"blocked":     true,
			"complete":    false,
		},
		textInput:           ti,
		contextInput:        ta,
		width:               80,
		height:              24,
		showPriorityColumn:  true,
		showTagsColumn:      true,
		agentStatus:         AgentStatus{},
		pendingKeySequence:  "",
		activityLog:         make([]ActivityEntry, 0),
	}

	fixedTime := time.Date(2025, 1, 13, 16, 41, 11, 0, time.UTC)
	model.nowFunc = func() time.Time {
		return fixedTime
	}

	catwalk.RunModel(t, "testdata/confirm_delete_list_view", model)
}

// TestConfirmDeleteSplitView tests the delete confirmation dialog in split view.
func TestConfirmDeleteSplitView(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "juggle-tui-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	store, err := session.NewStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Create balls
	balls := []*session.Ball{
		{ID: "juggle-1", Title: "Critical bug fix", State: session.StateInProgress, Priority: session.PriorityHigh, Tags: []string{"bug", "urgent"}},
		{ID: "juggle-2", Title: "Feature development", State: session.StatePending, Priority: session.PriorityMedium},
	}
	for _, ball := range balls {
		if err := store.AppendBall(ball); err != nil {
			t.Fatalf("failed to append ball: %v", err)
		}
	}

	sessionStore, err := session.NewSessionStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create session store: %v", err)
	}

	config := &session.Config{
		SearchPaths: []string{tmpDir},
	}

	ti := textinput.New()
	ti.CharLimit = 256
	ti.Width = 60

	ta := textarea.New()
	ta.CharLimit = 2000
	ta.SetWidth(60)
	ta.SetHeight(1)

	model := Model{
		store:         store,
		sessionStore:  sessionStore,
		config:        config,
		localOnly:     true,
		balls:         balls,
		filteredBalls: balls,
		sessions:      make([]*session.JuggleSession, 0),
		activePanel:   BallsPanel,
		mode:          confirmDeleteView,
		cursor:        0,
		filterStates: map[string]bool{
			"pending":     true,
			"in_progress": true,
			"blocked":     true,
			"complete":    false,
		},
		textInput:           ti,
		contextInput:        ta,
		width:               80,
		height:              24,
		showPriorityColumn:  true,
		showTagsColumn:      true,
		agentStatus:         AgentStatus{},
		pendingKeySequence:  "",
		activityLog:         make([]ActivityEntry, 0),
	}

	fixedTime := time.Date(2025, 1, 13, 16, 41, 11, 0, time.UTC)
	model.nowFunc = func() time.Time {
		return fixedTime
	}

	catwalk.RunModel(t, "testdata/confirm_delete_split_view", model)
}

// TestConfirmAgentLaunchDialog tests the agent launch confirmation dialog.
func TestConfirmAgentLaunchDialog(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.mode = confirmAgentLaunch
	model.selectedSession = &session.JuggleSession{
		ID:          "session-1",
		Description: "Backend development tasks",
	}
	model.balls = []*session.Ball{
		{ID: "juggle-1", Title: "Implement API endpoint", State: session.StatePending},
		{ID: "juggle-2", Title: "Add database migration", State: session.StatePending},
		{ID: "juggle-3", Title: "Write unit tests", State: session.StatePending},
	}
	model.filteredBalls = model.balls

	catwalk.RunModel(t, "testdata/confirm_agent_launch", model)
}

// TestConfirmAgentCancelDialog tests the agent cancel confirmation dialog.
func TestConfirmAgentCancelDialog(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.mode = confirmAgentCancel
	model.agentStatus = AgentStatus{
		Running:        true,
		SessionID:      "session-1",
		Iteration:      5,
		MaxIterations:  10,
	}

	catwalk.RunModel(t, "testdata/confirm_agent_cancel", model)
}

// TestHelpViewLegacy tests the legacy help view rendering.
func TestHelpViewLegacy(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.mode = helpView
	catwalk.RunModel(t, "testdata/help_view_legacy", model)
}

// TestHelpViewSplitComprehensive tests the comprehensive split help view.
func TestHelpViewSplitComprehensive(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.mode = splitHelpView
	catwalk.RunModel(t, "testdata/help_view_split_comprehensive", model)
}

// TestHelpViewSplitScrolling tests scrolling behavior in the help view.
func TestHelpViewSplitScrolling(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.mode = splitHelpView
	model.helpScrollOffset = 10 // Scroll to middle of help content
	catwalk.RunModel(t, "testdata/help_view_split_scrolling", model)
}

// TestCatwalkStatusBarSessionsPanel tests the status bar with sessions panel active.
func TestCatwalkStatusBarSessionsPanel(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.activePanel = SessionsPanel
	model.sessions = []*session.JuggleSession{
		{ID: "session-1", Description: "Backend work"},
		{ID: "session-2", Description: "Frontend tasks"},
	}
	model.sessionCursor = 2 // First real session after pseudo-sessions
	model.selectedSession = model.sessions[0]
	catwalk.RunModel(t, "testdata/status_bar_sessions_panel", model)
}

// TestCatwalkStatusBarBallsPanel tests the status bar with balls panel active.
func TestCatwalkStatusBarBallsPanel(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.activePanel = BallsPanel
	model.balls = []*session.Ball{
		{ID: "juggle-1", Title: "First task", State: session.StatePending},
		{ID: "juggle-2", Title: "Second task", State: session.StateInProgress},
	}
	model.filteredBalls = model.balls
	model.cursor = 0
	model.selectedBall = model.balls[0]
	catwalk.RunModel(t, "testdata/status_bar_balls_panel", model)
}

// TestCatwalkStatusBarActivityPanel tests the status bar with activity panel active.
func TestCatwalkStatusBarActivityPanel(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.activePanel = ActivityPanel
	model.bottomPaneMode = BottomPaneActivity

	fixedTime := time.Date(2025, 1, 13, 16, 41, 11, 0, time.UTC)
	model.activityLog = []ActivityEntry{
		{Time: fixedTime, Message: "Balls loaded"},
		{Time: fixedTime.Add(1 * time.Second), Message: "Sessions loaded"},
	}

	model.balls = []*session.Ball{
		{ID: "juggle-1", Title: "Sample ball", State: session.StatePending},
	}
	model.filteredBalls = model.balls
	catwalk.RunModel(t, "testdata/status_bar_activity_panel", model)
}

// TestCatwalkStatusBarWithFilters tests the status bar showing filter indicators.
func TestCatwalkStatusBarWithFilters(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.activePanel = BallsPanel
	// Adjust filter states to hide some categories
	model.filterStates["complete"] = true
	model.filterStates["blocked"] = false

	model.balls = []*session.Ball{
		{ID: "juggle-1", Title: "Pending task", State: session.StatePending},
		{ID: "juggle-2", Title: "In progress task", State: session.StateInProgress},
		{ID: "juggle-3", Title: "Complete task", State: session.StateComplete},
	}
	model.filteredBalls = model.balls
	model.cursor = 0
	model.selectedBall = model.balls[0]
	catwalk.RunModel(t, "testdata/status_bar_with_filters", model)
}

// TestCatwalkStatusBarWithAgentRunning tests the status bar showing agent status.
func TestCatwalkStatusBarWithAgentRunning(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.activePanel = SessionsPanel
	model.sessions = []*session.JuggleSession{
		{ID: "session-1", Description: "Backend work"},
	}
	model.sessionCursor = 2
	model.selectedSession = model.sessions[0]
	model.agentStatus = AgentStatus{
		Running:       true,
		SessionID:     "session-1",
		Iteration:     3,
		MaxIterations: 10,
	}
	catwalk.RunModel(t, "testdata/status_bar_with_agent_running", model)
}

// TestCatwalkStatusBarWithSearchQuery tests the status bar showing active search filter.
func TestCatwalkStatusBarWithSearchQuery(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.activePanel = BallsPanel
	model.panelSearchActive = true
	model.panelSearchQuery = "backend"

	model.balls = []*session.Ball{
		{ID: "juggle-1", Title: "Backend API task", State: session.StatePending},
		{ID: "juggle-2", Title: "Frontend task", State: session.StateInProgress},
	}
	model.filteredBalls = model.balls[:1] // Only first ball matches filter
	model.cursor = 0
	model.selectedBall = model.balls[0]
	catwalk.RunModel(t, "testdata/status_bar_with_search_query", model)
}

// TestInputSessionViewAdd tests the session name input dialog for adding a new session.
func TestInputSessionViewAdd(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.mode = inputSessionView
	model.inputAction = actionAdd
	model.textInput.SetValue("")
	model.textInput.Focus()
	catwalk.RunModel(t, "testdata/input_session_add", model)
}

// TestInputSessionViewEdit tests the session name input dialog for editing a session.
func TestInputSessionViewEdit(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.mode = inputSessionView
	model.inputAction = actionEdit
	model.sessions = []*session.JuggleSession{
		{ID: "backend-work", Description: "Backend development tasks"},
		{ID: "frontend", Description: "Frontend improvements"},
	}
	model.sessionCursor = 0
	model.editingSession = model.sessions[0]
	model.textInput.SetValue("Updated description for backend")
	model.textInput.Focus()
	catwalk.RunModel(t, "testdata/input_session_edit", model)
}

// TestInputSessionViewEditFiltered tests editing a session that was found via filtering.
func TestInputSessionViewEditFiltered(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.mode = inputSessionView
	model.inputAction = actionEdit
	model.sessions = []*session.JuggleSession{
		{ID: "backend-work", Description: "Backend development tasks"},
		{ID: "frontend-auth", Description: "Frontend authentication"},
		{ID: "frontend-ui", Description: "Frontend UI improvements"},
	}
	// Simulate filtering - only the frontend-auth session is visible
	model.panelSearchActive = true
	model.panelSearchQuery = "auth"
	// sessionCursor points to the filtered result (which is sessions[1])
	model.sessionCursor = 1
	model.editingSession = model.sessions[1]
	model.textInput.SetValue("Updated auth description")
	model.textInput.Focus()
	catwalk.RunModel(t, "testdata/input_session_edit_filtered", model)
}

// TestInputBallViewAdd tests the ball title input dialog for adding a new ball.
func TestInputBallViewAdd(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.mode = inputBallView
	model.inputAction = actionAdd
	model.selectedSession = &session.JuggleSession{
		ID:          "backend-work",
		Description: "Backend development tasks",
	}
	model.textInput.SetValue("")
	model.textInput.Focus()
	catwalk.RunModel(t, "testdata/input_ball_add", model)
}

// TestInputBallViewEdit tests the ball title input dialog for editing a ball.
func TestInputBallViewEdit(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.mode = inputBallView
	model.inputAction = actionEdit
	model.editingBall = &session.Ball{
		ID:       "juggle-5",
		Title:    "Original title for the ball",
		State:    session.StatePending,
		Priority: session.PriorityHigh,
	}
	model.textInput.SetValue("Updated title for the ball")
	model.textInput.Focus()
	catwalk.RunModel(t, "testdata/input_ball_edit", model)
}

// TestInputBlockedView tests the blocked reason input dialog.
func TestInputBlockedView(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.mode = inputBlockedView
	model.editingBall = &session.Ball{
		ID:       "juggle-7",
		Title:    "Task that needs to be blocked",
		State:    session.StateInProgress,
		Priority: session.PriorityMedium,
	}
	model.textInput.SetValue("")
	model.textInput.Focus()
	catwalk.RunModel(t, "testdata/input_blocked_reason", model)
}

// TestInputBlockedViewWithReason tests the blocked reason input with partial input.
func TestInputBlockedViewWithReason(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.mode = inputBlockedView
	model.editingBall = &session.Ball{
		ID:       "juggle-7",
		Title:    "Task that needs to be blocked",
		State:    session.StateInProgress,
		Priority: session.PriorityMedium,
	}
	model.textInput.SetValue("Waiting for API credentials")
	model.textInput.Focus()
	catwalk.RunModel(t, "testdata/input_blocked_reason_with_text", model)
}

// TestPanelSearchViewSessions tests the panel search dialog for sessions.
func TestPanelSearchViewSessions(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.mode = panelSearchView
	model.activePanel = SessionsPanel
	model.textInput.SetValue("")
	model.textInput.Focus()
	catwalk.RunModel(t, "testdata/panel_search_sessions", model)
}

// TestPanelSearchViewBalls tests the panel search dialog for balls.
func TestPanelSearchViewBalls(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.mode = panelSearchView
	model.activePanel = BallsPanel
	model.textInput.SetValue("")
	model.textInput.Focus()
	catwalk.RunModel(t, "testdata/panel_search_balls", model)
}

// TestPanelSearchViewWithQuery tests the panel search dialog with existing query.
func TestPanelSearchViewWithQuery(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.mode = panelSearchView
	model.activePanel = BallsPanel
	model.panelSearchQuery = "backend"
	model.textInput.SetValue("api")
	model.textInput.Focus()
	catwalk.RunModel(t, "testdata/panel_search_with_query", model)
}

// TestInputAcceptanceCriteriaViewEmpty tests the acceptance criteria input dialog when empty.
func TestInputAcceptanceCriteriaViewEmpty(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.mode = inputAcceptanceCriteriaView
	model.pendingBallIntent = "Implement user authentication"
	model.pendingAcceptanceCriteria = []string{}
	model.textInput.SetValue("")
	model.textInput.Focus()
	catwalk.RunModel(t, "testdata/input_ac_empty", model)
}

// TestInputAcceptanceCriteriaViewWithEntries tests the AC input with existing entries.
func TestInputAcceptanceCriteriaViewWithEntries(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.mode = inputAcceptanceCriteriaView
	model.pendingBallIntent = "Implement user authentication"
	model.pendingAcceptanceCriteria = []string{
		"User can login with email and password",
		"Password reset flow works correctly",
		"Session tokens expire after 24 hours",
	}
	model.textInput.SetValue("")
	model.textInput.Focus()
	catwalk.RunModel(t, "testdata/input_ac_with_entries", model)
}

// TestInputAcceptanceCriteriaViewTyping tests the AC input while typing a new criterion.
func TestInputAcceptanceCriteriaViewTyping(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.mode = inputAcceptanceCriteriaView
	model.pendingBallIntent = "Implement user authentication"
	model.pendingAcceptanceCriteria = []string{
		"User can login with email and password",
	}
	model.textInput.SetValue("OAuth integration supports Google")
	model.textInput.Focus()
	catwalk.RunModel(t, "testdata/input_ac_typing", model)
}

// TestInputTagViewEmpty tests the tag input dialog with no existing tags.
func TestInputTagViewEmpty(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.mode = inputTagView
	model.editingBall = &session.Ball{
		ID:       "juggle-3",
		Title:    "Task without tags",
		State:    session.StatePending,
		Tags:     []string{},
	}
	model.textInput.SetValue("")
	model.textInput.Focus()
	catwalk.RunModel(t, "testdata/input_tag_empty", model)
}

// TestInputTagViewWithTags tests the tag input dialog with existing tags.
func TestInputTagViewWithTags(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.mode = inputTagView
	model.editingBall = &session.Ball{
		ID:       "juggle-3",
		Title:    "Task with multiple tags",
		State:    session.StatePending,
		Tags:     []string{"backend", "api", "authentication"},
	}
	model.textInput.SetValue("")
	model.textInput.Focus()
	catwalk.RunModel(t, "testdata/input_tag_with_tags", model)
}

// TestInputTagViewTyping tests the tag input dialog while typing a new tag.
func TestInputTagViewTyping(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.mode = inputTagView
	model.editingBall = &session.Ball{
		ID:       "juggle-3",
		Title:    "Task with some tags",
		State:    session.StatePending,
		Tags:     []string{"backend", "api"},
	}
	model.textInput.SetValue("security")
	model.textInput.Focus()
	catwalk.RunModel(t, "testdata/input_tag_typing", model)
}

// TestInputTagViewRemove tests the tag input dialog showing remove syntax.
func TestInputTagViewRemove(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.mode = inputTagView
	model.editingBall = &session.Ball{
		ID:       "juggle-3",
		Title:    "Task with tags to remove",
		State:    session.StatePending,
		Tags:     []string{"backend", "obsolete", "api"},
	}
	model.textInput.SetValue("-obsolete")
	model.textInput.Focus()
	catwalk.RunModel(t, "testdata/input_tag_remove", model)
}

// TestHistoryViewEmpty tests the history view with no agent run history.
func TestHistoryViewEmpty(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.mode = historyView
	model.agentHistory = make([]*session.AgentRunRecord, 0)
	model.historyCursor = 0
	model.historyScrollOffset = 0
	catwalk.RunModel(t, "testdata/history_view_empty", model)
}

// TestHistoryViewWithEntries tests the history view with multiple agent run records.
func TestHistoryViewWithEntries(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.mode = historyView

	fixedTime := time.Date(2025, 1, 13, 16, 41, 11, 0, time.UTC)
	model.agentHistory = []*session.AgentRunRecord{
		{
			ID:            "1736782871000000000",
			SessionID:     "backend-work",
			StartedAt:     fixedTime.Add(-2 * time.Hour),
			EndedAt:       fixedTime.Add(-2*time.Hour + 15*time.Minute),
			Iterations:    10,
			MaxIterations: 10,
			Result:        "complete",
			BallsComplete: 3,
			BallsBlocked:  0,
			BallsTotal:    3,
		},
		{
			ID:            "1736779271000000000",
			SessionID:     "frontend-tasks",
			StartedAt:     fixedTime.Add(-1 * time.Hour),
			EndedAt:       fixedTime.Add(-1*time.Hour + 8*time.Minute),
			Iterations:    5,
			MaxIterations: 10,
			Result:        "blocked",
			BlockedReason: "Missing API credentials",
			BallsComplete: 2,
			BallsBlocked:  1,
			BallsTotal:    3,
		},
		{
			ID:            "1736775671000000000",
			SessionID:     "devops",
			StartedAt:     fixedTime.Add(-30 * time.Minute),
			EndedAt:       fixedTime.Add(-30*time.Minute + 3*time.Minute),
			Iterations:    10,
			MaxIterations: 10,
			Result:        "max_iterations",
			BallsComplete: 1,
			BallsBlocked:  0,
			BallsTotal:    5,
		},
	}
	model.historyCursor = 0
	model.historyScrollOffset = 0
	catwalk.RunModel(t, "testdata/history_view_with_entries", model)
}

// TestHistoryViewSelectedDetails tests the history view with a selected entry showing details.
func TestHistoryViewSelectedDetails(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.mode = historyView

	fixedTime := time.Date(2025, 1, 13, 16, 41, 11, 0, time.UTC)
	model.agentHistory = []*session.AgentRunRecord{
		{
			ID:            "1736782871000000000",
			SessionID:     "backend-work",
			StartedAt:     fixedTime.Add(-2 * time.Hour),
			EndedAt:       fixedTime.Add(-2*time.Hour + 15*time.Minute),
			Iterations:    10,
			MaxIterations: 10,
			Result:        "complete",
			BallsComplete: 3,
			BallsBlocked:  0,
			BallsTotal:    3,
			OutputFile:    "/tmp/juggle/backend-work/last_output.txt",
		},
		{
			ID:             "1736779271000000000",
			SessionID:      "frontend-tasks",
			StartedAt:      fixedTime.Add(-1 * time.Hour),
			EndedAt:        fixedTime.Add(-1*time.Hour + 8*time.Minute),
			Iterations:     5,
			MaxIterations:  10,
			Result:         "blocked",
			BlockedReason:  "Missing API credentials from DevOps team",
			BallsComplete:  2,
			BallsBlocked:   1,
			BallsTotal:     3,
			TotalWaitTime:  30 * time.Second,
			OutputFile:     "/tmp/juggle/frontend-tasks/last_output.txt",
		},
		{
			ID:            "1736775671000000000",
			SessionID:     "devops",
			StartedAt:     fixedTime.Add(-30 * time.Minute),
			EndedAt:       fixedTime.Add(-30*time.Minute + 3*time.Minute),
			Iterations:    10,
			MaxIterations: 10,
			Result:        "error",
			ErrorMessage:  "Failed to connect to database",
			BallsComplete: 0,
			BallsBlocked:  0,
			BallsTotal:    2,
		},
	}
	// Select the second entry (blocked one) to show detailed info
	model.historyCursor = 1
	model.historyScrollOffset = 0
	catwalk.RunModel(t, "testdata/history_view_selected_details", model)
}

// TestHistoryOutputView tests the output viewer with content.
func TestHistoryOutputView(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.mode = historyOutputView

	fixedTime := time.Date(2025, 1, 13, 16, 41, 11, 0, time.UTC)
	model.agentHistory = []*session.AgentRunRecord{
		{
			ID:            "1736782871000000000",
			SessionID:     "backend-work",
			StartedAt:     fixedTime.Add(-2 * time.Hour),
			EndedAt:       fixedTime.Add(-2*time.Hour + 15*time.Minute),
			Iterations:    10,
			MaxIterations: 10,
			Result:        "complete",
			BallsComplete: 3,
			BallsTotal:    3,
			OutputFile:    "/tmp/juggle/backend-work/last_output.txt",
		},
	}
	model.historyCursor = 0
	model.historyOutputOffset = 0
	model.historyOutput = `Starting agent loop for session: backend-work
Iteration 1/10: Processing ball juggle-1
Ball juggle-1 completed successfully
Iteration 2/10: Processing ball juggle-2
Ball juggle-2 completed successfully
Iteration 3/10: Processing ball juggle-3
Ball juggle-3 completed successfully
All balls complete. Exiting agent loop.
Total iterations: 3
Total time: 15m0s`

	catwalk.RunModel(t, "testdata/history_output_view", model)
}

// TestHistoryOutputViewScrolling tests scrolling behavior in the output viewer.
func TestHistoryOutputViewScrolling(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.mode = historyOutputView

	fixedTime := time.Date(2025, 1, 13, 16, 41, 11, 0, time.UTC)
	model.agentHistory = []*session.AgentRunRecord{
		{
			ID:            "1736782871000000000",
			SessionID:     "backend-work",
			StartedAt:     fixedTime,
			EndedAt:       fixedTime.Add(30 * time.Minute),
			Iterations:    10,
			MaxIterations: 10,
			Result:        "complete",
			BallsComplete: 10,
			BallsTotal:    10,
			OutputFile:    "/tmp/juggle/backend-work/last_output.txt",
		},
	}
	model.historyCursor = 0

	// Create output content with many lines to test scrolling
	var lines []string
	lines = append(lines, "Starting agent loop for session: backend-work")
	for i := 1; i <= 25; i++ {
		lines = append(lines, fmt.Sprintf("Iteration %d/10: Processing ball juggle-%d", i, i))
		lines = append(lines, fmt.Sprintf("Ball juggle-%d completed successfully", i))
	}
	lines = append(lines, "All balls complete. Exiting agent loop.")
	lines = append(lines, "Total iterations: 10")
	lines = append(lines, "Total time: 30m0s")

	model.historyOutput = strings.Join(lines, "\n")
	model.historyOutputOffset = 10 // Scroll to middle of content
	catwalk.RunModel(t, "testdata/history_output_view_scrolling", model)
}

// TestBallFormPriorityCycling tests priority field cycling with left/right arrow keys in the legacy ball form.
func TestBallFormPriorityCycling(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.mode = inputBallFormView
	model.pendingBallIntent = "Test task for priority cycling"
	model.pendingBallFormField = 0 // priority field
	model.pendingBallPriority = 1  // medium
	model.pendingBallSession = 0   // none
	model.pendingBallTags = ""
	catwalk.RunModel(t, "testdata/ball_form_priority_cycling", model)
}

// TestBallFormModelSizeCycling tests model size field cycling with left/right arrow keys.
func TestBallFormModelSizeCycling(t *testing.T) {
	model := createTestStandaloneBallModel(t)
	// Navigate to model size field (field 5: context=0, intent=1, AC=2, tags=3, session=4, modelSize=5)
	model.pendingBallFormField = 5
	model.pendingBallModelSize = 0 // default
	catwalk.RunModel(t, "testdata/ball_form_model_size_cycling", model)
}

// TestBallFormSessionSelection tests session selection with left/right arrow keys.
func TestBallFormSessionSelection(t *testing.T) {
	model := createTestStandaloneBallModel(t)
	// Add some sessions for selection
	model.sessions = []*session.JuggleSession{
		{ID: "backend-work", Description: "Backend development tasks"},
		{ID: "frontend-tasks", Description: "Frontend improvements"},
		{ID: "devops", Description: "DevOps work"},
	}
	// Navigate to session field (field 4: context=0, intent=1, AC=2, tags=3, session=4)
	model.pendingBallFormField = 4
	model.pendingBallSession = 0 // none
	catwalk.RunModel(t, "testdata/ball_form_session_selection", model)
}

// TestBallFormDependencySelector tests the dependency selection dropdown.
func TestBallFormDependencySelector(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "juggle-tui-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	store, err := session.NewStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Create some balls that can be selected as dependencies
	balls := []*session.Ball{
		{ID: "juggle-1", Title: "First pending task", State: session.StatePending, Priority: session.PriorityMedium},
		{ID: "juggle-2", Title: "Second in progress task", State: session.StateInProgress, Priority: session.PriorityHigh},
		{ID: "juggle-3", Title: "Third blocked task", State: session.StateBlocked, Priority: session.PriorityLow},
	}
	for _, ball := range balls {
		if err := store.AppendBall(ball); err != nil {
			t.Fatalf("failed to append ball: %v", err)
		}
	}

	ti := textinput.New()
	ti.CharLimit = 256
	ti.Width = 60
	ti.Placeholder = "What is this ball about? (50 char recommended)"
	ti.Blur()

	ta := textarea.New()
	ta.Placeholder = "Background context for this task"
	ta.CharLimit = 2000
	ta.SetWidth(60)
	ta.SetHeight(1)
	ta.ShowLineNumbers = false
	ta.Focus()

	model := StandaloneBallModel{
		store:                  store,
		textInput:              ti,
		contextInput:           ta,
		pendingBallPriority:    1,
		fileAutocomplete:       NewAutocompleteState(store.ProjectDir()),
		width:                  80,
		height:                 24,
		inDependencySelector:   true,
		dependencySelectBalls:  balls,
		dependencySelectIndex:  0,
		dependencySelectActive: make(map[string]bool),
	}
	catwalk.RunModel(t, "testdata/ball_form_dependency_selector", model)
}

// TestBallFormDependencySelection tests selecting dependencies in the selector.
func TestBallFormDependencySelection(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "juggle-tui-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	store, err := session.NewStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	balls := []*session.Ball{
		{ID: "juggle-1", Title: "First pending task", State: session.StatePending, Priority: session.PriorityMedium},
		{ID: "juggle-2", Title: "Second in progress task", State: session.StateInProgress, Priority: session.PriorityHigh},
		{ID: "juggle-3", Title: "Third blocked task", State: session.StateBlocked, Priority: session.PriorityLow},
	}
	for _, ball := range balls {
		if err := store.AppendBall(ball); err != nil {
			t.Fatalf("failed to append ball: %v", err)
		}
	}

	ti := textinput.New()
	ti.CharLimit = 256
	ti.Width = 60
	ti.Blur()

	ta := textarea.New()
	ta.CharLimit = 2000
	ta.SetWidth(60)
	ta.SetHeight(1)
	ta.ShowLineNumbers = false
	ta.Focus()

	// Pre-select some dependencies
	selectedActive := make(map[string]bool)
	selectedActive["juggle-1"] = true

	model := StandaloneBallModel{
		store:                  store,
		textInput:              ti,
		contextInput:           ta,
		pendingBallPriority:    1,
		fileAutocomplete:       NewAutocompleteState(store.ProjectDir()),
		width:                  80,
		height:                 24,
		inDependencySelector:   true,
		dependencySelectBalls:  balls,
		dependencySelectIndex:  1, // cursor on second item
		dependencySelectActive: selectedActive,
	}
	catwalk.RunModel(t, "testdata/ball_form_dependency_selection", model)
}

// =============================================================================
// EDGE CASE TESTS
// =============================================================================

// createNarrowTerminalModel creates a Model with narrow terminal width (40 cols).
func createNarrowTerminalModel(t *testing.T) Model {
	t.Helper()
	model := createTestSplitViewModel(t)
	model.width = 40
	model.height = 24
	return model
}

// createWideTerminalModel creates a Model with wide terminal width (200 cols).
func createWideTerminalModel(t *testing.T) Model {
	t.Helper()
	model := createTestSplitViewModel(t)
	model.width = 200
	model.height = 24
	return model
}

// createShortTerminalModel creates a Model with short terminal height (10 rows).
func createShortTerminalModel(t *testing.T) Model {
	t.Helper()
	model := createTestSplitViewModel(t)
	model.width = 80
	model.height = 10
	return model
}

// TestEdgeCaseNarrowTerminal tests rendering with a narrow terminal (40 cols).
func TestEdgeCaseNarrowTerminal(t *testing.T) {
	model := createNarrowTerminalModel(t)
	model.activePanel = BallsPanel
	model.balls = []*session.Ball{
		{ID: "juggle-1", Title: "A task with a fairly long title", State: session.StatePending, Priority: session.PriorityMedium},
		{ID: "juggle-2", Title: "Another task here", State: session.StateInProgress, Priority: session.PriorityHigh},
	}
	model.filteredBalls = model.balls
	model.cursor = 0
	model.selectedBall = model.balls[0]
	model.showPriorityColumn = true
	catwalk.RunModel(t, "testdata/edge_case_narrow_terminal", model)
}

// TestEdgeCaseNarrowTerminalWithSessions tests the sessions panel at narrow width.
func TestEdgeCaseNarrowTerminalWithSessions(t *testing.T) {
	model := createNarrowTerminalModel(t)
	model.activePanel = SessionsPanel
	model.sessions = []*session.JuggleSession{
		{ID: "session-with-very-long-name", Description: "A session with a long description"},
		{ID: "short", Description: "Short desc"},
	}
	model.sessionCursor = 2
	model.selectedSession = model.sessions[0]
	catwalk.RunModel(t, "testdata/edge_case_narrow_sessions", model)
}

// TestEdgeCaseWideTerminal tests rendering with a wide terminal (200 cols).
func TestEdgeCaseWideTerminal(t *testing.T) {
	model := createWideTerminalModel(t)
	model.activePanel = BallsPanel
	model.balls = []*session.Ball{
		{ID: "juggle-1", Title: "First task", State: session.StatePending, Priority: session.PriorityMedium, Tags: []string{"feature", "backend"}},
		{ID: "juggle-2", Title: "Second task", State: session.StateInProgress, Priority: session.PriorityHigh, Tags: []string{"bug"}},
	}
	model.filteredBalls = model.balls
	model.cursor = 0
	model.selectedBall = model.balls[0]
	model.showPriorityColumn = true
	model.showTagsColumn = true
	model.showTestsColumn = true
	catwalk.RunModel(t, "testdata/edge_case_wide_terminal", model)
}

// TestEdgeCaseWideTerminalWithAllColumns tests wide terminal with all columns visible.
func TestEdgeCaseWideTerminalWithAllColumns(t *testing.T) {
	model := createWideTerminalModel(t)
	model.activePanel = BallsPanel
	model.balls = []*session.Ball{
		{
			ID:                 "juggle-1",
			Title:              "A comprehensive task with many details",
			State:              session.StatePending,
			Priority:           session.PriorityUrgent,
			Tags:               []string{"feature", "backend", "api", "testing"},
			AcceptanceCriteria: []string{"AC 1", "AC 2", "AC 3", "AC 4"},
		},
		{
			ID:                 "juggle-2",
			Title:              "Another detailed task",
			State:              session.StateInProgress,
			Priority:           session.PriorityHigh,
			Tags:               []string{"refactor"},
			AcceptanceCriteria: []string{"AC 1"},
		},
	}
	model.filteredBalls = model.balls
	model.cursor = 0
	model.selectedBall = model.balls[0]
	model.showPriorityColumn = true
	model.showTagsColumn = true
	model.showTestsColumn = true
	catwalk.RunModel(t, "testdata/edge_case_wide_all_columns", model)
}

// TestEdgeCaseShortTerminal tests rendering with a short terminal (10 rows).
func TestEdgeCaseShortTerminal(t *testing.T) {
	model := createShortTerminalModel(t)
	model.activePanel = BallsPanel
	model.balls = []*session.Ball{
		{ID: "juggle-1", Title: "First task", State: session.StatePending},
		{ID: "juggle-2", Title: "Second task", State: session.StateInProgress},
		{ID: "juggle-3", Title: "Third task", State: session.StateBlocked},
	}
	model.filteredBalls = model.balls
	model.cursor = 0
	model.selectedBall = model.balls[0]
	catwalk.RunModel(t, "testdata/edge_case_short_terminal", model)
}

// TestEdgeCaseShortTerminalWithManyBalls tests short terminal with many balls to scroll.
func TestEdgeCaseShortTerminalWithManyBalls(t *testing.T) {
	model := createShortTerminalModel(t)
	model.activePanel = BallsPanel
	// Create 15 balls to test scrolling in short terminal
	for i := 1; i <= 15; i++ {
		model.balls = append(model.balls, &session.Ball{
			ID:    formatBallID(i),
			Title: formatBallTitle(i),
			State: session.StatePending,
		})
	}
	model.filteredBalls = model.balls
	model.cursor = 7 // Scroll to middle
	model.selectedBall = model.balls[7]
	catwalk.RunModel(t, "testdata/edge_case_short_many_balls", model)
}

// TestEdgeCaseLongTitleTruncation tests truncation of very long ball titles.
func TestEdgeCaseLongTitleTruncation(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.activePanel = BallsPanel
	model.balls = []*session.Ball{
		{
			ID:    "juggle-1",
			Title: "This is an extremely long title that should definitely be truncated because it exceeds the available space in the balls panel",
			State: session.StatePending,
		},
		{
			ID:    "juggle-2",
			Title: "Another very long title that goes on and on and on and should also be truncated to fit the display",
			State: session.StateInProgress,
		},
		{ID: "juggle-3", Title: "Short title", State: session.StateBlocked},
	}
	model.filteredBalls = model.balls
	model.cursor = 0
	model.selectedBall = model.balls[0]
	catwalk.RunModel(t, "testdata/edge_case_long_title_truncation", model)
}

// TestEdgeCaseLongTitleInNarrowTerminal tests long titles in narrow terminal.
func TestEdgeCaseLongTitleInNarrowTerminal(t *testing.T) {
	model := createNarrowTerminalModel(t)
	model.activePanel = BallsPanel
	model.balls = []*session.Ball{
		{
			ID:    "juggle-1",
			Title: "This is a long title that must be truncated in narrow view",
			State: session.StatePending,
		},
		{ID: "juggle-2", Title: "Short", State: session.StateInProgress},
	}
	model.filteredBalls = model.balls
	model.cursor = 0
	model.selectedBall = model.balls[0]
	catwalk.RunModel(t, "testdata/edge_case_long_title_narrow", model)
}

// TestEdgeCaseEmptyBallsList tests empty balls list display.
func TestEdgeCaseEmptyBallsList(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.activePanel = BallsPanel
	model.balls = make([]*session.Ball, 0)
	model.filteredBalls = make([]*session.Ball, 0)
	model.selectedBall = nil
	catwalk.RunModel(t, "testdata/edge_case_empty_balls", model)
}

// TestEdgeCaseEmptySessionsList tests empty sessions list display.
func TestEdgeCaseEmptySessionsList(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.activePanel = SessionsPanel
	model.sessions = make([]*session.JuggleSession, 0)
	model.selectedSession = nil
	model.sessionCursor = 0
	catwalk.RunModel(t, "testdata/edge_case_empty_sessions", model)
}

// TestEdgeCaseEmptyActivityLog tests empty activity log display.
func TestEdgeCaseEmptyActivityLog(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.activePanel = ActivityPanel
	model.bottomPaneMode = BottomPaneActivity
	model.activityLog = make([]ActivityEntry, 0)
	catwalk.RunModel(t, "testdata/edge_case_empty_activity", model)
}

// TestEdgeCaseEmptyHistory tests empty history view display.
func TestEdgeCaseEmptyHistory(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.mode = historyView
	model.agentHistory = make([]*session.AgentRunRecord, 0)
	model.historyCursor = 0
	catwalk.RunModel(t, "testdata/edge_case_empty_history", model)
}

// TestEdgeCaseEmptyDependencySelector tests empty dependency selector.
func TestEdgeCaseEmptyDependencySelector(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.mode = dependencySelectorView
	model.dependencySelectBalls = make([]*session.Ball, 0)
	model.dependencySelectIndex = 0
	model.dependencySelectActive = make(map[string]bool)
	catwalk.RunModel(t, "testdata/edge_case_empty_dependency", model)
}

// TestEdgeCaseMaxScrollBallsAtBottom tests scroll position at the bottom of balls list.
func TestEdgeCaseMaxScrollBallsAtBottom(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.activePanel = BallsPanel
	// Create 30 balls to ensure we need to scroll
	for i := 1; i <= 30; i++ {
		model.balls = append(model.balls, &session.Ball{
			ID:    formatBallID(i),
			Title: formatBallTitle(i),
			State: session.StatePending,
		})
	}
	model.filteredBalls = model.balls
	model.cursor = 29 // Last ball
	model.selectedBall = model.balls[29]
	catwalk.RunModel(t, "testdata/edge_case_max_scroll_balls_bottom", model)
}

// TestEdgeCaseMaxScrollBallsAtTop tests scroll position at the top of balls list.
func TestEdgeCaseMaxScrollBallsAtTop(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.activePanel = BallsPanel
	// Create 30 balls to ensure we need to scroll
	for i := 1; i <= 30; i++ {
		model.balls = append(model.balls, &session.Ball{
			ID:    formatBallID(i),
			Title: formatBallTitle(i),
			State: session.StatePending,
		})
	}
	model.filteredBalls = model.balls
	model.cursor = 0 // First ball
	model.selectedBall = model.balls[0]
	catwalk.RunModel(t, "testdata/edge_case_max_scroll_balls_top", model)
}

// TestEdgeCaseMaxScrollActivityAtBottom tests scroll position at bottom of activity log.
func TestEdgeCaseMaxScrollActivityAtBottom(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.activePanel = ActivityPanel
	model.bottomPaneMode = BottomPaneActivity

	fixedTime := time.Date(2025, 1, 13, 16, 41, 11, 0, time.UTC)
	// Create 50 activity entries
	for i := 0; i < 50; i++ {
		model.activityLog = append(model.activityLog, ActivityEntry{
			Time:    fixedTime.Add(time.Duration(i) * time.Second),
			Message: fmt.Sprintf("Activity entry number %d", i+1),
		})
	}
	// Scroll to bottom
	model.activityLogOffset = 45 // Near the end
	catwalk.RunModel(t, "testdata/edge_case_max_scroll_activity_bottom", model)
}

// TestEdgeCaseMaxScrollActivityAtTop tests scroll position at top of activity log.
func TestEdgeCaseMaxScrollActivityAtTop(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.activePanel = ActivityPanel
	model.bottomPaneMode = BottomPaneActivity

	fixedTime := time.Date(2025, 1, 13, 16, 41, 11, 0, time.UTC)
	// Create 50 activity entries
	for i := 0; i < 50; i++ {
		model.activityLog = append(model.activityLog, ActivityEntry{
			Time:    fixedTime.Add(time.Duration(i) * time.Second),
			Message: fmt.Sprintf("Activity entry number %d", i+1),
		})
	}
	model.activityLogOffset = 0 // At the top
	catwalk.RunModel(t, "testdata/edge_case_max_scroll_activity_top", model)
}

// TestEdgeCaseMaxScrollHistoryAtBottom tests scroll position at bottom of history.
func TestEdgeCaseMaxScrollHistoryAtBottom(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.mode = historyView

	fixedTime := time.Date(2025, 1, 13, 16, 41, 11, 0, time.UTC)
	// Create 20 history entries
	for i := 0; i < 20; i++ {
		model.agentHistory = append(model.agentHistory, &session.AgentRunRecord{
			ID:            fmt.Sprintf("17367828%d1000000000", i),
			SessionID:     fmt.Sprintf("session-%d", i+1),
			StartedAt:     fixedTime.Add(time.Duration(-i) * time.Hour),
			EndedAt:       fixedTime.Add(time.Duration(-i)*time.Hour + 15*time.Minute),
			Iterations:    i + 1,
			MaxIterations: 10,
			Result:        "complete",
			BallsComplete: i + 1,
			BallsTotal:    i + 2,
		})
	}
	model.historyCursor = 19 // Last entry
	model.historyScrollOffset = 15
	catwalk.RunModel(t, "testdata/edge_case_max_scroll_history_bottom", model)
}

// TestEdgeCaseCursorAtFirstBall tests cursor behavior when at first ball.
func TestEdgeCaseCursorAtFirstBall(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.activePanel = BallsPanel
	model.balls = []*session.Ball{
		{ID: "juggle-1", Title: "First ball", State: session.StatePending},
		{ID: "juggle-2", Title: "Second ball", State: session.StateInProgress},
		{ID: "juggle-3", Title: "Third ball", State: session.StateBlocked},
	}
	model.filteredBalls = model.balls
	model.cursor = 0 // At the top
	model.selectedBall = model.balls[0]
	catwalk.RunModel(t, "testdata/edge_case_cursor_first_ball", model)
}

// TestEdgeCaseCursorAtLastBall tests cursor behavior when at last ball.
func TestEdgeCaseCursorAtLastBall(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.activePanel = BallsPanel
	model.balls = []*session.Ball{
		{ID: "juggle-1", Title: "First ball", State: session.StatePending},
		{ID: "juggle-2", Title: "Second ball", State: session.StateInProgress},
		{ID: "juggle-3", Title: "Third ball", State: session.StateBlocked},
	}
	model.filteredBalls = model.balls
	model.cursor = 2 // At the end
	model.selectedBall = model.balls[2]
	catwalk.RunModel(t, "testdata/edge_case_cursor_last_ball", model)
}

// TestEdgeCaseCursorAtFirstSession tests cursor behavior at first session.
func TestEdgeCaseCursorAtFirstSession(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.activePanel = SessionsPanel
	model.sessions = []*session.JuggleSession{
		{ID: "session-1", Description: "First session"},
		{ID: "session-2", Description: "Second session"},
		{ID: "session-3", Description: "Third session"},
	}
	model.sessionCursor = 0 // At the top (All pseudo-session)
	model.selectedSession = nil
	catwalk.RunModel(t, "testdata/edge_case_cursor_first_session", model)
}

// TestEdgeCaseCursorAtLastSession tests cursor behavior at last session.
func TestEdgeCaseCursorAtLastSession(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.activePanel = SessionsPanel
	model.sessions = []*session.JuggleSession{
		{ID: "session-1", Description: "First session"},
		{ID: "session-2", Description: "Second session"},
		{ID: "session-3", Description: "Third session"},
	}
	// With 2 pseudo-sessions + 3 real sessions = 5 total, last is index 4
	model.sessionCursor = 4
	model.selectedSession = model.sessions[2]
	catwalk.RunModel(t, "testdata/edge_case_cursor_last_session", model)
}

// TestEdgeCaseSingleBall tests display with only one ball.
func TestEdgeCaseSingleBall(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.activePanel = BallsPanel
	model.balls = []*session.Ball{
		{ID: "juggle-only", Title: "The only ball", State: session.StatePending},
	}
	model.filteredBalls = model.balls
	model.cursor = 0
	model.selectedBall = model.balls[0]
	catwalk.RunModel(t, "testdata/edge_case_single_ball", model)
}

// TestEdgeCaseSingleSession tests display with only one session.
func TestEdgeCaseSingleSession(t *testing.T) {
	model := createTestSplitViewModel(t)
	model.activePanel = SessionsPanel
	model.sessions = []*session.JuggleSession{
		{ID: "only-session", Description: "The only session"},
	}
	// Pseudo-sessions are at 0 and 1, the only real session is at 2
	model.sessionCursor = 2
	model.selectedSession = model.sessions[0]
	catwalk.RunModel(t, "testdata/edge_case_single_session", model)
}

// Helper functions for creating test data
func formatBallID(i int) string {
	return fmt.Sprintf("juggle-%d", i)
}

func formatBallTitle(i int) string {
	return fmt.Sprintf("Ball %d: Task description", i)
}
