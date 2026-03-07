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
	headerStyle    = lipgloss.NewStyle().Bold(true)
)

// section identifies which group a table belongs to.
type section int

const (
	sectionUnstaged section = iota
	sectionStaged
	sectionClean
)

// displayItem is an entry in the flat display list — either a section
// header or a table row.
type displayItem struct {
	isHeader bool
	section  section
	table    *domain.Table // nil for headers
}

// TablesModel displays a navigable list of tables grouped by status.
type TablesModel struct {
	Tables    []domain.Table
	Cursor    int
	Focused   bool
	Height    int
	HScroll   int // horizontal scroll offset (columns)
	Collapsed map[section]bool
}

// Init is a no-op.
func (m TablesModel) Init() tea.Cmd { return nil }

// Update handles key events when focused.
func (m TablesModel) Update(msg tea.Msg) (TablesModel, tea.Cmd) {
	if !m.Focused {
		return m, nil
	}

	items := m.displayItems()

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if m.Cursor < len(items)-1 {
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
			if m.Cursor >= 0 && m.Cursor < len(items) && items[m.Cursor].isHeader {
				m.toggleSection(items[m.Cursor].section)
				return m, nil
			}
			return m, m.toggleStage()
		case "enter":
			if m.Cursor >= 0 && m.Cursor < len(items) && items[m.Cursor].isHeader {
				m.toggleSection(items[m.Cursor].section)
				return m, nil
			}
		case "a":
			return m, func() tea.Msg { return StageAllMsg{} }
		}
	}
	return m, nil
}

// toggleSection flips the collapsed state of a section.
func (m *TablesModel) toggleSection(s section) {
	if m.Collapsed == nil {
		m.Collapsed = make(map[section]bool)
	}
	m.Collapsed[s] = !m.Collapsed[s]
}

// View renders the table list grouped by status, clipped to the visible height.
func (m TablesModel) View() string {
	if len(m.Tables) == 0 {
		return "No tables"
	}

	items := m.displayItems()
	if len(items) == 0 {
		return "No tables"
	}

	start, end := visibleRange(m.Cursor, len(items), m.Height)

	var s string
	for i := start; i < end; i++ {
		item := items[i]
		var line string

		if item.isHeader {
			line = m.renderHeader(item.section)
		} else {
			line = m.renderTable(item.table)
		}

		if i == m.Cursor && m.Focused {
			line = selectedStyle.Render(line)
		}
		s += line + "\n"
	}

	return HScrollContent(s, m.HScroll)
}

// renderHeader renders a section header line like "▼ Unstaged (2)".
func (m TablesModel) renderHeader(s section) string {
	unstaged, staged, clean := m.groupTables()

	var name string
	var count int
	switch s {
	case sectionUnstaged:
		name = "Unstaged changes"
		count = len(unstaged)
	case sectionStaged:
		name = "Staged changes"
		count = len(staged)
	case sectionClean:
		name = "Clean"
		count = len(clean)
	}

	arrow := "▼"
	if m.Collapsed[s] {
		arrow = "▶"
	}

	return headerStyle.Render(fmt.Sprintf("%s %s (%d)", arrow, name, count))
}

// renderTable renders a single table line with its status marker.
func (m TablesModel) renderTable(t *domain.Table) string {
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

	return fmt.Sprintf("  %s %s", markerFn(marker), t.Name)
}

// groupTables splits the Tables slice into unstaged, staged, and clean groups.
func (m TablesModel) groupTables() (unstaged, staged, clean []domain.Table) {
	for _, t := range m.Tables {
		if t.Status == nil {
			clean = append(clean, t)
		} else if t.Status.Staged {
			staged = append(staged, t)
		} else {
			unstaged = append(unstaged, t)
		}
	}
	return
}

// displayItems builds the flat display list of headers and table entries,
// respecting collapsed sections. Only sections with items are shown.
func (m TablesModel) displayItems() []displayItem {
	unstaged, staged, clean := m.groupTables()

	var items []displayItem

	if len(unstaged) > 0 {
		items = append(items, displayItem{isHeader: true, section: sectionUnstaged})
		if !m.Collapsed[sectionUnstaged] {
			for i := range unstaged {
				items = append(items, displayItem{table: &unstaged[i]})
			}
		}
	}

	if len(staged) > 0 {
		items = append(items, displayItem{isHeader: true, section: sectionStaged})
		if !m.Collapsed[sectionStaged] {
			for i := range staged {
				items = append(items, displayItem{table: &staged[i]})
			}
		}
	}

	if len(clean) > 0 {
		items = append(items, displayItem{isHeader: true, section: sectionClean})
		if !m.Collapsed[sectionClean] {
			for i := range clean {
				items = append(items, displayItem{table: &clean[i]})
			}
		}
	}

	return items
}

// SelectedTable returns the currently selected table name, or empty string.
// Returns empty if the cursor is on a section header.
func (m TablesModel) SelectedTable() string {
	items := m.displayItems()
	if m.Cursor < 0 || m.Cursor >= len(items) {
		return ""
	}
	item := items[m.Cursor]
	if item.isHeader || item.table == nil {
		return ""
	}
	return item.table.Name
}

// selectedEntry returns the StatusEntry for the table under the cursor,
// or nil if the cursor is on a header or clean table.
func (m TablesModel) selectedEntry() *domain.StatusEntry {
	items := m.displayItems()
	if m.Cursor < 0 || m.Cursor >= len(items) {
		return nil
	}
	item := items[m.Cursor]
	if item.isHeader || item.table == nil {
		return nil
	}
	return item.table.Status
}

// ClampCursor ensures the cursor is within the display list bounds.
func (m *TablesModel) ClampCursor() {
	items := m.displayItems()
	if len(items) == 0 {
		m.Cursor = 0
		return
	}
	if m.Cursor >= len(items) {
		m.Cursor = len(items) - 1
	}
	if m.Cursor < 0 {
		m.Cursor = 0
	}
}

// DisplayLen returns the number of items in the display list (for title counts etc.).
func (m TablesModel) DisplayLen() int {
	return len(m.displayItems())
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
	entry := m.selectedEntry()
	if entry != nil && entry.Staged {
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
