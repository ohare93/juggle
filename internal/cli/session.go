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
	// The help template will show them as both top-level and available via 'juggle session'
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

	balls, err := store.GetInProgressBalls()
	if err != nil {
		return fmt.Errorf("failed to load in-progress balls: %w", err)
	}

	if len(balls) == 0 {
		return fmt.Errorf("no in-progress balls found in current project\n\nStart a ball with: juggle <ball-id>")
	}

	fmt.Printf("In progress: %d ball(s) in this project:\n\n", len(balls))
	for _, ball := range balls {
		renderCurrentSession(ball)
		fmt.Println()
	}
	return nil
}

func renderCurrentSession(sess *session.Ball) {
	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	valueStyle := lipgloss.NewStyle()

	// State color
	var statusStyle lipgloss.Style
	switch sess.State {
	case session.StateInProgress:
		statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	case session.StateBlocked:
		statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	case session.StatePending:
		statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	case session.StateComplete:
		statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	default:
		statusStyle = valueStyle
	}

	fmt.Println(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Render("Current Session"))
	fmt.Println()
	fmt.Println(labelStyle.Render("ID:"), valueStyle.Render(sess.ID))
	fmt.Println(labelStyle.Render("Title:"), valueStyle.Render(sess.Title))
	fmt.Println(labelStyle.Render("State:"), statusStyle.Render(string(sess.State)))
	fmt.Println(labelStyle.Render("Priority:"), valueStyle.Render(string(sess.Priority)))

	if sess.BlockedReason != "" {
		messageStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
		fmt.Println(labelStyle.Render("Blocked:"), messageStyle.Render(sess.BlockedReason))
	}

	fmt.Println(labelStyle.Render("Started:"), valueStyle.Render(sess.StartedAt.Format("2006-01-02 15:04:05")))
	fmt.Println(labelStyle.Render("Last Activity:"), valueStyle.Render(sess.LastActivity.Format("2006-01-02 15:04:05")))

	if len(sess.Tags) > 0 {
		fmt.Println(labelStyle.Render("Tags:"), valueStyle.Render(strings.Join(sess.Tags, ", ")))
	}

	// Acceptance Criteria section
	if len(sess.AcceptanceCriteria) > 0 {
		fmt.Printf("\n%s\n", labelStyle.Render("Acceptance Criteria:"))
		for i, ac := range sess.AcceptanceCriteria {
			fmt.Printf("  %d. %s\n", i+1, ac)
		}
	}

	if sess.UpdateCount > 0 {
		fmt.Printf("\n%s %d\n", labelStyle.Render("Activity updates:"), sess.UpdateCount)
	}
}
