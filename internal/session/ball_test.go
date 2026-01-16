package session

import "testing"

func TestExtractTitleFirstSentence(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "title with period and more content",
			input:    "Lots of context. More stuff",
			expected: "Lots of context",
		},
		{
			name:     "title with multiple periods",
			input:    "First. Second.",
			expected: "First",
		},
		{
			name:     "title without period",
			input:    "No period",
			expected: "No period",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only a period",
			input:    ".",
			expected: "",
		},
		{
			name:     "period at end with no space after",
			input:    "Single sentence.",
			expected: "Single sentence",
		},
		{
			name:     "abbreviation with period but no space",
			input:    "Mr.Smith went to the store",
			expected: "Mr.Smith went to the store",
		},
		{
			name:     "period in middle of word",
			input:    "file.txt is a filename",
			expected: "file.txt is a filename",
		},
		{
			name:     "multiple spaces after period",
			input:    "First sentence.  Second sentence",
			expected: "First sentence",
		},
		{
			name:     "period followed by newline should not extract",
			input:    "First sentence.\nSecond sentence",
			expected: "First sentence.\nSecond sentence",
		},
		{
			name:     "version number with periods",
			input:    "Upgrade to v1.2.3 for new features",
			expected: "Upgrade to v1.2.3 for new features",
		},
		{
			name:     "sentence after version number",
			input:    "Upgrade to v1.2.3. New features include X",
			expected: "Upgrade to v1.2.3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractTitleFirstSentence(tt.input)
			if result != tt.expected {
				t.Errorf("ExtractTitleFirstSentence(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSetTitle(t *testing.T) {
	ball := &Ball{Title: "Original"}

	ball.SetTitle("New title. With more content")

	if ball.Title != "New title" {
		t.Errorf("SetTitle() should extract first sentence, got %q", ball.Title)
	}
}

func TestNewBallExtractsFirstSentence(t *testing.T) {
	tmpDir := t.TempDir()

	ball, err := NewBall(tmpDir, "First sentence. Second sentence.", PriorityMedium)
	if err != nil {
		t.Fatalf("NewBall() error = %v", err)
	}

	if ball.Title != "First sentence" {
		t.Errorf("NewBall() should extract first sentence, got %q", ball.Title)
	}
}
