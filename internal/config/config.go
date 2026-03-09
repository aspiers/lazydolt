// Package config handles lazydolt's persistent user configuration,
// stored under $XDG_CONFIG_HOME/lazydolt/config.yaml.
package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds persistent user preferences.
type Config struct {
	LeftRatio int `yaml:"left_ratio,omitempty"`
}

// DefaultConfig returns the default configuration.
func DefaultConfig() Config {
	return Config{
		LeftRatio: 30,
	}
}

// Path returns the config file path under $XDG_CONFIG_HOME/lazydolt/config.yaml.
// Falls back to ~/.config if XDG_CONFIG_HOME is unset.
func Path() string {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "."
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "lazydolt", "config.yaml")
}

// Load reads the config file, returning defaults if the file doesn't exist.
func Load() Config {
	cfg := DefaultConfig()
	data, err := os.ReadFile(Path())
	if err != nil {
		return cfg
	}
	_ = yaml.Unmarshal(data, &cfg)
	// Clamp to valid range
	if cfg.LeftRatio < 10 || cfg.LeftRatio > 90 {
		cfg.LeftRatio = 30
	}
	return cfg
}

// Save writes the config to disk, creating directories as needed.
func Save(cfg Config) error {
	path := Path()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
