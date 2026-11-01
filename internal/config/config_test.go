package config

import (
	"os"
	"path/filepath"
	"strings"
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

func TestLoad_missingFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing config file")
	}
}

func TestLoad_multiplexerDefaultsToAuto(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeConfig(t, home, `base_dirs = ["/tmp"]`)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Multiplexer != MultiplexerAuto {
		t.Errorf("Multiplexer = %q, want auto", cfg.Multiplexer)
	}
	if got := cfg.MultiplexerFor("anything"); got != MultiplexerAuto {
		t.Errorf("MultiplexerFor() = %q, want auto", got)
	}
}

func TestLoad_multiplexerPerRepo(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeConfig(t, home, `
base_dirs = ["/tmp"]
multiplexer = "tmux"

[[repos]]
match = "work-*"
multiplexer = "rex"

[[repos]]
match = "work-legacy"
multiplexer = "tmux"

[[repos]]
match = "notes"
`)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cases := map[string]string{
		"work-api":    "rex",
		"work-legacy": "rex",  // first match wins
		"notes":       "tmux", // entry without multiplexer falls through
		"personal":    "tmux",
		"":            "tmux",
	}
	for repo, want := range cases {
		if got := cfg.MultiplexerFor(repo); got != want {
			t.Errorf("MultiplexerFor(%q) = %q, want %q", repo, got, want)
		}
	}
}

func TestLoad_invalidMultiplexer(t *testing.T) {
	cases := map[string]string{
		"top-level": `base_dirs = ["/tmp"]
multiplexer = "screen"`,
		"per-repo": `base_dirs = ["/tmp"]
[[repos]]
match = "x"
multiplexer = "zellij"`,
		"bad glob": `base_dirs = ["/tmp"]
[[repos]]
match = "[x"
multiplexer = "tmux"`,
		"missing match": `base_dirs = ["/tmp"]
[[repos]]
multiplexer = "tmux"`,
	}
	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			writeConfig(t, home, content)
			if _, err := Load(); err == nil {
				t.Error("expected an error")
			}
		})
	}
}

const layoutConfig = `
base_dirs = ["/tmp"]
layout = "dev"

[layouts.dev]
windows = [
  { name = "code", panes = [
      { command = "nvim" },
      { split = "below", size = 25 },
  ]},
]

[layouts.plain]
windows = [{ panes = [{}] }]

[[repos]]
match = "work-*"
layout = "plain"

[[repos]]
match = "notes"
multiplexer = "tmux"
`

func TestLayoutFor(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeConfig(t, home, layoutConfig)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	dev := cfg.LayoutFor("personal")
	if dev == nil || len(dev.Windows) != 1 || dev.Windows[0].Name != "code" {
		t.Fatalf("LayoutFor(personal) = %+v, want the dev layout", dev)
	}
	if p := dev.Windows[0].Panes[1]; p.Split != "below" || p.Size != 25 {
		t.Errorf("second pane = %+v", p)
	}
	if got := cfg.LayoutFor("work-api"); got == nil || got.Windows[0].Name != "" {
		t.Errorf("LayoutFor(work-api) = %+v, want the plain layout", got)
	}
	// A matching entry without a layout falls back to the top-level one.
	if got := cfg.LayoutFor("notes"); got == nil || got.Windows[0].Name != "code" {
		t.Errorf("LayoutFor(notes) = %+v, want the dev layout", got)
	}
}

func TestLayoutFor_noneConfigured(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeConfig(t, home, `base_dirs = ["/tmp"]`)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := cfg.LayoutFor("anything"); got != nil {
		t.Errorf("LayoutFor() = %+v, want nil", got)
	}
}

func TestLoad_invalidLayouts(t *testing.T) {
	cases := map[string]string{
		"unknown top-level": `base_dirs = ["/tmp"]
layout = "missing"`,
		"unknown per-repo": `base_dirs = ["/tmp"]
[[repos]]
match = "x"
layout = "missing"`,
		"no panes": `base_dirs = ["/tmp"]
[layouts.a]
windows = [{ name = "w", panes = [] }]`,
		"no windows": `base_dirs = ["/tmp"]
[layouts.a]
windows = []`,
		"bad split": `base_dirs = ["/tmp"]
[layouts.a]
windows = [{ panes = [{}, { split = "diagonal" }] }]`,
		"bad size": `base_dirs = ["/tmp"]
[layouts.a]
windows = [{ panes = [{}, { size = 100 }] }]`,
		"first pane splits": `base_dirs = ["/tmp"]
[layouts.a]
windows = [{ panes = [{ split = "right" }] }]`,
	}
	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			writeConfig(t, home, content)
			if _, err := Load(); err == nil {
				t.Error("expected an error")
			}
		})
	}
}

func TestLoad_tmuxinatorKeysAreRejected(t *testing.T) {
	for _, key := range []string{
		`use_tmuxinator = "auto"`,
		`tmuxinator_layout = "dev"`,
	} {
		t.Run(key, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			writeConfig(t, home, "base_dirs = [\"/tmp\"]\n"+key)
			_, err := Load()
			if err == nil || !strings.Contains(err.Error(), "layouts") {
				t.Errorf("err = %v, want a hint about layouts", err)
			}
		})
	}
}
