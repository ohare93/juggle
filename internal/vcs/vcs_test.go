package vcs

import (
	"os"
	"path/filepath"
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
