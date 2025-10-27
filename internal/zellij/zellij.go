package zellij

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Info contains Zellij session and tab information
type Info struct {
	SessionName string
	TabName     string
	IsActive    bool
}

// DetectInfo checks if running in Zellij and extracts session/tab info
func DetectInfo() (*Info, error) {
	sessionName := os.Getenv("ZELLIJ_SESSION_NAME")
	if sessionName == "" {
		return &Info{IsActive: false}, nil
	}

	// Try to get current tab name from layout dump
	// Use a short timeout to avoid hanging if zellij isn't responding
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	cmd := exec.CommandContext(ctx, "zellij", "action", "dump-layout")
	output, err := cmd.Output()
	if err != nil {
		// Zellij is detected but can't get layout, use empty tab name
		return &Info{
			SessionName: sessionName,
			TabName:     "",
			IsActive:    true,
		}, nil
	}

	tabName := extractCurrentTab(string(output))

	return &Info{
		SessionName: sessionName,
		TabName:     tabName,
		IsActive:    true,
	}, nil
}

// extractCurrentTab parses the Zellij layout dump to find the current tab name
func extractCurrentTab(layout string) string {
	// Look for 'tab name="..." focus=true' patterns
	lines := strings.Split(layout, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Check if this line has both "tab name=" and "focus=true"
		if strings.HasPrefix(trimmed, "tab name=") && strings.Contains(trimmed, "focus=true") {
			// Extract name from: tab name="foo" focus=true ...
			parts := strings.SplitN(trimmed, "\"", 3)
			if len(parts) >= 2 {
				return parts[1]
			}
		}
	}
	
	// Fallback: if no focused tab found, return first tab
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "tab name=") {
			parts := strings.SplitN(trimmed, "\"", 3)
			if len(parts) >= 2 {
				return parts[1]
			}
		}
	}
	
	return ""
}

// GoToTab switches to a tab by name in the current Zellij session
func GoToTab(tabName string) error {
	if tabName == "" {
		return fmt.Errorf("tab name is empty")
	}

	cmd := exec.Command("zellij", "action", "go-to-tab-name", tabName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to switch to tab %q: %w", tabName, err)
	}

	return nil
}

// IsInstalled checks if Zellij is installed
func IsInstalled() bool {
	_, err := exec.LookPath("zellij")
	return err == nil
}

// GetSessionInfo returns info about the Zellij session
func (i *Info) String() string {
	if !i.IsActive {
		return "Not running in Zellij"
	}

	if i.TabName != "" {
		return fmt.Sprintf("Session: %s, Tab: %s", i.SessionName, i.TabName)
	}

	return fmt.Sprintf("Session: %s", i.SessionName)
}
