package components

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/aspiers/lazydolt/internal/domain"
)

func sampleBranches() []domain.Branch {
	return []domain.Branch{
		{Name: "main", Hash: "abc1234567890", IsCurrent: true, LatestMessage: "initial commit"},
		{Name: "feature", Hash: "def4567890123", IsCurrent: false, LatestMessage: "add feature"},
		{Name: "bugfix", Hash: "789abcdef0123", IsCurrent: false, LatestMessage: "fix bug"},
	}
}

func newBranchesModel(branches []domain.Branch, cursor int, focused bool) BranchesModel {
	return BranchesModel{
		Branches: branches,
		Cursor:   cursor,
		Focused:  focused,
		Height:   20,
	}
}

func TestBranchesModel_NavigateDown(t *testing.T) {
	m := newBranchesModel(sampleBranches(), 0, true)
	m, _ = m.Update(keyMsg("j"))
	if m.Cursor != 1 {
		t.Errorf("cursor = %d, want 1", m.Cursor)
	}
}

func TestBranchesModel_NavigateUp(t *testing.T) {
	m := newBranchesModel(sampleBranches(), 2, true)
	m, _ = m.Update(keyMsg("k"))
	if m.Cursor != 1 {
		t.Errorf("cursor = %d, want 1", m.Cursor)
	}
}

func TestBranchesModel_BoundsCheck(t *testing.T) {
	m := newBranchesModel(sampleBranches(), 2, true)
	m, _ = m.Update(keyMsg("j"))
	if m.Cursor != 2 {
		t.Errorf("cursor = %d, want 2 (should not go past end)", m.Cursor)
	}

	m = newBranchesModel(sampleBranches(), 0, true)
	m, _ = m.Update(keyMsg("k"))
	if m.Cursor != 0 {
		t.Errorf("cursor = %d, want 0 (should not go below 0)", m.Cursor)
	}
}

func TestBranchesModel_IgnoreWhenUnfocused(t *testing.T) {
	m := newBranchesModel(sampleBranches(), 0, false)
	m, _ = m.Update(keyMsg("j"))
	if m.Cursor != 0 {
		t.Errorf("cursor = %d, want 0 (unfocused)", m.Cursor)
	}
}

func TestBranchesModel_CheckoutOnEnter(t *testing.T) {
	m := newBranchesModel(sampleBranches(), 1, true) // cursor on "feature"
	_, cmd := m.Update(specialKeyMsg(tea.KeyEnter))
	msg := execCmd(t, cmd)
	checkoutMsg, ok := msg.(CheckoutBranchMsg)
	if !ok {
		t.Fatalf("expected CheckoutBranchMsg, got %T", msg)
	}
	if checkoutMsg.Branch != "feature" {
		t.Errorf("branch = %q, want %q", checkoutMsg.Branch, "feature")
	}
}

func TestBranchesModel_NewBranchPrompt(t *testing.T) {
	m := newBranchesModel(sampleBranches(), 0, true)
	_, cmd := m.Update(keyMsg("n"))
	msg := execCmd(t, cmd)
	if _, ok := msg.(NewBranchPromptMsg); !ok {
		t.Fatalf("expected NewBranchPromptMsg, got %T", msg)
	}
}

func TestBranchesModel_DeleteBranch(t *testing.T) {
	m := newBranchesModel(sampleBranches(), 1, true) // cursor on "feature"
	_, cmd := m.Update(keyMsg("D"))
	msg := execCmd(t, cmd)
	deleteMsg, ok := msg.(DeleteBranchMsg)
	if !ok {
		t.Fatalf("expected DeleteBranchMsg, got %T", msg)
	}
	if deleteMsg.Branch != "feature" {
		t.Errorf("branch = %q, want %q", deleteMsg.Branch, "feature")
	}
}

func TestBranchesModel_ViewCurrentBranchMarker(t *testing.T) {
	m := newBranchesModel(sampleBranches(), -1, false)
	view := m.View()
	if !strings.Contains(view, "* ") {
		t.Error("View() should contain '* ' prefix for current branch")
	}
}

func TestBranchesModel_ViewEmptyList(t *testing.T) {
	m := newBranchesModel(nil, 0, true)
	if got := m.View(); got != "No branches" {
		t.Errorf("View() = %q, want %q", got, "No branches")
	}
}

func TestBranchesModel_ViewShowsHashPrefix(t *testing.T) {
	m := newBranchesModel(sampleBranches(), -1, false)
	view := m.View()
	// Hash should be truncated to 7 chars
	if !strings.Contains(view, "abc1234") {
		t.Error("View() should contain truncated hash 'abc1234'")
	}
}
