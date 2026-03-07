package dolt

import (
	"fmt"
	"strings"

	"github.com/aspiers/lazydolt/internal/domain"
)

// Schema returns the CREATE TABLE statement for a table.
func (r *Runner) Schema(table string) (domain.Schema, error) {
	out, err := r.Exec("schema", "show", table)
	if err != nil {
		return domain.Schema{}, fmt.Errorf("schema for %q: %w", table, err)
	}

	// Output format:
	//   <table> @ working
	//   CREATE TABLE `<table>` (
	//     ...
	//   );
	// Skip the first line (header).
	lines := strings.SplitN(out, "\n", 2)
	createStmt := ""
	if len(lines) > 1 {
		createStmt = strings.TrimSpace(lines[1])
	}

	return domain.Schema{
		TableName:       table,
		CreateStatement: createStmt,
	}, nil
}
