package tui

import (
	"bufio"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/ohare93/juggle/internal/session"
	"github.com/ohare93/juggle/internal/watcher"
)

type ballsLoadedMsg struct {
	balls []*session.Ball
	err   error
}

func loadBalls(store *session.Store, config *session.Config, localOnly bool) tea.Cmd {
	return func() tea.Msg {
		var balls []*session.Ball

		if localOnly {
			// Load only from current project
			localBalls, err := store.LoadBalls()
			if err != nil {
				return ballsLoadedMsg{err: err}
			}
			balls = localBalls
		} else {
			// Load from all discovered projects
			projects, err := session.DiscoverProjects(config)
			if err != nil {
				return ballsLoadedMsg{err: err}
			}

			balls, err = session.LoadAllBalls(projects)
			if err != nil {
				return ballsLoadedMsg{err: err}
			}
		}

		return ballsLoadedMsg{balls: balls}
	}
}

type ballUpdatedMsg struct {
	ball *session.Ball
	err  error
}

func updateBall(store *session.Store, ball *session.Ball) tea.Cmd {
	return func() tea.Msg {
		if err := store.UpdateBall(ball); err != nil {
			return ballUpdatedMsg{err: err}
		}
		return ballUpdatedMsg{ball: ball}
	}
}

type ballArchivedMsg struct {
	ball *session.Ball
	err  error
}

// updateAndArchiveBall updates the ball and then archives it
func updateAndArchiveBall(store *session.Store, ball *session.Ball) tea.Cmd {
	return func() tea.Msg {
		// First update the ball to persist state changes
		if err := store.UpdateBall(ball); err != nil {
			return ballArchivedMsg{err: err}
		}
		// Then archive it (moves from balls.jsonl to archive/balls.jsonl)
		if err := store.ArchiveBall(ball); err != nil {
			return ballArchivedMsg{err: err}
		}
		return ballArchivedMsg{ball: ball}
	}
}

// Sessions loading for split view
type sessionsLoadedMsg struct {
	sessions []*session.JuggleSession
	err      error
}

func loadSessions(sessionStore *session.SessionStore) tea.Cmd {
	return func() tea.Msg {
		if sessionStore == nil {
			return sessionsLoadedMsg{sessions: []*session.JuggleSession{}}
		}

		sessions, err := sessionStore.ListSessions()
		if err != nil {
			return sessionsLoadedMsg{err: err}
		}

		return sessionsLoadedMsg{sessions: sessions}
	}
}

// Watcher event messages
type watcherEventMsg struct {
	event watcher.Event
}

type watcherErrorMsg struct {
	err error
}

// listenForWatcherEvents creates a command that listens for watcher events
func listenForWatcherEvents(w *watcher.Watcher) tea.Cmd {
	return func() tea.Msg {
		select {
		case event := <-w.Events:
			return watcherEventMsg{event: event}
		case err := <-w.Errors:
			return watcherErrorMsg{err: err}
		}
	}
}

// Agent-related messages
type agentStartedMsg struct {
	sessionID string
}

type agentIterationMsg struct {
	sessionID string
	iteration int
	maxIter   int
}

type agentFinishedMsg struct {
	sessionID     string
	complete      bool
	blocked       bool
	blockedReason string
	iterations    int
	ballsComplete int
	ballsTotal    int
	err           error
}

// agentOutputMsg is sent when agent produces output
type agentOutputMsg struct {
	line    string
	isError bool // true if this is stderr output
}

// agentCancelledMsg is sent when the agent is cancelled by user
type agentCancelledMsg struct {
	sessionID string
}

// agentProcessStartedMsg is sent when agent process is started, providing reference for cancellation
type agentProcessStartedMsg struct {
	process   *AgentProcess
	sessionID string
}

// AgentStatus tracks the state of a running agent
type AgentStatus struct {
	Running       bool
	SessionID     string
	Iteration     int
	MaxIterations int
}

// AgentProcess holds state for a running agent with output streaming
type AgentProcess struct {
	cmd        *exec.Cmd
	stdout     io.ReadCloser
	stderr     io.ReadCloser
	outputCh   chan<- agentOutputMsg
	sessionID  string
	cancelled  bool
}

// Kill terminates the running agent process
func (p *AgentProcess) Kill() error {
	if p == nil || p.cmd == nil || p.cmd.Process == nil {
		return nil
	}
	p.cancelled = true
	return p.cmd.Process.Kill()
}

// launchAgentCmd creates a command that runs the agent for a session
func launchAgentCmd(sessionID string) tea.Cmd {
	return func() tea.Msg {
		// Launch "juggle agent run" as a subprocess
		// This allows the TUI to continue running while the agent works
		cmd := exec.Command("juggle", "agent", "run", sessionID)

		// Start the command in the background
		if err := cmd.Start(); err != nil {
			return agentFinishedMsg{
				sessionID: sessionID,
				err:       err,
			}
		}

		// Wait for the command to complete in this goroutine
		// The TUI will continue to be responsive because this runs
		// in a background goroutine (tea.Cmd runs async)
		if err := cmd.Wait(); err != nil {
			// Check if it was just a non-zero exit (common for blocked/incomplete)
			if _, ok := err.(*exec.ExitError); !ok {
				return agentFinishedMsg{
					sessionID: sessionID,
					err:       err,
				}
			}
		}

		// Agent finished - file watcher will pick up ball changes
		return agentFinishedMsg{
			sessionID: sessionID,
			complete:  true,
		}
	}
}

// launchAgentWithOutputCmd creates a command that runs the agent and streams output
// It returns the process reference via agentProcessStartedMsg for cancellation support
func launchAgentWithOutputCmd(sessionID string, outputCh chan<- agentOutputMsg) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("juggle", "agent", "run", sessionID)

		// Create pipes for stdout and stderr
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return agentFinishedMsg{sessionID: sessionID, err: err}
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			return agentFinishedMsg{sessionID: sessionID, err: err}
		}

		// Start the command
		if err := cmd.Start(); err != nil {
			return agentFinishedMsg{sessionID: sessionID, err: err}
		}

		// Create process reference for cancellation
		process := &AgentProcess{
			cmd:       cmd,
			stdout:    stdout,
			stderr:    stderr,
			outputCh:  outputCh,
			sessionID: sessionID,
		}

		// Stream stdout in a goroutine
		go func() {
			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				outputCh <- agentOutputMsg{line: scanner.Text(), isError: false}
			}
		}()

		// Stream stderr in a goroutine
		go func() {
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				outputCh <- agentOutputMsg{line: scanner.Text(), isError: true}
			}
		}()

		// Return process reference immediately so TUI can track it for cancellation
		// Wait for completion in a separate goroutine
		go func() {
			cmd.Wait() // Ignore error - will be handled by exit code or cancelled flag
		}()

		return agentProcessStartedMsg{process: process, sessionID: sessionID}
	}
}

// waitForAgentCmd waits for the agent process to complete
func waitForAgentCmd(process *AgentProcess) tea.Cmd {
	return func() tea.Msg {
		if process == nil || process.cmd == nil {
			return agentFinishedMsg{sessionID: "", complete: true}
		}

		// Wait for the command to finish
		err := process.cmd.Wait()

		// Check if cancelled
		if process.cancelled {
			return agentCancelledMsg{sessionID: process.sessionID}
		}

		// Check for errors
		if err != nil {
			if _, ok := err.(*exec.ExitError); !ok {
				return agentFinishedMsg{sessionID: process.sessionID, err: err}
			}
		}

		return agentFinishedMsg{sessionID: process.sessionID, complete: true}
	}
}

// listenForAgentOutput returns a command that waits for an output message on the channel
func listenForAgentOutput(outputCh <-chan agentOutputMsg) tea.Cmd {
	return func() tea.Msg {
		select {
		case msg := <-outputCh:
			return msg
		case <-time.After(100 * time.Millisecond):
			// Return nil to keep the listener alive without blocking
			return nil
		}
	}
}

// historyLoadedMsg is sent when agent history has been loaded
type historyLoadedMsg struct {
	history []*session.AgentRunRecord
	err     error
}

// loadAgentHistory creates a command to load agent run history
func loadAgentHistory(projectDir string) tea.Cmd {
	return func() tea.Msg {
		historyStore, err := session.NewAgentHistoryStore(projectDir)
		if err != nil {
			return historyLoadedMsg{err: err}
		}

		// Load the 50 most recent runs
		records, err := historyStore.LoadRecentHistory(50)
		if err != nil {
			return historyLoadedMsg{err: err}
		}

		return historyLoadedMsg{history: records}
	}
}

// historyOutputLoadedMsg is sent when last_output.txt content is loaded
type historyOutputLoadedMsg struct {
	content string
	err     error
}

// loadHistoryOutput creates a command to load the output file for a history record
func loadHistoryOutput(outputFile string) tea.Cmd {
	return func() tea.Msg {
		if outputFile == "" {
			return historyOutputLoadedMsg{content: "(no output file)", err: nil}
		}

		data, err := readFile(outputFile)
		if err != nil {
			return historyOutputLoadedMsg{content: "", err: err}
		}

		return historyOutputLoadedMsg{content: string(data), err: nil}
	}
}

// readFile is a helper to read file content
func readFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}
