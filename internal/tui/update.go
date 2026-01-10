package tui

import (
	"strings"

	"github.com/charmbracelet/bubbletea"
	"github.com/ohare93/juggle/internal/session"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
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
		return m, nil

	case ballUpdatedMsg:
		if msg.err != nil {
			m.message = "Error: " + msg.err.Error()
		} else {
			m.message = "Ball updated successfully"
		}
		// Reload balls
		return m, loadBalls(m.store, m.config, m.localOnly)
	}

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
