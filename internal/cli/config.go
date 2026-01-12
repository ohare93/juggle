package cli

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/ohare93/juggle/internal/session"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage repository-level configuration",
	Long: `Manage repository-level juggler configuration.

Commands:
  config ac list              List repo-level acceptance criteria
  config ac add "criterion"   Add an acceptance criterion
  config ac set --edit        Edit acceptance criteria in $EDITOR
  config ac clear             Remove all acceptance criteria`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var configACCmd = &cobra.Command{
	Use:   "ac",
	Short: "Manage repository-level acceptance criteria",
	Long: `Manage repository-level acceptance criteria.

These criteria are inherited by all new sessions in this repository
and apply to every task/ball completed within those sessions.

Common use cases:
  - "All tests must pass before marking complete"
  - "Build must succeed"
  - "Update documentation if relevant"

Commands:
  config ac list              List current acceptance criteria
  config ac add "criterion"   Add a new criterion
  config ac set --edit        Edit all criteria in $EDITOR
  config ac clear             Remove all criteria`,
	RunE: runConfigACList,
}

var configACListCmd = &cobra.Command{
	Use:   "list",
	Short: "List repository-level acceptance criteria",
	RunE:  runConfigACList,
}

var configACAddCmd = &cobra.Command{
	Use:   "add <criterion>",
	Short: "Add a repository-level acceptance criterion",
	Args:  cobra.ExactArgs(1),
	RunE:  runConfigACAdd,
}

var configACSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Set repository-level acceptance criteria",
	Long: `Set repository-level acceptance criteria.

Without flags, prompts for criteria one per line.
With --edit, opens criteria in $EDITOR for editing.

Examples:
  juggle config ac set --edit
  juggle config ac set  # Interactive prompt`,
	RunE: runConfigACSet,
}

var configACClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear all repository-level acceptance criteria",
	RunE:  runConfigACClear,
}

var configACEditFlag bool

func init() {
	configACSetCmd.Flags().BoolVar(&configACEditFlag, "edit", false, "Open criteria in $EDITOR")

	configACCmd.AddCommand(configACListCmd)
	configACCmd.AddCommand(configACAddCmd)
	configACCmd.AddCommand(configACSetCmd)
	configACCmd.AddCommand(configACClearCmd)

	configCmd.AddCommand(configACCmd)
}

func runConfigACList(cmd *cobra.Command, args []string) error {
	cwd, err := GetWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	criteria, err := session.GetProjectAcceptanceCriteria(cwd)
	if err != nil {
		return fmt.Errorf("failed to load acceptance criteria: %w", err)
	}

	if len(criteria) == 0 {
		fmt.Println("No repository-level acceptance criteria configured.")
		fmt.Println("\nAdd criteria with: juggle config ac add \"criterion\"")
		fmt.Println("Or edit in $EDITOR: juggle config ac set --edit")
		return nil
	}

	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	fmt.Println(labelStyle.Render("Repository-Level Acceptance Criteria:"))
	fmt.Println()
	for i, ac := range criteria {
		fmt.Printf("  %d. %s\n", i+1, ac)
	}
	fmt.Println()
	fmt.Printf("These criteria apply to all new sessions in this repository.\n")

	return nil
}

func runConfigACAdd(cmd *cobra.Command, args []string) error {
	criterion := args[0]

	cwd, err := GetWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Load existing criteria
	criteria, err := session.GetProjectAcceptanceCriteria(cwd)
	if err != nil {
		// If config doesn't exist, start with empty list
		criteria = []string{}
	}

	// Add new criterion
	criteria = append(criteria, criterion)

	// Save
	if err := session.UpdateProjectAcceptanceCriteria(cwd, criteria); err != nil {
		return fmt.Errorf("failed to save acceptance criteria: %w", err)
	}

	fmt.Printf("Added acceptance criterion: %s\n", criterion)
	fmt.Printf("Total criteria: %d\n", len(criteria))

	return nil
}

func runConfigACSet(cmd *cobra.Command, args []string) error {
	cwd, err := GetWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Load existing criteria
	existingCriteria, err := session.GetProjectAcceptanceCriteria(cwd)
	if err != nil {
		existingCriteria = []string{}
	}

	var newCriteria []string

	if configACEditFlag {
		// Edit in $EDITOR
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "vi"
		}

		// Create temp file with current criteria
		tmpFile, err := os.CreateTemp("", "juggle-ac-*.txt")
		if err != nil {
			return fmt.Errorf("failed to create temp file: %w", err)
		}
		tmpPath := tmpFile.Name()
		defer os.Remove(tmpPath)

		// Write header and existing criteria
		tmpFile.WriteString("# Repository-Level Acceptance Criteria\n")
		tmpFile.WriteString("# One criterion per line. Lines starting with # are comments.\n")
		tmpFile.WriteString("# Empty lines are ignored.\n\n")
		for _, ac := range existingCriteria {
			tmpFile.WriteString(ac + "\n")
		}
		tmpFile.Close()

		// Open editor
		editorCmd := exec.Command(editor, tmpPath)
		editorCmd.Stdin = os.Stdin
		editorCmd.Stdout = os.Stdout
		editorCmd.Stderr = os.Stderr

		if err := editorCmd.Run(); err != nil {
			return fmt.Errorf("editor failed: %w", err)
		}

		// Read edited content
		file, err := os.Open(tmpPath)
		if err != nil {
			return fmt.Errorf("failed to read edited content: %w", err)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			// Skip comments and empty lines
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			newCriteria = append(newCriteria, line)
		}
	} else {
		// Interactive prompt
		fmt.Println("Enter acceptance criteria (one per line, empty line to finish):")
		if len(existingCriteria) > 0 {
			fmt.Println("Current criteria will be replaced.")
		}

		reader := bufio.NewReader(os.Stdin)
		for {
			fmt.Print("  > ")
			input, err := reader.ReadString('\n')
			if err != nil {
				break
			}
			criterion := strings.TrimSpace(input)
			if criterion == "" {
				break
			}
			newCriteria = append(newCriteria, criterion)
		}
	}

	// Save
	if err := session.UpdateProjectAcceptanceCriteria(cwd, newCriteria); err != nil {
		return fmt.Errorf("failed to save acceptance criteria: %w", err)
	}

	if len(newCriteria) == 0 {
		fmt.Println("Cleared all acceptance criteria.")
	} else {
		fmt.Printf("Updated acceptance criteria (%d items):\n", len(newCriteria))
		for i, ac := range newCriteria {
			fmt.Printf("  %d. %s\n", i+1, ac)
		}
	}

	return nil
}

func runConfigACClear(cmd *cobra.Command, args []string) error {
	cwd, err := GetWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Confirm
	confirmed, err := ConfirmSingleKey("Clear all repository-level acceptance criteria?")
	if err != nil {
		return err
	}
	if !confirmed {
		fmt.Println("Cancelled.")
		return nil
	}

	// Save empty list
	if err := session.UpdateProjectAcceptanceCriteria(cwd, []string{}); err != nil {
		return fmt.Errorf("failed to clear acceptance criteria: %w", err)
	}

	fmt.Println("Cleared all repository-level acceptance criteria.")
	return nil
}
