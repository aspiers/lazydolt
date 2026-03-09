package dolt

import (
	"fmt"
	"strings"

	"github.com/aspiers/lazydolt/internal/domain"
)

// Log returns the commit history, most recent first.
// If limit is 0, all commits are returned.
// Each commit includes its parent hashes from dolt_commit_ancestors.
func (r *Runner) Log(limit int) ([]domain.Commit, error) {
	query := `SELECT l.commit_hash, l.committer, l.email, l.date, l.message,
       GROUP_CONCAT(a.parent_hash ORDER BY a.parent_index SEPARATOR ',') as parents
FROM dolt_log l
LEFT JOIN dolt_commit_ancestors a ON l.commit_hash = a.commit_hash
GROUP BY l.commit_hash, l.committer, l.email, l.date, l.message
ORDER BY l.date DESC`
	if limit > 0 {
		query = fmt.Sprintf("%s LIMIT %d", query, limit)
	}

	rows, err := r.SQL(query)
	if err != nil {
		// New repo with no commits: return empty
		return nil, nil
	}

	commits := make([]domain.Commit, 0, len(rows))
	for _, row := range rows {
		hash, _ := row["commit_hash"].(string)
		author, _ := row["committer"].(string)
		email, _ := row["email"].(string)
		msg, _ := row["message"].(string)
		dateStr, _ := row["date"].(string)
		parentsStr, _ := row["parents"].(string)

		var parents []string
		if parentsStr != "" {
			parents = strings.Split(parentsStr, ",")
		}

		commits = append(commits, domain.Commit{
			Hash:    hash,
			Message: msg,
			Author:  author,
			Email:   email,
			Date:    parseDoltTime(dateStr),
			Parents: parents,
		})
	}

	return commits, nil
}
