# Configuration Reference

Juggle uses two configuration files:
- **Global config**: `~/.juggle/config.json` - User-wide settings
- **Project config**: `.juggle/config.json` - Repository-specific settings

## Global Configuration

Location: `~/.juggle/config.json`

### Full Schema

```json
{
  "search_paths": [
    "/home/user/Development",
    "/home/user/projects"
  ],
  "iteration_delay_minutes": 5,
  "iteration_delay_fuzz": 2,
  "overload_retry_minutes": 10,
  "vcs": "jj",
  "agent_provider": "claude",
  "model_overrides": {
    "opus": "anthropic/claude-opus-4-5"
  }
}
```

### Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `search_paths` | string[] | `[]` | Directories to scan for `.juggle/` projects. Added automatically when creating balls. |
| `iteration_delay_minutes` | int | `0` | Base delay between agent iterations in minutes. 0 = no delay. |
| `iteration_delay_fuzz` | int | `0` | Random variance (+/-) in delay minutes. Example: 5 ± 2 means 3-7 minutes. |
| `overload_retry_minutes` | int | `10` | Minutes to wait before retrying after rate limit retries are exhausted (529 errors). |
| `vcs` | string | `""` | Global VCS preference: `"git"`, `"jj"`, or `""` (auto-detect). |
| `agent_provider` | string | `""` | Global agent provider: `"claude"`, `"opencode"`, or `""` (defaults to claude). |
| `model_overrides` | object | `{}` | Custom model mappings. Keys: `small`, `medium`, `large`, `haiku`, `sonnet`, `opus`. Values: provider-specific model IDs. |

### Managing Global Config via CLI

```bash
# View all configuration
juggle config

# Iteration delay
juggle config delay show
juggle config delay set 5           # 5 minutes
juggle config delay set 5 --fuzz 2  # 5 ± 2 minutes
juggle config delay clear

# VCS preference
juggle config vcs show
juggle config vcs set jj
juggle config vcs clear
```

### Search Path Behavior

Search paths are automatically added when you create a ball in a new project:

```bash
cd ~/new-project
juggle plan --title "First task"
# ~/new-project is automatically added to search_paths
```

To manually manage:

```bash
juggle projects list          # See discovered projects
juggle projects add ~/myproj  # Manually add
juggle projects rm ~/myproj   # Remove from discovery
```

## Project Configuration

Location: `.juggle/config.json` (in project root)

### Full Schema

```json
{
  "default_acceptance_criteria": [
    "All tests pass",
    "No linting errors"
  ],
  "vcs": "jj",
  "agent_provider": "opencode",
  "model_overrides": {
    "large": "anthropic/claude-opus-4-5"
  }
}
```

### Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `default_acceptance_criteria` | string[] | `[]` | Repository-level ACs applied to all balls and sessions in this project. |
| `vcs` | string | `""` | Project VCS preference: `"git"`, `"jj"`, or `""` (inherit from global/auto-detect). |
| `agent_provider` | string | `""` | Project agent provider: `"claude"`, `"opencode"`, or `""` (inherit from global). |
| `model_overrides` | object | `{}` | Project-specific model mappings. Merged with global overrides (project takes precedence). |

### Managing Project Config via CLI

```bash
# View project configuration
juggle config

# Acceptance criteria
juggle config ac list
juggle config ac add "All tests pass"
juggle config ac add "Build succeeds"
juggle config ac set --edit   # Open in $EDITOR
juggle config ac clear

# VCS preference
juggle config vcs show
juggle config vcs set git
juggle config vcs clear
```

### Acceptance Criteria Hierarchy

Acceptance criteria are inherited at three levels:

1. **Repository level** (`default_acceptance_criteria` in `.juggle/config.json`)
   - Applied to ALL balls in the project
   - Example: "Build passes", "No linting errors"

2. **Session level** (`acceptance_criteria` in `.juggle/sessions/<id>/session.json`)
   - Applied to balls tagged with that session
   - Example: "Feature tests pass", "Documentation updated"

3. **Ball level** (`acceptance_criteria` in the ball itself)
   - Specific to that ball
   - Example: "Login button appears on homepage"

The agent sees all three levels combined when working on a ball.

## Session Configuration

Location: `.juggle/sessions/<session-id>/session.json`

### Full Schema

```json
{
  "id": "my-feature",
  "description": "Implement user authentication",
  "context": "We need OAuth2 with Google provider...",
  "default_model": "medium",
  "acceptance_criteria": [
    "OAuth flow works end-to-end",
    "Error messages are user-friendly"
  ],
  "created_at": "2025-01-10T10:30:00Z",
  "updated_at": "2025-01-14T15:45:00Z"
}
```

### Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `id` | string | required | Session identifier (must match directory name) |
| `description` | string | `""` | Human-readable description |
| `context` | string | `""` | Rich context for agent memory across iterations |
| `default_model` | string | `""` | Default model size for balls: `"small"`, `"medium"`, `"large"`, or `""` |
| `acceptance_criteria` | string[] | `[]` | Session-level ACs applied to all balls with this session tag |
| `created_at` | string | auto | ISO 8601 timestamp |
| `updated_at` | string | auto | ISO 8601 timestamp |

### Managing Sessions via CLI

```bash
# Create session with ACs
juggle sessions create my-feature \
  --ac "All tests pass" \
  --ac "Documentation updated"

# Edit session properties
juggle sessions edit my-feature

# Update context
juggle sessions context my-feature --edit

# View session details
juggle sessions show my-feature
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `JUGGLER_CURRENT_BALL` | Explicitly target a specific ball (useful for multi-agent setups) |
| `EDITOR` | Editor for `--edit` commands (defaults to `vi`) |

## VCS Resolution Order

When determining which VCS to use:

1. **Project config** (`.juggle/config.json` → `vcs`)
2. **Global config** (`~/.juggle/config.json` → `vcs`)
3. **Auto-detection**: Check for `.jj` directory, then `.git`

## Agent Provider Resolution Order

When determining which agent provider to use:

1. **CLI flag** (`--provider claude` or `--provider opencode`)
2. **Project config** (`.juggle/config.json` → `agent_provider`)
3. **Global config** (`~/.juggle/config.json` → `agent_provider`)
4. **Default**: `claude`

### Supported Providers

| Provider | Binary | Description |
|----------|--------|-------------|
| `claude` | `claude` | Claude Code CLI (default) |
| `opencode` | `opencode` | OpenCode CLI |

### Model Mapping

Models are mapped from canonical names to provider-specific identifiers:

| Canonical | Claude Code | OpenCode |
|-----------|-------------|----------|
| `small` / `haiku` | `haiku` | `anthropic/claude-3-5-haiku-latest` |
| `medium` / `sonnet` | `sonnet` | `anthropic/claude-sonnet-4-5` |
| `large` / `opus` | `opus` | `anthropic/claude-opus-4-5` |

Use `model_overrides` to customize these mappings when new models are released:

```json
{
  "model_overrides": {
    "opus": "anthropic/claude-opus-5"
  }
}
```

### Model Override Priority

When both global and project configs define `model_overrides`, settings are merged with project taking precedence:

1. **Project config overrides** (`.juggle/config.json`)
2. **Global config overrides** (`~/.juggle/config.json`)
3. **Provider default mappings**

Example: If global has `"opus": "anthropic/claude-opus-4"` and project has `"opus": "anthropic/claude-opus-5"`, the project version wins.

When an override is applied, juggle logs the mapping:
```
Model override: opus → anthropic/claude-opus-5
```

### Using the --provider Flag

```bash
# Use OpenCode for this run
juggle agent run --session my-session --provider opencode

# Use Claude (explicit)
juggle agent run --session my-session --provider claude
```

## Rate Limit Handling

When Claude returns rate limit errors (429 or overloaded):

1. Claude's built-in retry logic handles initial retries with exponential backoff
2. If retries exhaust (529 error), juggle waits `overload_retry_minutes` (default: 10)
3. Can be overridden per-run with `--max-wait` flag
4. Set `--max-wait 0` to wait indefinitely

## Testing Configuration

For testing, you can override configuration locations:

```bash
# Override global config directory
juggle --config-home /tmp/test-juggle agent run

# Override .juggle directory name
juggle --juggle-dir .juggle-test plan --title "Test"

# Override working directory
juggle --project-dir /other/path balls
```

These flags are primarily for integration testing and CI/CD environments.
