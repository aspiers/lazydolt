package components

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/exp/golden"
)

// Note: init() in diffview_test.go sets lipgloss.SetColorProfile(termenv.ANSI)
// which applies to all tests in this package.

func TestBrowserView_RenderTable_Basic(t *testing.T) {
	bv := NewBrowserView(80, 20)
	columns := []string{"email", "id", "name"}
	rows := []map[string]interface{}{
		{"id": float64(1), "name": "Alice", "email": "alice@example.com"},
		{"id": float64(2), "name": "Bob", "email": "bob@example.com"},
	}
	bv.SetData("users", columns, rows, 2, 0)
	golden.RequireEqual(t, bv.Viewport.View())
}

func TestBrowserView_RenderTable_NullValues(t *testing.T) {
	bv := NewBrowserView(80, 20)
	columns := []string{"id", "name"}
	rows := []map[string]interface{}{
		{"id": float64(1)}, // name is missing (NULL)
	}
	bv.SetData("users", columns, rows, 1, 0)
	golden.RequireEqual(t, bv.Viewport.View())
}

func TestBrowserView_RenderTable_Empty(t *testing.T) {
	bv := NewBrowserView(80, 20)
	bv.SetData("empty_table", nil, nil, 0, 0)
	golden.RequireEqual(t, bv.Viewport.View())
}

func TestBrowserView_Pagination(t *testing.T) {
	bv := NewBrowserView(80, 20)
	bv.PageSize = 10

	// Simulate page 1 of 3
	columns := []string{"id"}
	rows := []map[string]interface{}{
		{"id": float64(1)},
	}
	bv.SetData("big_table", columns, rows, 25, 0)

	if bv.CurrentPage() != 1 {
		t.Errorf("expected current page 1, got %d", bv.CurrentPage())
	}
	if bv.TotalPages() != 3 {
		t.Errorf("expected 3 total pages, got %d", bv.TotalPages())
	}

	// Page 2
	bv.SetData("big_table", columns, rows, 25, 10)
	if bv.CurrentPage() != 2 {
		t.Errorf("expected current page 2, got %d", bv.CurrentPage())
	}

	// Page 3
	bv.SetData("big_table", columns, rows, 25, 20)
	if bv.CurrentPage() != 3 {
		t.Errorf("expected current page 3, got %d", bv.CurrentPage())
	}
}

func TestBrowserView_PaginationFooter(t *testing.T) {
	bv := NewBrowserView(80, 20)
	bv.PageSize = 10

	columns := []string{"id"}
	rows := []map[string]interface{}{
		{"id": float64(1)},
	}
	bv.SetData("big_table", columns, rows, 25, 0)
	golden.RequireEqual(t, bv.Viewport.View())
}

func TestBrowserView_NoPaginationFooterForSinglePage(t *testing.T) {
	bv := NewBrowserView(80, 20)
	bv.PageSize = 100

	columns := []string{"id"}
	rows := []map[string]interface{}{
		{"id": float64(1)},
	}
	bv.SetData("small_table", columns, rows, 1, 0)
	golden.RequireEqual(t, bv.Viewport.View())
}

func TestBrowserView_ColumnWidthTruncation(t *testing.T) {
	bv := NewBrowserView(80, 20)
	columns := []string{"data"}
	longVal := strings.Repeat("x", 60) // exceeds maxColWidth (40)
	rows := []map[string]interface{}{
		{"data": longVal},
	}
	bv.SetData("test", columns, rows, 1, 0)
	golden.RequireEqual(t, bv.Viewport.View())
}

func TestBrowserView_CalcColWidths(t *testing.T) {
	bv := NewBrowserView(80, 20)
	bv.Columns = []string{"id", "name"}
	bv.Rows = []map[string]interface{}{
		{"id": float64(1), "name": "Alice"},
		{"id": float64(2), "name": "Bob"},
	}

	widths := bv.calcColWidths()

	// "id" header is 2 chars, values are 1 char -> width should be 2
	if widths[0] != 2 {
		t.Errorf("expected width[0]=2 for 'id', got %d", widths[0])
	}

	// "name" header is 4 chars, "Alice" is 5 chars -> width should be 5
	if widths[1] != 5 {
		t.Errorf("expected width[1]=5 for 'name', got %d", widths[1])
	}
}

func TestBrowserView_BuildSeparator(t *testing.T) {
	bv := NewBrowserView(80, 20)
	sep := bv.buildSeparator([]int{3, 5})

	expected := "+-----+-------+"
	if sep != expected {
		t.Errorf("expected separator %q, got %q", expected, sep)
	}
}

func TestBrowserView_BuildRow(t *testing.T) {
	bv := NewBrowserView(80, 20)
	row := bv.buildRow([]int{3, 5}, []string{"id", "Alice"}, nil)

	// Should have pipes and padding
	if !strings.HasPrefix(row, "|") || !strings.HasSuffix(row, "|") {
		t.Errorf("expected row to start and end with '|', got %q", row)
	}
	if !strings.Contains(row, "id") {
		t.Errorf("expected 'id' in row, got %q", row)
	}
	if !strings.Contains(row, "Alice") {
		t.Errorf("expected 'Alice' in row, got %q", row)
	}
}

func TestBrowserView_DefaultContent(t *testing.T) {
	bv := NewBrowserView(80, 20)
	golden.RequireEqual(t, bv.View())
}
