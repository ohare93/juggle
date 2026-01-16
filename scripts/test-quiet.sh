#!/usr/bin/env bash
# test-quiet.sh - Run tests with minimal output
# On success: exit 0 with "All tests passed (N tests)"
# On failure: exit non-zero, show only failing test output

# Run tests with JSON output to a temp file
TMPFILE=$(mktemp)
trap "rm -f $TMPFILE" EXIT

go test -json ./... > "$TMPFILE" 2>&1
TEST_EXIT_CODE=$?

# Count pass/fail from JSON output using grep
# Each test has a "pass" or "fail" action with a "Test" field (non-empty means it's a test, not package)
PASS_COUNT=$(grep '"Action":"pass"' "$TMPFILE" | grep -c '"Test":"[^"]\+"' 2>/dev/null || echo "0")
FAIL_COUNT=$(grep '"Action":"fail"' "$TMPFILE" | grep -c '"Test":"[^"]\+"' 2>/dev/null || echo "0")

# Ensure counts are single numbers (trim whitespace/newlines)
PASS_COUNT=$(echo "$PASS_COUNT" | tr -d '\n' | xargs)
FAIL_COUNT=$(echo "$FAIL_COUNT" | tr -d '\n' | xargs)

TOTAL_COUNT=$((PASS_COUNT + FAIL_COUNT))

if [[ $TEST_EXIT_CODE -eq 0 ]] && [[ $FAIL_COUNT -eq 0 ]]; then
    echo "All tests passed ($TOTAL_COUNT tests)"
    exit 0
else
    echo "Tests failed ($FAIL_COUNT failed, $PASS_COUNT passed)"
    echo ""
    echo "Failed tests:"
    # Extract failed test names - look for lines with "fail" action and non-empty Test field
    grep '"Action":"fail"' "$TMPFILE" | grep '"Test":"[^"]\+"' | sed 's/.*"Test":"\([^"]*\)".*/  \1/' | sort -u
    echo ""
    echo "Run 'devbox run test' for full output"
    exit 1
fi
