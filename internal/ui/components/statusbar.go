package components

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	repoStyle  = lipgloss.NewStyle().Bold(true)
	pathStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("8")) // dim
	dirtyStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("3")) // yellow
)

// StatusBar displays repo info and dirty indicator.
type StatusBar struct {
	Dirty     bool
	RepoDir   string
	ParentDir string
	Width     int
}

// Lines returns the number of lines the status bar will render.
func (s StatusBar) Lines() int {
	if s.ParentDir != "" {
		repo := repoStyle.Render(s.RepoDir)
		dirty := ""
		if s.Dirty {
			dirty = dirtyStyle.Render(" *")
		}
		dirText := pathStyle.Render(s.ParentDir)
		combined := repo + dirty + " " + dirText
		if lipgloss.Width(combined) > s.Width {
			return 2 // repo on one line, dir on next
		}
	}
	return 1 // repo (+dir on same line)
}

// View renders the status bar.
func (s StatusBar) View() string {
	repo := repoStyle.Render(s.RepoDir)

	dirty := ""
	if s.Dirty {
		dirty = dirtyStyle.Render(" *")
	}

	// Show parent directory path after repo name.
	// If both fit on one line, join them; otherwise wrap.
	if s.ParentDir != "" {
		dirText := pathStyle.Render(s.ParentDir)
		combined := repo + dirty + " " + dirText
		if lipgloss.Width(combined) <= s.Width {
			return combined
		}
		return repo + dirty + "\n" + dirText
	}

	return repo + dirty
}
