package components

import (
	"strings"
	"testing"
)

// Note: init() in diffview_test.go sets lipgloss.SetColorProfile(termenv.ANSI)
// which applies to all tests in this package.

func TestHighlightSQL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string
		styled   bool
	}{
		{
			name:     "CREATE TABLE keywords",
			input:    "CREATE TABLE users (",
			contains: "CREATE",
			styled:   true,
		},
		{
			name:     "type keywords",
			input:    "  id INT PRIMARY KEY,",
			contains: "INT",
			styled:   true,
		},
		{
			name:     "backtick identifiers",
			input:    "  `user_name` VARCHAR(100),",
			contains: "`user_name`",
			styled:   true,
		},
		{
			name:     "no keywords - comment",
			input:    "-- this is a comment",
			contains: "-- this is a comment",
			styled:   false,
		},
		{
			name:     "case insensitive",
			input:    "create table USERS",
			contains: "create",
			styled:   true,
		},
		{
			name:     "empty input",
			input:    "",
			contains: "",
			styled:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := highlightSQL(tt.input)
			if !strings.Contains(got, tt.contains) {
				t.Errorf("highlightSQL(%q) = %q, should contain %q", tt.input, got, tt.contains)
			}
			if tt.styled && got == tt.input {
				t.Errorf("highlightSQL(%q) returned unstyled output, expected styling", tt.input)
			}
			if !tt.styled && got != tt.input {
				t.Errorf("highlightSQL(%q) = %q, expected no styling", tt.input, got)
			}
		})
	}
}

func TestHighlightSQL_MultiLine(t *testing.T) {
	input := "CREATE TABLE `users` (\n  `id` INT PRIMARY KEY,\n  `name` VARCHAR(100)\n);"
	got := highlightSQL(input)
	lines := strings.Split(got, "\n")
	if len(lines) != 4 {
		t.Fatalf("expected 4 lines, got %d", len(lines))
	}
	// Each line with keywords should be styled
	for i, line := range lines[:3] {
		original := strings.Split(input, "\n")[i]
		if line == original {
			t.Errorf("line %d should be styled: %q", i, line)
		}
	}
}
