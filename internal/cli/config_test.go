package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ohare93/juggle/internal/session"
)

// setupTestProject creates a temp directory with a .juggle directory
func setupTestProject(t *testing.T) (string, func()) {
	tmpDir := t.TempDir()
	juggleDir := filepath.Join(tmpDir, ".juggle")
	if err := os.MkdirAll(juggleDir, 0755); err != nil {
		t.Fatalf("failed to create .juggle dir: %v", err)
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

// Test delay configuration

// TestConfigDelayShow_Empty tests showing delay when not configured
func TestConfigDelayShow_Empty(t *testing.T) {
	_, cleanup := setupTestProject(t)
	defer cleanup()

	// Run show command - should not error
	err := runConfigDelayShow(configDelayShowCmd, []string{})
	if err != nil {
		t.Errorf("expected no error with no delay configured, got: %v", err)
	}
}

// TestConfigDelaySet tests setting the delay
func TestConfigDelaySet(t *testing.T) {
	tmpDir, cleanup := setupTestProject(t)
	defer cleanup()

	// Set delay
	configDelayFuzz = 0 // No fuzz
	err := runConfigDelaySet(configDelaySetCmd, []string{"5"})
	if err != nil {
		t.Fatalf("failed to set delay: %v", err)
	}

	// Verify via session package
	opts := session.ConfigOptions{
		ConfigHome:    tmpDir,
		JuggleDirName: ".juggle",
	}
	delayMinutes, fuzz, err := session.GetGlobalIterationDelayWithOptions(opts)
	if err != nil {
		t.Fatalf("failed to get delay: %v", err)
	}

	if delayMinutes != 5 {
		t.Errorf("expected delay 5 minutes, got %d", delayMinutes)
	}
	if fuzz != 0 {
		t.Errorf("expected fuzz 0, got %d", fuzz)
	}
}

// TestConfigDelaySet_WithFuzz tests setting delay with fuzz
func TestConfigDelaySet_WithFuzz(t *testing.T) {
	tmpDir, cleanup := setupTestProject(t)
	defer cleanup()

	// Set delay with fuzz
	configDelayFuzz = 2
	err := runConfigDelaySet(configDelaySetCmd, []string{"10"})
	if err != nil {
		t.Fatalf("failed to set delay: %v", err)
	}

	// Verify via session package
	opts := session.ConfigOptions{
		ConfigHome:    tmpDir,
		JuggleDirName: ".juggle",
	}
	delayMinutes, fuzz, err := session.GetGlobalIterationDelayWithOptions(opts)
	if err != nil {
		t.Fatalf("failed to get delay: %v", err)
	}

	if delayMinutes != 10 {
		t.Errorf("expected delay 10 minutes, got %d", delayMinutes)
	}
	if fuzz != 2 {
		t.Errorf("expected fuzz 2, got %d", fuzz)
	}

	// Reset global variable
	configDelayFuzz = 0
}

// TestConfigDelayClear tests clearing the delay
func TestConfigDelayClear(t *testing.T) {
	tmpDir, cleanup := setupTestProject(t)
	defer cleanup()

	// First set a delay
	opts := session.ConfigOptions{
		ConfigHome:    tmpDir,
		JuggleDirName: ".juggle",
	}
	if err := session.UpdateGlobalIterationDelayWithOptions(opts, 5, 1); err != nil {
		t.Fatalf("failed to set initial delay: %v", err)
	}

	// Clear it
	err := runConfigDelayClear(configDelayClearCmd, []string{})
	if err != nil {
		t.Fatalf("failed to clear delay: %v", err)
	}

	// Verify cleared
	delayMinutes, fuzz, err := session.GetGlobalIterationDelayWithOptions(opts)
	if err != nil {
		t.Fatalf("failed to get delay: %v", err)
	}

	if delayMinutes != 0 {
		t.Errorf("expected delay 0 after clear, got %d", delayMinutes)
	}
	if fuzz != 0 {
		t.Errorf("expected fuzz 0 after clear, got %d", fuzz)
	}
}

// TestConfigDelaySet_InvalidInput tests invalid delay input
func TestConfigDelaySet_InvalidInput(t *testing.T) {
	_, cleanup := setupTestProject(t)
	defer cleanup()

	// Test invalid input
	err := runConfigDelaySet(configDelaySetCmd, []string{"abc"})
	if err == nil {
		t.Error("expected error for invalid input, got nil")
	}

	// Test negative input
	err = runConfigDelaySet(configDelaySetCmd, []string{"-5"})
	if err == nil {
		t.Error("expected error for negative input, got nil")
	}
}

// TestConfigDelayShow_WithValues tests showing delay when configured
func TestConfigDelayShow_WithValues(t *testing.T) {
	tmpDir, cleanup := setupTestProject(t)
	defer cleanup()

	// Set a delay
	opts := session.ConfigOptions{
		ConfigHome:    tmpDir,
		JuggleDirName: ".juggle",
	}
	if err := session.UpdateGlobalIterationDelayWithOptions(opts, 5, 2); err != nil {
		t.Fatalf("failed to set delay: %v", err)
	}

	// Run show command - should not error
	err := runConfigDelayShow(configDelayShowCmd, []string{})
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

// TestCalculateFuzzyDelay_NoFuzz tests delay calculation without fuzz
func TestCalculateFuzzyDelay_NoFuzz(t *testing.T) {
	delay := CalculateFuzzyDelayForTest(5, 0)

	// With no fuzz, should always be exactly 5 minutes
	expected := 5 * time.Minute
	if delay != expected {
		t.Errorf("expected %v, got %v", expected, delay)
	}
}

// TestCalculateFuzzyDelay_WithFuzz tests delay calculation with fuzz
func TestCalculateFuzzyDelay_WithFuzz(t *testing.T) {
	baseMinutes := 5
	fuzz := 2

	// Run multiple times to verify range
	minExpected := time.Duration(baseMinutes-fuzz) * time.Minute // 3 minutes
	maxExpected := time.Duration(baseMinutes+fuzz) * time.Minute // 7 minutes

	for i := 0; i < 100; i++ {
		delay := CalculateFuzzyDelayForTest(baseMinutes, fuzz)
		if delay < minExpected || delay > maxExpected {
			t.Errorf("delay %v out of expected range [%v, %v]", delay, minExpected, maxExpected)
		}
	}
}

// TestCalculateFuzzyDelay_NonNegative tests that delay never goes negative
func TestCalculateFuzzyDelay_NonNegative(t *testing.T) {
	// Test with fuzz larger than base - should never go negative
	baseMinutes := 2
	fuzz := 5

	for i := 0; i < 100; i++ {
		delay := CalculateFuzzyDelayForTest(baseMinutes, fuzz)
		if delay < 0 {
			t.Errorf("delay should never be negative, got %v", delay)
		}
	}
}

// TestCalculateFuzzyDelay_Zero tests delay calculation with zero base
func TestCalculateFuzzyDelay_Zero(t *testing.T) {
	delay := CalculateFuzzyDelayForTest(0, 0)
	if delay != 0 {
		t.Errorf("expected 0 delay for 0 base, got %v", delay)
	}
}

// Test provider configuration

// TestConfigProviderShow_Empty tests showing provider when no provider configured
func TestConfigProviderShow_Empty(t *testing.T) {
	_, cleanup := setupTestProject(t)
	defer cleanup()

	// Run show command - should not error
	err := runConfigProviderShow(configProviderShowCmd, []string{})
	if err != nil {
		t.Errorf("expected no error with no provider configured, got: %v", err)
	}
}

// TestConfigProviderShow_GlobalSet tests showing global provider
func TestConfigProviderShow_GlobalSet(t *testing.T) {
	tmpDir, cleanup := setupTestProject(t)
	defer cleanup()

	// Set global provider
	opts := session.ConfigOptions{
		ConfigHome:    tmpDir,
		JuggleDirName: ".juggle",
	}
	if err := session.UpdateGlobalAgentProviderWithOptions(opts, "claude"); err != nil {
		t.Fatalf("failed to set global provider: %v", err)
	}

	// Run show command - should not error
	err := runConfigProviderShow(configProviderShowCmd, []string{})
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

// TestConfigProviderShow_ProjectSet tests showing project provider
func TestConfigProviderShow_ProjectSet(t *testing.T) {
	tmpDir, cleanup := setupTestProject(t)
	defer cleanup()

	// Set project provider
	if err := session.UpdateProjectAgentProvider(tmpDir, "opencode"); err != nil {
		t.Fatalf("failed to set project provider: %v", err)
	}

	// Run show command - should not error
	err := runConfigProviderShow(configProviderShowCmd, []string{})
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

// TestConfigProviderShow_BothSet tests effective resolution (project > global)
func TestConfigProviderShow_BothSet(t *testing.T) {
	// Need separate directories for global and project configs to test resolution
	globalDir := t.TempDir()
	projectDir := t.TempDir()

	// Create .juggle directories
	if err := os.MkdirAll(filepath.Join(globalDir, ".juggle"), 0755); err != nil {
		t.Fatalf("failed to create global .juggle dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectDir, ".juggle"), 0755); err != nil {
		t.Fatalf("failed to create project .juggle dir: %v", err)
	}

	// Save and restore global options
	origConfigHome := GlobalOpts.ConfigHome
	origProjectDir := GlobalOpts.ProjectDir
	GlobalOpts.ConfigHome = globalDir
	GlobalOpts.ProjectDir = projectDir
	defer func() {
		GlobalOpts.ConfigHome = origConfigHome
		GlobalOpts.ProjectDir = origProjectDir
	}()

	// Set global provider (using globalDir as config home)
	globalOpts := session.ConfigOptions{
		ConfigHome:    globalDir,
		JuggleDirName: ".juggle",
	}
	if err := session.UpdateGlobalAgentProviderWithOptions(globalOpts, "claude"); err != nil {
		t.Fatalf("failed to set global provider: %v", err)
	}

	// Set project provider (using separate projectDir)
	if err := session.UpdateProjectAgentProvider(projectDir, "opencode"); err != nil {
		t.Fatalf("failed to set project provider: %v", err)
	}

	// Verify providers are set correctly
	projectProvider, err := session.GetProjectAgentProvider(projectDir)
	if err != nil {
		t.Fatalf("failed to get project provider: %v", err)
	}

	globalProvider, err := session.GetGlobalAgentProviderWithOptions(globalOpts)
	if err != nil {
		t.Fatalf("failed to get global provider: %v", err)
	}

	// Should have different values now
	if projectProvider != "opencode" {
		t.Errorf("expected project provider 'opencode', got '%s'", projectProvider)
	}
	if globalProvider != "claude" {
		t.Errorf("expected global provider 'claude', got '%s'", globalProvider)
	}

	// Run show command - should not error
	err = runConfigProviderShow(configProviderShowCmd, []string{})
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

// TestConfigProviderSet_Global tests setting global provider
func TestConfigProviderSet_Global(t *testing.T) {
	tmpDir, cleanup := setupTestProject(t)
	defer cleanup()

	// Set global provider
	configProviderProjectFlag = false
	err := runConfigProviderSet(configProviderSetCmd, []string{"claude"})
	if err != nil {
		t.Fatalf("failed to set global provider: %v", err)
	}

	// Verify it was set
	opts := session.ConfigOptions{
		ConfigHome:    tmpDir,
		JuggleDirName: ".juggle",
	}
	provider, err := session.GetGlobalAgentProviderWithOptions(opts)
	if err != nil {
		t.Fatalf("failed to get global provider: %v", err)
	}

	if provider != "claude" {
		t.Errorf("expected provider 'claude', got '%s'", provider)
	}
}

// TestConfigProviderSet_Project tests setting project provider
func TestConfigProviderSet_Project(t *testing.T) {
	tmpDir, cleanup := setupTestProject(t)
	defer cleanup()

	// Set project provider with flag
	configProviderProjectFlag = true
	err := runConfigProviderSet(configProviderSetCmd, []string{"opencode"})
	if err != nil {
		t.Fatalf("failed to set project provider: %v", err)
	}
	configProviderProjectFlag = false

	// Verify it was set
	provider, err := session.GetProjectAgentProvider(tmpDir)
	if err != nil {
		t.Fatalf("failed to get project provider: %v", err)
	}

	if provider != "opencode" {
		t.Errorf("expected provider 'opencode', got '%s'", provider)
	}
}

// TestConfigProviderSet_Invalid tests invalid provider rejection
func TestConfigProviderSet_Invalid(t *testing.T) {
	_, cleanup := setupTestProject(t)
	defer cleanup()

	// Try to set invalid provider
	configProviderProjectFlag = false
	err := runConfigProviderSet(configProviderSetCmd, []string{"invalid-provider"})
	if err == nil {
		t.Error("expected error for invalid provider, got nil")
	}

	// Check error message mentions the provider
	if !strings.Contains(err.Error(), "invalid provider") {
		t.Errorf("expected 'invalid provider' error message, got: %v", err)
	}
}

// TestConfigProviderClear_Global tests clearing global setting
func TestConfigProviderClear_Global(t *testing.T) {
	tmpDir, cleanup := setupTestProject(t)
	defer cleanup()

	// First set a global provider
	opts := session.ConfigOptions{
		ConfigHome:    tmpDir,
		JuggleDirName: ".juggle",
	}
	if err := session.UpdateGlobalAgentProviderWithOptions(opts, "claude"); err != nil {
		t.Fatalf("failed to set global provider: %v", err)
	}

	// Clear it
	configProviderProjectFlag = false
	err := runConfigProviderClear(configProviderClearCmd, []string{})
	if err != nil {
		t.Fatalf("failed to clear global provider: %v", err)
	}

	// Verify it was cleared
	provider, err := session.GetGlobalAgentProviderWithOptions(opts)
	if err != nil {
		t.Fatalf("failed to get global provider: %v", err)
	}

	if provider != "" {
		t.Errorf("expected empty provider after clear, got '%s'", provider)
	}
}

// TestConfigProviderClear_Project tests clearing project setting
func TestConfigProviderClear_Project(t *testing.T) {
	tmpDir, cleanup := setupTestProject(t)
	defer cleanup()

	// First set a project provider
	if err := session.UpdateProjectAgentProvider(tmpDir, "opencode"); err != nil {
		t.Fatalf("failed to set project provider: %v", err)
	}

	// Clear it with flag
	configProviderProjectFlag = true
	err := runConfigProviderClear(configProviderClearCmd, []string{})
	if err != nil {
		t.Fatalf("failed to clear project provider: %v", err)
	}
	configProviderProjectFlag = false

	// Verify it was cleared
	provider, err := session.GetProjectAgentProvider(tmpDir)
	if err != nil {
		t.Fatalf("failed to get project provider: %v", err)
	}

	if provider != "" {
		t.Errorf("expected empty provider after clear, got '%s'", provider)
	}
}
