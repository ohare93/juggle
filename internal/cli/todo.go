package cli

import (
	"fmt"
	"strconv"

	"github.com/ohare93/juggle/internal/session"
	"github.com/spf13/cobra"
)

var (
	todoBallID          string
	todoDescriptionFlag string
	todoVerboseFlag     bool
)

var todoCmd = &cobra.Command{
	Use:     "todo",
	Aliases: []string{"t", "todos"},
	Short: "Manage todos for a session",
	Long:  `Add, list, complete, and manage todo items for a work session.`,
}

var todoAddCmd = &cobra.Command{
	Use:   "add <todo> [<todo2> <todo3> ...]",
	Short: "Add one or more todos to the current session",
	Long: `Add todo items to a session. You can add multiple todos at once.

Examples:
  juggler todo add "Fix bug in auth flow"
  juggler todo add "Write tests" "Update docs" "Deploy to staging"
  juggler todo add --ball my-app-1 "Review PR"
  juggler todo add --description "More context here" "Main task"`,
	Args: cobra.MinimumNArgs(1),
	RunE: runTodoAdd,
}

var todoListCmd = &cobra.Command{
	Use:   "list",
	Short: "List todos for the current session",
	Long: `Display all todos for the current session with their completion status.

Examples:
  juggler todo list
  juggler todo list --ball my-app-1`,
	RunE: runTodoList,
}

var todoDoneCmd = &cobra.Command{
	Use:   "done <index> [<index2> <index3> ...]",
	Short: "Mark todos as complete by index (1-based)",
	Long: `Mark one or more todos as done. Use the index from 'juggler todo list'.

Examples:
  juggler todo done 1
  juggler todo done 1 3 5`,
	Args: cobra.MinimumNArgs(1),
	RunE: runTodoDone,
}

var todoRmCmd = &cobra.Command{
	Use:   "rm <index> [<index2> <index3> ...]",
	Short: "Remove todos by index (1-based)",
	Long: `Remove one or more todos. Use the index from 'juggler todo list'.

Examples:
  juggler todo rm 2
  juggler todo rm 1 3`,
	Args: cobra.MinimumNArgs(1),
	RunE: runTodoRm,
}

var todoEditCmd = &cobra.Command{
	Use:   "edit <index> <new-text>",
	Short: "Edit a todo's text",
	Long: `Edit the text of a todo item. Use the index from 'juggler todo list'.

Examples:
  juggler todo edit 1 "Updated task description"`,
	Args: cobra.ExactArgs(2),
	RunE: runTodoEdit,
}

var todoClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Remove all todos from the current session",
	Long:  `Clear all todo items from the current session.`,
	RunE:  runTodoClear,
}

var todoDescribeCmd = &cobra.Command{
	Use:   "describe <index> <description>",
	Short: "Add or update a todo's description",
	Long: `Add or update the description of a todo item. Use the index from 'juggler todo list'.

Examples:
  juggler todo describe 1 "Need to check with team before proceeding"
  juggler todo describe 2 "Blocked by API changes"`,
	Args: cobra.ExactArgs(2),
	RunE: runTodoDescribe,
}

func init() {
	// Add --ball flag to all todo subcommands
	todoAddCmd.Flags().StringVar(&todoBallID, "ball", "", "Target specific ball by ID")
	todoListCmd.Flags().StringVar(&todoBallID, "ball", "", "Target specific ball by ID")
	todoDoneCmd.Flags().StringVar(&todoBallID, "ball", "", "Target specific ball by ID")
	todoRmCmd.Flags().StringVar(&todoBallID, "ball", "", "Target specific ball by ID")
	todoEditCmd.Flags().StringVar(&todoBallID, "ball", "", "Target specific ball by ID")
	todoClearCmd.Flags().StringVar(&todoBallID, "ball", "", "Target specific ball by ID")
	todoDescribeCmd.Flags().StringVar(&todoBallID, "ball", "", "Target specific ball by ID")

	// Add --description flag to add command
	todoAddCmd.Flags().StringVarP(&todoDescriptionFlag, "description", "d", "", "Description/context for the todo")

	// Add --verbose flag to list command
	todoListCmd.Flags().BoolVarP(&todoVerboseFlag, "verbose", "v", false, "Show todo descriptions")

	todoCmd.AddCommand(todoAddCmd)
	todoCmd.AddCommand(todoListCmd)
	todoCmd.AddCommand(todoDoneCmd)
	todoCmd.AddCommand(todoRmCmd)
	todoCmd.AddCommand(todoEditCmd)
	todoCmd.AddCommand(todoClearCmd)
	todoCmd.AddCommand(todoDescribeCmd)
}

// getCurrentBall finds the appropriate ball to operate on
// Priority: 1) explicit --ball flag, 2) current Zellij tab match, 3) most recent active ball
func getCurrentBall(store *session.Store) (*session.Session, error) {
	// If explicit ball ID provided, use that
	if todoBallID != "" {
		ball, err := store.GetBallByID(todoBallID)
		if err != nil {
			return nil, fmt.Errorf("ball %s not found: %w", todoBallID, err)
		}
		return ball, nil
	}

	// Try to get most recently active juggling ball
	jugglingBalls, err := store.GetJugglingBalls()
	if err != nil || len(jugglingBalls) == 0 {
		// Provide helpful error
		balls, _ := store.LoadBalls()
		activeBalls := 0
		for _, b := range balls {
			if b.ActiveState == session.ActiveJuggling || b.ActiveState == session.ActiveDropped {
				activeBalls++
			}
		}

		if activeBalls == 0 {
			return nil, fmt.Errorf("no active balls found in current project. Use 'juggle <ball-id>' to start juggling")
		} else if activeBalls > 1 {
			return nil, fmt.Errorf("multiple active balls found. Use --ball <id> to specify which one")
		}
		return nil, fmt.Errorf("no juggling balls found. Use --ball <id> to specify which ball")
	}

	return jugglingBalls[0], nil
}

func runTodoAdd(cmd *cobra.Command, args []string) error {
	cwd, err := GetWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	store, err := NewStoreForCommand(cwd)
	if err != nil {
		return fmt.Errorf("failed to initialize store: %w", err)
	}

	ball, err := getCurrentBall(store)
	if err != nil {
		return err
	}

	// Add todos (with description if provided for first todo)
	if todoDescriptionFlag != "" {
		// If description provided, only add first todo with it
		if len(args) > 1 {
			return fmt.Errorf("description can only be used with a single todo")
		}
		ball.AddTodoWithDescription(args[0], todoDescriptionFlag)
	} else {
		ball.AddTodos(args)
	}

	if err := store.UpdateBall(ball); err != nil {
		return fmt.Errorf("failed to update ball: %w", err)
	}

	// Success message
	count := len(args)
	plural := "todo"
	if count > 1 {
		plural = "todos"
	}

	fmt.Printf("✓ Added %d %s to ball: %s\n", count, plural, ball.ID)
	fmt.Printf("  Intent: %s\n", ball.Intent)

	// Show the added todos
	if count <= 5 {
		fmt.Println("\nAdded:")
		startIdx := len(ball.Todos) - count
		for i, todo := range ball.Todos[startIdx:] {
			fmt.Printf("  %d. [ ] %s\n", startIdx+i+1, todo.Text)
			if todo.Description != "" {
				fmt.Printf("      %s\n", todo.Description)
			}
		}
	}

	return nil
}

func runTodoList(cmd *cobra.Command, args []string) error {
	cwd, err := GetWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	store, err := NewStoreForCommand(cwd)
	if err != nil {
		return fmt.Errorf("failed to initialize store: %w", err)
	}

	ball, err := getCurrentBall(store)
	if err != nil {
		return err
	}

	if len(ball.Todos) == 0 {
		fmt.Printf("No todos for ball: %s\n", ball.ID)
		fmt.Printf("  Intent: %s\n", ball.Intent)
		fmt.Println("\nAdd todos with: juggler todo add <todo-text>")
		return nil
	}

	total, completed := ball.TodoStats()
	fmt.Printf("Todos for ball: %s\n", ball.ID)
	fmt.Printf("  Intent: %s\n", ball.Intent)
	fmt.Printf("  Progress: %d/%d complete (%.0f%%)\n\n", completed, total, float64(completed)/float64(total)*100)

	for i, todo := range ball.Todos {
		checkbox := "[ ]"
		if todo.Done {
			checkbox = "[✓]"
		}
		fmt.Printf("  %d. %s %s\n", i+1, checkbox, todo.Text)
		if todoVerboseFlag && todo.Description != "" {
			fmt.Printf("      %s\n", todo.Description)
		}
	}

	return nil
}

func runTodoDone(cmd *cobra.Command, args []string) error {
	cwd, err := GetWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	store, err := NewStoreForCommand(cwd)
	if err != nil {
		return fmt.Errorf("failed to initialize store: %w", err)
	}

	ball, err := getCurrentBall(store)
	if err != nil {
		return err
	}

	// Parse and validate all indices
	indices := make([]int, 0, len(args))
	for _, arg := range args {
		idx, err := strconv.Atoi(arg)
		if err != nil {
			return fmt.Errorf("invalid index: %s (must be a number)", arg)
		}
		if idx < 1 || idx > len(ball.Todos) {
			return fmt.Errorf("invalid index: %d (must be between 1 and %d)", idx, len(ball.Todos))
		}
		indices = append(indices, idx-1) // Convert to 0-based
	}

	// Toggle todos
	for _, idx := range indices {
		if err := ball.ToggleTodo(idx); err != nil {
			return err
		}
	}

	if err := store.UpdateBall(ball); err != nil {
		return fmt.Errorf("failed to update ball: %w", err)
	}

	// Show what was done
	for _, idx := range indices {
		todo := ball.Todos[idx]
		status := "completed"
		if !todo.Done {
			status = "marked incomplete"
		}
		fmt.Printf("✓ Todo %d %s: %s\n", idx+1, status, todo.Text)
	}

	// Show stats
	total, completed := ball.TodoStats()
	fmt.Printf("\nProgress: %d/%d complete (%.0f%%)\n", completed, total, float64(completed)/float64(total)*100)

	return nil
}

func runTodoRm(cmd *cobra.Command, args []string) error {
	cwd, err := GetWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	store, err := NewStoreForCommand(cwd)
	if err != nil {
		return fmt.Errorf("failed to initialize store: %w", err)
	}

	ball, err := getCurrentBall(store)
	if err != nil {
		return err
	}

	// Parse indices and sort in descending order (so we can remove from end to start)
	indices := make([]int, 0, len(args))
	for _, arg := range args {
		idx, err := strconv.Atoi(arg)
		if err != nil {
			return fmt.Errorf("invalid index: %s (must be a number)", arg)
		}
		if idx < 1 || idx > len(ball.Todos) {
			return fmt.Errorf("invalid index: %d (must be between 1 and %d)", idx, len(ball.Todos))
		}
		indices = append(indices, idx-1) // Convert to 0-based
	}

	// Sort descending so we remove from end first
	for i := 0; i < len(indices)-1; i++ {
		for j := i + 1; j < len(indices); j++ {
			if indices[i] < indices[j] {
				indices[i], indices[j] = indices[j], indices[i]
			}
		}
	}

	// Store removed todos for display
	removed := make([]string, 0, len(indices))
	for _, idx := range indices {
		removed = append(removed, ball.Todos[idx].Text)
	}

	// Remove todos (from end to start so indices remain valid)
	for _, idx := range indices {
		if err := ball.RemoveTodo(idx); err != nil {
			return err
		}
	}

	if err := store.UpdateBall(ball); err != nil {
		return fmt.Errorf("failed to update ball: %w", err)
	}

	// Show what was removed
	for _, text := range removed {
		fmt.Printf("✓ Removed: %s\n", text)
	}

	fmt.Printf("\n%d todo(s) remaining\n", len(ball.Todos))

	return nil
}

func runTodoEdit(cmd *cobra.Command, args []string) error {
	cwd, err := GetWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	store, err := NewStoreForCommand(cwd)
	if err != nil {
		return fmt.Errorf("failed to initialize store: %w", err)
	}

	ball, err := getCurrentBall(store)
	if err != nil {
		return err
	}

	idx, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid index: %s (must be a number)", args[0])
	}

	if idx < 1 || idx > len(ball.Todos) {
		return fmt.Errorf("invalid index: %d (must be between 1 and %d)", idx, len(ball.Todos))
	}

	oldText := ball.Todos[idx-1].Text
	newText := args[1]

	if err := ball.EditTodo(idx-1, newText); err != nil {
		return err
	}

	if err := store.UpdateBall(ball); err != nil {
		return fmt.Errorf("failed to update ball: %w", err)
	}

	fmt.Printf("✓ Updated todo %d\n", idx)
	fmt.Printf("  Old: %s\n", oldText)
	fmt.Printf("  New: %s\n", newText)

	return nil
}

func runTodoClear(cmd *cobra.Command, args []string) error {
	cwd, err := GetWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	store, err := NewStoreForCommand(cwd)
	if err != nil {
		return fmt.Errorf("failed to initialize store: %w", err)
	}

	ball, err := getCurrentBall(store)
	if err != nil {
		return err
	}

	count := len(ball.Todos)
	if count == 0 {
		fmt.Println("No todos to clear")
		return nil
	}

	ball.ClearTodos()

	if err := store.UpdateBall(ball); err != nil {
		return fmt.Errorf("failed to update ball: %w", err)
	}

	fmt.Printf("✓ Cleared %d todo(s) from ball: %s\n", count, ball.ID)

	return nil
}

func runTodoDescribe(cmd *cobra.Command, args []string) error {
	cwd, err := GetWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	store, err := NewStoreForCommand(cwd)
	if err != nil {
		return fmt.Errorf("failed to initialize store: %w", err)
	}

	ball, err := getCurrentBall(store)
	if err != nil {
		return err
	}

	idx, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid index: %s (must be a number)", args[0])
	}

	if idx < 1 || idx > len(ball.Todos) {
		return fmt.Errorf("invalid index: %d (must be between 1 and %d)", idx, len(ball.Todos))
	}

	description := args[1]

	if err := ball.SetTodoDescription(idx-1, description); err != nil {
		return err
	}

	if err := store.UpdateBall(ball); err != nil {
		return fmt.Errorf("failed to update ball: %w", err)
	}

	fmt.Printf("✓ Updated description for todo %d: %s\n", idx, ball.Todos[idx-1].Text)
	fmt.Printf("  Description: %s\n", description)

	return nil
}
