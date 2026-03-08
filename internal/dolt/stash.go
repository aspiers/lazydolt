package dolt

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/aspiers/lazydolt/internal/domain"
)

// stashRegex parses lines like:
// stash@{0}: WIP on refs/heads/main: abc1234 Commit message
var stashRegex = regexp.MustCompile(`^stash@\{(\d+)\}: WIP on refs/heads/(\S+): (\S+) (.+)$`)

// Stash saves the current working set to the stash.
func (r *Runner) Stash() error {
	_, err := r.Exec("stash")
	return err
}

// StashList returns all stash entries.
func (r *Runner) StashList() ([]domain.StashEntry, error) {
	out, err := r.Exec("stash", "list")
	if err != nil {
		return nil, fmt.Errorf("dolt stash list: %w", err)
	}

	out = strings.TrimSpace(out)
	if out == "" {
		return nil, nil
	}

	var entries []domain.StashEntry
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		m := stashRegex.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		idx, _ := strconv.Atoi(m[1])
		entries = append(entries, domain.StashEntry{
			Index:   idx,
			Branch:  m[2],
			Hash:    m[3],
			Message: m[4],
		})
	}
	return entries, nil
}

// StashPop applies and removes a stash entry.
func (r *Runner) StashPop(index int) error {
	ref := fmt.Sprintf("stash@{%d}", index)
	_, err := r.Exec("stash", "pop", ref)
	return err
}

// StashDrop removes a stash entry without applying it.
func (r *Runner) StashDrop(index int) error {
	ref := fmt.Sprintf("stash@{%d}", index)
	_, err := r.Exec("stash", "drop", ref)
	return err
}
