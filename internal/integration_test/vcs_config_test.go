package integration_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ohare93/juggle/internal/session"
	"github.com/ohare93/juggle/internal/vcs"
)

// TestVCSConfig_AutoDetectGit tests that git is detected when .git exists
func TestVCSConfig_AutoDetectGit(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create .git directory
	if err := os.MkdirAll(filepath.Join(env.ProjectDir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}

	// Without any config, should auto-detect git
	detected := vcs.AutoDetect(env.ProjectDir)
	if detected != vcs.VCSTypeGit {
		t.Errorf("expected git, got %s", detected)
	}
}

// TestVCSConfig_AutoDetectJJ tests that jj is detected when .jj exists
func TestVCSConfig_AutoDetectJJ(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create .jj directory
	if err := os.MkdirAll(filepath.Join(env.ProjectDir, ".jj"), 0755); err != nil {
		t.Fatal(err)
	}

	detected := vcs.AutoDetect(env.ProjectDir)
	if detected != vcs.VCSTypeJJ {
		t.Errorf("expected jj, got %s", detected)
	}
}

// TestVCSConfig_JJPriorityInAutoDetect tests that jj takes priority over git
func TestVCSConfig_JJPriorityInAutoDetect(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create both .jj and .git directories
	if err := os.MkdirAll(filepath.Join(env.ProjectDir, ".jj"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(env.ProjectDir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}

	// JJ should take priority
	detected := vcs.AutoDetect(env.ProjectDir)
	if detected != vcs.VCSTypeJJ {
		t.Errorf("expected jj (priority), got %s", detected)
	}
}

// TestVCSConfig_DefaultToGit tests that git is the default when nothing is detected
func TestVCSConfig_DefaultToGit(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// No VCS directories created
	detected := vcs.AutoDetect(env.ProjectDir)
	if detected != vcs.VCSTypeGit {
		t.Errorf("expected git (default), got %s", detected)
	}
}

// TestVCSConfig_GlobalConfigOverridesAutoDetect tests that global config overrides auto-detection
func TestVCSConfig_GlobalConfigOverridesAutoDetect(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create .jj directory (would be auto-detected)
	if err := os.MkdirAll(filepath.Join(env.ProjectDir, ".jj"), 0755); err != nil {
		t.Fatal(err)
	}

	// Set global config to git
	opts := session.ConfigOptions{
		ConfigHome:    env.ConfigHome,
		JuggleDirName: ".juggle",
	}
	if err := session.UpdateGlobalVCSWithOptions(opts, "git"); err != nil {
		t.Fatalf("failed to set global VCS: %v", err)
	}

	// Get the VCS settings
	globalVCS, err := session.GetGlobalVCSWithOptions(opts)
	if err != nil {
		t.Fatalf("failed to get global VCS: %v", err)
	}

	// Should use global config over auto-detect
	effective := vcs.Detect(env.ProjectDir, "", vcs.VCSType(globalVCS))
	if effective != vcs.VCSTypeGit {
		t.Errorf("expected git (global config), got %s", effective)
	}
}

// TestVCSConfig_ProjectConfigOverridesGlobal tests that project config overrides global
func TestVCSConfig_ProjectConfigOverridesGlobal(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create the .juggle directory first
	if err := os.MkdirAll(filepath.Join(env.ProjectDir, ".juggle"), 0755); err != nil {
		t.Fatal(err)
	}

	// Set global config to git
	opts := session.ConfigOptions{
		ConfigHome:    env.ConfigHome,
		JuggleDirName: ".juggle",
	}
	if err := session.UpdateGlobalVCSWithOptions(opts, "git"); err != nil {
		t.Fatalf("failed to set global VCS: %v", err)
	}

	// Set project config to jj
	if err := session.UpdateProjectVCS(env.ProjectDir, "jj"); err != nil {
		t.Fatalf("failed to set project VCS: %v", err)
	}

	// Get VCS settings
	projectVCS, err := session.GetProjectVCS(env.ProjectDir)
	if err != nil {
		t.Fatalf("failed to get project VCS: %v", err)
	}
	globalVCS, err := session.GetGlobalVCSWithOptions(opts)
	if err != nil {
		t.Fatalf("failed to get global VCS: %v", err)
	}

	// Project should take priority
	effective := vcs.Detect(env.ProjectDir, vcs.VCSType(projectVCS), vcs.VCSType(globalVCS))
	if effective != vcs.VCSTypeJJ {
		t.Errorf("expected jj (project config), got %s", effective)
	}
}

// TestVCSConfig_SetAndClearGlobal tests setting and clearing global VCS
func TestVCSConfig_SetAndClearGlobal(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	opts := session.ConfigOptions{
		ConfigHome:    env.ConfigHome,
		JuggleDirName: ".juggle",
	}

	// Initially should be empty
	globalVCS, err := session.GetGlobalVCSWithOptions(opts)
	if err != nil {
		t.Fatalf("failed to get global VCS: %v", err)
	}
	if globalVCS != "" {
		t.Errorf("expected empty global VCS initially, got %s", globalVCS)
	}

	// Set to jj
	if err := session.UpdateGlobalVCSWithOptions(opts, "jj"); err != nil {
		t.Fatalf("failed to set global VCS: %v", err)
	}

	globalVCS, err = session.GetGlobalVCSWithOptions(opts)
	if err != nil {
		t.Fatalf("failed to get global VCS: %v", err)
	}
	if globalVCS != "jj" {
		t.Errorf("expected jj, got %s", globalVCS)
	}

	// Clear
	if err := session.ClearGlobalVCSWithOptions(opts); err != nil {
		t.Fatalf("failed to clear global VCS: %v", err)
	}

	globalVCS, err = session.GetGlobalVCSWithOptions(opts)
	if err != nil {
		t.Fatalf("failed to get global VCS: %v", err)
	}
	if globalVCS != "" {
		t.Errorf("expected empty after clear, got %s", globalVCS)
	}
}

// TestVCSConfig_SetAndClearProject tests setting and clearing project VCS
func TestVCSConfig_SetAndClearProject(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	// Create the .juggle directory first
	if err := os.MkdirAll(filepath.Join(env.ProjectDir, ".juggle"), 0755); err != nil {
		t.Fatal(err)
	}

	// Initially should be empty
	projectVCS, err := session.GetProjectVCS(env.ProjectDir)
	if err != nil {
		t.Fatalf("failed to get project VCS: %v", err)
	}
	if projectVCS != "" {
		t.Errorf("expected empty project VCS initially, got %s", projectVCS)
	}

	// Set to git
	if err := session.UpdateProjectVCS(env.ProjectDir, "git"); err != nil {
		t.Fatalf("failed to set project VCS: %v", err)
	}

	projectVCS, err = session.GetProjectVCS(env.ProjectDir)
	if err != nil {
		t.Fatalf("failed to get project VCS: %v", err)
	}
	if projectVCS != "git" {
		t.Errorf("expected git, got %s", projectVCS)
	}

	// Clear
	if err := session.ClearProjectVCS(env.ProjectDir); err != nil {
		t.Fatalf("failed to clear project VCS: %v", err)
	}

	projectVCS, err = session.GetProjectVCS(env.ProjectDir)
	if err != nil {
		t.Fatalf("failed to get project VCS: %v", err)
	}
	if projectVCS != "" {
		t.Errorf("expected empty after clear, got %s", projectVCS)
	}
}

// TestVCSConfig_InvalidVCSType tests that invalid VCS types are rejected
func TestVCSConfig_InvalidVCSType(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	opts := session.ConfigOptions{
		ConfigHome:    env.ConfigHome,
		JuggleDirName: ".juggle",
	}

	// Try to set invalid VCS type
	err := session.UpdateGlobalVCSWithOptions(opts, "svn")
	if err == nil {
		t.Error("expected error for invalid VCS type 'svn'")
	}

	// Create .juggle directory for project config
	if err := os.MkdirAll(filepath.Join(env.ProjectDir, ".juggle"), 0755); err != nil {
		t.Fatal(err)
	}

	err = session.UpdateProjectVCS(env.ProjectDir, "mercurial")
	if err == nil {
		t.Error("expected error for invalid VCS type 'mercurial'")
	}
}

// TestVCSConfig_VCSPreservedInConfig tests that VCS field is preserved in config
func TestVCSConfig_VCSPreservedInConfig(t *testing.T) {
	env := SetupTestEnv(t)
	defer CleanupTestEnv(t, env)

	opts := session.ConfigOptions{
		ConfigHome:    env.ConfigHome,
		JuggleDirName: ".juggle",
	}

	// Set VCS to jj
	if err := session.UpdateGlobalVCSWithOptions(opts, "jj"); err != nil {
		t.Fatalf("failed to set global VCS: %v", err)
	}

	// Load config and add a search path (modify something else)
	config, err := session.LoadConfigWithOptions(opts)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	config.AddSearchPath("/test/path")
	if err := config.SaveWithOptions(opts); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Reload and verify VCS is still set
	config2, err := session.LoadConfigWithOptions(opts)
	if err != nil {
		t.Fatalf("failed to reload config: %v", err)
	}

	if config2.GetVCS() != "jj" {
		t.Errorf("expected VCS to be preserved as 'jj', got '%s'", config2.GetVCS())
	}
}

// TestVCSConfig_GetBackend tests GetBackend returns correct backend types
func TestVCSConfig_GetBackend(t *testing.T) {
	tests := []struct {
		vcsType vcs.VCSType
		want    vcs.VCSType
	}{
		{vcs.VCSTypeJJ, vcs.VCSTypeJJ},
		{vcs.VCSTypeGit, vcs.VCSTypeGit},
		{"unknown", vcs.VCSTypeGit}, // defaults to git
		{"", vcs.VCSTypeGit},        // defaults to git
	}

	for _, tt := range tests {
		t.Run(string(tt.vcsType), func(t *testing.T) {
			backend := vcs.GetBackend(tt.vcsType)
			if backend.Type() != tt.want {
				t.Errorf("GetBackend(%s).Type() = %s, want %s", tt.vcsType, backend.Type(), tt.want)
			}
		})
	}
}
