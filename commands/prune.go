package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/pean/twine/internal/config"
	"github.com/pean/twine/internal/git"
	"github.com/pean/twine/internal/repo"
	"github.com/pean/twine/internal/tmux"
)

var (
	pruneDryRun bool
	pruneAll    bool
)

var pruneCmd = &cobra.Command{
	Use:   "prune [repo]",
	Short: "Clean up gone branches, worktrees, and sessions",
	Long: `Prune branches whose remote tracking ref has been deleted.

By default only repos with active tmux sessions are checked (fast).
Use --all to scan every repo in base_dirs.

What it does:
  1. Fetches all remotes and prunes remote tracking branches
  2. Finds branches marked as 'gone' (remote deleted)
  3. Kills tmux sessions for gone branches
  4. Removes worktrees for gone branches (bare repos)
  5. Deletes local branches marked as gone`,
	Args: cobra.MaximumNArgs(1),
	RunE: runPrune,
}

func init() {
	pruneCmd.Flags().BoolVarP(
		&pruneDryRun, "dry-run", "n", false,
		"show what would be removed without doing it",
	)
	pruneCmd.Flags().BoolVarP(
		&pruneAll, "all", "a", false,
		"prune all repos in base_dirs, not just those with active sessions",
	)
}

func runPrune(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if len(cfg.BaseDirs) == 0 {
		return fmt.Errorf(
			"base_dirs not set in config\nRun: twine config init",
		)
	}

	var repos []*repo.Repo

	switch {
	case len(args) == 1:
		r, err := repo.Find(cfg.BaseDirs, args[0])
		if err != nil {
			return fmt.Errorf("repository not found: %s", args[0])
		}
		repos = []*repo.Repo{r}
	case pruneAll:
		repos, err = repo.FindAll(cfg.BaseDirs)
		if err != nil {
			return err
		}
	default:
		// Only repos that have at least one active tmux session.
		sessions, _ := tmux.ListSessions()
		for _, name := range activeRepoNames(sessions, cfg.SessionPrefix) {
			r, err := repo.Find(cfg.BaseDirs, name)
			if err == nil {
				repos = append(repos, r)
			}
		}
	}

	if len(repos) == 0 {
		if !pruneAll && len(args) == 0 {
			fmt.Println("No active sessions found. Run with --all to check all repos.")
		} else {
			fmt.Println("No repositories found.")
		}
		return nil
	}

	if pruneDryRun {
		fmt.Println("(dry run)")
		fmt.Println()
	}

	totalPruned := 0

	for _, r := range repos {
		fmt.Printf("→ %s\n", r.Name)

		if err := r.Fetch(); err != nil {
			fmt.Fprintf(os.Stderr, "  Warning: fetch failed: %v\n", err)
		}

		goneBranches, err := r.GoneBranches()
		if err != nil {
			fmt.Fprintf(
				os.Stderr, "  Warning: could not list gone branches: %v\n", err,
			)
			continue
		}

		if len(goneBranches) == 0 {
			fmt.Println("  ✓ Nothing to prune")
			continue
		}

		for _, branch := range goneBranches {
			sessionName := cfg.SessionPrefix + r.Name + "/" + branch
			wtPath := r.WorktreePathForBranch(branch)

			if pruneDryRun {
				if wtPath != "" {
					fmt.Printf("  would remove worktree: %s\n", branch)
				}
				if tmux.HasSession(sessionName) {
					fmt.Printf("  would kill session: %s\n", sessionName)
				}
				fmt.Printf("  would delete branch: %s\n", branch)
				totalPruned++
				continue
			}

			if tmux.HasSession(sessionName) {
				if err := tmux.KillSession(sessionName); err != nil {
					fmt.Fprintf(
						os.Stderr,
						"  Warning: failed to kill session %s: %v\n",
						sessionName, err,
					)
				} else {
					fmt.Printf("  ✓ Killed session: %s\n", sessionName)
				}
			}

			if wtPath != "" {
				if err := git.RunQuiet(r.Path, "worktree", "remove", "--force", wtPath); err != nil {
					fmt.Fprintf(
						os.Stderr,
						"  Warning: failed to remove worktree %s: %v\n",
						branch, err,
					)
				} else {
					fmt.Printf("  ✓ Removed worktree: %s\n", branch)
				}
			}

			if err := git.RunQuiet(r.Path, "branch", "-D", branch); err != nil {
				fmt.Fprintf(
					os.Stderr,
					"  Warning: failed to delete branch %s: %v\n",
					branch, err,
				)
			} else {
				fmt.Printf("  ✓ Deleted branch: %s\n", branch)
				totalPruned++
			}
		}
	}

	repoWord := "repo"
	if len(repos) != 1 {
		repoWord = "repos"
	}
	if pruneDryRun {
		fmt.Printf(
			"Would prune %d branch(es) across %d %s.\n",
			totalPruned, len(repos), repoWord,
		)
	} else {
		fmt.Printf(
			"Done. Pruned %d branch(es) across %d %s.\n",
			totalPruned, len(repos), repoWord,
		)
	}
	return nil
}

// activeRepoNames returns unique repo names derived from active tmux session
// names, stripping the optional session prefix.
// Session format: {prefix}{repo}/{branch}
func activeRepoNames(sessions []string, prefix string) []string {
	seen := map[string]bool{}
	var names []string
	for _, s := range sessions {
		s = strings.TrimPrefix(s, prefix)
		if idx := strings.Index(s, "/"); idx > 0 {
			name := s[:idx]
			if !seen[name] {
				seen[name] = true
				names = append(names, name)
			}
		}
	}
	return names
}