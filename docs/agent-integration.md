# Agent Integration Guide

This guide is designed for AI agents using juggler to manage tasks across conversations.

## Installation for Agents

Users can install juggler instructions with:

```bash
# Install to local project
juggle setup-claude

# Install to global configuration
juggle setup-claude --global

# Preview without installing
juggle setup-claude --dry-run
```

This adds comprehensive instructions to CLAUDE.md that teach you when and how to use juggler.

## The Juggling Metaphor

Juggler uses a metaphor of juggling balls to represent concurrent tasks:

### Active States

- **ready**: Ball is ready to start juggling (planned work)
- **juggling**: Ball is currently being juggled (active work)
  - **needs-thrown**: Needs user direction/input
  - **in-air**: Agent is actively working
  - **needs-caught**: Agent finished, needs user verification
- **dropped**: Ball was explicitly dropped (abandoned)
- **complete**: Ball successfully caught and complete (done)

### State Flow

```
ready → juggling:needs-thrown → juggling:in-air → juggling:needs-caught → complete
                                        ↓
                                     dropped
```

## When to Use Juggler

### At Conversation Start

**ALWAYS** check current state first:

```bash
juggle
```

This shows:
- Current ball (if any)
- Ball state and intent
- Todos and progress
- Last activity timestamp

### Decision Tree

```
1. Is there a ball that matches this user's task?
   ├─ YES → Use existing ball, review state and todos
   └─ NO → Check if user has a clear task
      ├─ YES → Create new ball with `juggle start`
      └─ NO → Don't create a ball yet, wait for clarity
```

**IMPORTANT**: Only create a new ball if:
- No existing ball matches the user's current task
- The user has a clear, defined task to work on
- You're starting actual work (not just chatting)

## State Management

### When to Update States

**needs-thrown** - Use before asking user for input:
```bash
juggle <ball-id> needs-thrown "Need database schema approval"
```

**in-air** - Use when you start active work:
```bash
juggle <ball-id> in-air
```

**needs-caught** - Use after completing work that needs verification:
```bash
juggle <ball-id> needs-caught "Implemented authentication, ready for review"
```

**complete** - Use when task is fully done:
```bash
juggle <ball-id> complete "Feature complete with tests passing"
```

**dropped** - Use when task is abandoned:
```bash
juggle <ball-id> dropped "User decided not to pursue this approach"
```

## Todo Management

### Adding Todos

When you identify subtasks, add them all at once:

```bash
juggle <ball-id> todo add \
  "Create User model" \
  "Implement password hashing" \
  "Add JWT generation" \
  "Write tests"
```

### Marking Complete

As you finish tasks:

```bash
juggle <ball-id> todo done 1
juggle <ball-id> todo done 2
```

### Viewing Progress

Check progress at any time:

```bash
juggle <ball-id>
```

Shows: `Todos: 2/4 complete (50%)`

## Common Workflows

### Workflow 1: New Task

```bash
# User: "Help me add authentication"

# Step 1: Check current state
juggle

# Step 2: No matching ball exists, create one
juggle start
# (interactive prompts for intent, priority)

# Step 3: Add todos for task breakdown
juggle <ball-id> todo add \
  "Design auth schema" \
  "Implement login endpoint" \
  "Add JWT middleware" \
  "Write tests"

# Step 4: Start working
juggle <ball-id> in-air

# Step 5: As you complete tasks
juggle <ball-id> todo done 1
# ... do work ...
juggle <ball-id> todo done 2

# Step 6: When done
juggle <ball-id> needs-caught "Auth implemented, needs testing"
```

### Workflow 2: Resuming Work

```bash
# User: "Let's continue with the authentication work"

# Step 1: Check current state
juggle

# Output shows:
# Ball: myapp-5
# Intent: Add authentication system
# State: juggling:needs-thrown (waiting for schema approval)
# Todos: 1/4 complete (25%)

# Step 2: Continue from where you left off
# Review todos, resume work
juggle <ball-id> in-air
```

### Workflow 3: Blocked Work

```bash
# You need user input

# Step 1: Mark as needs-thrown
juggle <ball-id> needs-thrown "Which authentication method? OAuth or JWT?"

# Step 2: Ask user for input
# (agent asks question)

# Step 3: When user responds, continue
juggle <ball-id> in-air
```

## Multiple Balls

Projects can have multiple balls. When multiple exist:

### Listing All Balls

```bash
juggle balls
```

Shows all balls in current project with states and priorities.

### Finding Next Ball

```bash
juggle next
```

Intelligently determines which ball needs attention based on:
1. State (needs-caught has highest priority)
2. Priority level
3. Last activity time

### Specifying Ball ID

When multiple balls exist, use explicit ball IDs:

```bash
# List balls first
juggle balls

# Work with specific ball
juggle myapp-5 in-air
juggle myapp-5 todo add "New task"
```

## Ball IDs

Ball IDs follow the format: `<directory-name>-<counter>`

Examples:
- `juggler-1`
- `myapp-5`
- `api-client-12`

The counter increments for each new ball in that directory.

## Best Practices

### 1. Check State First

Always start conversations with:
```bash
juggle
```

### 2. Use Descriptive Intents

```bash
# Good
juggle start --intent "Add OAuth2 authentication with Google provider"

# Less helpful
juggle start --intent "Fix stuff"
```

### 3. Break Down Work

Add todos for complex tasks:
```bash
juggle <ball-id> todo add \
  "Research OAuth2 flow" \
  "Set up Google OAuth credentials" \
  "Implement /auth/google endpoint" \
  "Add callback handler" \
  "Store tokens securely" \
  "Test login flow"
```

### 4. Update State Proactively

Don't wait for user to ask. Update state as you work:
- Before asking questions: `needs-thrown`
- When actively coding: `in-air`
- After completing work: `needs-caught`

### 5. Use Tags for Organization

```bash
juggle <ball-id> tag add backend authentication security
```

Later search by tags:
```bash
juggle search --tags authentication
```

## Common Commands Reference

| Command | Purpose | Example |
|---------|---------|---------|
| `juggle` | Show current ball | `juggle` |
| `juggle balls` | List all balls | `juggle balls` |
| `juggle start` | Create new ball | `juggle start` |
| `juggle <ball-id> <state>` | Update state | `juggle myapp-5 in-air` |
| `juggle <ball-id> todo add` | Add todos | `juggle myapp-5 todo add "Task"` |
| `juggle <ball-id> todo done` | Complete todo | `juggle myapp-5 todo done 1` |
| `juggle next` | Find next ball | `juggle next` |
| `juggle search` | Search balls | `juggle search auth` |

## Error Handling

### "No current ball found"

This means:
- No balls exist in current project, OR
- Multiple balls exist and you need to specify which one

Solutions:
```bash
# List all balls
juggle balls

# Create new ball
juggle start

# Use explicit ball ID
juggle myapp-5 todo add "Task"
```

### "Ball not found"

The ball ID doesn't exist. Check:
```bash
# List all balls
juggle balls

# Search for balls
juggle search <keyword>
```

## Example Conversation Flow

```
User: "I want to add a search feature to the app"

Agent: Let me check the current project state.
[runs: juggle]

Agent: I see there's no active ball for this task. Let me create one.
[runs: juggle start]
[interactively sets intent: "Add search feature with filters"]

Agent: I've created a new ball. Let me break down this task into steps.
[runs: juggle myapp-6 todo add "Design search API" "Implement backend search" "Add frontend UI" "Add tests"]

Agent: I'll start with the search API design.
[runs: juggle myapp-6 in-air]
[does work...]

Agent: I've designed the search API. Let me mark that complete.
[runs: juggle myapp-6 todo done 1]

Agent: Now implementing the backend...
[does work...]

Agent: Backend is complete. The feature needs your testing.
[runs: juggle myapp-6 needs-caught "Search implemented, please test"]

User: [tests and approves]

Agent: Great! Marking as complete.
[runs: juggle myapp-6 complete "Search feature complete and tested"]
```

## Integration with Claude Code

If installed via `juggle setup-claude`, these instructions are automatically available in your CLAUDE.md file. The activity tracking hook (if installed) automatically updates timestamps when you interact, helping juggler track which balls are active.

## Summary

1. **Always check state first**: `juggle`
2. **Only create balls when needed**: User has clear task
3. **Use explicit ball IDs**: When multiple balls exist
4. **Update state proactively**: Reflect your current activity
5. **Break down complex work**: Use todos for task tracking
6. **Complete balls when done**: `juggle <ball-id> complete`

This creates a seamless workflow where the agent maintains context across conversations and the user can easily see what's being worked on.
