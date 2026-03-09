package dolt

import "fmt"

// QueryDiff runs dolt query-diff to compare the results of two SQL queries.
// It returns the tabular diff output showing added, deleted, and modified rows.
// An empty result means the two queries produce identical output.
func (r *Runner) QueryDiff(query1, query2 string) (string, error) {
	out, err := r.Exec("query-diff", query1, query2)
	if err != nil {
		return "", fmt.Errorf("query-diff: %w", err)
	}
	return out, nil
}
