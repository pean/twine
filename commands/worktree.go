package commands

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/spf13/cobra"

	"github.com/pean/twine/internal/config"
	"github.com/pean/twine/internal/git"
	"github.com/pean/twine/internal/repo"
	"github.com/pean/twine/internal/tmux"
	"github.com/pean/twine/internal/ui"
)

var (
	worktreeCreate   bool
	worktreeFromBase string
	worktreeRemote   bool
)

var worktreeCmd = &cobra.Command{
	Use:     "worktree [repo] [branch]",
	Aliases: []string{"tw"},
	Short:   "Create worktree and switch tmux session",
	Long: `Create a git worktree for branch and switch to its tmux session.

When no arguments are given, interactive selection is offered.
When the repository is not found, you are prompted to clone it.
With --create, a new branch is created rather than checked out from remote.`,
	RunE: runWorktree,
}

func init() {
	worktreeCmd.Flags().BoolVarP(
		&worktreeCreate, "create", "c", false,
		"create new branch if it doesn't exist",
	)
	worktreeCmd.Flags().StringVarP(
		&worktreeFromBase, "from", "f", "",
		"base branch for new branch (default: main/master)",
	)
	worktreeCmd.Flags().BoolVarP(
		&worktreeRemote, "remote", "r", false,
		"fetch and show remote branches instead of local",
	)
}

func runWorktree(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if len(cfg.BaseDirs) == 0 {
		return fmt.Errorf(
			"base_dirs not set in config\nRun: twine config init",
		)
	}

	repoName := ""
	branch := ""
	if len(args) >= 1 {
		repoName = strings.TrimSuffix(args[0], ".git")
	}
	if len(args) >= 2 {
		branch = args[1]
	}

	// ---- repo selection ----
	if repoName == "" {
		name, err := selectRepo(cfg)
		if err != nil {
			return err
		}
		repoName = name
	}

	// ---- find repo, clone if missing ----
	r, err := repo.Find(cfg.BaseDirs, repoName)
	if err != nil {
		r, err = cloneBarPrompt(repoName, cfg)
		if err != nil {
			return err
		}
		// Default branch becomes the worktree if none specified.
		if branch == "" {
			branch, _ = r.DefaultBranch()
		}
	}

	// ---- offer conversion for regular repos ----
	if !r.IsBare {
		fmt.Printf("Found regular repository: %s\n", r.Path)
		fmt.Print("Convert to bare repo for full worktree support? (y/N) ")
		var answer string
		fmt.Scan(&answer)
		fmt.Println()
		if strings.ToLower(strings.TrimSpace(answer)) == "y" {
			if err := convertToBare(r.Path); err != nil {
				return err
			}
			r, err = repo.Find(cfg.BaseDirs, repoName)
			if err != nil {
				return err
			}
		}
	}

	// ---- branch selection ----
	if branch == "" {
		b, err := selectBranch(r, repoName, worktreeRemote)
		if err != nil {
			return err
		}
		branch = b
	} else if worktreeRemote {
		var fetchErr error
		_ = ui.Spinner("Fetching branches…", func() error {
			fetchErr = r.Fetch()
			return nil
		})
		if fetchErr != nil {
			return fmt.Errorf("fetch failed: %w", fetchErr)
		}
	}

	worktreePath := filepath.Join(r.Path, branch)
	sessionName := cfg.SessionPrefix + repoName + "/" + branch

	// ---- create worktree if needed ----
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		fmt.Printf("Creating worktree for %q…\n", branch)
		if err := r.AddWorktree(branch, worktreeFromBase, worktreeCreate); err != nil {
			return err
		}
		fmt.Printf("✓ Worktree created: %s\n", branch)
	}

	// ---- create or switch tmux session ----
	if !tmux.HasSession(sessionName) {
		fmt.Printf("Creating tmux session: %s\n", sessionName)
		if cfg.ShouldUseTmuxinator() {
			err = runTmuxinator(sessionName, worktreePath, cfg)
		} else {
			err = tmux.NewSession(sessionName, worktreePath)
		}
		if err != nil {
			return fmt.Errorf("failed to create session: %w", err)
		}
	}

	return tmux.AttachOrSwitch(sessionName)
}

// cloneBarPrompt asks the user for a git URL (or org/repo shorthand) and
// clones it as a bare repo into BaseDirs[0].
func cloneBarPrompt(repoName string, cfg *config.Config) (*repo.Repo, error) {
	fmt.Printf("Repository %q not found.\n", repoName)
	fmt.Print("Git URL or org/repo to clone (empty to cancel): ")

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	input := strings.TrimSpace(scanner.Text())
	if input == "" {
		return nil, fmt.Errorf("repository %q not found", repoName)
	}

	barePath := filepath.Join(cfg.BaseDirs[0], repoName+".git")

	// Accept "org/repo" shorthand — requires gh CLI.
	if isOrgRepo(input) {
		fmt.Printf("Cloning %s via gh…\n", input)
		if err := git.CloneGH(input, barePath); err != nil {
			return nil, fmt.Errorf("gh clone failed: %w", err)
		}
	} else {
		fmt.Printf("Cloning %s…\n", input)
		if err := git.Clone(input, barePath, true); err != nil {
			return nil, fmt.Errorf("clone failed: %w", err)
		}
	}

	fmt.Printf("✓ Cloned: %s\n", barePath)

	r := &repo.Repo{Path: barePath, Name: repoName, IsBare: true}

	// A bare clone does not populate refs/remotes/origin/; fetch now so that
	// AddWorktree (called next) can resolve remote tracking refs correctly.
	_ = ui.Spinner("Fetching remote branches…", func() error {
		return r.Fetch()
	})

	return r, nil
}

// isOrgRepo returns true for "org/repo" style shorthands (no slashes elsewhere,
// no protocol prefix).
func isOrgRepo(s string) bool {
	if strings.Contains(s, "://") || strings.HasPrefix(s, "git@") {
		return false
	}
	parts := strings.Split(s, "/")
	return len(parts) == 2 && parts[0] != "" && parts[1] != ""
}

// convertToBare converts a regular git repo to a bare + worktree layout.
func convertToBare(repoPath string) error {
	repoName := filepath.Base(repoPath)
	barePath := filepath.Join(filepath.Dir(repoPath), repoName+".git")

	if isDir(barePath) {
		return fmt.Errorf("bare repo already exists: %s", barePath)
	}

	currentBranch, err := git.Run(repoPath, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil || currentBranch == "" {
		return fmt.Errorf("not a git repository: %s", repoPath)
	}

	originURL, _ := git.Run(repoPath, "remote", "get-url", "origin")

	fmt.Printf("Cloning %s as bare…\n", repoName)
	if err := git.Clone(repoPath, barePath, true); err != nil {
		return fmt.Errorf("failed to clone as bare: %w", err)
	}

	if originURL != "" {
		_ = git.RunQuiet(barePath, "remote", "set-url", "origin", originURL)
	}

	// A bare clone from a local path doesn't set a fetch refspec; add one now
	// so that fetch populates refs/remotes/origin/ for proper tracking.
	_ = git.RunQuiet(
		barePath, "config", "remote.origin.fetch",
		"+refs/heads/*:refs/remotes/origin/*",
	)
	_ = git.RunQuiet(barePath, "fetch", "--all", "-p")

	wtPath := filepath.Join(barePath, currentBranch)
	if err := git.RunQuiet(
		barePath, "worktree", "add", wtPath, currentBranch,
	); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to create worktree: %v\n", err)
	} else {
		fmt.Printf("✓ Worktree created: %s\n", wtPath)
		_ = git.RunQuiet(
			barePath, "branch", "--set-upstream-to=origin/"+currentBranch, currentBranch,
		)
	}

	fmt.Printf("Old repository at %s — remove it? (y/N) ", repoPath)
	var answer string
	fmt.Scan(&answer)
	fmt.Println()
	if strings.ToLower(strings.TrimSpace(answer)) == "y" {
		if err := os.RemoveAll(repoPath); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		} else {
			fmt.Printf("✓ Removed: %s\n", repoPath)
		}
	}
	return nil
}

// selectRepo presents an interactive list of repos + active sessions.
func selectRepo(cfg *config.Config) (string, error) {
	var (
		sessions []string
		allRepos []*repo.Repo
		wg       sync.WaitGroup
	)
	wg.Add(2)
	go func() { defer wg.Done(); sessions, _ = tmux.ListSessions() }()
	go func() { defer wg.Done(); allRepos, _ = repo.FindAll(cfg.BaseDirs) }()
	wg.Wait()

	sessionRepos := map[string]bool{}
	for _, s := range sessions {
		if idx := strings.Index(s, "/"); idx >= 0 {
			sessionRepos[s[:idx]] = true
		}
	}

	var items []ui.Item
	for name := range sessionRepos {
		items = append(items, ui.Item{Title: name, Value: name, Active: true})
	}
	for _, r := range allRepos {
		if !sessionRepos[r.Name] {
			items = append(items, ui.Item{Title: r.Name, Value: r.Name})
		}
	}

	chosen, err := ui.Select(items, "Select repo:")
	if errors.Is(err, ui.ErrCancelled) {
		return "", fmt.Errorf("selection cancelled")
	}
	return chosen.Value, err
}

// selectBranch presents an interactive branch list.
// Without remote: shows worktrees + local branches (no network call).
// With remote: fetches and shows remote branches only.
func selectBranch(r *repo.Repo, repoName string, remote bool) (string, error) {
	var items []ui.Item

	if remote {
		var fetchErr error
		_ = ui.Spinner("Fetching branches…", func() error {
			fetchErr = r.Fetch()
			return nil
		})
		if fetchErr != nil {
			return "", fmt.Errorf("fetch failed: %w", fetchErr)
		}
		branches, _ := r.ListRemoteBranches()
		for _, b := range branches {
			items = append(items, ui.Item{Title: b, Value: b})
		}
	} else {
		worktrees, _ := r.ListWorktrees()
		wtSet := map[string]bool{}
		for _, wt := range worktrees {
			wtSet[wt] = true
		}
		locals, _ := r.ListLocalBranches()

		seen := map[string]bool{}
		for _, wt := range worktrees {
			if !seen[wt] {
				items = append(items, ui.Item{Title: wt, Value: wt, Active: true})
				seen[wt] = true
			}
		}
		for _, b := range locals {
			if !seen[b] {
				items = append(items, ui.Item{Title: b, Value: b, Active: wtSet[b]})
				seen[b] = true
			}
		}
	}

	chosen, err := ui.Select(items, "Select branch for "+repoName+":")
	if errors.Is(err, ui.ErrCancelled) {
		return "", fmt.Errorf("selection cancelled")
	}
	return chosen.Value, err
}