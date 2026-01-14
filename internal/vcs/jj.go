package vcs

import (
	"os/exec"
	"strings"
)

// JJBackend implements VCS for Jujutsu (jj).
type JJBackend struct{}

// NewJJBackend creates a new JJ backend instance.
func NewJJBackend() *JJBackend {
	return &JJBackend{}
}

// Type returns VCSTypeJJ.
func (j *JJBackend) Type() VCSType {
	return VCSTypeJJ
}

// Status returns the output of jj status.
func (j *JJBackend) Status(projectDir string) (string, error) {
	cmd := exec.Command("jj", "status")
	cmd.Dir = projectDir
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// HasChanges returns true if the working copy has changes.
func (j *JJBackend) HasChanges(projectDir string) (bool, error) {
	output, err := j.Status(projectDir)
	if err != nil {
		return false, err
	}
	// jj status outputs "The working copy has no changes." when clean
	return !strings.Contains(output, "The working copy has no changes."), nil
}

// Commit creates a jj commit with the given message.
func (j *JJBackend) Commit(projectDir, message string) (*CommitResult, error) {
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
	hasChanges, err := j.HasChanges(projectDir)
	if err != nil {
		result.ErrorMessage = err.Error()
		return result, nil
	}
	if !hasChanges {
		result.Success = true
		result.StatusOutput = "No changes to commit"
		return result, nil
	}

	// Perform the commit
	commitCmd := exec.Command("jj", "commit", "-m", message)
	commitCmd.Dir = projectDir
	commitOutput, err := commitCmd.CombinedOutput()
	if err != nil {
		result.ErrorMessage = string(commitOutput)
		return result, nil
	}

	result.Success = true

	// Get commit hash (best effort - don't fail if this doesn't work)
	if hash, err := j.GetLastCommitHash(projectDir); err == nil {
		result.CommitHash = hash
	}

	// Get final status (best effort)
	if status, err := j.Status(projectDir); err == nil {
		result.StatusOutput = strings.TrimSpace(status)
	}

	return result, nil
}

// GetLastCommitHash returns the short hash of the last commit.
func (j *JJBackend) GetLastCommitHash(projectDir string) (string, error) {
	cmd := exec.Command("jj", "log", "-n", "1", "--no-graph", "-T", "commit_id.short()")
	cmd.Dir = projectDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}
