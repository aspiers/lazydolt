// Package domain defines pure data types shared between the dolt command
// layer and the TUI layer. This package has no external dependencies.
package domain

import "time"

// StatusEntry represents a table's status in the dolt working set.
// Maps to columns from the dolt_status system table.
type StatusEntry struct {
	TableName string `json:"table_name"`
	Staged    bool   `json:"staged"`
	Status    string `json:"status"` // "new table", "modified", "deleted", "renamed", "conflict"
}

// Table represents a table in the working set with optional change status.
type Table struct {
	Name   string
	Status *StatusEntry // nil if table is clean (no uncommitted changes)
}

// Branch represents a dolt branch.
// Maps to columns from the dolt_branches system table.
type Branch struct {
	Name          string    `json:"name"`
	Hash          string    `json:"hash"`
	IsCurrent     bool      // set by comparing with active_branch()
	LatestMessage string    `json:"latest_commit_message"`
	LatestAuthor  string    `json:"latest_committer"`
	LatestDate    time.Time `json:"latest_commit_date"`
}

// Commit represents a single commit in the dolt log.
// Maps to columns from the dolt_log system table.
type Commit struct {
	Hash    string    `json:"commit_hash"`
	Message string    `json:"message"`
	Author  string    `json:"committer"`
	Email   string    `json:"email"`
	Date    time.Time `json:"date"`
}

// Tag represents a dolt tag.
// Maps to columns from the dolt_tags system table.
type Tag struct {
	Name    string    `json:"tag_name"`
	Hash    string    `json:"tag_hash"`
	Tagger  string    `json:"tagger"`
	Email   string    `json:"email"`
	Date    time.Time `json:"date"`
	Message string    `json:"message"`
}

// DiffType classifies how a row changed.
type DiffType string

const (
	DiffAdded    DiffType = "added"
	DiffModified DiffType = "modified"
	DiffDeleted  DiffType = "deleted"
)

// DiffRow represents a single changed row in a table diff.
type DiffRow struct {
	Type       DiffType
	FromValues map[string]interface{} // column -> old value (nil for added rows)
	ToValues   map[string]interface{} // column -> new value (nil for deleted rows)
}

// DiffSummary holds the diff output for a single table.
type DiffSummary struct {
	TableName string
	Rows      []DiffRow
	RawDiff   string // raw text output from 'dolt diff'
}

// Schema holds the CREATE TABLE statement for a table.
type Schema struct {
	TableName       string
	CreateStatement string // raw CREATE TABLE from 'dolt schema show'
}

// ConflictSummary holds conflict data for a table after a merge.
type ConflictSummary struct {
	TableName    string
	NumConflicts int
	RawOutput    string // raw text from 'dolt conflicts cat <table>'
}

// StashEntry represents a single stash entry.
type StashEntry struct {
	Index   int    // 0-based index (stash@{N})
	Branch  string // branch name when stashed
	Hash    string // commit hash
	Message string // commit message at stash time
}

// CommandLogEntry records a single dolt CLI invocation and its result.
type CommandLogEntry struct {
	Command string    // full command (e.g. "dolt add users")
	Output  string    // stdout/stderr output (truncated for display)
	Error   bool      // true if command failed
	Time    time.Time // when the command was executed
}

// DiffStatEntry holds row-level change statistics for a single table
// between two revisions.
type DiffStatEntry struct {
	TableName    string `json:"table_name"`
	RowsAdded    int    `json:"rows_added"`
	RowsDeleted  int    `json:"rows_deleted"`
	RowsModified int    `json:"rows_modified"`
}

// QueryResult holds the result of an arbitrary SQL query.
type QueryResult struct {
	Columns []string                 // column names in display order
	Rows    []map[string]interface{} // each row as column -> value
	Total   int                      // total row count (for pagination)
}
