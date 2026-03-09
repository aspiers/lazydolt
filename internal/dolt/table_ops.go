package dolt

// TableRename renames a table.
func (r *Runner) TableRename(oldName, newName string) error {
	_, err := r.Exec("table", "mv", oldName, newName)
	return err
}

// TableCopy copies a table to a new name.
func (r *Runner) TableCopy(srcName, dstName string) error {
	_, err := r.Exec("table", "cp", srcName, dstName)
	return err
}

// TableDrop removes a table from the working set.
func (r *Runner) TableDrop(name string) error {
	_, err := r.Exec("table", "rm", name)
	return err
}

// TableExport exports a table's contents to a file.
func (r *Runner) TableExport(table, filePath string) error {
	_, err := r.Exec("table", "export", "-f", table, filePath)
	return err
}
