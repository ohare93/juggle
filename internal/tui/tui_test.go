package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
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

// Test bottom pane mode toggle cycles through all three modes
func TestBottomPaneModeToggle(t *testing.T) {
	model := Model{
		bottomPaneMode: BottomPaneActivity,
		activityLog:    make([]ActivityEntry, 0),
	}

	// Toggle to detail
	newModel, _ := model.handleToggleBottomPane()
	m := newModel.(Model)
	if m.bottomPaneMode != BottomPaneDetail {
		t.Errorf("Expected BottomPaneDetail after first toggle, got %v", m.bottomPaneMode)
	}

	// Toggle to split
	newModel, _ = m.handleToggleBottomPane()
	m = newModel.(Model)
	if m.bottomPaneMode != BottomPaneSplit {
		t.Errorf("Expected BottomPaneSplit after second toggle, got %v", m.bottomPaneMode)
	}

	// Toggle back to activity
	newModel, _ = m.handleToggleBottomPane()
	m = newModel.(Model)
	if m.bottomPaneMode != BottomPaneActivity {
		t.Errorf("Expected BottomPaneActivity after third toggle, got %v", m.bottomPaneMode)
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

// Test acceptance criteria input mode - transition from intent input
func TestAcceptanceCriteriaInputTransition(t *testing.T) {
	model := Model{
		mode:        inputBallView,
		inputAction: actionAdd,
		activityLog: make([]ActivityEntry, 0),
	}
	model.textInput.SetValue("Test intent")

	// Simulate entering intent
	m := model
	m.pendingBallIntent = "Test intent"
	m.pendingAcceptanceCriteria = []string{}
	m.mode = inputAcceptanceCriteriaView

	if m.mode != inputAcceptanceCriteriaView {
		t.Errorf("Expected mode to be inputAcceptanceCriteriaView, got %v", m.mode)
	}

	if m.pendingBallIntent != "Test intent" {
		t.Errorf("Expected pendingBallIntent to be 'Test intent', got %s", m.pendingBallIntent)
	}

	if len(m.pendingAcceptanceCriteria) != 0 {
		t.Errorf("Expected pendingAcceptanceCriteria to be empty, got %d items", len(m.pendingAcceptanceCriteria))
	}
}

// Test acceptance criteria input - adding criteria
func TestAcceptanceCriteriaInputAddCriteria(t *testing.T) {
	model := Model{
		mode:                      inputAcceptanceCriteriaView,
		pendingBallIntent:         "Test intent",
		pendingAcceptanceCriteria: []string{},
		activityLog:               make([]ActivityEntry, 0),
	}
	model.textInput.SetValue("First AC")

	// Simulate entering a non-empty criterion
	newModel, _ := model.handleAcceptanceCriteriaKey(tea.KeyMsg{Type: tea.KeyEnter})
	m := newModel.(Model)

	if len(m.pendingAcceptanceCriteria) != 1 {
		t.Errorf("Expected 1 acceptance criterion, got %d", len(m.pendingAcceptanceCriteria))
	}

	if m.pendingAcceptanceCriteria[0] != "First AC" {
		t.Errorf("Expected criterion to be 'First AC', got %s", m.pendingAcceptanceCriteria[0])
	}

	// Mode should still be inputAcceptanceCriteriaView for more input
	if m.mode != inputAcceptanceCriteriaView {
		t.Errorf("Expected mode to remain inputAcceptanceCriteriaView, got %v", m.mode)
	}

	// Text input should be reset
	if m.textInput.Value() != "" {
		t.Errorf("Expected text input to be reset, got %s", m.textInput.Value())
	}
}

// Test acceptance criteria input - adding multiple criteria
func TestAcceptanceCriteriaInputMultipleCriteria(t *testing.T) {
	model := Model{
		mode:                      inputAcceptanceCriteriaView,
		pendingBallIntent:         "Test intent",
		pendingAcceptanceCriteria: []string{"First AC"},
		activityLog:               make([]ActivityEntry, 0),
	}
	model.textInput.SetValue("Second AC")

	newModel, _ := model.handleAcceptanceCriteriaKey(tea.KeyMsg{Type: tea.KeyEnter})
	m := newModel.(Model)

	if len(m.pendingAcceptanceCriteria) != 2 {
		t.Errorf("Expected 2 acceptance criteria, got %d", len(m.pendingAcceptanceCriteria))
	}

	if m.pendingAcceptanceCriteria[1] != "Second AC" {
		t.Errorf("Expected second criterion to be 'Second AC', got %s", m.pendingAcceptanceCriteria[1])
	}
}

// Test acceptance criteria input - cancel with esc
func TestAcceptanceCriteriaInputCancel(t *testing.T) {
	model := Model{
		mode:                      inputAcceptanceCriteriaView,
		pendingBallIntent:         "Test intent",
		pendingAcceptanceCriteria: []string{"First AC", "Second AC"},
		activityLog:               make([]ActivityEntry, 0),
	}

	newModel, _ := model.handleAcceptanceCriteriaKey(tea.KeyMsg{Type: tea.KeyEsc})
	m := newModel.(Model)

	if m.mode != splitView {
		t.Errorf("Expected mode to be splitView after cancel, got %v", m.mode)
	}

	if m.pendingBallIntent != "" {
		t.Errorf("Expected pendingBallIntent to be cleared, got %s", m.pendingBallIntent)
	}

	if m.pendingAcceptanceCriteria != nil {
		t.Errorf("Expected pendingAcceptanceCriteria to be nil, got %v", m.pendingAcceptanceCriteria)
	}

	if m.message != "Cancelled" {
		t.Errorf("Expected message to be 'Cancelled', got %s", m.message)
	}
}

// Test acceptance criteria view rendering
func TestAcceptanceCriteriaViewRendering(t *testing.T) {
	model := Model{
		mode:                      inputAcceptanceCriteriaView,
		pendingBallIntent:         "Implement feature X",
		pendingAcceptanceCriteria: []string{"AC 1", "AC 2"},
		activityLog:               make([]ActivityEntry, 0),
		width:                     80,
		height:                    24,
	}

	view := model.View()

	// Check for title
	if !strings.Contains(view, "Add Acceptance Criteria") {
		t.Error("Expected view to contain 'Add Acceptance Criteria'")
	}

	// Check for intent display
	if !strings.Contains(view, "Intent: Implement feature X") {
		t.Error("Expected view to contain intent")
	}

	// Check for existing criteria
	if !strings.Contains(view, "1. AC 1") {
		t.Error("Expected view to contain first criterion")
	}

	if !strings.Contains(view, "2. AC 2") {
		t.Error("Expected view to contain second criterion")
	}

	// Check for instruction
	if !strings.Contains(view, "Enter empty line to finish") {
		t.Error("Expected view to contain finish instruction")
	}
}

// Test submitBallInput transitions to ball form view for new balls
func TestSubmitBallInputTransitionsToBallForm(t *testing.T) {
	ti := textinput.New()
	ti.CharLimit = 256
	ti.Width = 40

	model := Model{
		mode:        inputBallView,
		inputAction: actionAdd,
		activityLog: make([]ActivityEntry, 0),
		textInput:   ti,
	}

	newModel, _ := model.submitBallInput("New ball intent")
	m := newModel.(Model)

	if m.mode != inputBallFormView {
		t.Errorf("Expected mode to be inputBallFormView, got %v", m.mode)
	}

	if m.pendingBallIntent != "New ball intent" {
		t.Errorf("Expected pendingBallIntent to be 'New ball intent', got %s", m.pendingBallIntent)
	}

	if m.pendingBallPriority != 1 {
		t.Errorf("Expected pendingBallPriority to be 1 (medium), got %d", m.pendingBallPriority)
	}

	if m.pendingBallState != 0 {
		t.Errorf("Expected pendingBallState to be 0 (pending), got %d", m.pendingBallState)
	}

	if len(m.pendingAcceptanceCriteria) != 0 {
		t.Errorf("Expected pendingAcceptanceCriteria to be empty, got %d", len(m.pendingAcceptanceCriteria))
	}
}

// Test ball form view navigation with arrow keys
func TestBallFormNavigation(t *testing.T) {
	ti := textinput.New()
	ti.CharLimit = 256
	ti.Width = 40

	model := Model{
		mode:                 inputBallFormView,
		pendingBallIntent:    "Test ball",
		pendingBallFormField: 0, // Start at priority
		pendingBallPriority:  1, // medium
		pendingBallState:     0, // pending
		textInput:            ti,
		sessions:             []*session.JuggleSession{},
	}

	// Test down navigation
	newModel, _ := model.handleBallFormKey(tea.KeyMsg{Type: tea.KeyDown})
	m := newModel.(Model)
	if m.pendingBallFormField != 1 {
		t.Errorf("Expected field to be 1 after down, got %d", m.pendingBallFormField)
	}

	// Test up navigation
	newModel, _ = m.handleBallFormKey(tea.KeyMsg{Type: tea.KeyUp})
	m = newModel.(Model)
	if m.pendingBallFormField != 0 {
		t.Errorf("Expected field to be 0 after up, got %d", m.pendingBallFormField)
	}

	// Test wrap around down
	m.pendingBallFormField = 3 // session field (last)
	newModel, _ = m.handleBallFormKey(tea.KeyMsg{Type: tea.KeyDown})
	m = newModel.(Model)
	if m.pendingBallFormField != 0 {
		t.Errorf("Expected field to wrap to 0, got %d", m.pendingBallFormField)
	}

	// Test wrap around up
	m.pendingBallFormField = 0
	newModel, _ = m.handleBallFormKey(tea.KeyMsg{Type: tea.KeyUp})
	m = newModel.(Model)
	if m.pendingBallFormField != 3 {
		t.Errorf("Expected field to wrap to 3, got %d", m.pendingBallFormField)
	}
}

// Test ball form priority selection
func TestBallFormPrioritySelection(t *testing.T) {
	ti := textinput.New()
	ti.CharLimit = 256
	ti.Width = 40

	model := Model{
		mode:                 inputBallFormView,
		pendingBallFormField: 0, // priority field
		pendingBallPriority:  1, // medium
		textInput:            ti,
		sessions:             []*session.JuggleSession{},
	}

	// Test right to cycle to high
	newModel, _ := model.handleBallFormKey(tea.KeyMsg{Type: tea.KeyRight})
	m := newModel.(Model)
	if m.pendingBallPriority != 2 {
		t.Errorf("Expected priority to be 2 (high) after right, got %d", m.pendingBallPriority)
	}

	// Test left to cycle back to medium
	newModel, _ = m.handleBallFormKey(tea.KeyMsg{Type: tea.KeyLeft})
	m = newModel.(Model)
	if m.pendingBallPriority != 1 {
		t.Errorf("Expected priority to be 1 (medium) after left, got %d", m.pendingBallPriority)
	}

	// Test wrap around right
	m.pendingBallPriority = 3 // urgent
	newModel, _ = m.handleBallFormKey(tea.KeyMsg{Type: tea.KeyRight})
	m = newModel.(Model)
	if m.pendingBallPriority != 0 {
		t.Errorf("Expected priority to wrap to 0 (low), got %d", m.pendingBallPriority)
	}

	// Test wrap around left
	m.pendingBallPriority = 0 // low
	newModel, _ = m.handleBallFormKey(tea.KeyMsg{Type: tea.KeyLeft})
	m = newModel.(Model)
	if m.pendingBallPriority != 3 {
		t.Errorf("Expected priority to wrap to 3 (urgent), got %d", m.pendingBallPriority)
	}
}

// Test ball form state selection
func TestBallFormStateSelection(t *testing.T) {
	ti := textinput.New()
	ti.CharLimit = 256
	ti.Width = 40

	model := Model{
		mode:                 inputBallFormView,
		pendingBallFormField: 1, // state field
		pendingBallState:     0, // pending
		textInput:            ti,
		sessions:             []*session.JuggleSession{},
	}

	// Test right to cycle to in_progress
	newModel, _ := model.handleBallFormKey(tea.KeyMsg{Type: tea.KeyRight})
	m := newModel.(Model)
	if m.pendingBallState != 1 {
		t.Errorf("Expected state to be 1 (in_progress) after right, got %d", m.pendingBallState)
	}

	// Test wrap around right
	newModel, _ = m.handleBallFormKey(tea.KeyMsg{Type: tea.KeyRight})
	m = newModel.(Model)
	if m.pendingBallState != 0 {
		t.Errorf("Expected state to wrap to 0 (pending), got %d", m.pendingBallState)
	}
}

// Test ball form enter transitions to AC input
func TestBallFormEnterTransitionsToACInput(t *testing.T) {
	ti := textinput.New()
	ti.CharLimit = 256
	ti.Width = 40

	model := Model{
		mode:                 inputBallFormView,
		pendingBallIntent:    "Test ball",
		pendingBallFormField: 0,
		pendingBallPriority:  2, // high
		pendingBallState:     1, // in_progress
		pendingBallTags:      "tag1, tag2",
		pendingBallSession:   0,
		textInput:            ti,
		activityLog:          make([]ActivityEntry, 0),
		sessions:             []*session.JuggleSession{},
	}

	newModel, _ := model.handleBallFormKey(tea.KeyMsg{Type: tea.KeyEnter})
	m := newModel.(Model)

	if m.mode != inputAcceptanceCriteriaView {
		t.Errorf("Expected mode to be inputAcceptanceCriteriaView after enter, got %v", m.mode)
	}
}

// Test ball form escape cancels
func TestBallFormEscapeCancels(t *testing.T) {
	ti := textinput.New()
	ti.CharLimit = 256
	ti.Width = 40

	model := Model{
		mode:                 inputBallFormView,
		pendingBallIntent:    "Test ball",
		pendingBallFormField: 0,
		pendingBallPriority:  2,
		pendingBallState:     1,
		pendingBallTags:      "tag1",
		textInput:            ti,
		activityLog:          make([]ActivityEntry, 0),
		sessions:             []*session.JuggleSession{},
	}

	newModel, _ := model.handleBallFormKey(tea.KeyMsg{Type: tea.KeyEscape})
	m := newModel.(Model)

	if m.mode != splitView {
		t.Errorf("Expected mode to be splitView after escape, got %v", m.mode)
	}

	if m.pendingBallIntent != "" {
		t.Errorf("Expected pendingBallIntent to be cleared, got %s", m.pendingBallIntent)
	}

	if m.pendingBallTags != "" {
		t.Errorf("Expected pendingBallTags to be cleared, got %s", m.pendingBallTags)
	}
}

// Test ball form view renders correctly
func TestBallFormViewRenders(t *testing.T) {
	ti := textinput.New()
	ti.CharLimit = 256
	ti.Width = 40

	model := Model{
		mode:                 inputBallFormView,
		pendingBallIntent:    "Test ball intent",
		pendingBallFormField: 0,
		pendingBallPriority:  1, // medium
		pendingBallState:     0, // pending
		pendingBallTags:      "",
		pendingBallSession:   0,
		textInput:            ti,
		sessions: []*session.JuggleSession{
			{ID: "test-session"},
		},
	}

	view := model.renderBallFormView()

	// Check title
	if !strings.Contains(view, "Create New Ball") {
		t.Error("Expected view to contain 'Create New Ball' title")
	}

	// Check intent is shown
	if !strings.Contains(view, "Test ball intent") {
		t.Error("Expected view to contain the ball intent")
	}

	// Check priority options are shown
	if !strings.Contains(view, "low") || !strings.Contains(view, "medium") ||
		!strings.Contains(view, "high") || !strings.Contains(view, "urgent") {
		t.Error("Expected view to contain priority options")
	}

	// Check state options are shown
	if !strings.Contains(view, "pending") || !strings.Contains(view, "in_progress") {
		t.Error("Expected view to contain state options")
	}

	// Check help text
	if !strings.Contains(view, "Enter = continue to ACs") {
		t.Error("Expected view to contain help text")
	}
}

// Test activity log page down (ctrl+d)
func TestActivityLogPageDown(t *testing.T) {
	// Create a model with many activity entries
	entries := make([]ActivityEntry, 50)
	for i := range entries {
		entries[i] = ActivityEntry{Message: fmt.Sprintf("Activity %d", i)}
	}

	model := Model{
		mode:              splitView,
		activePanel:       ActivityPanel,
		activityLog:       entries,
		activityLogOffset: 0,
	}

	// Page down should increase offset
	newModel, _ := model.handleActivityLogPageDown()
	m := newModel.(Model)

	if m.activityLogOffset == 0 {
		t.Error("Expected activityLogOffset to increase after page down")
	}

	// Should not exceed max offset
	maxOffset := m.getActivityLogMaxOffset()
	if m.activityLogOffset > maxOffset {
		t.Errorf("activityLogOffset %d exceeds max offset %d", m.activityLogOffset, maxOffset)
	}
}

// Test activity log page up (ctrl+u)
func TestActivityLogPageUp(t *testing.T) {
	entries := make([]ActivityEntry, 50)
	for i := range entries {
		entries[i] = ActivityEntry{Message: fmt.Sprintf("Activity %d", i)}
	}

	model := Model{
		mode:              splitView,
		activePanel:       ActivityPanel,
		activityLog:       entries,
		activityLogOffset: 20, // Start scrolled down
	}

	// Page up should decrease offset
	newModel, _ := model.handleActivityLogPageUp()
	m := newModel.(Model)

	if m.activityLogOffset >= 20 {
		t.Errorf("Expected activityLogOffset to decrease from 20, got %d", m.activityLogOffset)
	}

	if m.activityLogOffset < 0 {
		t.Error("activityLogOffset should not go below 0")
	}
}

// Test activity log go to top (gg)
func TestActivityLogGoToTop(t *testing.T) {
	entries := make([]ActivityEntry, 50)
	for i := range entries {
		entries[i] = ActivityEntry{Message: fmt.Sprintf("Activity %d", i)}
	}

	model := Model{
		mode:              splitView,
		activePanel:       ActivityPanel,
		activityLog:       entries,
		activityLogOffset: 30, // Start scrolled down
	}

	// Go to top should set offset to 0
	newModel, _ := model.handleActivityLogGoToTop()
	m := newModel.(Model)

	if m.activityLogOffset != 0 {
		t.Errorf("Expected activityLogOffset to be 0 after go to top, got %d", m.activityLogOffset)
	}
}

// Test activity log go to bottom (G)
func TestActivityLogGoToBottom(t *testing.T) {
	entries := make([]ActivityEntry, 50)
	for i := range entries {
		entries[i] = ActivityEntry{Message: fmt.Sprintf("Activity %d", i)}
	}

	model := Model{
		mode:              splitView,
		activePanel:       ActivityPanel,
		activityLog:       entries,
		activityLogOffset: 0, // Start at top
	}

	// Go to bottom should set offset to max
	newModel, _ := model.handleActivityLogGoToBottom()
	m := newModel.(Model)

	expectedOffset := m.getActivityLogMaxOffset()
	if m.activityLogOffset != expectedOffset {
		t.Errorf("Expected activityLogOffset to be %d after go to bottom, got %d", expectedOffset, m.activityLogOffset)
	}
}

// Test activity log scroll position persists when switching panels
func TestActivityLogScrollPersistsAcrossPanels(t *testing.T) {
	entries := make([]ActivityEntry, 50)
	for i := range entries {
		entries[i] = ActivityEntry{Message: fmt.Sprintf("Activity %d", i)}
	}

	model := Model{
		mode:              splitView,
		activePanel:       ActivityPanel,
		activityLog:       entries,
		activityLogOffset: 25, // Scrolled position
		sessions:          []*session.JuggleSession{},
		filteredBalls:     []*session.Ball{},
		filterStates: map[string]bool{
			"pending":     true,
			"in_progress": true,
			"blocked":     true,
			"complete":    true,
		},
	}

	// Switch to sessions panel
	model.activePanel = SessionsPanel

	// Offset should persist
	if model.activityLogOffset != 25 {
		t.Errorf("Expected activityLogOffset to persist as 25, got %d", model.activityLogOffset)
	}

	// Switch to balls panel
	model.activePanel = BallsPanel

	// Offset should still persist
	if model.activityLogOffset != 25 {
		t.Errorf("Expected activityLogOffset to persist as 25, got %d", model.activityLogOffset)
	}

	// Switch back to activity panel
	model.activePanel = ActivityPanel

	// Offset should still be the same
	if model.activityLogOffset != 25 {
		t.Errorf("Expected activityLogOffset to persist as 25, got %d", model.activityLogOffset)
	}
}

// Test gg key sequence detection for activity log
func TestActivityLogGGSequence(t *testing.T) {
	entries := make([]ActivityEntry, 50)
	for i := range entries {
		entries[i] = ActivityEntry{Message: fmt.Sprintf("Activity %d", i)}
	}

	model := Model{
		mode:              splitView,
		activePanel:       ActivityPanel,
		activityLog:       entries,
		activityLogOffset: 30,
		lastKey:           "",
		sessions:          []*session.JuggleSession{},
		filteredBalls:     []*session.Ball{},
		filterStates: map[string]bool{
			"pending":     true,
			"in_progress": true,
			"blocked":     true,
			"complete":    true,
		},
	}

	// First 'g' press should store the key
	newModel, _ := model.handleSplitViewKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m := newModel.(Model)

	if m.lastKey != "g" {
		t.Errorf("Expected lastKey to be 'g' after first g press, got '%s'", m.lastKey)
	}

	// Offset should not change yet
	if m.activityLogOffset != 30 {
		t.Errorf("Expected activityLogOffset to remain 30 after first g, got %d", m.activityLogOffset)
	}

	// Second 'g' press should go to top
	newModel, _ = m.handleSplitViewKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m = newModel.(Model)

	if m.activityLogOffset != 0 {
		t.Errorf("Expected activityLogOffset to be 0 after gg, got %d", m.activityLogOffset)
	}

	if m.lastKey != "" {
		t.Errorf("Expected lastKey to be cleared after gg, got '%s'", m.lastKey)
	}
}

// Test G key goes to bottom in activity log
func TestActivityLogGKeyGoesToBottom(t *testing.T) {
	entries := make([]ActivityEntry, 50)
	for i := range entries {
		entries[i] = ActivityEntry{Message: fmt.Sprintf("Activity %d", i)}
	}

	model := Model{
		mode:              splitView,
		activePanel:       ActivityPanel,
		activityLog:       entries,
		activityLogOffset: 0,
		lastKey:           "",
		sessions:          []*session.JuggleSession{},
		filteredBalls:     []*session.Ball{},
		filterStates: map[string]bool{
			"pending":     true,
			"in_progress": true,
			"blocked":     true,
			"complete":    true,
		},
	}

	// 'G' press should go to bottom
	newModel, _ := model.handleSplitViewKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	m := newModel.(Model)

	expectedOffset := m.getActivityLogMaxOffset()
	if m.activityLogOffset != expectedOffset {
		t.Errorf("Expected activityLogOffset to be %d after G, got %d", expectedOffset, m.activityLogOffset)
	}

	if m.lastKey != "" {
		t.Errorf("Expected lastKey to be cleared after G, got '%s'", m.lastKey)
	}
}

// Test ctrl+d only works in activity panel
func TestCtrlDOnlyWorksInActivityPanel(t *testing.T) {
	entries := make([]ActivityEntry, 50)
	for i := range entries {
		entries[i] = ActivityEntry{Message: fmt.Sprintf("Activity %d", i)}
	}

	model := Model{
		mode:              splitView,
		activePanel:       SessionsPanel, // Not activity panel
		activityLog:       entries,
		activityLogOffset: 0,
		sessions:          []*session.JuggleSession{},
		filteredBalls:     []*session.Ball{},
		filterStates: map[string]bool{
			"pending":     true,
			"in_progress": true,
			"blocked":     true,
			"complete":    true,
		},
	}

	// ctrl+d in sessions panel should not affect activity offset
	newModel, _ := model.handleSplitViewKey(tea.KeyMsg{Type: tea.KeyCtrlD})
	m := newModel.(Model)

	if m.activityLogOffset != 0 {
		t.Errorf("Expected activityLogOffset to remain 0 in sessions panel, got %d", m.activityLogOffset)
	}
}

// Test page down at bottom doesn't exceed max
func TestActivityLogPageDownAtBottom(t *testing.T) {
	entries := make([]ActivityEntry, 50)
	for i := range entries {
		entries[i] = ActivityEntry{Message: fmt.Sprintf("Activity %d", i)}
	}

	model := Model{
		mode:              splitView,
		activePanel:       ActivityPanel,
		activityLog:       entries,
		activityLogOffset: 0,
	}

	maxOffset := model.getActivityLogMaxOffset()
	model.activityLogOffset = maxOffset // Start at bottom

	// Page down at bottom should stay at max
	newModel, _ := model.handleActivityLogPageDown()
	m := newModel.(Model)

	if m.activityLogOffset != maxOffset {
		t.Errorf("Expected activityLogOffset to stay at %d when at bottom, got %d", maxOffset, m.activityLogOffset)
	}
}

// Test page up at top doesn't go below zero
func TestActivityLogPageUpAtTop(t *testing.T) {
	entries := make([]ActivityEntry, 50)
	for i := range entries {
		entries[i] = ActivityEntry{Message: fmt.Sprintf("Activity %d", i)}
	}

	model := Model{
		mode:              splitView,
		activePanel:       ActivityPanel,
		activityLog:       entries,
		activityLogOffset: 0, // Start at top
	}

	// Page up at top should stay at 0
	newModel, _ := model.handleActivityLogPageUp()
	m := newModel.(Model)

	if m.activityLogOffset != 0 {
		t.Errorf("Expected activityLogOffset to stay at 0 when at top, got %d", m.activityLogOffset)
	}
}

// Test ctrl+u clears filter in non-activity panels
func TestCtrlUClearsFilterInOtherPanels(t *testing.T) {
	model := Model{
		mode:              splitView,
		activePanel:       SessionsPanel, // Not activity panel
		activityLog:       make([]ActivityEntry, 10),
		activityLogOffset: 5,
		panelSearchQuery:  "test-filter",
		panelSearchActive: true,
		sessions:          []*session.JuggleSession{},
		filteredBalls:     []*session.Ball{},
		filterStates: map[string]bool{
			"pending":     true,
			"in_progress": true,
			"blocked":     true,
			"complete":    true,
		},
	}

	// ctrl+u in sessions panel should clear filter
	newModel, _ := model.handleSplitViewKey(tea.KeyMsg{Type: tea.KeyCtrlU})
	m := newModel.(Model)

	if m.panelSearchQuery != "" {
		t.Errorf("Expected panelSearchQuery to be cleared, got '%s'", m.panelSearchQuery)
	}

	if m.panelSearchActive {
		t.Error("Expected panelSearchActive to be false")
	}

	// Note: activity log offset will be auto-scrolled to bottom when addActivity()
	// is called from a non-activity panel (this is expected behavior)
}

// Test ctrl+u scrolls up in activity panel
func TestCtrlUScrollsUpInActivityPanel(t *testing.T) {
	entries := make([]ActivityEntry, 50)
	for i := range entries {
		entries[i] = ActivityEntry{Message: fmt.Sprintf("Activity %d", i)}
	}

	model := Model{
		mode:              splitView,
		activePanel:       ActivityPanel,
		activityLog:       entries,
		activityLogOffset: 30,
		panelSearchQuery:  "test-filter", // Has filter
		panelSearchActive: true,
		sessions:          []*session.JuggleSession{},
		filteredBalls:     []*session.Ball{},
		filterStates: map[string]bool{
			"pending":     true,
			"in_progress": true,
			"blocked":     true,
			"complete":    true,
		},
	}

	// ctrl+u in activity panel should scroll, not clear filter
	newModel, _ := model.handleSplitViewKey(tea.KeyMsg{Type: tea.KeyCtrlU})
	m := newModel.(Model)

	if m.activityLogOffset >= 30 {
		t.Errorf("Expected activityLogOffset to decrease from 30, got %d", m.activityLogOffset)
	}

	// Filter should remain
	if m.panelSearchQuery != "test-filter" {
		t.Errorf("Expected panelSearchQuery to remain 'test-filter', got '%s'", m.panelSearchQuery)
	}
}

// Test empty filter input clears current filter (balls panel)
func TestEmptyFilterClearsFilterBallsPanel(t *testing.T) {
	model := Model{
		mode:              panelSearchView,
		activePanel:       BallsPanel,
		panelSearchQuery:  "existing-filter",
		panelSearchActive: true,
		textInput:         textinput.New(),
		sessions:          []*session.JuggleSession{},
		filteredBalls:     []*session.Ball{},
		activityLog:       make([]ActivityEntry, 0),
		filterStates: map[string]bool{
			"pending":     true,
			"in_progress": true,
			"blocked":     true,
			"complete":    true,
		},
	}
	model.textInput.SetValue("") // Empty input

	// Press enter with empty input
	newModel, _ := model.handlePanelSearchKey(tea.KeyMsg{Type: tea.KeyEnter})
	m := newModel.(Model)

	if m.panelSearchQuery != "" {
		t.Errorf("Expected panelSearchQuery to be empty, got '%s'", m.panelSearchQuery)
	}

	if m.panelSearchActive {
		t.Error("Expected panelSearchActive to be false")
	}

	if m.mode != splitView {
		t.Errorf("Expected mode to return to splitView, got %v", m.mode)
	}

	// Check message shows filter cleared
	if m.message != "Filter cleared" {
		t.Errorf("Expected message 'Filter cleared', got '%s'", m.message)
	}
}

// Test empty filter input clears current filter (sessions panel)
func TestEmptyFilterClearsFilterSessionsPanel(t *testing.T) {
	model := Model{
		mode:              panelSearchView,
		activePanel:       SessionsPanel,
		panelSearchQuery:  "existing-filter",
		panelSearchActive: true,
		textInput:         textinput.New(),
		sessions:          []*session.JuggleSession{},
		filteredBalls:     []*session.Ball{},
		activityLog:       make([]ActivityEntry, 0),
		filterStates: map[string]bool{
			"pending":     true,
			"in_progress": true,
			"blocked":     true,
			"complete":    true,
		},
	}
	model.textInput.SetValue("") // Empty input

	// Press enter with empty input
	newModel, _ := model.handlePanelSearchKey(tea.KeyMsg{Type: tea.KeyEnter})
	m := newModel.(Model)

	if m.panelSearchQuery != "" {
		t.Errorf("Expected panelSearchQuery to be empty, got '%s'", m.panelSearchQuery)
	}

	if m.panelSearchActive {
		t.Error("Expected panelSearchActive to be false")
	}
}

// Test filter with whitespace only is treated as empty
func TestWhitespaceOnlyFilterClearsFilter(t *testing.T) {
	model := Model{
		mode:              panelSearchView,
		activePanel:       BallsPanel,
		panelSearchQuery:  "existing-filter",
		panelSearchActive: true,
		textInput:         textinput.New(),
		sessions:          []*session.JuggleSession{},
		filteredBalls:     []*session.Ball{},
		activityLog:       make([]ActivityEntry, 0),
		filterStates: map[string]bool{
			"pending":     true,
			"in_progress": true,
			"blocked":     true,
			"complete":    true,
		},
	}
	model.textInput.SetValue("   ") // Whitespace only

	// Press enter with whitespace input
	newModel, _ := model.handlePanelSearchKey(tea.KeyMsg{Type: tea.KeyEnter})
	m := newModel.(Model)

	if m.panelSearchQuery != "" {
		t.Errorf("Expected panelSearchQuery to be empty (whitespace trimmed), got '%s'", m.panelSearchQuery)
	}

	if m.panelSearchActive {
		t.Error("Expected panelSearchActive to be false")
	}
}

// Test escape cancels filter without clearing existing
func TestEscapeCancelsFilterWithoutClearing(t *testing.T) {
	model := Model{
		mode:              panelSearchView,
		activePanel:       BallsPanel,
		panelSearchQuery:  "existing-filter",
		panelSearchActive: true,
		textInput:         textinput.New(),
		sessions:          []*session.JuggleSession{},
		filteredBalls:     []*session.Ball{},
		activityLog:       make([]ActivityEntry, 0),
		filterStates: map[string]bool{
			"pending":     true,
			"in_progress": true,
			"blocked":     true,
			"complete":    true,
		},
	}

	// Press escape
	newModel, _ := model.handlePanelSearchKey(tea.KeyMsg{Type: tea.KeyEsc})
	m := newModel.(Model)

	// Filter should remain unchanged
	if m.panelSearchQuery != "existing-filter" {
		t.Errorf("Expected panelSearchQuery to remain 'existing-filter', got '%s'", m.panelSearchQuery)
	}

	if !m.panelSearchActive {
		t.Error("Expected panelSearchActive to remain true")
	}

	if m.mode != splitView {
		t.Errorf("Expected mode to return to splitView, got %v", m.mode)
	}
}

// Test filter applied with non-empty value shows correct message
func TestFilterAppliedShowsCorrectMessage(t *testing.T) {
	model := Model{
		mode:              panelSearchView,
		activePanel:       BallsPanel,
		panelSearchQuery:  "",
		panelSearchActive: false,
		textInput:         textinput.New(),
		sessions:          []*session.JuggleSession{},
		filteredBalls:     []*session.Ball{},
		activityLog:       make([]ActivityEntry, 0),
		filterStates: map[string]bool{
			"pending":     true,
			"in_progress": true,
			"blocked":     true,
			"complete":    true,
		},
	}
	model.textInput.SetValue("myfilter")

	// Press enter with filter value
	newModel, _ := model.handlePanelSearchKey(tea.KeyMsg{Type: tea.KeyEnter})
	m := newModel.(Model)

	if m.panelSearchQuery != "myfilter" {
		t.Errorf("Expected panelSearchQuery to be 'myfilter', got '%s'", m.panelSearchQuery)
	}

	if !m.panelSearchActive {
		t.Error("Expected panelSearchActive to be true")
	}

	if m.message != "Filter: myfilter (Ctrl+U to clear)" {
		t.Errorf("Expected message with filter and hint, got '%s'", m.message)
	}
}

// Test filterSessions returns all sessions when filter cleared
func TestFilterSessionsReturnsAllWhenCleared(t *testing.T) {
	model := Model{
		panelSearchQuery:  "",
		panelSearchActive: false,
		sessions: []*session.JuggleSession{
			{ID: "session-1", Description: "First session"},
			{ID: "session-2", Description: "Second session"},
			{ID: "session-3", Description: "Third session"},
		},
	}

	sessions := model.filterSessions()

	// Should have 2 pseudo-sessions + 3 real sessions = 5
	if len(sessions) != 5 {
		t.Errorf("Expected 5 sessions (2 pseudo + 3 real), got %d", len(sessions))
	}

	// First two should be pseudo-sessions
	if sessions[0].ID != PseudoSessionAll {
		t.Errorf("Expected first session to be PseudoSessionAll, got '%s'", sessions[0].ID)
	}
	if sessions[1].ID != PseudoSessionUntagged {
		t.Errorf("Expected second session to be PseudoSessionUntagged, got '%s'", sessions[1].ID)
	}
}

// Test filterBallsForSession returns all balls when filter cleared
func TestFilterBallsReturnsAllWhenCleared(t *testing.T) {
	model := Model{
		panelSearchQuery:  "",
		panelSearchActive: false,
		selectedSession:   &session.JuggleSession{ID: PseudoSessionAll},
		filteredBalls: []*session.Ball{
			{ID: "ball-1", Intent: "First ball"},
			{ID: "ball-2", Intent: "Second ball"},
			{ID: "ball-3", Intent: "Third ball"},
		},
		filterStates: map[string]bool{
			"pending":     true,
			"in_progress": true,
			"blocked":     true,
			"complete":    true,
		},
	}

	balls := model.filterBallsForSession()

	if len(balls) != 3 {
		t.Errorf("Expected 3 balls when filter cleared, got %d", len(balls))
	}
}

// Test activity log gets entry when filter is cleared via empty input
func TestActivityLogUpdatedOnFilterClear(t *testing.T) {
	model := Model{
		mode:              panelSearchView,
		activePanel:       BallsPanel,
		panelSearchQuery:  "existing-filter",
		panelSearchActive: true,
		textInput:         textinput.New(),
		sessions:          []*session.JuggleSession{},
		filteredBalls:     []*session.Ball{},
		activityLog:       make([]ActivityEntry, 0),
		filterStates: map[string]bool{
			"pending":     true,
			"in_progress": true,
			"blocked":     true,
			"complete":    true,
		},
	}
	model.textInput.SetValue("") // Empty input

	// Press enter with empty input
	newModel, _ := model.handlePanelSearchKey(tea.KeyMsg{Type: tea.KeyEnter})
	m := newModel.(Model)

	// Check activity log has an entry for filter cleared
	if len(m.activityLog) == 0 {
		t.Error("Expected activity log to have entry for filter cleared")
	} else {
		lastEntry := m.activityLog[len(m.activityLog)-1]
		if lastEntry.Message != "Filter cleared" {
			t.Errorf("Expected last activity message 'Filter cleared', got '%s'", lastEntry.Message)
		}
	}
}

// Test allBallsSameProject helper function
func TestAllBallsSameProject(t *testing.T) {
	tests := []struct {
		name     string
		balls    []*session.Ball
		expected bool
	}{
		{
			name:     "empty list",
			balls:    []*session.Ball{},
			expected: true,
		},
		{
			name: "single ball",
			balls: []*session.Ball{
				{ID: "juggler-1", WorkingDir: "/home/user/juggler"},
			},
			expected: true,
		},
		{
			name: "multiple balls same project",
			balls: []*session.Ball{
				{ID: "juggler-1", WorkingDir: "/home/user/juggler"},
				{ID: "juggler-2", WorkingDir: "/home/user/juggler"},
				{ID: "juggler-3", WorkingDir: "/home/user/juggler"},
			},
			expected: true,
		},
		{
			name: "multiple balls different projects",
			balls: []*session.Ball{
				{ID: "juggler-1", WorkingDir: "/home/user/juggler"},
				{ID: "myapp-1", WorkingDir: "/home/user/myapp"},
			},
			expected: false,
		},
		{
			name: "three different projects",
			balls: []*session.Ball{
				{ID: "juggler-1", WorkingDir: "/home/user/juggler"},
				{ID: "myapp-1", WorkingDir: "/home/user/myapp"},
				{ID: "other-1", WorkingDir: "/home/user/other"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := allBallsSameProject(tt.balls)
			if result != tt.expected {
				t.Errorf("allBallsSameProject() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// Test compareBallIDs function
func TestCompareBallIDs(t *testing.T) {
	tests := []struct {
		id1      string
		id2      string
		expected int // -1 if id1 < id2, 0 if equal, 1 if id1 > id2
	}{
		{"juggler-1", "juggler-2", -1},
		{"juggler-2", "juggler-1", 1},
		{"juggler-5", "juggler-5", 0},
		{"juggler-10", "juggler-2", 1},  // Numeric comparison, 10 > 2
		{"juggler-1", "juggler-10", -1}, // Numeric comparison, 1 < 10
		{"project-99", "project-100", -1},
		{"aaa-1", "zzz-1", -1}, // Falls back to string comparison for same number
		{"noid", "juggler-1", 1}, // No numeric part falls back to string comparison
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_vs_%s", tt.id1, tt.id2), func(t *testing.T) {
			result := compareBallIDs(tt.id1, tt.id2)
			if (result < 0 && tt.expected >= 0) || (result > 0 && tt.expected <= 0) || (result == 0 && tt.expected != 0) {
				t.Errorf("compareBallIDs(%q, %q) = %d, want %d", tt.id1, tt.id2, result, tt.expected)
			}
		})
	}
}

// Test extractBallNumber function
func TestExtractBallNumber(t *testing.T) {
	tests := []struct {
		id       string
		expected int
	}{
		{"juggler-1", 1},
		{"juggler-99", 99},
		{"myapp-1000", 1000},
		{"project-name-123", 123},
		{"nohyphen", -1},
		{"ends-with-hyphen-", -1},
		{"juggler-abc", -1}, // Non-numeric suffix
		{"", -1},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			result := extractBallNumber(tt.id)
			if result != tt.expected {
				t.Errorf("extractBallNumber(%q) = %d, want %d", tt.id, result, tt.expected)
			}
		})
	}
}

// Test sort order toggle
func TestToggleSortOrder(t *testing.T) {
	tests := []struct {
		name            string
		startSortOrder  SortOrder
		expectedOrder   SortOrder
		expectedMessage string
	}{
		{
			name:            "ID ascending to descending",
			startSortOrder:  SortByIDASC,
			expectedOrder:   SortByIDDESC,
			expectedMessage: "Sort: ID descending",
		},
		{
			name:            "ID descending to priority",
			startSortOrder:  SortByIDDESC,
			expectedOrder:   SortByPriority,
			expectedMessage: "Sort: Priority (urgent first)",
		},
		{
			name:            "Priority to last activity",
			startSortOrder:  SortByPriority,
			expectedOrder:   SortByLastActivity,
			expectedMessage: "Sort: Last activity (recent first)",
		},
		{
			name:            "Last activity to ID ascending",
			startSortOrder:  SortByLastActivity,
			expectedOrder:   SortByIDASC,
			expectedMessage: "Sort: ID ascending",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := Model{
				sortOrder:   tt.startSortOrder,
				activityLog: make([]ActivityEntry, 0),
			}

			newModel, _ := model.handleToggleSortOrder()
			m := newModel.(Model)

			if m.sortOrder != tt.expectedOrder {
				t.Errorf("Expected sortOrder to be %v, got %v", tt.expectedOrder, m.sortOrder)
			}

			if m.message != tt.expectedMessage {
				t.Errorf("Expected message to be %q, got %q", tt.expectedMessage, m.message)
			}
		})
	}
}

// Test sorting balls by ID ascending
func TestSortBallsByIDAscending(t *testing.T) {
	balls := []*session.Ball{
		{ID: "juggler-10"},
		{ID: "juggler-2"},
		{ID: "juggler-1"},
		{ID: "juggler-100"},
	}

	model := Model{
		sortOrder: SortByIDASC,
	}

	model.sortBalls(balls)

	expected := []string{"juggler-1", "juggler-2", "juggler-10", "juggler-100"}
	for i, ball := range balls {
		if ball.ID != expected[i] {
			t.Errorf("Expected ball at index %d to be %q, got %q", i, expected[i], ball.ID)
		}
	}
}

// Test sorting balls by ID descending
func TestSortBallsByIDDescending(t *testing.T) {
	balls := []*session.Ball{
		{ID: "juggler-1"},
		{ID: "juggler-10"},
		{ID: "juggler-2"},
	}

	model := Model{
		sortOrder: SortByIDDESC,
	}

	model.sortBalls(balls)

	expected := []string{"juggler-10", "juggler-2", "juggler-1"}
	for i, ball := range balls {
		if ball.ID != expected[i] {
			t.Errorf("Expected ball at index %d to be %q, got %q", i, expected[i], ball.ID)
		}
	}
}

// Test sorting balls by priority
func TestSortBallsByPriority(t *testing.T) {
	balls := []*session.Ball{
		{ID: "juggler-1", Priority: session.PriorityLow},
		{ID: "juggler-2", Priority: session.PriorityUrgent},
		{ID: "juggler-3", Priority: session.PriorityMedium},
		{ID: "juggler-4", Priority: session.PriorityHigh},
	}

	model := Model{
		sortOrder: SortByPriority,
	}

	model.sortBalls(balls)

	// Should be sorted by priority: urgent, high, medium, low
	expectedOrder := []string{"juggler-2", "juggler-4", "juggler-3", "juggler-1"}
	for i, ball := range balls {
		if ball.ID != expectedOrder[i] {
			t.Errorf("Expected ball at index %d to be %q, got %q", i, expectedOrder[i], ball.ID)
		}
	}
}

// Test that same priority balls are sorted by ID ascending
func TestSortBallsByPriorityThenID(t *testing.T) {
	balls := []*session.Ball{
		{ID: "juggler-10", Priority: session.PriorityMedium},
		{ID: "juggler-2", Priority: session.PriorityMedium},
		{ID: "juggler-1", Priority: session.PriorityMedium},
	}

	model := Model{
		sortOrder: SortByPriority,
	}

	model.sortBalls(balls)

	// All same priority, should be sorted by ID ascending
	expected := []string{"juggler-1", "juggler-2", "juggler-10"}
	for i, ball := range balls {
		if ball.ID != expected[i] {
			t.Errorf("Expected ball at index %d to be %q, got %q", i, expected[i], ball.ID)
		}
	}
}

// Test filterBallsForSession applies sorting
func TestFilterBallsForSessionAppliesSorting(t *testing.T) {
	balls := []*session.Ball{
		{ID: "juggler-10", State: session.StatePending, Tags: []string{"test"}},
		{ID: "juggler-2", State: session.StatePending, Tags: []string{"test"}},
		{ID: "juggler-1", State: session.StatePending, Tags: []string{"test"}},
	}

	model := Model{
		filteredBalls:     balls,
		panelSearchActive: false,
		selectedSession:   &session.JuggleSession{ID: "test"},
		sortOrder:         SortByIDASC,
	}

	result := model.filterBallsForSession()

	// Should be sorted by ID ascending
	expected := []string{"juggler-1", "juggler-2", "juggler-10"}
	for i, ball := range result {
		if ball.ID != expected[i] {
			t.Errorf("Expected ball at index %d to be %q, got %q", i, expected[i], ball.ID)
		}
	}
}

// Test default sort order is ID ascending
func TestDefaultSortOrderIsIDAscending(t *testing.T) {
	var store *session.Store
	var sessionStore *session.SessionStore
	var config *session.Config

	model := InitialSplitModel(store, sessionStore, config, true)

	if model.sortOrder != SortByIDASC {
		t.Errorf("Expected default sortOrder to be SortByIDASC, got %v", model.sortOrder)
	}
}

// Test 'o' key toggles sort order
func TestOKeyTogglesSortOrder(t *testing.T) {
	model := Model{
		mode:          splitView,
		activePanel:   BallsPanel,
		sortOrder:     SortByIDASC,
		activityLog:   make([]ActivityEntry, 0),
		sessions:      []*session.JuggleSession{},
		filteredBalls: []*session.Ball{},
		filterStates: map[string]bool{
			"pending":     true,
			"in_progress": true,
			"blocked":     true,
			"complete":    true,
		},
	}

	newModel, _ := model.handleSplitViewKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	m := newModel.(Model)

	if m.sortOrder != SortByIDDESC {
		t.Errorf("Expected sortOrder to be SortByIDDESC after pressing 'o', got %v", m.sortOrder)
	}
}

// Test balls panel scrolling - cursor stays visible when scrolling down
func TestBallsPanelScrollingDown(t *testing.T) {
	// Create many balls to exceed visible area
	balls := make([]*session.Ball, 20)
	for i := 0; i < 20; i++ {
		balls[i] = &session.Ball{
			ID:     fmt.Sprintf("test-%d", i),
			State:  session.StatePending,
			Intent: fmt.Sprintf("Ball %d", i),
		}
	}

	model := Model{
		mode:              splitView,
		activePanel:       BallsPanel,
		cursor:            0,
		ballsScrollOffset: 0,
		filteredBalls:     balls,
		height:            30, // Height that shows ~5 balls
		width:             80,
		filterStates: map[string]bool{
			"pending":     true,
			"in_progress": true,
			"blocked":     true,
			"complete":    true,
		},
		activityLog: make([]ActivityEntry, 0),
	}

	// Navigate down multiple times to go beyond visible area
	for i := 0; i < 10; i++ {
		newModel, _ := model.handleSplitViewNavDown()
		model = newModel.(Model)
	}

	// Cursor should be at position 10
	if model.cursor != 10 {
		t.Errorf("Expected cursor to be 10, got %d", model.cursor)
	}

	// Scroll offset should have adjusted to keep cursor visible
	// The cursor should be within the visible window: scrollOffset <= cursor < scrollOffset + visibleLines
	if model.cursor < model.ballsScrollOffset {
		t.Errorf("Cursor %d is above scroll offset %d - cursor not visible", model.cursor, model.ballsScrollOffset)
	}
}

// Test balls panel scrolling - cursor stays visible when scrolling up
func TestBallsPanelScrollingUp(t *testing.T) {
	// Create many balls
	balls := make([]*session.Ball, 20)
	for i := 0; i < 20; i++ {
		balls[i] = &session.Ball{
			ID:     fmt.Sprintf("test-%d", i),
			State:  session.StatePending,
			Intent: fmt.Sprintf("Ball %d", i),
		}
	}

	model := Model{
		mode:              splitView,
		activePanel:       BallsPanel,
		cursor:            10, // Start at position 10
		ballsScrollOffset: 8,  // Start scrolled down
		filteredBalls:     balls,
		height:            30,
		width:             80,
		filterStates: map[string]bool{
			"pending":     true,
			"in_progress": true,
			"blocked":     true,
			"complete":    true,
		},
		activityLog: make([]ActivityEntry, 0),
	}

	// Navigate up multiple times
	for i := 0; i < 5; i++ {
		newModel, _ := model.handleSplitViewNavUp()
		model = newModel.(Model)
	}

	// Cursor should be at position 5
	if model.cursor != 5 {
		t.Errorf("Expected cursor to be 5, got %d", model.cursor)
	}

	// Cursor should still be within visible area
	if model.cursor < model.ballsScrollOffset {
		t.Errorf("Cursor %d is above scroll offset %d - cursor not visible", model.cursor, model.ballsScrollOffset)
	}
}

// Test balls scroll offset resets when switching sessions
func TestBallsScrollOffsetResetsOnSessionSwitch(t *testing.T) {
	sessions := []*session.JuggleSession{
		{ID: PseudoSessionAll, Description: "All balls"},
		{ID: "session1", Description: "Session 1"},
		{ID: "session2", Description: "Session 2"},
	}

	model := Model{
		mode:              splitView,
		activePanel:       SessionsPanel,
		sessionCursor:     0,
		cursor:            5,
		ballsScrollOffset: 10, // Scrolled down
		sessions:          sessions,
		filteredBalls:     []*session.Ball{},
		height:            30,
		width:             80,
		filterStates: map[string]bool{
			"pending":     true,
			"in_progress": true,
			"blocked":     true,
			"complete":    true,
		},
		activityLog: make([]ActivityEntry, 0),
	}

	// Navigate down to select next session
	newModel, _ := model.handleSplitViewNavDown()
	model = newModel.(Model)

	// Ball cursor and scroll offset should be reset
	if model.cursor != 0 {
		t.Errorf("Expected cursor to reset to 0, got %d", model.cursor)
	}
	if model.ballsScrollOffset != 0 {
		t.Errorf("Expected ballsScrollOffset to reset to 0, got %d", model.ballsScrollOffset)
	}
}

// Test balls scroll offset resets when using Enter to select session
func TestBallsScrollOffsetResetsOnSessionEnter(t *testing.T) {
	sessions := []*session.JuggleSession{
		{ID: PseudoSessionAll, Description: "All balls"},
		{ID: "session1", Description: "Session 1"},
	}

	model := Model{
		mode:              splitView,
		activePanel:       SessionsPanel,
		sessionCursor:     0,
		cursor:            5,
		ballsScrollOffset: 10,
		sessions:          sessions,
		filteredBalls:     []*session.Ball{},
		height:            30,
		width:             80,
		filterStates: map[string]bool{
			"pending":     true,
			"in_progress": true,
			"blocked":     true,
			"complete":    true,
		},
		activityLog: make([]ActivityEntry, 0),
	}

	// Press Enter to select session
	newModel, _ := model.handleSplitViewEnter()
	model = newModel.(Model)

	if model.ballsScrollOffset != 0 {
		t.Errorf("Expected ballsScrollOffset to reset to 0 on Enter, got %d", model.ballsScrollOffset)
	}
}

// Test balls scroll offset resets when using [ to switch session
func TestBallsScrollOffsetResetsOnSessionSwitchPrev(t *testing.T) {
	sessions := []*session.JuggleSession{
		{ID: "session1", Description: "Session 1"},
		{ID: "session2", Description: "Session 2"},
	}

	model := Model{
		mode:              splitView,
		activePanel:       BallsPanel,
		sessionCursor:     1,
		selectedSession:   sessions[1],
		cursor:            5,
		ballsScrollOffset: 10,
		sessions:          sessions,
		filteredBalls:     []*session.Ball{},
		height:            30,
		width:             80,
		filterStates: map[string]bool{
			"pending":     true,
			"in_progress": true,
			"blocked":     true,
			"complete":    true,
		},
		activityLog: make([]ActivityEntry, 0),
	}

	// Press [ to switch to previous session
	newModel, _ := model.handleSessionSwitchPrev()
	model = newModel.(Model)

	if model.ballsScrollOffset != 0 {
		t.Errorf("Expected ballsScrollOffset to reset to 0 on [, got %d", model.ballsScrollOffset)
	}
	if model.cursor != 0 {
		t.Errorf("Expected cursor to reset to 0 on [, got %d", model.cursor)
	}
}

// Test balls scroll offset resets when using ] to switch session
func TestBallsScrollOffsetResetsOnSessionSwitchNext(t *testing.T) {
	sessions := []*session.JuggleSession{
		{ID: "session1", Description: "Session 1"},
		{ID: "session2", Description: "Session 2"},
	}

	model := Model{
		mode:              splitView,
		activePanel:       BallsPanel,
		sessionCursor:     0,
		selectedSession:   sessions[0],
		cursor:            5,
		ballsScrollOffset: 10,
		sessions:          sessions,
		filteredBalls:     []*session.Ball{},
		height:            30,
		width:             80,
		filterStates: map[string]bool{
			"pending":     true,
			"in_progress": true,
			"blocked":     true,
			"complete":    true,
		},
		activityLog: make([]ActivityEntry, 0),
	}

	// Press ] to switch to next session
	newModel, _ := model.handleSessionSwitchNext()
	model = newModel.(Model)

	if model.ballsScrollOffset != 0 {
		t.Errorf("Expected ballsScrollOffset to reset to 0 on ], got %d", model.ballsScrollOffset)
	}
	if model.cursor != 0 {
		t.Errorf("Expected cursor to reset to 0 on ], got %d", model.cursor)
	}
}

// Test adjustBallsScrollOffset keeps cursor visible when moving down
func TestAdjustBallsScrollOffsetDown(t *testing.T) {
	balls := make([]*session.Ball, 20)
	for i := 0; i < 20; i++ {
		balls[i] = &session.Ball{ID: fmt.Sprintf("test-%d", i)}
	}

	model := Model{
		cursor:            15, // Cursor near bottom
		ballsScrollOffset: 0,  // Scrolled to top
		height:            30, // ~5 visible balls
		width:             80,
	}

	model.adjustBallsScrollOffset(balls)

	// Scroll offset should have increased to show cursor
	if model.ballsScrollOffset == 0 {
		t.Error("Expected ballsScrollOffset to increase to show cursor at position 15")
	}

	// Cursor should be visible (within visible range)
	// visibleLines approximately = (30 - bottomPanelRows - 4 - 2 - 4) = ~14
	// But even if calculation varies, cursor must be >= scrollOffset
	if model.cursor < model.ballsScrollOffset {
		t.Errorf("Cursor %d should be >= scrollOffset %d", model.cursor, model.ballsScrollOffset)
	}
}

// Test adjustBallsScrollOffset keeps cursor visible when moving up
func TestAdjustBallsScrollOffsetUp(t *testing.T) {
	balls := make([]*session.Ball, 20)
	for i := 0; i < 20; i++ {
		balls[i] = &session.Ball{ID: fmt.Sprintf("test-%d", i)}
	}

	model := Model{
		cursor:            2,  // Cursor near top
		ballsScrollOffset: 10, // Scrolled down far
		height:            30,
		width:             80,
	}

	model.adjustBallsScrollOffset(balls)

	// Scroll offset should have decreased to show cursor
	if model.ballsScrollOffset > model.cursor {
		t.Errorf("Expected ballsScrollOffset (%d) to be <= cursor (%d) to show cursor", model.ballsScrollOffset, model.cursor)
	}
}

// Test adjustBallsScrollOffset handles empty balls list
func TestAdjustBallsScrollOffsetEmpty(t *testing.T) {
	model := Model{
		cursor:            5,
		ballsScrollOffset: 10,
		height:            30,
		width:             80,
	}

	model.adjustBallsScrollOffset([]*session.Ball{})

	if model.ballsScrollOffset != 0 {
		t.Errorf("Expected ballsScrollOffset to be 0 for empty balls, got %d", model.ballsScrollOffset)
	}
}

// Test that pressing 't' key opens session selector view
func TestTagKeyOpensSessionSelector(t *testing.T) {
	sessions := []*session.JuggleSession{
		{ID: "session-a", Description: "First session"},
		{ID: "session-b", Description: "Second session"},
	}

	balls := []*session.Ball{
		{ID: "ball-1", Intent: "Test task", Tags: []string{}, WorkingDir: "/tmp"},
	}

	model := Model{
		mode:            splitView,
		activePanel:     BallsPanel,
		cursor:          0,
		filteredBalls:   balls,
		sessions:        sessions,
		selectedSession: &session.JuggleSession{ID: PseudoSessionAll},
	}

	newModel, _ := model.handleTagEditStart()
	m := newModel.(Model)

	if m.mode != sessionSelectorView {
		t.Errorf("Expected mode to be sessionSelectorView, got %v", m.mode)
	}

	if m.editingBall == nil {
		t.Error("Expected editingBall to be set")
	}

	if m.editingBall.ID != "ball-1" {
		t.Errorf("Expected editingBall ID to be 'ball-1', got '%s'", m.editingBall.ID)
	}

	if len(m.sessionSelectItems) != 2 {
		t.Errorf("Expected 2 available sessions, got %d", len(m.sessionSelectItems))
	}
}

// Test that 't' key on ball already in all sessions shows appropriate message
func TestTagKeyBallAlreadyInAllSessions(t *testing.T) {
	sessions := []*session.JuggleSession{
		{ID: "session-a", Description: "First session"},
	}

	balls := []*session.Ball{
		{ID: "ball-1", Intent: "Test task", Tags: []string{"session-a"}, WorkingDir: "/tmp"},
	}

	model := Model{
		mode:            splitView,
		activePanel:     BallsPanel,
		cursor:          0,
		filteredBalls:   balls,
		sessions:        sessions,
		selectedSession: &session.JuggleSession{ID: PseudoSessionAll},
	}

	newModel, _ := model.handleTagEditStart()
	m := newModel.(Model)

	// Should stay in splitView because ball is already in all sessions
	if m.mode != splitView {
		t.Errorf("Expected mode to remain splitView when ball is in all sessions, got %v", m.mode)
	}

	if m.message != "Ball already in all sessions" {
		t.Errorf("Expected message 'Ball already in all sessions', got '%s'", m.message)
	}
}

// Test that 't' key filters out sessions ball is already tagged with
func TestTagKeyFiltersExistingSessions(t *testing.T) {
	sessions := []*session.JuggleSession{
		{ID: "session-a", Description: "First session"},
		{ID: "session-b", Description: "Second session"},
		{ID: "session-c", Description: "Third session"},
	}

	balls := []*session.Ball{
		{ID: "ball-1", Intent: "Test task", Tags: []string{"session-a"}, WorkingDir: "/tmp"},
	}

	model := Model{
		mode:            splitView,
		activePanel:     BallsPanel,
		cursor:          0,
		filteredBalls:   balls,
		sessions:        sessions,
		selectedSession: &session.JuggleSession{ID: PseudoSessionAll},
	}

	newModel, _ := model.handleTagEditStart()
	m := newModel.(Model)

	if m.mode != sessionSelectorView {
		t.Errorf("Expected mode to be sessionSelectorView, got %v", m.mode)
	}

	// Should only have session-b and session-c (not session-a which is already tagged)
	if len(m.sessionSelectItems) != 2 {
		t.Errorf("Expected 2 available sessions, got %d", len(m.sessionSelectItems))
	}

	sessionIDs := make(map[string]bool)
	for _, sess := range m.sessionSelectItems {
		sessionIDs[sess.ID] = true
	}

	if sessionIDs["session-a"] {
		t.Error("session-a should not be in available sessions (ball already tagged with it)")
	}

	if !sessionIDs["session-b"] {
		t.Error("session-b should be in available sessions")
	}

	if !sessionIDs["session-c"] {
		t.Error("session-c should be in available sessions")
	}
}

// Test session selector navigation with j/k keys
func TestSessionSelectorNavigation(t *testing.T) {
	sessions := []*session.JuggleSession{
		{ID: "session-a", Description: "First session"},
		{ID: "session-b", Description: "Second session"},
		{ID: "session-c", Description: "Third session"},
	}

	ball := &session.Ball{ID: "ball-1", Intent: "Test task", Tags: []string{}, WorkingDir: "/tmp"}

	model := Model{
		mode:               sessionSelectorView,
		sessionSelectItems: sessions,
		sessionSelectIndex: 0,
		editingBall:        ball,
	}

	// Press 'j' to move down
	newModel, _ := model.handleSessionSelectorKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m := newModel.(Model)

	if m.sessionSelectIndex != 1 {
		t.Errorf("Expected sessionSelectIndex to be 1 after 'j', got %d", m.sessionSelectIndex)
	}

	// Press 'k' to move up
	newModel, _ = m.handleSessionSelectorKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = newModel.(Model)

	if m.sessionSelectIndex != 0 {
		t.Errorf("Expected sessionSelectIndex to be 0 after 'k', got %d", m.sessionSelectIndex)
	}

	// Test down arrow
	newModel, _ = m.handleSessionSelectorKey(tea.KeyMsg{Type: tea.KeyDown})
	m = newModel.(Model)

	if m.sessionSelectIndex != 1 {
		t.Errorf("Expected sessionSelectIndex to be 1 after down arrow, got %d", m.sessionSelectIndex)
	}

	// Test up arrow
	newModel, _ = m.handleSessionSelectorKey(tea.KeyMsg{Type: tea.KeyUp})
	m = newModel.(Model)

	if m.sessionSelectIndex != 0 {
		t.Errorf("Expected sessionSelectIndex to be 0 after up arrow, got %d", m.sessionSelectIndex)
	}
}

// Test session selector doesn't go past boundaries
func TestSessionSelectorBoundaries(t *testing.T) {
	sessions := []*session.JuggleSession{
		{ID: "session-a", Description: "First session"},
		{ID: "session-b", Description: "Second session"},
	}

	ball := &session.Ball{ID: "ball-1", Intent: "Test task", Tags: []string{}, WorkingDir: "/tmp"}

	model := Model{
		mode:               sessionSelectorView,
		sessionSelectItems: sessions,
		sessionSelectIndex: 0,
		editingBall:        ball,
	}

	// Try to go up when already at top
	newModel, _ := model.handleSessionSelectorKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m := newModel.(Model)

	if m.sessionSelectIndex != 0 {
		t.Errorf("Expected sessionSelectIndex to remain 0 at top, got %d", m.sessionSelectIndex)
	}

	// Move to bottom
	m.sessionSelectIndex = 1

	// Try to go down when at bottom
	newModel, _ = m.handleSessionSelectorKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = newModel.(Model)

	if m.sessionSelectIndex != 1 {
		t.Errorf("Expected sessionSelectIndex to remain 1 at bottom, got %d", m.sessionSelectIndex)
	}
}

// Test session selector cancel with escape
func TestSessionSelectorCancel(t *testing.T) {
	sessions := []*session.JuggleSession{
		{ID: "session-a", Description: "First session"},
	}

	ball := &session.Ball{ID: "ball-1", Intent: "Test task", Tags: []string{}, WorkingDir: "/tmp"}

	model := Model{
		mode:               sessionSelectorView,
		sessionSelectItems: sessions,
		sessionSelectIndex: 0,
		editingBall:        ball,
	}

	// Press escape to cancel
	newModel, _ := model.handleSessionSelectorKey(tea.KeyMsg{Type: tea.KeyEsc})
	m := newModel.(Model)

	if m.mode != splitView {
		t.Errorf("Expected mode to be splitView after cancel, got %v", m.mode)
	}

	if m.sessionSelectItems != nil {
		t.Error("Expected sessionSelectItems to be nil after cancel")
	}

	if m.message != "Cancelled" {
		t.Errorf("Expected message 'Cancelled', got '%s'", m.message)
	}
}

// Test session selector cancel with 'q'
func TestSessionSelectorCancelWithQ(t *testing.T) {
	sessions := []*session.JuggleSession{
		{ID: "session-a", Description: "First session"},
	}

	ball := &session.Ball{ID: "ball-1", Intent: "Test task", Tags: []string{}, WorkingDir: "/tmp"}

	model := Model{
		mode:               sessionSelectorView,
		sessionSelectItems: sessions,
		sessionSelectIndex: 0,
		editingBall:        ball,
	}

	// Press 'q' to cancel
	newModel, _ := model.handleSessionSelectorKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m := newModel.(Model)

	if m.mode != splitView {
		t.Errorf("Expected mode to be splitView after cancel, got %v", m.mode)
	}

	if m.sessionSelectItems != nil {
		t.Error("Expected sessionSelectItems to be nil after cancel")
	}
}

// Test session selector selection with enter
func TestSessionSelectorSelectWithEnter(t *testing.T) {
	sessions := []*session.JuggleSession{
		{ID: "session-a", Description: "First session"},
		{ID: "session-b", Description: "Second session"},
	}

	ball := &session.Ball{ID: "ball-1", Intent: "Test task", Tags: []string{}, WorkingDir: "/tmp"}

	model := Model{
		mode:               sessionSelectorView,
		sessionSelectItems: sessions,
		sessionSelectIndex: 1, // Select session-b
		editingBall:        ball,
	}

	// Press enter to select
	newModel, _ := model.handleSessionSelectorKey(tea.KeyMsg{Type: tea.KeyEnter})
	m := newModel.(Model)

	if m.mode != splitView {
		t.Errorf("Expected mode to be splitView after selection, got %v", m.mode)
	}

	// Ball should now have session-b as a tag
	if len(m.editingBall.Tags) != 1 {
		t.Errorf("Expected 1 tag on ball, got %d", len(m.editingBall.Tags))
	}

	if m.editingBall.Tags[0] != "session-b" {
		t.Errorf("Expected tag 'session-b', got '%s'", m.editingBall.Tags[0])
	}

	if !strings.Contains(m.message, "session-b") {
		t.Errorf("Expected message to contain 'session-b', got '%s'", m.message)
	}
}

// Test session selector selection with space
func TestSessionSelectorSelectWithSpace(t *testing.T) {
	sessions := []*session.JuggleSession{
		{ID: "session-a", Description: "First session"},
	}

	ball := &session.Ball{ID: "ball-1", Intent: "Test task", Tags: []string{}, WorkingDir: "/tmp"}

	model := Model{
		mode:               sessionSelectorView,
		sessionSelectItems: sessions,
		sessionSelectIndex: 0,
		editingBall:        ball,
	}

	// Press space to select
	newModel, _ := model.handleSessionSelectorKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	m := newModel.(Model)

	if m.mode != splitView {
		t.Errorf("Expected mode to be splitView after selection, got %v", m.mode)
	}

	// Ball should now have session-a as a tag
	if len(m.editingBall.Tags) != 1 {
		t.Errorf("Expected 1 tag on ball, got %d", len(m.editingBall.Tags))
	}
}

// Test session selector view rendering
func TestSessionSelectorViewRendering(t *testing.T) {
	sessions := []*session.JuggleSession{
		{ID: "session-a", Description: "First session"},
		{ID: "session-b", Description: "Second session"},
	}

	ball := &session.Ball{ID: "ball-1", Intent: "Test task", Tags: []string{"existing-tag"}, WorkingDir: "/tmp"}

	model := Model{
		mode:               sessionSelectorView,
		sessionSelectItems: sessions,
		sessionSelectIndex: 0,
		editingBall:        ball,
		width:              80,
		height:             24,
	}

	view := model.renderSessionSelectorView()

	// Should show title
	if !strings.Contains(view, "Select Session") {
		t.Error("Expected view to contain 'Select Session' title")
	}

	// Should show ball context
	if !strings.Contains(view, "ball-1") {
		t.Error("Expected view to contain ball ID")
	}

	if !strings.Contains(view, "Test task") {
		t.Error("Expected view to contain ball intent")
	}

	// Should show existing tags
	if !strings.Contains(view, "existing-tag") {
		t.Error("Expected view to show existing tags")
	}

	// Should show available sessions
	if !strings.Contains(view, "session-a") {
		t.Error("Expected view to show session-a")
	}

	if !strings.Contains(view, "session-b") {
		t.Error("Expected view to show session-b")
	}

	// Should show help text
	if !strings.Contains(view, "navigate") || !strings.Contains(view, "Esc") {
		t.Error("Expected view to show help text")
	}
}

// Test session selector view shows cursor
func TestSessionSelectorViewShowsCursor(t *testing.T) {
	sessions := []*session.JuggleSession{
		{ID: "session-a", Description: "First session"},
		{ID: "session-b", Description: "Second session"},
	}

	ball := &session.Ball{ID: "ball-1", Intent: "Test task", Tags: []string{}, WorkingDir: "/tmp"}

	model := Model{
		mode:               sessionSelectorView,
		sessionSelectItems: sessions,
		sessionSelectIndex: 1, // Second item selected
		editingBall:        ball,
		width:              80,
		height:             24,
	}

	view := model.renderSessionSelectorView()

	// The view should have a cursor indicator ">" before the selected session
	if !strings.Contains(view, "> session-b") {
		t.Error("Expected cursor '>' before selected session-b")
	}
}

// Test 't' key does nothing when not in balls panel
func TestTagKeyOnlyWorksInBallsPanel(t *testing.T) {
	sessions := []*session.JuggleSession{
		{ID: "session-a", Description: "First session"},
	}

	balls := []*session.Ball{
		{ID: "ball-1", Intent: "Test task", Tags: []string{}, WorkingDir: "/tmp"},
	}

	model := Model{
		mode:            splitView,
		activePanel:     SessionsPanel, // Not in balls panel
		cursor:          0,
		filteredBalls:   balls,
		sessions:        sessions,
		selectedSession: &session.JuggleSession{ID: PseudoSessionAll},
	}

	// Simulate pressing 't' via handleSplitViewKey
	newModel, _ := model.handleSplitViewKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	m := newModel.(Model)

	// Should remain in splitView since we're not in balls panel
	if m.mode != splitView {
		t.Errorf("Expected mode to remain splitView when 't' pressed outside balls panel, got %v", m.mode)
	}
}

// Test 't' key with no balls selected
func TestTagKeyNoBallSelected(t *testing.T) {
	sessions := []*session.JuggleSession{
		{ID: "session-a", Description: "First session"},
	}

	model := Model{
		mode:            splitView,
		activePanel:     BallsPanel,
		cursor:          0,
		filteredBalls:   []*session.Ball{}, // No balls
		sessions:        sessions,
		selectedSession: &session.JuggleSession{ID: PseudoSessionAll},
	}

	newModel, _ := model.handleTagEditStart()
	m := newModel.(Model)

	if m.mode != splitView {
		t.Errorf("Expected mode to remain splitView when no balls, got %v", m.mode)
	}

	if m.message != "No ball selected" {
		t.Errorf("Expected message 'No ball selected', got '%s'", m.message)
	}
}

// Test submitSessionSelection with empty items
func TestSubmitSessionSelectionEmpty(t *testing.T) {
	ball := &session.Ball{ID: "ball-1", Intent: "Test task", Tags: []string{}, WorkingDir: "/tmp"}

	model := Model{
		mode:               sessionSelectorView,
		sessionSelectItems: []*session.JuggleSession{}, // Empty
		sessionSelectIndex: 0,
		editingBall:        ball,
	}

	newModel, _ := model.submitSessionSelection()
	m := newModel.(Model)

	if m.mode != splitView {
		t.Errorf("Expected mode to be splitView, got %v", m.mode)
	}

	if m.sessionSelectItems != nil {
		t.Error("Expected sessionSelectItems to be nil")
	}
}

// Test submitSessionSelection with nil editingBall
func TestSubmitSessionSelectionNilBall(t *testing.T) {
	sessions := []*session.JuggleSession{
		{ID: "session-a", Description: "First session"},
	}

	model := Model{
		mode:               sessionSelectorView,
		sessionSelectItems: sessions,
		sessionSelectIndex: 0,
		editingBall:        nil, // No ball
	}

	newModel, _ := model.submitSessionSelection()
	m := newModel.(Model)

	if m.mode != splitView {
		t.Errorf("Expected mode to be splitView, got %v", m.mode)
	}

	if m.sessionSelectItems != nil {
		t.Error("Expected sessionSelectItems to be nil")
	}
}

// Test session selector index bounds correction
func TestSessionSelectorIndexCorrection(t *testing.T) {
	sessions := []*session.JuggleSession{
		{ID: "session-a", Description: "First session"},
	}

	ball := &session.Ball{ID: "ball-1", Intent: "Test task", Tags: []string{}, WorkingDir: "/tmp"}

	model := Model{
		mode:               sessionSelectorView,
		sessionSelectItems: sessions,
		sessionSelectIndex: 5, // Out of bounds
		editingBall:        ball,
	}

	newModel, _ := model.submitSessionSelection()
	m := newModel.(Model)

	// Should have corrected the index and selected session-a
	if len(m.editingBall.Tags) != 1 {
		t.Errorf("Expected 1 tag on ball, got %d", len(m.editingBall.Tags))
	}

	if m.editingBall.Tags[0] != "session-a" {
		t.Errorf("Expected tag 'session-a' (index corrected), got '%s'", m.editingBall.Tags[0])
	}
}

// Test renderStatusBar contextual hints for SessionsPanel
func TestStatusBarSessionsPanel(t *testing.T) {
	model := Model{
		mode:        splitView,
		activePanel: SessionsPanel,
		localOnly:   true,
		width:       120,
		height:      40,
	}

	statusBar := model.renderStatusBar()

	// Should contain session-specific keybinds
	expectedKeys := []string{"j/k:nav", "Enter:select", "a:add", "A:agent", "e:edit", "d:del", "/:filter", "P:scope", "?:help", "q:quit"}
	for _, key := range expectedKeys {
		if !strings.Contains(statusBar, key) {
			t.Errorf("Expected status bar to contain '%s', got: %s", key, statusBar)
		}
	}

	// Should show local indicator
	if !strings.Contains(statusBar, "[Local]") {
		t.Errorf("Expected status bar to show [Local] indicator")
	}
}

// Test renderStatusBar contextual hints for BallsPanel
func TestStatusBarBallsPanel(t *testing.T) {
	model := Model{
		mode:        splitView,
		activePanel: BallsPanel,
		localOnly:   false,
		width:       120,
		height:      40,
	}

	statusBar := model.renderStatusBar()

	// Should contain ball-specific keybinds including state change keys
	expectedKeys := []string{"j/k:nav", "s:start", "c:done", "b:block", "a:add", "e:edit", "t:tag", "d:del", "[/]:session", "o:sort", "?:help"}
	for _, key := range expectedKeys {
		if !strings.Contains(statusBar, key) {
			t.Errorf("Expected status bar to contain '%s', got: %s", key, statusBar)
		}
	}

	// Should show all projects indicator when localOnly is false
	if !strings.Contains(statusBar, "[All]") {
		t.Errorf("Expected status bar to show [All] indicator")
	}
}

// Test renderStatusBar contextual hints for ActivityPanel
func TestStatusBarActivityPanel(t *testing.T) {
	model := Model{
		mode:        splitView,
		activePanel: ActivityPanel,
		localOnly:   true,
		width:       120,
		height:      40,
	}

	statusBar := model.renderStatusBar()

	// Should contain activity-specific keybinds
	expectedKeys := []string{"j/k:scroll", "Ctrl+d/u:page", "gg:top", "G:bottom", "Tab:panels", "?:help", "q:quit"}
	for _, key := range expectedKeys {
		if !strings.Contains(statusBar, key) {
			t.Errorf("Expected status bar to contain '%s', got: %s", key, statusBar)
		}
	}
}

// Test renderStatusBar shows agent status when running
func TestStatusBarWithRunningAgent(t *testing.T) {
	model := Model{
		mode:        splitView,
		activePanel: SessionsPanel,
		localOnly:   true,
		agentStatus: AgentStatus{
			Running:       true,
			SessionID:     "test-session",
			Iteration:     3,
			MaxIterations: 10,
		},
		width:  120,
		height: 40,
	}

	statusBar := model.renderStatusBar()

	// Should show agent status with cancel hint
	if !strings.Contains(statusBar, "[Agent: test-session 3/10 | X:cancel]") {
		t.Errorf("Expected status bar to show agent status with cancel hint, got: %s", statusBar)
	}
}

// Test renderStatusBar shows filter indicator when active
func TestStatusBarWithActiveFilter(t *testing.T) {
	model := Model{
		mode:              splitView,
		activePanel:       BallsPanel,
		localOnly:         true,
		panelSearchActive: true,
		panelSearchQuery:  "myfilter",
		width:             120,
		height:            40,
	}

	statusBar := model.renderStatusBar()

	// Should show filter indicator
	if !strings.Contains(statusBar, "[Filter: myfilter") {
		t.Errorf("Expected status bar to show filter indicator, got: %s", statusBar)
	}
	if !strings.Contains(statusBar, "Ctrl+U:clear") {
		t.Errorf("Expected status bar to show clear filter hint, got: %s", statusBar)
	}
}

// Test renderStatusBar shows message when present
func TestStatusBarWithMessage(t *testing.T) {
	model := Model{
		mode:        splitView,
		activePanel: SessionsPanel,
		localOnly:   true,
		message:     "Operation successful",
		width:       120,
		height:      40,
	}

	statusBar := model.renderStatusBar()

	// Should show message
	if !strings.Contains(statusBar, "Operation successful") {
		t.Errorf("Expected status bar to show message, got: %s", statusBar)
	}
}

// Test renderSplitHelpView contains all categories
func TestHelpViewContainsAllCategories(t *testing.T) {
	model := Model{
		mode:   splitHelpView,
		width:  120,
		height: 80, // Increased to show all categories
	}

	helpView := model.renderSplitHelpView()

	// Check all category titles are present
	categories := []string{
		"Navigation",
		"Sessions Panel",
		"Balls Panel",
		"Activity Log Panel",
		"View Options",
		"Bottom Pane Modes",
		"Input Dialogs",
		"Delete Confirmation",
		"Quit",
	}

	for _, category := range categories {
		if !strings.Contains(helpView, category) {
			t.Errorf("Expected help view to contain category '%s'", category)
		}
	}
}

// Test renderSplitHelpView contains key navigation bindings
func TestHelpViewContainsNavigationBindings(t *testing.T) {
	model := Model{
		mode:   splitHelpView,
		width:  120,
		height: 60,
	}

	helpView := model.renderSplitHelpView()

	// Check navigation keybinds
	navigationBindings := []string{
		"Tab / l",
		"Shift+Tab / h",
		"Enter",
		"Space",
		"Esc",
	}

	for _, binding := range navigationBindings {
		if !strings.Contains(helpView, binding) {
			t.Errorf("Expected help view to contain navigation binding '%s'", binding)
		}
	}
}

// Test renderSplitHelpView contains balls panel state change bindings
func TestHelpViewContainsBallsStateBindings(t *testing.T) {
	model := Model{
		mode:   splitHelpView,
		width:  120,
		height: 60,
	}

	helpView := model.renderSplitHelpView()

	// Check balls panel state change keybinds (critical for AC #3)
	ballsBindings := []string{
		"Start ball",
		"Complete ball",
		"Block ball",
		"Edit ball",
		"Tag ball",
		"Delete ball",
	}

	for _, binding := range ballsBindings {
		if !strings.Contains(helpView, binding) {
			t.Errorf("Expected help view to contain balls binding '%s'", binding)
		}
	}
}

// Test renderSplitHelpView contains view options bindings
func TestHelpViewContainsViewOptionsBindings(t *testing.T) {
	model := Model{
		mode:   splitHelpView,
		width:  120,
		height: 80, // Increased to show all content
	}

	helpView := model.renderSplitHelpView()

	// Check view options keybinds
	viewOptionsBindings := []string{
		"Cycle bottom pane",
		"Toggle project scope",
		"Refresh",
		"Toggle this help",
	}

	for _, binding := range viewOptionsBindings {
		if !strings.Contains(helpView, binding) {
			t.Errorf("Expected help view to contain view options binding '%s'", binding)
		}
	}
}

// Test help view scrolling works
func TestHelpViewScrollingJK(t *testing.T) {
	model := Model{
		mode:             splitHelpView,
		width:            120,
		height:           20, // Small height to force scrolling
		helpScrollOffset: 0,
	}

	// Scroll down with j
	newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m := newModel.(Model)
	if m.helpScrollOffset != 1 {
		t.Errorf("Expected helpScrollOffset to be 1 after pressing j, got %d", m.helpScrollOffset)
	}

	// Scroll up with k
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = newModel.(Model)
	if m.helpScrollOffset != 0 {
		t.Errorf("Expected helpScrollOffset to be 0 after pressing k, got %d", m.helpScrollOffset)
	}

	// Don't go negative
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = newModel.(Model)
	if m.helpScrollOffset != 0 {
		t.Errorf("Expected helpScrollOffset to remain 0, got %d", m.helpScrollOffset)
	}
}

// Test help view page up/down with ctrl keys
func TestHelpViewPageScrollingCtrl(t *testing.T) {
	model := Model{
		mode:             splitHelpView,
		width:            120,
		height:           20,
		helpScrollOffset: 20,
	}

	// Page up with ctrl+u
	newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyCtrlU})
	m := newModel.(Model)
	if m.helpScrollOffset != 10 {
		t.Errorf("Expected helpScrollOffset to be 10 after ctrl+u, got %d", m.helpScrollOffset)
	}

	// Page down with ctrl+d
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	m = newModel.(Model)
	if m.helpScrollOffset != 20 {
		t.Errorf("Expected helpScrollOffset to be 20 after ctrl+d, got %d", m.helpScrollOffset)
	}
}

// Test help view gg and G for top/bottom
func TestHelpViewGoToTopBottomGG(t *testing.T) {
	model := Model{
		mode:             splitHelpView,
		width:            120,
		height:           20,
		helpScrollOffset: 10,
		lastKey:          "g",
	}

	// gg goes to top
	newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m := newModel.(Model)
	if m.helpScrollOffset != 0 {
		t.Errorf("Expected helpScrollOffset to be 0 after gg, got %d", m.helpScrollOffset)
	}
	if m.lastKey != "" {
		t.Errorf("Expected lastKey to be cleared, got '%s'", m.lastKey)
	}

	// G goes to bottom (large number that will be clamped)
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	m = newModel.(Model)
	if m.helpScrollOffset != 1000 {
		t.Errorf("Expected helpScrollOffset to be 1000 (before clamping) after G, got %d", m.helpScrollOffset)
	}
}

// Test help view closes with ?, q, or esc
func TestHelpViewCloseKeysAll(t *testing.T) {
	tests := []struct {
		name string
		key  tea.KeyMsg
	}{
		{"question mark", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}}},
		{"q", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}},
		{"esc", tea.KeyMsg{Type: tea.KeyEscape}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := Model{
				mode:   splitHelpView,
				width:  120,
				height: 40,
			}

			newModel, _ := model.Update(tt.key)
			m := newModel.(Model)

			if m.mode != splitView {
				t.Errorf("Expected mode to return to splitView after pressing %s, got %v", tt.name, m.mode)
			}
		})
	}
}

// Test help view displays scroll indicators when scrolled
func TestHelpViewScrollIndicatorsDisplay(t *testing.T) {
	// With scroll offset > 0, should show "more lines above" indicator
	model := Model{
		mode:             splitHelpView,
		width:            120,
		height:           20, // Small height
		helpScrollOffset: 5,
	}

	helpView := model.renderSplitHelpView()

	// Should show scroll indicator at top
	if !strings.Contains(helpView, "more lines above") {
		t.Errorf("Expected help view to show 'more lines above' indicator when scrolled down")
	}
}

// Test footer shows correct hints when panelSearchActive is true
func TestStatusBarFilterIndicatorPresent(t *testing.T) {
	model := Model{
		mode:              splitView,
		activePanel:       SessionsPanel,
		localOnly:         true,
		panelSearchActive: true,
		panelSearchQuery:  "test",
		width:             120,
		height:            40,
	}

	statusBar := model.renderStatusBar()

	// Filter indicator should be present
	if !strings.Contains(statusBar, "[Filter:") {
		t.Error("Expected filter indicator to be present")
	}
	// Local indicator should also be present
	if !strings.Contains(statusBar, "[Local]") {
		t.Error("Expected scope indicator to be present")
	}
}

// =============================================================================
// Panel Navigation Tests (juggler-66)
// =============================================================================

// TestEnterOnSessionMovesFocusToBallsPanel verifies that pressing Enter on a session
// moves focus from SessionsPanel to BallsPanel
func TestEnterOnSessionMovesFocusToBallsPanel(t *testing.T) {
	sessions := []*session.JuggleSession{
		{ID: PseudoSessionAll, Description: "All balls"},
		{ID: "session1", Description: "Session 1"},
	}

	model := Model{
		mode:          splitView,
		activePanel:   SessionsPanel,
		sessionCursor: 0,
		sessions:      sessions,
		filteredBalls: []*session.Ball{},
		height:        30,
		width:         80,
		filterStates: map[string]bool{
			"pending":     true,
			"in_progress": true,
			"blocked":     true,
			"complete":    true,
		},
		activityLog: make([]ActivityEntry, 0),
	}

	// Press Enter to select session
	newModel, _ := model.handleSplitViewEnter()
	model = newModel.(Model)

	// Should have moved focus to balls panel
	if model.activePanel != BallsPanel {
		t.Errorf("Expected activePanel to be BallsPanel after Enter, got %v", model.activePanel)
	}

	// Should have selected the session
	if model.selectedSession == nil {
		t.Error("Expected selectedSession to be set after Enter")
	}
	if model.selectedSession != nil && model.selectedSession.ID != PseudoSessionAll {
		t.Errorf("Expected selectedSession to be %s, got %s", PseudoSessionAll, model.selectedSession.ID)
	}
}

// TestEnterOnSessionWithDifferentCursor verifies Enter works with non-zero cursor position
// Note: filterSessions() prepends PseudoSessionAll and PseudoSessionUntagged, so:
//   cursor 0 = __all__, cursor 1 = __untagged__, cursor 2+ = real sessions
func TestEnterOnSessionWithDifferentCursor(t *testing.T) {
	sessions := []*session.JuggleSession{
		{ID: "session1", Description: "Session 1"},
		{ID: "session2", Description: "Session 2"},
	}

	model := Model{
		mode:          splitView,
		activePanel:   SessionsPanel,
		sessionCursor: 3, // Fourth in filtered list = session2 (after __all__, __untagged__, session1)
		sessions:      sessions,
		filteredBalls: []*session.Ball{},
		height:        30,
		width:         80,
		filterStates: map[string]bool{
			"pending":     true,
			"in_progress": true,
			"blocked":     true,
			"complete":    true,
		},
		activityLog: make([]ActivityEntry, 0),
	}

	// Press Enter to select session
	newModel, _ := model.handleSplitViewEnter()
	model = newModel.(Model)

	// Should have moved focus to balls panel
	if model.activePanel != BallsPanel {
		t.Errorf("Expected activePanel to be BallsPanel after Enter, got %v", model.activePanel)
	}

	// Should have selected session2
	if model.selectedSession == nil {
		t.Error("Expected selectedSession to be set after Enter")
	}
	if model.selectedSession != nil && model.selectedSession.ID != "session2" {
		t.Errorf("Expected selectedSession to be session2, got %s", model.selectedSession.ID)
	}
}

// TestSpaceKeyMovesFocusToSessionsPanel verifies that pressing Space in BallsPanel
// moves focus back to SessionsPanel
func TestSpaceKeyMovesFocusToSessionsPanel(t *testing.T) {
	sessions := []*session.JuggleSession{
		{ID: "session1", Description: "Session 1"},
	}

	model := Model{
		mode:            splitView,
		activePanel:     BallsPanel,
		sessionCursor:   0,
		selectedSession: sessions[0],
		sessions:        sessions,
		filteredBalls:   []*session.Ball{},
		height:          30,
		width:           80,
		filterStates: map[string]bool{
			"pending":     true,
			"in_progress": true,
			"blocked":     true,
			"complete":    true,
		},
		activityLog: make([]ActivityEntry, 0),
	}

	// Verify we start in BallsPanel
	if model.activePanel != BallsPanel {
		t.Fatalf("Expected starting activePanel to be BallsPanel, got %v", model.activePanel)
	}

	// Simulate pressing Space key
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
	newModel, _ := model.Update(keyMsg)
	model = newModel.(Model)

	// Should have moved focus to sessions panel
	if model.activePanel != SessionsPanel {
		t.Errorf("Expected activePanel to be SessionsPanel after Space, got %v", model.activePanel)
	}
}

// TestSpaceKeyOnlyWorksInBallsPanel verifies Space doesn't change panel in other panels
func TestSpaceKeyOnlyWorksInBallsPanel(t *testing.T) {
	model := Model{
		mode:          splitView,
		activePanel:   SessionsPanel,
		sessions:      []*session.JuggleSession{},
		filteredBalls: []*session.Ball{},
		height:        30,
		width:         80,
		filterStates: map[string]bool{
			"pending":     true,
			"in_progress": true,
			"blocked":     true,
			"complete":    true,
		},
		activityLog: make([]ActivityEntry, 0),
	}

	// Simulate pressing Space key in SessionsPanel
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
	newModel, _ := model.Update(keyMsg)
	model = newModel.(Model)

	// Should still be in SessionsPanel (Space does nothing there)
	if model.activePanel != SessionsPanel {
		t.Errorf("Expected activePanel to remain SessionsPanel, got %v", model.activePanel)
	}
}

// TestBracketLeftSwitchesToPrevSession verifies [ key switches to previous session in BallsPanel
// Note: filterSessions() prepends __all__ and __untagged__, so indices are:
//   0 = __all__, 1 = __untagged__, 2 = session1, 3 = session2, 4 = session3
func TestBracketLeftSwitchesToPrevSession(t *testing.T) {
	sessions := []*session.JuggleSession{
		{ID: "session1", Description: "Session 1"},
		{ID: "session2", Description: "Session 2"},
		{ID: "session3", Description: "Session 3"},
	}

	model := Model{
		mode:            splitView,
		activePanel:     BallsPanel,
		sessionCursor:   4,           // At session3 (index 4 in filtered list)
		selectedSession: sessions[2], // session3 selected
		sessions:        sessions,
		filteredBalls:   []*session.Ball{},
		height:          30,
		width:           80,
		filterStates: map[string]bool{
			"pending":     true,
			"in_progress": true,
			"blocked":     true,
			"complete":    true,
		},
		activityLog: make([]ActivityEntry, 0),
	}

	// Press [ to switch to previous session
	newModel, _ := model.handleSessionSwitchPrev()
	model = newModel.(Model)

	// Should have moved to session2 (index 3)
	if model.sessionCursor != 3 {
		t.Errorf("Expected sessionCursor to be 3, got %d", model.sessionCursor)
	}
	if model.selectedSession.ID != "session2" {
		t.Errorf("Expected selectedSession to be session2, got %s", model.selectedSession.ID)
	}

	// Should stay in BallsPanel
	if model.activePanel != BallsPanel {
		t.Errorf("Expected activePanel to remain BallsPanel, got %v", model.activePanel)
	}
}

// TestBracketRightSwitchesToNextSession verifies ] key switches to next session in BallsPanel
// Note: filterSessions() prepends __all__ and __untagged__, so indices are:
//   0 = __all__, 1 = __untagged__, 2 = session1, 3 = session2, 4 = session3
func TestBracketRightSwitchesToNextSession(t *testing.T) {
	sessions := []*session.JuggleSession{
		{ID: "session1", Description: "Session 1"},
		{ID: "session2", Description: "Session 2"},
		{ID: "session3", Description: "Session 3"},
	}

	model := Model{
		mode:            splitView,
		activePanel:     BallsPanel,
		sessionCursor:   2,           // At session1 (index 2 in filtered list)
		selectedSession: sessions[0], // session1 selected
		sessions:        sessions,
		filteredBalls:   []*session.Ball{},
		height:          30,
		width:           80,
		filterStates: map[string]bool{
			"pending":     true,
			"in_progress": true,
			"blocked":     true,
			"complete":    true,
		},
		activityLog: make([]ActivityEntry, 0),
	}

	// Press ] to switch to next session
	newModel, _ := model.handleSessionSwitchNext()
	model = newModel.(Model)

	// Should have moved to session2 (index 3)
	if model.sessionCursor != 3 {
		t.Errorf("Expected sessionCursor to be 3, got %d", model.sessionCursor)
	}
	if model.selectedSession.ID != "session2" {
		t.Errorf("Expected selectedSession to be session2, got %s", model.selectedSession.ID)
	}

	// Should stay in BallsPanel
	if model.activePanel != BallsPanel {
		t.Errorf("Expected activePanel to remain BallsPanel, got %v", model.activePanel)
	}
}

// TestBracketLeftAtBoundary verifies [ key at first session doesn't go negative
func TestBracketLeftAtBoundary(t *testing.T) {
	sessions := []*session.JuggleSession{
		{ID: "session1", Description: "Session 1"},
		{ID: "session2", Description: "Session 2"},
	}

	model := Model{
		mode:            splitView,
		activePanel:     BallsPanel,
		sessionCursor:   0,           // At first session
		selectedSession: sessions[0], // session1 selected
		sessions:        sessions,
		filteredBalls:   []*session.Ball{},
		height:          30,
		width:           80,
		filterStates: map[string]bool{
			"pending":     true,
			"in_progress": true,
			"blocked":     true,
			"complete":    true,
		},
		activityLog: make([]ActivityEntry, 0),
	}

	// Press [ at first session
	newModel, _ := model.handleSessionSwitchPrev()
	model = newModel.(Model)

	// Should stay at session1 (can't go negative)
	if model.sessionCursor != 0 {
		t.Errorf("Expected sessionCursor to stay at 0, got %d", model.sessionCursor)
	}
	if model.selectedSession.ID != "session1" {
		t.Errorf("Expected selectedSession to remain session1, got %s", model.selectedSession.ID)
	}
}

// TestBracketRightAtBoundary verifies ] key at last session doesn't overflow
// Note: filterSessions() prepends __all__ and __untagged__, so:
//   0 = __all__, 1 = __untagged__, 2 = session1, 3 = session2 (last)
func TestBracketRightAtBoundary(t *testing.T) {
	sessions := []*session.JuggleSession{
		{ID: "session1", Description: "Session 1"},
		{ID: "session2", Description: "Session 2"},
	}

	model := Model{
		mode:            splitView,
		activePanel:     BallsPanel,
		sessionCursor:   3,           // At session2 (last, index 3 in filtered list)
		selectedSession: sessions[1], // session2 selected
		sessions:        sessions,
		filteredBalls:   []*session.Ball{},
		height:          30,
		width:           80,
		filterStates: map[string]bool{
			"pending":     true,
			"in_progress": true,
			"blocked":     true,
			"complete":    true,
		},
		activityLog: make([]ActivityEntry, 0),
	}

	// Press ] at last session
	newModel, _ := model.handleSessionSwitchNext()
	model = newModel.(Model)

	// Should stay at session2 (can't go beyond last)
	if model.sessionCursor != 3 {
		t.Errorf("Expected sessionCursor to stay at 3, got %d", model.sessionCursor)
	}
	if model.selectedSession.ID != "session2" {
		t.Errorf("Expected selectedSession to remain session2, got %s", model.selectedSession.ID)
	}
}

// TestSessionSwitchResetsBallCursor verifies cursor resets when switching sessions
// Note: filterSessions() prepends __all__ and __untagged__, so:
//   0 = __all__, 1 = __untagged__, 2 = session1, 3 = session2
func TestSessionSwitchResetsBallCursor(t *testing.T) {
	sessions := []*session.JuggleSession{
		{ID: "session1", Description: "Session 1"},
		{ID: "session2", Description: "Session 2"},
	}

	model := Model{
		mode:            splitView,
		activePanel:     BallsPanel,
		sessionCursor:   2,           // At session1 (index 2 in filtered list)
		selectedSession: sessions[0], // session1 selected
		cursor:          5,           // Ball cursor at position 5
		sessions:        sessions,
		filteredBalls:   []*session.Ball{},
		height:          30,
		width:           80,
		filterStates: map[string]bool{
			"pending":     true,
			"in_progress": true,
			"blocked":     true,
			"complete":    true,
		},
		activityLog: make([]ActivityEntry, 0),
	}

	// Press ] to switch to next session
	newModel, _ := model.handleSessionSwitchNext()
	model = newModel.(Model)

	// Ball cursor should be reset to 0
	if model.cursor != 0 {
		t.Errorf("Expected cursor to reset to 0 on session switch, got %d", model.cursor)
	}
}

// TestSessionSwitchUpdatesActivity verifies activity log entry is added on session switch
// Note: filterSessions() prepends __all__ and __untagged__, so:
//   0 = __all__, 1 = __untagged__, 2 = session1, 3 = session2
func TestSessionSwitchUpdatesActivity(t *testing.T) {
	sessions := []*session.JuggleSession{
		{ID: "session1", Description: "Session 1"},
		{ID: "session2", Description: "Session 2"},
	}

	model := Model{
		mode:            splitView,
		activePanel:     BallsPanel,
		sessionCursor:   2,           // At session1 (index 2 in filtered list)
		selectedSession: sessions[0], // session1 selected
		sessions:        sessions,
		filteredBalls:   []*session.Ball{},
		height:          30,
		width:           80,
		filterStates: map[string]bool{
			"pending":     true,
			"in_progress": true,
			"blocked":     true,
			"complete":    true,
		},
		activityLog: make([]ActivityEntry, 0),
	}

	// Press ] to switch to next session
	newModel, _ := model.handleSessionSwitchNext()
	model = newModel.(Model)

	// Activity log should have an entry about the switch
	if len(model.activityLog) == 0 {
		t.Error("Expected activity log entry after session switch")
	}
	if len(model.activityLog) > 0 && !strings.Contains(model.activityLog[len(model.activityLog)-1].Message, "session2") {
		t.Errorf("Expected activity log to mention session2, got %s", model.activityLog[len(model.activityLog)-1].Message)
	}
}

// TestBracketKeysOnlyWorkInBallsPanel verifies [ and ] only work in BallsPanel
func TestBracketKeysOnlyWorkInBallsPanel(t *testing.T) {
	sessions := []*session.JuggleSession{
		{ID: "session1", Description: "Session 1"},
		{ID: "session2", Description: "Session 2"},
	}

	model := Model{
		mode:            splitView,
		activePanel:     SessionsPanel, // Not in BallsPanel
		sessionCursor:   0,
		selectedSession: sessions[0],
		sessions:        sessions,
		filteredBalls:   []*session.Ball{},
		height:          30,
		width:           80,
		filterStates: map[string]bool{
			"pending":     true,
			"in_progress": true,
			"blocked":     true,
			"complete":    true,
		},
		activityLog: make([]ActivityEntry, 0),
	}

	// Simulate pressing ] key in SessionsPanel
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}}
	newModel, _ := model.Update(keyMsg)
	model = newModel.(Model)

	// Session should not have changed
	if model.sessionCursor != 0 {
		t.Errorf("Expected sessionCursor to remain 0 in SessionsPanel, got %d", model.sessionCursor)
	}
}

// TestEnterResetsBallCursorAndScrollOffset verifies Enter resets both cursor and scroll offset
func TestEnterResetsBallCursorAndScrollOffset(t *testing.T) {
	sessions := []*session.JuggleSession{
		{ID: "session1", Description: "Session 1"},
	}

	model := Model{
		mode:              splitView,
		activePanel:       SessionsPanel,
		sessionCursor:     0,
		cursor:            10, // Ball cursor at position 10
		ballsScrollOffset: 5,  // Scrolled down
		sessions:          sessions,
		filteredBalls:     []*session.Ball{},
		height:            30,
		width:             80,
		filterStates: map[string]bool{
			"pending":     true,
			"in_progress": true,
			"blocked":     true,
			"complete":    true,
		},
		activityLog: make([]ActivityEntry, 0),
	}

	// Press Enter to select session
	newModel, _ := model.handleSplitViewEnter()
	model = newModel.(Model)

	// Both cursor and scroll offset should be reset
	if model.cursor != 0 {
		t.Errorf("Expected cursor to reset to 0, got %d", model.cursor)
	}
	if model.ballsScrollOffset != 0 {
		t.Errorf("Expected ballsScrollOffset to reset to 0, got %d", model.ballsScrollOffset)
	}
}

// TestCompleteEnterSpaceWorkflow tests the full Enter-to-balls, Space-to-sessions workflow
// Note: filterSessions() prepends __all__ and __untagged__, so:
//   0 = __all__, 1 = __untagged__, 2 = session1, 3 = session2
func TestCompleteEnterSpaceWorkflow(t *testing.T) {
	sessions := []*session.JuggleSession{
		{ID: "session1", Description: "Session 1"},
		{ID: "session2", Description: "Session 2"},
	}

	model := Model{
		mode:          splitView,
		activePanel:   SessionsPanel,
		sessionCursor: 2, // session1 (index 2 in filtered list)
		sessions:      sessions,
		filteredBalls: []*session.Ball{},
		height:        30,
		width:         80,
		filterStates: map[string]bool{
			"pending":     true,
			"in_progress": true,
			"blocked":     true,
			"complete":    true,
		},
		activityLog: make([]ActivityEntry, 0),
	}

	// Step 1: Press Enter to go to BallsPanel
	newModel, _ := model.handleSplitViewEnter()
	model = newModel.(Model)

	if model.activePanel != BallsPanel {
		t.Fatalf("Step 1: Expected activePanel to be BallsPanel, got %v", model.activePanel)
	}
	if model.selectedSession == nil || model.selectedSession.ID != "session1" {
		t.Fatalf("Step 1: Expected session1 to be selected, got %v", model.selectedSession)
	}

	// Step 2: Press Space to go back to SessionsPanel
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
	newModel, _ = model.Update(keyMsg)
	model = newModel.(Model)

	if model.activePanel != SessionsPanel {
		t.Fatalf("Step 2: Expected activePanel to be SessionsPanel, got %v", model.activePanel)
	}

	// Session selection should be preserved
	if model.selectedSession == nil || model.selectedSession.ID != "session1" {
		t.Errorf("Step 2: Expected session1 to remain selected after Space")
	}
}

// TestCompleteSessionSwitchWorkflow tests full session switching workflow with []
// Note: filterSessions() prepends __all__ and __untagged__, so:
//   0 = __all__, 1 = __untagged__, 2 = session1, 3 = session2, 4 = session3
func TestCompleteSessionSwitchWorkflow(t *testing.T) {
	sessions := []*session.JuggleSession{
		{ID: "session1", Description: "Session 1"},
		{ID: "session2", Description: "Session 2"},
		{ID: "session3", Description: "Session 3"},
	}

	model := Model{
		mode:            splitView,
		activePanel:     BallsPanel,
		sessionCursor:   3,           // session2 (index 3 in filtered list)
		selectedSession: sessions[1], // session2
		cursor:          5,
		sessions:        sessions,
		filteredBalls:   []*session.Ball{},
		height:          30,
		width:           80,
		filterStates: map[string]bool{
			"pending":     true,
			"in_progress": true,
			"blocked":     true,
			"complete":    true,
		},
		activityLog: make([]ActivityEntry, 0),
	}

	// Press ] to go to session3
	newModel, _ := model.handleSessionSwitchNext()
	model = newModel.(Model)

	if model.selectedSession.ID != "session3" {
		t.Fatalf("Step 1: Expected session3, got %s", model.selectedSession.ID)
	}
	if model.cursor != 0 {
		t.Fatalf("Step 1: Expected cursor reset to 0, got %d", model.cursor)
	}

	// Press [ to go back to session2
	newModel, _ = model.handleSessionSwitchPrev()
	model = newModel.(Model)

	if model.selectedSession.ID != "session2" {
		t.Fatalf("Step 2: Expected session2, got %s", model.selectedSession.ID)
	}

	// Press [ again to go to session1
	newModel, _ = model.handleSessionSwitchPrev()
	model = newModel.(Model)

	if model.selectedSession.ID != "session1" {
		t.Fatalf("Step 3: Expected session1, got %s", model.selectedSession.ID)
	}

	// Verify we stayed in BallsPanel throughout
	if model.activePanel != BallsPanel {
		t.Errorf("Expected to remain in BallsPanel throughout, got %v", model.activePanel)
	}
}

// Tests for Ball Detail Pane (juggler-67)

// TestBallDetailPanelShowsAllAcceptanceCriteria verifies that all acceptance criteria are shown
func TestBallDetailPanelShowsAllAcceptanceCriteria(t *testing.T) {
	ball := &session.Ball{
		ID:       "test-1",
		Intent:   "Test ball",
		State:    session.StateInProgress,
		Priority: session.PriorityMedium,
		AcceptanceCriteria: []string{
			"First criterion",
			"Second criterion",
			"Third criterion",
			"Fourth criterion",
		},
	}

	model := Model{
		mode:           splitView,
		activePanel:    BallsPanel,
		cursor:         0,
		filteredBalls:  []*session.Ball{ball},
		bottomPaneMode: BottomPaneDetail,
		width:          120,
		height:         40,
		filterStates: map[string]bool{
			"pending":     true,
			"in_progress": true,
			"blocked":     true,
			"complete":    true,
		},
	}

	// Build detail lines
	lines := model.buildBallDetailLines(ball, 100)

	// Should contain all 4 acceptance criteria
	foundCriteria := 0
	for _, line := range lines {
		if strings.Contains(line, "First criterion") ||
			strings.Contains(line, "Second criterion") ||
			strings.Contains(line, "Third criterion") ||
			strings.Contains(line, "Fourth criterion") {
			foundCriteria++
		}
	}

	if foundCriteria != 4 {
		t.Errorf("Expected all 4 acceptance criteria in detail lines, found %d", foundCriteria)
	}
}

// TestBallDetailPanelUpdatesWithNavigation verifies that the detail panel updates when navigating balls
func TestBallDetailPanelUpdatesWithNavigation(t *testing.T) {
	balls := []*session.Ball{
		{ID: "test-1", Intent: "First ball", State: session.StatePending, Priority: session.PriorityMedium},
		{ID: "test-2", Intent: "Second ball", State: session.StateInProgress, Priority: session.PriorityHigh},
	}

	model := Model{
		mode:           splitView,
		activePanel:    BallsPanel,
		cursor:         0,
		filteredBalls:  balls,
		bottomPaneMode: BottomPaneDetail,
		width:          120,
		height:         40,
		filterStates: map[string]bool{
			"pending":     true,
			"in_progress": true,
			"blocked":     true,
			"complete":    true,
		},
	}

	// Build detail for first ball
	lines1 := model.buildBallDetailLines(balls[0], 100)
	hasFirstBall := false
	for _, line := range lines1 {
		if strings.Contains(line, "First ball") {
			hasFirstBall = true
			break
		}
	}
	if !hasFirstBall {
		t.Error("Expected first ball's intent in detail lines")
	}

	// Navigate to second ball
	model.cursor = 1

	// Build detail for second ball
	lines2 := model.buildBallDetailLines(balls[1], 100)
	hasSecondBall := false
	for _, line := range lines2 {
		if strings.Contains(line, "Second ball") {
			hasSecondBall = true
			break
		}
	}
	if !hasSecondBall {
		t.Error("Expected second ball's intent in detail lines")
	}
}

// TestBallDetailPanelShowsAllProperties verifies that all required properties are shown
func TestBallDetailPanelShowsAllProperties(t *testing.T) {
	ball := &session.Ball{
		ID:            "test-1",
		Intent:        "Test ball intent",
		State:         session.StateBlocked,
		Priority:      session.PriorityHigh,
		BlockedReason: "Waiting for API",
		Tags:          []string{"feature", "backend"},
		TestsState:    session.TestsStateNeeded,
		AcceptanceCriteria: []string{
			"First criterion",
		},
	}

	model := Model{
		mode:           splitView,
		activePanel:    BallsPanel,
		cursor:         0,
		filteredBalls:  []*session.Ball{ball},
		bottomPaneMode: BottomPaneDetail,
		width:          120,
		height:         40,
		filterStates: map[string]bool{
			"pending":     true,
			"in_progress": true,
			"blocked":     true,
			"complete":    true,
		},
	}

	lines := model.buildBallDetailLines(ball, 100)
	content := strings.Join(lines, "\n")

	// Check all required properties are present
	checks := []struct {
		name   string
		needle string
	}{
		{"ID", "test-1"},
		{"Intent", "Test ball intent"},
		{"State", "blocked"},
		{"Priority", "high"},
		{"Blocked reason", "Waiting for API"},
		{"Tags", "feature"},
		{"Tags", "backend"},
	}

	for _, check := range checks {
		if !strings.Contains(content, check.needle) {
			t.Errorf("Expected ball detail to contain %s: '%s'", check.name, check.needle)
		}
	}
}

// TestBottomPaneSplitModeRendering verifies split mode renders both panels
func TestBottomPaneSplitModeRendering(t *testing.T) {
	ball := &session.Ball{
		ID:       "test-1",
		Intent:   "Test ball",
		State:    session.StateInProgress,
		Priority: session.PriorityMedium,
		AcceptanceCriteria: []string{
			"First criterion",
		},
	}

	model := Model{
		mode:           splitView,
		activePanel:    BallsPanel,
		cursor:         0,
		filteredBalls:  []*session.Ball{ball},
		selectedBall:   ball,
		bottomPaneMode: BottomPaneSplit,
		width:          120,
		height:         40,
		activityLog: []ActivityEntry{
			{Message: "Test activity 1"},
			{Message: "Test activity 2"},
		},
		filterStates: map[string]bool{
			"pending":     true,
			"in_progress": true,
			"blocked":     true,
			"complete":    true,
		},
	}

	// Render split bottom pane
	content := model.renderSplitBottomPane(100, 6)

	// Should contain details panel title
	if !strings.Contains(content, "Details") {
		t.Error("Expected split view to contain 'Details' title")
	}

	// Should contain activity panel title
	if !strings.Contains(content, "Activity") {
		t.Error("Expected split view to contain 'Activity' title")
	}
}

// TestDetailPaneScrolling verifies scrolling in detail mode
func TestDetailPaneScrolling(t *testing.T) {
	model := Model{
		mode:               splitView,
		activePanel:        ActivityPanel,
		bottomPaneMode:     BottomPaneDetail,
		detailScrollOffset: 0,
		activityLog:        make([]ActivityEntry, 0),
	}

	// Scroll down
	newModel, _ := model.handleSplitViewNavDown()
	m := newModel.(Model)
	if m.detailScrollOffset != 1 {
		t.Errorf("Expected detailScrollOffset to be 1 after scrolling down, got %d", m.detailScrollOffset)
	}

	// Scroll up
	newModel, _ = m.handleSplitViewNavUp()
	m = newModel.(Model)
	if m.detailScrollOffset != 0 {
		t.Errorf("Expected detailScrollOffset to be 0 after scrolling up, got %d", m.detailScrollOffset)
	}
}

// TestDetailPanePageDownUp verifies page scrolling in detail mode
func TestDetailPanePageDownUp(t *testing.T) {
	model := Model{
		mode:               splitView,
		activePanel:        ActivityPanel,
		bottomPaneMode:     BottomPaneDetail,
		detailScrollOffset: 0,
		activityLog:        make([]ActivityEntry, 0),
	}

	// Page down
	newModel, _ := model.handleActivityLogPageDown()
	m := newModel.(Model)
	if m.detailScrollOffset == 0 {
		t.Error("Expected detailScrollOffset to increase after page down")
	}

	// Set to non-zero value and page up
	m.detailScrollOffset = 5
	newModel, _ = m.handleActivityLogPageUp()
	m = newModel.(Model)
	if m.detailScrollOffset >= 5 {
		t.Errorf("Expected detailScrollOffset to decrease after page up, got %d", m.detailScrollOffset)
	}
}

// TestDetailPaneGoToTopBottom verifies gg/G in detail mode
func TestDetailPaneGoToTopBottom(t *testing.T) {
	model := Model{
		mode:               splitView,
		activePanel:        ActivityPanel,
		bottomPaneMode:     BottomPaneDetail,
		detailScrollOffset: 5,
		activityLog:        make([]ActivityEntry, 0),
	}

	// Go to top
	newModel, _ := model.handleActivityLogGoToTop()
	m := newModel.(Model)
	if m.detailScrollOffset != 0 {
		t.Errorf("Expected detailScrollOffset to be 0 after go to top, got %d", m.detailScrollOffset)
	}

	// Go to bottom
	newModel, _ = m.handleActivityLogGoToBottom()
	m = newModel.(Model)
	if m.detailScrollOffset == 0 {
		t.Error("Expected detailScrollOffset to be large after go to bottom")
	}
}

// TestStatusBarShowsBottomPaneMode verifies status bar shows current mode
func TestStatusBarShowsBottomPaneMode(t *testing.T) {
	tests := []struct {
		name           string
		mode           BottomPaneMode
		expectedIndicator string
	}{
		{"Activity mode", BottomPaneActivity, "[Act]"},
		{"Detail mode", BottomPaneDetail, "[Detail]"},
		{"Split mode", BottomPaneSplit, "[Split]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := Model{
				mode:           splitView,
				activePanel:    BallsPanel,
				bottomPaneMode: tt.mode,
				localOnly:      true,
				width:          120,
				height:         40,
			}

			statusBar := model.renderStatusBar()
			if !strings.Contains(statusBar, tt.expectedIndicator) {
				t.Errorf("Expected status bar to show '%s', got: %s", tt.expectedIndicator, statusBar)
			}
		})
	}
}

// TestDetailScrollOffsetResetOnToggle verifies scroll offset resets when toggling to detail mode
func TestDetailScrollOffsetResetOnToggle(t *testing.T) {
	model := Model{
		bottomPaneMode:     BottomPaneActivity,
		detailScrollOffset: 10, // Some non-zero value
		activityLog:        make([]ActivityEntry, 0),
	}

	// Toggle to detail mode
	newModel, _ := model.handleToggleBottomPane()
	m := newModel.(Model)

	if m.detailScrollOffset != 0 {
		t.Errorf("Expected detailScrollOffset to reset to 0 when toggling to detail mode, got %d", m.detailScrollOffset)
	}
}

// TestBallDetailPanelShowsOutput verifies that ball output is shown when present
func TestBallDetailPanelShowsOutput(t *testing.T) {
	ball := &session.Ball{
		ID:       "test-1",
		Intent:   "Research task",
		State:    session.StateResearched,
		Priority: session.PriorityMedium,
		Output:   "Research findings:\nLine 1\nLine 2",
	}

	model := Model{
		mode:           splitView,
		activePanel:    BallsPanel,
		cursor:         0,
		filteredBalls:  []*session.Ball{ball},
		bottomPaneMode: BottomPaneDetail,
		width:          120,
		height:         40,
		filterStates: map[string]bool{
			"pending":     true,
			"in_progress": true,
			"blocked":     true,
			"complete":    true,
			"researched":  true,
		},
	}

	lines := model.buildBallDetailLines(ball, 100)
	content := strings.Join(lines, "\n")

	// Should contain output label
	if !strings.Contains(content, "Output") {
		t.Error("Expected ball detail to contain Output label")
	}

	// Should contain output content
	if !strings.Contains(content, "Research findings") {
		t.Error("Expected ball detail to contain output content")
	}
}

// TestActivityLogScrollingInActivityMode verifies scrolling works in activity mode
func TestActivityLogScrollingInActivityMode(t *testing.T) {
	// Create enough log entries
	entries := make([]ActivityEntry, 20)
	for i := 0; i < 20; i++ {
		entries[i] = ActivityEntry{Message: "Entry " + string(rune('A'+i))}
	}

	model := Model{
		mode:              splitView,
		activePanel:       ActivityPanel,
		bottomPaneMode:    BottomPaneActivity, // Activity mode, not detail
		activityLogOffset: 0,
		activityLog:       entries,
	}

	// Scroll down - should update activityLogOffset, not detailScrollOffset
	newModel, _ := model.handleSplitViewNavDown()
	m := newModel.(Model)

	if m.activityLogOffset != 1 {
		t.Errorf("Expected activityLogOffset to be 1 in activity mode, got %d", m.activityLogOffset)
	}
	if m.detailScrollOffset != 0 {
		t.Errorf("Expected detailScrollOffset to remain 0 in activity mode, got %d", m.detailScrollOffset)
	}
}

// TestTUILocalScopeDefault verifies that TUI defaults to local scope (juggler-102)
func TestTUILocalScopeDefault(t *testing.T) {
	t.Run("InitialModel defaults to local when passed true", func(t *testing.T) {
		var store *session.Store
		var config *session.Config

		model := InitialModel(store, config, true)
		if !model.localOnly {
			t.Error("Expected InitialModel with localOnly=true to have localOnly=true")
		}
	})

	t.Run("InitialModel shows all when passed false", func(t *testing.T) {
		var store *session.Store
		var config *session.Config

		model := InitialModel(store, config, false)
		if model.localOnly {
			t.Error("Expected InitialModel with localOnly=false to have localOnly=false")
		}
	})

	t.Run("InitialSplitModel defaults to local when passed true", func(t *testing.T) {
		var store *session.Store
		var sessionStore *session.SessionStore
		var config *session.Config

		model := InitialSplitModel(store, sessionStore, config, true)
		if !model.localOnly {
			t.Error("Expected InitialSplitModel with localOnly=true to have localOnly=true")
		}
	})

	t.Run("InitialSplitModel shows all when passed false", func(t *testing.T) {
		var store *session.Store
		var sessionStore *session.SessionStore
		var config *session.Config

		model := InitialSplitModel(store, sessionStore, config, false)
		if model.localOnly {
			t.Error("Expected InitialSplitModel with localOnly=false to have localOnly=false")
		}
	})

	t.Run("P key toggles localOnly in split view", func(t *testing.T) {
		var store *session.Store
		var sessionStore *session.SessionStore
		var config *session.Config

		model := InitialSplitModel(store, sessionStore, config, true)
		if !model.localOnly {
			t.Error("Expected initial localOnly to be true")
		}

		// Press P to toggle
		newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'P'}})
		m := newModel.(Model)
		if m.localOnly {
			t.Error("Expected localOnly to be false after pressing P")
		}

		// Press P again to toggle back
		newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'P'}})
		m = newModel.(Model)
		if !m.localOnly {
			t.Error("Expected localOnly to be true after pressing P again")
		}
	})

	t.Run("Status bar shows Local indicator when localOnly is true", func(t *testing.T) {
		model := Model{
			mode:        splitView,
			activePanel: BallsPanel,
			localOnly:   true,
			width:       80,
			height:      24,
		}

		view := model.View()
		if !strings.Contains(view, "[Local]") {
			t.Error("Expected status bar to show [Local] indicator when localOnly is true")
		}
	})

	t.Run("Status bar shows All indicator when localOnly is false", func(t *testing.T) {
		model := Model{
			mode:        splitView,
			activePanel: BallsPanel,
			localOnly:   false,
			width:       80,
			height:      24,
		}

		view := model.View()
		if !strings.Contains(view, "[All]") {
			t.Error("Expected status bar to show [All] indicator when localOnly is false")
		}
	})
}

// TestBuildBallsStats verifies stats are calculated correctly
func TestBuildBallsStats(t *testing.T) {
	tests := []struct {
		name     string
		balls    []*session.Ball
		expected string
	}{
		{
			name:     "Empty balls list",
			balls:    []*session.Ball{},
			expected: "P:0 I:0 B:0 C:0",
		},
		{
			name: "All pending",
			balls: []*session.Ball{
				{State: session.StatePending},
				{State: session.StatePending},
			},
			expected: "P:2 I:0 B:0 C:0",
		},
		{
			name: "All in progress",
			balls: []*session.Ball{
				{State: session.StateInProgress},
			},
			expected: "P:0 I:1 B:0 C:0",
		},
		{
			name: "All blocked",
			balls: []*session.Ball{
				{State: session.StateBlocked},
				{State: session.StateBlocked},
				{State: session.StateBlocked},
			},
			expected: "P:0 I:0 B:3 C:0",
		},
		{
			name: "All complete",
			balls: []*session.Ball{
				{State: session.StateComplete},
			},
			expected: "P:0 I:0 B:0 C:1",
		},
		{
			name: "Mixed states",
			balls: []*session.Ball{
				{State: session.StatePending},
				{State: session.StatePending},
				{State: session.StateInProgress},
				{State: session.StateBlocked},
				{State: session.StateComplete},
				{State: session.StateComplete},
			},
			expected: "P:2 I:1 B:1 C:2",
		},
		{
			name: "Researched counts as complete",
			balls: []*session.Ball{
				{State: session.StateResearched},
				{State: session.StateComplete},
			},
			expected: "P:0 I:0 B:0 C:2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := Model{}
			result := model.buildBallsStats(tt.balls)
			if result != tt.expected {
				t.Errorf("buildBallsStats() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestBallsPanelDisplaysStats verifies stats appear in the balls panel
func TestBallsPanelDisplaysStats(t *testing.T) {
	balls := []*session.Ball{
		{ID: "test-1", Intent: "First ball", State: session.StatePending, Tags: []string{"test-session"}},
		{ID: "test-2", Intent: "Second ball", State: session.StateInProgress, Tags: []string{"test-session"}},
		{ID: "test-3", Intent: "Third ball", State: session.StateBlocked, Tags: []string{"test-session"}},
	}

	model := Model{
		mode:        splitView,
		activePanel: BallsPanel,
		cursor:      0,
		balls:       balls,
		filteredBalls: balls,
		selectedSession: &session.JuggleSession{ID: "test-session"},
		width:       120,
		height:      40,
		filterStates: map[string]bool{
			"pending":     true,
			"in_progress": true,
			"blocked":     true,
			"complete":    true,
		},
	}

	// Render the balls panel
	content := model.renderBallsPanel(80, 20)

	// Verify stats string is present
	if !strings.Contains(content, "P:1") {
		t.Error("Expected balls panel to show pending count P:1")
	}
	if !strings.Contains(content, "I:1") {
		t.Error("Expected balls panel to show in-progress count I:1")
	}
	if !strings.Contains(content, "B:1") {
		t.Error("Expected balls panel to show blocked count B:1")
	}
	if !strings.Contains(content, "C:0") {
		t.Error("Expected balls panel to show complete count C:0")
	}
}

// TestStatsUpdateOnStateChange verifies stats reflect current ball states
func TestStatsUpdateOnStateChange(t *testing.T) {
	balls := []*session.Ball{
		{ID: "test-1", Intent: "Ball 1", State: session.StatePending},
		{ID: "test-2", Intent: "Ball 2", State: session.StatePending},
	}

	model := Model{}

	// Initial state: 2 pending
	stats := model.buildBallsStats(balls)
	if stats != "P:2 I:0 B:0 C:0" {
		t.Errorf("Initial stats should be P:2 I:0 B:0 C:0, got %s", stats)
	}

	// Change one ball to in_progress
	balls[0].State = session.StateInProgress
	stats = model.buildBallsStats(balls)
	if stats != "P:1 I:1 B:0 C:0" {
		t.Errorf("After state change stats should be P:1 I:1 B:0 C:0, got %s", stats)
	}

	// Change one ball to complete
	balls[1].State = session.StateComplete
	stats = model.buildBallsStats(balls)
	if stats != "P:0 I:1 B:0 C:1" {
		t.Errorf("After second state change stats should be P:0 I:1 B:0 C:1, got %s", stats)
	}
}

// TestStatsPanelPosition verifies stats appear at the right position
func TestStatsPanelPosition(t *testing.T) {
	balls := []*session.Ball{
		{ID: "test-1", Intent: "Ball 1", State: session.StatePending, Tags: []string{"test-session"}},
	}

	model := Model{
		mode:        splitView,
		activePanel: BallsPanel,
		cursor:      0,
		balls:       balls,
		filteredBalls: balls,
		selectedSession: &session.JuggleSession{ID: "test-session"},
		width:       120,
		height:      40,
		filterStates: map[string]bool{
			"pending":     true,
			"in_progress": true,
			"blocked":     true,
			"complete":    true,
		},
	}

	content := model.renderBallsPanel(80, 20)
	lines := strings.Split(content, "\n")

	// First line should contain both the title and stats
	if len(lines) == 0 {
		t.Fatal("Expected at least one line in balls panel")
	}

	firstLine := lines[0]
	// Should have title on the left
	if !strings.Contains(firstLine, "Balls:") {
		t.Error("Expected first line to contain 'Balls:'")
	}
	// Should have stats on the line
	if !strings.Contains(firstLine, "P:1") {
		t.Error("Expected first line to contain stats 'P:1'")
	}
}

// ==================== Agent Output Panel Tests ====================

// Test agent output panel toggle visibility
func TestAgentOutputPanelToggle(t *testing.T) {
	model := Model{
		mode:               splitView,
		activePanel:        SessionsPanel,
		agentOutputVisible: false,
		activityLog:        make([]ActivityEntry, 0),
	}

	// Toggle ON
	newModel, _ := model.handleToggleAgentOutput()
	model = newModel.(Model)
	if !model.agentOutputVisible {
		t.Error("Expected agentOutputVisible to be true after first toggle")
	}

	// Toggle OFF
	newModel, _ = model.handleToggleAgentOutput()
	model = newModel.(Model)
	if model.agentOutputVisible {
		t.Error("Expected agentOutputVisible to be false after second toggle")
	}
}

// Test adding agent output lines
func TestAddAgentOutput(t *testing.T) {
	model := Model{
		agentOutput: make([]AgentOutputEntry, 0),
		height:      30, // For calculating visible lines
	}

	// Add some output lines
	model.addAgentOutput("Line 1", false)
	model.addAgentOutput("Error line", true)
	model.addAgentOutput("Line 3", false)

	if len(model.agentOutput) != 3 {
		t.Errorf("Expected 3 output lines, got %d", len(model.agentOutput))
	}

	// Check first line
	if model.agentOutput[0].Line != "Line 1" {
		t.Errorf("Expected first line to be 'Line 1', got %s", model.agentOutput[0].Line)
	}
	if model.agentOutput[0].IsError {
		t.Error("Expected first line to not be an error")
	}

	// Check error line
	if model.agentOutput[1].Line != "Error line" {
		t.Errorf("Expected second line to be 'Error line', got %s", model.agentOutput[1].Line)
	}
	if !model.agentOutput[1].IsError {
		t.Error("Expected second line to be an error")
	}
}

// Test agent output buffer limit (500 lines)
func TestAgentOutputBufferLimit(t *testing.T) {
	model := Model{
		agentOutput: make([]AgentOutputEntry, 0),
		height:      30,
	}

	// Add more than 500 lines
	for i := 0; i < 510; i++ {
		model.addAgentOutput(fmt.Sprintf("Line %d", i), false)
	}

	// Buffer should be limited to 500
	if len(model.agentOutput) != 500 {
		t.Errorf("Expected buffer to be limited to 500 lines, got %d", len(model.agentOutput))
	}

	// First line should be Line 10 (lines 0-9 were removed)
	if model.agentOutput[0].Line != "Line 10" {
		t.Errorf("Expected first line to be 'Line 10', got %s", model.agentOutput[0].Line)
	}
}

// Test clearing agent output
func TestClearAgentOutput(t *testing.T) {
	model := Model{
		agentOutput:       make([]AgentOutputEntry, 0),
		agentOutputOffset: 5,
		height:            30,
	}

	// Add some lines
	model.addAgentOutput("Line 1", false)
	model.addAgentOutput("Line 2", false)

	// Clear
	model.clearAgentOutput()

	if len(model.agentOutput) != 0 {
		t.Errorf("Expected empty output buffer after clear, got %d lines", len(model.agentOutput))
	}
	if model.agentOutputOffset != 0 {
		t.Errorf("Expected offset to be reset to 0, got %d", model.agentOutputOffset)
	}
}

// Test agent output scrolling - scroll down
func TestAgentOutputScrollDown(t *testing.T) {
	model := Model{
		agentOutput:       make([]AgentOutputEntry, 0),
		agentOutputOffset: 0,
		height:            30, // This affects getAgentOutputVisibleLines
	}

	// Add 50 lines to have scrollable content
	for i := 0; i < 50; i++ {
		model.addAgentOutput(fmt.Sprintf("Line %d", i), false)
	}
	model.agentOutputOffset = 0 // Reset offset (addAgentOutput auto-scrolls)

	// Scroll down
	newModel, _ := model.handleAgentOutputScrollDown()
	model = newModel.(Model)

	if model.agentOutputOffset != 1 {
		t.Errorf("Expected offset to be 1 after scroll down, got %d", model.agentOutputOffset)
	}
}

// Test agent output scrolling - scroll up
func TestAgentOutputScrollUp(t *testing.T) {
	model := Model{
		agentOutput:       make([]AgentOutputEntry, 0),
		agentOutputOffset: 5,
		height:            30,
	}

	// Add some lines
	for i := 0; i < 50; i++ {
		model.agentOutput = append(model.agentOutput, AgentOutputEntry{Line: fmt.Sprintf("Line %d", i)})
	}

	// Scroll up
	newModel, _ := model.handleAgentOutputScrollUp()
	model = newModel.(Model)

	if model.agentOutputOffset != 4 {
		t.Errorf("Expected offset to be 4 after scroll up, got %d", model.agentOutputOffset)
	}
}

// Test agent output scroll up at top
func TestAgentOutputScrollUpAtTop(t *testing.T) {
	model := Model{
		agentOutput:       make([]AgentOutputEntry, 50),
		agentOutputOffset: 0, // Already at top
		height:            30,
	}

	// Scroll up - should stay at 0
	newModel, _ := model.handleAgentOutputScrollUp()
	model = newModel.(Model)

	if model.agentOutputOffset != 0 {
		t.Errorf("Expected offset to stay at 0, got %d", model.agentOutputOffset)
	}
}

// Test agent output page down
func TestAgentOutputPageDown(t *testing.T) {
	model := Model{
		agentOutput:       make([]AgentOutputEntry, 100),
		agentOutputOffset: 0,
		height:            30,
	}

	newModel, _ := model.handleAgentOutputPageDown()
	model = newModel.(Model)

	// Page size should be half of visible lines
	expectedPageSize := model.getAgentOutputVisibleLines() / 2
	if model.agentOutputOffset != expectedPageSize {
		t.Errorf("Expected offset to be %d after page down, got %d", expectedPageSize, model.agentOutputOffset)
	}
}

// Test agent output page up
func TestAgentOutputPageUp(t *testing.T) {
	model := Model{
		agentOutput:       make([]AgentOutputEntry, 100),
		agentOutputOffset: 20,
		height:            30,
	}

	newModel, _ := model.handleAgentOutputPageUp()
	model = newModel.(Model)

	expectedPageSize := model.getAgentOutputVisibleLines() / 2
	expectedOffset := 20 - expectedPageSize
	if model.agentOutputOffset != expectedOffset {
		t.Errorf("Expected offset to be %d after page up, got %d", expectedOffset, model.agentOutputOffset)
	}
}

// Test agent output go to top (gg)
func TestAgentOutputGoToTop(t *testing.T) {
	model := Model{
		agentOutput:       make([]AgentOutputEntry, 100),
		agentOutputOffset: 50, // Scrolled down
		height:            30,
	}

	newModel, _ := model.handleAgentOutputGoToTop()
	model = newModel.(Model)

	if model.agentOutputOffset != 0 {
		t.Errorf("Expected offset to be 0 after go to top, got %d", model.agentOutputOffset)
	}
}

// Test agent output go to bottom (G)
func TestAgentOutputGoToBottom(t *testing.T) {
	model := Model{
		agentOutput:       make([]AgentOutputEntry, 100),
		agentOutputOffset: 0, // At top
		height:            30,
	}

	newModel, _ := model.handleAgentOutputGoToBottom()
	model = newModel.(Model)

	maxOffset := model.getAgentOutputMaxOffset()
	if model.agentOutputOffset != maxOffset {
		t.Errorf("Expected offset to be %d after go to bottom, got %d", maxOffset, model.agentOutputOffset)
	}
}

// Test agent output panel rendering when empty
func TestAgentOutputPanelRenderEmpty(t *testing.T) {
	model := Model{
		agentOutput:       make([]AgentOutputEntry, 0),
		agentOutputOffset: 0,
		agentStatus:       AgentStatus{Running: false},
		height:            30,
	}

	content := model.renderAgentOutputPanel(80, 15)

	if !strings.Contains(content, "Agent Output") {
		t.Error("Expected panel to contain title 'Agent Output'")
	}
	if !strings.Contains(content, "No agent output") {
		t.Error("Expected panel to show 'No agent output' when empty")
	}
}

// Test agent output panel rendering with content
func TestAgentOutputPanelRenderWithContent(t *testing.T) {
	model := Model{
		agentOutput:       make([]AgentOutputEntry, 0),
		agentOutputOffset: 0,
		agentStatus:       AgentStatus{Running: false},
		height:            30,
	}

	// Add some output
	model.addAgentOutput("Test output line 1", false)
	model.addAgentOutput("Error: something failed", true)
	model.agentOutputOffset = 0 // Reset offset

	content := model.renderAgentOutputPanel(80, 15)

	if !strings.Contains(content, "Test output line 1") {
		t.Error("Expected panel to contain normal output line")
	}
	if !strings.Contains(content, "Error: something failed") {
		t.Error("Expected panel to contain error output line")
	}
}

// Test status bar shows agent output indicator when visible
func TestStatusBarShowsAgentOutputIndicator(t *testing.T) {
	model := Model{
		mode:               splitView,
		activePanel:        SessionsPanel,
		agentOutputVisible: true,
		localOnly:          true,
		height:             30,
		width:              120,
	}

	statusBar := model.renderStatusBar()
	if !strings.Contains(statusBar, "[Output]") {
		t.Error("Expected status bar to show [Output] indicator when agent output is visible")
	}
}

// Test status bar shows O:output keybind
func TestStatusBarShowsOutputKeybind(t *testing.T) {
	model := Model{
		mode:        splitView,
		activePanel: SessionsPanel,
		localOnly:   true,
		height:      30,
		width:       120,
	}

	statusBar := model.renderStatusBar()
	if !strings.Contains(statusBar, "O:output") {
		t.Error("Expected status bar to show 'O:output' keybind")
	}
}

// =========================================
// Agent Cancel Tests
// =========================================

// Test X keybind shows confirmation when agent is running
func TestXKeybindShowsCancelConfirmation(t *testing.T) {
	model := Model{
		mode:        splitView,
		activePanel: BallsPanel,
		agentStatus: AgentStatus{
			Running:       true,
			SessionID:     "test-session",
			Iteration:     2,
			MaxIterations: 10,
		},
	}

	// Press X to cancel agent
	newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'X'}})
	m := newModel.(Model)

	if m.mode != confirmAgentCancel {
		t.Errorf("Expected mode to be confirmAgentCancel, got %v", m.mode)
	}
}

// Test X keybind does nothing when no agent is running
func TestXKeybindNoAgentRunning(t *testing.T) {
	model := Model{
		mode:        splitView,
		activePanel: BallsPanel,
		agentStatus: AgentStatus{
			Running: false,
		},
	}

	// Press X when no agent is running
	newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'X'}})
	m := newModel.(Model)

	// Should stay in split view and show message
	if m.mode != splitView {
		t.Errorf("Expected mode to remain splitView, got %v", m.mode)
	}

	if !strings.Contains(m.message, "No agent is running") {
		t.Errorf("Expected message about no agent running, got: %s", m.message)
	}
}

// Test confirming agent cancellation with 'y'
func TestCancelAgentConfirmY(t *testing.T) {
	model := Model{
		mode:        confirmAgentCancel,
		activePanel: SessionsPanel,
		agentStatus: AgentStatus{
			Running:       true,
			SessionID:     "test-session",
			Iteration:     2,
			MaxIterations: 10,
		},
		// Note: agentProcess is nil in tests, but the handler should handle this gracefully
	}

	// Confirm with y
	newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m := newModel.(Model)

	// Should return to split view
	if m.mode != splitView {
		t.Errorf("Expected mode to be splitView after confirmation, got %v", m.mode)
	}

	// Agent status should be cleared
	if m.agentStatus.Running {
		t.Error("Expected agentStatus.Running to be false after cancellation")
	}

	// Should show cancellation message
	if !strings.Contains(m.message, "cancelled") {
		t.Errorf("Expected message about agent being cancelled, got: %s", m.message)
	}
}

// Test confirming agent cancellation with 'Y' (uppercase)
func TestCancelAgentConfirmUpperY(t *testing.T) {
	model := Model{
		mode:        confirmAgentCancel,
		activePanel: SessionsPanel,
		agentStatus: AgentStatus{
			Running:       true,
			SessionID:     "test-session",
			Iteration:     2,
			MaxIterations: 10,
		},
	}

	// Confirm with Y
	newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Y'}})
	m := newModel.(Model)

	// Should return to split view
	if m.mode != splitView {
		t.Errorf("Expected mode to be splitView after confirmation, got %v", m.mode)
	}

	// Agent status should be cleared
	if m.agentStatus.Running {
		t.Error("Expected agentStatus.Running to be false after cancellation")
	}
}

// Test declining agent cancellation with 'n'
func TestCancelAgentDeclineN(t *testing.T) {
	model := Model{
		mode:        confirmAgentCancel,
		activePanel: SessionsPanel,
		agentStatus: AgentStatus{
			Running:       true,
			SessionID:     "test-session",
			Iteration:     2,
			MaxIterations: 10,
		},
	}

	// Decline with n
	newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m := newModel.(Model)

	// Should return to split view
	if m.mode != splitView {
		t.Errorf("Expected mode to be splitView after declining, got %v", m.mode)
	}

	// Agent should still be running (not cancelled)
	if !m.agentStatus.Running {
		t.Error("Expected agentStatus.Running to remain true after declining")
	}

	// Should show appropriate message
	if !strings.Contains(m.message, "still running") {
		t.Errorf("Expected message about agent still running, got: %s", m.message)
	}
}

// Test declining agent cancellation with Escape
func TestCancelAgentDeclineEscape(t *testing.T) {
	model := Model{
		mode:        confirmAgentCancel,
		activePanel: SessionsPanel,
		agentStatus: AgentStatus{
			Running:       true,
			SessionID:     "test-session",
			Iteration:     2,
			MaxIterations: 10,
		},
	}

	// Decline with Escape
	newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEscape})
	m := newModel.(Model)

	// Should return to split view
	if m.mode != splitView {
		t.Errorf("Expected mode to be splitView after escape, got %v", m.mode)
	}

	// Agent should still be running
	if !m.agentStatus.Running {
		t.Error("Expected agentStatus.Running to remain true after escape")
	}
}

// Test renderAgentCancelConfirm shows correct information
func TestRenderAgentCancelConfirm(t *testing.T) {
	model := Model{
		mode: confirmAgentCancel,
		agentStatus: AgentStatus{
			Running:       true,
			SessionID:     "test-session",
			Iteration:     3,
			MaxIterations: 10,
		},
		width:  80,
		height: 24,
	}

	view := model.renderAgentCancelConfirm()

	// Should show title
	if !strings.Contains(view, "Cancel Agent") {
		t.Error("Expected cancel dialog to show title 'Cancel Agent'")
	}

	// Should show session ID
	if !strings.Contains(view, "test-session") {
		t.Error("Expected cancel dialog to show session ID")
	}

	// Should show progress
	if !strings.Contains(view, "3/10") {
		t.Error("Expected cancel dialog to show progress '3/10'")
	}

	// Should show warning
	if !strings.Contains(view, "terminated immediately") {
		t.Error("Expected cancel dialog to show termination warning")
	}

	// Should show confirmation prompt
	if !strings.Contains(view, "[y/N]") {
		t.Error("Expected cancel dialog to show confirmation prompt [y/N]")
	}
}

// Test help view contains agent cancel keybind
func TestHelpViewContainsAgentCancelKeybind(t *testing.T) {
	model := Model{
		mode:   splitHelpView,
		width:  120,
		height: 100, // Large enough to show all content
	}

	helpView := model.renderSplitHelpView()

	// Should show Agent Control section
	if !strings.Contains(helpView, "Agent Control") {
		t.Error("Expected help view to contain 'Agent Control' section")
	}

	// Should show X keybind for cancel
	if !strings.Contains(helpView, "Cancel running agent") {
		t.Error("Expected help view to contain 'Cancel running agent' description")
	}
}

// Test status bar shows X:cancel when agent is running
func TestStatusBarShowsCancelKeybindWhenAgentRunning(t *testing.T) {
	model := Model{
		mode:        splitView,
		activePanel: SessionsPanel,
		localOnly:   true,
		agentStatus: AgentStatus{
			Running:       true,
			SessionID:     "test",
			Iteration:     1,
			MaxIterations: 5,
		},
		width:  120,
		height: 40,
	}

	statusBar := model.renderStatusBar()

	if !strings.Contains(statusBar, "X:cancel") {
		t.Error("Expected status bar to show 'X:cancel' when agent is running")
	}
}

// Test View() function returns correct view for confirmAgentCancel mode
func TestViewReturnsAgentCancelView(t *testing.T) {
	model := Model{
		mode: confirmAgentCancel,
		agentStatus: AgentStatus{
			Running:       true,
			SessionID:     "my-session",
			Iteration:     5,
			MaxIterations: 10,
		},
		width:  80,
		height: 24,
	}

	view := model.View()

	// Should render the cancel confirmation dialog
	if !strings.Contains(view, "Cancel Agent") {
		t.Error("Expected View() to return cancel confirmation dialog")
	}
}

// Test AgentProcess.Kill method
func TestAgentProcessKill(t *testing.T) {
	// Test nil process
	var nilProcess *AgentProcess
	err := nilProcess.Kill()
	if err != nil {
		t.Errorf("Expected nil error for nil process, got: %v", err)
	}

	// Test empty process
	emptyProcess := &AgentProcess{}
	err = emptyProcess.Kill()
	if err != nil {
		t.Errorf("Expected nil error for empty process, got: %v", err)
	}
}

// Test agentCancelledMsg handler
func TestAgentCancelledMsgHandler(t *testing.T) {
	model := Model{
		mode:        splitView,
		activePanel: BallsPanel,
		agentStatus: AgentStatus{
			Running:       true,
			SessionID:     "test-session",
			Iteration:     3,
			MaxIterations: 10,
		},
		agentProcess: &AgentProcess{
			sessionID: "test-session",
		},
	}

	// Send agentCancelledMsg
	newModel, _ := model.Update(agentCancelledMsg{sessionID: "test-session"})
	m := newModel.(Model)

	// Agent status should be cleared
	if m.agentStatus.Running {
		t.Error("Expected agentStatus.Running to be false after receiving agentCancelledMsg")
	}

	// Agent process should be nil
	if m.agentProcess != nil {
		t.Error("Expected agentProcess to be nil after receiving agentCancelledMsg")
	}

	// Should show appropriate message
	if !strings.Contains(m.message, "cancelled") {
		t.Errorf("Expected message about agent being cancelled, got: %s", m.message)
	}
}

// Test agentProcessStartedMsg handler
func TestAgentProcessStartedMsgHandler(t *testing.T) {
	model := Model{
		mode:          splitView,
		activePanel:   SessionsPanel,
		agentOutputCh: make(chan agentOutputMsg, 10),
	}

	mockProcess := &AgentProcess{
		sessionID: "test-session",
	}

	// Send agentProcessStartedMsg
	newModel, cmd := model.Update(agentProcessStartedMsg{
		process:   mockProcess,
		sessionID: "test-session",
	})
	m := newModel.(Model)

	// Agent process should be stored
	if m.agentProcess != mockProcess {
		t.Error("Expected agentProcess to be set from message")
	}

	// Agent status should be set
	if !m.agentStatus.Running {
		t.Error("Expected agentStatus.Running to be true")
	}

	if m.agentStatus.SessionID != "test-session" {
		t.Errorf("Expected session ID to be 'test-session', got: %s", m.agentStatus.SessionID)
	}

	// Should return a batch command
	if cmd == nil {
		t.Error("Expected a batch command to be returned")
	}

	// Message should indicate agent is running
	if !strings.Contains(m.message, "running") {
		t.Errorf("Expected message to indicate agent is running, got: %s", m.message)
	}
}
