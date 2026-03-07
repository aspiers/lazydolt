# lazydolt

A terminal UI for [Dolt](https://www.dolthub.com/blog/2021-09-17-dolt/) databases, inspired by [lazygit](https://github.com/jesseduffield/lazygit).

Built with Go using [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Lip Gloss](https://github.com/charmbracelet/lipgloss).

## Features

- **Table management** — view, stage, and unstage tables
- **Branch operations** — checkout, create, and delete branches
- **Commit history** — browse commits and view diffs
- **Diff viewer** — color-coded diffs with scrolling
- **Schema viewer** — syntax-highlighted CREATE TABLE statements
- **Table data browser** — paginated, formatted table viewer
- **Commit dialog** — stage changes and commit from the TUI
- **Resizable layout** — adjust column widths with `<` / `>` keys

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
| `Tab` | Next panel |
| `1`-`3` | Jump to panel |
| `c` | Commit |
| `+` / `_` | Zoom panel |
| `Esc` | Back / reset zoom |
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
| `n` | New branch |
| `D` | Delete branch |

### Commits Panel

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate |
| `Enter` | View commit details |

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

MIT
