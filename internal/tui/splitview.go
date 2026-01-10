package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/ohare93/juggle/internal/session"
)

// Panel dimensions as fractions of available space
const (
	leftPanelRatio   = 0.25 // 25% for sessions
	rightPanelRatio  = 0.75 // 75% for balls+todos
	bottomPanelRows  = 6    // Fixed height for activity log
	minLeftWidth     = 20
	minRightWidth    = 40
)

// Panel styles
var (
	panelBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("240"))

	activePanelBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("6")) // Cyan for active

	panelTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("6")).
			Padding(0, 1)

	activePanelTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("3")). // Yellow for active
				Padding(0, 1)

	sessionItemStyle = lipgloss.NewStyle().
				Padding(0, 1)

	selectedSessionItemStyle = lipgloss.NewStyle().
					Padding(0, 1).
					Background(lipgloss.Color("240")).
					Bold(true)

	activityLogStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("8"))
)

// renderSplitView renders the three-panel split view
func (m Model) renderSplitView() string {
	// Calculate dimensions
	mainHeight := m.height - bottomPanelRows - 4 // Account for borders and status
	leftWidth := int(float64(m.width) * leftPanelRatio)
	rightWidth := m.width - leftWidth - 3 // Account for borders

	// Enforce minimum widths
	if leftWidth < minLeftWidth {
		leftWidth = minLeftWidth
		rightWidth = m.width - leftWidth - 3
	}
	if rightWidth < minRightWidth && m.width > minLeftWidth+minRightWidth+3 {
		rightWidth = minRightWidth
		leftWidth = m.width - rightWidth - 3
	}

	// Render each panel
	sessionsPanel := m.renderSessionsPanel(leftWidth-2, mainHeight-2)
	ballsPanel := m.renderBallsPanel(rightWidth-2, mainHeight-2)
	activityPanel := m.renderActivityPanel(m.width-2, bottomPanelRows-2)

	// Apply panel styling based on active panel
	var sessionsBorder, ballsBorder lipgloss.Style
	if m.activePanel == SessionsPanel {
		sessionsBorder = activePanelBorderStyle.Width(leftWidth).Height(mainHeight)
	} else {
		sessionsBorder = panelBorderStyle.Width(leftWidth).Height(mainHeight)
	}

	if m.activePanel == BallsPanel || m.activePanel == TodosPanel {
		ballsBorder = activePanelBorderStyle.Width(rightWidth).Height(mainHeight)
	} else {
		ballsBorder = panelBorderStyle.Width(rightWidth).Height(mainHeight)
	}

	activityBorder := panelBorderStyle.Width(m.width).Height(bottomPanelRows)

	// Build the layout
	topRow := lipgloss.JoinHorizontal(
		lipgloss.Top,
		sessionsBorder.Render(sessionsPanel),
		ballsBorder.Render(ballsPanel),
	)

	// Status bar
	statusBar := m.renderStatusBar()

	// Combine all sections
	return lipgloss.JoinVertical(
		lipgloss.Left,
		topRow,
		activityBorder.Render(activityPanel),
		statusBar,
	)
}

// renderSessionsPanel renders the left panel with session list
func (m Model) renderSessionsPanel(width, height int) string {
	var b strings.Builder

	// Get filtered sessions
	sessions := m.filterSessions()

	// Title with filter indicator
	title := "Sessions"
	if m.panelSearchActive && m.activePanel == SessionsPanel {
		title = fmt.Sprintf("Sessions [%s]", m.panelSearchQuery)
	}
	if m.activePanel == SessionsPanel {
		b.WriteString(activePanelTitleStyle.Render(truncate(title, width-2)) + "\n")
	} else {
		b.WriteString(panelTitleStyle.Render(truncate(title, width-2)) + "\n")
	}
	b.WriteString(strings.Repeat("─", width) + "\n")

	if len(sessions) == 0 {
		if m.panelSearchActive {
			b.WriteString(helpStyle.Render("  No matching sessions\n"))
			b.WriteString(helpStyle.Render("  Ctrl+U to clear filter"))
		} else {
			b.WriteString(helpStyle.Render("  No sessions\n"))
			b.WriteString(helpStyle.Render("  Press 'a' to create"))
		}
	} else {
		// Calculate available lines for sessions
		availableLines := height - 3 // Account for title and separator

		for i, sess := range sessions {
			if i >= availableLines {
				remaining := len(sessions) - availableLines
				b.WriteString(helpStyle.Render(fmt.Sprintf("  ... +%d more", remaining)))
				break
			}

			// Count balls for this session
			ballCount := m.countBallsForSession(sess.ID)

			line := fmt.Sprintf("%-*s (%d)",
				width-6,
				truncate(sess.ID, width-6),
				ballCount,
			)

			if i == m.sessionCursor && m.activePanel == SessionsPanel {
				b.WriteString(selectedSessionItemStyle.Render(line) + "\n")
			} else if m.selectedSession != nil && m.selectedSession.ID == sess.ID {
				// Highlight selected session even when not in sessions panel
				b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Render(line) + "\n")
			} else {
				b.WriteString(sessionItemStyle.Render(line) + "\n")
			}
		}
	}

	return b.String()
}

// renderBallsPanel renders the right panel with balls and optionally todos
func (m Model) renderBallsPanel(width, height int) string {
	var b strings.Builder

	// Get filtered balls for current session
	balls := m.filterBallsForSession()

	// Title with filter indicator
	var title string
	if m.selectedSession != nil {
		title = fmt.Sprintf("Balls: %s", m.selectedSession.ID)
	} else {
		title = "Balls: All"
	}
	if m.panelSearchActive && m.activePanel == BallsPanel {
		title = fmt.Sprintf("%s [%s]", title, m.panelSearchQuery)
	}

	if m.activePanel == BallsPanel {
		b.WriteString(activePanelTitleStyle.Render(truncate(title, width-2)) + "\n")
	} else {
		b.WriteString(panelTitleStyle.Render(truncate(title, width-2)) + "\n")
	}
	b.WriteString(strings.Repeat("─", width) + "\n")

	if len(balls) == 0 {
		if m.panelSearchActive {
			b.WriteString(helpStyle.Render("  No matching balls\n"))
			b.WriteString(helpStyle.Render("  Ctrl+U to clear filter"))
		} else {
			b.WriteString(helpStyle.Render("  No balls"))
			if m.selectedSession != nil {
				b.WriteString(helpStyle.Render(fmt.Sprintf(" in session '%s'", m.selectedSession.ID)))
			}
		}
		b.WriteString("\n")
		return b.String()
	}

	// Split height between balls and todos if a ball is selected
	var ballsHeight, todosHeight int
	if m.selectedBall != nil && m.activePanel == TodosPanel {
		ballsHeight = (height - 4) / 2
		todosHeight = height - 4 - ballsHeight
	} else {
		ballsHeight = height - 4
		todosHeight = 0
	}

	// Render balls list
	for i, ball := range balls {
		if i >= ballsHeight {
			remaining := len(balls) - ballsHeight
			b.WriteString(helpStyle.Render(fmt.Sprintf("  ... +%d more", remaining)) + "\n")
			break
		}

		stateIcon := getStateIcon(ball.State)
		var line string
		if ball.State == session.StateBlocked && ball.BlockedReason != "" {
			// Show blocked reason inline for blocked balls
			intent := truncate(ball.Intent, width-25)
			reason := truncate(ball.BlockedReason, width-len(intent)-15)
			line = fmt.Sprintf("%s %s [%s]",
				stateIcon,
				intent,
				reason,
			)
		} else {
			line = fmt.Sprintf("%s %-*s %s",
				stateIcon,
				width-15,
				truncate(ball.Intent, width-15),
				string(ball.State),
			)
		}
		line = styleBallByState(ball, truncate(line, width-2))

		if i == m.cursor && m.activePanel == BallsPanel {
			b.WriteString(selectedBallStyle.Render(line) + "\n")
		} else if m.selectedBall != nil && m.selectedBall.ID == ball.ID {
			b.WriteString(lipgloss.NewStyle().Background(lipgloss.Color("235")).Render(line) + "\n")
		} else {
			b.WriteString(ballStyle.Render(line) + "\n")
		}
	}

	// Render todos section if ball is selected and in todos panel
	if m.selectedBall != nil && todosHeight > 0 {
		b.WriteString("\n")
		if m.activePanel == TodosPanel {
			b.WriteString(activePanelTitleStyle.Render("Todos") + "\n")
		} else {
			b.WriteString(panelTitleStyle.Render("Todos") + "\n")
		}
		b.WriteString(strings.Repeat("─", width) + "\n")

		if len(m.selectedBall.Todos) == 0 {
			b.WriteString(helpStyle.Render("  No todos") + "\n")
		} else {
			for i, todo := range m.selectedBall.Todos {
				if i >= todosHeight-3 {
					remaining := len(m.selectedBall.Todos) - (todosHeight - 3)
					b.WriteString(helpStyle.Render(fmt.Sprintf("  ... +%d more", remaining)) + "\n")
					break
				}

				checkbox := "[ ]"
				style := lipgloss.NewStyle()
				if todo.Done {
					checkbox = "[x]"
					style = style.Foreground(lipgloss.Color("8")).Strikethrough(true)
				}

				todoLine := fmt.Sprintf("%s %s", checkbox, truncate(todo.Text, width-6))

				if i == m.todoCursor && m.activePanel == TodosPanel {
					b.WriteString(selectedBallStyle.Render(todoLine) + "\n")
				} else {
					b.WriteString(style.Render(todoLine) + "\n")
				}
			}
		}
	}

	return b.String()
}

// renderActivityPanel renders the bottom activity log panel
func (m Model) renderActivityPanel(width, height int) string {
	var b strings.Builder

	b.WriteString(panelTitleStyle.Render("Activity Log") + "\n")

	if len(m.activityLog) == 0 {
		b.WriteString(activityLogStyle.Render("  No activity yet"))
		return b.String()
	}

	// Show most recent entries that fit
	startIdx := len(m.activityLog) - height
	if startIdx < 0 {
		startIdx = 0
	}

	for i := startIdx; i < len(m.activityLog); i++ {
		entry := m.activityLog[i]
		timeStr := entry.Time.Format("15:04:05")
		line := fmt.Sprintf("  %s %s", timeStr, truncate(entry.Message, width-12))
		b.WriteString(activityLogStyle.Render(line) + "\n")
	}

	return b.String()
}

// renderStatusBar renders the bottom status bar with keybindings
func (m Model) renderStatusBar() string {
	var hints []string

	switch m.activePanel {
	case SessionsPanel:
		hints = []string{"Tab:panels", "j/k:navigate", "Enter:select", "a:add", "/:filter", "?:help", "q:quit"}
	case BallsPanel:
		hints = []string{"Tab:panels", "j/k:navigate", "Enter:todos", "s:start", "c:complete", "/:filter", "?:help", "q:quit"}
	case TodosPanel:
		hints = []string{"Tab:panels", "j/k:navigate", "Esc:back", "Space:toggle", "/:filter", "?:help", "q:quit"}
	}

	status := strings.Join(hints, " | ")

	// Add filter indicator if active
	if m.panelSearchActive {
		status = fmt.Sprintf("[Filter: %s] %s", m.panelSearchQuery, status)
	}

	// Add message if present
	if m.message != "" {
		status = messageStyle.Render(m.message) + "  " + status
	}

	return helpStyle.Render(status)
}

// countBallsForSession counts balls that belong to a session
func (m Model) countBallsForSession(sessionID string) int {
	count := 0
	for _, ball := range m.filteredBalls {
		for _, tag := range ball.Tags {
			if tag == sessionID {
				count++
				break
			}
		}
	}
	return count
}

// getStateIcon returns an icon for the ball state
func getStateIcon(state session.BallState) string {
	switch state {
	case session.StatePending:
		return "○"
	case session.StateInProgress:
		return "●"
	case session.StateComplete:
		return "✓"
	case session.StateBlocked:
		return "✗"
	default:
		return "?"
	}
}
