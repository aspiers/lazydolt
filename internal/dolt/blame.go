package dolt

// Blame returns the blame output for a table, showing which commit
// last modified each row.
func (r *Runner) Blame(table string) (string, error) {
	return r.Exec("blame", table)
}
