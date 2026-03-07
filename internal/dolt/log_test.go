package dolt

import (
	"testing"

	"github.com/aspiers/lazydolt/internal/testutil"
)

func TestLog_HasCommits(t *testing.T) {
	runner, repo := newTestRunner(t)
	testutil.PopulateTestData(repo) // creates 1 commit + init

	commits, err := runner.Log(50)
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

	commits, err := runner.Log(1)
	if err != nil {
		t.Fatalf("Log(1): %v", err)
	}
	if len(commits) != 1 {
		t.Errorf("Log(1) returned %d commits, want 1", len(commits))
	}
}

func TestLog_Order(t *testing.T) {
	runner, repo := newTestRunner(t)
	testutil.PopulateTestData(repo)

	commits, err := runner.Log(50)
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
