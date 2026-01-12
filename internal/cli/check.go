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

	cwd, err := GetWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	store, err := NewStoreForCommand(cwd)
	if err != nil {
		return fmt.Errorf("failed to create store: %w", err)
	}

	// Discover projects (respects --all flag)
	projects, err := DiscoverProjectsForCommand(config, store)
	if err != nil {
		return fmt.Errorf("failed to discover projects: %w", err)
	}

	// Handle no .juggler directory
	if len(projects) == 0 {
		printNoJugglerDir()
		return nil
	}

	// Load all ball states
	inProgressBalls, err := session.LoadInProgressBalls(projects)
	if err != nil {
		return fmt.Errorf("failed to load in-progress balls: %w", err)
	}

	pendingBalls, err := session.LoadPendingBalls(projects)
	if err != nil {
		return fmt.Errorf("failed to load pending balls: %w", err)
	}

	// Case 1: No in-progress balls
	if len(inProgressBalls) == 0 {
		return handleNoInProgressBalls(pendingBalls)
	}

	// Case 2: Single in-progress ball
	if len(inProgressBalls) == 1 {
		return handleSingleInProgressBall(inProgressBalls[0], pendingBalls)
	}

	// Case 3: Multiple in-progress balls
	return handleMultipleInProgressBalls(inProgressBalls, pendingBalls)
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

func handleNoInProgressBalls(pendingBalls []*session.Ball) error {
	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	dimStyle := StyleDim

	fmt.Println(successStyle.Render("✅ No active balls"))
	fmt.Println()
	fmt.Println("Ready to start new work.")
	fmt.Println()

	// Case 4: Pending balls exist
	if len(pendingBalls) > 0 {
		return promptForPendingBalls(pendingBalls)
	}

	fmt.Println(dimStyle.Render("Create a ball:"))
	fmt.Println("  juggle start   - Create and start juggling immediately")
	fmt.Println("  juggle plan    - Plan work for later")
	return nil
}

func handleSingleInProgressBall(ball *session.Ball, pendingBalls []*session.Ball) error {
	focusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true)
	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	valueStyle := lipgloss.NewStyle()

	// Get state style
	stateStyle := StyleInProgress
	stateStr := string(ball.State)

	fmt.Println(focusStyle.Render("🎯 Currently in progress: " + ball.ShortID()))
	fmt.Println(labelStyle.Render("Title:"), valueStyle.Render(ball.Title))
	fmt.Println(labelStyle.Render("State:"), stateStyle.Render(stateStr))
	if ball.BlockedReason != "" {
		messageStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
		fmt.Println(labelStyle.Render("Note:"), messageStyle.Render(ball.BlockedReason))
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

	// User said no - check for pending balls
	if len(pendingBalls) > 0 {
		fmt.Println()
		warningStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
		fmt.Println(warningStyle.Render("⚠️  You have other pending balls that need attention."))
		fmt.Println()
		return promptForPendingBalls(pendingBalls)
	}

	// No pending balls - suggest moving current ball to pending
	fmt.Println()
	dimStyle := StyleDim
	fmt.Println(dimStyle.Render("Consider moving the current ball to pending:"))
	fmt.Printf("  juggle %s pending\n", ball.ShortID())
	fmt.Println()
	fmt.Println(dimStyle.Render("Then create a new ball:"))
	fmt.Println("  juggle start")

	return nil
}

func handleMultipleInProgressBalls(inProgressBalls []*session.Ball, pendingBalls []*session.Ball) error {
	warningStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true)
	dimStyle := StyleDim

	fmt.Println(warningStyle.Render(fmt.Sprintf("⚠️  Multiple balls in progress (%d):", len(inProgressBalls))))
	fmt.Println()

	// Show all in-progress balls with numbers
	for i, ball := range inProgressBalls {
		stateStr := string(ball.State)
		stateStyle := StyleInProgress

		fmt.Printf("%d. %s: %s [%s]\n",
			i+1,
			ball.ShortID(),
			ball.Title,
			stateStyle.Render(stateStr),
		)
	}
	fmt.Println()

	// Prompt user to select
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("Which are you working on? (1-%d): ", len(inProgressBalls))
	input, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	choice := strings.TrimSpace(input)
	selected, err := strconv.Atoi(choice)
	if err != nil || selected < 1 || selected > len(inProgressBalls) {
		return fmt.Errorf("invalid choice: %s (must be 1-%d)", choice, len(inProgressBalls))
	}

	selectedBall := inProgressBalls[selected-1]
	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	fmt.Println()
	fmt.Println(successStyle.Render(fmt.Sprintf("✓ Working on: %s - %s", selectedBall.ShortID(), selectedBall.Title)))
	fmt.Println()
	fmt.Println(dimStyle.Render("Consider moving other balls to pending:"))
	fmt.Println(dimStyle.Render("  juggle <ball-id> pending"))

	return nil
}

func promptForPendingBalls(pendingBalls []*session.Ball) error {
	warningStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true)
	dimStyle := StyleDim

	fmt.Println(warningStyle.Render(fmt.Sprintf("⚠️  Found %d pending ball%s that need attention:", len(pendingBalls), pluralS(len(pendingBalls)))))
	fmt.Println()

	// Show up to 5 pending balls
	displayCount := len(pendingBalls)
	if displayCount > 5 {
		displayCount = 5
	}

	for i := 0; i < displayCount; i++ {
		ball := pendingBalls[i]
		priorityStyle := GetPriorityStyle(string(ball.Priority))
		fmt.Printf("%d. %s: %s [%s]\n",
			i+1,
			ball.ShortID(),
			ball.Title,
			priorityStyle.Render(string(ball.Priority)),
		)
	}

	if len(pendingBalls) > 5 {
		fmt.Printf("%s\n", dimStyle.Render(fmt.Sprintf("[... %d more]", len(pendingBalls)-5)))
	}
	fmt.Println()
	fmt.Println("You should work on these before creating new balls.")
	fmt.Println()

	// Prompt for action
	fmt.Println("What would you like to do?")
	fmt.Println("1) Start working on a pending ball")
	fmt.Println("2) View all pending balls")
	fmt.Println("3) Block some pending balls")
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
		return promptToStartPendingBall(pendingBalls)
	case "2":
		return showAllPendingBalls(pendingBalls)
	case "3":
		return suggestBlockingBalls(pendingBalls)
	case "4":
		fmt.Println()
		fmt.Println(dimStyle.Render("Continuing anyway..."))
		fmt.Println(dimStyle.Render("Remember: working on too many things reduces focus and effectiveness."))
		return nil
	default:
		return fmt.Errorf("invalid choice: %s (must be 1-4)", choice)
	}
}

func promptToStartPendingBall(pendingBalls []*session.Ball) error {
	dimStyle := StyleDim
	fmt.Println()
	fmt.Println(dimStyle.Render("To start working on a pending ball:"))
	fmt.Println("  juggle <ball-id>")
	fmt.Println()
	fmt.Println(dimStyle.Render("Example:"))
	if len(pendingBalls) > 0 {
		fmt.Printf("  juggle %s\n", pendingBalls[0].ShortID())
	}
	return nil
}

func showAllPendingBalls(pendingBalls []*session.Ball) error {
	fmt.Println()
	headerStyle := StyleHeader
	fmt.Println(headerStyle.Render(fmt.Sprintf(" All Pending Balls (%d) ", len(pendingBalls))))
	fmt.Println()

	for i, ball := range pendingBalls {
		priorityStyle := GetPriorityStyle(string(ball.Priority))
		fmt.Printf("%3d. [%s] [%s] %s\n",
			i+1,
			ball.ShortID(),
			priorityStyle.Render(string(ball.Priority)),
			ball.Title,
		)
	}
	fmt.Println()

	dimStyle := StyleDim
	fmt.Println(dimStyle.Render("To start working on a ball:"))
	fmt.Println("  juggle <ball-id>")
	return nil
}

func suggestBlockingBalls(pendingBalls []*session.Ball) error {
	dimStyle := StyleDim
	fmt.Println()
	fmt.Println(dimStyle.Render("To block a ball you're not planning to work on:"))
	fmt.Println("  juggle <ball-id> blocked <reason>")
	fmt.Println()
	fmt.Println(dimStyle.Render("Example:"))
	if len(pendingBalls) > 0 {
		fmt.Printf("  juggle %s blocked \"waiting for dependencies\"\n", pendingBalls[0].ShortID())
	}
	return nil
}

func pluralS(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}
