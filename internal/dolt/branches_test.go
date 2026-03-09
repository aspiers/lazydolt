package dolt

import (
	"testing"

	"github.com/aspiers/lazydolt/internal/testutil"
)

func TestBranches_DefaultBranch(t *testing.T) {
	runner, _ := newTestRunner(t)
	branches, err := runner.Branches("")
	if err != nil {
		t.Fatalf("Branches(): %v", err)
	}
	if len(branches) == 0 {
		t.Fatal("expected at least one branch")
	}

	found := false
	for _, b := range branches {
		if b.Name == "main" && b.IsCurrent {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'main' branch with IsCurrent=true, got %+v", branches)
	}
}

func TestBranches_MultipleBranches(t *testing.T) {
	runner, repo := newTestRunner(t)
	testutil.PopulateTestData(repo)
	repo.Exec("branch", "feature-test")

	branches, err := runner.Branches("")
	if err != nil {
		t.Fatalf("Branches(): %v", err)
	}
	if len(branches) < 2 {
		t.Fatalf("expected at least 2 branches, got %d", len(branches))
	}

	names := make(map[string]bool)
	for _, b := range branches {
		names[b.Name] = true
	}
	if !names["main"] || !names["feature-test"] {
		t.Errorf("expected 'main' and 'feature-test', got %v", names)
	}
}
