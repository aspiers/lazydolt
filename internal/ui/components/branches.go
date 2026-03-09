package components

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/aspiers/lazydolt/internal/domain"
)

var (
	currentBranchStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("2")) // green
	hashStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("8")) // dim
	tagStyle           = lipgloss.NewStyle().Foreground(lipgloss.Color("3")) // yellow
)

// BranchesModel displays a navigable list of branches and tags.
type BranchesModel struct {
	Branches []domain.Branch
	Tags     []domain.Tag
	Cursor   int
	Focused  bool
	Height   int
	HScroll  int    // horizontal scroll offset (columns)
	Filter   string // case-insensitive substring filter
}

// Init is a no-op.
func (m BranchesModel) Init() tea.Cmd { return nil }

// ItemCount returns the total number of items (branches + tags).
func (m BranchesModel) ItemCount() int {
	return len(m.Branches) + len(m.Tags)
}

// isTagIndex returns true if the given index refers to a tag.
func (m BranchesModel) isTagIndex(idx int) bool {
	return idx >= len(m.Branches)
}

// tagAt returns the tag at the given combined index.
func (m BranchesModel) tagAt(idx int) domain.Tag {
	return m.Tags[idx-len(m.Branches)]
}

// Update handles key events when focused.
func (m BranchesModel) Update(msg tea.Msg) (BranchesModel, tea.Cmd) {
	if !m.Focused {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			indices := m.filteredIndices()
			for ci, idx := range indices {
				if idx == m.Cursor && ci+1 < len(indices) {
					m.Cursor = indices[ci+1]
					break
				}
			}
		case "k", "up":
			indices := m.filteredIndices()
			for ci, idx := range indices {
				if idx == m.Cursor && ci > 0 {
					m.Cursor = indices[ci-1]
					break
				}
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
			if t := m.SelectedTag(); t != "" {
				return m, func() tea.Msg { return DeleteTagMsg{Tag: t} }
			}
		}
	}
	return m, nil
}

// View renders the branches and tags list, clipped to the visible height.
func (m BranchesModel) View() string {
	indices := m.filteredIndices()
	if len(indices) == 0 {
		if m.Filter != "" {
			return "No matching branches"
		}
		return "No branches"
	}

	// Find cursor position within filtered list
	cursorPos := 0
	for i, idx := range indices {
		if idx == m.Cursor {
			cursorPos = i
			break
		}
	}

	// Account for tags header line if tags are present in the visible range
	height := m.Height
	if len(m.Tags) > 0 {
		height-- // reserve one line for the "▼ Tags" header
	}
	start, end := visibleRange(cursorPos, len(indices), height)

	var s string
	// Track whether we've rendered the tags header
	tagHeaderShown := false

	for fi := start; fi < end; fi++ {
		i := indices[fi]

		if m.isTagIndex(i) {
			// Show tags header before first tag
			if !tagHeaderShown {
				s += headerStyle.Render("▼ Tags") + "\n"
				tagHeaderShown = true
			}
			t := m.tagAt(i)
			selected := i == m.Cursor && m.Focused
			s += m.renderTag(t, selected)
		} else {
			b := m.Branches[i]
			selected := i == m.Cursor && m.Focused
			s += m.renderBranch(b, selected)
		}
	}

	return HScrollContent(s, m.HScroll)
}

func (m BranchesModel) renderBranch(b domain.Branch, selected bool) string {
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
		return line + "\n"
	}
	line := fmt.Sprintf("%s%s %s %s", prefix, nameStyle.Render(b.Name), hash, hashStyle.Render(msg))
	return line + "\n"
}

func (m BranchesModel) renderTag(t domain.Tag, selected bool) string {
	nameStyle := tagStyle
	hash := ""
	hStyle := hashStyle
	if len(t.Hash) >= 7 {
		if selected {
			hStyle = hStyle.Reverse(true)
		}
		hash = hStyle.Render(t.Hash[:7])
	}

	msg := t.Message
	if len(msg) > 30 {
		msg = msg[:27] + "..."
	}

	if selected {
		nameStyle = nameStyle.Reverse(true)
		sp := selectedStyle.Render(" ")
		line := selectedStyle.Render("  ") +
			nameStyle.Render(t.Name) + sp +
			hash + sp +
			hashStyle.Reverse(true).Render(msg)
		return line + "\n"
	}
	line := fmt.Sprintf("  %s %s %s", nameStyle.Render(t.Name), hash, hashStyle.Render(msg))
	return line + "\n"
}

// SelectedBranch returns the name of the currently selected branch,
// or "" if a tag is selected.
func (m BranchesModel) SelectedBranch() string {
	if m.Cursor >= 0 && m.Cursor < len(m.Branches) {
		return m.Branches[m.Cursor].Name
	}
	return ""
}

// SelectedTag returns the name of the currently selected tag,
// or "" if a branch is selected.
func (m BranchesModel) SelectedTag() string {
	if m.isTagIndex(m.Cursor) && m.Cursor < m.ItemCount() {
		return m.tagAt(m.Cursor).Name
	}
	return ""
}

// matchesFilter returns true if an item (branch or tag) matches the filter.
func (m BranchesModel) matchesFilterBranch(b domain.Branch) bool {
	if m.Filter == "" {
		return true
	}
	f := strings.ToLower(m.Filter)
	return strings.Contains(strings.ToLower(b.Name), f) ||
		strings.Contains(strings.ToLower(b.LatestMessage), f)
}

func (m BranchesModel) matchesFilterTag(t domain.Tag) bool {
	if m.Filter == "" {
		return true
	}
	f := strings.ToLower(m.Filter)
	return strings.Contains(strings.ToLower(t.Name), f) ||
		strings.Contains(strings.ToLower(t.Message), f)
}

// filteredIndices returns the combined indices of branches and tags
// matching the filter. Branches are 0..len(Branches)-1, tags are
// len(Branches)..len(Branches)+len(Tags)-1.
func (m BranchesModel) filteredIndices() []int {
	var indices []int
	for i, b := range m.Branches {
		if m.matchesFilterBranch(b) {
			indices = append(indices, i)
		}
	}
	for i, t := range m.Tags {
		if m.matchesFilterTag(t) {
			indices = append(indices, len(m.Branches)+i)
		}
	}
	return indices
}

// Message types for parent to handle.
type CheckoutBranchMsg struct{ Branch string }
type NewBranchPromptMsg struct{}
type DeleteBranchMsg struct{ Branch string }
type DeleteTagMsg struct{ Tag string }
