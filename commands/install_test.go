package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRexConfigDir(t *testing.T) {
	env := func(vars map[string]string) func(string) string {
		return func(k string) string { return vars[k] }
	}
	cases := []struct {
		desc string
		vars map[string]string
		want string
	}{
		{"default", nil, "/home/u/.config/rex"},
		{"xdg", map[string]string{"XDG_CONFIG_HOME": "/x"}, "/x/rex"},
		{
			"REX_CONFIG wins",
			map[string]string{"XDG_CONFIG_HOME": "/x", "REX_CONFIG": "/c/my.lua"},
			"/c",
		},
	}
	for _, tc := range cases {
		if got := rexConfigDir(env(tc.vars), "/home/u"); got != tc.want {
			t.Errorf("%s: rexConfigDir = %q, want %q", tc.desc, got, tc.want)
		}
	}
}

func TestWriteRexModule(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "rex")
	wrote, err := writeRexModule(dir)
	if err != nil || !wrote {
		t.Fatalf("writeRexModule() = %v, %v; want written", wrote, err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "twine.lua"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), `name  = "twine_agents"`) {
		t.Errorf("module does not define twine_agents:\n%s", got)
	}

	// An existing file, possibly edited, is left alone.
	custom := []byte("-- mine")
	if err := os.WriteFile(filepath.Join(dir, "twine.lua"), custom, 0o644); err != nil {
		t.Fatal(err)
	}
	wrote, err = writeRexModule(dir)
	if err != nil || wrote {
		t.Fatalf("second writeRexModule() = %v, %v; want left alone", wrote, err)
	}
	got, _ = os.ReadFile(filepath.Join(dir, "twine.lua"))
	if string(got) != string(custom) {
		t.Errorf("existing file was overwritten: %q", got)
	}
}
