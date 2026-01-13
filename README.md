# Juggler

A task runner for AI agents. Define tasks with acceptance criteria, let agents execute them autonomously.

## What It Does

Juggler is the "ralph loop" with nice tooling:

1. **Define tasks** ("balls") with clear acceptance criteria via TUI or CLI
2. **Run `juggle agent run`** to start an autonomous agent loop
3. **Add/edit tasks while it runs** - the TUI lets you manage work without touching JSON
4. **Agent refines tasks** - use `juggle agent refine` to get AI help improving your ACs

```
                    ┌─────────────────────┐
                    │   juggle agent run  │
                    │   (autonomous loop) │
                    └──────────┬──────────┘
                               │
          ┌────────────────────┼────────────────────┐
          │                    │                    │
          ▼                    ▼                    ▼
    ┌───────────┐       ┌───────────┐       ┌───────────┐
    │  Ball 1   │       │  Ball 2   │       │  Ball 3   │
    │ pending   │       │ in_progress│      │ complete  │
    └───────────┘       └───────────┘       └───────────┘
          │                    │
          │                    │
          ▼                    ▼
    ┌─────────────────────────────────────────────────┐
    │              TUI (juggle tui)                   │
    │   View • Add • Edit • Filter while running     │
    └─────────────────────────────────────────────────┘
```

## Installation

### Quick Install

```bash
curl -sSL https://raw.githubusercontent.com/ohare93/juggler/main/install.sh | bash
```

### Build from Source

```bash
devbox shell
go build -o ~/.local/bin/juggle ./cmd/juggle
```

## Quick Start

### 1. Create a session and some tasks

```bash
cd ~/your-project
juggle sessions create my-feature

# Add tasks via TUI (recommended)
juggle tui

# Or via CLI
juggle plan --session my-feature \
  --title "Add user authentication" \
  --context "We need OAuth2 with Google" \
  --ac "Login button on homepage" \
  --ac "JWT tokens stored in httpOnly cookies" \
  --ac "Tests pass"
```

### 2. Run the agent loop

```bash
# Interactive session selector
juggle agent run

# Or specify session directly
juggle agent run my-feature

# Work on ALL balls in repo (no session filter)
juggle agent run all
```

The agent will:
- Pick a pending ball based on priority and dependencies
- Execute until acceptance criteria are met
- Mark complete and move to next ball
- Signal BLOCKED if it can't proceed

### 3. Manage tasks while it runs

Open another terminal:

```bash
juggle tui
```

From the TUI you can:
- View agent output in real-time (`O` to toggle output panel)
- Add new tasks (`n`)
- Edit existing tasks (`Enter` on a ball)
- Change priorities, add tags, modify ACs
- Filter by state: pending, in_progress, blocked, complete

## Key Commands

| Command | Description |
|---------|-------------|
| `juggle tui` | Full-screen TUI for managing balls |
| `juggle agent run [session]` | Start autonomous agent loop |
| `juggle agent refine [session]` | AI-assisted AC improvement |
| `juggle plan` | Create a new ball via CLI |
| `juggle show <ball-id>` | View ball details |
| `juggle update <ball-id>` | Update ball properties |
| `juggle status` | List all balls across projects |
| `juggle export` | Export balls (JSON, CSV, agent prompt) |

## Ball Properties

Each ball ("task") has:

- **Title**: Short description (shows in lists)
- **Context**: Background info for the agent
- **Acceptance Criteria**: Specific, testable conditions for completion
- **State**: `pending` → `in_progress` → `complete` (or `blocked`)
- **Priority**: `low`, `medium`, `high`, `urgent`
- **Model Size**: `small` (haiku), `medium` (sonnet), `large` (opus)
- **Dependencies**: Other balls that must complete first
- **Tags**: For filtering and session grouping

## Sessions

Sessions group related balls and provide:

- **Session-level acceptance criteria** (inherited by all balls)
- **Progress tracking** across the session
- **Scoped agent runs** (`juggle agent run my-feature`)

```bash
# Create session
juggle sessions create my-feature --ac "All tests pass" --ac "No linting errors"

# List sessions
juggle sessions list

# Run agent for session
juggle agent run my-feature
```

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

## Workflow Example

```bash
# Start a feature session
cd ~/my-project
juggle sessions create auth-system

# Add balls via TUI
juggle tui
# Press 'n' to create new ball
# Fill in: Title, Context, Acceptance Criteria

# Or add via CLI
juggle plan --session auth-system \
  --title "Add login endpoint" \
  --context "REST API using JWT tokens" \
  --ac "POST /auth/login accepts email/password" \
  --ac "Returns JWT on success" \
  --ac "Returns 401 on failure" \
  --ac "Integration tests pass"

# Refine ACs with AI assistance
juggle agent refine auth-system

# Run the agent loop
juggle agent run auth-system

# In another terminal, monitor and add tasks
juggle tui
# Press 'O' to see live agent output
# Add more balls as needed - agent picks them up automatically
```

## File Structure

```
your-project/
├── .juggler/
│   ├── balls.jsonl           # Active balls
│   ├── config.json           # Project config
│   ├── archive/
│   │   └── balls.jsonl       # Completed balls
│   └── sessions/
│       └── my-feature/
│           ├── session.json  # Session config
│           ├── progress.txt  # Agent progress log
│           └── last_output.txt

~/.juggler/
├── config.json               # Global config (search paths)
```

## Agent Flags

```bash
juggle agent run my-feature \
  --iterations 5            # Max iterations (default: 10)
  --model sonnet           # Model: opus, sonnet, haiku
  --ball juggler-123       # Work on specific ball only
  --interactive            # Full Claude TUI (not headless)
  --timeout 5m             # Per-iteration timeout
  --trust                  # Skip permission prompts (dangerous)
  --delay 5                # Minutes between iterations
  --fuzz 2                 # Random delay variance (+/- minutes)
```

## Documentation

- [TUI Guide](docs/tui.md) - Full TUI documentation
- [Agent Integration](docs/agent-integration.md) - How agents use juggler
- [Installation](docs/installation.md) - Detailed setup instructions

## License

MIT
