# Agent Integration Tests

This package contains integration tests that run against the **real Claude agent**. These tests validate that the agent behaves correctly in specific scenarios.

> **Note on BlockedTools:** The `BlockedTools` field in `TestConfig` is **honor-system based** and not enforced by the testing framework. It instructs the agent via the prompt that certain tools should not be used, but the Claude CLI doesn't actually disable those tools. Tests using `BlockedTools` verify that the agent *respects* the instruction, not that the tools are technically blocked.

## Running Tests

```bash
# Run all integration tests (using devbox script)
devbox run test-agent-integration

# Run all integration tests (without devbox)
go test -tags=integration ./internal/agent_integration_test/...

# Run with verbose output
go test -v -tags=integration ./internal/agent_integration_test/...

# Run a specific test
go test -v -tags=integration ./internal/agent_integration_test/... -run TestAgent_BlockedTools

# Skip long-running tests (useful for quick checks)
go test -short -tags=integration ./internal/agent_integration_test/...
```

## Requirements

- `claude` CLI must be installed and available in PATH
- Valid Claude API credentials configured
- Network access to Claude API

## Test Scenarios

| Test | Description |
|------|-------------|
| `TestAgent_BasicResponse` | Verifies basic agent communication works |
| `TestAgent_BlockedTools` | Tests behavior when certain tools are unavailable |
| `TestAgent_DisabledSkills` | Tests behavior with slash commands disabled |
| `TestAgent_DisabledClaudeMd` | Tests behavior without CLAUDE.md configuration |
| `TestAgent_JsonOutputFormat` | Verifies agent correctly outputs JSON format |
| `TestAgent_MultipleAssertions` | Tests handling of multiple assertions |

## JSON Output Format

Tests instruct the agent to output results in JSON format:

```json
{
  "test_id": "test-name",
  "assertions": [
    {
      "name": "assertion_name",
      "passed": true,
      "detail": "optional details"
    }
  ]
}
```

On error:

```json
{
  "test_id": "test-name",
  "error": "description of the error"
}
```

## CI Integration

These tests should **not** run on every push due to:
- API costs
- Network dependency
- Non-deterministic agent responses

Recommended CI configuration:

```yaml
# GitHub Actions example
agent-integration-tests:
  if: contains(github.event.pull_request.labels.*.name, 'run-integration-tests') || github.event_name == 'workflow_dispatch'
  runs-on: ubuntu-latest
  steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v5
      with:
        go-version: '1.21'
    - name: Install Claude CLI
      run: npm install -g @anthropic-ai/claude-cli
    - name: Run integration tests
      run: go test -v -tags=integration ./internal/agent_integration_test/...
      env:
        ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
```

## Adding New Tests

1. Create a new test function with `integration` build tag
2. Use `runAgentTest()` with a `TestConfig`
3. Define clear assertions in the prompt
4. Call `validateResult()` to check all assertions passed

Example:

```go
func TestAgent_NewScenario(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }

    dir := setupTestDir(t)

    result := runAgentTest(t, TestConfig{
        TestID:  "new-scenario",
        WorkDir: dir,
        Timeout: 2 * time.Minute,
        Prompt: `
Describe what you want the agent to test.

Assertions to check:
1. "first_assertion" - What the first assertion checks
2. "second_assertion" - What the second assertion checks
`,
    })

    validateResult(t, result)
}
```
