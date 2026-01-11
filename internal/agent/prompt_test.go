package agent

import (
	"strings"
	"testing"
)

func TestGetPromptTemplate(t *testing.T) {
	template := GetPromptTemplate()

	// Template should not be empty
	if template == "" {
		t.Error("GetPromptTemplate() returned empty string")
	}

	// Template should contain key sections
	requiredSections := []string{
		"Juggler Agent Instructions",
		"<promise>COMPLETE</promise>",
		"<promise>BLOCKED:",
		"juggle update",
		"juggle progress append",
		"ONE BALL PER ITERATION",
	}

	for _, section := range requiredSections {
		if !strings.Contains(template, section) {
			t.Errorf("Template missing required content: %q", section)
		}
	}
}

func TestPromptTemplateVariable(t *testing.T) {
	// PromptTemplate should be the same as GetPromptTemplate()
	if PromptTemplate != GetPromptTemplate() {
		t.Error("PromptTemplate variable differs from GetPromptTemplate()")
	}
}
