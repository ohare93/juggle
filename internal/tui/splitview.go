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
	rightPanelRatio  = 0.75 // 75% for balls
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
	// Guard against rendering before window size is received
	if m.width < minLeftWidth+minRightWidth+10 || m.height < bottomPanelRows+10 {
		return "Loading..."
	}

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

	// Render bottom panel based on mode
	var bottomPanel string
	if m.bottomPaneMode == BottomPaneDetail {
		bottomPanel = m.renderBallDetailPanel(m.width-2, bottomPanelRows-2)
	} else {
		bottomPanel = m.renderActivityPanel(m.width-2, bottomPanelRows-2)
	}

	// Apply panel styling based on active panel
	var sessionsBorder, ballsBorder lipgloss.Style
	if m.activePanel == SessionsPanel {
		sessionsBorder = activePanelBorderStyle.Width(leftWidth).Height(mainHeight)
	} else {
		sessionsBorder = panelBorderStyle.Width(leftWidth).Height(mainHeight)
	}

	if m.activePanel == BallsPanel {
		ballsBorder = activePanelBorderStyle.Width(rightWidth).Height(mainHeight)
	} else {
		ballsBorder = panelBorderStyle.Width(rightWidth).Height(mainHeight)
	}

	var activityBorder lipgloss.Style
	if m.activePanel == ActivityPanel {
		activityBorder = activePanelBorderStyle.Width(m.width).Height(bottomPanelRows)
	} else {
		activityBorder = panelBorderStyle.Width(m.width).Height(bottomPanelRows)
	}

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
		activityBorder.Render(bottomPanel),
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

			// Format display name for pseudo-sessions
			displayName := sess.ID
			if sess.ID == PseudoSessionAll {
				displayName = "★ All"
			} else if sess.ID == PseudoSessionUntagged {
				displayName = "○ Untagged"
			}

			line := fmt.Sprintf("%-*s (%d)",
				width-6,
				truncate(displayName, width-6),
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
		// Use display names for pseudo-sessions
		switch m.selectedSession.ID {
		case PseudoSessionAll:
			title = "Balls: All"
		case PseudoSessionUntagged:
			title = "Balls: Untagged"
		default:
			title = fmt.Sprintf("Balls: %s", m.selectedSession.ID)
		}
	} else {
		title = "Balls: (none selected)"
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

	// Calculate available height for balls
	ballsHeight := height - 4

	// Render balls list
	for i, ball := range balls {
		if i >= ballsHeight {
			remaining := len(balls) - ballsHeight
			b.WriteString(helpStyle.Render(fmt.Sprintf("  ... +%d more", remaining)) + "\n")
			break
		}

		stateIcon := getStateIcon(ball.State)
		var line string

		// Add project prefix when showing all projects
		projectPrefix := ""
		if !m.localOnly {
			// Extract project name from working directory
			projectName := ball.ID // ID already contains project prefix
			if idx := strings.LastIndex(ball.WorkingDir, "/"); idx >= 0 {
				projectName = ball.WorkingDir[idx+1:]
			}
			projectPrefix = projectName + ": "
		}

		if ball.State == session.StateBlocked && ball.BlockedReason != "" {
			// Show blocked reason inline for blocked balls
			intent := truncate(ball.Intent, width-25-len(projectPrefix))
			reason := truncate(ball.BlockedReason, width-len(intent)-15-len(projectPrefix))
			line = fmt.Sprintf("%s %s%s [%s]",
				stateIcon,
				projectPrefix,
				intent,
				reason,
			)
		} else {
			availWidth := width - 15 - len(projectPrefix)
			line = fmt.Sprintf("%s %s%-*s %s",
				stateIcon,
				projectPrefix,
				availWidth,
				truncate(ball.Intent, availWidth),
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

	return b.String()
}

// renderActivityPanel renders the bottom activity log panel
func (m Model) renderActivityPanel(width, height int) string {
	var b strings.Builder

	// Title with active indicator
	title := "Activity Log"
	if m.activePanel == ActivityPanel {
		// Show scroll position and hints when active
		if len(m.activityLog) > height {
			title = fmt.Sprintf("Activity Log [%d/%d]", m.activityLogOffset+1, len(m.activityLog))
		}
		b.WriteString(activePanelTitleStyle.Render(title) + "\n")
	} else {
		b.WriteString(panelTitleStyle.Render(title) + "\n")
	}

	if len(m.activityLog) == 0 {
		b.WriteString(activityLogStyle.Render("  No activity yet"))
		return b.String()
	}

	// Calculate visible range using scroll offset
	visibleLines := height - 1 // Account for title
	if visibleLines < 1 {
		visibleLines = 1
	}

	startIdx := m.activityLogOffset
	endIdx := startIdx + visibleLines
	if endIdx > len(m.activityLog) {
		endIdx = len(m.activityLog)
	}

	// Show scroll indicator at top if not at beginning
	if startIdx > 0 && m.activePanel == ActivityPanel {
		b.WriteString(helpStyle.Render(fmt.Sprintf("  ↑ %d more entries above", startIdx)) + "\n")
		endIdx-- // Reduce visible entries to make room for indicator
	}

	for i := startIdx; i < endIdx; i++ {
		entry := m.activityLog[i]
		timeStr := entry.Time.Format("15:04:05")
		line := fmt.Sprintf("  %s %s", timeStr, truncate(entry.Message, width-12))
		b.WriteString(activityLogStyle.Render(line) + "\n")
	}

	// Show scroll indicator at bottom if more entries
	remaining := len(m.activityLog) - endIdx
	if remaining > 0 && m.activePanel == ActivityPanel {
		b.WriteString(helpStyle.Render(fmt.Sprintf("  ↓ %d more entries below", remaining)))
	}

	return b.String()
}

// renderBallDetailPanel renders the bottom panel with highlighted ball details
func (m Model) renderBallDetailPanel(width, height int) string {
	var b strings.Builder

	// Title
	title := "Ball Details"
	if m.activePanel == ActivityPanel {
		b.WriteString(activePanelTitleStyle.Render(title) + "\n")
	} else {
		b.WriteString(panelTitleStyle.Render(title) + "\n")
	}

	// Get the currently highlighted ball based on active panel
	var ball *session.Ball
	if m.activePanel == BallsPanel {
		balls := m.filterBallsForSession()
		if m.cursor < len(balls) {
			ball = balls[m.cursor]
		}
	}
	if ball == nil && m.selectedBall != nil {
		ball = m.selectedBall
	}

	if ball == nil {
		b.WriteString(helpStyle.Render("  No ball selected - navigate to a ball to see details"))
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("  Press 'i' to toggle back to activity log"))
		return b.String()
	}

	// Calculate column widths for two-column layout
	labelWidth := 12
	availableWidth := width - labelWidth - 4

	// Render ball properties in a compact format
	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).Width(labelWidth)
	valueStyle := lipgloss.NewStyle()

	// Row 1: ID and State
	idLabel := labelStyle.Render("ID:")
	idValue := truncate(ball.ID, 20)
	stateLabel := labelStyle.Render("State:")
	stateValue := string(ball.State)
	if ball.State == session.StateBlocked && ball.BlockedReason != "" {
		stateValue += " (" + truncate(ball.BlockedReason, 20) + ")"
	}
	b.WriteString(fmt.Sprintf("  %s %s    %s %s\n", idLabel, valueStyle.Render(idValue), stateLabel, styleBallByState(ball, stateValue)))

	// Row 2: Priority and Intent
	priorityLabel := labelStyle.Render("Priority:")
	priorityValue := string(ball.Priority)
	intentLabel := labelStyle.Render("Intent:")
	intentValue := truncate(ball.Intent, availableWidth-30)
	b.WriteString(fmt.Sprintf("  %s %s    %s %s\n", priorityLabel, valueStyle.Render(priorityValue), intentLabel, valueStyle.Render(intentValue)))

	// Row 3: Tags and Acceptance Criteria count
	tagsLabel := labelStyle.Render("Tags:")
	tagsValue := "(none)"
	if len(ball.Tags) > 0 {
		tagsValue = truncate(strings.Join(ball.Tags, ", "), 30)
	}
	acLabel := labelStyle.Render("Criteria:")
	acValue := fmt.Sprintf("%d items", len(ball.AcceptanceCriteria))
	b.WriteString(fmt.Sprintf("  %s %s    %s %s\n", tagsLabel, valueStyle.Render(tagsValue), acLabel, valueStyle.Render(acValue)))

	// Footer hint
	b.WriteString(helpStyle.Render("  Press 'i' to toggle to activity log | 'e' to edit ball in $EDITOR"))

	return b.String()
}

// renderStatusBar renders the bottom status bar with keybindings
func (m Model) renderStatusBar() string {
	var hints []string

	// Build base hints based on active panel
	switch m.activePanel {
	case SessionsPanel:
		hints = []string{"Tab:panels", "j/k:nav", "a:add", "A:agent", "e:edit", "d:del", "/:filter", "i:info", "?:help", "q:quit"}
	case BallsPanel:
		hints = []string{"Tab:panels", "j/k:nav", "[/]:session", "Space:back", "a:add", "e:edit", "t:tag", "i:info", "?:help"}
	case ActivityPanel:
		hints = []string{"Tab:panels", "j/k:scroll", "Ctrl+d/u:page", "gg:top", "G:bottom", "i:info", "?:help", "q:quit"}
	}

	status := strings.Join(hints, " | ")

	// Add project scope indicator
	var scopeIndicator string
	if m.localOnly {
		scopeIndicator = "[Local]"
	} else {
		scopeIndicator = "[All Projects]"
	}
	status = scopeIndicator + " " + status

	// Add agent status indicator if running
	if m.agentStatus.Running {
		agentIndicator := fmt.Sprintf("[Agent: %s %d/%d]",
			m.agentStatus.SessionID,
			m.agentStatus.Iteration,
			m.agentStatus.MaxIterations)
		status = agentIndicator + " " + status
	}

	// Add filter indicator if active
	if m.panelSearchActive {
		status = fmt.Sprintf("[Filter: %s Ctrl+U:clear] %s", m.panelSearchQuery, status)
	}

	// Add message if present
	if m.message != "" {
		status = messageStyle.Render(m.message) + "  " + status
	}

	return helpStyle.Render(status)
}

// countBallsForSession counts balls that belong to a session
func (m Model) countBallsForSession(sessionID string) int {
	// Handle pseudo-sessions
	switch sessionID {
	case PseudoSessionAll:
		return len(m.filteredBalls)
	case PseudoSessionUntagged:
		// Count balls with no session tags
		count := 0
		sessionIDs := make(map[string]bool)
		for _, sess := range m.sessions {
			sessionIDs[sess.ID] = true
		}
		for _, ball := range m.filteredBalls {
			hasSessionTag := false
			for _, tag := range ball.Tags {
				if sessionIDs[tag] {
					hasSessionTag = true
					break
				}
			}
			if !hasSessionTag {
				count++
			}
		}
		return count
	default:
		// Regular session - count balls with matching tag
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
