# Workflow Enforcement Guide

This guide explains juggle's workflow enforcement system for Claude Code, designed to maintain task discipline and prevent workflow violations.

## Overview

Juggler's enforcement system uses **strict mode** with blocking language and automated checks to ensure AI agents follow the check → start → complete workflow cycle. Based on research showing 90-95% compliance with top-of-document blocking instructions, the system provides maximum effectiveness while remaining non-intrusive.

## Installation

### Quick Install (Recommended)

```bash
# Install with all enforcement features
cd your-project
juggle setup-claude --install-hooks
```

This creates:
- `.claude/CLAUDE.md` - Agent instructions with blocking requirements
- `.claude/hooks.json` - Pre-interaction hook configuration
- `/tmp/juggle-check-*` - Marker files for reminder tracking

### What Gets Installed

#### 1. Top-of-Document Blocking Instructions

Added to `.claude/CLAUDE.md`:

```markdown
═══════════════════════════════════════════════════════════════
🚫 CRITICAL BLOCKING REQUIREMENT - READ FIRST 🚫
═══════════════════════════════════════════════════════════════

## MANDATORY JUGGLER CHECK - YOU ARE BLOCKED UNTIL COMPLETE

**STOP.** You MUST run this command BEFORE ANY other action:

1. Run: `juggle` (shows currently juggling balls)
2. IF ready/juggling balls exist:
   - You are **BLOCKED** from other work
   - You **MUST** address existing balls FIRST
3. ONLY after handling existing balls may you proceed with new work
```

**Design Principles:**
- **Position**: Top of file (first thing agents see)
- **Visual separators**: `═══` lines for maximum visibility
- **Blocking language**: "YOU ARE BLOCKED" creates sense of requirement
- **Concrete steps**: Clear 1-2-3 instructions
- **Consequences**: Explicit violation consequences

#### 2. Pre-Interaction Hook

Added to `.claude/hooks.json`:

```json
{
  "hooks": [
    {
      "name": "juggle-reminder",
      "description": "Remind to check juggle workflow state",
      "command": "juggle reminder",
      "when": "pre-interaction",
      "shell": "bash"
    }
  ]
}
```

**How It Works:**
1. Runs before every Claude interaction
2. Checks marker file timestamp
3. Shows reminder if >5 minutes since last check
4. Performance: <50ms overhead
5. Non-blocking (failures don't break session)

#### 3. Marker File System

Files created in `/tmp/juggle-check-<hash>`:

**Format:**
```json
{
  "timestamp": "2025-10-27T10:30:00Z",
  "project": "/home/user/projects/myapp"
}
```

**Characteristics:**
- **Location**: `/tmp` (automatically cleaned on reboot)
- **Naming**: SHA256 hash of project path (first 16 hex chars)
- **Safety**: Atomic write operations (write temp, rename)
- **Performance**: Simple timestamp comparison

**Example:**
```bash
$ ls /tmp/juggle-check-*
/tmp/juggle-check-a1b2c3d4e5f6g7h8  # For project A
/tmp/juggle-check-9i8j7k6l5m4n3o2p  # For project B
```

## Commands

### `juggle check`

Interactive workflow helper that detects current state and provides guidance.

**Usage:**
```bash
juggle check
```

**What It Does:**
1. Discovers all projects with `.juggle` directories
2. Loads juggling and ready balls
3. Analyzes state and provides guidance
4. Offers interactive options for next steps

**Scenarios:**

#### Scenario 1: No Juggler Directory
```bash
$ juggle check
✅ No juggle directory found

Ready to start new work.

Initialize juggle:
  juggle plan    - Plan work for later
  juggle start   - Create and start juggling immediately
```

#### Scenario 2: No Active Balls
```bash
$ juggle check
✅ No active balls

Ready to start new work.

Create a ball:
  juggle start   - Create and start juggling immediately
  juggle plan    - Plan work for later
```

#### Scenario 3: Single Juggling Ball
```bash
$ juggle check
🎯 Currently juggling: myapp-5
Intent: Add user authentication
State: in-air

Is this what you're working on? (y/n): y

✓ Great! Continue working on this ball.
```

#### Scenario 4: Multiple Juggling Balls
```bash
$ juggle check
⚠️  Multiple balls juggling (3):

1. myapp-5: Add user authentication [in-air]
2. myapp-6: Fix login bug [needs-thrown]
3. myapp-7: Update documentation [needs-caught]

Which are you working on? (1-3): 1

✓ Working on: myapp-5 - Add user authentication

Consider moving other balls to ready:
  juggle <ball-id> ready
```

#### Scenario 5: Ready Balls Exist
```bash
$ juggle check
⚠️  Found 3 ready ball(s) that need attention:

1. myapp-8: Implement search feature [medium]
2. myapp-9: Add error handling [high]
3. myapp-10: Update API docs [low]

You should work on these before creating new balls.

What would you like to do?
1) Start working on a ready ball
2) View all ready balls
3) Drop some ready balls
4) Continue anyway (not recommended)

Choice (1-4): 1
```

**Updates Marker File:**
Running `juggle check` updates the marker file timestamp, resetting the reminder countdown.

### `juggle audit`

Analyzes project health and provides actionable recommendations.

**Usage:**
```bash
juggle audit
```

**What It Shows:**

#### Per-Project Metrics
```
/home/user/projects/myapp:
  Ready: 2
  Juggling: 1
  Dropped: 3
  Completed: 15
  Completion ratio: 75%
  Stale ready balls: 1 (>30 days old)
```

#### Recommendations
```
Recommendations:
• api-client: 2 stale ready balls - drop or start them
• frontend: Many balls juggling - consider completing some before starting more
• backend: High dropped count - review why work is being dropped
```

**Metrics Calculated:**
- **Completion ratio**: `completed / (total_non_complete + completed) * 100`
- **Stale threshold**: 30 days for ready balls
- **Warning levels**: <40% completion ratio triggers warning

**Use Cases:**
- Weekly workflow review
- Identifying problematic patterns
- Cleaning up stale work
- Project health monitoring

### `juggle reminder`

Checks if workflow reminder should be shown (used by hooks).

**Usage:**
```bash
juggle reminder
```

**Exit Codes:**
- **0**: Reminder should be shown
- **1**: Reminder not needed (recently checked)

**Output When Reminder Needed:**
```
⚠️  Workflow Check Recommended
Run 'juggle check' to verify current state
(helps maintain workflow discipline)
```

**Threshold:**
- **5 minutes** since last check
- Configurable via `ReminderThreshold` constant

**Performance:**
- File stat operation only
- No ball loading or analysis
- Typically <10ms

**Hook Integration:**
```json
{
  "command": "juggle reminder",
  "when": "pre-interaction"
}
```

Runs before each Claude interaction, shows reminder if threshold exceeded.

## Enforcement Philosophy

### Research-Based Design

The enforcement system is based on empirical research (WP3) showing:

1. **Top-of-Document Instructions: 90-95% Compliance**
   - Agents reliably see and follow first instructions
   - Critical requirement sections work best at document start
   - Decreasing effectiveness further down the document

2. **Blocking Language 5x More Effective**
   - "YOU ARE BLOCKED" vs "please consider"
   - Creates sense of requirement, not suggestion
   - Explicit consequences improve compliance

3. **Visual Separators Improve Visibility**
   - `═══` lines draw attention
   - Box borders create visual hierarchy
   - Emojis (🚫, ⚠️, ✅) aid scanning

### Strict Mode Philosophy

**Core Principle:** Prevent violations before they happen, not after.

**Why Strict Mode:**
- AI agents need explicit boundaries
- Gentle suggestions get ignored under task pressure
- Multiple concurrent balls reduce effectiveness
- Clear rules reduce decision fatigue

**What It Enforces:**
1. Check state before any work
2. Use existing balls when appropriate
3. Update state as work progresses
4. Complete balls when done

**What It Doesn't Enforce:**
- How you organize todos
- Priority levels you choose
- Project structure decisions
- Ball naming conventions

### Performance Characteristics

**Hook Overhead:**
- Marker file stat: ~1-5ms
- File read + JSON parse: ~10-20ms
- Total overhead: <50ms per interaction
- Non-blocking: failures don't break session

**Marker File I/O:**
- Atomic operations (write temp, rename)
- Safe for concurrent access
- Auto-cleanup on system reboot
- No file locking needed

**Scalability:**
- Per-project marker files (no global bottleneck)
- SHA256 hashing prevents collisions
- Works with hundreds of projects

## Troubleshooting

### Reminder Not Showing

**Problem:** Hook doesn't show reminder
**Solutions:**
```bash
# Check hook is installed
cat .claude/hooks.json

# Manually test reminder
juggle reminder

# Check marker file exists
ls /tmp/juggle-check-*

# Force new check
rm /tmp/juggle-check-*
juggle check
```

### Too Many Reminders

**Problem:** Reminder shows too frequently
**Solutions:**
```bash
# Check marker file timestamp
cat /tmp/juggle-check-$(echo -n $(pwd) | sha256sum | cut -c1-16)

# Adjust threshold in code (requires rebuild)
# Edit internal/session/reminder.go
# const ReminderThreshold = 10 * time.Minute  // Increase to 10 min
```

### Hook Failures

**Problem:** Hook command fails
**Solutions:**
```bash
# Check juggle is in PATH
which juggle

# Test command manually
bash -c "juggle reminder"

# Check hook syntax
jq . .claude/hooks.json

# Reinstall hooks
juggle setup-claude --install-hooks --force
```

### Instructions Not Followed

**Problem:** Agent ignores instructions
**Solutions:**
1. Verify instructions at top of CLAUDE.md (not buried)
2. Check for conflicting instructions
3. Ensure visual separators intact
4. Restart Claude Code after changes
5. Use `--update` flag to refresh instructions

### Marker Files Accumulate

**Problem:** Many marker files in /tmp
**Solutions:**
```bash
# These are automatically cleaned on reboot
# Manual cleanup if needed:
rm /tmp/juggle-check-*

# Or clean old markers (>7 days)
find /tmp -name "juggle-check-*" -mtime +7 -delete
```

## Advanced Configuration

### Custom Reminder Threshold

Edit `internal/session/reminder.go`:

```go
// ReminderThreshold is how long before showing reminder again
const ReminderThreshold = 10 * time.Minute  // Changed from 5
```

Rebuild:
```bash
go build -o ~/.local/bin/juggle ./cmd/juggle
```

### Custom Instructions Template

Edit `internal/claude/instructions.go`:

```go
const InstructionsTemplate = `
Your custom blocking instructions here...
`
```

Reinstall:
```bash
juggle setup-claude --install-hooks --update --force
```

### Disable Hook

Temporarily disable without uninstalling:

```bash
# Comment out hook in .claude/hooks.json
{
  "hooks": [
    // {
    //   "name": "juggle-reminder",
    //   ...
    // }
  ]
}
```

### Global vs Local Installation

**Local** (`.claude/CLAUDE.md`):
- Project-specific instructions
- Includes hook installation
- Recommended for most users

**Global** (`~/.claude/CLAUDE.md`):
- Instructions available in all projects
- Points to project-specific CLAUDE.md
- No hooks (hooks are project-specific)
- Use when working across many projects

**Best Practice:**
```bash
# Global instructions
juggle setup-claude --global

# Then in each project:
juggle setup-claude --install-hooks
```

## Integration Examples

### Example 1: New Project Setup

```bash
# Initialize project
cd myapp
git init

# Install juggle integration
juggle setup-claude --install-hooks

# Verify installation
ls .claude/
# CLAUDE.md  hooks.json

# Start first ball
juggle start "Initial project setup"
```

### Example 2: Existing Project

```bash
# Already has .claude/CLAUDE.md with other instructions
cd existing-project

# Update to add juggle (preserves existing content)
juggle setup-claude --install-hooks

# Instructions are added at top (critical position)
head -20 .claude/CLAUDE.md
```

### Example 3: Team Workflow

```bash
# Commit juggle config to repo
git add .claude/
git commit -m "Add juggle workflow enforcement"

# Team members clone repo
git clone ...
cd project

# Juggler hooks work automatically (no per-user setup)
# Marker files are per-user (in /tmp)
```

## Best Practices

### 1. Run Check Before Each Session

```bash
# Start of day
juggle check

# Shows what's active
# Prompts for next steps
# Updates marker file
```

### 2. Regular Audits

```bash
# Weekly review
juggle audit

# Clean up stale work
# Review completion ratios
# Identify patterns
```

### 3. Complete Balls Promptly

```bash
# When work is done
juggle <ball-id> complete "Summary of changes"

# Don't leave balls in needs-caught indefinitely
# Maintains accurate project state
```

### 4. Use Tags for Filtering

```bash
# Tag balls appropriately
juggle <ball-id> tag add bug urgent backend

# Audit by category
juggle audit --tags bug
```

### 5. Keep Instructions Updated

```bash
# After juggle updates
juggle setup-claude --install-hooks --update

# Restarts Claude Code after update
```

## Summary

Juggler's workflow enforcement system provides:

✅ **High Effectiveness** - 90-95% compliance via blocking instructions
✅ **Low Overhead** - <50ms per interaction
✅ **Non-Intrusive** - 5-minute reminder threshold
✅ **Atomic Safety** - Safe concurrent access
✅ **Clear Guidance** - Interactive help at every step

The system maintains workflow discipline without being annoying, preventing violations before they happen through clear requirements and automated reminders.

For more information:
- [Claude Integration Guide](./claude-integration.md) - User workflow patterns
- [Agent Integration Guide](./agent-integration.md) - AI agent usage guide
- [Installation Guide](./installation.md) - Setup instructions
