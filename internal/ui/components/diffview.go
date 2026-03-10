package components

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// newViewport creates a viewport with the project's customized keymap.
// Adds backspace as an additional page-up key alongside pgup/b.
func newViewport(width, height int) viewport.Model {
	vp := viewport.New(width, height)
	km := viewport.DefaultKeyMap()
	km.PageUp = key.NewBinding(
		key.WithKeys("pgup", "b", "backspace"),
		key.WithHelp("b/pgup", "page up"),
	)
	vp.KeyMap = km
	vp.SetHorizontalStep(4)
	return vp
}

var (
	addedStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("2")) // green
	removedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("1")) // red
	diffDimStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("8")) // dim
	diffBoldStyle = lipgloss.NewStyle().Bold(true)
)

// DiffView renders diff output with color coding in a scrollable viewport.
type DiffView struct {
	Viewport viewport.Model
	Table    string
	Content  string
	Ready    bool
}

// NewDiffView creates a new diff viewer with the given dimensions.
func NewDiffView(width, height int) DiffView {
	vp := newViewport(width, height)
	vp.SetContent("No changes to display")
	return DiffView{
		Viewport: vp,
		Ready:    true,
	}
}

// SetContent updates the diff content with color coding.
func (d *DiffView) SetContent(table, content string) {
	d.Table = table
	d.Content = content

	if content == "" {
		d.Viewport.SetContent("No changes to display")
		return
	}

	d.Viewport.SetContent(colorizeDiff(SanitizeForDisplay(content)))
	d.Viewport.GotoTop()
}

// SetSize updates the viewport dimensions.
func (d *DiffView) SetSize(width, height int) {
	d.Viewport.Width = width
	d.Viewport.Height = height
}

// Update passes events to the viewport.
func (d DiffView) Update(msg tea.Msg) (DiffView, tea.Cmd) {
	var cmd tea.Cmd
	d.Viewport, cmd = d.Viewport.Update(msg)
	return d, cmd
}

// View renders the viewport.
func (d DiffView) View() string {
	return d.Viewport.View()
}

// colorizeDiff applies lipgloss styles to diff output lines.
func colorizeDiff(s string) string {
	lines := strings.Split(s, "\n")
	var out []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "diff --dolt"):
			out = append(out, diffBoldStyle.Render(line))
		case strings.HasPrefix(trimmed, "| + |"), strings.HasPrefix(trimmed, "| > |"):
			out = append(out, addedStyle.Render(line))
		case strings.HasPrefix(trimmed, "| - |"), strings.HasPrefix(trimmed, "| < |"):
			out = append(out, removedStyle.Render(line))
		case strings.HasPrefix(trimmed, "+") && !strings.HasPrefix(trimmed, "+---"):
			out = append(out, addedStyle.Render(line))
		case strings.HasPrefix(trimmed, "-") && !strings.HasPrefix(trimmed, "---"):
			out = append(out, removedStyle.Render(line))
		case strings.HasPrefix(trimmed, "+---") || strings.HasPrefix(trimmed, "---"):
			out = append(out, diffDimStyle.Render(line))
		default:
			out = append(out, line)
		}
	}

	return strings.Join(out, "\n")
}
