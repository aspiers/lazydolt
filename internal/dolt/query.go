package dolt

import (
	"fmt"
	"sort"

	"github.com/aspiers/lazydolt/internal/domain"
)

// Query runs an arbitrary SQL query and returns structured results.
func (r *Runner) Query(sql string) (*domain.QueryResult, error) {
	rows, err := r.SQL(sql)
	if err != nil {
		return nil, err
	}

	result := &domain.QueryResult{
		Rows: rows,
	}

	// Extract column names from the first row (sorted for consistency)
	if len(rows) > 0 {
		cols := make([]string, 0, len(rows[0]))
		for k := range rows[0] {
			cols = append(cols, k)
		}
		sort.Strings(cols)
		result.Columns = cols
	}

	return result, nil
}

// QueryPage runs a paginated SELECT and includes total row count.
func (r *Runner) QueryPage(table string, limit, offset int) (*domain.QueryResult, error) {
	sql := fmt.Sprintf("SELECT * FROM `%s` LIMIT %d OFFSET %d", table, limit, offset)
	result, err := r.Query(sql)
	if err != nil {
		return nil, err
	}

	// Get total count for pagination
	countRows, err := r.SQL(fmt.Sprintf("SELECT COUNT(*) as count FROM `%s`", table))
	if err != nil {
		return nil, fmt.Errorf("counting rows in %q: %w", table, err)
	}
	if len(countRows) > 0 {
		if v, ok := countRows[0]["count"].(float64); ok {
			result.Total = int(v)
		}
	}

	return result, nil
}
