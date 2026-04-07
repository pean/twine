package repo

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

// setupClonedBareRepo creates a regular origin repo with a commit and clones it
// bare, then sets the fetch refspec and runs fetch so refs/remotes/origin/ is
// populated (matching what a real remote bare clone produces).
func setupClonedBareRepo(t *testing.T) (originPath, barePath string) {
	t.Helper()
	base := t.TempDir()

	// Create origin repo with a commit on main.
	originPath = filepath.Join(base, "origin")
	run := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v: %s", args, out)
		}
	}
	run(base, "git", "init", "-b", "main", originPath)
	run(originPath, "git", "config", "user.email", "test@example.com")
	run(originPath, "git", "config", "user.name", "Test")
	run(originPath, "git", "commit", "--allow-empty", "-m", "init")

	// Bare clone of origin.
	barePath = filepath.Join(base, "repo.git")
	run(base, "git", "clone", "--bare", originPath, barePath)

	// Set fetch refspec (bare clones from local paths don't set one; remote
	// clones do — replicate that here so the test matches production behaviour).
	run(barePath, "git", "config", "remote.origin.fetch",
		"+refs/heads/*:refs/remotes/origin/*")
	run(barePath, "git", "fetch", "--all")
	return originPath, barePath
}

func TestListRemoteBranches_bare(t *testing.T) {
	_, barePath := setupClonedBareRepo(t)

	r := &Repo{Path: barePath, IsBare: true}
	branches, err := r.ListRemoteBranches()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, b := range branches {
		if strings.HasPrefix(b, "origin/") {
			t.Errorf("branch should not include origin/ prefix: %q", b)
		}
		if b == "main" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'main' in remote branches, got: %v", branches)
	}
}

func TestAddWorktree_setsUpstream(t *testing.T) {
	_, barePath := setupClonedBareRepo(t)

	r := &Repo{Path: barePath, Name: "repo", IsBare: true}
	if err := r.AddWorktree("main", "", false); err != nil {
		t.Fatalf("AddWorktree: %v", err)
	}

	// Verify branch.main.remote is set in the bare repo config.
	cmd := exec.Command("git", "-C", barePath, "config", "branch.main.remote")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("branch.main.remote not set: %v", err)
	}
	if remote := strings.TrimSpace(string(out)); remote != "origin" {
		t.Errorf("expected branch.main.remote=origin, got %q", remote)
	}
}

// TestSetupRemoteTracking verifies that a bare clone without a fetch refspec
// (the state produced by both `git clone --bare` and `gh repo clone -- --bare`
// from a real remote) is fixed by SetupRemoteTracking.
func TestSetupRemoteTracking(t *testing.T) {
	base := t.TempDir()

	// Create origin repo.
	originPath := filepath.Join(base, "origin")
	run := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v: %s", args, out)
		}
	}
	run(base, "git", "init", "-b", "main", originPath)
	run(originPath, "git", "config", "user.email", "test@example.com")
	run(originPath, "git", "config", "user.name", "Test")
	run(originPath, "git", "commit", "--allow-empty", "-m", "init")

	// Bare clone — intentionally skip setting the fetch refspec so we start
	// in the broken state that `git clone --bare` from a real remote produces.
	barePath := filepath.Join(base, "repo.git")
	run(base, "git", "clone", "--bare", originPath, barePath)

	r := &Repo{Path: barePath, Name: "repo", IsBare: true}

	// Before fix: no remote tracking refs.
	branches, _ := r.ListRemoteBranches()
	if len(branches) != 0 {
		t.Fatalf("expected no remote branches before setup, got: %v", branches)
	}

	if err := r.SetupRemoteTracking("origin"); err != nil {
		t.Fatalf("SetupRemoteTracking: %v", err)
	}

	// After fix: refs/remotes/origin/main should be present.
	branches, err := r.ListRemoteBranches()
	if err != nil {
		t.Fatalf("ListRemoteBranches: %v", err)
	}
	found := false
	for _, b := range branches {
		if b == "main" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'main' after SetupRemoteTracking, got: %v", branches)
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
