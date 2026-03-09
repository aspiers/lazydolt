package components

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/aspiers/lazydolt/internal/domain"
)

var (
	commitHashStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("3")) // yellow
	commitDateStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8")) // dim

	// authorColors are the ANSI colors used for per-author initials,
	// matching lazygit's approach of deterministic color per author.
	authorColors = []lipgloss.Color{"1", "2", "3", "4", "5", "6", "9", "10", "11", "12", "13", "14"}
)

// CommitsModel displays a scrollable list of commits.
type CommitsModel struct {
	Commits []domain.Commit
	Cursor  int
	Focused bool
	Height  int
	HScroll int    // horizontal scroll offset (columns)
	Filter  string // case-insensitive substring filter
}

// Init is a no-op.
func (m CommitsModel) Init() tea.Cmd { return nil }

// Update handles key events when focused.
func (m CommitsModel) Update(msg tea.Msg) (CommitsModel, tea.Cmd) {
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
			if h := m.SelectedHash(); h != "" {
				return m, func() tea.Msg { return ViewCommitMsg{Hash: h} }
			}
		}
	}
	return m, nil
}

// View renders the commits list, clipped to the visible height.
func (m CommitsModel) View() string {
	indices := m.filteredIndices()
	if len(indices) == 0 {
		if m.Filter != "" {
			return "No matching commits"
		}
		return "No commits"
	}

	cursorPos := 0
	for i, idx := range indices {
		if idx == m.Cursor {
			cursorPos = i
			break
		}
	}

	start, end := visibleRange(cursorPos, len(indices), m.Height)

	// Build graph for all commits (not just visible ones) to get
	// correct lane tracking, then use only the visible slice.
	graphLines := BuildGraph(m.Commits)
	maxGW := MaxGraphWidth(graphLines)

	var s string
	for fi := start; fi < end; fi++ {
		i := indices[fi]
		c := m.Commits[i]
		selected := i == m.Cursor && m.Focused

		// Graph prefix (pad to consistent width)
		graphPrefix := ""
		if i < len(graphLines) {
			gw := lipgloss.Width(graphLines[i].Text)
			graphPrefix = graphLines[i].Text + strings.Repeat(" ", maxGW-gw) + " "
		}

		hash := c.Hash
		if len(hash) > 7 {
			hash = hash[:7]
		}

		msg := c.Message
		if len(msg) > 40 {
			msg = msg[:37] + "..."
		}

		authorInit := initials(c.Author)
		authorStyle := authorColorStyle(c.Author)

		var line string
		if selected {
			hStyle := commitHashStyle.Reverse(true)
			aStyle := authorStyle.Reverse(true)
			dStyle := commitDateStyle.Reverse(true)
			sp := selectedStyle.Render(" ")
			line = graphPrefix + hStyle.Render(hash) + sp +
				aStyle.Render(authorInit) + sp +
				selectedStyle.Render(msg) + sp +
				dStyle.Render(relativeTime(c.Date))
		} else {
			line = graphPrefix + fmt.Sprintf("%s %s %s %s",
				commitHashStyle.Render(hash),
				authorStyle.Render(authorInit),
				msg,
				commitDateStyle.Render(relativeTime(c.Date)),
			)
		}
		s += line + "\n"
	}

	return HScrollContent(s, m.HScroll)
}

// MoveUp moves the cursor up by one item in the filtered list.
func (m *CommitsModel) MoveUp() {
	indices := m.filteredIndices()
	for ci, idx := range indices {
		if idx == m.Cursor && ci > 0 {
			m.Cursor = indices[ci-1]
			return
		}
	}
}

// MoveDown moves the cursor down by one item in the filtered list.
func (m *CommitsModel) MoveDown() {
	indices := m.filteredIndices()
	for ci, idx := range indices {
		if idx == m.Cursor && ci+1 < len(indices) {
			m.Cursor = indices[ci+1]
			return
		}
	}
}

// ClickRow sets the cursor to the item at the given visible row offset.
func (m *CommitsModel) ClickRow(row int) {
	indices := m.filteredIndices()
	if len(indices) == 0 {
		return
	}
	cursorPos := 0
	for ci, idx := range indices {
		if idx == m.Cursor {
			cursorPos = ci
			break
		}
	}
	start, _ := visibleRange(cursorPos, len(indices), m.Height)
	idx := start + row
	if idx >= 0 && idx < len(indices) {
		m.Cursor = indices[idx]
	}
}

// SelectedHash returns the hash of the currently selected commit, or empty string.
func (m CommitsModel) SelectedHash() string {
	if m.Cursor >= 0 && m.Cursor < len(m.Commits) {
		return m.Commits[m.Cursor].Hash
	}
	return ""
}

// matchesFilter returns true if a commit matches the current filter.
func (m CommitsModel) matchesFilter(c domain.Commit) bool {
	if m.Filter == "" {
		return true
	}
	f := strings.ToLower(m.Filter)
	return strings.Contains(strings.ToLower(c.Message), f) ||
		strings.Contains(strings.ToLower(c.Author), f) ||
		strings.Contains(strings.ToLower(c.Hash), f)
}

// filteredIndices returns the indices of commits matching the filter.
func (m CommitsModel) filteredIndices() []int {
	var indices []int
	for i, c := range m.Commits {
		if m.matchesFilter(c) {
			indices = append(indices, i)
		}
	}
	return indices
}

// Message types for parent to handle.
type ViewCommitMsg struct{ Hash string }

// authorColorStyle returns a lipgloss style with a deterministic foreground
// color derived from the author name, so each committer gets a consistent
// color across the list.
func authorColorStyle(name string) lipgloss.Style {
	var h uint32
	for _, c := range name {
		h = h*31 + uint32(c)
	}
	return lipgloss.NewStyle().Foreground(authorColors[h%uint32(len(authorColors))])
}

// initials extracts uppercase initials from a name, e.g.
// "Adam Spiers" → "AS", "alice" → "A".
func initials(name string) string {
	parts := strings.Fields(name)
	if len(parts) == 0 {
		return "?"
	}
	var b strings.Builder
	for _, p := range parts {
		r, _ := utf8.DecodeRuneInString(p)
		if r != utf8.RuneError {
			b.WriteRune(r)
		}
	}
	result := strings.ToUpper(b.String())
	if result == "" {
		return "?"
	}
	return result
}

// relativeTime formats a time as a human-readable relative string.
func relativeTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}

	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	case d < 365*24*time.Hour:
		return fmt.Sprintf("%dmo ago", int(d.Hours()/(24*30)))
	default:
		return fmt.Sprintf("%dy ago", int(d.Hours()/(24*365)))
	}
}
