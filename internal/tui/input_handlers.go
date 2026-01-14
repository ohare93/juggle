package tui

import (
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
