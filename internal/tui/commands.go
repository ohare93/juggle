package tui

import (
	"github.com/charmbracelet/bubbletea"
	"github.com/ohare93/juggle/internal/session"
)

type ballsLoadedMsg struct {
	balls []*session.Session
	err   error
}

func loadBalls(store *session.Store, config *session.Config, localOnly bool) tea.Cmd {
	return func() tea.Msg {
		var balls []*session.Session

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
	ball *session.Session
	err  error
}

func updateBall(store *session.Store, ball *session.Session) tea.Cmd {
	return func() tea.Msg {
		if err := store.UpdateBall(ball); err != nil {
			return ballUpdatedMsg{err: err}
		}
		return ballUpdatedMsg{ball: ball}
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
