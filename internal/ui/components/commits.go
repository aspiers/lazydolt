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
	commitHashStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("3")) // yellow
	commitAuthorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8")) // dim
	commitDateStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("8")) // dim
)

// CommitsModel displays a scrollable list of commits.
type CommitsModel struct {
	Commits []domain.Commit
	Cursor  int
	Focused bool
	Height  int
	HScroll int // horizontal scroll offset (columns)
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
			if m.Cursor < len(m.Commits)-1 {
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
			if h := m.SelectedHash(); h != "" {
				return m, func() tea.Msg { return ViewCommitMsg{Hash: h} }
			}
		}
	}
	return m, nil
}

// View renders the commits list, clipped to the visible height.
func (m CommitsModel) View() string {
	if len(m.Commits) == 0 {
		return "No commits"
	}

	start, end := visibleRange(m.Cursor, len(m.Commits), m.Height)

	var s string
	for i := start; i < end; i++ {
		c := m.Commits[i]
		selected := i == m.Cursor && m.Focused

		hash := c.Hash
		if len(hash) > 7 {
			hash = hash[:7]
		}

		msg := c.Message
		if len(msg) > 40 {
			msg = msg[:37] + "..."
		}

		var line string
		if selected {
			hStyle := commitHashStyle.Reverse(true)
			aStyle := commitAuthorStyle.Reverse(true)
			dStyle := commitDateStyle.Reverse(true)
			sp := selectedStyle.Render(" ")
			line = hStyle.Render(hash) + sp +
				selectedStyle.Render(msg) + sp +
				aStyle.Render(initials(c.Author)) + sp +
				dStyle.Render(relativeTime(c.Date))
		} else {
			line = fmt.Sprintf("%s %s %s %s",
				commitHashStyle.Render(hash),
				msg,
				commitAuthorStyle.Render(initials(c.Author)),
				commitDateStyle.Render(relativeTime(c.Date)),
			)
		}
		s += line + "\n"
	}

	return HScrollContent(s, m.HScroll)
}

// SelectedHash returns the hash of the currently selected commit, or empty string.
func (m CommitsModel) SelectedHash() string {
	if m.Cursor >= 0 && m.Cursor < len(m.Commits) {
		return m.Commits[m.Cursor].Hash
	}
	return ""
}

// Message types for parent to handle.
type ViewCommitMsg struct{ Hash string }

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
