package components

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

var (
	repoStyle   = lipgloss.NewStyle().Bold(true)
	branchStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("6")) // cyan
	dirtyStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("3")) // yellow
)

// StatusBar displays current branch and repo info.
type StatusBar struct {
	Branch  string
	Dirty   bool
	RepoDir string
	Width   int
}

// View renders the status bar.
func (s StatusBar) View() string {
	repo := repoStyle.Render(s.RepoDir)

	branchLine := "on " + branchStyle.Render(s.Branch)
	if s.Dirty {
		branchLine += dirtyStyle.Render(" *")
	}

	content := fmt.Sprintf("%s\n%s", repo, branchLine)
	return content
}
