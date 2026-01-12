package tui

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// AutocompleteState tracks file autocomplete suggestions
type AutocompleteState struct {
	Active      bool     // Whether autocomplete popup is visible
	Query       string   // Current search query (text after @)
	QueryStart  int      // Position of @ in input
	Suggestions []string // Matching file paths
	Selected    int      // Currently selected suggestion index
	RepoRoot    string   // Root directory of the repository
}

// NewAutocompleteState creates a new autocomplete state
func NewAutocompleteState(repoRoot string) *AutocompleteState {
	return &AutocompleteState{
		RepoRoot: repoRoot,
	}
}

// Reset clears the autocomplete state
func (a *AutocompleteState) Reset() {
	a.Active = false
	a.Query = ""
	a.QueryStart = 0
	a.Suggestions = nil
	a.Selected = 0
}

// UpdateFromText checks the input text for @ triggers and updates state
// Returns true if autocomplete state changed
func (a *AutocompleteState) UpdateFromText(text string, cursorPos int) bool {
	// Find the most recent @ before cursor position
	lastAt := -1
	for i := cursorPos - 1; i >= 0; i-- {
		if text[i] == '@' {
			lastAt = i
			break
		}
		// Stop at whitespace - @ must be directly followed by query
		if text[i] == ' ' || text[i] == '\t' || text[i] == '\n' {
			break
		}
	}

	if lastAt == -1 {
		// No @ found - deactivate if was active
		if a.Active {
			a.Reset()
			return true
		}
		return false
	}

	// Extract query after @
	query := text[lastAt+1 : cursorPos]

	// Don't activate if @ is preceded by non-whitespace (unless at start)
	if lastAt > 0 && text[lastAt-1] != ' ' && text[lastAt-1] != '\t' && text[lastAt-1] != '\n' {
		if a.Active {
			a.Reset()
			return true
		}
		return false
	}

	// Update state
	wasActive := a.Active
	oldQuery := a.Query

	a.Active = true
	a.Query = query
	a.QueryStart = lastAt

	// Refresh suggestions if query changed
	if query != oldQuery {
		a.RefreshSuggestions()
		a.Selected = 0
	}

	return !wasActive || query != oldQuery
}

// RefreshSuggestions updates the suggestions based on current query
func (a *AutocompleteState) RefreshSuggestions() {
	a.Suggestions = findMatchingFiles(a.RepoRoot, a.Query, 10)
}

// SelectNext moves selection to next suggestion
func (a *AutocompleteState) SelectNext() {
	if len(a.Suggestions) > 0 {
		a.Selected = (a.Selected + 1) % len(a.Suggestions)
	}
}

// SelectPrev moves selection to previous suggestion
func (a *AutocompleteState) SelectPrev() {
	if len(a.Suggestions) > 0 {
		a.Selected = (a.Selected - 1 + len(a.Suggestions)) % len(a.Suggestions)
	}
}

// GetSelectedSuggestion returns the currently selected suggestion
func (a *AutocompleteState) GetSelectedSuggestion() string {
	if a.Selected >= 0 && a.Selected < len(a.Suggestions) {
		return a.Suggestions[a.Selected]
	}
	return ""
}

// ApplyCompletion returns the new text with the selected suggestion applied
// The @ and query are replaced with the full path
func (a *AutocompleteState) ApplyCompletion(text string) string {
	if !a.Active || len(a.Suggestions) == 0 {
		return text
	}

	selected := a.GetSelectedSuggestion()
	if selected == "" {
		return text
	}

	// Replace @query with the selected path
	before := text[:a.QueryStart]
	after := text[a.QueryStart+1+len(a.Query):]

	return before + selected + after
}

// Deactivate hides the autocomplete without applying
func (a *AutocompleteState) Deactivate() {
	a.Active = false
}

// findMatchingFiles searches for files matching the query in the repo
func findMatchingFiles(repoRoot, query string, maxResults int) []string {
	if repoRoot == "" {
		return nil
	}

	query = strings.ToLower(query)
	var matches []string

	// Walk the repository, excluding common non-relevant directories
	excludeDirs := map[string]bool{
		".git":         true,
		"node_modules": true,
		"vendor":       true,
		".juggler":     true,
		"__pycache__":  true,
		".venv":        true,
		"venv":         true,
		".cache":       true,
	}

	err := filepath.Walk(repoRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip paths with errors
		}

		// Skip excluded directories
		if info.IsDir() {
			if excludeDirs[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		// Get relative path from repo root
		relPath, err := filepath.Rel(repoRoot, path)
		if err != nil {
			return nil
		}

		// Check if path matches query (case-insensitive substring match)
		if query == "" || strings.Contains(strings.ToLower(relPath), query) {
			matches = append(matches, relPath)
		}

		// Stop early if we have enough matches for display
		// But continue for better sorting
		if len(matches) > maxResults*10 {
			return filepath.SkipAll
		}

		return nil
	})

	if err != nil {
		return nil
	}

	// Sort matches: prefer shorter paths and prefix matches
	sort.Slice(matches, func(i, j int) bool {
		mi := strings.ToLower(matches[i])
		mj := strings.ToLower(matches[j])

		// Prefix matches first
		iPrefixMatch := strings.HasPrefix(mi, query) || strings.HasPrefix(filepath.Base(mi), query)
		jPrefixMatch := strings.HasPrefix(mj, query) || strings.HasPrefix(filepath.Base(mj), query)
		if iPrefixMatch != jPrefixMatch {
			return iPrefixMatch
		}

		// Shorter paths next
		if len(matches[i]) != len(matches[j]) {
			return len(matches[i]) < len(matches[j])
		}

		// Alphabetical
		return matches[i] < matches[j]
	})

	// Limit results
	if len(matches) > maxResults {
		matches = matches[:maxResults]
	}

	return matches
}
