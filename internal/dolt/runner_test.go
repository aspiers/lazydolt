package dolt

import (
	"strings"
	"testing"

	"github.com/aspiers/lazydolt/internal/testutil"
)

func TestStripANSI(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "no escapes",
			input: "plain text",
			want:  "plain text",
		},
		{
			name:  "single color code",
			input: "\x1b[31mred text\x1b[0m",
			want:  "red text",
		},
		{
			name:  "multiple color codes",
			input: "\x1b[1m\x1b[32mbold green\x1b[0m normal",
			want:  "bold green normal",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "only escapes",
			input: "\x1b[31m\x1b[0m",
			want:  "",
		},
		{
			name:  "complex SGR parameters",
			input: "\x1b[38;5;196mextended color\x1b[0m",
			want:  "extended color",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripANSI(tt.input)
			if got != tt.want {
				t.Errorf("stripANSI(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNewRunner_ValidRepo(t *testing.T) {
	repo := testutil.NewDoltTestRepo(t)
	runner, err := NewRunner(repo.Dir)
	if err != nil {
		t.Fatalf("NewRunner(%q): %v", repo.Dir, err)
	}
	if runner.RepoDir != repo.Dir {
		t.Errorf("RepoDir = %q, want %q", runner.RepoDir, repo.Dir)
	}
}

func TestNewRunner_InvalidDir(t *testing.T) {
	_, err := NewRunner("/nonexistent/path/to/nowhere")
	if err == nil {
		t.Error("NewRunner with invalid dir should return error")
	}
}

func TestExec_Version(t *testing.T) {
	runner, _ := newTestRunner(t)
	out, err := runner.Exec("version")
	if err != nil {
		t.Fatalf("Exec(version): %v", err)
	}
	if !strings.Contains(out, "dolt") {
		t.Errorf("Exec(version) = %q, expected to contain 'dolt'", out)
	}
}

func TestSQL_SimpleQuery(t *testing.T) {
	runner, _ := newTestRunner(t)
	rows, err := runner.SQL("SELECT 1 as test")
	if err != nil {
		t.Fatalf("SQL(SELECT 1): %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	// Value might be float64 from JSON parsing
	val, ok := rows[0]["test"]
	if !ok {
		t.Fatal("expected 'test' column in result")
	}
	if v, ok := val.(float64); !ok || v != 1 {
		t.Errorf("test = %v (%T), want 1", val, val)
	}
}

func TestSQL_SyntaxError(t *testing.T) {
	runner, _ := newTestRunner(t)
	_, err := runner.SQL("SELECTX INVALID SYNTAX")
	if err == nil {
		t.Error("SQL with bad syntax should return error")
	}
}
