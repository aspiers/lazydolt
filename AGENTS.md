# Agent Instructions

This project uses **bd** (beads) for issue tracking. Run `bd onboard` to get started.

## Project Overview

**lazydolt** is a TUI for [Dolt](https://www.dolthub.com/blog/2021-09-17-dolt/) databases, inspired by lazygit. Written in Go using [Bubble Tea](https://github.com/charmbracelet/bubbletea), it wraps the `dolt` CLI for all database interactions.

- **Go module**: `github.com/aspiers/lazydolt`
- **Entry point**: `cmd/lazydolt/main.go`
- **Architecture**: 3 layers with strict dependency direction:
  - `internal/domain/` — pure types (no imports from other internal packages)
  - `internal/dolt/` — CLI wrapper (imports domain only)
  - `internal/ui/` — Bubble Tea TUI (imports domain + dolt)
- **Test utilities**: `internal/testutil/` — creates temporary dolt repos for tests

## Building & Testing

```bash
# Build
go build -o ~/bin/lazydolt ./cmd/lazydolt

# Run all tests
go test ./...

# Run tests for a specific package
go test ./internal/dolt/...
go test ./internal/ui/...

# Run with verbose output
go test -v ./internal/ui/components/...

# Vet (static analysis)
go vet ./...
```

## Dolt CLI Notes

- **JSON output**: `dolt sql -r json -q "SELECT * FROM dolt_status"` — used for all data loading
- **Mutations**: Use dolt CLI directly (`dolt add`, `dolt commit`, etc.) not SQL procedures
- **Time format**: `"2006-01-02 15:04:05.000000"` — multi-format parser in `internal/dolt/time.go`
- **Diff markers**: `+` added, `-` deleted, `<` old value, `>` new value (not unified diff)
- **ANSI stripping**: dolt outputs color codes by default — stripped via regex in `runner.go`
- **Config syntax**: `dolt config --global --set user.email "..."` (requires `--set` flag)
- **`schema show` on nonexistent table**: Returns exit code 0 with "not found" message

## Testing Strategy

**Both automated AND interactive testing are required** when working on UI changes. Run unit tests first, then verify interactively via tmux.

### Automated Tests (Unit / Integration)

```bash
# Run all tests
go test ./...

# Run tests for a specific package
go test ./internal/dolt/...
go test ./internal/ui/...
go test -v ./internal/ui/components/...
```

### Interactive QA Testing via tmux

Use a **dedicated tmux session called `lazydolt-test`** for interactive testing. This session runs lazydolt in a controlled environment that can be inspected programmatically.

**CRITICAL RULES for tmux session management:**

- **NEVER kill existing tmux sessions or windows.** Take great care to preserve any running sessions.
- **To restart lazydolt**, send `q` or `C-c` to quit gracefully, then start a new instance in the same session.
- **Do NOT use `tmux kill-session`, `tmux kill-window`, or `tmux kill-pane`** unless you are absolutely certain the target is a stale `lazydolt-test` session with no running process.
- **Session name is always `lazydolt-test`** — do not use other names, and do not reuse session names belonging to other tools.

### Setting up the test repo

A test dolt repo lives at `tmp/test-dolt/`. To recreate it:

```bash
rm -rf tmp/test-dolt
mkdir -p tmp/test-dolt
cd tmp/test-dolt
dolt init
dolt sql -q "CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(100), email VARCHAR(200))"
dolt sql -q "INSERT INTO users VALUES (1, 'Alice', 'alice@example.com'), (2, 'Bob', 'bob@example.com')"
dolt add .
dolt commit -m "Initial commit: users table"
dolt sql -q "INSERT INTO users VALUES (3, 'Charlie', 'charlie@example.com')"
dolt add .
dolt commit -m "Add Charlie to users"
dolt sql -q "CREATE TABLE orders (id INT PRIMARY KEY, user_id INT, amount DECIMAL(10,2))"
dolt checkout -b feature-branch
```

### Running lazydolt in tmux

```bash
# Build first
go build -o ~/bin/lazydolt ./cmd/lazydolt

# Check if session already exists
tmux has-session -t lazydolt-test 2>/dev/null

# If lazydolt is running in the session, quit it gracefully first:
tmux send-keys -t lazydolt-test 'q'
sleep 0.5

# If no session exists, create one (start a shell, NOT lazydolt directly,
# so quitting lazydolt doesn't kill the session):
tmux new-session -d -s lazydolt-test -x 120 -y 40
tmux send-keys -t lazydolt-test '~/bin/lazydolt tmp/test-dolt' Enter

# If session exists but lazydolt has exited, start a new instance:
tmux send-keys -t lazydolt-test '~/bin/lazydolt tmp/test-dolt' Enter

# Force a specific terminal size (tmux resizes to match client)
tmux resize-window -t lazydolt-test -x 120 -y 40
```

### Sending keys and capturing output

```bash
# Send a single key
tmux send-keys -t lazydolt-test 'j'         # Navigate down
tmux send-keys -t lazydolt-test 'Tab'       # Cycle focus
tmux send-keys -t lazydolt-test 'Space'     # Stage/unstage
tmux send-keys -t lazydolt-test 'Enter'     # Select/confirm
tmux send-keys -t lazydolt-test '?'         # Help overlay
tmux send-keys -t lazydolt-test 'Escape'    # Cancel/back
tmux send-keys -t lazydolt-test 'C-c'       # Ctrl+C

# Capture current screen content
tmux capture-pane -t lazydolt-test -p

# Wait briefly between actions for TUI to update
sleep 0.3
```

### Important tmux notes

- **Window size**: tmux resizes to match the connecting client terminal, so `-x 120 -y 40` at creation doesn't persist. Use `tmux resize-window` to force a specific size.
- **Graceful shutdown only**: Always send `q` or `C-c` to quit lazydolt. Never kill sessions/windows/panes violently.
- **PTY sessions**: Alternatively, use `pty_spawn` to run lazydolt in a managed PTY session for automated testing within the agent environment. But this is far less preferable because it cannot easily be observed by the user during testing, so ask permission before using this fallback.

### Typical QA workflow

1. Build the binary
2. Set up or verify the test repo state
3. Launch lazydolt in the `lazydolt-test` tmux session (or restart if already running)
4. Send keys and capture pane output to verify behavior
5. Check for correct rendering, focus indicators, data updates
6. When done: send `q` to quit lazydolt (leave the tmux session intact for next time)

## Bubble Tea Testing Notes

- **Lipgloss in tests**: Call `lipgloss.SetColorProfile(termenv.ANSI)` in test `init()` to force ANSI color output. Without this, styles are no-ops and tests checking rendered output may fail.
- **Value receiver bug**: `View()` is a value receiver on `App`, so field modifications (like setting `Focused` on sub-models) only affect the local copy. Use `syncFocus()` helper to propagate focus state.
- **Layout height**: Each `RoundedBorder` panel adds 2 rows (top+bottom). With 4 panels + hints bar, account for `4*2 + 1 = 9` extra rows.
- **Component messages**: Messages in `components` package are exported types (e.g. `StageTableMsg`), aliased as lowercase in `app.go` for the switch statement.

## Non-Interactive Shell Commands

**ALWAYS use non-interactive flags** with file operations to avoid hanging on confirmation prompts.

Shell commands like `cp`, `mv`, and `rm` may be aliased to include `-i` (interactive) mode on some systems, causing the agent to hang indefinitely waiting for y/n input.

**Use these forms instead:**
```bash
# Force overwrite without prompting
cp -f source dest           # NOT: cp source dest
mv -f source dest           # NOT: mv source dest
rm -f file                  # NOT: rm file

# For recursive operations
rm -rf directory            # NOT: rm -r directory
cp -rf source dest          # NOT: cp -r source dest
```

<!-- BEGIN BEADS INTEGRATION -->
## Issue Tracking with bd (beads)

**IMPORTANT**: This project uses **bd (beads)** for ALL issue tracking. Do NOT use markdown TODOs, task lists, or other tracking methods.

### Why bd?

- Dependency-aware: Track blockers and relationships between issues
- Version-controlled: Built on Dolt with cell-level merge
- Agent-optimized: JSON output, ready work detection, discovered-from links
- Prevents duplicate tracking systems and confusion

### Quick Start

**Check for ready work:**

```bash
bd ready --json
```

**Create new issues:**

```bash
bd create "Issue title" --description="Detailed context" -t bug|feature|task -p 0-4 --json
bd create "Issue title" --description="What this issue is about" -p 1 --deps discovered-from:bd-123 --json
```

**Claim and update:**

```bash
bd update <id> --claim --json
bd update bd-42 --priority 1 --json
```

**Complete work:**

```bash
bd close bd-42 --reason "Completed" --json
```

### Issue Types

- `bug` - Something broken
- `feature` - New functionality
- `task` - Work item (tests, docs, refactoring)
- `epic` - Large feature with subtasks
- `chore` - Maintenance (dependencies, tooling)

### Priorities

- `0` - Critical (security, data loss, broken builds)
- `1` - High (major features, important bugs)
- `2` - Medium (default, nice-to-have)
- `3` - Low (polish, optimization)
- `4` - Backlog (future ideas)

### Workflow for AI Agents

1. **Check ready work**: `bd ready` shows unblocked issues
2. **Claim your task atomically**: `bd update <id> --claim`
3. **Work on it**: Implement, test, document
4. **Discover new work?** Create linked issue:
   - `bd create "Found bug" --description="Details about what was found" -p 1 --deps discovered-from:<parent-id>`
5. **Complete**: `bd close <id> --reason "Done"`

### Backup & Export

bd uses a Dolt database backend (no SQLite/JSONL sync). To back up issues into git:

```bash
bd export -o .beads/issues.jsonl
git add .beads/issues.jsonl && git commit -m "Export beads"
```

This project has no Dolt remote configured, so `bd dolt push`/`pull` are not available.
**Note:** `bd sync` has been removed — do NOT use it.

### Important Rules

- ✅ Use bd for ALL task tracking
- ✅ Always use `--json` flag for programmatic use
- ✅ Link discovered work with `discovered-from` dependencies
- ✅ Check `bd ready` before asking "what should I work on?"
- ❌ Do NOT create markdown TODO lists
- ❌ Do NOT use external issue trackers
- ❌ Do NOT duplicate tracking systems

For more details, see README.md and docs/QUICKSTART.md.

## Landing the Plane (Session Completion)

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** - Create issues for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --rebase
   bd export -o .beads/issues.jsonl
   git add .beads/issues.jsonl && git commit -m "Export beads"
   git push
   git status  # MUST show "up to date with origin"
   ```
5. **Clean up** - Clear stashes, prune remote branches
6. **Verify** - All changes committed AND pushed
7. **Hand off** - Provide context for next session

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds

<!-- END BEADS INTEGRATION -->
