# WP3 Code Examples: Before and After

## Example 1: Delete Command

### Before (using bufio.ReadString)

```go
// Confirm deletion unless --force is used
if !deleteForce {
    reader := bufio.NewReader(os.Stdin)
    fmt.Printf("Are you sure you want to delete this ball? This cannot be undone. [y/N]: ")
    input, _ := reader.ReadString('\n')
    input = strings.TrimSpace(strings.ToLower(input))

    if input != "y" && input != "yes" {
        fmt.Println("Deletion cancelled.")
        return nil
    }
}
```

**Issues:**
- User must press 'y' then Enter (two keypresses)
- Extra newline processing needed
- Multiple acceptable responses ('y', 'yes') causing inconsistency

### After (using ConfirmSingleKey)

```go
// Confirm deletion unless --force is used
if !deleteForce {
    fmt.Print("Are you sure you want to delete this ball? This cannot be undone. ")
    confirmed, err := ConfirmSingleKey("")
    if err != nil {
        return fmt.Errorf("operation cancelled")
    }

    if !confirmed {
        fmt.Println("Deletion cancelled.")
        return nil
    }
}
```

**Benefits:**
- Single keypress (just 'y' or 'n')
- Immediate response
- Clear error handling for Ctrl+C
- Consistent behavior across all confirmations

---

## Example 2: Check Command

### Before (using bufio.ReadString)

```go
// Prompt user
reader := bufio.NewReader(os.Stdin)
fmt.Print("Is this what you're working on? (y/n): ")
input, err := reader.ReadString('\n')
if err != nil {
    return fmt.Errorf("failed to read input: %w", err)
}
response := strings.TrimSpace(strings.ToLower(input))

if response == "y" || response == "yes" {
    successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
    fmt.Println()
    fmt.Println(successStyle.Render("✓ Great! Continue working on this ball."))
    return nil
}
```

**Issues:**
- Requires Enter key
- String processing needed
- Inconsistent handling of 'yes' vs 'y'

### After (using ConfirmSingleKey)

```go
// Prompt user
confirmed, err := ConfirmSingleKey("Is this what you're working on?")
if err != nil {
    return fmt.Errorf("operation cancelled")
}

if confirmed {
    successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
    fmt.Println()
    fmt.Println(successStyle.Render("✓ Great! Continue working on this ball."))
    return nil
}
```

**Benefits:**
- Single keypress
- No string processing
- Boolean result (true/false) is cleaner
- Explicit error handling

---

## Example 3: Setup Agent Command

### Before (using fmt.Scanln)

```go
// Confirm unless forced
if !opts.Force {
    fmt.Printf("Install these instructions? [Y/n]: ")
    var response string
    fmt.Scanln(&response)
    response = strings.ToLower(strings.TrimSpace(response))
    if response != "" && response != "y" && response != "yes" {
        fmt.Println("Cancelled.")
        return nil
    }
}
```

**Issues:**
- Requires Enter key
- Complex default handling (empty = yes)
- Multiple acceptable responses
- Inconsistent with other confirmations

### After (using ConfirmSingleKey)

```go
// Confirm unless forced
if !opts.Force {
    confirmed, err := ConfirmSingleKey("Install these instructions?")
    if err != nil {
        return fmt.Errorf("operation cancelled")
    }
    if !confirmed {
        fmt.Println("Cancelled.")
        return nil
    }
}
```

**Benefits:**
- Single keypress
- Explicit yes/no (no default behavior)
- Consistent with all other confirmations
- Better user experience

---

## The ConfirmSingleKey Implementation

```go
func ConfirmSingleKey(prompt string) (bool, error) {
    fmt.Printf("%s (y/n): ", prompt)

    // Get terminal file descriptor
    fd := int(os.Stdin.Fd())

    // Save original terminal state and ensure it's restored
    oldState, err := term.MakeRaw(fd)
    if err != nil {
        return false, fmt.Errorf("failed to set raw mode: %w", err)
    }
    defer term.Restore(fd, oldState)

    // Read single byte
    b := make([]byte, 1)
    _, err = os.Stdin.Read(b)
    if err != nil {
        return false, fmt.Errorf("failed to read input: %w", err)
    }

    // Check key pressed
    key := b[0]

    // Handle Ctrl+C (ASCII 3)
    if key == 3 {
        fmt.Println("\n^C")
        return false, fmt.Errorf("interrupted")
    }

    // Echo the key and newline for valid responses
    if key == 'y' || key == 'Y' {
        fmt.Println("y")
        return true, nil
    } else if key == 'n' || key == 'N' {
        fmt.Println("n")
        return false, nil
    }

    // Invalid key - restore terminal and recurse
    term.Restore(fd, oldState)
    fmt.Println()
    fmt.Println("Invalid key. Please press 'y' or 'n'.")
    return ConfirmSingleKey(prompt)
}
```

**Key Features:**
1. **Raw Terminal Mode:** Captures keypress immediately
2. **Defer Pattern:** Ensures terminal is restored even on panic
3. **Echo Feedback:** Shows user what they pressed
4. **Ctrl+C Handling:** Graceful interruption
5. **Invalid Key Recovery:** Re-prompts on bad input
6. **Case Insensitive:** Accepts Y/y and N/n
7. **Boolean Return:** Simple true/false result

---

## User Experience Comparison

### Old Behavior
```
Are you sure? [y/N]: y<Enter>
✓ Confirmed
```
**Keypresses:** 2 (y + Enter)

### New Behavior
```
Are you sure? (y/n): y
✓ Confirmed
```
**Keypresses:** 1 (just y)

**Time Saved:** ~50% fewer keypresses, immediate response

---

## Edge Cases Handled

1. **Uppercase Keys:** Y and N work same as y and n
2. **Invalid Keys:** Re-prompts with helpful message
3. **Ctrl+C:** Shows ^C and returns error
4. **Terminal Errors:** Returns descriptive error messages
5. **Force Flags:** Bypass confirmation entirely (unchanged behavior)

---

## Backward Compatibility

- Force flags (`--force`, `-f`) continue to work
- All existing integration tests pass
- No breaking changes to command-line interface
- Scripts using `--force` are unaffected
