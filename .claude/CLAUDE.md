# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build and Test Commands

### Building

```bash
# Enter devbox shell (sets up Go environment)
devbox shell

# Build the binary
go build -o juggle ./cmd/juggle

# Install locally for testing
go install ./cmd/juggle
```

### Testing

```bash
# Run integration tests
devbox run test
# or: go test -v ./internal/integration_test/...

# Run all tests
devbox run test-all
# or: go test -v ./...

# Generate coverage report
devbox run test-coverage
# or: go test -v -coverprofile=coverage.out ./internal/integration_test/...
#     go tool cover -html=coverage.out -o coverage.html

# Run single test
go test -v ./internal/integration_test/... -run TestTrackActivity
```

### Development

```bash
# Clean build artifacts
go clean

# Update dependencies
go mod tidy

# Check formatting
go fmt ./...
```

## Architecture Overview

### Core Concepts

**Juggler** tracks concurrent work sessions ("balls") across multiple projects using a juggling metaphor. Each ball represents a task with state tracking, todos, and context preservation across conversations.

### State Machine

Balls follow this state flow:

- **ready** → Ball is planned but not started
- **juggling** → Ball is actively being worked on (with substates):
  - **needs-thrown** → Waiting for user input/direction
  - **in-air** → Agent is actively working
  - **needs-caught** → Work complete, needs user verification
- **dropped** → Task abandoned
- **complete** → Task finished and archived

### Key Components

#### 1. Session Package (`internal/session/`)

**`session.go`** - Core data model:

- `Session` struct: Represents a ball with ID, intent, priority, state, todos, tags, Zellij info
- State types: `ActiveState` (ready/juggling/dropped/complete), `JuggleState` (needs-thrown/in-air/needs-caught)
- Priority levels: low/medium/high/urgent
- Methods for state transitions, todo management, activity tracking

**`store.go`** - Persistent storage:

- JSONL format: `.juggler/balls.jsonl` (active), `.juggler/archive/balls.jsonl` (completed)
- `Store` type handles CRUD operations for balls
- Methods: `AppendBall()`, `LoadBalls()`, `UpdateBall()`, `ArchiveBall()`
- Ball resolution by ID or short ID

**`config.go`** - Global configuration:

- Location: `~/.juggler/config.json`
- Manages search paths for discovering projects with `.juggler/` directories
- Default paths: `~/Development`, `~/projects`, `~/work`

**`discovery.go`** - Cross-project discovery:

- `DiscoverProjects()`: Scans search paths for `.juggler/` directories
- `LoadAllBalls()`, `LoadJugglingBalls()`: Load balls across all discovered projects
- Enables global views like `juggle status` and `juggle next`

**`archive.go`** - Archival operations:

- `ArchiveBall()`: Moves completed balls to archive
- `LoadArchive()`: Query historical completed work

#### 2. CLI Package (`internal/cli/`)

**Command structure:**

- `root.go`: Main command dispatcher, handles `juggle` with no args (shows juggling balls) or `juggle <ball-id> <action>`
- Each major command has its own file (e.g., `start.go`, `status.go`, `todo.go`)
- Helper functions: `GetWorkingDir()`, `NewStoreForCommand()`, `LoadConfigForCommand()`

**Key command patterns:**

- Commands operating on current ball: Get store → resolve current ball → operate → update store
- Cross-project commands: Load config → discover projects → load balls → operate
- Ball-specific commands: Find ball by ID across all projects → create store for that ball's directory → operate

**Activity tracking (`track.go`):**

- `track-activity` command updates last activity timestamp (called by Claude hooks)
- Resolution order:
  1. `JUGGLER_CURRENT_BALL` env var (explicit override)
  2. Zellij session+tab matching
  3. Single juggling ball in project
  4. Most recently active ball

#### 3. Zellij Integration (`internal/zellij/`)

**`zellij.go`** - Terminal multiplexer integration:

- `DetectInfo()`: Checks `ZELLIJ_SESSION_NAME` env var, parses layout dump for current tab
- `GoToTab()`: Switches to a tab by name
- Balls store Zellij session+tab when created in Zellij
- `jump` and `next` commands use this for seamless tab switching

#### 4. Claude Integration (`internal/claude/`)

**`instructions.go`** - Agent instructions:

- Template for teaching Claude agents how to use juggler
- Markers: `<!-- juggler-instructions-start/end -->` for idempotent installs
- Functions for reading/writing/updating CLAUDE.md files

**`setup_claude.go`** (CLI) - Installation command:

- `juggle setup-claude`: Install instructions to `.claude/CLAUDE.md` (local) or `~/.claude/CLAUDE.md` (global)
- Flags: `--global`, `--dry-run`, `--update`, `--uninstall`, `--force`

## Storage Format

### JSONL Structure

Each ball is one line of JSON in `.juggler/balls.jsonl`:

```json
{
  "id": "juggler-5",
  "zellij_session": "main",
  "zellij_tab": "juggler",
  "intent": "Add search feature",
  "priority": "high",
  "active_state": "juggling",
  "juggle_state": "in-air",
  "started_at": "2025-10-16T10:30:00Z",
  "last_activity": "2025-10-16T11:45:00Z",
  "update_count": 12,
  "todos": [
    { "text": "Design API", "done": true },
    { "text": "Implement backend", "done": false }
  ],
  "tags": ["feature", "backend"]
}
```

### File Locations

- Per-project: `.juggler/balls.jsonl` (active), `.juggler/archive/balls.jsonl` (complete)
- Global config: `~/.juggler/config.json`
- Local instructions: `.claude/CLAUDE.md`
- Global instructions: `~/.claude/CLAUDE.md`

## Important Patterns

### Resolving Current Ball

When multiple balls exist in a project, resolution logic (in `internal/cli/todo.go`, `session.go`, etc.):

1. Check for explicit ball ID argument
2. If no ID provided, find current ball:
   - If exactly one juggling ball exists → use it
   - If multiple juggling balls → error, require explicit ID
   - Special: `track-activity` uses resolution order (env var → Zellij → single → most recent)

### Cross-Project Operations

Commands like `status`, `next`, `search`, `history`:

1. Load config via `LoadConfigForCommand()`
2. Discover all projects with `session.DiscoverProjects(config)`
3. Load balls from all projects
4. Operate on aggregated data
5. When updating a ball, create a store for that ball's working directory

### State Transitions

Valid transitions (enforced in various command handlers):

- `ready` → `juggling` (via `start`)
- `juggling` → `complete` (via session commands)
- `juggling` → `dropped` (via session commands)
- Within `juggling`: `needs-thrown` ↔ `in-air` ↔ `needs-caught`

### Testing Utilities

Integration tests use `testutil_test.go`:

- `TestEnv`: Sets up isolated test environment with temp directories
- `SetupTestStore()`: Creates store with temp config
- Environment variable mocking for testing activity tracking resolution

## Multi-Agent Support

When multiple agents/users work simultaneously:

**Activity Tracking Resolution:**
Set `JUGGLER_CURRENT_BALL` environment variable to explicitly target a ball:

```bash
export JUGGLER_CURRENT_BALL="juggler-5"
```

This overrides Zellij matching and ensures activity updates go to the correct ball when:

- Multiple AI agents work in same repo
- Multiple terminal sessions are active
- You want explicit control over which ball is tracked

## Code Style Notes

- Use `lipgloss` for terminal styling (colors, formatting)
- Commands return `error`, not `fmt.Errorf()` directly - wrap with context
- Silent failures for hook commands (return `nil` instead of error)
- JSONL append-only writes for better version control diffs
- Ball IDs format: `<directory-name>-<counter>` (e.g., `juggler-5`)
