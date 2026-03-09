package dolt

import (
	"fmt"

	"github.com/aspiers/lazydolt/internal/domain"
)

// Tags returns the list of tags in the repository.
func (r *Runner) Tags() ([]domain.Tag, error) {
	rows, err := r.SQL("SELECT tag_name, tag_hash, tagger, email, date, message FROM dolt_tags ORDER BY date DESC")
	if err != nil {
		return nil, fmt.Errorf("querying dolt_tags: %w", err)
	}

	tags := make([]domain.Tag, 0, len(rows))
	for _, row := range rows {
		name, _ := row["tag_name"].(string)
		hash, _ := row["tag_hash"].(string)
		tagger, _ := row["tagger"].(string)
		email, _ := row["email"].(string)
		message, _ := row["message"].(string)
		dateStr, _ := row["date"].(string)

		t := domain.Tag{
			Name:    name,
			Hash:    hash,
			Tagger:  tagger,
			Email:   email,
			Message: message,
			Date:    parseDoltTime(dateStr),
		}
		tags = append(tags, t)
	}
	return tags, nil
}

// CreateTag creates a new tag pointing at the given ref.
// If ref is empty, it points at HEAD.
func (r *Runner) CreateTag(name, ref, message string) error {
	args := []string{"tag", name}
	if ref != "" {
		args = append(args, ref)
	}
	if message != "" {
		args = append(args, "-m", message)
	}
	_, err := r.Exec(args...)
	return err
}

// DeleteTag deletes a tag.
func (r *Runner) DeleteTag(name string) error {
	_, err := r.Exec("tag", "-d", name)
	return err
}
