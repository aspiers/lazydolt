package components

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	sqlKeywordStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("4")) // blue
	sqlTypeStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))            // cyan
	sqlIdentStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))            // yellow

	sqlKeywords = regexp.MustCompile(`(?i)\b(CREATE|TABLE|PRIMARY|KEY|NOT|NULL|DEFAULT|ENGINE|CHARSET|COLLATE|UNIQUE|INDEX|FOREIGN|REFERENCES|AUTO_INCREMENT|INT|VARCHAR|TEXT|BLOB|DATETIME|TIMESTAMP|DECIMAL|FLOAT|DOUBLE|BOOLEAN|DATE|TIME|BIGINT|SMALLINT|TINYINT|MEDIUMINT|CHAR|ENUM|SET)\b`)
	sqlBacktick = regexp.MustCompile("`[^`]+`")
)

// SchemaView renders a CREATE TABLE statement with syntax highlighting.
type SchemaView struct {
	Viewport viewport.Model
	Table    string
	Schema   string
	Ready    bool
}

// NewSchemaView creates a new schema viewer with the given dimensions.
func NewSchemaView(width, height int) SchemaView {
	vp := newViewport(width, height)
	vp.SetContent("Select a table to view its schema")
	return SchemaView{
		Viewport: vp,
		Ready:    true,
	}
}

// SetContent updates the schema content with highlighting.
func (sv *SchemaView) SetContent(table, schema string) {
	sv.Table = table
	sv.Schema = schema

	if schema == "" {
		sv.Viewport.SetContent("No schema to display")
		return
	}

	sv.Viewport.SetContent(highlightSQL(schema))
	sv.Viewport.GotoTop()
}

// Clear resets the schema view with a placeholder message.
func (sv *SchemaView) Clear(message string) {
	sv.Table = ""
	sv.Schema = ""
	sv.Viewport.SetContent(message)
}

// SetSize updates the viewport dimensions.
func (sv *SchemaView) SetSize(width, height int) {
	sv.Viewport.Width = width
	sv.Viewport.Height = height
}

// Update passes events to the viewport.
func (sv SchemaView) Update(msg tea.Msg) (SchemaView, tea.Cmd) {
	var cmd tea.Cmd
	sv.Viewport, cmd = sv.Viewport.Update(msg)
	return sv, cmd
}

// View renders the viewport.
func (sv SchemaView) View() string {
	return sv.Viewport.View()
}

// highlightSQL applies basic syntax highlighting to a SQL statement.
func highlightSQL(sql string) string {
	lines := strings.Split(sql, "\n")
	var out []string

	for _, line := range lines {
		// Highlight backtick-quoted identifiers first
		highlighted := sqlBacktick.ReplaceAllStringFunc(line, func(match string) string {
			return sqlIdentStyle.Render(match)
		})

		// Highlight SQL keywords
		highlighted = sqlKeywords.ReplaceAllStringFunc(highlighted, func(match string) string {
			return sqlKeywordStyle.Render(match)
		})

		out = append(out, highlighted)
	}

	return strings.Join(out, "\n")
}
