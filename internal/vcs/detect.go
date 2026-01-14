package vcs

import (
	"os"
	"path/filepath"
)

// Detect determines the VCS type for a project directory.
// Priority (highest to lowest):
//  1. Project config (if set and non-empty)
//  2. Global config (if set and non-empty)
//  3. Auto-detect: check for .jj directory first, then .git
//  4. Default: git
func Detect(projectDir string, projectVCS, globalVCS VCSType) VCSType {
	// 1. Project config has highest priority
	if projectVCS != "" {
		return projectVCS
	}

	// 2. Global config is next
	if globalVCS != "" {
		return globalVCS
	}

	// 3. Auto-detect: check for .jj first, then .git
	return AutoDetect(projectDir)
}

// AutoDetect checks the filesystem for VCS directories.
// Returns VCSTypeJJ if .jj exists, VCSTypeGit if .git exists.
// Defaults to VCSTypeGit if neither is found.
func AutoDetect(projectDir string) VCSType {
	// Check for jj first (higher priority)
	if _, err := os.Stat(filepath.Join(projectDir, ".jj")); err == nil {
		return VCSTypeJJ
	}

	// Check for git
	if _, err := os.Stat(filepath.Join(projectDir, ".git")); err == nil {
		return VCSTypeGit
	}

	// Default to git
	return VCSTypeGit
}
