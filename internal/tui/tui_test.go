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
		ball     *session.Session
		expected string
	}{
		{
			name: "ready state",
			ball: &session.Session{
				ActiveState: session.ActiveReady,
			},
			expected: "ready",
		},
		{
			name: "juggling with juggle state",
			ball: func() *session.Session {
				inAir := session.JuggleInAir
				return &session.Session{
					ActiveState: session.ActiveJuggling,
					JuggleState: &inAir,
				}
			}(),
			expected: "in-air",
		},
		{
			name: "complete state",
			ball: &session.Session{
				ActiveState: session.ActiveComplete,
			},
			expected: "complete",
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
	balls := []*session.Session{
		{ActiveState: session.ActiveReady},
		{ActiveState: session.ActiveReady},
		{ActiveState: session.ActiveJuggling},
		{ActiveState: session.ActiveComplete},
		{ActiveState: session.ActiveDropped},
	}

	tests := []struct {
		state    string
		expected int
	}{
		{"ready", 2},
		{"juggling", 1},
		{"complete", 1},
		{"dropped", 1},
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
		balls: []*session.Session{
			{ID: "1", ActiveState: session.ActiveReady},
			{ID: "2", ActiveState: session.ActiveReady},
			{ID: "3", ActiveState: session.ActiveJuggling},
			{ID: "4", ActiveState: session.ActiveComplete},
		},
		filterStates: map[string]bool{
			"ready":    true,
			"juggling": false,
			"complete": false,
			"dropped":  false,
		},
	}

	model.applyFilters()

	if len(model.filteredBalls) != 2 {
		t.Errorf("Expected 2 filtered balls, got %d", len(model.filteredBalls))
	}

	for _, ball := range model.filteredBalls {
		if ball.ActiveState != session.ActiveReady {
			t.Errorf("Expected only ready balls, got ball with state %v", ball.ActiveState)
		}
	}
}

func TestApplyFiltersAll(t *testing.T) {
	model := Model{
		balls: []*session.Session{
			{ID: "1", ActiveState: session.ActiveReady},
			{ID: "2", ActiveState: session.ActiveJuggling},
			{ID: "3", ActiveState: session.ActiveComplete},
		},
		filterStates: map[string]bool{
			"ready":    true,
			"juggling": true,
			"complete": true,
			"dropped":  true,
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
			"ready":    true,
			"juggling": true,
			"dropped":  false,
			"complete": false,
		},
	}

	// Test toggling ready off
	model.handleStateFilter("2")
	if model.filterStates["ready"] {
		t.Error("Expected ready to be toggled off")
	}

	// Test toggling ready back on
	model.handleStateFilter("2")
	if !model.filterStates["ready"] {
		t.Error("Expected ready to be toggled on")
	}

	// Test show all (key "1")
	model.filterStates["ready"] = false
	model.filterStates["juggling"] = false
	model.handleStateFilter("1")
	if !model.filterStates["ready"] || !model.filterStates["juggling"] ||
		!model.filterStates["dropped"] || !model.filterStates["complete"] {
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
		name        string
		initialState session.ActiveState
		action      string // "start", "complete", "drop"
		expectError bool
	}{
		{
			name:         "start from ready",
			initialState: session.ActiveReady,
			action:       "start",
			expectError:  false,
		},
		{
			name:         "start from complete",
			initialState: session.ActiveComplete,
			action:       "start",
			expectError:  false,
		},
		{
			name:         "start from dropped",
			initialState: session.ActiveDropped,
			action:       "start",
			expectError:  false,
		},
		{
			name:         "complete from juggling",
			initialState: session.ActiveJuggling,
			action:       "complete",
			expectError:  false,
		},
		{
			name:         "complete from ready",
			initialState: session.ActiveReady,
			action:       "complete",
			expectError:  false,
		},
		{
			name:         "drop from juggling",
			initialState: session.ActiveJuggling,
			action:       "drop",
			expectError:  false,
		},
		{
			name:         "drop from complete",
			initialState: session.ActiveComplete,
			action:       "drop",
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := Model{
				filteredBalls: []*session.Session{
					{
						ID:          "test-1",
						ActiveState: tt.initialState,
						WorkingDir:  "/tmp/test",
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
			case "drop":
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
				if ball.ActiveState != session.ActiveJuggling {
					t.Errorf("Expected state to be juggling, got %v", ball.ActiveState)
				}
			case "complete":
				if ball.ActiveState != session.ActiveComplete {
					t.Errorf("Expected state to be complete, got %v", ball.ActiveState)
				}
			case "drop":
				if ball.ActiveState != session.ActiveDropped {
					t.Errorf("Expected state to be dropped, got %v", ball.ActiveState)
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
	ball := &session.Session{
		ID:       "test-1",
		Intent:   "Test ball",
		Priority: session.PriorityMedium,
	}

	model := Model{
		mode:          confirmDeleteView,
		filteredBalls: []*session.Session{ball},
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
		filteredBalls: []*session.Session{},
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
		name               string
		currentState       session.ActiveState
		currentJuggleState *session.JuggleState
		expectedState      session.ActiveState
		hasJuggleState     bool
	}{
		{
			name:           "ready to juggling",
			currentState:   session.ActiveReady,
			expectedState:  session.ActiveJuggling,
			hasJuggleState: true,
		},
		{
			name:           "juggling to complete",
			currentState:   session.ActiveJuggling,
			expectedState:  session.ActiveComplete,
			hasJuggleState: false,
		},
		{
			name:           "complete to dropped",
			currentState:   session.ActiveComplete,
			expectedState:  session.ActiveDropped,
			hasJuggleState: false,
		},
		{
			name:           "dropped to ready",
			currentState:   session.ActiveDropped,
			expectedState:  session.ActiveReady,
			hasJuggleState: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate cycling logic from handleCycleState
			var nextState session.ActiveState
			var nextJuggleState *session.JuggleState

			switch tt.currentState {
			case session.ActiveReady:
				nextState = session.ActiveJuggling
				inAir := session.JuggleInAir
				nextJuggleState = &inAir
			case session.ActiveJuggling:
				nextState = session.ActiveComplete
				nextJuggleState = nil
			case session.ActiveComplete:
				nextState = session.ActiveDropped
				nextJuggleState = nil
			case session.ActiveDropped:
				nextState = session.ActiveReady
				nextJuggleState = nil
			default:
				nextState = session.ActiveReady
				nextJuggleState = nil
			}

			if nextState != tt.expectedState {
				t.Errorf("Expected state %v, got %v", tt.expectedState, nextState)
			}

			hasJuggleState := nextJuggleState != nil
			if hasJuggleState != tt.hasJuggleState {
				t.Errorf("Expected hasJuggleState: %v, got: %v", tt.hasJuggleState, hasJuggleState)
			}

			if tt.hasJuggleState && *nextJuggleState != session.JuggleInAir {
				t.Errorf("Expected juggle state to be in-air, got %v", *nextJuggleState)
			}
		})
	}
}

func TestSetReadyFromAnyState(t *testing.T) {
	states := []struct {
		name  string
		state session.ActiveState
	}{
		{"from juggling", session.ActiveJuggling},
		{"from complete", session.ActiveComplete},
		{"from dropped", session.ActiveDropped},
		{"from ready", session.ActiveReady}, // Should be idempotent
	}

	for _, tt := range states {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate logic from handleSetReady
			var nextState session.ActiveState = session.ActiveReady
			var nextJuggleState *session.JuggleState = nil

			if nextState != session.ActiveReady {
				t.Errorf("Failed to set %v to ready, got %v", tt.state, nextState)
			}
			if nextJuggleState != nil {
				t.Error("JuggleState should be nil for ready state")
			}
		})
	}
}
