package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/ohare93/juggle/internal/session"
)

// newTestTextInput creates a textinput for testing
func newTestTextInput() textinput.Model {
	ti := textinput.New()
	ti.CharLimit = 256
	ti.Width = 40
	return ti
}

func TestBallToYAML(t *testing.T) {
	tests := []struct {
		name     string
		ball     *session.Ball
		contains []string
	}{
		{
			name: "basic ball",
			ball: &session.Ball{
				ID:       "test-1",
				Intent:   "Test intent",
				Priority: session.PriorityMedium,
				State:    session.StatePending,
			},
			contains: []string{
				"id: test-1",
				"intent: Test intent",
				"priority: medium",
				"state: pending",
			},
		},
		{
			name: "ball with acceptance criteria",
			ball: &session.Ball{
				ID:                 "test-2",
				Intent:             "Multi-criteria task",
				Priority:           session.PriorityHigh,
				State:              session.StateInProgress,
				AcceptanceCriteria: []string{"First criterion", "Second criterion"},
			},
			contains: []string{
				"id: test-2",
				"intent: Multi-criteria task",
				"priority: high",
				"state: in_progress",
				"acceptance_criteria:",
				"- First criterion",
				"- Second criterion",
			},
		},
		{
			name: "ball with blocked reason",
			ball: &session.Ball{
				ID:            "test-3",
				Intent:        "Blocked task",
				Priority:      session.PriorityLow,
				State:         session.StateBlocked,
				BlockedReason: "Waiting for dependency",
			},
			contains: []string{
				"id: test-3",
				"state: blocked",
				"blocked_reason: Waiting for dependency",
			},
		},
		{
			name: "ball with tags",
			ball: &session.Ball{
				ID:       "test-4",
				Intent:   "Tagged task",
				Priority: session.PriorityMedium,
				State:    session.StatePending,
				Tags:     []string{"backend", "api"},
			},
			contains: []string{
				"id: test-4",
				"tags:",
				"- backend",
				"- api",
			},
		},
		{
			name: "ball with model size",
			ball: &session.Ball{
				ID:        "test-5",
				Intent:    "Model size task",
				Priority:  session.PriorityMedium,
				State:     session.StatePending,
				ModelSize: session.ModelSizeLarge,
			},
			contains: []string{
				"id: test-5",
				"model_size: large",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yaml, err := ballToYAML(tt.ball)
			if err != nil {
				t.Fatalf("ballToYAML() error = %v", err)
			}

			for _, expected := range tt.contains {
				if !strings.Contains(yaml, expected) {
					t.Errorf("ballToYAML() output missing %q\nGot:\n%s", expected, yaml)
				}
			}

			// Verify header comment exists
			if !strings.Contains(yaml, "# Edit ball properties") {
				t.Error("ballToYAML() should include header comment")
			}
		})
	}
}

func TestYamlToBall_ValidInput(t *testing.T) {
	tests := []struct {
		name         string
		yamlContent  string
		initialBall  *session.Ball
		expectedBall *session.Ball
	}{
		{
			name: "update intent",
			yamlContent: `
id: test-1
intent: Updated intent
priority: medium
state: pending
`,
			initialBall: &session.Ball{
				ID:       "test-1",
				Intent:   "Original intent",
				Priority: session.PriorityMedium,
				State:    session.StatePending,
			},
			expectedBall: &session.Ball{
				ID:       "test-1",
				Intent:   "Updated intent",
				Priority: session.PriorityMedium,
				State:    session.StatePending,
			},
		},
		{
			name: "update priority",
			yamlContent: `
id: test-2
intent: Test task
priority: urgent
state: pending
`,
			initialBall: &session.Ball{
				ID:       "test-2",
				Intent:   "Test task",
				Priority: session.PriorityMedium,
				State:    session.StatePending,
			},
			expectedBall: &session.Ball{
				ID:       "test-2",
				Intent:   "Test task",
				Priority: session.PriorityUrgent,
				State:    session.StatePending,
			},
		},
		{
			name: "update state",
			yamlContent: `
id: test-3
intent: Test task
priority: medium
state: in_progress
`,
			initialBall: &session.Ball{
				ID:       "test-3",
				Intent:   "Test task",
				Priority: session.PriorityMedium,
				State:    session.StatePending,
			},
			expectedBall: &session.Ball{
				ID:       "test-3",
				Intent:   "Test task",
				Priority: session.PriorityMedium,
				State:    session.StateInProgress,
			},
		},
		{
			name: "update acceptance criteria",
			yamlContent: `
id: test-4
intent: Test task
priority: medium
state: pending
acceptance_criteria:
  - New criterion 1
  - New criterion 2
`,
			initialBall: &session.Ball{
				ID:                 "test-4",
				Intent:             "Test task",
				Priority:           session.PriorityMedium,
				State:              session.StatePending,
				AcceptanceCriteria: []string{"Old criterion"},
			},
			expectedBall: &session.Ball{
				ID:                 "test-4",
				Intent:             "Test task",
				Priority:           session.PriorityMedium,
				State:              session.StatePending,
				AcceptanceCriteria: []string{"New criterion 1", "New criterion 2"},
			},
		},
		{
			name: "update tags",
			yamlContent: `
id: test-5
intent: Test task
priority: medium
state: pending
tags:
  - new-tag
  - another-tag
`,
			initialBall: &session.Ball{
				ID:       "test-5",
				Intent:   "Test task",
				Priority: session.PriorityMedium,
				State:    session.StatePending,
				Tags:     []string{"old-tag"},
			},
			expectedBall: &session.Ball{
				ID:       "test-5",
				Intent:   "Test task",
				Priority: session.PriorityMedium,
				State:    session.StatePending,
				Tags:     []string{"new-tag", "another-tag"},
			},
		},
		{
			name: "update blocked reason",
			yamlContent: `
id: test-6
intent: Blocked task
priority: medium
state: blocked
blocked_reason: New blocking reason
`,
			initialBall: &session.Ball{
				ID:            "test-6",
				Intent:        "Blocked task",
				Priority:      session.PriorityMedium,
				State:         session.StateBlocked,
				BlockedReason: "Old reason",
			},
			expectedBall: &session.Ball{
				ID:            "test-6",
				Intent:        "Blocked task",
				Priority:      session.PriorityMedium,
				State:         session.StateBlocked,
				BlockedReason: "New blocking reason",
			},
		},
		{
			name: "update model size",
			yamlContent: `
id: test-7
intent: Test task
priority: medium
state: pending
model_size: small
`,
			initialBall: &session.Ball{
				ID:        "test-7",
				Intent:    "Test task",
				Priority:  session.PriorityMedium,
				State:     session.StatePending,
				ModelSize: session.ModelSizeLarge,
			},
			expectedBall: &session.Ball{
				ID:        "test-7",
				Intent:    "Test task",
				Priority:  session.PriorityMedium,
				State:     session.StatePending,
				ModelSize: session.ModelSizeSmall,
			},
		},
		{
			name: "clear model size with empty string",
			yamlContent: `
id: test-8
intent: Test task
priority: medium
state: pending
model_size: ""
`,
			initialBall: &session.Ball{
				ID:        "test-8",
				Intent:    "Test task",
				Priority:  session.PriorityMedium,
				State:     session.StatePending,
				ModelSize: session.ModelSizeLarge,
			},
			expectedBall: &session.Ball{
				ID:        "test-8",
				Intent:    "Test task",
				Priority:  session.PriorityMedium,
				State:     session.StatePending,
				ModelSize: session.ModelSizeBlank,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a copy of the initial ball
			ball := *tt.initialBall

			err := yamlToBall(tt.yamlContent, &ball)
			if err != nil {
				t.Fatalf("yamlToBall() error = %v", err)
			}

			// Check fields
			if ball.Intent != tt.expectedBall.Intent {
				t.Errorf("Intent = %q, want %q", ball.Intent, tt.expectedBall.Intent)
			}
			if ball.Priority != tt.expectedBall.Priority {
				t.Errorf("Priority = %q, want %q", ball.Priority, tt.expectedBall.Priority)
			}
			if ball.State != tt.expectedBall.State {
				t.Errorf("State = %q, want %q", ball.State, tt.expectedBall.State)
			}
			if ball.BlockedReason != tt.expectedBall.BlockedReason {
				t.Errorf("BlockedReason = %q, want %q", ball.BlockedReason, tt.expectedBall.BlockedReason)
			}
			if ball.ModelSize != tt.expectedBall.ModelSize {
				t.Errorf("ModelSize = %q, want %q", ball.ModelSize, tt.expectedBall.ModelSize)
			}

			// Check acceptance criteria
			if len(ball.AcceptanceCriteria) != len(tt.expectedBall.AcceptanceCriteria) {
				t.Errorf("AcceptanceCriteria length = %d, want %d", len(ball.AcceptanceCriteria), len(tt.expectedBall.AcceptanceCriteria))
			} else {
				for i, ac := range ball.AcceptanceCriteria {
					if ac != tt.expectedBall.AcceptanceCriteria[i] {
						t.Errorf("AcceptanceCriteria[%d] = %q, want %q", i, ac, tt.expectedBall.AcceptanceCriteria[i])
					}
				}
			}

			// Check tags
			if len(ball.Tags) != len(tt.expectedBall.Tags) {
				t.Errorf("Tags length = %d, want %d", len(ball.Tags), len(tt.expectedBall.Tags))
			} else {
				for i, tag := range ball.Tags {
					if tag != tt.expectedBall.Tags[i] {
						t.Errorf("Tags[%d] = %q, want %q", i, tag, tt.expectedBall.Tags[i])
					}
				}
			}
		})
	}
}

func TestYamlToBall_InvalidInput(t *testing.T) {
	tests := []struct {
		name        string
		yamlContent string
		errContains string
	}{
		{
			name: "invalid priority",
			yamlContent: `
id: test-1
intent: Test task
priority: invalid_priority
state: pending
`,
			errContains: "invalid priority",
		},
		{
			name: "invalid state",
			yamlContent: `
id: test-2
intent: Test task
priority: medium
state: invalid_state
`,
			errContains: "invalid state",
		},
		{
			name: "invalid model size",
			yamlContent: `
id: test-3
intent: Test task
priority: medium
state: pending
model_size: extra_large
`,
			errContains: "invalid model_size",
		},
		{
			name:        "malformed yaml",
			yamlContent: `{{{not valid yaml`,
			errContains: "failed to parse YAML",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ball := &session.Ball{
				ID:       "test",
				Intent:   "Test",
				Priority: session.PriorityMedium,
				State:    session.StatePending,
			}

			err := yamlToBall(tt.yamlContent, ball)
			if err == nil {
				t.Fatal("yamlToBall() expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("yamlToBall() error = %q, should contain %q", err.Error(), tt.errContains)
			}
		})
	}
}

func TestYamlToBall_PreservesID(t *testing.T) {
	// Verify that ID cannot be changed through YAML editing
	yamlContent := `
id: new-id-attempt
intent: Test task
priority: medium
state: pending
`
	ball := &session.Ball{
		ID:       "original-id",
		Intent:   "Original",
		Priority: session.PriorityMedium,
		State:    session.StatePending,
	}

	err := yamlToBall(yamlContent, ball)
	if err != nil {
		t.Fatalf("yamlToBall() error = %v", err)
	}

	// ID should remain unchanged (ID is not modified by yamlToBall)
	if ball.ID != "original-id" {
		t.Errorf("ID was changed from 'original-id' to %q", ball.ID)
	}
}

func TestBallToYAML_RoundTrip(t *testing.T) {
	// Test that a ball can be converted to YAML and back without data loss
	originalBall := &session.Ball{
		ID:                 "roundtrip-1",
		Intent:             "Roundtrip test task",
		Priority:           session.PriorityHigh,
		State:              session.StateBlocked,
		BlockedReason:      "Test blocker",
		Tags:               []string{"test", "roundtrip"},
		AcceptanceCriteria: []string{"Criterion 1", "Criterion 2", "Criterion 3"},
		ModelSize:          session.ModelSizeMedium,
	}

	// Convert to YAML
	yamlContent, err := ballToYAML(originalBall)
	if err != nil {
		t.Fatalf("ballToYAML() error = %v", err)
	}

	// Create new ball and parse YAML back
	parsedBall := &session.Ball{ID: originalBall.ID}
	err = yamlToBall(yamlContent, parsedBall)
	if err != nil {
		t.Fatalf("yamlToBall() error = %v", err)
	}

	// Verify all fields match
	if parsedBall.Intent != originalBall.Intent {
		t.Errorf("Intent mismatch: got %q, want %q", parsedBall.Intent, originalBall.Intent)
	}
	if parsedBall.Priority != originalBall.Priority {
		t.Errorf("Priority mismatch: got %q, want %q", parsedBall.Priority, originalBall.Priority)
	}
	if parsedBall.State != originalBall.State {
		t.Errorf("State mismatch: got %q, want %q", parsedBall.State, originalBall.State)
	}
	if parsedBall.BlockedReason != originalBall.BlockedReason {
		t.Errorf("BlockedReason mismatch: got %q, want %q", parsedBall.BlockedReason, originalBall.BlockedReason)
	}
	if parsedBall.ModelSize != originalBall.ModelSize {
		t.Errorf("ModelSize mismatch: got %q, want %q", parsedBall.ModelSize, originalBall.ModelSize)
	}

	// Verify acceptance criteria
	if len(parsedBall.AcceptanceCriteria) != len(originalBall.AcceptanceCriteria) {
		t.Fatalf("AcceptanceCriteria length mismatch: got %d, want %d",
			len(parsedBall.AcceptanceCriteria), len(originalBall.AcceptanceCriteria))
	}
	for i, ac := range parsedBall.AcceptanceCriteria {
		if ac != originalBall.AcceptanceCriteria[i] {
			t.Errorf("AcceptanceCriteria[%d] mismatch: got %q, want %q", i, ac, originalBall.AcceptanceCriteria[i])
		}
	}

	// Verify tags
	if len(parsedBall.Tags) != len(originalBall.Tags) {
		t.Fatalf("Tags length mismatch: got %d, want %d", len(parsedBall.Tags), len(originalBall.Tags))
	}
	for i, tag := range parsedBall.Tags {
		if tag != originalBall.Tags[i] {
			t.Errorf("Tags[%d] mismatch: got %q, want %q", i, tag, originalBall.Tags[i])
		}
	}
}

func TestHandleEditorResult_Success(t *testing.T) {
	ball := &session.Ball{
		ID:         "test-1",
		Intent:     "Original intent",
		Priority:   session.PriorityMedium,
		State:      session.StatePending,
		WorkingDir: "/tmp/test-project",
	}

	model := Model{
		activityLog: make([]ActivityEntry, 0),
	}

	editedYAML := `
id: test-1
intent: Updated intent
priority: high
state: in_progress
`

	msg := editorResultMsg{
		ball:       ball,
		editedYAML: editedYAML,
		cancelled:  false,
		err:        nil,
	}

	// Note: handleEditorResult tries to create a store, which will fail in tests
	// We're primarily testing that it processes the message correctly
	newModel, _ := model.handleEditorResult(msg)
	updatedModel := newModel.(Model)

	// The ball should have been updated
	if ball.Intent != "Updated intent" {
		t.Errorf("Ball intent not updated: got %q, want %q", ball.Intent, "Updated intent")
	}
	if ball.Priority != session.PriorityHigh {
		t.Errorf("Ball priority not updated: got %q, want %q", ball.Priority, session.PriorityHigh)
	}
	if ball.State != session.StateInProgress {
		t.Errorf("Ball state not updated: got %q, want %q", ball.State, session.StateInProgress)
	}

	// Check activity log was updated (the error message about store will be logged)
	if len(updatedModel.activityLog) == 0 {
		t.Error("Activity log should have an entry")
	}
}

func TestHandleEditorResult_Cancelled(t *testing.T) {
	ball := &session.Ball{
		ID:         "test-1",
		Intent:     "Original intent",
		Priority:   session.PriorityMedium,
		State:      session.StatePending,
		WorkingDir: "/tmp/test-project",
	}

	model := Model{
		activityLog: make([]ActivityEntry, 0),
	}

	msg := editorResultMsg{
		ball:      ball,
		cancelled: true,
	}

	newModel, _ := model.handleEditorResult(msg)
	updatedModel := newModel.(Model)

	// Ball should not have changed
	if ball.Intent != "Original intent" {
		t.Errorf("Ball intent should not have changed on cancel: got %q", ball.Intent)
	}

	// Check message indicates cancellation
	if !strings.Contains(updatedModel.message, "cancel") {
		t.Errorf("Message should indicate cancellation: got %q", updatedModel.message)
	}
}

func TestHandleEditorResult_Error(t *testing.T) {
	ball := &session.Ball{
		ID:         "test-1",
		Intent:     "Original intent",
		Priority:   session.PriorityMedium,
		State:      session.StatePending,
		WorkingDir: "/tmp/test-project",
	}

	model := Model{
		activityLog: make([]ActivityEntry, 0),
	}

	msg := editorResultMsg{
		ball: ball,
		err:  fmt.Errorf("test editor error"),
	}

	newModel, _ := model.handleEditorResult(msg)
	updatedModel := newModel.(Model)

	// Ball should not have changed
	if ball.Intent != "Original intent" {
		t.Errorf("Ball intent should not have changed on error: got %q", ball.Intent)
	}

	// Check message indicates error
	if !strings.Contains(updatedModel.message, "error") && !strings.Contains(updatedModel.message, "Error") {
		t.Errorf("Message should indicate error: got %q", updatedModel.message)
	}
}

func TestHandleEditorResult_ParseError(t *testing.T) {
	ball := &session.Ball{
		ID:         "test-1",
		Intent:     "Original intent",
		Priority:   session.PriorityMedium,
		State:      session.StatePending,
		WorkingDir: "/tmp/test-project",
	}

	model := Model{
		activityLog: make([]ActivityEntry, 0),
	}

	// Invalid YAML that will cause parse error
	msg := editorResultMsg{
		ball:       ball,
		editedYAML: `{{{invalid yaml content`,
		cancelled:  false,
	}

	newModel, _ := model.handleEditorResult(msg)
	updatedModel := newModel.(Model)

	// Ball should not have changed
	if ball.Intent != "Original intent" {
		t.Errorf("Ball intent should not have changed on parse error: got %q", ball.Intent)
	}

	// Check message indicates parse error
	if !strings.Contains(updatedModel.message, "Parse error") && !strings.Contains(updatedModel.message, "error") {
		t.Errorf("Message should indicate parse error: got %q", updatedModel.message)
	}
}

func TestHandleSplitEditItem_NoBallSelected(t *testing.T) {
	model := Model{
		mode:          splitView,
		activePanel:   BallsPanel,
		cursor:        0,
		filteredBalls: []*session.Ball{}, // Empty list
		activityLog:   make([]ActivityEntry, 0),
		textInput:     newTestTextInput(),
		filterStates: map[string]bool{
			"pending":     true,
			"in_progress": true,
			"blocked":     true,
			"complete":    true,
		},
	}

	newModel, _ := model.handleSplitEditItem()
	updatedModel := newModel.(Model)

	// Should show "No ball selected" message
	if !strings.Contains(updatedModel.message, "No ball selected") {
		t.Errorf("Expected 'No ball selected' message, got %q", updatedModel.message)
	}
}

func TestHandleSplitEditItem_SessionCannotEditPseudo(t *testing.T) {
	tests := []struct {
		name      string
		sessionID string
	}{
		{"all pseudo session", PseudoSessionAll},
		{"untagged pseudo session", PseudoSessionUntagged},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := Model{
				mode:          splitView,
				activePanel:   SessionsPanel,
				sessionCursor: 0,
				sessions: []*session.JuggleSession{
					{ID: tt.sessionID, Description: "Pseudo session"},
				},
				activityLog: make([]ActivityEntry, 0),
				textInput:   newTestTextInput(),
			}

			newModel, _ := model.handleSplitEditItem()
			updatedModel := newModel.(Model)

			// Should show "Cannot edit built-in session" message
			if !strings.Contains(updatedModel.message, "Cannot edit built-in session") {
				t.Errorf("Expected 'Cannot edit built-in session' message, got %q", updatedModel.message)
			}
		})
	}
}

func TestBallYAMLStruct(t *testing.T) {
	// Test that BallYAML struct has all expected fields
	yamlBall := BallYAML{
		ID:                 "test-id",
		Intent:             "test intent",
		Priority:           "high",
		State:              "pending",
		BlockedReason:      "test reason",
		Tags:               []string{"tag1", "tag2"},
		AcceptanceCriteria: []string{"ac1", "ac2"},
		ModelSize:          "large",
	}

	if yamlBall.ID != "test-id" {
		t.Errorf("ID = %q, want %q", yamlBall.ID, "test-id")
	}
	if yamlBall.Intent != "test intent" {
		t.Errorf("Intent = %q, want %q", yamlBall.Intent, "test intent")
	}
	if yamlBall.Priority != "high" {
		t.Errorf("Priority = %q, want %q", yamlBall.Priority, "high")
	}
	if yamlBall.State != "pending" {
		t.Errorf("State = %q, want %q", yamlBall.State, "pending")
	}
	if yamlBall.BlockedReason != "test reason" {
		t.Errorf("BlockedReason = %q, want %q", yamlBall.BlockedReason, "test reason")
	}
	if len(yamlBall.Tags) != 2 {
		t.Errorf("Tags length = %d, want 2", len(yamlBall.Tags))
	}
	if len(yamlBall.AcceptanceCriteria) != 2 {
		t.Errorf("AcceptanceCriteria length = %d, want 2", len(yamlBall.AcceptanceCriteria))
	}
	if yamlBall.ModelSize != "large" {
		t.Errorf("ModelSize = %q, want %q", yamlBall.ModelSize, "large")
	}
}

func TestEditKeyInSplitView(t *testing.T) {
	// Test that 'e' key triggers edit in BallsPanel
	ball := &session.Ball{
		ID:         "test-1",
		Intent:     "Test task",
		Priority:   session.PriorityMedium,
		State:      session.StatePending,
		WorkingDir: "/tmp/test",
	}

	model := Model{
		mode:        splitView,
		activePanel: BallsPanel,
		cursor:      0,
		balls:       []*session.Ball{ball},
		filteredBalls: []*session.Ball{ball},
		filterStates: map[string]bool{
			"pending":     true,
			"in_progress": true,
			"blocked":     true,
			"complete":    true,
		},
		selectedSession: &session.JuggleSession{ID: PseudoSessionAll},
		activityLog:     make([]ActivityEntry, 0),
		textInput:       newTestTextInput(),
	}

	// Simulate pressing 'e' key
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}}
	newModel, cmd := model.Update(msg)
	updatedModel := newModel.(Model)

	// Verify that edit was initiated
	if updatedModel.editingBall != ball {
		t.Error("editingBall should be set to the selected ball")
	}

	// The cmd should be openEditorCmd (we can't easily check the specific command)
	if cmd == nil {
		t.Error("Expected a command to be returned for opening editor")
	}

	// Check that activity log was updated
	if len(updatedModel.activityLog) == 0 {
		t.Error("Activity log should have an entry for edit initiation")
	}
}

func TestYamlToBall_AllStates(t *testing.T) {
	// Test all valid states can be set via YAML
	states := []struct {
		yamlState string
		expected  session.BallState
	}{
		{"pending", session.StatePending},
		{"in_progress", session.StateInProgress},
		{"complete", session.StateComplete},
		{"blocked", session.StateBlocked},
		{"researched", session.StateResearched},
	}

	for _, s := range states {
		t.Run(s.yamlState, func(t *testing.T) {
			yamlContent := fmt.Sprintf(`
id: test
intent: Test
priority: medium
state: %s
`, s.yamlState)

			ball := &session.Ball{
				ID:       "test",
				Intent:   "Test",
				Priority: session.PriorityMedium,
				State:    session.StatePending,
			}

			err := yamlToBall(yamlContent, ball)
			if err != nil {
				t.Fatalf("yamlToBall() error = %v", err)
			}
			if ball.State != s.expected {
				t.Errorf("State = %q, want %q", ball.State, s.expected)
			}
		})
	}
}

func TestYamlToBall_AllPriorities(t *testing.T) {
	// Test all valid priorities can be set via YAML
	priorities := []struct {
		yamlPriority string
		expected     session.Priority
	}{
		{"low", session.PriorityLow},
		{"medium", session.PriorityMedium},
		{"high", session.PriorityHigh},
		{"urgent", session.PriorityUrgent},
	}

	for _, p := range priorities {
		t.Run(p.yamlPriority, func(t *testing.T) {
			yamlContent := fmt.Sprintf(`
id: test
intent: Test
priority: %s
state: pending
`, p.yamlPriority)

			ball := &session.Ball{
				ID:       "test",
				Intent:   "Test",
				Priority: session.PriorityMedium,
				State:    session.StatePending,
			}

			err := yamlToBall(yamlContent, ball)
			if err != nil {
				t.Fatalf("yamlToBall() error = %v", err)
			}
			if ball.Priority != p.expected {
				t.Errorf("Priority = %q, want %q", ball.Priority, p.expected)
			}
		})
	}
}

func TestYamlToBall_AllModelSizes(t *testing.T) {
	// Test all valid model sizes can be set via YAML
	sizes := []struct {
		yamlSize string
		expected session.ModelSize
	}{
		{"small", session.ModelSizeSmall},
		{"medium", session.ModelSizeMedium},
		{"large", session.ModelSizeLarge},
		{"", session.ModelSizeBlank},
	}

	for _, s := range sizes {
		name := s.yamlSize
		if name == "" {
			name = "empty"
		}
		t.Run(name, func(t *testing.T) {
			var yamlContent string
			if s.yamlSize == "" {
				yamlContent = `
id: test
intent: Test
priority: medium
state: pending
`
			} else {
				yamlContent = fmt.Sprintf(`
id: test
intent: Test
priority: medium
state: pending
model_size: %s
`, s.yamlSize)
			}

			ball := &session.Ball{
				ID:        "test",
				Intent:    "Test",
				Priority:  session.PriorityMedium,
				State:     session.StatePending,
				ModelSize: session.ModelSizeLarge, // Start with different value
			}

			err := yamlToBall(yamlContent, ball)
			if err != nil {
				t.Fatalf("yamlToBall() error = %v", err)
			}
			if ball.ModelSize != s.expected {
				t.Errorf("ModelSize = %q, want %q", ball.ModelSize, s.expected)
			}
		})
	}
}

func TestYamlToBall_UpdatesActivity(t *testing.T) {
	yamlContent := `
id: test
intent: Test
priority: medium
state: pending
`
	ball := &session.Ball{
		ID:           "test",
		Intent:       "Test",
		Priority:     session.PriorityMedium,
		State:        session.StatePending,
		LastActivity: time.Time{}, // Zero time
	}

	err := yamlToBall(yamlContent, ball)
	if err != nil {
		t.Fatalf("yamlToBall() error = %v", err)
	}

	// LastActivity should be updated to current time
	if ball.LastActivity.IsZero() {
		t.Error("LastActivity should be updated after YAML parsing")
	}

	// Should be approximately now
	if time.Since(ball.LastActivity) > time.Second {
		t.Error("LastActivity should be updated to approximately now")
	}
}
