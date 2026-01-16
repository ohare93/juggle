package cli

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ohare93/juggle/internal/session"
	"github.com/ohare93/juggle/internal/tui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	"gopkg.in/yaml.v3"
)

var planCmd = &cobra.Command{
	Use:   "plan [intent...]",
	Short: "Add a planned ball for future work",
	Long: `Add a planned ball to track future work you intend to do.

By default, opens an interactive TUI form for ball creation with all fields.
Use --edit to open $EDITOR with a YAML template instead.

TUI mode (default):
  juggle plan                         # Opens TUI form
  juggle plan --intent "Fix bug"      # Pre-populates title in TUI
  juggle plan -c "AC1" -c "AC2"       # Pre-populates acceptance criteria

Editor mode:
  juggle plan --edit                  # Opens $EDITOR with YAML template
  juggle plan --edit --intent "Task"  # Pre-populates fields in template

Non-interactive mode (for headless agents):
  juggle plan "Task intent" --non-interactive              # Uses defaults
  juggle plan "Task" -p high -c "AC1" --non-interactive    # With options
  juggle plan "Task" --context "Background info" --non-interactive

In non-interactive mode:
  - Intent is required (via args or --intent flag)
  - Context provides background info for agents (highly recommended)
  - Priority defaults to 'medium' if not specified
  - State is always 'pending' (new balls start in pending state)
  - Tags, session, and acceptance criteria default to empty if not specified

Planned balls can be started later with: juggle <ball-id>`,
	RunE: runPlan,
}

var acceptanceCriteriaFlag []string
var criteriaAliasFlag []string // Alias for --ac
var dependsOnFlag []string
var contextFlag string
var nonInteractiveFlag bool
var editFlag bool

func init() {
	planCmd.Flags().StringVarP(&intentFlag, "intent", "i", "", "What are you planning to work on?")
	planCmd.Flags().StringVar(&contextFlag, "context", "", "Background context for the task (important for agents)")
	planCmd.Flags().StringArrayVarP(&acceptanceCriteriaFlag, "ac", "c", []string{}, "Acceptance criteria (can be specified multiple times)")
	planCmd.Flags().StringArrayVar(&criteriaAliasFlag, "criteria", []string{}, "Alias for --ac (acceptance criteria)")
	planCmd.Flags().StringVarP(&priorityFlag, "priority", "p", "", "Priority: low, medium, high, urgent (default: medium)")
	planCmd.Flags().StringSliceVarP(&tagsFlag, "tags", "t", []string{}, "Tags for categorization")
	planCmd.Flags().StringVarP(&sessionFlag, "session", "s", "", "Session ID to link this ball to (adds session ID as tag)")
	planCmd.Flags().StringVarP(&modelSizeFlag, "model-size", "m", "", "Preferred LLM model size: small, medium, large (blank for default)")
	planCmd.Flags().StringSliceVar(&dependsOnFlag, "depends-on", []string{}, "Ball IDs this ball depends on (can be specified multiple times)")
	planCmd.Flags().BoolVar(&nonInteractiveFlag, "non-interactive", false, "Skip interactive prompts, use defaults for unspecified fields (headless mode)")
	planCmd.Flags().BoolVar(&editFlag, "edit", false, "Open $EDITOR with YAML template instead of TUI form")
}

func runPlan(cmd *cobra.Command, args []string) error {
	cwd, err := GetWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	store, err := NewStoreForCommand(cwd)
	if err != nil {
		return fmt.Errorf("failed to initialize store: %w", err)
	}

	// Get intent from positional args or --intent flag
	intent := ""
	if len(args) > 0 {
		intent = strings.Join(args, " ")
	} else if intentFlag != "" {
		intent = intentFlag
	}

	// Build acceptance criteria list from flags (merge --ac and --criteria)
	acceptanceCriteria := append(acceptanceCriteriaFlag, criteriaAliasFlag...)

	// Determine which mode to use
	isTTY := term.IsTerminal(int(os.Stdin.Fd()))

	if nonInteractiveFlag {
		// Non-interactive mode: require intent, use defaults
		return runPlanNonInteractive(store, cwd, intent, acceptanceCriteria)
	}

	if editFlag {
		// Editor mode: open $EDITOR with YAML template
		return runPlanEditor(store, cwd, intent, acceptanceCriteria)
	}

	if !isTTY {
		// Not a TTY and not non-interactive: fall back to non-interactive
		if intent == "" {
			return fmt.Errorf("intent is required when not running in a terminal (use --intent or positional args)")
		}
		return runPlanNonInteractive(store, cwd, intent, acceptanceCriteria)
	}

	// Default: TUI mode
	return runPlanTUI(store, cwd, intent, acceptanceCriteria)
}

// runPlanTUI launches the TUI ball creation form
func runPlanTUI(store *session.Store, cwd, intent string, acceptanceCriteria []string) error {
	// Create session store for the TUI
	sessionStore, err := session.NewSessionStore(cwd)
	if err != nil {
		// Session store is optional, continue without it
		sessionStore = nil
	}

	// Create standalone ball model
	model := tui.NewStandaloneBallModel(store, sessionStore)

	// Pre-populate from flags
	model.PrePopulate(intent, contextFlag, tagsFlag, sessionFlag, priorityFlag, modelSizeFlag, acceptanceCriteria, dependsOnFlag)

	// Run the TUI
	p := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	// Get result from final model
	result := finalModel.(tui.StandaloneBallModel).Result()

	if result.Err != nil {
		return fmt.Errorf("failed to create ball: %w", result.Err)
	}

	if result.Cancelled {
		fmt.Println("Cancelled")
		return nil
	}

	// Ensure project is in search paths for discovery
	_ = session.EnsureProjectInSearchPaths(cwd)

	fmt.Printf("✓ Planned ball added: %s\n", result.Ball.ID)
	fmt.Printf("  Title: %s\n", result.Ball.Title)
	fmt.Printf("  Priority: %s\n", result.Ball.Priority)
	fmt.Printf("  State: %s\n", result.Ball.State)
	if len(result.Ball.Tags) > 0 {
		fmt.Printf("  Tags: %s\n", strings.Join(result.Ball.Tags, ", "))
	}
	if len(result.Ball.AcceptanceCriteria) > 0 {
		fmt.Printf("  Acceptance Criteria: %d\n", len(result.Ball.AcceptanceCriteria))
	}

	// Check if user requested to run agent after creation
	if result.RunAgentForBall != "" {
		fmt.Printf("\nStarting agent for ball %s...\n", result.RunAgentForBall)

		// Run the agent loop directly
		agentConfig := AgentLoopConfig{
			SessionID:     "all", // Use "all" meta-session since we're targeting a specific ball
			ProjectDir:    cwd,
			MaxIterations: 1,             // Single iteration for "run now"
			BallID:        result.RunAgentForBall,
			Interactive:   true,          // Interactive mode for user involvement
		}

		_, err := RunAgentLoop(agentConfig)
		if err != nil {
			return fmt.Errorf("agent error: %w", err)
		}
		return nil
	}

	fmt.Printf("\nStart working on this ball with: juggle %s in-progress\n", result.Ball.ID)

	return nil
}

// runPlanEditor opens $EDITOR with a YAML template for ball creation
func runPlanEditor(store *session.Store, cwd, intent string, acceptanceCriteria []string) error {
	// Create a template ball with defaults or flag values
	priority := priorityFlag
	if priority == "" {
		priority = "medium"
	}
	if !session.ValidatePriority(priority) {
		return fmt.Errorf("invalid priority %q, must be one of: low, medium, high, urgent", priority)
	}

	// Create YAML template
	yamlContent := createNewBallYAMLTemplate(intent, contextFlag, priority, tagsFlag, sessionFlag, modelSizeFlag, acceptanceCriteria)

	// Run the editor-based creation
	result, err := runEditorForNewBall(yamlContent)
	if err != nil {
		return err
	}

	if result.cancelled {
		fmt.Println("Cancelled - no changes made")
		return nil
	}

	// Parse the edited YAML and create the ball
	ball, err := parseNewBallYAML(result.content, cwd, store)
	if err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Resolve and set dependencies
	if len(ball.DependsOn) > 0 {
		resolvedDeps, err := resolveDependencyIDs(store, ball.DependsOn)
		if err != nil {
			return fmt.Errorf("failed to resolve dependencies: %w", err)
		}
		ball.SetDependencies(resolvedDeps)

		// Detect circular dependencies
		balls, err := store.LoadBalls()
		if err != nil {
			return fmt.Errorf("failed to load balls for dependency check: %w", err)
		}
		allBalls := append(balls, ball)
		if err := session.DetectCircularDependencies(allBalls); err != nil {
			return fmt.Errorf("dependency error: %w", err)
		}
	}

	// Save the ball
	if err := store.AppendBall(ball); err != nil {
		return fmt.Errorf("failed to save planned ball: %w", err)
	}

	// Ensure project is in search paths for discovery
	_ = session.EnsureProjectInSearchPaths(cwd)

	fmt.Printf("✓ Planned ball added: %s\n", ball.ID)
	fmt.Printf("  Title: %s\n", ball.Title)
	fmt.Printf("  Priority: %s\n", ball.Priority)
	fmt.Printf("  State: %s\n", ball.State)
	if len(ball.Tags) > 0 {
		fmt.Printf("  Tags: %s\n", strings.Join(ball.Tags, ", "))
	}
	if len(ball.AcceptanceCriteria) > 0 {
		fmt.Printf("  Acceptance Criteria: %d\n", len(ball.AcceptanceCriteria))
	}
	fmt.Printf("\nStart working on this ball with: juggle %s in-progress\n", ball.ID)

	return nil
}

// runPlanNonInteractive creates a ball without any interactive prompts
func runPlanNonInteractive(store *session.Store, cwd, intent string, acceptanceCriteria []string) error {
	if intent == "" {
		return fmt.Errorf("intent is required in non-interactive mode (use positional args or --intent)")
	}

	// Get priority from flag or default
	priority := priorityFlag
	if priority == "" {
		priority = "medium"
	}
	if !session.ValidatePriority(priority) {
		return fmt.Errorf("invalid priority %q, must be one of: low, medium, high, urgent", priority)
	}

	// Create the planned ball
	ball, err := session.NewBall(cwd, intent, session.Priority(priority))
	if err != nil {
		return fmt.Errorf("failed to create planned ball: %w", err)
	}

	ball.State = session.StatePending

	// Set context if provided
	if contextFlag != "" {
		ball.Context = contextFlag
	}

	// Set acceptance criteria if provided
	if len(acceptanceCriteria) > 0 {
		ball.SetAcceptanceCriteria(acceptanceCriteria)
	}

	// Add tags if provided
	for _, tag := range tagsFlag {
		ball.AddTag(tag)
	}

	// Add session ID as tag if provided
	if sessionFlag != "" {
		ball.AddTag(sessionFlag)
	}

	// Set model size if provided
	if modelSizeFlag != "" {
		ms := session.ModelSize(modelSizeFlag)
		if ms != session.ModelSizeSmall && ms != session.ModelSizeMedium && ms != session.ModelSizeLarge {
			return fmt.Errorf("invalid model size %q, must be one of: small, medium, large", modelSizeFlag)
		}
		ball.ModelSize = ms
	}

	// Set dependencies if provided
	if len(dependsOnFlag) > 0 {
		resolvedDeps, err := resolveDependencyIDs(store, dependsOnFlag)
		if err != nil {
			return fmt.Errorf("failed to resolve dependencies: %w", err)
		}
		ball.SetDependencies(resolvedDeps)

		balls, err := store.LoadBalls()
		if err != nil {
			return fmt.Errorf("failed to load balls for dependency check: %w", err)
		}
		allBalls := append(balls, ball)
		if err := session.DetectCircularDependencies(allBalls); err != nil {
			return fmt.Errorf("dependency error: %w", err)
		}
	}

	// Save the ball
	if err := store.AppendBall(ball); err != nil {
		return fmt.Errorf("failed to save planned ball: %w", err)
	}

	// Ensure project is in search paths for discovery
	_ = session.EnsureProjectInSearchPaths(cwd)

	fmt.Printf("✓ Planned ball added: %s\n", ball.ID)
	fmt.Printf("  Title: %s\n", ball.Title)
	fmt.Printf("  Priority: %s\n", ball.Priority)
	fmt.Printf("  State: %s\n", ball.State)
	if len(ball.Tags) > 0 {
		fmt.Printf("  Tags: %s\n", strings.Join(ball.Tags, ", "))
	}
	if ball.State == session.StatePending {
		fmt.Printf("\nStart working on this ball with: juggle %s in-progress\n", ball.ID)
	}

	return nil
}

// createNewBallYAMLTemplate creates a YAML template for new ball creation
func createNewBallYAMLTemplate(intent, context, priority string, tags []string, sessionID, modelSize string, acceptanceCriteria []string) string {
	// Add session ID to tags if provided
	allTags := tags
	if sessionID != "" {
		allTags = append(allTags, sessionID)
	}

	// Format tags for YAML
	tagsYAML := "[]"
	if len(allTags) > 0 {
		var tagLines []string
		for _, tag := range allTags {
			tagLines = append(tagLines, fmt.Sprintf("  - %s", tag))
		}
		tagsYAML = "\n" + strings.Join(tagLines, "\n")
	}

	// Format acceptance criteria for YAML
	acYAML := "[]"
	if len(acceptanceCriteria) > 0 {
		var acLines []string
		for _, ac := range acceptanceCriteria {
			acLines = append(acLines, fmt.Sprintf("  - %s", ac))
		}
		acYAML = "\n" + strings.Join(acLines, "\n")
	}

	return fmt.Sprintf(`# Create New Ball
# Edit the fields below and save to create the ball
# Close without saving to cancel
#
# Required: title
# Optional: context, priority, tags, acceptance_criteria, model_size, depends_on

# Brief title describing what this ball is about (50 chars recommended)
title: %s

# Background context for this task (important for agents to understand the task)
context: %q

# Priority: low, medium, high, urgent
priority: %s

# Tags for categorization (include session ID to link to a session)
tags: %s

# Acceptance criteria for completion
acceptance_criteria: %s

# Preferred LLM model size: small, medium, large (or empty for default)
model_size: %s

# Ball IDs this ball depends on (must complete before this one)
depends_on: []
`, intent, context, priority, tagsYAML, acYAML, modelSize)
}

// editorResult holds the result of running the editor
type editorResult struct {
	content   string
	cancelled bool
}

// runEditorForNewBall opens $EDITOR for creating a new ball
func runEditorForNewBall(yamlContent string) (editorResult, error) {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	// Create temp file
	tmpFile, err := os.CreateTemp("", "juggle-new-ball-*.yaml")
	if err != nil {
		return editorResult{}, fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	// Write YAML to temp file
	if _, err := tmpFile.WriteString(yamlContent); err != nil {
		tmpFile.Close()
		return editorResult{}, fmt.Errorf("failed to write temp file: %w", err)
	}
	tmpFile.Close()

	// Store original content
	originalContent := yamlContent

	// Run editor
	editorParts := strings.Fields(editor)
	var cmd *exec.Cmd
	if len(editorParts) > 1 {
		cmd = exec.Command(editorParts[0], append(editorParts[1:], tmpPath)...)
	} else {
		cmd = exec.Command(editor, tmpPath)
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return editorResult{}, fmt.Errorf("editor failed: %w", err)
	}

	// Read edited content
	editedContent, err := os.ReadFile(tmpPath)
	if err != nil {
		return editorResult{}, fmt.Errorf("failed to read edited file: %w", err)
	}

	// Check if content was modified
	if string(editedContent) == originalContent {
		return editorResult{cancelled: true}, nil
	}

	return editorResult{content: string(editedContent)}, nil
}

// NewBallYAML is the YAML representation for new ball creation
type NewBallYAML struct {
	Title              string   `yaml:"title"`
	Context            string   `yaml:"context"`
	Priority           string   `yaml:"priority"`
	Tags               []string `yaml:"tags"`
	AcceptanceCriteria []string `yaml:"acceptance_criteria"`
	ModelSize          string   `yaml:"model_size"`
	DependsOn          []string `yaml:"depends_on"`
}

// parseNewBallYAML parses edited YAML and creates a new ball
func parseNewBallYAML(yamlContent, cwd string, store *session.Store) (*session.Ball, error) {
	var yamlBall NewBallYAML
	if err := yaml.Unmarshal([]byte(yamlContent), &yamlBall); err != nil {
		return nil, fmt.Errorf("invalid YAML: %w", err)
	}

	// Validate required fields
	title := strings.TrimSpace(yamlBall.Title)
	if title == "" {
		return nil, fmt.Errorf("title is required")
	}

	// Validate priority
	priority := strings.TrimSpace(yamlBall.Priority)
	if priority == "" {
		priority = "medium"
	}
	if !session.ValidatePriority(priority) {
		return nil, fmt.Errorf("invalid priority: %s (must be low, medium, high, or urgent)", priority)
	}

	// Create the ball
	ball, err := session.NewBall(cwd, title, session.Priority(priority))
	if err != nil {
		return nil, err
	}

	ball.State = session.StatePending
	ball.Context = strings.TrimSpace(yamlBall.Context)

	// Set tags
	var cleanTags []string
	for _, tag := range yamlBall.Tags {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			cleanTags = append(cleanTags, tag)
		}
	}
	ball.Tags = cleanTags

	// Set acceptance criteria
	var cleanAC []string
	for _, ac := range yamlBall.AcceptanceCriteria {
		ac = strings.TrimSpace(ac)
		if ac != "" {
			cleanAC = append(cleanAC, ac)
		}
	}
	if len(cleanAC) > 0 {
		ball.SetAcceptanceCriteria(cleanAC)
	}

	// Set model size
	modelSize := strings.TrimSpace(yamlBall.ModelSize)
	if modelSize != "" {
		ms := session.ModelSize(modelSize)
		switch ms {
		case session.ModelSizeSmall, session.ModelSizeMedium, session.ModelSizeLarge:
			ball.ModelSize = ms
		default:
			return nil, fmt.Errorf("invalid model_size: %s (must be small, medium, large, or empty)", modelSize)
		}
	}

	// Store depends_on for later resolution (not resolved here to avoid circular import)
	ball.DependsOn = yamlBall.DependsOn

	return ball, nil
}

// promptSelection prompts user to select from options using numbers
func promptSelection(reader *bufio.Reader, label string, options []string, defaultIdx int) string {
	fmt.Printf("%s:\n", label)
	for i, opt := range options {
		marker := " "
		if i == defaultIdx {
			marker = "*"
		}
		fmt.Printf("  %s %d. %s\n", marker, i+1, opt)
	}
	fmt.Printf("Enter number (default %d): ", defaultIdx+1)

	input, err := reader.ReadString('\n')
	if err != nil || strings.TrimSpace(input) == "" {
		return options[defaultIdx]
	}

	var idx int
	_, err = fmt.Sscanf(strings.TrimSpace(input), "%d", &idx)
	if err != nil || idx < 1 || idx > len(options) {
		return options[defaultIdx]
	}

	return options[idx-1]
}

// resolveDependencyIDs resolves ball IDs (full or short) to full ball IDs
func resolveDependencyIDs(store *session.Store, ids []string) ([]string, error) {
	balls, err := store.LoadBalls()
	if err != nil {
		return nil, err
	}

	resolved := make([]string, 0, len(ids))
	for _, id := range ids {
		// Use prefix matching
		matches := session.ResolveBallByPrefix(balls, id)
		if len(matches) == 0 {
			return nil, fmt.Errorf("ball not found: %s", id)
		}
		if len(matches) > 1 {
			matchingIDs := make([]string, len(matches))
			for i, m := range matches {
				matchingIDs[i] = m.ID
			}
			return nil, fmt.Errorf("ambiguous ID '%s' matches %d balls: %s", id, len(matches), strings.Join(matchingIDs, ", "))
		}
		resolved = append(resolved, matches[0].ID)
	}
	return resolved, nil
}
