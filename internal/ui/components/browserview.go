package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const maxColWidth = 40

var (
	nullStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Italic(true)
	browserDimText = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

// BrowserPageMsg requests a new page of data from the parent.
type BrowserPageMsg struct {
	Table  string
	Offset int
}

// BrowserView renders table data in a scrollable formatted text table.
type BrowserView struct {
	Viewport viewport.Model
	Table    string
	Columns  []string
	Rows     []map[string]interface{}
	Total    int
	Offset   int
	PageSize int
	Ready    bool
}

// NewBrowserView creates a new browser view with the given dimensions.
func NewBrowserView(width, height int) BrowserView {
	vp := viewport.New(width, height)
	vp.SetContent("Select a table and press Enter to browse data")
	vp.SetHorizontalStep(4)
	return BrowserView{
		Viewport: vp,
		PageSize: 100,
		Ready:    true,
	}
}

// SetData updates the browser with new query results.
func (bv *BrowserView) SetData(table string, columns []string, rows []map[string]interface{}, total, offset int) {
	bv.Table = table
	bv.Columns = columns
	bv.Rows = rows
	bv.Total = total
	bv.Offset = offset

	if len(columns) == 0 && len(rows) == 0 {
		bv.Viewport.SetContent("No data to display")
		return
	}

	bv.Viewport.SetContent(bv.renderTable())
	bv.Viewport.GotoTop()
}

// SetSize updates the viewport dimensions.
func (bv *BrowserView) SetSize(width, height int) {
	bv.Viewport.Width = width
	bv.Viewport.Height = height
}

// Update passes events to the viewport and handles pagination keys.
func (bv BrowserView) Update(msg tea.Msg) (BrowserView, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "[", "shift+tab":
			// Previous page
			if bv.Offset > 0 {
				newOffset := bv.Offset - bv.PageSize
				if newOffset < 0 {
					newOffset = 0
				}
				return bv, func() tea.Msg {
					return BrowserPageMsg{Table: bv.Table, Offset: newOffset}
				}
			}
		case "]", "tab":
			// Next page
			if bv.Offset+bv.PageSize < bv.Total {
				newOffset := bv.Offset + bv.PageSize
				return bv, func() tea.Msg {
					return BrowserPageMsg{Table: bv.Table, Offset: newOffset}
				}
			}
		}
	}

	var cmd tea.Cmd
	bv.Viewport, cmd = bv.Viewport.Update(msg)
	return bv, cmd
}

// View renders the viewport.
func (bv BrowserView) View() string {
	return bv.Viewport.View()
}

// CurrentPage returns the 1-based current page number.
func (bv BrowserView) CurrentPage() int {
	if bv.PageSize == 0 {
		return 1
	}
	return bv.Offset/bv.PageSize + 1
}

// TotalPages returns the total number of pages.
func (bv BrowserView) TotalPages() int {
	if bv.PageSize == 0 || bv.Total == 0 {
		return 1
	}
	return (bv.Total + bv.PageSize - 1) / bv.PageSize
}

// renderTable builds a text table from columns and rows.
func (bv BrowserView) renderTable() string {
	if len(bv.Columns) == 0 {
		return "No data to display"
	}

	// Calculate column widths
	widths := bv.calcColWidths()

	var sb strings.Builder

	// Header info
	sb.WriteString(fmt.Sprintf("%s (%d rows)", bv.Table, bv.Total))
	sb.WriteString("\n")

	// Separator line
	sep := bv.buildSeparator(widths)
	sb.WriteString(sep)
	sb.WriteString("\n")

	// Header row
	sb.WriteString(bv.buildRow(widths, bv.Columns, nil))
	sb.WriteString("\n")

	// Separator after header
	sb.WriteString(sep)
	sb.WriteString("\n")

	// Data rows
	for _, row := range bv.Rows {
		vals := make([]string, len(bv.Columns))
		isNull := make([]bool, len(bv.Columns))
		for i, col := range bv.Columns {
			v, exists := row[col]
			if !exists || v == nil {
				vals[i] = "NULL"
				isNull[i] = true
			} else {
				vals[i] = fmt.Sprintf("%v", v)
			}
		}
		sb.WriteString(bv.buildRow(widths, vals, isNull))
		sb.WriteString("\n")
	}

	// Bottom separator
	sb.WriteString(sep)

	// Pagination footer
	if bv.TotalPages() > 1 {
		sb.WriteString("\n")
		sb.WriteString(browserDimText.Render(
			fmt.Sprintf("Page %d/%d  ([/] to navigate)", bv.CurrentPage(), bv.TotalPages()),
		))
	}

	return sb.String()
}

// calcColWidths returns the display width for each column.
func (bv BrowserView) calcColWidths() []int {
	widths := make([]int, len(bv.Columns))

	// Start with header widths
	for i, col := range bv.Columns {
		widths[i] = len(col)
	}

	// Expand based on data
	for _, row := range bv.Rows {
		for i, col := range bv.Columns {
			v, exists := row[col]
			var s string
			if !exists || v == nil {
				s = "NULL"
			} else {
				s = fmt.Sprintf("%v", v)
			}
			if len(s) > widths[i] {
				widths[i] = len(s)
			}
		}
	}

	// Cap at maxColWidth
	for i := range widths {
		if widths[i] > maxColWidth {
			widths[i] = maxColWidth
		}
	}

	return widths
}

// buildSeparator builds a "+----+----+" line.
func (bv BrowserView) buildSeparator(widths []int) string {
	var parts []string
	for _, w := range widths {
		parts = append(parts, strings.Repeat("-", w+2))
	}
	return "+" + strings.Join(parts, "+") + "+"
}

// buildRow builds a "| val | val |" line. isNull marks which values are NULL
// (rendered dim). isNull may be nil for header rows.
func (bv BrowserView) buildRow(widths []int, values []string, isNull []bool) string {
	var parts []string
	for i, val := range values {
		w := widths[i]
		display := val
		if len(display) > w {
			display = display[:w-3] + "..."
		}
		padded := fmt.Sprintf(" %-*s ", w, display)
		if isNull != nil && i < len(isNull) && isNull[i] {
			padded = " " + nullStyle.Render(fmt.Sprintf("%-*s", w, display)) + " "
		}
		parts = append(parts, padded)
	}
	return "|" + strings.Join(parts, "|") + "|"
}
