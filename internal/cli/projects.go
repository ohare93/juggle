package cli

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/ohare93/juggle/internal/session"
	"github.com/spf13/cobra"
)

var projectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "List and manage tracked projects",
	Long:  `List all projects with .juggler directories and manage search paths.`,
	RunE:  runProjects,
}

var projectsAddCmd = &cobra.Command{
	Use:   "add <path>",
	Short: "Add a search path for project discovery",
	Args:  cobra.ExactArgs(1),
	RunE:  runProjectsAdd,
}

var projectsRemoveCmd = &cobra.Command{
	Use:   "remove <path>",
	Short: "Remove a search path",
	Args:  cobra.ExactArgs(1),
	RunE:  runProjectsRemove,
}

func init() {
	projectsCmd.AddCommand(projectsAddCmd)
	projectsCmd.AddCommand(projectsRemoveCmd)
}

func runProjects(cmd *cobra.Command, args []string) error {
	config, err := LoadConfigForCommand()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get project info
	projectInfos, err := session.GetProjectsInfo(config)
	if err != nil {
		return fmt.Errorf("failed to get project info: %w", err)
	}

	if len(projectInfos) == 0 {
		fmt.Println("No projects with .juggler directories found.")
		fmt.Println("\nSearch paths:")
		for _, path := range config.SearchPaths {
			if _, err := os.Stat(path); os.IsNotExist(err) {
				fmt.Printf("  - %s (does not exist)\n", path)
			} else {
				fmt.Printf("  - %s\n", path)
			}
		}
		fmt.Println("\nAdd a search path with: juggler projects add <path>")
		return nil
	}

	// Define styles
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("8")).
		Padding(0, 1)

	activeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	plannedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	blockedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("11"))

	// Print header
	fmt.Println(
		headerStyle.Render(padRight("PROJECT", 50)) +
		headerStyle.Render(padRight("JUGGLING", 10)) +
		headerStyle.Render(padRight("READY", 8)) +
		headerStyle.Render(padRight("DROPPED", 9)) +
		headerStyle.Render(padRight("COMPLETE", 10)),
	)

	// Print projects
	for _, info := range projectInfos {
		projectCell := info.Path
		if len(projectCell) > 48 {
			projectCell = "..." + projectCell[len(projectCell)-45:]
		}

		fmt.Println(
			padRight(projectCell, 50) +
			activeStyle.Render(padRight(fmt.Sprintf("%d", info.JugglingBalls), 10)) +
			plannedStyle.Render(padRight(fmt.Sprintf("%d", info.ReadyBalls), 8)) +
			blockedStyle.Render(padRight(fmt.Sprintf("%d", info.DroppedBalls), 9)) +
			padRight(fmt.Sprintf("%d", info.CompleteBalls), 10),
		)
	}

	fmt.Printf("\n%d project(s) found\n", len(projectInfos))

	// Show search paths
	fmt.Println("\nSearch paths:")
	for _, path := range config.SearchPaths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			fmt.Printf("  - %s (does not exist)\n", path)
		} else {
			fmt.Printf("  - %s\n", path)
		}
	}

	return nil
}

func runProjectsAdd(cmd *cobra.Command, args []string) error {
	path := args[0]

	// Check if path exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("path does not exist: %s", path)
	}

	config, err := LoadConfigForCommand()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if !config.AddSearchPath(path) {
		fmt.Printf("Path already in search paths: %s\n", path)
		return nil
	}

	if err := config.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("✓ Added search path: %s\n", path)
	return nil
}

func runProjectsRemove(cmd *cobra.Command, args []string) error {
	path := args[0]

	config, err := LoadConfigForCommand()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if !config.RemoveSearchPath(path) {
		return fmt.Errorf("path not found in search paths: %s", path)
	}

	if err := config.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("✓ Removed search path: %s\n", path)
	return nil
}
