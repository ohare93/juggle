package claude

// GlobalInstructionsTemplate is the minimal template for ~/.claude/CLAUDE.md
// It points to project-specific CLAUDE.md files where juggler may be in use
const GlobalInstructionsTemplate = `
## Juggler Task Management

**CRITICAL**: Check project directory for ` + "`.claude/CLAUDE.md`" + ` file.
Project-specific instructions OVERRIDE these defaults.
Look for BLOCKING requirements at TOP of project CLAUDE.md files.

Juggler may be in use - ALWAYS check project instructions first.

### What is Juggler?

Juggler is a task management system that tracks concurrent work sessions ("balls")
across multiple projects. If a project uses juggler, you will find MANDATORY
workflow instructions at the top of the project's ` + "`.claude/CLAUDE.md`" + ` file.

### How to Check

1. Look for ` + "`.claude/CLAUDE.md`" + ` in the current working directory
2. If found, READ the top section for BLOCKING requirements
3. Follow the project-specific instructions exactly

**WARNING**: Failing to follow project juggler instructions will result in:
- Critical workflow violations
- Request to restart the session
- Corrupted task state

Do NOT assume juggler is not in use - ALWAYS check the project directory first.
`
