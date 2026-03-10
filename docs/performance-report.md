# Lazydolt Performance Profiling Report

**Date:** 2026-03-10
**Profiling environment:** Linux amd64, Intel Haswell, Go benchmarks + dolt CLI timing
**Dolt version:** 1.82.3
**Test database:** 3 tables, 20 commits, 2 branches

## Executive Summary

The dominant performance bottleneck in lazydolt is **dolt CLI subprocess overhead** (~180–220ms per call, 8 calls per full refresh). Rendering costs are secondary but contain several opportunities where work is needlessly repeated on every frame. The most impactful optimizations are:

1. **Cache commit graph** — eliminates ~2ms/frame of CPU + 25KB allocs on every render
2. **Cache `displayItems()` / `filteredIndices()`** — eliminates repeated O(n) recomputation per frame
3. **Selective refresh** — avoid reloading branches/tags/remotes when only staging changes
4. **Eliminate redundant `CurrentBranch()` call** — saves ~200ms per refresh

## 1. Dolt CLI Call Analysis

### Per-call latency

Each `dolt` subprocess invocation costs **150–220ms** on this system:

| Operation | Avg Time | Notes |
|---|---|---|
| `active_branch()` | ~200ms | Simple SQL query |
| `dolt_status` | ~180ms | System table scan |
| `dolt ls` | ~190ms | Lists tables |
| `dolt_branches` | ~200ms | System table scan |
| `dolt_log` (50 commits) | ~200ms | Complex JOIN query |
| `dolt_tags` | ~140ms | System table scan |
| `dolt remote -v` | ~110ms | Config read |
| `dolt diff <table>` | ~190ms | Diff computation |

### Full refresh cost (`loadData()`)

`loadData()` fires **8 CLI calls in 6 goroutines** (running concurrently):

- `CurrentBranch()` → 1 call (~200ms)
- `Tables()` → 2 calls: `dolt ls` + `dolt_status` (~200ms sequential)
- `Branches()` → 2 calls: `dolt_branches` + `active_branch()` (~200ms sequential)
- `Log()` → 1 call (~200ms)
- `Tags()` → 1 call (~140ms)
- `Remotes()` → 1 call (~110ms)

**Wall clock for `loadData()`:** ~400ms (limited by the `Tables()` and `Branches()` goroutines which run 2 sequential calls each).

**Redundancy:** `Branches()` calls `CurrentBranch()` internally, and `loadData()` also calls `CurrentBranch()` separately — that's **2 subprocess calls** for the same data.

### Refresh frequency

`loadData()` is triggered on ~25 different code paths: every mutation (stage, unstage, commit, branch checkout, delete, etc.) plus the initial load and manual `R` refresh. There is no distinction between mutations that change branches/tags/remotes vs. mutations that only affect the working set.

### Staging operation cost

Staging a single table triggers:
1. `dolt add <table>` → 1 CLI call
2. `loadData()` → 8 CLI calls (full refresh)
3. `autoPreview()` → 1 CLI call (diff)
4. **Total: 10 CLI calls, ~600ms**

## 2. Rendering Pipeline Analysis

### Benchmark Results

| Operation | Time/op | Allocs/op | Bytes/op |
|---|---|---|---|
| `BuildGraph` (50 commits) | **750µs** | 1,385 | 25KB |
| `BuildGraph` (200 commits) | **3.4ms** | 6,065 | 110KB |
| `MaxGraphWidth` (50 lines) | 26µs | 50 | 800B |
| `CommitsModel.View()` (50) | **2.0ms** | 2,596 | 108KB |
| `CommitsModel.View()` (200) | **5.1ms** | 7,428 | 199KB |
| `DiffView.SetContent` (100 lines) | 620µs | 419 | 33KB |
| `DiffView.SetContent` (1000 lines) | **5.3ms** | 4,022 | 329KB |
| `DiffView.SetContent` (5000 lines) | **28ms** | 20,027 | 1.6MB |
| `SanitizeForDisplay` (1000 lines) | 1.4ms | 1 | 66KB |
| `HScrollContent` (50 lines) | 54µs | 202 | 10KB |
| `clipLines` simulation (40 lines) | 170µs | 243 | 14KB |
| `TablesModel.View()` (20 tables) | 196µs | 294 | 17KB |
| `lipgloss.Width()` (styled line) | 527ns | 1 | 16B |
| `lipgloss.Style.Render()` | **4.9µs** | 8 | 114B |

### CPU Profile Top Consumers

| Function | % of total CPU | Notes |
|---|---|---|
| `lipgloss.Style.Render` | **38%** | Called hundreds of times per render |
| `runtime.duffcopy` | **22%** | Large struct copies (lipgloss value semantics) |
| `BuildGraph` | **19%** | Rebuilt on every `View()` call |
| `ansi.StringWidth` | **15%** | ANSI-aware width measurement |
| `SanitizeForDisplay` | **11%** | Control-char filtering |
| `colorizeDiff` | **10%** | Per-line style application |
| GC overhead | **8.5%** | From allocation pressure |
| `MaxGraphWidth` | **5.9%** | Width measurement of styled graph lines |

### Memory Profile Top Allocators

| Source | Allocation | Notes |
|---|---|---|
| `strings.Builder.grow` | **2.9GB** (32.7%) | String building across all operations |
| `strings.makeStringFinder` | **2.8GB** (31.6%) | ANSI stripping regex replacement |
| `strings.genSplit` | **744MB** (8.4%) | Splitting strings by `\n` repeatedly |
| `termenv.Style.Styled` | **361MB** (4.1%) | Core lipgloss styling |
| `BuildGraph` | **503MB** (5.7%) | Graph line construction |
| `SanitizeForDisplay` | **460MB** (5.2%) | String sanitization |
| `colorizeDiff` | **651MB** (7.3%) | Diff colorization |
| `HScrollContent` | **893MB** (10.0%) | ANSI-aware truncation |

## 3. Key Findings

### Finding 1: `BuildGraph()` recomputed on every render frame

**Location:** `internal/ui/components/commits.go:100`
**Impact:** HIGH — 750µs + 25KB per frame at 50 commits, scales linearly

`CommitsModel.View()` calls `BuildGraph(m.Commits)` and `MaxGraphWidth()` on every render. The graph only changes when the commits list changes (data reload), not on cursor moves, scrolling, or other UI interactions. With Bubble Tea's reactive model, `View()` is called on every message, so this runs potentially dozens of times per second during navigation.

### Finding 2: `displayItems()` and `filteredIndices()` recomputed per call

**Location:** `internal/ui/components/tables.go:61,115,265,291,333,346,373,381`
**Impact:** MEDIUM — Called 3-4 times per keypress, O(n) each time

`TablesModel.displayItems()` is called from 10 different methods. During a single key press, it may be called in `Update()`, then `SelectedTable()` (for auto-preview), then `View()`. Each call iterates all tables, groups them by status, and applies the filter. Same pattern for `CommitsModel.filteredIndices()` and `BranchesModel.filteredIndices()`.

### Finding 3: Redundant `CurrentBranch()` call

**Location:** `internal/dolt/branches.go:32` and `internal/ui/app.go:2048`
**Impact:** MEDIUM — Saves ~200ms per refresh (one fewer subprocess)

`Branches()` internally calls `CurrentBranch()` to mark which branch is current. `loadData()` also calls `CurrentBranch()` in a separate goroutine. The result from the goroutine is used to set `a.currentBranch`, but the same information is already embedded in the `Branches` result (each `Branch` has `IsCurrent` set).

### Finding 4: Full refresh on every mutation

**Location:** `internal/ui/app.go:999-1182`
**Impact:** MEDIUM-HIGH — ~400ms wall clock per mutation

Every mutation (stage, commit, branch create/delete, etc.) triggers `loadData()` which reloads *everything*: tables, branches, commits, tags, remotes. Most mutations only affect a subset. For example, staging a table doesn't change branches, commits, tags, or remotes.

### Finding 5: `lipgloss.Style.Render()` is the #1 CPU consumer

**Impact:** MEDIUM — 38% of CPU in benchmarks, ~5µs + 8 allocs per call

Every styled string goes through `lipgloss.Style.Render()`, which internally uses `termenv.Style.Styled()`. With hundreds of styled spans per frame (commit hashes, graph characters, table names, diff lines), this adds up. The underlying `duffcopy` cost (22%) comes from lipgloss's value-receiver semantics copying large `Style` structs.

### Finding 6: Triple string processing pass for diffs

**Location:** `internal/ui/components/diffview.go:61`
**Impact:** LOW-MEDIUM — 5.3ms for 1000-line diff, 28ms for 5000-line diff

`SetContent()` runs: `SanitizeForDisplay(content)` → `colorizeDiff(result)` → viewport `SetContent()`. Each pass allocates and splits strings. For large diffs this is noticeable but only runs once per diff load (not per frame).

### Finding 7: ANSI stripping applied to JSON output

**Location:** `internal/dolt/runner.go:74`
**Impact:** LOW — ~4µs for small JSON, but wasteful

`stripANSI()` runs on *every* `Exec()` output, including `dolt sql -r json` results that shouldn't contain ANSI codes. Minor CPU cost but unnecessary allocation.

### Finding 8: `clipLines()` + `truncateToVisualHeight()` double iteration

**Location:** `internal/ui/app.go:1765-1769`
**Impact:** LOW — ~170µs per panel, 4 panels = ~680µs/frame

These two functions both split content by `\n` and iterate all lines, calling `lipgloss.Width()` per line. They could be combined into a single pass.

## 4. Recommendations (Priority Order)

### P0: Cache commit graph (HIGH impact, LOW effort)

Cache `BuildGraph()` result and `MaxGraphWidth()` in `CommitsModel`. Invalidate only when `Commits` slice changes (compare length or a generation counter). This eliminates ~2ms per render frame.

```go
type CommitsModel struct {
    // ... existing fields ...
    cachedGraph    []GraphLine
    cachedGraphW   int
    graphGeneration int  // bumped when Commits changes
}
```

### P1: Cache `displayItems()` and `filteredIndices()` (MEDIUM impact, LOW effort)

Cache the result of `displayItems()` in `TablesModel`, invalidating when `Tables`, `Filter`, or `Collapsed` changes. Same pattern for `filteredIndices()` in `CommitsModel` and `BranchesModel`.

### P2: Selective/partial refresh (HIGH impact, MEDIUM effort)

Introduce targeted refresh commands:
- `refreshWorkingSet()` — only reloads tables + status (for stage/unstage)
- `refreshAll()` — current `loadData()` behavior (for branch operations, commits)

This would reduce staging from 8 CLI calls to 2-3, saving ~400ms.

### P3: Eliminate redundant `CurrentBranch()` (MEDIUM impact, LOW effort)

Remove the separate `CurrentBranch()` goroutine in `loadData()`. Instead, extract the current branch from the `Branches()` result by finding the branch with `IsCurrent == true`.

### P4: Pre-compute styled graph lines (MEDIUM impact, LOW effort)

`BuildGraph` creates `lipgloss.NewStyle()` per lane per commit (line 78). These styles are deterministic per lane index — pre-compute and reuse them.

### P5: Combine `clipLines` + `truncateToVisualHeight` (LOW impact, LOW effort)

Merge into a single pass that truncates width and counts visual height simultaneously, avoiding the double `strings.Split` and double `lipgloss.Width()` per line.

### P6: Skip ANSI stripping for JSON queries (LOW impact, LOW effort)

Add a `runner.ExecJSON()` or pass `--no-color` flag for JSON queries to avoid unnecessary `stripANSI()` allocation.

### P7: Reduce lipgloss style allocations (LOW-MEDIUM impact, MEDIUM effort)

Pre-allocate commonly-used styles as package-level variables (many already are, but `BuildGraph` creates new ones per-call). Consider using `lipgloss.Style` pointers or caching styled strings for repeated values.

### P8: Combine diff processing passes (LOW impact, MEDIUM effort)

Merge `SanitizeForDisplay` + `colorizeDiff` into a single pass that sanitizes and colorizes simultaneously, avoiding intermediate string allocation.

### P9: Explore `dolt sql-server` for eliminating subprocess overhead (HIGH impact, HIGH effort)

Long-term: instead of spawning a new `dolt` process per query, connect to `dolt sql-server` via SQL protocol. This would eliminate the ~150-200ms process startup cost per query, reducing typical operations from 400ms to potentially <50ms. This is a significant architectural change.

## 5. Benchmark Data

Raw benchmark output is in `tmp/bench-results.txt`.
CPU profile: `tmp/cpu.prof` (view with `go tool pprof tmp/cpu.prof`)
Memory profile: `tmp/mem.prof` (view with `go tool pprof tmp/mem.prof`)

### Reproduction

```bash
# Run benchmarks
go test -bench=. -benchmem -count=3 -cpuprofile=tmp/cpu.prof -memprofile=tmp/mem.prof ./tmp/

# Time individual dolt CLI calls
cd tmp/test-dolt
/usr/bin/time -f "%e seconds" dolt sql -r json -q "SELECT active_branch() AS b"
```
