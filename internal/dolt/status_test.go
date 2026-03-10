package dolt

import (
	"testing"

	"github.com/aspiers/lazydolt/internal/testutil"
)

func newTestRunner(t *testing.T) (*CLIRunner, *testutil.DoltTestRepo) {
	t.Helper()
	repo := testutil.NewDoltTestRepo(t)
	runner, err := NewCLIRunner(repo.Dir)
	if err != nil {
		t.Fatalf("NewCLIRunner(%q): %v", repo.Dir, err)
	}
	return runner, repo
}

func TestStatus_CleanRepo(t *testing.T) {
	runner, _ := newTestRunner(t)
	status, err := runner.Status()
	if err != nil {
		t.Fatalf("Status(): %v", err)
	}
	if len(status) != 0 {
		t.Errorf("expected empty status for clean repo, got %d entries", len(status))
	}
}

func TestStatus_UntrackedTable(t *testing.T) {
	runner, repo := newTestRunner(t)
	repo.SQL("CREATE TABLE foo (id INT PRIMARY KEY)")

	status, err := runner.Status()
	if err != nil {
		t.Fatalf("Status(): %v", err)
	}
	if len(status) == 0 {
		t.Fatal("expected at least one status entry for new table")
	}

	found := false
	for _, s := range status {
		if s.TableName == "foo" && !s.Staged {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected unstaged entry for 'foo', got %+v", status)
	}
}

func TestStatus_StagedTable(t *testing.T) {
	runner, repo := newTestRunner(t)
	repo.SQL("CREATE TABLE bar (id INT PRIMARY KEY)")
	repo.Exec("add", "bar")

	status, err := runner.Status()
	if err != nil {
		t.Fatalf("Status(): %v", err)
	}

	found := false
	for _, s := range status {
		if s.TableName == "bar" && s.Staged {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected staged entry for 'bar', got %+v", status)
	}
}

func TestCurrentBranch(t *testing.T) {
	runner, _ := newTestRunner(t)
	branch, err := runner.CurrentBranch()
	if err != nil {
		t.Fatalf("CurrentBranch(): %v", err)
	}
	if branch != "main" {
		t.Errorf("CurrentBranch() = %q, want %q", branch, "main")
	}
}

func TestIsDirty_Clean(t *testing.T) {
	runner, _ := newTestRunner(t)
	dirty, err := runner.IsDirty()
	if err != nil {
		t.Fatalf("IsDirty(): %v", err)
	}
	if dirty {
		t.Error("IsDirty() = true for clean repo, want false")
	}
}

func TestIsDirty_WithChanges(t *testing.T) {
	runner, repo := newTestRunner(t)
	repo.SQL("CREATE TABLE test (id INT PRIMARY KEY)")
	dirty, err := runner.IsDirty()
	if err != nil {
		t.Fatalf("IsDirty(): %v", err)
	}
	if !dirty {
		t.Error("IsDirty() = false after creating table, want true")
	}
}
