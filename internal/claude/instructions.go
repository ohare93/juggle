package claude

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// InstructionsMarkerStart marks the beginning of juggler instructions
	InstructionsMarkerStart = "<!-- juggler-instructions-start -->"
	// InstructionsMarkerEnd marks the end of juggler instructions
	InstructionsMarkerEnd = "<!-- juggler-instructions-end -->"
)

// InstructionsTemplate is the template for CLAUDE.md instructions
const InstructionsTemplate = `
═══════════════════════════════════════════════════════════════
🚫 CRITICAL BLOCKING REQUIREMENT - READ FIRST 🚫
═══════════════════════════════════════════════════════════════

<!-- juggler-instructions-start -->

## MANDATORY JUGGLER CHECK - YOU ARE BLOCKED UNTIL COMPLETE

**STOP.** You MUST run this command BEFORE ANY other action:

1. Run: ` + "`juggle`" + ` (shows currently juggling balls)
2. IF ready/juggling balls exist:
   - You are **BLOCKED** from other work
   - You **MUST** address existing balls FIRST
3. ONLY after handling existing balls may you proceed with new work

**VIOLATION CONSEQUENCES:**
- Creating balls while others are ready = **CRITICAL ERROR**
- Skipping this check = **INVALID SESSION**
- You will be asked to restart if you violate this

═══════════════════════════════════════════════════════════════

## Juggler Workflow - MANDATORY STEPS

### 🔴 Step 1: ALWAYS Check State First

**STOP.** Before doing ANY research, code investigation, or planning:

` + "```bash\njuggle  # Shows currently juggling balls\n```" + `

**YOU ARE BLOCKED from proceeding until you run this command.**

If you see ANY ready or juggling balls, you **MUST** handle them before starting new work.

### 🟡 Step 2: Handle Existing Tasks

**IF balls exist** - You **MUST** determine if current task matches existing ball.

A ball matches the current task if **ANY** of these are true:

1. **Intent/description overlap** - The ball describes the same or related goal
   - Example: Ball "Fix zellij integration" matches task "fix juggle command showing error"
2. **Same component/file** - Working on the same area of code
   - Example: Ball has todos about ` + "`root.go`" + ` matches task involving ` + "`root.go`" + `
3. **Related tags** - Ball has tags matching the task domain
   - Example: Ball tagged "cli" matches any command-line behavior task
4. **Same working directory** - For multi-project setups

**When in doubt:** Ask the user "I see ball X is about Y. Should I use this or create a new ball?"

**CHECKPOINT:** Have you confirmed match/no-match with existing balls?

**If match found - USE EXISTING BALL:**
` + "```bash\njuggle <ball-id>              # Review details and todos\njuggle <ball-id> in-air       # Mark as actively working\n```" + `

**If no match - CREATE NEW BALL:**
` + "```bash\njuggle start  # Interactive creation (recommended)\n# OR\njuggle plan --intent \"...\" --priority medium  # Non-interactive\n```" + `

### 🟢 Step 3: Update Status After Work

These state updates are **MANDATORY**, not optional:

**CHECKPOINT:** Are you marking state transitions as you work?

✅ **When starting work:**
` + "```bash\njuggle <ball-id> in-air\n```" + `

✅ **When you need user input:**
` + "```bash\njuggle <ball-id> needs-thrown\n```" + `

✅ **After completing work:**
` + "```bash\njuggle <ball-id> needs-caught\n```" + `

✅ **When fully done:**
` + "```bash\njuggle <ball-id> complete \"Brief summary\"\n```" + `

**CHECKPOINT:** Did you update ball state after completing work?

═══════════════════════════════════════════════════════════════

## Examples of Compliance

### ❌ WRONG - NEVER DO THIS:

` + "```\nUser: \"Fix the help text for start command\"\nAssistant: *Immediately runs find_symbol and starts investigating*\n\n❌ CRITICAL ERROR - Didn't check juggler first!\n❌ This is a BLOCKING violation\n❌ Session must restart\n```" + `

### ✅ CORRECT - ALWAYS DO THIS:

` + "```\nUser: \"Fix the help text for start command\"\n\nAssistant: STOP - Let me check juggler state first.\n*Runs: juggle*\n*Sees: juggler-8 - \"improving CLI help text\"*\n\nAssistant: Found existing ball (juggler-8) about CLI help. \nI MUST use this existing ball before creating new work.\n\n*Runs: juggle juggler-8*\n*Reviews todos*\n*Runs: juggle juggler-8 in-air*\n\n✓ CORRECT - Checked state FIRST\n✓ CORRECT - Found match and used existing ball\n✓ CORRECT - Marked as in-air before working\n```" + `

═══════════════════════════════════════════════════════════════

## Detailed Reference Information

### 🎯 The Juggling Metaphor

Think of tasks as balls being juggled:
- **needs-thrown**: Ball needs your throw (user must give direction)
- **in-air**: Ball is flying (you're actively working)
- **needs-caught**: Ball coming down (user must verify/catch)
- **complete**: Ball successfully caught and put away
- **dropped**: Ball fell and is no longer being juggled

### 📚 Common Commands Reference

- ` + "`juggle`" + ` - Show currently juggling balls
- ` + "`juggle <ball-id>`" + ` - Show ball details
- ` + "`juggle balls`" + ` - List all balls (any state)
- ` + "`juggle <ball-id> <state>`" + ` - Update ball state
- ` + "`juggle <ball-id> todo add \"task\"`" + ` - Add todo
- ` + "`juggle <ball-id> todo done <N>`" + ` - Complete todo N
- ` + "`juggle next`" + ` - Find ball needing attention

### 🔄 Multi-Agent / Multi-Session Support

When multiple agents/users work simultaneously, activity tracking resolution:

1. **JUGGLER_CURRENT_BALL env var** - Explicit override
2. **Zellij session+tab matching** - Auto-detects from environment
3. **Single juggling ball** - If only one is juggling
4. **Most recently active** - Fallback

Set explicit ball:
` + "```bash\nexport JUGGLER_CURRENT_BALL=\"juggler-5\"\n```" + `

### 📝 Technical Notes

- Ball IDs: ` + "`<directory-name>-<counter>`" + ` (e.g., ` + "`juggler-1`" + `, ` + "`myapp-5`" + `)
- Activity tracking via hooks updates timestamps automatically
- Balls store Zellij session/tab info when created in Zellij
- Multiple balls can coexist per project (use explicit IDs)

<!-- juggler-instructions-end -->
`

// InstallOptions holds options for installing instructions
type InstallOptions struct {
	Global    bool   // Install to global CLAUDE.md
	Local     bool   // Install to local .claude/CLAUDE.md
	DryRun    bool   // Don't actually install, just show what would happen
	Update    bool   // Update existing instructions
	Uninstall bool   // Remove instructions
	Force     bool   // Don't prompt for confirmation
}

// GetTargetPath returns the path to CLAUDE.md based on options
func GetTargetPath(opts InstallOptions) (string, error) {
	if opts.Global {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		return filepath.Join(homeDir, ".claude", "CLAUDE.md"), nil
	}

	// Local is default
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}
	return filepath.Join(wd, ".claude", "CLAUDE.md"), nil
}

// GetProjectDir returns the current working directory for project-level operations
func GetProjectDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}
	return wd, nil
}

// FileExists checks if a file exists
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// HasInstructions checks if file already contains juggler instructions
func HasInstructions(path string) (bool, error) {
	if !FileExists(path) {
		return false, nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("failed to read file: %w", err)
	}

	return strings.Contains(string(content), InstructionsMarkerStart), nil
}

// ReadFile reads the content of CLAUDE.md
func ReadFile(path string) (string, error) {
	if !FileExists(path) {
		return "", nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}
	return string(content), nil
}

// RemoveInstructions removes juggler instructions from content
func RemoveInstructions(content string) string {
	startIdx := strings.Index(content, InstructionsMarkerStart)
	if startIdx == -1 {
		return content
	}

	endIdx := strings.Index(content, InstructionsMarkerEnd)
	if endIdx == -1 {
		return content
	}

	// Find the start of removal - look for ANY heading/separator related to Juggler before the marker
	// This handles cases where there might be an orphaned old header above the current one
	removeStart := startIdx

	// Search backward line by line from the marker
	searchPos := startIdx - 1

	// Skip backward to find the last heading/separator before the marker (even with blank lines)
	for searchPos >= 0 {
		// Find the start of the current line
		lineStart := searchPos
		for lineStart > 0 && content[lineStart-1] != '\n' {
			lineStart--
		}

		// Find the end of the current line
		lineEnd := searchPos + 1
		for lineEnd < len(content) && content[lineEnd] != '\n' {
			lineEnd++
		}

		line := content[lineStart:lineEnd]
		trimmedLine := strings.TrimSpace(line)

		// If we find a heading that mentions Juggler or a visual separator, that's our removal start
		if (strings.HasPrefix(trimmedLine, "##") &&
		   (strings.Contains(strings.ToLower(trimmedLine), "juggler") ||
		    strings.Contains(trimmedLine, "⚠️"))) ||
		   strings.HasPrefix(trimmedLine, "═══") ||
		   (strings.Contains(trimmedLine, "🚫") && strings.Contains(trimmedLine, "BLOCKING")) {
			removeStart = lineStart
			// Don't break - keep looking backward in case there's an older header
		} else if strings.HasPrefix(trimmedLine, "#") && !strings.HasPrefix(trimmedLine, "##") {
			// Hit a different top-level heading, stop searching
			break
		}

		// Move to previous line
		if lineStart == 0 {
			break
		}
		searchPos = lineStart - 2 // Skip past the previous newline
	}

	// Remove from removeStart to end marker plus trailing newline
	endIdx += len(InstructionsMarkerEnd)
	if endIdx < len(content) && content[endIdx] == '\n' {
		endIdx++
	}

	return content[:removeStart] + content[endIdx:]
}

// AddInstructions adds juggler instructions to content
func AddInstructions(content string) string {
	return addInstructionsWithTemplate(content, InstructionsTemplate)
}

// AddGlobalInstructions adds minimal global instructions to content
func AddGlobalInstructions(content string) string {
	return addInstructionsWithTemplate(content, GlobalInstructionsTemplate)
}

// addInstructionsWithTemplate is the internal implementation for adding instructions
func addInstructionsWithTemplate(content, template string) string {
	// If content is empty, just return the instructions
	if strings.TrimSpace(content) == "" {
		return strings.TrimSpace(template) + "\n"
	}

	// Add instructions at the start with proper spacing
	// This ensures the CRITICAL BLOCKING REQUIREMENT is read first
	return strings.TrimSpace(template) + "\n\n" + content
}

// UpdateInstructions updates existing instructions or adds new ones
func UpdateInstructions(content string, isGlobal bool) string {
	// If instructions exist, remove them first
	if strings.Contains(content, InstructionsMarkerStart) {
		content = RemoveInstructions(content)
	}

	// Add new instructions
	if isGlobal {
		return AddGlobalInstructions(content)
	}
	return AddInstructions(content)
}

// WriteFile writes content to path, creating directories if needed
func WriteFile(path, content string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write file
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
