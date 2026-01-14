package vcs

import (
	"os/exec"
	"strings"
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
