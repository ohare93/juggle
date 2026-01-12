package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderConfirmDeleteView() string {
	if len(m.filteredBalls) == 0 {
		return "No ball selected"
	}

	ball := m.filteredBalls[m.cursor]

	var b strings.Builder

	// Title
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("1")). // Red
		Render("⚠️  DELETE BALL")
	b.WriteString(title + "\n\n")

	// Ball info
	b.WriteString(fmt.Sprintf("ID:       %s\n", ball.ID))
	b.WriteString(fmt.Sprintf("Title:    %s\n", ball.Title))
	b.WriteString(fmt.Sprintf("State:    %s\n", formatState(ball)))
	b.WriteString(fmt.Sprintf("Priority: %s\n", ball.Priority))
	if len(ball.Tags) > 0 {
		b.WriteString(fmt.Sprintf("Tags:     %s\n", strings.Join(ball.Tags, ", ")))
	}
	b.WriteString("\n")

	// Warning
	warning := lipgloss.NewStyle().
		Foreground(lipgloss.Color("3")). // Yellow
		Render("This will permanently delete the ball.")
	b.WriteString(warning + "\n\n")

	// Prompt
	prompt := lipgloss.NewStyle().
		Bold(true).
		Render("Delete this ball? [y/N]")
	b.WriteString(prompt + "\n\n")

	// Help
	help := lipgloss.NewStyle().
		Faint(true).
		Render("y = confirm | n/Esc = cancel")
	b.WriteString(help)

	return b.String()
}
