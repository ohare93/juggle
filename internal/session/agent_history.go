package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

const (
	historyFile = "agent_history.jsonl"
)

// AgentRunRecord stores information about a past agent run
type AgentRunRecord struct {
	ID             string        `json:"id"`              // Unique run ID (timestamp-based)
	SessionID      string        `json:"session_id"`      // Session the agent ran on
	StartedAt      time.Time     `json:"started_at"`      // When the run started
	EndedAt        time.Time     `json:"ended_at"`        // When the run ended
	Iterations     int           `json:"iterations"`      // Number of iterations completed
	MaxIterations  int           `json:"max_iterations"`  // Maximum iterations configured
	Result         string        `json:"result"`          // "complete", "blocked", "timeout", "max_iterations", "rate_limit", "cancelled", "error"
	BlockedReason  string        `json:"blocked_reason,omitempty"`
	TimeoutMessage string        `json:"timeout_message,omitempty"`
	ErrorMessage   string        `json:"error_message,omitempty"`
	BallsComplete  int           `json:"balls_complete"`  // Number of balls completed
	BallsBlocked   int           `json:"balls_blocked"`   // Number of balls blocked
	BallsTotal     int           `json:"balls_total"`     // Total balls in session
	TotalWaitTime  time.Duration `json:"total_wait_time"` // Time spent waiting for rate limits
	OutputFile     string        `json:"output_file"`     // Path to last_output.txt
	ProjectDir     string        `json:"project_dir"`     // Project directory where agent ran
}

// NewAgentRunRecord creates a new agent run record with a unique ID
func NewAgentRunRecord(sessionID, projectDir string, startTime time.Time) *AgentRunRecord {
	id := fmt.Sprintf("%d", startTime.UnixNano())
	return &AgentRunRecord{
		ID:         id,
		SessionID:  sessionID,
		StartedAt:  startTime,
		ProjectDir: projectDir,
	}
}

// SetComplete marks the run as complete
func (r *AgentRunRecord) SetComplete(iterations int, ballsComplete, ballsBlocked, ballsTotal int) {
	r.Result = "complete"
	r.Iterations = iterations
	r.BallsComplete = ballsComplete
	r.BallsBlocked = ballsBlocked
	r.BallsTotal = ballsTotal
	r.EndedAt = time.Now()
}

// SetBlocked marks the run as blocked
func (r *AgentRunRecord) SetBlocked(iterations int, reason string, ballsComplete, ballsBlocked, ballsTotal int) {
	r.Result = "blocked"
	r.Iterations = iterations
	r.BlockedReason = reason
	r.BallsComplete = ballsComplete
	r.BallsBlocked = ballsBlocked
	r.BallsTotal = ballsTotal
	r.EndedAt = time.Now()
}

// SetTimeout marks the run as timed out
func (r *AgentRunRecord) SetTimeout(iterations int, message string, ballsComplete, ballsBlocked, ballsTotal int) {
	r.Result = "timeout"
	r.Iterations = iterations
	r.TimeoutMessage = message
	r.BallsComplete = ballsComplete
	r.BallsBlocked = ballsBlocked
	r.BallsTotal = ballsTotal
	r.EndedAt = time.Now()
}

// SetMaxIterations marks the run as reaching max iterations
func (r *AgentRunRecord) SetMaxIterations(iterations int, ballsComplete, ballsBlocked, ballsTotal int) {
	r.Result = "max_iterations"
	r.Iterations = iterations
	r.BallsComplete = ballsComplete
	r.BallsBlocked = ballsBlocked
	r.BallsTotal = ballsTotal
	r.EndedAt = time.Now()
}

// SetRateLimitExceeded marks the run as exceeding rate limit wait time
func (r *AgentRunRecord) SetRateLimitExceeded(iterations int, waitTime time.Duration, ballsComplete, ballsBlocked, ballsTotal int) {
	r.Result = "rate_limit"
	r.Iterations = iterations
	r.TotalWaitTime = waitTime
	r.BallsComplete = ballsComplete
	r.BallsBlocked = ballsBlocked
	r.BallsTotal = ballsTotal
	r.EndedAt = time.Now()
}

// SetCancelled marks the run as cancelled
func (r *AgentRunRecord) SetCancelled(iterations int, ballsComplete, ballsBlocked, ballsTotal int) {
	r.Result = "cancelled"
	r.Iterations = iterations
	r.BallsComplete = ballsComplete
	r.BallsBlocked = ballsBlocked
	r.BallsTotal = ballsTotal
	r.EndedAt = time.Now()
}

// SetError marks the run as errored
func (r *AgentRunRecord) SetError(iterations int, errMsg string, ballsComplete, ballsBlocked, ballsTotal int) {
	r.Result = "error"
	r.Iterations = iterations
	r.ErrorMessage = errMsg
	r.BallsComplete = ballsComplete
	r.BallsBlocked = ballsBlocked
	r.BallsTotal = ballsTotal
	r.EndedAt = time.Now()
}

// Duration returns the duration of the run
func (r *AgentRunRecord) Duration() time.Duration {
	if r.EndedAt.IsZero() {
		return time.Since(r.StartedAt)
	}
	return r.EndedAt.Sub(r.StartedAt)
}

// AgentHistoryStore handles persistence of agent run history
type AgentHistoryStore struct {
	projectDir string
	config     StoreConfig
}

// NewAgentHistoryStore creates a new history store for the given project directory
func NewAgentHistoryStore(projectDir string) (*AgentHistoryStore, error) {
	return NewAgentHistoryStoreWithConfig(projectDir, DefaultStoreConfig())
}

// NewAgentHistoryStoreWithConfig creates a new history store with custom configuration
func NewAgentHistoryStoreWithConfig(projectDir string, config StoreConfig) (*AgentHistoryStore, error) {
	if projectDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current directory: %w", err)
		}
		projectDir = cwd
	}

	return &AgentHistoryStore{
		projectDir: projectDir,
		config:     config,
	}, nil
}

// historyFilePath returns the path to the agent history file
func (s *AgentHistoryStore) historyFilePath() string {
	return filepath.Join(s.projectDir, s.config.JugglerDirName, historyFile)
}

// AppendRecord appends a run record to the history file
func (s *AgentHistoryStore) AppendRecord(record *AgentRunRecord) error {
	// Ensure .juggler directory exists
	jugglerDir := filepath.Join(s.projectDir, s.config.JugglerDirName)
	if err := os.MkdirAll(jugglerDir, 0755); err != nil {
		return fmt.Errorf("failed to create juggler directory: %w", err)
	}

	// Marshal record to JSON
	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("failed to marshal record: %w", err)
	}

	// Open file in append mode
	f, err := os.OpenFile(s.historyFilePath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open history file: %w", err)
	}
	defer f.Close()

	// Write JSON line
	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write history record: %w", err)
	}

	return nil
}

// LoadHistory loads all agent run records from the history file
func (s *AgentHistoryStore) LoadHistory() ([]*AgentRunRecord, error) {
	filePath := s.historyFilePath()

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []*AgentRunRecord{}, nil // No history yet
		}
		return nil, fmt.Errorf("failed to read history file: %w", err)
	}

	records := make([]*AgentRunRecord, 0)

	// Parse JSONL - each line is a JSON object
	lines := splitLines(string(data))
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}

		var record AgentRunRecord
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			// Skip malformed records
			continue
		}
		records = append(records, &record)
	}

	// Sort by start time descending (most recent first)
	sort.Slice(records, func(i, j int) bool {
		return records[i].StartedAt.After(records[j].StartedAt)
	})

	return records, nil
}

// LoadHistoryBySession loads agent run records for a specific session
func (s *AgentHistoryStore) LoadHistoryBySession(sessionID string) ([]*AgentRunRecord, error) {
	allRecords, err := s.LoadHistory()
	if err != nil {
		return nil, err
	}

	filtered := make([]*AgentRunRecord, 0)
	for _, record := range allRecords {
		if record.SessionID == sessionID {
			filtered = append(filtered, record)
		}
	}

	return filtered, nil
}

// LoadRecentHistory loads the most recent N records
func (s *AgentHistoryStore) LoadRecentHistory(limit int) ([]*AgentRunRecord, error) {
	records, err := s.LoadHistory()
	if err != nil {
		return nil, err
	}

	if len(records) > limit {
		records = records[:limit]
	}

	return records, nil
}

// ProjectDir returns the project directory for this store
func (s *AgentHistoryStore) ProjectDir() string {
	return s.projectDir
}

// splitLines splits a string into lines, handling different line endings
func splitLines(s string) []string {
	lines := make([]string, 0)
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			line := s[start:i]
			// Remove trailing \r if present
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			lines = append(lines, line)
			start = i + 1
		}
	}
	// Add final line if not empty
	if start < len(s) {
		line := s[start:]
		if len(line) > 0 && line[len(line)-1] == '\r' {
			line = line[:len(line)-1]
		}
		lines = append(lines, line)
	}
	return lines
}
