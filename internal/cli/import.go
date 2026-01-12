package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/ohare93/juggle/internal/session"
	"github.com/spf13/cobra"
)

// Regex patterns for parsing acceptance criteria from issue bodies
// Support optional leading whitespace for indented lists
var (
	numberedListRegex = regexp.MustCompile(`^\s*\d+\.\s+(.+)$`)
	checkboxRegex     = regexp.MustCompile(`^\s*-\s*\[[xX ]\]\s+(.+)$`)
	bulletRegex       = regexp.MustCompile(`^\s*[-*]\s+(.+)$`)
)

var (
	importSessionID        string
	importGitHubMilestone  string
	importGitHubLabel      string
	importGitHubState      string
	importGitHubLimit      int
)

// importCmd is the parent command for import operations
var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import external data into juggler",
	Long:  `Import external data (like prd.json from ralph) into juggler balls.`,
}

// importRalphCmd imports prd.json user stories as balls
var importRalphCmd = &cobra.Command{
	Use:   "ralph <prd.json>",
	Short: "Import prd.json user stories as balls",
	Long: `Import user stories from a prd.json file as juggler balls.

Creates balls from user stories with the following mappings:
  - title           → intent
  - acceptanceCriteria → acceptance_criteria
  - priority 1-2    → urgent
  - priority 3-5    → high
  - priority 6-10   → medium
  - priority 11+    → low
  - passes: true    → state: complete
  - passes: false   → state: pending

Skips stories that already exist (matching by title/intent).

Examples:
  # Import from prd.json and tag with session
  juggle import ralph prd.json --session my-feature

  # Import from .agent/prd.json
  juggle import ralph .agent/prd.json`,
	Args: cobra.ExactArgs(1),
	RunE: runImportRalph,
}

// importGitHubCmd imports GitHub issues as balls
var importGitHubCmd = &cobra.Command{
	Use:   "github <owner/repo>",
	Short: "Import GitHub issues as balls",
	Long: `Import issues from a GitHub repository as juggler balls.

Creates balls from issues with the following mappings:
  - issue title     → intent
  - issue body      → acceptance criteria (parsed from numbered list or body lines)
  - issue labels    → tags
  - state: open     → state: pending
  - state: closed   → state: complete

Requires the GitHub CLI (gh) to be installed and authenticated.

Supports filtering:
  - --milestone    Filter by milestone title
  - --label        Filter by label name
  - --state        Filter by state (open, closed, all) - default: open
  - --limit        Maximum number of issues to import (default: 100)

Skips issues that already exist (matching by title/intent).

Examples:
  # Import open issues from a repository
  juggle import github owner/repo

  # Import issues from a specific milestone
  juggle import github owner/repo --milestone "v1.0"

  # Import issues with a specific label
  juggle import github owner/repo --label "bug"

  # Import all issues (including closed) and tag with session
  juggle import github owner/repo --state all --session my-project`,
	Args: cobra.ExactArgs(1),
	RunE: runImportGitHub,
}

func init() {
	importRalphCmd.Flags().StringVarP(&importSessionID, "session", "s", "", "Session ID to tag imported balls with")

	importGitHubCmd.Flags().StringVarP(&importSessionID, "session", "s", "", "Session ID to tag imported balls with")
	importGitHubCmd.Flags().StringVar(&importGitHubMilestone, "milestone", "", "Filter by milestone title")
	importGitHubCmd.Flags().StringVar(&importGitHubLabel, "label", "", "Filter by label name")
	importGitHubCmd.Flags().StringVar(&importGitHubState, "state", "open", "Filter by state (open, closed, all)")
	importGitHubCmd.Flags().IntVar(&importGitHubLimit, "limit", 100, "Maximum number of issues to import")

	importCmd.AddCommand(importRalphCmd)
	importCmd.AddCommand(importGitHubCmd)
	rootCmd.AddCommand(importCmd)
}

func runImportRalph(cmd *cobra.Command, args []string) error {
	prdPath := args[0]

	// Get current directory
	cwd, err := GetWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Make path absolute if relative
	if !filepath.IsAbs(prdPath) {
		prdPath = filepath.Join(cwd, prdPath)
	}

	// Check if file exists
	if _, err := os.Stat(prdPath); os.IsNotExist(err) {
		return fmt.Errorf("prd.json not found: %s", prdPath)
	}

	// Validate session exists if specified
	if importSessionID != "" {
		sessionStore, err := session.NewSessionStore(cwd)
		if err != nil {
			return fmt.Errorf("failed to create session store: %w", err)
		}
		if _, err := sessionStore.LoadSession(importSessionID); err != nil {
			return fmt.Errorf("session not found: %s", importSessionID)
		}
	}

	return importPRDFile(prdPath, cwd, importSessionID)
}

// importPRDFile reads prd.json and creates balls from user stories
func importPRDFile(prdPath, projectDir, sessionID string) error {
	// Read prd.json
	data, err := os.ReadFile(prdPath)
	if err != nil {
		return fmt.Errorf("failed to read prd.json: %w", err)
	}

	var prd PRDFile
	if err := json.Unmarshal(data, &prd); err != nil {
		return fmt.Errorf("failed to parse prd.json: %w", err)
	}

	// Create store for project
	store, err := NewStoreForCommand(projectDir)
	if err != nil {
		return fmt.Errorf("failed to create store: %w", err)
	}

	// Load existing balls
	balls, err := store.LoadBalls()
	if err != nil {
		return fmt.Errorf("failed to load balls: %w", err)
	}

	// Build lookup by title (intent) to check for existing balls
	existingTitles := make(map[string]bool)
	for _, ball := range balls {
		existingTitles[ball.Title] = true
	}

	var imported, skipped int

	for _, story := range prd.UserStories {
		// Check if ball already exists (match by title)
		if existingTitles[story.Title] {
			fmt.Printf("Skipped: %s - \"%s\" (already exists)\n", story.ID, story.Title)
			skipped++
			continue
		}

		// Create new ball
		ball, err := session.NewBall(projectDir, story.Title, mapPriorityNumber(story.Priority))
		if err != nil {
			fmt.Printf("Warning: failed to create ball for %s: %v\n", story.ID, err)
			continue
		}

		// Set acceptance criteria
		if len(story.AcceptanceCriteria) > 0 {
			ball.SetAcceptanceCriteria(story.AcceptanceCriteria)
		}

		// Set state based on passes
		if story.Passes {
			ball.State = session.StateComplete
			now := time.Now()
			ball.CompletedAt = &now
		} else {
			ball.State = session.StatePending
		}

		// Add story ID as tag for reference
		ball.AddTag(story.ID)

		// Add session tag if specified
		if sessionID != "" {
			ball.AddTag(sessionID)
		}

		if err := store.AppendBall(ball); err != nil {
			fmt.Printf("Warning: failed to create ball for %s: %v\n", story.ID, err)
			continue
		}
		imported++
		fmt.Printf("Imported: %s → %s (%s)\n", story.ID, ball.ID, ball.State)

		// Add to lookup for subsequent stories
		existingTitles[story.Title] = true
	}

	fmt.Printf("\nImport complete: %d imported, %d skipped\n", imported, skipped)
	return nil
}

// GitHubIssue represents an issue from the GitHub API
type GitHubIssue struct {
	Number    int           `json:"number"`
	Title     string        `json:"title"`
	Body      string        `json:"body"`
	State     string        `json:"state"`
	Labels    []GitHubLabel `json:"labels"`
	Milestone *struct {
		Title string `json:"title"`
	} `json:"milestone"`
}

// GitHubLabel represents a label from the GitHub API
type GitHubLabel struct {
	Name string `json:"name"`
}

// GhRunner defines the interface for running gh CLI commands
type GhRunner interface {
	Run(args ...string) ([]byte, error)
}

// DefaultGhRunner is the default implementation using exec.Command
type DefaultGhRunner struct{}

// Run executes a gh command and returns the output
func (r *DefaultGhRunner) Run(args ...string) ([]byte, error) {
	cmd := exec.Command("gh", args...)
	return cmd.Output()
}

// GhRunnerInstance is the global GhRunner used for testing
var GhRunnerInstance GhRunner = &DefaultGhRunner{}

func runImportGitHub(cmd *cobra.Command, args []string) error {
	repo := args[0]

	// Validate repo format (owner/repo)
	if !strings.Contains(repo, "/") || strings.Count(repo, "/") != 1 {
		return fmt.Errorf("invalid repository format: %s (expected: owner/repo)", repo)
	}
	parts := strings.Split(repo, "/")
	if parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("invalid repository format: %s (owner and repo cannot be empty)", repo)
	}

	// Validate state filter
	if importGitHubState != "open" && importGitHubState != "closed" && importGitHubState != "all" {
		return fmt.Errorf("invalid state: %s (must be open, closed, or all)", importGitHubState)
	}

	// Get current directory
	cwd, err := GetWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Validate session exists if specified
	if importSessionID != "" {
		sessionStore, err := session.NewSessionStore(cwd)
		if err != nil {
			return fmt.Errorf("failed to create session store: %w", err)
		}
		if _, err := sessionStore.LoadSession(importSessionID); err != nil {
			return fmt.Errorf("session not found: %s", importSessionID)
		}
	}

	// Fetch issues from GitHub using gh CLI
	issues, err := fetchGitHubIssues(repo)
	if err != nil {
		return fmt.Errorf("failed to fetch issues: %w", err)
	}

	if len(issues) == 0 {
		fmt.Println("No issues found matching the criteria.")
		return nil
	}

	return ImportGitHubIssues(issues, cwd, importSessionID)
}

// fetchGitHubIssues fetches issues from GitHub using the gh CLI
func fetchGitHubIssues(repo string) ([]GitHubIssue, error) {
	// Build gh command arguments
	args := []string{
		"issue", "list",
		"--repo", repo,
		"--json", "number,title,body,state,labels,milestone",
		"--limit", fmt.Sprintf("%d", importGitHubLimit),
		"--state", importGitHubState,
	}

	// Add milestone filter if specified
	if importGitHubMilestone != "" {
		args = append(args, "--milestone", importGitHubMilestone)
	}

	// Add label filter if specified
	if importGitHubLabel != "" {
		args = append(args, "--label", importGitHubLabel)
	}

	// Execute gh command
	output, err := GhRunnerInstance.Run(args...)
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("gh command failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("gh command failed: %w (is gh CLI installed and authenticated?)", err)
	}

	// Parse JSON output
	var issues []GitHubIssue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("failed to parse gh output: %w", err)
	}

	return issues, nil
}

// FetchGitHubIssuesWithOptions fetches issues with explicit options (exported for testing)
func FetchGitHubIssuesWithOptions(repo, milestone, label, state string, limit int) ([]GitHubIssue, error) {
	// Build gh command arguments
	args := []string{
		"issue", "list",
		"--repo", repo,
		"--json", "number,title,body,state,labels,milestone",
		"--limit", fmt.Sprintf("%d", limit),
		"--state", state,
	}

	// Add milestone filter if specified
	if milestone != "" {
		args = append(args, "--milestone", milestone)
	}

	// Add label filter if specified
	if label != "" {
		args = append(args, "--label", label)
	}

	// Execute gh command
	output, err := GhRunnerInstance.Run(args...)
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("gh command failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("gh command failed: %w (is gh CLI installed and authenticated?)", err)
	}

	// Parse JSON output
	var issues []GitHubIssue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("failed to parse gh output: %w", err)
	}

	return issues, nil
}

// ImportGitHubIssues imports GitHub issues as balls (exported for testing)
func ImportGitHubIssues(issues []GitHubIssue, projectDir, sessionID string) error {
	// Create store for project
	store, err := NewStoreForCommand(projectDir)
	if err != nil {
		return fmt.Errorf("failed to create store: %w", err)
	}

	// Load existing balls
	balls, err := store.LoadBalls()
	if err != nil {
		return fmt.Errorf("failed to load balls: %w", err)
	}

	// Build lookup by title (intent) to check for existing balls
	existingTitles := make(map[string]bool)
	for _, ball := range balls {
		existingTitles[ball.Title] = true
	}

	var imported, skipped int

	for _, issue := range issues {
		// Check if ball already exists (match by title)
		if existingTitles[issue.Title] {
			fmt.Printf("Skipped: #%d - \"%s\" (already exists)\n", issue.Number, issue.Title)
			skipped++
			continue
		}

		// Create new ball (default to medium priority for imports)
		ball, err := session.NewBall(projectDir, issue.Title, session.PriorityMedium)
		if err != nil {
			fmt.Printf("Warning: failed to create ball for #%d: %v\n", issue.Number, err)
			continue
		}

		// Parse acceptance criteria from issue body
		criteria := ParseAcceptanceCriteria(issue.Body)
		if len(criteria) > 0 {
			ball.SetAcceptanceCriteria(criteria)
		}

		// Set state based on issue state (case-insensitive)
		if strings.EqualFold(issue.State, "closed") {
			ball.State = session.StateComplete
			now := time.Now()
			ball.CompletedAt = &now
		} else {
			ball.State = session.StatePending
		}

		// Add issue number as tag for reference
		ball.AddTag(fmt.Sprintf("gh#%d", issue.Number))

		// Add issue labels as tags
		for _, label := range issue.Labels {
			ball.AddTag(label.Name)
		}

		// Add session tag if specified
		if sessionID != "" {
			ball.AddTag(sessionID)
		}

		if err := store.AppendBall(ball); err != nil {
			fmt.Printf("Warning: failed to create ball for #%d: %v\n", issue.Number, err)
			continue
		}
		imported++
		fmt.Printf("Imported: #%d → %s (%s)\n", issue.Number, ball.ID, ball.State)

		// Add to lookup for subsequent issues
		existingTitles[issue.Title] = true
	}

	fmt.Printf("\nImport complete: %d imported, %d skipped\n", imported, skipped)
	return nil
}

// ParseAcceptanceCriteria extracts acceptance criteria from issue body (exported for testing)
// It looks for:
// 1. Numbered lists (1. item, 2. item, etc.)
// 2. Checkbox lists (- [ ] item, - [x] item)
// 3. Bullet lists (- item, * item)
// If none found, uses the entire body as a single criterion
func ParseAcceptanceCriteria(body string) []string {
	if body == "" {
		return nil
	}

	body = strings.TrimSpace(body)
	lines := strings.Split(body, "\n")
	var criteria []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check for numbered list (1. item, 2. item, etc.)
		if matched := numberedListRegex.FindStringSubmatch(line); len(matched) > 1 {
			criteria = append(criteria, strings.TrimSpace(matched[1]))
			continue
		}

		// Check for checkbox list (- [ ] item, - [x] item)
		if matched := checkboxRegex.FindStringSubmatch(line); len(matched) > 1 {
			criteria = append(criteria, strings.TrimSpace(matched[1]))
			continue
		}

		// Check for bullet list (- item, * item)
		if matched := bulletRegex.FindStringSubmatch(line); len(matched) > 1 {
			criteria = append(criteria, strings.TrimSpace(matched[1]))
			continue
		}
	}

	// If we found list items, return them
	if len(criteria) > 0 {
		return criteria
	}

	// Otherwise, use the entire body as the criterion (if not too long)
	if len(body) <= 500 {
		return []string{body}
	}

	// If body is too long, truncate and indicate continuation
	return []string{body[:497] + "..."}
}
