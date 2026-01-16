package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ohare93/juggle/internal/session"
)

// handleInputKey handles keyboard input in text input modes
func (m Model) handleInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Cancel input
		m.editingSession = nil // Clear the editing session
		m.editingBall = nil // Clear the editing ball
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
		if m.editingSession == nil {
			m.message = "No session selected for editing"
			m.mode = splitView
			return m, nil
		}
		err := m.sessionStore.UpdateSessionDescription(m.editingSession.ID, value)
		if err != nil {
			m.message = "Error updating session: " + err.Error()
			m.mode = splitView
			return m, nil
		}
		m.addActivity("Updated session description: " + m.editingSession.ID)
		m.message = "Updated session: " + m.editingSession.ID
	}

	m.editingSession = nil // Clear the editing session
	m.mode = splitView
	return m, loadSessions(m.sessionStore, m.config, m.localOnly)
}

// submitBallInput handles ball title edit submission
// Note: Ball creation is handled via unifiedBallFormView, not this function
func (m Model) submitBallInput(value string) (tea.Model, tea.Cmd) {
	// Edit ball title
	if m.editingBall == nil {
		m.mode = splitView
		return m, nil
	}
	m.editingBall.SetTitle(value)
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

// submitBlockedInput handles blocked reason submission
func (m Model) submitBlockedInput(value string) (tea.Model, tea.Cmd) {
	if m.editingBall == nil {
		m.mode = splitView
		return m, nil
	}

	if err := m.editingBall.SetBlocked(value); err != nil {
		m.message = "Error: " + err.Error()
		m.mode = splitView
		return m, nil
	}
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

// handleSessionSelectorKey handles keyboard input in session selector mode
func (m Model) handleSessionSelectorKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		// Cancel selection
		m.mode = splitView
		m.sessionSelectItems = nil
		m.sessionSelectActive = nil
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

	case " ":
		// Toggle selection of current session
		if len(m.sessionSelectItems) > 0 && m.sessionSelectIndex < len(m.sessionSelectItems) {
			if m.sessionSelectActive == nil {
				m.sessionSelectActive = make(map[string]bool)
			}
			sess := m.sessionSelectItems[m.sessionSelectIndex]
			m.sessionSelectActive[sess.ID] = !m.sessionSelectActive[sess.ID]
		}
		return m, nil

	case "enter":
		// Confirm selection (add all selected sessions as tags)
		return m.submitSessionSelection()
	}
	return m, nil
}

// submitSessionSelection adds all selected sessions as tags to the ball.
// If no sessions are selected via checkbox (multi-select), falls back to adding the focused session.
func (m Model) submitSessionSelection() (tea.Model, tea.Cmd) {
	if m.editingBall == nil || len(m.sessionSelectItems) == 0 {
		m.mode = splitView
		m.sessionSelectItems = nil
		m.sessionSelectActive = nil
		return m, nil
	}

	// Collect all selected sessions (from checkboxes)
	selectedSessions := make([]string, 0)
	if m.sessionSelectActive != nil {
		for _, sess := range m.sessionSelectItems {
			if m.sessionSelectActive[sess.ID] {
				selectedSessions = append(selectedSessions, sess.ID)
			}
		}
	}

	// If no checkboxes selected, fall back to cursor position (legacy behavior)
	if len(selectedSessions) == 0 {
		if m.sessionSelectIndex >= len(m.sessionSelectItems) {
			m.sessionSelectIndex = len(m.sessionSelectItems) - 1
		}
		selectedSessions = []string{m.sessionSelectItems[m.sessionSelectIndex].ID}
	}

	// Add all selected sessions as tags
	for _, sessionID := range selectedSessions {
		m.editingBall.AddTag(sessionID)
	}

	if len(selectedSessions) == 1 {
		m.addActivity("Added to session: " + selectedSessions[0])
		m.message = "Added to session: " + selectedSessions[0]
	} else {
		m.addActivity("Added to " + fmt.Sprintf("%d", len(selectedSessions)) + " sessions")
		m.message = "Added to " + fmt.Sprintf("%d", len(selectedSessions)) + " sessions"
	}

	store, err := session.NewStore(m.editingBall.WorkingDir)
	if err != nil {
		m.message = "Error: " + err.Error()
		m.mode = splitView
		m.sessionSelectItems = nil
		m.sessionSelectActive = nil
		return m, nil
	}

	m.mode = splitView
	m.sessionSelectItems = nil
	m.sessionSelectActive = nil
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
	m.sessionSelectActive = make(map[string]bool) // Initialize multi-select map
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
