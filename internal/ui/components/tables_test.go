package components

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/aspiers/lazydolt/internal/domain"
)

func init() {
	lipgloss.SetColorProfile(termenv.ANSI)
}

// execCmd is a helper to extract the tea.Msg from a tea.Cmd.
func execCmd(t *testing.T, cmd tea.Cmd) tea.Msg {
	t.Helper()
	if cmd == nil {
		t.Fatal("expected a command, got nil")
	}
	return cmd()
}

func newTablesModel(tables []domain.Table, cursor int, focused bool) TablesModel {
	return TablesModel{
		Tables:  tables,
		Cursor:  cursor,
		Focused: focused,
		Height:  20,
	}
}

// sampleTables returns tables that produce this display list:
//
//	0: ▼ Unstaged changes (1)  [header]
//	1:   M  users              [table]
//	2: ▼ Staged changes (1)    [header]
//	3:   A  orders             [table]
//	4: ▼ Clean (1)             [header]
//	5:   products              [table]
func sampleTables() []domain.Table {
	return []domain.Table{
		{Name: "users", Status: &domain.StatusEntry{TableName: "users", Status: "modified", Staged: false}},
		{Name: "orders", Status: &domain.StatusEntry{TableName: "orders", Status: "new table", Staged: true}},
		{Name: "products"},
	}
}

func keyMsg(key string) tea.Msg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
}

func specialKeyMsg(k tea.KeyType) tea.Msg {
	return tea.KeyMsg{Type: k}
}

func TestTablesModel_NavigateDown(t *testing.T) {
	m := newTablesModel(sampleTables(), 0, true) // on header
	m, _ = m.Update(keyMsg("j"))
	if m.Cursor != 1 {
		t.Errorf("cursor = %d, want 1", m.Cursor)
	}
}

func TestTablesModel_NavigateUp(t *testing.T) {
	m := newTablesModel(sampleTables(), 3, true) // on "orders" table
	m, _ = m.Update(keyMsg("k"))
	if m.Cursor != 2 {
		t.Errorf("cursor = %d, want 2", m.Cursor)
	}
}

func TestTablesModel_NavigateDownAtBottom(t *testing.T) {
	m := newTablesModel(sampleTables(), 5, true) // last item (products)
	m, _ = m.Update(keyMsg("j"))
	if m.Cursor != 5 {
		t.Errorf("cursor = %d, want 5 (should not wrap)", m.Cursor)
	}
}

func TestTablesModel_NavigateUpAtTop(t *testing.T) {
	m := newTablesModel(sampleTables(), 0, true)
	m, _ = m.Update(keyMsg("k"))
	if m.Cursor != 0 {
		t.Errorf("cursor = %d, want 0 (should not wrap)", m.Cursor)
	}
}

func TestTablesModel_IgnoreKeysWhenUnfocused(t *testing.T) {
	m := newTablesModel(sampleTables(), 0, false)
	m, _ = m.Update(keyMsg("j"))
	if m.Cursor != 0 {
		t.Errorf("cursor = %d, want 0 (unfocused should ignore keys)", m.Cursor)
	}
}

func TestTablesModel_SpaceStagesUnstagedTable(t *testing.T) {
	m := newTablesModel(sampleTables(), 1, true) // cursor on "users" (unstaged)
	_, cmd := m.Update(keyMsg(" "))
	msg := execCmd(t, cmd)
	stageMsg, ok := msg.(StageTableMsg)
	if !ok {
		t.Fatalf("expected StageTableMsg, got %T", msg)
	}
	if stageMsg.Table != "users" {
		t.Errorf("table = %q, want %q", stageMsg.Table, "users")
	}
}

func TestTablesModel_SpaceUnstagesStagedTable(t *testing.T) {
	m := newTablesModel(sampleTables(), 3, true) // cursor on "orders" (staged)
	_, cmd := m.Update(keyMsg(" "))
	msg := execCmd(t, cmd)
	unstageMsg, ok := msg.(UnstageTableMsg)
	if !ok {
		t.Fatalf("expected UnstageTableMsg, got %T", msg)
	}
	if unstageMsg.Table != "orders" {
		t.Errorf("table = %q, want %q", unstageMsg.Table, "orders")
	}
}

func TestTablesModel_SpaceTogglesSection(t *testing.T) {
	m := newTablesModel(sampleTables(), 0, true) // cursor on "Unstaged" header
	m, cmd := m.Update(keyMsg(" "))
	if cmd != nil {
		t.Error("pressing space on header should not produce a command")
	}
	if !m.Collapsed[sectionUnstaged] {
		t.Error("unstaged section should be collapsed after space on header")
	}

	// Press space again to expand
	m, cmd = m.Update(keyMsg(" "))
	if cmd != nil {
		t.Error("pressing space on header should not produce a command")
	}
	if m.Collapsed[sectionUnstaged] {
		t.Error("unstaged section should be expanded after second space on header")
	}
}

func TestTablesModel_EnterTogglesSection(t *testing.T) {
	m := newTablesModel(sampleTables(), 0, true) // cursor on header
	m, _ = m.Update(specialKeyMsg(tea.KeyEnter))
	if !m.Collapsed[sectionUnstaged] {
		t.Error("unstaged section should be collapsed after enter on header")
	}
}

func TestTablesModel_StageAll(t *testing.T) {
	m := newTablesModel(sampleTables(), 0, true)
	_, cmd := m.Update(keyMsg("a"))
	msg := execCmd(t, cmd)
	if _, ok := msg.(StageAllMsg); !ok {
		t.Fatalf("expected StageAllMsg, got %T", msg)
	}
}

func TestTablesModel_SelectedTable(t *testing.T) {
	m := newTablesModel(sampleTables(), 1, true) // cursor on "users"
	if got := m.SelectedTable(); got != "users" {
		t.Errorf("SelectedTable() = %q, want %q", got, "users")
	}
}

func TestTablesModel_SelectedTableOnHeader(t *testing.T) {
	m := newTablesModel(sampleTables(), 0, true) // cursor on header
	if got := m.SelectedTable(); got != "" {
		t.Errorf("SelectedTable() on header = %q, want empty", got)
	}
}

func TestTablesModel_SelectedTableEmpty(t *testing.T) {
	m := newTablesModel(nil, 0, true)
	if got := m.SelectedTable(); got != "" {
		t.Errorf("SelectedTable() = %q, want empty", got)
	}
}

func TestTablesModel_ViewEmptyList(t *testing.T) {
	m := newTablesModel(nil, 0, true)
	if got := m.View(); got != "No tables" {
		t.Errorf("View() = %q, want %q", got, "No tables")
	}
}

func TestTablesModel_ViewShowsSectionHeaders(t *testing.T) {
	m := newTablesModel(sampleTables(), 0, false)
	view := m.View()

	for _, header := range []string{"Unstaged changes (1)", "Staged changes (1)", "Clean (1)"} {
		if !strings.Contains(view, header) {
			t.Errorf("View() should contain header %q, got:\n%s", header, view)
		}
	}
}

func TestTablesModel_ViewShowsTableNames(t *testing.T) {
	m := newTablesModel(sampleTables(), -1, false)
	view := m.View()

	for _, name := range []string{"users", "orders", "products"} {
		if !strings.Contains(view, name) {
			t.Errorf("View() should contain table name %q", name)
		}
	}
}

func TestTablesModel_ViewCollapsedSection(t *testing.T) {
	m := newTablesModel(sampleTables(), 0, false)
	m.Collapsed = map[section]bool{sectionUnstaged: true}
	view := m.View()

	// Header should still show
	if !strings.Contains(view, "Unstaged changes") {
		t.Error("collapsed section header should still be visible")
	}
	// Collapsed arrow
	if !strings.Contains(view, "▶") {
		t.Error("collapsed section should show ▶ arrow")
	}
	// Table under collapsed section should be hidden
	if strings.Contains(view, "users") {
		t.Error("table under collapsed section should be hidden")
	}
	// Other sections should still show their tables
	if !strings.Contains(view, "orders") {
		t.Error("tables in other sections should be visible")
	}
}

func TestTablesModel_ViewStatusMarkers(t *testing.T) {
	tables := []domain.Table{
		{Name: "modified_unstaged", Status: &domain.StatusEntry{Status: "modified", Staged: false}},
		{Name: "new_staged", Status: &domain.StatusEntry{Status: "new table", Staged: true}},
		{Name: "modified_staged", Status: &domain.StatusEntry{Status: "modified", Staged: true}},
		{Name: "deleted", Status: &domain.StatusEntry{Status: "deleted", Staged: false}},
		{Name: "new_unstaged", Status: &domain.StatusEntry{Status: "new table", Staged: false}},
		{Name: "clean"},
	}
	m := newTablesModel(tables, -1, false) // no cursor highlight
	view := m.View()

	// Check each table line contains its name
	for _, tbl := range tables {
		if !strings.Contains(view, tbl.Name) {
			t.Errorf("View() should contain table name %q", tbl.Name)
		}
	}
}

func TestTablesModel_DisplayItems(t *testing.T) {
	m := newTablesModel(sampleTables(), 0, true)
	items := m.displayItems()

	// Should be: header, users, header, orders, header, products = 6 items
	if len(items) != 6 {
		t.Fatalf("displayItems() len = %d, want 6", len(items))
	}

	// Check structure
	expectations := []struct {
		isHeader bool
		name     string // table name (empty for headers)
	}{
		{true, ""},
		{false, "users"},
		{true, ""},
		{false, "orders"},
		{true, ""},
		{false, "products"},
	}
	for i, exp := range expectations {
		got := items[i]
		if got.isHeader != exp.isHeader {
			t.Errorf("items[%d].isHeader = %v, want %v", i, got.isHeader, exp.isHeader)
		}
		if !exp.isHeader && got.table.Name != exp.name {
			t.Errorf("items[%d].table.Name = %q, want %q", i, got.table.Name, exp.name)
		}
	}
}

func TestTablesModel_DisplayItemsCollapsed(t *testing.T) {
	m := newTablesModel(sampleTables(), 0, true)
	m.Collapsed = map[section]bool{sectionUnstaged: true}
	items := m.displayItems()

	// Collapsed: header(unstaged), header(staged), orders, header(clean), products = 5 items
	if len(items) != 5 {
		t.Fatalf("displayItems() with collapsed section len = %d, want 5", len(items))
	}

	if !items[0].isHeader {
		t.Error("items[0] should be the collapsed unstaged header")
	}
	if !items[1].isHeader {
		t.Error("items[1] should be the staged header")
	}
	if items[2].isHeader || items[2].table.Name != "orders" {
		t.Errorf("items[2] should be orders, got header=%v name=%s",
			items[2].isHeader, items[2].table.Name)
	}
}

func TestTablesModel_ClampCursor(t *testing.T) {
	m := newTablesModel(sampleTables(), 99, true)
	m.ClampCursor()
	items := m.displayItems()
	if m.Cursor != len(items)-1 {
		t.Errorf("ClampCursor: cursor = %d, want %d", m.Cursor, len(items)-1)
	}
}

func TestTablesModel_GroupTables(t *testing.T) {
	m := newTablesModel(sampleTables(), 0, true)
	unstaged, staged, clean := m.groupTables()

	if len(unstaged) != 1 || unstaged[0].Name != "users" {
		t.Errorf("unstaged = %v, want [users]", unstaged)
	}
	if len(staged) != 1 || staged[0].Name != "orders" {
		t.Errorf("staged = %v, want [orders]", staged)
	}
	if len(clean) != 1 || clean[0].Name != "products" {
		t.Errorf("clean = %v, want [products]", clean)
	}
}

func TestTablesModel_EmptySectionsOmitted(t *testing.T) {
	// Only unstaged tables — no staged or clean headers should appear
	tables := []domain.Table{
		{Name: "foo", Status: &domain.StatusEntry{Status: "modified", Staged: false}},
	}
	m := newTablesModel(tables, 0, true)
	items := m.displayItems()

	// Should be: header(unstaged), foo = 2 items
	if len(items) != 2 {
		t.Fatalf("displayItems() with only unstaged: len = %d, want 2", len(items))
	}
	if !items[0].isHeader {
		t.Error("items[0] should be the unstaged header")
	}
	if items[1].table.Name != "foo" {
		t.Errorf("items[1].table.Name = %q, want %q", items[1].table.Name, "foo")
	}
}
