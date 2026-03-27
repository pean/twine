package git

import (
	"os/exec"
	"strings"
)

// Run executes a git command in repoPath and returns trimmed stdout.
func Run(repoPath string, args ...string) (string, error) {
	cmdArgs := append([]string{"-C", repoPath}, args...)
	out, err := exec.Command("git", cmdArgs...).Output()
	return strings.TrimSpace(string(out)), err
}

// RunQuiet executes a git command in repoPath, discarding output.
func RunQuiet(repoPath string, args ...string) error {
	cmdArgs := append([]string{"-C", repoPath}, args...)
	return exec.Command("git", cmdArgs...).Run()
}

// Clone clones url into dest. Bare clone when bare is true.
func Clone(url, dest string, bare bool) error {
	args := []string{"clone"}
	if bare {
		args = append(args, "--bare")
	}
	args = append(args, url, dest)
	return exec.Command("git", args...).Run()
}

// CloneGH clones an "org/repo" shorthand as bare using the gh CLI.
func CloneGH(orgRepo, dest string) error {
	return exec.Command("gh", "repo", "clone", orgRepo, dest, "--", "--bare").Run()
}

// IsGitRepo returns true if path is a git repository (bare or not).
func IsGitRepo(path string) bool {
	err := exec.Command("git", "-C", path, "rev-parse", "--git-dir").Run()
	return err == nil
}

// IsBareRepo returns true if path is a bare git repository.
func IsBareRepo(path string) bool {
	out, err := exec.Command(
		"git", "-C", path, "rev-parse", "--is-bare-repository",
	).Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "true"
}