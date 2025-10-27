# Claude Code Integration Guide

This guide explains how to integrate Juggler with Claude Code for AI-assisted task management.

## Installation for AI Agents

### Recommended: Install with Workflow Enforcement

```bash
# Install instructions + enforcement hooks (recommended)
cd your-project
juggle setup-claude --install-hooks

# Preview before installing
juggle setup-claude --install-hooks --dry-run
```

This installs:
- **Agent instructions** in `.claude/CLAUDE.md`
- **Workflow enforcement** with strict mode
- **Pre-interaction hook** for automated reminders
- **Marker file system** for check tracking

### Alternative: Instructions Only

```bash
# Install to local project
juggle setup-claude

# Or install globally for all projects
juggle setup-claude --global
```

### What Gets Installed

**Top-of-Document Blocking Instructions:**
- Critical requirement section at document start
- Visual separators (═══) for maximum visibility
- Explicit "YOU ARE BLOCKED" language (90-95% effective)
- Concrete check → start → complete workflow

**Pre-Interaction Hook** (with `--install-hooks`):
- Runs before each Claude interaction
- Shows reminder if >5 minutes since last check
- <50ms performance overhead
- Non-blocking (failures don't break session)

**Marker Files** (with `--install-hooks`):
- Located in `/tmp/juggle-check-*`
- Track last check timestamp per project
- SHA256-based project hashing
- Atomic write operations for safety

See [Workflow Enforcement Guide](./workflow-enforcement.md) for complete details on the enforcement system and [Agent Integration Guide](./agent-integration.md) for detailed agent usage patterns.

## Overview

Juggler is designed to work seamlessly with Claude Code, allowing you to:

- Let Claude automatically break down complex tasks into subtasks
- Have Claude add all planned tasks at once using batch operations
- Track progress with visual todo lists and completion percentages
- Maintain context across multiple work sessions
- **Enforce workflow discipline** through automated checks and reminders

## Workflow Enforcement Features

When installed with `--install-hooks`, juggler maintains workflow discipline:

### The Check → Start → Complete Cycle

```bash
# 1. Check state before any work
$ juggle check
✅ No active balls
Ready to start new work.

# 2. Start new ball only if needed
$ juggle start "Add user authentication"
✓ Started ball: myapp-5

# 3. Work proceeds with state updates
$ juggle myapp-5 in-air
$ juggle myapp-5 needs-caught "Ready for review"

# 4. Complete when done
$ juggle myapp-5 complete "Auth implemented and tested"
✓ Ball marked as complete
```

### Automated Reminders

The pre-interaction hook shows reminders when needed:

```
⚠️  Workflow Check Recommended
Run 'juggle check' to verify current state
(helps maintain workflow discipline)
```

**Reminder Logic:**
- Shows if >5 minutes since last check
- Only in projects with `.juggler` directories
- Non-intrusive (doesn't block work)
- <50ms performance overhead

### Interactive Guidance

The `juggle check` command provides context-aware guidance:

**With Existing Balls:**
```
⚠️  Found 3 ready ball(s) that need attention:

1. myapp-8: Implement search [medium]
2. myapp-9: Add error handling [high]
3. myapp-10: Update docs [low]

What would you like to do?
1) Start working on a ready ball
2) View all ready balls
3) Drop some ready balls
4) Continue anyway (not recommended)
```

**With Multiple Juggling:**
```
⚠️  Multiple balls juggling (2):

1. myapp-5: Add authentication [in-air]
2. myapp-6: Fix login bug [needs-thrown]

Which are you working on? (1-2):
```

See [Workflow Enforcement Guide](./workflow-enforcement.md) for complete enforcement system documentation.

## Quick Start

### 1. Start a New Session

When beginning work with Claude:

```bash
juggle start "Implement user authentication system"
```

### 2. Let Claude Add Tasks

When Claude identifies tasks, it can add them all at once:

```bash
# Claude can run this command with multiple tasks
juggle todo add \
  "Create User model with email and password fields" \
  "Implement password hashing with bcrypt" \
  "Add JWT token generation" \
  "Create login and register endpoints" \
  "Write unit tests for auth functions"
```

### 3. Track Progress

View your current ball and todos:

```bash
juggle show
```

Check status across all balls:

```bash
juggle status
```

### 4. Mark Tasks Complete

As you complete tasks:

```bash
juggle todo done 1
juggle todo done 2
```

## Multi-Ball Context Resolution

When working in a repository with multiple active balls, Juggler intelligently determines which ball to use:

1. **Explicit selection**: Use `--ball <id>` flag
2. **Zellij tab match**: If running in Zellij, matches the current tab name
3. **Most recent active**: Falls back to the most recently updated ball

This means Claude can manage todos without you having to specify which ball every time.

## Best Practices with Claude

### Task Breakdown Pattern

When working with Claude on a complex feature:

1. **Start with intent**: Create a ball with high-level goal
```bash
juggle start "Build real-time chat feature"
```

2. **Let Claude plan**: Ask Claude to break it down
```
You: "What tasks are needed to implement this?"
Claude: [analyzes and uses `juggle todo add` with all tasks]
```

3. **Work iteratively**: Check off tasks as you complete them
```bash
juggle todo done 1
juggle show  # See remaining work
```

### Using Tags for Organization

Add tags to help Claude filter and organize:

```bash
# Claude can add context tags
juggle tag add backend database
juggle tag add frontend ui-component

# Later, filter by tags
juggle status --tags backend
juggle search --tags database
```

### Priority Management

Set priorities for Claude to understand urgency:

```bash
juggle edit <ball-id> --priority urgent
juggle status --priority urgent  # Show urgent items
```

## Workflow Examples

### Example 1: Feature Development

```bash
# Start feature ball
juggle start "Add export functionality"

# Claude adds breakdown
juggle todo add \
  "Design export data structure" \
  "Implement JSON export" \
  "Implement CSV export" \
  "Add export command to CLI" \
  "Write tests for export functions"

# Work through tasks
juggle todo done 1
# ... implement ...
juggle todo done 2

# Check progress
juggle show
# Shows: Todos: 2/5 complete (40%)
```

### Example 2: Bug Fixing

```bash
# Start bug fix session
juggle start "Fix memory leak in session tracking"
juggle tag add bug hotfix
juggle edit <ball-id> --priority high

# Add investigation tasks
juggle todo add \
  "Profile memory usage" \
  "Identify leak source" \
  "Implement fix" \
  "Verify fix with profiler"

# Track progress
juggle todo list
juggle todo done 1
```

### Example 3: Multiple Parallel Tasks

```bash
# Start multiple work streams
juggle start "Refactor authentication"
juggle start "Update documentation"
juggle start "Performance optimization"

# Claude can work on any by specifying --ball
juggle todo add --ball auth-ball-id \
  "Extract auth logic to middleware" \
  "Update tests"

# Or rely on Zellij tab matching
# (If you're in the correct tab, no --ball needed)
juggle todo add "Add performance benchmarks"

# View all active work
juggle status
```

## Advanced Features

### Search and History

Find specific work:

```bash
# Search active balls
juggle search authentication
juggle search --tags backend --priority high

# Query archived work
juggle history bug
juggle history --after 2025-10-01
juggle history --stats  # See completion statistics
```

### Export for Analysis

Export data for further analysis:

```bash
# Export to JSON
juggle export --format json --output backup.json

# Export to CSV for spreadsheet analysis
juggle export --format csv --output analysis.csv --include-done
```

### Edit Ball Properties

Modify ball details on the fly:

```bash
# Interactive edit
juggle edit <ball-id>

# Direct edit
juggle edit <ball-id> --priority urgent --status blocked
juggle edit <ball-id> --intent "Updated description"
```

## Integration Tips

### 1. Leverage Batch Operations

Always use multi-task adds when Claude identifies multiple subtasks:

```bash
# Good: Single command for all tasks
juggle todo add "Task 1" "Task 2" "Task 3"

# Avoid: Multiple separate commands
juggle todo add "Task 1"
juggle todo add "Task 2"
juggle todo add "Task 3"
```

### 2. Use Clear, Actionable Task Descriptions

Write tasks as concrete actions:

```bash
# Good
juggle todo add \
  "Create database migration for users table" \
  "Implement password hashing in auth service" \
  "Add /login endpoint to API"

# Less specific
juggle todo add \
  "Database stuff" \
  "Security" \
  "API"
```

### 3. Regular Status Checks

Incorporate status checks into your workflow:

```bash
# Quick overview
juggle status

# Detailed view of current ball
juggle show

# Filter to what matters now
juggle status --priority high --tags urgent
```

### 4. Archive Completed Work

Mark balls done when finished:

```bash
juggle done "Successfully implemented feature X"
```

This keeps your active list clean and builds a searchable history.

## Troubleshooting

### "No current ball found"

This happens when:
- No balls are active in the current project
- You're in a project without `.juggler` directory

Solution:
```bash
juggle start "New work session"
# or
juggle todo add --ball <explicit-id> "Task"
```

### Multiple balls with same intent

Use descriptive, unique intents:

```bash
# Good
juggle start "Fix login bug in mobile app"
juggle start "Fix login bug in web app"

# Confusing
juggle start "Fix login bug"
juggle start "Fix login bug"
```

### Lost track of ball IDs

Use search to find them:

```bash
juggle search <keyword>
juggle status  # Shows all active balls
```

## Summary

Juggler + Claude Code provides powerful task management:

- ✅ Batch todo operations for efficient task planning
- ✅ Intelligent context resolution across multiple balls
- ✅ Visual progress tracking with checkboxes and percentages
- ✅ Flexible filtering and search capabilities
- ✅ Complete history and analytics

Start using Juggler with Claude today to stay organized and productive!
