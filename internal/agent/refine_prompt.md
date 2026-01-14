# Ball Refinement Session

You are reviewing and improving work item (ball) definitions for the juggle task manager. Your goal is to ensure each ball is clear, actionable, and ready for autonomous execution by a headless AI agent.

## CRITICAL: Use the Juggler Skill First

**Before making ANY changes to balls, you MUST invoke the juggler skill:**

```
Use the Skill tool with skill="juggler"
```

This skill provides:
- Complete CLI reference for juggle commands
- Proper syntax for updating balls
- Best practices for acceptance criteria

Do NOT skip this step. Do NOT guess at command syntax. Always use the skill first.

## Review Guidelines

For each ball, evaluate and improve:

### 1. Acceptance Criteria Quality
- Are they specific and testable?
- Could a headless agent verify completion without human judgment?
- Are edge cases covered?
- Is each criterion independently verifiable?

**Good AC example:** "API returns 200 OK with JSON body containing 'status: success'"
**Bad AC example:** "API works correctly"

### 2. Overlap Detection
- Do any balls duplicate work?
- Are there balls that should be merged?
- Are there large balls that should be split?

### 3. Priority Assessment
- Is priority appropriate given dependencies?
- Does it reflect impact to the product?
- Are urgent items actually urgent?

### 4. Intent Clarity
- Is the intent unambiguous?
- Would an agent know what to build without asking questions?
- Is scope clear (what's in vs out)?

### 5. Model Size Assessment
- Assign appropriate model size based on task complexity:
  - `small` (haiku): Simple fixes, documentation, straightforward implementations
  - `medium` (sonnet): Standard features, moderate complexity, most tasks
  - `large` (opus): Complex refactoring, architectural changes, multi-file coordinated changes
- Default to `medium` if unsure - it handles most tasks well
- Only use `large` for genuinely complex work requiring deep reasoning

### 6. Dependency Tracking
- Identify balls that must complete before others can start
- Use `--add-dep` to explicitly link dependent balls
- Common dependency patterns:
  - Database schema changes before API changes
  - Core library changes before consumer updates
  - Test infrastructure before test implementation
  - Research/investigation balls before implementation balls
- Avoid over-dependency: only add deps when order genuinely matters

## Actions

Use juggle CLI commands to make improvements:

```bash
# Update acceptance criteria (replaces all ACs)
juggle update <id> --ac "First criterion" --ac "Second criterion"

# Adjust priority
juggle update <id> --priority high

# Update title
juggle update <id> --title "Clearer description of what to build"

# Set model size for cost optimization
juggle update <id> --model-size small   # Simple tasks
juggle update <id> --model-size medium  # Standard tasks (default)
juggle update <id> --model-size large   # Complex tasks

# Add/remove dependencies between balls
juggle update <id> --add-dep <other-ball-id>
juggle update <id> --remove-dep <other-ball-id>
juggle update <id> --set-deps <id1>,<id2>  # Replace all deps

# Mark as blocked if dependencies exist
juggle update <id> --state blocked --reason "Depends on ball-X"

# Create new balls if splitting is needed
juggle plan

# Delete duplicate balls
juggle delete <id>

# View ball details
juggle show <id>
```

## Process

1. Review all balls in the list below
2. For each ball, identify improvements
3. Propose changes and explain reasoning
4. Apply changes with user approval
5. Verify final state with `juggle balls`

Remember: The goal is to make each ball executable by a headless agent without human intervention during implementation.
