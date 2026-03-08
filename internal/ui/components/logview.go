package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/aspiers/lazydolt/internal/domain"
)

var (
	logCmdStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")) // cyan
	logErrStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))            // red
	logTimeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))            // dim
	logOkStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))            // green
)

// LogView renders a scrollable log of recent dolt CLI commands.
type LogView struct {
	Viewport viewport.Model
	Entries  []domain.CommandLogEntry
	Ready    bool
}

// NewLogView creates a new log viewer with the given dimensions.
func NewLogView(width, height int) LogView {
	vp := viewport.New(width, height)
	vp.SetContent("No commands executed yet")
	return LogView{
		Viewport: vp,
		Ready:    true,
	}
}

// AddEntry appends a command log entry and refreshes the viewport.
func (l *LogView) AddEntry(entry domain.CommandLogEntry) {
	l.Entries = append(l.Entries, entry)
	l.RefreshContent()
}

// SetSize updates the viewport dimensions.
func (l *LogView) SetSize(width, height int) {
	l.Viewport.Width = width
	l.Viewport.Height = height
}

// Update passes events to the viewport.
func (l LogView) Update(msg tea.Msg) (LogView, tea.Cmd) {
	var cmd tea.Cmd
	l.Viewport, cmd = l.Viewport.Update(msg)
	return l, cmd
}

// View renders the viewport.
func (l LogView) View() string {
	return l.Viewport.View()
}

// RefreshContent rebuilds the viewport content from all entries.
func (l *LogView) RefreshContent() {
	if len(l.Entries) == 0 {
		l.Viewport.SetContent("No commands executed yet")
		return
	}

	var sb strings.Builder
	for i, entry := range l.Entries {
		if i > 0 {
			sb.WriteString("\n")
		}

		timeStr := entry.Time.Format("15:04:05")
		sb.WriteString(logTimeStyle.Render(timeStr))
		sb.WriteString(" ")
		sb.WriteString(logCmdStyle.Render(entry.Command))
		sb.WriteString("\n")

		if entry.Error {
			// Show error output (truncated to a few lines)
			lines := strings.Split(strings.TrimSpace(entry.Output), "\n")
			maxLines := 5
			if len(lines) > maxLines {
				lines = lines[:maxLines]
				lines = append(lines, fmt.Sprintf("  ... (%d more lines)", len(lines)-maxLines))
			}
			for _, line := range lines {
				sb.WriteString(logErrStyle.Render("  " + line))
				sb.WriteString("\n")
			}
		} else {
			sb.WriteString(logOkStyle.Render("  ✓"))
			sb.WriteString("\n")
		}
	}

	l.Viewport.SetContent(sb.String())
	// Auto-scroll to bottom to show latest entries
	l.Viewport.GotoBottom()
}
