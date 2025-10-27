package session

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	projectStorePath = ".juggler"
	ballsFile        = "balls.jsonl"
	archiveDir       = "archive"
	archiveBallsFile = "balls.jsonl"
)

// Store handles persistence of sessions in a project directory
// StoreConfig holds configurable options for Store
type StoreConfig struct {
	JugglerDirName string // Name of the juggler directory (default: ".juggler")
}

// DefaultStoreConfig returns the default store configuration
func DefaultStoreConfig() StoreConfig {
	return StoreConfig{
		JugglerDirName: projectStorePath,
	}
}

type Store struct {
	projectDir  string
	ballsPath   string
	archivePath string
	config      StoreConfig
}

// ProjectDir returns the project directory for this store
func (s *Store) ProjectDir() string {
	return s.projectDir
}

// NewStore creates a new store for the given project directory
func NewStore(projectDir string) (*Store, error) {
	return NewStoreWithConfig(projectDir, DefaultStoreConfig())
}

// NewStoreWithConfig creates a new store with custom configuration
func NewStoreWithConfig(projectDir string, config StoreConfig) (*Store, error) {
	if projectDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current directory: %w", err)
		}
		projectDir = cwd
	}

	storePath := filepath.Join(projectDir, config.JugglerDirName)
	ballsPath := filepath.Join(storePath, ballsFile)
	archivePath := filepath.Join(storePath, archiveDir, archiveBallsFile)

	// Ensure directories exist
	if err := os.MkdirAll(storePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create %s directory: %w", config.JugglerDirName, err)
	}

	archiveDirPath := filepath.Join(storePath, archiveDir)
	if err := os.MkdirAll(archiveDirPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create archive directory: %w", err)
	}

	return &Store{
		projectDir:  projectDir,
		ballsPath:   ballsPath,
		archivePath: archivePath,
		config:      config,
	}, nil
}

// AppendBall adds a new ball to the JSONL file
func (s *Store) AppendBall(ball *Session) error {
	data, err := json.Marshal(ball)
	if err != nil {
		return fmt.Errorf("failed to marshal ball: %w", err)
	}

	// Open file in append mode
	f, err := os.OpenFile(s.ballsPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open balls file: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("failed to write ball: %w", err)
	}

	if _, err := f.WriteString("\n"); err != nil {
		return fmt.Errorf("failed to write newline: %w", err)
	}

	return nil
}

// LoadBalls reads all balls from the JSONL file
func (s *Store) LoadBalls() ([]*Session, error) {
	// If file doesn't exist, return empty slice
	if _, err := os.Stat(s.ballsPath); os.IsNotExist(err) {
		return []*Session{}, nil
	}

	f, err := os.Open(s.ballsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open balls file: %w", err)
	}
	defer f.Close()

	balls := make([]*Session, 0)
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue // Skip empty lines
		}

		var ball Session
		if err := json.Unmarshal([]byte(line), &ball); err != nil {
			// Log error but continue
			fmt.Fprintf(os.Stderr, "Warning: failed to parse ball line: %v\n", err)
			continue
		}

		// Set WorkingDir from store location (not stored in JSON)
		ball.WorkingDir = s.projectDir

		balls = append(balls, &ball)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading balls file: %w", err)
	}

	return balls, nil
}

// LoadArchivedBalls reads all balls from the archive JSONL file
func (s *Store) LoadArchivedBalls() ([]*Session, error) {
	// If file doesn't exist, return empty slice
	if _, err := os.Stat(s.archivePath); os.IsNotExist(err) {
		return []*Session{}, nil
	}

	f, err := os.Open(s.archivePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open archive file: %w", err)
	}
	defer f.Close()

	balls := make([]*Session, 0)
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue // Skip empty lines
		}

		var ball Session
		if err := json.Unmarshal([]byte(line), &ball); err != nil {
			// Log error but continue
			fmt.Fprintf(os.Stderr, "Warning: failed to parse archived ball line: %v\n", err)
			continue
		}

		// Set WorkingDir from store location (not stored in JSON)
		ball.WorkingDir = s.projectDir

		balls = append(balls, &ball)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading archive file: %w", err)
	}

	return balls, nil
}

// UpdateBall updates an existing ball by rewriting the JSONL file
func (s *Store) UpdateBall(updated *Session) error {
	balls, err := s.LoadBalls()
	if err != nil {
		return err
	}

	// Find and update the ball
	found := false
	for i, ball := range balls {
		if ball.ID == updated.ID {
			balls[i] = updated
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("ball %s not found", updated.ID)
	}

	// Rewrite entire file
	return s.writeBalls(balls)
}

// DeleteBall removes a ball from the JSONL file
func (s *Store) DeleteBall(id string) error {
	balls, err := s.LoadBalls()
	if err != nil {
		return err
	}

	// Filter out the ball to delete
	filtered := make([]*Session, 0, len(balls))
	for _, ball := range balls {
		if ball.ID != id {
			filtered = append(filtered, ball)
		}
	}

	return s.writeBalls(filtered)
}

// ArchiveBall moves a ball to the archive
func (s *Store) ArchiveBall(ball *Session) error {
	// Append to archive
	data, err := json.Marshal(ball)
	if err != nil {
		return fmt.Errorf("failed to marshal ball: %w", err)
	}

	f, err := os.OpenFile(s.archivePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open archive file: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("failed to write to archive: %w", err)
	}

	if _, err := f.WriteString("\n"); err != nil {
		return fmt.Errorf("failed to write newline: %w", err)
	}

	// Remove from active balls
	return s.DeleteBall(ball.ID)
}

// GetCurrentBall attempts to find an active ball in the current directory
// GetJugglingBalls returns all balls currently being juggled in this project
func (s *Store) GetJugglingBalls() ([]*Session, error) {
	balls, err := s.LoadBalls()
	if err != nil {
		return nil, err
	}

	// Filter for juggling balls
	juggling := make([]*Session, 0)
	for _, ball := range balls {
		if ball.ActiveState == ActiveJuggling {
			juggling = append(juggling, ball)
		}
	}

	// Sort by most recently active first
	sort.Slice(juggling, func(i, j int) bool {
		return juggling[i].LastActivity.After(juggling[j].LastActivity)
	})

	return juggling, nil
}

// GetBallsByStatus returns all balls with the given status
// GetBallsByActiveState returns all balls with the given active state
func (s *Store) GetBallsByActiveState(state ActiveState) ([]*Session, error) {
	all, err := s.LoadBalls()
	if err != nil {
		return nil, err
	}

	filtered := make([]*Session, 0)
	for _, ball := range all {
		if ball.ActiveState == state {
			filtered = append(filtered, ball)
		}
	}

	return filtered, nil
}

// GetBallsByJuggleState returns all juggling balls with the given juggle state
func (s *Store) GetBallsByJuggleState(state JuggleState) ([]*Session, error) {
	all, err := s.LoadBalls()
	if err != nil {
		return nil, err
	}

	filtered := make([]*Session, 0)
	for _, ball := range all {
		if ball.ActiveState == ActiveJuggling && ball.JuggleState != nil && *ball.JuggleState == state {
			filtered = append(filtered, ball)
		}
	}

	return filtered, nil
}

// GetBallByID finds a ball by its ID
func (s *Store) GetBallByID(id string) (*Session, error) {
	balls, err := s.LoadBalls()
	if err != nil {
		return nil, err
	}

	for _, ball := range balls {
		if ball.ID == id {
			return ball, nil
		}
	}

	return nil, fmt.Errorf("ball %s not found", id)
}


// GetBallByShortID finds a ball by its short ID (numeric part)
// If multiple balls match, returns the most recently active
func (s *Store) GetBallByShortID(shortID string) (*Session, error) {
	balls, err := s.LoadBalls()
	if err != nil {
		return nil, err
	}

	matches := make([]*Session, 0)
	for _, ball := range balls {
		if ball.ShortID() == shortID {
			matches = append(matches, ball)
		}
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("ball with short ID %s not found", shortID)
	}

	// If multiple matches, return most recently active
	if len(matches) > 1 {
		sort.Slice(matches, func(i, j int) bool {
			return matches[i].LastActivity.After(matches[j].LastActivity)
		})
	}

	return matches[0], nil
}

// ResolveBallID resolves a ball ID from either full ID or short ID
func (s *Store) ResolveBallID(id string) (*Session, error) {
	// Try as full ID first
	ball, err := s.GetBallByID(id)
	if err == nil {
		return ball, nil
	}

	// Try as short ID
	return s.GetBallByShortID(id)
}

// writeBalls rewrites the entire balls.jsonl file
func (s *Store) writeBalls(balls []*Session) error {
	// Write to temp file first
	tempPath := s.ballsPath + ".tmp"
	f, err := os.Create(tempPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	for _, ball := range balls {
		data, err := json.Marshal(ball)
		if err != nil {
			f.Close()
			os.Remove(tempPath)
			return fmt.Errorf("failed to marshal ball: %w", err)
		}

		if _, err := f.Write(data); err != nil {
			f.Close()
			os.Remove(tempPath)
			return fmt.Errorf("failed to write ball: %w", err)
		}

		if _, err := f.WriteString("\n"); err != nil {
			f.Close()
			os.Remove(tempPath)
			return fmt.Errorf("failed to write newline: %w", err)
		}
	}

	if err := f.Close(); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, s.ballsPath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// UnarchiveBall restores a completed ball from archive back to ready state
func (s *Store) UnarchiveBall(ballID string) (*Session, error) {
	// Load archived balls
	archived, err := s.LoadArchivedBalls()
	if err != nil {
		return nil, fmt.Errorf("failed to load archived balls: %w", err)
	}

	// Find ball with matching ID
	var ball *Session
	var ballIndex int
	for i, b := range archived {
		if b.ID == ballID {
			ball = b
			ballIndex = i
			break
		}
	}
	if ball == nil {
		return nil, fmt.Errorf("ball not found in archive: %s", ballID)
	}

	// Change state to ready
	ball.ActiveState = ActiveReady
	ball.JuggleState = nil
	ball.CompletedAt = nil
	ball.CompletionNote = ""

	// Append to active balls
	if err := s.AppendBall(ball); err != nil {
		return nil, fmt.Errorf("failed to restore ball to active: %w", err)
	}

	// Remove from archive by rewriting archive file without this ball
	updatedArchive := make([]*Session, 0, len(archived)-1)
	for i, b := range archived {
		if i != ballIndex {
			updatedArchive = append(updatedArchive, b)
		}
	}

	if err := s.writeArchivedBalls(updatedArchive); err != nil {
		// Ball was added to active, but we failed to remove from archive
		// This is not ideal but not critical - archive will have duplicate
		return ball, fmt.Errorf("ball restored but failed to remove from archive: %w", err)
	}

	return ball, nil
}

// writeArchivedBalls rewrites the entire archive/balls.jsonl file
func (s *Store) writeArchivedBalls(balls []*Session) error {
	// Write to temp file first
	tempPath := s.archivePath + ".tmp"
	f, err := os.Create(tempPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	for _, ball := range balls {
		data, err := json.Marshal(ball)
		if err != nil {
			f.Close()
			os.Remove(tempPath)
			return fmt.Errorf("failed to marshal ball: %w", err)
		}

		if _, err := f.Write(data); err != nil {
			f.Close()
			os.Remove(tempPath)
			return fmt.Errorf("failed to write ball: %w", err)
		}

		if _, err := f.WriteString("\n"); err != nil {
			f.Close()
			os.Remove(tempPath)
			return fmt.Errorf("failed to write newline: %w", err)
		}
	}

	if err := f.Close(); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, s.archivePath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// Save is an alias for UpdateBall for backwards compatibility
func (s *Store) Save(ball *Session) error {
	// Check if ball already exists
	existing, err := s.GetBallByID(ball.ID)
	if err != nil || existing == nil {
		// New ball, append it
		return s.AppendBall(ball)
	}

	// Existing ball, update it
	return s.UpdateBall(ball)
}
