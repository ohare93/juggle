package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/ohare93/juggle/internal/session"
	"github.com/spf13/cobra"
)

// runRootCommand handles dynamic routing for the root command
func runRootCommand(cmd *cobra.Command, args []string) error {
	// No args: list juggling balls
	if len(args) == 0 {
		return listJugglingBalls(cmd)
	}

	// First arg determines action
	firstArg := args[0]

	// Check for special commands
	switch firstArg {
	case "balls":
		return listAllBalls(cmd)
	default:
		// Try to resolve as ball ID and perform operation
		return handleBallCommand(cmd, args)
	}
}

// listJugglingBalls lists all balls currently being juggled
func listJugglingBalls(cmd *cobra.Command) error {
	config, err := LoadConfigForCommand()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get current directory to create a store (needed for DiscoverProjectsForCommand)
	cwd, err := GetWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	store, err := NewStoreForCommand(cwd)
	if err != nil {
		return fmt.Errorf("failed to create store: %w", err)
	}

	projects, err := DiscoverProjectsForCommand(config, store)
	if err != nil {
		return fmt.Errorf("failed to discover projects: %w", err)
	}

	juggling, err := session.LoadJugglingBalls(projects)
	if err != nil {
		return fmt.Errorf("failed to load juggling balls: %w", err)
	}

	// cwd already retrieved above for highlighting and marker updates

	if len(juggling) == 0 {
		fmt.Println("No balls currently being juggled")
		fmt.Println()
		fmt.Println("To start juggling:")
		fmt.Println("  juggle start              - Create and start juggling a new ball")
		fmt.Println("  juggle plan               - Plan a ball for later")
		fmt.Println("  juggle <ball-id>          - Start juggling a planned ball")
		return nil
	}

	// Render juggling balls
	fmt.Printf("\n🎯 Currently Juggling (%d ball%s)\n\n", len(juggling), pluralize(len(juggling)))

	// Use consistent styles from styles.go
	inAirStyle := StyleInAir
	cwdHighlight := StyleHighlight
	projectStyle := StyleProject
	dimStyle := StyleDim

	// Calculate column widths
	maxIDLen := 0
	maxProjectLen := 0
	for _, ball := range juggling {
		idLen := len(ball.ShortID())
		if idLen > maxIDLen {
			maxIDLen = idLen
		}
		projectLen := len(filepath.Base(ball.WorkingDir))
		if projectLen > maxProjectLen {
			maxProjectLen = projectLen
		}
	}

	for _, ball := range juggling {
		// Determine style based on state
		stateStr := string(ball.State)
		stateStyle := inAirStyle // in_progress uses in-air style

		// Format columns without styling first (for padding)
		idStr := ball.ShortID()
		projectName := filepath.Base(ball.WorkingDir)
		priorityStr := string(ball.Priority)

		// Apply padding to plain text
		idPadded := padRight(idStr, maxIDLen)
		projectPadded := padRight(projectName, maxProjectLen)
		statePadded := padRight(stateStr, 13)
		priorityPadded := padRight(priorityStr, 7)

		// Apply styling after padding
		if ball.WorkingDir == cwd {
			idPadded = cwdHighlight.Render(idPadded)
		}
		projectPadded = projectStyle.Render(projectPadded)
		statePadded = stateStyle.Render(statePadded)
		priorityPadded = GetPriorityStyle(priorityStr).Render(priorityPadded)

		// Build the line with optional blocked reason and todo count
		intentDisplay := ball.Intent
		if ball.BlockedReason != "" {
			intentDisplay = fmt.Sprintf("%s %s", ball.Intent, dimStyle.Render("("+ball.BlockedReason+")"))
		}

		// Add todo completion summary if todos exist
		todoSummary := ""
		if len(ball.Todos) > 0 {
			todoSummary = fmt.Sprintf(" %s", dimStyle.Render("["+ball.TodoCompletionSummary()+"]"))
		}

		fmt.Printf("  [%s] %s  %s  %s  %s%s\n",
			idPadded,
			projectPadded,
			statePadded,
			priorityPadded,
			intentDisplay,
			todoSummary,
		)

		// Show description on next line if present
		if ball.Description != "" {
			descDisplay := ball.Description
			if len(descDisplay) > 80 {
				descDisplay = descDisplay[:77] + "..."
			}
			fmt.Printf("      %s\n", dimStyle.Render(descDisplay))
		}
	}

	fmt.Println()

	return nil
}

// listAllBalls lists all balls regardless of state
func listAllBalls(cmd *cobra.Command) error {
	// Get current directory
	cwd, err := GetWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// If current directory has .juggler, ensure it's tracked
	if _, err := os.Stat(filepath.Join(cwd, ".juggler")); err == nil {
		_ = session.EnsureProjectInSearchPaths(cwd)
	}

	config, err := LoadConfigForCommand()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	store, err := NewStoreForCommand(cwd)
	if err != nil {
		return fmt.Errorf("failed to create store: %w", err)
	}

	projects, err := DiscoverProjectsForCommand(config, store)
	if err != nil {
		return fmt.Errorf("failed to discover projects: %w", err)
	}

	allBalls, err := session.LoadAllBalls(projects)
	if err != nil {
		return fmt.Errorf("failed to load balls: %w", err)
	}

	if len(allBalls) == 0 {
		fmt.Println("No balls found")
		return nil
	}

	// Group by project (WorkingDir)
	byProject := make(map[string][]*session.Session)
	for _, ball := range allBalls {
		byProject[ball.WorkingDir] = append(byProject[ball.WorkingDir], ball)
	}

	// Get current directory for highlighting (cwd already declared above)
	// cwd variable is already set at function start
	cwdHighlight := StyleHighlight
	projectStyle := StyleProject.Bold(true)
	
	// State styles
	jugglingStyle := StyleJuggling
	readyStyle := StyleReady
	droppedStyle := StyleDropped
	completeStyle := StyleComplete

	stateStyles := map[session.BallState]lipgloss.Style{
		session.StateInProgress: jugglingStyle,
		session.StatePending:    readyStyle,
		session.StateBlocked:    droppedStyle,
		session.StateComplete:   completeStyle,
	}

	// Sort projects for consistent ordering
	projectPaths := make([]string, 0, len(byProject))
	for path := range byProject {
		projectPaths = append(projectPaths, path)
	}

	// Calculate max ID length across all balls for consistent alignment
	maxIDLen := 0
	for _, ball := range allBalls {
		idLen := len(ball.ShortID())
		if idLen > maxIDLen {
			maxIDLen = idLen
		}
	}

	// Display each project
	for _, projectPath := range projectPaths {
		balls := byProject[projectPath]
		projectName := filepath.Base(projectPath)

		fmt.Printf("\n%s (%d ball%s)\n\n",
			projectStyle.Render(projectName),
			len(balls),
			pluralize(len(balls)))

		// Group by state within project
		byState := make(map[session.BallState][]*session.Session)
		for _, ball := range balls {
			byState[ball.State] = append(byState[ball.State], ball)
		}

		// Display in state order
		stateOrder := []session.BallState{
			session.StateInProgress,
			session.StatePending,
			session.StateBlocked,
			session.StateComplete,
		}

		for _, state := range stateOrder {
			stateBalls := byState[state]
			if len(stateBalls) == 0 {
				continue
			}

			for _, ball := range stateBalls {
				// Determine state string and style
				stateStr := string(state)
				stateStyle := stateStyles[state]

				// Format columns without styling first (for padding)
				idStr := ball.ShortID()
				priorityStr := string(ball.Priority)

				// Apply padding to plain text
				idPadded := padRight(idStr, maxIDLen)
				statePadded := padRight(stateStr, 13)
				priorityPadded := padRight(priorityStr, 7)

				// Apply styling after padding
				if ball.WorkingDir == cwd {
					idPadded = cwdHighlight.Render(idPadded)
				}
				statePadded = stateStyle.Render(statePadded)
				priorityPadded = GetPriorityStyle(priorityStr).Render(priorityPadded)

				// Build the line with optional blocked reason and todo count
				intentDisplay := ball.Intent
				if ball.BlockedReason != "" {
					dimStyle := StyleDim
					intentDisplay = fmt.Sprintf("%s %s", ball.Intent, dimStyle.Render("("+ball.BlockedReason+")"))
				}

				// Add todo completion summary if todos exist
				todoSummary := ""
				if len(ball.Todos) > 0 {
					todoSummary = fmt.Sprintf(" %s", StyleDim.Render("["+ball.TodoCompletionSummary()+"]"))
				}

				fmt.Printf("  [%s] %s  %s  %s%s\n",
					idPadded,
					statePadded,
					priorityPadded,
					intentDisplay,
					todoSummary,
				)

				// Show description on next line if present
				if ball.Description != "" {
					descDisplay := ball.Description
					if len(descDisplay) > 80 {
						descDisplay = descDisplay[:77] + "..."
					}
					fmt.Printf("      %s\n", StyleDim.Render(descDisplay))
				}
			}
		}
	}

	fmt.Println()
	return nil
}

// handleBallCommand routes ball-specific commands
// findBallByID searches for a ball by ID across all discovered projects
// Returns the ball and a store configured for that ball's working directory
func findBallByID(ballID string) (*session.Session, *session.Store, error) {
	config, err := LoadConfigForCommand()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load config: %w", err)
	}

	projects, err := session.DiscoverProjects(config)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to discover projects: %w", err)
	}

	allBalls, err := session.LoadAllBalls(projects)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load balls: %w", err)
	}

	// Search for ball by full ID or short ID
	for _, ball := range allBalls {
		if ball.ID == ballID || ball.ShortID() == ballID {
			// Create store for this ball's working directory
			store, err := NewStoreForCommand(ball.WorkingDir)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to create store for ball: %w", err)
			}
			return ball, store, nil
		}
	}

	return nil, nil, fmt.Errorf("ball not found: %s", ballID)
}

func handleBallCommand(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("ball ID required")
	}

	ballID := args[0]

	// Special case: unarchive needs to look in archives, not active balls
	if len(args) > 1 && args[1] == "unarchive" {
		ball, store, err := findArchivedBallByID(ballID)
		if err != nil {
			return err
		}
		return handleBallUnarchive(ball, store)
	}

	// Find ball across all projects
	ball, store, err := findBallByID(ballID)
	if err != nil {
		return err
	}

	// If only ball ID provided, activate it
	if len(args) == 1 {
		return activateBall(ball, store)
	}

	// Handle ball operations
	operation := args[1]
	operationArgs := args[2:]

	switch operation {
	// New simplified state commands
	case "pending":
		return setBallState(ball, session.StatePending, operationArgs, store)
	case "in-progress":
		return setBallState(ball, session.StateInProgress, operationArgs, store)
	case "complete":
		return setBallComplete(ball, operationArgs, store)
	case "blocked":
		return setBallBlocked(ball, operationArgs, store)
	// Legacy active state commands (aliased to new states)
	case "ready":
		return setBallState(ball, session.StatePending, operationArgs, store)
	case "drop":
		return setBallBlocked(ball, operationArgs, store)
	case "todo", "todos":
		return handleBallTodo(ball, operationArgs, store)
	case "tag", "tags":
		return handleBallTag(ball, operationArgs, store)
	case "edit":
		return handleBallEdit(ball, operationArgs, store)
	case "update":
		return handleBallUpdate(ball, operationArgs, store)
	case "delete":
		return handleBallDelete(ball, operationArgs, store)
	default:
		return fmt.Errorf("unknown operation: %s", operation)
	}
}

// activateBall transitions a pending ball to in_progress
func activateBall(ball *session.Session, store *session.Store) error {
	// If ball is already in progress (or any non-pending state), show its details
	if ball.State != session.StatePending {
		renderSessionDetails(ball)
		return nil
	}

	ball.Start()

	if err := store.Save(ball); err != nil {
		return fmt.Errorf("failed to save ball: %w", err)
	}

	fmt.Printf("✓ Started ball: %s\n", ball.ID)
	fmt.Printf("  State: in_progress\n")
	fmt.Printf("  Intent: %s\n", ball.Intent)

	return nil
}

// setBallState sets the ball to a new state (pending, in_progress)
func setBallState(ball *session.Session, state session.BallState, args []string, store *session.Store) error {
	ball.SetState(state)

	if err := store.Save(ball); err != nil {
		return fmt.Errorf("failed to save ball: %w", err)
	}

	fmt.Printf("✓ Ball %s → %s\n", ball.ShortID(), state)
	return nil
}

// setBallComplete marks the ball as complete with optional note and archives it
func setBallComplete(ball *session.Session, args []string, store *session.Store) error {
	note := ""
	if len(args) > 0 {
		note = strings.Join(args, " ")
	}
	ball.MarkComplete(note)

	if err := store.Save(ball); err != nil {
		return fmt.Errorf("failed to save ball: %w", err)
	}

	fmt.Printf("✓ Ball %s → complete\n", ball.ShortID())
	if note != "" {
		fmt.Printf("  Note: %s\n", note)
	}

	// Archive completed ball
	if err := store.ArchiveBall(ball); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to archive ball: %v\n", err)
	}

	return nil
}

// setBallBlocked marks the ball as blocked with a reason
func setBallBlocked(ball *session.Session, args []string, store *session.Store) error {
	reason := ""
	if len(args) > 0 {
		reason = strings.Join(args, " ")
	}

	if reason == "" {
		return fmt.Errorf("blocked reason required: juggle <ball-id> blocked <reason>")
	}

	ball.SetBlocked(reason)

	if err := store.Save(ball); err != nil {
		return fmt.Errorf("failed to save ball: %w", err)
	}

	fmt.Printf("✓ Ball %s → blocked\n", ball.ShortID())
	fmt.Printf("  Reason: %s\n", reason)
	return nil
}

// handleBallTodo handles todo operations for a ball
func handleBallTodo(ball *session.Session, args []string, store *session.Store) error {
	if len(args) == 0 {
		// List todos
		return listBallTodos(ball)
	}

	subCmd := args[0]
	subArgs := args[1:]

	switch subCmd {
	case "add":
		return addBallTodos(ball, subArgs, store)
	case "done":
		return markBallTodoDone(ball, subArgs, store)
	case "rm", "remove":
		return removeBallTodo(ball, subArgs, store)
	default:
		return fmt.Errorf("unknown todo command: %s", subCmd)
	}
}

// listBallTodos lists todos for a ball
func listBallTodos(ball *session.Session) error {
	if len(ball.Todos) == 0 {
		fmt.Println("No todos")
		return nil
	}

	total, completed := ball.TodoStats()
	fmt.Printf("Todos: %d/%d complete (%d%%)\n\n", completed, total, (completed*100)/total)

	for i, todo := range ball.Todos {
		checkbox := "[ ]"
		if todo.Done {
			checkbox = "[✓]"
		}
		fmt.Printf("  %d. %s %s\n", i+1, checkbox, todo.Text)
	}

	return nil
}

// addBallTodos adds todos to a ball
func addBallTodos(ball *session.Session, tasks []string, store *session.Store) error {
	if len(tasks) == 0 {
		return fmt.Errorf("no tasks provided")
	}

	ball.AddTodos(tasks)
	
	if err := store.Save(ball); err != nil {
		return fmt.Errorf("failed to save ball: %w", err)
	}

	fmt.Printf("✓ Added %d todo%s to ball %s\n", len(tasks), pluralize(len(tasks)), ball.ShortID())
	return nil
}

// markBallTodoDone marks a todo as done
func markBallTodoDone(ball *session.Session, args []string, store *session.Store) error {
	if len(args) == 0 {
		return fmt.Errorf("todo index required")
	}

	var index int
	if _, err := fmt.Sscanf(args[0], "%d", &index); err != nil {
		return fmt.Errorf("invalid todo index: %s", args[0])
	}

	// Convert to 0-based
	index--

	if err := ball.ToggleTodo(index); err != nil {
		return err
	}

	if err := store.Save(ball); err != nil {
		return fmt.Errorf("failed to save ball: %w", err)
	}

	total, completed := ball.TodoStats()
	fmt.Printf("✓ Todo %d marked as done\n", index+1)
	fmt.Printf("Progress: %d/%d complete (%d%%)\n", completed, total, (completed*100)/total)

	return nil
}

// removeBallTodo removes a todo
func removeBallTodo(ball *session.Session, args []string, store *session.Store) error {
	if len(args) == 0 {
		return fmt.Errorf("todo index required")
	}

	var index int
	if _, err := fmt.Sscanf(args[0], "%d", &index); err != nil {
		return fmt.Errorf("invalid todo index: %s", args[0])
	}

	// Convert to 0-based
	index--

	if err := ball.RemoveTodo(index); err != nil {
		return err
	}

	if err := store.Save(ball); err != nil {
		return fmt.Errorf("failed to save ball: %w", err)
	}

	fmt.Printf("✓ Removed todo %d\n", index+1)
	return nil
}

// handleBallTag handles tag operations for a ball
func handleBallTag(ball *session.Session, args []string, store *session.Store) error {
	if len(args) == 0 {
		// List tags
		return listBallTags(ball)
	}

	subCmd := args[0]
	subArgs := args[1:]

	switch subCmd {
	case "add":
		return addBallTags(ball, subArgs, store)
	case "rm", "remove":
		return removeBallTag(ball, subArgs, store)
	default:
		return fmt.Errorf("unknown tag command: %s", subCmd)
	}
}

// listBallTags lists tags for a ball
func listBallTags(ball *session.Session) error {
	if len(ball.Tags) == 0 {
		fmt.Println("No tags")
		return nil
	}

	fmt.Printf("Tags (%d):\n\n", len(ball.Tags))
	for _, tag := range ball.Tags {
		fmt.Printf("  • %s\n", tag)
	}

	return nil
}

// addBallTags adds tags to a ball
func addBallTags(ball *session.Session, tags []string, store *session.Store) error {
	if len(tags) == 0 {
		return fmt.Errorf("no tags provided")
	}

	for _, tag := range tags {
		ball.AddTag(tag)
	}
	
	if err := store.Save(ball); err != nil {
		return fmt.Errorf("failed to save ball: %w", err)
	}

	fmt.Printf("✓ Added %d tag%s to ball %s\n", len(tags), pluralize(len(tags)), ball.ShortID())
	return nil
}

// removeBallTag removes a tag
func removeBallTag(ball *session.Session, args []string, store *session.Store) error {
	if len(args) == 0 {
		return fmt.Errorf("tag name required")
	}

	tag := args[0]
	
	if !ball.RemoveTag(tag) {
		return fmt.Errorf("tag not found: %s", tag)
	}

	if err := store.Save(ball); err != nil {
		return fmt.Errorf("failed to save ball: %w", err)
	}

	fmt.Printf("✓ Removed tag: %s\n", tag)
	return nil
}

// handleBallEdit handles editing ball properties
func handleBallEdit(ball *session.Session, args []string, store *session.Store) error {
	if len(args) == 0 {
		return fmt.Errorf("property required (intent, description, priority)")
	}

	property := args[0]
	
	switch property {
	case "intent":
		if len(args) < 2 {
			return fmt.Errorf("new intent text required")
		}
		newIntent := strings.Join(args[1:], " ")
		ball.Intent = newIntent
		
	case "description":
		if len(args) < 2 {
			return fmt.Errorf("new description text required")
		}
		newDescription := strings.Join(args[1:], " ")
		ball.SetDescription(newDescription)
		
	case "priority":
		if len(args) < 2 {
			return fmt.Errorf("priority value required (low, medium, high, urgent)")
		}
		if !session.ValidatePriority(args[1]) {
			return fmt.Errorf("invalid priority: %s (valid: low, medium, high, urgent)", args[1])
		}
		ball.Priority = session.Priority(args[1])
		
	default:
		return fmt.Errorf("unknown property: %s (valid: intent, description, priority)", property)
	}
	
	if err := store.Save(ball); err != nil {
		return fmt.Errorf("failed to save ball: %w", err)
	}

	fmt.Printf("✓ Updated %s for ball %s\n", property, ball.ShortID())
	return nil
}

// handleBallUpdate handles updating ball properties via juggle <ball-id> update ...
// This is a wrapper for the update command that works in the juggle <ball-id> update context
func handleBallUpdate(ball *session.Session, args []string, store *session.Store) error {
	// Parse flags from args
	modified := false
	i := 0
	for i < len(args) {
		arg := args[i]
		switch arg {
		case "--intent":
			if i+1 >= len(args) {
				return fmt.Errorf("--intent requires a value")
			}
			ball.Intent = args[i+1]
			modified = true
			fmt.Printf("✓ Updated intent: %s\n", ball.Intent)
			i += 2
		case "--priority":
			if i+1 >= len(args) {
				return fmt.Errorf("--priority requires a value")
			}
			if !session.ValidatePriority(args[i+1]) {
				return fmt.Errorf("invalid priority: %s (must be low|medium|high|urgent)", args[i+1])
			}
			ball.Priority = session.Priority(args[i+1])
			modified = true
			fmt.Printf("✓ Updated priority: %s\n", ball.Priority)
			i += 2
		case "--state":
			if i+1 >= len(args) {
				return fmt.Errorf("--state requires a value")
			}
			stateMap := map[string]session.BallState{
				"pending":     session.StatePending,
				"in_progress": session.StateInProgress,
				"blocked":     session.StateBlocked,
				"complete":    session.StateComplete,
			}
			newState, ok := stateMap[args[i+1]]
			if !ok {
				return fmt.Errorf("invalid state: %s (must be pending|in_progress|blocked|complete)", args[i+1])
			}
			// Check for --reason if setting to blocked
			if newState == session.StateBlocked {
				// Look for --reason in remaining args
				reason := ""
				for j := i + 2; j < len(args)-1; j++ {
					if args[j] == "--reason" {
						reason = args[j+1]
						break
					}
				}
				if reason == "" {
					return fmt.Errorf("blocked reason required: use --reason flag when setting state to blocked")
				}
				ball.SetBlocked(reason)
				fmt.Printf("✓ Updated state: blocked (reason: %s)\n", reason)
			} else {
				ball.SetState(newState)
				fmt.Printf("✓ Updated state: %s\n", ball.State)
			}
			modified = true
			i += 2
		case "--reason":
			// Skip - handled with --state blocked
			i += 2
		case "--criteria":
			if i+1 >= len(args) {
				return fmt.Errorf("--criteria requires a value")
			}
			// Collect all --criteria values
			var criteria []string
			for j := i; j < len(args)-1; j++ {
				if args[j] == "--criteria" {
					criteria = append(criteria, args[j+1])
				}
			}
			ball.SetAcceptanceCriteria(criteria)
			modified = true
			fmt.Printf("✓ Updated acceptance criteria (%d items)\n", len(criteria))
			// Skip all --criteria pairs we processed
			for j := i; j < len(args)-1; j += 2 {
				if args[j] != "--criteria" {
					break
				}
				i = j + 2
			}
		case "--tags":
			if i+1 >= len(args) {
				return fmt.Errorf("--tags requires a value")
			}
			tags := strings.Split(args[i+1], ",")
			for j := range tags {
				tags[j] = strings.TrimSpace(tags[j])
			}
			ball.Tags = tags
			modified = true
			fmt.Printf("✓ Updated tags: %s\n", strings.Join(tags, ", "))
			i += 2
		default:
			return fmt.Errorf("unknown flag: %s", arg)
		}
	}

	if !modified {
		fmt.Println("No updates specified. Use --intent, --priority, --state, --criteria, or --tags flags.")
		return nil
	}

	ball.UpdateActivity()
	if err := store.Save(ball); err != nil {
		return fmt.Errorf("failed to save ball: %w", err)
	}

	fmt.Printf("\n✓ Ball %s updated successfully\n", ball.ShortID())
	return nil
}

// handleBallDelete handles deleting a ball
func handleBallDelete(ball *session.Session, args []string, store *session.Store) error {
	// Check for --force flag
	force := false
	for _, arg := range args {
		if arg == "--force" || arg == "-f" {
			force = true
			break
		}
	}

	// Show ball information
	fmt.Printf("Ball to delete:\n")
	fmt.Printf("  ID: %s\n", ball.ID)
	fmt.Printf("  Intent: %s\n", ball.Intent)
	fmt.Printf("  Priority: %s\n", ball.Priority)
	fmt.Printf("  State: %s\n", ball.State)
	if len(ball.Todos) > 0 {
		fmt.Printf("  Todos: %d items\n", len(ball.Todos))
	}
	if len(ball.Tags) > 0 {
		fmt.Printf("  Tags: %s\n", strings.Join(ball.Tags, ", "))
	}
	fmt.Println()

	// Confirm deletion unless --force is used
	if !force {
		reader := bufio.NewReader(os.Stdin)
		fmt.Printf("Are you sure you want to delete this ball? This cannot be undone. [y/N]: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))

		if input != "y" && input != "yes" {
			fmt.Println("Deletion cancelled.")
			return nil
		}
	}

	// Delete the ball
	if err := store.DeleteBall(ball.ID); err != nil {
		return fmt.Errorf("failed to delete ball: %w", err)
	}

	fmt.Printf("✓ Ball %s deleted successfully\n", ball.ShortID())
	return nil
}

func pluralize(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}
