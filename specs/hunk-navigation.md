# Hunk-Based Navigation for Diff Viewer

## Status: Draft — needs human review before implementation

## Context

The diff viewer currently shows raw `dolt diff` output in a scrollable viewport.
Users can scroll line-by-line with j/k but have no way to jump between logical
change groups ("hunks"). This is a key UX gap compared to lazygit.

## Dolt's Diff Format vs Unified Diff

Dolt does **not** use unified diff format. Instead it outputs:

### Schema changes (unified-diff-like)
```
diff --dolt a/orders b/orders
--- a/orders
+++ b/orders
 CREATE TABLE `orders` (
   ...
-) ENGINE=InnoDB AUTO_INCREMENT=5 ...
+) ENGINE=InnoDB AUTO_INCREMENT=6 ...
```

### Data changes (tabular format)
```
+---+----+---------+--------+---------+---------------------+
|   | id | user_id | amount | status  | created_at          |
+---+----+---------+--------+---------+---------------------+
| + | 5  | 1       | 199.99 | pending | 2026-03-07 23:21:52 |
| < | 1  | Widget  | 19.99  | gadgets | 1                   |
| > | 1  | Widget  | 21.99  | gadgets | 1                   |
+---+----+---------+--------+---------+---------------------+
```

Markers:
- `| + |` — row added
- `| - |` — row deleted
- `| < |` — old value (before modification)
- `| > |` — new value (after modification)

### Key difference from unified diff
- No `@@ -line,count +line,count @@` hunk headers
- Schema changes use `+`/`-` line prefixes (like unified diff)
- Data changes use the tabular `| marker |` format
- A single `dolt diff` output may contain multiple tables, each starting
  with `diff --dolt a/table b/table`

## Proposed "Hunk" Concept for Dolt

Since dolt doesn't have native hunk headers, we define hunks as:

### Level 1: Table sections
Each `diff --dolt a/table b/table` block is a top-level section.
This is already partially supported — users can select individual tables
in the Tables panel and see per-table diffs.

### Level 2: Change type sections within a table
Within each table diff, there are logically distinct sections:
1. **Schema changes** — the `CREATE TABLE` block with `+`/`-` lines
2. **Data changes** — the tabular block with `| +/-/</>  |` rows

### Level 3: Individual change groups in data
Within the data table, contiguous groups of changed rows form hunks:
- A group of `| + |` rows (insertions)
- A pair of `| < |`/`| > |` rows (modifications)
- A group of `| - |` rows (deletions)
- Separated by unchanged `|   |` rows or table borders

## Proposed Keybindings

| Key | Action |
|-----|--------|
| `n` | Jump to next hunk (next change group) |
| `N` | Jump to previous hunk |
| `{` | Jump to next table section (`diff --dolt`) |
| `}` | Jump to previous table section |

These keys only work when the main panel (PanelMain) is focused.

## Implementation Plan

### Phase 1: Hunk detection (parser)

Create `internal/ui/components/diffhunks.go`:

```go
type DiffHunk struct {
    StartLine int    // 0-based line index in the diff content
    EndLine   int    // exclusive end
    Table     string // which table this hunk belongs to
    Type      string // "schema", "added", "modified", "deleted"
}

// ParseHunks analyzes colorized diff content and returns hunk positions.
func ParseHunks(rawDiff string) []DiffHunk
```

Hunk detection rules:
1. `diff --dolt a/X b/X` starts a new table section
2. Within schema section: contiguous `+`/`-` lines form a hunk
3. Within data section: contiguous changed rows (`| + |`, `| - |`,
   `| < |`, `| > |`) separated by separator rows (`+---+`) or
   unchanged rows form hunks

### Phase 2: Navigation

In `DiffView`, track `hunks []DiffHunk` and `currentHunk int`.

When `n`/`N` is pressed:
1. Find which hunk the current viewport top line falls in (or after)
2. Scroll to the start of the next/previous hunk
3. Update the hunk counter display

### Phase 3: Hunk count indicator

Add a hunk counter to the main panel title:
```
Diff: orders (3/7 hunks)
```

### Phase 4 (future): Hunk-level staging

This is significantly more complex because dolt's `add` command works
at the table level, not the hunk level. Hunk-level staging would require:
- Constructing SQL statements to apply/revert individual row changes
- Managing a "partial stage" state that dolt doesn't natively support
- Significant complexity for `<`/`>` (modification) hunks

**Recommendation**: Defer hunk-level staging. It requires deep integration
with dolt's SQL engine and is a large feature on its own. Focus on
navigation first.

### Phase 5 (future): Hunk folding/expanding

Allow collapsing unchanged sections between hunks to reduce visual noise.
Show a `... N lines hidden ...` indicator. This is purely a UI feature
and doesn't require any dolt integration.

## Risks and Open Questions

1. **Multi-table combined diffs**: When viewing "all unstaged" or "all staged",
   the diff contains multiple tables. Should `n`/`N` jump across table
   boundaries, or only within the current table? **Recommendation**: Jump
   across table boundaries — the user chose to view all tables together.

2. **Schema-only changes**: Some diffs only have schema changes (e.g.,
   adding a column). The hunk parser needs to handle schema-only diffs.

3. **Empty hunks**: After staging/unstaging, the diff content refreshes.
   The hunk list must be re-parsed on every `SetContent` call.

4. **Performance**: Dolt diffs for large tables could be thousands of lines.
   Hunk parsing should be O(n) in the number of lines — a single pass.

## Dependencies

- Requires PanelMain focus (lazydolt-0id) — **done**
- Requires diff scrolling to work (lazydolt-0id) — **done**
