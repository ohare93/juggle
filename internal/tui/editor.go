package tui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ohare93/juggle/internal/session"
	"gopkg.in/yaml.v3"
)

// BallYAML is the YAML-editable representation of a ball
type BallYAML struct {
	ID                 string   `yaml:"id"`
	Intent             string   `yaml:"intent"`
	Priority           string   `yaml:"priority"`
	State              string   `yaml:"state"`
	BlockedReason      string   `yaml:"blocked_reason,omitempty"`
	Tags               []string `yaml:"tags,omitempty"`
	AcceptanceCriteria []string `yaml:"acceptance_criteria,omitempty"`
	ModelSize          string   `yaml:"model_size,omitempty"`
}

// ballToYAML converts a ball to YAML format for editing
func ballToYAML(ball *session.Ball) (string, error) {
	yamlBall := BallYAML{
		ID:                 ball.ID,
		Intent:             ball.Intent,
		Priority:           string(ball.Priority),
		State:              string(ball.State),
		BlockedReason:      ball.BlockedReason,
		Tags:               ball.Tags,
		AcceptanceCriteria: ball.AcceptanceCriteria,
		ModelSize:          string(ball.ModelSize),
	}

	data, err := yaml.Marshal(&yamlBall)
	if err != nil {
		return "", fmt.Errorf("failed to marshal ball to YAML: %w", err)
	}

	// Add header comment with editing instructions
	header := `# Edit ball properties below
# Lines starting with # are ignored
# Save and close editor to apply changes
# Close without saving to cancel

`
	return header + string(data), nil
}

// yamlToBall parses edited YAML and applies changes to a ball
func yamlToBall(yamlContent string, ball *session.Ball) error {
	var yamlBall BallYAML
	if err := yaml.Unmarshal([]byte(yamlContent), &yamlBall); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Validate and apply changes
	// Note: ID is read-only (can't change ball ID)

	// Update intent
	if yamlBall.Intent != "" {
		ball.Intent = yamlBall.Intent
	}

	// Update priority
	if yamlBall.Priority != "" {
		priority := session.Priority(yamlBall.Priority)
		if !session.ValidatePriority(string(priority)) {
			return fmt.Errorf("invalid priority: %s (must be low, medium, high, or urgent)", yamlBall.Priority)
		}
		ball.Priority = priority
	}

	// Update state
	if yamlBall.State != "" {
		state := session.BallState(yamlBall.State)
		if !session.ValidateBallState(string(state)) {
			return fmt.Errorf("invalid state: %s (must be pending, in_progress, complete, or blocked)", yamlBall.State)
		}
		ball.State = state
	}

	// Update blocked reason
	ball.BlockedReason = yamlBall.BlockedReason

	// Update tags
	ball.Tags = yamlBall.Tags

	// Update acceptance criteria
	ball.AcceptanceCriteria = yamlBall.AcceptanceCriteria

	// Update model size
	if yamlBall.ModelSize != "" {
		modelSize := session.ModelSize(yamlBall.ModelSize)
		switch modelSize {
		case session.ModelSizeSmall, session.ModelSizeMedium, session.ModelSizeLarge, session.ModelSizeBlank:
			ball.ModelSize = modelSize
		default:
			return fmt.Errorf("invalid model_size: %s (must be small, medium, large, or empty)", yamlBall.ModelSize)
		}
	} else {
		ball.ModelSize = session.ModelSizeBlank
	}

	ball.UpdateActivity()
	return nil
}

// editorResultMsg is the message returned after editor closes
type editorResultMsg struct {
	ball        *session.Ball
	editedYAML  string
	cancelled   bool
	err         error
}

// openEditorCmd creates a tea.Cmd that opens an external editor for ball editing
func openEditorCmd(ball *session.Ball) tea.Cmd {
	// Get editor from environment
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi" // Default fallback
	}

	// Generate YAML content
	yamlContent, err := ballToYAML(ball)
	if err != nil {
		return func() tea.Msg {
			return editorResultMsg{ball: ball, err: err}
		}
	}

	// Create temp file
	tmpFile, err := os.CreateTemp("", "juggle-ball-*.yaml")
	if err != nil {
		return func() tea.Msg {
			return editorResultMsg{ball: ball, err: fmt.Errorf("failed to create temp file: %w", err)}
		}
	}
	tmpPath := tmpFile.Name()

	// Write YAML to temp file
	if _, err := tmpFile.WriteString(yamlContent); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return func() tea.Msg {
			return editorResultMsg{ball: ball, err: fmt.Errorf("failed to write temp file: %w", err)}
		}
	}
	tmpFile.Close()

	// Get the original content for comparison
	originalContent := yamlContent

	// Create the editor command
	editorParts := strings.Fields(editor)
	var editorCmd *exec.Cmd
	if len(editorParts) > 1 {
		editorCmd = exec.Command(editorParts[0], append(editorParts[1:], tmpPath)...)
	} else {
		editorCmd = exec.Command(editor, tmpPath)
	}

	// Use tea.ExecProcess to properly handle terminal suspension
	return tea.ExecProcess(editorCmd, func(err error) tea.Msg {
		defer os.Remove(tmpPath)

		if err != nil {
			return editorResultMsg{ball: ball, err: fmt.Errorf("editor failed: %w", err)}
		}

		// Read the edited content
		editedContent, err := os.ReadFile(tmpPath)
		if err != nil {
			return editorResultMsg{ball: ball, err: fmt.Errorf("failed to read edited file: %w", err)}
		}

		// Check if content was modified
		if string(editedContent) == originalContent {
			return editorResultMsg{ball: ball, cancelled: true}
		}

		return editorResultMsg{
			ball:       ball,
			editedYAML: string(editedContent),
		}
	})
}
