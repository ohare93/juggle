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
	splitView     // New three-panel split view
	splitHelpView // Comprehensive help view for split mode
	historyView   // Agent run history view

	// Input modes for CRUD operations
	inputSessionView     // Add/edit session
	inputBallView        // Add/edit ball
	inputBlockedView     // Prompt for blocked reason
	inputTagView         // Add/remove tags (legacy, kept for backwards compatibility)
	sessionSelectorView            // Session selector for tagging balls
	confirmSplitDelete             // Delete confirmation in split view
	panelSearchView                // Search/filter within current panel
	confirmAgentLaunch             // Agent launch confirmation
	confirmAgentCancel             // Agent cancel confirmation
	inputAcceptanceCriteriaView    // Multi-line acceptance criteria input
	inputBallFormView              // Full ball form with all fields
	historyOutputView              // Viewing last_output.txt from history
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
	ActivityPanel
)

// BottomPaneMode represents what the bottom pane displays
type BottomPaneMode int

const (
	BottomPaneActivity BottomPaneMode = iota // Show activity log
	BottomPaneDetail                         // Show highlighted ball details
	BottomPaneSplit                          // Show both activity and details side by side
)

// SortOrder represents how balls are sorted
type SortOrder int

const (
	SortByIDASC      SortOrder = iota // Sort by ID ascending (default)
	SortByIDDESC                      // Sort by ID descending
	SortByPriority                    // Sort by priority (urgent first)
	SortByLastActivity                // Sort by last activity (most recent first)
)

// Special pseudo-session IDs
const (
	PseudoSessionAll      = "__all__"
	PseudoSessionUntagged = "__untagged__"
)

// ActivityEntry represents a log entry in the activity log
type ActivityEntry struct {
	Time    time.Time
	Message string
}

// AgentOutputEntry represents a line of agent output
type AgentOutputEntry struct {
	Time    time.Time
	Line    string
	IsError bool // true if this is stderr output
}

type Model struct {
	store         *session.Store
	sessionStore  *session.SessionStore
	config        *session.Config
	localOnly     bool // restrict to local project only
	balls         []*session.Ball
	filteredBalls []*session.Ball

	// Session state (for split view)
	sessions        []*session.JuggleSession
	selectedSession *session.JuggleSession
	sessionCursor   int

	// View state
	mode         viewMode
	cursor       int
	selectedBall *session.Ball

	// Panel state (for split view)
	activePanel Panel

	// Activity log
	activityLog        []ActivityEntry
	activityLogOffset  int    // Scroll offset for activity log
	lastKey            string // Last key pressed (for gg detection)
	helpScrollOffset   int    // Scroll offset for help view
	ballsScrollOffset  int    // Scroll offset for balls panel viewport
	detailScrollOffset int    // Scroll offset for ball detail panel

	// Bottom pane mode (toggle between activity log and ball detail)
	bottomPaneMode BottomPaneMode

	// Sort order for balls
	sortOrder SortOrder

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
	textInput          textinput.Model
	inputAction        InputAction      // Add or Edit
	inputTarget        string           // What we're editing (e.g., "intent", "description")
	editingBall        *session.Ball // Ball being edited (for edit action)
	tagEditMode        TagEditMode      // Whether adding or removing a tag
	sessionSelectItems []*session.JuggleSession // Sessions available for selection
	sessionSelectIndex int                      // Current selection index in session selector

	// Pending ball creation state (for multi-step ball creation)
	pendingBallIntent          string   // Intent being created (stored during AC input)
	pendingBallPriority        int      // Index in priority options (0=low, 1=medium, 2=high, 3=urgent)
	pendingBallState           int      // Index in state options (0=pending, 1=in_progress)
	pendingBallTags            string   // Comma-separated tags
	pendingBallSession         int      // Index in session options (0=none, 1+ = session index)
	pendingBallFormField       int      // Current field in form (0=priority, 1=state, 2=tags, 3=session)
	pendingAcceptanceCriteria  []string // Acceptance criteria being collected

	// File watcher
	fileWatcher *watcher.Watcher

	// Agent state
	agentStatus AgentStatus // Status of running agent

	// Agent output panel state
	agentOutputVisible bool               // Whether agent output panel is shown
	agentOutput        []AgentOutputEntry // Buffer of agent output lines
	agentOutputOffset  int                // Scroll offset for agent output panel
	agentOutputCh      chan agentOutputMsg // Channel for receiving agent output

	// Agent process tracking for cancellation
	agentProcess *AgentProcess // Reference to running agent process for cancellation

	// Agent history state
	agentHistory        []*session.AgentRunRecord // Loaded agent run history
	historyCursor       int                       // Current selection in history view
	historyScrollOffset int                       // Scroll offset for history view
	historyOutput       string                    // Content of selected history's output file
	historyOutputOffset int                       // Scroll offset for output view
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
		activityLog:   make([]ActivityEntry, 0),
		textInput:     ti,
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
		// Adjust offset when we remove an entry
		if m.activityLogOffset > 0 {
			m.activityLogOffset--
		}
	}
	m.activityLog = append(m.activityLog, entry)

	// Auto-scroll to bottom unless actively viewing the activity panel
	// (user might be scrolled up to read history)
	if m.activePanel != ActivityPanel {
		m.activityLogOffset = m.getActivityLogMaxOffset()
	}
}

// SelectedSessionID returns the ID of the currently selected session (if any)
func (m Model) SelectedSessionID() string {
	if m.selectedSession != nil {
		return m.selectedSession.ID
	}
	return ""
}

// addAgentOutput adds a line to the agent output buffer
func (m *Model) addAgentOutput(line string, isError bool) {
	entry := AgentOutputEntry{
		Time:    time.Now(),
		Line:    line,
		IsError: isError,
	}
	// Keep last 500 lines
	if len(m.agentOutput) >= 500 {
		m.agentOutput = m.agentOutput[1:]
		// Adjust offset when we remove an entry
		if m.agentOutputOffset > 0 {
			m.agentOutputOffset--
		}
	}
	m.agentOutput = append(m.agentOutput, entry)

	// Auto-scroll to bottom when new output arrives
	m.agentOutputOffset = m.getAgentOutputMaxOffset()
}

// clearAgentOutput clears the agent output buffer
func (m *Model) clearAgentOutput() {
	m.agentOutput = make([]AgentOutputEntry, 0)
	m.agentOutputOffset = 0
}

// getAgentOutputMaxOffset returns the maximum scroll offset for the agent output panel
func (m Model) getAgentOutputMaxOffset() int {
	visibleLines := m.getAgentOutputVisibleLines()
	maxOffset := len(m.agentOutput) - visibleLines
	if maxOffset < 0 {
		maxOffset = 0
	}
	return maxOffset
}

// getAgentOutputVisibleLines returns the number of visible lines in the agent output panel
func (m Model) getAgentOutputVisibleLines() int {
	// Agent output panel takes up the right portion of the screen
	height := m.height - 6 // Account for borders, title, status
	if height < 3 {
		height = 3
	}
	return height
}

// getBallsForSession returns balls that belong to the selected session (by tag)
func (m *Model) getBallsForSession() []*session.Ball {
	if m.selectedSession == nil {
		return m.filteredBalls
	}

	// Handle pseudo-sessions
	switch m.selectedSession.ID {
	case PseudoSessionAll:
		// Return all balls
		return m.filteredBalls
	case PseudoSessionUntagged:
		// Return balls with no session tags (no tags that match any real session)
		untaggedBalls := make([]*session.Ball, 0)
		sessionIDs := make(map[string]bool)
		for _, sess := range m.sessions {
			sessionIDs[sess.ID] = true
		}
		for _, ball := range m.filteredBalls {
			hasSessionTag := false
			for _, tag := range ball.Tags {
				if sessionIDs[tag] {
					hasSessionTag = true
					break
				}
			}
			if !hasSessionTag {
				untaggedBalls = append(untaggedBalls, ball)
			}
		}
		return untaggedBalls
	default:
		// Regular session - return balls with matching tag
		sessionBalls := make([]*session.Ball, 0)
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
}
