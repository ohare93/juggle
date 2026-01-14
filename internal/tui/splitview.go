package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/ohare93/juggle/internal/session"
)

// Panel dimensions as fractions of available space
const (
	leftPanelRatio          = 0.25 // 25% for sessions
	rightPanelRatio         = 0.75 // 75% for balls
	bottomPanelRows         = 6    // Fixed height for activity log
	bottomPanelRowsExpanded = 15   // Expanded height for agent output panel
	minLeftWidth            = 20
	minRightWidth           = 40
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

	// Calculate effective bottom panel height (expanded when agent output visible and expanded)
	effectiveBottomRows := bottomPanelRows
	if m.agentOutputVisible && m.agentOutputExpanded {
		effectiveBottomRows = bottomPanelRowsExpanded
		// Cap at half the screen height
		halfScreen := m.height / 2
		if effectiveBottomRows > halfScreen {
			effectiveBottomRows = halfScreen
		}
	}

	// Calculate dimensions
	mainHeight := m.height - effectiveBottomRows - 4 // Account for borders and status
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
	if m.agentOutputVisible {
		// Agent output panel takes over the bottom pane when visible
		bottomPanel = m.renderAgentOutputPanel(m.width-2, effectiveBottomRows-2)
	} else {
		switch m.bottomPaneMode {
		case BottomPaneDetail:
			bottomPanel = m.renderBallDetailPanel(m.width-2, effectiveBottomRows-2)
		case BottomPaneSplit:
			bottomPanel = m.renderSplitBottomPane(m.width-2, effectiveBottomRows-2)
		default:
			bottomPanel = m.renderActivityPanel(m.width-2, effectiveBottomRows-2)
		}
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
		activityBorder = activePanelBorderStyle.Width(m.width).Height(effectiveBottomRows)
	} else {
		activityBorder = panelBorderStyle.Width(m.width).Height(effectiveBottomRows)
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

			// Check if agent is running for this session
			agentRunningForSession := m.agentStatus.Running && m.agentStatus.SessionID == sess.ID

			// Add agent indicator prefix
			prefix := "  "
			if agentRunningForSession {
				prefix = "▶ " // Running indicator
			}

			line := fmt.Sprintf("%s%-*s (%d)",
				prefix,
				width-8, // Adjusted for prefix width
				truncate(displayName, width-8),
				ballCount,
			)

			if i == m.sessionCursor && m.activePanel == SessionsPanel {
				if agentRunningForSession {
					// Use distinct style for running agent + selected
					runningSelectedStyle := lipgloss.NewStyle().
						Bold(true).
						Foreground(lipgloss.Color("3")). // Yellow for running
						Background(lipgloss.Color("237"))
					b.WriteString(runningSelectedStyle.Render(line) + "\n")
				} else {
					b.WriteString(selectedSessionItemStyle.Render(line) + "\n")
				}
			} else if agentRunningForSession {
				// Highlight running agent session distinctly
				runningStyle := lipgloss.NewStyle().
					Bold(true).
					Foreground(lipgloss.Color("3")) // Yellow for running
				b.WriteString(runningStyle.Render(line) + "\n")
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

	// Title with filter and sort indicator
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
	// Add sort indicator
	sortIndicator := ""
	switch m.sortOrder {
	case SortByIDASC:
		sortIndicator = " [↑ID]"
	case SortByIDDESC:
		sortIndicator = " [↓ID]"
	case SortByPriority:
		sortIndicator = " [Pri]"
	case SortByLastActivity:
		sortIndicator = " [Act]"
	}
	title += sortIndicator
	if m.panelSearchActive && m.activePanel == BallsPanel {
		title = fmt.Sprintf("%s [%s]", title, m.panelSearchQuery)
	}

	// Build stats string for the balls in the current view
	statsStr := m.buildBallsStats(balls)

	// Render title row with stats on the right
	if m.activePanel == BallsPanel {
		titleRendered := activePanelTitleStyle.Render(truncate(title, width-len(statsStr)-4))
		statsRendered := lipgloss.NewStyle().Faint(true).Render(statsStr)
		// Calculate padding to right-align stats
		titleLen := lipgloss.Width(titleRendered)
		statsLen := lipgloss.Width(statsRendered)
		padding := width - titleLen - statsLen - 1
		if padding < 1 {
			padding = 1
		}
		b.WriteString(titleRendered + strings.Repeat(" ", padding) + statsRendered + "\n")
	} else {
		titleRendered := panelTitleStyle.Render(truncate(title, width-len(statsStr)-4))
		statsRendered := lipgloss.NewStyle().Faint(true).Render(statsStr)
		titleLen := lipgloss.Width(titleRendered)
		statsLen := lipgloss.Width(statsRendered)
		padding := width - titleLen - statsLen - 1
		if padding < 1 {
			padding = 1
		}
		b.WriteString(titleRendered + strings.Repeat(" ", padding) + statsRendered + "\n")
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
	if ballsHeight < 1 {
		ballsHeight = 1
	}

	// Determine if all balls are from the same project (to shorten IDs)
	sameProject := allBallsSameProject(balls)

	// Compute minimal unique IDs for display
	minimalIDs := session.ComputeMinimalUniqueIDs(balls)

	// Calculate visible range using scroll offset
	startIdx := m.ballsScrollOffset

	// Calculate if we need scroll indicators
	needTopIndicator := startIdx > 0 && m.activePanel == BallsPanel

	// Calculate actual visible lines, accounting for top indicator
	actualVisible := ballsHeight
	if needTopIndicator {
		actualVisible--
	}
	if actualVisible < 1 {
		actualVisible = 1
	}

	endIdx := startIdx + actualVisible
	if endIdx > len(balls) {
		endIdx = len(balls)
	}

	// Show scroll indicator at top if not at beginning
	if needTopIndicator {
		b.WriteString(helpStyle.Render(fmt.Sprintf("  ↑ %d more items above", startIdx)) + "\n")
	}

	// Render balls list
	for i := startIdx; i < endIdx; i++ {
		ball := balls[i]

		stateIcon := getStateIcon(ball.State)
		var line string

		// Build ID display - show minimal unique ID if all balls from same project
		idDisplay := ball.ID
		if sameProject {
			// Use minimal unique ID computed for this view
			if minID, ok := minimalIDs[ball.ID]; ok {
				idDisplay = minID
			} else {
				idDisplay = ball.ShortID()
			}
		}

		// Build optional column suffixes based on visibility settings
		prioritySuffix := ""
		if m.showPriorityColumn {
			prioritySuffix = fmt.Sprintf(" [%s]", string(ball.Priority)[0:1]) // First letter: l/m/h/u
		}

		tagsSuffix := ""
		if m.showTagsColumn && len(ball.Tags) > 0 {
			// Show truncated tags
			tagsStr := strings.Join(ball.Tags, ",")
			if len(tagsStr) > 15 {
				tagsStr = tagsStr[:12] + "..."
			}
			tagsSuffix = fmt.Sprintf(" [%s]", tagsStr)
		}

		// Build model size suffix if set and visible
		modelSizeSuffix := ""
		if m.showModelSizeColumn && ball.ModelSize != "" {
			switch ball.ModelSize {
			case session.ModelSizeSmall:
				modelSizeSuffix = " [M:S]"
			case session.ModelSizeMedium:
				modelSizeSuffix = " [M:M]"
			case session.ModelSizeLarge:
				modelSizeSuffix = " [M:L]"
			}
		}

		// Add output marker if ball has output
		outputMarker := ""
		if ball.HasOutput() {
			outputMarker = " [📋]"
		}

		// Add dependency marker if ball has dependencies
		depMarker := ""
		if ball.HasDependencies() {
			depMarker = " [→]"
		}

		// ID prefix (shown before intent)
		idPrefix := fmt.Sprintf("[%s] ", idDisplay)

		// Calculate total suffix length for width calculation
		suffixLen := len(prioritySuffix) + len(tagsSuffix) + len(modelSizeSuffix) + len(outputMarker) + len(depMarker)

		if ball.State == session.StateBlocked && ball.BlockedReason != "" {
			// Show blocked reason inline for blocked balls
			intent := truncate(ball.Title, width-25-len(idPrefix)-suffixLen)
			reason := truncate(ball.BlockedReason, width-len(intent)-15-len(idPrefix)-suffixLen)
			line = fmt.Sprintf("%s %s%s [%s]%s%s%s%s%s",
				stateIcon,
				idPrefix,
				intent,
				reason,
				prioritySuffix,
				tagsSuffix,
				modelSizeSuffix,
				outputMarker,
				depMarker,
			)
		} else {
			availWidth := width - 15 - len(idPrefix) - suffixLen
			line = fmt.Sprintf("%s %s%-*s %s%s%s%s%s%s",
				stateIcon,
				idPrefix,
				availWidth,
				truncate(ball.Title, availWidth),
				string(ball.State),
				prioritySuffix,
				tagsSuffix,
				modelSizeSuffix,
				outputMarker,
				depMarker,
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

	// Show scroll indicator at bottom if more entries
	remaining := len(balls) - endIdx
	if remaining > 0 && m.activePanel == BallsPanel {
		b.WriteString(helpStyle.Render(fmt.Sprintf("  ↓ %d more items below", remaining)) + "\n")
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

	// Title with scroll indicator
	title := "Ball Details"
	if m.activePanel == ActivityPanel {
		b.WriteString(activePanelTitleStyle.Render(title) + "\n")
	} else {
		b.WriteString(panelTitleStyle.Render(title) + "\n")
	}

	if ball == nil {
		b.WriteString(helpStyle.Render("  No ball selected - navigate to a ball to see details"))
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("  Press 'i' to cycle views"))
		return b.String()
	}

	// Build content lines for the ball details
	lines := m.buildBallDetailLines(ball, width)

	// Calculate visible lines
	availableHeight := height - 1 // Account for title
	if availableHeight < 1 {
		availableHeight = 1
	}

	totalLines := len(lines)
	maxOffset := totalLines - availableHeight
	if maxOffset < 0 {
		maxOffset = 0
	}

	// Clamp scroll offset
	scrollOffset := m.detailScrollOffset
	if scrollOffset > maxOffset {
		scrollOffset = maxOffset
	}
	if scrollOffset < 0 {
		scrollOffset = 0
	}

	// Show scroll indicator at top if not at beginning
	if scrollOffset > 0 {
		b.WriteString(helpStyle.Render(fmt.Sprintf("  ↑ %d more lines above", scrollOffset)) + "\n")
		availableHeight--
	}

	// Render visible lines
	endIdx := scrollOffset + availableHeight
	if endIdx > totalLines {
		endIdx = totalLines
	}

	for i := scrollOffset; i < endIdx; i++ {
		b.WriteString(lines[i] + "\n")
	}

	// Show scroll indicator at bottom if more content
	remaining := totalLines - endIdx
	if remaining > 0 {
		b.WriteString(helpStyle.Render(fmt.Sprintf("  ↓ %d more lines below (j/k to scroll)", remaining)))
	}

	return b.String()
}

// buildBallDetailLines builds the content lines for ball details
func (m Model) buildBallDetailLines(ball *session.Ball, width int) []string {
	var lines []string
	labelWidth := 12
	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).Width(labelWidth)
	valueStyle := lipgloss.NewStyle()

	// Row 1: ID and State
	idLabel := labelStyle.Render("ID:")
	idValue := ball.ID
	stateLabel := labelStyle.Render("State:")
	stateValue := string(ball.State)
	if ball.State == session.StateBlocked && ball.BlockedReason != "" {
		stateValue += " (" + truncate(ball.BlockedReason, 30) + ")"
	}
	lines = append(lines, fmt.Sprintf("  %s %s    %s %s", idLabel, valueStyle.Render(idValue), stateLabel, styleBallByState(ball, stateValue)))

	// Row 2: Priority and Title
	priorityLabel := labelStyle.Render("Priority:")
	priorityValue := string(ball.Priority)
	titleLabel := labelStyle.Render("Title:")
	titleValue := truncate(ball.Title, width-50)
	lines = append(lines, fmt.Sprintf("  %s %s    %s %s", priorityLabel, valueStyle.Render(priorityValue), titleLabel, valueStyle.Render(titleValue)))

	// Row 3: Tags
	tagsLabel := labelStyle.Render("Tags:")
	tagsValue := "(none)"
	if len(ball.Tags) > 0 {
		tagsValue = strings.Join(ball.Tags, ", ")
		if len(tagsValue) > 40 {
			tagsValue = truncate(tagsValue, 40)
		}
	}
	lines = append(lines, fmt.Sprintf("  %s %s", tagsLabel, valueStyle.Render(tagsValue)))

	// Row 4: Dependencies (if present)
	if len(ball.DependsOn) > 0 {
		depsLabel := labelStyle.Render("Depends On:")
		depsValue := strings.Join(ball.DependsOn, ", ")
		if len(depsValue) > width-20 {
			depsValue = truncate(depsValue, width-20)
		}
		lines = append(lines, fmt.Sprintf("  %s %s", depsLabel, valueStyle.Render(depsValue)))
	}

	// Acceptance Criteria section
	acLabel := labelStyle.Render("Criteria:")
	if len(ball.AcceptanceCriteria) == 0 {
		lines = append(lines, fmt.Sprintf("  %s %s", acLabel, valueStyle.Render("(none)")))
	} else {
		lines = append(lines, fmt.Sprintf("  %s (%d items)", acLabel, len(ball.AcceptanceCriteria)))
		// Add each acceptance criterion
		acStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
		for i, ac := range ball.AcceptanceCriteria {
			acLine := fmt.Sprintf("    %d. %s", i+1, truncate(ac, width-10))
			lines = append(lines, acStyle.Render(acLine))
		}
	}

	// Output section if present
	if ball.HasOutput() {
		outputLabel := labelStyle.Render("Output:")
		lines = append(lines, fmt.Sprintf("  %s", outputLabel))
		outputStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10")) // Green for output
		// Split output into lines and add each
		outputLines := strings.Split(ball.Output, "\n")
		for _, line := range outputLines {
			if line != "" {
				lines = append(lines, outputStyle.Render("    "+truncate(line, width-8)))
			}
		}
	}

	return lines
}

// renderSplitBottomPane renders both activity log and ball details side by side
func (m Model) renderSplitBottomPane(width, height int) string {
	// Split width between activity and details (60% details, 40% activity)
	detailWidth := int(float64(width) * 0.6)
	activityWidth := width - detailWidth - 1

	// Build detail panel
	detailContent := m.renderBallDetailPanelCompact(detailWidth-2, height)
	// Build activity panel
	activityContent := m.renderActivityPanelCompact(activityWidth-2, height)

	// Apply borders
	detailBorder := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, true, false, false).
		BorderForeground(lipgloss.Color("240")).
		Width(detailWidth)

	// Join side by side
	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		detailBorder.Render(detailContent),
		activityContent,
	)
}

// renderBallDetailPanelCompact renders a compact version of ball details for split view
func (m Model) renderBallDetailPanelCompact(width, height int) string {
	var b strings.Builder

	// Get the currently highlighted ball
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

	b.WriteString(panelTitleStyle.Render("Details") + "\n")

	if ball == nil {
		b.WriteString(helpStyle.Render("No ball selected"))
		return b.String()
	}

	// Compact format: show key info
	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	valueStyle := lipgloss.NewStyle()

	b.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("ID:"), valueStyle.Render(ball.ID)))
	b.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("State:"), styleBallByState(ball, string(ball.State))))

	// Show first 2-3 ACs that fit
	if len(ball.AcceptanceCriteria) > 0 {
		acStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
		maxACs := height - 3
		if maxACs > len(ball.AcceptanceCriteria) {
			maxACs = len(ball.AcceptanceCriteria)
		}
		if maxACs > 3 {
			maxACs = 3
		}
		for i := 0; i < maxACs; i++ {
			b.WriteString(acStyle.Render(fmt.Sprintf("%d. %s\n", i+1, truncate(ball.AcceptanceCriteria[i], width-5))))
		}
		if len(ball.AcceptanceCriteria) > maxACs {
			b.WriteString(helpStyle.Render(fmt.Sprintf("  +%d more...", len(ball.AcceptanceCriteria)-maxACs)))
		}
	}

	return b.String()
}

// renderActivityPanelCompact renders a compact activity log for split view
func (m Model) renderActivityPanelCompact(width, height int) string {
	var b strings.Builder

	b.WriteString(panelTitleStyle.Render("Activity") + "\n")

	if len(m.activityLog) == 0 {
		b.WriteString(activityLogStyle.Render("No activity"))
		return b.String()
	}

	// Show most recent entries that fit
	visibleLines := height - 1
	if visibleLines < 1 {
		visibleLines = 1
	}

	startIdx := len(m.activityLog) - visibleLines
	if startIdx < 0 {
		startIdx = 0
	}

	for i := startIdx; i < len(m.activityLog); i++ {
		entry := m.activityLog[i]
		timeStr := entry.Time.Format("15:04")
		line := fmt.Sprintf("%s %s", timeStr, truncate(entry.Message, width-8))
		b.WriteString(activityLogStyle.Render(line) + "\n")
	}

	return b.String()
}

// renderStatusBar renders the bottom status bar with keybindings
func (m Model) renderStatusBar() string {
	var hints []string

	// Build contextual hints based on active panel
	// Show the most relevant keybinds for each panel context
	switch m.activePanel {
	case SessionsPanel:
		hints = []string{
			"j/k:nav", "Enter:select", "a:add", "A:agent",
			"e:edit", "d:del", "/:filter", "P:scope",
			"O:output", "H:history", "?:help", "q:quit",
		}
	case BallsPanel:
		hints = []string{
			"j/k:nav", "s+c/s/b/p:state", "t+c/b/i/p:filter",
			"a:add", "e:edit", "E:editor", "d:del", "v+p/t/s/m:columns",
			"[/]:session", "o:sort", "?:help",
		}
	case ActivityPanel:
		hints = []string{
			"j/k:scroll", "Ctrl+d/u:page", "gg:top", "G:bottom",
			"Tab:panels", "O:output", "H:history", "?:help", "q:quit",
		}
	}

	status := strings.Join(hints, " | ")

	// Add bottom pane mode indicator
	var modeIndicator string
	if m.agentOutputVisible {
		if m.agentOutputExpanded {
			modeIndicator = "[Output+]" // + indicates expanded
		} else {
			modeIndicator = "[Output]"
		}
	} else {
		switch m.bottomPaneMode {
		case BottomPaneActivity:
			modeIndicator = "[Act]"
		case BottomPaneDetail:
			modeIndicator = "[Detail]"
		case BottomPaneSplit:
			modeIndicator = "[Split]"
		}
	}

	// Add project scope indicator
	var scopeIndicator string
	if m.localOnly {
		scopeIndicator = "[Local]"
	} else {
		scopeIndicator = "[All]"
	}
	status = modeIndicator + " " + scopeIndicator + " " + status

	// Add agent status indicator if running
	if m.agentStatus.Running {
		agentIndicator := fmt.Sprintf("[Agent: %s %d/%d | X:cancel]",
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

// allBallsSameProject checks if all balls in the list are from the same project
func allBallsSameProject(balls []*session.Ball) bool {
	if len(balls) <= 1 {
		return true
	}
	firstProject := ""
	for i, ball := range balls {
		// Extract project prefix from ID (e.g., "juggle" from "juggle-5")
		project := ball.FolderName()
		if i == 0 {
			firstProject = project
		} else if project != firstProject {
			return false
		}
	}
	return true
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

// buildBallsStats builds a compact stats string showing ball counts by state
func (m Model) buildBallsStats(balls []*session.Ball) string {
	pending := 0
	inProgress := 0
	blocked := 0
	complete := 0

	for _, ball := range balls {
		switch ball.State {
		case session.StatePending:
			pending++
		case session.StateInProgress:
			inProgress++
		case session.StateBlocked:
			blocked++
		case session.StateComplete:
			complete++
		case session.StateResearched:
			complete++ // Count researched as complete for stats purposes
		}
	}

	// Build compact stats: P:2 I:1 B:0 C:3
	return fmt.Sprintf("P:%d I:%d B:%d C:%d", pending, inProgress, blocked, complete)
}

// renderAgentOutputPanel renders the dedicated agent output panel
func (m Model) renderAgentOutputPanel(width, height int) string {
	var b strings.Builder

	// Title with status indicator
	title := "Agent Output"
	if m.agentStatus.Running {
		title = fmt.Sprintf("Agent Output [%s %d/%d]",
			m.agentStatus.SessionID,
			m.agentStatus.Iteration,
			m.agentStatus.MaxIterations)
	}

	// Show scroll position if there's content
	if len(m.agentOutput) > 0 {
		visibleLines := height - 2 // Account for title and border
		if visibleLines < 1 {
			visibleLines = 1
		}
		title = fmt.Sprintf("%s [%d/%d]", title, m.agentOutputOffset+1, len(m.agentOutput))
	}

	titleStyled := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("3")). // Yellow for agent output
		Render(title)
	b.WriteString(titleStyled + "\n")
	b.WriteString(strings.Repeat("─", width) + "\n")

	if len(m.agentOutput) == 0 {
		emptyMsg := "No agent output"
		if !m.agentStatus.Running {
			emptyMsg += " - Press 'A' on a session to launch an agent"
		}
		b.WriteString(helpStyle.Render("  " + emptyMsg))
		return b.String()
	}

	// Calculate visible range
	visibleLines := height - 2 // Account for title and separator
	if visibleLines < 1 {
		visibleLines = 1
	}

	startIdx := m.agentOutputOffset
	needTopIndicator := startIdx > 0

	// Pre-calculate if we'll need scroll indicators
	tentativeEndIdx := startIdx + visibleLines
	needBottomIndicator := tentativeEndIdx < len(m.agentOutput)

	// Reduce visible lines for scroll indicators
	if needTopIndicator {
		visibleLines--
	}
	if needBottomIndicator {
		visibleLines--
	}
	if visibleLines < 1 {
		visibleLines = 1
	}

	endIdx := startIdx + visibleLines
	if endIdx > len(m.agentOutput) {
		endIdx = len(m.agentOutput)
	}

	// Show scroll indicator at top if not at beginning
	if needTopIndicator {
		b.WriteString(helpStyle.Render(fmt.Sprintf("  ↑ %d more lines above", startIdx)) + "\n")
	}

	// Render visible lines
	normalStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9")) // Red for errors

	for i := startIdx; i < endIdx; i++ {
		entry := m.agentOutput[i]
		timeStr := entry.Time.Format("15:04:05")
		line := fmt.Sprintf("  %s %s", timeStr, truncate(entry.Line, width-12))

		if entry.IsError {
			b.WriteString(errorStyle.Render(line) + "\n")
		} else {
			b.WriteString(normalStyle.Render(line) + "\n")
		}
	}

	// Show scroll indicator at bottom if more content
	remaining := len(m.agentOutput) - endIdx
	if remaining > 0 {
		b.WriteString(helpStyle.Render(fmt.Sprintf("  ↓ %d more lines below (j/k to scroll)", remaining)))
	}

	return b.String()
}
