package dolt

import "fmt"

// DumpFormat specifies the output format for database dumps.
type DumpFormat string

const (
	DumpFormatSQL     DumpFormat = "sql"
	DumpFormatCSV     DumpFormat = "csv"
	DumpFormatJSON    DumpFormat = "json"
	DumpFormatParquet DumpFormat = "parquet"
)

// Dump exports all tables in the working set using dolt dump.
// The format controls the output type (sql, csv, json, parquet).
// The filename is the output file (for SQL) or directory (for CSV/JSON/Parquet).
// Force overwrites existing files.
func (r *CLIRunner) Dump(format DumpFormat, filename string, force bool) error {
	args := []string{"dump", "-r", string(format)}
	if force {
		args = append(args, "-f")
	}
	if format == DumpFormatSQL {
		args = append(args, "-fn", filename)
	} else {
		args = append(args, "-d", filename)
	}
	_, err := r.Exec(args...)
	if err != nil {
		return fmt.Errorf("dump (%s): %w", format, err)
	}
	return nil
}
