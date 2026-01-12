package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
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
	StateResearched BallState = "researched" // Completed with no code changes, output contains results
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
	Output             string      `json:"output,omitempty"` // Research results or investigation output
	DependsOn          []string    `json:"depends_on,omitempty"` // Ball IDs this ball depends on
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
	b.CompletedAt = bj.CompletedAt
	b.UpdateCount = bj.UpdateCount
	b.Tags = bj.Tags
	b.CompletionNote = bj.CompletionNote
	b.ModelSize = bj.ModelSize
	b.TestsState = bj.TestsState
	b.Output = bj.Output
	b.DependsOn = bj.DependsOn

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
	State              string          `json:"state,omitempty"`            // New: pending/in_progress/complete/blocked/researched
	BlockedReason      string          `json:"blocked_reason,omitempty"`   // Reason when state is blocked
	TestsState         TestsState      `json:"tests_state,omitempty"`      // Whether tests are needed/done
	Output             string          `json:"output,omitempty"`           // Research results or investigation output
	DependsOn          []string        `json:"depends_on,omitempty"`       // Ball IDs this ball depends on
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
	CompletedAt        *time.Time      `json:"completed_at,omitempty"`
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

// generateID creates a unique ball ID using UUID
func generateID(workingDir string) (string, error) {
	// Generate a short UUID-based ID with project prefix for readability
	// Format: <project>-<short-uuid> where short-uuid is first 8 chars of UUID
	base := filepath.Base(workingDir)
	id := uuid.New().String()
	shortID := id[:8] // First 8 characters of UUID (e.g., "a1b2c3d4")
	return fmt.Sprintf("%s-%s", base, shortID), nil
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

// MarkResearched marks the ball as researched (completed with no code changes)
func (b *Ball) MarkResearched(output string) {
	b.State = StateResearched
	b.BlockedReason = ""
	b.Output = output
	now := time.Now()
	b.CompletedAt = &now
	b.UpdateActivity()
}

// SetOutput sets the output/research results for the ball
func (b *Ball) SetOutput(output string) {
	b.Output = output
	b.UpdateActivity()
}

// HasOutput returns true if the ball has output/research results
func (b *Ball) HasOutput() bool {
	return b.Output != ""
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


// ShortID extracts the unique portion from a ball ID
// e.g., "myapp-5" -> "5" (legacy numeric), "myapp-a1b2c3d4" -> "a1b2c3d4" (UUID-based)
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
	case StatePending, StateInProgress, StateComplete, StateBlocked, StateResearched:
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

// HasDependencies returns true if the ball has dependencies
func (b *Ball) HasDependencies() bool {
	return len(b.DependsOn) > 0
}

// AddDependency adds a dependency to the ball
func (b *Ball) AddDependency(ballID string) {
	for _, dep := range b.DependsOn {
		if dep == ballID {
			return // Already exists
		}
	}
	b.DependsOn = append(b.DependsOn, ballID)
	b.UpdateActivity()
}

// RemoveDependency removes a dependency from the ball
func (b *Ball) RemoveDependency(ballID string) bool {
	for i, dep := range b.DependsOn {
		if dep == ballID {
			b.DependsOn = append(b.DependsOn[:i], b.DependsOn[i+1:]...)
			b.UpdateActivity()
			return true
		}
	}
	return false
}

// SetDependencies sets the complete list of dependencies
func (b *Ball) SetDependencies(deps []string) {
	b.DependsOn = deps
	b.UpdateActivity()
}

// DetectCircularDependencies checks for circular dependencies in a set of balls.
// Returns an error describing the cycle if one is found, nil otherwise.
func DetectCircularDependencies(balls []*Ball) error {
	// Build a map for quick lookup
	ballMap := make(map[string]*Ball)
	for _, ball := range balls {
		ballMap[ball.ID] = ball
		// Also map by short ID if unique
		shortID := ball.ShortID()
		if _, exists := ballMap[shortID]; !exists {
			ballMap[shortID] = ball
		}
	}

	// Track visited and currently processing balls for cycle detection
	visited := make(map[string]bool)
	inStack := make(map[string]bool)

	var detectCycle func(ballID string, path []string) error
	detectCycle = func(ballID string, path []string) error {
		// Resolve the ball
		ball, exists := ballMap[ballID]
		if !exists {
			// Dependency not found - not a cycle error
			return nil
		}

		actualID := ball.ID
		if visited[actualID] {
			return nil
		}
		if inStack[actualID] {
			// Found a cycle
			cyclePath := append(path, actualID)
			return fmt.Errorf("circular dependency detected: %s", formatCyclePath(cyclePath))
		}

		inStack[actualID] = true
		path = append(path, actualID)

		for _, depID := range ball.DependsOn {
			if err := detectCycle(depID, path); err != nil {
				return err
			}
		}

		inStack[actualID] = false
		visited[actualID] = true
		return nil
	}

	for _, ball := range balls {
		if !visited[ball.ID] {
			if err := detectCycle(ball.ID, nil); err != nil {
				return err
			}
		}
	}

	return nil
}

// formatCyclePath formats a cycle path for display
func formatCyclePath(path []string) string {
	if len(path) == 0 {
		return ""
	}
	result := path[0]
	for i := 1; i < len(path); i++ {
		result += " → " + path[i]
	}
	return result
}
