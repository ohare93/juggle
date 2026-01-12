//go:build integration
// +build integration

package agent_integration_test

import (
	"testing"
)

// TestExtractJSONObjects tests the JSON extraction function with various inputs
func TestExtractJSONObjects(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int // number of JSON objects expected
		validate func(t *testing.T, results []string)
	}{
		{
			name:     "simple JSON",
			input:    `{"test_id": "foo", "error": "bar"}`,
			expected: 1,
		},
		{
			name:     "JSON with text before",
			input:    `Some text before {"test_id": "foo", "error": "bar"}`,
			expected: 1,
		},
		{
			name:     "JSON with text after",
			input:    `{"test_id": "foo", "error": "bar"} some text after`,
			expected: 1,
		},
		{
			name:     "JSON with text before and after",
			input:    `Prefix text {"test_id": "foo"} suffix text`,
			expected: 1,
		},
		{
			name:     "nested objects in assertions",
			input:    `{"test_id": "nested", "assertions": [{"name": "a", "passed": true, "detail": {"nested": "object"}}]}`,
			expected: 1,
		},
		{
			name:     "deeply nested objects",
			input:    `{"test_id": "deep", "assertions": [{"name": "a", "passed": true, "detail": {"level1": {"level2": {"level3": "value"}}}}]}`,
			expected: 1,
			validate: func(t *testing.T, results []string) {
				if len(results) != 1 {
					t.Fatalf("Expected 1 result, got %d", len(results))
				}
				// Verify we got the complete JSON
				if results[0] != `{"test_id": "deep", "assertions": [{"name": "a", "passed": true, "detail": {"level1": {"level2": {"level3": "value"}}}}]}` {
					t.Errorf("Did not extract complete JSON: %s", results[0])
				}
			},
		},
		{
			name:     "braces in string values",
			input:    `{"test_id": "braces", "detail": "This has {curly} braces"}`,
			expected: 1,
			validate: func(t *testing.T, results []string) {
				if len(results) != 1 {
					t.Fatalf("Expected 1 result, got %d", len(results))
				}
				// Verify we got the complete JSON including the string with braces
				if results[0] != `{"test_id": "braces", "detail": "This has {curly} braces"}` {
					t.Errorf("Did not handle string braces correctly: %s", results[0])
				}
			},
		},
		{
			name:     "escaped quotes in strings",
			input:    `{"test_id": "escape", "detail": "She said \"hello\""}`,
			expected: 1,
		},
		{
			name:     "multiple JSON objects",
			input:    `First: {"test_id": "one"} Second: {"test_id": "two"}`,
			expected: 2,
		},
		{
			name:     "no test_id filtered out",
			input:    `{"other": "json"} {"test_id": "valid"}`,
			expected: 1, // Only the one with test_id should be returned
		},
		{
			name:     "empty input",
			input:    "",
			expected: 0,
		},
		{
			name:     "no JSON",
			input:    "just plain text without any json",
			expected: 0,
		},
		{
			name:     "unclosed brace",
			input:    `{"test_id": "incomplete"`,
			expected: 0,
		},
		{
			name:     "real world output with surrounding text",
			input:    "I'll check the assertions now.\n\n{\"test_id\": \"test-1\", \"assertions\": [{\"name\": \"check1\", \"passed\": true, \"detail\": \"All good\"}]}\n\nLet me know if you need anything else.",
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := extractJSONObjects(tt.input)
			if len(results) != tt.expected {
				t.Errorf("Expected %d JSON objects, got %d: %v", tt.expected, len(results), results)
			}
			if tt.validate != nil {
				tt.validate(t, results)
			}
		})
	}
}

// TestParseTestResult tests the full parsing pipeline
func TestParseTestResult(t *testing.T) {
	tests := []struct {
		name        string
		testID      string
		output      string
		expectError bool
		validate    func(t *testing.T, result *TestResult)
	}{
		{
			name:   "simple success",
			testID: "simple",
			output: `{"test_id": "simple", "assertions": [{"name": "test1", "passed": true}]}`,
			validate: func(t *testing.T, result *TestResult) {
				if result.Error != "" {
					t.Errorf("Expected no error, got: %s", result.Error)
				}
				if len(result.Assertions) != 1 {
					t.Errorf("Expected 1 assertion, got %d", len(result.Assertions))
				}
			},
		},
		{
			name:   "with error field",
			testID: "error-test",
			output: `{"test_id": "error-test", "error": "something went wrong"}`,
			validate: func(t *testing.T, result *TestResult) {
				if result.Error != "something went wrong" {
					t.Errorf("Expected error message, got: %s", result.Error)
				}
			},
		},
		{
			name:   "nested detail objects",
			testID: "nested",
			output: `{"test_id": "nested", "assertions": [{"name": "check", "passed": true, "detail": {"status": "ok", "info": {"nested": "value"}}}]}`,
			validate: func(t *testing.T, result *TestResult) {
				if result.Error != "" {
					t.Errorf("Expected no error, got: %s", result.Error)
				}
				if len(result.Assertions) != 1 {
					t.Errorf("Expected 1 assertion, got %d", len(result.Assertions))
				}
				// The detail field should be a string representation in our struct
				// since Detail is defined as string type
			},
		},
		{
			name:   "surrounded by text",
			testID: "embedded",
			output: "Here is the result:\n{\"test_id\": \"embedded\", \"assertions\": [{\"name\": \"a\", \"passed\": true}]}\nDone!",
			validate: func(t *testing.T, result *TestResult) {
				if result.Error != "" {
					t.Errorf("Expected no error, got: %s", result.Error)
				}
				if result.TestID != "embedded" {
					t.Errorf("Expected test_id 'embedded', got %s", result.TestID)
				}
			},
		},
		{
			name:        "wrong test_id",
			testID:      "expected-id",
			output:      `{"test_id": "different-id", "assertions": []}`,
			expectError: true,
		},
		{
			name:        "no valid JSON",
			testID:      "missing",
			output:      "no json here at all",
			expectError: true,
		},
		{
			name:   "multiple assertions",
			testID: "multi",
			output: `{"test_id": "multi", "assertions": [{"name": "a", "passed": true}, {"name": "b", "passed": false, "detail": "failed check"}, {"name": "c", "passed": true}]}`,
			validate: func(t *testing.T, result *TestResult) {
				if len(result.Assertions) != 3 {
					t.Errorf("Expected 3 assertions, got %d", len(result.Assertions))
				}
				if result.Assertions[1].Passed {
					t.Error("Expected assertion 'b' to be failed")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseTestResult(t, tt.testID, tt.output)
			if tt.expectError {
				if result.Error == "" {
					t.Error("Expected error but got none")
				}
			} else if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

// TestParseTestResult_ComplexNestedObjects specifically tests the fix for nested JSON
func TestParseTestResult_ComplexNestedObjects(t *testing.T) {
	// This test case would have failed with the old regex-based parser
	testID := "nested-detail"
	output := `Here is the result:

{"test_id": "nested-detail", "assertions": [
  {"name": "file_check", "passed": true, "detail": {"path": "/tmp/test", "info": {"size": 100, "perms": "0644"}}},
  {"name": "content_check", "passed": true, "detail": {"matches": [{"line": 1, "text": "hello"}, {"line": 2, "text": "world"}]}}
]}

Test complete.`

	result := parseTestResult(t, testID, output)

	if result.Error != "" {
		t.Fatalf("Failed to parse nested JSON: %s", result.Error)
	}

	if result.TestID != testID {
		t.Errorf("Expected test_id %s, got %s", testID, result.TestID)
	}

	if len(result.Assertions) != 2 {
		t.Errorf("Expected 2 assertions, got %d", len(result.Assertions))
	}
}
