package components

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/aspiers/lazydolt/internal/domain"
)

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
	m := newTablesModel(sampleTables(), 0, true)
	m, _ = m.Update(keyMsg("j"))
	if m.Cursor != 1 {
		t.Errorf("cursor = %d, want 1", m.Cursor)
	}
}

func TestTablesModel_NavigateUp(t *testing.T) {
	m := newTablesModel(sampleTables(), 2, true)
	m, _ = m.Update(keyMsg("k"))
	if m.Cursor != 1 {
		t.Errorf("cursor = %d, want 1", m.Cursor)
	}
}

func TestTablesModel_NavigateDownAtBottom(t *testing.T) {
	m := newTablesModel(sampleTables(), 2, true)
	m, _ = m.Update(keyMsg("j"))
	if m.Cursor != 2 {
		t.Errorf("cursor = %d, want 2 (should not wrap)", m.Cursor)
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
	m := newTablesModel(sampleTables(), 0, true) // cursor on "users" (unstaged)
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
	m := newTablesModel(sampleTables(), 1, true) // cursor on "orders" (staged)
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

func TestTablesModel_StageAll(t *testing.T) {
	m := newTablesModel(sampleTables(), 0, true)
	_, cmd := m.Update(keyMsg("a"))
	msg := execCmd(t, cmd)
	if _, ok := msg.(StageAllMsg); !ok {
		t.Fatalf("expected StageAllMsg, got %T", msg)
	}
}

func TestTablesModel_ViewDiff(t *testing.T) {
	m := newTablesModel(sampleTables(), 0, true)
	_, cmd := m.Update(keyMsg("d"))
	msg := execCmd(t, cmd)
	diffMsg, ok := msg.(ViewDiffMsg)
	if !ok {
		t.Fatalf("expected ViewDiffMsg, got %T", msg)
	}
	if diffMsg.Table != "users" {
		t.Errorf("table = %q, want %q", diffMsg.Table, "users")
	}
}

func TestTablesModel_ViewSchema(t *testing.T) {
	m := newTablesModel(sampleTables(), 0, true)
	_, cmd := m.Update(keyMsg("s"))
	msg := execCmd(t, cmd)
	schemaMsg, ok := msg.(ViewSchemaMsg)
	if !ok {
		t.Fatalf("expected ViewSchemaMsg, got %T", msg)
	}
	if schemaMsg.Table != "users" {
		t.Errorf("table = %q, want %q", schemaMsg.Table, "users")
	}
}

func TestTablesModel_ViewData(t *testing.T) {
	m := newTablesModel(sampleTables(), 0, true)
	_, cmd := m.Update(specialKeyMsg(tea.KeyEnter))
	msg := execCmd(t, cmd)
	dataMsg, ok := msg.(ViewTableDataMsg)
	if !ok {
		t.Fatalf("expected ViewTableDataMsg, got %T", msg)
	}
	if dataMsg.Table != "users" {
		t.Errorf("table = %q, want %q", dataMsg.Table, "users")
	}
}

func TestTablesModel_SelectedTable(t *testing.T) {
	m := newTablesModel(sampleTables(), 1, true)
	if got := m.SelectedTable(); got != "orders" {
		t.Errorf("SelectedTable() = %q, want %q", got, "orders")
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
