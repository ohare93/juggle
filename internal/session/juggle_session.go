package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	sessionsDir     = "sessions"
	sessionFile     = "session.json"
	progressFile    = "progress.txt"
)

// JuggleSession represents a grouping of balls by tag
// Session ID equals the tag, providing a simple mapping
type JuggleSession struct {
	ID          string    `json:"id"`          // Session ID (same as tag)
	Description string    `json:"description"` // Human-readable description
	Context     string    `json:"context"`     // Rich context for agent memory
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
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

// SessionStore handles persistence of JuggleSessions
type SessionStore struct {
	projectDir string
	config     StoreConfig
}

// NewSessionStore creates a new session store for the given project directory
func NewSessionStore(projectDir string) (*SessionStore, error) {
	return NewSessionStoreWithConfig(projectDir, DefaultStoreConfig())
}

// NewSessionStoreWithConfig creates a new session store with custom configuration
func NewSessionStoreWithConfig(projectDir string, config StoreConfig) (*SessionStore, error) {
	if projectDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current directory: %w", err)
		}
		projectDir = cwd
	}

	return &SessionStore{
		projectDir: projectDir,
		config:     config,
	}, nil
}

// sessionPath returns the path to a session's directory
func (s *SessionStore) sessionPath(id string) string {
	return filepath.Join(s.projectDir, s.config.JugglerDirName, sessionsDir, id)
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
	sessionsPath := filepath.Join(s.projectDir, s.config.JugglerDirName, sessionsDir)

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
	// Verify session exists
	if _, err := s.LoadSession(id); err != nil {
		return err
	}

	progressPath := s.progressFilePath(id)

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
	// Verify session exists
	if _, err := s.LoadSession(id); err != nil {
		return "", err
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
