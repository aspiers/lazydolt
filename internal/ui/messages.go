// Package ui implements the Bubble Tea TUI for lazydolt.
package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/aspiers/lazydolt/internal/domain"
)

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
	Tags     []domain.Tag
	Remotes  []domain.Remote
}

// AddRemoteSuccessMsg is sent when a remote is added.
type AddRemoteSuccessMsg struct{ Name string }

// DeleteRemoteSuccessMsg is sent when a remote is removed.
type DeleteRemoteSuccessMsg struct{ Name string }

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

// TagSuccessMsg is sent when a tag is created.
type TagSuccessMsg struct{ Name string }

// DeleteTagSuccessMsg is sent when a tag is deleted.
type DeleteTagSuccessMsg struct{ Name string }

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

// BlameContentMsg carries blame text to the main panel.
type BlameContentMsg struct {
	Table   string
	Content string
}

// ReflogContentMsg carries reflog text to the main panel.
type ReflogContentMsg struct {
	Content string
}

// TableRenameSuccessMsg is sent when a table is renamed.
type TableRenameSuccessMsg struct{ OldName, NewName string }

// TableCopySuccessMsg is sent when a table is copied.
type TableCopySuccessMsg struct{ SrcName, DstName string }

// TableDropSuccessMsg is sent when a table is dropped.
type TableDropSuccessMsg struct{ Name string }

// TableExportSuccessMsg is sent when a table is exported.
type TableExportSuccessMsg struct{ Table, Path string }

// RebaseSuccessMsg is sent when a rebase completes successfully.
type RebaseSuccessMsg struct{ Branch string }

// RebaseConflictMsg is sent when a rebase results in conflicts.
type RebaseConflictMsg struct{ Branch string }

// RebaseAbortMsg is sent when a rebase is aborted.
type RebaseAbortMsg struct{}

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

// UndoSuccessMsg is sent when an undo operation succeeds.
type UndoSuccessMsg struct{ Hash string }

// RedoSuccessMsg is sent when a redo operation succeeds.
type RedoSuccessMsg struct{ Hash string }

// flashTimeoutMsg is sent after a delay to clear the flash message.
type flashTimeoutMsg struct{ ID int }

// undoableResultMsg wraps a mutation result with pre-mutation state for undo.
// This is unexported because it's only used internally by the App.
type undoableResultMsg struct {
	Inner tea.Msg          // the original success/error message
	Entry domain.UndoEntry // pre-mutation state to push onto undo stack
}

// undoResultMsg is sent when an undo operation succeeds, carrying the
// state to push onto the redo stack.
type undoResultMsg struct {
	RedoEntry  domain.UndoEntry
	TargetHash string // the hash we reset to
}

// redoResultMsg is sent when a redo operation succeeds, carrying the
// state to push onto the undo stack.
type redoResultMsg struct {
	UndoEntry  domain.UndoEntry
	TargetHash string // the hash we reset to
}
