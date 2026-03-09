package dolt

// Reflog returns the reflog output showing the history of named refs.
func (r *Runner) Reflog() (string, error) {
	return r.Exec("reflog")
}
