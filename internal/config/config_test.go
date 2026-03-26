package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeConfig(t *testing.T, dir, content string) {
	t.Helper()
	cfgPath := filepath.Join(dir, ConfigPath)
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoad_basic(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	writeConfig(t, home, `
base_dirs = ["/tmp/repos"]
session_prefix = "dev-"
`)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.BaseDirs) != 1 || cfg.BaseDirs[0] != "/tmp/repos" {
		t.Errorf("unexpected base_dirs: %v", cfg.BaseDirs)
	}
	if cfg.SessionPrefix != "dev-" {
		t.Errorf("unexpected session_prefix: %q", cfg.SessionPrefix)
	}
}

func TestLoad_tildeExpansion(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	writeConfig(t, home, `base_dirs = ["~/src"]`)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(home, "src")
	if len(cfg.BaseDirs) != 1 || cfg.BaseDirs[0] != want {
		t.Errorf("got %v, want %v", cfg.BaseDirs, want)
	}
}

func TestLoad_defaultsUseTmuxinator(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	writeConfig(t, home, `base_dirs = ["/tmp"]`)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.UseTmuxinator != "auto" {
		t.Errorf("expected default use_tmuxinator=auto, got %q", cfg.UseTmuxinator)
	}
}

func TestLoad_missingFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing config file")
	}
}

func TestShouldUseTmuxinator_false(t *testing.T) {
	cfg := &Config{UseTmuxinator: "false"}
	if cfg.ShouldUseTmuxinator() {
		t.Error("expected false")
	}
}
