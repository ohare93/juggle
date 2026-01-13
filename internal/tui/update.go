package tui

import (
	"fmt"
	"sort"
	"strconv"
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
		// Handle unified ball form view (all fields in one view)
		if m.mode == unifiedBallFormView {
			return m.handleUnifiedBallFormKey(msg)
		}

		// Handle acceptance criteria input mode (special handling for empty line)
		if m.mode == inputAcceptanceCriteriaView {
			return m.handleAcceptanceCriteriaKey(msg)
		}

		// Handle ball form view (multi-field form with selection)
		if m.mode == inputBallFormView {
			return m.handleBallFormKey(msg)
		}

		// Handle input modes (text entry)
		if m.mode == inputSessionView || m.mode == inputBallView || m.mode == inputBlockedView || m.mode == inputTagView {
			return m.handleInputKey(msg)
		}

		// Handle session selector mode
		if m.mode == sessionSelectorView {
			return m.handleSessionSelectorKey(msg)
		}

		// Handle dependency selector mode
		if m.mode == dependencySelectorView {
			return m.handleDependencySelectorKey(msg)
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

		// Handle agent cancel confirmation
		if m.mode == confirmAgentCancel {
			return m.handleAgentCancelConfirm(msg)
		}

		// Handle split help view
		if m.mode == splitHelpView {
			return m.handleSplitHelpKey(msg)
		}

		// Handle split view specific keys first
		if m.mode == splitView {
			return m.handleSplitViewKey(msg)
		}

		// Handle history view keys
		if m.mode == historyView {
			return m.handleHistoryViewKey(msg)
		}

		// Handle history output view keys
		if m.mode == historyOutputView {
			return m.handleHistoryOutputViewKey(msg)
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

		// Handle pending session select (from mode toggle)
		if m.pendingSessionSelect != "" {
			found := false
			for i, sess := range m.sessions {
				if sess.ID == m.pendingSessionSelect {
					m.selectedSession = sess
					m.sessionCursor = i
					found = true
					break
				}
			}
			if !found && len(m.sessions) > 0 {
				// Selected session not in new list, select nearest (first session)
				m.sessionCursor = 0
				m.selectedSession = m.sessions[0]
				m.addActivity("Previous session not available, selected: " + m.sessions[0].ID)
			} else if !found {
				m.selectedSession = nil
				m.sessionCursor = 0
			}
			m.pendingSessionSelect = ""
		} else if m.initialSessionID != "" && m.selectedSession == nil {
			// Pre-select session if initialSessionID was provided (initial load)
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

	case agentProcessStartedMsg:
		// Store the process reference for cancellation
		m.agentProcess = msg.process
		m.agentStatus = AgentStatus{
			Running:       true,
			SessionID:     msg.sessionID,
			Iteration:     0,
			MaxIterations: 10, // Default
		}
		m.addActivity("Agent process started for session: " + msg.sessionID)
		m.message = "Agent running... (X to cancel)"
		// Start waiting for the process completion and continue listening for output
		return m, tea.Batch(
			waitForAgentCmd(msg.process),
			listenForAgentOutput(m.agentOutputCh),
		)

	case agentCancelledMsg:
		m.agentStatus.Running = false
		m.agentProcess = nil
		// Close and nil out the output channel to prevent goroutine leaks
		if m.agentOutputCh != nil {
			close(m.agentOutputCh)
			m.agentOutputCh = nil
		}
		m.message = "Agent cancelled"
		m.addActivity("Agent cancelled for session: " + msg.sessionID)
		m.addAgentOutput("=== Agent cancelled by user ===", true)
		// Reload balls to reflect any changes made before cancellation
		return m, loadBalls(m.store, m.config, m.localOnly)

	case agentIterationMsg:
		m.agentStatus.Iteration = msg.iteration
		m.agentStatus.MaxIterations = msg.maxIter
		m.addActivity(fmt.Sprintf("Agent iteration %d/%d", msg.iteration, msg.maxIter))
		return m, nil

	case agentFinishedMsg:
		m.agentStatus.Running = false
		m.agentProcess = nil // Clear process reference
		// Close and nil out the output channel to prevent goroutine leaks
		if m.agentOutputCh != nil {
			close(m.agentOutputCh)
			m.agentOutputCh = nil
		}
		if msg.err != nil {
			m.message = "Agent error: " + msg.err.Error()
			m.addActivity("Agent error: " + msg.err.Error())
			m.addAgentOutput("=== Agent Error: "+msg.err.Error()+" ===", true)
		} else if msg.complete {
			m.message = "Agent complete!"
			m.addActivity("Agent completed: " + msg.sessionID)
			m.addAgentOutput("=== Agent completed ===", false)
		} else if msg.blocked {
			m.message = "Agent blocked: " + msg.blockedReason
			m.addActivity("Agent blocked: " + msg.blockedReason)
			m.addAgentOutput("=== Agent blocked: "+msg.blockedReason+" ===", true)
		} else {
			m.message = "Agent finished (max iterations)"
			m.addActivity("Agent finished: max iterations reached")
			m.addAgentOutput("=== Agent finished (max iterations) ===", false)
		}
		// Reload balls to reflect any changes
		return m, loadBalls(m.store, m.config, m.localOnly)

	case agentOutputMsg:
		// Add the output line to our buffer
		m.addAgentOutput(msg.line, msg.isError)
		// Continue listening for more output if agent is still running
		if m.agentStatus.Running && m.agentOutputCh != nil {
			return m, listenForAgentOutput(m.agentOutputCh)
		}
		return m, nil

	case historyLoadedMsg:
		if msg.err != nil {
			m.message = "Error loading history: " + msg.err.Error()
			m.addActivity("Error loading history: " + msg.err.Error())
			m.mode = splitView
			return m, nil
		}
		m.agentHistory = msg.history
		m.historyCursor = 0
		m.historyScrollOffset = 0
		m.mode = historyView
		m.addActivity("Loaded agent history: " + strconv.Itoa(len(msg.history)) + " runs")
		return m, nil

	case historyOutputLoadedMsg:
		if msg.err != nil {
			m.historyOutput = "Error loading output: " + msg.err.Error()
		} else {
			m.historyOutput = msg.content
		}
		m.historyOutputOffset = 0
		m.mode = historyOutputView
		return m, nil
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
// Uses two-key sequences for state changes (s+key) and toggles (t+key)
func (m Model) handleSplitViewKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Handle two-key sequences for state changes
	if m.pendingKeySequence == "s" {
		m.pendingKeySequence = ""
		return m.handleStateKeySequence(key)
	}

	// Handle two-key sequences for toggle filters
	if m.pendingKeySequence == "t" {
		m.pendingKeySequence = ""
		return m.handleToggleKeySequence(key)
	}

	// Handle two-key sequences for view columns (vc, vp, vt, va)
	if m.pendingKeySequence == "v" {
		m.pendingKeySequence = ""
		return m.handleViewColumnKeySequence(key)
	}

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
		// Route to agent output panel if visible
		if m.agentOutputVisible {
			return m.handleAgentOutputScrollUp()
		}
		return m.handleSplitViewNavUp()

	case "down", "j":
		m.message = ""
		// Route to agent output panel if visible
		if m.agentOutputVisible {
			return m.handleAgentOutputScrollDown()
		}
		return m.handleSplitViewNavDown()

	case "ctrl+d":
		// Page down in agent output panel if visible
		if m.agentOutputVisible {
			return m.handleAgentOutputPageDown()
		}
		// Page down in activity log
		if m.activePanel == ActivityPanel {
			return m.handleActivityLogPageDown()
		}
		return m, nil

	case "ctrl+u":
		// Page up in agent output panel if visible
		if m.agentOutputVisible {
			return m.handleAgentOutputPageUp()
		}
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
		// Handle gg for go to top of agent output panel if visible
		if m.agentOutputVisible {
			if m.lastKey == "g" {
				m.lastKey = ""
				return m.handleAgentOutputGoToTop()
			}
			m.lastKey = "g"
			return m, nil
		}
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
		// Go to bottom of agent output panel if visible
		if m.agentOutputVisible {
			m.lastKey = ""
			return m.handleAgentOutputGoToBottom()
		}
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
		// Start two-key sequence for state changes (sc=complete, sb=blocked, ss=start, sp=pending, sa=archive)
		if m.activePanel == BallsPanel {
			m.pendingKeySequence = "s"
			m.message = "s: State change... (c=complete, s=start, b=blocked, p=pending, a=archive)"
			return m, nil
		}
		return m, nil

	case "t":
		// Start two-key sequence for toggle filters (tc=complete, tb=blocked, ti=in_progress, tp=pending)
		m.pendingKeySequence = "t"
		m.message = "t: Toggle filter... (c=complete, b=blocked, i=in_progress, p=pending, a=all)"
		return m, nil

	case "R":
		// Refresh
		m.message = "Reloading..."
		m.addActivity("Refreshing data...")
		return m, tea.Batch(
			loadBalls(m.store, m.config, m.localOnly),
			loadSessions(m.sessionStore, m.config, m.localOnly),
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

	case "o":
		// Toggle sort order for balls
		return m.handleToggleSortOrder()

	case "v":
		// Start two-key sequence for view column toggles (vp=priority, vt=tags, vs=tests, va=all)
		if m.activePanel == BallsPanel {
			m.pendingKeySequence = "v"
			m.message = "v: View columns... (p=priority, t=tags, s=tests, a=all)"
			return m, nil
		}
		return m, nil

	case "O":
		// Toggle agent output panel visibility
		return m.handleToggleAgentOutput()

	case "E":
		// Toggle agent output panel expansion (when visible)
		if m.agentOutputVisible {
			return m.handleToggleAgentOutputExpand()
		}
		// Open editor for ball in BallsPanel
		if m.activePanel == BallsPanel {
			return m.handleBallEditInEditor()
		}
		return m, nil

	case "X":
		// Cancel running agent (with confirmation)
		return m.handleCancelAgent()

	case "H":
		// Show agent history view
		return m.handleShowHistory()
	}

	return m, nil
}

// handleStateKeySequence handles the second key in a state change sequence (s+key)
func (m Model) handleStateKeySequence(key string) (tea.Model, tea.Cmd) {
	m.message = ""

	if m.activePanel != BallsPanel {
		return m, nil
	}

	switch key {
	case "c":
		// sc = Complete ball
		return m.handleSplitCompleteBall()
	case "s":
		// ss = Start ball (set to in_progress)
		return m.handleSplitStartBall()
	case "b":
		// sb = Block ball
		return m.handleSplitBlockBall()
	case "p":
		// sp = Set to pending
		return m.handleSplitSetPending()
	case "a":
		// sa = Archive completed ball
		return m.handleSplitArchiveBall()
	case "esc":
		// Cancel sequence
		m.message = ""
		return m, nil
	default:
		m.message = "Unknown state: " + key + " (use c/s/b/p/a)"
		return m, nil
	}
}

// handleToggleKeySequence handles the second key in a toggle sequence (t+key)
func (m Model) handleToggleKeySequence(key string) (tea.Model, tea.Cmd) {
	m.message = ""

	// Track if we need to apply filters (for all filter-changing keys)
	needsFilterUpdate := true

	switch key {
	case "c":
		// tc = Toggle complete visibility
		m.filterStates["complete"] = !m.filterStates["complete"]
		if m.filterStates["complete"] {
			m.addActivity("Showing complete balls")
			m.message = "Complete: visible"
		} else {
			m.addActivity("Hiding complete balls")
			m.message = "Complete: hidden"
		}
	case "b":
		// tb = Toggle blocked visibility
		m.filterStates["blocked"] = !m.filterStates["blocked"]
		if m.filterStates["blocked"] {
			m.addActivity("Showing blocked balls")
			m.message = "Blocked: visible"
		} else {
			m.addActivity("Hiding blocked balls")
			m.message = "Blocked: hidden"
		}
	case "i":
		// ti = Toggle in_progress visibility
		m.filterStates["in_progress"] = !m.filterStates["in_progress"]
		if m.filterStates["in_progress"] {
			m.addActivity("Showing in-progress balls")
			m.message = "In-progress: visible"
		} else {
			m.addActivity("Hiding in-progress balls")
			m.message = "In-progress: hidden"
		}
	case "p":
		// tp = Toggle pending visibility
		m.filterStates["pending"] = !m.filterStates["pending"]
		if m.filterStates["pending"] {
			m.addActivity("Showing pending balls")
			m.message = "Pending: visible"
		} else {
			m.addActivity("Hiding pending balls")
			m.message = "Pending: hidden"
		}
	case "a":
		// ta = Show all states
		m.filterStates["pending"] = true
		m.filterStates["in_progress"] = true
		m.filterStates["blocked"] = true
		m.filterStates["complete"] = true
		m.addActivity("Showing all states")
		m.message = "All states visible"
	case "esc":
		// Cancel sequence
		m.message = ""
		needsFilterUpdate = false
	default:
		m.message = "Unknown toggle: " + key + " (use c/b/i/p/a)"
		needsFilterUpdate = false
	}

	// Apply filters and reset cursor for filter-changing operations
	if needsFilterUpdate {
		m.applyFilters()
		// Reset cursor if it's out of bounds
		if m.cursor >= len(m.filteredBalls) {
			m.cursor = 0
		}
	}

	return m, nil
}

// handleViewColumnKeySequence handles the second key in a view column sequence (v+key)
func (m Model) handleViewColumnKeySequence(key string) (tea.Model, tea.Cmd) {
	m.message = ""

	switch key {
	case "p":
		// vp = Toggle priority column visibility
		m.showPriorityColumn = !m.showPriorityColumn
		if m.showPriorityColumn {
			m.addActivity("Showing priority column")
			m.message = "Priority column: visible"
		} else {
			m.addActivity("Hiding priority column")
			m.message = "Priority column: hidden"
		}
		return m, nil
	case "t":
		// vt = Toggle tags column visibility
		m.showTagsColumn = !m.showTagsColumn
		if m.showTagsColumn {
			m.addActivity("Showing tags column")
			m.message = "Tags column: visible"
		} else {
			m.addActivity("Hiding tags column")
			m.message = "Tags column: hidden"
		}
		return m, nil
	case "s":
		// vs = Toggle tests state column visibility
		m.showTestsColumn = !m.showTestsColumn
		if m.showTestsColumn {
			m.addActivity("Showing tests column")
			m.message = "Tests column: visible"
		} else {
			m.addActivity("Hiding tests column")
			m.message = "Tests column: hidden"
		}
		return m, nil
	case "a":
		// va = Toggle all columns visibility
		allVisible := m.showPriorityColumn && m.showTagsColumn && m.showTestsColumn
		if allVisible {
			// Hide all
			m.showPriorityColumn = false
			m.showTagsColumn = false
			m.showTestsColumn = false
			m.addActivity("Hiding all optional columns")
			m.message = "All columns: hidden"
		} else {
			// Show all
			m.showPriorityColumn = true
			m.showTagsColumn = true
			m.showTestsColumn = true
			m.addActivity("Showing all optional columns")
			m.message = "All columns: visible"
		}
		return m, nil
	case "esc":
		// Cancel sequence
		m.message = ""
		return m, nil
	default:
		m.message = "Unknown view column: " + key + " (use p/t/s/a)"
		return m, nil
	}
}

// handleSplitSetPending sets the selected ball to pending state
func (m Model) handleSplitSetPending() (tea.Model, tea.Cmd) {
	balls := m.filterBallsForSession()
	if len(balls) == 0 || m.cursor >= len(balls) {
		return m, nil
	}

	ball := balls[m.cursor]
	ball.SetState(session.StatePending)
	m.addActivity("Set pending: " + ball.ID)

	store, err := session.NewStore(ball.WorkingDir)
	if err != nil {
		m.message = "Error: " + err.Error()
		return m, nil
	}

	return m, updateBall(store, ball)
}

// handleSplitArchiveBall archives a completed ball
func (m Model) handleSplitArchiveBall() (tea.Model, tea.Cmd) {
	balls := m.filterBallsForSession()
	if len(balls) == 0 || m.cursor >= len(balls) {
		return m, nil
	}

	ball := balls[m.cursor]

	// Only archive completed balls
	if ball.State != session.StateComplete {
		m.message = "Can only archive completed balls (use sc first)"
		return m, nil
	}

	store, err := session.NewStore(ball.WorkingDir)
	if err != nil {
		m.message = "Error: " + err.Error()
		return m, nil
	}

	m.addActivity("Archiving ball: " + ball.ID)
	return m, archiveBall(store, ball)
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
			m.ballsScrollOffset = 0 // Reset balls scroll offset for new session
		} else if m.sessionCursor >= len(sessions) && len(sessions) > 0 {
			m.sessionCursor = len(sessions) - 1
			m.selectedSession = sessions[m.sessionCursor]
			m.cursor = 0
			m.ballsScrollOffset = 0
		}
	case BallsPanel:
		if m.cursor > 0 {
			m.cursor--
			// Adjust scroll offset to keep cursor visible
			balls := m.filterBallsForSession()
			m.adjustBallsScrollOffset(balls)
		}
	case ActivityPanel:
		// Scroll based on bottom pane mode
		if m.bottomPaneMode == BottomPaneDetail {
			// Scroll up in detail view
			if m.detailScrollOffset > 0 {
				m.detailScrollOffset--
			}
		} else {
			// Scroll up one line in activity log
			if m.activityLogOffset > 0 {
				m.activityLogOffset--
			}
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
			m.ballsScrollOffset = 0 // Reset balls scroll offset for new session
		}
	case BallsPanel:
		balls := m.filterBallsForSession()
		if m.cursor < len(balls)-1 {
			m.cursor++
			// Adjust scroll offset to keep cursor visible
			m.adjustBallsScrollOffset(balls)
		}
	case ActivityPanel:
		// Scroll based on bottom pane mode
		if m.bottomPaneMode == BottomPaneDetail {
			// Scroll down in detail view
			m.detailScrollOffset++
			// The max offset will be clamped in the render function
		} else {
			// Scroll down one line in activity log
			maxOffset := m.getActivityLogMaxOffset()
			if m.activityLogOffset < maxOffset {
				m.activityLogOffset++
			}
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
		m.ballsScrollOffset = 0 // Reset balls scroll offset for new session
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
		m.ballsScrollOffset = 0 // Reset balls scroll offset for new session
		m.addActivity("Switched to session: " + m.selectedSession.ID)
	}
	return m, nil
}

// handleToggleBottomPane cycles through activity log, ball detail, and split view
func (m Model) handleToggleBottomPane() (tea.Model, tea.Cmd) {
	switch m.bottomPaneMode {
	case BottomPaneActivity:
		m.bottomPaneMode = BottomPaneDetail
		m.detailScrollOffset = 0 // Reset scroll on mode change
		m.addActivity("Showing ball details in bottom pane")
	case BottomPaneDetail:
		m.bottomPaneMode = BottomPaneSplit
		m.addActivity("Showing split view (details + activity)")
	case BottomPaneSplit:
		m.bottomPaneMode = BottomPaneActivity
		m.addActivity("Showing activity log in bottom pane")
	}
	return m, nil
}

// handleToggleLocalOnly toggles between local project only and all projects
func (m Model) handleToggleLocalOnly() (tea.Model, tea.Cmd) {
	// Remember the currently selected session ID before reloading
	selectedSessionID := ""
	if m.selectedSession != nil {
		selectedSessionID = m.selectedSession.ID
	}

	m.localOnly = !m.localOnly
	if m.localOnly {
		m.addActivity("Showing local project only")
		m.message = "Showing local project only"
	} else {
		m.addActivity("Showing all projects")
		m.message = "Showing all projects"
	}

	// Store the selected session ID for cursor adjustment after reload
	m.pendingSessionSelect = selectedSessionID

	// Reload both balls and sessions with new scope
	return m, tea.Batch(
		loadBalls(m.store, m.config, m.localOnly),
		loadSessions(m.sessionStore, m.config, m.localOnly),
	)
}

// handleToggleSortOrder cycles through sort orders for balls
func (m Model) handleToggleSortOrder() (tea.Model, tea.Cmd) {
	// Cycle through sort orders
	switch m.sortOrder {
	case SortByIDASC:
		m.sortOrder = SortByIDDESC
		m.addActivity("Sort: ID descending")
		m.message = "Sort: ID descending"
	case SortByIDDESC:
		m.sortOrder = SortByPriority
		m.addActivity("Sort: Priority")
		m.message = "Sort: Priority (urgent first)"
	case SortByPriority:
		m.sortOrder = SortByLastActivity
		m.addActivity("Sort: Last activity")
		m.message = "Sort: Last activity (recent first)"
	case SortByLastActivity:
		m.sortOrder = SortByIDASC
		m.addActivity("Sort: ID ascending")
		m.message = "Sort: ID ascending"
	}
	return m, nil
}

// handleToggleAgentOutput toggles the agent output panel visibility
func (m Model) handleToggleAgentOutput() (tea.Model, tea.Cmd) {
	m.agentOutputVisible = !m.agentOutputVisible
	if m.agentOutputVisible {
		m.addActivity("Agent output panel shown")
		m.message = "Agent output visible (O to hide, E to expand)"
	} else {
		m.addActivity("Agent output panel hidden")
		m.message = "Agent output hidden (O to show)"
	}
	return m, nil
}

// handleToggleAgentOutputExpand toggles the agent output panel between normal and expanded sizes
func (m Model) handleToggleAgentOutputExpand() (tea.Model, tea.Cmd) {
	if !m.agentOutputVisible {
		// Can't expand if not visible
		return m, nil
	}
	m.agentOutputExpanded = !m.agentOutputExpanded
	if m.agentOutputExpanded {
		m.addActivity("Agent output panel expanded")
		m.message = "Agent output expanded (E to collapse)"
	} else {
		m.addActivity("Agent output panel collapsed")
		m.message = "Agent output collapsed (E to expand)"
	}
	return m, nil
}

// handleAgentOutputScroll handles vim-style scrolling for the agent output panel
func (m Model) handleAgentOutputScrollUp() (tea.Model, tea.Cmd) {
	if m.agentOutputOffset > 0 {
		m.agentOutputOffset--
	}
	return m, nil
}

func (m Model) handleAgentOutputScrollDown() (tea.Model, tea.Cmd) {
	maxOffset := m.getAgentOutputMaxOffset()
	if m.agentOutputOffset < maxOffset {
		m.agentOutputOffset++
	}
	return m, nil
}

func (m Model) handleAgentOutputPageUp() (tea.Model, tea.Cmd) {
	pageSize := m.getAgentOutputVisibleLines() / 2
	if pageSize < 1 {
		pageSize = 1
	}
	m.agentOutputOffset -= pageSize
	if m.agentOutputOffset < 0 {
		m.agentOutputOffset = 0
	}
	return m, nil
}

func (m Model) handleAgentOutputPageDown() (tea.Model, tea.Cmd) {
	pageSize := m.getAgentOutputVisibleLines() / 2
	if pageSize < 1 {
		pageSize = 1
	}
	maxOffset := m.getAgentOutputMaxOffset()
	m.agentOutputOffset += pageSize
	if m.agentOutputOffset > maxOffset {
		m.agentOutputOffset = maxOffset
	}
	return m, nil
}

func (m Model) handleAgentOutputGoToTop() (tea.Model, tea.Cmd) {
	m.agentOutputOffset = 0
	return m, nil
}

func (m Model) handleAgentOutputGoToBottom() (tea.Model, tea.Cmd) {
	m.agentOutputOffset = m.getAgentOutputMaxOffset()
	return m, nil
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

// adjustBallsScrollOffset adjusts the scroll offset to keep the cursor visible
func (m *Model) adjustBallsScrollOffset(balls []*session.Ball) {
	if len(balls) == 0 {
		m.ballsScrollOffset = 0
		return
	}

	// Calculate visible lines in balls panel
	// This should match the calculation in renderBallsPanel
	// mainHeight = m.height - bottomPanelRows - 4
	// ballsHeight = (mainHeight - 2) - 4
	mainHeight := m.height - bottomPanelRows - 4
	ballsHeight := mainHeight - 2 - 4
	if ballsHeight < 1 {
		ballsHeight = 1
	}

	// If cursor is above the visible area, scroll up
	if m.cursor < m.ballsScrollOffset {
		m.ballsScrollOffset = m.cursor
	}

	// Calculate visible area accounting for top scroll indicator
	// When scrollOffset > 0, we show a top indicator taking one line
	visibleArea := ballsHeight
	if m.ballsScrollOffset > 0 {
		visibleArea-- // One line used by "↑ N more items above"
	}
	if visibleArea < 1 {
		visibleArea = 1
	}

	// If cursor is below the visible area, scroll down
	// Important: after scrolling down, we'll have a top indicator (unless we scroll to 0)
	// So use (ballsHeight - 1) as the visible area for the new offset
	if m.cursor >= m.ballsScrollOffset+visibleArea {
		// Calculate what the new offset should be to make cursor visible
		// After scrolling, we'll have a top indicator, so visible = ballsHeight - 1
		newVisibleArea := ballsHeight - 1
		if newVisibleArea < 1 {
			newVisibleArea = 1
		}
		m.ballsScrollOffset = m.cursor - newVisibleArea + 1
	}

	// Ensure scroll offset is within valid bounds
	// When scrolled down (offset > 0), we lose one line to the top indicator
	// So maxOffset is calculated to ensure all items can be seen when scrolled to max
	// At maxOffset, we can show (ballsHeight - 1) items due to top indicator
	maxOffset := len(balls) - (ballsHeight - 1)
	if maxOffset < 0 {
		maxOffset = 0
	}
	// Special case: if all balls fit without scrolling, maxOffset should be 0
	if len(balls) <= ballsHeight {
		maxOffset = 0
	}
	if m.ballsScrollOffset > maxOffset {
		m.ballsScrollOffset = maxOffset
	}
	if m.ballsScrollOffset < 0 {
		m.ballsScrollOffset = 0
	}
}

// handleActivityLogPageDown scrolls down half a page in the activity log (or detail view)
func (m Model) handleActivityLogPageDown() (tea.Model, tea.Cmd) {
	m.lastKey = "" // Clear gg state
	pageSize := (bottomPanelRows - 3) / 2
	if pageSize < 1 {
		pageSize = 1
	}
	if m.bottomPaneMode == BottomPaneDetail {
		// Scroll detail view
		m.detailScrollOffset += pageSize
		// Will be clamped in render
	} else {
		maxOffset := m.getActivityLogMaxOffset()
		m.activityLogOffset += pageSize
		if m.activityLogOffset > maxOffset {
			m.activityLogOffset = maxOffset
		}
	}
	return m, nil
}

// handleActivityLogPageUp scrolls up half a page in the activity log (or detail view)
func (m Model) handleActivityLogPageUp() (tea.Model, tea.Cmd) {
	m.lastKey = "" // Clear gg state
	pageSize := (bottomPanelRows - 3) / 2
	if pageSize < 1 {
		pageSize = 1
	}
	if m.bottomPaneMode == BottomPaneDetail {
		// Scroll detail view
		m.detailScrollOffset -= pageSize
		if m.detailScrollOffset < 0 {
			m.detailScrollOffset = 0
		}
	} else {
		m.activityLogOffset -= pageSize
		if m.activityLogOffset < 0 {
			m.activityLogOffset = 0
		}
	}
	return m, nil
}

// handleActivityLogGoToTop scrolls to the top of the activity log (or detail view)
func (m Model) handleActivityLogGoToTop() (tea.Model, tea.Cmd) {
	if m.bottomPaneMode == BottomPaneDetail {
		m.detailScrollOffset = 0
	} else {
		m.activityLogOffset = 0
	}
	return m, nil
}

// handleActivityLogGoToBottom scrolls to the bottom of the activity log (or detail view)
func (m Model) handleActivityLogGoToBottom() (tea.Model, tea.Cmd) {
	if m.bottomPaneMode == BottomPaneDetail {
		// Set to large number, will be clamped in render
		m.detailScrollOffset = 1000
	} else {
		m.activityLogOffset = m.getActivityLogMaxOffset()
	}
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
			m.ballsScrollOffset = 0 // Reset balls scroll offset for new session
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
		// Use unified ball form - all fields in one view
		m.pendingBallIntent = ""
		m.pendingBallContext = ""
		m.pendingBallPriority = 1 // Default to medium
		m.pendingBallTags = ""
		m.pendingAcceptanceCriteria = []string{}
		m.pendingACEditIndex = -1
		m.pendingBallDependsOn = nil
		m.pendingBallModelSize = 0
		// Initialize file autocomplete for @ mentions
		if m.store != nil {
			m.fileAutocomplete = NewAutocompleteState(m.store.ProjectDir())
		}
		// Default session to currently selected one (if a real session is selected)
		m.pendingBallSession = 0 // Start with (none)
		if m.selectedSession != nil && m.selectedSession.ID != PseudoSessionAll && m.selectedSession.ID != PseudoSessionUntagged {
			// Find the index of the selected session in real sessions
			realSessionIdx := 0
			for _, sess := range m.sessions {
				if sess.ID == PseudoSessionAll || sess.ID == PseudoSessionUntagged {
					continue
				}
				realSessionIdx++
				if sess.ID == m.selectedSession.ID {
					m.pendingBallSession = realSessionIdx
					break
				}
			}
		}
		m.pendingBallFormField = 0 // Start at context field
		m.textInput.Placeholder = "Background context for this task"
		m.mode = unifiedBallFormView
		m.addActivity("Creating new ball...")
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

		// Initialize file autocomplete for @ mentions
		if m.store != nil {
			m.fileAutocomplete = NewAutocompleteState(m.store.ProjectDir())
		}

		// Use unified ball form with prepopulated fields
		m.pendingBallContext = ball.Context
		m.pendingBallIntent = ball.Title
		// Convert priority to index (low=0, medium=1, high=2, urgent=3)
		switch ball.Priority {
		case session.PriorityLow:
			m.pendingBallPriority = 0
		case session.PriorityMedium:
			m.pendingBallPriority = 1
		case session.PriorityHigh:
			m.pendingBallPriority = 2
		case session.PriorityUrgent:
			m.pendingBallPriority = 3
		default:
			m.pendingBallPriority = 1 // Default to medium
		}
		m.pendingBallTags = strings.Join(ball.Tags, ", ")
		m.pendingAcceptanceCriteria = make([]string, len(ball.AcceptanceCriteria))
		copy(m.pendingAcceptanceCriteria, ball.AcceptanceCriteria)
		m.pendingACEditIndex = -1
		m.pendingBallDependsOn = make([]string, len(ball.DependsOn))
		copy(m.pendingBallDependsOn, ball.DependsOn)

		// Convert model size to index (blank=0, small=1, medium=2, large=3)
		switch ball.ModelSize {
		case session.ModelSizeSmall:
			m.pendingBallModelSize = 1
		case session.ModelSizeMedium:
			m.pendingBallModelSize = 2
		case session.ModelSizeLarge:
			m.pendingBallModelSize = 3
		default:
			m.pendingBallModelSize = 0 // Default
		}

		// Find session index from tags (first tag that matches a session)
		m.pendingBallSession = 0 // Default to (none)
		for _, tag := range ball.Tags {
			realSessionIdx := 0
			for _, sess := range m.sessions {
				if sess.ID == PseudoSessionAll || sess.ID == PseudoSessionUntagged {
					continue
				}
				realSessionIdx++
				if sess.ID == tag {
					m.pendingBallSession = realSessionIdx
					break
				}
			}
			if m.pendingBallSession > 0 {
				break // Found a session match
			}
		}

		m.pendingBallFormField = 0 // Start at context field
		m.textInput.SetValue(ball.Context)
		m.textInput.Placeholder = "Background context for this task"
		m.mode = unifiedBallFormView
		m.addActivity("Editing ball: " + ball.ID)
	}

	return m, nil
}

// handleBallEditInEditor opens the selected ball in an external editor (E key)
func (m Model) handleBallEditInEditor() (tea.Model, tea.Cmd) {
	if m.activePanel != BallsPanel {
		return m, nil
	}

	balls := m.filterBallsForSession()
	if len(balls) == 0 || m.cursor >= len(balls) {
		m.message = "No ball selected"
		return m, nil
	}

	ball := balls[m.cursor]
	m.editingBall = ball
	m.inputAction = actionEdit
	m.addActivity("Opening editor for: " + ball.ID)
	return m, openEditorCmd(ball)
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
	return m, loadSessions(m.sessionStore, m.config, m.localOnly)
}

// submitBallInput handles ball add/edit submission
func (m Model) submitBallInput(value string) (tea.Model, tea.Cmd) {
	if m.inputAction == actionAdd {
		// Store the intent and transition to ball form view
		m.pendingBallIntent = value
		m.pendingAcceptanceCriteria = []string{}
		m.pendingBallPriority = 1 // Default to medium (index 1)
		m.pendingBallTags = ""
		// Default session to currently selected one (if a real session is selected)
		m.pendingBallSession = 0 // Start with (none)
		if m.selectedSession != nil && m.selectedSession.ID != PseudoSessionAll && m.selectedSession.ID != PseudoSessionUntagged {
			// Find the index of the selected session in real sessions
			realSessionIdx := 0
			for _, sess := range m.sessions {
				if sess.ID == PseudoSessionAll || sess.ID == PseudoSessionUntagged {
					continue
				}
				realSessionIdx++
				if sess.ID == m.selectedSession.ID {
					m.pendingBallSession = realSessionIdx
					break
				}
			}
		}
		m.pendingBallFormField = 0
		m.textInput.Reset()
		m.textInput.Placeholder = "tag1, tag2, ..."
		m.textInput.Blur() // Start with selection fields, not text
		m.mode = inputBallFormView
		m.addActivity("Configure ball properties")
		return m, nil
	} else {
		// Edit ball intent
		if m.editingBall == nil {
			m.mode = splitView
			return m, nil
		}
		m.editingBall.Title = value
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

// handleAcceptanceCriteriaKey handles keyboard input for multi-line acceptance criteria entry
func (m Model) handleAcceptanceCriteriaKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Cancel input - clear pending state
		m.pendingBallIntent = ""
		m.pendingAcceptanceCriteria = nil
		m.mode = splitView
		m.message = "Cancelled"
		m.textInput.Blur()
		return m, nil

	case "enter":
		value := strings.TrimSpace(m.textInput.Value())
		if value == "" {
			// Empty line - create the ball with collected criteria
			return m.finalizeBallCreation()
		}
		// Non-empty - add to list and continue
		m.pendingAcceptanceCriteria = append(m.pendingAcceptanceCriteria, value)
		m.textInput.Reset()
		m.addActivity("Added AC: " + truncate(value, 30))
		return m, nil

	default:
		// Pass to textinput
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}
}

// handleBallFormKey handles keyboard input for the multi-field ball form
func (m Model) handleBallFormKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Field indices: 0=priority, 1=tags, 2=session (state removed - always pending)
	const (
		fieldPriority = 0
		fieldTags     = 1
		fieldSession  = 2
		numFields     = 3
	)

	// Number of options for each selection field
	numPriorityOptions := 4 // low, medium, high, urgent

	// Count real sessions (excluding pseudo-sessions)
	numSessionOptions := 1 // Start with "(none)"
	for _, sess := range m.sessions {
		if sess.ID != PseudoSessionAll && sess.ID != PseudoSessionUntagged {
			numSessionOptions++
		}
	}

	switch msg.String() {
	case "esc":
		// Cancel input - clear pending state
		m.pendingBallIntent = ""
		m.pendingAcceptanceCriteria = nil
		m.pendingBallTags = ""
		m.mode = splitView
		m.message = "Cancelled"
		m.textInput.Blur()
		return m, nil

	case "enter":
		// Save tags value if on tags field
		if m.pendingBallFormField == fieldTags {
			m.pendingBallTags = strings.TrimSpace(m.textInput.Value())
		}
		// Proceed to acceptance criteria entry
		m.textInput.Reset()
		m.textInput.Placeholder = "Acceptance criterion (empty to finish)"
		m.textInput.Focus()
		m.mode = inputAcceptanceCriteriaView
		m.addActivity("Enter acceptance criteria (empty line to finish)")
		return m, nil

	case "up", "k":
		// Move to previous field
		if m.pendingBallFormField == fieldTags {
			// Save current tags value before moving
			m.pendingBallTags = strings.TrimSpace(m.textInput.Value())
			m.textInput.Blur()
		}
		m.pendingBallFormField--
		if m.pendingBallFormField < 0 {
			m.pendingBallFormField = numFields - 1
		}
		// Focus text input if on tags field
		if m.pendingBallFormField == fieldTags {
			m.textInput.SetValue(m.pendingBallTags)
			m.textInput.Focus()
		}
		return m, nil

	case "down", "j":
		// Move to next field
		if m.pendingBallFormField == fieldTags {
			// Save current tags value before moving
			m.pendingBallTags = strings.TrimSpace(m.textInput.Value())
			m.textInput.Blur()
		}
		m.pendingBallFormField++
		if m.pendingBallFormField >= numFields {
			m.pendingBallFormField = 0
		}
		// Focus text input if on tags field
		if m.pendingBallFormField == fieldTags {
			m.textInput.SetValue(m.pendingBallTags)
			m.textInput.Focus()
		}
		return m, nil

	case "left", "h":
		// Cycle selection left (for non-text fields)
		switch m.pendingBallFormField {
		case fieldPriority:
			m.pendingBallPriority--
			if m.pendingBallPriority < 0 {
				m.pendingBallPriority = numPriorityOptions - 1
			}
		case fieldSession:
			m.pendingBallSession--
			if m.pendingBallSession < 0 {
				m.pendingBallSession = numSessionOptions - 1
			}
		}
		return m, nil

	case "right", "l", "tab":
		// Cycle selection right (for non-text fields)
		switch m.pendingBallFormField {
		case fieldPriority:
			m.pendingBallPriority++
			if m.pendingBallPriority >= numPriorityOptions {
				m.pendingBallPriority = 0
			}
		case fieldSession:
			m.pendingBallSession++
			if m.pendingBallSession >= numSessionOptions {
				m.pendingBallSession = 0
			}
		}
		return m, nil

	default:
		// Pass to textinput only if on tags field
		if m.pendingBallFormField == fieldTags {
			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd
		}
		return m, nil
	}
}

// finalizeBallCreation creates the ball with the collected intent and acceptance criteria
func (m Model) finalizeBallCreation() (tea.Model, tea.Cmd) {
	// Map priority index to Priority constant
	priorities := []session.Priority{session.PriorityLow, session.PriorityMedium, session.PriorityHigh, session.PriorityUrgent}
	priority := priorities[m.pendingBallPriority]

	// Map model size index to ModelSize constant
	modelSizes := []session.ModelSize{session.ModelSizeBlank, session.ModelSizeSmall, session.ModelSizeMedium, session.ModelSizeLarge}
	modelSize := modelSizes[m.pendingBallModelSize]

	// Build tags list
	var tags []string
	if m.pendingBallTags != "" {
		tagList := strings.Split(m.pendingBallTags, ",")
		for _, tag := range tagList {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				tags = append(tags, tag)
			}
		}
	}

	// Add session tag if selected in form (0 = none, 1+ = session index)
	if m.pendingBallSession > 0 {
		// Get real sessions (excluding pseudo-sessions)
		realSessions := []*session.JuggleSession{}
		for _, sess := range m.sessions {
			if sess.ID != PseudoSessionAll && sess.ID != PseudoSessionUntagged {
				realSessions = append(realSessions, sess)
			}
		}
		if m.pendingBallSession-1 < len(realSessions) {
			tags = append(tags, realSessions[m.pendingBallSession-1].ID)
		}
	}

	// Check if we're editing an existing ball or creating a new one
	if m.inputAction == actionEdit && m.editingBall != nil {
		// Update existing ball
		ball := m.editingBall
		ball.Context = m.pendingBallContext
		ball.Title = m.pendingBallIntent
		ball.Priority = priority
		ball.Tags = tags
		ball.ModelSize = modelSize

		// Set acceptance criteria
		if len(m.pendingAcceptanceCriteria) > 0 {
			ball.SetAcceptanceCriteria(m.pendingAcceptanceCriteria)
		} else {
			ball.AcceptanceCriteria = nil
		}

		// Set dependencies
		if len(m.pendingBallDependsOn) > 0 {
			ball.SetDependencies(m.pendingBallDependsOn)
		} else {
			ball.DependsOn = nil
		}

		// Update the ball in store
		err := m.store.UpdateBall(ball)
		if err != nil {
			m.message = "Error updating ball: " + err.Error()
			m.clearPendingBallState()
			m.mode = splitView
			return m, nil
		}

		m.addActivity("Updated ball: " + ball.ID)
		m.message = "Updated ball: " + ball.ID

		// Clear editing state
		m.editingBall = nil
	} else {
		// Create new ball using the store's project directory
		ball, err := session.NewBall(m.store.ProjectDir(), m.pendingBallIntent, priority)
		if err != nil {
			m.message = "Error creating ball: " + err.Error()
			m.clearPendingBallState()
			m.mode = splitView
			return m, nil
		}

		// New balls always start in pending state
		ball.State = session.StatePending
		ball.Context = m.pendingBallContext // Set context from form
		ball.Tags = tags
		ball.ModelSize = modelSize

		// Set acceptance criteria if any were collected
		if len(m.pendingAcceptanceCriteria) > 0 {
			ball.SetAcceptanceCriteria(m.pendingAcceptanceCriteria)
		}

		// Set dependencies if any were selected
		if len(m.pendingBallDependsOn) > 0 {
			ball.SetDependencies(m.pendingBallDependsOn)
		}

		// Use the store's working directory
		err = m.store.AppendBall(ball)
		if err != nil {
			m.message = "Error creating ball: " + err.Error()
			m.clearPendingBallState()
			m.mode = splitView
			return m, nil
		}

		m.addActivity("Created ball: " + ball.ID)
		m.message = "Created ball: " + ball.ID
	}

	// Clear pending state
	m.clearPendingBallState()
	m.textInput.Blur()
	m.mode = splitView

	return m, loadBalls(m.store, m.config, m.localOnly)
}

// clearPendingBallState clears all pending ball creation/editing state
func (m *Model) clearPendingBallState() {
	m.pendingBallContext = ""
	m.pendingBallIntent = ""
	m.pendingAcceptanceCriteria = nil
	m.pendingBallPriority = 1   // Reset to default (medium)
	m.pendingBallModelSize = 0  // Reset to default
	m.pendingBallTags = ""
	m.pendingBallSession = 0
	m.pendingBallDependsOn = nil
	m.pendingBallFormField = 0
	m.pendingACEditIndex = -1
	m.dependencySelectBalls = nil
	m.dependencySelectIndex = 0
	m.dependencySelectActive = nil
	m.editingBall = nil
	m.inputAction = actionAdd // Reset to default action
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
		return m, loadSessions(m.sessionStore, m.config, m.localOnly)

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

// filterBallsForSession returns balls filtered by session and search query, sorted by current sort order
func (m *Model) filterBallsForSession() []*session.Ball {
	balls := m.getBallsForSession()

	var result []*session.Ball
	if !m.panelSearchActive || m.panelSearchQuery == "" {
		result = balls
	} else {
		query := strings.ToLower(m.panelSearchQuery)
		filtered := make([]*session.Ball, 0)
		for _, ball := range balls {
			if strings.Contains(strings.ToLower(ball.Title), query) ||
				strings.Contains(strings.ToLower(ball.ID), query) {
				filtered = append(filtered, ball)
			}
		}
		result = filtered
	}

	// Apply sorting
	m.sortBalls(result)
	return result
}

// sortBalls sorts a slice of balls according to the current sort order
func (m *Model) sortBalls(balls []*session.Ball) {
	switch m.sortOrder {
	case SortByIDASC:
		sort.Slice(balls, func(i, j int) bool {
			return compareBallIDs(balls[i].ID, balls[j].ID) < 0
		})
	case SortByIDDESC:
		sort.Slice(balls, func(i, j int) bool {
			return compareBallIDs(balls[i].ID, balls[j].ID) > 0
		})
	case SortByPriority:
		sort.Slice(balls, func(i, j int) bool {
			// Higher priority first
			if balls[i].PriorityWeight() != balls[j].PriorityWeight() {
				return balls[i].PriorityWeight() > balls[j].PriorityWeight()
			}
			// Then by ID ascending
			return compareBallIDs(balls[i].ID, balls[j].ID) < 0
		})
	case SortByLastActivity:
		sort.Slice(balls, func(i, j int) bool {
			// More recent first
			return balls[i].LastActivity.After(balls[j].LastActivity)
		})
	}
}

// compareBallIDs compares two ball IDs numerically
// IDs are in format "project-N" where N is a number
func compareBallIDs(id1, id2 string) int {
	// Extract numeric parts for comparison
	num1 := extractBallNumber(id1)
	num2 := extractBallNumber(id2)

	// If both have numeric parts, compare numerically first
	if num1 != -1 && num2 != -1 {
		if num1 < num2 {
			return -1
		} else if num1 > num2 {
			return 1
		}
		// Numbers are equal, fall through to string comparison to compare prefixes
	}

	// Fall back to string comparison (also handles same numbers with different prefixes)
	if id1 < id2 {
		return -1
	} else if id1 > id2 {
		return 1
	}
	return 0
}

// extractBallNumber extracts the numeric suffix from a ball ID
// Returns -1 if no numeric suffix is found
func extractBallNumber(id string) int {
	lastHyphen := strings.LastIndex(id, "-")
	if lastHyphen >= 0 && lastHyphen < len(id)-1 {
		numStr := id[lastHyphen+1:]
		num, err := strconv.Atoi(numStr)
		if err == nil {
			return num
		}
	}
	return -1
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
		cmds = append(cmds, loadSessions(m.sessionStore, m.config, m.localOnly))

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

	// Prevent launching on untagged pseudo-session (but allow "All" which maps to meta-session "all")
	if m.selectedSession.ID == PseudoSessionUntagged {
		m.message = "Cannot launch agent on untagged session"
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
		// Map PseudoSessionAll to "all" meta-session for agent command
		if sessionID == PseudoSessionAll {
			sessionID = "all"
		}
		m.mode = splitView
		m.addActivity("Launching agent for: " + sessionID)
		m.message = "Starting agent for: " + sessionID

		// Clear previous output and create new output channel
		m.clearAgentOutput()
		m.agentOutputCh = make(chan agentOutputMsg, 100)
		m.agentOutputVisible = true // Auto-show agent output when launching
		m.addAgentOutput("=== Starting agent for session: "+sessionID+" ===", false)

		// Set initial agent status
		m.agentStatus = AgentStatus{
			Running:       true,
			SessionID:     sessionID,
			Iteration:     0,
			MaxIterations: 10, // Default iterations
		}

		// Launch agent in background with output streaming
		return m, tea.Batch(
			launchAgentWithOutputCmd(sessionID, m.agentOutputCh),
			listenForAgentOutput(m.agentOutputCh),
		)

	case "n", "N", "esc", "q":
		// Cancel
		m.mode = splitView
		m.message = "Agent launch cancelled"
		return m, nil
	}

	return m, nil
}

// handleCancelAgent shows confirmation dialog for cancelling a running agent
func (m Model) handleCancelAgent() (tea.Model, tea.Cmd) {
	// Check if agent is running
	if !m.agentStatus.Running {
		m.message = "No agent is running"
		return m, nil
	}

	// Show confirmation dialog
	m.mode = confirmAgentCancel
	return m, nil
}

// handleAgentCancelConfirm handles the agent cancel confirmation
func (m Model) handleAgentCancelConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		// Confirm cancellation
		m.mode = splitView
		m.addActivity("Cancelling agent...")
		m.message = "Cancelling agent..."

		// Kill the process if we have a reference
		if m.agentProcess != nil {
			if err := m.agentProcess.Kill(); err != nil {
				m.addActivity("Error killing agent: " + err.Error())
				m.message = "Error killing agent: " + err.Error()
			} else {
				m.addActivity("Agent process terminated")
				m.addAgentOutput("=== Agent cancelled by user ===", true)
			}
		}

		// Clear agent status
		m.agentStatus.Running = false
		m.agentProcess = nil
		m.message = "Agent cancelled"

		// Reload balls to reflect any changes made before cancellation
		return m, loadBalls(m.store, m.config, m.localOnly)

	case "n", "N", "esc", "q":
		// Don't cancel
		m.mode = splitView
		m.message = "Agent still running"
		return m, nil
	}

	return m, nil
}

// handleShowHistory loads and displays agent run history
func (m Model) handleShowHistory() (tea.Model, tea.Cmd) {
	m.addActivity("Loading agent history...")
	m.message = "Loading history..."
	return m, loadAgentHistory(m.store.ProjectDir())
}

// handleHistoryViewKey handles keyboard input in history view
func (m Model) handleHistoryViewKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc", "H":
		// Return to split view
		m.mode = splitView
		m.message = ""
		return m, nil

	case "up", "k":
		// Move cursor up
		if m.historyCursor > 0 {
			m.historyCursor--
			// Adjust scroll if needed
			if m.historyCursor < m.historyScrollOffset {
				m.historyScrollOffset = m.historyCursor
			}
		}
		return m, nil

	case "down", "j":
		// Move cursor down
		if m.historyCursor < len(m.agentHistory)-1 {
			m.historyCursor++
			// Adjust scroll if needed (assuming 15 visible lines)
			visibleLines := 15
			if m.historyCursor >= m.historyScrollOffset+visibleLines {
				m.historyScrollOffset = m.historyCursor - visibleLines + 1
			}
		}
		return m, nil

	case "ctrl+d":
		// Page down
		pageSize := 7
		m.historyCursor += pageSize
		if m.historyCursor >= len(m.agentHistory) {
			m.historyCursor = len(m.agentHistory) - 1
		}
		if m.historyCursor < 0 {
			m.historyCursor = 0
		}
		// Adjust scroll
		visibleLines := 15
		if m.historyCursor >= m.historyScrollOffset+visibleLines {
			m.historyScrollOffset = m.historyCursor - visibleLines + 1
		}
		return m, nil

	case "ctrl+u":
		// Page up
		pageSize := 7
		m.historyCursor -= pageSize
		if m.historyCursor < 0 {
			m.historyCursor = 0
		}
		// Adjust scroll
		if m.historyCursor < m.historyScrollOffset {
			m.historyScrollOffset = m.historyCursor
		}
		return m, nil

	case "g":
		// Handle gg for go to top
		if m.lastKey == "g" {
			m.lastKey = ""
			m.historyCursor = 0
			m.historyScrollOffset = 0
			return m, nil
		}
		m.lastKey = "g"
		return m, nil

	case "G":
		// Go to bottom
		m.lastKey = ""
		if len(m.agentHistory) > 0 {
			m.historyCursor = len(m.agentHistory) - 1
			// Adjust scroll to show cursor at bottom
			visibleLines := 15
			if m.historyCursor >= visibleLines {
				m.historyScrollOffset = m.historyCursor - visibleLines + 1
			}
		}
		return m, nil

	case "enter", " ":
		// View output file for selected record
		if len(m.agentHistory) > 0 && m.historyCursor < len(m.agentHistory) {
			record := m.agentHistory[m.historyCursor]
			m.addActivity("Loading output for run: " + record.ID)
			return m, loadHistoryOutput(record.OutputFile)
		}
		return m, nil
	}

	// Reset gg detection for any other key
	m.lastKey = ""
	return m, nil
}

// handleHistoryOutputViewKey handles keyboard input in history output view
func (m Model) handleHistoryOutputViewKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc", "b":
		// Return to history view
		m.mode = historyView
		m.historyOutput = ""
		return m, nil

	case "up", "k":
		// Scroll up
		if m.historyOutputOffset > 0 {
			m.historyOutputOffset--
		}
		return m, nil

	case "down", "j":
		// Scroll down
		m.historyOutputOffset++
		return m, nil

	case "ctrl+d":
		// Page down
		m.historyOutputOffset += 15
		return m, nil

	case "ctrl+u":
		// Page up
		m.historyOutputOffset -= 15
		if m.historyOutputOffset < 0 {
			m.historyOutputOffset = 0
		}
		return m, nil

	case "g":
		// Handle gg for go to top
		if m.lastKey == "g" {
			m.lastKey = ""
			m.historyOutputOffset = 0
			return m, nil
		}
		m.lastKey = "g"
		return m, nil

	case "G":
		// Go to bottom (set to large value, will be clamped in render)
		m.lastKey = ""
		m.historyOutputOffset = 10000
		return m, nil
	}

	// Reset gg detection for any other key
	m.lastKey = ""
	return m, nil
}

// adjustContextTextareaHeight dynamically adjusts the context textarea height based on content
// The textarea grows as the user types more content, and shrinks when content is deleted
func adjustContextTextareaHeight(m *Model) {
	content := m.contextInput.Value()
	width := m.contextInput.Width()
	if width <= 0 {
		width = 60 // Default width
	}

	// Count lines needed for content
	lines := 1
	if content != "" {
		// Count wrapped lines
		for _, line := range strings.Split(content, "\n") {
			if len(line) == 0 {
				lines++
			} else {
				// Calculate how many display lines this line needs
				lineCount := (len(line) + width - 1) / width
				if lineCount == 0 {
					lineCount = 1
				}
				lines += lineCount
			}
		}
		// Don't double-count the initial line
		if lines > 1 && content[len(content)-1] != '\n' {
			lines--
		}
	}

	// Minimum 1 line, maximum 5 lines
	if lines < 1 {
		lines = 1
	}
	if lines > 5 {
		lines = 5
	}

	m.contextInput.SetHeight(lines)
}

// handleUnifiedBallFormKey handles keyboard input for the unified ball creation form
// Field order: Context, Title, Acceptance Criteria, Tags, Session, Model Size, Depends On
func (m Model) handleUnifiedBallFormKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Field indices are dynamic due to variable AC count
	// Order: Context(0), Title(1), ACs(2 to 2+len(ACs)), Tags, Session, ModelSize, DependsOn
	const (
		fieldContext = 0
		fieldIntent  = 1 // Title field (was intent)
		fieldACStart = 2 // ACs start at index 2
	)
	// Dynamic field indices calculated after ACs
	fieldACEnd := fieldACStart + len(m.pendingAcceptanceCriteria) // The "new AC" field
	fieldTags := fieldACEnd + 1
	fieldSession := fieldTags + 1
	fieldModelSize := fieldSession + 1
	fieldDependsOn := fieldModelSize + 1

	// Number of options for selection fields
	numModelSizeOptions := 4 // (default), small, medium, large

	// Count real sessions (excluding pseudo-sessions)
	numSessionOptions := 1 // Start with "(none)"
	for _, sess := range m.sessions {
		if sess.ID != PseudoSessionAll && sess.ID != PseudoSessionUntagged {
			numSessionOptions++
		}
	}

	// Calculate the maximum field index (DependsOn is the last field)
	maxFieldIndex := fieldDependsOn

	// Helper to check if we're on a text input field
	isTextInputField := func(field int) bool {
		return field == fieldContext || field == fieldIntent || field == fieldTags ||
			(field >= fieldACStart && field <= fieldACEnd)
	}

	// Helper to check if we're on an AC field
	isACField := func(field int) bool {
		return field >= fieldACStart && field <= fieldACEnd
	}

	// Helper to check if we're on a field that supports @ file autocomplete
	// (Context, Title, and ACs - but NOT Tags)
	isAutocompleteField := func(field int) bool {
		return field == fieldContext || field == fieldIntent ||
			(field >= fieldACStart && field <= fieldACEnd)
	}

	// Helper to update autocomplete state after text changes
	updateAutocomplete := func() {
		if m.fileAutocomplete != nil && isAutocompleteField(m.pendingBallFormField) {
			text := m.textInput.Value()
			cursorPos := m.textInput.Position()
			m.fileAutocomplete.UpdateFromText(text, cursorPos)
		} else if m.fileAutocomplete != nil {
			m.fileAutocomplete.Reset()
		}
	}

	// Helper to save current field value before moving
	saveCurrentFieldValue := func() {
		switch m.pendingBallFormField {
		case fieldContext:
			// Get value from textarea for context field
			m.pendingBallContext = strings.TrimSpace(m.contextInput.Value())
		case fieldIntent:
			m.pendingBallIntent = strings.TrimSpace(m.textInput.Value())
		default:
			value := strings.TrimSpace(m.textInput.Value())
			// Check if it's Tags field (dynamic index)
			if m.pendingBallFormField == fieldTags {
				m.pendingBallTags = value
			} else if isACField(m.pendingBallFormField) {
				// AC field
				acIndex := m.pendingBallFormField - fieldACStart
				if acIndex < len(m.pendingAcceptanceCriteria) {
					// Editing existing AC
					if value == "" {
						// Remove empty ACs
						m.pendingAcceptanceCriteria = append(
							m.pendingAcceptanceCriteria[:acIndex],
							m.pendingAcceptanceCriteria[acIndex+1:]...,
						)
					} else {
						m.pendingAcceptanceCriteria[acIndex] = value
					}
				}
				// Don't save if on the "new AC" field - that's handled by Enter
			}
		}
	}

	// Helper to recalculate dynamic field indices after AC changes
	recalcFieldIndices := func() (int, int, int, int, int) {
		newFieldACEnd := fieldACStart + len(m.pendingAcceptanceCriteria)
		newFieldTags := newFieldACEnd + 1
		newFieldSession := newFieldTags + 1
		newFieldModelSize := newFieldSession + 1
		newFieldDependsOn := newFieldModelSize + 1
		return newFieldACEnd, newFieldTags, newFieldSession, newFieldModelSize, newFieldDependsOn
	}

	// Helper to load field value into text input when entering field
	loadFieldValue := func(field int) {
		// Recalculate indices since ACs may have changed
		acEnd, tagsField, _, _, _ := recalcFieldIndices()

		m.textInput.Reset()
		switch field {
		case fieldContext:
			// Use textarea for context field
			m.contextInput.SetValue(m.pendingBallContext)
			m.contextInput.Focus()
			m.textInput.Blur()
			// Dynamically adjust height based on content
			adjustContextTextareaHeight(&m)
		case fieldIntent:
			m.contextInput.Blur()
			m.textInput.SetValue(m.pendingBallIntent)
			m.textInput.Placeholder = "What is this ball about? (50 char recommended)"
			m.textInput.Focus()
		default:
			m.contextInput.Blur()
			if field == tagsField {
				m.textInput.SetValue(m.pendingBallTags)
				m.textInput.Placeholder = "tag1, tag2, ..."
				m.textInput.Focus()
			} else if field >= fieldACStart && field <= acEnd {
				acIndex := field - fieldACStart
				if acIndex < len(m.pendingAcceptanceCriteria) {
					m.textInput.SetValue(m.pendingAcceptanceCriteria[acIndex])
					m.textInput.Placeholder = "Edit acceptance criterion"
				} else {
					m.textInput.SetValue("")
					m.textInput.Placeholder = "New acceptance criterion (Enter on empty = save)"
				}
				m.textInput.Focus()
			} else {
				// Selection field
				m.textInput.Blur()
			}
		}
	}

	switch msg.String() {
	case "esc":
		// Cancel input - clear pending state
		m.clearPendingBallState()
		m.mode = splitView
		m.message = "Cancelled"
		m.textInput.Blur()
		m.contextInput.Blur()
		return m, nil

	case "ctrl+enter":
		// Create the ball
		// Save current field value first
		saveCurrentFieldValue()

		// Validate required fields
		if m.pendingBallIntent == "" {
			m.message = "Title is required"
			return m, nil
		}

		return m.finalizeBallCreation()

	case "enter":
		// Behavior depends on current field
		if m.pendingBallFormField == fieldContext {
			// For context field, Enter adds newline in textarea
			var cmd tea.Cmd
			m.contextInput, cmd = m.contextInput.Update(msg)
			adjustContextTextareaHeight(&m)
			return m, cmd
		} else if m.pendingBallFormField == fieldDependsOn {
			// Open dependency selector
			return m.openDependencySelector()
		} else if isACField(m.pendingBallFormField) {
			acIndex := m.pendingBallFormField - fieldACStart
			value := strings.TrimSpace(m.textInput.Value())

			if acIndex == len(m.pendingAcceptanceCriteria) {
				// On the "new AC" field
				if value == "" {
					// Empty enter on the new AC field - create the ball
					saveCurrentFieldValue()
					// Validate required fields
					if m.pendingBallIntent == "" {
						m.message = "Title is required"
						return m, nil
					}
					return m.finalizeBallCreation()
				} else {
					// Add new AC and stay on the new AC field
					m.pendingAcceptanceCriteria = append(m.pendingAcceptanceCriteria, value)
					m.textInput.Reset()
					m.textInput.Placeholder = "New acceptance criterion (Enter on empty = save)"
					m.pendingBallFormField = fieldACStart + len(m.pendingAcceptanceCriteria) // Move to new "add" field
				}
			} else {
				// Editing existing AC - save and move to next field
				saveCurrentFieldValue()
				m.pendingBallFormField++
				// Recalculate indices after potential removal
				newACEnd, _, _, _, newDependsOn := recalcFieldIndices()
				maxFieldIndex = newDependsOn
				// Clamp to valid range
				if m.pendingBallFormField > newACEnd {
					// If we went past AC section, jump to Tags
					_, newFieldTags, _, _, _ := recalcFieldIndices()
					m.pendingBallFormField = newFieldTags
				}
				loadFieldValue(m.pendingBallFormField)
			}
		} else {
			// On other fields - save and move to next
			saveCurrentFieldValue()
			m.pendingBallFormField++
			// Recalculate after potential changes
			_, _, _, _, newDependsOn := recalcFieldIndices()
			maxFieldIndex = newDependsOn
			if m.pendingBallFormField > maxFieldIndex {
				m.pendingBallFormField = maxFieldIndex
			}
			loadFieldValue(m.pendingBallFormField)
		}
		return m, nil

	case "up":
		// If autocomplete is active, navigate suggestions instead of fields
		if m.fileAutocomplete != nil && m.fileAutocomplete.Active && len(m.fileAutocomplete.Suggestions) > 0 {
			m.fileAutocomplete.SelectPrev()
			return m, nil
		}
		// Arrow key up always moves to previous field
		saveCurrentFieldValue()
		m.pendingBallFormField--
		// Recalculate after potential removal
		_, _, _, _, newDependsOn := recalcFieldIndices()
		maxFieldIndex = newDependsOn
		if m.pendingBallFormField < 0 {
			m.pendingBallFormField = maxFieldIndex
		}
		loadFieldValue(m.pendingBallFormField)
		return m, nil

	case "down":
		// If autocomplete is active, navigate suggestions instead of fields
		if m.fileAutocomplete != nil && m.fileAutocomplete.Active && len(m.fileAutocomplete.Suggestions) > 0 {
			m.fileAutocomplete.SelectNext()
			return m, nil
		}
		// Arrow key down always moves to next field
		saveCurrentFieldValue()
		// Check if we're on the "new AC" field - if so, move to Tags
		newACEnd, newFieldTags, _, _, newDependsOn := recalcFieldIndices()
		if m.pendingBallFormField == newACEnd {
			// On "new AC" field, down arrow moves to Tags
			m.pendingBallFormField = newFieldTags
		} else {
			m.pendingBallFormField++
			maxFieldIndex = newDependsOn
			if m.pendingBallFormField > maxFieldIndex {
				m.pendingBallFormField = 0
			}
		}
		loadFieldValue(m.pendingBallFormField)
		return m, nil

	case "k":
		// k should ONLY be used for typing in text fields, never for navigation
		if isTextInputField(m.pendingBallFormField) {
			if m.pendingBallFormField == fieldContext {
				var cmd tea.Cmd
				m.contextInput, cmd = m.contextInput.Update(msg)
				adjustContextTextareaHeight(&m)
				return m, cmd
			}
			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd
		}
		// On selection fields, k is just ignored (can't type in selection fields)
		return m, nil

	case "j":
		// j should ONLY be used for typing in text fields, never for navigation
		if isTextInputField(m.pendingBallFormField) {
			if m.pendingBallFormField == fieldContext {
				var cmd tea.Cmd
				m.contextInput, cmd = m.contextInput.Update(msg)
				adjustContextTextareaHeight(&m)
				return m, cmd
			}
			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd
		}
		// On selection fields, j is just ignored (can't type in selection fields)
		return m, nil

	case "left":
		// Arrow key left only cycles selection left for selection fields
		_, _, sessionField, modelSizeField, _ := recalcFieldIndices()
		if m.pendingBallFormField == sessionField {
			m.pendingBallSession--
			if m.pendingBallSession < 0 {
				m.pendingBallSession = numSessionOptions - 1
			}
		} else if m.pendingBallFormField == modelSizeField {
			m.pendingBallModelSize--
			if m.pendingBallModelSize < 0 {
				m.pendingBallModelSize = numModelSizeOptions - 1
			}
		}
		return m, nil

	case "right":
		// Arrow key right only cycles selection right for selection fields
		_, _, sessionField, modelSizeField, _ := recalcFieldIndices()
		if m.pendingBallFormField == sessionField {
			m.pendingBallSession++
			if m.pendingBallSession >= numSessionOptions {
				m.pendingBallSession = 0
			}
		} else if m.pendingBallFormField == modelSizeField {
			m.pendingBallModelSize++
			if m.pendingBallModelSize >= numModelSizeOptions {
				m.pendingBallModelSize = 0
			}
		}
		return m, nil

	case "h":
		// h should ONLY be used for typing in text fields, never for navigation
		if isTextInputField(m.pendingBallFormField) {
			if m.pendingBallFormField == fieldContext {
				var cmd tea.Cmd
				m.contextInput, cmd = m.contextInput.Update(msg)
				adjustContextTextareaHeight(&m)
				return m, cmd
			}
			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd
		}
		// On selection fields, h is just ignored (can't type in selection fields)
		return m, nil

	case "l":
		// l should ONLY be used for typing in text fields, never for navigation
		if isTextInputField(m.pendingBallFormField) {
			if m.pendingBallFormField == fieldContext {
				var cmd tea.Cmd
				m.contextInput, cmd = m.contextInput.Update(msg)
				adjustContextTextareaHeight(&m)
				return m, cmd
			}
			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd
		}
		// On selection fields, l is just ignored (can't type in selection fields)
		return m, nil

	case "tab":
		// If autocomplete is active and we're on an autocomplete field, accept the completion
		if m.fileAutocomplete != nil && m.fileAutocomplete.Active && len(m.fileAutocomplete.Suggestions) > 0 {
			// Apply the selected completion
			if m.pendingBallFormField == fieldContext {
				newText := m.fileAutocomplete.ApplyCompletion(m.contextInput.Value())
				m.contextInput.SetValue(newText)
				adjustContextTextareaHeight(&m)
			} else {
				newText := m.fileAutocomplete.ApplyCompletion(m.textInput.Value())
				m.textInput.SetValue(newText)
				m.textInput.SetCursor(len(newText))
			}
			m.fileAutocomplete.Reset()
			return m, nil
		}

		// Tab cycles through selection options or moves to next field
		_, _, sessionField, modelSizeField, _ := recalcFieldIndices()
		if m.pendingBallFormField == sessionField {
			m.pendingBallSession++
			if m.pendingBallSession >= numSessionOptions {
				m.pendingBallSession = 0
			}
		} else if m.pendingBallFormField == modelSizeField {
			m.pendingBallModelSize++
			if m.pendingBallModelSize >= numModelSizeOptions {
				m.pendingBallModelSize = 0
			}
		} else {
			// For text fields, tab moves to next field
			saveCurrentFieldValue()
			// Check if we're on the "new AC" field - if so, move to Tags
			newACEnd, newFieldTags, _, _, newDependsOn := recalcFieldIndices()
			if m.pendingBallFormField == newACEnd {
				m.pendingBallFormField = newFieldTags
			} else {
				m.pendingBallFormField++
				maxFieldIndex = newDependsOn
				if m.pendingBallFormField > maxFieldIndex {
					m.pendingBallFormField = 0
				}
			}
			loadFieldValue(m.pendingBallFormField)
		}
		return m, nil

	case "backspace", "delete":
		// Allow deletion in text fields
		// Note: Backspace doesn't re-trigger autocomplete (per AC requirement)
		if isTextInputField(m.pendingBallFormField) {
			if m.pendingBallFormField == fieldContext {
				// Use textarea for context
				var cmd tea.Cmd
				m.contextInput, cmd = m.contextInput.Update(msg)
				// Adjust height after deletion
				adjustContextTextareaHeight(&m)
				return m, cmd
			}
			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)
			// Don't update autocomplete on backspace - only @ typing triggers it
			return m, cmd
		}
		return m, nil

	case " ":
		// Space dismisses autocomplete (per AC requirement)
		if isTextInputField(m.pendingBallFormField) {
			if m.pendingBallFormField == fieldContext {
				// Use textarea for context
				var cmd tea.Cmd
				m.contextInput, cmd = m.contextInput.Update(msg)
				adjustContextTextareaHeight(&m)
				// Dismiss autocomplete on space
				if m.fileAutocomplete != nil {
					m.fileAutocomplete.Deactivate()
				}
				return m, cmd
			}
			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)
			// Dismiss autocomplete on space
			if m.fileAutocomplete != nil {
				m.fileAutocomplete.Deactivate()
			}
			return m, cmd
		}
		return m, nil

	default:
		// Pass to textinput only if on text input field
		if isTextInputField(m.pendingBallFormField) {
			if m.pendingBallFormField == fieldContext {
				// Use textarea for context
				var cmd tea.Cmd
				m.contextInput, cmd = m.contextInput.Update(msg)
				// Adjust height after typing
				adjustContextTextareaHeight(&m)
				// Update autocomplete state after text changes (for @ detection)
				if m.fileAutocomplete != nil {
					text := m.contextInput.Value()
					cursorPos := m.contextInput.LineInfo().CharOffset
					m.fileAutocomplete.UpdateFromText(text, cursorPos)
				}
				return m, cmd
			}
			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)
			// Update autocomplete state after text changes (for @ detection)
			updateAutocomplete()
			return m, cmd
		}
		return m, nil
	}
}

// openDependencySelector opens the dependency selector for the ball form
func (m Model) openDependencySelector() (tea.Model, tea.Cmd) {
	// Build list of non-complete balls that can be dependencies
	m.dependencySelectBalls = make([]*session.Ball, 0)
	for _, ball := range m.balls {
		// Exclude complete/researched balls
		if ball.State != session.StateComplete && ball.State != session.StateResearched {
			m.dependencySelectBalls = append(m.dependencySelectBalls, ball)
		}
	}

	if len(m.dependencySelectBalls) == 0 {
		m.message = "No non-complete balls available as dependencies"
		return m, nil
	}

	// Initialize selection state from current pendingBallDependsOn
	m.dependencySelectActive = make(map[string]bool)
	for _, depID := range m.pendingBallDependsOn {
		m.dependencySelectActive[depID] = true
	}
	m.dependencySelectIndex = 0
	m.mode = dependencySelectorView
	return m, nil
}

// handleDependencySelectorKey handles keyboard input in the dependency selector view
func (m Model) handleDependencySelectorKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		// Cancel selection - return to form without saving
		m.mode = unifiedBallFormView
		m.dependencySelectBalls = nil
		m.dependencySelectActive = nil
		m.message = "Cancelled"
		return m, nil

	case "up", "k":
		// Move selection up
		if m.dependencySelectIndex > 0 {
			m.dependencySelectIndex--
		}
		return m, nil

	case "down", "j":
		// Move selection down
		if m.dependencySelectIndex < len(m.dependencySelectBalls)-1 {
			m.dependencySelectIndex++
		}
		return m, nil

	case " ":
		// Toggle selection on current item
		if len(m.dependencySelectBalls) > 0 && m.dependencySelectIndex < len(m.dependencySelectBalls) {
			ball := m.dependencySelectBalls[m.dependencySelectIndex]
			if m.dependencySelectActive[ball.ID] {
				delete(m.dependencySelectActive, ball.ID)
			} else {
				m.dependencySelectActive[ball.ID] = true
			}
		}
		return m, nil

	case "enter":
		// Confirm selection - save to pendingBallDependsOn and return to form
		m.pendingBallDependsOn = make([]string, 0)
		for ballID := range m.dependencySelectActive {
			m.pendingBallDependsOn = append(m.pendingBallDependsOn, ballID)
		}
		// Sort for consistent display
		sort.Strings(m.pendingBallDependsOn)

		m.mode = unifiedBallFormView
		m.dependencySelectBalls = nil
		m.dependencySelectActive = nil
		if len(m.pendingBallDependsOn) > 0 {
			m.message = fmt.Sprintf("Selected %d dependencies", len(m.pendingBallDependsOn))
		} else {
			m.message = "Cleared dependencies"
		}
		return m, nil
	}
	return m, nil
}
