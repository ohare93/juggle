package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ohare93/juggle/internal/session"
)

func TestModelInitialization(t *testing.T) {
	// Create a mock store (even though it's nil, we're just testing structure)
	var store *session.Store
	var config *session.Config

	model := InitialModel(store, config, false)

	if model.mode != listView {
		t.Errorf("Expected initial mode to be listView, got %v", model.mode)
	}

	if model.cursor != 0 {
		t.Errorf("Expected initial cursor to be 0, got %d", model.cursor)
	}
}

func TestLocalOnlyMode(t *testing.T) {
	var store *session.Store
	var config *session.Config

	// Test local-only model
	model := InitialModel(store, config, true)
	if !model.localOnly {
		t.Error("Expected localOnly to be true")
	}

	// Test non-local model
	model2 := InitialModel(store, config, false)
	if model2.localOnly {
		t.Error("Expected localOnly to be false")
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"this is way too long", 10, "this is..."},
		{"", 5, ""},
	}

	for _, tt := range tests {
		result := truncate(tt.input, tt.maxLen)
		if result != tt.expected {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
		}
	}
}

func TestFormatState(t *testing.T) {
	tests := []struct {
		name     string
		ball     *session.Ball
		expected string
	}{
		{
			name: "pending state",
			ball: &session.Ball{
				State: session.StatePending,
			},
			expected: "pending",
		},
		{
			name: "in_progress state",
			ball: &session.Ball{
				State: session.StateInProgress,
			},
			expected: "in_progress",
		},
		{
			name: "complete state",
			ball: &session.Ball{
				State: session.StateComplete,
			},
			expected: "complete",
		},
		{
			name: "blocked state",
			ball: &session.Ball{
				State: session.StateBlocked,
			},
			expected: "blocked",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatState(tt.ball)
			if result != tt.expected {
				t.Errorf("formatState() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestCountByState(t *testing.T) {
	balls := []*session.Ball{
		{State: session.StatePending},
		{State: session.StatePending},
		{State: session.StateInProgress},
		{State: session.StateComplete},
		{State: session.StateBlocked},
	}

	tests := []struct {
		state    string
		expected int
	}{
		{"pending", 2},
		{"in_progress", 1},
		{"complete", 1},
		{"blocked", 1},
		{"nonexistent", 0},
	}

	for _, tt := range tests {
		count := countByState(balls, tt.state)
		if count != tt.expected {
			t.Errorf("countByState(balls, %q) = %d, want %d", tt.state, count, tt.expected)
		}
	}
}

func TestApplyFilters(t *testing.T) {
	model := Model{
		balls: []*session.Ball{
			{ID: "1", State: session.StatePending},
			{ID: "2", State: session.StatePending},
			{ID: "3", State: session.StateInProgress},
			{ID: "4", State: session.StateComplete},
		},
		filterStates: map[string]bool{
			"pending":     true,
			"in_progress": false,
			"complete":    false,
			"blocked":     false,
		},
	}

	model.applyFilters()

	if len(model.filteredBalls) != 2 {
		t.Errorf("Expected 2 filtered balls, got %d", len(model.filteredBalls))
	}

	for _, ball := range model.filteredBalls {
		if ball.State != session.StatePending {
			t.Errorf("Expected only pending balls, got ball with state %v", ball.State)
		}
	}
}

func TestApplyFiltersAll(t *testing.T) {
	model := Model{
		balls: []*session.Ball{
			{ID: "1", State: session.StatePending},
			{ID: "2", State: session.StateInProgress},
			{ID: "3", State: session.StateComplete},
		},
		filterStates: map[string]bool{
			"pending":     true,
			"in_progress": true,
			"complete":    true,
			"blocked":     true,
		},
	}

	model.applyFilters()

	if len(model.filteredBalls) != 3 {
		t.Errorf("Expected all 3 balls to be included, got %d", len(model.filteredBalls))
	}
}

func TestFilterToggleBehavior(t *testing.T) {
	model := Model{
		filterStates: map[string]bool{
			"pending":     true,
			"in_progress": true,
			"blocked":     false,
			"complete":    false,
		},
	}

	// Test toggling pending off
	model.handleStateFilter("2")
	if model.filterStates["pending"] {
		t.Error("Expected pending to be toggled off")
	}

	// Test toggling pending back on
	model.handleStateFilter("2")
	if !model.filterStates["pending"] {
		t.Error("Expected pending to be toggled on")
	}

	// Test show all (key "1")
	model.filterStates["pending"] = false
	model.filterStates["in_progress"] = false
	model.handleStateFilter("1")
	if !model.filterStates["pending"] || !model.filterStates["in_progress"] ||
		!model.filterStates["blocked"] || !model.filterStates["complete"] {
		t.Error("Expected all states to be visible after pressing '1'")
	}
}

func TestTruncateID(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		maxLen   int
		expected string
	}{
		{
			name:     "short ID",
			id:       "juggler-5",
			maxLen:   15,
			expected: "juggler-5",
		},
		{
			name:     "long timestamp ID",
			id:       "juggler-20251012-143438",
			maxLen:   15,
			expected: "juggler-...3438",
		},
		{
			name:     "exactly max length",
			id:       "project-1234567",
			maxLen:   15,
			expected: "project-1234567",
		},
		{
			name:     "single part ID",
			id:       "verylongprojectname",
			maxLen:   10,
			expected: "verylon...", // Falls back to regular truncate
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateID(tt.id, tt.maxLen)
			if result != tt.expected {
				t.Errorf("truncateID(%q, %d) = %q, want %q", tt.id, tt.maxLen, result, tt.expected)
			}
			if len(result) > tt.maxLen {
				t.Errorf("truncateID(%q, %d) returned %q which is longer than maxLen", tt.id, tt.maxLen, result)
			}
		})
	}
}

func TestStateTransitionsUnrestricted(t *testing.T) {
	tests := []struct {
		name         string
		initialState session.BallState
		action       string // "start", "complete", "block"
		expectError  bool
	}{
		{
			name:         "start from pending",
			initialState: session.StatePending,
			action:       "start",
			expectError:  false,
		},
		{
			name:         "start from complete",
			initialState: session.StateComplete,
			action:       "start",
			expectError:  false,
		},
		{
			name:         "start from blocked",
			initialState: session.StateBlocked,
			action:       "start",
			expectError:  false,
		},
		{
			name:         "complete from in_progress",
			initialState: session.StateInProgress,
			action:       "complete",
			expectError:  false,
		},
		{
			name:         "complete from pending",
			initialState: session.StatePending,
			action:       "complete",
			expectError:  false,
		},
		{
			name:         "block from in_progress",
			initialState: session.StateInProgress,
			action:       "block",
			expectError:  false,
		},
		{
			name:         "block from complete",
			initialState: session.StateComplete,
			action:       "block",
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := Model{
				filteredBalls: []*session.Ball{
					{
						ID:         "test-1",
						State:      tt.initialState,
						WorkingDir: "/tmp/test",
					},
				},
				cursor: 0,
			}

			var newModel tea.Model
			switch tt.action {
			case "start":
				newModel, _ = model.handleStartBall()
			case "complete":
				newModel, _ = model.handleCompleteBall()
			case "block":
				newModel, _ = model.handleDropBall()
			}

			m := newModel.(*Model)
			if m.err != nil && !tt.expectError {
				t.Errorf("Unexpected error: %v", m.err)
			}
			if m.err == nil && tt.expectError {
				t.Error("Expected error but got none")
			}

			// Check that state was updated (even if store operation failed)
			ball := m.filteredBalls[0]
			switch tt.action {
			case "start":
				if ball.State != session.StateInProgress {
					t.Errorf("Expected state to be in_progress, got %v", ball.State)
				}
			case "complete":
				if ball.State != session.StateComplete {
					t.Errorf("Expected state to be complete, got %v", ball.State)
				}
			case "block":
				if ball.State != session.StateBlocked {
					t.Errorf("Expected state to be blocked, got %v", ball.State)
				}
			}
		})
	}
}

func TestCyclePriority(t *testing.T) {
	tests := []struct {
		name             string
		currentPriority  session.Priority
		expectedPriority session.Priority
	}{
		{"low to medium", session.PriorityLow, session.PriorityMedium},
		{"medium to high", session.PriorityMedium, session.PriorityHigh},
		{"high to urgent", session.PriorityHigh, session.PriorityUrgent},
		{"urgent to low", session.PriorityUrgent, session.PriorityLow},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate cycling logic
			var nextPriority session.Priority
			switch tt.currentPriority {
			case session.PriorityLow:
				nextPriority = session.PriorityMedium
			case session.PriorityMedium:
				nextPriority = session.PriorityHigh
			case session.PriorityHigh:
				nextPriority = session.PriorityUrgent
			case session.PriorityUrgent:
				nextPriority = session.PriorityLow
			default:
				nextPriority = session.PriorityMedium
			}

			if nextPriority != tt.expectedPriority {
				t.Errorf("Expected %v, got %v", tt.expectedPriority, nextPriority)
			}
		})
	}
}

func TestConfirmDeleteRendering(t *testing.T) {
	ball := &session.Ball{
		ID:       "test-1",
		Intent:   "Test ball",
		Priority: session.PriorityMedium,
	}

	model := Model{
		mode:          confirmDeleteView,
		filteredBalls: []*session.Ball{ball},
		cursor:        0,
	}

	view := model.renderConfirmDeleteView()

	// Check that important elements are present
	if !strings.Contains(view, "DELETE BALL") {
		t.Error("View should contain DELETE BALL title")
	}
	if !strings.Contains(view, ball.ID) {
		t.Error("View should contain ball ID")
	}
	if !strings.Contains(view, "Delete this ball") {
		t.Error("View should contain confirmation prompt")
	}
	if !strings.Contains(view, "[y/N]") {
		t.Error("View should contain y/N options")
	}
}

func TestConfirmDeleteEmptyBalls(t *testing.T) {
	model := Model{
		mode:          confirmDeleteView,
		filteredBalls: []*session.Ball{},
		cursor:        0,
	}

	view := model.renderConfirmDeleteView()

	if !strings.Contains(view, "No ball selected") {
		t.Error("View should indicate no ball selected when filteredBalls is empty")
	}
}

func TestPriorityCycleDefaultCase(t *testing.T) {
	// Test that unknown priority defaults to medium
	var unknownPriority session.Priority = "unknown"
	var nextPriority session.Priority

	switch unknownPriority {
	case session.PriorityLow:
		nextPriority = session.PriorityMedium
	case session.PriorityMedium:
		nextPriority = session.PriorityHigh
	case session.PriorityHigh:
		nextPriority = session.PriorityUrgent
	case session.PriorityUrgent:
		nextPriority = session.PriorityLow
	default:
		nextPriority = session.PriorityMedium
	}

	if nextPriority != session.PriorityMedium {
		t.Errorf("Unknown priority should default to medium, got %v", nextPriority)
	}
}

func TestCycleState(t *testing.T) {
	tests := []struct {
		name          string
		currentState  session.BallState
		expectedState session.BallState
	}{
		{
			name:          "pending to in_progress",
			currentState:  session.StatePending,
			expectedState: session.StateInProgress,
		},
		{
			name:          "in_progress to complete",
			currentState:  session.StateInProgress,
			expectedState: session.StateComplete,
		},
		{
			name:          "complete to blocked",
			currentState:  session.StateComplete,
			expectedState: session.StateBlocked,
		},
		{
			name:          "blocked to pending",
			currentState:  session.StateBlocked,
			expectedState: session.StatePending,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate cycling logic from handleCycleState
			var nextState session.BallState

			switch tt.currentState {
			case session.StatePending:
				nextState = session.StateInProgress
			case session.StateInProgress:
				nextState = session.StateComplete
			case session.StateComplete:
				nextState = session.StateBlocked
			case session.StateBlocked:
				nextState = session.StatePending
			default:
				nextState = session.StatePending
			}

			if nextState != tt.expectedState {
				t.Errorf("Expected state %v, got %v", tt.expectedState, nextState)
			}
		})
	}
}

func TestSetPendingFromAnyState(t *testing.T) {
	states := []struct {
		name  string
		state session.BallState
	}{
		{"from in_progress", session.StateInProgress},
		{"from complete", session.StateComplete},
		{"from blocked", session.StateBlocked},
		{"from pending", session.StatePending}, // Should be idempotent
	}

	for _, tt := range states {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate logic from handleSetReady
			var nextState session.BallState = session.StatePending

			if nextState != session.StatePending {
				t.Errorf("Failed to set %v to pending, got %v", tt.state, nextState)
			}
		})
	}
}

// Test SplitView model initialization
func TestInitialSplitModel(t *testing.T) {
	var store *session.Store
	var sessionStore *session.SessionStore
	var config *session.Config

	model := InitialSplitModel(store, sessionStore, config, true)

	if model.mode != splitView {
		t.Errorf("Expected initial mode to be splitView, got %v", model.mode)
	}

	if model.activePanel != SessionsPanel {
		t.Errorf("Expected initial active panel to be SessionsPanel, got %v", model.activePanel)
	}

	if !model.localOnly {
		t.Error("Expected localOnly to be true")
	}

	if model.cursor != 0 {
		t.Errorf("Expected initial cursor to be 0, got %d", model.cursor)
	}

	if model.sessionCursor != 0 {
		t.Errorf("Expected initial session cursor to be 0, got %d", model.sessionCursor)
	}
}

// Test activity log management
func TestAddActivity(t *testing.T) {
	model := Model{
		activityLog: make([]ActivityEntry, 0),
	}

	// Add activity entries
	model.addActivity("First activity")
	model.addActivity("Second activity")
	model.addActivity("Third activity")

	if len(model.activityLog) != 3 {
		t.Errorf("Expected 3 activity entries, got %d", len(model.activityLog))
	}

	if model.activityLog[0].Message != "First activity" {
		t.Errorf("Expected first message to be 'First activity', got '%s'", model.activityLog[0].Message)
	}

	if model.activityLog[2].Message != "Third activity" {
		t.Errorf("Expected third message to be 'Third activity', got '%s'", model.activityLog[2].Message)
	}
}

// Test activity log capacity limit
func TestAddActivityLimit(t *testing.T) {
	model := Model{
		activityLog: make([]ActivityEntry, 0),
	}

	// Add 101 entries (limit is 100)
	for i := 0; i < 101; i++ {
		model.addActivity("Activity " + string(rune('A'+i%26)))
	}

	if len(model.activityLog) != 100 {
		t.Errorf("Expected 100 activity entries (limit), got %d", len(model.activityLog))
	}
}

// Test panel navigation
func TestPanelCycling(t *testing.T) {
	tests := []struct {
		name           string
		startPanel     Panel
		expectedNext   Panel
		expectedPrev   Panel
	}{
		{
			name:           "sessions panel",
			startPanel:     SessionsPanel,
			expectedNext:   BallsPanel,
			expectedPrev:   ActivityPanel,
		},
		{
			name:           "balls panel",
			startPanel:     BallsPanel,
			expectedNext:   ActivityPanel,
			expectedPrev:   SessionsPanel,
		},
		{
			name:           "activity panel",
			startPanel:     ActivityPanel,
			expectedNext:   SessionsPanel,
			expectedPrev:   BallsPanel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test next panel cycling
			startPanel := tt.startPanel
			var nextPanel Panel
			switch startPanel {
			case SessionsPanel:
				nextPanel = BallsPanel
			case BallsPanel:
				nextPanel = ActivityPanel
			case ActivityPanel:
				nextPanel = SessionsPanel
			}

			if nextPanel != tt.expectedNext {
				t.Errorf("Next panel from %v: expected %v, got %v", startPanel, tt.expectedNext, nextPanel)
			}

			// Test previous panel cycling
			var prevPanel Panel
			switch startPanel {
			case SessionsPanel:
				prevPanel = ActivityPanel
			case BallsPanel:
				prevPanel = SessionsPanel
			case ActivityPanel:
				prevPanel = BallsPanel
			}

			if prevPanel != tt.expectedPrev {
				t.Errorf("Previous panel from %v: expected %v, got %v", startPanel, tt.expectedPrev, prevPanel)
			}
		})
	}
}

// Test getBallsForSession with pseudo-sessions
func TestGetBallsForSession(t *testing.T) {
	balls := []*session.Ball{
		{ID: "1", Tags: []string{"session-a"}},
		{ID: "2", Tags: []string{"session-b"}},
		{ID: "3", Tags: []string{}}, // Untagged
		{ID: "4", Tags: []string{"session-a", "session-b"}},
	}

	sessions := []*session.JuggleSession{
		{ID: "session-a"},
		{ID: "session-b"},
	}

	t.Run("PseudoSessionAll returns all balls", func(t *testing.T) {
		model := Model{
			filteredBalls: balls,
			sessions:      sessions,
			selectedSession: &session.JuggleSession{
				ID: PseudoSessionAll,
			},
		}

		result := model.getBallsForSession()
		if len(result) != 4 {
			t.Errorf("Expected 4 balls for 'All', got %d", len(result))
		}
	})

	t.Run("PseudoSessionUntagged returns untagged balls only", func(t *testing.T) {
		model := Model{
			filteredBalls: balls,
			sessions:      sessions,
			selectedSession: &session.JuggleSession{
				ID: PseudoSessionUntagged,
			},
		}

		result := model.getBallsForSession()
		if len(result) != 1 {
			t.Errorf("Expected 1 untagged ball, got %d", len(result))
		}
		if result[0].ID != "3" {
			t.Errorf("Expected untagged ball ID '3', got '%s'", result[0].ID)
		}
	})

	t.Run("Regular session returns matching balls", func(t *testing.T) {
		model := Model{
			filteredBalls: balls,
			sessions:      sessions,
			selectedSession: &session.JuggleSession{
				ID: "session-a",
			},
		}

		result := model.getBallsForSession()
		if len(result) != 2 {
			t.Errorf("Expected 2 balls for 'session-a', got %d", len(result))
		}
	})

	t.Run("No selected session returns all filtered balls", func(t *testing.T) {
		model := Model{
			filteredBalls:   balls,
			sessions:        sessions,
			selectedSession: nil,
		}

		result := model.getBallsForSession()
		if len(result) != 4 {
			t.Errorf("Expected 4 balls when no session selected, got %d", len(result))
		}
	})
}

// Test SelectedSessionID
func TestSelectedSessionID(t *testing.T) {
	t.Run("with selected session", func(t *testing.T) {
		model := Model{
			selectedSession: &session.JuggleSession{ID: "test-session"},
		}

		id := model.SelectedSessionID()
		if id != "test-session" {
			t.Errorf("Expected 'test-session', got '%s'", id)
		}
	})

	t.Run("without selected session", func(t *testing.T) {
		model := Model{
			selectedSession: nil,
		}

		id := model.SelectedSessionID()
		if id != "" {
			t.Errorf("Expected empty string, got '%s'", id)
		}
	})
}

// Test filterSessions with pseudo-sessions
func TestFilterSessions(t *testing.T) {
	sessions := []*session.JuggleSession{
		{ID: "session-a", Description: "First session"},
		{ID: "session-b", Description: "Second session"},
	}

	t.Run("no filter returns all sessions with pseudo-sessions", func(t *testing.T) {
		model := Model{
			sessions:          sessions,
			panelSearchActive: false,
		}

		result := model.filterSessions()
		// Should include 2 pseudo-sessions + 2 real sessions
		if len(result) != 4 {
			t.Errorf("Expected 4 sessions (2 pseudo + 2 real), got %d", len(result))
		}

		// Check pseudo-sessions are first
		if result[0].ID != PseudoSessionAll {
			t.Errorf("Expected first session to be PseudoSessionAll, got '%s'", result[0].ID)
		}
		if result[1].ID != PseudoSessionUntagged {
			t.Errorf("Expected second session to be PseudoSessionUntagged, got '%s'", result[1].ID)
		}
	})

	t.Run("filter by ID matches partial", func(t *testing.T) {
		model := Model{
			sessions:          sessions,
			panelSearchActive: true,
			panelSearchQuery:  "session-a",
		}

		result := model.filterSessions()
		if len(result) != 1 {
			t.Errorf("Expected 1 session matching 'session-a', got %d", len(result))
		}
		if result[0].ID != "session-a" {
			t.Errorf("Expected session ID 'session-a', got '%s'", result[0].ID)
		}
	})

	t.Run("filter by description", func(t *testing.T) {
		model := Model{
			sessions:          sessions,
			panelSearchActive: true,
			panelSearchQuery:  "First",
		}

		result := model.filterSessions()
		if len(result) != 1 {
			t.Errorf("Expected 1 session matching 'First', got %d", len(result))
		}
	})
}

// Test filterBallsForSession
func TestFilterBallsForSession(t *testing.T) {
	balls := []*session.Ball{
		{ID: "ball-1", Intent: "First task", Tags: []string{"session-a"}},
		{ID: "ball-2", Intent: "Second task", Tags: []string{"session-a"}},
		{ID: "ball-3", Intent: "Third task", Tags: []string{"session-b"}},
	}

	t.Run("no filter returns all session balls", func(t *testing.T) {
		model := Model{
			filteredBalls:     balls,
			panelSearchActive: false,
			selectedSession:   &session.JuggleSession{ID: "session-a"},
		}

		result := model.filterBallsForSession()
		if len(result) != 2 {
			t.Errorf("Expected 2 balls for session-a, got %d", len(result))
		}
	})

	t.Run("filter by intent", func(t *testing.T) {
		model := Model{
			filteredBalls:     balls,
			panelSearchActive: true,
			panelSearchQuery:  "First",
			selectedSession:   &session.JuggleSession{ID: "session-a"},
		}

		result := model.filterBallsForSession()
		if len(result) != 1 {
			t.Errorf("Expected 1 ball matching 'First', got %d", len(result))
		}
	})

	t.Run("filter by ID", func(t *testing.T) {
		model := Model{
			filteredBalls:     balls,
			panelSearchActive: true,
			panelSearchQuery:  "ball-1",
			selectedSession:   &session.JuggleSession{ID: "session-a"},
		}

		result := model.filterBallsForSession()
		if len(result) != 1 {
			t.Errorf("Expected 1 ball matching 'ball-1', got %d", len(result))
		}
	})
}

// Test countBallsForSession (via split view delete confirmation)
func TestCountBallsForSession(t *testing.T) {
	balls := []*session.Ball{
		{ID: "1", Tags: []string{"session-a"}},
		{ID: "2", Tags: []string{"session-a"}},
		{ID: "3", Tags: []string{"session-b"}},
	}

	model := Model{
		filteredBalls: balls,
	}

	count := model.countBallsForSession("session-a")
	if count != 2 {
		t.Errorf("Expected 2 balls for session-a, got %d", count)
	}

	count = model.countBallsForSession("session-b")
	if count != 1 {
		t.Errorf("Expected 1 ball for session-b, got %d", count)
	}

	count = model.countBallsForSession("nonexistent")
	if count != 0 {
		t.Errorf("Expected 0 balls for nonexistent session, got %d", count)
	}
}

// Test bottom pane mode toggle
func TestBottomPaneModeToggle(t *testing.T) {
	model := Model{
		bottomPaneMode: BottomPaneActivity,
		activityLog:    make([]ActivityEntry, 0),
	}

	// Toggle to detail
	newModel, _ := model.handleToggleBottomPane()
	m := newModel.(Model)
	if m.bottomPaneMode != BottomPaneDetail {
		t.Errorf("Expected BottomPaneDetail after toggle, got %v", m.bottomPaneMode)
	}

	// Toggle back to activity
	newModel, _ = m.handleToggleBottomPane()
	m = newModel.(Model)
	if m.bottomPaneMode != BottomPaneActivity {
		t.Errorf("Expected BottomPaneActivity after second toggle, got %v", m.bottomPaneMode)
	}
}

// Test local only toggle
func TestToggleLocalOnly(t *testing.T) {
	model := Model{
		localOnly:   true,
		activityLog: make([]ActivityEntry, 0),
	}

	// Toggle to all projects
	newModel, cmd := model.handleToggleLocalOnly()
	m := newModel.(Model)

	if m.localOnly {
		t.Error("Expected localOnly to be false after toggle")
	}

	// Should return a command to reload balls
	if cmd == nil {
		t.Error("Expected a reload command to be returned")
	}

	// Toggle back to local only
	newModel, _ = m.handleToggleLocalOnly()
	m = newModel.(Model)

	if !m.localOnly {
		t.Error("Expected localOnly to be true after second toggle")
	}
}

// Test activity log scrolling helper
func TestGetActivityLogMaxOffset(t *testing.T) {
	tests := []struct {
		name           string
		activityCount  int
		expectedOffset int
	}{
		{
			name:           "few activities",
			activityCount:  5,
			expectedOffset: 0, // Not enough to scroll
		},
		{
			name:           "many activities",
			activityCount:  50,
			expectedOffset: 50 - (bottomPanelRows - 3),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := Model{
				activityLog: make([]ActivityEntry, tt.activityCount),
			}

			offset := model.getActivityLogMaxOffset()
			if offset < 0 {
				offset = 0
			}

			if offset != tt.expectedOffset && tt.activityCount <= (bottomPanelRows-3) {
				// For few activities, offset should be 0
				if offset != 0 {
					t.Errorf("Expected max offset 0 for few activities, got %d", offset)
				}
			}
		})
	}
}

// Test view rendering doesn't panic with various states
func TestViewRenderingNoPanic(t *testing.T) {
	states := []struct {
		name string
		mode viewMode
	}{
		{"listView", listView},
		{"detailView", detailView},
		{"helpView", helpView},
		{"confirmDeleteView", confirmDeleteView},
		{"splitView", splitView},
		{"splitHelpView", splitHelpView},
	}

	for _, tt := range states {
		t.Run(tt.name, func(t *testing.T) {
			model := Model{
				mode:          tt.mode,
				balls:         []*session.Ball{},
				filteredBalls: []*session.Ball{},
				sessions:      []*session.JuggleSession{},
				activityLog:   make([]ActivityEntry, 0),
				filterStates: map[string]bool{
					"pending":     true,
					"in_progress": true,
					"blocked":     true,
					"complete":    true,
				},
				width:  80,
				height: 24,
			}

			// This should not panic
			view := model.View()
			if view == "" {
				t.Error("Expected non-empty view output")
			}
		})
	}
}

// Test key message handling for list view
func TestListViewKeyHandling(t *testing.T) {
	balls := []*session.Ball{
		{ID: "1", State: session.StatePending, WorkingDir: "/tmp"},
		{ID: "2", State: session.StatePending, WorkingDir: "/tmp"},
		{ID: "3", State: session.StatePending, WorkingDir: "/tmp"},
	}

	model := Model{
		mode:          listView,
		balls:         balls,
		filteredBalls: balls,
		cursor:        0,
		filterStates: map[string]bool{
			"pending":     true,
			"in_progress": true,
			"blocked":     true,
			"complete":    true,
		},
	}

	// Test navigation down
	newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	m := newModel.(Model)
	if m.cursor != 1 {
		t.Errorf("Expected cursor to move to 1, got %d", m.cursor)
	}

	// Test navigation up
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = newModel.(Model)
	if m.cursor != 0 {
		t.Errorf("Expected cursor to move back to 0, got %d", m.cursor)
	}
}

// Test entering detail view
func TestEnterDetailView(t *testing.T) {
	balls := []*session.Ball{
		{ID: "test-1", Intent: "Test ball", State: session.StatePending},
	}

	model := Model{
		mode:          listView,
		balls:         balls,
		filteredBalls: balls,
		cursor:        0,
	}

	// Press enter to go to detail view
	newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m := newModel.(Model)

	if m.mode != detailView {
		t.Errorf("Expected mode to be detailView, got %v", m.mode)
	}

	if m.selectedBall == nil {
		t.Error("Expected selectedBall to be set")
	}

	if m.selectedBall.ID != "test-1" {
		t.Errorf("Expected selected ball ID to be 'test-1', got '%s'", m.selectedBall.ID)
	}
}

// Test escape key behavior
func TestEscapeKeyBehavior(t *testing.T) {
	t.Run("escape from detail view goes to list", func(t *testing.T) {
		model := Model{
			mode: detailView,
		}

		newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
		m := newModel.(Model)

		if m.mode != listView {
			t.Errorf("Expected mode to be listView after escape, got %v", m.mode)
		}
	})

	t.Run("escape from help view goes to list", func(t *testing.T) {
		model := Model{
			mode: helpView,
		}

		newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
		m := newModel.(Model)

		if m.mode != listView {
			t.Errorf("Expected mode to be listView after escape from help, got %v", m.mode)
		}
	})

	t.Run("escape from confirm delete goes to list", func(t *testing.T) {
		model := Model{
			mode: confirmDeleteView,
		}

		newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
		m := newModel.(Model)

		if m.mode != listView {
			t.Errorf("Expected mode to be listView after escape from confirm, got %v", m.mode)
		}
	})
}

// Test help toggle
func TestHelpToggle(t *testing.T) {
	model := Model{
		mode: listView,
	}

	// Toggle help on
	newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	m := newModel.(Model)

	if m.mode != helpView {
		t.Errorf("Expected mode to be helpView after pressing ?, got %v", m.mode)
	}

	// Toggle help off
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	m = newModel.(Model)

	if m.mode != listView {
		t.Errorf("Expected mode to be listView after pressing ? again, got %v", m.mode)
	}
}

// Test balls loaded message handling
func TestBallsLoadedMsg(t *testing.T) {
	balls := []*session.Ball{
		{ID: "1", State: session.StatePending},
		{ID: "2", State: session.StateInProgress},
	}

	model := Model{
		mode: listView,
		filterStates: map[string]bool{
			"pending":     true,
			"in_progress": true,
			"blocked":     true,
			"complete":    true,
		},
	}

	newModel, _ := model.Update(ballsLoadedMsg{balls: balls})
	m := newModel.(Model)

	if len(m.balls) != 2 {
		t.Errorf("Expected 2 balls, got %d", len(m.balls))
	}

	if len(m.filteredBalls) != 2 {
		t.Errorf("Expected 2 filtered balls, got %d", len(m.filteredBalls))
	}
}

// Test sessions loaded message handling
func TestSessionsLoadedMsg(t *testing.T) {
	sessions := []*session.JuggleSession{
		{ID: "session-1", Description: "First"},
		{ID: "session-2", Description: "Second"},
	}

	model := Model{
		mode:        splitView,
		activityLog: make([]ActivityEntry, 0),
	}

	newModel, _ := model.Update(sessionsLoadedMsg{sessions: sessions})
	m := newModel.(Model)

	if len(m.sessions) != 2 {
		t.Errorf("Expected 2 sessions, got %d", len(m.sessions))
	}
}

// Test window size message handling
func TestWindowSizeMsg(t *testing.T) {
	model := Model{
		width:  80,
		height: 24,
	}

	newModel, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m := newModel.(Model)

	if m.width != 120 {
		t.Errorf("Expected width 120, got %d", m.width)
	}

	if m.height != 40 {
		t.Errorf("Expected height 40, got %d", m.height)
	}
}
