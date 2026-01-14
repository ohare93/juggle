# Commands Reference

Complete CLI documentation for Juggle.

## Quick Reference

| Command | Description |
|---------|-------------|
| `juggle` | Launch interactive TUI (same as `juggle tui`) |
| `juggle tui` | Full-screen TUI for managing balls |
| `juggle agent run [session]` | Start autonomous agent loop |
| `juggle agent refine [session]` | AI-assisted acceptance criteria improvement |
| `juggle plan` | Create a new ball via CLI |
| `juggle show <ball-id>` | View ball details |
| `juggle update <ball-id>` | Update ball properties |
| `juggle status` | List all balls across projects |
| `juggle export` | Export balls (JSON, CSV, agent prompt) |

## Sessions

Sessions group related balls and provide:

- **Session-level acceptance criteria** (inherited by all balls)
- **Progress tracking** across the session
- **Scoped agent runs** (`juggle agent run my-feature`)

### Session Commands

```bash
# Create session
juggle sessions create my-feature --ac "All tests pass" --ac "No linting errors"

# List sessions
juggle sessions list

# Show session details
juggle sessions show my-feature

# Edit session
juggle sessions edit my-feature

# Delete session
juggle sessions delete my-feature

# Run agent for session
juggle agent run my-feature
```

## Creating Balls

### Via TUI (Recommended)

```bash
juggle tui
# Press 'n' to create new ball
# Fill in: Title, Context, Acceptance Criteria
```

### Via CLI

```bash
juggle plan --session my-feature \
  --title "Add user authentication" \
  --context "We need OAuth2 with Google" \
  --ac "Login button on homepage" \
  --ac "JWT tokens stored in httpOnly cookies" \
  --ac "Tests pass"
```

## Agent Commands

### Running the Agent Loop

```bash
# Interactive session selector
juggle agent run

# Specify session directly
juggle agent run my-feature

# Work on ALL balls in repo (no session filter)
juggle agent run all
```

### Agent Flags

```bash
juggle agent run my-feature \
  --iterations 5            # Max iterations (default: 10)
  --model sonnet           # Model: opus, sonnet, haiku
  --ball juggle-123       # Work on specific ball only
  --interactive            # Full Claude TUI (not headless)
  --timeout 5m             # Per-iteration timeout
  --trust                  # Skip permission prompts (dangerous)
  --delay 5                # Minutes between iterations
  --fuzz 2                 # Random delay variance (+/- minutes)
```

### Refining Balls

```bash
# AI-assisted acceptance criteria improvement
juggle agent refine my-feature
```

## Ball Properties

Each ball has:

- **Title**: Short description (shows in lists)
- **Context**: Background info for the agent
- **Acceptance Criteria**: Specific, testable conditions for completion
- **State**: `pending` → `in_progress` → `complete` (or `blocked`)
- **Priority**: `low`, `medium`, `high`, `urgent`
- **Model Size**: `small` (haiku), `medium` (sonnet), `large` (opus)
- **Dependencies**: Other balls that must complete first
- **Tags**: For filtering and session grouping

## TUI Keyboard Shortcuts

### Navigation
- `j/k` or `↓/↑` - Move up/down
- `Enter` - View/edit ball
- `Esc` - Back/cancel
- `?` - Help

### Ball State (two-key sequences)
- `sc` - Mark complete
- `ss` - Mark in_progress (start)
- `sb` - Mark blocked
- `sp` - Mark pending
- `sa` - Archive (complete + hide)

### Filters (two-key sequences)
- `tc` - Toggle complete visibility
- `tb` - Toggle blocked
- `ti` - Toggle in_progress
- `tp` - Toggle pending
- `ta` - Show all

### Agent Output
- `O` - Toggle output panel
- `X` - Cancel running agent
- `H` - View agent history

## Export

```bash
# Export session to JSON
juggle export --session my-feature --format json

# Export to CSV
juggle export --session my-feature --format csv

# Export as agent prompt (Ralph format)
juggle export --session my-feature --format agent
```

## File Structure

```
your-project/
├── .juggle/
│   ├── balls.jsonl           # Active balls
│   ├── config.json           # Project config
│   ├── archive/
│   │   └── balls.jsonl       # Completed balls
│   └── sessions/
│       └── my-feature/
│           ├── session.json  # Session config
│           ├── progress.txt  # Agent progress log
│           └── last_output.txt

~/.juggle/
├── config.json               # Global config (search paths)
```
