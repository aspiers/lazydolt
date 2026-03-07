package dolt

import (
	"fmt"

	"github.com/aspiers/lazydolt/internal/domain"
)

// Branches returns all branches sorted by latest commit date (most recent first).
func (r *Runner) Branches() ([]domain.Branch, error) {
	rows, err := r.SQL(`SELECT name, hash, latest_commit_message, latest_committer,
		latest_committer_email, latest_commit_date
		FROM dolt_branches
		ORDER BY latest_commit_date DESC`)
	if err != nil {
		return nil, fmt.Errorf("querying dolt_branches: %w", err)
	}

	currentBranch, err := r.CurrentBranch()
	if err != nil {
		return nil, err
	}

	branches := make([]domain.Branch, 0, len(rows))
	for _, row := range rows {
		name, _ := row["name"].(string)
		hash, _ := row["hash"].(string)
		msg, _ := row["latest_commit_message"].(string)
		author, _ := row["latest_committer"].(string)
		dateStr, _ := row["latest_commit_date"].(string)

		date := parseDoltTime(dateStr)

		branches = append(branches, domain.Branch{
			Name:          name,
			Hash:          hash,
			IsCurrent:     name == currentBranch,
			LatestMessage: msg,
			LatestAuthor:  author,
			LatestDate:    date,
		})
	}

	return branches, nil
}
