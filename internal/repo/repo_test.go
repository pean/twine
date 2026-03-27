package repo

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func initBareRepo(t *testing.T, base, name string) string {
	t.Helper()
	path := filepath.Join(base, name+".git")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "init", "--bare", path).Run(); err != nil {
		t.Fatalf("git init --bare: %v", err)
	}
	return path
}

func initRegularRepo(t *testing.T, base, name string) string {
	t.Helper()
	path := filepath.Join(base, name)
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "init", path).Run(); err != nil {
		t.Fatalf("git init: %v", err)
	}
	return path
}

func TestFind_bare(t *testing.T) {
	base := t.TempDir()
	initBareRepo(t, base, "myrepo")

	r, err := Find([]string{base}, "myrepo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !r.IsBare {
		t.Error("expected bare repo")
	}
	if r.Name != "myrepo" {
		t.Errorf("unexpected name: %q", r.Name)
	}
}

func TestFind_regular(t *testing.T) {
	base := t.TempDir()
	initRegularRepo(t, base, "myrepo")

	r, err := Find([]string{base}, "myrepo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.IsBare {
		t.Error("expected regular repo")
	}
}

func TestFind_prefersbare(t *testing.T) {
	base := t.TempDir()
	initBareRepo(t, base, "myrepo")
	initRegularRepo(t, base, "myrepo")

	r, err := Find([]string{base}, "myrepo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !r.IsBare {
		t.Error("expected bare repo to be preferred")
	}
}

func TestFind_notFound(t *testing.T) {
	base := t.TempDir()
	_, err := Find([]string{base}, "missing")
	if err == nil {
		t.Error("expected error for missing repo")
	}
}

func TestFind_stripsGitSuffix(t *testing.T) {
	base := t.TempDir()
	initBareRepo(t, base, "myrepo")

	r, err := Find([]string{base}, "myrepo.git")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Name != "myrepo" {
		t.Errorf("expected name %q, got %q", "myrepo", r.Name)
	}
}

func TestFindAll(t *testing.T) {
	base := t.TempDir()
	initBareRepo(t, base, "alpha")
	initRegularRepo(t, base, "beta")
	if err := os.MkdirAll(filepath.Join(base, "notarepo"), 0o755); err != nil {
		t.Fatal(err)
	}

	repos, err := FindAll([]string{base})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repos) != 2 {
		t.Errorf("expected 2 repos, got %d", len(repos))
	}

	names := map[string]bool{}
	for _, r := range repos {
		names[r.Name] = true
	}
	if !names["alpha"] || !names["beta"] {
		t.Errorf("unexpected repos: %v", names)
	}
}
