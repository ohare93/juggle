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
- `Tab` / `l` - Next panel (Sessions → Balls → Activity)
- `Shift+Tab` / `h` - Previous panel
- `j/k` or `↓/↑` - Move up/down
- `Enter` - Select item / Edit ball
- `Space` - Go back (in Balls panel)
- `Esc` - Back/deselect/close
- `?` - Help

### Ball State (two-key sequences with `s`)
- `sc` - Mark complete (archives the ball)
- `ss` - Mark in_progress (start)
- `sb` - Mark blocked (prompts for reason)
- `sp` - Mark pending
- `sa` - Archive completed ball

### Filters (two-key sequences with `t`)
- `tc` - Toggle complete visibility
- `tb` - Toggle blocked visibility
- `ti` - Toggle in_progress visibility
- `tp` - Toggle pending visibility
- `ta` - Show all states

### Ball Management
- `a` - Add new ball (tagged to current session)
- `A` - Add followup ball (depends on selected ball)
- `e` - Edit ball in $EDITOR (YAML format)
- `d` - Delete ball (with confirmation)
- `[ / ]` - Switch session (previous / next)
- `o` - Toggle sort order
- `/` - Filter balls
- `Ctrl+U` - Clear filter

### View Options
- `i` - Cycle bottom pane (activity → detail → split)
- `O` - Toggle agent output panel
- `P` - Toggle project scope (local ↔ all projects)
- `R` - Refresh/reload data

### Agent Control
- `X` - Cancel running agent (with confirmation)
- `O` - Toggle agent output visibility
- `H` - View agent run history

## Export

```bash
# Export session to JSON
juggle export --session my-feature --format json

# Export to CSV
juggle export --session my-feature --format csv

# Export as agent prompt (Ralph format)
juggle export --session my-feature --format agent
```

## Configuration

### VCS Settings

Juggle auto-detects version control (`.jj` preferred over `.git`), but you can override:

```bash
# Show current settings and detection
juggle config vcs show

# Set globally
juggle config vcs set jj
juggle config vcs set git

# Set per-project
juggle config vcs set jj --project

# Clear to use auto-detection
juggle config vcs clear
juggle config vcs clear --project
```

### Acceptance Criteria (repo-level)

```bash
# List current criteria
juggle config ac list

# Add criterion
juggle config ac add "All tests must pass"

# Edit in $EDITOR
juggle config ac set --edit

# Clear all
juggle config ac clear
```

### Agent Iteration Delay

```bash
# Show delay settings
juggle config delay show

# Set 5 minute delay
juggle config delay set 5

# Set 5 ± 2 minute delay (range: 3-7 min)
juggle config delay set 5 --fuzz 2

# Clear delay
juggle config delay clear
```

## Worktrees

Manage worktree links for running parallel agent loops across different VCS worktrees while sharing the same ball state.

```bash
# Register a worktree (run from main repo)
juggle worktree add ../my-feature-worktree

# List registered worktrees
juggle worktree list

# Check worktree status for current directory
juggle worktree status

# Unregister a worktree (doesn't delete it)
juggle worktree forget ../my-feature-worktree
```

**Note:** The `workspace` alias works for all worktree commands (e.g., `juggle workspace add`).

See [Installation Guide - Worktrees](./installation.md#worktrees-parallel-agent-loops) for full setup instructions.

## File Structure

```
your-project/
├── .juggle/
│   ├── balls.jsonl           # Active balls
│   ├── config.json           # Project config (vcs, acceptance criteria)
│   ├── archive/
│   │   └── balls.jsonl       # Completed balls
│   └── sessions/
│       └── my-feature/
│           ├── session.json  # Session config
│           ├── progress.txt  # Agent progress log
│           └── last_output.txt

~/.juggle/
├── config.json               # Global config (search paths, vcs, delay)
```
