package integration_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/ohare93/juggle/internal/cli"
	"github.com/ohare93/juggle/internal/session"
)

// MockGhRunner is a mock implementation of GhRunner for testing
type MockGhRunner struct {
	Output []byte
	Err    error
	Args   []string
}

func (m *MockGhRunner) Run(args ...string) ([]byte, error) {
	m.Args = args
	return m.Output, m.Err
}

// TestImportGitHubBasic tests basic GitHub issue import
func TestImportGitHubBasic(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create mock issues
	issues := []cli.GitHubIssue{
		{
			Number: 1,
			Title:  "Fix login bug",
			Body:   "1. Fix the login button\n2. Add validation",
			State:  "OPEN",
			Labels: []cli.GitHubLabel{
				{Name: "bug"},
				{Name: "priority:high"},
			},
		},
		{
			Number: 2,
			Title:  "Add dark mode",
			Body:   "- Add theme toggle\n- Store preference",
			State:  "OPEN",
			Labels: []cli.GitHubLabel{
				{Name: "enhancement"},
			},
		},
	}

	issuesJSON, _ := json.Marshal(issues)

	// Install mock runner
	originalRunner := cli.GhRunnerInstance
	mockRunner := &MockGhRunner{Output: issuesJSON}
	cli.GhRunnerInstance = mockRunner
	defer func() { cli.GhRunnerInstance = originalRunner }()

	// Run import
	err := cli.ImportGitHubIssues(issues, env.ProjectDir, "")
	if err != nil {
		t.Fatalf("ImportGitHubIssues failed: %v", err)
	}

	// Verify balls were created
	store := env.GetStore(t)
	balls, err := store.LoadBalls()
	if err != nil {
		t.Fatalf("Failed to load balls: %v", err)
	}

	if len(balls) != 2 {
		t.Errorf("Expected 2 balls, got %d", len(balls))
	}

	// Find and verify first ball
	var loginBall, darkModeBall *session.Ball
	for _, b := range balls {
		if b.Title == "Fix login bug" {
			loginBall = b
		} else if b.Title == "Add dark mode" {
			darkModeBall = b
		}
	}

	if loginBall == nil {
		t.Fatal("Login bug ball not found")
	}
	if darkModeBall == nil {
		t.Fatal("Dark mode ball not found")
	}

	// Verify login ball properties
	if loginBall.State != session.StatePending {
		t.Errorf("Expected pending state, got %s", loginBall.State)
	}
	if len(loginBall.AcceptanceCriteria) != 2 {
		t.Errorf("Expected 2 acceptance criteria, got %d", len(loginBall.AcceptanceCriteria))
	}
	if len(loginBall.AcceptanceCriteria) > 0 && loginBall.AcceptanceCriteria[0] != "Fix the login button" {
		t.Errorf("Expected 'Fix the login button', got '%s'", loginBall.AcceptanceCriteria[0])
	}

	// Verify tags include gh# prefix and labels
	hasGhTag := false
	hasBugTag := false
	for _, tag := range loginBall.Tags {
		if tag == "gh#1" {
			hasGhTag = true
		}
		if tag == "bug" {
			hasBugTag = true
		}
	}
	if !hasGhTag {
		t.Error("Expected gh#1 tag, not found")
	}
	if !hasBugTag {
		t.Error("Expected bug tag, not found")
	}
}

// TestImportGitHubClosedIssues tests that closed issues are marked complete
func TestImportGitHubClosedIssues(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	issues := []cli.GitHubIssue{
		{
			Number: 10,
			Title:  "Old bug fix",
			Body:   "This was fixed",
			State:  "CLOSED",
			Labels: []cli.GitHubLabel{},
		},
	}

	err := cli.ImportGitHubIssues(issues, env.ProjectDir, "")
	if err != nil {
		t.Fatalf("ImportGitHubIssues failed: %v", err)
	}

	store := env.GetStore(t)
	balls, _ := store.LoadBalls()

	if len(balls) != 1 {
		t.Fatalf("Expected 1 ball, got %d", len(balls))
	}

	if balls[0].State != session.StateComplete {
		t.Errorf("Expected complete state for closed issue, got %s", balls[0].State)
	}
	if balls[0].CompletedAt == nil {
		t.Error("Expected CompletedAt to be set for closed issue")
	}
}

// TestImportGitHubWithSession tests that session tag is added when specified
func TestImportGitHubWithSession(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create a session first
	sessionStore, err := session.NewSessionStore(env.ProjectDir)
	if err != nil {
		t.Fatalf("Failed to create session store: %v", err)
	}
	if _, err := sessionStore.CreateSession("my-feature", "Test feature session"); err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	issues := []cli.GitHubIssue{
		{
			Number: 5,
			Title:  "Feature task",
			Body:   "Do something",
			State:  "OPEN",
			Labels: []cli.GitHubLabel{},
		},
	}

	err = cli.ImportGitHubIssues(issues, env.ProjectDir, "my-feature")
	if err != nil {
		t.Fatalf("ImportGitHubIssues failed: %v", err)
	}

	store := env.GetStore(t)
	balls, _ := store.LoadBalls()

	if len(balls) != 1 {
		t.Fatalf("Expected 1 ball, got %d", len(balls))
	}

	hasSessionTag := false
	for _, tag := range balls[0].Tags {
		if tag == "my-feature" {
			hasSessionTag = true
			break
		}
	}
	if !hasSessionTag {
		t.Errorf("Expected session tag 'my-feature', tags: %v", balls[0].Tags)
	}
}

// TestImportGitHubSkipsDuplicates tests that existing balls are not re-imported
func TestImportGitHubSkipsDuplicates(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create existing ball
	env.CreateBall(t, "Existing bug", session.PriorityMedium)

	issues := []cli.GitHubIssue{
		{
			Number: 1,
			Title:  "Existing bug", // Same title as existing ball
			Body:   "Should be skipped",
			State:  "OPEN",
			Labels: []cli.GitHubLabel{},
		},
		{
			Number: 2,
			Title:  "New bug", // Different title
			Body:   "Should be imported",
			State:  "OPEN",
			Labels: []cli.GitHubLabel{},
		},
	}

	err := cli.ImportGitHubIssues(issues, env.ProjectDir, "")
	if err != nil {
		t.Fatalf("ImportGitHubIssues failed: %v", err)
	}

	store := env.GetStore(t)
	balls, _ := store.LoadBalls()

	// Should have 2 balls: existing one + new one
	if len(balls) != 2 {
		t.Errorf("Expected 2 balls (1 existing + 1 new), got %d", len(balls))
	}

	// Verify new ball was imported
	found := false
	for _, b := range balls {
		if b.Title == "New bug" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected 'New bug' ball to be imported")
	}
}

// TestParseAcceptanceCriteriaNumberedList tests parsing numbered lists
func TestParseAcceptanceCriteriaNumberedList(t *testing.T) {
	body := `1. First item
2. Second item
3. Third item`

	criteria := cli.ParseAcceptanceCriteria(body)

	if len(criteria) != 3 {
		t.Fatalf("Expected 3 criteria, got %d", len(criteria))
	}
	if criteria[0] != "First item" {
		t.Errorf("Expected 'First item', got '%s'", criteria[0])
	}
	if criteria[1] != "Second item" {
		t.Errorf("Expected 'Second item', got '%s'", criteria[1])
	}
}

// TestParseAcceptanceCriteriaCheckboxList tests parsing checkbox lists
func TestParseAcceptanceCriteriaCheckboxList(t *testing.T) {
	body := `- [ ] Unchecked item
- [x] Checked item
- [X] Also checked`

	criteria := cli.ParseAcceptanceCriteria(body)

	if len(criteria) != 3 {
		t.Fatalf("Expected 3 criteria, got %d", len(criteria))
	}
	if criteria[0] != "Unchecked item" {
		t.Errorf("Expected 'Unchecked item', got '%s'", criteria[0])
	}
	if criteria[1] != "Checked item" {
		t.Errorf("Expected 'Checked item', got '%s'", criteria[1])
	}
}

// TestParseAcceptanceCriteriaBulletList tests parsing bullet lists
func TestParseAcceptanceCriteriaBulletList(t *testing.T) {
	body := `- First bullet
* Second bullet
- Third bullet`

	criteria := cli.ParseAcceptanceCriteria(body)

	if len(criteria) != 3 {
		t.Fatalf("Expected 3 criteria, got %d", len(criteria))
	}
	if criteria[0] != "First bullet" {
		t.Errorf("Expected 'First bullet', got '%s'", criteria[0])
	}
}

// TestParseAcceptanceCriteriaPlainText tests fallback to full body text
func TestParseAcceptanceCriteriaPlainText(t *testing.T) {
	body := `This is just plain text without any list formatting.
It should be used as a single criterion.`

	criteria := cli.ParseAcceptanceCriteria(body)

	if len(criteria) != 1 {
		t.Fatalf("Expected 1 criterion, got %d", len(criteria))
	}
	if criteria[0] != body {
		t.Errorf("Expected full body as criterion")
	}
}

// TestParseAcceptanceCriteriaEmpty tests empty body handling
func TestParseAcceptanceCriteriaEmpty(t *testing.T) {
	criteria := cli.ParseAcceptanceCriteria("")

	if criteria != nil {
		t.Errorf("Expected nil for empty body, got %v", criteria)
	}
}

// TestParseAcceptanceCriteriaTruncation tests long body truncation
func TestParseAcceptanceCriteriaTruncation(t *testing.T) {
	// Create a body longer than 500 characters
	longBody := ""
	for i := 0; i < 100; i++ {
		longBody += "This is a long line of text. "
	}

	criteria := cli.ParseAcceptanceCriteria(longBody)

	if len(criteria) != 1 {
		t.Fatalf("Expected 1 criterion, got %d", len(criteria))
	}
	if len(criteria[0]) != 500 {
		t.Errorf("Expected truncated body of 500 chars, got %d", len(criteria[0]))
	}
	if criteria[0][len(criteria[0])-3:] != "..." {
		t.Errorf("Expected truncated body to end with '...', got '%s'", criteria[0][len(criteria[0])-3:])
	}
}

// TestFetchGitHubIssuesWithFilters tests that filters are passed to gh CLI
func TestFetchGitHubIssuesWithFilters(t *testing.T) {
	// Save original values
	originalRunner := cli.GhRunnerInstance
	defer func() { cli.GhRunnerInstance = originalRunner }()

	tests := []struct {
		name       string
		milestone  string
		label      string
		state      string
		limit      int
		expectArgs []string
	}{
		{
			name:  "with milestone filter",
			milestone: "v1.0",
			state: "open",
			limit: 100,
			expectArgs: []string{"--milestone", "v1.0"},
		},
		{
			name:  "with label filter",
			label: "bug",
			state: "open",
			limit: 100,
			expectArgs: []string{"--label", "bug"},
		},
		{
			name:  "with both filters",
			milestone: "v2.0",
			label: "enhancement",
			state: "closed",
			limit: 50,
			expectArgs: []string{"--milestone", "v2.0", "--label", "enhancement"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRunner := &MockGhRunner{Output: []byte("[]")}
			cli.GhRunnerInstance = mockRunner

			// Set global filter variables (these would normally be set by cobra flags)
			// We need to call FetchGitHubIssues directly with our test values
			issues, err := cli.FetchGitHubIssuesWithOptions("owner/repo", tt.milestone, tt.label, tt.state, tt.limit)
			if err != nil {
				t.Fatalf("FetchGitHubIssues failed: %v", err)
			}

			if len(issues) != 0 {
				t.Errorf("Expected 0 issues from empty response, got %d", len(issues))
			}

			// Verify expected args are in the command
			for _, expectedArg := range tt.expectArgs {
				found := false
				for _, arg := range mockRunner.Args {
					if arg == expectedArg {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected arg '%s' not found in: %v", expectedArg, mockRunner.Args)
				}
			}
		})
	}
}

// TestImportGitHubMultipleLabels tests that multiple labels are all added as tags
func TestImportGitHubMultipleLabels(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	issues := []cli.GitHubIssue{
		{
			Number: 42,
			Title:  "Multi-label issue",
			Body:   "Test",
			State:  "OPEN",
			Labels: []cli.GitHubLabel{
				{Name: "bug"},
				{Name: "urgent"},
				{Name: "frontend"},
				{Name: "needs-review"},
			},
		},
	}

	err := cli.ImportGitHubIssues(issues, env.ProjectDir, "")
	if err != nil {
		t.Fatalf("ImportGitHubIssues failed: %v", err)
	}

	store := env.GetStore(t)
	balls, _ := store.LoadBalls()

	if len(balls) != 1 {
		t.Fatalf("Expected 1 ball, got %d", len(balls))
	}

	expectedTags := map[string]bool{
		"gh#42":        true,
		"bug":          true,
		"urgent":       true,
		"frontend":     true,
		"needs-review": true,
	}

	for _, tag := range balls[0].Tags {
		delete(expectedTags, tag)
	}

	if len(expectedTags) > 0 {
		missing := []string{}
		for tag := range expectedTags {
			missing = append(missing, tag)
		}
		t.Errorf("Missing expected tags: %v", missing)
	}
}

// TestImportGitHubLowerCaseState tests that lowercase state values are handled
func TestImportGitHubLowerCaseState(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	issues := []cli.GitHubIssue{
		{
			Number: 1,
			Title:  "Lowercase closed",
			Body:   "Test",
			State:  "closed", // lowercase
			Labels: []cli.GitHubLabel{},
		},
	}

	err := cli.ImportGitHubIssues(issues, env.ProjectDir, "")
	if err != nil {
		t.Fatalf("ImportGitHubIssues failed: %v", err)
	}

	store := env.GetStore(t)
	balls, _ := store.LoadBalls()

	if len(balls) != 1 {
		t.Fatalf("Expected 1 ball, got %d", len(balls))
	}

	if balls[0].State != session.StateComplete {
		t.Errorf("Expected complete state for lowercase 'closed', got %s", balls[0].State)
	}
}

// TestGhRunnerError tests error handling when gh CLI fails
func TestGhRunnerError(t *testing.T) {
	originalRunner := cli.GhRunnerInstance
	defer func() { cli.GhRunnerInstance = originalRunner }()

	mockRunner := &MockGhRunner{
		Err: fmt.Errorf("gh command not found"),
	}
	cli.GhRunnerInstance = mockRunner

	_, err := cli.FetchGitHubIssuesWithOptions("owner/repo", "", "", "open", 100)
	if err == nil {
		t.Error("Expected error when gh CLI fails")
	}
}

// TestParseAcceptanceCriteriaMixedList tests parsing mixed list types
func TestParseAcceptanceCriteriaMixedList(t *testing.T) {
	body := `1. First numbered
- First bullet
- [ ] First checkbox`

	criteria := cli.ParseAcceptanceCriteria(body)

	if len(criteria) != 3 {
		t.Fatalf("Expected 3 criteria, got %d: %v", len(criteria), criteria)
	}
}

// TestParseAcceptanceCriteriaIndentedList tests parsing indented lists
func TestParseAcceptanceCriteriaIndentedList(t *testing.T) {
	body := `Requirements:
  1. First indented item
  2. Second indented item
  - Indented bullet`

	criteria := cli.ParseAcceptanceCriteria(body)

	if len(criteria) != 3 {
		t.Fatalf("Expected 3 criteria from indented list, got %d: %v", len(criteria), criteria)
	}
	if criteria[0] != "First indented item" {
		t.Errorf("Expected 'First indented item', got '%s'", criteria[0])
	}
}
