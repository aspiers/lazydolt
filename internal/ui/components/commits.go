package components

import (
	"fmt"
	"time"

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
		case "enter":
			if h := m.selectedHash(); h != "" {
				return m, func() tea.Msg { return ViewCommitMsg{Hash: h} }
			}
		}
	}
	return m, nil
}

// View renders the commits list.
func (m CommitsModel) View() string {
	if len(m.Commits) == 0 {
		return "No commits"
	}

	var s string
	for i, c := range m.Commits {
		hash := c.Hash
		if len(hash) > 7 {
			hash = hash[:7]
		}

		msg := c.Message
		if len(msg) > 40 {
			msg = msg[:37] + "..."
		}

		line := fmt.Sprintf("%s %s %s %s",
			commitHashStyle.Render(hash),
			msg,
			commitAuthorStyle.Render(c.Author),
			commitDateStyle.Render(relativeTime(c.Date)),
		)
		if i == m.Cursor && m.Focused {
			line = selectedStyle.Render(line)
		}
		s += line + "\n"
	}

	return s
}

func (m CommitsModel) selectedHash() string {
	if m.Cursor >= 0 && m.Cursor < len(m.Commits) {
		return m.Commits[m.Cursor].Hash
	}
	return ""
}

// Message types for parent to handle.
type ViewCommitMsg struct{ Hash string }

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
