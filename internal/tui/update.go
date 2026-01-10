package tui

import (
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
		if m.mode == inputSessionView || m.mode == inputBallView || m.mode == inputTodoView || m.mode == inputBlockedView {
			return m.handleInputKey(msg)
		}

		// Handle delete confirmation in split view
		if m.mode == confirmSplitDelete {
			return m.handleSplitConfirmDelete(msg)
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

	case watcherEventMsg:
		return m.handleWatcherEvent(msg.event)

	case watcherErrorMsg:
		m.addActivity("Watcher error: " + msg.err.Error())
		// Continue listening for more events
		if m.fileWatcher != nil {
			return m, listenForWatcherEvents(m.fileWatcher)
		}
		return m, nil
	}

	return m, nil
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
			if m.selectedBall != nil && len(m.selectedBall.Todos) > 0 {
				m.activePanel = TodosPanel
			} else {
				m.activePanel = SessionsPanel
			}
		case TodosPanel:
			m.activePanel = SessionsPanel
		}
		return m, nil

	case "shift+tab", "h":
		// Cycle to previous panel
		m.message = ""
		switch m.activePanel {
		case SessionsPanel:
			if m.selectedBall != nil && len(m.selectedBall.Todos) > 0 {
				m.activePanel = TodosPanel
			} else {
				m.activePanel = BallsPanel
			}
		case BallsPanel:
			m.activePanel = SessionsPanel
		case TodosPanel:
			m.activePanel = BallsPanel
		}
		return m, nil

	case "up", "k":
		m.message = ""
		return m.handleSplitViewNavUp()

	case "down", "j":
		m.message = ""
		return m.handleSplitViewNavDown()

	case "enter":
		return m.handleSplitViewEnter()

	case "esc":
		// Go back or deselect
		if m.activePanel == TodosPanel {
			m.activePanel = BallsPanel
			m.todoCursor = 0
		} else if m.selectedBall != nil {
			m.selectedBall = nil
			m.todoCursor = 0
		} else if m.selectedSession != nil {
			m.selectedSession = nil
			m.cursor = 0
		} else {
			return m, tea.Quit
		}
		return m, nil

	case " ":
		// Toggle todo completion in todos panel
		if m.activePanel == TodosPanel && m.selectedBall != nil && len(m.selectedBall.Todos) > 0 {
			return m.handleToggleTodo()
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
		// TODO: Show help for split view
		m.message = "Help: Tab=panels j/k=nav Enter=select a=add e=edit d=delete s=start c=complete b=block q=quit"
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
	}

	return m, nil
}

// handleSplitViewNavUp handles up navigation in split view
func (m Model) handleSplitViewNavUp() (tea.Model, tea.Cmd) {
	switch m.activePanel {
	case SessionsPanel:
		if m.sessionCursor > 0 {
			m.sessionCursor--
		}
	case BallsPanel:
		if m.cursor > 0 {
			m.cursor--
		}
	case TodosPanel:
		if m.todoCursor > 0 {
			m.todoCursor--
		}
	}
	return m, nil
}

// handleSplitViewNavDown handles down navigation in split view
func (m Model) handleSplitViewNavDown() (tea.Model, tea.Cmd) {
	switch m.activePanel {
	case SessionsPanel:
		if m.sessionCursor < len(m.sessions)-1 {
			m.sessionCursor++
		}
	case BallsPanel:
		balls := m.getBallsForSession()
		if m.cursor < len(balls)-1 {
			m.cursor++
		}
	case TodosPanel:
		if m.selectedBall != nil && m.todoCursor < len(m.selectedBall.Todos)-1 {
			m.todoCursor++
		}
	}
	return m, nil
}

// handleSplitViewEnter handles enter key in split view
func (m Model) handleSplitViewEnter() (tea.Model, tea.Cmd) {
	switch m.activePanel {
	case SessionsPanel:
		// Select session
		if len(m.sessions) > 0 && m.sessionCursor < len(m.sessions) {
			m.selectedSession = m.sessions[m.sessionCursor]
			m.cursor = 0 // Reset ball cursor for new session
			m.addActivity("Selected session: " + m.selectedSession.ID)
		}
	case BallsPanel:
		// Select ball and show todos
		balls := m.getBallsForSession()
		if len(balls) > 0 && m.cursor < len(balls) {
			m.selectedBall = balls[m.cursor]
			m.todoCursor = 0
			if len(m.selectedBall.Todos) > 0 {
				m.activePanel = TodosPanel
			}
			m.addActivity("Selected ball: " + m.selectedBall.ID)
		}
	case TodosPanel:
		// Toggle todo
		return m.handleToggleTodo()
	}
	return m, nil
}

// handleToggleTodo toggles a todo's completion status
func (m Model) handleToggleTodo() (tea.Model, tea.Cmd) {
	if m.selectedBall == nil || len(m.selectedBall.Todos) == 0 {
		return m, nil
	}
	if m.todoCursor >= len(m.selectedBall.Todos) {
		return m, nil
	}

	todo := &m.selectedBall.Todos[m.todoCursor]
	todo.Done = !todo.Done

	status := "incomplete"
	if todo.Done {
		status = "complete"
	}
	m.addActivity("Todo marked " + status + ": " + truncate(todo.Text, 20))

	store, err := session.NewStore(m.selectedBall.WorkingDir)
	if err != nil {
		m.message = "Error: " + err.Error()
		return m, nil
	}

	return m, updateBall(store, m.selectedBall)
}

// handleSplitStartBall starts the selected ball in split view
func (m Model) handleSplitStartBall() (tea.Model, tea.Cmd) {
	balls := m.getBallsForSession()
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

// handleSplitCompleteBall completes the selected ball in split view
func (m Model) handleSplitCompleteBall() (tea.Model, tea.Cmd) {
	balls := m.getBallsForSession()
	if len(balls) == 0 || m.cursor >= len(balls) {
		return m, nil
	}

	ball := balls[m.cursor]
	ball.SetState(session.StateComplete)
	m.addActivity("Completed ball: " + ball.ID)

	store, err := session.NewStore(ball.WorkingDir)
	if err != nil {
		m.message = "Error: " + err.Error()
		return m, nil
	}

	return m, updateBall(store, ball)
}

// handleSplitBlockBall prompts for a blocked reason
func (m Model) handleSplitBlockBall() (tea.Model, tea.Cmd) {
	balls := m.getBallsForSession()
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
	return m, updateBall(store, ball)
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
	m.filteredBalls = make([]*session.Session, 0)

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
	case TodosPanel:
		if m.selectedBall == nil {
			m.message = "No ball selected"
			return m, nil
		}
		m.textInput.Placeholder = "Todo text"
		m.inputTarget = "todo"
		m.editingTodo = -1 // -1 means new todo
		m.mode = inputTodoView
		m.addActivity("Adding new todo...")
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
		if len(m.sessions) == 0 || m.sessionCursor >= len(m.sessions) {
			m.message = "No session selected"
			return m, nil
		}
		sess := m.sessions[m.sessionCursor]
		m.textInput.Placeholder = "Session description"
		m.textInput.SetValue(sess.Description)
		m.inputTarget = "session_description"
		m.mode = inputSessionView
		m.addActivity("Editing session: " + sess.ID)

	case BallsPanel:
		balls := m.getBallsForSession()
		if len(balls) == 0 || m.cursor >= len(balls) {
			m.message = "No ball selected"
			return m, nil
		}
		ball := balls[m.cursor]
		m.editingBall = ball
		m.textInput.Placeholder = "Ball intent"
		m.textInput.SetValue(ball.Intent)
		m.inputTarget = "intent"
		m.mode = inputBallView
		m.addActivity("Editing ball: " + ball.ID)

	case TodosPanel:
		if m.selectedBall == nil || len(m.selectedBall.Todos) == 0 {
			m.message = "No todo selected"
			return m, nil
		}
		if m.todoCursor >= len(m.selectedBall.Todos) {
			m.message = "No todo selected"
			return m, nil
		}
		todo := m.selectedBall.Todos[m.todoCursor]
		m.textInput.Placeholder = "Todo text"
		m.textInput.SetValue(todo.Text)
		m.editingTodo = m.todoCursor
		m.mode = inputTodoView
		m.addActivity("Editing todo...")
	}

	return m, nil
}

// handleSplitDeletePrompt shows delete confirmation
func (m Model) handleSplitDeletePrompt() (tea.Model, tea.Cmd) {
	switch m.activePanel {
	case SessionsPanel:
		if len(m.sessions) == 0 || m.sessionCursor >= len(m.sessions) {
			m.message = "No session selected"
			return m, nil
		}
		m.confirmAction = "delete_session"
		m.mode = confirmSplitDelete
		m.addActivity("Confirming session deletion...")

	case BallsPanel:
		balls := m.getBallsForSession()
		if len(balls) == 0 || m.cursor >= len(balls) {
			m.message = "No ball selected"
			return m, nil
		}
		m.confirmAction = "delete_ball"
		m.mode = confirmSplitDelete
		m.addActivity("Confirming ball deletion...")

	case TodosPanel:
		if m.selectedBall == nil || len(m.selectedBall.Todos) == 0 {
			m.message = "No todo selected"
			return m, nil
		}
		if m.todoCursor >= len(m.selectedBall.Todos) {
			m.message = "No todo selected"
			return m, nil
		}
		m.confirmAction = "delete_todo"
		m.mode = confirmSplitDelete
		m.addActivity("Confirming todo deletion...")
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
	case inputTodoView:
		return m.submitTodoInput(value)
	case inputBlockedView:
		return m.submitBlockedInput(value)
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
		if m.sessionCursor >= len(m.sessions) {
			m.mode = splitView
			return m, nil
		}
		sess := m.sessions[m.sessionCursor]
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
		ball, err := session.New(m.store.ProjectDir(), value, session.PriorityMedium)
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

// submitTodoInput handles todo add/edit submission
func (m Model) submitTodoInput(value string) (tea.Model, tea.Cmd) {
	if m.selectedBall == nil {
		m.mode = splitView
		return m, nil
	}

	if m.editingTodo == -1 {
		// Add new todo
		m.selectedBall.AddTodo(value)
		m.addActivity("Added todo: " + truncate(value, 20))
		m.message = "Added todo"
	} else {
		// Edit existing todo
		if m.editingTodo < len(m.selectedBall.Todos) {
			m.selectedBall.Todos[m.editingTodo].Text = value
			m.addActivity("Updated todo")
			m.message = "Updated todo"
		}
	}

	store, err := session.NewStore(m.selectedBall.WorkingDir)
	if err != nil {
		m.message = "Error: " + err.Error()
		m.mode = splitView
		return m, nil
	}

	m.mode = splitView
	return m, updateBall(store, m.selectedBall)
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
		if m.sessionCursor >= len(m.sessions) {
			m.mode = splitView
			return m, nil
		}
		sess := m.sessions[m.sessionCursor]
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
		balls := m.getBallsForSession()
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

	case "delete_todo":
		if m.selectedBall == nil || m.todoCursor >= len(m.selectedBall.Todos) {
			m.mode = splitView
			return m, nil
		}
		todoText := m.selectedBall.Todos[m.todoCursor].Text
		// Remove the todo
		m.selectedBall.Todos = append(
			m.selectedBall.Todos[:m.todoCursor],
			m.selectedBall.Todos[m.todoCursor+1:]...,
		)
		// Adjust cursor if needed
		if m.todoCursor >= len(m.selectedBall.Todos) && m.todoCursor > 0 {
			m.todoCursor--
		}
		store, err := session.NewStore(m.selectedBall.WorkingDir)
		if err != nil {
			m.message = "Error: " + err.Error()
			m.mode = splitView
			return m, nil
		}
		m.addActivity("Deleted todo: " + truncate(todoText, 20))
		m.message = "Deleted todo"
		m.mode = splitView
		return m, updateBall(store, m.selectedBall)
	}

	m.mode = splitView
	return m, nil
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
