package dolt

import (
	"fmt"

	"github.com/aspiers/lazydolt/internal/domain"
)

// Status returns the working set status (staged/unstaged table changes).
func (r *CLIRunner) Status() ([]domain.StatusEntry, error) {
	rows, err := r.SQL("SELECT * FROM dolt_status")
	if err != nil {
		return nil, fmt.Errorf("querying dolt_status: %w", err)
	}

	entries := make([]domain.StatusEntry, 0, len(rows))
	for _, row := range rows {
		tableName, _ := row["table_name"].(string)
		status, _ := row["status"].(string)

		// dolt returns staged as a number (0 or 1)
		var staged bool
		switch v := row["staged"].(type) {
		case float64:
			staged = v == 1
		case bool:
			staged = v
		}

		entries = append(entries, domain.StatusEntry{
			TableName: tableName,
			Staged:    staged,
			Status:    status,
		})
	}

	return entries, nil
}

// CurrentBranch returns the name of the currently checked-out branch.
func (r *CLIRunner) CurrentBranch() (string, error) {
	rows, err := r.SQL("SELECT active_branch() as branch")
	if err != nil {
		return "", fmt.Errorf("querying active_branch(): %w", err)
	}
	if len(rows) == 0 {
		return "", fmt.Errorf("no result from active_branch()")
	}

	branch, _ := rows[0]["branch"].(string)
	return branch, nil
}

// IsDirty returns true if the working set has uncommitted changes.
func (r *CLIRunner) IsDirty() (bool, error) {
	entries, err := r.Status()
	if err != nil {
		return false, err
	}
	return len(entries) > 0, nil
}
