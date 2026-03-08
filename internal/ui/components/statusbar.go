package components

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

var (
	repoStyle   = lipgloss.NewStyle().Bold(true)
	pathStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("8")) // dim
	branchStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("6")) // cyan
	dirtyStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("3")) // yellow
)

// StatusBar displays current branch and repo info.
type StatusBar struct {
	Branch    string
	Dirty     bool
	RepoDir   string
	ParentDir string
	Width     int
}

// Lines returns the number of lines the status bar will render.
func (s StatusBar) Lines() int {
	if s.ParentDir != "" {
		repo := repoStyle.Render(s.RepoDir)
		dirText := pathStyle.Render(s.ParentDir)
		combined := repo + " " + dirText
		if lipgloss.Width(combined) > s.Width {
			return 3 // repo, dir (wrapped), branch
		}
	}
	return 2 // repo (+dir on same line), branch
}

// View renders the status bar.
func (s StatusBar) View() string {
	repo := repoStyle.Render(s.RepoDir)

	// Show parent directory path after repo name.
	// If both fit on one line, join them; otherwise wrap.
	var firstLine string
	if s.ParentDir != "" {
		dirText := pathStyle.Render(s.ParentDir)
		combined := repo + " " + dirText
		if lipgloss.Width(combined) <= s.Width {
			firstLine = combined
		} else {
			firstLine = repo + "\n" + dirText
		}
	} else {
		firstLine = repo
	}

	branchLine := "on " + branchStyle.Render(s.Branch)
	if s.Dirty {
		branchLine += dirtyStyle.Render(" *")
	}

	content := fmt.Sprintf("%s\n%s", firstLine, branchLine)
	return content
}
