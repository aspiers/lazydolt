package dolt

import (
	"regexp"
	"strings"

	"github.com/aspiers/lazydolt/internal/domain"
)

// reflogRegex parses one line of dolt reflog output (after ANSI stripping).
// Format: "<hash> (<refs>) <message>"
var reflogRegex = regexp.MustCompile(`^(\S+)\s+\(([^)]+)\)\s+(.*)$`)

// Reflog returns the raw reflog output showing the history of named refs.
func (r *CLIRunner) Reflog() (string, error) {
	return r.Exec("reflog")
}

// ReflogEntries returns the reflog as structured entries.
func (r *CLIRunner) ReflogEntries() ([]domain.ReflogEntry, error) {
	raw, err := r.Reflog()
	if err != nil {
		return nil, err
	}
	return ParseReflog(raw), nil
}

// HeadHash returns the current HEAD commit hash via dolt_hashof('HEAD').
func (r *CLIRunner) HeadHash() (string, error) {
	rows, err := r.SQL("SELECT dolt_hashof('HEAD') AS hash")
	if err != nil {
		return "", err
	}
	if len(rows) == 0 {
		return "", nil
	}
	hash, _ := rows[0]["hash"].(string)
	return hash, nil
}

// ParseReflog parses raw reflog text into structured entries.
func ParseReflog(raw string) []domain.ReflogEntry {
	var entries []domain.ReflogEntry
	for _, line := range strings.Split(strings.TrimSpace(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		m := reflogRegex.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		entries = append(entries, domain.ReflogEntry{
			Hash:    m[1],
			Ref:     m[2],
			Message: m[3],
		})
	}
	return entries
}
