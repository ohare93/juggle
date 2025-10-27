package cli

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/ohare93/juggle/internal/session"
	"github.com/spf13/cobra"
)

var sessionCmd = &cobra.Command{
	Use:     "session",
	Aliases: []string{"ball", "project", "repo", "current"},
	Short: "Manage the current session",
	Long: `Manage the current work session in this project.

When called without subcommands, displays information about the current session
including intent, status, todos, and other metadata.

Subcommands allow you to modify the current session (block, done, todo, etc.)`,
	RunE: runSession,
}

func init() {
	// Note: We don't add subcommands here because they're already registered
	// at root level for convenience. Adding them here would cause duplicate registration.
	// The help template will show them as both top-level and available via 'juggler session'
	
	// Set custom help function (defined in root.go)
	sessionCmd.SetHelpFunc(customSessionHelpFunc)
}



func runSession(cmd *cobra.Command, args []string) error {
	cwd, err := GetWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	store, err := NewStoreForCommand(cwd)
	if err != nil {
		return fmt.Errorf("failed to initialize store: %w", err)
	}

	balls, err := store.GetJugglingBalls()
	if err != nil {
		return fmt.Errorf("failed to load juggling balls: %w", err)
	}

	if len(balls) == 0 {
		return fmt.Errorf("no juggling balls found in current project\n\nStart juggling with: juggle <ball-id>")
	}

	fmt.Printf("Juggling %d ball(s) in this project:\n\n", len(balls))
	for _, ball := range balls {
		renderCurrentSession(ball)
		fmt.Println()
	}
	return nil
}

func renderCurrentSession(sess *session.Session) {
	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	valueStyle := lipgloss.NewStyle()
	
	// State color
	var statusStyle lipgloss.Style
	switch sess.ActiveState {
	case session.ActiveJuggling:
		statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	case session.ActiveDropped:
		statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	case session.ActiveReady:
		statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	case session.ActiveComplete:
		statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	default:
		statusStyle = valueStyle
	}

	fmt.Println(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Render("Current Session"))
	fmt.Println()
	fmt.Println(labelStyle.Render("ID:"), valueStyle.Render(sess.ID))
	fmt.Println(labelStyle.Render("Intent:"), valueStyle.Render(sess.Intent))
	stateStr := string(sess.ActiveState)
	if sess.JuggleState != nil {
		stateStr = stateStr + ":" + string(*sess.JuggleState)
	}
	fmt.Println(labelStyle.Render("State:"), statusStyle.Render(stateStr))
	fmt.Println(labelStyle.Render("Priority:"), valueStyle.Render(string(sess.Priority)))

	if sess.StateMessage != "" {
		messageStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
		fmt.Println(labelStyle.Render("Message:"), messageStyle.Render(sess.StateMessage))
	}

	fmt.Println(labelStyle.Render("Started:"), valueStyle.Render(sess.StartedAt.Format("2006-01-02 15:04:05")))
	fmt.Println(labelStyle.Render("Last Activity:"), valueStyle.Render(sess.LastActivity.Format("2006-01-02 15:04:05")))

	if sess.ZellijSession != "" {
		fmt.Println(labelStyle.Render("Zellij Session:"), valueStyle.Render(sess.ZellijSession))
		if sess.ZellijTab != "" {
			fmt.Println(labelStyle.Render("Zellij Tab:"), valueStyle.Render(sess.ZellijTab))
		}
	}

	if len(sess.Tags) > 0 {
		fmt.Println(labelStyle.Render("Tags:"), valueStyle.Render(strings.Join(sess.Tags, ", ")))
	}

	// Todos section
	if len(sess.Todos) > 0 {
		total, completed := sess.TodoStats()
		percentage := 0.0
		if total > 0 {
			percentage = float64(completed) / float64(total) * 100
		}
		fmt.Printf("\n%s %d/%d complete (%.0f%%)\n", 
			labelStyle.Render("Todos:"), completed, total, percentage)
		
		for i, todo := range sess.Todos {
			checkbox := "[ ]"
			checkStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
			if todo.Done {
				checkbox = "[✓]"
				checkStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
			}
			fmt.Printf("  %d. %s %s\n", i+1, checkStyle.Render(checkbox), todo.Text)
		}
	} else {
		fmt.Println()
		fmt.Println(labelStyle.Render("Todos:"), valueStyle.Render("none"))
		fmt.Println("  Add todos with: juggler session todo add <text>")
	}

	if sess.UpdateCount > 0 {
		fmt.Printf("\n%s %d\n", labelStyle.Render("Activity updates:"), sess.UpdateCount)
	}
}
