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
	"sync"
	"time"

	"github.com/aspiers/lazydolt/internal/domain"
)

// ansiRegex strips ANSI escape codes from dolt output.
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// CLIRunner executes dolt CLI commands against a repository directory
// by spawning dolt subprocesses. It implements the Runner interface.
type CLIRunner struct {
	doltPath string // path to dolt binary
	repoDir  string // working directory for commands

	logMu  sync.Mutex
	cmdLog []domain.CommandLogEntry
}

// RepoDir returns the absolute path to the dolt repository.
func (r *CLIRunner) RepoDir() string {
	return r.repoDir
}

// NewCLIRunner creates a CLIRunner for the given repository directory.
// It verifies that the dolt binary exists and the directory is a dolt repo.
func NewCLIRunner(repoDir string) (*CLIRunner, error) {
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

	return &CLIRunner{
		doltPath: doltPath,
		repoDir:  absDir,
	}, nil
}

// Exec runs a dolt CLI command and returns stdout with ANSI codes stripped.
// Returns an error containing stderr on non-zero exit.
func (r *CLIRunner) Exec(args ...string) (string, error) {
	return r.exec(args, true)
}

// ExecRaw runs a dolt CLI command and returns stdout without stripping
// ANSI codes. Use this for commands whose output is known to be
// ANSI-free (e.g. JSON output from "dolt sql -r json").
func (r *CLIRunner) ExecRaw(args ...string) (string, error) {
	return r.exec(args, false)
}

// exec is the shared implementation for Exec and ExecRaw.
func (r *CLIRunner) exec(args []string, stripANSICodes bool) (string, error) {
	cmd := exec.Command(r.doltPath, args...)
	cmd.Dir = r.repoDir
	cmdStr := "dolt " + strings.Join(args, " ")

	out, err := cmd.Output()
	if err != nil {
		errMsg := ""
		if exitErr, ok := err.(*exec.ExitError); ok {
			errMsg = strings.TrimSpace(string(exitErr.Stderr))
		} else {
			errMsg = err.Error()
		}
		r.logCommand(cmdStr, errMsg, true)
		return "", fmt.Errorf("%s: %s", cmdStr, errMsg)
	}

	result := string(out)
	if stripANSICodes {
		result = stripANSI(result)
	}
	r.logCommand(cmdStr, result, false)
	return result, nil
}

// logCommand records a command execution in the internal log.
func (r *CLIRunner) logCommand(cmd, output string, isErr bool) {
	r.logMu.Lock()
	defer r.logMu.Unlock()
	r.cmdLog = append(r.cmdLog, domain.CommandLogEntry{
		Command: cmd,
		Output:  output,
		Error:   isErr,
		Time:    time.Now(),
	})
	// Keep at most 100 entries
	if len(r.cmdLog) > 100 {
		r.cmdLog = r.cmdLog[len(r.cmdLog)-100:]
	}
}

// CommandLog returns a copy of all recorded command log entries.
func (r *CLIRunner) CommandLog() []domain.CommandLogEntry {
	r.logMu.Lock()
	defer r.logMu.Unlock()
	result := make([]domain.CommandLogEntry, len(r.cmdLog))
	copy(result, r.cmdLog)
	return result
}

// SQL runs a SQL query via 'dolt sql -r json' and returns parsed rows.
// Dolt returns JSON like {"rows": [{...}, {...}]} or {} for empty results.
func (r *CLIRunner) SQL(query string) ([]map[string]interface{}, error) {
	out, err := r.ExecRaw("sql", "-r", "json", "-q", query)
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

// SQLRaw runs a SQL query via 'dolt sql -r tabular' and returns the
// human-readable tabular output as a string. Useful for displaying
// arbitrary query results to the user.
func (r *CLIRunner) SQLRaw(query string) (string, error) {
	return r.Exec("sql", "-r", "tabular", "-q", query)
}

// DiffStatBetween returns per-table change statistics between two revisions
// using the dolt_diff_stat SQL function.
func (r *CLIRunner) DiffStatBetween(fromRef, toRef string) ([]domain.DiffStatEntry, error) {
	query := fmt.Sprintf(
		`SELECT table_name, rows_added, rows_deleted, rows_modified FROM dolt_diff_stat(%q, %q)`,
		fromRef, toRef,
	)
	rows, err := r.SQL(query)
	if err != nil {
		return nil, err
	}

	var entries []domain.DiffStatEntry
	for _, row := range rows {
		e := domain.DiffStatEntry{}
		if v, ok := row["table_name"].(string); ok {
			e.TableName = v
		}
		if v, ok := row["rows_added"].(float64); ok {
			e.RowsAdded = int(v)
		}
		if v, ok := row["rows_deleted"].(float64); ok {
			e.RowsDeleted = int(v)
		}
		if v, ok := row["rows_modified"].(float64); ok {
			e.RowsModified = int(v)
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// stripANSI removes ANSI escape codes from a string.
func stripANSI(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}
