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
				Title:   "Test intent",
				Priority: session.PriorityMedium,
				State:    session.StatePending,
			},
			contains: []string{
				"id: test-1",
				"title: Test intent",
				"priority: medium",
				"state: pending",
			},
		},
		{
			name: "ball with acceptance criteria",
			ball: &session.Ball{
				ID:                 "test-2",
				Title:             "Multi-criteria task",
				Priority:           session.PriorityHigh,
				State:              session.StateInProgress,
				AcceptanceCriteria: []string{"First criterion", "Second criterion"},
			},
			contains: []string{
				"id: test-2",
				"title: Multi-criteria task",
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
				Title:        "Blocked task",
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
				Title:   "Tagged task",
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
				Title:    "Model size task",
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
title: Updated intent
priority: medium
state: pending
`,
			initialBall: &session.Ball{
				ID:       "test-1",
				Title:   "Original intent",
				Priority: session.PriorityMedium,
				State:    session.StatePending,
			},
			expectedBall: &session.Ball{
				ID:       "test-1",
				Title:   "Updated intent",
				Priority: session.PriorityMedium,
				State:    session.StatePending,
			},
		},
		{
			name: "update priority",
			yamlContent: `
id: test-2
title: Test task
priority: urgent
state: pending
`,
			initialBall: &session.Ball{
				ID:       "test-2",
				Title:   "Test task",
				Priority: session.PriorityMedium,
				State:    session.StatePending,
			},
			expectedBall: &session.Ball{
				ID:       "test-2",
				Title:   "Test task",
				Priority: session.PriorityUrgent,
				State:    session.StatePending,
			},
		},
		{
			name: "update state",
			yamlContent: `
id: test-3
title: Test task
priority: medium
state: in_progress
`,
			initialBall: &session.Ball{
				ID:       "test-3",
				Title:   "Test task",
				Priority: session.PriorityMedium,
				State:    session.StatePending,
			},
			expectedBall: &session.Ball{
				ID:       "test-3",
				Title:   "Test task",
				Priority: session.PriorityMedium,
				State:    session.StateInProgress,
			},
		},
		{
			name: "update acceptance criteria",
			yamlContent: `
id: test-4
title: Test task
priority: medium
state: pending
acceptance_criteria:
  - New criterion 1
  - New criterion 2
`,
			initialBall: &session.Ball{
				ID:                 "test-4",
				Title:             "Test task",
				Priority:           session.PriorityMedium,
				State:              session.StatePending,
				AcceptanceCriteria: []string{"Old criterion"},
			},
			expectedBall: &session.Ball{
				ID:                 "test-4",
				Title:             "Test task",
				Priority:           session.PriorityMedium,
				State:              session.StatePending,
				AcceptanceCriteria: []string{"New criterion 1", "New criterion 2"},
			},
		},
		{
			name: "update tags",
			yamlContent: `
id: test-5
title: Test task
priority: medium
state: pending
tags:
  - new-tag
  - another-tag
`,
			initialBall: &session.Ball{
				ID:       "test-5",
				Title:   "Test task",
				Priority: session.PriorityMedium,
				State:    session.StatePending,
				Tags:     []string{"old-tag"},
			},
			expectedBall: &session.Ball{
				ID:       "test-5",
				Title:   "Test task",
				Priority: session.PriorityMedium,
				State:    session.StatePending,
				Tags:     []string{"new-tag", "another-tag"},
			},
		},
		{
			name: "update blocked reason",
			yamlContent: `
id: test-6
title: Blocked task
priority: medium
state: blocked
blocked_reason: New blocking reason
`,
			initialBall: &session.Ball{
				ID:            "test-6",
				Title:        "Blocked task",
				Priority:      session.PriorityMedium,
				State:         session.StateBlocked,
				BlockedReason: "Old reason",
			},
			expectedBall: &session.Ball{
				ID:            "test-6",
				Title:        "Blocked task",
				Priority:      session.PriorityMedium,
				State:         session.StateBlocked,
				BlockedReason: "New blocking reason",
			},
		},
		{
			name: "update model size",
			yamlContent: `
id: test-7
title: Test task
priority: medium
state: pending
model_size: small
`,
			initialBall: &session.Ball{
				ID:        "test-7",
				Title:    "Test task",
				Priority:  session.PriorityMedium,
				State:     session.StatePending,
				ModelSize: session.ModelSizeLarge,
			},
			expectedBall: &session.Ball{
				ID:        "test-7",
				Title:    "Test task",
				Priority:  session.PriorityMedium,
				State:     session.StatePending,
				ModelSize: session.ModelSizeSmall,
			},
		},
		{
			name: "clear model size with empty string",
			yamlContent: `
id: test-8
title: Test task
priority: medium
state: pending
model_size: ""
`,
			initialBall: &session.Ball{
				ID:        "test-8",
				Title:    "Test task",
				Priority:  session.PriorityMedium,
				State:     session.StatePending,
				ModelSize: session.ModelSizeLarge,
			},
			expectedBall: &session.Ball{
				ID:        "test-8",
				Title:    "Test task",
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
			if ball.Title != tt.expectedBall.Title {
				t.Errorf("Intent = %q, want %q", ball.Title, tt.expectedBall.Title)
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
title: Test task
priority: invalid_priority
state: pending
`,
			errContains: "invalid priority",
		},
		{
			name: "invalid state",
			yamlContent: `
id: test-2
title: Test task
priority: medium
state: invalid_state
`,
			errContains: "invalid state",
		},
		{
			name: "invalid model size",
			yamlContent: `
id: test-3
title: Test task
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
				Title:   "Test",
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
title: Test task
priority: medium
state: pending
`
	ball := &session.Ball{
		ID:       "original-id",
		Title:   "Original",
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
		Title:             "Roundtrip test task",
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
	if parsedBall.Title != originalBall.Title {
		t.Errorf("Intent mismatch: got %q, want %q", parsedBall.Title, originalBall.Title)
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
		Title:     "Original intent",
		Priority:   session.PriorityMedium,
		State:      session.StatePending,
		WorkingDir: "/tmp/test-project",
	}

	model := Model{
		activityLog: make([]ActivityEntry, 0),
	}

	editedYAML := `
id: test-1
title: Updated intent
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
	if ball.Title != "Updated intent" {
		t.Errorf("Ball intent not updated: got %q, want %q", ball.Title, "Updated intent")
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
		Title:     "Original intent",
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
	if ball.Title != "Original intent" {
		t.Errorf("Ball intent should not have changed on cancel: got %q", ball.Title)
	}

	// Check message indicates cancellation
	if !strings.Contains(updatedModel.message, "cancel") {
		t.Errorf("Message should indicate cancellation: got %q", updatedModel.message)
	}
}

func TestHandleEditorResult_Error(t *testing.T) {
	ball := &session.Ball{
		ID:         "test-1",
		Title:     "Original intent",
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
	if ball.Title != "Original intent" {
		t.Errorf("Ball intent should not have changed on error: got %q", ball.Title)
	}

	// Check message indicates error
	if !strings.Contains(updatedModel.message, "error") && !strings.Contains(updatedModel.message, "Error") {
		t.Errorf("Message should indicate error: got %q", updatedModel.message)
	}
}

func TestHandleEditorResult_ParseError(t *testing.T) {
	ball := &session.Ball{
		ID:         "test-1",
		Title:     "Original intent",
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
	if ball.Title != "Original intent" {
		t.Errorf("Ball intent should not have changed on parse error: got %q", ball.Title)
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
		Title:             "test intent",
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
	if yamlBall.Title != "test intent" {
		t.Errorf("Intent = %q, want %q", yamlBall.Title, "test intent")
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
	// Test that 'e' key opens the unified ball form (not editor)
	ball := &session.Ball{
		ID:         "test-1",
		Title:     "Test task",
		Priority:   session.PriorityMedium,
		State:      session.StatePending,
		WorkingDir: "/tmp/test",
		Tags:       []string{"tag1"},
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
	newModel, _ := model.Update(msg)
	updatedModel := newModel.(Model)

	// Verify that edit form was opened
	if updatedModel.mode != unifiedBallFormView {
		t.Errorf("Expected mode to be unifiedBallFormView, got %v", updatedModel.mode)
	}

	if updatedModel.editingBall != ball {
		t.Error("editingBall should be set to the selected ball")
	}

	if updatedModel.inputAction != actionEdit {
		t.Error("inputAction should be actionEdit")
	}

	// Verify form was prepopulated with ball values
	if updatedModel.pendingBallIntent != ball.Title {
		t.Errorf("Expected pendingBallIntent to be %q, got %q", ball.Title, updatedModel.pendingBallIntent)
	}

	// Check that activity log was updated
	if len(updatedModel.activityLog) == 0 {
		t.Error("Activity log should have an entry for edit initiation")
	}
}

func TestShiftEKeyOpenEditorInSplitView(t *testing.T) {
	// Test that 'E' key (uppercase) opens the editor
	ball := &session.Ball{
		ID:         "test-1",
		Title:     "Test task",
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

	// Simulate pressing 'E' key
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'E'}}
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
title: Test
priority: medium
state: %s
`, s.yamlState)

			ball := &session.Ball{
				ID:       "test",
				Title:   "Test",
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
title: Test
priority: %s
state: pending
`, p.yamlPriority)

			ball := &session.Ball{
				ID:       "test",
				Title:   "Test",
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
title: Test
priority: medium
state: pending
`
			} else {
				yamlContent = fmt.Sprintf(`
id: test
title: Test
priority: medium
state: pending
model_size: %s
`, s.yamlSize)
			}

			ball := &session.Ball{
				ID:        "test",
				Title:    "Test",
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
title: Test
priority: medium
state: pending
`
	ball := &session.Ball{
		ID:           "test",
		Title:       "Test",
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

func TestBallToYAML_ShowsAllFieldsEvenWhenEmpty(t *testing.T) {
	// Test that all fields are shown in YAML even when empty
	ball := &session.Ball{
		ID:                 "test-1",
		Title:             "Test task",
		Priority:           session.PriorityMedium,
		State:              session.StatePending,
		BlockedReason:      "",    // Empty
		Tags:               nil,   // Nil
		AcceptanceCriteria: nil,   // Nil
		ModelSize:          "",    // Empty
	}

	yaml, err := ballToYAML(ball)
	if err != nil {
		t.Fatalf("ballToYAML() error = %v", err)
	}

	// All fields should be present
	requiredFields := []string{
		"id:",
		"title:",
		"priority:",
		"state:",
		"blocked_reason:",
		"tags:",
		"acceptance_criteria:",
		"model_size:",
	}

	for _, field := range requiredFields {
		if !strings.Contains(yaml, field) {
			t.Errorf("YAML missing field %q:\n%s", field, yaml)
		}
	}
}

func TestYamlToBall_EmptyValuesPreserveRequired(t *testing.T) {
	// Test that empty required fields preserve existing values
	yamlContent := `
id: test
title: ""
priority: ""
state: ""
`
	ball := &session.Ball{
		ID:       "test",
		Title:   "Original Intent",
		Priority: session.PriorityHigh,
		State:    session.StateInProgress,
	}

	err := yamlToBall(yamlContent, ball)
	if err != nil {
		t.Fatalf("yamlToBall() error = %v", err)
	}

	// Required fields should keep original values
	if ball.Title != "Original Intent" {
		t.Errorf("Intent should preserve original value, got %q", ball.Title)
	}
	if ball.Priority != session.PriorityHigh {
		t.Errorf("Priority should preserve original value, got %q", ball.Priority)
	}
	if ball.State != session.StateInProgress {
		t.Errorf("State should preserve original value, got %q", ball.State)
	}
}

func TestYamlToBall_WhitespaceOnlyValuesPreserveRequired(t *testing.T) {
	// Test that whitespace-only required fields preserve existing values
	yamlContent := `
id: test
title: "   "
priority: "  "
state: "	"
`
	ball := &session.Ball{
		ID:       "test",
		Title:   "Original Intent",
		Priority: session.PriorityHigh,
		State:    session.StateInProgress,
	}

	err := yamlToBall(yamlContent, ball)
	if err != nil {
		t.Fatalf("yamlToBall() error = %v", err)
	}

	// Required fields should keep original values when whitespace-only
	if ball.Title != "Original Intent" {
		t.Errorf("Intent should preserve original value, got %q", ball.Title)
	}
	if ball.Priority != session.PriorityHigh {
		t.Errorf("Priority should preserve original value, got %q", ball.Priority)
	}
	if ball.State != session.StateInProgress {
		t.Errorf("State should preserve original value, got %q", ball.State)
	}
}

func TestYamlToBall_EmptyOptionalFieldsClear(t *testing.T) {
	// Test that empty optional fields can clear existing values
	yamlContent := `
id: test
title: Test
priority: medium
state: pending
blocked_reason: ""
tags: []
acceptance_criteria: []
model_size: ""
`
	ball := &session.Ball{
		ID:                 "test",
		Title:             "Test",
		Priority:           session.PriorityMedium,
		State:              session.StatePending,
		BlockedReason:      "Was blocked",
		Tags:               []string{"tag1", "tag2"},
		AcceptanceCriteria: []string{"AC1", "AC2"},
		ModelSize:          session.ModelSizeLarge,
	}

	err := yamlToBall(yamlContent, ball)
	if err != nil {
		t.Fatalf("yamlToBall() error = %v", err)
	}

	// Optional fields should be cleared
	if ball.BlockedReason != "" {
		t.Errorf("BlockedReason should be cleared, got %q", ball.BlockedReason)
	}
	if len(ball.Tags) != 0 {
		t.Errorf("Tags should be cleared, got %v", ball.Tags)
	}
	if len(ball.AcceptanceCriteria) != 0 {
		t.Errorf("AcceptanceCriteria should be cleared, got %v", ball.AcceptanceCriteria)
	}
	if ball.ModelSize != session.ModelSizeBlank {
		t.Errorf("ModelSize should be blank, got %q", ball.ModelSize)
	}
}

func TestYamlToBall_TrimsWhitespaceFromArrayElements(t *testing.T) {
	// Test that whitespace is trimmed from array elements
	yamlContent := `
id: test
title: Test
priority: medium
state: pending
tags:
  - "  tag1  "
  - "tag2"
  - "   "
  - ""
acceptance_criteria:
  - "  AC1  "
  - "AC2"
  - "   "
`
	ball := &session.Ball{
		ID:       "test",
		Title:   "Test",
		Priority: session.PriorityMedium,
		State:    session.StatePending,
	}

	err := yamlToBall(yamlContent, ball)
	if err != nil {
		t.Fatalf("yamlToBall() error = %v", err)
	}

	// Tags should be trimmed and empty ones removed
	if len(ball.Tags) != 2 {
		t.Errorf("Expected 2 tags (empty ones removed), got %d: %v", len(ball.Tags), ball.Tags)
	}
	if ball.Tags[0] != "tag1" {
		t.Errorf("Tag should be trimmed, got %q", ball.Tags[0])
	}

	// ACs should be trimmed and empty ones removed
	if len(ball.AcceptanceCriteria) != 2 {
		t.Errorf("Expected 2 ACs (empty ones removed), got %d: %v", len(ball.AcceptanceCriteria), ball.AcceptanceCriteria)
	}
	if ball.AcceptanceCriteria[0] != "AC1" {
		t.Errorf("AC should be trimmed, got %q", ball.AcceptanceCriteria[0])
	}
}
