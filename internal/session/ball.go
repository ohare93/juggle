package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Priority levels for balls
type Priority string

const (
	PriorityLow    Priority = "low"
	PriorityMedium Priority = "medium"
	PriorityHigh   Priority = "high"
	PriorityUrgent Priority = "urgent"
)

// ModelSize specifies preferred LLM model size for cost optimization
type ModelSize string

const (
	ModelSizeBlank  ModelSize = ""       // Default - use large or session default
	ModelSizeSmall  ModelSize = "small"  // Maps to haiku or equivalent fast model
	ModelSizeMedium ModelSize = "medium" // Maps to sonnet or equivalent balanced model
	ModelSizeLarge  ModelSize = "large"  // Maps to opus or equivalent capable model
)

// BallState represents the lifecycle state of a ball
type BallState string

const (
	StatePending    BallState = "pending"
	StateInProgress BallState = "in_progress"
	StateComplete   BallState = "complete"
	StateBlocked    BallState = "blocked"
)

// TestsState represents whether tests are needed/done for a ball
type TestsState string

const (
	TestsStateUnset    TestsState = ""           // Default - not specified
	TestsStateNotNeeded TestsState = "not_needed" // Tests are not required for this task
	TestsStateNeeded   TestsState = "needed"     // Tests are required but not yet done
	TestsStateDone     TestsState = "done"       // Tests have been completed
)


// Ball represents a task being tracked
type Ball struct {
	ID                 string      `json:"id"`
	WorkingDir         string      `json:"-"` // Computed from file location, not stored
	Intent             string      `json:"intent"`
	AcceptanceCriteria []string    `json:"acceptance_criteria,omitempty"`
	Priority           Priority    `json:"priority"`
	State              BallState   `json:"state"`
	BlockedReason      string      `json:"blocked_reason,omitempty"`
	TestsState         TestsState  `json:"tests_state,omitempty"`
	StartedAt          time.Time   `json:"started_at"`
	LastActivity       time.Time   `json:"last_activity"`
	CompletedAt        *time.Time  `json:"completed_at,omitempty"`
	UpdateCount        int         `json:"update_count"`
	Tags               []string    `json:"tags,omitempty"`
	CompletionNote     string      `json:"completion_note,omitempty"`
	ModelSize          ModelSize   `json:"model_size,omitempty"`
}

// UnmarshalJSON implements custom unmarshaling to handle migration from old format
func (b *Ball) UnmarshalJSON(data []byte) error {
	var bj ballJSON
	if err := json.Unmarshal(data, &bj); err != nil {
		return err
	}

	// Copy all standard fields
	b.ID = bj.ID
	b.Intent = bj.Intent
	b.Priority = bj.Priority
	b.StartedAt = bj.StartedAt
	b.LastActivity = bj.LastActivity
	b.UpdateCount = bj.UpdateCount
	b.Tags = bj.Tags
	b.CompletionNote = bj.CompletionNote
	b.ModelSize = bj.ModelSize
	b.TestsState = bj.TestsState

	// Handle acceptance criteria (migrate from legacy description if needed)
	if len(bj.AcceptanceCriteria) > 0 {
		b.AcceptanceCriteria = bj.AcceptanceCriteria
	} else if bj.Description != "" {
		b.AcceptanceCriteria = []string{bj.Description}
	}

	// Migrate state from various formats to new BallState
	if bj.State != "" {
		// Newest format - use State directly
		b.State = BallState(bj.State)
		b.BlockedReason = bj.BlockedReason
	} else if bj.ActiveState != "" {
		// Previous format with active_state/juggle_state - migrate
		switch bj.ActiveState {
		case "ready":
			b.State = StatePending
		case "juggling":
			b.State = StateInProgress
		case "dropped":
			b.State = StateBlocked
			if bj.StateMessage != "" {
				b.BlockedReason = bj.StateMessage
			} else {
				b.BlockedReason = "dropped"
			}
		case "complete":
			b.State = StateComplete
		default:
			b.State = StatePending
		}
		// JuggleState substates are collapsed into in_progress
		// StateMessage becomes BlockedReason only for blocked state
		if b.State != StateBlocked && bj.StateMessage != "" {
			// For non-blocked states, preserve message in BlockedReason temporarily
			// This will be empty on next save unless state is blocked
		}
	} else if bj.Status != "" {
		// Oldest format with status field - migrate
		switch bj.Status {
		case "planned":
			b.State = StatePending
		case "active":
			b.State = StateInProgress
		case "blocked":
			b.State = StateBlocked
			b.BlockedReason = bj.Blocker
		case "needs-review":
			b.State = StateInProgress
		case "done":
			b.State = StateComplete
		default:
			b.State = StatePending
		}
	} else {
		// No state info, default to pending
		b.State = StatePending
	}

	return nil
}

// ballJSON is used for custom JSON unmarshaling to handle migration from old format
type ballJSON struct {
	ID                 string          `json:"id"`
	Intent             string          `json:"intent"`
	AcceptanceCriteria []string        `json:"acceptance_criteria,omitempty"` // New: list of acceptance criteria
	Description        string          `json:"description,omitempty"`         // Legacy: single description
	Priority           Priority        `json:"priority"`
	// Newest format (v3)
	State              string          `json:"state,omitempty"`            // New: pending/in_progress/complete/blocked
	BlockedReason      string          `json:"blocked_reason,omitempty"`   // Reason when state is blocked
	TestsState         TestsState      `json:"tests_state,omitempty"`      // Whether tests are needed/done
	// Previous format (v2)
	ActiveState        string          `json:"active_state,omitempty"`     // Old: ready/juggling/dropped/complete
	JuggleState        *string         `json:"juggle_state,omitempty"`     // Old: needs-thrown/in-air/needs-caught
	StateMessage       string          `json:"state_message,omitempty"`    // Old state context message
	// Oldest format (v1)
	Status             string          `json:"status,omitempty"`           // Old: planned/active/blocked/needs-review/done
	Blocker            string          `json:"blocker,omitempty"`          // Old blocker field
	// Common fields
	StartedAt          time.Time       `json:"started_at"`
	LastActivity       time.Time       `json:"last_activity"`
	UpdateCount        int             `json:"update_count"`
	Tags               []string        `json:"tags,omitempty"`
	CompletionNote     string          `json:"completion_note,omitempty"`
	ModelSize          ModelSize       `json:"model_size,omitempty"` // Preferred LLM model size
}

// NewBall creates a new ball with the given parameters in pending state
func NewBall(workingDir, intent string, priority Priority) (*Ball, error) {
	now := time.Now()
	id, err := generateID(workingDir)
	if err != nil {
		return nil, err
	}

	ball := &Ball{
		ID:           id,
		WorkingDir:   workingDir,
		Intent:       intent,
		Priority:     priority,
		State:        StatePending,
		StartedAt:    now,
		LastActivity: now,
		UpdateCount:  0,
		Tags:         []string{},
	}
	return ball, nil
}

// generateID creates a unique ball ID from working dir and counter
func generateID(workingDir string) (string, error) {
	base := filepath.Base(workingDir)
	count, err := GetAndIncrementBallCount(workingDir)
	if err != nil {
		return "", fmt.Errorf("failed to get ball count: %w", err)
	}
	return fmt.Sprintf("%s-%d", base, count), nil
}

// GetCwd returns the current working directory
func GetCwd() (string, error) {
	return os.Getwd()
}

// UpdateActivity updates the last activity timestamp
func (b *Ball) UpdateActivity() {
	b.LastActivity = time.Now()
}

// IncrementUpdateCount increments the update counter
func (b *Ball) IncrementUpdateCount() {
	b.UpdateCount++
	b.UpdateActivity()
}

// SetState sets the ball state
func (b *Ball) SetState(state BallState) {
	b.State = state
	if state != StateBlocked {
		b.BlockedReason = ""
	}
	b.UpdateActivity()
}

// SetBlocked sets the ball to blocked state with a reason
func (b *Ball) SetBlocked(reason string) {
	b.State = StateBlocked
	b.BlockedReason = reason
	b.UpdateActivity()
}

// MarkComplete marks the ball as complete
func (b *Ball) MarkComplete(note string) {
	b.State = StateComplete
	b.BlockedReason = ""
	b.CompletionNote = note
	now := time.Now()
	b.CompletedAt = &now
	b.UpdateActivity()
}

// Start transitions a pending ball to in_progress
func (b *Ball) Start() {
	if b.State == StatePending {
		b.State = StateInProgress
		b.StartedAt = time.Now()
		b.UpdateActivity()
	}
}

// SetAcceptanceCriteria sets the complete list of acceptance criteria
func (b *Ball) SetAcceptanceCriteria(criteria []string) {
	b.AcceptanceCriteria = criteria
	b.UpdateActivity()
}

// AddAcceptanceCriterion adds a single acceptance criterion to the list
func (b *Ball) AddAcceptanceCriterion(criterion string) {
	b.AcceptanceCriteria = append(b.AcceptanceCriteria, criterion)
	b.UpdateActivity()
}

// RemoveAcceptanceCriterion removes an acceptance criterion by index (0-based)
func (b *Ball) RemoveAcceptanceCriterion(index int) error {
	if index < 0 || index >= len(b.AcceptanceCriteria) {
		return fmt.Errorf("invalid acceptance criterion index: %d (have %d criteria)", index, len(b.AcceptanceCriteria))
	}
	b.AcceptanceCriteria = append(b.AcceptanceCriteria[:index], b.AcceptanceCriteria[index+1:]...)
	b.UpdateActivity()
	return nil
}

// AddTag adds a tag to the ball
func (b *Ball) AddTag(tag string) {
	for _, t := range b.Tags {
		if t == tag {
			return // Tag already exists
		}
	}
	b.Tags = append(b.Tags, tag)
}

// RemoveTag removes a tag from the ball
func (b *Ball) RemoveTag(tag string) bool {
	for i, t := range b.Tags {
		if t == tag {
			b.Tags = append(b.Tags[:i], b.Tags[i+1:]...)
			b.UpdateActivity()
			return true
		}
	}
	return false // Tag not found
}

// IdleDuration returns how long since the last activity
func (b *Ball) IdleDuration() time.Duration {
	return time.Since(b.LastActivity)
}

// IsInCurrentDir checks if the ball is in the current working directory
func (b *Ball) IsInCurrentDir() bool {
	cwd, err := os.Getwd()
	if err != nil {
		return false
	}
	return b.WorkingDir == cwd
}

// FolderName returns the base name of the working directory
func (b *Ball) FolderName() string {
	return filepath.Base(b.WorkingDir)
}


// ShortID extracts the numeric portion from a ball ID
// e.g., "myapp-5" -> "5", "myapp-143022" -> "143022"
func (b *Ball) ShortID() string {
	// Find the last hyphen and return everything after it
	lastHyphen := -1
	for i := len(b.ID) - 1; i >= 0; i-- {
		if b.ID[i] == '-' {
			lastHyphen = i
			break
		}
	}
	if lastHyphen >= 0 && lastHyphen < len(b.ID)-1 {
		return b.ID[lastHyphen+1:]
	}
	return b.ID
}

// ValidatePriority checks if a priority string is valid
func ValidatePriority(p string) bool {
	switch Priority(p) {
	case PriorityLow, PriorityMedium, PriorityHigh, PriorityUrgent:
		return true
	default:
		return false
	}
}


// ValidateBallState checks if a ball state string is valid
func ValidateBallState(s string) bool {
	switch BallState(s) {
	case StatePending, StateInProgress, StateComplete, StateBlocked:
		return true
	default:
		return false
	}
}

// PriorityWeight returns a numeric weight for sorting
func (b *Ball) PriorityWeight() int {
	switch b.Priority {
	case PriorityUrgent:
		return 4
	case PriorityHigh:
		return 3
	case PriorityMedium:
		return 2
	case PriorityLow:
		return 1
	default:
		return 0
	}
}

// ValidateTestsState checks if a tests state string is valid
func ValidateTestsState(s string) bool {
	switch TestsState(s) {
	case TestsStateUnset, TestsStateNotNeeded, TestsStateNeeded, TestsStateDone:
		return true
	default:
		return false
	}
}

// SetTestsState sets the tests state for the ball
func (b *Ball) SetTestsState(state TestsState) {
	b.TestsState = state
	b.UpdateActivity()
}

// TestsStateLabel returns a human-readable label for the tests state
func (b *Ball) TestsStateLabel() string {
	switch b.TestsState {
	case TestsStateNotNeeded:
		return "not needed"
	case TestsStateNeeded:
		return "needed"
	case TestsStateDone:
		return "done"
	default:
		return ""
	}
}
