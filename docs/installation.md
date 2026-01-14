# Installation Guide

## Prerequisites

- Go 1.21 or later (for building from source)
- Git
- (Optional) Zellij for terminal workspace integration

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

## Claude Code Integration (Optional)

### Recommended Setup with Workflow Enforcement

For projects using Claude Code:

```bash
cd your-project
juggle setup-claude --install-hooks
```

This installs:
- **Agent instructions** in `.claude/CLAUDE.md`
- **Workflow enforcement hooks** in `.claude/hooks.json`
- **Marker file system** for check tracking

**Validation Steps:**
```bash
# Verify installation
ls .claude/
# Expected: CLAUDE.md  hooks.json

# Check instructions at document start
head -30 .claude/CLAUDE.md
# Should see: "🚫 CRITICAL BLOCKING REQUIREMENT"

# Test hook manually
juggle reminder
# Should show reminder (first time)

# Test check command
juggle check
# Should show current state and guidance
```

**Important:**
- Restart Claude Code after installation
- Instructions appear at top of CLAUDE.md (critical positioning)
- Hook runs before each interaction (<50ms overhead)
- See [Workflow Enforcement Guide](./workflow-enforcement.md) for details

### Alternative: Global Installation

For use across all projects:

```bash
juggle setup-claude --global
```

Installs to `~/.claude/CLAUDE.md` instead of project-local `.claude/CLAUDE.md`.

**Note:** Hooks are project-specific, so use `--install-hooks` in each project even with global instructions.

## Zellij Integration (Optional)

For automatic tab switching and better context tracking:

### 1. Install Zellij

```bash
# macOS
brew install zellij

# Linux (cargo)
cargo install zellij

# Or download from https://github.com/zellij-org/zellij/releases
```

### 2. Configure Zellij

Juggle automatically detects Zellij sessions. Start Zellij in your project:

```bash
cd your-project
zellij
```

Now Juggle commands will track which Zellij tab you're in and can automatically switch tabs:

```bash
juggle jump <ball-id>  # Switches to the ball's tab
juggle next            # Jumps to highest priority ball
```

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
  "search_paths": [
    "/home/user/Development",
    "/home/user/projects"
  ],
  "vcs": "jj"
}
```

You can edit this file directly or use:

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

### Zellij integration not working

Verify Zellij is running:

```bash
echo $ZELLIJ_SESSION_NAME
```

If empty, you're not in a Zellij session. Start one:

```bash
zellij
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

- Read the [Claude Integration Guide](./claude-integration.md) for AI-assisted workflows
- Explore commands with `juggle --help`
- Set up your first project: `juggle sessions create my-feature -m "Description"`
