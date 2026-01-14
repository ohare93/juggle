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

### 3. Start Your First Session

```bash
cd your-project
juggle start "My first work session"
```

This creates `.juggle/balls.jsonl` in your project directory.

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

Juggle stores configuration in `~/.config/juggle/config.json`:

```json
{
  "search_paths": ["/home/user/Development", "/home/user/projects"]
}
```

Simply running juggle in a directory will add it to the known projects (search_paths). You can edit this file directly or use:

```bash
juggle projects add /new/path
juggle projects remove /old/path
```

## Uninstallation

```bash
# Remove binary
rm ~/.local/bin/juggle
# or
sudo rm /usr/local/bin/juggle

# Remove configuration
rm -rf ~/.config/juggle

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

Or start a session in a specific directory to create `.juggle`:

```bash
cd your-project
juggle start "Initial session"
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
- Set up your first project: `juggle start "Description of your work"`
