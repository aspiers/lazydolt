// Package ui implements the Bubble Tea TUI for lazydolt.
package ui

import "github.com/aspiers/lazydolt/internal/domain"

// RefreshMsg triggers a reload of all data from dolt.
type RefreshMsg struct{}

// ErrorMsg carries an error to display to the user.
type ErrorMsg struct {
	Err error
}

// DataLoadedMsg carries refreshed data from dolt.
type DataLoadedMsg struct {
	Branch   string
	Dirty    bool
	Tables   []domain.Table
	Branches []domain.Branch
	Commits  []domain.Commit
}

// CommitSuccessMsg is sent when a commit succeeds.
type CommitSuccessMsg struct{ Hash string }

// DiffContentMsg carries diff text to the main panel.
type DiffContentMsg struct {
	Table   string
	Content string
}

// SchemaContentMsg carries schema text to the main panel.
type SchemaContentMsg struct {
	Table  string
	Schema string
}

// BrowserDataMsg carries table data to the browser.
type BrowserDataMsg struct {
	Table   string
	Columns []string
	Rows    []map[string]interface{}
	Total   int
	Offset  int
}
