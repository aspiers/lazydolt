package dolt

import (
	"fmt"

	"github.com/aspiers/lazydolt/internal/domain"
)

// Log returns the commit history, most recent first.
// If limit is 0, all commits are returned.
func (r *Runner) Log(limit int) ([]domain.Commit, error) {
	query := "SELECT commit_hash, committer, email, date, message FROM dolt_log ORDER BY date DESC"
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

		commits = append(commits, domain.Commit{
			Hash:    hash,
			Message: msg,
			Author:  author,
			Email:   email,
			Date:    parseDoltTime(dateStr),
		})
	}

	return commits, nil
}
