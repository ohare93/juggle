package session

import (
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
	StateOnHold     BallState = "on_hold"    // Deferred - not currently being worked on, excluded from agent loops
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
	Context            string      `json:"context,omitempty"` // Detailed description/background for the ball
	Title              string      `json:"title"`             // Short title (50 char soft limit)
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

// NewBall creates a new ball with the given parameters in pending state
func NewBall(workingDir, title string, priority Priority) (*Ball, error) {
	now := time.Now()
	id, err := generateID(workingDir)
	if err != nil {
		return nil, err
	}

	ball := &Ball{
		ID:           id,
		WorkingDir:   workingDir,
		Title:        title,
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

	// Resolve to main repo if this is a worktree, so ball IDs use the
	// main project name rather than the worktree folder name
	resolvedDir, err := ResolveStorageDir(workingDir, projectStorePath)
	if err != nil {
		resolvedDir = workingDir
	}

	base := filepath.Base(resolvedDir)
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

// ComputeMinimalUniqueIDs computes the shortest unique ID prefix for each ball in the slice.
// Returns a map from full ball ID to the minimal display ID needed to uniquely identify it.
// For example, if balls have short IDs "01234abc" and "56789def", the map would contain
// "project-01234abc" -> "0" and "project-56789def" -> "5" (single char is enough).
// If two balls have short IDs "1111222244" and "1122334455", the map would contain
// minimal prefixes like "111" and "112" to disambiguate.
func ComputeMinimalUniqueIDs(balls []*Ball) map[string]string {
	result := make(map[string]string)
	if len(balls) == 0 {
		return result
	}

	// Extract short IDs for each ball
	shortIDs := make([]string, len(balls))
	for i, ball := range balls {
		shortIDs[i] = ball.ShortID()
	}

	// For each ball, find the minimal prefix that uniquely identifies it
	for i, ball := range balls {
		myShortID := shortIDs[i]
		minLen := 1

		// Compare against all other balls' short IDs
		for j, otherShortID := range shortIDs {
			if i == j {
				continue
			}

			// Find the minimum length needed to distinguish from this other ID
			commonLen := 0
			maxCheck := len(myShortID)
			if len(otherShortID) < maxCheck {
				maxCheck = len(otherShortID)
			}
			for k := 0; k < maxCheck; k++ {
				if myShortID[k] == otherShortID[k] {
					commonLen++
				} else {
					break
				}
			}

			// Need at least commonLen+1 characters to distinguish
			needed := commonLen + 1
			if needed > len(myShortID) {
				// If other is a prefix of ours, use full length
				needed = len(myShortID)
			}
			if needed > minLen {
				minLen = needed
			}
		}

		// Cap at actual length
		if minLen > len(myShortID) {
			minLen = len(myShortID)
		}

		result[ball.ID] = myShortID[:minLen]
	}

	return result
}

// ResolveBallByPrefix finds balls that match the given prefix.
// It tries to match against the short ID (part after last hyphen) first,
// then falls back to full ID prefix matching.
// Returns all matching balls - callers should handle ambiguity.
func ResolveBallByPrefix(balls []*Ball, prefix string) []*Ball {
	if prefix == "" {
		return nil
	}

	// Convert prefix to lowercase for case-insensitive matching
	prefixLower := lowerString(prefix)

	var matches []*Ball

	// First, try exact short ID match (case-insensitive)
	for _, ball := range balls {
		if lowerString(ball.ShortID()) == prefixLower {
			return []*Ball{ball}
		}
	}

	// Try exact full ID match (case-insensitive)
	for _, ball := range balls {
		if lowerString(ball.ID) == prefixLower {
			return []*Ball{ball}
		}
	}

	// Try prefix matching on short ID
	for _, ball := range balls {
		shortID := ball.ShortID()
		if len(shortID) >= len(prefix) && lowerString(shortID[:len(prefix)]) == prefixLower {
			matches = append(matches, ball)
		}
	}

	if len(matches) > 0 {
		return matches
	}

	// Fall back to full ID prefix matching
	for _, ball := range balls {
		if len(ball.ID) >= len(prefix) && lowerString(ball.ID[:len(prefix)]) == prefixLower {
			matches = append(matches, ball)
		}
	}

	return matches
}

// lowerString returns lowercase version of a string
func lowerString(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			result[i] = c + 32
		} else {
			result[i] = c
		}
	}
	return string(result)
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
	case StatePending, StateInProgress, StateComplete, StateBlocked, StateResearched, StateOnHold:
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

// ValidateModelSize checks if a model size string is valid
func ValidateModelSize(s string) bool {
	switch ModelSize(s) {
	case ModelSizeBlank, ModelSizeSmall, ModelSizeMedium, ModelSizeLarge:
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

// SetModelSize sets the preferred model size for the ball
func (b *Ball) SetModelSize(size ModelSize) {
	b.ModelSize = size
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
