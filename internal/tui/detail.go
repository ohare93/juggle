package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/ohare93/juggle/internal/session"
)

func renderBallDetail(ball *session.Session) string {
	var b strings.Builder

	// Header
	title := fmt.Sprintf("🎯 Ball: %s", ball.ID)
	b.WriteString(titleStyle.Render(title) + "\n\n")

	// Basic info
	b.WriteString(renderField("Intent", ball.Intent))
	b.WriteString(renderField("Priority", string(ball.Priority)))
	b.WriteString(renderField("State", formatState(ball)))
	if ball.State == session.StateBlocked && ball.BlockedReason != "" {
		reasonStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Italic(true)
		b.WriteString(renderField("Blocked Reason", reasonStyle.Render(ball.BlockedReason)))
	}
	b.WriteString(renderField("Working Dir", ball.WorkingDir))

	// Timestamps
	if !ball.StartedAt.IsZero() {
		b.WriteString(renderField("Started", formatTime(ball.StartedAt)))
	}
	if !ball.LastActivity.IsZero() {
		b.WriteString(renderField("Last Activity", formatTime(ball.LastActivity)))
	}

	// Tags
	if len(ball.Tags) > 0 {
		b.WriteString(renderField("Tags", strings.Join(ball.Tags, ", ")))
	}

	// Todos
	if len(ball.Todos) > 0 {
		b.WriteString("\n" + lipgloss.NewStyle().Bold(true).Render("Todos:") + "\n")
		for i, todo := range ball.Todos {
			status := "[ ]"
			style := lipgloss.NewStyle()
			if todo.Done {
				status = "[✓]"
				style = style.Foreground(lipgloss.Color("8")).Strikethrough(true)
			}
			todoLine := fmt.Sprintf("  %s %d. %s", status, i+1, todo.Text)
			b.WriteString(style.Render(todoLine) + "\n")
		}
	}

	// Footer
	b.WriteString("\n" + helpStyle.Render("Press 'b' to go back, 'q' to quit") + "\n")

	return b.String()
}

func renderField(name, value string) string {
	nameStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	return fmt.Sprintf("%s: %s\n", nameStyle.Render(name), value)
}

func formatTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	if diff < time.Minute {
		return "just now"
	} else if diff < time.Hour {
		mins := int(diff.Minutes())
		return fmt.Sprintf("%d minute%s ago", mins, pluralize(mins))
	} else if diff < 24*time.Hour {
		hours := int(diff.Hours())
		return fmt.Sprintf("%d hour%s ago", hours, pluralize(hours))
	} else if diff < 7*24*time.Hour {
		days := int(diff.Hours() / 24)
		return fmt.Sprintf("%d day%s ago", days, pluralize(days))
	}

	return t.Format("2006-01-02 15:04")
}

func pluralize(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
