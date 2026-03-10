package dolt

import (
	"fmt"
	"strings"

	"github.com/aspiers/lazydolt/internal/domain"
)

// Add overrides CLIRunner.Add to use CALL dolt_add() via sql-server.
func (r *SQLRunner) Add(table string) error {
	_, err := r.db.Exec("CALL dolt_add(?)", table)
	if err != nil {
		r.logCommand("dolt add "+table, err.Error(), true)
		return fmt.Errorf("dolt add %s: %w", table, err)
	}
	r.logCommand("dolt add "+table, "", false)
	return nil
}

// AddAll overrides CLIRunner.AddAll to use CALL dolt_add('.') via sql-server.
func (r *SQLRunner) AddAll() error {
	_, err := r.db.Exec("CALL dolt_add('.')")
	if err != nil {
		r.logCommand("dolt add .", err.Error(), true)
		return fmt.Errorf("dolt add .: %w", err)
	}
	r.logCommand("dolt add .", "", false)
	return nil
}

// Reset overrides CLIRunner.Reset to use CALL dolt_reset() via sql-server.
func (r *SQLRunner) Reset(table string) error {
	_, err := r.db.Exec("CALL dolt_reset(?)", table)
	if err != nil {
		r.logCommand("dolt reset "+table, err.Error(), true)
		return fmt.Errorf("dolt reset %s: %w", table, err)
	}
	r.logCommand("dolt reset "+table, "", false)
	return nil
}

// ResetAll overrides CLIRunner.ResetAll to use CALL dolt_reset() via sql-server.
func (r *SQLRunner) ResetAll() error {
	_, err := r.db.Exec("CALL dolt_reset()")
	if err != nil {
		r.logCommand("dolt reset", err.Error(), true)
		return fmt.Errorf("dolt reset: %w", err)
	}
	r.logCommand("dolt reset", "", false)
	return nil
}

// ResetSoft overrides CLIRunner.ResetSoft via sql-server.
func (r *SQLRunner) ResetSoft(commit string) error {
	_, err := r.db.Exec("CALL dolt_reset('--soft', ?)", commit)
	if err != nil {
		r.logCommand("dolt reset --soft "+commit, err.Error(), true)
		return fmt.Errorf("dolt reset --soft %s: %w", commit, err)
	}
	r.logCommand("dolt reset --soft "+commit, "", false)
	return nil
}

// ResetHard overrides CLIRunner.ResetHard via sql-server.
func (r *SQLRunner) ResetHard(commit string) error {
	_, err := r.db.Exec("CALL dolt_reset('--hard', ?)", commit)
	if err != nil {
		r.logCommand("dolt reset --hard "+commit, err.Error(), true)
		return fmt.Errorf("dolt reset --hard %s: %w", commit, err)
	}
	r.logCommand("dolt reset --hard "+commit, "", false)
	return nil
}

// Commit overrides CLIRunner.Commit to use CALL dolt_commit() via sql-server.
// Returns the new commit hash.
func (r *SQLRunner) Commit(message string) (string, error) {
	var hash string
	err := r.db.QueryRow("CALL dolt_commit('-m', ?)", message).Scan(&hash)
	if err != nil {
		errStr := err.Error()
		r.logCommand("dolt commit -m ...", errStr, true)
		if strings.Contains(errStr, "nothing to commit") {
			return "", ErrNothingToCommit
		}
		return "", fmt.Errorf("dolt commit: %w", err)
	}
	r.logCommand("dolt commit -m ...", "commit "+hash, false)
	return hash, nil
}

// CommitAmend overrides CLIRunner.CommitAmend via sql-server.
func (r *SQLRunner) CommitAmend(message string) (string, error) {
	var hash string
	err := r.db.QueryRow("CALL dolt_commit('--amend', '-m', ?)", message).Scan(&hash)
	if err != nil {
		r.logCommand("dolt commit --amend -m ...", err.Error(), true)
		return "", fmt.Errorf("dolt commit --amend: %w", err)
	}
	r.logCommand("dolt commit --amend -m ...", "commit "+hash, false)
	return hash, nil
}

// Checkout overrides CLIRunner.Checkout to use CALL dolt_checkout() via sql-server.
func (r *SQLRunner) Checkout(branch string) error {
	_, err := r.db.Exec("CALL dolt_checkout(?)", branch)
	if err != nil {
		r.logCommand("dolt checkout "+branch, err.Error(), true)
		return fmt.Errorf("dolt checkout %s: %w", branch, err)
	}
	r.logCommand("dolt checkout "+branch, "", false)
	return nil
}

// CheckoutTable overrides CLIRunner.CheckoutTable via sql-server.
func (r *SQLRunner) CheckoutTable(table string) error {
	_, err := r.db.Exec("CALL dolt_checkout(?)", table)
	if err != nil {
		r.logCommand("dolt checkout "+table, err.Error(), true)
		return fmt.Errorf("dolt checkout %s: %w", table, err)
	}
	r.logCommand("dolt checkout "+table, "", false)
	return nil
}

// NewBranch overrides CLIRunner.NewBranch via sql-server.
func (r *SQLRunner) NewBranch(name string) error {
	_, err := r.db.Exec("CALL dolt_branch(?)", name)
	if err != nil {
		r.logCommand("dolt branch "+name, err.Error(), true)
		return fmt.Errorf("dolt branch %s: %w", name, err)
	}
	r.logCommand("dolt branch "+name, "", false)
	return nil
}

// RenameBranch overrides CLIRunner.RenameBranch via sql-server.
func (r *SQLRunner) RenameBranch(oldName, newName string) error {
	_, err := r.db.Exec("CALL dolt_branch('-m', ?, ?)", oldName, newName)
	if err != nil {
		r.logCommand("dolt branch -m "+oldName+" "+newName, err.Error(), true)
		return fmt.Errorf("dolt branch -m %s %s: %w", oldName, newName, err)
	}
	r.logCommand("dolt branch -m "+oldName+" "+newName, "", false)
	return nil
}

// DeleteBranch overrides CLIRunner.DeleteBranch via sql-server.
func (r *SQLRunner) DeleteBranch(name string) error {
	_, err := r.db.Exec("CALL dolt_branch('-d', ?)", name)
	if err != nil {
		r.logCommand("dolt branch -d "+name, err.Error(), true)
		return fmt.Errorf("dolt branch -d %s: %w", name, err)
	}
	r.logCommand("dolt branch -d "+name, "", false)
	return nil
}

// Merge overrides CLIRunner.Merge to use CALL dolt_merge() via sql-server.
// Returns ErrMergeConflict if the merge results in conflicts.
func (r *SQLRunner) Merge(branch string) (string, error) {
	var hash string
	var fastForward, conflicts int
	err := r.db.QueryRow("CALL dolt_merge(?)", branch).Scan(&hash, &fastForward, &conflicts)
	if err != nil {
		errStr := err.Error()
		r.logCommand("dolt merge "+branch, errStr, true)
		if strings.Contains(errStr, "conflict") || strings.Contains(errStr, "CONFLICT") {
			return errStr, ErrMergeConflict
		}
		return "", fmt.Errorf("dolt merge %s: %w", branch, err)
	}
	if conflicts > 0 {
		r.logCommand("dolt merge "+branch, "CONFLICT", true)
		return fmt.Sprintf("merge %s: %d conflicts", branch, conflicts), ErrMergeConflict
	}
	r.logCommand("dolt merge "+branch, hash, false)
	return hash, nil
}

// MergeSquash overrides CLIRunner.MergeSquash via sql-server.
func (r *SQLRunner) MergeSquash(branch string) (string, error) {
	_, err := r.db.Exec("CALL dolt_merge('--squash', ?)", branch)
	if err != nil {
		r.logCommand("dolt merge --squash "+branch, err.Error(), true)
		return "", fmt.Errorf("dolt merge --squash %s: %w", branch, err)
	}
	r.logCommand("dolt merge --squash "+branch, "", false)
	return "", nil
}

// MergeAbort overrides CLIRunner.MergeAbort via sql-server.
func (r *SQLRunner) MergeAbort() error {
	_, err := r.db.Exec("CALL dolt_merge('--abort')")
	if err != nil {
		r.logCommand("dolt merge --abort", err.Error(), true)
		return fmt.Errorf("dolt merge --abort: %w", err)
	}
	r.logCommand("dolt merge --abort", "", false)
	return nil
}

// CherryPick overrides CLIRunner.CherryPick via sql-server.
func (r *SQLRunner) CherryPick(hash string) (string, error) {
	var newHash string
	var dataConflicts, schemaConflicts int
	err := r.db.QueryRow("CALL dolt_cherry_pick(?)", hash).Scan(&newHash, &dataConflicts, &schemaConflicts)
	if err != nil {
		errStr := err.Error()
		r.logCommand("dolt cherry-pick "+hash, errStr, true)
		if strings.Contains(errStr, "conflict") || strings.Contains(errStr, "CONFLICT") {
			return errStr, ErrMergeConflict
		}
		return "", fmt.Errorf("dolt cherry-pick %s: %w", hash, err)
	}
	if dataConflicts > 0 || schemaConflicts > 0 {
		r.logCommand("dolt cherry-pick "+hash, "CONFLICT", true)
		return fmt.Sprintf("cherry-pick %s: conflicts", hash), ErrMergeConflict
	}
	r.logCommand("dolt cherry-pick "+hash, newHash, false)
	return newHash, nil
}

// CherryPickAbort overrides CLIRunner.CherryPickAbort via sql-server.
func (r *SQLRunner) CherryPickAbort() error {
	_, err := r.db.Exec("CALL dolt_cherry_pick('--abort')")
	if err != nil {
		r.logCommand("dolt cherry-pick --abort", err.Error(), true)
		return fmt.Errorf("dolt cherry-pick --abort: %w", err)
	}
	r.logCommand("dolt cherry-pick --abort", "", false)
	return nil
}

// Revert overrides CLIRunner.Revert via sql-server.
func (r *SQLRunner) Revert(hash string) (string, error) {
	_, err := r.db.Exec("CALL dolt_revert(?)", hash)
	if err != nil {
		r.logCommand("dolt revert "+hash, err.Error(), true)
		return "", fmt.Errorf("dolt revert %s: %w", hash, err)
	}
	r.logCommand("dolt revert "+hash, "", false)
	return "", nil
}

// Push overrides CLIRunner.Push via sql-server.
func (r *SQLRunner) Push() (string, error) {
	_, err := r.db.Exec("CALL dolt_push()")
	if err != nil {
		r.logCommand("dolt push", err.Error(), true)
		return "", fmt.Errorf("dolt push: %w", err)
	}
	r.logCommand("dolt push", "", false)
	return "", nil
}

// Pull overrides CLIRunner.Pull via sql-server.
func (r *SQLRunner) Pull() (string, error) {
	_, err := r.db.Exec("CALL dolt_pull()")
	if err != nil {
		r.logCommand("dolt pull", err.Error(), true)
		return "", fmt.Errorf("dolt pull: %w", err)
	}
	r.logCommand("dolt pull", "", false)
	return "", nil
}

// Fetch overrides CLIRunner.Fetch via sql-server.
func (r *SQLRunner) Fetch() (string, error) {
	_, err := r.db.Exec("CALL dolt_fetch()")
	if err != nil {
		r.logCommand("dolt fetch", err.Error(), true)
		return "", fmt.Errorf("dolt fetch: %w", err)
	}
	r.logCommand("dolt fetch", "", false)
	return "", nil
}

// CreateTag overrides CLIRunner.CreateTag via sql-server.
func (r *SQLRunner) CreateTag(name, ref, message string) error {
	var err error
	if message != "" {
		_, err = r.db.Exec("CALL dolt_tag(?, ?, '-m', ?)", name, ref, message)
	} else if ref != "" {
		_, err = r.db.Exec("CALL dolt_tag(?, ?)", name, ref)
	} else {
		_, err = r.db.Exec("CALL dolt_tag(?)", name)
	}
	if err != nil {
		r.logCommand("dolt tag "+name, err.Error(), true)
		return fmt.Errorf("dolt tag %s: %w", name, err)
	}
	r.logCommand("dolt tag "+name, "", false)
	return nil
}

// DeleteTag overrides CLIRunner.DeleteTag via sql-server.
func (r *SQLRunner) DeleteTag(name string) error {
	_, err := r.db.Exec("CALL dolt_tag('-d', ?)", name)
	if err != nil {
		r.logCommand("dolt tag -d "+name, err.Error(), true)
		return fmt.Errorf("dolt tag -d %s: %w", name, err)
	}
	r.logCommand("dolt tag -d "+name, "", false)
	return nil
}

// TableRename overrides CLIRunner.TableRename via sql-server.
func (r *SQLRunner) TableRename(oldName, newName string) error {
	_, err := r.db.Exec(fmt.Sprintf("ALTER TABLE `%s` RENAME TO `%s`", oldName, newName))
	if err != nil {
		r.logCommand("dolt table mv "+oldName+" "+newName, err.Error(), true)
		return fmt.Errorf("rename table %s to %s: %w", oldName, newName, err)
	}
	r.logCommand("dolt table mv "+oldName+" "+newName, "", false)
	return nil
}

// TableDrop overrides CLIRunner.TableDrop via sql-server.
func (r *SQLRunner) TableDrop(name string) error {
	_, err := r.db.Exec(fmt.Sprintf("DROP TABLE `%s`", name))
	if err != nil {
		r.logCommand("dolt table rm "+name, err.Error(), true)
		return fmt.Errorf("drop table %s: %w", name, err)
	}
	r.logCommand("dolt table rm "+name, "", false)
	return nil
}

// Schema overrides CLIRunner.Schema via sql-server.
func (r *SQLRunner) Schema(table string) (domain.Schema, error) {
	var tableName, createStmt string
	err := r.db.QueryRow(fmt.Sprintf("SHOW CREATE TABLE `%s`", table)).Scan(&tableName, &createStmt)
	if err != nil {
		r.logCommand("dolt schema show "+table, err.Error(), true)
		return domain.Schema{}, fmt.Errorf("schema show %s: %w", table, err)
	}
	r.logCommand("dolt schema show "+table, createStmt, false)
	return domain.Schema{
		TableName:       tableName,
		CreateStatement: createStmt,
	}, nil
}
