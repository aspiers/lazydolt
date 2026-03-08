package dolt

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// ErrNothingToCommit is returned when there are no staged changes to commit.
var ErrNothingToCommit = errors.New("nothing to commit")

// commitHashRegex extracts the commit hash from dolt commit output.
var commitHashRegex = regexp.MustCompile(`commit\s+([a-z0-9]+)`)

// Add stages a table for commit.
func (r *Runner) Add(table string) error {
	_, err := r.Exec("add", table)
	return err
}

// AddAll stages all changed tables.
func (r *Runner) AddAll() error {
	_, err := r.Exec("add", ".")
	return err
}

// Reset unstages a table.
func (r *Runner) Reset(table string) error {
	_, err := r.Exec("reset", table)
	return err
}

// ResetAll unstages all tables.
func (r *Runner) ResetAll() error {
	_, err := r.Exec("reset")
	return err
}

// ResetSoft moves HEAD to the given commit, keeping working changes.
func (r *Runner) ResetSoft(commit string) error {
	_, err := r.Exec("reset", "--soft", commit)
	return err
}

// ResetHard moves HEAD to the given commit and discards all changes.
func (r *Runner) ResetHard(commit string) error {
	_, err := r.Exec("reset", "--hard", commit)
	return err
}

// Commit creates a new commit with the given message.
// Returns the commit hash on success.
func (r *Runner) Commit(message string) (string, error) {
	out, err := r.Exec("commit", "-m", message)
	if err != nil {
		if strings.Contains(err.Error(), "nothing to commit") {
			return "", ErrNothingToCommit
		}
		return "", err
	}

	// Parse commit hash from output like:
	//   commit abc123def456 (HEAD -> main)
	matches := commitHashRegex.FindStringSubmatch(out)
	if len(matches) < 2 {
		return "", fmt.Errorf("could not parse commit hash from: %s", out)
	}

	return matches[1], nil
}

// Push pushes the current branch to the remote.
func (r *Runner) Push() (string, error) {
	return r.Exec("push")
}

// Pull fetches and merges from the remote.
func (r *Runner) Pull() (string, error) {
	return r.Exec("pull")
}

// Fetch downloads objects and refs from the remote.
func (r *Runner) Fetch() (string, error) {
	return r.Exec("fetch")
}

// CheckoutTable restores a table to its HEAD state, discarding changes.
func (r *Runner) CheckoutTable(table string) error {
	_, err := r.Exec("checkout", table)
	return err
}

// CommitAmend amends the last commit with staged changes and a new message.
func (r *Runner) CommitAmend(message string) (string, error) {
	out, err := r.Exec("commit", "--amend", "-m", message)
	if err != nil {
		return "", err
	}
	matches := commitHashRegex.FindStringSubmatch(out)
	if len(matches) < 2 {
		return "", fmt.Errorf("could not parse commit hash from: %s", out)
	}
	return matches[1], nil
}

// Checkout switches to the given branch.
func (r *Runner) Checkout(branch string) error {
	_, err := r.Exec("checkout", branch)
	return err
}

// NewBranch creates a new branch from the current HEAD.
func (r *Runner) NewBranch(name string) error {
	_, err := r.Exec("branch", name)
	return err
}

// DeleteBranch deletes a branch.
func (r *Runner) DeleteBranch(name string) error {
	_, err := r.Exec("branch", "-d", name)
	return err
}

// Merge merges the given branch into the current branch.
func (r *Runner) Merge(branch string) (string, error) {
	return r.Exec("merge", branch)
}

// MergeSquash merges the given branch into the current branch as a squash.
func (r *Runner) MergeSquash(branch string) (string, error) {
	return r.Exec("merge", "--squash", branch)
}
