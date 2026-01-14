package cli

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ohare93/juggle/internal/agent"
	"github.com/ohare93/juggle/internal/agent/provider"
	"github.com/ohare93/juggle/internal/session"
	"github.com/ohare93/juggle/internal/vcs"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// isTerminal checks if the given file descriptor is a terminal
func isTerminal(fd uintptr) bool {
	return term.IsTerminal(int(fd))
}

var (
	agentIterations  int
	agentTrust       bool
	agentTimeout     time.Duration
	agentDebug       bool
	agentDryRun      bool
	agentMaxWait     time.Duration
	agentBallID      string
	agentInteractive bool
	agentModel       string
	agentDelay       int    // Delay between iterations in minutes (overrides config)
	agentFuzz        int    // +/- variance in delay minutes (overrides config)
	agentProvider    string // Agent provider (claude, opencode)
)

// agentCmd is the parent command for agent operations
var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Run AI agents on sessions",
	Long:  `Run AI agents to work on session balls autonomously.`,
}

// agentRunCmd runs the agent loop
var agentRunCmd = &cobra.Command{
	Use:   "run [session-id]",
	Short: "Run agent loop for a session",
	Long: `Run an AI agent loop for a session.

If no session-id is provided, a selector will be shown to choose from
available sessions. Use --all to show sessions from all discovered projects.

Special session "all":
Use "all" as the session-id to run the agent against ALL balls in the current
repo, without requiring a session file. This is useful for working on balls
that aren't tagged to any specific session.

The agent will:
1. Generate a prompt using 'juggle export --format agent'
2. Spawn claude with the prepared prompt
3. Capture output to .juggle/sessions/<id>/last_output.txt
4. Check for COMPLETE/BLOCKED signals after each iteration
5. Repeat until done or max iterations reached

Rate Limit Handling:
When Claude returns a rate limit error (429 or overloaded), the agent
automatically waits with exponential backoff before retrying. If Claude
specifies a retry-after time, that time is used instead.

Examples:
  # Show session selector (interactive)
  juggle agent run

  # Run agent for 10 iterations (default)
  juggle agent run my-feature

  # Run agent against ALL balls in repo (no session filter)
  juggle agent run all

  # Run for specific number of iterations
  juggle agent run my-feature --iterations 5

  # Work on a specific ball only (1 iteration, interactive mode)
  juggle agent run my-feature --ball juggle-5

  # Work on specific ball without specifying session (uses "all" meta-session)
  juggle agent run --ball juggle-5

  # Work on specific ball with multiple iterations (non-interactive)
  juggle agent run my-feature --ball juggle-5 -n 3

  # Run in interactive mode (full Claude TUI)
  juggle agent run my-feature --interactive

  # Run with specific model (small for quick tasks, large for complex work)
  juggle agent run my-feature --model sonnet

  # Run with full permissions (dangerous)
  juggle agent run my-feature --trust

  # Run with 5 minute timeout per iteration
  juggle agent run my-feature --timeout 5m

  # Set maximum wait time for rate limits (give up if exceeded)
  juggle agent run my-feature --max-wait 30m

  # Show prompt info without running (dry run)
  juggle agent run my-feature --dry-run

  # Show prompt info before running (debug mode)
  juggle agent run my-feature --debug

  # Select from sessions across all discovered projects
  juggle agent run --all

  # Override iteration delay (5 minutes, overrides config)
  juggle agent run my-feature --delay 5

  # Override delay with variance (5 minutes ± 2 minutes)
  juggle agent run my-feature --delay 5 --fuzz 2

  # Disable delay entirely (overrides config even if set)
  juggle agent run my-feature --delay 0`,
	Args: cobra.MaximumNArgs(1),
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
	agentRunCmd.Flags().StringVarP(&agentModel, "model", "m", "", "Model to use (opus, sonnet, haiku). Default: opus for large balls, sonnet for others")
	agentRunCmd.Flags().IntVar(&agentDelay, "delay", 0, "Delay between iterations in minutes (overrides config, 0 = no delay)")
	agentRunCmd.Flags().IntVar(&agentFuzz, "fuzz", 0, "Random +/- variance in delay minutes (overrides config)")
	agentRunCmd.Flags().StringVar(&agentProvider, "provider", "", "Agent provider to use (claude, opencode). Default: from config or claude")

	agentCmd.AddCommand(agentRunCmd)
	agentCmd.AddCommand(agentRefineCmd)
	rootCmd.AddCommand(agentCmd)
}

// AgentResult holds the result of an agent run
type AgentResult struct {
	Iterations         int           `json:"iterations"`
	Complete           bool          `json:"complete"`
	Blocked            bool          `json:"blocked"`
	BlockedReason      string        `json:"blocked_reason,omitempty"`
	TimedOut           bool          `json:"timed_out"`
	TimeoutMessage     string        `json:"timeout_message,omitempty"`
	RateLimitExceded   bool          `json:"rate_limit_exceeded"`
	TotalWaitTime      time.Duration `json:"total_wait_time,omitempty"`
	OverloadRetries    int           `json:"overload_retries,omitempty"`    // Number of 529 overload retry waits
	OverloadWaitTime   time.Duration `json:"overload_wait_time,omitempty"` // Total time spent waiting for overload recovery
	BallsComplete      int           `json:"balls_complete"`
	BallsBlocked       int           `json:"balls_blocked"`
	BallsTotal         int           `json:"balls_total"`
	StartedAt          time.Time     `json:"started_at"`
	EndedAt            time.Time     `json:"ended_at"`
}

// AgentLoopConfig configures the agent loop behavior
type AgentLoopConfig struct {
	SessionID            string
	ProjectDir           string
	MaxIterations        int
	Trust                bool
	Debug                bool          // Add debug reasoning instructions to prompt
	IterDelay            time.Duration // Delay between iterations (set to 0 for tests)
	Timeout              time.Duration // Timeout per iteration (0 = no timeout)
	MaxWait              time.Duration // Maximum time to wait for rate limits (0 = wait indefinitely)
	BallID               string        // Specific ball to work on (empty = all session balls)
	Interactive          bool          // Run in interactive mode (full Claude TUI)
	Model                string        // Model to use (opus, sonnet, haiku). Empty = auto-select based on ball model_size
	OverloadRetryMinutes int           // Minutes to wait before retrying after 529 overload exhaustion (-1 = use config default, 0 = no wait)
	Provider             string        // Agent provider to use (claude, opencode). Empty = from config or claude
}

// sessionStorageID returns the session ID used for storage (progress, output, lock)
// For the "all" meta-session, this returns "_all" since "all" is reserved as a meta-session name
func sessionStorageID(sessionID string) string {
	if sessionID == "all" {
		return "_all"
	}
	return sessionID
}

// RunAgentLoop executes the agent loop with the given configuration.
// This is the testable core of the agent run command.
func RunAgentLoop(config AgentLoopConfig) (*AgentResult, error) {
	startTime := time.Now()

	sessionStore, err := session.NewSessionStore(config.ProjectDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create session store: %w", err)
	}

	// "all" is a special meta-session that targets all balls in repo without requiring a session file
	isAllSession := config.SessionID == "all"

	// Load session to get default model (for non-"all" sessions)
	var juggleSession *session.JuggleSession
	if !isAllSession {
		var err error
		juggleSession, err = sessionStore.LoadSession(config.SessionID)
		if err != nil {
			return nil, fmt.Errorf("session not found: %s", config.SessionID)
		}
	}

	// Acquire exclusive lock on session to prevent concurrent agent runs
	// For "all" meta-session, we lock with "_all" as the storage ID
	storageID := sessionStorageID(config.SessionID)
	lock, err := sessionStore.AcquireSessionLock(storageID)
	if err != nil {
		return nil, err
	}
	defer lock.Release()

	// Create output file path using storage ID
	// For "all" meta-session, ensure the _all session directory exists
	if isAllSession {
		allDir := filepath.Join(config.ProjectDir, ".juggle", "sessions", "_all")
		if err := os.MkdirAll(allDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create _all session directory: %w", err)
		}
	}
	outputPath := filepath.Join(config.ProjectDir, ".juggle", "sessions", storageID, "last_output.txt")

	result := &AgentResult{
		StartedAt: startTime,
	}

	// Track rate limit state
	var totalWaitTime time.Duration
	rateLimitRetries := 0
	rateLimitRetrying := false // Skip header when retrying after rate limit

	// Track 529 overload exhaustion state
	var overloadWaitTime time.Duration
	overloadRetries := 0
	overloadRetrying := false // Skip header when retrying after overload

	// Load overload retry interval from config (or use provided override)
	// -1 means "use config default", 0 means "no wait" (for testing), >0 is explicit minutes
	overloadRetryMinutes := config.OverloadRetryMinutes
	if overloadRetryMinutes < 0 {
		overloadRetryMinutes, _ = session.GetGlobalOverloadRetryMinutesWithOptions(GetConfigOptions())
	}

	// Configure agent provider based on CLI flag, project config, and global config
	globalProvider, err := session.GetGlobalAgentProviderWithOptions(GetConfigOptions())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load global agent provider config: %v\n", err)
	}
	projectProvider, err := session.GetProjectAgentProvider(config.ProjectDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load project agent provider config: %v\n", err)
	}
	providerType := provider.Detect(config.Provider, projectProvider, globalProvider)

	// Verify provider binary is available
	if !provider.IsAvailable(providerType) {
		return nil, fmt.Errorf("agent provider %q is not available (binary %q not found in PATH)",
			providerType, provider.BinaryName(providerType))
	}

	agentProv := provider.Get(providerType)
	agent.SetProvider(agentProv)

	// Configure model overrides
	globalOverrides, err := session.GetGlobalModelOverridesWithOptions(GetConfigOptions())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load global model overrides: %v\n", err)
	}
	projectOverrides, err := session.GetProjectModelOverrides(config.ProjectDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load project model overrides: %v\n", err)
	}
	modelOverrides := session.MergeModelOverrides(globalOverrides, projectOverrides)
	agent.SetModelOverrides(modelOverrides)

	// Pre-loop check: is there any work the agent can do?
	// Exit early if all balls are blocked (need human intervention) or no actionable balls exist
	// Exception: --ball or --interactive means human IS intervening, so blocked balls are workable
	workable, blockedCount, totalCount, err := countWorkableBalls(config.ProjectDir, config.SessionID, config.BallID, config.Interactive)
	if err != nil {
		return nil, fmt.Errorf("checking workable balls: %w", err)
	}

	if workable == 0 {
		result.EndedAt = time.Now()
		result.Iterations = 0
		result.BallsTotal = totalCount
		result.BallsBlocked = blockedCount

		if blockedCount > 0 {
			fmt.Fprintf(os.Stderr, "⏸ No actionable work: %d ball(s) blocked, waiting for human intervention\n", blockedCount)
			result.Blocked = true
			return result, nil
		}
		// No balls at all (all complete/researched or truly empty)
		fmt.Fprintf(os.Stderr, "✓ No actionable balls in session\n")
		result.Complete = true
		return result, nil
	}

	for iteration := 1; iteration <= config.MaxIterations; iteration++ {
		result.Iterations = iteration

		// Print iteration separator and header (skip when retrying after rate limit or overload)
		if !rateLimitRetrying && !overloadRetrying {
			if iteration > 1 {
				fmt.Println()
				fmt.Println()
				fmt.Println("════════════════════════════════════════════════════════════════════════════════")
				fmt.Println()
				fmt.Println()
			}
			fmt.Printf("════════════════════════════════ Iteration %d/%d ════════════════════════════════\n\n", iteration, config.MaxIterations)
		}
		rateLimitRetrying = false  // Reset for next iteration
		overloadRetrying = false   // Reset for next iteration

		// Record progress state before iteration (for validation)
		// Use storageID (maps "all" to "_all") for progress tracking
		progressBefore := getProgressLineCount(sessionStore, storageID)

		// Load balls for model selection
		balls, err := loadBallsForModelSelection(config.ProjectDir, config.SessionID, config.BallID)
		if err != nil {
			return nil, fmt.Errorf("failed to load balls for model selection: %w", err)
		}

		// Get session default model
		var sessionDefaultModel session.ModelSize
		if juggleSession != nil {
			sessionDefaultModel = juggleSession.DefaultModel
		}

		// Select optimal model for this iteration
		modelSelection := selectModelForIteration(config, balls, sessionDefaultModel)

		// Log model selection (only if not explicitly set)
		if config.Model == "" {
			fmt.Printf("🤖 Model: %s (%s)\n\n", modelSelection.Model, modelSelection.Reason)
		}

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
			Model:      modelSelection.Model,
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
				logRateLimitToProgress(config.ProjectDir, storageID,
					fmt.Sprintf("Rate limit exceeded max-wait of %v (total waited: %v)", config.MaxWait, totalWaitTime))
				break
			}

			// Log waiting status
			logRateLimitToProgress(config.ProjectDir, storageID,
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

		// Check for 529 overload exhaustion (Claude's built-in retries exhausted)
		if runResult.OverloadExhausted {
			waitTime := time.Duration(overloadRetryMinutes) * time.Minute

			// Check if we've exceeded max wait
			if config.MaxWait > 0 && totalWaitTime+overloadWaitTime+waitTime > config.MaxWait {
				result.RateLimitExceded = true
				result.TotalWaitTime = totalWaitTime + overloadWaitTime
				result.OverloadRetries = overloadRetries
				result.OverloadWaitTime = overloadWaitTime
				logOverloadToProgress(config.ProjectDir, storageID,
					fmt.Sprintf("Overload retry exceeded max-wait of %v (total waited: %v)", config.MaxWait, totalWaitTime+overloadWaitTime))
				break
			}

			// Log waiting status
			logOverloadToProgress(config.ProjectDir, storageID,
				fmt.Sprintf("Claude API overloaded (529), waiting %v before retry (attempt %d)", waitTime, overloadRetries+1))

			fmt.Printf("🔥 Claude API overloaded (529). Built-in retries exhausted.\n")
			fmt.Printf("⏳ Waiting %v before restarting agent...\n", waitTime)

			// Wait with countdown display
			waitWithCountdown(waitTime)

			overloadWaitTime += waitTime
			overloadRetries++
			overloadRetrying = true // Skip header on retry

			// Retry this iteration (don't increment)
			iteration--
			continue
		}

		// Check for timeout
		if runResult.TimedOut {
			result.TimedOut = true
			result.TimeoutMessage = fmt.Sprintf("Iteration %d timed out after %v", iteration, config.Timeout)
			// Log timeout to progress
			logTimeoutToProgress(config.ProjectDir, storageID, result.TimeoutMessage)
			break
		}

		// Save output to file (ignore errors for test compatibility)
		_ = os.WriteFile(outputPath, []byte(runResult.Output), 0644)

		// Check for completion signals (already parsed by Runner)
		if runResult.Complete {
			// VALIDATE: Check if progress was updated this iteration
			progressAfter := getProgressLineCount(sessionStore, storageID)
			if progressAfter <= progressBefore {
				fmt.Println()
				fmt.Printf("⚠️  Agent signaled COMPLETE but did not update progress. Continuing iteration...\n")
				// Don't accept the signal - continue to check terminal state
			} else {
				// VALIDATE: Check if all balls are actually in terminal state (complete or blocked)
				terminal, complete, blocked, total := checkBallsTerminal(config.ProjectDir, config.SessionID, config.BallID)
				if total > 0 && terminal == total {
					// Commit changes if agent provided a commit message
					if runResult.CommitMessage != "" {
						commitResult, err := performJJCommit(config.ProjectDir, runResult.CommitMessage)
						if err == nil && commitResult != nil {
							if commitResult.Success {
								if commitResult.CommitHash != "" {
									fmt.Printf("📝 Committed: %s\n", commitResult.CommitHash)
								}
								if commitResult.StatusOutput != "No changes to commit" {
									fmt.Printf("📊 Status: %s\n", commitResult.StatusOutput)
								}
							} else if commitResult.ErrorMessage != "" {
								fmt.Printf("⚠️  Commit failed: %s\n", commitResult.ErrorMessage)
							}
						}
					}
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
			progressAfter := getProgressLineCount(sessionStore, storageID)
			if progressAfter <= progressBefore {
				fmt.Println()
				fmt.Printf("⚠️  Agent signaled CONTINUE but did not update progress. Continuing iteration...\n")
				// Don't accept the signal - fall through to terminal state check
			} else {
				// Agent completed one ball, more remain - continue to next iteration
				fmt.Println()
				fmt.Printf("✓ Agent completed a ball, continuing to next iteration...\n")

				// Commit changes if agent provided a commit message
				if runResult.CommitMessage != "" {
					commitResult, err := performJJCommit(config.ProjectDir, runResult.CommitMessage)
					if err == nil && commitResult != nil {
						if commitResult.Success {
							if commitResult.CommitHash != "" {
								fmt.Printf("📝 Committed: %s\n", commitResult.CommitHash)
							}
							if commitResult.StatusOutput != "No changes to commit" {
								fmt.Printf("📊 Status: %s\n", commitResult.StatusOutput)
							}
						} else if commitResult.ErrorMessage != "" {
							fmt.Printf("⚠️  Commit failed: %s\n", commitResult.ErrorMessage)
						}
					}
				}

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
			progressAfter := getProgressLineCount(sessionStore, storageID)
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

	result.TotalWaitTime = totalWaitTime + overloadWaitTime
	result.OverloadRetries = overloadRetries
	result.OverloadWaitTime = overloadWaitTime
	result.EndedAt = time.Now()

	// Save run history (best-effort, don't fail the run if this errors)
	saveAgentHistory(config, result, outputPath)

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

// calculateFuzzyDelay calculates the actual delay to use with random variance.
// baseMinutes is the base delay in minutes, fuzz is the +/- variance in minutes.
// The actual delay will be: base + random(-fuzz, fuzz) minutes.
// The result is guaranteed to be >= 0.
func calculateFuzzyDelay(baseMinutes, fuzz int) time.Duration {
	delay := baseMinutes
	if fuzz > 0 {
		// Add random variance: -fuzz to +fuzz
		variance := rand.Intn(2*fuzz+1) - fuzz
		delay += variance
	}
	// Ensure non-negative
	if delay < 0 {
		delay = 0
	}
	return time.Duration(delay) * time.Minute
}

// CalculateFuzzyDelayForTest is an exported wrapper for testing
func CalculateFuzzyDelayForTest(baseMinutes, fuzz int) time.Duration {
	return calculateFuzzyDelay(baseMinutes, fuzz)
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

// logOverloadToProgress logs a 529 overload event to the session's progress file
func logOverloadToProgress(projectDir, sessionID, message string) {
	sessionStore, err := session.NewSessionStore(projectDir)
	if err != nil {
		return // Ignore errors - logging is best-effort
	}

	entry := fmt.Sprintf("[OVERLOAD_529] %s", message)
	_ = sessionStore.AppendProgress(sessionID, entry)
}

// SessionSelection holds the result of selecting a session for agent run
type SessionSelection struct {
	SessionID  string
	ProjectDir string
}

// selectSessionForAgent shows an interactive session selector for agent run.
// Returns the selected session info or nil if cancelled.
func selectSessionForAgent(cwd string) (*SessionSelection, error) {
	// Load config to discover projects
	config, err := LoadConfigForCommand()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Create store for current directory
	store, err := NewStoreForCommand(cwd)
	if err != nil {
		return nil, fmt.Errorf("failed to create store: %w", err)
	}

	// Get session store for local sessions
	sessionStore, err := session.NewSessionStoreWithConfig(cwd, GetStoreConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize session store: %w", err)
	}

	// Collect sessions based on scope (--all flag)
	type sessionInfo struct {
		ID          string
		Description string
		ProjectDir  string
		BallCount   int
	}
	var sessions []sessionInfo

	if GlobalOpts.AllProjects {
		// Discover all projects and their sessions
		projects, err := DiscoverProjectsForCommand(config, store)
		if err != nil {
			return nil, fmt.Errorf("failed to discover projects: %w", err)
		}

		for _, projectPath := range projects {
			projSessionStore, err := session.NewSessionStore(projectPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to create session store for %s: %v\n", projectPath, err)
				continue
			}

			projSessions, err := projSessionStore.ListSessions()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to list sessions for %s: %v\n", projectPath, err)
				continue
			}

			for _, s := range projSessions {
				// Count balls for this session
				balls, _ := session.LoadBallsBySession([]string{projectPath}, s.ID)
				sessions = append(sessions, sessionInfo{
					ID:          s.ID,
					Description: s.Description,
					ProjectDir:  projectPath,
					BallCount:   len(balls),
				})
			}
		}
	} else {
		// Local sessions only
		localSessions, err := sessionStore.ListSessions()
		if err != nil {
			return nil, fmt.Errorf("failed to list sessions: %w", err)
		}

		for _, s := range localSessions {
			// Count balls for this session
			balls, _ := session.LoadBallsBySession([]string{cwd}, s.ID)
			sessions = append(sessions, sessionInfo{
				ID:          s.ID,
				Description: s.Description,
				ProjectDir:  cwd,
				BallCount:   len(balls),
			})
		}
	}

	if len(sessions) == 0 {
		scopeMsg := "this project"
		if GlobalOpts.AllProjects {
			scopeMsg = "any discovered project"
		}
		return nil, fmt.Errorf("no sessions found in %s. Create one with: juggle sessions create <id>", scopeMsg)
	}

	// Display session selector
	fmt.Println("Select a session to run the agent on:")
	fmt.Println()

	for i, s := range sessions {
		prefix := fmt.Sprintf("  %d. %s", i+1, s.ID)
		ballInfo := fmt.Sprintf("(%d balls)", s.BallCount)
		if s.Description != "" {
			fmt.Printf("%s - %s %s\n", prefix, s.Description, ballInfo)
		} else {
			fmt.Printf("%s %s\n", prefix, ballInfo)
		}
		// Show project directory if viewing all projects
		if GlobalOpts.AllProjects {
			fmt.Printf("     📁 %s\n", s.ProjectDir)
		}
	}
	fmt.Println()
	fmt.Print("Enter number (or 'q' to cancel): ")

	// Read selection
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read input: %w", err)
	}
	input = strings.TrimSpace(input)

	// Handle cancel
	if input == "q" || input == "Q" || input == "" {
		return nil, nil
	}

	// Parse selection
	var idx int
	_, err = fmt.Sscanf(input, "%d", &idx)
	if err != nil || idx < 1 || idx > len(sessions) {
		return nil, fmt.Errorf("invalid selection: %s", input)
	}

	selected := sessions[idx-1]

	// If the selected session is from a different project, notify the user
	if selected.ProjectDir != cwd {
		fmt.Printf("\n📁 Session is in project: %s\n", selected.ProjectDir)
		fmt.Printf("   Running agent in that directory...\n\n")
	}

	return &SessionSelection{
		SessionID:  selected.ID,
		ProjectDir: selected.ProjectDir,
	}, nil
}

// SelectSessionForAgentForTest is an exported wrapper for testing
func SelectSessionForAgentForTest(cwd string) (*SessionSelection, error) {
	return selectSessionForAgent(cwd)
}

// SessionInfo holds information about a session for testing/display
type SessionInfo struct {
	ID          string
	Description string
	ProjectDir  string
	BallCount   int
}

// GetSessionsForSelectorForTest returns the list of sessions that would be shown in the selector.
// This is for testing purposes to verify cross-project session discovery.
func GetSessionsForSelectorForTest(cwd string) ([]SessionInfo, error) {
	// Load config to discover projects
	config, err := LoadConfigForCommand()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Create store for current directory
	store, err := NewStoreForCommand(cwd)
	if err != nil {
		return nil, fmt.Errorf("failed to create store: %w", err)
	}

	// Get session store for local sessions
	sessionStore, err := session.NewSessionStoreWithConfig(cwd, GetStoreConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize session store: %w", err)
	}

	var sessions []SessionInfo

	if GlobalOpts.AllProjects {
		// Discover all projects and their sessions
		projects, err := DiscoverProjectsForCommand(config, store)
		if err != nil {
			return nil, fmt.Errorf("failed to discover projects: %w", err)
		}

		for _, projectPath := range projects {
			projSessionStore, err := session.NewSessionStore(projectPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to create session store for %s: %v\n", projectPath, err)
				continue
			}

			projSessions, err := projSessionStore.ListSessions()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to list sessions for %s: %v\n", projectPath, err)
				continue
			}

			for _, s := range projSessions {
				// Count balls for this session
				balls, _ := session.LoadBallsBySession([]string{projectPath}, s.ID)
				sessions = append(sessions, SessionInfo{
					ID:          s.ID,
					Description: s.Description,
					ProjectDir:  projectPath,
					BallCount:   len(balls),
				})
			}
		}
	} else {
		// Local sessions only
		localSessions, err := sessionStore.ListSessions()
		if err != nil {
			return nil, fmt.Errorf("failed to list sessions: %w", err)
		}

		for _, s := range localSessions {
			// Count balls for this session
			balls, _ := session.LoadBallsBySession([]string{cwd}, s.ID)
			sessions = append(sessions, SessionInfo{
				ID:          s.ID,
				Description: s.Description,
				ProjectDir:  cwd,
				BallCount:   len(balls),
			})
		}
	}

	return sessions, nil
}

func runAgentRun(cmd *cobra.Command, args []string) error {
	// Get current directory
	cwd, err := GetWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Track which project directory to use (may change if session is in different project)
	projectDir := cwd

	// Determine session ID from args or selector
	var sessionID string
	if len(args) > 0 {
		sessionID = args[0]
	} else if agentBallID != "" {
		// --ball specified without session - default to "all" meta-session
		sessionID = "all"
	} else {
		// No session provided - check if stdin is a terminal before showing selector
		// In headless mode (no tty), error gracefully instead of hanging on selector
		if !isTerminal(os.Stdin.Fd()) {
			return fmt.Errorf("session-id is required in non-interactive mode (use 'all' to target all balls)")
		}
		// Show selector
		selected, err := selectSessionForAgent(cwd)
		if err != nil {
			return err
		}
		if selected == nil {
			// User cancelled
			return nil
		}
		sessionID = selected.SessionID
		projectDir = selected.ProjectDir
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
		prompt, err := generateAgentPrompt(projectDir, sessionID, true, agentBallID) // debug=true for reasoning instructions
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
		if agentModel != "" {
			fmt.Printf("Model: %s\n", agentModel)
		}
		if agentProvider != "" {
			fmt.Printf("Provider: %s\n", agentProvider)
		}
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

	// Load iteration delay settings (flags override config)
	var iterDelay time.Duration
	var delayMinutes, fuzz int

	// Check if --delay flag was explicitly provided
	if cmd.Flags().Changed("delay") {
		delayMinutes = agentDelay
		// Check if --fuzz was also provided, otherwise default to 0
		if cmd.Flags().Changed("fuzz") {
			fuzz = agentFuzz
		}
	} else {
		// Load from config
		var err error
		delayMinutes, fuzz, err = session.GetGlobalIterationDelayWithOptions(GetConfigOptions())
		if err != nil {
			delayMinutes = 0
			fuzz = 0
		}
		// Override fuzz from flag if set
		if cmd.Flags().Changed("fuzz") {
			fuzz = agentFuzz
		}
	}

	// If delay is 0, skip the delay feature entirely (regardless of fuzz)
	if delayMinutes > 0 {
		iterDelay = calculateFuzzyDelay(delayMinutes, fuzz)
		fmt.Printf("Iteration delay: %v", iterDelay.Round(time.Second))
		if fuzz > 0 {
			fmt.Printf(" (base: %dm ± %dm)", delayMinutes, fuzz)
		}
		fmt.Println()
	}

	// Run the agent loop
	loopConfig := AgentLoopConfig{
		SessionID:            sessionID,
		ProjectDir:           projectDir,
		MaxIterations:        iterations,
		Trust:                agentTrust,
		Debug:                false, // Debug mode now just shows prompt info, doesn't affect prompt content
		IterDelay:            iterDelay,
		Timeout:              agentTimeout,
		MaxWait:              agentMaxWait,
		BallID:               agentBallID,
		Interactive:          interactive,
		Model:                agentModel,
		OverloadRetryMinutes: -1,           // Use config default
		Provider:             agentProvider, // Use CLI flag (empty = auto-detect from config)
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
		fmt.Printf("Total wait time: %v\n", result.TotalWaitTime.Round(time.Second))
		if result.OverloadRetries > 0 {
			fmt.Printf("  - Overload (529) retries: %d (waited %v)\n", result.OverloadRetries, result.OverloadWaitTime.Round(time.Second))
		}
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

	// Map "all" meta-session to "_all" for output path
	outputStorageID := sessionStorageID(sessionID)
	outputPath := filepath.Join(projectDir, ".juggle", "sessions", outputStorageID, "last_output.txt")
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
		return "", fmt.Errorf("no projects with .juggle directories found")
	}

	// Load all balls from discovered projects
	allBalls, err := session.LoadAllBalls(projects)
	if err != nil {
		return "", fmt.Errorf("failed to load balls: %w", err)
	}

	// Filter by session tag
	// "all" is a special meta-session that means "all balls in repo" (no session filtering)
	var balls []*session.Ball
	if sessionID == "all" {
		balls = allBalls
	} else {
		balls = make([]*session.Ball, 0)
		for _, ball := range allBalls {
			for _, tag := range ball.Tags {
				if tag == sessionID {
					balls = append(balls, ball)
					break
				}
			}
		}
	}

	// Filter out complete and blocked balls by default (they clutter the context for no gain)
	// Exception: when a specific ball is requested, allow it even if complete/blocked
	if ballID == "" {
		filteredBalls := make([]*session.Ball, 0, len(balls))
		for _, ball := range balls {
			if ball.State != session.StateComplete && ball.State != session.StateResearched && ball.State != session.StateBlocked {
				filteredBalls = append(filteredBalls, ball)
			}
		}
		balls = filteredBalls
	}

	// Filter to specific ball if ballID is specified
	singleBall := false
	if ballID != "" {
		matches := session.ResolveBallByPrefix(balls, ballID)
		if len(matches) == 0 {
			return "", fmt.Errorf("ball %s not found in session %s", ballID, sessionID)
		}
		if len(matches) > 1 {
			matchingIDs := make([]string, len(matches))
			for i, m := range matches {
				matchingIDs[i] = m.ID
			}
			return "", fmt.Errorf("ambiguous ID '%s' matches %d balls: %s", ballID, len(matches), strings.Join(matchingIDs, ", "))
		}
		balls = []*session.Ball{matches[0]}
		singleBall = true
	}

	// Call exportAgent directly
	output, err := exportAgent(projectDir, sessionID, balls, debug, singleBall)
	if err != nil {
		return "", err
	}

	return string(output), nil
}

// countWorkableBalls returns counts of balls the agent can work on (pending/in_progress) vs blocked
// This is used for pre-loop validation to exit early when there's no actionable work
// Balls in complete/researched states are excluded (same as agent export)
// If ballID is specified, only counts that specific ball
// If interactive is true, blocked balls are treated as workable (human is present to intervene)
// "all" is a special meta-session that includes all balls in the repo without filtering by tag
func countWorkableBalls(projectDir, sessionID, ballID string, interactive bool) (workable, blocked, total int, err error) {
	// Load config
	config, err := LoadConfigForCommand()
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to load config: %w", err)
	}

	// Create store
	store, err := NewStoreForCommand(projectDir)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to create store: %w", err)
	}

	// Discover projects
	projects, err := DiscoverProjectsForCommand(config, store)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to discover projects: %w", err)
	}

	// Load all balls
	allBalls, err := session.LoadAllBalls(projects)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to load balls: %w", err)
	}

	// "all" is a meta-session that means "all balls in repo"
	isAllSession := sessionID == "all"

	// Count balls with session tag (or all balls if using "all" meta-session)
	for _, ball := range allBalls {
		var matchesSession bool
		if isAllSession {
			matchesSession = true // Include all balls
		} else {
			for _, tag := range ball.Tags {
				if tag == sessionID {
					matchesSession = true
					break
				}
			}
		}

		if matchesSession {
			// If filtering by specific ball, skip others
			if ballID != "" && ball.ID != ballID && ball.ShortID() != ballID {
				continue
			}

			// Skip states that are excluded from agent exports
			// (complete, researched are not shown to the agent)
			switch ball.State {
			case session.StateComplete, session.StateResearched:
				continue
			case session.StatePending, session.StateInProgress:
				workable++
				total++
			case session.StateBlocked:
				// If user is running interactively or explicitly targeted this ball,
				// treat it as workable (they ARE the human intervention)
				if interactive || (ballID != "" && (ball.ID == ballID || ball.ShortID() == ballID)) {
					workable++
				} else {
					blocked++
				}
				total++
			}
		}
	}

	return workable, blocked, total, nil
}

// checkBallsTerminal returns counts of balls in terminal states (complete or blocked) and total balls for session
// If ballID is specified, only counts that specific ball
// "all" is a special meta-session that includes all balls in the repo without filtering by tag
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

	// "all" is a meta-session that means "all balls in repo"
	isAllSession := sessionID == "all"

	// Count balls with session tag (or all balls if using "all" meta-session)
	for _, ball := range allBalls {
		var matchesSession bool
		if isAllSession {
			matchesSession = true // Include all balls
		} else {
			for _, tag := range ball.Tags {
				if tag == sessionID {
					matchesSession = true
					break
				}
			}
		}

		if matchesSession {
			// If filtering by specific ball, skip others
			if ballID != "" && ball.ID != ballID && ball.ShortID() != ballID {
				continue
			}
			total++
			if ball.State == session.StateComplete {
				complete++
				terminal++
			} else if ball.State == session.StateBlocked {
				blocked++
				terminal++
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

// saveAgentHistory saves the agent run history to the history file
func saveAgentHistory(config AgentLoopConfig, result *AgentResult, outputPath string) {
	historyStore, err := session.NewAgentHistoryStore(config.ProjectDir)
	if err != nil {
		return // Best-effort, ignore errors
	}

	record := session.NewAgentRunRecord(config.SessionID, config.ProjectDir, result.StartedAt)
	record.MaxIterations = config.MaxIterations
	record.OutputFile = outputPath

	// Set the appropriate result type
	if result.Complete {
		record.SetComplete(result.Iterations, result.BallsComplete, result.BallsBlocked, result.BallsTotal)
	} else if result.Blocked {
		record.SetBlocked(result.Iterations, result.BlockedReason, result.BallsComplete, result.BallsBlocked, result.BallsTotal)
	} else if result.TimedOut {
		record.SetTimeout(result.Iterations, result.TimeoutMessage, result.BallsComplete, result.BallsBlocked, result.BallsTotal)
	} else if result.RateLimitExceded {
		record.SetRateLimitExceeded(result.Iterations, result.TotalWaitTime, result.BallsComplete, result.BallsBlocked, result.BallsTotal)
	} else {
		// Max iterations reached
		record.SetMaxIterations(result.Iterations, result.BallsComplete, result.BallsBlocked, result.BallsTotal)
	}

	// Preserve total wait time and ended time from result
	record.TotalWaitTime = result.TotalWaitTime
	record.EndedAt = result.EndedAt

	_ = historyStore.AppendRecord(record)
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
		return nil, fmt.Errorf("no projects with .juggle directories found")
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

	// Title
	buf.WriteString(fmt.Sprintf("Title: %s\n", ball.Title))

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

// ModelSelection contains model selection results
type ModelSelection struct {
	Model      string   // Model to use for this iteration (opus, sonnet, haiku)
	Reason     string   // Why this model was selected
	BallsCount int      // Number of balls that prefer this model
}

// selectModelForIteration analyzes remaining balls and chooses the optimal model.
// Priority order:
// 1. If config.Model is explicitly set (via --model flag), use it
// 2. Use session.DefaultModel if available
// 3. Choose based on ball model preferences (prioritize matching balls)
// 4. Default to "opus" (largest/most capable model)
//
// The function returns the model to use and reason for selection.
func selectModelForIteration(config AgentLoopConfig, balls []*session.Ball, defaultSessionModel session.ModelSize) *ModelSelection {
	// If model explicitly provided via --model flag, use it
	if config.Model != "" {
		return &ModelSelection{
			Model:  config.Model,
			Reason: "explicitly set via --model flag",
		}
	}

	// Filter to non-terminal balls only
	activeBalls := filterActiveBalls(balls)
	if len(activeBalls) == 0 {
		return &ModelSelection{
			Model:  "opus",
			Reason: "no active balls",
		}
	}

	// Count balls by model preference
	modelCounts := countBallsByModel(activeBalls)

	// If session has a default model and there are balls without explicit preference,
	// count those as preferring the session default
	if defaultSessionModel != "" && defaultSessionModel != session.ModelSizeBlank {
		blankCount := modelCounts[""]
		sessionModel := mapModelSizeToString(defaultSessionModel)
		modelCounts[sessionModel] += blankCount
		delete(modelCounts, "")
	}

	// Find the model with most balls (prefer larger models on tie)
	selectedModel := "opus"
	maxCount := 0
	selectedReason := "default (no model preferences specified)"

	// Check in order of preference (larger models first for ties)
	modelPriority := []string{"opus", "sonnet", "haiku"}
	for _, model := range modelPriority {
		count := modelCounts[model]
		if count > maxCount {
			maxCount = count
			selectedModel = model
			selectedReason = fmt.Sprintf("%d ball(s) prefer %s model", count, model)
		}
	}

	// If only blank preferences and no session default, use opus
	if maxCount == 0 {
		if defaultSessionModel != "" && defaultSessionModel != session.ModelSizeBlank {
			selectedModel = mapModelSizeToString(defaultSessionModel)
			selectedReason = "session default model"
		} else {
			selectedModel = "opus"
			selectedReason = "default (no preferences)"
		}
	}

	return &ModelSelection{
		Model:      selectedModel,
		Reason:     selectedReason,
		BallsCount: maxCount,
	}
}

// filterActiveBalls returns only balls that are not in terminal state (complete/researched)
func filterActiveBalls(balls []*session.Ball) []*session.Ball {
	active := make([]*session.Ball, 0)
	for _, ball := range balls {
		if ball.State != session.StateComplete && ball.State != session.StateResearched {
			active = append(active, ball)
		}
	}
	return active
}

// countBallsByModel counts how many balls prefer each model size
func countBallsByModel(balls []*session.Ball) map[string]int {
	counts := make(map[string]int)
	for _, ball := range balls {
		model := mapModelSizeToString(ball.ModelSize)
		counts[model]++
	}
	return counts
}

// mapModelSizeToString converts ModelSize to the string used by Claude CLI
func mapModelSizeToString(size session.ModelSize) string {
	switch size {
	case session.ModelSizeSmall:
		return "haiku"
	case session.ModelSizeMedium:
		return "sonnet"
	case session.ModelSizeLarge:
		return "opus"
	default:
		return "" // blank/unset
	}
}

// prioritizeBallsByModel sorts balls so those matching the current model come first.
// This is called after sortBallsForAgent to further prioritize by model match.
// Within model-matched balls, the existing sort order (state, deps, priority) is preserved.
func prioritizeBallsByModel(balls []*session.Ball, currentModel string, sessionDefaultModel session.ModelSize) {
	if currentModel == "" {
		return // No prioritization needed if no model set
	}

	// Determine which ModelSize values match the current model
	matchesModel := func(ball *session.Ball) bool {
		ballModel := ball.ModelSize
		// If ball has no preference, use session default
		if ballModel == "" || ballModel == session.ModelSizeBlank {
			ballModel = sessionDefaultModel
		}
		// Convert to string and compare
		return mapModelSizeToString(ballModel) == currentModel || ballModel == session.ModelSizeBlank
	}

	// Stable sort: matching balls first, preserving relative order within each group
	sort.SliceStable(balls, func(i, j int) bool {
		matchI := matchesModel(balls[i])
		matchJ := matchesModel(balls[j])
		// Only reorder if one matches and other doesn't
		if matchI && !matchJ {
			return true
		}
		return false
	})
}

// SelectModelForIterationForTest is an exported wrapper for testing
func SelectModelForIterationForTest(config AgentLoopConfig, balls []*session.Ball, defaultSessionModel session.ModelSize) *ModelSelection {
	return selectModelForIteration(config, balls, defaultSessionModel)
}

// PrioritizeBallsByModelForTest is an exported wrapper for testing
func PrioritizeBallsByModelForTest(balls []*session.Ball, currentModel string, sessionDefaultModel session.ModelSize) {
	prioritizeBallsByModel(balls, currentModel, sessionDefaultModel)
}

// FilterActiveBallsForTest is an exported wrapper for testing
func FilterActiveBallsForTest(balls []*session.Ball) []*session.Ball {
	return filterActiveBalls(balls)
}

// CountBallsByModelForTest is an exported wrapper for testing
func CountBallsByModelForTest(balls []*session.Ball) map[string]int {
	return countBallsByModel(balls)
}

// loadBallsForModelSelection loads balls for model selection purposes.
// This is similar to generateAgentPrompt but returns the balls instead of generating a prompt.
func loadBallsForModelSelection(projectDir, sessionID, ballID string) ([]*session.Ball, error) {
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

	// Discover projects
	projects, err := DiscoverProjectsForCommand(config, store)
	if err != nil {
		return nil, fmt.Errorf("failed to discover projects: %w", err)
	}

	if len(projects) == 0 {
		return nil, fmt.Errorf("no projects with .juggle directories found")
	}

	// Load all balls from discovered projects
	allBalls, err := session.LoadAllBalls(projects)
	if err != nil {
		return nil, fmt.Errorf("failed to load balls: %w", err)
	}

	// Filter by session tag
	// "all" is a special meta-session that means "all balls in repo" (no session filtering)
	var balls []*session.Ball
	if sessionID == "all" {
		balls = allBalls
	} else {
		balls = make([]*session.Ball, 0)
		for _, ball := range allBalls {
			for _, tag := range ball.Tags {
				if tag == sessionID {
					balls = append(balls, ball)
					break
				}
			}
		}
	}

	// Filter out complete and blocked balls by default (they clutter the context for no gain)
	// Exception: when a specific ball is requested, allow it even if complete/blocked
	if ballID == "" {
		filteredBalls := make([]*session.Ball, 0, len(balls))
		for _, ball := range balls {
			if ball.State != session.StateComplete && ball.State != session.StateResearched && ball.State != session.StateBlocked {
				filteredBalls = append(filteredBalls, ball)
			}
		}
		balls = filteredBalls
	}

	// Filter to specific ball if ballID is specified
	if ballID != "" {
		matches := session.ResolveBallByPrefix(balls, ballID)
		if len(matches) == 0 {
			return nil, fmt.Errorf("ball %s not found in session %s", ballID, sessionID)
		}
		if len(matches) > 1 {
			matchingIDs := make([]string, len(matches))
			for i, m := range matches {
				matchingIDs[i] = m.ID
			}
			return nil, fmt.Errorf("ambiguous ID '%s' matches %d balls: %s", ballID, len(matches), strings.Join(matchingIDs, ", "))
		}
		return []*session.Ball{matches[0]}, nil
	}

	return balls, nil
}

// LoadBallsForModelSelectionForTest is an exported wrapper for testing
func LoadBallsForModelSelectionForTest(projectDir, sessionID, ballID string) ([]*session.Ball, error) {
	return loadBallsForModelSelection(projectDir, sessionID, ballID)
}

// CommitResult represents the outcome of a VCS commit operation
type CommitResult struct {
	Success       bool   // Whether the commit succeeded
	CommitHash    string // Short hash of the new commit (if successful)
	StatusOutput  string // Output from status after commit
	ErrorMessage  string // Error message if commit failed
}

// performVCSCommit executes a commit using the configured VCS backend.
// This is called by juggle after the agent signals completion.
// Returns nil if there are no changes to commit.
func performVCSCommit(projectDir, commitMessage string) (*CommitResult, error) {
	// Load VCS settings
	globalVCS, _ := session.GetGlobalVCSWithOptions(GetConfigOptions())
	projectVCS, _ := session.GetProjectVCS(projectDir)

	// Get the appropriate backend
	backend := vcs.GetBackendForProject(projectDir, vcs.VCSType(projectVCS), vcs.VCSType(globalVCS))

	// Perform commit
	vcsResult, err := backend.Commit(projectDir, commitMessage)
	if err != nil {
		return nil, err
	}

	// Convert to our CommitResult type
	return &CommitResult{
		Success:      vcsResult.Success,
		CommitHash:   vcsResult.CommitHash,
		StatusOutput: vcsResult.StatusOutput,
		ErrorMessage: vcsResult.ErrorMessage,
	}, nil
}

// performJJCommit is kept for backward compatibility - delegates to performVCSCommit
func performJJCommit(projectDir, commitMessage string) (*CommitResult, error) {
	return performVCSCommit(projectDir, commitMessage)
}

// PerformJJCommitForTest is an exported wrapper for testing
func PerformJJCommitForTest(projectDir, commitMessage string) (*CommitResult, error) {
	return performVCSCommit(projectDir, commitMessage)
}
