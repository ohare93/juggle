package cli

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/ohare93/juggle/internal/session"
	"github.com/ohare93/juggle/internal/vcs"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage juggle configuration",
	Long: `Manage juggle configuration (repository and global).

Without arguments, displays all current configuration entries.

Commands:
  config ac list              List repo-level acceptance criteria
  config ac add "criterion"   Add an acceptance criterion
  config ac set --edit        Edit acceptance criteria in $EDITOR
  config ac clear             Remove all acceptance criteria

  config delay show           Show current iteration delay settings
  config delay set <mins>     Set delay between iterations (in minutes)
  config delay clear          Remove iteration delay`,
	RunE: runConfigShow,
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	// Load global config
	globalConfig, err := session.LoadConfigWithOptions(GetConfigOptions())
	if err != nil {
		return fmt.Errorf("failed to load global config: %w", err)
	}

	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	warningStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("208"))

	// Display global config
	fmt.Println(labelStyle.Render("Global Configuration:"))
	fmt.Println()

	// Search paths
	fmt.Printf("  %s: ", keyStyle.Render("search_paths"))
	if len(globalConfig.SearchPaths) == 0 {
		fmt.Println("(empty)")
	} else {
		fmt.Println()
		for _, path := range globalConfig.SearchPaths {
			fmt.Printf("    - %s\n", path)
		}
	}

	// Iteration delay
	fmt.Printf("  %s: %d\n", keyStyle.Render("iteration_delay_minutes"), globalConfig.IterationDelayMinutes)
	fmt.Printf("  %s: %d\n", keyStyle.Render("iteration_delay_fuzz"), globalConfig.IterationDelayFuzz)

	// Show warnings for unknown fields
	unknownFields := globalConfig.GetUnknownFields()
	if len(unknownFields) > 0 {
		fmt.Println()
		for _, key := range unknownFields {
			fmt.Println(warningStyle.Render(fmt.Sprintf("Unknown config key: %s", key)))
		}
	}

	// Try to load project config if we're in a project
	cwd, err := GetWorkingDir()
	if err == nil {
		projectConfig, err := session.LoadProjectConfig(cwd)
		if err == nil {
			fmt.Println()
			fmt.Println(labelStyle.Render("Project Configuration:"))
			fmt.Println()

			// Default acceptance criteria
			fmt.Printf("  %s: ", keyStyle.Render("default_acceptance_criteria"))
			if len(projectConfig.DefaultAcceptanceCriteria) == 0 {
				fmt.Println("(empty)")
			} else {
				fmt.Println()
				for _, ac := range projectConfig.DefaultAcceptanceCriteria {
					fmt.Printf("    - %s\n", ac)
				}
			}
		}
	}

	return nil
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
	Long: `Clear all repository-level acceptance criteria.

Use --yes (-y) to skip the confirmation prompt (for headless/automated use).`,
	RunE: runConfigACClear,
}

var configACEditFlag bool
var configACYesFlag bool

func init() {
	configACSetCmd.Flags().BoolVar(&configACEditFlag, "edit", false, "Open criteria in $EDITOR")
	configACClearCmd.Flags().BoolVarP(&configACYesFlag, "yes", "y", false, "Skip confirmation prompt (for headless mode)")

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

	// Confirm (skip with --yes flag)
	if !configACYesFlag {
		confirmed, err := ConfirmSingleKey("Clear all repository-level acceptance criteria?")
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	// Save empty list
	if err := session.UpdateProjectAcceptanceCriteria(cwd, []string{}); err != nil {
		return fmt.Errorf("failed to clear acceptance criteria: %w", err)
	}

	fmt.Println("Cleared all repository-level acceptance criteria.")
	return nil
}

// Delay command variables
var configDelayFuzz int

// configDelayCmd is the parent command for delay settings
var configDelayCmd = &cobra.Command{
	Use:   "delay",
	Short: "Manage iteration delay settings (global)",
	Long: `Manage the delay between agent iterations.

This is a global setting stored in ~/.juggle/config.json.

The delay adds a wait time between each agent iteration, with an optional
"fuzz" factor that adds randomness (+/- the specified minutes).

Commands:
  config delay show           Show current delay settings
  config delay set <mins>     Set delay in minutes (use --fuzz for variance)
  config delay clear          Remove delay settings

Examples:
  juggle config delay show
  juggle config delay set 5              # 5 minute delay
  juggle config delay set 5 --fuzz 2     # 5 ± 2 minutes (3-7 min range)
  juggle config delay clear`,
	RunE: runConfigDelayShow,
}

var configDelayShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current iteration delay settings",
	RunE:  runConfigDelayShow,
}

var configDelaySetCmd = &cobra.Command{
	Use:   "set <minutes>",
	Short: "Set the delay between agent iterations",
	Long: `Set the delay between agent iterations in minutes.

The delay is applied after each agent iteration before starting the next.
Use --fuzz to add randomness: the actual delay will be base ± fuzz minutes.

Examples:
  juggle config delay set 5              # Fixed 5 minute delay
  juggle config delay set 10 --fuzz 3    # 10 ± 3 minutes (7-13 min range)
  juggle config delay set 2 --fuzz 1     # 2 ± 1 minutes (1-3 min range)`,
	Args: cobra.ExactArgs(1),
	RunE: runConfigDelaySet,
}

var configDelayClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Remove iteration delay settings",
	RunE:  runConfigDelayClear,
}

func init() {
	configDelaySetCmd.Flags().IntVarP(&configDelayFuzz, "fuzz", "f", 0, "Random variance (+/-) in minutes")

	configDelayCmd.AddCommand(configDelayShowCmd)
	configDelayCmd.AddCommand(configDelaySetCmd)
	configDelayCmd.AddCommand(configDelayClearCmd)

	configCmd.AddCommand(configDelayCmd)
}

func runConfigDelayShow(cmd *cobra.Command, args []string) error {
	delayMinutes, fuzz, err := session.GetGlobalIterationDelayWithOptions(GetConfigOptions())
	if err != nil {
		return fmt.Errorf("failed to load delay settings: %w", err)
	}

	if delayMinutes == 0 {
		fmt.Println("No iteration delay configured.")
		fmt.Println("\nSet a delay with: juggle config delay set <minutes>")
		fmt.Println("Add variance with: juggle config delay set <minutes> --fuzz <minutes>")
		return nil
	}

	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	fmt.Println(labelStyle.Render("Iteration Delay Settings:"))
	fmt.Println()
	fmt.Printf("  Base delay: %d minute(s)\n", delayMinutes)
	if fuzz > 0 {
		minDelay := delayMinutes - fuzz
		if minDelay < 0 {
			minDelay = 0
		}
		maxDelay := delayMinutes + fuzz
		fmt.Printf("  Fuzz: ± %d minute(s)\n", fuzz)
		fmt.Printf("  Actual range: %d - %d minutes\n", minDelay, maxDelay)
	} else {
		fmt.Printf("  Fuzz: none (fixed delay)\n")
	}
	fmt.Println()
	fmt.Println("This delay is applied between each agent iteration.")

	return nil
}

func runConfigDelaySet(cmd *cobra.Command, args []string) error {
	var delayMinutes int
	_, err := fmt.Sscanf(args[0], "%d", &delayMinutes)
	if err != nil || delayMinutes < 0 {
		return fmt.Errorf("invalid delay: %s (must be a non-negative integer)", args[0])
	}

	if configDelayFuzz < 0 {
		return fmt.Errorf("invalid fuzz: %d (must be a non-negative integer)", configDelayFuzz)
	}

	if err := session.UpdateGlobalIterationDelayWithOptions(GetConfigOptions(), delayMinutes, configDelayFuzz); err != nil {
		return fmt.Errorf("failed to save delay settings: %w", err)
	}

	fmt.Printf("Set iteration delay: %d minute(s)", delayMinutes)
	if configDelayFuzz > 0 {
		minDelay := delayMinutes - configDelayFuzz
		if minDelay < 0 {
			minDelay = 0
		}
		maxDelay := delayMinutes + configDelayFuzz
		fmt.Printf(" ± %d (range: %d-%d minutes)", configDelayFuzz, minDelay, maxDelay)
	}
	fmt.Println()

	return nil
}

func runConfigDelayClear(cmd *cobra.Command, args []string) error {
	if err := session.ClearGlobalIterationDelayWithOptions(GetConfigOptions()); err != nil {
		return fmt.Errorf("failed to clear delay settings: %w", err)
	}

	fmt.Println("Cleared iteration delay settings.")
	return nil
}

// VCS command variables
var configVCSProjectFlag bool

// configVCSCmd is the parent command for VCS settings
var configVCSCmd = &cobra.Command{
	Use:   "vcs",
	Short: "Manage version control system settings",
	Long: `Manage the version control system used for agent commits.

By default, juggle auto-detects VCS by checking for .jj (preferred) then .git.
You can override this globally or per-project.

Resolution order (highest to lowest priority):
  1. Project config (.juggle/config.json vcs field)
  2. Global config (~/.juggle/config.json vcs field)
  3. Auto-detect: .jj directory > .git directory > git (default)

Commands:
  config vcs show              Show current VCS settings and detection
  config vcs set <type>        Set VCS type (git or jj)
  config vcs clear             Clear VCS setting (use auto-detection)

Examples:
  juggle config vcs show
  juggle config vcs set git           # Use git globally
  juggle config vcs set jj            # Use jj globally
  juggle config vcs set git --project # Use git for this project only
  juggle config vcs clear             # Clear global setting
  juggle config vcs clear --project   # Clear project setting`,
	RunE: runConfigVCSShow,
}

var configVCSShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current VCS settings and detection",
	RunE:  runConfigVCSShow,
}

var configVCSSetCmd = &cobra.Command{
	Use:   "set <type>",
	Short: "Set VCS type (git or jj)",
	Long: `Set the version control system type.

Valid types: git, jj

Use --project to set for the current project only (stored in .juggle/config.json).
Without --project, sets the global default (stored in ~/.juggle/config.json).`,
	Args: cobra.ExactArgs(1),
	RunE: runConfigVCSSet,
}

var configVCSClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear VCS setting (use auto-detection)",
	Long: `Clear the VCS setting to use auto-detection.

Use --project to clear the project setting only.
Without --project, clears the global setting.`,
	RunE: runConfigVCSClear,
}

func init() {
	configVCSSetCmd.Flags().BoolVar(&configVCSProjectFlag, "project", false, "Set for this project only (vs global)")
	configVCSClearCmd.Flags().BoolVar(&configVCSProjectFlag, "project", false, "Clear for this project only (vs global)")

	configVCSCmd.AddCommand(configVCSShowCmd)
	configVCSCmd.AddCommand(configVCSSetCmd)
	configVCSCmd.AddCommand(configVCSClearCmd)

	configCmd.AddCommand(configVCSCmd)
}

func runConfigVCSShow(cmd *cobra.Command, args []string) error {
	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	// Get global config
	globalVCS, err := session.GetGlobalVCSWithOptions(GetConfigOptions())
	if err != nil {
		return fmt.Errorf("failed to load global config: %w", err)
	}

	fmt.Println(labelStyle.Render("VCS Settings:"))
	fmt.Println()

	// Global setting
	fmt.Printf("  %s: ", keyStyle.Render("global"))
	if globalVCS == "" {
		fmt.Println(dimStyle.Render("(not set)"))
	} else {
		fmt.Println(valueStyle.Render(globalVCS))
	}

	// Try to load project config
	cwd, err := GetWorkingDir()
	if err == nil {
		projectVCS, err := session.GetProjectVCS(cwd)
		if err == nil {
			fmt.Printf("  %s: ", keyStyle.Render("project"))
			if projectVCS == "" {
				fmt.Println(dimStyle.Render("(not set)"))
			} else {
				fmt.Println(valueStyle.Render(projectVCS))
			}

			// Show auto-detection result
			detected := autoDetectVCS(cwd)
			fmt.Printf("  %s: ", keyStyle.Render("auto-detected"))
			fmt.Println(valueStyle.Render(detected))

			// Show effective VCS
			effective := resolveVCS(cwd, projectVCS, globalVCS)
			fmt.Println()
			fmt.Printf("  %s: ", keyStyle.Render("effective"))
			fmt.Println(valueStyle.Render(effective))
		}
	}

	return nil
}

func runConfigVCSSet(cmd *cobra.Command, args []string) error {
	vcsType := vcs.VCSType(strings.ToLower(strings.TrimSpace(args[0])))
	if !vcsType.IsValid() {
		return fmt.Errorf("invalid VCS type: %s (must be 'git' or 'jj')", args[0])
	}

	if configVCSProjectFlag {
		cwd, err := GetWorkingDir()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		if err := session.UpdateProjectVCS(cwd, string(vcsType)); err != nil {
			return fmt.Errorf("failed to set project VCS: %w", err)
		}
		fmt.Printf("Set project VCS to: %s\n", vcsType)
	} else {
		if err := session.UpdateGlobalVCSWithOptions(GetConfigOptions(), string(vcsType)); err != nil {
			return fmt.Errorf("failed to set global VCS: %w", err)
		}
		fmt.Printf("Set global VCS to: %s\n", vcsType)
	}

	return nil
}

func runConfigVCSClear(cmd *cobra.Command, args []string) error {
	if configVCSProjectFlag {
		cwd, err := GetWorkingDir()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		if err := session.ClearProjectVCS(cwd); err != nil {
			return fmt.Errorf("failed to clear project VCS: %w", err)
		}
		fmt.Println("Cleared project VCS setting.")
	} else {
		if err := session.ClearGlobalVCSWithOptions(GetConfigOptions()); err != nil {
			return fmt.Errorf("failed to clear global VCS: %w", err)
		}
		fmt.Println("Cleared global VCS setting.")
	}

	return nil
}

// autoDetectVCS checks for .jj or .git directories
func autoDetectVCS(projectDir string) string {
	if _, err := os.Stat(filepath.Join(projectDir, ".jj")); err == nil {
		return "jj"
	}
	if _, err := os.Stat(filepath.Join(projectDir, ".git")); err == nil {
		return "git"
	}
	return "git" // default
}

// resolveVCS determines the effective VCS using resolution priority
func resolveVCS(projectDir, projectVCS, globalVCS string) string {
	if projectVCS != "" {
		return projectVCS
	}
	if globalVCS != "" {
		return globalVCS
	}
	return autoDetectVCS(projectDir)
}
