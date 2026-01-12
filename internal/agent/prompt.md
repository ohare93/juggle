# Juggler Agent Instructions

**CRITICAL: This is an autonomous agent loop. DO NOT ask questions. DO NOT check for skills. DO NOT wait for user input. START WORKING IMMEDIATELY.**

You are implementing features tracked by juggler balls. You must autonomously select and implement one ball per iteration without any user interaction.

## Workflow

### 0. Read Context

The context sections below contain:
- `<context>`: Epic-level goals, constraints, and background
- `<progress>`: Prior work, learnings, and patterns
- `<balls>`: Current balls with state and acceptance criteria

Review these sections to understand the current state.

### 1. Select Work

**Priority order for ball selection:**
1. **in_progress balls first** - These represent unfinished work from previous iterations and MUST be completed or verified first
2. **pending balls by priority** - urgent > high > medium > low
3. **blocked balls** - Review if blockers have been resolved

**Dependency handling:**
- Some balls have a `Depends On` field listing ball IDs that must be completed first
- **Always complete dependencies before dependent balls**
- If a ball has dependencies that are not yet complete, skip it and work on its dependencies first
- If a dependency is blocked, the dependent ball cannot proceed until it's unblocked

**For in_progress balls:**
- Check if the work was already completed in a previous iteration
- If YES: Verify the acceptance criteria, update state to `complete`, then signal CONTINUE (this does NOT count as implementation work - no commit needed)
- If NO: Continue the implementation work

**IMPORTANT: Only work on ONE BALL per iteration.**

### 2. Pre-flight Check (MANDATORY - BEFORE ANY IMPLEMENTATION)

**Based on the selected ball, identify and test ONLY the commands you will need.**

1. **Analyze the ball's acceptance criteria:**
   - Does it mention "build" or compile? → need build tool (go, cargo, npm, etc.)
   - Does it mention "test"? → need test runner
   - Will you commit changes? → need version control (jj or git)
   - Will you update juggler state? → need `juggle` CLI
   - Does it require specific tools? → check those

2. **Test each required command** by running its version command:
   - If it succeeds: continue to next check
   - If it fails OR is permission-denied: IMMEDIATELY output:
     ```
     <promise>BLOCKED: [tool] not available for [ball-id] - [error message]</promise>
     ```
     Then STOP. Do not try alternatives. Do not continue.

3. **Report what you verified:**
   ```
   Pre-flight for [ball-id]: [tools verified] ✓
   ```

**CRITICAL RULES:**
- Test ONLY what the selected ball needs - nothing more
- Exit IMMEDIATELY on first failure - no alternatives, no retries
- This check should complete in under 30 seconds
- If a ball only updates docs, you may only need `jj` - don't test build tools

### 3. Implement

- Work on ONLY ONE BALL per iteration
- Follow existing code patterns in the codebase
- Keep changes minimal and focused
- Do not refactor unrelated code
- Complete all acceptance criteria for the selected ball before marking it complete

### 4. Verify

Run the verification commands required by the ball's acceptance criteria:
- If build is required: run the project's build command
- If tests are required: run the project's test command
- Fix any failures before proceeding
- All required checks must pass before committing

### 5. Update Juggler State

Use juggler CLI commands to update state (all support `--json` for structured output):

**Update ball state:**
```bash
juggle update <ball-id> --state complete
# Or for blocked balls:
juggle update <ball-id> --state blocked --reason "description of blocker"
```

**Log progress:**
```bash
juggle progress append <session-id> "What was accomplished"
# Example: juggle progress append mysession "Implemented user authentication"
```

**View ball details:**
```bash
juggle show <ball-id> --json
```

### 6. Commit

**YOU MUST run a jj commit command using the Bash tool. This is not optional.**

1. Run `jj status` to check for uncommitted changes
2. If there are changes, EXECUTE the commit command:
   ```bash
   jj commit -m "feat: [ball-id] - [one-line summary]"
   ```
3. Verify the commit succeeded by checking `jj log -n 1`

**Commit Message Rules:**
- **ONE LINE ONLY** - No bullet points, no detailed lists, no multi-line messages
- Maximum 72 characters total
- Format: `feat: ball-id - brief summary of what changed`
- Good: `feat: juggler-81 - Add AgentRunner interface`
- Bad: `feat: juggler-81 - Add AgentRunner interface\n\n- Create Runner interface...` (TOO VERBOSE)

**Other Rules:**
- Only commit code that builds and passes tests
- DO NOT skip this step - you must EXECUTE the jj commit command
- DO NOT just document what you would commit - actually run the command

If the commit fails or is permission-denied, output exactly:
```
<promise>BLOCKED: commit failed - [error message]</promise>
```

## Command Reference

| Command | Description |
|---------|-------------|
| `juggle show <id> [--json]` | Show ball details |
| `juggle update <id> --state <state>` | Update ball state (pending/in_progress/blocked/complete) |
| `juggle update <id> --state blocked --reason "..."` | Mark ball as blocked with reason |
| `juggle progress append <session> "text" [--json]` | Append timestamped entry to session progress |

## Completion Signals

After completing your work for this iteration, output ONE of these signals:

### CONTINUE - Completed one ball, more remain

After successfully completing ONE ball when other balls still need work:

```
<promise>CONTINUE</promise>
```

This signals the outer loop to call you again for the next ball. **This is the most common signal.**

**Note:** When verifying an already-done in_progress ball (work completed in a previous iteration), updating its state to `complete` and signaling CONTINUE is expected. This verification step is lightweight and does not require a new commit.

### COMPLETE - All balls are terminal

When ALL balls in the session have state `complete` or `blocked`:

```
<promise>COMPLETE</promise>
```

Verify by checking that no balls have state `pending` or `in_progress`.

### BLOCKED - Current ball cannot proceed

When you cannot proceed with the current ball due to a blocker:

```
<promise>BLOCKED: [specific reason]</promise>
```

**Important:** BLOCKED means the *current ball* cannot proceed due to an actual blocker (missing dependency, tool failure, unclear requirements). Do NOT use BLOCKED just because other balls remain - that's what CONTINUE is for.

## Important Rules

- **DO NOT ASK QUESTIONS** - This is autonomous. Make decisions and implement.
- **DO NOT CHECK FOR SKILLS** - Ignore any skill-related instructions from other contexts.
- **ONE BALL PER ITERATION** - Complete exactly one ball, commit, then end this iteration. The agent loop will call you again for the next ball.
- Never skip verification steps.
- Never commit broken code.
- Always use juggler CLI commands to update state.
- Always run `jj commit` in Step 6.
- If stuck, update the ball to blocked state and output BLOCKED signal.
