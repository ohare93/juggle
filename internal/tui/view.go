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

	stats := fmt.Sprintf("Total: %d | Ready: %d | Juggling: %d | Dropped: %d | Complete: %d | Filter: %s",
		len(m.balls),
		countByState(m.balls, "ready"),
		countByState(m.balls, "juggling"),
		countByState(m.balls, "dropped"),
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
	b.WriteString(helpStyle.Render("Filter: 1 all • 2 toggle ready • 3 toggle juggling • 4 toggle dropped • 5 toggle complete\n"))
	b.WriteString(helpStyle.Render("Other: r set ready • R refresh • tab cycle state • ? help • q quit\n"))

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
		{"s", "Start ball (ready → juggling:in-air)"},
		{"c", "Complete ball (juggling → complete)"},
		{"d", "Drop ball (→ dropped)"},
		{"x", "Delete ball (with confirmation)"},
		{"p", "Cycle priority (low → medium → high → urgent → low)"},
		{"r", "Set ball to ready state"},
		{"tab", "Cycle state (ready → juggling:in-air → complete → dropped → ready)"},
	}))

	b.WriteString(helpSection("Filters", []helpItem{
		{"1", "Show all balls"},
		{"2", "Toggle ready balls"},
		{"3", "Toggle juggling balls"},
		{"4", "Toggle dropped balls"},
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
