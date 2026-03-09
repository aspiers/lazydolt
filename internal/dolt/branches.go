package dolt

import (
	"fmt"

	"github.com/aspiers/lazydolt/internal/domain"
)

// BranchOrderBy specifies the ORDER BY clause for branch queries.
type BranchOrderBy string

const (
	BranchOrderByDate   BranchOrderBy = "latest_commit_date DESC"
	BranchOrderByName   BranchOrderBy = "name ASC"
	BranchOrderByAuthor BranchOrderBy = "latest_committer ASC, latest_commit_date DESC"
)

// Branches returns all branches sorted by the given order.
// If order is empty, it defaults to BranchOrderByDate.
func (r *Runner) Branches(order BranchOrderBy) ([]domain.Branch, error) {
	if order == "" {
		order = BranchOrderByDate
	}
	rows, err := r.SQL(fmt.Sprintf(`SELECT name, hash, latest_commit_message, latest_committer,
		latest_committer_email, latest_commit_date
		FROM dolt_branches
		ORDER BY %s`, order))
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
