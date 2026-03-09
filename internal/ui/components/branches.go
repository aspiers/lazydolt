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
	remoteStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("5")) // magenta
)

// BranchesModel displays a navigable list of branches, tags, and remotes.
type BranchesModel struct {
	Branches []domain.Branch
	Tags     []domain.Tag
	Remotes  []domain.Remote
	Cursor   int
	Focused  bool
	Height   int
	HScroll  int    // horizontal scroll offset (columns)
	Filter   string // case-insensitive substring filter
}

// Init is a no-op.
func (m BranchesModel) Init() tea.Cmd { return nil }

// ItemCount returns the total number of items (branches + tags + remotes).
func (m BranchesModel) ItemCount() int {
	return len(m.Branches) + len(m.Tags) + len(m.Remotes)
}

// isTagIndex returns true if the given index refers to a tag.
func (m BranchesModel) isTagIndex(idx int) bool {
	return idx >= len(m.Branches) && idx < len(m.Branches)+len(m.Tags)
}

// isRemoteIndex returns true if the given index refers to a remote.
func (m BranchesModel) isRemoteIndex(idx int) bool {
	return idx >= len(m.Branches)+len(m.Tags)
}

// tagAt returns the tag at the given combined index.
func (m BranchesModel) tagAt(idx int) domain.Tag {
	return m.Tags[idx-len(m.Branches)]
}

// remoteAt returns the remote at the given combined index.
func (m BranchesModel) remoteAt(idx int) domain.Remote {
	return m.Remotes[idx-len(m.Branches)-len(m.Tags)]
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
				return m, func() tea.Msg { return ViewBranchMsg{Branch: b} }
			}
		case " ":
			if b := m.SelectedBranch(); b != "" {
				return m, func() tea.Msg { return CheckoutBranchMsg{Branch: b} }
			}
		case "n":
			return m, func() tea.Msg { return NewBranchPromptMsg{} }
		case "a":
			return m, func() tea.Msg { return AddRemotePromptMsg{} }
		case "D":
			if b := m.SelectedBranch(); b != "" {
				return m, func() tea.Msg { return DeleteBranchMsg{Branch: b} }
			}
			if t := m.SelectedTag(); t != "" {
				return m, func() tea.Msg { return DeleteTagMsg{Tag: t} }
			}
			if r := m.SelectedRemote(); r != "" {
				return m, func() tea.Msg { return DeleteRemoteMsg{Remote: r} }
			}
		}
	}
	return m, nil
}

// View renders the branches, tags, and remotes list, clipped to the visible height.
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

	// Account for section header lines
	height := m.Height
	if len(m.Tags) > 0 {
		height-- // reserve one line for the "▼ Tags" header
	}
	if len(m.Remotes) > 0 {
		height-- // reserve one line for the "▼ Remotes" header
	}
	start, end := visibleRange(cursorPos, len(indices), height)

	var s string
	tagHeaderShown := false
	remoteHeaderShown := false

	for fi := start; fi < end; fi++ {
		i := indices[fi]

		if m.isRemoteIndex(i) {
			if !remoteHeaderShown {
				s += headerStyle.Render("▼ Remotes") + "\n"
				remoteHeaderShown = true
			}
			r := m.remoteAt(i)
			selected := i == m.Cursor && m.Focused
			s += m.renderRemote(r, selected)
		} else if m.isTagIndex(i) {
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

func (m BranchesModel) renderRemote(r domain.Remote, selected bool) string {
	nameStyle := remoteStyle
	urlStyle := hashStyle

	if selected {
		nameStyle = nameStyle.Reverse(true)
		urlStyle = urlStyle.Reverse(true)
		sp := selectedStyle.Render(" ")
		line := selectedStyle.Render("  ") +
			nameStyle.Render(r.Name) + sp +
			urlStyle.Render(r.URL)
		return line + "\n"
	}
	line := fmt.Sprintf("  %s %s", nameStyle.Render(r.Name), urlStyle.Render(r.URL))
	return line + "\n"
}

// MoveUp moves the cursor up by one item in the filtered list.
func (m *BranchesModel) MoveUp() {
	indices := m.filteredIndices()
	for ci, idx := range indices {
		if idx == m.Cursor && ci > 0 {
			m.Cursor = indices[ci-1]
			return
		}
	}
}

// MoveDown moves the cursor down by one item in the filtered list.
func (m *BranchesModel) MoveDown() {
	indices := m.filteredIndices()
	for ci, idx := range indices {
		if idx == m.Cursor && ci+1 < len(indices) {
			m.Cursor = indices[ci+1]
			return
		}
	}
}

// ClickRow sets the cursor to the item at the given visible row offset.
// Accounts for section headers (Tags, Remotes) in the visible output.
func (m *BranchesModel) ClickRow(row int) {
	indices := m.filteredIndices()
	if len(indices) == 0 {
		return
	}
	// Find current cursor position in filtered list
	cursorPos := 0
	for ci, idx := range indices {
		if idx == m.Cursor {
			cursorPos = ci
			break
		}
	}
	// Account for section headers reducing available height
	height := m.Height
	if len(m.Tags) > 0 {
		height--
	}
	if len(m.Remotes) > 0 {
		height--
	}
	start, _ := visibleRange(cursorPos, len(indices), height)

	// Map visual row to filtered index, skipping section header rows
	visualRow := 0
	for fi := start; fi < len(indices); fi++ {
		if visualRow == row {
			m.Cursor = indices[fi]
			return
		}
		visualRow++
	}
}

// SelectedBranch returns the name of the currently selected branch,
// or "" if a tag or remote is selected.
func (m BranchesModel) SelectedBranch() string {
	if m.Cursor >= 0 && m.Cursor < len(m.Branches) {
		return m.Branches[m.Cursor].Name
	}
	return ""
}

// SelectedTag returns the name of the currently selected tag,
// or "" if a branch or remote is selected.
func (m BranchesModel) SelectedTag() string {
	if m.isTagIndex(m.Cursor) {
		return m.tagAt(m.Cursor).Name
	}
	return ""
}

// SelectedRemote returns the name of the currently selected remote,
// or "" if a branch or tag is selected.
func (m BranchesModel) SelectedRemote() string {
	if m.isRemoteIndex(m.Cursor) && m.Cursor < m.ItemCount() {
		return m.remoteAt(m.Cursor).Name
	}
	return ""
}

// matchesFilter returns true if a branch matches the filter.
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

func (m BranchesModel) matchesFilterRemote(r domain.Remote) bool {
	if m.Filter == "" {
		return true
	}
	f := strings.ToLower(m.Filter)
	return strings.Contains(strings.ToLower(r.Name), f) ||
		strings.Contains(strings.ToLower(r.URL), f)
}

// filteredIndices returns the combined indices of branches, tags, and remotes
// matching the filter. Branches are 0..len(Branches)-1, tags are
// len(Branches)..len(Branches)+len(Tags)-1, remotes are
// len(Branches)+len(Tags)..len(Branches)+len(Tags)+len(Remotes)-1.
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
	for i, r := range m.Remotes {
		if m.matchesFilterRemote(r) {
			indices = append(indices, len(m.Branches)+len(m.Tags)+i)
		}
	}
	return indices
}

// Message types for parent to handle.
type CheckoutBranchMsg struct{ Branch string }
type ViewBranchMsg struct{ Branch string }
type NewBranchPromptMsg struct{}
type DeleteBranchMsg struct{ Branch string }
type DeleteTagMsg struct{ Tag string }
type AddRemotePromptMsg struct{}
type DeleteRemoteMsg struct{ Remote string }
