package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/ohare93/juggle/internal/session"
	"github.com/spf13/cobra"
)

var showJSONFlag bool

var showCmd = &cobra.Command{
	Use:   "show <session-id>",
	Short: "Show detailed information about a session",
	Long:  `Display detailed information about a specific session.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runShow,
}

func init() {
	showCmd.Flags().BoolVar(&showJSONFlag, "json", false, "Output as JSON")
}

func runShow(cmd *cobra.Command, args []string) error {
	ballID := args[0]

	// Use findBallByID which respects --all flag
	foundBall, _, err := findBallByID(ballID)
	if err != nil {
		if showJSONFlag {
			return printJSONError(err)
		}
		return err
	}

	if showJSONFlag {
		return printBallJSON(foundBall)
	}

	renderSessionDetails(foundBall)
	return nil
}

// printBallJSON outputs the ball as JSON
func printBallJSON(ball *session.Session) error {
	data, err := json.MarshalIndent(ball, "", "  ")
	if err != nil {
		return printJSONError(err)
	}
	fmt.Println(string(data))
	return nil
}

// printJSONError outputs an error in JSON format
func printJSONError(err error) error {
	errResp := map[string]string{"error": err.Error()}
	data, _ := json.Marshal(errResp)
	fmt.Println(string(data))
	return nil // Return nil so the error is in JSON, not stderr
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
	fmt.Println(labelStyle.Render("State:"), valueStyle.Render(string(sess.State)))

	if sess.BlockedReason != "" {
		fmt.Println(labelStyle.Render("Blocked:"), valueStyle.Render(sess.BlockedReason))
	}

	fmt.Println(labelStyle.Render("Started:"), valueStyle.Render(sess.StartedAt.Format("2006-01-02 15:04:05")))
	fmt.Println(labelStyle.Render("Last Activity:"), valueStyle.Render(sess.LastActivity.Format("2006-01-02 15:04:05")))
	fmt.Println(labelStyle.Render("Updates:"), valueStyle.Render(fmt.Sprintf("%d", sess.UpdateCount)))

	if len(sess.Tags) > 0 {
		fmt.Println(labelStyle.Render("Tags:"), valueStyle.Render(strings.Join(sess.Tags, ", ")))
	}

	if len(sess.AcceptanceCriteria) > 0 {
		fmt.Printf("\n%s\n", labelStyle.Render("Acceptance Criteria:"))
		for i, ac := range sess.AcceptanceCriteria {
			fmt.Printf("  %d. %s\n", i+1, ac)
		}
	}

	if sess.CompletionNote != "" {
		fmt.Println(labelStyle.Render("\nCompletion Note:"), valueStyle.Render(sess.CompletionNote))
	}
}
