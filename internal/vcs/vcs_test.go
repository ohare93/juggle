package vcs

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestVCSType_IsValid(t *testing.T) {
	tests := []struct {
		vcsType VCSType
		want    bool
	}{
		{VCSTypeJJ, true},
		{VCSTypeGit, true},
		{"svn", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.vcsType), func(t *testing.T) {
			if got := tt.vcsType.IsValid(); got != tt.want {
				t.Errorf("VCSType.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAutoDetect_JJ(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, ".jj"), 0755); err != nil {
		t.Fatal(err)
	}

	result := AutoDetect(tmpDir)
	if result != VCSTypeJJ {
		t.Errorf("expected jj, got %s", result)
	}
}

func TestAutoDetect_Git(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}

	result := AutoDetect(tmpDir)
	if result != VCSTypeGit {
		t.Errorf("expected git, got %s", result)
	}
}

func TestAutoDetect_JJPriority(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, ".jj"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}

	result := AutoDetect(tmpDir)
	if result != VCSTypeJJ {
		t.Errorf("expected jj (priority), got %s", result)
	}
}

func TestAutoDetect_DefaultToGit(t *testing.T) {
	tmpDir := t.TempDir()

	result := AutoDetect(tmpDir)
	if result != VCSTypeGit {
		t.Errorf("expected git (default), got %s", result)
	}
}

func TestDetect_ProjectConfigPriority(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, ".jj"), 0755); err != nil {
		t.Fatal(err)
	}

	// Project config overrides auto-detection
	result := Detect(tmpDir, VCSTypeGit, "")
	if result != VCSTypeGit {
		t.Errorf("expected git (project config), got %s", result)
	}
}

func TestDetect_GlobalConfigUsed(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, ".jj"), 0755); err != nil {
		t.Fatal(err)
	}

	// Global config is used when project config is empty
	result := Detect(tmpDir, "", VCSTypeGit)
	if result != VCSTypeGit {
		t.Errorf("expected git (global config), got %s", result)
	}
}

func TestDetect_ProjectOverridesGlobal(t *testing.T) {
	tmpDir := t.TempDir()

	// Project config overrides global config
	result := Detect(tmpDir, VCSTypeJJ, VCSTypeGit)
	if result != VCSTypeJJ {
		t.Errorf("expected jj (project overrides global), got %s", result)
	}
}

func TestDetect_AutoDetectWhenNoConfig(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}

	// No config set, should auto-detect
	result := Detect(tmpDir, "", "")
	if result != VCSTypeGit {
		t.Errorf("expected git (auto-detected), got %s", result)
	}
}

func TestGetBackend(t *testing.T) {
	tests := []struct {
		vcsType VCSType
		want    VCSType
	}{
		{VCSTypeJJ, VCSTypeJJ},
		{VCSTypeGit, VCSTypeGit},
		{"unknown", VCSTypeGit}, // defaults to git
		{"", VCSTypeGit},        // defaults to git
	}

	for _, tt := range tests {
		t.Run(string(tt.vcsType), func(t *testing.T) {
			backend := GetBackend(tt.vcsType)
			if backend.Type() != tt.want {
				t.Errorf("GetBackend(%s).Type() = %s, want %s", tt.vcsType, backend.Type(), tt.want)
			}
		})
	}
}

// =============================================================================
// Git Backend Tests
// =============================================================================

// setupGitRepo creates a git repo in the given directory with initial commit
func setupGitRepo(t *testing.T, dir string) {
	t.Helper()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %s: %v", output, err)
	}

	// Configure user for commits
	cmd = exec.Command("git", "config", "user.email", "test@test.com")
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git config email failed: %s: %v", output, err)
	}

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git config name failed: %s: %v", output, err)
	}

	// Create initial file and commit
	testFile := filepath.Join(dir, "README.md")
	if err := os.WriteFile(testFile, []byte("# Test\n"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git add failed: %s: %v", output, err)
	}

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit failed: %s: %v", output, err)
	}

	// Rename default branch to main (for consistency)
	cmd = exec.Command("git", "branch", "-M", "main")
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git branch -M main failed: %s: %v", output, err)
	}
}

func TestGitBackend_Type(t *testing.T) {
	backend := NewGitBackend()
	if backend.Type() != VCSTypeGit {
		t.Errorf("expected git, got %s", backend.Type())
	}
}

func TestGitBackend_DescribeWorkingCopy_NoOp(t *testing.T) {
	tmpDir := t.TempDir()
	setupGitRepo(t, tmpDir)

	backend := NewGitBackend()

	// DescribeWorkingCopy should be a no-op for git
	err := backend.DescribeWorkingCopy(tmpDir, "test message")
	if err != nil {
		t.Errorf("DescribeWorkingCopy should be no-op, got error: %v", err)
	}
}

func TestGitBackend_GetCurrentRevision_Branch(t *testing.T) {
	tmpDir := t.TempDir()
	setupGitRepo(t, tmpDir)

	backend := NewGitBackend()

	rev, err := backend.GetCurrentRevision(tmpDir)
	if err != nil {
		t.Fatalf("GetCurrentRevision failed: %v", err)
	}

	if rev != "main" {
		t.Errorf("expected 'main', got %q", rev)
	}
}

func TestGitBackend_GetCurrentRevision_DetachedHead(t *testing.T) {
	tmpDir := t.TempDir()
	setupGitRepo(t, tmpDir)

	// Create a second commit
	testFile := filepath.Join(tmpDir, "file2.txt")
	if err := os.WriteFile(testFile, []byte("content\n"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	cmd := exec.Command("git", "add", ".")
	cmd.Dir = tmpDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git add failed: %s: %v", output, err)
	}

	cmd = exec.Command("git", "commit", "-m", "Second commit")
	cmd.Dir = tmpDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit failed: %s: %v", output, err)
	}

	// Get the first commit hash
	cmd = exec.Command("git", "rev-parse", "HEAD~1")
	cmd.Dir = tmpDir
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("git rev-parse failed: %v", err)
	}
	firstCommit := strings.TrimSpace(string(output))

	// Checkout detached HEAD at first commit
	cmd = exec.Command("git", "checkout", firstCommit)
	cmd.Dir = tmpDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git checkout failed: %s: %v", output, err)
	}

	backend := NewGitBackend()
	rev, err := backend.GetCurrentRevision(tmpDir)
	if err != nil {
		t.Fatalf("GetCurrentRevision failed: %v", err)
	}

	// Should return short hash, not "HEAD"
	if rev == "HEAD" || rev == "" {
		t.Errorf("expected short commit hash, got %q", rev)
	}
	if len(rev) < 7 {
		t.Errorf("expected hash of at least 7 chars, got %q", rev)
	}
}

func TestGitBackend_GetCurrentRevision_NonRepo(t *testing.T) {
	tmpDir := t.TempDir()
	// Don't initialize git repo

	backend := NewGitBackend()

	_, err := backend.GetCurrentRevision(tmpDir)
	if err == nil {
		t.Error("expected error for non-repo directory")
	}
}

func TestGitBackend_HasChanges_Clean(t *testing.T) {
	tmpDir := t.TempDir()
	setupGitRepo(t, tmpDir)

	backend := NewGitBackend()

	hasChanges, err := backend.HasChanges(tmpDir)
	if err != nil {
		t.Fatalf("HasChanges failed: %v", err)
	}

	if hasChanges {
		t.Error("expected no changes in clean repo")
	}
}

func TestGitBackend_HasChanges_Dirty(t *testing.T) {
	tmpDir := t.TempDir()
	setupGitRepo(t, tmpDir)

	// Create uncommitted change
	testFile := filepath.Join(tmpDir, "newfile.txt")
	if err := os.WriteFile(testFile, []byte("new content\n"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	backend := NewGitBackend()

	hasChanges, err := backend.HasChanges(tmpDir)
	if err != nil {
		t.Fatalf("HasChanges failed: %v", err)
	}

	if !hasChanges {
		t.Error("expected changes in dirty repo")
	}
}

func TestGitBackend_IsolateAndReset_WithTarget(t *testing.T) {
	tmpDir := t.TempDir()
	setupGitRepo(t, tmpDir)

	// Get the main branch commit hash for verification
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = tmpDir
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("git rev-parse failed: %v", err)
	}
	mainCommit := strings.TrimSpace(string(output))

	// Create some changes (new file, not committed)
	testFile := filepath.Join(tmpDir, "work.txt")
	if err := os.WriteFile(testFile, []byte("work in progress\n"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	backend := NewGitBackend()

	// Isolate and reset to main
	branchName, err := backend.IsolateAndReset(tmpDir, "main")
	if err != nil {
		t.Fatalf("IsolateAndReset failed: %v", err)
	}

	// Branch name should be created
	if !strings.HasPrefix(branchName, "blocked-") {
		t.Errorf("expected branch name starting with 'blocked-', got %q", branchName)
	}

	// Should be back on main
	rev, err := backend.GetCurrentRevision(tmpDir)
	if err != nil {
		t.Fatalf("GetCurrentRevision failed: %v", err)
	}
	if rev != "main" {
		t.Errorf("expected to be on 'main', got %q", rev)
	}

	// Working directory should be clean
	hasChanges, err := backend.HasChanges(tmpDir)
	if err != nil {
		t.Fatalf("HasChanges failed: %v", err)
	}
	if hasChanges {
		t.Error("expected clean working directory after IsolateAndReset")
	}

	// Verify current commit is the original main commit
	cmd = exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = tmpDir
	output, err = cmd.Output()
	if err != nil {
		t.Fatalf("git rev-parse failed: %v", err)
	}
	currentCommit := strings.TrimSpace(string(output))
	if currentCommit != mainCommit {
		t.Errorf("expected to be at commit %s, got %s", mainCommit, currentCommit)
	}

	// Verify the blocked branch exists and has our work
	cmd = exec.Command("git", "rev-parse", "--verify", branchName)
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Errorf("blocked branch %q should exist: %v", branchName, err)
	}
}

func TestGitBackend_IsolateAndReset_EmptyTarget(t *testing.T) {
	tmpDir := t.TempDir()
	setupGitRepo(t, tmpDir)

	backend := NewGitBackend()

	// With empty target, should use main/master
	branchName, err := backend.IsolateAndReset(tmpDir, "")
	if err != nil {
		t.Fatalf("IsolateAndReset with empty target failed: %v", err)
	}

	if !strings.HasPrefix(branchName, "blocked-") {
		t.Errorf("expected branch name starting with 'blocked-', got %q", branchName)
	}

	// Should be on main
	rev, err := backend.GetCurrentRevision(tmpDir)
	if err != nil {
		t.Fatalf("GetCurrentRevision failed: %v", err)
	}
	if rev != "main" {
		t.Errorf("expected to be on 'main', got %q", rev)
	}
}

func TestGitBackend_IsolateAndReset_InvalidTarget(t *testing.T) {
	tmpDir := t.TempDir()
	setupGitRepo(t, tmpDir)

	backend := NewGitBackend()

	// With invalid target, should fail
	_, err := backend.IsolateAndReset(tmpDir, "nonexistent-branch-12345")
	if err == nil {
		t.Error("expected error for invalid target revision")
	}

	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("expected 'does not exist' error, got: %v", err)
	}
}

func TestGitBackend_IsolateAndReset_CreatesWIPCommit(t *testing.T) {
	tmpDir := t.TempDir()
	setupGitRepo(t, tmpDir)

	// Create uncommitted changes
	testFile := filepath.Join(tmpDir, "uncommitted.txt")
	if err := os.WriteFile(testFile, []byte("uncommitted work\n"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	backend := NewGitBackend()

	branchName, err := backend.IsolateAndReset(tmpDir, "main")
	if err != nil {
		t.Fatalf("IsolateAndReset failed: %v", err)
	}

	// Check that the blocked branch has a WIP commit
	cmd := exec.Command("git", "log", "-1", "--format=%s", branchName)
	cmd.Dir = tmpDir
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("git log failed: %v", err)
	}

	commitMsg := strings.TrimSpace(string(output))
	if !strings.HasPrefix(commitMsg, "BLOCKED:") {
		t.Errorf("expected commit message starting with 'BLOCKED:', got %q", commitMsg)
	}
}

func TestGitBackend_IsolateAndReset_NoChanges(t *testing.T) {
	tmpDir := t.TempDir()
	setupGitRepo(t, tmpDir)

	backend := NewGitBackend()

	// No uncommitted changes - should still work
	branchName, err := backend.IsolateAndReset(tmpDir, "main")
	if err != nil {
		t.Fatalf("IsolateAndReset with no changes failed: %v", err)
	}

	if !strings.HasPrefix(branchName, "blocked-") {
		t.Errorf("expected branch name starting with 'blocked-', got %q", branchName)
	}
}

func TestGitBackend_findMainBranch_Main(t *testing.T) {
	tmpDir := t.TempDir()
	setupGitRepo(t, tmpDir) // This creates 'main' branch

	backend := NewGitBackend()
	mainBranch := backend.findMainBranch(tmpDir)

	if mainBranch != "main" {
		t.Errorf("expected 'main', got %q", mainBranch)
	}
}

func TestGitBackend_findMainBranch_Master(t *testing.T) {
	tmpDir := t.TempDir()

	// Initialize git repo with master branch
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %s: %v", output, err)
	}

	cmd = exec.Command("git", "config", "user.email", "test@test.com")
	cmd.Dir = tmpDir
	cmd.Run()
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	cmd.Run()

	// Create initial commit on whatever default branch
	testFile := filepath.Join(tmpDir, "README.md")
	if err := os.WriteFile(testFile, []byte("# Test\n"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = tmpDir
	cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tmpDir
	cmd.Run()

	// Rename to master explicitly
	cmd = exec.Command("git", "branch", "-M", "master")
	cmd.Dir = tmpDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git branch -M master failed: %s: %v", output, err)
	}

	backend := NewGitBackend()
	mainBranch := backend.findMainBranch(tmpDir)

	if mainBranch != "master" {
		t.Errorf("expected 'master', got %q", mainBranch)
	}
}

func TestGitBackend_Commit(t *testing.T) {
	tmpDir := t.TempDir()
	setupGitRepo(t, tmpDir)

	// Create a new file to commit
	testFile := filepath.Join(tmpDir, "newfile.txt")
	if err := os.WriteFile(testFile, []byte("new content\n"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	backend := NewGitBackend()

	result, err := backend.Commit(tmpDir, "Test commit message")
	if err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	if !result.Success {
		t.Errorf("expected commit to succeed, got error: %s", result.ErrorMessage)
	}

	if result.CommitHash == "" {
		t.Error("expected commit hash to be set")
	}
}

func TestGitBackend_Commit_NoChanges(t *testing.T) {
	tmpDir := t.TempDir()
	setupGitRepo(t, tmpDir)

	backend := NewGitBackend()

	result, err := backend.Commit(tmpDir, "Test commit message")
	if err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	if !result.Success {
		t.Errorf("expected success even with no changes")
	}

	if result.StatusOutput != "No changes to commit" {
		t.Errorf("expected 'No changes to commit', got %q", result.StatusOutput)
	}
}

func TestGitBackend_Commit_EmptyMessage(t *testing.T) {
	tmpDir := t.TempDir()
	setupGitRepo(t, tmpDir)

	backend := NewGitBackend()

	result, err := backend.Commit(tmpDir, "")
	if err != nil {
		t.Fatalf("Commit returned error: %v", err)
	}

	if result.Success {
		t.Error("expected failure with empty message")
	}

	if !strings.Contains(result.ErrorMessage, "empty") {
		t.Errorf("expected 'empty' in error message, got %q", result.ErrorMessage)
	}
}

// =============================================================================
// JJ Backend Tests
// =============================================================================

// skipIfNoJJ skips the test if jj is not available
func skipIfNoJJ(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("jj"); err != nil {
		t.Skip("jj not installed, skipping test")
	}
}

// setupJJRepo creates a jj repo in the given directory with initial commit
func setupJJRepo(t *testing.T, dir string) {
	t.Helper()
	skipIfNoJJ(t)

	// Initialize jj repo
	cmd := exec.Command("jj", "git", "init")
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("jj git init failed: %s: %v", output, err)
	}

	// Create initial file
	testFile := filepath.Join(dir, "README.md")
	if err := os.WriteFile(testFile, []byte("# Test\n"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Describe the working copy
	cmd = exec.Command("jj", "desc", "-m", "Initial commit")
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("jj desc failed: %s: %v", output, err)
	}

	// Create new change (so we have a committed revision)
	cmd = exec.Command("jj", "new")
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("jj new failed: %s: %v", output, err)
	}
}

func TestJJBackend_Type(t *testing.T) {
	backend := NewJJBackend()
	if backend.Type() != VCSTypeJJ {
		t.Errorf("expected jj, got %s", backend.Type())
	}
}

func TestJJBackend_DescribeWorkingCopy(t *testing.T) {
	skipIfNoJJ(t)
	tmpDir := t.TempDir()
	setupJJRepo(t, tmpDir)

	backend := NewJJBackend()

	err := backend.DescribeWorkingCopy(tmpDir, "Test description")
	if err != nil {
		t.Fatalf("DescribeWorkingCopy failed: %v", err)
	}

	// Verify the description was set
	cmd := exec.Command("jj", "log", "-r", "@", "--no-graph", "-T", "description")
	cmd.Dir = tmpDir
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("jj log failed: %v", err)
	}

	desc := strings.TrimSpace(string(output))
	if desc != "Test description" {
		t.Errorf("expected description 'Test description', got %q", desc)
	}
}

func TestJJBackend_DescribeWorkingCopy_NonRepo(t *testing.T) {
	skipIfNoJJ(t)
	tmpDir := t.TempDir()
	// Don't initialize jj repo

	backend := NewJJBackend()

	err := backend.DescribeWorkingCopy(tmpDir, "Test description")
	if err == nil {
		t.Error("expected error for non-repo directory")
	}
}

func TestJJBackend_GetCurrentRevision(t *testing.T) {
	skipIfNoJJ(t)
	tmpDir := t.TempDir()
	setupJJRepo(t, tmpDir)

	backend := NewJJBackend()

	rev, err := backend.GetCurrentRevision(tmpDir)
	if err != nil {
		t.Fatalf("GetCurrentRevision failed: %v", err)
	}

	// Should return a non-empty change_id
	if rev == "" {
		t.Error("expected non-empty revision ID")
	}

	// JJ short change IDs are typically 8+ characters
	if len(rev) < 8 {
		t.Errorf("expected revision ID of at least 8 chars, got %q (len=%d)", rev, len(rev))
	}
}

func TestJJBackend_GetCurrentRevision_NonRepo(t *testing.T) {
	skipIfNoJJ(t)
	tmpDir := t.TempDir()
	// Don't initialize jj repo

	backend := NewJJBackend()

	_, err := backend.GetCurrentRevision(tmpDir)
	if err == nil {
		t.Error("expected error for non-repo directory")
	}
}

func TestJJBackend_HasChanges_Clean(t *testing.T) {
	skipIfNoJJ(t)
	tmpDir := t.TempDir()
	setupJJRepo(t, tmpDir)

	backend := NewJJBackend()

	hasChanges, err := backend.HasChanges(tmpDir)
	if err != nil {
		t.Fatalf("HasChanges failed: %v", err)
	}

	// Fresh working copy should have no changes
	if hasChanges {
		t.Error("expected no changes in clean working copy")
	}
}

func TestJJBackend_HasChanges_Dirty(t *testing.T) {
	skipIfNoJJ(t)
	tmpDir := t.TempDir()
	setupJJRepo(t, tmpDir)

	// Create uncommitted change
	testFile := filepath.Join(tmpDir, "newfile.txt")
	if err := os.WriteFile(testFile, []byte("new content\n"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	backend := NewJJBackend()

	hasChanges, err := backend.HasChanges(tmpDir)
	if err != nil {
		t.Fatalf("HasChanges failed: %v", err)
	}

	if !hasChanges {
		t.Error("expected changes in dirty working copy")
	}
}

func TestJJBackend_IsolateAndReset_EmptyTarget(t *testing.T) {
	skipIfNoJJ(t)
	tmpDir := t.TempDir()
	setupJJRepo(t, tmpDir)

	// Create some changes
	testFile := filepath.Join(tmpDir, "work.txt")
	if err := os.WriteFile(testFile, []byte("work in progress\n"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Describe the work
	cmd := exec.Command("jj", "desc", "-m", "Work in progress")
	cmd.Dir = tmpDir
	cmd.Run()

	backend := NewJJBackend()

	// Get current revision before isolating
	revBefore, err := backend.GetCurrentRevision(tmpDir)
	if err != nil {
		t.Fatalf("GetCurrentRevision failed: %v", err)
	}

	// Isolate with empty target (should use @-)
	isolatedRev, err := backend.IsolateAndReset(tmpDir, "")
	if err != nil {
		t.Fatalf("IsolateAndReset failed: %v", err)
	}

	// Isolated revision should be the one we were working on
	if isolatedRev != revBefore {
		t.Errorf("expected isolated revision %q, got %q", revBefore, isolatedRev)
	}

	// Current revision should be different now
	revAfter, err := backend.GetCurrentRevision(tmpDir)
	if err != nil {
		t.Fatalf("GetCurrentRevision failed: %v", err)
	}

	if revAfter == revBefore {
		t.Error("expected current revision to change after IsolateAndReset")
	}
}

func TestJJBackend_IsolateAndReset_WithTarget(t *testing.T) {
	skipIfNoJJ(t)
	tmpDir := t.TempDir()
	setupJJRepo(t, tmpDir)

	// Get the parent revision (root) for targeting
	cmd := exec.Command("jj", "log", "-r", "root()", "--no-graph", "-T", "change_id.short()")
	cmd.Dir = tmpDir
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("jj log failed: %v", err)
	}
	rootRev := strings.TrimSpace(string(output))

	// Create some changes
	testFile := filepath.Join(tmpDir, "work.txt")
	if err := os.WriteFile(testFile, []byte("work in progress\n"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	backend := NewJJBackend()

	// Get current revision before isolating
	revBefore, err := backend.GetCurrentRevision(tmpDir)
	if err != nil {
		t.Fatalf("GetCurrentRevision failed: %v", err)
	}

	// Isolate with specific target
	isolatedRev, err := backend.IsolateAndReset(tmpDir, rootRev)
	if err != nil {
		t.Fatalf("IsolateAndReset failed: %v", err)
	}

	// Isolated revision should be the one we were working on
	if isolatedRev != revBefore {
		t.Errorf("expected isolated revision %q, got %q", revBefore, isolatedRev)
	}

	// New working copy should have root as parent
	cmd = exec.Command("jj", "log", "-r", "@-", "--no-graph", "-T", "change_id.short()")
	cmd.Dir = tmpDir
	output, err = cmd.Output()
	if err != nil {
		t.Fatalf("jj log failed: %v", err)
	}
	parentRev := strings.TrimSpace(string(output))

	if parentRev != rootRev {
		t.Errorf("expected parent to be %q, got %q", rootRev, parentRev)
	}
}

func TestJJBackend_IsolateAndReset_InvalidTarget(t *testing.T) {
	skipIfNoJJ(t)
	tmpDir := t.TempDir()
	setupJJRepo(t, tmpDir)

	backend := NewJJBackend()

	// With invalid target, should fail
	_, err := backend.IsolateAndReset(tmpDir, "nonexistent12345")
	if err == nil {
		t.Error("expected error for invalid target revision")
	}
}

func TestJJBackend_Commit(t *testing.T) {
	skipIfNoJJ(t)
	tmpDir := t.TempDir()
	setupJJRepo(t, tmpDir)

	// Create a new file
	testFile := filepath.Join(tmpDir, "newfile.txt")
	if err := os.WriteFile(testFile, []byte("new content\n"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	backend := NewJJBackend()

	result, err := backend.Commit(tmpDir, "Test commit message")
	if err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	if !result.Success {
		t.Errorf("expected commit to succeed, got error: %s", result.ErrorMessage)
	}
}

func TestJJBackend_Commit_NoChanges(t *testing.T) {
	skipIfNoJJ(t)
	tmpDir := t.TempDir()
	setupJJRepo(t, tmpDir)

	backend := NewJJBackend()

	result, err := backend.Commit(tmpDir, "Test commit message")
	if err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	if !result.Success {
		t.Errorf("expected success even with no changes")
	}

	if result.StatusOutput != "No changes to commit" {
		t.Errorf("expected 'No changes to commit', got %q", result.StatusOutput)
	}
}

func TestJJBackend_Commit_EmptyMessage(t *testing.T) {
	skipIfNoJJ(t)
	tmpDir := t.TempDir()
	setupJJRepo(t, tmpDir)

	backend := NewJJBackend()

	result, err := backend.Commit(tmpDir, "")
	if err != nil {
		t.Fatalf("Commit returned error: %v", err)
	}

	if result.Success {
		t.Error("expected failure with empty message")
	}

	if !strings.Contains(result.ErrorMessage, "empty") {
		t.Errorf("expected 'empty' in error message, got %q", result.ErrorMessage)
	}
}

func TestJJBackend_GetLastCommitHash(t *testing.T) {
	skipIfNoJJ(t)
	tmpDir := t.TempDir()
	setupJJRepo(t, tmpDir)

	backend := NewJJBackend()

	hash, err := backend.GetLastCommitHash(tmpDir)
	if err != nil {
		t.Fatalf("GetLastCommitHash failed: %v", err)
	}

	if hash == "" {
		t.Error("expected non-empty commit hash")
	}
}

// =============================================================================
// Integration Tests
// =============================================================================

func TestVCS_InterfaceCompliance_Git(t *testing.T) {
	var _ VCS = (*GitBackend)(nil)
}

func TestVCS_InterfaceCompliance_JJ(t *testing.T) {
	var _ VCS = (*JJBackend)(nil)
}
