package tui

import (
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbletea"
	"github.com/ohare93/juggle/internal/session"
	"github.com/ohare93/juggle/internal/watcher"
)

type viewMode int

const (
	listView viewMode = iota
	detailView
	helpView
	confirmDeleteView
	splitView // New three-panel split view

	// Input modes for CRUD operations
	inputSessionView    // Add/edit session
	inputBallView       // Add/edit ball
	inputTodoView       // Add/edit todo
	inputBlockedView    // Prompt for blocked reason
	inputTagView        // Add/remove tags
	confirmSplitDelete  // Delete confirmation in split view
	panelSearchView     // Search/filter within current panel
)

// InputAction represents what action triggered the input mode
type InputAction int

const (
	actionAdd InputAction = iota
	actionEdit
)

// TagEditMode represents whether we're adding or removing a tag
type TagEditMode int

const (
	tagModeAdd TagEditMode = iota
	tagModeRemove
)

// Panel represents which panel is active in split view
type Panel int

const (
	SessionsPanel Panel = iota
	BallsPanel
	TodosPanel
)

// ActivityEntry represents a log entry in the activity log
type ActivityEntry struct {
	Time    time.Time
	Message string
}

type Model struct {
	store         *session.Store
	sessionStore  *session.SessionStore
	config        *session.Config
	localOnly     bool // restrict to local project only
	balls         []*session.Session
	filteredBalls []*session.Session

	// Session state (for split view)
	sessions        []*session.JuggleSession
	selectedSession *session.JuggleSession
	sessionCursor   int

	// View state
	mode         viewMode
	cursor       int
	selectedBall *session.Session
	todoCursor   int // cursor position in todos panel

	// Panel state (for split view)
	activePanel Panel

	// Activity log
	activityLog []ActivityEntry

	// Filter state
	filterStates      map[string]bool // State visibility toggles
	filterPriority    string
	searchQuery       string
	initialSessionID  string // Pre-select session by ID (from --session flag)
	panelSearchQuery  string // Current search/filter query within a panel
	panelSearchActive bool   // Whether search/filter is active

	// UI state
	width         int
	height        int
	message       string // Success/error messages
	err           error
	confirmAction string // What action is being confirmed (e.g., "delete")

	// Input state for CRUD operations
	textInput   textinput.Model
	inputAction InputAction      // Add or Edit
	inputTarget string           // What we're editing (e.g., "intent", "description")
	editingBall *session.Session // Ball being edited (for edit action)
	editingTodo int              // Todo index being edited (-1 for new)
	tagEditMode TagEditMode      // Whether adding or removing a tag

	// File watcher
	fileWatcher *watcher.Watcher
}

// InitialModel creates a model for the legacy list view
func InitialModel(store *session.Store, config *session.Config, localOnly bool) Model {
	ti := textinput.New()
	ti.CharLimit = 256
	ti.Width = 40

	return Model{
		store:     store,
		config:    config,
		localOnly: localOnly,
		mode:      listView,
		filterStates: map[string]bool{
			"pending":     true,
			"in_progress": true,
			"blocked":     true,
			"complete":    true,
		},
		cursor:      0,
		activityLog: make([]ActivityEntry, 0),
		textInput:   ti,
		editingTodo: -1,
	}
}

// InitialSplitModel creates a model for the new split-view mode
func InitialSplitModel(store *session.Store, sessionStore *session.SessionStore, config *session.Config, localOnly bool) Model {
	return InitialSplitModelWithWatcher(store, sessionStore, config, localOnly, nil, "")
}

// InitialSplitModelWithWatcher creates a model for the new split-view mode with file watching
func InitialSplitModelWithWatcher(store *session.Store, sessionStore *session.SessionStore, config *session.Config, localOnly bool, w *watcher.Watcher, initialSessionID string) Model {
	ti := textinput.New()
	ti.CharLimit = 256
	ti.Width = 40

	return Model{
		store:            store,
		sessionStore:     sessionStore,
		config:           config,
		localOnly:        localOnly,
		mode:             splitView,
		activePanel:      SessionsPanel,
		initialSessionID: initialSessionID,
		filterStates: map[string]bool{
			"pending":     true,
			"in_progress": true,
			"blocked":     true,
			"complete":    true,
		},
		cursor:        0,
		sessionCursor: 0,
		todoCursor:    0,
		activityLog:   make([]ActivityEntry, 0),
		textInput:     ti,
		editingTodo:   -1,
		fileWatcher:   w,
	}
}

func (m Model) Init() tea.Cmd {
	if m.mode == splitView {
		cmds := []tea.Cmd{
			loadBalls(m.store, m.config, m.localOnly),
			loadSessions(m.sessionStore),
		}
		// Start file watcher if available
		if m.fileWatcher != nil {
			cmds = append(cmds, listenForWatcherEvents(m.fileWatcher))
		}
		return tea.Batch(cmds...)
	}
	return loadBalls(m.store, m.config, m.localOnly)
}

// addActivity adds an entry to the activity log
func (m *Model) addActivity(msg string) {
	entry := ActivityEntry{
		Time:    time.Now(),
		Message: msg,
	}
	// Keep last 100 entries
	if len(m.activityLog) >= 100 {
		m.activityLog = m.activityLog[1:]
	}
	m.activityLog = append(m.activityLog, entry)
}

// SelectedSessionID returns the ID of the currently selected session (if any)
func (m Model) SelectedSessionID() string {
	if m.selectedSession != nil {
		return m.selectedSession.ID
	}
	return ""
}

// getBallsForSession returns balls that belong to the selected session (by tag)
func (m *Model) getBallsForSession() []*session.Session {
	if m.selectedSession == nil {
		return m.filteredBalls
	}

	sessionBalls := make([]*session.Session, 0)
	for _, ball := range m.filteredBalls {
		for _, tag := range ball.Tags {
			if tag == m.selectedSession.ID {
				sessionBalls = append(sessionBalls, ball)
				break
			}
		}
	}
	return sessionBalls
}
