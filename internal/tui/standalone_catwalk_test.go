package tui

import (
	"fmt"
	"os"
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

	tmpDir, err := os.MkdirTemp("", "juggler-tui-test-*")
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
	tmpDir, err := os.MkdirTemp("", "juggler-tui-test-*")
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
	tmpDir, err := os.MkdirTemp("", "juggler-tui-test-*")
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

// createTestSplitViewModel creates a Model configured for split view testing.
func createTestSplitViewModel(t *testing.T) Model {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "juggler-tui-test-*")
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
		{ID: "juggler-1", Title: "First pending task", State: session.StatePending, Priority: session.PriorityMedium},
		{ID: "juggler-2", Title: "Second in progress task", State: session.StateInProgress, Priority: session.PriorityHigh},
		{ID: "juggler-3", Title: "Third blocked task", State: session.StateBlocked, Priority: session.PriorityLow},
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
		{ID: "juggler-1", Title: "First pending task", State: session.StatePending, Priority: session.PriorityMedium},
		{ID: "juggler-2", Title: "Second in progress task", State: session.StateInProgress, Priority: session.PriorityHigh},
		{ID: "juggler-3", Title: "Third blocked task", State: session.StateBlocked, Priority: session.PriorityLow},
	}
	selectedActive := make(map[string]bool)
	selectedActive["juggler-1"] = true
	selectedActive["juggler-3"] = true

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
			ID:       "juggler-1",
			Title:    "First pending task",
			State:    session.StatePending,
			Priority: session.PriorityLow,
		},
		{
			ID:       "juggler-2",
			Title:    "Second in progress task",
			State:    session.StateInProgress,
			Priority: session.PriorityMedium,
		},
		{
			ID:            "juggler-3",
			Title:         "Third blocked task",
			State:         session.StateBlocked,
			BlockedReason: "Waiting for dependencies",
			Priority:      session.PriorityHigh,
		},
		{
			ID:       "juggler-4",
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
		{ID: "juggler-1", Title: "Low priority task", State: session.StatePending, Priority: session.PriorityLow},
		{ID: "juggler-2", Title: "Medium priority task", State: session.StateInProgress, Priority: session.PriorityMedium},
		{ID: "juggler-3", Title: "High priority task", State: session.StateBlocked, Priority: session.PriorityHigh},
		{ID: "juggler-4", Title: "Urgent task", State: session.StateComplete, Priority: session.PriorityUrgent},
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
		{ID: "juggler-1", Title: "Tagged task one", State: session.StatePending, Tags: []string{"feature", "backend"}},
		{ID: "juggler-2", Title: "Tagged task two", State: session.StateInProgress, Tags: []string{"bugfix", "ui"}},
		{ID: "juggler-3", Title: "Task with many tags", State: session.StateBlocked, Tags: []string{"docs", "refactor", "testing"}},
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
			ID:                   "juggler-1",
			Title:                "Task with no tests",
			State:                session.StatePending,
			AcceptanceCriteria:   []string{},
		},
		{
			ID:                   "juggler-2",
			Title:                "Task with some tests",
			State:                session.StateInProgress,
			AcceptanceCriteria:   []string{"AC 1: Test X", "AC 2: Test Y"},
		},
		{
			ID:                   "juggler-3",
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
			ID:                   "juggler-1",
			Title:                "First complex task",
			State:                session.StatePending,
			Priority:             session.PriorityHigh,
			Tags:                 []string{"feature", "backend"},
			AcceptanceCriteria:   []string{"AC 1", "AC 2"},
		},
		{
			ID:                   "juggler-2",
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
		{ID: "juggler-3", Title: "Third task", State: session.StatePending},
		{ID: "juggler-1", Title: "First task", State: session.StateInProgress},
		{ID: "juggler-2", Title: "Second task", State: session.StateBlocked},
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
		{ID: "juggler-1", Title: "Low priority", State: session.StatePending, Priority: session.PriorityLow},
		{ID: "juggler-4", Title: "Urgent task", State: session.StateInProgress, Priority: session.PriorityUrgent},
		{ID: "juggler-2", Title: "Medium priority", State: session.StateBlocked, Priority: session.PriorityMedium},
		{ID: "juggler-3", Title: "High priority", State: session.StateComplete, Priority: session.PriorityHigh},
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
			ID:            "juggler-1",
			Title:         "Blocked by dependency",
			State:         session.StateBlocked,
			BlockedReason: "Waiting for juggler-2 to complete",
		},
		{
			ID:            "juggler-2",
			Title:         "Blocked by external event",
			State:         session.StateBlocked,
			BlockedReason: "Awaiting API credentials from DevOps",
		},
		{
			ID:       "juggler-3",
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
			ID:       "juggler-1",
			Title:    "Task with no deps",
			State:    session.StatePending,
		},
		{
			ID:        "juggler-2",
			Title:     "Task with dependencies",
			State:     session.StatePending,
			DependsOn: []string{"juggler-1"},
		},
		{
			ID:        "juggler-3",
			Title:     "Task with multiple deps",
			State:     session.StateInProgress,
			DependsOn: []string{"juggler-1", "juggler-2"},
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
		ID:    "juggler-1",
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
		{Time: fixedTime.Add(2 * time.Second), Message: "Ball juggler-1 selected"},
		{Time: fixedTime.Add(3 * time.Second), Message: "Activity log refreshed"},
	}

	model.balls = append(model.balls, &session.Ball{
		ID:    "juggler-1",
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
		ID:        "juggler-1",
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
		ID:        "juggler-1",
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
		ID:    "juggler-1",
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
		ID:    "juggler-1",
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
		ID:        "juggler-1",
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
		ID:    "juggler-1",
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
		ID:    "juggler-1",
		Title: "Test ball",
		State: session.StatePending,
	})
	model.filteredBalls = model.balls
	catwalk.RunModel(t, "testdata/agent_output_panel_expanded", model)
}

// Helper functions for creating test data
func formatBallID(i int) string {
	return fmt.Sprintf("juggler-%d", i)
}

func formatBallTitle(i int) string {
	return fmt.Sprintf("Ball %d: Task description", i)
}
