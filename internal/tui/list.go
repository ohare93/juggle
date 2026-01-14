package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/ohare93/juggle/internal/session"
)

func renderBallList(balls []*session.Ball, cursor int, width int) string {
	var output strings.Builder

	// Header
	header := fmt.Sprintf("%-15s %-40s %-20s %-10s %s",
		"ID", "Title", "State", "Priority", "Tags")
	output.WriteString(lipgloss.NewStyle().Bold(true).Render(header) + "\n")
	output.WriteString(strings.Repeat("─", width) + "\n")

	for i, ball := range balls {
		// Format ball info
		title := truncate(ball.Title, 40)
		stateStr := formatState(ball)
		tagsStr := strings.Join(ball.Tags, ", ")
		if len(tagsStr) > 20 {
			tagsStr = truncate(tagsStr, 20)
		}

		line := fmt.Sprintf("%-15s %-40s %-20s %-10s %s",
			truncateID(ball.ID, 15),
			title,
			stateStr,
			ball.Priority,
			tagsStr,
		)

		// Color code by state and priority
		line = styleBallByState(ball, line)

		// Style based on selection
		if i == cursor {
			line = selectedBallStyle.Render(line)
		} else {
			line = ballStyle.Render(line)
		}

		output.WriteString(line + "\n")
	}

	return output.String()
}

func formatState(ball *session.Ball) string {
	stateStr := string(ball.State)
	// Add output marker if ball has output
	if ball.HasOutput() {
		stateStr += " [📋]"
	}
	return stateStr
}

func styleBallByState(ball *session.Ball, line string) string {
	var color lipgloss.Color

	// Choose color based on state
	switch ball.State {
	case session.StatePending:
		color = readyColor
	case session.StateInProgress:
		color = jugglingColor
	case session.StateBlocked:
		color = droppedColor
	case session.StateComplete:
		color = completeColor
	case session.StateResearched:
		color = researchedColor
	case session.StateOnHold:
		color = onHoldColor
	default:
		color = lipgloss.Color("7") // Default white
	}

	return lipgloss.NewStyle().Foreground(color).Render(line)
}

func truncate(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

func truncateID(id string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if len(id) <= maxLen {
		return id
	}
	// Keep project name and last chars that fit
	// Example: juggle-20251012-143438 → juggle-...3438
	parts := strings.Split(id, "-")
	if len(parts) >= 2 {
		projectName := parts[0]
		// Calculate how many chars we have left: maxLen - projectName - "-..."
		remainingChars := maxLen - len(projectName) - 4 // 4 for "-..."
		if remainingChars > 0 && len(id) >= remainingChars {
			lastChars := id[len(id)-remainingChars:]
			return projectName + "-..." + lastChars
		}
	}
	return truncate(id, maxLen)
}

func countByState(balls []*session.Ball, state string) int {
	count := 0
	for _, ball := range balls {
		if string(ball.State) == state {
			count++
		}
	}
	return count
}
