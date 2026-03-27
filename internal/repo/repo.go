package repo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/pean/twine/internal/git"
)

// Repo represents a discovered git repository.
type Repo struct {
	Path   string
	Name   string
	IsBare bool
}

// Find searches baseDirs for a repo named name (bare preferred).
func Find(baseDirs []string, name string) (*Repo, error) {
	name = strings.TrimSuffix(name, ".git")
	for _, base := range baseDirs {
		barePath := filepath.Join(base, name+".git")
		if info, err := os.Stat(barePath); err == nil && info.IsDir() {
			return &Repo{Path: barePath, Name: name, IsBare: true}, nil
		}
		regPath := filepath.Join(base, name)
		if info, err := os.Stat(regPath); err == nil && info.IsDir() {
			return &Repo{Path: regPath, Name: name, IsBare: false}, nil
		}
	}
	return nil, fmt.Errorf("repository %q not found in base dirs", name)
}

// FindAll returns all repos in baseDirs.
// Entries are checked in parallel using stat calls rather than git subprocesses.
func FindAll(baseDirs []string) ([]*Repo, error) {
	var mu sync.Mutex
	var repos []*Repo
	seen := map[string]bool{}

	for _, base := range baseDirs {
		entries, err := os.ReadDir(base)
		if err != nil {
			continue
		}

		var wg sync.WaitGroup
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			fullPath := filepath.Join(base, e.Name())
			mu.Lock()
			if seen[fullPath] {
				mu.Unlock()
				continue
			}
			seen[fullPath] = true
			mu.Unlock()

			wg.Add(1)
			go func(name, path string) {
				defer wg.Done()
				var r *Repo
				if strings.HasSuffix(name, ".git") {
					// Bare repos have a HEAD file at their root.
					if _, err := os.Stat(filepath.Join(path, "HEAD")); err == nil {
						r = &Repo{
							Path:   path,
							Name:   strings.TrimSuffix(name, ".git"),
							IsBare: true,
						}
					}
				} else {
					// Regular repos have a .git entry at their root.
					if _, err := os.Stat(filepath.Join(path, ".git")); err == nil {
						r = &Repo{Path: path, Name: name, IsBare: false}
					}
				}
				if r != nil {
					mu.Lock()
					repos = append(repos, r)
					mu.Unlock()
				}
			}(e.Name(), fullPath)
		}
		wg.Wait()
	}
	return repos, nil
}

// ListWorktrees returns branch/worktree names (relative), excluding bare root.
func (r *Repo) ListWorktrees() ([]string, error) {
	out, err := git.Run(r.Path, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	var result []string
	for _, line := range strings.Split(out, "\n") {
		if !strings.HasPrefix(line, "worktree ") {
			continue
		}
		wt := strings.TrimPrefix(line, "worktree ")
		if wt == r.Path {
			continue // skip bare root
		}
		rel, err := filepath.Rel(r.Path, wt)
		if err != nil {
			rel = wt
		}
		if rel != "" && rel != "." {
			result = append(result, rel)
		}
	}
	return result, nil
}

// ListRemoteBranches returns remote branch names (strips "origin/" prefix,
// excludes HEAD).
func (r *Repo) ListRemoteBranches() ([]string, error) {
	out, err := git.Run(r.Path, "branch", "-r")
	if err != nil {
		return nil, err
	}
	var result []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "* ")
		if !strings.HasPrefix(line, "origin/") {
			continue
		}
		name := strings.TrimPrefix(line, "origin/")
		if name == "HEAD" || strings.Contains(name, "->") {
			continue
		}
		result = append(result, name)
	}
	return result, nil
}

// ListLocalBranches returns local branch names.
func (r *Repo) ListLocalBranches() ([]string, error) {
	out, err := git.Run(r.Path, "branch")
	if err != nil {
		return nil, err
	}
	var result []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "* ")
		line = strings.TrimPrefix(line, "+ ")
		if line != "" {
			result = append(result, line)
		}
	}
	return result, nil
}

// DefaultBranch returns "main" or "master" by checking remote refs.
func (r *Repo) DefaultBranch() (string, error) {
	remoteBranches, _ := r.ListRemoteBranches()
	for _, b := range remoteBranches {
		if b == "main" {
			return "main", nil
		}
	}
	for _, b := range remoteBranches {
		if b == "master" {
			return "master", nil
		}
	}
	// Try symbolic-ref
	out, err := git.Run(
		r.Path, "symbolic-ref", "refs/remotes/origin/HEAD",
	)
	if err == nil && out != "" {
		parts := strings.Split(out, "/")
		return parts[len(parts)-1], nil
	}
	return "", fmt.Errorf("could not determine default branch")
}

// hasRemoteBranch checks whether origin/$branch exists.
func (r *Repo) hasRemoteBranch(branch string) bool {
	remoteBranches, _ := r.ListRemoteBranches()
	for _, b := range remoteBranches {
		if b == branch {
			return true
		}
	}
	return false
}

// hasLocalBranch checks whether a local branch exists.
func (r *Repo) hasLocalBranch(branch string) bool {
	localBranches, _ := r.ListLocalBranches()
	for _, b := range localBranches {
		if b == branch {
			return true
		}
	}
	return false
}

// AddWorktree creates a worktree for branch. Three cases:
//  1. Remote AND local branch exist → worktree add + set-upstream
//  2. Remote only → worktree add -b from origin/$branch
//  3. New branch (createNew) → worktree add -b from startPoint
func (r *Repo) AddWorktree(branch, startPoint string, createNew bool) error {
	wtPath := filepath.Join(r.Path, branch)

	hasRemote := r.hasRemoteBranch(branch)
	hasLocal := r.hasLocalBranch(branch)

	if hasRemote && hasLocal {
		if err := git.RunQuiet(
			r.Path, "worktree", "add", wtPath, branch,
		); err != nil {
			return fmt.Errorf("worktree add failed: %w", err)
		}
		// Explicitly set upstream (may not be configured).
		_ = git.RunQuiet(
			r.Path,
			"branch",
			"--set-upstream-to=origin/"+branch,
			branch,
		)
		return nil
	}

	if hasRemote {
		if err := git.RunQuiet(
			r.Path,
			"worktree", "add",
			"-b", branch,
			wtPath,
			"origin/"+branch,
		); err != nil {
			return fmt.Errorf("worktree add failed: %w", err)
		}
		return nil
	}

	if createNew {
		if startPoint == "" {
			base, err := r.DefaultBranch()
			if err != nil {
				return err
			}
			// Prefer remote start point.
			if r.hasRemoteBranch(base) {
				startPoint = "origin/" + base
			} else if r.hasLocalBranch(base) {
				startPoint = base
			} else {
				return fmt.Errorf(
					"base branch %q not found (neither remote nor local)",
					base,
				)
			}
		} else {
			// Validate the caller-supplied base.
			base := startPoint
			if r.hasRemoteBranch(base) {
				startPoint = "origin/" + base
			} else if !r.hasLocalBranch(base) {
				return fmt.Errorf(
					"base branch %q not found (neither remote nor local)",
					base,
				)
			}
		}
		if err := git.RunQuiet(
			r.Path,
			"worktree", "add",
			"-b", branch,
			wtPath,
			startPoint,
		); err != nil {
			return fmt.Errorf("worktree add failed: %w", err)
		}
		return nil
	}

	return fmt.Errorf(
		"branch %q not found on remote; use --create to create a new branch",
		branch,
	)
}

// RemoveWorktree removes a worktree with --force.
func (r *Repo) RemoveWorktree(branch string) error {
	wtPath := filepath.Join(r.Path, branch)
	return git.RunQuiet(r.Path, "worktree", "remove", wtPath, "--force")
}

// SetUpstream sets upstream tracking for branch.
func (r *Repo) SetUpstream(branch, upstream string) error {
	return git.RunQuiet(
		r.Path, "branch", "--set-upstream-to="+upstream, branch,
	)
}

// GoneBranches returns branches whose remote tracking ref is "gone".
func (r *Repo) GoneBranches() ([]string, error) {
	out, err := git.Run(r.Path, "branch", "-vv")
	if err != nil {
		return nil, err
	}
	var result []string
	for _, line := range strings.Split(out, "\n") {
		if !strings.Contains(line, ": gone]") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 1 {
			continue
		}
		name := fields[0]
		if name == "*" || name == "+" {
			if len(fields) >= 2 {
				name = fields[1]
			} else {
				continue
			}
		}
		result = append(result, name)
	}
	return result, nil
}

// Fetch runs git fetch --all -p and prunes stale worktree refs.
func (r *Repo) Fetch() error {
	if err := git.RunQuiet(r.Path, "fetch", "--all", "-p"); err != nil {
		return err
	}
	// Prune stale worktree admin files (equivalent to git worktree prune).
	_ = git.RunQuiet(r.Path, "worktree", "prune")
	return nil
}