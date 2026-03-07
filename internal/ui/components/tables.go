package components

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/aspiers/lazydolt/internal/domain"
)

var (
	stagedNewStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("2")) // green
	modifiedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("3")) // yellow
	deletedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("1")) // red
	selectedStyle  = lipgloss.NewStyle().Reverse(true)
	normalStyle    = lipgloss.NewStyle()
)

// TablesModel displays a navigable list of tables with status markers.
type TablesModel struct {
	Tables  []domain.Table
	Cursor  int
	Focused bool
	Height  int
	HScroll int // horizontal scroll offset (columns)
}

// Init is a no-op.
func (m TablesModel) Init() tea.Cmd { return nil }

// Update handles key events when focused.
func (m TablesModel) Update(msg tea.Msg) (TablesModel, tea.Cmd) {
	if !m.Focused {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if m.Cursor < len(m.Tables)-1 {
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
		case " ":
			return m, m.toggleStage()
		case "a":
			return m, func() tea.Msg { return StageAllMsg{} }
		}
	}
	return m, nil
}

// View renders the table list, clipped to the visible height.
func (m TablesModel) View() string {
	if len(m.Tables) == 0 {
		return "No tables"
	}

	start, end := visibleRange(m.Cursor, len(m.Tables), m.Height)

	var s string
	for i := start; i < end; i++ {
		t := m.Tables[i]
		marker := "  "
		markerFn := normalStyle.Render

		if t.Status != nil {
			switch {
			case t.Status.Staged && (t.Status.Status == "new table"):
				marker = "A "
				markerFn = stagedNewStyle.Render
			case t.Status.Staged:
				marker = "SM"
				markerFn = stagedNewStyle.Render
			case t.Status.Status == "modified":
				marker = "M "
				markerFn = modifiedStyle.Render
			case t.Status.Status == "new table":
				marker = "? "
				markerFn = modifiedStyle.Render
			case t.Status.Status == "deleted":
				marker = "D "
				markerFn = deletedStyle.Render
			}
		}

		line := fmt.Sprintf("%s %s", markerFn(marker), t.Name)
		if i == m.Cursor && m.Focused {
			line = selectedStyle.Render(line)
		}
		s += line + "\n"
	}

	return HScrollContent(s, m.HScroll)
}

// SelectedTable returns the currently selected table name, or empty string.
func (m TablesModel) SelectedTable() string {
	if m.Cursor >= 0 && m.Cursor < len(m.Tables) {
		return m.Tables[m.Cursor].Name
	}
	return ""
}

// Message types for parent to handle.
type StageTableMsg struct{ Table string }
type UnstageTableMsg struct{ Table string }
type StageAllMsg struct{}
type ViewDiffMsg struct{ Table string }
type ViewSchemaMsg struct{ Table string }
type ViewTableDataMsg struct{ Table string }

func (m TablesModel) toggleStage() tea.Cmd {
	t := m.SelectedTable()
	if t == "" {
		return nil
	}
	if m.Cursor < len(m.Tables) && m.Tables[m.Cursor].Status != nil && m.Tables[m.Cursor].Status.Staged {
		return func() tea.Msg { return UnstageTableMsg{Table: t} }
	}
	return func() tea.Msg { return StageTableMsg{Table: t} }
}

func (m TablesModel) viewDiff() tea.Cmd {
	t := m.SelectedTable()
	if t == "" {
		return nil
	}
	return func() tea.Msg { return ViewDiffMsg{Table: t} }
}

func (m TablesModel) viewSchema() tea.Cmd {
	t := m.SelectedTable()
	if t == "" {
		return nil
	}
	return func() tea.Msg { return ViewSchemaMsg{Table: t} }
}

func (m TablesModel) viewData() tea.Cmd {
	t := m.SelectedTable()
	if t == "" {
		return nil
	}
	return func() tea.Msg { return ViewTableDataMsg{Table: t} }
}
