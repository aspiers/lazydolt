package components

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func init() {
	// Force ANSI color output in tests (no TTY detection).
	lipgloss.SetColorProfile(termenv.ANSI)
}

func TestColorizeDiff(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string // text must be preserved
		styled   bool   // output should differ from input (styling applied)
	}{
		{
			name:     "added row with + marker",
			input:    "| + | 1 | Alice |",
			contains: "| + | 1 | Alice |",
			styled:   true,
		},
		{
			name:     "deleted row with - marker",
			input:    "| - | 1 | Alice |",
			contains: "| - | 1 | Alice |",
			styled:   true,
		},
		{
			name:     "old value with < marker",
			input:    "| < | 1 | OldName |",
			contains: "| < | 1 | OldName |",
			styled:   true,
		},
		{
			name:     "new value with > marker",
			input:    "| > | 1 | NewName |",
			contains: "| > | 1 | NewName |",
			styled:   true,
		},
		{
			name:     "diff header",
			input:    "diff --dolt a/users b/users",
			contains: "diff --dolt a/users b/users",
			styled:   true,
		},
		{
			name:     "table border",
			input:    "+---+-------+",
			contains: "+---+-------+",
			styled:   true,
		},
		{
			name:     "plain line no style",
			input:    "| 1 | Alice |",
			contains: "| 1 | Alice |",
			styled:   false,
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
			got := colorizeDiff(tt.input)
			if !strings.Contains(got, tt.contains) {
				t.Errorf("colorizeDiff(%q) = %q, should contain %q", tt.input, got, tt.contains)
			}
			if tt.styled && got == tt.input {
				t.Errorf("colorizeDiff(%q) returned unstyled output, expected styling", tt.input)
			}
			if !tt.styled && got != tt.input {
				t.Errorf("colorizeDiff(%q) = %q, expected no styling (should equal input)", tt.input, got)
			}
		})
	}
}

func TestColorizeDiff_MultiLine(t *testing.T) {
	input := "diff --dolt a/users b/users\n| + | 1 | Alice |\n| 1 | Bob |"
	got := colorizeDiff(input)
	lines := strings.Split(got, "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	// First line (header) should be styled
	if lines[0] == "diff --dolt a/users b/users" {
		t.Error("header line should have styling applied")
	}
	// Second line (added) should be styled
	if lines[1] == "| + | 1 | Alice |" {
		t.Error("added line should have styling applied")
	}
	// Third line (plain) should not be styled
	if lines[2] != "| 1 | Bob |" {
		t.Errorf("plain line should not be styled, got %q", lines[2])
	}
}
