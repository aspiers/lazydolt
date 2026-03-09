package components

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/golden"

	"github.com/aspiers/lazydolt/internal/domain"
)

func sampleCommits() []domain.Commit {
	return []domain.Commit{
		{Hash: "abc1234567890abcdef1234567890abcdef123456", Message: "Add user authentication", Author: "Alice", Date: time.Now().Add(-1 * time.Hour)},
		{Hash: "def4567890123abcdef4567890123abcdef456789", Message: "Fix login bug that caused crashes on mobile devices when using biometrics", Author: "Bob", Date: time.Now().Add(-24 * time.Hour)},
		{Hash: "789abcdef0123456789abcdef0123456789abcdef", Message: "Initial commit", Author: "Charlie", Date: time.Now().Add(-72 * time.Hour)},
	}
}

func newCommitsModel(commits []domain.Commit, cursor int, focused bool) CommitsModel {
	return CommitsModel{
		Commits: commits,
		Cursor:  cursor,
		Focused: focused,
		Height:  20,
	}
}

func TestCommitsModel_NavigateDown(t *testing.T) {
	m := newCommitsModel(sampleCommits(), 0, true)
	m, _ = m.Update(keyMsg("j"))
	if m.Cursor != 1 {
		t.Errorf("cursor = %d, want 1", m.Cursor)
	}
}

func TestCommitsModel_NavigateUp(t *testing.T) {
	m := newCommitsModel(sampleCommits(), 2, true)
	m, _ = m.Update(keyMsg("k"))
	if m.Cursor != 1 {
		t.Errorf("cursor = %d, want 1", m.Cursor)
	}
}

func TestCommitsModel_BoundsCheck(t *testing.T) {
	m := newCommitsModel(sampleCommits(), 2, true)
	m, _ = m.Update(keyMsg("j"))
	if m.Cursor != 2 {
		t.Errorf("cursor = %d, want 2", m.Cursor)
	}

	m = newCommitsModel(sampleCommits(), 0, true)
	m, _ = m.Update(keyMsg("k"))
	if m.Cursor != 0 {
		t.Errorf("cursor = %d, want 0", m.Cursor)
	}
}

func TestCommitsModel_IgnoreWhenUnfocused(t *testing.T) {
	m := newCommitsModel(sampleCommits(), 0, false)
	m, _ = m.Update(keyMsg("j"))
	if m.Cursor != 0 {
		t.Errorf("cursor = %d, want 0 (unfocused)", m.Cursor)
	}
}

func TestCommitsModel_EnterSendsViewCommit(t *testing.T) {
	m := newCommitsModel(sampleCommits(), 1, true) // cursor on second commit
	_, cmd := m.Update(specialKeyMsg(tea.KeyEnter))
	msg := execCmd(t, cmd)
	viewMsg, ok := msg.(ViewCommitMsg)
	if !ok {
		t.Fatalf("expected ViewCommitMsg, got %T", msg)
	}
	if viewMsg.Hash != sampleCommits()[1].Hash {
		t.Errorf("hash = %q, want %q", viewMsg.Hash, sampleCommits()[1].Hash)
	}
}

func TestCommitsModel_ViewTruncatesHash(t *testing.T) {
	m := newCommitsModel(sampleCommits(), -1, false)
	view := m.View()
	// Full 40-char hash should NOT appear
	if strings.Contains(view, sampleCommits()[0].Hash) {
		t.Error("View() should truncate hash, not show full 40 chars")
	}
	// First 7 chars should appear
	if !strings.Contains(view, sampleCommits()[0].Hash[:7]) {
		t.Error("View() should show first 7 chars of hash")
	}
}

func TestCommitsModel_ViewTruncatesMessage(t *testing.T) {
	m := newCommitsModel(sampleCommits(), -1, false)
	view := m.View()
	// Long message (commit 1) should be truncated with "..."
	if !strings.Contains(view, "...") {
		t.Error("View() should truncate long messages with '...'")
	}
	// Short message (commit 0) should appear in full
	if !strings.Contains(view, "Add user authentication") {
		t.Error("View() should show short messages in full")
	}
}

func TestCommitsModel_ViewEmptyList(t *testing.T) {
	m := newCommitsModel(nil, 0, true)
	golden.RequireEqual(t, m.View())
}

func TestInitials(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"Adam Spiers", "AS"},
		{"alice", "A "},
		{"Bob", "B "},
		{"Charlie Brown Jr", "CB"},
		{"", "? "},
		{"  ", "? "},
		{"A B", "AB"},
	}
	for _, tt := range tests {
		got := initials(tt.name)
		if got != tt.want {
			t.Errorf("initials(%q) = %q, want %q", tt.name, got, tt.want)
		}
		if len(got) != 2 {
			t.Errorf("initials(%q) length = %d, want 2", tt.name, len(got))
		}
	}
}
