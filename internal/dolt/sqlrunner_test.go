package dolt

import (
	"testing"

	"github.com/aspiers/lazydolt/internal/testutil"
)

func TestNewSQLRunner_StartsServer(t *testing.T) {
	repo := testutil.NewDoltTestRepo(t)
	testutil.PopulateTestData(repo)

	runner, err := NewSQLRunner(repo.Dir)
	if err != nil {
		t.Fatalf("NewSQLRunner: %v", err)
	}
	defer runner.Close()

	// Verify the server is running and we have a connection.
	if runner.db == nil {
		t.Fatal("expected non-nil db connection")
	}
	if runner.serverPort == 0 {
		t.Fatal("expected non-zero server port")
	}

	// The SQLRunner should satisfy the Runner interface.
	var _ Runner = runner

	// Verify that SQL queries work through the connection.
	var branch string
	err = runner.db.QueryRow("SELECT active_branch() AS branch").Scan(&branch)
	if err != nil {
		t.Fatalf("querying active_branch: %v", err)
	}
	if branch != "main" {
		t.Errorf("active_branch() = %q, want %q", branch, "main")
	}
}

func TestSQLRunner_DelegatesToCLI(t *testing.T) {
	repo := testutil.NewDoltTestRepo(t)
	testutil.PopulateTestData(repo)

	runner, err := NewSQLRunner(repo.Dir)
	if err != nil {
		t.Fatalf("NewSQLRunner: %v", err)
	}
	defer runner.Close()

	// Status() currently delegates to CLIRunner. Verify it works.
	entries, err := runner.Status()
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	// PopulateTestData creates staged and unstaged changes.
	if len(entries) == 0 {
		t.Error("expected non-empty status from populated repo")
	}
}

func TestSQLRunner_RepoDir(t *testing.T) {
	repo := testutil.NewDoltTestRepo(t)

	runner, err := NewSQLRunner(repo.Dir)
	if err != nil {
		t.Fatalf("NewSQLRunner: %v", err)
	}
	defer runner.Close()

	if runner.RepoDir() != repo.Dir {
		t.Errorf("RepoDir() = %q, want %q", runner.RepoDir(), repo.Dir)
	}
}

func TestSQLRunner_Close(t *testing.T) {
	repo := testutil.NewDoltTestRepo(t)

	runner, err := NewSQLRunner(repo.Dir)
	if err != nil {
		t.Fatalf("NewSQLRunner: %v", err)
	}

	// Close should not error.
	if err := runner.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}

	// Double close should also not error.
	if err := runner.Close(); err != nil {
		t.Errorf("double Close: %v", err)
	}
}
