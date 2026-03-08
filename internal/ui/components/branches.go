package components

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/aspiers/lazydolt/internal/domain"
)

var (
	currentBranchStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("2")) // green
	hashStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("8")) // dim
)

// BranchesModel displays a navigable list of branches.
type BranchesModel struct {
	Branches []domain.Branch
	Cursor   int
	Focused  bool
	Height   int
	HScroll  int // horizontal scroll offset (columns)
}

// Init is a no-op.
func (m BranchesModel) Init() tea.Cmd { return nil }

// Update handles key events when focused.
func (m BranchesModel) Update(msg tea.Msg) (BranchesModel, tea.Cmd) {
	if !m.Focused {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if m.Cursor < len(m.Branches)-1 {
				m.Cursor++
			}
		case "k", "up":
			if m.Cursor > 0 {
				m.Cursor--
			}
		case "H":
			m.HScroll -= HScrollStep
			if m.HScroll < 0 {
				m.HScroll = 0
			}
		case "L":
			m.HScroll += HScrollStep
		case "enter":
			if b := m.SelectedBranch(); b != "" {
				return m, func() tea.Msg { return CheckoutBranchMsg{Branch: b} }
			}
		case "n":
			return m, func() tea.Msg { return NewBranchPromptMsg{} }
		case "D":
			if b := m.SelectedBranch(); b != "" {
				return m, func() tea.Msg { return DeleteBranchMsg{Branch: b} }
			}
		}
	}
	return m, nil
}

// View renders the branches list, clipped to the visible height.
func (m BranchesModel) View() string {
	if len(m.Branches) == 0 {
		return "No branches"
	}

	start, end := visibleRange(m.Cursor, len(m.Branches), m.Height)

	var s string
	for i := start; i < end; i++ {
		b := m.Branches[i]
		selected := i == m.Cursor && m.Focused

		prefix := "  "
		nameStyle := normalStyle
		if b.IsCurrent {
			prefix = "* "
			nameStyle = currentBranchStyle
		}

		hash := ""
		hStyle := hashStyle
		if len(b.Hash) >= 7 {
			if selected {
				hStyle = hStyle.Reverse(true)
			}
			hash = hStyle.Render(b.Hash[:7])
		}

		msg := b.LatestMessage
		if len(msg) > 30 {
			msg = msg[:27] + "..."
		}

		if selected {
			nameStyle = nameStyle.Reverse(true)
			sp := selectedStyle.Render(" ")
			line := selectedStyle.Render(prefix) +
				nameStyle.Render(b.Name) + sp +
				hash + sp +
				hashStyle.Reverse(true).Render(msg)
			s += line + "\n"
		} else {
			line := fmt.Sprintf("%s%s %s %s", prefix, nameStyle.Render(b.Name), hash, hashStyle.Render(msg))
			s += line + "\n"
		}
	}

	return HScrollContent(s, m.HScroll)
}

// SelectedBranch returns the name of the currently selected branch.
func (m BranchesModel) SelectedBranch() string {
	if m.Cursor >= 0 && m.Cursor < len(m.Branches) {
		return m.Branches[m.Cursor].Name
	}
	return ""
}

// Message types for parent to handle.
type CheckoutBranchMsg struct{ Branch string }
type NewBranchPromptMsg struct{}
type DeleteBranchMsg struct{ Branch string }
