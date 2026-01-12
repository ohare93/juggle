package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ohare93/juggle/internal/agent"
	"github.com/ohare93/juggle/internal/session"
	"github.com/spf13/cobra"
)

var (
	agentIterations  int
	agentTrust       bool
	agentTimeout     time.Duration
	agentDebug       bool
	agentDryRun      bool
	agentMaxWait     time.Duration
	agentBallID      string
	agentInteractive bool
)

// agentCmd is the parent command for agent operations
var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Run AI agents on sessions",
	Long:  `Run AI agents to work on session balls autonomously.`,
}

// agentRunCmd runs the agent loop
var agentRunCmd = &cobra.Command{
	Use:   "run <session-id>",
	Short: "Run agent loop for a session",
	Long: `Run an AI agent loop for a session.

The agent will:
1. Generate a prompt using 'juggle export --format agent'
2. Spawn claude with the prepared prompt
3. Capture output to .juggler/sessions/<id>/last_output.txt
4. Check for COMPLETE/BLOCKED signals after each iteration
5. Repeat until done or max iterations reached

Rate Limit Handling:
When Claude returns a rate limit error (429 or overloaded), the agent
automatically waits with exponential backoff before retrying. If Claude
specifies a retry-after time, that time is used instead.

Examples:
  # Run agent for 10 iterations (default)
  juggle agent run my-feature

  # Run for specific number of iterations
  juggle agent run my-feature --iterations 5

  # Work on a specific ball only (1 iteration, interactive mode)
  juggle agent run my-feature --ball juggler-5

  # Work on specific ball with multiple iterations (non-interactive)
  juggle agent run my-feature --ball juggler-5 -n 3

  # Run in interactive mode (full Claude TUI)
  juggle agent run my-feature --interactive

  # Run with full permissions (dangerous)
  juggle agent run my-feature --trust

  # Run with 5 minute timeout per iteration
  juggle agent run my-feature --timeout 5m

  # Set maximum wait time for rate limits (give up if exceeded)
  juggle agent run my-feature --max-wait 30m

  # Show prompt info without running (dry run)
  juggle agent run my-feature --dry-run

  # Show prompt info before running (debug mode)
  juggle agent run my-feature --debug`,
	Args: cobra.ExactArgs(1),
	RunE: runAgentRun,
}

// agentRefineCmd launches an interactive session to review and improve ball definitions
var agentRefineCmd = &cobra.Command{
	Use:     "refine [session]",
	Aliases: []string{"refinement"},
	Short:   "Review and improve ball definitions interactively",
	Long: `Launch an interactive Claude session in plan mode to review and improve balls.

This command helps you:
- Improve acceptance criteria to be specific and testable
- Identify overlapping or duplicate work items
- Adjust priorities based on impact and dependencies
- Clarify vague intents for better autonomous execution

Ball Selection:
- No argument: Review all balls in current repo
- Session arg: Review balls with that session tag
- --all flag: Review balls from all discovered projects

Examples:
  # Review balls in current repo
  juggle agent refine

  # Review balls for a specific session
  juggle agent refine my-feature

  # Review all balls across all projects
  juggle agent refine --all`,
	Args: cobra.MaximumNArgs(1),
	RunE: runAgentRefine,
}

func init() {
	agentRunCmd.Flags().IntVarP(&agentIterations, "iterations", "n", 10, "Maximum number of iterations")
	agentRunCmd.Flags().BoolVar(&agentTrust, "trust", false, "Run with --dangerously-skip-permissions (dangerous!)")
	agentRunCmd.Flags().DurationVarP(&agentTimeout, "timeout", "T", 0, "Timeout per iteration (e.g., 5m, 1h). 0 = no timeout")
	agentRunCmd.Flags().BoolVarP(&agentDebug, "debug", "d", false, "Show prompt info before running the agent")
	agentRunCmd.Flags().BoolVar(&agentDryRun, "dry-run", false, "Show prompt info without running the agent")
	agentRunCmd.Flags().DurationVar(&agentMaxWait, "max-wait", 0, "Maximum wait time for rate limits before giving up (e.g., 30m). 0 = wait indefinitely")
	agentRunCmd.Flags().StringVarP(&agentBallID, "ball", "b", "", "Work on a specific ball only (defaults to 1 iteration, interactive)")
	agentRunCmd.Flags().BoolVarP(&agentInteractive, "interactive", "i", false, "Run in interactive mode (full Claude TUI, defaults to 1 iteration)")

	agentCmd.AddCommand(agentRunCmd)
	agentCmd.AddCommand(agentRefineCmd)
	rootCmd.AddCommand(agentCmd)
}

// AgentResult holds the result of an agent run
type AgentResult struct {
	Iterations       int           `json:"iterations"`
	Complete         bool          `json:"complete"`
	Blocked          bool          `json:"blocked"`
	BlockedReason    string        `json:"blocked_reason,omitempty"`
	TimedOut         bool          `json:"timed_out"`
	TimeoutMessage   string        `json:"timeout_message,omitempty"`
	RateLimitExceded bool          `json:"rate_limit_exceeded"`
	TotalWaitTime    time.Duration `json:"total_wait_time,omitempty"`
	BallsComplete    int           `json:"balls_complete"`
	BallsBlocked     int           `json:"balls_blocked"`
	BallsTotal       int           `json:"balls_total"`
	StartedAt        time.Time     `json:"started_at"`
	EndedAt          time.Time     `json:"ended_at"`
}

// AgentLoopConfig configures the agent loop behavior
type AgentLoopConfig struct {
	SessionID     string
	ProjectDir    string
	MaxIterations int
	Trust         bool
	Debug         bool          // Add debug reasoning instructions to prompt
	IterDelay     time.Duration // Delay between iterations (set to 0 for tests)
	Timeout       time.Duration // Timeout per iteration (0 = no timeout)
	MaxWait       time.Duration // Maximum time to wait for rate limits (0 = wait indefinitely)
	BallID        string        // Specific ball to work on (empty = all session balls)
	Interactive   bool          // Run in interactive mode (full Claude TUI)
}

// RunAgentLoop executes the agent loop with the given configuration.
// This is the testable core of the agent run command.
func RunAgentLoop(config AgentLoopConfig) (*AgentResult, error) {
	startTime := time.Now()

	// Verify session exists
	sessionStore, err := session.NewSessionStore(config.ProjectDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create session store: %w", err)
	}

	if _, err := sessionStore.LoadSession(config.SessionID); err != nil {
		return nil, fmt.Errorf("session not found: %s", config.SessionID)
	}

	// Acquire exclusive lock on session to prevent concurrent agent runs
	lock, err := sessionStore.AcquireSessionLock(config.SessionID)
	if err != nil {
		return nil, err
	}
	defer lock.Release()

	// Create output file path
	outputPath := filepath.Join(config.ProjectDir, ".juggler", "sessions", config.SessionID, "last_output.txt")

	result := &AgentResult{
		StartedAt: startTime,
	}

	// Track rate limit state
	var totalWaitTime time.Duration
	rateLimitRetries := 0
	rateLimitRetrying := false // Skip header when retrying after rate limit

	for iteration := 1; iteration <= config.MaxIterations; iteration++ {
		result.Iterations = iteration

		// Print iteration separator and header (skip when retrying after rate limit)
		if !rateLimitRetrying {
			if iteration > 1 {
				fmt.Println()
				fmt.Println()
				fmt.Println("════════════════════════════════════════════════════════════════════════════════")
				fmt.Println()
				fmt.Println()
			}
			fmt.Printf("════════════════════════════════ Iteration %d/%d ════════════════════════════════\n\n", iteration, config.MaxIterations)
		}
		rateLimitRetrying = false // Reset for next iteration

		// Record progress state before iteration (for validation)
		progressBefore := getProgressLineCount(sessionStore, config.SessionID)

		// Generate prompt using export command
		prompt, err := generateAgentPrompt(config.ProjectDir, config.SessionID, config.Debug, config.BallID)
		if err != nil {
			return nil, fmt.Errorf("failed to generate prompt: %w", err)
		}

		// Build run options
		opts := agent.RunOptions{
			Prompt:     prompt,
			Mode:       agent.ModeHeadless,
			Permission: agent.PermissionAcceptEdits,
			Timeout:    config.Timeout,
		}
		if config.Interactive {
			opts.Mode = agent.ModeInteractive
		}
		if config.Trust {
			opts.Permission = agent.PermissionBypass
		}
		// Add autonomous system prompt for headless mode
		if !config.Interactive {
			opts.SystemPrompt = agent.AutonomousSystemPrompt
		}

		// Run agent with options using the Runner interface
		runResult, err := agent.DefaultRunner.Run(opts)
		if err != nil {
			return nil, fmt.Errorf("failed to run agent: %w", err)
		}

		// Check for rate limit
		if runResult.RateLimited {
			waitTime := calculateWaitTime(runResult.RetryAfter, rateLimitRetries)

			// Check if we've exceeded max wait
			if config.MaxWait > 0 && totalWaitTime+waitTime > config.MaxWait {
				result.RateLimitExceded = true
				result.TotalWaitTime = totalWaitTime
				logRateLimitToProgress(config.ProjectDir, config.SessionID,
					fmt.Sprintf("Rate limit exceeded max-wait of %v (total waited: %v)", config.MaxWait, totalWaitTime))
				break
			}

			// Log waiting status
			logRateLimitToProgress(config.ProjectDir, config.SessionID,
				fmt.Sprintf("Rate limited, waiting %v before retry (attempt %d)", waitTime, rateLimitRetries+1))

			fmt.Printf("⏳ Rate limited. Waiting %v before retry...\n", waitTime)

			// Wait with countdown display
			waitWithCountdown(waitTime)

			totalWaitTime += waitTime
			rateLimitRetries++
			rateLimitRetrying = true // Skip header on retry

			// Retry this iteration (don't increment)
			iteration--
			continue
		}

		// Reset retry counter on successful run
		rateLimitRetries = 0

		// Check for timeout
		if runResult.TimedOut {
			result.TimedOut = true
			result.TimeoutMessage = fmt.Sprintf("Iteration %d timed out after %v", iteration, config.Timeout)
			// Log timeout to progress
			logTimeoutToProgress(config.ProjectDir, config.SessionID, result.TimeoutMessage)
			break
		}

		// Save output to file (ignore errors for test compatibility)
		_ = os.WriteFile(outputPath, []byte(runResult.Output), 0644)

		// Check for completion signals (already parsed by Runner)
		if runResult.Complete {
			// VALIDATE: Check if progress was updated this iteration
			progressAfter := getProgressLineCount(sessionStore, config.SessionID)
			if progressAfter <= progressBefore {
				fmt.Println()
				fmt.Printf("⚠️  Agent signaled COMPLETE but did not update progress. Continuing iteration...\n")
				// Don't accept the signal - continue to check terminal state
			} else {
				// VALIDATE: Check if all balls are actually in terminal state (complete or blocked)
				terminal, complete, blocked, total := checkBallsTerminal(config.ProjectDir, config.SessionID, config.BallID)
				if total > 0 && terminal == total {
					result.Complete = true
					result.BallsComplete = complete
					result.BallsBlocked = blocked
					result.BallsTotal = total
					break
				}
				// Signal was premature - log warning and continue
				fmt.Println()
				fmt.Printf("⚠️  Agent signaled COMPLETE but only %d/%d balls are in terminal state (%d complete, %d blocked). Continuing...\n",
					terminal, total, complete, blocked)
			}
		}

		if runResult.Continue {
			// VALIDATE: Check if progress was updated this iteration
			progressAfter := getProgressLineCount(sessionStore, config.SessionID)
			if progressAfter <= progressBefore {
				fmt.Println()
				fmt.Printf("⚠️  Agent signaled CONTINUE but did not update progress. Continuing iteration...\n")
				// Don't accept the signal - fall through to terminal state check
			} else {
				// Agent completed one ball, more remain - continue to next iteration
				fmt.Println()
				fmt.Printf("✓ Agent completed a ball, continuing to next iteration...\n")

				// Update ball counts for progress tracking
				_, complete, blocked, total := checkBallsTerminal(config.ProjectDir, config.SessionID, config.BallID)
				result.BallsComplete = complete
				result.BallsBlocked = blocked
				result.BallsTotal = total

				continue
			}
		}

		if runResult.Blocked {
			// VALIDATE: Check if progress was updated this iteration
			progressAfter := getProgressLineCount(sessionStore, config.SessionID)
			if progressAfter <= progressBefore {
				fmt.Println()
				fmt.Printf("⚠️  Agent signaled BLOCKED but did not update progress. Continuing iteration...\n")
				// Don't accept the signal - fall through to terminal state check
			} else {
				result.Blocked = true
				result.BlockedReason = runResult.BlockedReason
				break
			}
		}

		// Check if all balls are in terminal state (complete or blocked)
		terminal, complete, blocked, total := checkBallsTerminal(config.ProjectDir, config.SessionID, config.BallID)
		result.BallsComplete = complete
		result.BallsBlocked = blocked
		result.BallsTotal = total

		if total > 0 && terminal == total {
			result.Complete = true
			break
		}

		// Delay before next iteration (unless this was the last one)
		if iteration < config.MaxIterations && config.IterDelay > 0 {
			time.Sleep(config.IterDelay)
		}
	}

	result.TotalWaitTime = totalWaitTime
	result.EndedAt = time.Now()
	return result, nil
}

// calculateWaitTime determines how long to wait before retrying after rate limit
// Uses the explicit retry-after time if provided, otherwise exponential backoff
func calculateWaitTime(retryAfter time.Duration, retryCount int) time.Duration {
	if retryAfter > 0 {
		// Use the time specified by Claude, with a small buffer
		return retryAfter + 5*time.Second
	}

	// Exponential backoff: 30s, 1m, 2m, 4m, 8m, 16m (capped at 16m)
	baseWait := 30 * time.Second
	maxWait := 16 * time.Minute

	wait := baseWait * time.Duration(1<<retryCount) // 2^retryCount
	if wait > maxWait {
		wait = maxWait
	}

	return wait
}

// waitWithCountdown waits for the specified duration, showing periodic countdown updates
func waitWithCountdown(duration time.Duration) {
	remaining := duration
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for remaining > 0 {
		select {
		case <-ticker.C:
			remaining -= 10 * time.Second
			if remaining > 0 {
				fmt.Printf("  ... %v remaining\n", remaining.Round(time.Second))
			}
		case <-time.After(remaining):
			return
		}
	}
}

// logRateLimitToProgress logs a rate limit event to the session's progress file
func logRateLimitToProgress(projectDir, sessionID, message string) {
	sessionStore, err := session.NewSessionStore(projectDir)
	if err != nil {
		return // Ignore errors - logging is best-effort
	}

	entry := fmt.Sprintf("[RATE_LIMIT] %s", message)
	_ = sessionStore.AppendProgress(sessionID, entry)
}

func runAgentRun(cmd *cobra.Command, args []string) error {
	sessionID := args[0]

	// Get current directory
	cwd, err := GetWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Determine iterations and interactive mode
	// Default to 1 iteration when --ball or --interactive is specified (unless -n was explicitly set)
	iterations := agentIterations
	interactive := agentInteractive
	if (agentBallID != "" || agentInteractive) && !cmd.Flags().Changed("iterations") {
		iterations = 1
	}
	// --ball implies interactive mode (unless -n was explicitly set for multiple iterations)
	if agentBallID != "" && !cmd.Flags().Changed("iterations") {
		interactive = true
	}

	// Handle --dry-run and --debug: show prompt info
	if agentDryRun || agentDebug {
		prompt, err := generateAgentPrompt(cwd, sessionID, true, agentBallID) // debug=true for reasoning instructions
		if err != nil {
			return fmt.Errorf("failed to generate prompt: %w", err)
		}

		fmt.Println("=== Agent Prompt Info ===")
		fmt.Println()
		fmt.Printf("Session: %s\n", sessionID)
		if agentBallID != "" {
			fmt.Printf("Ball: %s\n", agentBallID)
		}
		fmt.Printf("Max iterations: %d\n", iterations)
		fmt.Printf("Trust mode: %v\n", agentTrust)
		fmt.Printf("Interactive mode: %v\n", interactive)
		if agentTimeout > 0 {
			fmt.Printf("Timeout per iteration: %v\n", agentTimeout)
		}
		if agentMaxWait > 0 {
			fmt.Printf("Max rate limit wait: %v\n", agentMaxWait)
		}
		fmt.Println()
		fmt.Println("=== Generated Prompt ===")
		fmt.Println()
		fmt.Println(prompt)
		fmt.Println()
		fmt.Printf("=== Prompt Length: %d characters ===\n", len(prompt))

		// If dry-run, exit without running
		if agentDryRun {
			fmt.Println()
			fmt.Println("(Dry run - agent not started)")
			return nil
		}

		// If debug, continue to run the agent
		fmt.Println()
		fmt.Println("=== Starting Agent ===")
		fmt.Println()
	}

	// Print warning if --trust is used
	if agentTrust {
		fmt.Println("⚠️  WARNING: Running with --trust flag. Agent has full system permissions.")
		fmt.Println("    Only use this if you trust the agent and understand the risks.")
		fmt.Println()
	}

	if agentBallID != "" {
		fmt.Printf("Starting agent for ball: %s (session: %s)\n", agentBallID, sessionID)
	} else {
		fmt.Printf("Starting agent for session: %s\n", sessionID)
	}
	fmt.Printf("Max iterations: %d\n", iterations)
	fmt.Println()

	// Print timeout if specified
	if agentTimeout > 0 {
		fmt.Printf("Timeout per iteration: %v\n", agentTimeout)
	}

	// Run the agent loop
	loopConfig := AgentLoopConfig{
		SessionID:     sessionID,
		ProjectDir:    cwd,
		MaxIterations: iterations,
		Trust:         agentTrust,
		Debug:         false, // Debug mode now just shows prompt info, doesn't affect prompt content
		IterDelay:     2 * time.Second,
		Timeout:       agentTimeout,
		MaxWait:       agentMaxWait,
		BallID:        agentBallID,
		Interactive:   interactive,
	}

	result, err := RunAgentLoop(loopConfig)
	if err != nil {
		return err
	}

	elapsed := result.EndedAt.Sub(result.StartedAt)

	// Print summary
	fmt.Println()
	fmt.Println("=== Summary ===")
	fmt.Printf("Iterations: %d\n", result.Iterations)
	fmt.Printf("Balls: %d complete, %d blocked, %d total\n", result.BallsComplete, result.BallsBlocked, result.BallsTotal)
	fmt.Printf("Time elapsed: %s\n", elapsed.Round(time.Second))

	if result.TotalWaitTime > 0 {
		fmt.Printf("Rate limit wait time: %v\n", result.TotalWaitTime.Round(time.Second))
	}

	if result.Complete {
		fmt.Println("Status: COMPLETE")
	} else if result.Blocked {
		fmt.Printf("Status: BLOCKED (%s)\n", result.BlockedReason)
	} else if result.TimedOut {
		fmt.Printf("Status: TIMEOUT (%s)\n", result.TimeoutMessage)
	} else if result.RateLimitExceded {
		fmt.Printf("Status: RATE_LIMIT_EXCEEDED (max-wait: %v)\n", agentMaxWait)
	} else {
		fmt.Println("Status: Max iterations reached")
	}

	outputPath := filepath.Join(cwd, ".juggler", "sessions", sessionID, "last_output.txt")
	fmt.Printf("\nOutput saved to: %s\n", outputPath)

	return nil
}

// generateAgentPrompt generates the agent prompt using export command
func generateAgentPrompt(projectDir, sessionID string, debug bool, ballID string) (string, error) {
	// Use the export functionality directly instead of shelling out
	// This is more efficient and avoids subprocess overhead

	// Load config to discover projects
	config, err := LoadConfigForCommand()
	if err != nil {
		return "", fmt.Errorf("failed to load config: %w", err)
	}

	// Create store for current directory
	store, err := NewStoreForCommand(projectDir)
	if err != nil {
		return "", fmt.Errorf("failed to create store: %w", err)
	}

	// Discover projects
	projects, err := DiscoverProjectsForCommand(config, store)
	if err != nil {
		return "", fmt.Errorf("failed to discover projects: %w", err)
	}

	if len(projects) == 0 {
		return "", fmt.Errorf("no projects with .juggler directories found")
	}

	// Load all balls from discovered projects
	allBalls, err := session.LoadAllBalls(projects)
	if err != nil {
		return "", fmt.Errorf("failed to load balls: %w", err)
	}

	// Filter by session tag
	balls := make([]*session.Ball, 0)
	for _, ball := range allBalls {
		for _, tag := range ball.Tags {
			if tag == sessionID {
				balls = append(balls, ball)
				break
			}
		}
	}

	// Filter to specific ball if ballID is specified
	singleBall := false
	if ballID != "" {
		var targetBall *session.Ball
		for _, ball := range balls {
			if ball.ID == ballID || ball.ShortID() == ballID {
				targetBall = ball
				break
			}
		}
		if targetBall == nil {
			return "", fmt.Errorf("ball %s not found in session %s", ballID, sessionID)
		}
		balls = []*session.Ball{targetBall}
		singleBall = true
	}

	// Call exportAgent directly
	output, err := exportAgent(projectDir, sessionID, balls, debug, singleBall)
	if err != nil {
		return "", err
	}

	return string(output), nil
}

// checkBallsTerminal returns counts of balls in terminal states (complete or blocked) and total balls for session
// If ballID is specified, only counts that specific ball
func checkBallsTerminal(projectDir, sessionID, ballID string) (terminal, complete, blocked, total int) {
	// Load config
	config, err := LoadConfigForCommand()
	if err != nil {
		return 0, 0, 0, 0
	}

	// Create store
	store, err := NewStoreForCommand(projectDir)
	if err != nil {
		return 0, 0, 0, 0
	}

	// Discover projects
	projects, err := DiscoverProjectsForCommand(config, store)
	if err != nil {
		return 0, 0, 0, 0
	}

	// Load all balls
	allBalls, err := session.LoadAllBalls(projects)
	if err != nil {
		return 0, 0, 0, 0
	}

	// Count balls with session tag
	for _, ball := range allBalls {
		for _, tag := range ball.Tags {
			if tag == sessionID {
				// If filtering by specific ball, skip others
				if ballID != "" && ball.ID != ballID && ball.ShortID() != ballID {
					break
				}
				total++
				if ball.State == session.StateComplete {
					complete++
					terminal++
				} else if ball.State == session.StateBlocked {
					blocked++
					terminal++
				}
				break
			}
		}
	}

	return terminal, complete, blocked, total
}

// logTimeoutToProgress logs a timeout event to the session's progress file
func logTimeoutToProgress(projectDir, sessionID, message string) {
	sessionStore, err := session.NewSessionStore(projectDir)
	if err != nil {
		return // Ignore errors - logging is best-effort
	}

	entry := fmt.Sprintf("[TIMEOUT] %s", message)
	_ = sessionStore.AppendProgress(sessionID, entry)
}

// getProgressLineCount returns the number of lines in the session's progress file.
// Used to detect if progress was updated during an iteration.
func getProgressLineCount(store *session.SessionStore, sessionID string) int {
	progress, err := store.LoadProgress(sessionID)
	if err != nil {
		return 0
	}
	if progress == "" {
		return 0
	}
	// Count newlines + 1 for content (unless trailing newline only)
	lines := strings.Count(progress, "\n")
	// If there's content but no trailing newline, add 1
	if len(progress) > 0 && !strings.HasSuffix(progress, "\n") {
		lines++
	}
	return lines
}

// GetProgressLineCountForTest is an exported wrapper for testing
func GetProgressLineCountForTest(store *session.SessionStore, sessionID string) int {
	return getProgressLineCount(store, sessionID)
}

// runAgentRefine implements the agent refine command
func runAgentRefine(cmd *cobra.Command, args []string) error {
	// Parse optional session argument
	var sessionID string
	if len(args) > 0 {
		sessionID = args[0]
	}

	// Get current directory
	cwd, err := GetWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Load balls based on scope
	balls, err := loadBallsForRefine(cwd, sessionID)
	if err != nil {
		return fmt.Errorf("failed to load balls: %w", err)
	}

	if len(balls) == 0 {
		fmt.Println("No balls found to refine.")
		return nil
	}

	// Generate the refinement prompt
	prompt, err := generateRefinePrompt(cwd, sessionID, balls)
	if err != nil {
		return fmt.Errorf("failed to generate prompt: %w", err)
	}

	fmt.Printf("Starting refinement session for %d ball(s)...\n", len(balls))
	if sessionID != "" {
		fmt.Printf("Session filter: %s\n", sessionID)
	}
	fmt.Println()

	// Run Claude in interactive + plan mode
	opts := agent.RunOptions{
		Prompt:     prompt,
		Mode:       agent.ModeInteractive,
		Permission: agent.PermissionPlan,
	}

	_, err = agent.DefaultRunner.Run(opts)
	if err != nil {
		return fmt.Errorf("refinement session failed: %w", err)
	}

	return nil
}

// loadBallsForRefine loads balls based on scope:
// - If sessionID provided, filter by session tag
// - If GlobalOpts.AllProjects, load from all discovered projects
// - Otherwise, load from current repo only
func loadBallsForRefine(projectDir, sessionID string) ([]*session.Ball, error) {
	// Load config to discover projects
	config, err := LoadConfigForCommand()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Create store for current directory
	store, err := NewStoreForCommand(projectDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create store: %w", err)
	}

	// Discover projects (respects --all flag)
	projects, err := DiscoverProjectsForCommand(config, store)
	if err != nil {
		return nil, fmt.Errorf("failed to discover projects: %w", err)
	}

	if len(projects) == 0 {
		return nil, fmt.Errorf("no projects with .juggler directories found")
	}

	// Load all balls from discovered projects
	allBalls, err := session.LoadAllBalls(projects)
	if err != nil {
		return nil, fmt.Errorf("failed to load balls: %w", err)
	}

	// If no session filter or "all" is specified, return all non-complete balls
	// "all" is a special meta-session meaning "all balls in repo"
	if sessionID == "" || sessionID == "all" {
		balls := make([]*session.Ball, 0)
		for _, ball := range allBalls {
			if ball.State != session.StateComplete {
				balls = append(balls, ball)
			}
		}
		return balls, nil
	}

	// Filter by session tag
	balls := make([]*session.Ball, 0)
	for _, ball := range allBalls {
		for _, tag := range ball.Tags {
			if tag == sessionID {
				balls = append(balls, ball)
				break
			}
		}
	}

	return balls, nil
}

// generateRefinePrompt creates the prompt for the refinement session
func generateRefinePrompt(projectDir, sessionID string, balls []*session.Ball) (string, error) {
	var buf strings.Builder

	// Write context section with session info if available
	buf.WriteString("<context>\n")
	if sessionID != "" {
		// Try to load session context
		sessionStore, err := session.NewSessionStore(projectDir)
		if err == nil {
			juggleSession, err := sessionStore.LoadSession(sessionID)
			if err == nil && juggleSession.Description != "" {
				buf.WriteString(fmt.Sprintf("Session: %s\n", sessionID))
				buf.WriteString(fmt.Sprintf("Description: %s\n", juggleSession.Description))
				if juggleSession.Context != "" {
					buf.WriteString("\n")
					buf.WriteString(juggleSession.Context)
					if !strings.HasSuffix(juggleSession.Context, "\n") {
						buf.WriteString("\n")
					}
				}
			}
		}
	}
	buf.WriteString("</context>\n\n")

	// Write balls section with all details
	buf.WriteString("<balls>\n")
	for i, ball := range balls {
		if i > 0 {
			buf.WriteString("\n")
		}
		writeBallForRefine(&buf, ball)
	}
	buf.WriteString("</balls>\n\n")

	// Write instructions section with refinement template
	buf.WriteString("<instructions>\n")
	buf.WriteString(agent.GetRefinePromptTemplate())
	if !strings.HasSuffix(agent.GetRefinePromptTemplate(), "\n") {
		buf.WriteString("\n")
	}
	buf.WriteString("</instructions>\n")

	return buf.String(), nil
}

// LoadBallsForRefineForTest is an exported wrapper for testing
func LoadBallsForRefineForTest(projectDir, sessionID string) ([]*session.Ball, error) {
	return loadBallsForRefine(projectDir, sessionID)
}

// GenerateRefinePromptForTest is an exported wrapper for testing
func GenerateRefinePromptForTest(projectDir, sessionID string, balls []*session.Ball) (string, error) {
	return generateRefinePrompt(projectDir, sessionID, balls)
}

// RunAgentRefineForTest runs the agent refine logic for testing
func RunAgentRefineForTest(projectDir, sessionID string) error {
	// Override GlobalOpts.ProjectDir for test
	oldProjectDir := GlobalOpts.ProjectDir
	GlobalOpts.ProjectDir = projectDir
	defer func() { GlobalOpts.ProjectDir = oldProjectDir }()

	// Load balls based on scope
	balls, err := loadBallsForRefine(projectDir, sessionID)
	if err != nil {
		return fmt.Errorf("failed to load balls: %w", err)
	}

	if len(balls) == 0 {
		return nil // No balls to refine
	}

	// Generate the refinement prompt
	prompt, err := generateRefinePrompt(projectDir, sessionID, balls)
	if err != nil {
		return fmt.Errorf("failed to generate prompt: %w", err)
	}

	// Run Claude in interactive + plan mode
	opts := agent.RunOptions{
		Prompt:     prompt,
		Mode:       agent.ModeInteractive,
		Permission: agent.PermissionPlan,
	}

	_, err = agent.DefaultRunner.Run(opts)
	return err
}

// GenerateAgentPromptForTest is an exported wrapper for testing prompt generation
func GenerateAgentPromptForTest(projectDir, sessionID string, debug bool, ballID string) (string, error) {
	return generateAgentPrompt(projectDir, sessionID, debug, ballID)
}

// writeBallForRefine writes a single ball with all details for refinement
func writeBallForRefine(buf *strings.Builder, ball *session.Ball) {
	// Header with ID, state, and priority
	buf.WriteString(fmt.Sprintf("## %s [%s] (priority: %s)\n", ball.ID, ball.State, ball.Priority))

	// Intent
	buf.WriteString(fmt.Sprintf("Intent: %s\n", ball.Intent))

	// Project directory
	if ball.WorkingDir != "" {
		buf.WriteString(fmt.Sprintf("Project: %s\n", ball.WorkingDir))
	}

	// Acceptance criteria
	if len(ball.AcceptanceCriteria) > 0 {
		buf.WriteString("Acceptance Criteria:\n")
		for i, ac := range ball.AcceptanceCriteria {
			buf.WriteString(fmt.Sprintf("  %d. %s\n", i+1, ac))
		}
	} else {
		buf.WriteString("Acceptance Criteria: (none - needs definition)\n")
	}

	// Blocked reason if blocked
	if ball.State == session.StateBlocked && ball.BlockedReason != "" {
		buf.WriteString(fmt.Sprintf("Blocked: %s\n", ball.BlockedReason))
	}

	// Tags
	if len(ball.Tags) > 0 {
		buf.WriteString(fmt.Sprintf("Tags: %s\n", strings.Join(ball.Tags, ", ")))
	}

	// Output if researched
	if ball.Output != "" {
		buf.WriteString(fmt.Sprintf("Research Output: %s\n", ball.Output))
	}
}
