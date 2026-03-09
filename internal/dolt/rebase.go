package dolt

import (
	"fmt"
	"os/exec"
	"strings"
)

// Rebase rebases the current branch onto the given upstream branch.
// Returns ErrMergeConflict if the rebase results in conflicts.
func (r *Runner) Rebase(upstream string) (string, error) {
	cmd := exec.Command(r.DoltPath, "rebase", upstream)
	cmd.Dir = r.RepoDir
	out, err := cmd.CombinedOutput()
	text := stripANSI(string(out))
	cmdStr := "dolt rebase " + upstream

	if err != nil && strings.Contains(text, "CONFLICT") {
		r.logCommand(cmdStr, text, true)
		return text, ErrMergeConflict
	}
	if err != nil {
		errText := strings.TrimSpace(text)
		r.logCommand(cmdStr, errText, true)
		return "", fmt.Errorf("%s: %s", cmdStr, errText)
	}
	r.logCommand(cmdStr, text, false)
	return text, nil
}

// RebaseAbort aborts an in-progress rebase.
func (r *Runner) RebaseAbort() error {
	_, err := r.Exec("rebase", "--abort")
	return err
}

// RebaseContinue continues a rebase after conflict resolution.
func (r *Runner) RebaseContinue() (string, error) {
	cmd := exec.Command(r.DoltPath, "rebase", "--continue")
	cmd.Dir = r.RepoDir
	out, err := cmd.CombinedOutput()
	text := stripANSI(string(out))
	cmdStr := "dolt rebase --continue"

	if err != nil && strings.Contains(text, "CONFLICT") {
		r.logCommand(cmdStr, text, true)
		return text, ErrMergeConflict
	}
	if err != nil {
		errText := strings.TrimSpace(text)
		r.logCommand(cmdStr, errText, true)
		return "", fmt.Errorf("%s: %s", cmdStr, errText)
	}
	r.logCommand(cmdStr, text, false)
	return text, nil
}
