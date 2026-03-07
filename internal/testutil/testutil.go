// Package testutil provides shared test infrastructure for lazydolt tests.
// It manages temporary dolt repositories with consistent seed data,
// avoiding import cycles between internal/dolt/ and internal/ui/ test packages.
package testutil

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// DoltTestRepo manages a temporary dolt repository for tests.
type DoltTestRepo struct {
	Dir      string // absolute path to the temp dolt repo
	DoltPath string // path to the dolt binary
	t        testing.TB
}

// NewDoltTestRepo creates a fresh dolt repo in a temp directory.
// It runs `dolt init` and returns a helper for executing commands.
// Cleanup is automatic via t.Cleanup().
func NewDoltTestRepo(t testing.TB) *DoltTestRepo {
	t.Helper()

	doltPath, err := exec.LookPath("dolt")
	if err != nil {
		t.Fatalf("dolt not found in PATH: %v", err)
	}

	dir := t.TempDir() // automatically cleaned up

	r := &DoltTestRepo{
		Dir:      dir,
		DoltPath: doltPath,
		t:        t,
	}

	// Configure user globally within isolated HOME (before init, which needs it)
	r.Exec("config", "--global", "--set", "user.email", "test@lazydolt.dev")
	r.Exec("config", "--global", "--set", "user.name", "Test User")
	r.Exec("init")

	return r
}

// Exec runs a dolt command in the test repo and returns stdout.
// Fails the test on error.
func (r *DoltTestRepo) Exec(args ...string) string {
	r.t.Helper()
	out, err := r.ExecErr(args...)
	if err != nil {
		r.t.Fatalf("dolt %s: %v", strings.Join(args, " "), err)
	}
	return out
}

// ExecErr runs a dolt command and returns stdout + error.
// Does NOT fail the test on error (for testing error paths).
func (r *DoltTestRepo) ExecErr(args ...string) (string, error) {
	r.t.Helper()
	cmd := exec.Command(r.DoltPath, args...)
	cmd.Dir = r.Dir

	// Ensure dolt doesn't try to use a global config that might interfere
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("HOME=%s", r.Dir),
		"DOLT_ROOT_PATH="+filepath.Join(r.Dir, ".dolt_root"),
	)

	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr := strings.TrimSpace(string(exitErr.Stderr))
			return string(out), fmt.Errorf("%s (stderr: %s)", err, stderr)
		}
		return string(out), err
	}
	return strings.TrimSpace(string(out)), nil
}

// SQL runs a SQL statement via `dolt sql -q`.
// Fails the test on error.
func (r *DoltTestRepo) SQL(query string) {
	r.t.Helper()
	r.Exec("sql", "-q", query)
}

// SQLBatch runs multiple SQL statements in one call via `dolt sql -q`.
func (r *DoltTestRepo) SQLBatch(queries ...string) {
	r.t.Helper()
	combined := strings.Join(queries, ";\n")
	r.Exec("sql", "-q", combined)
}

// Commit stages all changes and creates a commit with the given message.
func (r *DoltTestRepo) Commit(msg string) {
	r.t.Helper()
	r.Exec("add", "-A")
	r.Exec("commit", "-m", msg)
}

// WriteFile creates a file in the test repo directory.
func (r *DoltTestRepo) WriteFile(name, content string) {
	r.t.Helper()
	path := filepath.Join(r.Dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		r.t.Fatalf("creating parent dirs for %s: %v", name, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		r.t.Fatalf("writing %s: %v", name, err)
	}
}

// PopulateTestData creates a standard test dataset:
//   - Committed: users table with 2 rows (Alice, Bob)
//   - Uncommitted: orders table (unstaged)
//   - Uncommitted: extra row in users (unstaged)
func PopulateTestData(r *DoltTestRepo) {
	r.t.Helper()

	// Create and populate users table, then commit
	r.SQLBatch(
		"CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(100), email VARCHAR(200))",
		"INSERT INTO users VALUES (1, 'Alice', 'alice@example.com')",
		"INSERT INTO users VALUES (2, 'Bob', 'bob@example.com')",
	)
	r.Commit("Add users table with initial data")

	// Create orders table (uncommitted, unstaged)
	r.SQL("CREATE TABLE orders (id INT PRIMARY KEY, user_id INT, total DECIMAL(10,2))")

	// Add a row to users (uncommitted, unstaged)
	r.SQL("INSERT INTO users VALUES (3, 'Charlie', 'charlie@example.com')")
}
