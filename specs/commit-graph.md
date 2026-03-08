# Non-Linear Commit History in the Commits Panel

## Status: Draft — needs human review before implementation

## Context

The commits panel currently shows a flat linear list from `dolt_log`, which
only includes commits reachable from HEAD. When branches diverge and merge,
users need to see the relationship between commits, similar to lazygit's
commit graph.

## Current State

### Data model
- `domain.Commit` has: Hash, Message, Author, Email, Date
- No parent hash information, no branch ref decoration
- `dolt.Runner.Log()` queries `dolt_log ORDER BY date DESC`

### Rendering
- `CommitsModel.View()` renders: `hash initials message time`
- Flat list, one commit per line, no graph characters

### Available dolt data sources

1. **`dolt_log`** — commits reachable from HEAD (current branch only)
   - Columns: commit_hash, committer, email, date, message, commit_order

2. **`dolt_commit_ancestors`** — parent-child relationships for ALL commits
   - Columns: commit_hash, parent_hash, parent_index
   - `parent_index` > 0 indicates merge commits

3. **`dolt_branches`** — branch tip commit hashes
   - Already loaded for the Branches panel

4. **`dolt log --graph --all --oneline`** — CLI graph visualization
   - Produces ASCII art graph with `*`, `|`, `/`, `\` characters
   - Shows branch decorations like `(HEAD -> main)`

## Design Options

### Option A: Parse `dolt log --graph` output (pragmatic)

Use `dolt log --graph --all --oneline` and parse the ASCII graph characters.
The graph is already computed by dolt — we just need to display it.

**Pros**: Minimal code, leverages dolt's graph algorithm
**Cons**: Fragile parsing, dolt output format may change, limited control
over styling, hard to add interactive features (like filtering by branch)

### Option B: Compute graph from `dolt_commit_ancestors` (recommended)

Query the commit ancestry DAG and compute the graph layout ourselves.
This is what lazygit does (using `git log --oneline --graph` as data
source, then computing its own layout).

**Pros**: Full control over rendering, can style per-branch, can filter,
can do interactive features (collapse branches, etc.)
**Cons**: More code, need a graph layout algorithm

### Option C: Hybrid — query ancestors, use existing flat list as fallback

Start with Option B's data loading but render simply at first (just adding
parent info and branch decorations), then add graph characters in Phase 2.

**Recommended: Option C (incremental approach)**

## Implementation Plan

### Phase 1: Add parent hashes and branch decorations to commits

**Domain changes** (`internal/domain/types.go`):
```go
type Commit struct {
    Hash       string    `json:"commit_hash"`
    Message    string    `json:"message"`
    Author     string    `json:"committer"`
    Email      string    `json:"email"`
    Date       time.Time `json:"date"`
    Parents    []string  // parent commit hashes
    Refs       []string  // branch/tag names pointing at this commit
    IsMerge    bool      // len(Parents) > 1
}
```

**Data loading** (`internal/dolt/log.go`):

Replace the single `dolt_log` query with:

1. Query `dolt_log` for all branches (`--all` equivalent):
   ```sql
   SELECT DISTINCT l.commit_hash, l.committer, l.email, l.date, l.message
   FROM dolt_log AS l
   UNION
   SELECT DISTINCT l.commit_hash, l.committer, l.email, l.date, l.message
   FROM dolt_branches b
   CROSS JOIN dolt_log(b.name) l
   ORDER BY date DESC
   LIMIT ?
   ```
   Note: Need to verify if `dolt_log(branch_name)` syntax works.
   Alternative: iterate branches and query `dolt_log` for each.

2. Query `dolt_commit_ancestors` for parent relationships:
   ```sql
   SELECT commit_hash, parent_hash, parent_index
   FROM dolt_commit_ancestors
   WHERE commit_hash IN (?)
   ```

3. Cross-reference with `dolt_branches` to add ref decorations.

**Rendering** — add branch decoration after hash:
```
6thro8q AS (HEAD → main) Update product pricing 1h ago
9ppqlvd AS (feature-branch) Add bio column 1h ago
```

### Phase 2: Simple branch indicators

Add a single character column before the hash to indicate branching:

```
* 6thro8q AS Update product pricing         1h ago
* 8qbdguq AS Add bulk order data            1h ago
| * 9ppqlvd AS Add bio column               1h ago
* | s4b93c9 AS Update user profiles         1h ago
|/
* 40jqr6j AS Add products table             1h ago
```

This requires computing which "column" each commit's branch occupies
in the graph layout.

### Phase 3: Full graph layout algorithm

Implement a proper DAG layout:

1. **Assign columns**: Each active branch gets a column index.
   When a branch starts, allocate the next available column.
   When branches merge, free the merged column.

2. **Draw connectors**: Between rows, draw `│`, `╭`, `╯`, `─`
   characters to show parent-child relationships.

3. **Color per-branch**: Each column/branch gets a distinct color,
   matching lazygit's approach.

The algorithm is well-documented in lazygit's source:
- `pkg/gui/controllers/local_commits_controller.go`
- `pkg/commands/git_commands/commit_loader.go`

Key data structure:
```go
type GraphLine struct {
    Commit  *domain.Commit
    Columns []GraphCell  // what to draw in each column at this row
}

type GraphCell struct {
    Type  CellType  // Commit, Pipe, MergeTo, MergeFrom, Empty
    Color int       // ANSI color index
}
```

### Phase 4: Branch filtering

Allow users to filter which branches appear in the graph:
- Show only current branch (current behavior)
- Show all branches (new default)
- Show specific branches (toggle with keybindings)

Keybinding: `b` to cycle between "current branch" / "all branches"

### Phase 5: Interactive features

- Collapse/expand branches in the graph
- Cherry-pick indicators
- Rebase indicators (if dolt supports rebase)

## Risks and Open Questions

1. **`dolt_log` scoping**: `dolt_log` only shows commits reachable from
   HEAD by default. Need to verify how to query commits from other
   branches. Options: `dolt_log(branch_name)`, `DOLT_LOG('branch')`,
   or use `dolt log --all` CLI output.

2. **Performance with many branches**: If there are many branches with
   long histories, querying all commits could be slow. Consider limiting
   to the N most recent commits across all branches.

3. **Merge commits in dolt**: Dolt supports merges. Need to verify
   how merge commits appear in `dolt_commit_ancestors` (they should
   have `parent_index` 0 and 1).

4. **Graph layout complexity**: Computing a clean graph layout is
   non-trivial. Consider using a simplified algorithm first (just
   showing which column each branch occupies) before implementing
   full connector drawing.

5. **Panel width**: Graph characters take horizontal space. With many
   branches, the graph could consume significant panel width. Need
   to limit the number of visible columns and truncate gracefully.

## Dependencies

- None — this is independent of other current work items.

## References

- lazygit graph rendering: `pkg/gui/presentation/graph.go`
- git-graph algorithm paper: "Drawing Git Graphs" by Edsko de Vries
- dolt system tables docs: https://docs.dolthub.com/sql-reference/version-control/dolt-system-tables
