package git

import (
	"os/exec"
	"testing"
)

func initRepo(t *testing.T, dir string, bare bool) {
	t.Helper()
	args := []string{"init"}
	if bare {
		args = append(args, "--bare")
	}
	args = append(args, dir)
	if err := exec.Command("git", args...).Run(); err != nil {
		t.Fatalf("git init: %v", err)
	}
}

func TestIsGitRepo(t *testing.T) {
	dir := t.TempDir()
	if IsGitRepo(dir) {
		t.Error("expected false for empty dir")
	}
	initRepo(t, dir, false)
	if !IsGitRepo(dir) {
		t.Error("expected true after git init")
	}
}

func TestIsBareRepo(t *testing.T) {
	regular := t.TempDir()
	initRepo(t, regular, false)
	if IsBareRepo(regular) {
		t.Error("expected false for regular repo")
	}

	bare := t.TempDir()
	initRepo(t, bare, true)
	if !IsBareRepo(bare) {
		t.Error("expected true for bare repo")
	}
}
