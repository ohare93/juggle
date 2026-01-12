package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	if m.err != nil {
		return errorStyle.Render(fmt.Sprintf("Error: %v\n\nPress q to quit", m.err))
	}

	switch m.mode {
	case listView:
		return m.renderListView()
	case detailView:
		return m.renderDetailView()
	case helpView:
		return m.renderHelpView()
	case confirmDeleteView:
		return m.renderConfirmDeleteView()
	case splitView:
		return m.renderSplitView()
	case splitHelpView:
		return m.renderSplitHelpView()
	case inputSessionView, inputBallView, inputBlockedView, inputAcceptanceCriteriaView:
		return m.renderInputView()
	case inputTagView:
		return m.renderTagView()
	case sessionSelectorView:
		return m.renderSessionSelectorView()
	case confirmSplitDelete:
		return m.renderSplitConfirmDelete()
	case confirmAgentLaunch:
		return m.renderAgentLaunchConfirm()
	case panelSearchView:
		return m.renderPanelSearchView()
	default:
		return "Unknown view"
	}
}

func (m Model) renderListView() string {
	var b strings.Builder

	// Title
	title := titleStyle.Render("🎯 Juggler - Task Manager")
	b.WriteString(title + "\n\n")

	// Stats with active filters
	var activeFilters []string
	for state, visible := range m.filterStates {
		if visible {
			activeFilters = append(activeFilters, state)
		}
	}
	filterStr := strings.Join(activeFilters, ", ")
	if len(activeFilters) == 4 {
		filterStr = "all"
	}

	stats := fmt.Sprintf("Total: %d | Pending: %d | In Progress: %d | Blocked: %d | Complete: %d | Filter: %s",
		len(m.balls),
		countByState(m.balls, "pending"),
		countByState(m.balls, "in_progress"),
		countByState(m.balls, "blocked"),
		countByState(m.balls, "complete"),
		filterStr,
	)
	b.WriteString(stats + "\n\n")

	// Ball list
	if len(m.filteredBalls) == 0 {
		b.WriteString("No balls to display\n")
	} else {
		b.WriteString(renderBallList(m.filteredBalls, m.cursor, m.width))
	}

	// Footer with keybindings
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("Navigation: ↑/k up • ↓/j down • enter details • esc exit\n"))
	b.WriteString(helpStyle.Render("Actions: s start • c complete • d drop • x delete • p cycle priority\n"))
	b.WriteString(helpStyle.Render("Filter: 1 all • 2 toggle pending • 3 toggle in_progress • 4 toggle blocked • 5 toggle complete\n"))
	b.WriteString(helpStyle.Render("Other: r set pending • R refresh • tab cycle state • ? help • q quit\n"))

	// Message
	if m.message != "" {
		b.WriteString("\n" + messageStyle.Render(m.message))
	}

	return b.String()
}

func (m Model) renderDetailView() string {
	if m.selectedBall == nil {
		return "No ball selected"
	}
	return renderBallDetail(m.selectedBall)
}

func (m Model) renderHelpView() string {
	var b strings.Builder

	title := titleStyle.Render("🎯 Juggler TUI - Help")
	b.WriteString(title + "\n\n")

	b.WriteString(helpSection("Navigation", []helpItem{
		{"↑ / k", "Move up"},
		{"↓ / j", "Move down"},
		{"Enter", "View ball details"},
		{"b", "Back to list"},
		{"Esc", "Back to list (or exit from list view)"},
	}))

	b.WriteString(helpSection("Quick Actions", []helpItem{
		{"s", "Start ball (→ in_progress)"},
		{"c", "Complete ball (→ complete, archives)"},
		{"d", "Block ball (→ blocked)"},
		{"x", "Delete ball (with confirmation)"},
		{"p", "Cycle priority (low → medium → high → urgent → low)"},
		{"r", "Set ball to pending state"},
		{"tab", "Cycle state (pending → in_progress → complete → blocked → pending)"},
	}))

	b.WriteString(helpSection("Filters", []helpItem{
		{"1", "Show all balls"},
		{"2", "Toggle pending balls"},
		{"3", "Toggle in_progress balls"},
		{"4", "Toggle blocked balls"},
		{"5", "Toggle complete balls"},
	}))

	b.WriteString(helpSection("Other", []helpItem{
		{"R", "Refresh/reload balls"},
		{"?", "Toggle this help"},
		{"q / Ctrl+C", "Quit"},
	}))

	b.WriteString("\n" + helpStyle.Render("Press 'b' or '?' to go back"))

	return b.String()
}

type helpItem struct {
	key   string
	desc  string
}

func helpSection(title string, items []helpItem) string {
	var b strings.Builder

	b.WriteString(titleStyle.Render(title) + "\n")
	for _, item := range items {
		keyStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
		b.WriteString(fmt.Sprintf("  %s  %s\n", keyStyle.Render(item.key), item.desc))
	}
	b.WriteString("\n")

	return b.String()
}

// renderInputView renders the text input dialog
func (m Model) renderInputView() string {
	var b strings.Builder

	// Determine title based on mode and action
	var title string
	switch m.mode {
	case inputSessionView:
		if m.inputAction == actionAdd {
			title = "Create New Session"
		} else {
			title = "Edit Session"
		}
	case inputBallView:
		if m.inputAction == actionAdd {
			title = "Create New Ball"
		} else {
			title = "Edit Ball"
		}
	case inputBlockedView:
		title = "Block Ball"
	case inputAcceptanceCriteriaView:
		title = "Add Acceptance Criteria"
	}

	titleStyled := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("6")).
		Render(title)
	b.WriteString(titleStyled + "\n\n")

	// Show context based on mode
	switch m.mode {
	case inputSessionView:
		if m.inputAction == actionEdit && m.sessionCursor < len(m.sessions) {
			sess := m.sessions[m.sessionCursor]
			b.WriteString(fmt.Sprintf("Session: %s\n\n", sess.ID))
		}
	case inputBallView:
		if m.inputAction == actionEdit && m.editingBall != nil {
			b.WriteString(fmt.Sprintf("Ball: %s\n\n", m.editingBall.ID))
		}
		if m.selectedSession != nil && m.inputAction == actionAdd {
			b.WriteString(fmt.Sprintf("Session: %s\n\n", m.selectedSession.ID))
		}
	case inputBlockedView:
		if m.editingBall != nil {
			b.WriteString(fmt.Sprintf("Ball: %s\n", m.editingBall.ID))
			b.WriteString(fmt.Sprintf("Intent: %s\n\n", m.editingBall.Intent))
		}
	case inputAcceptanceCriteriaView:
		b.WriteString(fmt.Sprintf("Intent: %s\n", m.pendingBallIntent))
		if len(m.pendingAcceptanceCriteria) > 0 {
			b.WriteString("\nCriteria entered:\n")
			for i, ac := range m.pendingAcceptanceCriteria {
				b.WriteString(fmt.Sprintf("  %d. %s\n", i+1, ac))
			}
		}
		b.WriteString("\n(Enter empty line to finish)\n\n")
	}

	// Show input field
	inputStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("6")).
		Padding(0, 1).
		Width(50)
	b.WriteString(inputStyle.Render(m.textInput.View()) + "\n\n")

	// Show message if any
	if m.message != "" {
		b.WriteString(messageStyle.Render(m.message) + "\n\n")
	}

	// Help
	help := lipgloss.NewStyle().
		Faint(true).
		Render("Enter = submit | Esc = cancel")
	b.WriteString(help)

	return b.String()
}

// renderSplitConfirmDelete renders the delete confirmation for split view
func (m Model) renderSplitConfirmDelete() string {
	var b strings.Builder

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("1")). // Red
		Render("Confirm Delete")
	b.WriteString(title + "\n\n")

	// Show what will be deleted
	switch m.confirmAction {
	case "delete_session":
		if m.sessionCursor < len(m.sessions) {
			sess := m.sessions[m.sessionCursor]
			b.WriteString(fmt.Sprintf("Session: %s\n", sess.ID))
			if sess.Description != "" {
				b.WriteString(fmt.Sprintf("Description: %s\n", sess.Description))
			}
			ballCount := m.countBallsForSession(sess.ID)
			b.WriteString(fmt.Sprintf("Balls: %d\n", ballCount))
		}
	case "delete_ball":
		balls := m.getBallsForSession()
		if m.cursor < len(balls) {
			ball := balls[m.cursor]
			b.WriteString(fmt.Sprintf("Ball: %s\n", ball.ID))
			b.WriteString(fmt.Sprintf("Intent: %s\n", ball.Intent))
			b.WriteString(fmt.Sprintf("State: %s\n", ball.State))
			b.WriteString(fmt.Sprintf("Criteria: %d\n", len(ball.AcceptanceCriteria)))
		}
	}

	b.WriteString("\n")

	warning := lipgloss.NewStyle().
		Foreground(lipgloss.Color("3")). // Yellow
		Render("This action cannot be undone.")
	b.WriteString(warning + "\n\n")

	prompt := lipgloss.NewStyle().
		Bold(true).
		Render("Delete? [y/N]")
	b.WriteString(prompt + "\n\n")

	help := lipgloss.NewStyle().
		Faint(true).
		Render("y = confirm | n/Esc = cancel")
	b.WriteString(help)

	return b.String()
}

// renderAgentLaunchConfirm renders the agent launch confirmation dialog
func (m Model) renderAgentLaunchConfirm() string {
	var b strings.Builder

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("3")). // Yellow
		Render("Launch Agent")
	b.WriteString(title + "\n\n")

	// Show session details
	if m.selectedSession != nil {
		b.WriteString(fmt.Sprintf("Session: %s\n", m.selectedSession.ID))
		if m.selectedSession.Description != "" {
			b.WriteString(fmt.Sprintf("Description: %s\n", m.selectedSession.Description))
		}
		ballCount := m.countBallsForSession(m.selectedSession.ID)
		b.WriteString(fmt.Sprintf("Balls: %d\n", ballCount))
	}

	b.WriteString(fmt.Sprintf("\nIterations: %d (default)\n", 10))
	b.WriteString("\n")

	info := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")). // Gray
		Render("The agent will work on pending balls in this session.")
	b.WriteString(info + "\n\n")

	prompt := lipgloss.NewStyle().
		Bold(true).
		Render("Launch agent? [y/N]")
	b.WriteString(prompt + "\n\n")

	help := lipgloss.NewStyle().
		Faint(true).
		Render("y = confirm | n/Esc = cancel")
	b.WriteString(help)

	return b.String()
}

// renderPanelSearchView renders the search/filter input dialog
func (m Model) renderPanelSearchView() string {
	var b strings.Builder

	// Title based on active panel
	var panelName string
	switch m.activePanel {
	case SessionsPanel:
		panelName = "Sessions"
	case BallsPanel:
		panelName = "Balls"
	}

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("6")).
		Render("Filter " + panelName)
	b.WriteString(title + "\n\n")

	// Show input field
	inputStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("6")).
		Padding(0, 1).
		Width(50)
	b.WriteString(inputStyle.Render(m.textInput.View()) + "\n\n")

	// Show message if any
	if m.message != "" {
		b.WriteString(messageStyle.Render(m.message) + "\n\n")
	}

	// Help
	help := lipgloss.NewStyle().
		Faint(true).
		Render("Enter = apply filter | Esc = cancel")
	b.WriteString(help + "\n")

	// Additional help
	if m.panelSearchQuery != "" {
		helpClear := lipgloss.NewStyle().
			Faint(true).
			Render("Current filter: " + m.panelSearchQuery + " (Ctrl+U to clear in panel)")
		b.WriteString(helpClear)
	}

	return b.String()
}

// renderSessionSelectorView renders the session selection dialog for tagging
func (m Model) renderSessionSelectorView() string {
	var b strings.Builder

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("6")).
		Render("Select Session")
	b.WriteString(title + "\n\n")

	// Show ball context
	if m.editingBall != nil {
		b.WriteString(fmt.Sprintf("Ball: %s\n", m.editingBall.ID))
		b.WriteString(fmt.Sprintf("Intent: %s\n\n", m.editingBall.Intent))

		// Show current sessions/tags
		if len(m.editingBall.Tags) > 0 {
			currentLabel := lipgloss.NewStyle().
				Faint(true).
				Render("Current sessions:")
			b.WriteString(currentLabel + " ")

			tagStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("8"))
			tags := strings.Join(m.editingBall.Tags, ", ")
			b.WriteString(tagStyle.Render(tags) + "\n\n")
		}
	}

	// Show session list
	sessionLabel := lipgloss.NewStyle().
		Bold(true).
		Render("Available sessions:")
	b.WriteString(sessionLabel + "\n\n")

	if len(m.sessionSelectItems) == 0 {
		noSessions := lipgloss.NewStyle().
			Faint(true).
			Render("  No sessions available")
		b.WriteString(noSessions + "\n")
	} else {
		selectedStyle := lipgloss.NewStyle().
			Bold(true).
			Background(lipgloss.Color("240")).
			Foreground(lipgloss.Color("15"))

		normalStyle := lipgloss.NewStyle()

		for i, sess := range m.sessionSelectItems {
			cursor := "  "
			if i == m.sessionSelectIndex {
				cursor = "> "
			}

			line := fmt.Sprintf("%s%s", cursor, sess.ID)
			if sess.Description != "" {
				line += fmt.Sprintf(" - %s", truncate(sess.Description, 40))
			}

			if i == m.sessionSelectIndex {
				b.WriteString(selectedStyle.Render(line) + "\n")
			} else {
				b.WriteString(normalStyle.Render(line) + "\n")
			}
		}
	}

	b.WriteString("\n")

	// Show message if any
	if m.message != "" {
		b.WriteString(messageStyle.Render(m.message) + "\n\n")
	}

	// Help
	help := lipgloss.NewStyle().
		Faint(true).
		Render("j/k or ↑/↓ = navigate | Enter/Space = select | Esc = cancel")
	b.WriteString(help)

	return b.String()
}

// renderTagView renders the tag editing dialog
func (m Model) renderTagView() string {
	var b strings.Builder

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("6")).
		Render("Edit Tags")
	b.WriteString(title + "\n\n")

	// Show ball context
	if m.editingBall != nil {
		b.WriteString(fmt.Sprintf("Ball: %s\n", m.editingBall.ID))
		b.WriteString(fmt.Sprintf("Intent: %s\n\n", m.editingBall.Intent))

		// Show current tags
		if len(m.editingBall.Tags) > 0 {
			tagsLabel := lipgloss.NewStyle().
				Bold(true).
				Render("Current Tags:")
			b.WriteString(tagsLabel + "\n")

			tagStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("12")).
				Background(lipgloss.Color("236")).
				Padding(0, 1)

			for _, tag := range m.editingBall.Tags {
				b.WriteString("  " + tagStyle.Render(tag) + "\n")
			}
			b.WriteString("\n")
		} else {
			noTags := lipgloss.NewStyle().
				Faint(true).
				Render("No tags")
			b.WriteString(noTags + "\n\n")
		}
	}

	// Show input field
	inputStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("6")).
		Padding(0, 1).
		Width(50)
	b.WriteString(inputStyle.Render(m.textInput.View()) + "\n\n")

	// Show message if any
	if m.message != "" {
		b.WriteString(messageStyle.Render(m.message) + "\n\n")
	}

	// Help
	help := lipgloss.NewStyle().
		Faint(true).
		Render("Enter = submit | Esc = cancel\n")
	b.WriteString(help)

	helpAdd := lipgloss.NewStyle().
		Faint(true).
		Render("Type tag name to add | Prefix with - to remove (e.g., -mytag)")
	b.WriteString(helpAdd)

	return b.String()
}

// renderSplitHelpView renders the comprehensive help view for split mode
func (m Model) renderSplitHelpView() string {
	var b strings.Builder

	title := titleStyle.Render("Juggler TUI - Complete Keybindings Reference")
	b.WriteString(title + "\n\n")

	// Build all help sections
	sections := []struct {
		title string
		items []helpItem
	}{
		{
			title: "Global Navigation",
			items: []helpItem{
				{"Tab / l", "Next panel (Sessions → Balls → Activity)"},
				{"Shift+Tab / h", "Previous panel"},
				{"j / ↓", "Move down / Scroll down"},
				{"k / ↑", "Move up / Scroll up"},
				{"Enter", "Select item / Expand"},
				{"Esc", "Back / Deselect / Close"},
				{"q / Ctrl+C", "Quit application"},
			},
		},
		{
			title: "Sessions Panel",
			items: []helpItem{
				{"j/k", "Navigate sessions (auto-selects)"},
				{"Enter", "Select session and go to balls panel"},
				{"a", "Add new session"},
				{"A", "Launch agent for selected session"},
				{"e", "Edit session description"},
				{"d", "Delete session (with confirmation)"},
				{"/", "Filter sessions"},
				{"Ctrl+U", "Clear filter"},
			},
		},
		{
			title: "Balls Panel",
			items: []helpItem{
				{"j/k", "Navigate balls"},
				{"[ / ]", "Switch session (previous / next)"},
				{"Space", "Go back to sessions panel"},
				{"Enter", "Select ball"},
				{"a", "Add new ball (tagged to current session)"},
				{"e", "Edit ball in $EDITOR (YAML format)"},
				{"d", "Delete ball (with confirmation)"},
				{"t", "Tag ball (add to session)"},
				{"s", "Start ball (→ in_progress)"},
				{"c", "Complete ball (→ complete, archives)"},
				{"b", "Block ball (prompts for reason)"},
				{"/", "Filter balls"},
				{"Ctrl+U", "Clear filter"},
			},
		},
		{
			title: "Activity Log Panel",
			items: []helpItem{
				{"j/k", "Scroll one line"},
				{"Ctrl+D", "Page down (half screen)"},
				{"Ctrl+U", "Page up (half screen)"},
				{"gg", "Go to top"},
				{"G", "Go to bottom"},
			},
		},
		{
			title: "Other",
			items: []helpItem{
				{"i", "Toggle bottom pane (activity log ↔ ball details)"},
				{"P", "Toggle project scope (local ↔ all projects)"},
				{"?", "Toggle this help"},
				{"R", "Refresh / Reload data"},
			},
		},
		{
			title: "Input Dialogs",
			items: []helpItem{
				{"Enter", "Submit / Confirm"},
				{"Esc", "Cancel"},
			},
		},
		{
			title: "Delete Confirmation",
			items: []helpItem{
				{"y", "Confirm delete"},
				{"n / Esc", "Cancel delete"},
			},
		},
	}

	// Build content lines
	var lines []string
	for _, section := range sections {
		lines = append(lines, titleStyle.Render(section.title))
		for _, item := range section.items {
			keyStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).Width(15)
			line := fmt.Sprintf("  %s  %s", keyStyle.Render(item.key), item.desc)
			lines = append(lines, line)
		}
		lines = append(lines, "") // Empty line between sections
	}

	// Calculate visible area
	availableHeight := m.height - 5 // Account for title and footer
	if availableHeight < 5 {
		availableHeight = 5
	}

	totalLines := len(lines)
	maxOffset := totalLines - availableHeight
	if maxOffset < 0 {
		maxOffset = 0
	}

	// Clamp scroll offset
	if m.helpScrollOffset > maxOffset {
		m.helpScrollOffset = maxOffset
	}
	if m.helpScrollOffset < 0 {
		m.helpScrollOffset = 0
	}

	// Show scroll indicator at top if not at beginning
	if m.helpScrollOffset > 0 {
		b.WriteString(helpStyle.Render(fmt.Sprintf("  ↑ %d more lines above", m.helpScrollOffset)) + "\n")
		availableHeight--
	}

	// Render visible lines
	endIdx := m.helpScrollOffset + availableHeight
	if endIdx > totalLines {
		endIdx = totalLines
	}

	for i := m.helpScrollOffset; i < endIdx; i++ {
		b.WriteString(lines[i] + "\n")
	}

	// Show scroll indicator at bottom if more content
	remaining := totalLines - endIdx
	if remaining > 0 {
		b.WriteString(helpStyle.Render(fmt.Sprintf("  ↓ %d more lines below", remaining)) + "\n")
	}

	// Footer
	b.WriteString("\n")
	footerStyle := lipgloss.NewStyle().Faint(true)
	b.WriteString(footerStyle.Render("j/k = scroll | ? or Esc = close help"))

	return b.String()
}
