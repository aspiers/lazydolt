package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	hintKeyStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	hintSepStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

// KeyHint represents a single keybinding hint.
type KeyHint struct {
	Key  string
	Desc string
}

// Panel identifies which panel is focused.
type Panel int

const (
	PanelTables Panel = iota
	PanelBranches
	PanelCommits
	PanelMain // right-hand panel (diff/schema/browser)
)

// RenderKeyHints renders the hint bar for the given panel and width.
func RenderKeyHints(panel Panel, width int) string {
	hints := hintsForPanel(panel)
	return renderHints(hints, width)
}

func hintsForPanel(panel Panel) []KeyHint {
	global := []KeyHint{
		{Key: "+/_", Desc: "zoom"},
		{Key: "</>/=", Desc: "width"},
		{Key: "H/L", Desc: "scroll"},
		{Key: "Esc", Desc: "back"},
		{Key: "c", Desc: "commit"},
		{Key: "?", Desc: "help"},
		{Key: "q", Desc: "quit"},
	}

	switch panel {
	case PanelTables:
		return append([]KeyHint{
			{Key: "Space", Desc: "stage"},
			{Key: "[/]", Desc: "tab"},
			{Key: "a", Desc: "stage all"},
		}, global...)
	case PanelBranches:
		return append([]KeyHint{
			{Key: "Enter", Desc: "checkout"},
			{Key: "n", Desc: "new"},
			{Key: "D", Desc: "delete"},
		}, global...)
	case PanelCommits:
		return append([]KeyHint{
			{Key: "Enter", Desc: "details"},
			{Key: "g", Desc: "reset"},
		}, global...)
	case PanelMain:
		return append([]KeyHint{
			{Key: "j/k", Desc: "scroll"},
			{Key: "PgUp/Dn", Desc: "page"},
		}, global...)
	default:
		return global
	}
}

func renderHints(hints []KeyHint, width int) string {
	sep := hintSepStyle.Render(" | ")
	var parts []string
	for _, h := range hints {
		parts = append(parts, hintKeyStyle.Render(h.Key)+" "+h.Desc)
	}

	result := strings.Join(parts, sep)

	// Truncate if too wide
	// (rough check — lipgloss styles add invisible chars)
	if lipgloss.Width(result) > width && len(parts) > 2 {
		for len(parts) > 2 && lipgloss.Width(strings.Join(parts, sep)) > width {
			parts = parts[:len(parts)-1]
		}
		result = strings.Join(parts, sep)
	}

	return result
}
