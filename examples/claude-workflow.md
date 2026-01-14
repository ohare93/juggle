# Claude Code Workflow Examples

This document provides realistic examples of how Claude Code agents use juggle in practice.

## Example 1: Starting a New Feature

**Scenario**: User wants to add a new feature to their application.

```
User: "I want to add a user profile page with avatar upload"

Agent: Let me check the current project state.
```

**Agent runs**: `juggle`

```
Output: No current ball found in this project
```

**Agent continues**:
```
I see there's no active ball for this task. Let me create one to track this work.
```

**Agent runs**: `juggle start`
- Intent: "Add user profile page with avatar upload"
- Priority: medium

```
Output: ✓ Started ball: myapp-12
```

**Agent continues**:
```
I've created a ball to track this work. Let me break down the tasks:
```

**Agent runs**:
```bash
juggle myapp-12 todo add \
  "Design profile page UI mockup" \
  "Create Profile component with React" \
  "Add avatar upload with file validation" \
  "Implement image storage (S3 or local)" \
  "Add profile update API endpoint" \
  "Write tests for profile functionality"
```

**Agent runs**: `juggle myapp-12 in-air`

**Agent continues**:
```
I'll start with the UI design. [proceeds with work...]
```

---

## Example 2: Resuming Previous Work

**Scenario**: User returns to continue work from a previous session.

```
User: "Let's continue working on the profile page"

Agent: Let me check the current state.
```

**Agent runs**: `juggle`

```
Output:
Ball: myapp-12
Intent: Add user profile page with avatar upload
State: juggling:in-air
Priority: medium
Started: 2 hours ago
Last activity: 5 minutes ago

Todos: 2/6 complete (33%)
  1. [✓] Design profile page UI mockup
  2. [✓] Create Profile component with React
  3. [ ] Add avatar upload with file validation
  4. [ ] Implement image storage (S3 or local)
  5. [ ] Add profile update API endpoint
  6. [ ] Write tests for profile functionality
```

**Agent continues**:
```
I see we've completed the UI mockup and Profile component. 
The next task is adding avatar upload with file validation. Let me continue from there.
[proceeds with work on task 3...]
```

**Agent marks progress**:
```bash
juggle myapp-12 todo done 3
```

---

## Example 3: Asking for User Input

**Scenario**: Agent needs user decision during work.

```
Agent: I need to decide on the image storage approach. 
```

**Agent runs**: `juggle myapp-12 needs-thrown "Which storage: S3 or local filesystem?"`

**Agent continues**:
```
For the avatar storage, we have two options:

1. **AWS S3**: More scalable, handles high traffic, costs money
2. **Local filesystem**: Simpler, free, but requires backup strategy

Which would you prefer?
```

```
User: "Let's use S3, we'll scale eventually"

Agent: Great choice! I'll implement S3 storage.
```

**Agent runs**: `juggle myapp-12 in-air`

**Agent continues**:
```
[implements S3 storage solution...]
```

---

## Example 4: Multiple Concurrent Tasks

**Scenario**: User has multiple things in progress.

```
User: "While you're working on the profile page, 
       I also need to fix a bug in the login flow"

Agent: Let me check current balls.
```

**Agent runs**: `juggle balls`

```
Output:
 ID          STATE                PRIORITY  TODOS     INTENT
myapp-12   juggling:in-air     medium    3/6       Add user profile page
```

**Agent continues**:
```
I see one active ball for the profile page. Since this is a separate 
concern, I'll create a new ball for the login bug fix.
```

**Agent runs**: `juggle start`
- Intent: "Fix bug in login flow"
- Priority: high (bugs get higher priority)

**Agent runs**:
```bash
juggle myapp-13 tag add bug authentication
juggle myapp-13 todo add \
  "Reproduce login bug" \
  "Identify root cause" \
  "Implement fix" \
  "Add regression test"
```

**Agent runs**: `juggle myapp-13 in-air`

**Agent continues**:
```
I've created a separate ball for the login bug (myapp-13) with high priority.
Let me investigate the issue. [proceeds with debugging...]
```

---

## Example 5: Completing Work

**Scenario**: Agent finishes a task and prepares for user review.

```
Agent: I've completed the login bug fix. The issue was in the token refresh logic.
```

**Agent runs**:
```bash
juggle myapp-13 todo done 1
juggle myapp-13 todo done 2
juggle myapp-13 todo done 3
juggle myapp-13 todo done 4
```

**Agent runs**: `juggle myapp-13 needs-caught "Login bug fixed, please test the flow"`

**Agent continues**:
```
All tasks complete! The fix is ready for testing:
- Reproduced the bug ✓
- Identified root cause (token refresh timing) ✓
- Implemented fix ✓
- Added regression test ✓

Please test the login flow to verify the fix works correctly.
```

```
User: "Tested and works great!"

Agent: Excellent!
```

**Agent runs**: `juggle myapp-13 complete "Login bug fixed and verified"`

---

## Example 6: Switching Between Balls

**Scenario**: Agent needs to switch focus between tasks.

```
User: "The profile page can wait, let's prioritize the API documentation"

Agent: Let me check what's currently active.
```

**Agent runs**: `juggle balls`

```
Output:
 ID          STATE                PRIORITY  TODOS     INTENT
myapp-12   juggling:in-air     medium    4/6       Add user profile page
myapp-14   ready               medium    -         Write API documentation
```

**Agent continues**:
```
I see the profile page is active and we have a planned ball for API documentation.
Let me switch focus.
```

**Agent runs**: `juggle myapp-12 needs-thrown "Pausing for API docs priority"`

**Agent runs**: `juggle myapp-14 in-air`

**Agent runs**:
```bash
juggle myapp-14 todo add \
  "Document authentication endpoints" \
  "Document user CRUD endpoints" \
  "Add API examples" \
  "Generate OpenAPI spec"
```

**Agent continues**:
```
I've paused the profile work and activated the API documentation ball.
Starting with authentication endpoints...
```

---

## Example 7: Finding Next Priority

**Scenario**: Multiple balls exist and agent needs to determine what to work on.

```
User: "What should we work on next?"

Agent: Let me check what needs attention.
```

**Agent runs**: `juggle next`

```
Output:
→ Next ball: myapp-15
  Intent: Fix critical security vulnerability
  State: juggling:needs-caught
  Priority: urgent
  Last activity: 1 hour ago

This ball needs your review before completion.
```

**Agent continues**:
```
The security vulnerability fix (myapp-15) is the highest priority item.
It's marked as needing review. Would you like to review that work,
or should we start something new?
```

---

## Example 8: Working With Existing Ball

**Scenario**: User mentions a task that already has a ball.

```
User: "Can you add error handling to the API?"

Agent: Let me check if we're already tracking this.
```

**Agent runs**: `juggle search API`

```
Output:
Found 2 ball(s)
 ID          STATE                INTENT
myapp-14   juggling:in-air     Write API documentation
myapp-16   ready               Add error handling to API endpoints
```

**Agent continues**:
```
I found an existing planned ball (myapp-16) for adding error handling to API endpoints!
Let me activate that ball rather than creating a duplicate.
```

**Agent runs**: `juggle myapp-16 in-air`

**Agent runs**:
```bash
juggle myapp-16 todo add \
  "Add try-catch blocks to all endpoints" \
  "Implement error response format" \
  "Add error logging" \
  "Update API documentation with error codes"
```

**Agent continues**:
```
I've activated the existing ball for this work. Starting implementation...
```

---

## Example 9: Dropping a Ball

**Scenario**: User decides not to pursue a feature.

```
User: "Actually, let's not do the dark mode feature. It's not worth the complexity."

Agent: I'll mark that ball as dropped.
```

**Agent runs**: `juggle balls`

```
Output:
 ID          STATE            INTENT
myapp-17   ready            Add dark mode support
```

**Agent runs**: `juggle myapp-17 dropped "User decided complexity not worth it"`

**Agent continues**:
```
I've marked the dark mode ball as dropped. It won't appear in active lists anymore
but remains in the archive for reference.
```

---

## Example 10: Complex Multi-Step Feature

**Scenario**: Large feature requiring extensive planning.

```
User: "I want to implement a complete payment system with Stripe"

Agent: This is a significant feature. Let me create a ball and break it down thoroughly.
```

**Agent runs**: `juggle start`
- Intent: "Implement complete Stripe payment system"
- Priority: high

**Agent runs**:
```bash
juggle myapp-18 tag add backend payment stripe critical

juggle myapp-18 todo add \
  "Set up Stripe account and get API keys" \
  "Install and configure Stripe SDK" \
  "Design payment database schema" \
  "Implement checkout session creation" \
  "Add payment webhook handlers" \
  "Implement subscription management" \
  "Add payment history page" \
  "Implement refund functionality" \
  "Add comprehensive error handling" \
  "Write integration tests" \
  "Test with Stripe test mode" \
  "Document payment flow for team"
```

**Agent runs**: `juggle myapp-18 needs-thrown "Need Stripe API keys before starting implementation"`

**Agent continues**:
```
I've created a comprehensive plan with 12 tasks for the payment system.

Before I can begin implementation, I'll need:
1. Stripe account credentials (API keys)
2. Confirmation on which subscription tiers to support

Once you provide those, I'll mark the ball as in-air and begin implementation.
```

---

## Best Practices Demonstrated

### 1. Always Check State First
Every example starts with checking current state via `juggle` or `juggle balls`.

### 2. Reuse Existing Balls
Example 8 shows searching for existing balls before creating new ones.

### 3. Clear State Transitions
- `needs-thrown` when user input needed
- `in-air` when actively working
- `needs-caught` when ready for review
- `complete` when fully done
- `dropped` when abandoned

### 4. Comprehensive Task Breakdown
Examples show adding multiple todos at once for better planning.

### 5. Use Tags for Context
Examples include relevant tags (bug, authentication, payment, etc.).

### 6. Explicit Communication
Agents clearly explain what they're doing with juggle and why.

### 7. Priority Management
Higher priority (urgent, high) for bugs and critical features.

### 8. Multiple Ball Management
Examples show handling concurrent work streams effectively.

---

## Common Patterns

### Pattern: Start → Plan → Execute → Review → Complete

```bash
juggle start                        # Create ball
juggle <id> todo add "..." "..."    # Plan tasks
juggle <id> in-air                  # Start work
# ... do work ...
juggle <id> todo done 1             # Mark progress
juggle <id> needs-caught            # Ready for review
# ... user reviews ...
juggle <id> complete                # Finish
```

### Pattern: Check → Resume → Continue

```bash
juggle                              # Check state
# See existing ball with todos
juggle <id> in-air                  # Resume work
# ... continue from where left off ...
```

### Pattern: Search → Reuse vs Create

```bash
juggle search <keyword>             # Search existing
# If found: use existing ball
# If not found: juggle start
```

These examples show how juggle integrates naturally into Claude Code workflows, providing structure and context across conversations while keeping the user informed of progress.
