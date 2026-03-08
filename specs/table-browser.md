# Interactive Table Browser — Feature Plan

## Status: Draft — needs human review before implementation

## Context

The existing Browse tab (`BrowserView`) provides basic read-only table viewing
with text-based formatting. This spec plans enhancements to make the table
browser more interactive and useful, while staying true to lazydolt's identity
as a **version control tool** (not a full SQL client — see sql-query-feature.md).

## Current State

### What exists today
- `BrowserView` wraps a `viewport.Model` for scrollable content
- `renderTable()` formats data as a fixed-width text table
- Pagination with `[`/`]` keys (100 rows per page)
- Column auto-sizing (capped at 40 chars)
- NULL value styling (dim italic)
- Accessible via `Enter` on a table in the Tables panel, or `]` to cycle tabs

### Limitations
- Text-based rendering — no interactive cell selection
- No column sorting
- No filtering
- No column resizing
- No copy to clipboard
- No horizontal scrolling of wide tables (columns get truncated)
- No row/column count display (only total rows shown)

## Proposed Enhancements

### Tier 1: Quick wins (low complexity, high value)

#### 1.1 Column sorting
- Press `s` on a column header to sort ascending, `S` for descending
- Re-query with `ORDER BY column ASC/DESC`
- Show sort indicator (▲/▼) in column header
- Implementation: add sort state to BrowserView, modify the query in
  `loadTableDataPage`

#### 1.2 Row/column count display
- Show `Row 1-100 of 1,234 | 12 columns` in the Browse tab title
- Already have `Total` and `Offset` in BrowserView
- Add column count from `len(Columns)`

#### 1.3 Better column truncation
- When a table is wider than the viewport, truncate from the right
- Show a `→` indicator when columns are hidden
- Consider: allow horizontal scrolling of the table content (h/l already
  work for the viewport, but the rendered text table itself needs to be
  wider than the viewport)

#### 1.4 Copy cell value
- `y` to copy the value under cursor to clipboard
- Requires adding a cursor (row + column position) to BrowserView
- Use `atotto/clipboard` or similar Go clipboard library

### Tier 2: Medium complexity

#### 2.1 Cell selection and navigation
Transform from text-rendered viewport to an interactive table model:

- Arrow keys / hjkl to move between cells
- Visual highlight on the selected cell
- Status bar shows: column name, data type, full value (for truncated cells)

This is a significant architectural change. The current text-based approach
(render entire table to a string, put it in a viewport) would need to be
replaced with a proper table widget that knows about individual cells.

**Options:**
- **a) Use `charmbracelet/bubbles/table`**: Pre-built table component.
  Limited customization but works out of the box.
- **b) Custom table renderer**: Build our own cell-aware table. More work
  but full control over styling, selection, and interaction.
- **c) Use a community table component**: e.g. `evertras/bubble-table`
  which has sorting, filtering, and selection built in.

**Recommendation**: Start with option (a) `bubbles/table` for basic
cell selection, evaluate if it meets needs, then consider (b) if
customization is needed.

#### 2.2 Filtering (WHERE clause)
- `/` to open a filter input
- Enter a WHERE clause: `price > 10 AND category = 'gadgets'`
- Re-query with the WHERE clause appended
- Show active filter indicator in title: `Browse: products (filtered)`
- Esc to clear filter

#### 2.3 Column type indicators
- Show column types in the header row: `id (INT)`, `name (VARCHAR)`
- Query `INFORMATION_SCHEMA.COLUMNS` or parse schema for type info
- Style type indicators dim/small

### Tier 3: Advanced features (high complexity)

#### 3.1 Inline editing
- `c` to edit the selected cell
- Opens a text input with the current value
- Requires tracking pending changes and committing them
- **Warning**: This overlaps heavily with lazysql's CRUD features.
  Per the sql-query-feature.md analysis, we should NOT build this.
  If a user needs to edit data, recommend lazysql.

#### 3.2 Column resizing
- `+`/`-` to increase/decrease column width (conflicts with zoom keys)
- Or: auto-fit column width based on content
- Manual column width persistence (save to config)

#### 3.3 Export (CSV/JSON)
- `E` to export current page or all rows
- Choose format: CSV, JSON, SQL INSERT statements
- Write to file or clipboard

#### 3.4 Foreign key navigation
- When a cell contains a foreign key value, `Enter` to jump to the
  referenced row in the referenced table
- Requires querying `INFORMATION_SCHEMA.KEY_COLUMN_USAGE`
- Opens a new Browse tab for the referenced table, scrolled to the row

#### 3.5 Dolt-specific: time-travel browsing
- `t` to select a commit from history
- Browse the table `AS OF` that commit
- Show title: `Browse: users AS OF abc1234`
- This is a unique lazydolt feature that lazysql cannot provide

#### 3.6 Dolt-specific: cross-branch browsing
- Select a branch, then browse a table on that branch
- Uses dolt's `branch/table` syntax
- Show title: `Browse: feature-branch/users`

## Implementation Priority

Based on the lazysql analysis (don't duplicate general SQL features,
focus on dolt-specific value):

| Priority | Feature | Rationale |
|---|---|---|
| P1 | 1.2 Row/column count | Trivial, improves UX |
| P1 | 1.1 Column sorting | High value, moderate effort |
| P2 | 2.2 Filtering | Standard feature, moderate effort |
| P2 | 3.5 Time-travel browsing | **Unique to lazydolt** |
| P2 | 3.6 Cross-branch browsing | **Unique to lazydolt** |
| P3 | 2.1 Cell selection | Significant refactor |
| P3 | 1.3 Better column truncation | Nice to have |
| P3 | 2.3 Column type indicators | Nice to have |
| P4 | 1.4 Copy cell value | Requires cell selection |
| P4 | 3.3 Export | Nice to have |
| Skip | 3.1 Inline editing | Use lazysql instead |
| Skip | 3.2 Column resizing | Low value vs complexity |
| Skip | 3.4 FK navigation | Very complex, low frequency |

## Dependencies

- Column sorting, filtering: require changes to `internal/dolt/query.go`
- Cell selection: requires either `bubbles/table` or custom table model
- Time-travel/cross-branch: require `dolt sql -q "SELECT ... AS OF '...'"` support
- Copy to clipboard: requires `atotto/clipboard` or similar dependency
