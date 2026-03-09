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

// ResetSuccessMsg is sent when a reset to commit succeeds.
type ResetSuccessMsg struct{ Mode string }

// RemoteOpSuccessMsg is sent when a push/pull/fetch succeeds.
type RemoteOpSuccessMsg struct{ Op string }

// NewBranchSuccessMsg is sent when a new branch is created.
type NewBranchSuccessMsg struct{ Name string }

// RenameBranchSuccessMsg is sent when a branch is renamed.
type RenameBranchSuccessMsg struct{ OldName, NewName string }

// MergeSuccessMsg is sent when a merge completes.
type MergeSuccessMsg struct{ Branch string }

// MergeConflictMsg is sent when a merge results in conflicts.
type MergeConflictMsg struct{ Branch string }

// ConflictResolveMsg is sent when conflicts are resolved for a table.
type ConflictResolveMsg struct {
	Table string
	Ours  bool // true if resolved with --ours, false for --theirs
}

// MergeAbortMsg is sent when a merge is aborted.
type MergeAbortMsg struct{}

// CherryPickSuccessMsg is sent when a cherry-pick completes.
type CherryPickSuccessMsg struct{ Hash string }

// CherryPickConflictMsg is sent when a cherry-pick results in conflicts.
type CherryPickConflictMsg struct{ Hash string }

// RevertSuccessMsg is sent when a revert completes.
type RevertSuccessMsg struct{ Hash string }

// StashSuccessMsg is sent when changes are stashed.
type StashSuccessMsg struct{}

// StashPopMsg is sent when a stash entry is popped/applied.
type StashPopMsg struct{ Index int }

// StashDropMsg is sent when a stash entry is dropped.
type StashDropMsg struct{ Index int }

// StashListMsg carries the list of stash entries.
type StashListMsg struct{ Entries []domain.StashEntry }

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

// SQLResultMsg carries the result of a SQL query to display.
type SQLResultMsg struct {
	Query  string
	Result string
}

// CommitDetailMsg carries the diff stat for a commit's changed tables.
type CommitDetailMsg struct {
	Hash   string
	Header string // commit metadata header
	Tables []domain.DiffStatEntry
}

// BrowserDataMsg carries table data to the browser.
type BrowserDataMsg struct {
	Table   string
	Columns []string
	Rows    []map[string]interface{}
	Total   int
	Offset  int
}
