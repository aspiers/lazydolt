package dolt

import (
	"testing"

	"github.com/aspiers/lazydolt/internal/testutil"
)

func TestLog_HasCommits(t *testing.T) {
	runner, repo := newTestRunner(t)
	testutil.PopulateTestData(repo) // creates 1 commit + init

	commits, err := runner.Log("", 50, "")
	if err != nil {
		t.Fatalf("Log(50): %v", err)
	}
	if len(commits) < 2 {
		t.Fatalf("expected at least 2 commits (init + data), got %d", len(commits))
	}
}

func TestLog_Limit(t *testing.T) {
	runner, repo := newTestRunner(t)
	testutil.PopulateTestData(repo)

	commits, err := runner.Log("", 1, "")
	if err != nil {
		t.Fatalf("Log(1): %v", err)
	}
	if len(commits) != 1 {
		t.Errorf("Log(1) returned %d commits, want 1", len(commits))
	}
}

func TestLog_Branch(t *testing.T) {
	runner, repo := newTestRunner(t)
	testutil.PopulateTestData(repo) // creates 1 commit + init on main

	// Create a feature branch with an extra commit
	repo.Exec("checkout", "-b", "feature")
	repo.SQL("INSERT INTO users VALUES (10, 'Feature User', 'feature@example.com')")
	repo.Exec("add", "-A")
	repo.Commit("Feature branch commit")

	// Switch back to main
	repo.Exec("checkout", "main")

	// Log for "feature" should include the feature branch commit
	featureCommits, err := runner.Log("feature", 50, "")
	if err != nil {
		t.Fatalf("Log('feature', 50): %v", err)
	}

	// Log for "" (current = main) should NOT include the feature commit
	mainCommits, err := runner.Log("", 50, "")
	if err != nil {
		t.Fatalf("Log('', 50): %v", err)
	}

	// Feature branch should have one more commit than main
	if len(featureCommits) != len(mainCommits)+1 {
		t.Errorf("feature has %d commits, main has %d; expected feature = main + 1",
			len(featureCommits), len(mainCommits))
	}

	// The latest feature commit should be the one we added
	if len(featureCommits) > 0 && featureCommits[0].Message != "Feature branch commit" {
		t.Errorf("latest feature commit = %q, want %q", featureCommits[0].Message, "Feature branch commit")
	}
}

func TestLog_Order(t *testing.T) {
	runner, repo := newTestRunner(t)
	testutil.PopulateTestData(repo)

	commits, err := runner.Log("", 50, "")
	if err != nil {
		t.Fatalf("Log(50): %v", err)
	}
	if len(commits) < 2 {
		t.Fatalf("need at least 2 commits to check order, got %d", len(commits))
	}
	// Most recent first
	if !commits[0].Date.After(commits[len(commits)-1].Date) && !commits[0].Date.Equal(commits[len(commits)-1].Date) {
		t.Errorf("expected newest first: %v > %v", commits[0].Date, commits[len(commits)-1].Date)
	}
}
