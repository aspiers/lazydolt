package dolt

import (
	"fmt"
	"strings"

	"github.com/aspiers/lazydolt/internal/domain"
)

// Remotes returns the list of configured remotes.
func (r *CLIRunner) Remotes() ([]domain.Remote, error) {
	out, err := r.Exec("remote", "-v")
	if err != nil {
		return nil, fmt.Errorf("listing remotes: %w", err)
	}

	out = strings.TrimSpace(out)
	if out == "" {
		return nil, nil
	}

	var remotes []domain.Remote
	seen := make(map[string]bool)
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Format: "<name> <url> "
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		name := parts[0]
		url := parts[1]
		// dolt remote -v may list each remote once (unlike git which
		// shows fetch/push separately), but deduplicate just in case.
		if seen[name] {
			continue
		}
		seen[name] = true
		remotes = append(remotes, domain.Remote{Name: name, URL: url})
	}
	return remotes, nil
}

// RemoteAdd adds a new remote with the given name and URL.
func (r *CLIRunner) RemoteAdd(name, url string) error {
	_, err := r.Exec("remote", "add", name, url)
	return err
}

// RemoteRemove removes a remote by name.
func (r *CLIRunner) RemoteRemove(name string) error {
	_, err := r.Exec("remote", "remove", name)
	return err
}
