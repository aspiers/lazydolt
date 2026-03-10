package dolt

// DiffText returns the human-readable diff output for a table.
// If table is empty, diffs all changed tables.
// If staged is true, shows the diff between HEAD and the staging area
// (i.e. what would be committed); otherwise shows unstaged changes.
func (r *CLIRunner) DiffText(table string, staged bool) (string, error) {
	args := []string{"diff"}
	if staged {
		args = append(args, "--staged")
	}
	if table != "" {
		args = append(args, table)
	}

	out, err := r.Exec(args...)
	if err != nil {
		return "", err
	}

	return out, nil
}

// DiffSchema returns the schema-only diff for a table.
// If table is empty, diffs schemas for all changed tables.
// If staged is true, shows staged schema changes.
func (r *CLIRunner) DiffSchema(table string, staged bool) (string, error) {
	args := []string{"diff", "--schema"}
	if staged {
		args = append(args, "--staged")
	}
	if table != "" {
		args = append(args, table)
	}

	out, err := r.Exec(args...)
	if err != nil {
		return "", err
	}

	return out, nil
}

// DiffSchemaRefs returns the schema-only diff between two refs for a table.
func (r *CLIRunner) DiffSchemaRefs(fromRef, toRef, table string) (string, error) {
	args := []string{"diff", "--schema", fromRef, toRef}
	if table != "" {
		args = append(args, table)
	}

	out, err := r.Exec(args...)
	if err != nil {
		return "", err
	}

	return out, nil
}

// DiffRefs returns the text diff between two refs (branches, commits, etc.).
// If table is empty, diffs all tables.
func (r *CLIRunner) DiffRefs(fromRef, toRef, table string) (string, error) {
	args := []string{"diff", fromRef, toRef}
	if table != "" {
		args = append(args, table)
	}

	out, err := r.Exec(args...)
	if err != nil {
		return "", err
	}

	return out, nil
}

// DiffStatRefs returns diff statistics between two refs.
// If table is empty, returns stats for all changed tables.
func (r *CLIRunner) DiffStatRefs(fromRef, toRef, table string) (string, error) {
	args := []string{"diff", "--stat", fromRef, toRef}
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
// If staged is true, shows stats for staged changes.
func (r *CLIRunner) DiffStat(table string, staged bool) (string, error) {
	args := []string{"diff", "--stat"}
	if staged {
		args = append(args, "--staged")
	}
	if table != "" {
		args = append(args, table)
	}

	out, err := r.Exec(args...)
	if err != nil {
		return "", err
	}

	return out, nil
}
