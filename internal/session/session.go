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

// Todo represents a single todo item with completion status
type Todo struct {
	Text        string    `json:"text"`
	Description string    `json:"description,omitempty"`
	Done        bool      `json:"done"`
	CreatedAt   time.Time `json:"created_at"`
}

// BallState represents the lifecycle state of a ball
type BallState string

const (
	StatePending    BallState = "pending"
	StateInProgress BallState = "in_progress"
	StateComplete   BallState = "complete"
	StateBlocked    BallState = "blocked"
)

// Legacy type aliases for backward compatibility during migration
// TODO: Remove these after all code is updated
type ActiveState = BallState
type JuggleState = BallState

// Legacy constants mapped to new states
const (
	ActiveReady     = StatePending
	ActiveJuggling  = StateInProgress
	ActiveDropped   = StateBlocked // Dropped maps to blocked
	ActiveComplete  = StateComplete

	// JuggleState constants - all map to in_progress in new model
	JuggleNeedsThrown JuggleState = "needs-thrown" // Legacy - will be migrated
	JuggleInAir       JuggleState = "in-air"       // Legacy - will be migrated
	JuggleNeedsCaught JuggleState = "needs-caught" // Legacy - will be migrated
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
	Todos              []Todo      `json:"todos,omitempty"`
	Tags               []string    `json:"tags,omitempty"`
	CompletionNote     string      `json:"completion_note,omitempty"`
	ModelSize          ModelSize   `json:"model_size,omitempty"` // Preferred LLM model size for cost optimization

	// Legacy fields - kept for backward compatibility with existing code
	// TODO: Remove after full migration
	Description  string       `json:"-"` // DEPRECATED: Use AcceptanceCriteria instead
	ActiveState  ActiveState  `json:"-"` // Computed from State for legacy code
	JuggleState  *JuggleState `json:"-"` // Always nil in new model
	StateMessage string       `json:"-"` // Maps to BlockedReason for legacy code
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

	// Set legacy fields for backward compatibility with existing code
	s.syncLegacyFields()

	// Handle todos migration: support both []string (old) and []Todo (new)
	if len(sj.Todos) > 0 {
		// Try parsing as []Todo first
		var newTodos []Todo
		if err := json.Unmarshal(sj.Todos, &newTodos); err == nil {
			s.Todos = newTodos
		} else {
			// Fall back to []string (old format)
			var oldTodos []string
			if err := json.Unmarshal(sj.Todos, &oldTodos); err == nil {
				// Migrate old format to new format
				s.Todos = make([]Todo, len(oldTodos))
				for i, text := range oldTodos {
					s.Todos[i] = Todo{
						Text:      text,
						Done:      false,
						CreatedAt: s.StartedAt, // Use session start as approximate creation time
					}
				}
			}
		}
	}

	return nil
}

// syncLegacyFields populates legacy fields from new State field
// This allows existing code to continue using ActiveState/JuggleState
func (s *Session) syncLegacyFields() {
	switch s.State {
	case StatePending:
		s.ActiveState = ActiveReady
	case StateInProgress:
		s.ActiveState = ActiveJuggling
		inAir := JuggleInAir
		s.JuggleState = &inAir
	case StateBlocked:
		s.ActiveState = ActiveDropped
		s.StateMessage = s.BlockedReason
	case StateComplete:
		s.ActiveState = ActiveComplete
	default:
		s.ActiveState = ActiveReady
	}
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
	Todos              json.RawMessage `json:"todos,omitempty"`
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
		Todos:        []Todo{},
		Tags:         []string{},
	}
	sess.syncLegacyFields()
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
	s.syncLegacyFields()
	s.UpdateActivity()
}

// SetBlocked sets the ball to blocked state with a reason
func (s *Session) SetBlocked(reason string) {
	s.State = StateBlocked
	s.BlockedReason = reason
	s.syncLegacyFields()
	s.UpdateActivity()
}

// SetJuggleState sets the juggle state and optional message
// DEPRECATED: Use SetState instead. Kept for backward compatibility.
func (s *Session) SetJuggleState(state JuggleState, message string) {
	// In the new model, all juggle states map to in_progress
	s.State = StateInProgress
	s.syncLegacyFields()
	s.UpdateActivity()
}

// SetActiveState sets the active state and clears juggle state if not juggling
// DEPRECATED: Use SetState instead. Kept for backward compatibility.
func (s *Session) SetActiveState(state ActiveState) {
	// Map legacy ActiveState to new BallState
	switch state {
	case ActiveReady:
		s.State = StatePending
	case ActiveJuggling:
		s.State = StateInProgress
	case ActiveDropped:
		s.State = StateBlocked
	case ActiveComplete:
		s.State = StateComplete
	}
	s.syncLegacyFields()
	s.UpdateActivity()
}

// MarkComplete marks the session as complete
func (s *Session) MarkComplete(note string) {
	s.State = StateComplete
	s.BlockedReason = ""
	s.CompletionNote = note
	now := time.Now()
	s.CompletedAt = &now
	s.syncLegacyFields()
	s.UpdateActivity()
}

// StartJuggling transitions a pending session to in_progress
// DEPRECATED: Use SetState(StateInProgress) instead.
func (s *Session) StartJuggling() {
	if s.State == StatePending {
		s.State = StateInProgress
		s.StartedAt = time.Now()
		s.syncLegacyFields()
		s.UpdateActivity()
	}
}

// Start transitions a pending session to in_progress
func (s *Session) Start() {
	if s.State == StatePending {
		s.State = StateInProgress
		s.StartedAt = time.Now()
		s.syncLegacyFields()
		s.UpdateActivity()
	}
}

// AddTodo adds a todo item to the session
func (s *Session) AddTodo(text string) {
	s.Todos = append(s.Todos, Todo{
		Text:      text,
		Done:      false,
		CreatedAt: time.Now(),
	})
	s.UpdateActivity()
}

// AddTodos adds multiple todo items at once (useful for batch operations)
func (s *Session) AddTodos(texts []string) {
	for _, text := range texts {
		s.Todos = append(s.Todos, Todo{
			Text:      text,
			Done:      false,
			CreatedAt: time.Now(),
		})
	}
	s.UpdateActivity()
}

// AddTodoWithDescription adds a todo item with a description to the session
func (s *Session) AddTodoWithDescription(text, description string) {
	s.Todos = append(s.Todos, Todo{
		Text:        text,
		Description: description,
		Done:        false,
		CreatedAt:   time.Now(),
	})
	s.UpdateActivity()
}

// SetTodoDescription sets or updates the description of a todo by index (0-based)
func (s *Session) SetTodoDescription(index int, description string) error {
	if index < 0 || index >= len(s.Todos) {
		return fmt.Errorf("invalid todo index: %d (have %d todos)", index, len(s.Todos))
	}
	s.Todos[index].Description = description
	s.UpdateActivity()
	return nil
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

// ToggleTodo marks a todo as done or undone by index (0-based)
func (s *Session) ToggleTodo(index int) error {
	if index < 0 || index >= len(s.Todos) {
		return fmt.Errorf("invalid todo index: %d (have %d todos)", index, len(s.Todos))
	}
	s.Todos[index].Done = !s.Todos[index].Done
	s.UpdateActivity()
	return nil
}

// RemoveTodo removes a todo by index (0-based)
func (s *Session) RemoveTodo(index int) error {
	if index < 0 || index >= len(s.Todos) {
		return fmt.Errorf("invalid todo index: %d (have %d todos)", index, len(s.Todos))
	}
	s.Todos = append(s.Todos[:index], s.Todos[index+1:]...)
	s.UpdateActivity()
	return nil
}

// EditTodo updates the text of a todo by index (0-based)
func (s *Session) EditTodo(index int, newText string) error {
	if index < 0 || index >= len(s.Todos) {
		return fmt.Errorf("invalid todo index: %d (have %d todos)", index, len(s.Todos))
	}
	s.Todos[index].Text = newText
	s.UpdateActivity()
	return nil
}

// ClearTodos removes all todos
func (s *Session) ClearTodos() {
	s.Todos = []Todo{}
	s.UpdateActivity()
}

// TodoStats returns counts of total and completed todos
func (s *Session) TodoStats() (total, completed int) {
	total = len(s.Todos)
	for _, todo := range s.Todos {
		if todo.Done {
			completed++
		}
	}
	return
}

// TodoCompletionSummary returns a string like "3/5" showing completed/total todos
func (s *Session) TodoCompletionSummary() string {
	total, completed := s.TodoStats()
	return fmt.Sprintf("%d/%d", completed, total)
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

// ValidateActiveState checks if an active state string is valid
// DEPRECATED: Use ValidateBallState instead.
func ValidateActiveState(s string) bool {
	// Accept both old and new state values for backward compatibility
	switch s {
	case "ready", "juggling", "dropped", "complete", // old values
		"pending", "in_progress", "blocked": // new values
		return true
	default:
		return false
	}
}

// ValidateJuggleState checks if a juggle state string is valid
// DEPRECATED: JuggleState is no longer used.
func ValidateJuggleState(s string) bool {
	switch JuggleState(s) {
	case JuggleNeedsThrown, JuggleInAir, JuggleNeedsCaught:
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
