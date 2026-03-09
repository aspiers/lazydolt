package dolt

import (
	"fmt"
	"strings"
)

// ConfigEntry represents a single config key-value pair.
type ConfigEntry struct {
	Key   string
	Value string
}

// Config returns the merged global and local dolt configuration.
// Global entries appear first, then local entries.
func (r *Runner) Config() (global []ConfigEntry, local []ConfigEntry, err error) {
	globalOut, gerr := r.Exec("config", "--global", "--list")
	if gerr != nil {
		// No global config is not an error
		globalOut = ""
	}
	localOut, lerr := r.Exec("config", "--local", "--list")
	if lerr != nil {
		// No local config is not an error
		localOut = ""
	}

	global = parseConfigOutput(globalOut)
	local = parseConfigOutput(localOut)
	return global, local, nil
}

// ConfigSet sets a config value at the specified scope.
func (r *Runner) ConfigSet(global bool, key, value string) error {
	args := []string{"config"}
	if global {
		args = append(args, "--global")
	} else {
		args = append(args, "--local")
	}
	args = append(args, "--set", key, value)
	_, err := r.Exec(args...)
	if err != nil {
		return fmt.Errorf("setting config %s: %w", key, err)
	}
	return nil
}

func parseConfigOutput(output string) []ConfigEntry {
	var entries []ConfigEntry
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " = ", 2)
		if len(parts) == 2 {
			entries = append(entries, ConfigEntry{
				Key:   strings.TrimSpace(parts[0]),
				Value: strings.TrimSpace(parts[1]),
			})
		}
	}
	return entries
}
