# Commands Reference

Complete CLI documentation for Juggle.

## Quick Reference

| Command                         | Description                                   |
| ------------------------------- | --------------------------------------------- |
| `juggle`                        | Launch interactive TUI (same as `juggle tui`) |
| `juggle tui`                    | Full-screen TUI for managing balls            |
| `juggle agent run [session]`    | Start autonomous agent loop                   |
| `juggle agent refine [session]` | AI-assisted acceptance criteria improvement   |
| `juggle plan`                   | Create a new ball via CLI                     |
| `juggle show <ball-id>`         | View ball details                             |
| `juggle update <ball-id>`       | Update ball properties                        |
| `juggle status`                 | List all balls across projects                |
| `juggle export`                 | Export balls (JSON, CSV, agent prompt)        |

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

# Work on specific ball only
juggle agent run --ball juggle-5
```

### Agent Run Flags

| Flag            | Short | Default | Description                                       |
| --------------- | ----- | ------- | ------------------------------------------------- |
| `--iterations`  | `-n`  | 10      | Maximum number of iterations                      |
| `--model`       | `-m`  | auto    | Model to use: `opus`, `sonnet`, `haiku`           |
| `--ball`        | `-b`  | -       | Work on a specific ball only                      |
| `--interactive` | `-i`  | false   | Run in interactive mode (full Claude TUI)         |
| `--timeout`     | `-T`  | 0       | Per-iteration timeout (e.g., `5m`, `1h`)          |
| `--trust`       | -     | false   | Skip permission prompts (dangerous!)              |
| `--delay`       | -     | 0       | Delay between iterations in minutes               |
| `--fuzz`        | -     | 0       | Random +/- variance in delay minutes              |
| `--dry-run`     | -     | false   | Show prompt info without running                  |
| `--debug`       | `-d`  | false   | Show prompt info before running                   |
| `--max-wait`    | -     | 0       | Maximum wait time for rate limits (0 = unlimited) |
| `--all`         | `-a`  | false   | Select from sessions across all projects          |

**Model auto-selection**: When `--model` is not specified:

- Large/opus for balls marked with `model_size: large`
- Sonnet for standard work
- Can be overridden per-ball via the `model_size` field

### Agent Refine

```bash
# AI-assisted acceptance criteria improvement
juggle agent refine my-feature

# Review balls across all projects
juggle agent refine --all
```

## Ball Properties

Each ball has:

- **Title**: Short description (shows in lists)
- **Context**: Background info for the agent
- **Acceptance Criteria**: Specific, testable conditions for completion
- **State**: `pending` → `in_progress` → `complete`/`researched` (or `blocked`)
- **Priority**: `low`, `medium`, `high`, `urgent`
- **Model Size**: `small` (haiku), `medium` (sonnet), `large` (opus)
- **Dependencies**: Other balls that must complete first
- **Tags**: For filtering and session grouping
- **Output**: Research results (for `researched` state)

## Configuration Commands

### Repository-Level Config

```bash
# Show all configuration
juggle config

# Manage acceptance criteria
juggle config ac list
juggle config ac add "All tests pass"
juggle config ac set --edit   # Open in $EDITOR
juggle config ac clear

# Manage VCS preference
juggle config vcs show
juggle config vcs set jj      # or "git"
juggle config vcs clear
```

### Global Config

```bash
# Manage iteration delay
juggle config delay show
juggle config delay set 5         # 5 minutes between iterations
juggle config delay set 5 --fuzz 2  # 5 ± 2 minutes
juggle config delay clear
```

## Workflow Commands

### Check Current State

```bash
# Get workflow guidance
juggle check
```

### Audit Project Health

```bash
# Analyze completion metrics
juggle audit

# Across all projects
juggle audit --all
```

## Project Management

### Worktree Support

For parallel agent execution in git worktrees:

```bash
# Register a worktree
juggle worktree add ../my-worktree

# List registered worktrees
juggle worktree list

# Check current directory status
juggle worktree status

# Unregister a worktree
juggle worktree forget ../my-worktree
```

### Move Balls Between Projects

```bash
# Transfer ball to another project
juggle move juggle-5 ~/other-project
```

### Unarchive Completed Balls

```bash
# Restore from archive to pending state
juggle unarchive juggle-5
```

## Sync Commands

### Sync with External Systems

```bash
# Sync prd.json status to balls
juggle sync ralph
```

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

## Export Formats

```bash
# Export session to JSON
juggle export --session my-feature --format json

# Export to CSV
juggle export --session my-feature --format csv

# Export as Ralph format (context + progress + tasks)
juggle export --session my-feature --format ralph

# Export as self-contained agent prompt
juggle export --session my-feature --format agent | claude -p
```

### Format Comparison

| Format  | Use Case                                                               |
| ------- | ---------------------------------------------------------------------- |
| `json`  | Data interchange, backups, programmatic access                         |
| `csv`   | Spreadsheet analysis, reporting                                        |
| `ralph` | Legacy agent prompts with structured sections                          |
| `agent` | Self-contained prompt for AI agents with full context and instructions |

### Export Filters

```bash
# Export specific balls
juggle export --ball-ids "juggle-5,48" --format json

# Export by state
juggle export --filter-state in_progress --format json

# Include completed balls (excluded by default)
juggle export --include-done --format json
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

### Run Commands Across Workspaces

Run any command in the main repo and all registered worktrees:

```bash
# Build in all workspaces
juggle worktree run "devbox run build"

# Run tests everywhere
juggle worktree run "go test ./..."

# Check VCS status across all
juggle worktree run "jj status"
```

| Flag | Description |
|------|-------------|
| `--continue-on-error` | Don't stop on first failure |

### Sync Settings Across Workspaces

Detect mismatches in `.claude/settings.local.json` and optionally symlink them to the main repo:

```bash
# Check for drift (dry run)
juggle worktree sync --dry-run

# Interactive sync with prompts
juggle worktree sync

# Auto-confirm all symlinks
juggle worktree sync --yes
```

| Flag | Description |
|------|-------------|
| `--dry-run` | Show what would be done without making changes |
| `-y, --yes` | Auto-confirm all symlink operations |

The sync command:
- Compares files between main repo and worktrees
- Shows status: `✓ symlinked`, `✓ identical`, `⚠ differs`, `- missing`
- Backs up existing files before replacing (with timestamp if `.bak` exists)
- Prompts for each differing file (y/n/all/quit)

**Note:** The `workspace` alias works for all worktree commands (e.g., `juggle workspace run`).

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

## Global Flags

These flags work with most commands:

| Flag            | Description                           |
| --------------- | ------------------------------------- |
| `--all`, `-a`   | Search across all discovered projects |
| `--json`        | Output as JSON                        |
| `--project-dir` | Override working directory            |
| `--config-home` | Override ~/.juggle directory          |
| `--juggle-dir`  | Override .juggle directory name       |
