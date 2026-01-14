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
// Note: no omitempty tags so all fields are always shown in the editor
type BallYAML struct {
	ID                 string   `yaml:"id"`
	Context            string   `yaml:"context"`
	Title              string   `yaml:"title"`
	Priority           string   `yaml:"priority"`
	State              string   `yaml:"state"`
	BlockedReason      string   `yaml:"blocked_reason"`
	Tags               []string `yaml:"tags"`
	AcceptanceCriteria []string `yaml:"acceptance_criteria"`
	ModelSize          string   `yaml:"model_size"`
}

// ballToYAML converts a ball to YAML format for editing
// All fields are always shown, even when empty, for discoverability
func ballToYAML(ball *session.Ball) (string, error) {
	// Initialize slices to empty (not nil) so they show as [] in YAML
	tags := ball.Tags
	if tags == nil {
		tags = []string{}
	}
	ac := ball.AcceptanceCriteria
	if ac == nil {
		ac = []string{}
	}

	yamlBall := BallYAML{
		ID:                 ball.ID,
		Context:            ball.Context,
		Title:              ball.Title,
		Priority:           string(ball.Priority),
		State:              string(ball.State),
		BlockedReason:      ball.BlockedReason,
		Tags:               tags,
		AcceptanceCriteria: ac,
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
# Empty values can be left as-is or cleared
# Empty arrays can be written as: tags: []

`
	return header + string(data), nil
}

// yamlToBall parses edited YAML and applies changes to a ball
// Empty values are handled gracefully:
// - Required fields (intent, priority, state): keep existing value if empty/whitespace
// - Optional fields (blocked_reason, tags, acceptance_criteria, model_size): can be cleared to empty
func yamlToBall(yamlContent string, ball *session.Ball) error {
	var yamlBall BallYAML
	if err := yaml.Unmarshal([]byte(yamlContent), &yamlBall); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Validate and apply changes
	// Note: ID is read-only (can't change ball ID)

	// Update context (can be cleared - trim whitespace)
	ball.Context = strings.TrimSpace(yamlBall.Context)

	// Update title (only if non-empty after trimming whitespace)
	title := strings.TrimSpace(yamlBall.Title)
	if title != "" {
		ball.Title = title
	}

	// Update priority (only if non-empty after trimming whitespace)
	priority := strings.TrimSpace(yamlBall.Priority)
	if priority != "" {
		p := session.Priority(priority)
		if !session.ValidatePriority(string(p)) {
			return fmt.Errorf("invalid priority: %s (must be low, medium, high, or urgent)", priority)
		}
		ball.Priority = p
	}

	// Update state (only if non-empty after trimming whitespace)
	state := strings.TrimSpace(yamlBall.State)
	if state != "" {
		s := session.BallState(state)
		if !session.ValidateBallState(string(s)) {
			return fmt.Errorf("invalid state: %s (must be pending, in_progress, complete, blocked, researched, or on_hold)", state)
		}
		ball.State = s
	}

	// Update blocked reason (can be cleared - trim whitespace)
	ball.BlockedReason = strings.TrimSpace(yamlBall.BlockedReason)

	// Update tags (can be cleared to empty array)
	// Trim whitespace from each tag and remove empty tags
	var cleanTags []string
	for _, tag := range yamlBall.Tags {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			cleanTags = append(cleanTags, tag)
		}
	}
	ball.Tags = cleanTags

	// Update acceptance criteria (can be cleared to empty array)
	// Trim whitespace from each criterion and remove empty ones
	var cleanAC []string
	for _, ac := range yamlBall.AcceptanceCriteria {
		ac = strings.TrimSpace(ac)
		if ac != "" {
			cleanAC = append(cleanAC, ac)
		}
	}
	ball.AcceptanceCriteria = cleanAC

	// Update model size (can be cleared to blank/default)
	modelSize := strings.TrimSpace(yamlBall.ModelSize)
	if modelSize != "" {
		ms := session.ModelSize(modelSize)
		switch ms {
		case session.ModelSizeSmall, session.ModelSizeMedium, session.ModelSizeLarge, session.ModelSizeBlank:
			ball.ModelSize = ms
		default:
			return fmt.Errorf("invalid model_size: %s (must be small, medium, large, or empty)", modelSize)
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
