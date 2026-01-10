package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ohare93/juggle/internal/session"
	"github.com/spf13/cobra"
)

var (
	agentIterations int
	agentTrust      bool
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

Examples:
  # Run agent for 10 iterations (default)
  juggle agent run my-feature

  # Run for specific number of iterations
  juggle agent run my-feature --iterations 5

  # Run with full permissions (dangerous)
  juggle agent run my-feature --trust`,
	Args: cobra.ExactArgs(1),
	RunE: runAgentRun,
}

func init() {
	agentRunCmd.Flags().IntVarP(&agentIterations, "iterations", "n", 10, "Maximum number of iterations")
	agentRunCmd.Flags().BoolVar(&agentTrust, "trust", false, "Run with --dangerously-skip-permissions (dangerous!)")

	agentCmd.AddCommand(agentRunCmd)
	rootCmd.AddCommand(agentCmd)
}

// AgentResult holds the result of an agent run
type AgentResult struct {
	Iterations    int       `json:"iterations"`
	Complete      bool      `json:"complete"`
	Blocked       bool      `json:"blocked"`
	BlockedReason string    `json:"blocked_reason,omitempty"`
	BallsComplete int       `json:"balls_complete"`
	BallsTotal    int       `json:"balls_total"`
	StartedAt     time.Time `json:"started_at"`
	EndedAt       time.Time `json:"ended_at"`
}

func runAgentRun(cmd *cobra.Command, args []string) error {
	sessionID := args[0]
	startTime := time.Now()

	// Get current directory
	cwd, err := GetWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Verify session exists
	sessionStore, err := session.NewSessionStore(cwd)
	if err != nil {
		return fmt.Errorf("failed to create session store: %w", err)
	}

	if _, err := sessionStore.LoadSession(sessionID); err != nil {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	// Create output file path
	outputPath := filepath.Join(cwd, ".juggler", "sessions", sessionID, "last_output.txt")

	// Print warning if --trust is used
	if agentTrust {
		fmt.Println("⚠️  WARNING: Running with --trust flag. Agent has full system permissions.")
		fmt.Println("    Only use this if you trust the agent and understand the risks.")
		fmt.Println()
	}

	fmt.Printf("Starting agent for session: %s\n", sessionID)
	fmt.Printf("Max iterations: %d\n", agentIterations)
	fmt.Println()

	result := AgentResult{
		StartedAt: startTime,
	}

	for iteration := 1; iteration <= agentIterations; iteration++ {
		fmt.Printf("=== Iteration %d/%d ===\n", iteration, agentIterations)
		result.Iterations = iteration

		// Generate prompt using export command
		prompt, err := generateAgentPrompt(cwd, sessionID)
		if err != nil {
			return fmt.Errorf("failed to generate prompt: %w", err)
		}

		// Run claude with prompt
		output, err := runClaude(prompt, agentTrust)
		if err != nil {
			fmt.Printf("Error running claude: %v\n", err)
			// Don't fail immediately, save output and continue
		}

		// Save output to file
		if err := os.WriteFile(outputPath, []byte(output), 0644); err != nil {
			fmt.Printf("Warning: failed to save output: %v\n", err)
		}

		// Check for completion signals
		if strings.Contains(output, "<promise>COMPLETE</promise>") {
			fmt.Println("\n✓ Agent signaled COMPLETE")
			result.Complete = true
			break
		}

		if idx := strings.Index(output, "<promise>BLOCKED:"); idx != -1 {
			// Extract blocked reason
			endIdx := strings.Index(output[idx:], "</promise>")
			if endIdx != -1 {
				reason := strings.TrimSpace(output[idx+len("<promise>BLOCKED:") : idx+endIdx])
				fmt.Printf("\n✗ Agent signaled BLOCKED: %s\n", reason)
				result.Blocked = true
				result.BlockedReason = reason
				break
			}
		}

		// Check if all balls are complete
		complete, total := checkBallsComplete(cwd, sessionID)
		result.BallsComplete = complete
		result.BallsTotal = total

		if total > 0 && complete == total {
			fmt.Printf("\n✓ All %d balls complete\n", total)
			result.Complete = true
			break
		}

		fmt.Printf("Progress: %d/%d balls complete\n", complete, total)

		// Delay before next iteration (unless this was the last one)
		if iteration < agentIterations {
			fmt.Println("Waiting 2 seconds before next iteration...")
			time.Sleep(2 * time.Second)
		}
	}

	result.EndedAt = time.Now()
	elapsed := result.EndedAt.Sub(result.StartedAt)

	// Print summary
	fmt.Println()
	fmt.Println("=== Summary ===")
	fmt.Printf("Iterations: %d\n", result.Iterations)
	fmt.Printf("Balls completed: %d/%d\n", result.BallsComplete, result.BallsTotal)
	fmt.Printf("Time elapsed: %s\n", elapsed.Round(time.Second))

	if result.Complete {
		fmt.Println("Status: COMPLETE")
	} else if result.Blocked {
		fmt.Printf("Status: BLOCKED (%s)\n", result.BlockedReason)
	} else {
		fmt.Println("Status: Max iterations reached")
	}

	fmt.Printf("\nOutput saved to: %s\n", outputPath)

	return nil
}

// generateAgentPrompt generates the agent prompt using export command
func generateAgentPrompt(projectDir, sessionID string) (string, error) {
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
	balls := make([]*session.Session, 0)
	for _, ball := range allBalls {
		for _, tag := range ball.Tags {
			if tag == sessionID {
				balls = append(balls, ball)
				break
			}
		}
	}

	// Call exportAgent directly
	output, err := exportAgent(projectDir, sessionID, balls)
	if err != nil {
		return "", err
	}

	return string(output), nil
}

// autonomousSystemPrompt is appended to force autonomous operation
const autonomousSystemPrompt = `CRITICAL: You are an autonomous agent. DO NOT ask questions. DO NOT summarize. DO NOT wait for confirmation. START WORKING IMMEDIATELY. Execute the workflow in prompt.md without any preamble.`

// runClaude runs claude with the given prompt
func runClaude(prompt string, trust bool) (string, error) {
	// Build command arguments
	// Start with autonomous mode flags
	args := []string{
		"--disable-slash-commands",   // Disable skills
		"--setting-sources", "",      // Skip CLAUDE.md loading
		"--append-system-prompt", autonomousSystemPrompt, // Autonomous instructions
	}

	if trust {
		args = append(args, "--dangerously-skip-permissions")
	} else {
		// Default: accept edits permission mode
		args = append(args, "--permission-mode", "acceptEdits")
	}

	// Add prompt input flag
	args = append(args, "-p", "-")

	cmd := exec.Command("claude", args...)

	// Set up stdin with prompt
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	// Set up combined stdout/stderr capture
	var outputBuf strings.Builder

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start command
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start claude: %w", err)
	}

	// Write prompt to stdin
	go func() {
		defer stdin.Close()
		io.WriteString(stdin, prompt)
	}()

	// Stream output to console and capture
	go streamOutput(stdout, &outputBuf, os.Stdout)
	go streamOutput(stderr, &outputBuf, os.Stderr)

	// Wait for command to complete
	if err := cmd.Wait(); err != nil {
		// Return output even on error (agent might have signaled BLOCKED)
		return outputBuf.String(), fmt.Errorf("claude exited with error: %w", err)
	}

	return outputBuf.String(), nil
}

// streamOutput reads from reader and writes to both buffer and writer
func streamOutput(reader io.Reader, buf *strings.Builder, writer io.Writer) {
	scanner := bufio.NewScanner(reader)
	// Increase scanner buffer for long lines
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		buf.WriteString(line)
		buf.WriteString("\n")
		fmt.Fprintln(writer, line)
	}
}

// checkBallsComplete returns count of complete balls and total balls for session
func checkBallsComplete(projectDir, sessionID string) (complete, total int) {
	// Load config
	config, err := LoadConfigForCommand()
	if err != nil {
		return 0, 0
	}

	// Create store
	store, err := NewStoreForCommand(projectDir)
	if err != nil {
		return 0, 0
	}

	// Discover projects
	projects, err := DiscoverProjectsForCommand(config, store)
	if err != nil {
		return 0, 0
	}

	// Load all balls
	allBalls, err := session.LoadAllBalls(projects)
	if err != nil {
		return 0, 0
	}

	// Count balls with session tag
	for _, ball := range allBalls {
		for _, tag := range ball.Tags {
			if tag == sessionID {
				total++
				if ball.State == session.StateComplete {
					complete++
				}
				break
			}
		}
	}

	return complete, total
}
