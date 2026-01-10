# Updated CLAUDE.md Content

**Note:** This file contains the updated content for `.claude/CLAUDE.md` for US-020.
Copy this content to replace the contents of `.claude/CLAUDE.md`.

---

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
go test -v ./internal/integration_test/... -run TestExport
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

Balls use a simplified 4-state model:

- **pending** → Ball is planned but not started
- **in_progress** → Ball is actively being worked on
- **complete** → Task finished and archived
- **blocked** → Task is blocked (with optional `BlockedReason` for context)

State transitions:
- `pending` → `in_progress` (via `start`)
- `in_progress` → `complete` (via completion commands)
- `in_progress` → `blocked` (via block command with reason)
- `blocked` → `in_progress` (via unblock/resume)
- Any state → `pending` (reset)

### Key Components

#### 1. Session Package (`internal/session/`)

**`session.go`** - Core data model:

- `Session` struct: Represents a ball with ID, intent, priority, state, todos, tags
- `BallState` type: pending/in_progress/complete/blocked
- `BlockedReason` field: Provides context when a ball is blocked
- Priority levels: low/medium/high/urgent
- Methods for state transitions, todo management

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
- `LoadAllBalls()`, `LoadInProgressBalls()`: Load balls across all discovered projects
- Enables global views like `juggle status` and `juggle next`

**`archive.go`** - Archival operations:

- `ArchiveBall()`: Moves completed balls to archive
- `LoadArchive()`: Query historical completed work

**`juggle_session.go`** - JuggleSession entity:

- `JuggleSession` struct: Groups balls by tag for context organization
- Fields: ID, Description, CreatedAt
- Sessions are referenced by ball tags (balls with matching tag belong to session)

**`session_store.go`** - Session persistence:

- JSONL format: `.juggler/sessions.jsonl`
- `SessionStore` type handles CRUD operations for sessions
- Methods: `AppendSession()`, `LoadSessions()`, `UpdateSession()`, `DeleteSession()`

#### 2. CLI Package (`internal/cli/`)

**Command structure:**

- `root.go`: Main command dispatcher, handles `juggle` with no args (shows in-progress balls) or `juggle <ball-id> <action>`
- Each major command has its own file (e.g., `start.go`, `status.go`, `todo.go`)
- Helper functions: `GetWorkingDir()`, `NewStoreForCommand()`, `LoadConfigForCommand()`

**Key command patterns:**

- Commands operating on current ball: Get store → resolve current ball → operate → update store
- Cross-project commands: Load config → discover projects → load balls → operate
- Ball-specific commands: Find ball by ID across all projects → create store for that ball's directory → operate

**Session commands (`sessions.go`):**

```bash
# List all sessions
juggle sessions list

# Create new session
juggle sessions create <session-id> [-m description]

# Delete session
juggle sessions delete <session-id>

# Show session details
juggle sessions show <session-id>
```

**Export command (`export.go`):**

```bash
# Export ball data for Ralph-style agent loops
juggle export --session <id> --format ralph
```

#### 3. TUI Package (`internal/tui/`)

**Bubble Tea-based terminal UI with two modes:**

**List View (Legacy):**
- Shows balls with state indicators
- Detail view: Shows ball details, todos, and actions
- State-based styling: Different colors for pending/in_progress/complete/blocked

**Split View Mode (`juggle tui --split`):**

Three-panel layout:
- **Left Panel (25%)**: Sessions list with ball counts
- **Right Panel (75%)**: Balls list with todos (expandable)
- **Bottom Panel**: Activity log showing recent operations

Key files:
- `model.go`: TUI state model with panel tracking
- `update.go`: Event handlers and state transitions
- `view.go`: View routing and legacy views
- `splitview.go`: Three-panel layout rendering

Navigation:
- `Tab`: Cycle between panels (Sessions → Balls → Todos)
- `j/k` or arrows: Navigate within panel
- `Enter`: Select item / expand todos
- `Esc`: Go back / collapse

CRUD Operations:
- `a`: Add new session/ball/todo (context-aware)
- `e`: Edit selected item
- `x`: Delete with confirmation
- `s`: Start ball (pending → in_progress)
- `c`: Complete ball
- `b`: Block ball (prompts for reason)
- `Space`: Toggle todo completion

Filtering:
- `/`: Open filter dialog for current panel
- `Ctrl+U`: Clear active filter
- `--session <id>`: Start with session pre-selected

#### 4. Watcher Package (`internal/watcher/`)

**File system watcher for live updates:**

- Uses `fsnotify` to monitor `.juggler/` directory
- Watches `balls.jsonl` and `sessions.jsonl` for changes
- Sends `FileChangedMsg` to TUI on detected changes
- Debounces rapid file changes (100ms window)
- Auto-reloads data when external changes detected

Usage in TUI:
```go
w := watcher.New(jugglerDir)
model := tui.InitialSplitModelWithWatcher(store, sessionStore, config, localOnly, w, "")
```

## Storage Format

### JSONL Structure

Each ball is one line of JSON in `.juggler/balls.jsonl`:

```json
{
  "id": "juggler-5",
  "intent": "Add search feature",
  "priority": "high",
  "state": "in_progress",
  "blocked_reason": "",
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

Each session is one line of JSON in `.juggler/sessions.jsonl`:

```json
{
  "id": "feature-auth",
  "description": "Implement user authentication",
  "created_at": "2025-10-16T09:00:00Z"
}
```

### Ralph Export Format

The `juggle export --session <id> --format ralph` command generates agent-compatible output:

```
<context>
Session description and context goes here
</context>

<progress>
Progress log entries from progress.txt
</progress>

<tasks>
## Ball: ball-id
State: in_progress
Priority: high
Intent: Task description

### Todos
1. [ ] First todo item
2. [x] Completed todo item
</tasks>
```

### File Locations

- Per-project balls: `.juggler/balls.jsonl` (active), `.juggler/archive/balls.jsonl` (complete)
- Per-project sessions: `.juggler/sessions.jsonl`
- Session context: `.juggler/sessions/<id>/session.json`
- Session progress: `.juggler/sessions/<id>/progress.txt`
- Global config: `~/.juggler/config.json`

## Important Patterns

### Resolving Current Ball

When multiple balls exist in a project, resolution logic:

1. Check for explicit ball ID argument
2. If no ID provided, find current ball:
   - If exactly one in-progress ball exists → use it
   - If multiple in-progress balls → error, require explicit ID

### Cross-Project Operations

Commands like `status`, `next`, `search`, `history`:

1. Load config via `LoadConfigForCommand()`
2. Discover all projects with `session.DiscoverProjects(config)`
3. Load balls from all projects
4. Operate on aggregated data
5. When updating a ball, create a store for that ball's working directory

### State Transitions

Valid transitions (enforced in command handlers):

- `pending` → `in_progress` (via `start`)
- `in_progress` → `complete` (via complete command)
- `in_progress` → `blocked` (via block command)
- `blocked` → `in_progress` (via resume)

### Testing Utilities

Integration tests use `testutil_test.go`:

- `TestEnv`: Sets up isolated test environment with temp directories
- `SetupTestStore()`: Creates store with temp config
- Environment variable mocking for testing

## Multi-Agent Support

When multiple agents/users work simultaneously, set `JUGGLER_CURRENT_BALL` environment variable to explicitly target a ball:

```bash
export JUGGLER_CURRENT_BALL="juggler-5"
```

This ensures operations go to the correct ball when:

- Multiple AI agents work in same repo
- Multiple terminal sessions are active
- You want explicit control over which ball is targeted

## Code Style Notes

- Use `lipgloss` for terminal styling (colors, formatting)
- Commands return `error`, not `fmt.Errorf()` directly - wrap with context
- JSONL append-only writes for better version control diffs
- Ball IDs format: `<directory-name>-<counter>` (e.g., `juggler-5`)
