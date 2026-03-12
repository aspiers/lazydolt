package components

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"

	"github.com/aspiers/lazydolt/internal/domain"
)

var (
	repoStyle   = lipgloss.NewStyle().Bold(true)
	pathStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("8")) // dim
	dirtyStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("3")) // yellow
	serverStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("6")) // cyan
)

// StatusBar displays repo info and dirty indicator.
type StatusBar struct {
	Dirty      bool
	RepoDir    string
	ParentDir  string
	ServerInfo *domain.ServerInfo
	Width      int
}

// Lines returns the number of lines the status bar will render.
func (s StatusBar) Lines() int {
	lines := 1 // repo name line
	if s.ParentDir != "" {
		repo := repoStyle.Render(s.RepoDir)
		dirty := ""
		if s.Dirty {
			dirty = dirtyStyle.Render(" *")
		}
		dirText := pathStyle.Render(s.ParentDir)
		combined := repo + dirty + " " + dirText
		if lipgloss.Width(combined) > s.Width {
			lines = 2 // repo on one line, dir on next
		}
	}
	if s.ServerInfo != nil {
		lines++ // server info line
	}
	return lines
}

// serverInfoText returns a human-readable string for the server connection.
func (s StatusBar) serverInfoText() string {
	if s.ServerInfo == nil {
		return ""
	}
	if s.ServerInfo.Pid > 0 {
		return fmt.Sprintf("sql-server :%d (pid %d)", s.ServerInfo.Port, s.ServerInfo.Pid)
	}
	return fmt.Sprintf("sql-server :%d", s.ServerInfo.Port)
}

// View renders the status bar.
func (s StatusBar) View() string {
	repo := repoStyle.Render(s.RepoDir)

	dirty := ""
	if s.Dirty {
		dirty = dirtyStyle.Render(" *")
	}

	var result string
	// Show parent directory path after repo name.
	// If both fit on one line, join them; otherwise wrap.
	if s.ParentDir != "" {
		dirText := pathStyle.Render(s.ParentDir)
		combined := repo + dirty + " " + dirText
		if lipgloss.Width(combined) <= s.Width {
			result = combined
		} else {
			result = repo + dirty + "\n" + dirText
		}
	} else {
		result = repo + dirty
	}

	if info := s.serverInfoText(); info != "" {
		result += "\n" + serverStyle.Render(info)
	}

	return result
}
