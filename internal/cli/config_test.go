package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ohare93/juggle/internal/session"
)

// setupTestProject creates a temp directory with a .juggler directory
func setupTestProject(t *testing.T) (string, func()) {
	tmpDir := t.TempDir()
	jugglerDir := filepath.Join(tmpDir, ".juggler")
	if err := os.MkdirAll(jugglerDir, 0755); err != nil {
		t.Fatalf("failed to create .juggler dir: %v", err)
	}

	// Set up global options
	origConfigHome := GlobalOpts.ConfigHome
	origProjectDir := GlobalOpts.ProjectDir
	GlobalOpts.ConfigHome = tmpDir
	GlobalOpts.ProjectDir = tmpDir

	cleanup := func() {
		GlobalOpts.ConfigHome = origConfigHome
		GlobalOpts.ProjectDir = origProjectDir
	}

	return tmpDir, cleanup
}

// TestConfigACList_Empty tests listing ACs when none exist
func TestConfigACList_Empty(t *testing.T) {
	_, cleanup := setupTestProject(t)
	defer cleanup()

	// Run list command - should not error
	err := runConfigACList(configACListCmd, []string{})
	if err != nil {
		t.Errorf("expected no error with empty ACs, got: %v", err)
	}
}

// TestConfigACAdd tests adding a criterion
func TestConfigACAdd(t *testing.T) {
	projectDir, cleanup := setupTestProject(t)
	defer cleanup()

	// Add a criterion
	err := runConfigACAdd(configACAddCmd, []string{"Tests pass"})
	if err != nil {
		t.Fatalf("failed to add criterion: %v", err)
	}

	// Verify it was added
	criteria, err := session.GetProjectAcceptanceCriteria(projectDir)
	if err != nil {
		t.Fatalf("failed to get criteria: %v", err)
	}

	if len(criteria) != 1 {
		t.Errorf("expected 1 criterion, got %d", len(criteria))
	}
	if criteria[0] != "Tests pass" {
		t.Errorf("expected 'Tests pass', got '%s'", criteria[0])
	}
}

// TestConfigACAdd_Multiple tests adding multiple criteria
func TestConfigACAdd_Multiple(t *testing.T) {
	projectDir, cleanup := setupTestProject(t)
	defer cleanup()

	// Add multiple criteria
	runConfigACAdd(configACAddCmd, []string{"Tests pass"})
	runConfigACAdd(configACAddCmd, []string{"Build succeeds"})
	runConfigACAdd(configACAddCmd, []string{"Documentation updated"})

	// Verify all were added
	criteria, err := session.GetProjectAcceptanceCriteria(projectDir)
	if err != nil {
		t.Fatalf("failed to get criteria: %v", err)
	}

	if len(criteria) != 3 {
		t.Errorf("expected 3 criteria, got %d", len(criteria))
	}
}

// TestConfigACList_WithCriteria tests listing when criteria exist
func TestConfigACList_WithCriteria(t *testing.T) {
	projectDir, cleanup := setupTestProject(t)
	defer cleanup()

	// Add some criteria
	criteria := []string{"Tests pass", "Build succeeds"}
	if err := session.UpdateProjectAcceptanceCriteria(projectDir, criteria); err != nil {
		t.Fatalf("failed to set criteria: %v", err)
	}

	// Run list command - should not error
	err := runConfigACList(configACListCmd, []string{})
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

// TestSessionInheritRepoACs tests that new sessions inherit repo-level ACs
func TestSessionInheritRepoACs(t *testing.T) {
	projectDir, cleanup := setupTestProject(t)
	defer cleanup()

	// Set repo-level ACs
	repoACs := []string{"Tests pass", "Build succeeds"}
	if err := session.UpdateProjectAcceptanceCriteria(projectDir, repoACs); err != nil {
		t.Fatalf("failed to set repo ACs: %v", err)
	}

	// Clear session flag so inheritance works
	sessionACFlag = []string{}

	// Create a session via the CLI
	err := runSessionsCreate(sessionsCreateCmd, []string{"test-session"})
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Load the session and check ACs were inherited
	store, err := session.NewSessionStore(projectDir)
	if err != nil {
		t.Fatalf("failed to create session store: %v", err)
	}

	sess, err := store.LoadSession("test-session")
	if err != nil {
		t.Fatalf("failed to load session: %v", err)
	}

	if len(sess.AcceptanceCriteria) != 2 {
		t.Errorf("expected 2 inherited ACs, got %d", len(sess.AcceptanceCriteria))
	}
}

// TestSessionExplicitACs tests that explicit ACs override repo defaults
func TestSessionExplicitACs(t *testing.T) {
	projectDir, cleanup := setupTestProject(t)
	defer cleanup()

	// Set repo-level ACs
	repoACs := []string{"Tests pass", "Build succeeds"}
	if err := session.UpdateProjectAcceptanceCriteria(projectDir, repoACs); err != nil {
		t.Fatalf("failed to set repo ACs: %v", err)
	}

	// Set explicit session ACs via flag
	sessionACFlag = []string{"Custom criterion"}

	// Create a session via the CLI
	err := runSessionsCreate(sessionsCreateCmd, []string{"custom-session"})
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Load the session and check explicit ACs were used
	store, err := session.NewSessionStore(projectDir)
	if err != nil {
		t.Fatalf("failed to create session store: %v", err)
	}

	sess, err := store.LoadSession("custom-session")
	if err != nil {
		t.Fatalf("failed to load session: %v", err)
	}

	if len(sess.AcceptanceCriteria) != 1 {
		t.Errorf("expected 1 explicit AC, got %d", len(sess.AcceptanceCriteria))
	}
	if sess.AcceptanceCriteria[0] != "Custom criterion" {
		t.Errorf("expected 'Custom criterion', got '%s'", sess.AcceptanceCriteria[0])
	}

	// Reset flag
	sessionACFlag = []string{}
}

// TestSessionEditDescription tests updating session description via CLI
func TestSessionEditDescription(t *testing.T) {
	projectDir, cleanup := setupTestProject(t)
	defer cleanup()

	// Create a session first
	sessionACFlag = []string{}
	sessionDescriptionFlag = "Initial description"
	err := runSessionsCreate(sessionsCreateCmd, []string{"test-edit-session"})
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}
	sessionDescriptionFlag = ""

	// Edit the description
	sessionEditDescriptionFlag = "Updated description"
	err = runSessionsEdit(sessionsEditCmd, []string{"test-edit-session"})
	if err != nil {
		t.Fatalf("failed to edit session: %v", err)
	}
	sessionEditDescriptionFlag = ""

	// Verify the update
	store, err := session.NewSessionStore(projectDir)
	if err != nil {
		t.Fatalf("failed to create session store: %v", err)
	}

	sess, err := store.LoadSession("test-edit-session")
	if err != nil {
		t.Fatalf("failed to load session: %v", err)
	}

	if sess.Description != "Updated description" {
		t.Errorf("expected 'Updated description', got '%s'", sess.Description)
	}
}

// TestSessionEditReplaceACs tests replacing acceptance criteria via CLI
func TestSessionEditReplaceACs(t *testing.T) {
	projectDir, cleanup := setupTestProject(t)
	defer cleanup()

	// Create a session with initial ACs
	sessionACFlag = []string{"AC1", "AC2"}
	err := runSessionsCreate(sessionsCreateCmd, []string{"test-ac-session"})
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}
	sessionACFlag = []string{}

	// Replace ACs via edit
	sessionEditACFlag = []string{"New AC1", "New AC2", "New AC3"}
	err = runSessionsEdit(sessionsEditCmd, []string{"test-ac-session"})
	if err != nil {
		t.Fatalf("failed to edit session: %v", err)
	}
	sessionEditACFlag = []string{}

	// Verify the update
	store, err := session.NewSessionStore(projectDir)
	if err != nil {
		t.Fatalf("failed to create session store: %v", err)
	}

	sess, err := store.LoadSession("test-ac-session")
	if err != nil {
		t.Fatalf("failed to load session: %v", err)
	}

	if len(sess.AcceptanceCriteria) != 3 {
		t.Errorf("expected 3 ACs, got %d", len(sess.AcceptanceCriteria))
	}
	if sess.AcceptanceCriteria[0] != "New AC1" {
		t.Errorf("expected 'New AC1', got '%s'", sess.AcceptanceCriteria[0])
	}
}

// TestSessionEditAppendACs tests appending acceptance criteria via CLI
func TestSessionEditAppendACs(t *testing.T) {
	projectDir, cleanup := setupTestProject(t)
	defer cleanup()

	// Create a session with initial ACs
	sessionACFlag = []string{"AC1"}
	err := runSessionsCreate(sessionsCreateCmd, []string{"test-append-ac"})
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}
	sessionACFlag = []string{}

	// Append ACs
	sessionEditACAppendFlag = []string{"AC2", "AC3"}
	err = runSessionsEdit(sessionsEditCmd, []string{"test-append-ac"})
	if err != nil {
		t.Fatalf("failed to edit session: %v", err)
	}
	sessionEditACAppendFlag = []string{}

	// Verify the update
	store, err := session.NewSessionStore(projectDir)
	if err != nil {
		t.Fatalf("failed to create session store: %v", err)
	}

	sess, err := store.LoadSession("test-append-ac")
	if err != nil {
		t.Fatalf("failed to load session: %v", err)
	}

	if len(sess.AcceptanceCriteria) != 3 {
		t.Errorf("expected 3 ACs, got %d", len(sess.AcceptanceCriteria))
	}
}

// TestSessionEditDefaultModel tests updating default model via CLI
func TestSessionEditDefaultModel(t *testing.T) {
	projectDir, cleanup := setupTestProject(t)
	defer cleanup()

	// Create a session
	sessionACFlag = []string{}
	err := runSessionsCreate(sessionsCreateCmd, []string{"test-model-session"})
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Set default model
	sessionEditDefaultModelFlag = "medium"
	err = runSessionsEdit(sessionsEditCmd, []string{"test-model-session"})
	if err != nil {
		t.Fatalf("failed to edit session: %v", err)
	}
	sessionEditDefaultModelFlag = ""

	// Verify the update
	store, err := session.NewSessionStore(projectDir)
	if err != nil {
		t.Fatalf("failed to create session store: %v", err)
	}

	sess, err := store.LoadSession("test-model-session")
	if err != nil {
		t.Fatalf("failed to load session: %v", err)
	}

	if sess.DefaultModel != session.ModelSizeMedium {
		t.Errorf("expected 'medium', got '%s'", sess.DefaultModel)
	}
}

// TestSessionEditContext tests updating session context via CLI
func TestSessionEditContext(t *testing.T) {
	projectDir, cleanup := setupTestProject(t)
	defer cleanup()

	// Create a session
	sessionACFlag = []string{}
	err := runSessionsCreate(sessionsCreateCmd, []string{"test-context-session"})
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Set context
	sessionEditContextSetFlag = "This is the new context"
	err = runSessionsEdit(sessionsEditCmd, []string{"test-context-session"})
	if err != nil {
		t.Fatalf("failed to edit session: %v", err)
	}
	sessionEditContextSetFlag = ""

	// Verify the update
	store, err := session.NewSessionStore(projectDir)
	if err != nil {
		t.Fatalf("failed to create session store: %v", err)
	}

	sess, err := store.LoadSession("test-context-session")
	if err != nil {
		t.Fatalf("failed to load session: %v", err)
	}

	if sess.Context != "This is the new context" {
		t.Errorf("expected 'This is the new context', got '%s'", sess.Context)
	}
}

// TestParseEditedSession tests the session file parsing
func TestParseEditedSession(t *testing.T) {
	content := `# Session: test
# Edit the values below and save.

description: My test session

default_model: large

acceptance_criteria:
  - AC One
  - AC Two
  - AC Three

context: |
  This is line one
  This is line two
`

	desc, model, acs, ctx, err := parseEditedSession(content)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if desc != "My test session" {
		t.Errorf("expected description 'My test session', got '%s'", desc)
	}

	if model != session.ModelSizeLarge {
		t.Errorf("expected model 'large', got '%s'", model)
	}

	if len(acs) != 3 {
		t.Errorf("expected 3 ACs, got %d", len(acs))
	}

	if ctx != "This is line one\nThis is line two" {
		t.Errorf("expected multi-line context, got '%s'", ctx)
	}
}
