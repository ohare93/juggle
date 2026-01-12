package tui

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAutocompleteStateReset(t *testing.T) {
	ac := NewAutocompleteState("/tmp/test")
	ac.Active = true
	ac.Query = "test"
	ac.QueryStart = 5
	ac.Suggestions = []string{"file1.go", "file2.go"}
	ac.Selected = 1

	ac.Reset()

	if ac.Active {
		t.Error("Expected Active to be false after reset")
	}
	if ac.Query != "" {
		t.Errorf("Expected Query to be empty, got %q", ac.Query)
	}
	if ac.QueryStart != 0 {
		t.Errorf("Expected QueryStart to be 0, got %d", ac.QueryStart)
	}
	if len(ac.Suggestions) != 0 {
		t.Errorf("Expected Suggestions to be empty, got %v", ac.Suggestions)
	}
	if ac.Selected != 0 {
		t.Errorf("Expected Selected to be 0, got %d", ac.Selected)
	}
}

func TestAutocompleteUpdateFromTextActivates(t *testing.T) {
	// Create a temp directory for testing
	tmpDir, err := os.MkdirTemp("", "autocomplete-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create some test files
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "utils.go"), []byte("package main"), 0644)

	ac := NewAutocompleteState(tmpDir)

	// Typing "@" should activate
	text := "Reference @m"
	changed := ac.UpdateFromText(text, len(text))

	if !changed {
		t.Error("Expected UpdateFromText to return true when state changes")
	}
	if !ac.Active {
		t.Error("Expected autocomplete to be active after typing @")
	}
	if ac.Query != "m" {
		t.Errorf("Expected Query to be 'm', got %q", ac.Query)
	}
	if ac.QueryStart != 10 { // Position of @ in "Reference @m"
		t.Errorf("Expected QueryStart to be 10, got %d", ac.QueryStart)
	}
}

func TestAutocompleteUpdateFromTextDeactivates(t *testing.T) {
	ac := NewAutocompleteState("/tmp")
	ac.Active = true
	ac.Query = "test"
	ac.QueryStart = 5

	// Text without @ should deactivate
	text := "No at sign here"
	changed := ac.UpdateFromText(text, len(text))

	if !changed {
		t.Error("Expected UpdateFromText to return true when deactivating")
	}
	if ac.Active {
		t.Error("Expected autocomplete to be inactive when no @ present")
	}
}

func TestAutocompleteDeactivate(t *testing.T) {
	ac := NewAutocompleteState("/tmp")
	ac.Active = true
	ac.Query = "test"
	ac.QueryStart = 5
	ac.Suggestions = []string{"file1.go"}

	ac.Deactivate()

	if ac.Active {
		t.Error("Expected Active to be false after Deactivate")
	}
	// Other fields should NOT be reset by Deactivate
	if ac.Query == "" {
		t.Error("Deactivate should only set Active to false, not reset other fields")
	}
}

func TestAutocompleteSelectNextPrev(t *testing.T) {
	ac := NewAutocompleteState("/tmp")
	ac.Active = true
	ac.Suggestions = []string{"file1.go", "file2.go", "file3.go"}
	ac.Selected = 0

	// SelectNext
	ac.SelectNext()
	if ac.Selected != 1 {
		t.Errorf("Expected Selected to be 1 after SelectNext, got %d", ac.Selected)
	}

	ac.SelectNext()
	if ac.Selected != 2 {
		t.Errorf("Expected Selected to be 2 after SelectNext, got %d", ac.Selected)
	}

	// Should wrap around
	ac.SelectNext()
	if ac.Selected != 0 {
		t.Errorf("Expected Selected to wrap to 0, got %d", ac.Selected)
	}

	// SelectPrev
	ac.SelectPrev()
	if ac.Selected != 2 {
		t.Errorf("Expected Selected to wrap to 2, got %d", ac.Selected)
	}

	ac.SelectPrev()
	if ac.Selected != 1 {
		t.Errorf("Expected Selected to be 1 after SelectPrev, got %d", ac.Selected)
	}
}

func TestAutocompleteGetSelectedSuggestion(t *testing.T) {
	ac := NewAutocompleteState("/tmp")
	ac.Suggestions = []string{"file1.go", "file2.go"}

	// Selected 0
	ac.Selected = 0
	if got := ac.GetSelectedSuggestion(); got != "file1.go" {
		t.Errorf("Expected 'file1.go', got %q", got)
	}

	// Selected 1
	ac.Selected = 1
	if got := ac.GetSelectedSuggestion(); got != "file2.go" {
		t.Errorf("Expected 'file2.go', got %q", got)
	}

	// Out of bounds
	ac.Selected = 5
	if got := ac.GetSelectedSuggestion(); got != "" {
		t.Errorf("Expected empty string for out-of-bounds, got %q", got)
	}
}

func TestAutocompleteApplyCompletion(t *testing.T) {
	ac := NewAutocompleteState("/tmp")
	ac.Active = true
	ac.Query = "main"
	ac.QueryStart = 10 // Position of @ in "Reference @main"
	ac.Suggestions = []string{"main.go"}
	ac.Selected = 0

	text := "Reference @main"
	newText := ac.ApplyCompletion(text)

	expected := "Reference main.go"
	if newText != expected {
		t.Errorf("Expected %q, got %q", expected, newText)
	}
}

func TestAutocompleteApplyCompletionMiddleOfText(t *testing.T) {
	ac := NewAutocompleteState("/tmp")
	ac.Active = true
	ac.Query = "ut"
	ac.QueryStart = 4 // Position of @ in "See @ut for details" (S=0,e=1,e=2, =3,@=4)
	ac.Suggestions = []string{"utils.go"}
	ac.Selected = 0

	text := "See @ut for details"
	newText := ac.ApplyCompletion(text)

	expected := "See utils.go for details"
	if newText != expected {
		t.Errorf("Expected %q, got %q", expected, newText)
	}
}

func TestAutocompleteNoApplyWhenInactive(t *testing.T) {
	ac := NewAutocompleteState("/tmp")
	ac.Active = false
	ac.Suggestions = []string{"file.go"}

	text := "Original text"
	newText := ac.ApplyCompletion(text)

	if newText != text {
		t.Errorf("Expected no change when inactive, got %q", newText)
	}
}

func TestAutocompleteNoApplyWhenNoSuggestions(t *testing.T) {
	ac := NewAutocompleteState("/tmp")
	ac.Active = true
	ac.Suggestions = []string{}

	text := "Original text"
	newText := ac.ApplyCompletion(text)

	if newText != text {
		t.Errorf("Expected no change when no suggestions, got %q", newText)
	}
}

func TestFindMatchingFiles(t *testing.T) {
	// Create a temp directory for testing
	tmpDir, err := os.MkdirTemp("", "findfiles-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test directory structure
	os.MkdirAll(filepath.Join(tmpDir, "internal", "tui"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "cmd"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "internal", "tui", "model.go"), []byte("package tui"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "internal", "tui", "view.go"), []byte("package tui"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "cmd", "main.go"), []byte("package main"), 0644)

	tests := []struct {
		name        string
		query       string
		expectCount int
		expectFile  string // At least one file should contain this
	}{
		{
			name:        "empty query returns all files",
			query:       "",
			expectCount: 4,
		},
		{
			name:        "query filters files",
			query:       "main",
			expectCount: 2, // main.go and cmd/main.go
			expectFile:  "main.go",
		},
		{
			name:        "query matches path",
			query:       "tui",
			expectCount: 2, // internal/tui/model.go and internal/tui/view.go
			expectFile:  "model.go",
		},
		{
			name:        "query matches view",
			query:       "view",
			expectCount: 1,
			expectFile:  "view.go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := findMatchingFiles(tmpDir, tt.query, 10)

			if len(results) < tt.expectCount {
				t.Errorf("Expected at least %d results for query %q, got %d: %v",
					tt.expectCount, tt.query, len(results), results)
			}

			if tt.expectFile != "" {
				found := false
				for _, r := range results {
					if filepath.Base(r) == tt.expectFile || r == tt.expectFile {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected to find file containing %q in results: %v",
						tt.expectFile, results)
				}
			}
		})
	}
}

func TestFindMatchingFilesExcludesGit(t *testing.T) {
	// Create a temp directory for testing
	tmpDir, err := os.MkdirTemp("", "findfiles-git-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create .git directory (should be excluded)
	os.MkdirAll(filepath.Join(tmpDir, ".git", "objects"), 0755)
	os.WriteFile(filepath.Join(tmpDir, ".git", "config"), []byte("git config"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644)

	results := findMatchingFiles(tmpDir, "", 10)

	for _, r := range results {
		if filepath.Base(filepath.Dir(r)) == ".git" || r == ".git/config" {
			t.Errorf("Expected .git to be excluded, but found: %s", r)
		}
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result (only main.go), got %d: %v", len(results), results)
	}
}

func TestAutocompleteAtPrecededByWhitespace(t *testing.T) {
	ac := NewAutocompleteState("/tmp")

	// @ at start of text should work
	text1 := "@main"
	ac.UpdateFromText(text1, len(text1))
	if !ac.Active {
		t.Error("@ at start of text should activate autocomplete")
	}

	// @ after space should work
	ac.Reset()
	text2 := "See @main"
	ac.UpdateFromText(text2, len(text2))
	if !ac.Active {
		t.Error("@ after space should activate autocomplete")
	}

	// @ in middle of word should NOT work (no activation)
	ac.Reset()
	text3 := "email@domain"
	ac.UpdateFromText(text3, len(text3))
	if ac.Active {
		t.Error("@ in middle of word should NOT activate autocomplete")
	}
}
