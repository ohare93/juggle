package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"
)

const (
	sessionsDir     = "sessions"
	sessionFile     = "session.json"
	progressFile    = "progress.txt"
)

// JuggleSession represents a grouping of balls by tag.
//
// Sessions provide organization and context for related work:
//   - Balls are linked to sessions via tags matching the session ID
//   - Session context persists across agent iterations
//   - Session-level acceptance criteria apply to all linked balls
//   - Default model size can be set for all balls in the session
//
// Sessions are stored in .juggle/sessions/<id>/session.json with
// accompanying progress.txt for logging agent activity.
//
// Example:
//
//	session := session.NewJuggleSession("auth-feature", "OAuth2 implementation")
//	session.AddAcceptanceCriterion("All tests pass")
type JuggleSession struct {
	ID                 string    `json:"id"`                         // Session ID (same as tag)
	Description        string    `json:"description"`                // Human-readable description
	Context            string    `json:"context"`                    // Rich context for agent memory
	DefaultModel       ModelSize `json:"default_model,omitempty"`    // Default model size for balls in this session
	AcceptanceCriteria []string  `json:"acceptance_criteria,omitempty"` // Session-level ACs applied to all balls
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// NewJuggleSession creates a new session with the given ID and description
func NewJuggleSession(id, description string) *JuggleSession {
	now := time.Now()
	return &JuggleSession{
		ID:          id,
		Description: description,
		Context:     "",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// SetContext updates the session context
func (s *JuggleSession) SetContext(context string) {
	s.Context = context
	s.UpdatedAt = time.Now()
}

// SetDescription updates the session description
func (s *JuggleSession) SetDescription(description string) {
	s.Description = description
	s.UpdatedAt = time.Now()
}

// SetDefaultModel updates the session's default model size
func (s *JuggleSession) SetDefaultModel(model ModelSize) {
	s.DefaultModel = model
	s.UpdatedAt = time.Now()
}

// SetAcceptanceCriteria sets the session-level acceptance criteria
func (s *JuggleSession) SetAcceptanceCriteria(criteria []string) {
	s.AcceptanceCriteria = criteria
	s.UpdatedAt = time.Now()
}

// AddAcceptanceCriterion adds a single acceptance criterion to the session
func (s *JuggleSession) AddAcceptanceCriterion(criterion string) {
	s.AcceptanceCriteria = append(s.AcceptanceCriteria, criterion)
	s.UpdatedAt = time.Now()
}

// HasAcceptanceCriteria returns true if the session has any acceptance criteria
func (s *JuggleSession) HasAcceptanceCriteria() bool {
	return len(s.AcceptanceCriteria) > 0
}

// SessionStore handles persistence of JuggleSessions.
//
// SessionStore manages session data in .juggle/sessions/<id>/ directories:
//   - session.json: Session metadata and configuration
//   - progress.txt: Append-only log of agent activity
//
// Thread-safe for concurrent access via file locking.
// Worktree-aware: resolves to main repo storage when in a git worktree.
type SessionStore struct {
	projectDir string
	config     StoreConfig
}

// NewSessionStore creates a new session store for the given project directory
func NewSessionStore(projectDir string) (*SessionStore, error) {
	return NewSessionStoreWithConfig(projectDir, DefaultStoreConfig())
}

// NewSessionStoreWithConfig creates a new session store with custom configuration.
// If running in a worktree (has .juggle/link file), uses the linked main repo for storage.
func NewSessionStoreWithConfig(projectDir string, config StoreConfig) (*SessionStore, error) {
	if projectDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current directory: %w", err)
		}
		projectDir = cwd
	}

	// Resolve to main repo if this is a worktree
	storageDir, err := ResolveStorageDir(projectDir, config.JuggleDirName)
	if err != nil {
		// If resolution fails, fall back to projectDir
		storageDir = projectDir
	}

	return &SessionStore{
		projectDir: storageDir,
		config:     config,
	}, nil
}

// sessionPath returns the path to a session's directory
func (s *SessionStore) sessionPath(id string) string {
	return filepath.Join(s.projectDir, s.config.JuggleDirName, sessionsDir, id)
}

// sessionFilePath returns the path to a session's JSON file
func (s *SessionStore) sessionFilePath(id string) string {
	return filepath.Join(s.sessionPath(id), sessionFile)
}

// progressFilePath returns the path to a session's progress file
func (s *SessionStore) progressFilePath(id string) string {
	return filepath.Join(s.sessionPath(id), progressFile)
}

// CreateSession creates a new session with the given ID and description
func (s *SessionStore) CreateSession(id, description string) (*JuggleSession, error) {
	// Check if session already exists
	if _, err := s.LoadSession(id); err == nil {
		return nil, fmt.Errorf("session %s already exists", id)
	}

	// Create session directory
	sessionDir := s.sessionPath(id)
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create session directory: %w", err)
	}

	// Create session
	session := NewJuggleSession(id, description)

	// Write session JSON
	if err := s.saveSession(session); err != nil {
		// Clean up on failure
		os.RemoveAll(sessionDir)
		return nil, err
	}

	// Create empty progress file
	progressPath := s.progressFilePath(id)
	if err := os.WriteFile(progressPath, []byte{}, 0644); err != nil {
		// Clean up on failure
		os.RemoveAll(sessionDir)
		return nil, fmt.Errorf("failed to create progress file: %w", err)
	}

	return session, nil
}

// LoadSession reads a session from disk
func (s *SessionStore) LoadSession(id string) (*JuggleSession, error) {
	filePath := s.sessionFilePath(id)

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("session %s not found", id)
		}
		return nil, fmt.Errorf("failed to read session file: %w", err)
	}

	var session JuggleSession
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to parse session file: %w", err)
	}

	return &session, nil
}

// ListSessions discovers all sessions in the project
func (s *SessionStore) ListSessions() ([]*JuggleSession, error) {
	sessionsPath := filepath.Join(s.projectDir, s.config.JuggleDirName, sessionsDir)

	// If sessions directory doesn't exist, return empty list
	if _, err := os.Stat(sessionsPath); os.IsNotExist(err) {
		return []*JuggleSession{}, nil
	}

	entries, err := os.ReadDir(sessionsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read sessions directory: %w", err)
	}

	sessions := make([]*JuggleSession, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		session, err := s.LoadSession(entry.Name())
		if err != nil {
			// Skip invalid sessions
			continue
		}

		sessions = append(sessions, session)
	}

	return sessions, nil
}

// UpdateSessionContext updates the context field of a session
func (s *SessionStore) UpdateSessionContext(id, context string) error {
	session, err := s.LoadSession(id)
	if err != nil {
		return err
	}

	session.SetContext(context)
	return s.saveSession(session)
}

// UpdateSessionDescription updates the description field of a session
func (s *SessionStore) UpdateSessionDescription(id, description string) error {
	session, err := s.LoadSession(id)
	if err != nil {
		return err
	}

	session.SetDescription(description)
	return s.saveSession(session)
}

// UpdateSessionAcceptanceCriteria updates the acceptance criteria for a session
func (s *SessionStore) UpdateSessionAcceptanceCriteria(id string, criteria []string) error {
	session, err := s.LoadSession(id)
	if err != nil {
		return err
	}

	session.SetAcceptanceCriteria(criteria)
	return s.saveSession(session)
}

// UpdateSessionDefaultModel updates the default model size for a session
func (s *SessionStore) UpdateSessionDefaultModel(id string, model ModelSize) error {
	session, err := s.LoadSession(id)
	if err != nil {
		return err
	}

	session.SetDefaultModel(model)
	return s.saveSession(session)
}

// DeleteSession removes a session and its directory
func (s *SessionStore) DeleteSession(id string) error {
	// Verify session exists
	if _, err := s.LoadSession(id); err != nil {
		return err
	}

	sessionDir := s.sessionPath(id)
	if err := os.RemoveAll(sessionDir); err != nil {
		return fmt.Errorf("failed to delete session directory: %w", err)
	}

	return nil
}

// AppendProgress appends content to a session's progress file
func (s *SessionStore) AppendProgress(id, content string) error {
	// Verify session exists (skip for "_all" virtual session)
	if id != "_all" {
		if _, err := s.LoadSession(id); err != nil {
			return err
		}
	} else {
		// For "_all", ensure the directory exists
		sessionDir := s.sessionPath(id)
		if err := os.MkdirAll(sessionDir, 0755); err != nil {
			return fmt.Errorf("failed to create _all session directory: %w", err)
		}
	}

	progressPath := s.progressFilePath(id)
	lockPath := progressPath + ".lock"

	// Acquire file lock
	fileLock := flock.New(lockPath)
	if err := fileLock.Lock(); err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer fileLock.Unlock()

	// Open file in append mode
	f, err := os.OpenFile(progressPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open progress file: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(content); err != nil {
		return fmt.Errorf("failed to write to progress file: %w", err)
	}

	return nil
}

// LoadProgress reads the contents of a session's progress file
func (s *SessionStore) LoadProgress(id string) (string, error) {
	// Verify session exists (skip for "_all" virtual session)
	if id != "_all" {
		if _, err := s.LoadSession(id); err != nil {
			return "", err
		}
	}

	progressPath := s.progressFilePath(id)

	data, err := os.ReadFile(progressPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil // Empty progress is valid
		}
		return "", fmt.Errorf("failed to read progress file: %w", err)
	}

	return string(data), nil
}

// ClearProgress truncates a session's progress file to empty
func (s *SessionStore) ClearProgress(id string) error {
	// Verify session exists (skip for "_all" virtual session)
	if id != "_all" {
		if _, err := s.LoadSession(id); err != nil {
			return err
		}
	}

	progressPath := s.progressFilePath(id)
	lockPath := progressPath + ".lock"

	// Check if progress file exists (nothing to clear for "_all" if it doesn't exist)
	if _, err := os.Stat(progressPath); os.IsNotExist(err) {
		if id == "_all" {
			return nil // Nothing to clear for non-existent _all progress
		}
		// For regular sessions, file should exist but treat as success if not
		return nil
	}

	// Acquire file lock
	fileLock := flock.New(lockPath)
	if err := fileLock.Lock(); err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer fileLock.Unlock()

	// Truncate the file
	if err := os.WriteFile(progressPath, []byte{}, 0644); err != nil {
		return fmt.Errorf("failed to clear progress file: %w", err)
	}

	return nil
}

// saveSession writes a session to disk
func (s *SessionStore) saveSession(session *JuggleSession) error {
	filePath := s.sessionFilePath(session.ID)

	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write session file: %w", err)
	}

	return nil
}

// ProjectDir returns the project directory for this store
func (s *SessionStore) ProjectDir() string {
	return s.projectDir
}
