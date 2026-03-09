package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.LeftRatio != 30 {
		t.Errorf("DefaultConfig().LeftRatio = %d, want 30", cfg.LeftRatio)
	}
}

func TestSaveAndLoad(t *testing.T) {
	// Use a temp dir as XDG_CONFIG_HOME
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	cfg := Config{LeftRatio: 45}
	if err := Save(cfg); err != nil {
		t.Fatalf("Save(): %v", err)
	}

	// Verify file exists
	path := filepath.Join(tmp, "lazydolt", "config.yaml")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("config file not created: %v", err)
	}

	loaded := Load()
	if loaded.LeftRatio != 45 {
		t.Errorf("Load().LeftRatio = %d, want 45", loaded.LeftRatio)
	}
}

func TestLoadMissingFile(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cfg := Load()
	if cfg.LeftRatio != 30 {
		t.Errorf("Load() with missing file: LeftRatio = %d, want 30", cfg.LeftRatio)
	}
}

func TestLoadClampsInvalidValues(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	// Write an out-of-range value
	dir := filepath.Join(tmp, "lazydolt")
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("left_ratio: 200\n"), 0o644)

	cfg := Load()
	if cfg.LeftRatio != 30 {
		t.Errorf("Load() with invalid value: LeftRatio = %d, want 30 (clamped)", cfg.LeftRatio)
	}
}
