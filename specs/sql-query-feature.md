# Interactive SQL Query Feature — lazysql Overlap Analysis

## Status: Draft — needs human review before implementation

## lazysql Analysis

[lazysql](https://github.com/jorgerojas26/lazysql) (3.6k stars, Go, tview-based)
is a cross-platform TUI database management tool. Key features:

### What lazysql does well
- **Multi-database support**: MySQL, PostgreSQL, SQLite, MSSQL, MongoDB
- **Connection management**: Multiple saved connections with custom commands
  (SSH tunnels, port forwarding, k8s kubectl)
- **SQL editor**: Built-in editor (Ctrl+E) with Ctrl+R to execute
- **Table browsing**: Tree navigation, pagination, column/row editing
- **CRUD operations**: Inline cell editing, row insertion/deletion, sidebar
  record editor, pending changes queue with Ctrl+S to commit
- **Data export**: CSV export with batch size control
- **Filtering**: WHERE-clause filter input per table
- **Sorting**: J/K to sort ascending/descending by column
- **Multiple tabs**: Open several tables simultaneously
- **JSON viewer**: View cell/row data as JSON (z/Z keys)
- **Query history**: Ctrl+_ to toggle saved query history
- **Read-only mode**: `--read-only` flag blocks mutations
- **Customizable keybindings**: TOML config with per-context key overrides

### What lazysql does NOT do
- No version control features (no diffs, no branches, no commits)
- No schema versioning or migration tracking
- No merge conflict resolution
- No commit history browsing
- No staging/unstaging workflow
- No branch management

### Architecture notes
- Built with **tview** (not Bubble Tea) — different widget ecosystem
- Uses database driver abstraction (`drivers/` package)
- Does NOT support dolt natively as a database driver
- Could potentially connect to dolt's MySQL-compatible server, but would
  only see it as a regular MySQL database (no dolt-specific features)

## Overlap Assessment

### Clear separation of concerns

| Feature Area | lazydolt | lazysql |
|---|---|---|
| Version control (diff, commit, branch, merge) | **Core focus** | Not supported |
| SQL browsing (SELECT, explore data) | Basic (Browse tab) | **Core focus** |
| SQL editing (INSERT, UPDATE, DELETE) | Not supported | **Core focus** |
| Schema browsing | Basic (Schema tab) | Supported |
| Connection management | Single repo | Multi-connection |
| Dolt-specific features | **Full** | None |

### Verdict: Minimal overlap, complementary tools

lazydolt and lazysql serve fundamentally different purposes:
- **lazydolt** = version control for dolt databases (like lazygit for git)
- **lazysql** = general-purpose SQL client (like DataGrip but TUI)

A user would realistically use **both** tools:
1. lazydolt to manage branches, review diffs, stage changes, commit
2. lazysql (or any SQL client) to browse/edit actual data

## Recommendation: Don't duplicate lazysql

**Do NOT build a full interactive SQL editor into lazydolt.** Instead:

### Keep what we have (the Browse tab)
The existing Browse tab provides read-only table viewing with pagination.
This is sufficient for the "glance at data while reviewing changes" use case.
It already supports:
- Column display with auto-sizing
- Pagination with [/] keys
- NULL value styling

### Add a minimal SQL input (Phase 1)
A simple one-line SQL input (not a full editor) for quick ad-hoc queries:

- Keybinding: `:` (vim command mode style) or `Ctrl+E`
- Single-line text input with history (up/down arrow)
- Results display in the Browse tab
- Read-only: only SELECT queries, no mutations
  (mutations should go through dolt's proper workflow: edit → stage → commit)

### Add dolt-specific SQL features (Phase 2)
Things only lazydolt can do, that lazysql cannot:

- **Time-travel queries**: `SELECT * FROM users AS OF 'commit_hash'`
- **Cross-branch queries**: `SELECT * FROM \`branch/table\``
- **Diff queries**: `SELECT * FROM dolt_diff_users WHERE to_commit = 'abc123'`
- **System table browsing**: dolt_status, dolt_branches, dolt_log, dolt_conflicts
- **Merge preview**: Show what would change in a merge before executing it

These are uniquely valuable because no other TUI tool can do them.

### Recommend lazysql for heavy SQL work
In the README and help text, suggest lazysql for full SQL editing:

```
For interactive SQL editing, try lazysql:
  go install github.com/jorgerojas26/lazysql@latest
  lazysql mysql://root:@localhost:3306/mydb
```

Users can connect lazysql to dolt's MySQL-compatible server
(`dolt sql-server`) for full CRUD operations.

## Implementation Plan

### Phase 1: Quick SQL input
- Add `:` keybinding to open a one-line text input
- Execute via `dolt sql -r json -q "..."`
- Display results in Browse tab
- History with up/down arrows (store last N queries)
- Only allow SELECT/SHOW/DESCRIBE (reject INSERT/UPDATE/DELETE/DROP)

### Phase 2: Dolt-specific query helpers
- `AS OF` selector in Browse tab (pick a commit from history)
- Cross-branch table viewing (pick a branch, see its data)
- dolt_diff browsing UI (see structured diffs, not just text)
- Conflict resolution UI (when merges conflict)

### Phase 3: Query history persistence
- Save query history to a file in the repo (.lazydolt/history)
- Load on startup
- Search through history

## Dependencies
- Phase 1 requires the Browse tab (already exists)
- Phase 2 requires commit history (exists) and branch management (exists)
