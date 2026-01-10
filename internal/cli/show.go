package cli

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/ohare93/juggle/internal/session"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show <session-id>",
	Short: "Show detailed information about a session",
	Long:  `Display detailed information about a specific session.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runShow,
}

func runShow(cmd *cobra.Command, args []string) error {
	ballID := args[0]

	// Load config to discover projects
	config, err := LoadConfigForCommand()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Discover all projects
	projects, err := session.DiscoverProjects(config)
	if err != nil {
		return fmt.Errorf("failed to discover projects: %w", err)
	}

	// Search for the ball across all projects
	var foundBall *session.Session
	for _, projectPath := range projects {
		store, err := NewStoreForCommand(projectPath)
		if err != nil {
			continue
		}

		ball, err := store.GetBallByID(ballID)
		if err == nil && ball != nil {
			foundBall = ball
			break
		}
	}

	if foundBall == nil {
		return fmt.Errorf("ball %s not found in any project", ballID)
	}

	renderSessionDetails(foundBall)
	return nil
}

func renderSessionDetails(sess *session.Session) {
	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	valueStyle := lipgloss.NewStyle()

	fmt.Println(labelStyle.Render("Session ID:"), valueStyle.Render(sess.ID))
	fmt.Println(labelStyle.Render("Working Dir:"), valueStyle.Render(sess.WorkingDir))
	fmt.Println(labelStyle.Render("Intent:"), valueStyle.Render(sess.Intent))
	if sess.Description != "" {
		fmt.Println(labelStyle.Render("Description:"), valueStyle.Render(sess.Description))
	}
	fmt.Println(labelStyle.Render("Priority:"), valueStyle.Render(string(sess.Priority)))
	stateStr := string(sess.ActiveState)
	if sess.JuggleState != nil {
		stateStr = stateStr + ":" + string(*sess.JuggleState)
	}
	fmt.Println(labelStyle.Render("State:"), valueStyle.Render(stateStr))

	if sess.StateMessage != "" {
		fmt.Println(labelStyle.Render("Message:"), valueStyle.Render(sess.StateMessage))
	}

	fmt.Println(labelStyle.Render("Started:"), valueStyle.Render(sess.StartedAt.Format("2006-01-02 15:04:05")))
	fmt.Println(labelStyle.Render("Last Activity:"), valueStyle.Render(sess.LastActivity.Format("2006-01-02 15:04:05")))
	fmt.Println(labelStyle.Render("Updates:"), valueStyle.Render(fmt.Sprintf("%d", sess.UpdateCount)))

	if len(sess.Tags) > 0 {
		fmt.Println(labelStyle.Render("Tags:"), valueStyle.Render(strings.Join(sess.Tags, ", ")))
	}

	if len(sess.Todos) > 0 {
		total, completed := sess.TodoStats()
		percentage := 0.0
		if total > 0 {
			percentage = float64(completed) / float64(total) * 100
		}
		fmt.Printf("\n%s %d/%d complete (%.0f%%)\n", labelStyle.Render("Todos:"), completed, total, percentage)
		
		for i, todo := range sess.Todos {
			checkbox := "[ ]"
			if todo.Done {
				checkbox = "[✓]"
			}
			fmt.Printf("  %d. %s %s\n", i+1, checkbox, todo.Text)
			if todo.Description != "" {
				fmt.Printf("      %s\n", todo.Description)
			}
		}
	}

	if sess.CompletionNote != "" {
		fmt.Println(labelStyle.Render("\nCompletion Note:"), valueStyle.Render(sess.CompletionNote))
	}
}
