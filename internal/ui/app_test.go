package ui

import (
	"bytes"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/muesli/termenv"

	"github.com/aspiers/lazydolt/internal/dolt"
	"github.com/aspiers/lazydolt/internal/testutil"
)

func init() {
	lipgloss.SetColorProfile(termenv.ANSI)
}

// newTestApp creates an App backed by a real dolt repo with test data.
func newTestApp(t testing.TB) App {
	t.Helper()
	repo := testutil.NewDoltTestRepo(t)
	testutil.PopulateTestData(repo)
	runner, err := dolt.NewRunner(repo.Dir)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	return NewApp(runner)
}

func TestApp_LaunchAndShowData(t *testing.T) {
	tm := teatest.NewTestModel(t, newTestApp(t),
		teatest.WithInitialTermSize(120, 40),
	)

	// Wait for the app to load data and display the tables panel
	teatest.WaitFor(t, tm.Output(),
		func(bts []byte) bool {
			return bytes.Contains(bts, []byte("orders")) &&
				bytes.Contains(bts, []byte("users"))
		},
		teatest.WithDuration(5*time.Second),
	)

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

func TestApp_QuitWithQ(t *testing.T) {
	tm := teatest.NewTestModel(t, newTestApp(t),
		teatest.WithInitialTermSize(120, 40),
	)

	// Wait for initial data to load
	teatest.WaitFor(t, tm.Output(),
		func(bts []byte) bool {
			return bytes.Contains(bts, []byte("Status"))
		},
		teatest.WithDuration(5*time.Second),
	)

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

func TestApp_QuitWithCtrlC(t *testing.T) {
	tm := teatest.NewTestModel(t, newTestApp(t),
		teatest.WithInitialTermSize(120, 40),
	)

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(),
		func(bts []byte) bool {
			return bytes.Contains(bts, []byte("Status"))
		},
		teatest.WithDuration(5*time.Second),
	)

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

func TestApp_TabCyclesFocus(t *testing.T) {
	tm := teatest.NewTestModel(t, newTestApp(t),
		teatest.WithInitialTermSize(120, 40),
	)

	// Wait for data
	teatest.WaitFor(t, tm.Output(),
		func(bts []byte) bool {
			return bytes.Contains(bts, []byte("orders"))
		},
		teatest.WithDuration(5*time.Second),
	)

	// Tab should cycle: Tables → Branches
	tm.Send(tea.KeyMsg{Type: tea.KeyTab})
	teatest.WaitFor(t, tm.Output(),
		func(bts []byte) bool {
			// Branches panel hints should show "checkout"
			return bytes.Contains(bts, []byte("checkout"))
		},
		teatest.WithDuration(2*time.Second),
	)

	// Tab again: Branches → Commits
	tm.Send(tea.KeyMsg{Type: tea.KeyTab})
	teatest.WaitFor(t, tm.Output(),
		func(bts []byte) bool {
			// Commits panel hints should show "details"
			return bytes.Contains(bts, []byte("details"))
		},
		teatest.WithDuration(2*time.Second),
	)

	// Tab again: Commits → Main
	tm.Send(tea.KeyMsg{Type: tea.KeyTab})
	teatest.WaitFor(t, tm.Output(),
		func(bts []byte) bool {
			// Main panel hints should show "j/k"
			return bytes.Contains(bts, []byte("j/k"))
		},
		teatest.WithDuration(2*time.Second),
	)

	// Tab again: Main → Tables (back to start)
	tm.Send(tea.KeyMsg{Type: tea.KeyTab})
	teatest.WaitFor(t, tm.Output(),
		func(bts []byte) bool {
			return bytes.Contains(bts, []byte("stage"))
		},
		teatest.WithDuration(2*time.Second),
	)

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

func TestApp_NavigateTablesWithJK(t *testing.T) {
	tm := teatest.NewTestModel(t, newTestApp(t),
		teatest.WithInitialTermSize(120, 40),
	)

	// Wait for tables to load
	teatest.WaitFor(t, tm.Output(),
		func(bts []byte) bool {
			return bytes.Contains(bts, []byte("orders")) &&
				bytes.Contains(bts, []byte("users"))
		},
		teatest.WithDuration(5*time.Second),
	)

	// Navigate down with j — should change the diff preview
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	time.Sleep(200 * time.Millisecond)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})

	// After moving down, we should see a specific table's diff
	teatest.WaitFor(t, tm.Output(),
		func(bts []byte) bool {
			// The diff title should change to show a specific table
			return bytes.Contains(bts, []byte("Diff:"))
		},
		teatest.WithDuration(2*time.Second),
	)

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

func TestApp_SpaceStagesTable(t *testing.T) {
	tm := teatest.NewTestModel(t, newTestApp(t),
		teatest.WithInitialTermSize(120, 40),
	)

	// Wait for tables to load — should show unstaged changes
	teatest.WaitFor(t, tm.Output(),
		func(bts []byte) bool {
			return bytes.Contains(bts, []byte("Unstaged"))
		},
		teatest.WithDuration(5*time.Second),
	)

	// Move to an unstaged table entry
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	time.Sleep(300 * time.Millisecond)

	// Press Space to stage
	tm.Send(tea.KeyMsg{Type: tea.KeySpace})

	// Wait for refresh — the table should move to staged
	teatest.WaitFor(t, tm.Output(),
		func(bts []byte) bool {
			return bytes.Contains(bts, []byte("Staged"))
		},
		teatest.WithDuration(5*time.Second),
	)

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

func TestApp_HelpOverlay(t *testing.T) {
	tm := teatest.NewTestModel(t, newTestApp(t),
		teatest.WithInitialTermSize(120, 40),
	)

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(),
		func(bts []byte) bool {
			return bytes.Contains(bts, []byte("Status"))
		},
		teatest.WithDuration(5*time.Second),
	)

	// Press ? to open help
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	teatest.WaitFor(t, tm.Output(),
		func(bts []byte) bool {
			return bytes.Contains(bts, []byte("Keyboard Shortcuts"))
		},
		teatest.WithDuration(2*time.Second),
	)

	// Press ? again to close help
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	teatest.WaitFor(t, tm.Output(),
		func(bts []byte) bool {
			// Help should be gone, back to normal view with tables
			return bytes.Contains(bts, []byte("stage")) &&
				!bytes.Contains(bts, []byte("Keyboard Shortcuts"))
		},
		teatest.WithDuration(2*time.Second),
	)

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

func TestApp_NumberKeysJumpToPanel(t *testing.T) {
	tm := teatest.NewTestModel(t, newTestApp(t),
		teatest.WithInitialTermSize(120, 40),
	)

	// Wait for data
	teatest.WaitFor(t, tm.Output(),
		func(bts []byte) bool {
			return bytes.Contains(bts, []byte("orders"))
		},
		teatest.WithDuration(5*time.Second),
	)

	// Press 2 to jump to Branches
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
	teatest.WaitFor(t, tm.Output(),
		func(bts []byte) bool {
			return bytes.Contains(bts, []byte("checkout"))
		},
		teatest.WithDuration(2*time.Second),
	)

	// Press 3 to jump to Commits
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	teatest.WaitFor(t, tm.Output(),
		func(bts []byte) bool {
			return bytes.Contains(bts, []byte("details"))
		},
		teatest.WithDuration(2*time.Second),
	)

	// Press 1 to jump back to Tables
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")})
	teatest.WaitFor(t, tm.Output(),
		func(bts []byte) bool {
			return bytes.Contains(bts, []byte("stage"))
		},
		teatest.WithDuration(2*time.Second),
	)

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}
