package dolt

import (
	"fmt"

	"github.com/aspiers/lazydolt/internal/domain"
)

// Conflicts returns the list of tables with merge conflicts and their counts.
func (r *Runner) Conflicts() ([]domain.ConflictSummary, error) {
	rows, err := r.SQL("SELECT * FROM dolt_conflicts")
	if err != nil {
		return nil, fmt.Errorf("querying dolt_conflicts: %w", err)
	}

	summaries := make([]domain.ConflictSummary, 0, len(rows))
	for _, row := range rows {
		tableName, _ := row["table"].(string)
		var numConflicts int
		switch v := row["num_conflicts"].(type) {
		case float64:
			numConflicts = int(v)
		}
		summaries = append(summaries, domain.ConflictSummary{
			TableName:    tableName,
			NumConflicts: numConflicts,
		})
	}
	return summaries, nil
}

// ConflictsCat returns the raw text output from 'dolt conflicts cat <table>'.
func (r *Runner) ConflictsCat(table string) (string, error) {
	out, err := r.Exec("conflicts", "cat", table)
	if err != nil {
		return "", fmt.Errorf("dolt conflicts cat %s: %w", table, err)
	}
	return out, nil
}

// ConflictsResolveOurs resolves all conflicts in a table by taking our version.
func (r *Runner) ConflictsResolveOurs(table string) error {
	_, err := r.Exec("conflicts", "resolve", "--ours", table)
	return err
}

// ConflictsResolveTheirs resolves all conflicts in a table by taking their version.
func (r *Runner) ConflictsResolveTheirs(table string) error {
	_, err := r.Exec("conflicts", "resolve", "--theirs", table)
	return err
}

// MergeAbort aborts an in-progress merge.
func (r *Runner) MergeAbort() error {
	_, err := r.Exec("merge", "--abort")
	return err
}
