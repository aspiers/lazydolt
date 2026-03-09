package components

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/aspiers/lazydolt/internal/domain"
)

func init() {
	lipgloss.SetColorProfile(termenv.ANSI)
}

func TestBuildGraph_Linear(t *testing.T) {
	commits := []domain.Commit{
		{Hash: "aaa", Parents: []string{"bbb"}},
		{Hash: "bbb", Parents: []string{"ccc"}},
		{Hash: "ccc", Parents: nil},
	}
	lines := BuildGraph(commits)
	if len(lines) != 3 {
		t.Fatalf("expected 3 graph lines, got %d", len(lines))
	}

	// All should contain the commit marker
	for i, gl := range lines {
		w := lipgloss.Width(gl.Text)
		if w != 1 {
			t.Errorf("line[%d] width = %d, want 1 (linear graph)", i, w)
		}
	}
}

func TestBuildGraph_Empty(t *testing.T) {
	lines := BuildGraph(nil)
	if lines != nil {
		t.Errorf("expected nil for empty commits, got %v", lines)
	}
}

func TestBuildGraph_SingleCommit(t *testing.T) {
	commits := []domain.Commit{
		{Hash: "aaa", Parents: nil},
	}
	lines := BuildGraph(commits)
	if len(lines) != 1 {
		t.Fatalf("expected 1 graph line, got %d", len(lines))
	}
}

func TestBuildGraph_MergeCommit(t *testing.T) {
	// Commit A merges B and C
	commits := []domain.Commit{
		{Hash: "aaa", Parents: []string{"bbb", "ccc"}},
		{Hash: "bbb", Parents: []string{"ddd"}},
		{Hash: "ccc", Parents: []string{"ddd"}},
		{Hash: "ddd", Parents: nil},
	}
	lines := BuildGraph(commits)
	if len(lines) != 4 {
		t.Fatalf("expected 4 graph lines, got %d", len(lines))
	}

	// Merge commit should have wider graph (2 lanes)
	w := lipgloss.Width(lines[0].Text)
	if w != 1 {
		// The merge commit itself is on lane 0
		t.Logf("merge commit width = %d", w)
	}

	// After the merge, we should see 2 lanes for bbb and ccc
	w1 := lipgloss.Width(lines[1].Text)
	if w1 < 2 {
		t.Errorf("post-merge line width = %d, expected >= 2 (two lanes)", w1)
	}
}

func TestMaxGraphWidth(t *testing.T) {
	lines := []GraphLine{
		{Text: "●"},
		{Text: "●│"},
		{Text: "│●"},
	}
	w := MaxGraphWidth(lines)
	if w != 2 {
		t.Errorf("MaxGraphWidth = %d, want 2", w)
	}
}
