package dolt

import (
	"testing"

	"github.com/aspiers/lazydolt/internal/testutil"
)

func TestDiffText_ModifiedTable(t *testing.T) {
	runner, repo := newTestRunner(t)
	testutil.PopulateTestData(repo) // users has an uncommitted row

	diff, err := runner.DiffText("users", false)
	if err != nil {
		t.Fatalf("DiffText(users): %v", err)
	}
	if diff == "" {
		t.Error("DiffText(users) returned empty string, expected diff content for modified table")
	}
}

func TestDiffText_NoChanges(t *testing.T) {
	runner, repo := newTestRunner(t)
	// Create a table and commit it — no further changes
	repo.SQL("CREATE TABLE clean (id INT PRIMARY KEY)")
	repo.Commit("add clean table")

	diff, err := runner.DiffText("clean", false)
	if err != nil {
		t.Fatalf("DiffText(clean): %v", err)
	}
	if diff != "" {
		t.Errorf("DiffText(clean) = %q, want empty string for unchanged table", diff)
	}
}

func TestDiffText_Staged(t *testing.T) {
	runner, repo := newTestRunner(t)
	testutil.PopulateTestData(repo)
	// Stage the users table
	repo.Exec("add", "users")

	diff, err := runner.DiffText("users", true)
	if err != nil {
		t.Fatalf("DiffText(users, staged): %v", err)
	}
	if diff == "" {
		t.Error("DiffText(users, staged) returned empty string, expected staged diff content")
	}
}

func TestDiffStat_HasStats(t *testing.T) {
	runner, repo := newTestRunner(t)
	testutil.PopulateTestData(repo)

	stat, err := runner.DiffStat("users", false)
	if err != nil {
		t.Fatalf("DiffStat(users): %v", err)
	}
	if stat == "" {
		t.Error("DiffStat(users) returned empty string, expected statistics")
	}
}
