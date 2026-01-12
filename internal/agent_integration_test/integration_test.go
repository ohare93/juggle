//go:build integration
// +build integration

// Package agent_integration_test provides integration tests that run against
// the real Claude agent. These tests validate agent behavior in specific
// scenarios like blocked tools, disabled skills, and disabled CLAUDE.md.
//
// Run with: go test -tags=integration ./internal/agent_integration_test/...
//
// These tests require:
// - The `claude` CLI to be installed and available in PATH
// - Network access to Claude API
// - Valid Claude API credentials
//
// Due to the nature of these tests (calling real Claude API), they should
// only run on manual trigger or specific CI label to avoid excessive API costs.
package agent_integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestResult represents the expected JSON output from the agent
type TestResult struct {
	TestID     string      `json:"test_id"`
	Assertions []Assertion `json:"assertions"`
	Error      string      `json:"error,omitempty"`
}

// Assertion represents a single test assertion from the agent
type Assertion struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail,omitempty"`
}

// TestConfig configures an integration test run
type TestConfig struct {
	TestID      string
	Prompt      string
	Timeout     time.Duration
	WorkDir     string
	DisableSkills  bool
	DisableClaude  bool
	BlockedTools   []string
}

// Default timeout for agent tests
const defaultTimeout = 5 * time.Minute

// runAgentTest executes Claude with the given configuration and parses the result
func runAgentTest(t *testing.T, config TestConfig) *TestResult {
	t.Helper()

	if config.Timeout == 0 {
		config.Timeout = defaultTimeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

	// Build the prompt that requests JSON output
	prompt := buildTestPrompt(config)

	// Build command arguments
	args := []string{
		"--print",
		"--output-format", "text",
	}

	if config.DisableSkills {
		args = append(args, "--disable-slash-commands")
	}

	// Pass prompt via stdin for -p -
	args = append(args, "-p", "-")

	cmd := exec.CommandContext(ctx, "claude", args...)
	cmd.Dir = config.WorkDir

	// Set up environment
	env := os.Environ()
	if config.DisableClaude {
		// Disable reading CLAUDE.md by setting a non-existent path
		env = append(env, "CLAUDE_CONFIG_DIR=/nonexistent-claude-config-dir-for-test")
	}
	cmd.Env = env

	// Write prompt to stdin
	cmd.Stdin = strings.NewReader(prompt)

	// Capture output
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if it was a timeout
		if ctx.Err() == context.DeadlineExceeded {
			return &TestResult{
				TestID: config.TestID,
				Error:  fmt.Sprintf("test timed out after %v", config.Timeout),
			}
		}
		// Agent returned error but might still have valid output
		t.Logf("Agent exited with error: %v\nOutput: %s", err, string(output))
	}

	// Parse the JSON result from output
	result := parseTestResult(t, config.TestID, string(output))
	return result
}

// buildTestPrompt creates a prompt that instructs the agent to output JSON
func buildTestPrompt(config TestConfig) string {
	blockedToolsSection := ""
	if len(config.BlockedTools) > 0 {
		blockedToolsSection = fmt.Sprintf(`
IMPORTANT: The following tools are NOT available for this test: %s
If you attempt to use any of these tools, the test will fail.
`, strings.Join(config.BlockedTools, ", "))
	}

	return fmt.Sprintf(`You are running an integration test. Your task is to perform the test and output a JSON result.

TEST ID: %s

%s

%s

INSTRUCTIONS:
1. Perform the test scenario described above
2. Check all the assertions
3. Output ONLY a JSON object in the following format (no other text before or after):

{"test_id": "%s", "assertions": [{"name": "assertion_name", "passed": true/false, "detail": "optional details"}]}

If you encounter an error that prevents testing, output:
{"test_id": "%s", "error": "description of the error"}

IMPORTANT: Output ONLY the JSON object, nothing else. No markdown code blocks, no explanation text.
`, config.TestID, config.Prompt, blockedToolsSection, config.TestID, config.TestID)
}

// parseTestResult extracts the JSON test result from agent output
func parseTestResult(t *testing.T, testID, output string) *TestResult {
	t.Helper()

	// Find all potential JSON objects using proper brace matching
	// This handles nested objects correctly (unlike regex)
	jsonCandidates := extractJSONObjects(output)

	for _, candidate := range jsonCandidates {
		var result TestResult
		if err := json.Unmarshal([]byte(candidate), &result); err == nil {
			if result.TestID == testID {
				return &result
			}
		}
	}

	// If we couldn't parse JSON, create an error result
	return &TestResult{
		TestID: testID,
		Error:  fmt.Sprintf("could not parse JSON result from output:\n%s", truncateOutput(output, 500)),
	}
}

// extractJSONObjects finds all top-level JSON objects in a string by properly
// matching braces, handling nested objects correctly.
func extractJSONObjects(s string) []string {
	var results []string
	i := 0

	for i < len(s) {
		// Find the next opening brace
		start := strings.Index(s[i:], "{")
		if start == -1 {
			break
		}
		start += i

		// Track brace nesting to find the matching closing brace
		depth := 0
		inString := false
		escaped := false
		end := -1

		for j := start; j < len(s); j++ {
			c := s[j]

			if escaped {
				escaped = false
				continue
			}

			if c == '\\' && inString {
				escaped = true
				continue
			}

			if c == '"' {
				inString = !inString
				continue
			}

			if inString {
				continue
			}

			if c == '{' {
				depth++
			} else if c == '}' {
				depth--
				if depth == 0 {
					end = j + 1
					break
				}
			}
		}

		if end > start {
			candidate := s[start:end]
			// Only include candidates that look like test results
			if strings.Contains(candidate, "test_id") {
				results = append(results, candidate)
			}
			i = end
		} else {
			i = start + 1
		}
	}

	return results
}

// truncateOutput shortens output for error messages
func truncateOutput(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "... (truncated)"
}

// setupTestDir creates a temporary directory for testing
func setupTestDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "agent-integration-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(dir)
	})
	return dir
}

// validateResult checks that all assertions passed
func validateResult(t *testing.T, result *TestResult) {
	t.Helper()

	if result.Error != "" {
		t.Fatalf("Test %s failed with error: %s", result.TestID, result.Error)
	}

	for _, assertion := range result.Assertions {
		if !assertion.Passed {
			t.Errorf("Assertion %q failed: %s", assertion.Name, assertion.Detail)
		}
	}
}

// TestClaude_Available verifies Claude CLI is available before running other tests
func TestClaude_Available(t *testing.T) {
	if _, err := exec.LookPath("claude"); err != nil {
		t.Skip("Claude CLI not available, skipping integration tests")
	}

	// Quick sanity check
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "claude", "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Skipf("Claude CLI not working properly: %v\nOutput: %s", err, string(output))
	}
	t.Logf("Claude version: %s", strings.TrimSpace(string(output)))
}

// TestAgent_BasicResponse verifies basic agent communication works
func TestAgent_BasicResponse(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	dir := setupTestDir(t)

	result := runAgentTest(t, TestConfig{
		TestID:  "basic-response",
		WorkDir: dir,
		Timeout: 2 * time.Minute,
		Prompt: `
Test that you can respond to a simple request.

Assertions to check:
1. "can_respond" - You can process this prompt and generate a response
2. "can_output_json" - You can output valid JSON in the required format
`,
	})

	validateResult(t, result)

	// Verify we got at least one assertion
	if len(result.Assertions) < 1 {
		t.Error("Expected at least 1 assertion in result")
	}
}

// TestAgent_BlockedTools verifies agent behavior when tools are unavailable
func TestAgent_BlockedTools(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	dir := setupTestDir(t)

	// Create a simple file to read
	testFile := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	result := runAgentTest(t, TestConfig{
		TestID:       "blocked-tools",
		WorkDir:      dir,
		Timeout:      2 * time.Minute,
		BlockedTools: []string{"Bash", "Write"},
		Prompt: fmt.Sprintf(`
Test behavior when certain tools are blocked.

The following tools are BLOCKED for this test: Bash, Write
You should still be able to use: Read, Glob, Grep

Assertions to check:
1. "can_read_files" - Attempt to read %s using the Read tool. This should work. Pass if you can read it.
2. "acknowledges_blocked" - Acknowledge that Bash and Write tools are not available for this test.
`, testFile),
	})

	validateResult(t, result)
}

// TestAgent_DisabledSkills verifies agent behavior with skills disabled
func TestAgent_DisabledSkills(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	dir := setupTestDir(t)

	result := runAgentTest(t, TestConfig{
		TestID:        "disabled-skills",
		WorkDir:       dir,
		Timeout:       2 * time.Minute,
		DisableSkills: true,
		Prompt: `
Test behavior when slash commands (skills) are disabled.

This test runs with --disable-slash-commands flag.

Assertions to check:
1. "skills_disabled" - Confirm that you are running with slash commands disabled (you may notice limited skill access).
2. "can_still_respond" - Confirm you can still process requests and generate output without skills.
`,
	})

	validateResult(t, result)
}

// TestAgent_DisabledClaudeMd verifies agent behavior without CLAUDE.md
func TestAgent_DisabledClaudeMd(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	dir := setupTestDir(t)

	result := runAgentTest(t, TestConfig{
		TestID:        "disabled-claude-md",
		WorkDir:       dir,
		Timeout:       2 * time.Minute,
		DisableClaude: true,
		Prompt: `
Test behavior when CLAUDE.md configuration is not available.

This test runs with CLAUDE_CONFIG_DIR set to a non-existent directory,
which means project-specific CLAUDE.md files may not be loaded.

Assertions to check:
1. "can_function_without_claude_md" - You can still respond and function without CLAUDE.md context.
2. "no_project_context" - Confirm you don't have access to specific project instructions from CLAUDE.md.
`,
	})

	validateResult(t, result)
}

// TestAgent_JsonOutputFormat verifies agent correctly outputs JSON
func TestAgent_JsonOutputFormat(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	dir := setupTestDir(t)

	result := runAgentTest(t, TestConfig{
		TestID:  "json-output-format",
		WorkDir: dir,
		Timeout: 2 * time.Minute,
		Prompt: `
Test that you can output properly formatted JSON.

Assertions to check:
1. "valid_json" - Your output is valid JSON that can be parsed.
2. "correct_structure" - The JSON has the correct structure with test_id and assertions array.
3. "assertion_format" - Each assertion has name, passed, and optional detail fields.
`,
	})

	// This test specifically validates the JSON parsing worked
	if result.Error != "" {
		t.Fatalf("Failed to get valid JSON: %s", result.Error)
	}

	if result.TestID != "json-output-format" {
		t.Errorf("Expected test_id 'json-output-format', got '%s'", result.TestID)
	}

	validateResult(t, result)
}

// TestAgent_MultipleAssertions verifies agent can check multiple conditions
func TestAgent_MultipleAssertions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	dir := setupTestDir(t)

	// Create test files
	if err := os.WriteFile(filepath.Join(dir, "file1.txt"), []byte("content 1"), 0644); err != nil {
		t.Fatalf("Failed to create file1: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "file2.txt"), []byte("content 2"), 0644); err != nil {
		t.Fatalf("Failed to create file2: %v", err)
	}

	result := runAgentTest(t, TestConfig{
		TestID:  "multiple-assertions",
		WorkDir: dir,
		Timeout: 3 * time.Minute,
		Prompt: fmt.Sprintf(`
Test that you can verify multiple conditions and report each separately.

Working directory: %s

Assertions to check:
1. "file1_exists" - file1.txt exists in the working directory
2. "file2_exists" - file2.txt exists in the working directory
3. "file1_content" - file1.txt contains "content 1"
4. "file2_content" - file2.txt contains "content 2"
5. "no_file3" - file3.txt does NOT exist (this assertion should pass)
`, dir),
	})

	validateResult(t, result)

	// Verify we got all 5 assertions
	if len(result.Assertions) < 5 {
		t.Errorf("Expected at least 5 assertions, got %d", len(result.Assertions))
	}
}
