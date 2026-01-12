package tui

import (
	"fmt"
	"strings"
	"time"

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
	case inputBallFormView:
		return m.renderBallFormView()
	case unifiedBallFormView:
		return m.renderUnifiedBallFormView()
	case inputTagView:
		return m.renderTagView()
	case sessionSelectorView:
		return m.renderSessionSelectorView()
	case confirmSplitDelete:
		return m.renderSplitConfirmDelete()
	case confirmAgentLaunch:
		return m.renderAgentLaunchConfirm()
	case confirmAgentCancel:
		return m.renderAgentCancelConfirm()
	case panelSearchView:
		return m.renderPanelSearchView()
	case historyView:
		return m.renderHistoryView()
	case historyOutputView:
		return m.renderHistoryOutputView()
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

// renderAgentCancelConfirm renders the agent cancel confirmation dialog
func (m Model) renderAgentCancelConfirm() string {
	var b strings.Builder

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("1")). // Red
		Render("Cancel Agent")
	b.WriteString(title + "\n\n")

	// Show running agent details
	if m.agentStatus.Running {
		b.WriteString(fmt.Sprintf("Session: %s\n", m.agentStatus.SessionID))
		b.WriteString(fmt.Sprintf("Progress: %d/%d iterations\n",
			m.agentStatus.Iteration,
			m.agentStatus.MaxIterations))
	}

	b.WriteString("\n")

	warning := lipgloss.NewStyle().
		Foreground(lipgloss.Color("3")). // Yellow
		Render("The agent will be terminated immediately.")
	b.WriteString(warning + "\n")

	info := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")). // Gray
		Render("Any completed work will be preserved, but the current task may be interrupted.")
	b.WriteString(info + "\n\n")

	prompt := lipgloss.NewStyle().
		Bold(true).
		Render("Cancel agent? [y/N]")
	b.WriteString(prompt + "\n\n")

	help := lipgloss.NewStyle().
		Faint(true).
		Render("y = terminate agent | n/Esc = continue running")
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

	// Build all help sections - organized by category
	sections := []struct {
		title string
		items []helpItem
	}{
		{
			title: "Navigation",
			items: []helpItem{
				{"Tab / l", "Next panel (Sessions → Balls → Activity)"},
				{"Shift+Tab / h", "Previous panel"},
				{"j / ↓", "Move down / Scroll down"},
				{"k / ↑", "Move up / Scroll up"},
				{"Enter", "Select item / Expand"},
				{"Space", "Go back (in Balls panel)"},
				{"Esc", "Back / Deselect / Close"},
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
			title: "Balls Panel - State Changes (s + key)",
			items: []helpItem{
				{"s", "Start two-key state change sequence:"},
				{"  sc", "  Complete ball (→ complete, archives)"},
				{"  ss", "  Start ball (→ in_progress)"},
				{"  sb", "  Block ball (prompts for reason)"},
				{"  sp", "  Set to pending"},
				{"  sa", "  Archive completed ball"},
			},
		},
		{
			title: "Balls Panel - Toggle Filters (t + key)",
			items: []helpItem{
				{"t", "Start two-key toggle filter sequence:"},
				{"  tc", "  Toggle complete balls visibility"},
				{"  tb", "  Toggle blocked balls visibility"},
				{"  ti", "  Toggle in_progress balls visibility"},
				{"  tp", "  Toggle pending balls visibility"},
				{"  ta", "  Show all states"},
			},
		},
		{
			title: "Balls Panel - Other Actions",
			items: []helpItem{
				{"j/k", "Navigate balls"},
				{"a", "Add new ball (tagged to current session)"},
				{"e", "Edit ball in $EDITOR (YAML format)"},
				{"d", "Delete ball (with confirmation)"},
				{"[ / ]", "Switch session (previous / next)"},
				{"o", "Toggle sort order (ID↑ → ID↓ → Priority → Activity)"},
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
			title: "Balls Panel - View Columns (v + key)",
			items: []helpItem{
				{"v", "Start two-key view columns sequence:"},
				{"  vp", "  Toggle priority column visibility"},
				{"  vt", "  Toggle tags column visibility"},
				{"  vs", "  Toggle tests state column visibility"},
				{"  va", "  Toggle all optional columns on/off"},
			},
		},
		{
			title: "View Options",
			items: []helpItem{
				{"i", "Cycle bottom pane (activity → detail → split → activity)"},
				{"O", "Toggle agent output panel (shows live agent stdout)"},
				{"P", "Toggle project scope (local ↔ all projects)"},
				{"R", "Refresh / Reload data"},
				{"?", "Toggle this help"},
			},
		},
		{
			title: "Agent Control",
			items: []helpItem{
				{"A", "Launch agent for selected session"},
				{"X", "Cancel running agent (with confirmation)"},
				{"O", "Toggle agent output visibility"},
				{"H", "View agent run history"},
			},
		},
		{
			title: "Bottom Pane Modes",
			items: []helpItem{
				{"[Act]", "Activity log - shows recent actions"},
				{"[Detail]", "Ball details - shows full ball info with ACs"},
				{"[Split]", "Split view - shows both details and activity"},
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
		{
			title: "Quit",
			items: []helpItem{
				{"q / Ctrl+C", "Quit application"},
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

// renderBallFormView renders the multi-field ball creation form
func (m Model) renderBallFormView() string {
	var b strings.Builder

	titleStyled := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("6")).
		Render("Create New Ball")
	b.WriteString(titleStyled + "\n\n")

	// Show the intent that was entered
	b.WriteString(fmt.Sprintf("Intent: %s\n\n", m.pendingBallIntent))

	// Priority options
	priorities := []string{"low", "medium", "high", "urgent"}

	// Build sessions list for display
	sessionOptions := []string{"(none)"}
	for _, sess := range m.sessions {
		if sess.ID != PseudoSessionAll && sess.ID != PseudoSessionUntagged {
			sessionOptions = append(sessionOptions, sess.ID)
		}
	}

	// Field labels and values (state removed - always pending)
	fields := []struct {
		label    string
		options  []string
		selected int
		isText   bool
		textVal  string
	}{
		{"Priority", priorities, m.pendingBallPriority, false, ""},
		{"Tags", nil, 0, true, m.pendingBallTags},
		{"Session", sessionOptions, m.pendingBallSession, false, ""},
	}

	selectedStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	normalStyle := lipgloss.NewStyle()
	activeFieldStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("2"))
	optionSelectedStyle := lipgloss.NewStyle().Bold(true).Background(lipgloss.Color("6")).Foreground(lipgloss.Color("0"))
	optionNormalStyle := lipgloss.NewStyle().Faint(true)

	for i, field := range fields {
		// Field label - highlight if active
		labelStyle := normalStyle
		if i == m.pendingBallFormField {
			labelStyle = activeFieldStyle
		}
		b.WriteString(labelStyle.Render(fmt.Sprintf("%s: ", field.label)))

		if field.isText {
			// Text field - show text input when active
			if i == m.pendingBallFormField {
				b.WriteString(m.textInput.View())
			} else {
				if field.textVal == "" {
					b.WriteString(optionNormalStyle.Render("(empty)"))
				} else {
					b.WriteString(field.textVal)
				}
			}
		} else {
			// Selection field - show all options with selected one highlighted
			for j, opt := range field.options {
				if j > 0 {
					b.WriteString(" | ")
				}
				if j == field.selected {
					if i == m.pendingBallFormField {
						b.WriteString(optionSelectedStyle.Render(opt))
					} else {
						b.WriteString(selectedStyle.Render(opt))
					}
				} else {
					b.WriteString(optionNormalStyle.Render(opt))
				}
			}
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")

	// Show message if any
	if m.message != "" {
		b.WriteString(messageStyle.Render(m.message) + "\n\n")
	}

	// Help
	help := lipgloss.NewStyle().
		Faint(true).
		Render("←/→ or Tab = cycle options | ↑/↓ or j/k = change field | Enter = continue to ACs | Esc = cancel")
	b.WriteString(help)

	return b.String()
}

// renderHistoryView renders the agent run history view
func (m Model) renderHistoryView() string {
	var b strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("33")).
		MarginBottom(1)
	b.WriteString(titleStyle.Render("📜 Agent Run History") + "\n\n")

	if len(m.agentHistory) == 0 {
		b.WriteString("No agent runs recorded yet.\n\n")
		b.WriteString(helpStyle.Render("Press H or Esc to return"))
		return b.String()
	}

	// Column headers
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("245"))
	b.WriteString(headerStyle.Render(fmt.Sprintf("  %-19s  %-15s  %-6s  %-14s  %-8s  %-7s\n",
		"Date", "Session", "Iter", "Result", "Duration", "Balls")))
	b.WriteString(strings.Repeat("─", 80) + "\n")

	// Calculate visible area
	visibleLines := m.height - 10 // Account for header, footer
	if visibleLines < 5 {
		visibleLines = 5
	}

	// Determine range to display
	startIdx := m.historyScrollOffset
	endIdx := startIdx + visibleLines
	if endIdx > len(m.agentHistory) {
		endIdx = len(m.agentHistory)
	}

	// Render history entries
	for i := startIdx; i < endIdx; i++ {
		record := m.agentHistory[i]

		// Format the entry
		cursor := "  "
		lineStyle := lipgloss.NewStyle()
		if i == m.historyCursor {
			cursor = "▶ "
			lineStyle = lipgloss.NewStyle().Background(lipgloss.Color("236")).Foreground(lipgloss.Color("252"))
		}

		// Format date
		dateStr := record.StartedAt.Format("2006-01-02 15:04:05")

		// Format session (truncate if needed)
		sessionStr := record.SessionID
		if len(sessionStr) > 15 {
			sessionStr = sessionStr[:12] + "..."
		}

		// Format iterations
		iterStr := fmt.Sprintf("%d/%d", record.Iterations, record.MaxIterations)

		// Format result with styling
		resultStr := formatHistoryResult(record.Result)

		// Format duration
		duration := record.Duration()
		durationStr := formatDuration(duration)

		// Format balls
		ballsStr := fmt.Sprintf("%d/%d", record.BallsComplete, record.BallsTotal)

		line := fmt.Sprintf("%s%-19s  %-15s  %-6s  %-14s  %-8s  %-7s",
			cursor, dateStr, sessionStr, iterStr, resultStr, durationStr, ballsStr)
		b.WriteString(lineStyle.Render(line) + "\n")
	}

	// Scroll indicators
	if m.historyScrollOffset > 0 {
		b.WriteString(helpStyle.Render(fmt.Sprintf("  ↑ %d more above\n", m.historyScrollOffset)))
	}
	if endIdx < len(m.agentHistory) {
		b.WriteString(helpStyle.Render(fmt.Sprintf("  ↓ %d more below\n", len(m.agentHistory)-endIdx)))
	}

	b.WriteString("\n")

	// Show details for selected record
	if m.historyCursor < len(m.agentHistory) {
		record := m.agentHistory[m.historyCursor]
		detailStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
		b.WriteString(detailStyle.Render("─── Selected Run Details ───") + "\n")

		if record.BlockedReason != "" {
			b.WriteString(detailStyle.Render(fmt.Sprintf("Blocked: %s\n", record.BlockedReason)))
		}
		if record.TimeoutMessage != "" {
			b.WriteString(detailStyle.Render(fmt.Sprintf("Timeout: %s\n", record.TimeoutMessage)))
		}
		if record.ErrorMessage != "" {
			b.WriteString(detailStyle.Render(fmt.Sprintf("Error: %s\n", record.ErrorMessage)))
		}
		if record.TotalWaitTime > 0 {
			b.WriteString(detailStyle.Render(fmt.Sprintf("Rate Limit Wait: %s\n", formatDuration(record.TotalWaitTime))))
		}
		if record.OutputFile != "" {
			b.WriteString(detailStyle.Render(fmt.Sprintf("Output: %s\n", record.OutputFile)))
		}
	}

	b.WriteString("\n")

	// Help
	help := lipgloss.NewStyle().Faint(true).Render("j/k = navigate | Enter = view output | H/Esc = close | gg/G = top/bottom")
	b.WriteString(help)

	return b.String()
}

// formatHistoryResult formats the result field with appropriate styling
func formatHistoryResult(result string) string {
	switch result {
	case "complete":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Render("✓ Complete")
	case "blocked":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Render("⊘ Blocked")
	case "timeout":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("⏱ Timeout")
	case "max_iterations":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Render("⟳ MaxIter")
	case "rate_limit":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("⚠ RateLimit")
	case "cancelled":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("✗ Cancelled")
	case "error":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("✗ Error")
	default:
		return result
	}
}

// formatDuration formats a duration into a human-readable string
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
}

// renderHistoryOutputView renders the output file content
func (m Model) renderHistoryOutputView() string {
	var b strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("33")).
		MarginBottom(1)

	if m.historyCursor < len(m.agentHistory) {
		record := m.agentHistory[m.historyCursor]
		b.WriteString(titleStyle.Render(fmt.Sprintf("📄 Output: %s (%s)", record.SessionID, record.StartedAt.Format("2006-01-02 15:04"))) + "\n")
	} else {
		b.WriteString(titleStyle.Render("📄 Agent Output") + "\n")
	}
	b.WriteString(strings.Repeat("─", 80) + "\n")

	// Split content into lines
	lines := strings.Split(m.historyOutput, "\n")

	// Calculate visible area
	visibleLines := m.height - 6 // Account for header, footer
	if visibleLines < 5 {
		visibleLines = 5
	}

	// Clamp offset
	maxOffset := len(lines) - visibleLines
	if maxOffset < 0 {
		maxOffset = 0
	}
	offset := m.historyOutputOffset
	if offset > maxOffset {
		offset = maxOffset
	}

	// Render visible lines
	endIdx := offset + visibleLines
	if endIdx > len(lines) {
		endIdx = len(lines)
	}

	for i := offset; i < endIdx; i++ {
		b.WriteString(lines[i] + "\n")
	}

	// Scroll indicators
	if offset > 0 {
		b.WriteString(helpStyle.Render(fmt.Sprintf("↑ %d lines above\n", offset)))
	}
	if endIdx < len(lines) {
		b.WriteString(helpStyle.Render(fmt.Sprintf("↓ %d lines below\n", len(lines)-endIdx)))
	}

	b.WriteString("\n")

	// Help
	help := lipgloss.NewStyle().Faint(true).Render("j/k = scroll | ctrl+d/u = page | gg/G = top/bottom | b/Esc = back to history")
	b.WriteString(help)

	return b.String()
}

// renderUnifiedBallFormView renders the unified ball creation form with all fields visible
func (m Model) renderUnifiedBallFormView() string {
	var b strings.Builder

	titleStyled := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("6")).
		Render("Create New Ball")
	b.WriteString(titleStyled + "\n\n")

	// Field constants
	const (
		fieldIntent   = 0
		fieldPriority = 1
		fieldTags     = 2
		fieldSession  = 3
		fieldACStart  = 4 // ACs start at index 4
	)

	// Priority options
	priorities := []string{"low", "medium", "high", "urgent"}

	// Build sessions list for display
	sessionOptions := []string{"(none)"}
	for _, sess := range m.sessions {
		if sess.ID != PseudoSessionAll && sess.ID != PseudoSessionUntagged {
			sessionOptions = append(sessionOptions, sess.ID)
		}
	}

	// Styles
	activeFieldStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("2"))
	normalStyle := lipgloss.NewStyle()
	selectedStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	optionSelectedStyle := lipgloss.NewStyle().Bold(true).Background(lipgloss.Color("6")).Foreground(lipgloss.Color("0"))
	optionNormalStyle := lipgloss.NewStyle().Faint(true)
	acNumberStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	editingACStyle := lipgloss.NewStyle().Bold(true).Background(lipgloss.Color("240"))

	// --- Intent field ---
	labelStyle := normalStyle
	if m.pendingBallFormField == fieldIntent {
		labelStyle = activeFieldStyle
	}
	b.WriteString(labelStyle.Render("Intent: "))
	if m.pendingBallFormField == fieldIntent {
		b.WriteString(m.textInput.View())
	} else {
		if m.pendingBallIntent == "" {
			b.WriteString(optionNormalStyle.Render("(empty)"))
		} else {
			b.WriteString(m.pendingBallIntent)
		}
	}
	b.WriteString("\n")

	// --- Priority field ---
	labelStyle = normalStyle
	if m.pendingBallFormField == fieldPriority {
		labelStyle = activeFieldStyle
	}
	b.WriteString(labelStyle.Render("Priority: "))
	for j, opt := range priorities {
		if j > 0 {
			b.WriteString(" | ")
		}
		if j == m.pendingBallPriority {
			if m.pendingBallFormField == fieldPriority {
				b.WriteString(optionSelectedStyle.Render(opt))
			} else {
				b.WriteString(selectedStyle.Render(opt))
			}
		} else {
			b.WriteString(optionNormalStyle.Render(opt))
		}
	}
	b.WriteString("\n")

	// --- Tags field ---
	labelStyle = normalStyle
	if m.pendingBallFormField == fieldTags {
		labelStyle = activeFieldStyle
	}
	b.WriteString(labelStyle.Render("Tags: "))
	if m.pendingBallFormField == fieldTags {
		b.WriteString(m.textInput.View())
	} else {
		if m.pendingBallTags == "" {
			b.WriteString(optionNormalStyle.Render("(empty)"))
		} else {
			b.WriteString(m.pendingBallTags)
		}
	}
	b.WriteString("\n")

	// --- Session field ---
	labelStyle = normalStyle
	if m.pendingBallFormField == fieldSession {
		labelStyle = activeFieldStyle
	}
	b.WriteString(labelStyle.Render("Session: "))
	for j, opt := range sessionOptions {
		if j > 0 {
			b.WriteString(" | ")
		}
		if j == m.pendingBallSession {
			if m.pendingBallFormField == fieldSession {
				b.WriteString(optionSelectedStyle.Render(opt))
			} else {
				b.WriteString(selectedStyle.Render(opt))
			}
		} else {
			b.WriteString(optionNormalStyle.Render(opt))
		}
	}
	b.WriteString("\n\n")

	// --- Acceptance Criteria section ---
	acLabel := normalStyle
	if m.pendingBallFormField >= fieldACStart {
		acLabel = activeFieldStyle
	}
	b.WriteString(acLabel.Render("Acceptance Criteria:") + "\n")

	// Show existing ACs with ability to edit
	for i, ac := range m.pendingAcceptanceCriteria {
		acFieldIndex := fieldACStart + i
		if m.pendingBallFormField == acFieldIndex {
			// This AC is being edited
			b.WriteString(acNumberStyle.Render(fmt.Sprintf("  %d. ", i+1)))
			b.WriteString(m.textInput.View())
		} else {
			b.WriteString(acNumberStyle.Render(fmt.Sprintf("  %d. ", i+1)))
			b.WriteString(ac)
		}
		b.WriteString("\n")
	}

	// Show new AC input field (always at the end)
	newACFieldIndex := fieldACStart + len(m.pendingAcceptanceCriteria)
	if m.pendingBallFormField == newACFieldIndex {
		// Show input for new AC
		b.WriteString(editingACStyle.Render("  + "))
		b.WriteString(m.textInput.View())
		b.WriteString("\n")
	} else {
		// Show placeholder for adding new AC
		b.WriteString(optionNormalStyle.Render("  + (press Enter to add)") + "\n")
	}

	b.WriteString("\n")

	// Show message if any
	if m.message != "" {
		b.WriteString(messageStyle.Render(m.message) + "\n\n")
	}

	// Help
	help := lipgloss.NewStyle().
		Faint(true).
		Render("↑/↓ or j/k = navigate | ←/→ = cycle options | Enter = confirm/add AC | Ctrl+Enter = create ball | Esc = cancel")
	b.WriteString(help)

	return b.String()
}
