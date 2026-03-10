package dolt

import "github.com/aspiers/lazydolt/internal/domain"

// Compile-time check that CLIRunner implements Runner.
var _ Runner = (*CLIRunner)(nil)

// Runner defines the interface for executing dolt operations.
// CLIRunner implements this by spawning dolt CLI subprocesses.
// Future implementations (e.g. SQLRunner) may connect to a
// dolt sql-server via MySQL protocol for better performance.
type Runner interface {
	// RepoDir returns the absolute path to the dolt repository.
	RepoDir() string

	// Low-level execution (CLI-specific but used by callers for
	// ad-hoc commands). Future implementations should provide
	// equivalent behavior via SQL where possible.
	Exec(args ...string) (string, error)
	ExecRaw(args ...string) (string, error)

	// SQL queries
	SQL(query string) ([]map[string]interface{}, error)
	SQLRaw(query string) (string, error)
	Query(sql string) (*domain.QueryResult, error)
	QueryPage(table string, limit, offset int) (*domain.QueryResult, error)

	// Status and branches
	Status() ([]domain.StatusEntry, error)
	CurrentBranch() (string, error)
	IsDirty() (bool, error)
	Branches(order BranchOrderBy) ([]domain.Branch, error)

	// Commit log
	Log(branch string, limit int, order CommitOrderBy, filters ...CommitFilter) ([]domain.Commit, error)
	HeadHash() (string, error)
	Reflog() (string, error)
	ReflogEntries() ([]domain.ReflogEntry, error)

	// Tables
	Tables() ([]domain.Table, error)
	Schema(table string) (domain.Schema, error)

	// Tags
	Tags() ([]domain.Tag, error)
	CreateTag(name, ref, message string) error
	DeleteTag(name string) error

	// Remotes
	Remotes() ([]domain.Remote, error)
	RemoteAdd(name, url string) error
	RemoteRemove(name string) error

	// Diff
	DiffText(table string, staged bool) (string, error)
	DiffSchema(table string, staged bool) (string, error)
	DiffSchemaRefs(fromRef, toRef, table string) (string, error)
	DiffRefs(fromRef, toRef, table string) (string, error)
	DiffStat(table string, staged bool) (string, error)
	DiffStatRefs(fromRef, toRef, table string) (string, error)
	DiffStatBetween(fromRef, toRef string) ([]domain.DiffStatEntry, error)
	QueryDiff(query1, query2 string) (string, error)

	// Staging and mutations
	Add(table string) error
	AddAll() error
	Reset(table string) error
	ResetAll() error
	ResetSoft(commit string) error
	ResetHard(commit string) error
	Commit(message string) (string, error)
	CommitAmend(message string) (string, error)
	Checkout(branch string) error
	CheckoutTable(table string) error
	NewBranch(name string) error
	RenameBranch(oldName, newName string) error
	DeleteBranch(name string) error

	// Merge, rebase, cherry-pick
	Merge(branch string) (string, error)
	MergeSquash(branch string) (string, error)
	MergeAbort() error
	CherryPick(hash string) (string, error)
	CherryPickAbort() error
	Revert(hash string) (string, error)
	Rebase(upstream string) (string, error)
	RebaseAbort() error
	RebaseContinue() (string, error)

	// Remote operations
	Push() (string, error)
	Pull() (string, error)
	Fetch() (string, error)

	// Conflicts
	Conflicts() ([]domain.ConflictSummary, error)
	ConflictsCat(table string) (string, error)
	ConflictsResolveOurs(table string) error
	ConflictsResolveTheirs(table string) error

	// Stash
	Stash() error
	StashList() ([]domain.StashEntry, error)
	StashPop(index int) error
	StashDrop(index int) error

	// Table operations
	TableRename(oldName, newName string) error
	TableCopy(srcName, dstName string) error
	TableDrop(name string) error
	TableExport(table, filePath string) error

	// Blame
	Blame(table string) (string, error)

	// Config
	Config() (global []ConfigEntry, local []ConfigEntry, err error)
	ConfigSet(global bool, key, value string) error

	// Dump
	Dump(format DumpFormat, filename string, force bool) error

	// Command log (for TUI display)
	CommandLog() []domain.CommandLogEntry

	// Close releases resources (e.g. database connections, child processes).
	// CLIRunner's Close is a no-op. SQLRunner stops the sql-server process.
	Close() error
}
