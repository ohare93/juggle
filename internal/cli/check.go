package cli

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/ohare93/juggle/internal/session"
	"github.com/spf13/cobra"
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Check current juggler state and get workflow guidance",
	Long: `Interactive workflow helper that detects your current juggling state
and provides appropriate guidance to comply with workflow.

This command helps you:
- Understand what you're currently working on
- See if there are ready balls needing attention
- Get actionable next steps
- Stay compliant with juggler workflow

Run this before creating new balls to ensure you're following best practices.`,
	RunE: runCheck,
}

func runCheck(cmd *cobra.Command, args []string) error {
	config, err := LoadConfigForCommand()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	projects, err := session.DiscoverProjects(config)
	if err != nil {
		return fmt.Errorf("failed to discover projects: %w", err)
	}

	// Handle no .juggler directory
	if len(projects) == 0 {
		printNoJugglerDir()
		return nil
	}

	// Load all ball states
	jugglingBalls, err := session.LoadJugglingBalls(projects)
	if err != nil {
		return fmt.Errorf("failed to load juggling balls: %w", err)
	}

	readyBalls, err := session.LoadReadyBalls(projects)
	if err != nil {
		return fmt.Errorf("failed to load ready balls: %w", err)
	}

	// Case 1: No juggling balls
	if len(jugglingBalls) == 0 {
		return handleNoJugglingBalls(readyBalls)
	}

	// Case 2: Single juggling ball
	if len(jugglingBalls) == 1 {
		return handleSingleJugglingBall(jugglingBalls[0], readyBalls)
	}

	// Case 3: Multiple juggling balls
	return handleMultipleJugglingBalls(jugglingBalls, readyBalls)
}

func printNoJugglerDir() {
	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	dimStyle := StyleDim

	fmt.Println(successStyle.Render("✅ No juggler directory found"))
	fmt.Println()
	fmt.Println("Ready to start new work.")
	fmt.Println()
	fmt.Println(dimStyle.Render("Initialize juggler:"))
	fmt.Println("  juggle plan    - Plan work for later")
	fmt.Println("  juggle start   - Create and start juggling immediately")
}

func handleNoJugglingBalls(readyBalls []*session.Session) error {
	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	dimStyle := StyleDim

	fmt.Println(successStyle.Render("✅ No active balls"))
	fmt.Println()
	fmt.Println("Ready to start new work.")
	fmt.Println()

	// Case 4: Ready balls exist
	if len(readyBalls) > 0 {
		return promptForReadyBalls(readyBalls)
	}

	fmt.Println(dimStyle.Render("Create a ball:"))
	fmt.Println("  juggle start   - Create and start juggling immediately")
	fmt.Println("  juggle plan    - Plan work for later")
	return nil
}

func handleSingleJugglingBall(ball *session.Session, readyBalls []*session.Session) error {
	focusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true)
	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	valueStyle := lipgloss.NewStyle()

	// Get state style
	var stateStyle lipgloss.Style
	stateStr := ""
	if ball.JuggleState != nil {
		stateStr = string(*ball.JuggleState)
		switch *ball.JuggleState {
		case session.JuggleNeedsCaught:
			stateStyle = StyleNeedsCaught
		case session.JuggleNeedsThrown:
			stateStyle = StyleNeedsThrown
		case session.JuggleInAir:
			stateStyle = StyleInAir
		}
	}

	fmt.Println(focusStyle.Render("🎯 Currently juggling: " + ball.ShortID()))
	fmt.Println(labelStyle.Render("Intent:"), valueStyle.Render(ball.Intent))
	fmt.Println(labelStyle.Render("State:"), stateStyle.Render(stateStr))
	if ball.StateMessage != "" {
		messageStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
		fmt.Println(labelStyle.Render("Message:"), messageStyle.Render(ball.StateMessage))
	}
	fmt.Println()

	// Prompt user
	confirmed, err := ConfirmSingleKey("Is this what you're working on?")
	if err != nil {
		return fmt.Errorf("operation cancelled")
	}

	if confirmed {
		successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
		fmt.Println()
		fmt.Println(successStyle.Render("✓ Great! Continue working on this ball."))
		return nil
	}

	// User said no - check for ready balls
	if len(readyBalls) > 0 {
		fmt.Println()
		warningStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
		fmt.Println(warningStyle.Render("⚠️  You have other ready balls that need attention."))
		fmt.Println()
		return promptForReadyBalls(readyBalls)
	}

	// No ready balls - suggest moving current ball to ready
	fmt.Println()
	dimStyle := StyleDim
	fmt.Println(dimStyle.Render("Consider moving the current ball to ready:"))
	fmt.Printf("  juggle %s ready\n", ball.ShortID())
	fmt.Println()
	fmt.Println(dimStyle.Render("Then create a new ball:"))
	fmt.Println("  juggle start")

	return nil
}

func handleMultipleJugglingBalls(jugglingBalls []*session.Session, readyBalls []*session.Session) error {
	warningStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true)
	dimStyle := StyleDim

	fmt.Println(warningStyle.Render(fmt.Sprintf("⚠️  Multiple balls juggling (%d):", len(jugglingBalls))))
	fmt.Println()

	// Show all juggling balls with numbers
	for i, ball := range jugglingBalls {
		stateStr := ""
		var stateStyle lipgloss.Style
		if ball.JuggleState != nil {
			stateStr = string(*ball.JuggleState)
			switch *ball.JuggleState {
			case session.JuggleNeedsCaught:
				stateStyle = StyleNeedsCaught
			case session.JuggleNeedsThrown:
				stateStyle = StyleNeedsThrown
			case session.JuggleInAir:
				stateStyle = StyleInAir
			}
		}

		fmt.Printf("%d. %s: %s [%s]\n",
			i+1,
			ball.ShortID(),
			ball.Intent,
			stateStyle.Render(stateStr),
		)
	}
	fmt.Println()

	// Prompt user to select
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("Which are you working on? (1-%d): ", len(jugglingBalls))
	input, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	choice := strings.TrimSpace(input)
	selected, err := strconv.Atoi(choice)
	if err != nil || selected < 1 || selected > len(jugglingBalls) {
		return fmt.Errorf("invalid choice: %s (must be 1-%d)", choice, len(jugglingBalls))
	}

	selectedBall := jugglingBalls[selected-1]
	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	fmt.Println()
	fmt.Println(successStyle.Render(fmt.Sprintf("✓ Working on: %s - %s", selectedBall.ShortID(), selectedBall.Intent)))
	fmt.Println()
	fmt.Println(dimStyle.Render("Consider moving other balls to ready:"))
	fmt.Println(dimStyle.Render("  juggle <ball-id> ready"))

	return nil
}

func promptForReadyBalls(readyBalls []*session.Session) error {
	warningStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true)
	dimStyle := StyleDim

	fmt.Println(warningStyle.Render(fmt.Sprintf("⚠️  Found %d ready ball%s that need attention:", len(readyBalls), pluralS(len(readyBalls)))))
	fmt.Println()

	// Show up to 5 ready balls
	displayCount := len(readyBalls)
	if displayCount > 5 {
		displayCount = 5
	}

	for i := 0; i < displayCount; i++ {
		ball := readyBalls[i]
		priorityStyle := GetPriorityStyle(string(ball.Priority))
		fmt.Printf("%d. %s: %s [%s]\n",
			i+1,
			ball.ShortID(),
			ball.Intent,
			priorityStyle.Render(string(ball.Priority)),
		)
	}

	if len(readyBalls) > 5 {
		fmt.Printf("%s\n", dimStyle.Render(fmt.Sprintf("[... %d more]", len(readyBalls)-5)))
	}
	fmt.Println()
	fmt.Println("You should work on these before creating new balls.")
	fmt.Println()

	// Prompt for action
	fmt.Println("What would you like to do?")
	fmt.Println("1) Start working on a ready ball")
	fmt.Println("2) View all ready balls")
	fmt.Println("3) Drop some ready balls")
	fmt.Println("4) Continue anyway (not recommended)")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Choice (1-4): ")
	input, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	choice := strings.TrimSpace(input)

	switch choice {
	case "1":
		return promptToStartReadyBall(readyBalls)
	case "2":
		return showAllReadyBalls(readyBalls)
	case "3":
		return suggestDroppingBalls(readyBalls)
	case "4":
		fmt.Println()
		fmt.Println(dimStyle.Render("Continuing anyway..."))
		fmt.Println(dimStyle.Render("Remember: working on too many things reduces focus and effectiveness."))
		return nil
	default:
		return fmt.Errorf("invalid choice: %s (must be 1-4)", choice)
	}
}

func promptToStartReadyBall(readyBalls []*session.Session) error {
	dimStyle := StyleDim
	fmt.Println()
	fmt.Println(dimStyle.Render("To start working on a ready ball:"))
	fmt.Println("  juggle <ball-id>")
	fmt.Println()
	fmt.Println(dimStyle.Render("Example:"))
	if len(readyBalls) > 0 {
		fmt.Printf("  juggle %s\n", readyBalls[0].ShortID())
	}
	return nil
}

func showAllReadyBalls(readyBalls []*session.Session) error {
	fmt.Println()
	headerStyle := StyleHeader
	fmt.Println(headerStyle.Render(fmt.Sprintf(" All Ready Balls (%d) ", len(readyBalls))))
	fmt.Println()

	for i, ball := range readyBalls {
		priorityStyle := GetPriorityStyle(string(ball.Priority))
		fmt.Printf("%3d. [%s] [%s] %s\n",
			i+1,
			ball.ShortID(),
			priorityStyle.Render(string(ball.Priority)),
			ball.Intent,
		)
	}
	fmt.Println()

	dimStyle := StyleDim
	fmt.Println(dimStyle.Render("To start working on a ball:"))
	fmt.Println("  juggle <ball-id>")
	return nil
}

func suggestDroppingBalls(readyBalls []*session.Session) error {
	dimStyle := StyleDim
	fmt.Println()
	fmt.Println(dimStyle.Render("To drop a ball you're not planning to work on:"))
	fmt.Println("  juggle <ball-id> drop")
	fmt.Println()
	fmt.Println(dimStyle.Render("Example:"))
	if len(readyBalls) > 0 {
		fmt.Printf("  juggle %s drop\n", readyBalls[0].ShortID())
	}
	return nil
}

func pluralS(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}
