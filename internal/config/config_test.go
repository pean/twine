package config

import (
	"os"
	"path/filepath"
	"reflect"
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

const examplePath = "../../config.example.toml"

func TestExampleConfigIsValid(t *testing.T) {
	data, err := os.ReadFile(examplePath)
	if err != nil {
		t.Fatal(err)
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeConfig(t, home, string(data))

	cfg, err := Load()
	if err != nil {
		t.Fatalf("example config does not load: %v", err)
	}
	if cfg.Layout != "dev" || len(cfg.Layouts) != 2 || len(cfg.Repos) != 2 {
		t.Errorf("unexpected example contents: %+v", cfg)
	}
	if got := cfg.MultiplexerFor("work-api"); got != "rex" {
		t.Errorf("MultiplexerFor(work-api) = %q, want rex", got)
	}
	if l := cfg.LayoutFor("work-api"); l == nil || l.Windows[0].Name != "review" {
		t.Errorf("LayoutFor(work-api) = %+v, want the review layout", l)
	}
}

// Every option twine reads must be set in the example, so adding one without
// documenting it fails here.
func TestExampleConfigSetsEveryOption(t *testing.T) {
	data, err := os.ReadFile(examplePath)
	if err != nil {
		t.Fatal(err)
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeConfig(t, home, string(data))
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}

	set := map[string]bool{}
	recordSetFields(reflect.ValueOf(*cfg), set)
	for _, field := range allFields(reflect.TypeOf(Config{})) {
		if !set[field] {
			t.Errorf("config.example.toml does not set %s", field)
		}
	}
}

// recordSetFields records "Type.Field" for every non-zero struct field
// reachable from v.
func recordSetFields(v reflect.Value, set map[string]bool) {
	switch v.Kind() {
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			recordSetFields(v.Index(i), set)
		}
	case reflect.Map:
		for _, key := range v.MapKeys() {
			recordSetFields(v.MapIndex(key), set)
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if !v.Field(i).IsZero() {
				set[v.Type().Name()+"."+v.Type().Field(i).Name] = true
			}
			recordSetFields(v.Field(i), set)
		}
	}
}

// allFields lists "Type.Field" for every struct field reachable from t.
func allFields(t reflect.Type) []string {
	for t.Kind() == reflect.Slice || t.Kind() == reflect.Map {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil
	}
	var fields []string
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		fields = append(fields, t.Name()+"."+f.Name)
		fields = append(fields, allFields(f.Type)...)
	}
	return fields
}
