package dolt

// DiffText returns the human-readable diff output for a table.
// If table is empty, diffs all changed tables.
func (r *Runner) DiffText(table string) (string, error) {
	args := []string{"diff"}
	if table != "" {
		args = append(args, table)
	}

	out, err := r.Exec(args...)
	if err != nil {
		return "", err
	}

	return out, nil
}

// DiffStat returns diff statistics for a table.
// If table is empty, returns stats for all changed tables.
func (r *Runner) DiffStat(table string) (string, error) {
	args := []string{"diff", "--stat"}
	if table != "" {
		args = append(args, table)
	}

	out, err := r.Exec(args...)
	if err != nil {
		return "", err
	}

	return out, nil
}
