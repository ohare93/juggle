package vcs

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// GitBackend implements VCS for Git.
type GitBackend struct{}

// NewGitBackend creates a new Git backend instance.
func NewGitBackend() *GitBackend {
	return &GitBackend{}
}

// Type returns VCSTypeGit.
func (g *GitBackend) Type() VCSType {
	return VCSTypeGit
}

// Status returns the output of git status.
func (g *GitBackend) Status(projectDir string) (string, error) {
	cmd := exec.Command("git", "status")
	cmd.Dir = projectDir
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// HasChanges returns true if there are uncommitted changes.
func (g *GitBackend) HasChanges(projectDir string) (bool, error) {
	output, err := g.Status(projectDir)
	if err != nil {
		return false, err
	}
	// git status outputs "nothing to commit, working tree clean" when clean
	return !strings.Contains(output, "nothing to commit"), nil
}

// Commit stages all changes and creates a git commit with the given message.
func (g *GitBackend) Commit(projectDir, message string) (*CommitResult, error) {
	result := &CommitResult{}

	// Validate commit message
	if message == "" {
		result.ErrorMessage = "commit message cannot be empty"
		return result, nil
	}
	if len(message) > 5000 {
		result.ErrorMessage = "commit message too long (max 5000 chars)"
		return result, nil
	}

	// Check for changes first
	hasChanges, err := g.HasChanges(projectDir)
	if err != nil {
		result.ErrorMessage = err.Error()
		return result, nil
	}
	if !hasChanges {
		result.Success = true
		result.StatusOutput = "No changes to commit"
		return result, nil
	}

	// Stage all changes
	stageCmd := exec.Command("git", "add", "-A")
	stageCmd.Dir = projectDir
	if output, err := stageCmd.CombinedOutput(); err != nil {
		result.ErrorMessage = string(output)
		return result, nil
	}

	// Perform the commit
	commitCmd := exec.Command("git", "commit", "-m", message)
	commitCmd.Dir = projectDir
	commitOutput, err := commitCmd.CombinedOutput()
	if err != nil {
		result.ErrorMessage = string(commitOutput)
		return result, nil
	}

	result.Success = true

	// Get commit hash (best effort - don't fail if this doesn't work)
	if hash, err := g.GetLastCommitHash(projectDir); err == nil {
		result.CommitHash = hash
	}

	// Get final status (best effort)
	if status, err := g.Status(projectDir); err == nil {
		result.StatusOutput = strings.TrimSpace(status)
	}

	return result, nil
}

// GetLastCommitHash returns the short hash of the last commit.
func (g *GitBackend) GetLastCommitHash(projectDir string) (string, error) {
	cmd := exec.Command("git", "log", "-1", "--format=%h")
	cmd.Dir = projectDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// DescribeWorkingCopy is a no-op for git (git doesn't have working copy descriptions).
func (g *GitBackend) DescribeWorkingCopy(projectDir, message string) error {
	// Git doesn't have a concept of working copy descriptions like jj does.
	// This is a no-op.
	return nil
}

// IsolateAndReset creates a branch for the current work and checks out a target revision.
// If targetRevision is empty, attempts to find main/master branch.
// Returns the name of the created branch containing the isolated work.
func (g *GitBackend) IsolateAndReset(projectDir, targetRevision string) (string, error) {
	// Generate a unique branch name for the blocked work
	branchName := fmt.Sprintf("blocked-%s", time.Now().Format("20060102-150405"))

	// Determine target revision first
	target := targetRevision
	if target == "" {
		// Fall back to main/master if no target specified
		target = g.findMainBranch(projectDir)
	}

	// Resolve target to a commit hash BEFORE making any commits
	// This is important because if we're on "main" and commit, main advances
	resolveCmd := exec.Command("git", "rev-parse", "--verify", target)
	resolveCmd.Dir = projectDir
	output, err := resolveCmd.Output()
	if err != nil {
		return "", fmt.Errorf("target revision %q does not exist: %w", target, err)
	}
	targetCommit := strings.TrimSpace(string(output))

	// Check for changes
	hasChanges, err := g.HasChanges(projectDir)
	if err != nil {
		return "", fmt.Errorf("failed to check for changes: %w", err)
	}

	var wipCommit string
	if hasChanges {
		// Stage all changes
		stageCmd := exec.Command("git", "add", "-A")
		stageCmd.Dir = projectDir
		if output, err := stageCmd.CombinedOutput(); err != nil {
			return "", fmt.Errorf("git add failed: %s: %w", strings.TrimSpace(string(output)), err)
		}

		// Create a WIP commit
		commitCmd := exec.Command("git", "commit", "-m", "BLOCKED: WIP - work in progress")
		commitCmd.Dir = projectDir
		if output, err := commitCmd.CombinedOutput(); err != nil {
			return "", fmt.Errorf("git commit failed: %s: %w", strings.TrimSpace(string(output)), err)
		}

		// Get the WIP commit hash so we can create a branch at it after checkout
		hashCmd := exec.Command("git", "rev-parse", "HEAD")
		hashCmd.Dir = projectDir
		output, err := hashCmd.Output()
		if err != nil {
			return "", fmt.Errorf("failed to get WIP commit hash: %w", err)
		}
		wipCommit = strings.TrimSpace(string(output))
	} else {
		// No changes - get current HEAD for the branch
		hashCmd := exec.Command("git", "rev-parse", "HEAD")
		hashCmd.Dir = projectDir
		output, err := hashCmd.Output()
		if err != nil {
			return "", fmt.Errorf("failed to get HEAD hash: %w", err)
		}
		wipCommit = strings.TrimSpace(string(output))
	}

	// Create a branch pointing to the WIP commit
	branchCmd := exec.Command("git", "branch", branchName, wipCommit)
	branchCmd.Dir = projectDir
	if output, err := branchCmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("git branch failed: %s: %w", strings.TrimSpace(string(output)), err)
	}

	// Reset the current branch to the target commit
	// This moves the branch pointer back to where it was before we committed
	resetCmd := exec.Command("git", "reset", "--hard", targetCommit)
	resetCmd.Dir = projectDir
	if output, err := resetCmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("git reset failed: %s: %w", strings.TrimSpace(string(output)), err)
	}

	return branchName, nil
}

// findMainBranch determines the default branch for the repo.
// Checks in order: origin/HEAD reference, main, master.
func (g *GitBackend) findMainBranch(projectDir string) string {
	// Try to get the default branch from origin/HEAD
	cmd := exec.Command("git", "symbolic-ref", "--short", "refs/remotes/origin/HEAD")
	cmd.Dir = projectDir
	if output, err := cmd.Output(); err == nil {
		// Returns something like "origin/main", we want just "main"
		ref := strings.TrimSpace(string(output))
		if strings.HasPrefix(ref, "origin/") {
			return strings.TrimPrefix(ref, "origin/")
		}
		return ref
	}

	// Check if "main" exists
	cmd = exec.Command("git", "rev-parse", "--verify", "main")
	cmd.Dir = projectDir
	if err := cmd.Run(); err == nil {
		return "main"
	}

	// Check if "master" exists
	cmd = exec.Command("git", "rev-parse", "--verify", "master")
	cmd.Dir = projectDir
	if err := cmd.Run(); err == nil {
		return "master"
	}

	// Default to "main" (will fail at checkout if doesn't exist, but that's handled)
	return "main"
}

// GetCurrentRevision returns the current branch name or commit hash.
func (g *GitBackend) GetCurrentRevision(projectDir string) (string, error) {
	// Try to get the current branch name first
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = projectDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git rev-parse failed: %s: %w", strings.TrimSpace(string(output)), err)
	}

	result := strings.TrimSpace(string(output))

	// If HEAD is detached, get the short commit hash instead
	if result == "HEAD" {
		return g.GetLastCommitHash(projectDir)
	}

	return result, nil
}
