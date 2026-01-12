package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ohare93/juggle/internal/session"
)

// TestExportRalph_WithGlobalACs tests that Ralph export includes global ACs
func TestExportRalph_WithGlobalACs(t *testing.T) {
	tmpDir := t.TempDir()
	jugglerDir := filepath.Join(tmpDir, ".juggler")
	if err := os.MkdirAll(jugglerDir, 0755); err != nil {
		t.Fatalf("failed to create .juggler dir: %v", err)
	}

	// Set repo-level ACs
	repoACs := []string{"Tests pass", "Build succeeds"}
	if err := session.UpdateProjectAcceptanceCriteria(tmpDir, repoACs); err != nil {
		t.Fatalf("failed to set repo ACs: %v", err)
	}

	// Create a session with session-level ACs
	store, err := session.NewSessionStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create session store: %v", err)
	}

	_, err = store.CreateSession("test-session", "Test session")
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	sessionACs := []string{"Documentation updated"}
	if err := store.UpdateSessionAcceptanceCriteria("test-session", sessionACs); err != nil {
		t.Fatalf("failed to set session ACs: %v", err)
	}

	// Create a ball
	ballStore, err := session.NewStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create ball store: %v", err)
	}

	ball, _ := session.NewBall(tmpDir, "Test ball", session.PriorityMedium)
	ball.AddTag("test-session")
	if err := ballStore.AppendBall(ball); err != nil {
		t.Fatalf("failed to save ball: %v", err)
	}

	// Export in Ralph format
	balls := []*session.Ball{ball}
	output, err := exportRalph(tmpDir, "test-session", balls)
	if err != nil {
		t.Fatalf("failed to export Ralph: %v", err)
	}

	outputStr := string(output)

	// Verify global-acceptance-criteria section exists
	if !strings.Contains(outputStr, "<global-acceptance-criteria>") {
		t.Error("expected output to contain <global-acceptance-criteria> section")
	}

	// Verify repo-level ACs are included
	if !strings.Contains(outputStr, "Tests pass") {
		t.Error("expected output to contain repo AC 'Tests pass'")
	}
	if !strings.Contains(outputStr, "Build succeeds") {
		t.Error("expected output to contain repo AC 'Build succeeds'")
	}

	// Verify session-level ACs are included
	if !strings.Contains(outputStr, "Documentation updated") {
		t.Error("expected output to contain session AC 'Documentation updated'")
	}

	// Verify section headers
	if !strings.Contains(outputStr, "Repository-Level Requirements") {
		t.Error("expected output to contain 'Repository-Level Requirements' header")
	}
	if !strings.Contains(outputStr, "Session-Level Requirements") {
		t.Error("expected output to contain 'Session-Level Requirements' header")
	}
}

// TestExportAgent_WithGlobalACs tests that Agent export includes global ACs
func TestExportAgent_WithGlobalACs(t *testing.T) {
	tmpDir := t.TempDir()
	jugglerDir := filepath.Join(tmpDir, ".juggler")
	if err := os.MkdirAll(jugglerDir, 0755); err != nil {
		t.Fatalf("failed to create .juggler dir: %v", err)
	}

	// Set repo-level ACs
	repoACs := []string{"All tests must pass"}
	if err := session.UpdateProjectAcceptanceCriteria(tmpDir, repoACs); err != nil {
		t.Fatalf("failed to set repo ACs: %v", err)
	}

	// Create a session
	store, err := session.NewSessionStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create session store: %v", err)
	}

	_, err = store.CreateSession("agent-session", "Agent session")
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Create a ball
	ballStore, err := session.NewStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create ball store: %v", err)
	}

	ball, _ := session.NewBall(tmpDir, "Agent test ball", session.PriorityMedium)
	ball.AddTag("agent-session")
	if err := ballStore.AppendBall(ball); err != nil {
		t.Fatalf("failed to save ball: %v", err)
	}

	// Export in Agent format
	balls := []*session.Ball{ball}
	output, err := exportAgent(tmpDir, "agent-session", balls, false, false)
	if err != nil {
		t.Fatalf("failed to export Agent: %v", err)
	}

	outputStr := string(output)

	// Verify global-acceptance-criteria section exists
	if !strings.Contains(outputStr, "<global-acceptance-criteria>") {
		t.Error("expected output to contain <global-acceptance-criteria> section")
	}

	// Verify repo-level AC is included
	if !strings.Contains(outputStr, "All tests must pass") {
		t.Error("expected output to contain repo AC 'All tests must pass'")
	}
}

// TestExportRalph_NoGlobalACs tests that export works without global ACs
func TestExportRalph_NoGlobalACs(t *testing.T) {
	tmpDir := t.TempDir()
	jugglerDir := filepath.Join(tmpDir, ".juggler")
	if err := os.MkdirAll(jugglerDir, 0755); err != nil {
		t.Fatalf("failed to create .juggler dir: %v", err)
	}

	// Create a session (no ACs at any level)
	store, err := session.NewSessionStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create session store: %v", err)
	}

	_, err = store.CreateSession("empty-session", "Empty session")
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Create a ball
	ballStore, err := session.NewStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create ball store: %v", err)
	}

	ball, _ := session.NewBall(tmpDir, "Test ball", session.PriorityMedium)
	ball.AddTag("empty-session")
	if err := ballStore.AppendBall(ball); err != nil {
		t.Fatalf("failed to save ball: %v", err)
	}

	// Export in Ralph format
	balls := []*session.Ball{ball}
	output, err := exportRalph(tmpDir, "empty-session", balls)
	if err != nil {
		t.Fatalf("failed to export Ralph: %v", err)
	}

	outputStr := string(output)

	// Verify global-acceptance-criteria section is NOT present when no ACs exist
	if strings.Contains(outputStr, "<global-acceptance-criteria>") {
		t.Error("expected output to NOT contain <global-acceptance-criteria> when no ACs exist")
	}
}

// TestExportRalph_OnlyRepoACs tests export with only repo-level ACs
func TestExportRalph_OnlyRepoACs(t *testing.T) {
	tmpDir := t.TempDir()
	jugglerDir := filepath.Join(tmpDir, ".juggler")
	if err := os.MkdirAll(jugglerDir, 0755); err != nil {
		t.Fatalf("failed to create .juggler dir: %v", err)
	}

	// Set repo-level ACs only
	repoACs := []string{"Tests pass"}
	if err := session.UpdateProjectAcceptanceCriteria(tmpDir, repoACs); err != nil {
		t.Fatalf("failed to set repo ACs: %v", err)
	}

	// Create a session (no session ACs)
	store, err := session.NewSessionStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create session store: %v", err)
	}

	_, err = store.CreateSession("repo-only", "Repo only session")
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Create a ball
	ballStore, err := session.NewStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create ball store: %v", err)
	}

	ball, _ := session.NewBall(tmpDir, "Test ball", session.PriorityMedium)
	ball.AddTag("repo-only")
	if err := ballStore.AppendBall(ball); err != nil {
		t.Fatalf("failed to save ball: %v", err)
	}

	// Export in Ralph format
	balls := []*session.Ball{ball}
	output, err := exportRalph(tmpDir, "repo-only", balls)
	if err != nil {
		t.Fatalf("failed to export Ralph: %v", err)
	}

	outputStr := string(output)

	// Should have global-acceptance-criteria section
	if !strings.Contains(outputStr, "<global-acceptance-criteria>") {
		t.Error("expected output to contain <global-acceptance-criteria> section")
	}

	// Should have Repository-Level Requirements
	if !strings.Contains(outputStr, "Repository-Level Requirements") {
		t.Error("expected output to contain 'Repository-Level Requirements' header")
	}

	// Should NOT have Session-Level Requirements (no session ACs)
	if strings.Contains(outputStr, "Session-Level Requirements") {
		t.Error("expected output to NOT contain 'Session-Level Requirements' when no session ACs")
	}
}

// TestExportRalph_OnlySessionACs tests export with only session-level ACs
func TestExportRalph_OnlySessionACs(t *testing.T) {
	tmpDir := t.TempDir()
	jugglerDir := filepath.Join(tmpDir, ".juggler")
	if err := os.MkdirAll(jugglerDir, 0755); err != nil {
		t.Fatalf("failed to create .juggler dir: %v", err)
	}

	// Create a session with ACs but no repo ACs
	store, err := session.NewSessionStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create session store: %v", err)
	}

	_, err = store.CreateSession("session-only", "Session only")
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	sessionACs := []string{"Session criterion"}
	if err := store.UpdateSessionAcceptanceCriteria("session-only", sessionACs); err != nil {
		t.Fatalf("failed to set session ACs: %v", err)
	}

	// Create a ball
	ballStore, err := session.NewStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create ball store: %v", err)
	}

	ball, _ := session.NewBall(tmpDir, "Test ball", session.PriorityMedium)
	ball.AddTag("session-only")
	if err := ballStore.AppendBall(ball); err != nil {
		t.Fatalf("failed to save ball: %v", err)
	}

	// Export in Ralph format
	balls := []*session.Ball{ball}
	output, err := exportRalph(tmpDir, "session-only", balls)
	if err != nil {
		t.Fatalf("failed to export Ralph: %v", err)
	}

	outputStr := string(output)

	// Should have global-acceptance-criteria section
	if !strings.Contains(outputStr, "<global-acceptance-criteria>") {
		t.Error("expected output to contain <global-acceptance-criteria> section")
	}

	// Should NOT have Repository-Level Requirements (no repo ACs)
	if strings.Contains(outputStr, "Repository-Level Requirements") {
		t.Error("expected output to NOT contain 'Repository-Level Requirements' when no repo ACs")
	}

	// Should have Session-Level Requirements
	if !strings.Contains(outputStr, "Session-Level Requirements") {
		t.Error("expected output to contain 'Session-Level Requirements' header")
	}
}
