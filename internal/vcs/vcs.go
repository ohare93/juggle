// Package vcs provides a unified interface for version control systems.
package vcs

// VCSType represents the version control system type.
type VCSType string

const (
	VCSTypeJJ  VCSType = "jj"
	VCSTypeGit VCSType = "git"
)

// String returns the string representation of VCSType.
func (v VCSType) String() string {
	return string(v)
}

// IsValid returns true if the VCSType is a known valid type.
func (v VCSType) IsValid() bool {
	return v == VCSTypeJJ || v == VCSTypeGit
}

// CommitResult represents the outcome of a commit operation.
type CommitResult struct {
	Success      bool   // Whether the commit succeeded
	CommitHash   string // Short hash of the new commit (if successful)
	StatusOutput string // Output from status after commit
	ErrorMessage string // Error message if commit failed
}

// VCS defines the interface for version control operations.
type VCS interface {
	// Type returns the VCS type (jj or git)
	Type() VCSType

	// Status returns the current status output
	Status(projectDir string) (string, error)

	// HasChanges returns true if there are uncommitted changes
	HasChanges(projectDir string) (bool, error)

	// Commit creates a commit with the given message
	Commit(projectDir, message string) (*CommitResult, error)

	// GetLastCommitHash returns the short hash of the last commit
	GetLastCommitHash(projectDir string) (string, error)
}

// GetBackend returns the appropriate VCS backend for the given type.
func GetBackend(vcsType VCSType) VCS {
	switch vcsType {
	case VCSTypeJJ:
		return NewJJBackend()
	case VCSTypeGit:
		return NewGitBackend()
	default:
		return NewGitBackend() // Default to git
	}
}

// GetBackendForProject returns the VCS backend for a project, using config resolution.
func GetBackendForProject(projectDir string, projectVCS, globalVCS VCSType) VCS {
	vcsType := Detect(projectDir, projectVCS, globalVCS)
	return GetBackend(vcsType)
}
