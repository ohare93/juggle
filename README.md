# Juggle - CLI Ralph Loops with Good UX

Run Ralph Loops while throwing in further tasks; no need to stop the loop or editing json.

- Define tasks with acceptance criteria, priority, model size, ...
- Let agents execute autonomously
- Manage work via TUI while they run.

## Why Juggle?

The [Ralph Loop](https://github.com/snarktank/ralph) is powerful but raw -
a bash script watching a prd.json file. Editing tasks while the loop runs
means fighting the agent for file writes. No priority ordering. No way to
refine tasks without stopping everything or risking multiple writes.

Juggle adds the missing structure:

- **Edit while running** - TUI file-watches, agent uses CLI; no conflicts
- **Manage the loop** - reorder, prioritize, group tasks into sessions
- **Never run dry** - keep feeding refined tasks so the agent always has work
- **Multiple modes** - headless batch, interactive hand-holding, agent-assisted
  refinement - all in separate terminals, all on the same task list

**TUI Main View** - Sessions on the left, balls (tasks) on the right, activity log at bottom:

![TUI Main Menu](assets/tui-main-menu.png)

**Task Creation** - Define context, acceptance criteria, priority, dependencies:

![Create New Ball](assets/tui-create-new-ball.png)

**Parallel Agent Loops** - Multiple agents running simultaneously with live feedback:

![Agent Loops Running](assets/loop-running-with-feedback.png)

---

The feedback from the Agent running in the loop is a bit raw, but it gets the job done.
Further work will be done to improve it from this early version.

The important part it that it _works_.

- Craft tasks (balls) in 5 different repos and run the loop. Update and append tasks into all of the projects at once.
- Yet for each Ralph loop your tasks remain isolated - and you can update and edit the tasks at the same time as the loop without issues.
- Quick shortcuts to run loops, run refinement for your tasks, run interactive sessions for hand-holding the Agent.

After you're running a loop in 4 different repos and a double loop using workspaces in your main repo... then you're truly succeeding at Agentic Development.

## Built With Itself

<!--
Commit stats generated with:
jj log --no-graph -r 'ancestors(@)' -T 'committer.timestamp() ++ "\n"' | cut -d' ' -f1 | sort | uniq -c | sort -k2

Output on 2026-01-14:
     44 2026-01-10
      6 2026-01-11
     92 2026-01-12
     22 2026-01-13
     66 2026-01-14
(plus 7 earlier commits from Oct/Nov 2025 initial setup)
-->

Juggle was built using juggle - after the first few clunky Ralph Loops with bash scripts and there was something working here.

Here's the commit activity from the first 5 days of development:

```
Jan 10  ████████████████████████░░░░░░░░░░░░░░░░░░░░░░░░░░░░  44
Jan 11  ███░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░   6
Jan 12  ████████████████████████████████████████████████████  92
Jan 13  ████████████░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░  22
Jan 14  █████████████████████████████████████░░░░░░░░░░░░░░░  66
        ──────────────────────────────────────────────────────
        230 commits in 5 days
```

The workflow was simple:

- Add a task (`juggle plan`) with good ACs (Acceptance Criteria) until I had enough
- Interactively refine the ACs with an agent (`juggle agent refine`)
- Run a loop in one tab (`juggle agent run`) and let it work through the tasks
- View its progress in the TUI (`juggle`) as tasks get marked complete and the commits roll in
- In the meantime I keep adding more and more tasks... it just keeps going

In the end I had multiple agents working in parallel on isolated worktrees, each focused on
different tasks group, all managed through the same TUI and a multiplex terminal (Zellij).

## Installation

### Windows (Scoop)

```bash
scoop bucket add ohare93 https://github.com/ohare93/scoop && scoop install juggle
```

### macOS (Homebrew)

```bash
brew tap ohare93/tap && brew install juggle
```

### Linux

**Download and inspect (recommended):**

```bash
curl -O https://raw.githubusercontent.com/ohare93/juggle/main/install.sh
less install.sh
chmod +x install.sh && ./install.sh
```

**Or if you trust the source:**

```bash
curl -sSL https://raw.githubusercontent.com/ohare93/juggle/main/install.sh | bash
```

**Or with Go:**

```bash
go install github.com/ohare93/juggle/cmd/juggle@latest
```

### Build from Source

See [Installation Guide](docs/installation.md) for building from source and additional options.

## Quick Start

### Create a session and add tasks

```bash
cd ~/your-project
juggle sessions create my-feature
juggle tui                        # Add tasks via TUI (recommended)
```

### Run the agent loop

```bash
juggle agent run                  # Interactive session selector
juggle agent run my-feature       # Or specify session directly
```

### Refine already existing tasks interactively

```bash
juggle agent refine
juggle agent refine my-feature
```

### Manage while it runs

Open the TUI:

```bash
juggle                            # Add/edit/reorder tasks live
```

Or just add / edit / view tasks directly in the terminal:

```bash
juggle plan

juggle update 162b4eb0 --tags bug-fixes,loop-a

# Large updates for the agent to run during refinement. Users would use the TUI instead
juggle update cc58e434 --ac "juggle worktree add <path> registers worktree in main repo config" \
       --ac "juggle worktree add creates .juggle/link file in worktree pointing to main repo" \
       --ac "All juggle commands in worktree use main repo's .juggle/ for storage" \
       --ac "Ball WorkingDir reflects actual worktree path (not main repo)" \
       --ac "juggle worktree remove <path> unregisters and removes link file" \
       --ac "juggle worktree list shows registered worktrees" \
       --ac "Integration tests for worktree registration and ball sharing" \
       --ac "devbox run test passes"
```

## Parallel Agents with Worktrees

Each agent can work in its own isolated worktree/workspace.

With git/jj it's simple to setup separate workspaces so that multiple agents can work without impacting each other.
In juggle we just need to link the two repos to the same juggle project, then they can each run on separate sessions (collections of tasks)

See [Worktrees (Parallel Agent Loops)](docs/installation.md#worktrees-parallel-agent-loops)
in the installation guide for setup instructions.

**Workspace management:**

```bash
juggle worktree run "devbox run build"  # Build in all workspaces
juggle worktree sync                     # Sync local files (e.g. .claude/settings.local.json)
```

![JJ Commit Log with Worktrees](assets/jj-commit-log-with-3-worktrees.png)

## Roadmap

- **More agents** - Support beyond Claude Code (Cursor, Aider, etc.)
- **TUI-integrated loop** - Run the agent loop inside TUI with live output
- **Workspace automation** - Automatic git worktree/branch setup per session
- **Endless mode** - Agent stays running, polling for new tasks when queue empties
- **Notifications** - Get notified when tasks complete or need attention

## Agent Skill

Juggle includes a skill that teaches AI agents how to manage tasks. Install it so your agent knows the CLI commands, state transitions, and best practices.

**Claude Code:**

```bash
claude plugin add github:ohare93/juggle
```

See [Agent Skill Setup](docs/installation.md#agent-skill) for other agents and details.

## Documentation

- [Installation Guide](docs/installation.md) - Build from source, worktrees, configuration
- [TUI Guide](docs/tui.md) - Keyboard shortcuts, views, workflows
- [Commands Reference](docs/commands.md) - Full CLI documentation

## License

MIT
