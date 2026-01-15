# Installation Guide

## Prerequisites

- Go 1.21 or later (for building from source)
- Git

## Installation Methods

### Method 1: Install Script (Recommended)

The quickest way to install Juggle:

```bash
curl -sSL https://raw.githubusercontent.com/ohare93/juggle/main/install.sh | bash
```

Or download and inspect first:

```bash
curl -O https://raw.githubusercontent.com/ohare93/juggle/main/install.sh
chmod +x install.sh
./install.sh
```

The script will:

1. Download the latest release for your platform
2. Install the binary to `~/.local/bin/juggle`
3. Add `~/.local/bin` to your PATH if needed

### Method 2: Build from Source

```bash
# Clone the repository
git clone https://github.com/ohare93/juggle.git
cd juggle

# Build the binary
go build -o juggle ./cmd/juggle

# Install to your PATH
sudo mv juggle /usr/local/bin/
# or
mv juggle ~/.local/bin/
```

### Method 3: Go Install

If you have Go installed:

```bash
go install github.com/ohare93/juggle/cmd/juggle@latest
```

Make sure `$GOPATH/bin` (usually `~/go/bin`) is in your PATH.

### Method 4: Download Pre-built Binary

1. Go to the [Releases page](https://github.com/ohare93/juggle/releases)
2. Download the binary for your platform
3. Extract and move to a directory in your PATH:

```bash
# Linux/macOS
tar -xzf juggle-linux-amd64.tar.gz
sudo mv juggle /usr/local/bin/

# Or install to user directory
mkdir -p ~/.local/bin
mv juggle ~/.local/bin/
```

## Initial Setup

### 1. Verify Installation

```bash
juggle --version
```

### 2. Configure Search Paths

Tell Juggle where to look for projects:

```bash
# Add your development directory
juggle projects add ~/Development

# Add multiple paths
juggle projects add ~/work
juggle projects add ~/personal-projects
```

Juggle will search these directories for any project containing a `.juggle` folder.

### 3. Initialize Your First Project

```bash
cd your-project

# Create a session to group related work
juggle sessions create my-feature -m "My first feature"

# Create your first task (ball)
juggle plan --session my-feature --title "First task"
```

This creates the `.juggle/` directory in your project with your session and ball.

## Agent Skill

Juggle includes a skill (`skills/juggle/SKILL.md`) that teaches AI agents how to manage tasks during agent loops. The skill is agent-agnostic - any agent supporting skills can use it.

### What the Skill Provides

The skill teaches agents:
- **CLI commands**: `juggle plan`, `juggle update`, `juggle progress append`
- **Ball states**: pending → in_progress → complete/blocked/researched
- **Session management**: Grouping balls, adding context, logging progress
- **Best practices**: Writing verifiable acceptance criteria, handling in-progress balls

### Claude Code

```bash
claude plugin add github:ohare93/juggle
```

The skill activates automatically when Claude detects a `.juggle/` directory or when you mention juggle, balls, or sessions.

### Other Agents

For agents that support skills but use different installation methods, point them to `skills/juggle/SKILL.md` in this repository. The skill content is standard markdown with YAML frontmatter.

## Worktrees (Parallel Agent Loops)

Run multiple agent loops simultaneously using VCS worktrees. Each worktree gets its own agent while sharing the same ball state from the main repo.

### Why Worktrees?

- **Parallel execution**: Run agents on different balls concurrently
- **Isolated workspaces**: Each agent works in its own directory
- **Shared state**: All worktrees read/write to the main repo's `.juggle/` directory
- **No conflicts**: Agents work on different branches/changes

### Setup Steps

#### 1. Create VCS Worktree

**For jj (Jujutsu):**

```bash
# From your main repo
jj workspace add ../my-feature-worktree

# Or create on a specific bookmark
jj workspace add ../my-feature-worktree --revision my-bookmark
```

**For Git:**

```bash
# From your main repo
git worktree add ../my-feature-worktree feature-branch
```

#### 2. Build in Worktree (if needed)

If your project requires building:

```bash
cd ../my-feature-worktree
go build ./...  # or npm install, cargo build, etc.
```

#### 3. Register with Juggle

From the **main repo** (not the worktree):

```bash
juggle worktree add ../my-feature-worktree
```

This creates:

- An entry in `.juggle/config.json` listing the worktree
- A `.juggle/link` file in the worktree pointing to the main repo

### Running Parallel Agents

After setup, run agents from different terminals:

```bash
# Terminal 1 (main repo)
cd ~/Development/my-project
juggle agent run loop-a

# Terminal 2 (worktree)
cd ~/Development/my-feature-worktree
juggle agent run loop-b
```

Both agents share the same balls and progress from the main repo's `.juggle/` directory.

### Worktree Commands

```bash
# List registered worktrees
juggle worktree list

# Check worktree status
juggle worktree status

# Unregister a worktree (doesn't delete it)
juggle worktree forget ../my-feature-worktree
```

### Cleanup

```bash
# Unregister from juggle
juggle worktree forget ../my-feature-worktree

# Remove VCS worktree
jj workspace forget my-feature-worktree  # for jj
git worktree remove ../my-feature-worktree  # for git
```

## Updating

### Using Install Script

```bash
curl -sSL https://raw.githubusercontent.com/ohare93/juggle/main/install.sh | bash
```

### Using Go

```bash
go install github.com/ohare93/juggle/cmd/juggle@latest
```

### Manual Update

Download the latest release and replace the binary.

## Configuration

Juggle stores configuration in `~/.juggle/config.json`:

```json
{
  "search_paths": ["/home/user/Development", "/home/user/projects"],
  "vcs": "jj"
}
```

Simply running juggle in a directory will add it to the known projects (search_paths). You can edit this file directly or use:

```bash
juggle projects add /new/path
juggle projects remove /old/path
```

### Version Control System (VCS)

Juggle supports both **Git** and **jj (Jujutsu)** for version control operations (committing changes after agent iterations).

**Auto-detection (default):** Juggle automatically detects which VCS to use:

1. If a `.jj` directory exists → uses jj
2. If a `.git` directory exists → uses git
3. Otherwise → defaults to git

**Manual configuration:** You can override auto-detection at the global or project level:

```bash
# View current VCS settings and detection
juggle config vcs show

# Set globally (stored in ~/.juggle/config.json)
juggle config vcs set jj
juggle config vcs set git

# Set for current project only (stored in .juggle/config.json)
juggle config vcs set jj --project
juggle config vcs set git --project

# Clear setting to use auto-detection
juggle config vcs clear
juggle config vcs clear --project
```

**Resolution priority (highest to lowest):**

1. Project config (`.juggle/config.json` `vcs` field)
2. Global config (`~/.juggle/config.json` `vcs` field)
3. Auto-detect: `.jj` directory > `.git` directory > git (default)

## Uninstallation

```bash
# Remove binary
rm ~/.local/bin/juggle
# or
sudo rm /usr/local/bin/juggle

# Remove configuration
rm -rf ~/.juggle

# Remove project data (optional - only if you want to delete all tracked sessions)
find ~/Development -name ".juggle" -type d -exec rm -rf {} +
```

## Troubleshooting

### Command not found

Ensure `~/.local/bin` is in your PATH:

```bash
# Add to ~/.bashrc or ~/.zshrc
export PATH="$HOME/.local/bin:$PATH"

# Reload shell
source ~/.bashrc  # or source ~/.zshrc
```

### No projects found

Add search paths:

```bash
juggle projects add ~/Development
```

Or initialize a session in a specific directory to create `.juggle`:

```bash
cd your-project
juggle sessions create my-feature -m "Initial feature"
```

### Permission denied

If installing to `/usr/local/bin`, you need sudo:

```bash
sudo mv juggle /usr/local/bin/
```

Or install to user directory without sudo:

```bash
mkdir -p ~/.local/bin
mv juggle ~/.local/bin/
```

## Getting Help

- Documentation: See `docs/` directory
- Issues: https://github.com/ohare93/juggle/issues
- Commands: `juggle --help` or `juggle <command> --help`

## Next Steps

- Explore commands with `juggle --help`
- Set up your first project: `juggle sessions create my-feature -m "Description"`
