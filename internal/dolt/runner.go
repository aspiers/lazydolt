// Package dolt wraps the dolt CLI for executing commands and parsing output.
package dolt

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// ansiRegex strips ANSI escape codes from dolt output.
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// Runner executes dolt CLI commands against a repository directory.
type Runner struct {
	DoltPath string // path to dolt binary
	RepoDir  string // working directory for commands
}

// NewRunner creates a Runner for the given repository directory.
// It verifies that the dolt binary exists and the directory is a dolt repo.
func NewRunner(repoDir string) (*Runner, error) {
	doltPath, err := exec.LookPath("dolt")
	if err != nil {
		return nil, fmt.Errorf("dolt not found in PATH: %w", err)
	}

	absDir, err := filepath.Abs(repoDir)
	if err != nil {
		return nil, fmt.Errorf("invalid repo dir %q: %w", repoDir, err)
	}

	dotDolt := filepath.Join(absDir, ".dolt")
	info, err := os.Stat(dotDolt)
	if err != nil || !info.IsDir() {
		return nil, fmt.Errorf("%q is not a dolt repository (no .dolt directory)", absDir)
	}

	return &Runner{
		DoltPath: doltPath,
		RepoDir:  absDir,
	}, nil
}

// Exec runs a dolt CLI command and returns stdout with ANSI codes stripped.
// Returns an error containing stderr on non-zero exit.
func (r *Runner) Exec(args ...string) (string, error) {
	cmd := exec.Command(r.DoltPath, args...)
	cmd.Dir = r.RepoDir

	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr := strings.TrimSpace(string(exitErr.Stderr))
			return "", fmt.Errorf("dolt %s: %s", strings.Join(args, " "), stderr)
		}
		return "", fmt.Errorf("dolt %s: %w", strings.Join(args, " "), err)
	}

	return stripANSI(string(out)), nil
}

// SQL runs a SQL query via 'dolt sql -r json' and returns parsed rows.
// Dolt returns JSON like {"rows": [{...}, {...}]} or {} for empty results.
func (r *Runner) SQL(query string) ([]map[string]interface{}, error) {
	out, err := r.Exec("sql", "-r", "json", "-q", query)
	if err != nil {
		return nil, err
	}

	out = strings.TrimSpace(out)
	if out == "" || out == "{}" {
		return nil, nil
	}

	var result struct {
		Rows []map[string]interface{} `json:"rows"`
	}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		return nil, fmt.Errorf("parsing dolt sql output: %w", err)
	}

	return result.Rows, nil
}

// stripANSI removes ANSI escape codes from a string.
func stripANSI(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}
