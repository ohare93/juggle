package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbletea"
	"github.com/ohare93/juggle/internal/session"
	"github.com/ohare93/juggle/internal/watcher"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		// Handle input modes (text entry)
		if m.mode == inputSessionView || m.mode == inputBallView || m.mode == inputBlockedView || m.mode == inputTagView {
			return m.handleInputKey(msg)
		}

		// Handle session selector mode
		if m.mode == sessionSelectorView {
			return m.handleSessionSelectorKey(msg)
		}

		// Handle panel search input
		if m.mode == panelSearchView {
			return m.handlePanelSearchKey(msg)
		}

		// Handle delete confirmation in split view
		if m.mode == confirmSplitDelete {
			return m.handleSplitConfirmDelete(msg)
		}

		// Handle agent launch confirmation
		if m.mode == confirmAgentLaunch {
			return m.handleAgentLaunchConfirm(msg)
		}

		// Handle split help view
		if m.mode == splitHelpView {
			return m.handleSplitHelpKey(msg)
		}

		// Handle split view specific keys first
		if m.mode == splitView {
			return m.handleSplitViewKey(msg)
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			if m.mode == listView && m.cursor > 0 {
				m.cursor--
				m.message = "" // Clear message on navigation
			}
			return m, nil

		case "down", "j":
			if m.mode == listView && m.cursor < len(m.filteredBalls)-1 {
				m.cursor++
				m.message = "" // Clear message on navigation
			}
			return m, nil

		case "enter":
			// Switch to detail view
			if m.mode == listView && len(m.filteredBalls) > 0 {
				m.selectedBall = m.filteredBalls[m.cursor]
				m.mode = detailView
			}
			return m, nil

		case "b":
			// Back to list view
			if m.mode == detailView || m.mode == helpView {
				m.mode = listView
				m.message = ""
			}
			return m, nil

		case "esc":
			// Back to list view, or exit if already at list view
			if m.mode == detailView || m.mode == helpView || m.mode == confirmDeleteView {
				m.mode = listView
				m.message = ""
				return m, nil
			} else if m.mode == listView {
				// At list view, exit the TUI
				return m, tea.Quit
			}
			return m, nil

		case "s":
			// Start ball (quick action)
			if m.mode == listView && len(m.filteredBalls) > 0 {
				return m.handleStartBall()
			}
			return m, nil

		case "c":
			// Complete ball (quick action)
			if m.mode == listView && len(m.filteredBalls) > 0 {
				return m.handleCompleteBall()
			}
			return m, nil

		case "d":
			// Drop ball (quick action)
			if m.mode == listView && len(m.filteredBalls) > 0 {
				return m.handleDropBall()
			}
			return m, nil

		case "x":
			// Delete ball (with confirmation)
			if m.mode == listView && len(m.filteredBalls) > 0 {
				m.mode = confirmDeleteView
				m.confirmAction = "delete"
				return m, nil
			}
			return m, nil

		case "p":
			// Cycle priority
			if m.mode == listView && len(m.filteredBalls) > 0 {
				return m.handleCyclePriority()
			}
			return m, nil

		case "y", "Y":
			// Confirm action (e.g., delete)
			if m.mode == confirmDeleteView {
				return m.handleConfirmDelete()
			}
			return m, nil

		case "n", "N":
			// Cancel confirmation
			if m.mode == confirmDeleteView {
				m.mode = listView
				m.message = "Cancelled"
				return m, nil
			}
			return m, nil

		case "?":
			// Toggle help
			if m.mode == helpView {
				m.mode = listView
			} else {
				m.mode = helpView
			}
			return m, nil

		case "r":
			// Set ball to ready state
			if m.mode == listView && len(m.filteredBalls) > 0 {
				return m.handleSetReady()
			}
			return m, nil

		case "R":
			// Refresh/reload balls
			m.message = "Reloading balls..."
			return m, loadBalls(m.store, m.config, m.localOnly)

		case "tab":
			// Cycle ball state
			if m.mode == listView && len(m.filteredBalls) > 0 {
				return m.handleCycleState()
			}
			return m, nil

		case "1", "2", "3", "4", "5":
			// Filter by state
			return m.handleStateFilter(msg.String())
		}

	case ballsLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.balls = msg.balls
		m.applyFilters()
		// Reset cursor if it's out of bounds
		if m.cursor >= len(m.filteredBalls) {
			m.cursor = 0
		}
		if m.mode == splitView {
			m.addActivity("Balls loaded")
		}
		return m, nil

	case sessionsLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.sessions = msg.sessions
		// Reset session cursor if out of bounds
		if m.sessionCursor >= len(m.sessions) {
			m.sessionCursor = 0
		}
		// Pre-select session if initialSessionID was provided
		if m.initialSessionID != "" && m.selectedSession == nil {
			for i, sess := range m.sessions {
				if sess.ID == m.initialSessionID {
					m.selectedSession = sess
					m.sessionCursor = i
					m.addActivity("Pre-selected session: " + sess.ID)
					break
				}
			}
			// Clear the initialSessionID after attempting selection
			m.initialSessionID = ""
		}
		if m.mode == splitView {
			m.addActivity("Sessions loaded")
		}
		return m, nil

	case ballUpdatedMsg:
		if msg.err != nil {
			m.message = "Error: " + msg.err.Error()
			if m.mode == splitView {
				m.addActivity("Error: " + msg.err.Error())
			}
		} else {
			m.message = "Ball updated successfully"
			if m.mode == splitView {
				m.addActivity("Ball updated: " + msg.ball.ID)
			}
		}
		// Reload balls
		return m, loadBalls(m.store, m.config, m.localOnly)

	case ballArchivedMsg:
		if msg.err != nil {
			m.message = "Error: " + msg.err.Error()
			if m.mode == splitView {
				m.addActivity("Error archiving: " + msg.err.Error())
			}
		} else {
			m.message = "Ball archived successfully"
			if m.mode == splitView {
				m.addActivity("Archived ball: " + msg.ball.ID)
			}
			// Clear selection if we archived the selected ball
			if m.selectedBall != nil && m.selectedBall.ID == msg.ball.ID {
				m.selectedBall = nil
			}
		}
		// Reload balls
		return m, loadBalls(m.store, m.config, m.localOnly)

	case watcherEventMsg:
		return m.handleWatcherEvent(msg.event)

	case watcherErrorMsg:
		m.addActivity("Watcher error: " + msg.err.Error())
		// Continue listening for more events
		if m.fileWatcher != nil {
			return m, listenForWatcherEvents(m.fileWatcher)
		}
		return m, nil

	case editorResultMsg:
		return m.handleEditorResult(msg)

	case agentStartedMsg:
		m.agentStatus = AgentStatus{
			Running:       true,
			SessionID:     msg.sessionID,
			Iteration:     0,
			MaxIterations: 10, // Default
		}
		m.addActivity("Agent started for session: " + msg.sessionID)
		m.message = "Agent running..."
		return m, nil

	case agentIterationMsg:
		m.agentStatus.Iteration = msg.iteration
		m.agentStatus.MaxIterations = msg.maxIter
		m.addActivity(fmt.Sprintf("Agent iteration %d/%d", msg.iteration, msg.maxIter))
		return m, nil

	case agentFinishedMsg:
		m.agentStatus.Running = false
		if msg.err != nil {
			m.message = "Agent error: " + msg.err.Error()
			m.addActivity("Agent error: " + msg.err.Error())
		} else if msg.complete {
			m.message = "Agent complete!"
			m.addActivity("Agent completed: " + msg.sessionID)
		} else if msg.blocked {
			m.message = "Agent blocked: " + msg.blockedReason
			m.addActivity("Agent blocked: " + msg.blockedReason)
		} else {
			m.message = "Agent finished (max iterations)"
			m.addActivity("Agent finished: max iterations reached")
		}
		// Reload balls to reflect any changes
		return m, loadBalls(m.store, m.config, m.localOnly)
	}

	return m, nil
}

// handleEditorResult handles the result from external editor
func (m Model) handleEditorResult(msg editorResultMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.message = "Editor error: " + msg.err.Error()
		m.addActivity("Editor error: " + msg.err.Error())
		return m, nil
	}

	if msg.cancelled {
		m.message = "Edit cancelled (no changes)"
		m.addActivity("Edit cancelled for: " + msg.ball.ID)
		return m, nil
	}

	// Parse the edited YAML and apply changes
	if err := yamlToBall(msg.editedYAML, msg.ball); err != nil {
		m.message = "Parse error: " + err.Error()
		m.addActivity("Parse error: " + err.Error())
		return m, nil
	}

	// Save the updated ball
	store, err := session.NewStore(msg.ball.WorkingDir)
	if err != nil {
		m.message = "Error: " + err.Error()
		m.addActivity("Store error: " + err.Error())
		return m, nil
	}

	m.addActivity("Updated ball: " + msg.ball.ID)
	m.message = "Updated ball: " + msg.ball.ID
	return m, updateBall(store, msg.ball)
}

// handleSplitViewKey handles keyboard input for split view mode
func (m Model) handleSplitViewKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "tab", "l":
		// Cycle to next panel
		m.message = ""
		switch m.activePanel {
		case SessionsPanel:
			m.activePanel = BallsPanel
		case BallsPanel:
			m.activePanel = ActivityPanel
		case ActivityPanel:
			m.activePanel = SessionsPanel
		}
		return m, nil

	case "shift+tab", "h":
		// Cycle to previous panel
		m.message = ""
		switch m.activePanel {
		case SessionsPanel:
			m.activePanel = ActivityPanel
		case BallsPanel:
			m.activePanel = SessionsPanel
		case ActivityPanel:
			m.activePanel = BallsPanel
		}
		return m, nil

	case "up", "k":
		m.message = ""
		return m.handleSplitViewNavUp()

	case "down", "j":
		m.message = ""
		return m.handleSplitViewNavDown()

	case "ctrl+d":
		// Page down in activity log
		if m.activePanel == ActivityPanel {
			return m.handleActivityLogPageDown()
		}
		return m, nil

	case "ctrl+u":
		// Page up in activity log (or clear filter in other panels)
		if m.activePanel == ActivityPanel {
			return m.handleActivityLogPageUp()
		}
		// Clear the current panel filter
		m.panelSearchQuery = ""
		m.panelSearchActive = false
		m.addActivity("Filter cleared")
		m.message = "Filter cleared"
		return m, nil

	case "g":
		// Handle gg for go to top of activity log
		if m.activePanel == ActivityPanel {
			if m.lastKey == "g" {
				m.lastKey = ""
				return m.handleActivityLogGoToTop()
			}
			m.lastKey = "g"
			return m, nil
		}
		return m, nil

	case "G":
		// Go to bottom of activity log
		if m.activePanel == ActivityPanel {
			m.lastKey = ""
			return m.handleActivityLogGoToBottom()
		}
		return m, nil

	case "enter":
		return m.handleSplitViewEnter()

	case "esc":
		// Go back or deselect
		if m.selectedBall != nil {
			m.selectedBall = nil
		} else if m.selectedSession != nil {
			m.selectedSession = nil
			m.cursor = 0
		} else {
			return m, tea.Quit
		}
		return m, nil

	case " ":
		// Space key: go back to sessions in BallsPanel
		if m.activePanel == BallsPanel {
			// Move focus back to sessions panel
			m.activePanel = SessionsPanel
			return m, nil
		}
		return m, nil

	case "s":
		// Start ball
		if m.activePanel == BallsPanel {
			return m.handleSplitStartBall()
		}
		return m, nil

	case "c":
		// Complete ball
		if m.activePanel == BallsPanel {
			return m.handleSplitCompleteBall()
		}
		return m, nil

	case "b":
		// Block ball
		if m.activePanel == BallsPanel {
			return m.handleSplitBlockBall()
		}
		return m, nil

	case "R":
		// Refresh
		m.message = "Reloading..."
		m.addActivity("Refreshing data...")
		return m, tea.Batch(
			loadBalls(m.store, m.config, m.localOnly),
			loadSessions(m.sessionStore),
		)

	case "?":
		// Show comprehensive help view
		m.helpScrollOffset = 0 // Reset scroll position
		m.mode = splitHelpView
		return m, nil

	case "a":
		// Add new item based on panel
		return m.handleSplitAddItem()

	case "e":
		// Edit selected item based on panel
		return m.handleSplitEditItem()

	case "d":
		// Delete selected item with confirmation
		return m.handleSplitDeletePrompt()

	case "/":
		// Open search/filter for current panel
		return m.handlePanelSearchStart()

	case "t":
		// Edit tags for selected ball
		if m.activePanel == BallsPanel {
			return m.handleTagEditStart()
		}
		return m, nil

	case "[":
		// Switch to previous session while in balls panel
		if m.activePanel == BallsPanel {
			return m.handleSessionSwitchPrev()
		}
		return m, nil

	case "]":
		// Switch to next session while in balls panel
		if m.activePanel == BallsPanel {
			return m.handleSessionSwitchNext()
		}
		return m, nil

	case "i":
		// Toggle bottom pane between activity log and ball detail
		return m.handleToggleBottomPane()

	case "P":
		// Toggle between local project only and all projects
		return m.handleToggleLocalOnly()

	case "A":
		// Launch agent for selected session
		if m.activePanel == SessionsPanel {
			return m.handleLaunchAgent()
		}
		return m, nil
	}

	return m, nil
}

// handleSplitViewNavUp handles up navigation in split view
func (m Model) handleSplitViewNavUp() (tea.Model, tea.Cmd) {
	m.lastKey = "" // Clear gg state
	switch m.activePanel {
	case SessionsPanel:
		sessions := m.filterSessions()
		if m.sessionCursor > 0 && m.sessionCursor < len(sessions) {
			m.sessionCursor--
			// Scroll-to-select: automatically select the session when navigating
			m.selectedSession = sessions[m.sessionCursor]
			m.cursor = 0 // Reset ball cursor for new session
		} else if m.sessionCursor >= len(sessions) && len(sessions) > 0 {
			m.sessionCursor = len(sessions) - 1
			m.selectedSession = sessions[m.sessionCursor]
			m.cursor = 0
		}
	case BallsPanel:
		if m.cursor > 0 {
			m.cursor--
		}
	case ActivityPanel:
		// Scroll up one line in activity log
		if m.activityLogOffset > 0 {
			m.activityLogOffset--
		}
	}
	return m, nil
}

// handleSplitViewNavDown handles down navigation in split view
func (m Model) handleSplitViewNavDown() (tea.Model, tea.Cmd) {
	m.lastKey = "" // Clear gg state
	switch m.activePanel {
	case SessionsPanel:
		sessions := m.filterSessions()
		if m.sessionCursor < len(sessions)-1 {
			m.sessionCursor++
			// Scroll-to-select: automatically select the session when navigating
			m.selectedSession = sessions[m.sessionCursor]
			m.cursor = 0 // Reset ball cursor for new session
		}
	case BallsPanel:
		balls := m.filterBallsForSession()
		if m.cursor < len(balls)-1 {
			m.cursor++
		}
	case ActivityPanel:
		// Scroll down one line in activity log
		maxOffset := m.getActivityLogMaxOffset()
		if m.activityLogOffset < maxOffset {
			m.activityLogOffset++
		}
	}
	return m, nil
}

// handleSessionSwitchPrev switches to the previous session while in balls panel
func (m Model) handleSessionSwitchPrev() (tea.Model, tea.Cmd) {
	sessions := m.filterSessions()
	if len(sessions) == 0 {
		return m, nil
	}

	if m.sessionCursor > 0 {
		m.sessionCursor--
		m.selectedSession = sessions[m.sessionCursor]
		m.cursor = 0 // Reset ball cursor for new session
		m.addActivity("Switched to session: " + m.selectedSession.ID)
	}
	return m, nil
}

// handleSessionSwitchNext switches to the next session while in balls panel
func (m Model) handleSessionSwitchNext() (tea.Model, tea.Cmd) {
	sessions := m.filterSessions()
	if len(sessions) == 0 {
		return m, nil
	}

	if m.sessionCursor < len(sessions)-1 {
		m.sessionCursor++
		m.selectedSession = sessions[m.sessionCursor]
		m.cursor = 0 // Reset ball cursor for new session
		m.addActivity("Switched to session: " + m.selectedSession.ID)
	}
	return m, nil
}

// handleToggleBottomPane toggles between activity log and ball detail in bottom pane
func (m Model) handleToggleBottomPane() (tea.Model, tea.Cmd) {
	if m.bottomPaneMode == BottomPaneActivity {
		m.bottomPaneMode = BottomPaneDetail
		m.addActivity("Showing ball details in bottom pane")
	} else {
		m.bottomPaneMode = BottomPaneActivity
		m.addActivity("Showing activity log in bottom pane")
	}
	return m, nil
}

// handleToggleLocalOnly toggles between local project only and all projects
func (m Model) handleToggleLocalOnly() (tea.Model, tea.Cmd) {
	m.localOnly = !m.localOnly
	if m.localOnly {
		m.addActivity("Showing local project only")
		m.message = "Showing local project only"
	} else {
		m.addActivity("Showing all projects")
		m.message = "Showing all projects"
	}
	// Reload balls with new scope
	return m, loadBalls(m.store, m.config, m.localOnly)
}

// getActivityLogMaxOffset calculates the maximum scroll offset for activity log
func (m Model) getActivityLogMaxOffset() int {
	visibleLines := bottomPanelRows - 3 // Account for title and borders
	if visibleLines < 1 {
		visibleLines = 1
	}
	maxOffset := len(m.activityLog) - visibleLines
	if maxOffset < 0 {
		maxOffset = 0
	}
	return maxOffset
}

// handleActivityLogPageDown scrolls down half a page in the activity log
func (m Model) handleActivityLogPageDown() (tea.Model, tea.Cmd) {
	m.lastKey = "" // Clear gg state
	pageSize := (bottomPanelRows - 3) / 2
	if pageSize < 1 {
		pageSize = 1
	}
	maxOffset := m.getActivityLogMaxOffset()
	m.activityLogOffset += pageSize
	if m.activityLogOffset > maxOffset {
		m.activityLogOffset = maxOffset
	}
	return m, nil
}

// handleActivityLogPageUp scrolls up half a page in the activity log
func (m Model) handleActivityLogPageUp() (tea.Model, tea.Cmd) {
	m.lastKey = "" // Clear gg state
	pageSize := (bottomPanelRows - 3) / 2
	if pageSize < 1 {
		pageSize = 1
	}
	m.activityLogOffset -= pageSize
	if m.activityLogOffset < 0 {
		m.activityLogOffset = 0
	}
	return m, nil
}

// handleActivityLogGoToTop scrolls to the top of the activity log
func (m Model) handleActivityLogGoToTop() (tea.Model, tea.Cmd) {
	m.activityLogOffset = 0
	return m, nil
}

// handleActivityLogGoToBottom scrolls to the bottom of the activity log
func (m Model) handleActivityLogGoToBottom() (tea.Model, tea.Cmd) {
	m.activityLogOffset = m.getActivityLogMaxOffset()
	return m, nil
}

// handleSplitViewEnter handles enter key in split view
func (m Model) handleSplitViewEnter() (tea.Model, tea.Cmd) {
	switch m.activePanel {
	case SessionsPanel:
		// Select session and move focus to balls panel
		sessions := m.filterSessions()
		if len(sessions) > 0 && m.sessionCursor < len(sessions) {
			m.selectedSession = sessions[m.sessionCursor]
			m.cursor = 0 // Reset ball cursor for new session
			m.activePanel = BallsPanel
			m.addActivity("Selected session: " + m.selectedSession.ID)
		}
	case BallsPanel:
		// Select ball
		balls := m.filterBallsForSession()
		if len(balls) > 0 && m.cursor < len(balls) {
			m.selectedBall = balls[m.cursor]
			m.addActivity("Selected ball: " + m.selectedBall.ID)
		}
	}
	return m, nil
}

// handleSplitStartBall starts the selected ball in split view
func (m Model) handleSplitStartBall() (tea.Model, tea.Cmd) {
	balls := m.filterBallsForSession()
	if len(balls) == 0 || m.cursor >= len(balls) {
		return m, nil
	}

	ball := balls[m.cursor]
	ball.SetState(session.StateInProgress)
	m.addActivity("Started ball: " + ball.ID)

	store, err := session.NewStore(ball.WorkingDir)
	if err != nil {
		m.message = "Error: " + err.Error()
		return m, nil
	}

	return m, updateBall(store, ball)
}

// handleSplitCompleteBall completes the selected ball in split view and archives it
func (m Model) handleSplitCompleteBall() (tea.Model, tea.Cmd) {
	balls := m.filterBallsForSession()
	if len(balls) == 0 || m.cursor >= len(balls) {
		return m, nil
	}

	ball := balls[m.cursor]
	ball.SetState(session.StateComplete)
	m.addActivity("Completing ball: " + ball.ID)

	store, err := session.NewStore(ball.WorkingDir)
	if err != nil {
		m.message = "Error: " + err.Error()
		return m, nil
	}

	// Update and archive the completed ball
	return m, updateAndArchiveBall(store, ball)
}

// handleSplitBlockBall prompts for a blocked reason
func (m Model) handleSplitBlockBall() (tea.Model, tea.Cmd) {
	balls := m.filterBallsForSession()
	if len(balls) == 0 || m.cursor >= len(balls) {
		return m, nil
	}

	ball := balls[m.cursor]
	m.editingBall = ball
	m.textInput.Reset()
	m.textInput.Focus()
	m.textInput.Placeholder = "Blocked reason (e.g., waiting for API access)"
	m.inputTarget = "blocked_reason"
	m.mode = inputBlockedView
	m.addActivity("Blocking ball: " + ball.ID)

	return m, nil
}

func (m *Model) handleStartBall() (tea.Model, tea.Cmd) {
	ball := m.filteredBalls[m.cursor]

	// Update state to in_progress (no validation - allow any state transition)
	ball.SetState(session.StateInProgress)

	// Get the store for this ball's working directory
	store, err := session.NewStore(ball.WorkingDir)
	if err != nil {
		m.message = "Error creating store: " + err.Error()
		return m, nil
	}
	return m, updateBall(store, ball)
}

func (m *Model) handleCompleteBall() (tea.Model, tea.Cmd) {
	ball := m.filteredBalls[m.cursor]

	// Update state to complete (no validation - allow any state transition)
	ball.SetState(session.StateComplete)

	// Get the store for this ball's working directory
	store, err := session.NewStore(ball.WorkingDir)
	if err != nil {
		m.message = "Error creating store: " + err.Error()
		return m, nil
	}
	// Update and archive the completed ball
	return m, updateAndArchiveBall(store, ball)
}

func (m *Model) handleDropBall() (tea.Model, tea.Cmd) {
	ball := m.filteredBalls[m.cursor]

	// Update state to blocked (no validation - allow any state transition)
	ball.SetBlocked("dropped")

	// Get the store for this ball's working directory
	store, err := session.NewStore(ball.WorkingDir)
	if err != nil {
		m.message = "Error creating store: " + err.Error()
		return m, nil
	}
	return m, updateBall(store, ball)
}

func (m *Model) handleStateFilter(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "1":
		// Show all - set all to true
		m.filterStates["pending"] = true
		m.filterStates["in_progress"] = true
		m.filterStates["blocked"] = true
		m.filterStates["complete"] = true
		m.message = "Filter: showing all states"
	case "2":
		m.filterStates["pending"] = !m.filterStates["pending"]
	case "3":
		m.filterStates["in_progress"] = !m.filterStates["in_progress"]
	case "4":
		m.filterStates["blocked"] = !m.filterStates["blocked"]
	case "5":
		m.filterStates["complete"] = !m.filterStates["complete"]
	}

	// Build message showing active filters
	if key != "1" {
		var active []string
		for state, visible := range m.filterStates {
			if visible {
				active = append(active, state)
			}
		}
		m.message = "Showing: " + strings.Join(active, ", ")
	}

	m.applyFilters()
	m.cursor = 0
	return m, nil
}

func (m *Model) applyFilters() {
	m.filteredBalls = make([]*session.Ball, 0)

	for _, ball := range m.balls {
		// Check if this ball's state is visible
		if m.filterStates[string(ball.State)] {
			m.filteredBalls = append(m.filteredBalls, ball)
		}
	}
}

func (m *Model) handleCycleState() (tea.Model, tea.Cmd) {
	ball := m.filteredBalls[m.cursor]

	// Determine next state
	var nextState session.BallState

	switch ball.State {
	case session.StatePending:
		nextState = session.StateInProgress
	case session.StateInProgress:
		nextState = session.StateComplete
	case session.StateComplete:
		nextState = session.StateBlocked
	case session.StateBlocked:
		nextState = session.StatePending
	default:
		nextState = session.StatePending
	}

	ball.SetState(nextState)

	store, err := session.NewStore(ball.WorkingDir)
	if err != nil {
		m.message = "Error: " + err.Error()
		return m, nil
	}

	m.message = "Cycled to: " + formatState(ball)
	return m, updateBall(store, ball)
}

func (m *Model) handleSetReady() (tea.Model, tea.Cmd) {
	ball := m.filteredBalls[m.cursor]

	// Set to pending state
	ball.SetState(session.StatePending)

	store, err := session.NewStore(ball.WorkingDir)
	if err != nil {
		m.message = "Error: " + err.Error()
		return m, nil
	}

	m.message = "Ball set to pending"
	return m, updateBall(store, ball)
}

func (m *Model) handleCyclePriority() (tea.Model, tea.Cmd) {
	ball := m.filteredBalls[m.cursor]

	// Determine next priority
	var nextPriority session.Priority
	switch ball.Priority {
	case session.PriorityLow:
		nextPriority = session.PriorityMedium
	case session.PriorityMedium:
		nextPriority = session.PriorityHigh
	case session.PriorityHigh:
		nextPriority = session.PriorityUrgent
	case session.PriorityUrgent:
		nextPriority = session.PriorityLow
	default:
		nextPriority = session.PriorityMedium
	}

	ball.Priority = nextPriority

	store, err := session.NewStore(ball.WorkingDir)
	if err != nil {
		m.message = "Error: " + err.Error()
		return m, nil
	}

	m.message = "Priority: " + string(nextPriority)
	return m, updateBall(store, ball)
}

func (m *Model) handleConfirmDelete() (tea.Model, tea.Cmd) {
	ball := m.filteredBalls[m.cursor]

	// Get store for this ball's working directory
	store, err := session.NewStore(ball.WorkingDir)
	if err != nil {
		m.mode = listView
		m.message = "Error: " + err.Error()
		return m, nil
	}

	// Delete the ball
	err = store.DeleteBall(ball.ID)
	if err != nil {
		m.mode = listView
		m.message = "Error deleting ball: " + err.Error()
		return m, nil
	}

	m.mode = listView
	m.message = "Deleted ball: " + ball.ID

	// Reload balls
	return m, loadBalls(m.store, m.config, m.localOnly)
}

// handleSplitAddItem handles adding a new item based on active panel
func (m Model) handleSplitAddItem() (tea.Model, tea.Cmd) {
	m.inputAction = actionAdd
	m.textInput.Reset()
	m.textInput.Focus()

	switch m.activePanel {
	case SessionsPanel:
		m.textInput.Placeholder = "Session ID (e.g., feature-auth)"
		m.inputTarget = "session_id"
		m.mode = inputSessionView
		m.addActivity("Adding new session...")
	case BallsPanel:
		m.textInput.Placeholder = "Ball intent (what you're doing)"
		m.inputTarget = "intent"
		m.mode = inputBallView
		m.addActivity("Adding new ball...")
	}

	return m, nil
}

// handleSplitEditItem handles editing the selected item
func (m Model) handleSplitEditItem() (tea.Model, tea.Cmd) {
	m.inputAction = actionEdit
	m.textInput.Reset()
	m.textInput.Focus()

	switch m.activePanel {
	case SessionsPanel:
		sessions := m.filterSessions()
		if len(sessions) == 0 || m.sessionCursor >= len(sessions) {
			m.message = "No session selected"
			return m, nil
		}
		sess := sessions[m.sessionCursor]
		// Prevent editing pseudo-sessions
		if sess.ID == PseudoSessionAll || sess.ID == PseudoSessionUntagged {
			m.message = "Cannot edit built-in session"
			return m, nil
		}
		m.textInput.Placeholder = "Session description"
		m.textInput.SetValue(sess.Description)
		m.inputTarget = "session_description"
		m.mode = inputSessionView
		m.addActivity("Editing session: " + sess.ID)

	case BallsPanel:
		balls := m.filterBallsForSession()
		if len(balls) == 0 || m.cursor >= len(balls) {
			m.message = "No ball selected"
			return m, nil
		}
		ball := balls[m.cursor]
		m.editingBall = ball
		m.addActivity("Opening editor for: " + ball.ID)
		// Launch external editor for full ball editing
		return m, openEditorCmd(ball)
	}

	return m, nil
}

// handleSplitDeletePrompt shows delete confirmation
func (m Model) handleSplitDeletePrompt() (tea.Model, tea.Cmd) {
	switch m.activePanel {
	case SessionsPanel:
		sessions := m.filterSessions()
		if len(sessions) == 0 || m.sessionCursor >= len(sessions) {
			m.message = "No session selected"
			return m, nil
		}
		// Prevent deleting pseudo-sessions
		sess := sessions[m.sessionCursor]
		if sess.ID == PseudoSessionAll || sess.ID == PseudoSessionUntagged {
			m.message = "Cannot delete built-in session"
			return m, nil
		}
		m.confirmAction = "delete_session"
		m.mode = confirmSplitDelete
		m.addActivity("Confirming session deletion...")

	case BallsPanel:
		balls := m.filterBallsForSession()
		if len(balls) == 0 || m.cursor >= len(balls) {
			m.message = "No ball selected"
			return m, nil
		}
		m.confirmAction = "delete_ball"
		m.mode = confirmSplitDelete
		m.addActivity("Confirming ball deletion...")
	}

	return m, nil
}

// handleInputKey handles keyboard input in text input modes
func (m Model) handleInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Cancel input
		m.mode = splitView
		m.message = "Cancelled"
		m.textInput.Blur()
		return m, nil

	case "enter":
		// Submit input
		return m.handleInputSubmit()

	default:
		// Pass to textinput
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}
}

// handleInputSubmit handles submitting the input value
func (m Model) handleInputSubmit() (tea.Model, tea.Cmd) {
	value := strings.TrimSpace(m.textInput.Value())
	if value == "" {
		m.message = "Value cannot be empty"
		return m, nil
	}

	m.textInput.Blur()

	switch m.mode {
	case inputSessionView:
		return m.submitSessionInput(value)
	case inputBallView:
		return m.submitBallInput(value)
	case inputBlockedView:
		return m.submitBlockedInput(value)
	case inputTagView:
		return m.submitTagInput(value)
	}

	m.mode = splitView
	return m, nil
}

// submitSessionInput handles session add/edit submission
func (m Model) submitSessionInput(value string) (tea.Model, tea.Cmd) {
	if m.inputAction == actionAdd {
		// Create new session
		if m.sessionStore == nil {
			m.message = "Session store not available"
			m.mode = splitView
			return m, nil
		}
		_, err := m.sessionStore.CreateSession(value, "")
		if err != nil {
			m.message = "Error creating session: " + err.Error()
			m.mode = splitView
			return m, nil
		}
		m.addActivity("Created session: " + value)
		m.message = "Created session: " + value
	} else {
		// Edit session description
		// Note: use filterSessions() since sessionCursor indexes into filtered list
		sessions := m.filterSessions()
		if m.sessionCursor >= len(sessions) {
			m.mode = splitView
			return m, nil
		}
		sess := sessions[m.sessionCursor]
		// Double-check we're not editing a pseudo-session (shouldn't happen due to guard in handleSplitEditItem)
		if sess.ID == PseudoSessionAll || sess.ID == PseudoSessionUntagged {
			m.message = "Cannot edit built-in session"
			m.mode = splitView
			return m, nil
		}
		err := m.sessionStore.UpdateSessionDescription(sess.ID, value)
		if err != nil {
			m.message = "Error updating session: " + err.Error()
			m.mode = splitView
			return m, nil
		}
		m.addActivity("Updated session description: " + sess.ID)
		m.message = "Updated session: " + sess.ID
	}

	m.mode = splitView
	return m, loadSessions(m.sessionStore)
}

// submitBallInput handles ball add/edit submission
func (m Model) submitBallInput(value string) (tea.Model, tea.Cmd) {
	if m.inputAction == actionAdd {
		// Create new ball using the store's project directory
		ball, err := session.NewBall(m.store.ProjectDir(), value, session.PriorityMedium)
		if err != nil {
			m.message = "Error creating ball: " + err.Error()
			m.mode = splitView
			return m, nil
		}

		// Add session tag if a session is selected
		if m.selectedSession != nil {
			ball.Tags = append(ball.Tags, m.selectedSession.ID)
		}

		// Use the store's working directory
		err = m.store.AppendBall(ball)
		if err != nil {
			m.message = "Error creating ball: " + err.Error()
			m.mode = splitView
			return m, nil
		}
		m.addActivity("Created ball: " + ball.ID)
		m.message = "Created ball: " + ball.ID
	} else {
		// Edit ball intent
		if m.editingBall == nil {
			m.mode = splitView
			return m, nil
		}
		m.editingBall.Intent = value
		store, err := session.NewStore(m.editingBall.WorkingDir)
		if err != nil {
			m.message = "Error: " + err.Error()
			m.mode = splitView
			return m, nil
		}
		m.addActivity("Updated ball: " + m.editingBall.ID)
		m.message = "Updated ball: " + m.editingBall.ID
		m.mode = splitView
		return m, updateBall(store, m.editingBall)
	}

	m.mode = splitView
	return m, loadBalls(m.store, m.config, m.localOnly)
}

// submitBlockedInput handles blocked reason submission
func (m Model) submitBlockedInput(value string) (tea.Model, tea.Cmd) {
	if m.editingBall == nil {
		m.mode = splitView
		return m, nil
	}

	m.editingBall.SetBlocked(value)
	m.addActivity("Blocked ball: " + m.editingBall.ID + " - " + truncate(value, 20))
	m.message = "Blocked ball: " + m.editingBall.ID

	store, err := session.NewStore(m.editingBall.WorkingDir)
	if err != nil {
		m.message = "Error: " + err.Error()
		m.mode = splitView
		return m, nil
	}

	m.mode = splitView
	return m, updateBall(store, m.editingBall)
}

// handleSplitConfirmDelete handles yes/no for delete confirmation
func (m Model) handleSplitConfirmDelete(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		return m.executeSplitDelete()
	case "n", "N", "esc":
		m.mode = splitView
		m.message = "Cancelled"
		return m, nil
	}
	return m, nil
}

// executeSplitDelete performs the actual deletion
func (m Model) executeSplitDelete() (tea.Model, tea.Cmd) {
	switch m.confirmAction {
	case "delete_session":
		sessions := m.filterSessions()
		if m.sessionCursor >= len(sessions) {
			m.mode = splitView
			return m, nil
		}
		sess := sessions[m.sessionCursor]
		// Double-check we're not deleting a pseudo-session (shouldn't happen due to guard in handleSplitDeletePrompt)
		if sess.ID == PseudoSessionAll || sess.ID == PseudoSessionUntagged {
			m.message = "Cannot delete built-in session"
			m.mode = splitView
			return m, nil
		}
		err := m.sessionStore.DeleteSession(sess.ID)
		if err != nil {
			m.message = "Error deleting session: " + err.Error()
			m.mode = splitView
			return m, nil
		}
		m.addActivity("Deleted session: " + sess.ID)
		m.message = "Deleted session: " + sess.ID
		m.mode = splitView
		// Reset selection if we deleted the selected session
		if m.selectedSession != nil && m.selectedSession.ID == sess.ID {
			m.selectedSession = nil
		}
		return m, loadSessions(m.sessionStore)

	case "delete_ball":
		balls := m.filterBallsForSession()
		if m.cursor >= len(balls) {
			m.mode = splitView
			return m, nil
		}
		ball := balls[m.cursor]
		store, err := session.NewStore(ball.WorkingDir)
		if err != nil {
			m.message = "Error: " + err.Error()
			m.mode = splitView
			return m, nil
		}
		err = store.DeleteBall(ball.ID)
		if err != nil {
			m.message = "Error deleting ball: " + err.Error()
			m.mode = splitView
			return m, nil
		}
		m.addActivity("Deleted ball: " + ball.ID)
		m.message = "Deleted ball: " + ball.ID
		// Reset selection if we deleted the selected ball
		if m.selectedBall != nil && m.selectedBall.ID == ball.ID {
			m.selectedBall = nil
		}
		m.mode = splitView
		return m, loadBalls(m.store, m.config, m.localOnly)
	}

	m.mode = splitView
	return m, nil
}

// handlePanelSearchStart initiates search/filter mode for the current panel
func (m Model) handlePanelSearchStart() (tea.Model, tea.Cmd) {
	m.textInput.Reset()
	m.textInput.Focus()

	switch m.activePanel {
	case SessionsPanel:
		m.textInput.Placeholder = "Filter sessions..."
	case BallsPanel:
		m.textInput.Placeholder = "Filter balls..."
	}

	// Pre-fill with current filter if any
	if m.panelSearchQuery != "" {
		m.textInput.SetValue(m.panelSearchQuery)
	}

	m.mode = panelSearchView
	m.addActivity("Search mode activated")
	return m, nil
}

// handlePanelSearchKey handles keyboard input in panel search mode
func (m Model) handlePanelSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Cancel search, keep existing filter
		m.mode = splitView
		m.textInput.Blur()
		return m, nil

	case "enter":
		// Apply the filter
		value := strings.TrimSpace(m.textInput.Value())
		m.panelSearchQuery = value
		m.panelSearchActive = value != ""
		m.textInput.Blur()
		m.mode = splitView

		if m.panelSearchActive {
			m.addActivity("Filter applied: " + value)
			m.message = "Filter: " + value + " (Ctrl+U to clear)"
		} else {
			m.addActivity("Filter cleared")
			m.message = "Filter cleared"
		}
		return m, nil

	default:
		// Pass to textinput
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}
}

// filterSessions returns sessions filtered by the panel search query
// It prepends pseudo-sessions ("All" and "Untagged") at the top
func (m *Model) filterSessions() []*session.JuggleSession {
	// Create pseudo-sessions
	pseudoSessions := []*session.JuggleSession{
		{ID: PseudoSessionAll, Description: "All balls across all sessions"},
		{ID: PseudoSessionUntagged, Description: "Balls with no session tags"},
	}

	// Combine pseudo-sessions with real sessions
	allSessions := make([]*session.JuggleSession, 0, len(pseudoSessions)+len(m.sessions))
	allSessions = append(allSessions, pseudoSessions...)
	allSessions = append(allSessions, m.sessions...)

	if !m.panelSearchActive || m.panelSearchQuery == "" {
		return allSessions
	}

	query := strings.ToLower(m.panelSearchQuery)
	filtered := make([]*session.JuggleSession, 0)
	for _, sess := range allSessions {
		if strings.Contains(strings.ToLower(sess.ID), query) ||
			strings.Contains(strings.ToLower(sess.Description), query) {
			filtered = append(filtered, sess)
		}
	}
	return filtered
}

// filterBallsForSession returns balls filtered by session and search query
func (m *Model) filterBallsForSession() []*session.Ball {
	balls := m.getBallsForSession()

	if !m.panelSearchActive || m.panelSearchQuery == "" {
		return balls
	}

	query := strings.ToLower(m.panelSearchQuery)
	filtered := make([]*session.Ball, 0)
	for _, ball := range balls {
		if strings.Contains(strings.ToLower(ball.Intent), query) ||
			strings.Contains(strings.ToLower(ball.ID), query) {
			filtered = append(filtered, ball)
		}
	}
	return filtered
}

// handleWatcherEvent handles file system change events
func (m Model) handleWatcherEvent(event watcher.Event) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch event.Type {
	case watcher.BallsChanged:
		m.addActivity("File changed: balls.jsonl - reloading...")
		cmds = append(cmds, loadBalls(m.store, m.config, m.localOnly))

	case watcher.SessionChanged:
		msg := "Session file changed"
		if event.SessionID != "" {
			msg += ": " + event.SessionID
		}
		m.addActivity(msg + " - reloading...")
		cmds = append(cmds, loadSessions(m.sessionStore))

	case watcher.ProgressChanged:
		msg := "Progress updated"
		if event.SessionID != "" {
			msg += " for session: " + event.SessionID
		}
		m.addActivity(msg)
		// Progress changes don't require reloading UI data,
		// but log it for awareness
	}

	// Continue listening for more events
	if m.fileWatcher != nil {
		cmds = append(cmds, listenForWatcherEvents(m.fileWatcher))
	}

	return m, tea.Batch(cmds...)
}

// handleSplitHelpKey handles keyboard input in split help view
func (m Model) handleSplitHelpKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "?", "esc":
		// Close help view
		m.mode = splitView
		return m, nil

	case "j", "down":
		// Scroll down
		m.helpScrollOffset++
		return m, nil

	case "k", "up":
		// Scroll up
		if m.helpScrollOffset > 0 {
			m.helpScrollOffset--
		}
		return m, nil

	case "ctrl+d":
		// Page down
		m.helpScrollOffset += 10
		return m, nil

	case "ctrl+u":
		// Page up
		m.helpScrollOffset -= 10
		if m.helpScrollOffset < 0 {
			m.helpScrollOffset = 0
		}
		return m, nil

	case "g":
		// Handle gg for go to top
		if m.lastKey == "g" {
			m.lastKey = ""
			m.helpScrollOffset = 0
			return m, nil
		}
		m.lastKey = "g"
		return m, nil

	case "G":
		// Go to bottom (set to large number, will be clamped in render)
		m.lastKey = ""
		m.helpScrollOffset = 1000 // Large number, will be clamped
		return m, nil
	}

	// Reset gg detection for any other key
	m.lastKey = ""
	return m, nil
}

// handleSessionSelectorKey handles keyboard input in session selector mode
func (m Model) handleSessionSelectorKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		// Cancel selection
		m.mode = splitView
		m.sessionSelectItems = nil
		m.message = "Cancelled"
		return m, nil

	case "up", "k":
		// Move selection up
		if m.sessionSelectIndex > 0 {
			m.sessionSelectIndex--
		}
		return m, nil

	case "down", "j":
		// Move selection down
		if m.sessionSelectIndex < len(m.sessionSelectItems)-1 {
			m.sessionSelectIndex++
		}
		return m, nil

	case "enter", " ":
		// Select this session
		return m.submitSessionSelection()
	}
	return m, nil
}

// submitSessionSelection adds the selected session as a tag to the ball
func (m Model) submitSessionSelection() (tea.Model, tea.Cmd) {
	if m.editingBall == nil || len(m.sessionSelectItems) == 0 {
		m.mode = splitView
		m.sessionSelectItems = nil
		return m, nil
	}

	if m.sessionSelectIndex >= len(m.sessionSelectItems) {
		m.sessionSelectIndex = len(m.sessionSelectItems) - 1
	}

	selectedSession := m.sessionSelectItems[m.sessionSelectIndex]
	m.editingBall.AddTag(selectedSession.ID)
	m.addActivity("Added to session: " + selectedSession.ID)
	m.message = "Added to session: " + selectedSession.ID

	store, err := session.NewStore(m.editingBall.WorkingDir)
	if err != nil {
		m.message = "Error: " + err.Error()
		m.mode = splitView
		m.sessionSelectItems = nil
		return m, nil
	}

	m.mode = splitView
	m.sessionSelectItems = nil
	return m, updateBall(store, m.editingBall)
}

// handleTagEditStart opens the session selector for tagging the selected ball
func (m Model) handleTagEditStart() (tea.Model, tea.Cmd) {
	balls := m.filterBallsForSession()
	if len(balls) == 0 || m.cursor >= len(balls) {
		m.message = "No ball selected"
		return m, nil
	}

	ball := balls[m.cursor]
	m.editingBall = ball
	m.sessionSelectIndex = 0

	// Build list of sessions that ball is not already tagged with
	// Exclude pseudo-sessions and sessions already tagged
	existingTags := make(map[string]bool)
	for _, tag := range ball.Tags {
		existingTags[tag] = true
	}

	availableSessions := make([]*session.JuggleSession, 0)
	for _, sess := range m.sessions {
		// Skip if ball already has this tag
		if existingTags[sess.ID] {
			continue
		}
		availableSessions = append(availableSessions, sess)
	}

	if len(availableSessions) == 0 {
		m.message = "Ball already in all sessions"
		return m, nil
	}

	m.sessionSelectItems = availableSessions
	m.mode = sessionSelectorView
	m.addActivity("Selecting session for: " + ball.ID)

	return m, nil
}

// submitTagInput handles tag add/remove submission
func (m Model) submitTagInput(value string) (tea.Model, tea.Cmd) {
	if m.editingBall == nil {
		m.mode = splitView
		return m, nil
	}

	// Check if removing a tag (prefix with -)
	if strings.HasPrefix(value, "-") {
		tagToRemove := strings.TrimPrefix(value, "-")
		tagToRemove = strings.TrimSpace(tagToRemove)
		if tagToRemove == "" {
			m.message = "Tag name cannot be empty"
			return m, nil
		}

		// Check if tag exists
		hasTag := false
		for _, t := range m.editingBall.Tags {
			if t == tagToRemove {
				hasTag = true
				break
			}
		}

		if !hasTag {
			m.message = "Tag not found: " + tagToRemove
			m.mode = splitView
			return m, nil
		}

		m.editingBall.RemoveTag(tagToRemove)
		m.addActivity("Removed tag: " + tagToRemove + " from " + m.editingBall.ID)
		m.message = "Removed tag: " + tagToRemove
	} else {
		// Adding a tag
		tagToAdd := strings.TrimSpace(value)

		// Check if tag already exists
		for _, t := range m.editingBall.Tags {
			if t == tagToAdd {
				m.message = "Tag already exists: " + tagToAdd
				m.mode = splitView
				return m, nil
			}
		}

		m.editingBall.AddTag(tagToAdd)
		m.addActivity("Added tag: " + tagToAdd + " to " + m.editingBall.ID)
		m.message = "Added tag: " + tagToAdd
	}

	store, err := session.NewStore(m.editingBall.WorkingDir)
	if err != nil {
		m.message = "Error: " + err.Error()
		m.mode = splitView
		return m, nil
	}

	m.mode = splitView
	return m, updateBall(store, m.editingBall)
}

// handleLaunchAgent shows confirmation dialog for launching an agent
func (m Model) handleLaunchAgent() (tea.Model, tea.Cmd) {
	// Check if agent is already running
	if m.agentStatus.Running {
		m.message = "Agent already running for: " + m.agentStatus.SessionID
		return m, nil
	}

	// Check if session is selected
	if m.selectedSession == nil {
		m.message = "No session selected"
		return m, nil
	}

	// Prevent launching on pseudo-sessions
	if m.selectedSession.ID == PseudoSessionAll || m.selectedSession.ID == PseudoSessionUntagged {
		m.message = "Cannot launch agent on built-in session"
		return m, nil
	}

	// Show confirmation dialog
	m.mode = confirmAgentLaunch
	return m, nil
}

// handleAgentLaunchConfirm handles the agent launch confirmation
func (m Model) handleAgentLaunchConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		// Confirm launch
		sessionID := m.selectedSession.ID
		m.mode = splitView
		m.addActivity("Launching agent for: " + sessionID)
		m.message = "Starting agent for: " + sessionID

		// Set initial agent status
		m.agentStatus = AgentStatus{
			Running:       true,
			SessionID:     sessionID,
			Iteration:     0,
			MaxIterations: 10, // Default iterations
		}

		// Launch agent in background
		return m, launchAgentCmd(sessionID)

	case "n", "N", "esc", "q":
		// Cancel
		m.mode = splitView
		m.message = "Agent launch cancelled"
		return m, nil
	}

	return m, nil
}
