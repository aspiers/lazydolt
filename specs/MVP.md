# lazydolt — MVP Plan

A TUI for [Dolt](https://www.dolthub.com/), inspired by
[lazygit](https://github.com/jesseduffield/lazygit).

## Tech Stack

- **Language**: Go
- **TUI**: [Bubble Tea](https://github.com/charmbracelet/bubbletea) +
  [Lip Gloss](https://github.com/charmbracelet/lipgloss) (styling) +
  [Bubbles](https://github.com/charmbracelet/bubbles) (reusable components:
  lists, tables, viewports, text inputs)
- **Dolt interaction**: Shell out to `dolt` CLI. Use `dolt sql -r json`
  for structured data from system tables (`dolt_status`, `dolt_log`,
  `dolt_branches`, `dolt_diff_<table>`) and plain CLI commands
  (`dolt add`, `dolt commit`, etc.) for mutations.

## Architecture

Three layers with strict dependency direction (top depends on bottom):

```
+-------------------------------------------+
|  TUI Layer (internal/ui/)                 |
|  Bubble Tea models, views, keybindings    |
+-------------------------------------------+
|  Domain Layer (internal/domain/)          |
|  Go structs: Table, Branch, Commit,       |
|  DiffEntry, StatusEntry, Schema           |
+-------------------------------------------+
|  Command Layer (internal/dolt/)           |
|  Wraps dolt CLI. Runs commands, parses    |
|  JSON output into domain structs.         |
+-------------------------------------------+
```

- **`internal/dolt/`** — All dolt interaction. No TUI imports. Testable
  independently.
  - `dolt sql -r json -q "SELECT * FROM dolt_status"` etc. for reads
  - `dolt add <table>`, `dolt commit -m "..."`, etc. for mutations
- **`internal/domain/`** — Pure data types (no dolt, no TUI deps). Shared
  between layers.
- **`internal/ui/`** — Bubble Tea models. Each panel is a sub-model. A
  root model composes them and handles focus/navigation.

## Layout

```
+----------------+--------------------------------+
|  Status        |                                |
|  (branch,      |  Main Panel                    |
|   repo info)   |  (contextual content:          |
+----------------|   diff, table data, schema,    |
|  Tables        |   commit details, log)         |
|  (list with    |                                |
|   status       |                                |
|   markers)     |                                |
+----------------|                                |
|  Branches      |                                |
|  (list)        |                                |
+----------------|                                |
|  Commits       |                                |
|  (log list)    |                                |
+----------------+                                |
|  [Key hints]   |                                |
+----------------+--------------------------------+
```

### Left side panels (Tab or 1-4 to switch)

1. **Status** — Current branch, repo name, dirty indicator
2. **Tables** — From `dolt ls` / `dolt_status`. Status markers:
   new/modified/deleted/staged
3. **Branches** — From `dolt_branches`
4. **Commits** — From `dolt_log`

### Right side panel (context-dependent)

- Tables focused + table selected: table diff (`dolt_diff_<table>`)
- Tables focused + Enter: table data browser (`SELECT * FROM <t>` with
  pagination)
- Tables focused + `s`: schema view (`dolt schema show <t>`)
- Branches focused: branch details
- Commits focused + commit selected: commit diff details

### Bottom bar

Context-sensitive keybinding hints.

## Keybindings

| Panel    | Key         | Action                       | Dolt command                                 |
| -------- | ----------- | ---------------------------- | -------------------------------------------- |
| Global   | Tab / 1-4   | Switch panel focus           | —                                            |
| Global   | q / Ctrl+C  | Quit                         | —                                            |
| Global   | ?           | Help overlay                 | —                                            |
| Tables   | Enter       | Browse table data            | `dolt sql -r json -q "SELECT ... LIMIT ..."` |
| Tables   | s           | View schema                  | `dolt schema show <t>`                       |
| Tables   | Space       | Stage/unstage table          | `dolt add <t>` / `dolt reset <t>`            |
| Tables   | d           | View diff                    | `dolt diff <t>`                              |
| Tables   | a           | Stage all                    | `dolt add .`                                 |
| Global   | c           | Commit (opens message input) | `dolt commit -m "..."`                       |
| Branches | n           | New branch                   | `dolt branch <name>`                         |
| Branches | Enter       | Checkout branch              | `dolt checkout <name>`                       |
| Branches | D           | Delete branch                | `dolt branch -d <name>`                      |
| Commits  | Enter       | View commit details          | `dolt show <hash>`                           |
| Browser  | Arrows      | Scroll rows/columns          | —                                            |
| Browser  | Esc         | Back to tables               | —                                            |

## Tasks

### Dependency graph

```
#1 Scaffolding
|-- #2 Domain types --+-- #4 Status()  --+-- #13 Status bar
|                     |-- #5 Tables()  --|-- #14 Tables panel -- #20 Stage/unstage
|                     |-- #6 Branches()--|-- #15 Branches panel - #22 Branch ops
|                     |-- #7 Log()     --|-- #16 Commits panel
|                     |-- #8 Diff()    --|-- #17 Diff viewer
|                     |-- #9 Schema()  --|-- #19 Schema viewer
|                     +-- #10 Query()  --|-- #18 Table browser
|-- #3 Runner --------+                  |
|   +-- #11 Mutations ------------------+-- #20, #21, #22
+-- #12 Root TUI ------------------------+
    |-- #23 Key hints
    +-- #24 Help overlay
```

### Task list

| #  | Title                                                          | Type    | Pri | Depends on  |
| -- | -------------------------------------------------------------- | ------- | --- | ----------- |
| 1  | Project scaffolding (go mod, dirs, editorconfig, main.go)      | task    | P1  | —           |
| 2  | Domain types (Table, Branch, Commit, StatusEntry, DiffRow)     | task    | P1  | 1           |
| 3  | Command layer: Runner (exec dolt, capture output)              | task    | P1  | 1           |
| 4  | Command layer: Status() — parse dolt_status                    | task    | P1  | 2, 3        |
| 5  | Command layer: Tables() — list tables                          | task    | P1  | 2, 3        |
| 6  | Command layer: Branches() — parse dolt_branches                | task    | P1  | 2, 3        |
| 7  | Command layer: Log() — parse dolt_log                          | task    | P1  | 2, 3        |
| 8  | Command layer: Diff(table) — get table diff                    | task    | P2  | 2, 3        |
| 9  | Command layer: Schema(table) — get table schema                | task    | P2  | 2, 3        |
| 10 | Command layer: Query(sql) — run arbitrary SQL                  | task    | P2  | 3           |
| 11 | Command layer: Mutations (Add, Reset, Commit, Checkout, etc.)  | task    | P1  | 3           |
| 12 | Root TUI model (Bubble Tea app, layout, focus management)      | feature | P1  | 1           |
| 13 | Side panel: Status bar (branch, repo, dirty indicator)         | task    | P2  | 4, 12       |
| 14 | Side panel: Tables list with status markers                    | task    | P1  | 4, 5, 12    |
| 15 | Side panel: Branches list                                      | task    | P2  | 6, 12       |
| 16 | Side panel: Commits log list                                   | task    | P2  | 7, 12       |
| 17 | Main panel: Diff viewer                                        | task    | P1  | 8, 12       |
| 18 | Main panel: Table data browser (scrollable, paginated)         | task    | P2  | 10, 12      |
| 19 | Main panel: Schema viewer                                      | task    | P2  | 9, 12       |
| 20 | Stage/unstage tables (Space key in Tables panel)               | task    | P1  | 11, 14      |
| 21 | Commit dialog (text input overlay)                             | task    | P1  | 11, 12      |
| 22 | Branch operations (create, checkout, delete)                   | task    | P2  | 11, 15      |
| 23 | Context-sensitive key hints bar                                | task    | P2  | 12          |
| 24 | Help overlay (? key)                                           | task    | P3  | 12          |
| 25 | Unit tests for command layer                                   | task    | P2  | 4-11        |

### Critical path to working demo

1. #1 Scaffolding
2. #2 + #3 Domain types + Runner (parallel)
3. #4 + #5 + #11 Status + Tables + Mutations (parallel)
4. #12 Root TUI model
5. #14 Tables panel
6. #17 Diff viewer
7. #20 + #21 Staging + Commit dialog

At this point the tool can show tables, view diffs, stage, and commit —
the core workflow.

## Out of scope for MVP

- Merge / conflict resolution
- Stash
- Remotes (push/pull/fetch)
- Interactive SQL query panel
- Row-level staging (dolt doesn't support this natively)
- Custom themes
- Config file
