package dolt

import (
	"testing"

	"github.com/aspiers/lazydolt/internal/testutil"
)

func TestAdd_StagesTable(t *testing.T) {
	runner, repo := newTestRunner(t)
	repo.SQL("CREATE TABLE stage_test (id INT PRIMARY KEY)")

	if err := runner.Add("stage_test"); err != nil {
		t.Fatalf("Add(stage_test): %v", err)
	}

	status, err := runner.Status()
	if err != nil {
		t.Fatalf("Status(): %v", err)
	}
	for _, s := range status {
		if s.TableName == "stage_test" {
			if !s.Staged {
				t.Error("stage_test should be staged after Add()")
			}
			return
		}
	}
	t.Error("stage_test not found in status after Add()")
}

func TestReset_UnstagesTable(t *testing.T) {
	runner, repo := newTestRunner(t)
	repo.SQL("CREATE TABLE reset_test (id INT PRIMARY KEY)")
	repo.Exec("add", "reset_test")

	if err := runner.Reset("reset_test"); err != nil {
		t.Fatalf("Reset(reset_test): %v", err)
	}

	status, err := runner.Status()
	if err != nil {
		t.Fatalf("Status(): %v", err)
	}
	for _, s := range status {
		if s.TableName == "reset_test" {
			if s.Staged {
				t.Error("reset_test should be unstaged after Reset()")
			}
			return
		}
	}
	t.Error("reset_test not found in status after Reset()")
}

func TestCommit_CreatesCommit(t *testing.T) {
	runner, repo := newTestRunner(t)
	repo.SQL("CREATE TABLE commit_test (id INT PRIMARY KEY)")
	repo.Exec("add", "-A")

	beforeCommits, err := runner.Log("", 100, "")
	if err != nil {
		t.Fatalf("Log before commit: %v", err)
	}

	hash, err := runner.Commit("test commit message")
	if err != nil {
		t.Fatalf("Commit(): %v", err)
	}
	if hash == "" {
		t.Error("Commit() returned empty hash")
	}

	afterCommits, err := runner.Log("", 100, "")
	if err != nil {
		t.Fatalf("Log after commit: %v", err)
	}
	if len(afterCommits) != len(beforeCommits)+1 {
		t.Errorf("expected %d commits after committing, got %d", len(beforeCommits)+1, len(afterCommits))
	}
}

func TestCommit_NothingStaged(t *testing.T) {
	runner, _ := newTestRunner(t)
	_, err := runner.Commit("should fail")
	if err == nil {
		t.Error("Commit() with nothing staged should return error")
	}
}

func TestNewBranch_Creates(t *testing.T) {
	runner, repo := newTestRunner(t)
	testutil.PopulateTestData(repo)

	if err := runner.NewBranch("test-branch"); err != nil {
		t.Fatalf("NewBranch(test-branch): %v", err)
	}

	branches, err := runner.Branches("")
	if err != nil {
		t.Fatalf("Branches(): %v", err)
	}
	found := false
	for _, b := range branches {
		if b.Name == "test-branch" {
			found = true
			break
		}
	}
	if !found {
		t.Error("test-branch not found in Branches() after NewBranch()")
	}
}

func TestCheckout_Switches(t *testing.T) {
	runner, repo := newTestRunner(t)
	testutil.PopulateTestData(repo)
	repo.Exec("branch", "other-branch")

	if err := runner.Checkout("other-branch"); err != nil {
		t.Fatalf("Checkout(other-branch): %v", err)
	}

	branch, err := runner.CurrentBranch()
	if err != nil {
		t.Fatalf("CurrentBranch(): %v", err)
	}
	if branch != "other-branch" {
		t.Errorf("CurrentBranch() = %q, want %q", branch, "other-branch")
	}
}

func TestDeleteBranch_Removes(t *testing.T) {
	runner, repo := newTestRunner(t)
	testutil.PopulateTestData(repo)
	repo.Exec("branch", "to-delete")

	if err := runner.DeleteBranch("to-delete"); err != nil {
		t.Fatalf("DeleteBranch(to-delete): %v", err)
	}

	branches, err := runner.Branches("")
	if err != nil {
		t.Fatalf("Branches(): %v", err)
	}
	for _, b := range branches {
		if b.Name == "to-delete" {
			t.Error("to-delete still in Branches() after DeleteBranch()")
		}
	}
}
