# Contributing to lazydolt

## Prerequisites

- Go 1.24+
- [Dolt](https://docs.dolthub.com/introduction/installation) on your `PATH`
- A terminal with ANSI color support

## Building

```bash
go build -o ~/bin/lazydolt ./cmd/lazydolt
```

## Architecture

Three layers with strict dependency direction:

- `internal/domain/` — pure types (no imports from other internal packages)
- `internal/dolt/` — CLI wrapper (imports domain only)
- `internal/ui/` — Bubble Tea TUI (imports domain + dolt)
- `internal/testutil/` — test helpers (creates temporary dolt repos)

## Testing Strategy

lazydolt uses a two-tier testing approach: automated tests for correctness, and interactive testing for visual/UX verification.

### Automated Tests

```bash
# Run all tests
go test ./...

# Run a specific package
go test ./internal/dolt/...
go test -v ./internal/ui/components/...

# Static analysis
go vet ./...
```

#### Test structure

| Package | Type | What it tests |
|---------|------|---------------|
| `internal/dolt/` | Integration | CLI wrapper methods against real temporary dolt repos |
| `internal/testutil/` | Integration | The test infrastructure itself |
| `internal/ui/components/` | Unit | Component rendering, navigation, key handling |

#### Golden snapshot tests

View() output for UI components is tested using [charmbracelet/x/exp/golden](https://pkg.go.dev/github.com/charmbracelet/x/exp/golden). These compare rendered output (including ANSI escape codes) against reference files in `testdata/*.golden`.

- **Run normally**: `go test ./internal/ui/components/` — compares against golden files
- **Regenerate after intentional changes**: `go test ./internal/ui/components/ -args -update`
- **Golden files are checked into git** — diffs in PRs show exactly what rendering changed

When adding a new View() test, use `golden.RequireEqual(t, output)` instead of `strings.Contains` checks. This catches unintended rendering regressions that substring checks miss.

#### Tips for writing component tests

- Call `lipgloss.SetColorProfile(termenv.ANSI)` in an `init()` function to force deterministic color output. Without this, styles are no-ops in CI and tests checking rendered output will fail.
- Component tests construct models with known data — no dolt repo needed.
- The `execCmd(t, cmd)` helper extracts the `tea.Msg` from a `tea.Cmd` for assertion.
- Avoid `time.Now()` in golden-file tests — relative timestamps change over time. Use `strings.Contains` for time-dependent output instead.

### Interactive Testing

For UI changes, also verify interactively. A test dolt repo with realistic data lives at `tmp/test-dolt/`:

```bash
# Create/reset the test repo (10 commits, staged + unstaged changes)
scripts/setup-test-db.sh

# Build and run
go build -o ~/bin/lazydolt ./cmd/lazydolt
~/bin/lazydolt tmp/test-dolt
```

### What to check before submitting

1. `go build ./...` — compiles cleanly
2. `go test ./...` — all tests pass
3. `go vet ./...` — no static analysis warnings
4. If you changed View() output, regenerate golden files with `-update` and review the diffs
5. For UI changes, verify visually that rendering looks correct

## Code Style

- Follow standard Go conventions
- Comments explain **why**, not what
- Use lipgloss styles for all terminal formatting — never raw ANSI codes
- Keep the three-layer architecture: domain has no internal imports, dolt imports only domain, ui imports both
- For selection highlighting, apply `selectedStyle.Render()` only to the text that should be highlighted (e.g. the table name), not the entire line. Wrapping a line that contains inner styled segments (e.g. colored status markers) breaks because embedded ANSI resets cancel the outer reverse attribute
