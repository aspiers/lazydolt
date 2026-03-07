package testutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewDoltTestRepo(t *testing.T) {
	r := NewDoltTestRepo(t)

	// Verify the temp dir exists and has a .dolt directory
	dotDolt := filepath.Join(r.Dir, ".dolt")
	info, err := os.Stat(dotDolt)
	if err != nil {
		t.Fatalf("expected .dolt directory in %s: %v", r.Dir, err)
	}
	if !info.IsDir() {
		t.Fatalf(".dolt is not a directory")
	}
}

func TestExec_Version(t *testing.T) {
	r := NewDoltTestRepo(t)
	out := r.Exec("version")
	if !strings.Contains(out, "dolt version") {
		t.Fatalf("expected 'dolt version' in output, got: %s", out)
	}
}

func TestExecErr_InvalidCommand(t *testing.T) {
	r := NewDoltTestRepo(t)
	_, err := r.ExecErr("not-a-real-command")
	if err == nil {
		t.Fatal("expected error for invalid dolt command")
	}
}

func TestSQL_SimpleQuery(t *testing.T) {
	r := NewDoltTestRepo(t)
	// Just verify it doesn't fail — SQL output goes to stdout
	r.SQL("SELECT 1 AS test")
}

func TestSQLBatch(t *testing.T) {
	r := NewDoltTestRepo(t)
	r.SQLBatch(
		"CREATE TABLE t1 (id INT PRIMARY KEY)",
		"INSERT INTO t1 VALUES (1)",
	)
	// Verify the table exists by querying it
	r.SQL("SELECT * FROM t1")
}

func TestCommit(t *testing.T) {
	r := NewDoltTestRepo(t)
	r.SQL("CREATE TABLE t1 (id INT PRIMARY KEY)")
	r.Commit("test commit")

	// Verify the commit exists in the log
	out := r.Exec("log", "--oneline")
	if !strings.Contains(out, "test commit") {
		t.Fatalf("expected 'test commit' in log output, got: %s", out)
	}
}

func TestWriteFile(t *testing.T) {
	r := NewDoltTestRepo(t)
	r.WriteFile("subdir/test.txt", "hello world")

	content, err := os.ReadFile(filepath.Join(r.Dir, "subdir", "test.txt"))
	if err != nil {
		t.Fatalf("reading written file: %v", err)
	}
	if string(content) != "hello world" {
		t.Fatalf("expected 'hello world', got: %s", string(content))
	}
}

func TestPopulateTestData(t *testing.T) {
	r := NewDoltTestRepo(t)
	PopulateTestData(r)

	// Verify committed data: users table with at least Alice and Bob
	out := r.Exec("sql", "-r", "json", "-q", "SELECT COUNT(*) AS cnt FROM users")
	// Should have 3 rows (2 committed + 1 uncommitted)
	if !strings.Contains(out, `"cnt":3`) {
		t.Fatalf("expected 3 users rows, got: %s", out)
	}

	// Verify orders table exists (uncommitted)
	out = r.Exec("sql", "-r", "json", "-q", "SELECT COUNT(*) AS cnt FROM orders")
	if !strings.Contains(out, `"cnt":0`) {
		t.Fatalf("expected 0 orders rows, got: %s", out)
	}

	// Verify there are uncommitted changes (dirty working set)
	out = r.Exec("sql", "-r", "json", "-q", "SELECT * FROM dolt_status")
	if !strings.Contains(out, "users") || !strings.Contains(out, "orders") {
		t.Fatalf("expected both users and orders in dolt_status, got: %s", out)
	}

	// Verify there is at least 1 commit (the seed data commit)
	out = r.Exec("log", "--oneline")
	if !strings.Contains(out, "Add users table") {
		t.Fatalf("expected seed data commit in log, got: %s", out)
	}
}
