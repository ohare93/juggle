package tui

import (
	"github.com/charmbracelet/bubbletea"
	"github.com/ohare93/juggle/internal/session"
)

type viewMode int

const (
	listView viewMode = iota
	detailView
	helpView
	confirmDeleteView
)

type Model struct {
	store         *session.Store
	config        *session.Config
	localOnly     bool // restrict to local project only
	balls         []*session.Session
	filteredBalls []*session.Session

	// View state
	mode         viewMode
	cursor       int
	selectedBall *session.Session

	// Filter state
	filterStates   map[string]bool // State visibility toggles
	filterPriority string
	searchQuery    string

	// UI state
	width         int
	height        int
	message       string // Success/error messages
	err           error
	confirmAction string // What action is being confirmed (e.g., "delete")
}

func InitialModel(store *session.Store, config *session.Config, localOnly bool) Model {
	return Model{
		store:     store,
		config:    config,
		localOnly: localOnly,
		mode:      listView,
		filterStates: map[string]bool{
			"ready":    true,
			"juggling": true,
			"dropped":  true,
			"complete": true,
		},
		cursor: 0,
	}
}

func (m Model) Init() tea.Cmd {
	return loadBalls(m.store, m.config, m.localOnly)
}
