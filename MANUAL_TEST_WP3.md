# Manual Testing Checklist for WP3: Single-Key Confirmations

This workpackage replaced Enter-required confirmations with immediate single-key responses.

## Test Setup

Build the binary:
```bash
go build -o juggle ./cmd/juggle
```

Create test balls:
```bash
./juggle plan --intent "Test ball for confirmation" --priority medium
# Note the ball ID (e.g., juggler-51)
```

## Test Cases

### 1. Delete Command - Press 'y'
```bash
./juggle delete juggler-51
# Expected: Prompt "Are you sure you want to delete this ball? This cannot be undone. (y/n): "
# Action: Press 'y' (no Enter needed)
# Expected: Immediate response, shows "y" echoed, deletes ball
```

### 2. Delete Command - Press 'n'
```bash
./juggle plan --intent "Test ball 2" --priority medium
./juggle delete juggler-52
# Expected: Prompt appears
# Action: Press 'n' (no Enter needed)
# Expected: Immediate response, shows "n" echoed, displays "Deletion cancelled."
```

### 3. Delete Command - Press 'Y' (uppercase)
```bash
./juggle plan --intent "Test ball 3" --priority medium
./juggle delete juggler-53
# Action: Press 'Y' (uppercase, no Enter needed)
# Expected: Works same as lowercase 'y', deletes ball
```

### 4. Delete Command - Press Ctrl+C
```bash
./juggle plan --intent "Test ball 4" --priority medium
./juggle delete juggler-54
# Action: Press Ctrl+C
# Expected: Shows "^C", displays "Error: operation cancelled", ball not deleted
```

### 5. Delete Command - Invalid key
```bash
./juggle plan --intent "Test ball 5" --priority medium
./juggle delete juggler-55
# Action: Press 'x' or any other invalid key
# Expected: Shows "Invalid key. Please press 'y' or 'n'.", re-prompts
# Then press 'n' to cancel
```

### 6. Check Command - Single Juggling Ball
```bash
./juggle plan --intent "Test check command" --priority medium
./juggle juggler-56 in-air
./juggle check
# Expected: Shows current ball, prompts "Is this what you're working on? (y/n): "
# Action: Press 'y'
# Expected: Immediate response, shows success message
```

### 7. Check Command - Decline Current Ball
```bash
./juggle check
# Action: Press 'n' when asked about current ball
# Expected: Shows guidance about other actions
```

### 8. Setup Agent Command - Uninstall Confirmation
```bash
./juggle setup-agent backend-architect --uninstall --local
# Expected: Prompt "Remove backend-architect instructions from .claude/CLAUDE.md? (y/n): "
# Action: Press 'n'
# Expected: Immediate cancellation
```

### 9. Setup Agent Command - Install Confirmation
```bash
./juggle setup-agent backend-architect --local
# Expected: Shows preview, prompt "Install these instructions? (y/n): "
# Action: Press 'y'
# Expected: Immediate installation
```

### 10. Force Flags Still Work
```bash
./juggle plan --intent "Force test" --priority medium
./juggle delete juggler-57 --force
# Expected: No prompt, immediate deletion

./juggle setup-agent backend-architect --local --force --update
# Expected: No prompt, immediate update
```

## Success Criteria

All test cases should:
- ✅ Respond immediately to keypress (no Enter required)
- ✅ Echo the selected key ('y' or 'n') to terminal
- ✅ Accept both uppercase and lowercase (Y/y, N/n)
- ✅ Handle Ctrl+C gracefully (show ^C, return error message)
- ✅ Re-prompt on invalid keys
- ✅ Force flags skip confirmation entirely

## Automated Tests

All integration tests pass:
```bash
go test -v ./internal/integration_test/...
```

## Notes

- Raw terminal mode is required for single-key input
- stdin redirection (echo "y" | command) does NOT work with raw mode
- All testing must be done interactively in a real terminal
- The functionality works correctly when tested manually
