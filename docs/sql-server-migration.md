# SQL Server Migration: Design & Findings

## Summary

Migrating lazydolt from spawning a new `dolt` subprocess per query to
connecting to a persistent `dolt sql-server` via MySQL protocol yields
**100-230x speedups** on individual operations. This document captures
the research, benchmark results, and implementation plan.

## Benchmark Results

Measured on the test database (`tmp/test-dolt`, 10 commits, 3 tables):

| Operation     | CLI (avg) | SQL Server (avg) | Speedup |
| ------------- | --------- | ---------------- | ------- |
| active_branch | 262ms     | 1.2ms            | 227x    |
| dolt_status   | 214ms     | 1.7ms            | 124x    |
| dolt_branches | 213ms     | 1.2ms            | 183x    |
| dolt_log_10   | 228ms     | 1.2ms            | 188x    |
| show_tables   | 252ms     | 1.2ms            | 216x    |

Stored procedures also work with sub-10ms latency:
- `CALL dolt_add('.')`: 7ms
- `CALL dolt_reset()`: 5ms
- `SHOW CREATE TABLE`: 2ms
- `dolt_diff_stat()`: 5ms

Benchmark code: `tmp/sql_server_bench_test.go`

## Impact on lazydolt

Currently `loadData()` fires 8 CLI calls in 6 goroutines (~400ms wall clock).
With sql-server, each query drops from ~200ms to ~2ms. Even running all 8
sequentially would take ~16ms. The full refresh would drop from ~400ms to
under 20ms — imperceptible to the user.

The selective `loadWorkingSet()` (2 CLI calls, ~200ms) would drop to ~4ms.

## Operation Classification

### SQL-Replaceable (58 of 70 methods)

**Already SQL** (13 methods): `SQL`, `SQLRaw`, `DiffStatBetween`, `Status`,
`CurrentBranch`, `IsDirty`, `Branches`, `Log`, `Tags`, `Conflicts`, `Query`,
`QueryPage`, `HeadHash`

**CLI → `CALL dolt_*()`** (29 methods): `Add`, `AddAll`, `Reset`, `ResetAll`,
`ResetSoft`, `ResetHard`, `Commit`, `CommitAmend`, `Checkout`, `CheckoutTable`,
`NewBranch`, `RenameBranch`, `DeleteBranch`, `Merge`, `MergeSquash`,
`MergeAbort`, `CherryPick`, `CherryPickAbort`, `Revert`, `Push`, `Pull`,
`Fetch`, `CreateTag`, `DeleteTag`, `ConflictsResolveOurs`,
`ConflictsResolveTheirs`, `Rebase`, `RebaseAbort`, `RebaseContinue`

**CLI → SQL system tables** (7 methods): `listTableNames` (`SHOW TABLES`),
`Tables`, `Schema` (`SHOW CREATE TABLE`), `Remotes` (`SELECT FROM dolt_remotes`),
`Reflog` (`SELECT FROM dolt_reflog`), `ConflictsCat` (`SELECT FROM dolt_conflicts_<table>`)

**CLI → SQL with client-side formatting** (9 methods): `DiffText`, `DiffSchema`,
`DiffSchemaRefs`, `DiffRefs`, `DiffStatRefs`, `DiffStat`, `TableRename`
(`ALTER TABLE RENAME`), `TableCopy` (`CREATE TABLE AS SELECT`), `TableDrop`
(`DROP TABLE`)

### CLI-Only (12 methods)

| Method          | Why CLI-only                              |
| --------------- | ----------------------------------------- |
| `Config()`      | Reads on-disk config files                |
| `ConfigSet()`   | Writes on-disk config files               |
| `Dump()`        | Bulk filesystem export                    |
| `TableExport()` | Writes to local filesystem                |
| `Blame()`       | Unique text format (borderline — has SQL) |
| `RemoteAdd()`   | No SQL stored procedure                   |
| `RemoteRemove()`| No SQL stored procedure                   |
| `Stash()`       | No SQL equivalent                         |
| `StashList()`   | No SQL equivalent                         |
| `StashPop()`    | No SQL equivalent                         |
| `StashDrop()`   | No SQL equivalent                         |
| `QueryDiff()`   | CLI-specific comparison tool              |

## Architecture Design

### Phase 1: Introduce Runner Interface + SQL-backed Implementation

Extract a `Runner` interface from the current concrete struct. This is the
key abstraction that enables the migration without rewriting all callers.

```go
// internal/dolt/runner.go

// Runner defines the interface for executing dolt operations.
type Runner interface {
    // Queries
    SQL(query string) ([]map[string]interface{}, error)
    SQLRaw(query string) (string, error)
    Status() ([]domain.TableStatus, error)
    CurrentBranch() (string, error)
    Branches(order string) ([]domain.Branch, error)
    Log(branch string, limit int, order string, filters LogFilters) ([]domain.Commit, error)
    // ... all other methods

    // Lifecycle
    CommandLog() []domain.CommandLogEntry
    Close() error
}
```

Two implementations:
- `CLIRunner` — current implementation (renamed from `Runner`)
- `SQLRunner` — uses `database/sql` + `go-sql-driver/mysql`

### Phase 2: SQL Server Lifecycle Management

lazydolt manages its own sql-server process:

1. **Startup**: On `NewSQLRunner(repoDir)`:
   - Check `.dolt/sql-server.info` for existing server
   - If found and process alive, connect to it
   - Otherwise, start `dolt sql-server` on a random high port with
     `--socket /tmp/lazydolt-<pid>.sock`
   - Wait for connection readiness (retry loop)

2. **Shutdown**: On `Close()`:
   - If we started the server, send SIGTERM
   - Clean up socket file

3. **Fallback**: If sql-server fails to start or connection drops:
   - Fall back to `CLIRunner` transparently
   - Log a warning in the command log

### Phase 3: Migrate Methods Incrementally

Migrate methods in priority order (highest frequency first):

1. `Status()`, `CurrentBranch()`, `Branches()`, `Tables()`/`listTableNames()`
   — called on every refresh
2. `Log()`, `Tags()`, `Remotes()` — called on every refresh
3. `Add()`, `Reset()`, `Commit()`, etc. — mutation operations
4. `Diff*()` methods — requires client-side diff formatting
5. CLI-only methods delegate to embedded `CLIRunner`

### Phase 4: Diff Rendering via SQL

The diff methods are the most complex migration. Options:
- **A**: Query `dolt_diff_<table>` and reconstruct the text format client-side
- **B**: Keep using CLI for diff text, SQL for everything else
- **C**: Render structured diff data in a richer TUI format (tables, not text)

Recommend starting with **B** (hybrid) and exploring **C** later.

## Key Design Decisions

### Unix Socket vs TCP

Use unix socket when available (Linux/macOS) — lower latency, no port conflicts.
Fall back to TCP on `localhost:random-port` for Windows compatibility.

### Server Lifetime

Two options:
1. **lazydolt-managed**: Start on launch, stop on exit. Simple lifecycle.
2. **Long-running**: Detect/reuse existing server. Better for rapid restarts.

Recommend option 1 for initial implementation, with option 2 as enhancement.

### Connection Pooling

`database/sql` provides built-in connection pooling. Set:
- `MaxOpenConns = 4` (matches current goroutine parallelism)
- `MaxIdleConns = 2`
- `ConnMaxLifetime = 0` (no timeout — server lifetime matches app)

### Error Handling

SQL errors from stored procedures differ from CLI exit codes. Need to:
- Parse MySQL error codes for dolt-specific errors
- Map to existing error types used by the UI layer
- Preserve error message formatting for flash messages

## Implementation Effort Estimate

| Phase | Effort | Risk |
| ----- | ------ | ---- |
| 1: Runner interface | Small (1-2 sessions) | Low — pure refactor |
| 2: Server lifecycle | Medium (1 session) | Medium — process mgmt |
| 3a: Read queries | Small (1 session) | Low — direct mapping |
| 3b: Mutations | Small (1 session) | Low — stored procs |
| 3c: Diff methods | Large (2-3 sessions) | High — format compat |
| 4: Polish & testing | Medium (1-2 sessions) | Medium |

Total: ~7-10 sessions for full migration.

## Dependencies

- `github.com/go-sql-driver/mysql` — already added to go.mod
- Dolt version: ≥ 1.0 (for stored procedures and system tables)

## Open Questions

1. Should lazydolt always start its own sql-server, or prefer connecting
   to an existing one? (Security: existing server may have auth enabled)
2. How to handle `dolt_transaction_commit` behavior — should lazydolt
   create Dolt commits explicitly or rely on auto-commit?
3. Should the Runner interface be in `internal/dolt/` or a new
   `internal/dolt/driver/` package?
