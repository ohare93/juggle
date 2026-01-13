package tui

import (
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
