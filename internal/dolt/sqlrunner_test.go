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

func TestSQLRunner_SQLMethodMatchesCLI(t *testing.T) {
	repo := testutil.NewDoltTestRepo(t)
	testutil.PopulateTestData(repo)

	sqlRunner, err := NewSQLRunner(repo.Dir)
	if err != nil {
		t.Fatalf("NewSQLRunner: %v", err)
	}
	defer sqlRunner.Close()

	cliRunner := sqlRunner.CLIRunner

	// Compare Status() results (both use SQL() internally).
	sqlStatus, err := sqlRunner.Status()
	if err != nil {
		t.Fatalf("SQLRunner.Status: %v", err)
	}
	cliStatus, err := cliRunner.Status()
	if err != nil {
		t.Fatalf("CLIRunner.Status: %v", err)
	}
	if len(sqlStatus) != len(cliStatus) {
		t.Errorf("Status: SQL returned %d entries, CLI returned %d", len(sqlStatus), len(cliStatus))
	}

	// Compare CurrentBranch() results.
	sqlBranch, err := sqlRunner.CurrentBranch()
	if err != nil {
		t.Fatalf("SQLRunner.CurrentBranch: %v", err)
	}
	cliBranch, err := cliRunner.CurrentBranch()
	if err != nil {
		t.Fatalf("CLIRunner.CurrentBranch: %v", err)
	}
	if sqlBranch != cliBranch {
		t.Errorf("CurrentBranch: SQL=%q, CLI=%q", sqlBranch, cliBranch)
	}

	// Compare Branches() results.
	sqlBranches, err := sqlRunner.Branches(BranchOrderByDate)
	if err != nil {
		t.Fatalf("SQLRunner.Branches: %v", err)
	}
	cliBranches, err := cliRunner.Branches(BranchOrderByDate)
	if err != nil {
		t.Fatalf("CLIRunner.Branches: %v", err)
	}
	if len(sqlBranches) != len(cliBranches) {
		t.Errorf("Branches: SQL returned %d, CLI returned %d", len(sqlBranches), len(cliBranches))
	} else {
		for i := range sqlBranches {
			if sqlBranches[i].Name != cliBranches[i].Name {
				t.Errorf("Branch[%d].Name: SQL=%q, CLI=%q", i, sqlBranches[i].Name, cliBranches[i].Name)
			}
			if sqlBranches[i].IsCurrent != cliBranches[i].IsCurrent {
				t.Errorf("Branch[%d].IsCurrent: SQL=%v, CLI=%v", i, sqlBranches[i].IsCurrent, cliBranches[i].IsCurrent)
			}
		}
	}

	// Compare Tables() results.
	sqlTables, err := sqlRunner.Tables()
	if err != nil {
		t.Fatalf("SQLRunner.Tables: %v", err)
	}
	cliTables, err := cliRunner.Tables()
	if err != nil {
		t.Fatalf("CLIRunner.Tables: %v", err)
	}
	if len(sqlTables) != len(cliTables) {
		t.Errorf("Tables: SQL returned %d, CLI returned %d", len(sqlTables), len(cliTables))
		for _, st := range sqlTables {
			t.Logf("  SQL table: %s (status=%v)", st.Name, st.Status != nil)
		}
		for _, ct := range cliTables {
			t.Logf("  CLI table: %s (status=%v)", ct.Name, ct.Status != nil)
		}
	} else {
		for i := range sqlTables {
			if sqlTables[i].Name != cliTables[i].Name {
				t.Errorf("Table[%d].Name: SQL=%q, CLI=%q", i, sqlTables[i].Name, cliTables[i].Name)
			}
		}
	}
}

func TestSQLRunner_Mutations(t *testing.T) {
	repo := testutil.NewDoltTestRepo(t)
	testutil.PopulateTestData(repo)

	runner, err := NewSQLRunner(repo.Dir)
	if err != nil {
		t.Fatalf("NewSQLRunner: %v", err)
	}
	defer runner.Close()

	// Test Add + Commit cycle.
	if err := runner.AddAll(); err != nil {
		t.Fatalf("AddAll: %v", err)
	}

	hash, err := runner.Commit("test commit via sql-server")
	if err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if hash == "" {
		t.Error("Commit returned empty hash")
	}
	t.Logf("Committed %s", hash)

	// Verify the commit appears in the log.
	branch, _ := runner.CurrentBranch()
	commits, err := runner.Log(branch, 1, CommitOrderByDateDesc)
	if err != nil {
		t.Fatalf("Log: %v", err)
	}
	if len(commits) == 0 {
		t.Fatal("expected at least one commit in log")
	}
	if commits[0].Hash != hash {
		t.Errorf("latest commit hash = %q, want %q", commits[0].Hash, hash)
	}

	// Test branch operations.
	if err := runner.NewBranch("test-branch"); err != nil {
		t.Fatalf("NewBranch: %v", err)
	}

	branches, err := runner.Branches(BranchOrderByName)
	if err != nil {
		t.Fatalf("Branches: %v", err)
	}
	found := false
	for _, b := range branches {
		if b.Name == "test-branch" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find 'test-branch' in branches")
	}

	if err := runner.DeleteBranch("test-branch"); err != nil {
		t.Fatalf("DeleteBranch: %v", err)
	}

	// Test Schema.
	schema, err := runner.Schema("users")
	if err != nil {
		t.Fatalf("Schema: %v", err)
	}
	if schema.TableName != "users" {
		t.Errorf("Schema.TableName = %q, want %q", schema.TableName, "users")
	}
	if schema.CreateStatement == "" {
		t.Error("expected non-empty CreateStatement")
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
