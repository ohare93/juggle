# Ralph Loop Integration - Manual Test Plan

This test plan validates the new Ralph loop integration, covering session creation, agent execution, and TUI integration.

## Prerequisites

- Juggler binary built and in PATH: `go install ./cmd/juggle`
- An existing repo **without** `.juggler/` directory
- Claude CLI installed and authenticated (`claude --version`)

---

## Phase 1: Initial Setup

```bash
# 1.1 Navigate to test repo
cd ~/path/to/your-test-repo

# 1.2 Verify no juggler state exists
ls -la .juggler/  # Should fail or not exist

# 1.3 Create a session for agent work
juggle sessions create test-agent -m "Test the Ralph loop integration"

# 1.4 Verify session created
juggle sessions list
# Expected: Shows test-agent session

juggle sessions show test-agent
# Expected: Shows empty session with description
```

**Checkpoints:**
- [ ] `.juggler/` directory created
- [ ] `sessions/test-agent/` directory exists

---

## Phase 2: Add Balls to Session

```bash
# 2.1 Add a simple ball tagged to session
juggle plan "Create a hello world script" --session test-agent --priority high

# 2.2 Add a ball with acceptance criteria
juggle plan "Add error handling" --session test-agent --priority medium \
  --criteria "Script exits with code 1 on error" \
  --criteria "Error message printed to stderr"

# 2.3 Verify balls exist
juggle balls
# Expected: Two pending balls, both tagged with test-agent
```

**Checkpoints:**
- [ ] `balls.jsonl` contains two entries
- [ ] Both balls have `test-agent` in tags
- [ ] Acceptance criteria visible with `juggle show <ball-id>`

---

## Phase 3: Export Verification

```bash
# 3.1 Test export --format agent (without running)
juggle export --format agent --session test-agent

# Expected output structure:
# <context>
#   Session description and context
# </context>
# <progress>
#   (empty or prior progress)
# </progress>
# <balls>
#   ## ball-id [pending] (priority: high)
#   Intent: Create a hello world script
#   ...
# </balls>
# <instructions>
#   (embedded prompt.md content)
# </instructions>

# 3.2 Verify prompt contains key elements
juggle export --format agent --session test-agent | grep -E "(COMPLETE|BLOCKED|juggle todo complete)"
# Expected: Should find completion signals and CLI command references
```

**Checkpoints:**
- [ ] Export produces 4-section XML structure
- [ ] `<instructions>` contains agent prompt template
- [ ] Ball details include acceptance criteria

---

## Phase 4: Single Iteration Test (Safe Mode)

```bash
# 4.1 Run agent with 1 iteration, NO --trust flag (requires approval)
juggle agent run test-agent --iterations 1

# Expected behavior:
# - Prompt generated and piped to claude
# - Claude runs with --permission-mode acceptEdits
# - You'll see agent working, may need to approve edits
# - After iteration: progress count shown

# 4.2 Check state after iteration
juggle balls
juggle show <ball-id> --json
juggle progress show test-agent
# Expected: Some progress made, possibly todos marked done
```

**Checkpoints:**
- [ ] Agent spawns and runs one iteration
- [ ] Permission prompts appear (safe mode)
- [ ] Progress file updated

---

## Phase 5: Full Loop Test (Trust Mode)

```bash
# 5.1 Run agent with trust flag (autonomous)
juggle agent run test-agent --iterations 5 --trust

# Expected:
# - Agent runs without permission prompts
# - Iterations continue until COMPLETE signal or max reached
# - Progress displayed: "X/Y balls complete"

# 5.2 Verify completion
juggle balls
# Expected: Balls should be in complete state

cat .juggler/sessions/test-agent/progress.txt
# Expected: Timestamped entries from agent
```

**Checkpoints:**
- [ ] Trust mode runs without prompts
- [ ] Multiple iterations execute
- [ ] Balls transition to complete state
- [ ] Progress file contains agent entries

---

## Phase 6: Blocked Scenario Testing

```bash
# 6.1 Create a ball that will likely block
juggle plan "Run a command that doesn't exist: nonexistent-tool --version" \
  --session test-agent --priority urgent

# 6.2 Run agent and observe blocking behavior
juggle agent run test-agent --iterations 2 --trust

# Expected:
# - Agent attempts work, hits pre-flight check failure
# - Outputs: <promise>BLOCKED: [tool] not available...</promise>
# - Loop exits early with blocked status

# 6.3 Verify blocked state
juggle balls
# Expected: Ball shows state=blocked with reason
```

**Checkpoints:**
- [ ] Agent detects unavailable tools in pre-flight
- [ ] `<promise>BLOCKED:...</promise>` signal emitted
- [ ] Ball state set to blocked with reason

---

## Phase 7: Progress Append Testing

```bash
# 7.1 Manually append progress (simulating agent)
juggle progress append test-agent "Manual test entry from CLI"

# 7.2 Verify appended
juggle progress show test-agent
# Expected: Entry with timestamp visible

# 7.3 Check progress included in next export
juggle export --format agent --session test-agent | grep "Manual test entry"
# Expected: Found in <progress> section (last 50 lines)
```

**Checkpoints:**
- [ ] Progress append adds timestamped entry
- [ ] Progress appears in subsequent exports

---

## Phase 8: Todo Complete Testing

```bash
# 8.1 Add todos to a ball
juggle todo add "First step" "Second step" --ball <ball-id>

# 8.2 Complete todo via CLI (as agent would)
juggle todo complete <ball-id> 1 --json

# Expected JSON output:
# {"success": true, "ball_id": "...", "todo_index": 1, "todo_text": "First step"}

# 8.3 Verify todo marked done
juggle show <ball-id>
# Expected: First todo shows [x]
```

**Checkpoints:**
- [ ] `todo complete` accepts ball-id and index
- [ ] JSON output suitable for agent parsing
- [ ] Todo state persisted correctly

---

## Phase 9: Sync Ralph Testing (Optional)

If using external prd.json files:

```bash
# 9.1 Create a simple prd.json
cat > test-prd.json << 'EOF'
{
  "userStories": [
    {"id": "US-001", "title": "Test story", "priority": 3, "passes": false}
  ]
}
EOF

# 9.2 Import as balls
juggle import ralph test-prd.json --session test-agent

# 9.3 Verify import
juggle balls
# Expected: New ball from prd.json

# 9.4 Sync updates (edit prd.json to set passes: true)
sed -i 's/"passes": false/"passes": true/' test-prd.json
juggle sync ralph test-prd.json

# Expected: Ball state updated to complete
```

**Checkpoints:**
- [ ] Import creates balls from prd.json
- [ ] Sync updates existing balls
- [ ] State maps correctly (passes: true -> complete)

---

## Phase 10: TUI Session Navigation

```bash
# 10.1 Launch TUI
juggle tui

# 10.2 Navigate to test-agent session
# - Use j/k to navigate sessions panel (left)
# - Verify test-agent appears in session list
# - Press Enter or Tab to move to balls panel

# 10.3 Verify balls display correctly
# - Check state icons: ○ pending, ● in_progress, ✓ complete, ✗ blocked
# - Verify acceptance criteria visible in detail area
# - Check todos appear when ball selected
```

**Checkpoints:**
- [ ] Sessions panel shows test-agent with ball count
- [ ] Balls display with correct state icons
- [ ] Acceptance criteria and todos visible

---

## Phase 11: TUI Agent Launch

```bash
# 11.1 Position on sessions panel
# - Navigate to test-agent session with j/k

# 11.2 Press 'A' to launch agent
# Expected: Confirmation dialog appears showing:
#   - Session ID
#   - Number of balls
#   - Prompt to confirm (y/n)

# 11.3 Press 'y' to confirm
# Expected:
#   - Agent spawns in background
#   - Status bar shows: [Agent: test-agent 1/10]
#   - TUI remains responsive

# 11.4 Observe progress
# - Watch iteration counter increment
# - Activity log shows ball updates as file watcher triggers
# - Balls list refreshes as states change

# 11.5 Wait for completion
# Expected: Agent status clears, all balls show updated states
```

**Checkpoints:**
- [ ] 'A' key triggers confirmation dialog
- [ ] Agent runs while TUI stays responsive
- [ ] Status bar shows iteration progress
- [ ] File watcher updates ball list in real-time

---

## Phase 12: TUI State Verification

```bash
# 12.1 After agent completes, verify in TUI:
# - Press 'i' to toggle bottom pane to detail view
# - Check ball properties (state, blocked_reason if any)
# - Navigate todos with Tab, verify completion status

# 12.2 Check activity log
# - Press 'i' to toggle back to activity log
# - Scroll with j/k to see agent-triggered updates
# - Verify timestamps align with agent run
```

**Checkpoints:**
- [ ] Detail view shows ball properties
- [ ] Activity log captured agent events
- [ ] All ball states reflect agent work

---

## Summary Checklist

### Core Functionality
- [ ] Session creation and management
- [ ] Ball creation with session tagging
- [ ] Export --format agent produces valid prompt
- [ ] Agent runs in safe mode (with approvals)
- [ ] Agent runs in trust mode (autonomous)

### Agent Behavior
- [ ] Iterations execute until COMPLETE or max
- [ ] BLOCKED signal exits early with reason
- [ ] Progress file accumulates entries
- [ ] Todos marked complete via CLI

### TUI Integration
- [ ] Sessions panel shows sessions with counts
- [ ] 'A' key launches agent with confirmation
- [ ] Status bar shows agent progress
- [ ] File watcher updates UI in real-time

### Edge Cases
- [ ] Empty session handled gracefully
- [ ] Blocked balls show reason
- [ ] PRD sync updates states correctly (if tested)
