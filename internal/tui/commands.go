package tui

import (
	"os/exec"

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

// AgentStatus tracks the state of a running agent
type AgentStatus struct {
	Running       bool
	SessionID     string
	Iteration     int
	MaxIterations int
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
