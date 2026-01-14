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

			// AC Templates
			fmt.Printf("  %s: ", keyStyle.Render("ac_templates"))
			if len(projectConfig.ACTemplates) == 0 {
				fmt.Println("(empty)")
			} else {
				fmt.Println()
				for _, template := range projectConfig.ACTemplates {
					fmt.Printf("    - %s\n", template)
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

// AC Templates command variables
var configTemplatesEditFlag bool
var configTemplatesYesFlag bool

var configTemplatesCmd = &cobra.Command{
	Use:   "templates",
	Short: "Manage AC templates for ball creation",
	Long: `Manage AC templates that appear as selectable suggestions when creating balls.

Unlike repo-level ACs (which apply automatically to all balls), templates are
optional suggestions that can be selected during ball creation.

Commands:
  config templates list              List current AC templates
  config templates add "template"    Add a new template
  config templates set --edit        Edit templates in $EDITOR
  config templates clear             Remove all templates`,
	RunE: runConfigTemplatesList,
}

var configTemplatesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List AC templates",
	RunE:  runConfigTemplatesList,
}

var configTemplatesAddCmd = &cobra.Command{
	Use:   "add <template>",
	Short: "Add an AC template",
	Args:  cobra.ExactArgs(1),
	RunE:  runConfigTemplatesAdd,
}

var configTemplatesSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Set AC templates",
	Long: `Set AC templates for ball creation.

Without flags, prompts for templates one per line.
With --edit, opens templates in $EDITOR for editing.

Examples:
  juggle config templates set --edit
  juggle config templates set  # Interactive prompt`,
	RunE: runConfigTemplatesSet,
}

var configTemplatesClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear all AC templates",
	Long: `Clear all AC templates.

Use --yes (-y) to skip the confirmation prompt (for headless/automated use).`,
	RunE: runConfigTemplatesClear,
}

func init() {
	configACSetCmd.Flags().BoolVar(&configACEditFlag, "edit", false, "Open criteria in $EDITOR")
	configACClearCmd.Flags().BoolVarP(&configACYesFlag, "yes", "y", false, "Skip confirmation prompt (for headless mode)")

	configACCmd.AddCommand(configACListCmd)
	configACCmd.AddCommand(configACAddCmd)
	configACCmd.AddCommand(configACSetCmd)
	configACCmd.AddCommand(configACClearCmd)

	configCmd.AddCommand(configACCmd)

	// AC Templates commands
	configTemplatesSetCmd.Flags().BoolVar(&configTemplatesEditFlag, "edit", false, "Open templates in $EDITOR")
	configTemplatesClearCmd.Flags().BoolVarP(&configTemplatesYesFlag, "yes", "y", false, "Skip confirmation prompt (for headless mode)")

	configTemplatesCmd.AddCommand(configTemplatesListCmd)
	configTemplatesCmd.AddCommand(configTemplatesAddCmd)
	configTemplatesCmd.AddCommand(configTemplatesSetCmd)
	configTemplatesCmd.AddCommand(configTemplatesClearCmd)

	configCmd.AddCommand(configTemplatesCmd)

	// Paths commands
	configPathsPruneCmd.Flags().BoolVarP(&configPathsPruneYesFlag, "yes", "y", false, "Skip confirmation prompt")
	configPathsCmd.AddCommand(configPathsListCmd)
	configPathsCmd.AddCommand(configPathsPruneCmd)

	configCmd.AddCommand(configPathsCmd)
}

var configPathsPruneYesFlag bool

var configPathsCmd = &cobra.Command{
	Use:   "paths",
	Short: "Manage search paths for project discovery",
	Long: `Manage the search paths used to discover juggle projects.

Commands:
  config paths list           List current search paths
  config paths prune          Remove non-existent paths from config`,
	RunE: runConfigPathsList,
}

var configPathsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List search paths",
	RunE:  runConfigPathsList,
}

var configPathsPruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Remove non-existent paths from search_paths",
	Long: `Remove non-existent directories from the global search_paths config.

This is useful for cleaning up stale paths (e.g., from deleted projects or
old test directories that were accidentally added).

Use --yes (-y) to skip the confirmation prompt.`,
	RunE: runConfigPathsPrune,
}

func runConfigPathsList(cmd *cobra.Command, args []string) error {
	config, err := session.LoadConfigWithOptions(GetConfigOptions())
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	fmt.Println(labelStyle.Render("Search Paths:"))
	fmt.Println()

	if len(config.SearchPaths) == 0 {
		fmt.Println("  (none)")
		return nil
	}

	existStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	missingStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9"))

	for _, path := range config.SearchPaths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			fmt.Printf("  %s %s\n", missingStyle.Render("✗"), path)
		} else {
			fmt.Printf("  %s %s\n", existStyle.Render("✓"), path)
		}
	}

	return nil
}

func runConfigPathsPrune(cmd *cobra.Command, args []string) error {
	config, err := session.LoadConfigWithOptions(GetConfigOptions())
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Find paths to remove
	var toRemove []string
	var toKeep []string
	for _, path := range config.SearchPaths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			toRemove = append(toRemove, path)
		} else {
			toKeep = append(toKeep, path)
		}
	}

	if len(toRemove) == 0 {
		fmt.Println("No non-existent paths to remove.")
		return nil
	}

	// Show what will be removed
	fmt.Printf("Found %d non-existent path(s) to remove:\n", len(toRemove))
	for _, path := range toRemove {
		fmt.Printf("  - %s\n", path)
	}
	fmt.Println()

	// Confirm unless --yes flag is set
	if !configPathsPruneYesFlag {
		confirmed, err := ConfirmSingleKey(fmt.Sprintf("Remove %d path(s)?", len(toRemove)))
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	// Update config
	config.SearchPaths = toKeep
	if err := config.SaveWithOptions(GetConfigOptions()); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Removed %d path(s). %d path(s) remaining.\n", len(toRemove), len(toKeep))
	return nil
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

// AC Templates command handlers

func runConfigTemplatesList(cmd *cobra.Command, args []string) error {
	cwd, err := GetWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	templates, err := session.GetProjectACTemplates(cwd)
	if err != nil {
		return fmt.Errorf("failed to load AC templates: %w", err)
	}

	if len(templates) == 0 {
		fmt.Println("No AC templates configured.")
		fmt.Println("\nAdd templates with: juggle config templates add \"template\"")
		fmt.Println("Or edit in $EDITOR: juggle config templates set --edit")
		return nil
	}

	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	fmt.Println(labelStyle.Render("AC Templates:"))
	fmt.Println()
	for i, template := range templates {
		fmt.Printf("  %d. %s\n", i+1, template)
	}
	fmt.Println()
	fmt.Printf("These templates appear as selectable options when creating balls.\n")

	return nil
}

func runConfigTemplatesAdd(cmd *cobra.Command, args []string) error {
	template := args[0]

	cwd, err := GetWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Load existing templates
	templates, err := session.GetProjectACTemplates(cwd)
	if err != nil {
		// If config doesn't exist, start with empty list
		templates = []string{}
	}

	// Add new template
	templates = append(templates, template)

	// Save
	if err := session.UpdateProjectACTemplates(cwd, templates); err != nil {
		return fmt.Errorf("failed to save AC templates: %w", err)
	}

	fmt.Printf("Added AC template: %s\n", template)
	fmt.Printf("Total templates: %d\n", len(templates))

	return nil
}

func runConfigTemplatesSet(cmd *cobra.Command, args []string) error {
	cwd, err := GetWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Load existing templates
	existingTemplates, err := session.GetProjectACTemplates(cwd)
	if err != nil {
		existingTemplates = []string{}
	}

	var newTemplates []string

	if configTemplatesEditFlag {
		// Edit in $EDITOR
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "vi"
		}

		// Create temp file with current templates
		tmpFile, err := os.CreateTemp("", "juggle-templates-*.txt")
		if err != nil {
			return fmt.Errorf("failed to create temp file: %w", err)
		}
		tmpPath := tmpFile.Name()
		defer os.Remove(tmpPath)

		// Write header and existing templates
		tmpFile.WriteString("# AC Templates\n")
		tmpFile.WriteString("# One template per line. Lines starting with # are comments.\n")
		tmpFile.WriteString("# Empty lines are ignored.\n\n")
		for _, template := range existingTemplates {
			tmpFile.WriteString(template + "\n")
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
			newTemplates = append(newTemplates, line)
		}
	} else {
		// Interactive prompt
		fmt.Println("Enter AC templates (one per line, empty line to finish):")
		if len(existingTemplates) > 0 {
			fmt.Println("Current templates will be replaced.")
		}

		reader := bufio.NewReader(os.Stdin)
		for {
			fmt.Print("  > ")
			input, err := reader.ReadString('\n')
			if err != nil {
				break
			}
			template := strings.TrimSpace(input)
			if template == "" {
				break
			}
			newTemplates = append(newTemplates, template)
		}
	}

	// Save
	if err := session.UpdateProjectACTemplates(cwd, newTemplates); err != nil {
		return fmt.Errorf("failed to save AC templates: %w", err)
	}

	if len(newTemplates) == 0 {
		fmt.Println("Cleared all AC templates.")
	} else {
		fmt.Printf("Updated AC templates (%d items):\n", len(newTemplates))
		for i, template := range newTemplates {
			fmt.Printf("  %d. %s\n", i+1, template)
		}
	}

	return nil
}

func runConfigTemplatesClear(cmd *cobra.Command, args []string) error {
	cwd, err := GetWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Confirm (skip with --yes flag)
	if !configTemplatesYesFlag {
		confirmed, err := ConfirmSingleKey("Clear all AC templates?")
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	// Save empty list
	if err := session.UpdateProjectACTemplates(cwd, []string{}); err != nil {
		return fmt.Errorf("failed to clear AC templates: %w", err)
	}

	fmt.Println("Cleared all AC templates.")
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

// Provider command variables
var configProviderProjectFlag bool

// configProviderCmd is the parent command for provider settings
var configProviderCmd = &cobra.Command{
	Use:   "provider",
	Short: "Manage agent provider settings",
	Long: `Manage the agent provider (CLI tool) used for running agents.

By default, juggle uses "claude" (Claude Code CLI).
You can override this globally or per-project.

Available providers:
  claude    - Claude Code CLI (default)
  opencode  - OpenCode CLI

Resolution order (highest to lowest priority):
  1. CLI flag (--provider on agent commands)
  2. Project config (.juggle/config.json agent_provider field)
  3. Global config (~/.juggle/config.json agent_provider field)
  4. Default: claude

Commands:
  config provider show              Show current provider settings
  config provider set <provider>    Set provider (claude or opencode)
  config provider clear             Clear provider setting

Examples:
  juggle config provider show
  juggle config provider set claude           # Use claude globally
  juggle config provider set opencode         # Use opencode globally
  juggle config provider set claude --project # Use claude for this project only
  juggle config provider clear                # Clear global setting
  juggle config provider clear --project      # Clear project setting`,
	RunE: runConfigProviderShow,
}

var configProviderShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current provider settings",
	RunE:  runConfigProviderShow,
}

var configProviderSetCmd = &cobra.Command{
	Use:   "set <provider>",
	Short: "Set agent provider (claude or opencode)",
	Long: `Set the agent provider.

Valid providers: claude, opencode

Use --project to set for the current project only (stored in .juggle/config.json).
Without --project, sets the global default (stored in ~/.juggle/config.json).`,
	Args: cobra.ExactArgs(1),
	RunE: runConfigProviderSet,
}

var configProviderClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear provider setting (use default)",
	Long: `Clear the provider setting to use the default (claude).

Use --project to clear the project setting only.
Without --project, clears the global setting.`,
	RunE: runConfigProviderClear,
}

func init() {
	configProviderSetCmd.Flags().BoolVar(&configProviderProjectFlag, "project", false, "Set for this project only (vs global)")
	configProviderClearCmd.Flags().BoolVar(&configProviderProjectFlag, "project", false, "Clear for this project only (vs global)")

	configProviderCmd.AddCommand(configProviderShowCmd)
	configProviderCmd.AddCommand(configProviderSetCmd)
	configProviderCmd.AddCommand(configProviderClearCmd)

	configCmd.AddCommand(configProviderCmd)
}

func runConfigProviderShow(cmd *cobra.Command, args []string) error {
	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	// Get global config
	globalProvider, err := session.GetGlobalAgentProviderWithOptions(GetConfigOptions())
	if err != nil {
		return fmt.Errorf("failed to load global config: %w", err)
	}

	fmt.Println(labelStyle.Render("Provider Settings:"))
	fmt.Println()

	// Global setting
	fmt.Printf("  %s: ", keyStyle.Render("global"))
	if globalProvider == "" {
		fmt.Println(dimStyle.Render("(not set)"))
	} else {
		fmt.Println(valueStyle.Render(globalProvider))
	}

	// Try to load project config
	cwd, err := GetWorkingDir()
	if err == nil {
		projectProvider, err := session.GetProjectAgentProvider(cwd)
		if err == nil {
			fmt.Printf("  %s: ", keyStyle.Render("project"))
			if projectProvider == "" {
				fmt.Println(dimStyle.Render("(not set)"))
			} else {
				fmt.Println(valueStyle.Render(projectProvider))
			}

			// Show effective provider
			effective := resolveProvider(projectProvider, globalProvider)
			fmt.Println()
			fmt.Printf("  %s: ", keyStyle.Render("effective"))
			fmt.Println(valueStyle.Render(effective))
		}
	}

	return nil
}

func runConfigProviderSet(cmd *cobra.Command, args []string) error {
	provider := strings.ToLower(strings.TrimSpace(args[0]))
	if provider != "claude" && provider != "opencode" {
		return fmt.Errorf("invalid provider: %s (must be 'claude' or 'opencode')", args[0])
	}

	if configProviderProjectFlag {
		cwd, err := GetWorkingDir()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		if err := session.UpdateProjectAgentProvider(cwd, provider); err != nil {
			return fmt.Errorf("failed to set project provider: %w", err)
		}
		fmt.Printf("Set project provider to: %s\n", provider)
	} else {
		if err := session.UpdateGlobalAgentProviderWithOptions(GetConfigOptions(), provider); err != nil {
			return fmt.Errorf("failed to set global provider: %w", err)
		}
		fmt.Printf("Set global provider to: %s\n", provider)
	}

	return nil
}

func runConfigProviderClear(cmd *cobra.Command, args []string) error {
	if configProviderProjectFlag {
		cwd, err := GetWorkingDir()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		if err := session.ClearProjectAgentProvider(cwd); err != nil {
			return fmt.Errorf("failed to clear project provider: %w", err)
		}
		fmt.Println("Cleared project provider setting.")
	} else {
		if err := session.ClearGlobalAgentProviderWithOptions(GetConfigOptions()); err != nil {
			return fmt.Errorf("failed to clear global provider: %w", err)
		}
		fmt.Println("Cleared global provider setting.")
	}

	return nil
}

// resolveProvider determines the effective provider using resolution priority
func resolveProvider(projectProvider, globalProvider string) string {
	if projectProvider != "" {
		return projectProvider
	}
	if globalProvider != "" {
		return globalProvider
	}
	return "claude" // default
}
