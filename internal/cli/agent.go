package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ohare93/juggle/internal/agent"
	"github.com/ohare93/juggle/internal/session"
	"github.com/spf13/cobra"
)

var (
	agentIterations int
	agentTrust      bool
	agentTimeout    time.Duration
	agentDebug      bool
	agentMaxWait    time.Duration
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

  # Run with full permissions (dangerous)
  juggle agent run my-feature --trust

  # Run with 5 minute timeout per iteration
  juggle agent run my-feature --timeout 5m

  # Set maximum wait time for rate limits (give up if exceeded)
  juggle agent run my-feature --max-wait 30m`,
	Args: cobra.ExactArgs(1),
	RunE: runAgentRun,
}

func init() {
	agentRunCmd.Flags().IntVarP(&agentIterations, "iterations", "n", 10, "Maximum number of iterations")
	agentRunCmd.Flags().BoolVar(&agentTrust, "trust", false, "Run with --dangerously-skip-permissions (dangerous!)")
	agentRunCmd.Flags().DurationVarP(&agentTimeout, "timeout", "T", 0, "Timeout per iteration (e.g., 5m, 1h). 0 = no timeout")
	agentRunCmd.Flags().BoolVar(&agentDebug, "debug", false, "Add reasoning instructions to agent prompt")
	agentRunCmd.Flags().DurationVar(&agentMaxWait, "max-wait", 0, "Maximum wait time for rate limits before giving up (e.g., 30m). 0 = wait indefinitely")

	agentCmd.AddCommand(agentRunCmd)
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

		// Generate prompt using export command
		prompt, err := generateAgentPrompt(config.ProjectDir, config.SessionID, config.Debug)
		if err != nil {
			return nil, fmt.Errorf("failed to generate prompt: %w", err)
		}

		// Run agent with prompt using the Runner interface
		runResult, err := agent.DefaultRunner.Run(prompt, config.Trust, config.Timeout)
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
			// VALIDATE: Check if all balls are actually in terminal state (complete or blocked)
			terminal, complete, blocked, total := checkBallsTerminal(config.ProjectDir, config.SessionID)
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

		if runResult.Continue {
			// Agent completed one ball, more remain - continue to next iteration
			fmt.Println()
			fmt.Printf("✓ Agent completed a ball, continuing to next iteration...\n")

			// Update ball counts for progress tracking
			_, complete, blocked, total := checkBallsTerminal(config.ProjectDir, config.SessionID)
			result.BallsComplete = complete
			result.BallsBlocked = blocked
			result.BallsTotal = total

			continue
		}

		if runResult.Blocked {
			result.Blocked = true
			result.BlockedReason = runResult.BlockedReason
			break
		}

		// Check if all balls are in terminal state (complete or blocked)
		terminal, complete, blocked, total := checkBallsTerminal(config.ProjectDir, config.SessionID)
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

	// Print warning if --trust is used
	if agentTrust {
		fmt.Println("⚠️  WARNING: Running with --trust flag. Agent has full system permissions.")
		fmt.Println("    Only use this if you trust the agent and understand the risks.")
		fmt.Println()
	}

	fmt.Printf("Starting agent for session: %s\n", sessionID)
	fmt.Printf("Max iterations: %d\n", agentIterations)
	fmt.Println()

	// Print timeout if specified
	if agentTimeout > 0 {
		fmt.Printf("Timeout per iteration: %v\n", agentTimeout)
	}

	// Run the agent loop
	loopConfig := AgentLoopConfig{
		SessionID:     sessionID,
		ProjectDir:    cwd,
		MaxIterations: agentIterations,
		Trust:         agentTrust,
		Debug:         agentDebug,
		IterDelay:     2 * time.Second,
		Timeout:       agentTimeout,
		MaxWait:       agentMaxWait,
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
func generateAgentPrompt(projectDir, sessionID string, debug bool) (string, error) {
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

	// Call exportAgent directly
	output, err := exportAgent(projectDir, sessionID, balls, debug)
	if err != nil {
		return "", err
	}

	return string(output), nil
}

// checkBallsTerminal returns counts of balls in terminal states (complete or blocked) and total balls for session
func checkBallsTerminal(projectDir, sessionID string) (terminal, complete, blocked, total int) {
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
