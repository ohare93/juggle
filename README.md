# Juggle

A task runner for AI agent loops. Define tasks with acceptance criteria,
let agents execute autonomously, manage work via TUI while they run.

## Why Juggle?

The [Ralph Loop](https://github.com/snarktank/ralph) is powerful but raw -
a bash script watching a prd.json file. Editing tasks while the loop runs
means fighting the agent for file writes. No priority ordering. No way to
refine tasks without stopping everything.

Juggle adds the missing structure:

- **Edit while running** - TUI file-watches, agent uses CLI; no conflicts
- **Manage the queue** - reorder, prioritize, group tasks into sessions
- **Never run dry** - keep feeding refined tasks so the agent always has work
- **Multiple modes** - headless batch, interactive hand-holding, agent-assisted
  refinement - all in separate terminals, all on the same task list

## Quick Start

### Install

```bash
curl -sSL https://raw.githubusercontent.com/ohare93/juggle/main/install.sh | bash
```

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

### Manage while it runs

Open another terminal:

```bash
juggle tui                        # Add/edit/reorder tasks live
```

## Roadmap

- **More agents** - Support beyond Claude Code (Cursor, Aider, etc.)
- **TUI-integrated loop** - Run the agent loop inside TUI with live output
- **Workspace automation** - Automatic git worktree/branch setup per session
- **Endless mode** - Agent stays running, polling for new tasks when queue empties
- **Notifications** - Get notified when tasks complete or need attention

## Documentation

- [Installation Guide](docs/installation.md) - Build from source, configuration
- [TUI Guide](docs/tui.md) - Keyboard shortcuts, views, workflows
- [Claude Integration](docs/claude-integration.md) - Agent setup and patterns
- [Commands Reference](docs/commands.md) - Full CLI documentation

## License

MIT
