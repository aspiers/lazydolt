package dolt

import (
	"fmt"
	"sort"
	"strings"

	"github.com/aspiers/lazydolt/internal/domain"
)

// Tables returns all tables in the working set with their status info.
func (r *CLIRunner) Tables() ([]domain.Table, error) {
	// Get table names via dolt ls (avoids information_schema quirks)
	tableNames, err := r.listTableNames()
	if err != nil {
		return nil, err
	}

	// Get status for changed tables
	statusEntries, err := r.Status()
	if err != nil {
		return nil, err
	}

	// Build a lookup map for status
	statusMap := make(map[string]*domain.StatusEntry, len(statusEntries))
	for i := range statusEntries {
		statusMap[statusEntries[i].TableName] = &statusEntries[i]
	}

	// Also include tables that appear in status but not in ls
	// (e.g. new untracked tables)
	seen := make(map[string]bool, len(tableNames))
	for _, name := range tableNames {
		seen[name] = true
	}
	for _, entry := range statusEntries {
		if !seen[entry.TableName] {
			tableNames = append(tableNames, entry.TableName)
		}
	}

	sort.Strings(tableNames)

	tables := make([]domain.Table, 0, len(tableNames))
	for _, name := range tableNames {
		t := domain.Table{Name: name}
		if s, ok := statusMap[name]; ok {
			t.Status = s
		}
		tables = append(tables, t)
	}

	return tables, nil
}

// listTableNames parses 'dolt ls' output for table names.
func (r *CLIRunner) listTableNames() ([]string, error) {
	out, err := r.Exec("ls")
	if err != nil {
		return nil, fmt.Errorf("listing tables: %w", err)
	}

	var names []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		// Skip header and empty lines
		if line == "" || strings.HasPrefix(line, "Tables in") || strings.HasPrefix(line, "No tables") {
			continue
		}
		names = append(names, line)
	}

	return names, nil
}
