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

// Todo represents a single todo item with completion status
type Todo struct {
	Text        string    `json:"text"`
	Description string    `json:"description,omitempty"`
	Done        bool      `json:"done"`
	CreatedAt   time.Time `json:"created_at"`
}

// ActiveState represents the lifecycle state of a ball
type ActiveState string

const (
	ActiveReady     ActiveState = "ready"
	ActiveJuggling  ActiveState = "juggling"
	ActiveDropped   ActiveState = "dropped"
	ActiveComplete  ActiveState = "complete"
)

// JuggleState represents the state within the juggling cycle
// Only valid when ActiveState is ActiveJuggling
type JuggleState string

const (
	JuggleNeedsThrown JuggleState = "needs-thrown"
	JuggleInAir       JuggleState = "in-air"
	JuggleNeedsCaught JuggleState = "needs-caught"
)

// Session represents a work session (ball) being tracked
type Session struct {
	ID             string        `json:"id"`
	WorkingDir     string        `json:"-"` // Computed from file location, not stored
	Intent         string        `json:"intent"`
	Description    string        `json:"description,omitempty"`
	Priority       Priority      `json:"priority"`
	ActiveState    ActiveState   `json:"active_state"`
	JuggleState    *JuggleState  `json:"juggle_state,omitempty"` // Only when ActiveState is juggling
	StateMessage   string        `json:"state_message,omitempty"` // Optional context about current state
	StartedAt      time.Time     `json:"started_at"`
	LastActivity   time.Time     `json:"last_activity"`
	CompletedAt    *time.Time    `json:"completed_at,omitempty"`
	UpdateCount    int           `json:"update_count"`
	Todos          []Todo        `json:"todos,omitempty"`
	Tags           []string      `json:"tags,omitempty"`
	CompletionNote string        `json:"completion_note,omitempty"`
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
	s.Description = sj.Description
	s.Priority = sj.Priority
	s.StartedAt = sj.StartedAt
	s.LastActivity = sj.LastActivity
	s.UpdateCount = sj.UpdateCount
	s.Tags = sj.Tags
	s.CompletionNote = sj.CompletionNote

	// Migrate state from old format to new format
	if sj.ActiveState != "" {
		// New format - use directly
		s.ActiveState = ActiveState(sj.ActiveState)
		if sj.JuggleState != nil {
			js := JuggleState(*sj.JuggleState)
			s.JuggleState = &js
		}
		s.StateMessage = sj.StateMessage
	} else if sj.Status != "" {
		// Old format - migrate
		switch sj.Status {
		case "planned":
			s.ActiveState = ActiveReady
		case "active":
			s.ActiveState = ActiveJuggling
			inAir := JuggleInAir
			s.JuggleState = &inAir
		case "blocked":
			s.ActiveState = ActiveJuggling
			needsThrown := JuggleNeedsThrown
			s.JuggleState = &needsThrown
			if sj.Blocker != "" {
				s.StateMessage = sj.Blocker
			}
		case "needs-review":
			s.ActiveState = ActiveJuggling
			needsCaught := JuggleNeedsCaught
			s.JuggleState = &needsCaught
		case "done":
			s.ActiveState = ActiveComplete
		default:
			// Unknown status, default to ready
			s.ActiveState = ActiveReady
		}
	} else {
		// No state info, default to ready
		s.ActiveState = ActiveReady
	}

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

// sessionJSON is used for custom JSON unmarshaling to handle migration from old format
type sessionJSON struct {
	ID             string          `json:"id"`
	Intent         string          `json:"intent"`
	Description    string          `json:"description,omitempty"`
	Priority       Priority        `json:"priority"`
	// Old format fields
	Status         string          `json:"status,omitempty"`           // Old: planned/active/blocked/needs-review/done
	Blocker        string          `json:"blocker,omitempty"`          // Old blocker field
	// New format fields
	ActiveState    string          `json:"active_state,omitempty"`     // New: ready/juggling/dropped/complete
	JuggleState    *string         `json:"juggle_state,omitempty"`     // New: needs-thrown/in-air/needs-caught
	StateMessage   string          `json:"state_message,omitempty"`
	StartedAt      time.Time       `json:"started_at"`
	LastActivity   time.Time       `json:"last_activity"`
	UpdateCount    int             `json:"update_count"`
	Todos          json.RawMessage `json:"todos,omitempty"`
	Tags           []string        `json:"tags,omitempty"`
	CompletionNote string          `json:"completion_note,omitempty"`
}

// New creates a new session with the given parameters in ready state
func New(workingDir, intent string, priority Priority) (*Session, error) {
	now := time.Now()
	id, err := generateID(workingDir)
	if err != nil {
		return nil, err
	}

	return &Session{
		ID:           id,
		WorkingDir:   workingDir,
		Intent:       intent,
		Priority:     priority,
		ActiveState:  ActiveReady,
		StartedAt:    now,
		LastActivity: now,
		UpdateCount:  0,
		Todos:        []Todo{},
		Tags:         []string{},
	}, nil
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

// SetJuggleState sets the juggle state and optional message
// Can only be called when ActiveState is juggling
func (s *Session) SetJuggleState(state JuggleState, message string) {
	if s.ActiveState == ActiveJuggling {
		s.JuggleState = &state
		s.StateMessage = message
		s.UpdateActivity()
	}
}

// SetActiveState sets the active state and clears juggle state if not juggling
func (s *Session) SetActiveState(state ActiveState) {
	s.ActiveState = state
	if state != ActiveJuggling {
		s.JuggleState = nil
		s.StateMessage = ""
	}
	s.UpdateActivity()
}

// MarkComplete marks the session as complete
func (s *Session) MarkComplete(note string) {
	s.ActiveState = ActiveComplete
	s.JuggleState = nil
	s.StateMessage = ""
	s.CompletionNote = note
	now := time.Now()
	s.CompletedAt = &now
	s.UpdateActivity()
}

// StartJuggling transitions a ready session to juggling:needs-thrown
func (s *Session) StartJuggling() {
	if s.ActiveState == ActiveReady {
		s.ActiveState = ActiveJuggling
		needsThrown := JuggleNeedsThrown
		s.JuggleState = &needsThrown
		s.StartedAt = time.Now()
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
func (s *Session) SetDescription(description string) {
	s.Description = description
	s.UpdateActivity()
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


// ValidateActiveState checks if an active state string is valid
func ValidateActiveState(s string) bool {
	switch ActiveState(s) {
	case ActiveReady, ActiveJuggling, ActiveDropped, ActiveComplete:
		return true
	default:
		return false
	}
}

// ValidateJuggleState checks if a juggle state string is valid
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
