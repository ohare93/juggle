package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Priority levels for sessions
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


// Session represents a work session (ball) being tracked
type Session struct {
	ID                 string      `json:"id"`
	WorkingDir         string      `json:"-"` // Computed from file location, not stored
	Intent             string      `json:"intent"`
	AcceptanceCriteria []string    `json:"acceptance_criteria,omitempty"` // List of acceptance criteria
	Priority           Priority    `json:"priority"`
	State              BallState   `json:"state"`                    // New simplified state
	BlockedReason      string      `json:"blocked_reason,omitempty"` // Reason when state is blocked
	StartedAt          time.Time   `json:"started_at"`
	LastActivity       time.Time   `json:"last_activity"`
	CompletedAt        *time.Time  `json:"completed_at,omitempty"`
	UpdateCount        int         `json:"update_count"`
	Tags               []string    `json:"tags,omitempty"`
	CompletionNote     string      `json:"completion_note,omitempty"`
	ModelSize          ModelSize   `json:"model_size,omitempty"` // Preferred LLM model size for cost optimization

	// Legacy field for backward compatibility - use AcceptanceCriteria instead
	Description  string       `json:"-"` // DEPRECATED: Use AcceptanceCriteria instead
}

// UnmarshalJSON implements custom unmarshaling to handle migration from old format
func (s *Session) UnmarshalJSON(data []byte) error {
	var sj sessionJSON
	if err := json.Unmarshal(data, &sj); err != nil {
		return err
	}

	// Copy all standard fields
	s.ID = sj.ID
	s.Intent = sj.Intent
	s.Priority = sj.Priority
	s.StartedAt = sj.StartedAt
	s.LastActivity = sj.LastActivity
	s.UpdateCount = sj.UpdateCount
	s.Tags = sj.Tags
	s.CompletionNote = sj.CompletionNote
	s.ModelSize = sj.ModelSize

	// Handle acceptance criteria with migration from description
	if len(sj.AcceptanceCriteria) > 0 {
		// New format - use acceptance criteria directly
		s.AcceptanceCriteria = sj.AcceptanceCriteria
	} else if sj.Description != "" {
		// Migrate legacy description to first acceptance criterion
		s.AcceptanceCriteria = []string{sj.Description}
	}
	// Populate legacy Description field for backward compatibility
	if len(s.AcceptanceCriteria) > 0 {
		s.Description = s.AcceptanceCriteria[0]
	}

	// Migrate state from various formats to new BallState
	if sj.State != "" {
		// Newest format - use State directly
		s.State = BallState(sj.State)
		s.BlockedReason = sj.BlockedReason
	} else if sj.ActiveState != "" {
		// Previous format with active_state/juggle_state - migrate
		switch sj.ActiveState {
		case "ready":
			s.State = StatePending
		case "juggling":
			s.State = StateInProgress
		case "dropped":
			s.State = StateBlocked
			if sj.StateMessage != "" {
				s.BlockedReason = sj.StateMessage
			} else {
				s.BlockedReason = "dropped"
			}
		case "complete":
			s.State = StateComplete
		default:
			s.State = StatePending
		}
		// JuggleState substates are collapsed into in_progress
		// StateMessage becomes BlockedReason only for blocked state
		if s.State != StateBlocked && sj.StateMessage != "" {
			// For non-blocked states, preserve message in BlockedReason temporarily
			// This will be empty on next save unless state is blocked
		}
	} else if sj.Status != "" {
		// Oldest format with status field - migrate
		switch sj.Status {
		case "planned":
			s.State = StatePending
		case "active":
			s.State = StateInProgress
		case "blocked":
			s.State = StateBlocked
			s.BlockedReason = sj.Blocker
		case "needs-review":
			s.State = StateInProgress
		case "done":
			s.State = StateComplete
		default:
			s.State = StatePending
		}
	} else {
		// No state info, default to pending
		s.State = StatePending
	}

	return nil
}

// sessionJSON is used for custom JSON unmarshaling to handle migration from old format
type sessionJSON struct {
	ID                 string          `json:"id"`
	Intent             string          `json:"intent"`
	AcceptanceCriteria []string        `json:"acceptance_criteria,omitempty"` // New: list of acceptance criteria
	Description        string          `json:"description,omitempty"`         // Legacy: single description
	Priority           Priority        `json:"priority"`
	// Newest format (v3)
	State              string          `json:"state,omitempty"`            // New: pending/in_progress/complete/blocked
	BlockedReason      string          `json:"blocked_reason,omitempty"`   // Reason when state is blocked
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

// New creates a new session with the given parameters in pending state
func New(workingDir, intent string, priority Priority) (*Session, error) {
	now := time.Now()
	id, err := generateID(workingDir)
	if err != nil {
		return nil, err
	}

	sess := &Session{
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
	return sess, nil
}

// generateID creates a unique session ID from working dir and timestamp
// generateID creates a unique session ID from working dir and counter
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
func (s *Session) UpdateActivity() {
	s.LastActivity = time.Now()
}

// IncrementUpdateCount increments the update counter
func (s *Session) IncrementUpdateCount() {
	s.UpdateCount++
	s.UpdateActivity()
}

// SetState sets the ball state
func (s *Session) SetState(state BallState) {
	s.State = state
	if state != StateBlocked {
		s.BlockedReason = ""
	}
	s.UpdateActivity()
}

// SetBlocked sets the ball to blocked state with a reason
func (s *Session) SetBlocked(reason string) {
	s.State = StateBlocked
	s.BlockedReason = reason
	s.UpdateActivity()
}

// MarkComplete marks the session as complete
func (s *Session) MarkComplete(note string) {
	s.State = StateComplete
	s.BlockedReason = ""
	s.CompletionNote = note
	now := time.Now()
	s.CompletedAt = &now
	s.UpdateActivity()
}

// Start transitions a pending session to in_progress
func (s *Session) Start() {
	if s.State == StatePending {
		s.State = StateInProgress
		s.StartedAt = time.Now()
		s.UpdateActivity()
	}
}

// SetDescription sets the session's description
// DEPRECATED: Use SetAcceptanceCriteria or AddAcceptanceCriterion instead
func (s *Session) SetDescription(description string) {
	s.Description = description
	// Also set as first acceptance criterion for forward compatibility
	if len(s.AcceptanceCriteria) == 0 {
		s.AcceptanceCriteria = []string{description}
	} else {
		s.AcceptanceCriteria[0] = description
	}
	s.UpdateActivity()
}

// SetAcceptanceCriteria sets the complete list of acceptance criteria
func (s *Session) SetAcceptanceCriteria(criteria []string) {
	s.AcceptanceCriteria = criteria
	// Populate legacy Description field for backward compatibility
	if len(criteria) > 0 {
		s.Description = criteria[0]
	} else {
		s.Description = ""
	}
	s.UpdateActivity()
}

// AddAcceptanceCriterion adds a single acceptance criterion to the list
func (s *Session) AddAcceptanceCriterion(criterion string) {
	s.AcceptanceCriteria = append(s.AcceptanceCriteria, criterion)
	// Update legacy Description if this is the first criterion
	if len(s.AcceptanceCriteria) == 1 {
		s.Description = criterion
	}
	s.UpdateActivity()
}

// RemoveAcceptanceCriterion removes an acceptance criterion by index (0-based)
func (s *Session) RemoveAcceptanceCriterion(index int) error {
	if index < 0 || index >= len(s.AcceptanceCriteria) {
		return fmt.Errorf("invalid acceptance criterion index: %d (have %d criteria)", index, len(s.AcceptanceCriteria))
	}
	s.AcceptanceCriteria = append(s.AcceptanceCriteria[:index], s.AcceptanceCriteria[index+1:]...)
	// Update legacy Description field
	if len(s.AcceptanceCriteria) > 0 {
		s.Description = s.AcceptanceCriteria[0]
	} else {
		s.Description = ""
	}
	s.UpdateActivity()
	return nil
}

// AddTag adds a tag to the session
func (s *Session) AddTag(tag string) {
	for _, t := range s.Tags {
		if t == tag {
			return // Tag already exists
		}
	}
	s.Tags = append(s.Tags, tag)
}

// RemoveTag removes a tag from the session
func (s *Session) RemoveTag(tag string) bool {
	for i, t := range s.Tags {
		if t == tag {
			s.Tags = append(s.Tags[:i], s.Tags[i+1:]...)
			s.UpdateActivity()
			return true
		}
	}
	return false // Tag not found
}

// IdleDuration returns how long since the last activity
func (s *Session) IdleDuration() time.Duration {
	return time.Since(s.LastActivity)
}

// IsInCurrentDir checks if the session is in the current working directory
func (s *Session) IsInCurrentDir() bool {
	cwd, err := os.Getwd()
	if err != nil {
		return false
	}
	return s.WorkingDir == cwd
}

// FolderName returns the base name of the working directory
func (s *Session) FolderName() string {
	return filepath.Base(s.WorkingDir)
}


// ShortID extracts the numeric portion from a ball ID
// e.g., "myapp-5" -> "5", "myapp-143022" -> "143022"
func (s *Session) ShortID() string {
	// Find the last hyphen and return everything after it
	lastHyphen := -1
	for i := len(s.ID) - 1; i >= 0; i-- {
		if s.ID[i] == '-' {
			lastHyphen = i
			break
		}
	}
	if lastHyphen >= 0 && lastHyphen < len(s.ID)-1 {
		return s.ID[lastHyphen+1:]
	}
	return s.ID
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
func (s *Session) PriorityWeight() int {
	switch s.Priority {
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
