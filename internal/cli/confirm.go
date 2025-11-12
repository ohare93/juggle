package cli

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

// ConfirmSingleKey displays a yes/no prompt and waits for a single keypress.
// Returns true for 'y'/'Y', false for 'n'/'N', or error on Ctrl+C.
// No Enter key is required - responds immediately to keypress.
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
