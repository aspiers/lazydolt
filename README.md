# lazydolt

A terminal UI for [Dolt](https://www.dolthub.com/blog/2021-09-17-dolt/) databases, inspired by [lazygit](https://github.com/jesseduffield/lazygit).

Built with Go using [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Lip Gloss](https://github.com/charmbracelet/lipgloss).

![lazydolt demo](demo.gif)

## Features

- **Table management** — view, stage, unstage, and discard table changes
- **Branch operations** — checkout, create, rename, merge, and delete branches
- **Merge conflict resolution** — resolve conflicts with ours/theirs or abort merge
- **Commit history** — browse commits with metadata and view diffs
- **Diff viewer** — color-coded diffs with scrolling
- **Schema viewer** — syntax-highlighted CREATE TABLE statements
- **Table data browser** — paginated, formatted table viewer
- **SQL query panel** — run arbitrary SQL queries with `:` key
- **Stash support** — stash/pop/drop changes with `S` key
- **Command log** — persistent log of all dolt CLI commands and errors
- **Search/filter** — filter tables, branches, and commits with `/`
- **Commit dialog** — stage changes and commit from the TUI
- **Zoomable layout** — zoom panels with `+` / `_` keys, adjustable column width

## Requirements

- Go 1.24+
- [Dolt](https://docs.dolthub.com/introduction/installation) installed and on your `PATH`

## Installation

```bash
go install github.com/aspiers/lazydolt/cmd/lazydolt@latest
```

Or build from source:

```bash
git clone https://github.com/aspiers/lazydolt.git
cd lazydolt
go build -o ~/bin/lazydolt ./cmd/lazydolt
```

## Usage

```bash
lazydolt /path/to/dolt/repo
```

## Keyboard Shortcuts

### Global

| Key | Action |
|-----|--------|
| `q` / `Ctrl+C` | Quit |
| `Tab` / `Shift+Tab` | Next / previous panel |
| `1`-`3` | Jump to panel |
| `c` | Commit |
| `:` | SQL query |
| `S` | Stash / show stash list |
| `R` | Refresh all data |
| `P` / `p` / `f` | Push / pull / fetch |
| `/` | Filter panel items |
| `+` / `_` | Zoom panel |
| `<` / `>` / `=` | Adjust column width |
| `Esc` | Back / reset zoom / clear filter |
| `?` | Toggle help |

### Tables Panel

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate |
| `Space` | Stage/unstage table |
| `a` | Stage all |
| `d` | View diff |
| `s` | View schema |
| `Enter` | Browse table data |

### Branches Panel

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate |
| `Enter` | Checkout branch |
| `m` | Merge into current |
| `n` | New branch |
| `r` | Rename branch |
| `D` | Delete branch |

### Commits Panel

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate |
| `Enter` | View commit details |
| `A` | Amend last commit |
| `g` | Reset to commit |

## Architecture

Three layers with strict dependency direction:

- `internal/domain/` — pure types (no imports from other internal packages)
- `internal/dolt/` — CLI wrapper (imports domain only)
- `internal/ui/` — Bubble Tea TUI (imports domain + dolt)

## Development

```bash
# Run tests
go test ./...

# Run static analysis
go vet ./...

# Build
go build -o ~/bin/lazydolt ./cmd/lazydolt
```

## License

[GPL-3.0](LICENSE)
