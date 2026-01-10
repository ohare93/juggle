# Juggler

Track and manage multiple parallel work sessions with intelligent prioritization.

## Problem

When juggling multiple concurrent work sessions across different projects:
- You lose track of which session needs attention
- You forget the context and goals of each session
- Switching between sessions requires manual navigation
- Decision fatigue sets in: "what should I work on next?"
- You can't easily prioritize work across sessions
- Future work gets forgotten without proper planning

## Solution

Juggler provides:
- **Per-project storage** with version-controlled `.juggler/` directories (optional)
- **Session tracking** with intent, priority, and status
- **Planning future work** with "planned" balls that can be started later
- **Intelligent "next" algorithm** to determine what needs attention and reduce decision fatigue
- **Cross-project discovery** to track all work across multiple repositories
- **Hook-based automation** for activity tracking (works great with AI coding tools)
- **Rich terminal UI** with color-coded status

## Installation

### Quick Install

```bash
curl -sSL https://raw.githubusercontent.com/jmoiron/juggler/main/install.sh | bash
```

### Build from Source

```bash
# Build the binary
devbox shell
go build -o ~/.local/bin/juggle ./cmd/juggle

# Add to PATH (if not already)
export PATH="$HOME/.local/bin:$PATH"
```

See [Installation Guide](docs/installation.md) for detailed instructions

### Optional: Claude Code Integration

If you use Claude Code, juggler can integrate seamlessly with AI agents:

**For AI Agents (Recommended):**
```bash
# Install with workflow enforcement hooks (recommended)
juggle setup-claude --install-hooks

# Or install globally for all projects
juggle setup-claude --global

# Preview what will be added
juggle setup-claude --dry-run
```

This installs:
- **Agent instructions** teaching Claude when and how to use juggler
- **Workflow enforcement** with strict mode preventing workflow violations
- **Activity tracking hooks** that auto-update ball timestamps

**What Gets Installed:**
- `.claude/CLAUDE.md` - Top-of-document blocking instructions
- `.claude/hooks.json` - Pre-interaction hook for enforcement
- Marker files in `/tmp/juggle-check-*` for reminder tracking

See [Claude Integration Guide](docs/claude-integration.md), [Agent Integration Guide](docs/agent-integration.md), and [Workflow Enforcement Guide](docs/workflow-enforcement.md) for detailed information.

## Key Features

### 💡 Intuitive Command Aliases
- Use `juggle session`, `juggle ball`, `juggle current`, `juggle project`, or `juggle repo` - they're all the same!
- Natural language that fits your mental model
- Quick access to current session with any alias you prefer

### 🤖 Claude Code Integration
- **Batch todo operations**: Add multiple tasks at once for AI-assisted planning
- **Smart context resolution**: Automatically determines which ball to update in multi-ball repos
- **Visual progress tracking**: Checkboxes and completion percentages
- See [Claude Integration Guide](docs/claude-integration.md) for detailed workflows

### 📝 Todo Management
- Break down work into subtasks with checkboxes
- Track progress with completion percentages
- Batch add support: `juggle todo add "Task 1" "Task 2" "Task 3"`

### 🏷️ Tag System
- Organize balls with tags: `juggle tag add bug hotfix`
- Filter views by tags: `juggle status --tags bug`
- See tag statistics: `juggle tag list`

### 🔍 Search & History
- Search active balls: `juggle search authentication`
- Query archive: `juggle history --after 2025-10-01 --tags feature`
- Export data: `juggle export --format csv --output analysis.csv`

### ✏️ Flexible Editing
- Interactive mode: `juggle edit <ball-id>`
- Direct updates: `juggle edit <ball-id> --priority urgent --status blocked`

## Quick Start

```bash
# Start tracking a new session
cd ~/projects/my-app
juggle start --intent "Fix authentication bug" --priority high

# Plan future work
juggle plan --intent "Refactor login flow" --priority medium

# See all sessions across all projects
juggle status

# See only sessions in current project
juggle status --local

# View current session details
juggle session  # or: juggle ball / juggle current

# Find what needs attention
juggle next

# Mark current session as blocked
juggle session block "waiting for API keys"

# Unblock when ready
juggle session unblock

# Complete session (will prompt to start next planned ball)
juggle session done "Fixed OAuth token refresh issue"
```

## Interactive TUI

For a more visual and interactive experience, use the TUI mode:

```bash
juggle tui
juggle --local tui  # Local project only
```

The TUI provides:
- **Full-screen interface** with keyboard navigation
- **Real-time filtering** by state (ready/juggling/dropped/complete)
- **Quick actions** without typing commands
- **Ball details view** showing todos, tags, and timestamps
- **Color-coded status** for easy visual scanning
- **Cross-project view** of all your work
- **Safe delete** with confirmation dialog

### Keyboard Shortcuts

**Navigation:**
- `↑/k` - Move up
- `↓/j` - Move down
- `Enter` - View ball details
- `b/Esc` - Back to list (or exit from list)
- `?` - Toggle help

**Filters (toggleable):**
- `1` - Show all states
- `2` - Toggle ready visibility
- `3` - Toggle juggling visibility
- `4` - Toggle dropped visibility
- `5` - Toggle complete visibility

**State Management:**
- `Tab` - Cycle state (ready → juggling → complete → dropped → ready)
- `s` - Start ball (ready → juggling:in-air)
- `r` - Set ball to ready
- `c` - Complete ball
- `d` - Drop ball
- `R` - Refresh/reload (shift+r)

**Ball Operations:**
- `x` - Delete ball (with confirmation)
- `p` - Cycle priority (low → medium → high → urgent → low)

**Other:**
- `q` - Quit

## Commands

### Workflow Commands

- `juggle check` - Interactive workflow helper showing current state and guidance
  - Detects juggling balls and provides next-step recommendations
  - Helps maintain workflow discipline
  - Use before creating new balls
- `juggle audit` - Analyze project health and completion metrics
  - Shows completion ratios per project
  - Identifies stale ready balls (>30 days)
  - Provides actionable recommendations
- `juggle reminder` - Check if workflow reminder should be shown (for hooks)

### Session Commands

The `session` command (aliases: `ball`, `project`, `repo`, `current`) manages the current work session:

- `juggle session` - Show current session details with todos and status
- `juggle session block [reason]` - Mark current session as blocked
- `juggle session unblock` - Clear blocker and mark active
- `juggle session done [note]` - Complete session and archive (prompts for next planned ball)
- `juggle session todo add <task> [<task2> ...]` - Add one or more todos
- `juggle session todo list` - List todos for current session
- `juggle session todo done <index>` - Mark todo as complete
- `juggle session todo rm <index>` - Remove a todo
- `juggle session todo edit <index> <text>` - Edit todo text
- `juggle session todo clear` - Clear all todos
- `juggle session tag add <tag> [<tag2> ...]` - Add tags
- `juggle session tag rm <tag> [<tag2> ...]` - Remove tags

### Project Commands

- `juggle start [ball-id]` - Start tracking a new session or activate a planned ball
- `juggle plan` - Add a planned ball for future work
- `juggle status` - Show all balls across all projects (grouped by project)
  - `--local` - Show only balls from current project
  - `--tags <tags>` - Filter by tags
  - `--priority <level>` - Filter by priority
- `juggle next` - Determine next ball needing attention
- `juggle show <id>` - Show detailed ball information
- `juggle list` - Alias for status
- `juggle edit <id>` - Edit ball properties (interactive or with flags)
- `juggle projects` - List all tracked projects and manage search paths

### Search & History

- `juggle search [query]` - Search active balls by intent or filters
- `juggle history [query]` - Query archived (done) balls
- `juggle export` - Export balls to JSON or CSV

### Filtering

Most commands support filtering flags:
- `--tags <tag1>,<tag2>` - Filter by tags (OR logic)
- `--priority <level>` - Filter by priority
- `--status <state>` - Filter by status

### Flags

```bash
juggle start \
  --intent "What you're doing" \
  --priority high|medium|low|urgent \
  --tags bug-fix,auth
```

## Ball States

- **planned** 📋 - Future work not yet started
- **active** 🟢 - Currently working
- **blocked** 🟡 - Waiting on something
- **needs-review** 🔴 - Requires human decision
- **done** ✓ - Completed and archived

## Next Algorithm

The `juggle next` command uses this priority order:

1. **needs-review** status (highest priority)
2. **Recently unblocked** (was blocked, now active)
3. **Longest idle** time (active sessions)
4. **Higher priority** level
5. **Current directory** (tie-breaker)

## Workflow Enforcement (Claude Code)

When installed with `juggle setup-claude --install-hooks`, juggler enforces workflow discipline:

### Strict Mode Features

1. **Top-of-Document Blocking Instructions**
   - Critical requirement section at top of CLAUDE.md
   - Visual separators (═══) for maximum visibility
   - Explicit "YOU ARE BLOCKED" language (90-95% effective)

2. **Check → Start → Complete Cycle**
   ```bash
   juggle check    # Before any work - shows state and guidance
   juggle start    # Create new ball only if needed
   juggle complete # Mark work complete when done
   ```

3. **Marker File System**
   - Files in `/tmp/juggle-check-*` (SHA256-based project hashing)
   - Tracks last check time per project
   - 5-minute reminder threshold (avoids spam)
   - Performance: <50ms overhead per interaction

4. **Pre-Interaction Hook**
   - Runs before every Claude interaction
   - Shows reminder if check threshold exceeded
   - Non-blocking (failures don't break workflow)
   - Atomic file operations for safety

### Enforcement Effectiveness

Based on research from WP3:
- **90-95% compliance** with top-of-document instructions
- **5x more effective** than suggestion-based approaches
- **Visual separators** significantly improve visibility
- **Blocking language** creates sense of requirement

### Example Workflow

```bash
# User starts Claude session
# Hook shows: "⚠️ Run 'juggle check' to verify workflow state"

$ juggle check
✅ No active balls
Ready to start new work.

$ juggle start "Add user authentication"
✓ Started ball: myapp-5

# Work proceeds...

$ juggle complete "Auth system implemented and tested"
✓ Ball marked as complete
```

See [Workflow Enforcement Guide](docs/workflow-enforcement.md) for complete details.

## Claude Code Plugin (Optional)

Juggler can integrate with Claude Code via the plugin system to automatically:
- Track activity on each interaction
- Update idle timestamps
- Keep session state in sync

Install as a Claude Code plugin by placing this repository where Claude can find it. The `.claude-plugin/marketplace.json` defines the hook integration.

The core tool works perfectly fine without plugin integration - you'll just update session state manually with commands.

## File Storage

Juggler uses per-project storage with JSONL format:

### Per-Project Storage
- **Active balls**: `.juggler/balls.jsonl` (one JSON object per line)
- **Completed balls**: `.juggler/archive/balls.jsonl`
- **Version control friendly**: You can commit `.juggler/` to track work planning

### Global Configuration
- **Config file**: `~/.juggler/config.json`
- **Search paths**: Directories to scan for projects with `.juggler/` folders
- **Default paths**: `~/Development`, `~/projects`, `~/work`

### JSONL Format Benefits
- **Append-only**: New balls just append a line, no file rewrite needed
- **Version control**: Line-based diffs work great with git
- **Simple parsing**: Each line is a complete JSON object
- **Efficient**: No need to read entire file to append

## Multiple Balls Per Project

Juggler supports multiple balls in the same directory:
- Each ball gets a unique ID with timestamp
- Mix planned and active balls in one project
- `status` highlights balls in current directory
- `juggle next` can route between projects and balls
- After completing a ball, juggler prompts to start next planned ball

## Examples

### Planning future work

```bash
$ juggle plan
What do you plan to work on? Upgrade to React 19
✓ Planned ball added: my-app-20251012-150000
  Intent: Upgrade to React 19
  Priority: medium

Start this ball with: juggle start my-app-20251012-150000
```

### Starting a planned ball

```bash
$ juggle start my-app-20251012-150000
✓ Started planned ball: my-app-20251012-150000
  Intent: Upgrade to React 19
  Priority: medium
```

### Starting a new ball interactively

```bash
$ juggle start
What are you trying to accomplish? Fix authentication bug in OAuth flow
✓ Started session: my-app-20250112-143022
  Intent: Fix authentication bug in OAuth flow
  Priority: medium
```

### Checking status across projects

```bash
$ juggle status
▸ /home/user/projects/my-app (current)
 ID                         STATUS      PRIORITY  TODOS     INTENT                                    
my-app-20251012-143022   active      high      2/5       Fix auth bug                             
my-app-20251012-150000   planned     medium    -         Upgrade to React 19                      

▸ /home/user/projects/api-client
 ID                         STATUS      PRIORITY  TODOS     INTENT                                    
api-client-20251012-120  blocked     medium    -         Upgrade dependencies                     

▸ /home/user/projects/frontend
 ID                         STATUS      PRIORITY  TODOS     INTENT                                    
frontend-20251012-095    active      low       3/3       Redesign landing page                    

4 ball(s) total | 2 active, 1 blocked, 1 planned
```

### Finding next task across projects

```bash
$ juggle next
→ Next ball: api-client-20251012-120000
  Intent: Upgrade dependencies
  Project: /home/user/projects/api-client
  Status: blocked
  Priority: medium
  Idle: 2h
```

## Development

```bash
# Enter dev environment
devbox shell

# Build
go build -o juggle ./cmd/juggle

# Run
./juggle status

# Install for testing
go install ./cmd/juggle
```

### Using todos for task breakdown

```bash
# Add multiple tasks at once (great for Claude!)
$ juggle session todo add \
  "Create User model with email and password fields" \
  "Implement password hashing with bcrypt" \
  "Add JWT token generation" \
  "Create login and register endpoints" \
  "Write unit tests for auth functions"
✓ Added 5 todos to ball: my-app-20251012-143022

# View progress
$ juggle show
Ball: my-app-20251012-143022
...
Todos: 0/5 complete (0%)
  1. [ ] Create User model with email and password fields
  2. [ ] Implement password hashing with bcrypt
  3. [ ] Add JWT token generation
  4. [ ] Create login and register endpoints
  5. [ ] Write unit tests for auth functions

# Mark tasks complete
$ juggle session todo done 1
✓ Todo 1 completed: Create User model with email and password fields
Progress: 1/5 complete (20%)

$ juggle todo done 2
✓ Todo 2 completed: Implement password hashing with bcrypt
Progress: 2/5 complete (40%)
```

### Using tags and search

```bash
# Add tags for organization
$ juggle session tag add bug backend authentication
✓ Added 3 tags to ball: my-app-20251012-143022

# Filter status by tags
$ juggle status --tags bug
Active filters:
  Tags: bug

▸ /home/user/projects/my-app (current)
 ID                         STATUS      PRIORITY  TODOS     INTENT                    
my-app-20251012-143022   active      high      2/5       Fix auth bug

# Search across all active balls
$ juggle search authentication --tags backend
Found 2 ball(s)
Search criteria:
  Query: "authentication"
  Tags: backend

 ID                         PROJECT                    STATUS      PRIORITY  INTENT                    
my-app-20251012-143022   .../projects/my-app       active      high      Fix auth bug              
api-20251012-140000      .../projects/api          blocked     medium    Update auth middleware    
```

### Querying history

```bash
# View recently completed work
$ juggle history --limit 5
Found 5 archived ball(s) (limited to 5)

 ID                         COMPLETED     DURATION  PRIORITY  INTENT                    
my-app-20251012-120000   2025-10-12    2h 15m    high      Fix login bug             
api-20251012-110000      2025-10-11    4h 30m    medium    Add rate limiting         
frontend-20251012-090    2025-10-11    1h 45m    low       Update styles             

# Query by date range
$ juggle history --after 2025-10-01 --before 2025-10-15

# View archive statistics
$ juggle history --stats
Archive Statistics

Total archived balls: 45

By Priority:
  urgent: 5
  high: 15
  medium: 20
  low: 5

Top Tags:
  bug-fix: 12
  feature: 18
  refactor: 8
  documentation: 7

Duration Statistics:
  Total time: 3d 12h 45m
  Average: 1h 52m
  Shortest: 15m
  Longest: 8h 30m
```

### Editing balls

```bash
# Interactive edit
$ juggle edit my-app-20251012-143022
Current values for ball my-app-20251012-143022:
  Intent: Fix auth bug
  Priority: high
  Status: active
  Tags: bug, backend, authentication

New intent (or press Enter to keep current): Fix OAuth token refresh issue
New priority (low|medium|high|urgent, or press Enter to keep current): urgent
...

# Direct edit
$ juggle edit my-app-20251012-143022 --priority urgent --status needs-review
✓ Updated priority: urgent
✓ Updated status: needs-review

✓ Ball my-app-20251012-143022 updated successfully
```

### Exporting data

```bash
# Export to JSON for analysis
$ juggle export --format json --output backup.json
✓ Exported 15 ball(s) to backup.json

# Export to CSV including archived balls
$ juggle export --format csv --output analysis.csv --include-done
✓ Exported 60 ball(s) to analysis.csv
```

## Workflow Examples

### Planning and executing work

```bash
# Plan future work in your project
$ cd ~/projects/my-app
$ juggle plan --intent "Add user profile page" --priority high
$ juggle plan --intent "Fix search performance" --priority medium

# Start the high-priority ball
$ juggle start my-app-20251012-160000
✓ Started planned ball: my-app-20251012-160000

# When done, juggler prompts for the next planned ball
$ juggle done "Completed user profile page"
✓ Ball marked as done and archived

1 planned ball(s) in this project:
  1. [my-app-20251012-160100] Fix search performance (priority: medium)

Start a planned ball? (enter number, or press Enter to skip): 1
✓ Started ball: my-app-20251012-160100
```

### Managing search paths

```bash
# List all projects juggler is tracking
$ juggle projects
Projects found:
  - /home/user/Development/juggler (2 balls: 2 planned)
  - /home/user/projects/my-app (3 balls: 1 active, 1 blocked, 1 planned)

Search paths:
  - /home/user/Development
  - /home/user/projects

# Add a new search path
$ juggle projects add ~/work
✓ Added search path: /home/user/work
```

## Architecture

```
juggler/
├── cmd/juggle/           # Main entry point
├── internal/
│   ├── session/          # Session data model & storage
│   │   ├── session.go    # Ball structure and state
│   │   ├── store.go      # Per-project JSONL storage
│   │   ├── config.go     # Global configuration
│   │   └── discovery.go  # Cross-project discovery
│   └── cli/              # Command implementations
├── .juggler/             # Per-project storage (example)
│   ├── balls.jsonl       # Active balls (one JSON per line)
│   └── archive/
│       └── balls.jsonl   # Completed balls
└── scripts/              # Hook installation
```

## Documentation

- [Installation Guide](docs/installation.md) - Detailed installation instructions
- [Claude Code Integration](docs/claude-integration.md) - User guide for AI-assisted workflows  
- [Agent Integration Guide](docs/agent-integration.md) - Complete guide for AI agents using juggler
- [Workflow Examples](examples/claude-workflow.md) - Real-world examples of agent usage patterns
- Command help: `juggle --help` or `juggle <command> --help`

## Future Enhancements

- Team collaboration with shared `.juggler/` files in git
- Ball templates for common workflows
- Time tracking with pause/resume functionality
- Ball dependency tracking (this ball blocks that ball)
- Web UI dashboard for visual project overview
- Smart reminders for stale blocked balls
- Integration with other task management tools
- GitHub issues/PRs integration
- Metrics and analytics dashboard

## License

MIT
