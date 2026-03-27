package commands

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/pean/twine/internal/config"
	"github.com/pean/twine/internal/repo"
	"github.com/pean/twine/internal/tmux"
	"github.com/pean/twine/internal/ui"
)

var killWorktree bool

var killCmd = &cobra.Command{
	Use:     "kill [session...]",
	Aliases: []string{"tk"},
	Short:   "Kill tmux sessions with optional worktree removal",
	Long: `Kill one or more tmux sessions.

When called without session arguments, interactive multi-select is shown and
you are asked whether to also remove worktrees for repo/branch sessions.

With --worktree (-w), worktrees are removed without prompting.`,
	RunE: runKill,
}

func init() {
	killCmd.Flags().BoolVarP(
		&killWorktree, "worktree", "w", false,
		"also remove the associated git worktree",
	)
}

func runKill(cmd *cobra.Command, args []string) error {
	sessions := args
	interactive := len(sessions) == 0

	if interactive {
		all, err := tmux.ListSessionsFull()
		if err != nil {
			return fmt.Errorf("failed to list sessions: %w", err)
		}
		if len(all) == 0 {
			fmt.Println("No tmux sessions running.")
			return nil
		}

		items := make([]ui.Item, 0, len(all))
		for _, s := range all {
			items = append(items, ui.Item{
				Title:  fmt.Sprintf("%s  %s  %dw", s.Name, s.Path, s.Windows),
				Value:  s.Name,
				Active: true,
			})
		}

		chosen, err := ui.MultiSelect(
			items, "Kill session(s): (Tab to select, Enter to confirm)",
		)
		if errors.Is(err, ui.ErrCancelled) {
			return nil
		}
		if err != nil {
			return err
		}
		for _, item := range chosen {
			sessions = append(sessions, item.Value)
		}

		// In interactive mode, ask about worktree removal if any selected
		// sessions look like repo/branch (and -w was not already set).
		if !killWorktree && hasRepoBranchSessions(sessions) {
			fmt.Print("Also remove worktrees? (y/N) ")
			var answer string
			fmt.Scan(&answer)
			if strings.ToLower(strings.TrimSpace(answer)) == "y" {
				killWorktree = true
			}
		}
	}

	cfg, _ := config.Load()

	for _, session := range sessions {
		if err := tmux.KillSession(session); err != nil {
			fmt.Fprintf(os.Stderr,
				"Warning: could not kill session %q: %v\n", session, err,
			)
		} else {
			fmt.Printf("Killed session: %s\n", session)
		}

		if !killWorktree {
			continue
		}

		repoRaw, branch, ok := splitRepoSession(session, cfg)
		if !ok {
			continue
		}

		baseDirs := []string{}
		if cfg != nil {
			baseDirs = cfg.BaseDirs
		}

		r, err := repo.Find(baseDirs, repoRaw)
		if err != nil {
			fmt.Fprintf(os.Stderr,
				"Warning: no bare repo found for %q, skipping worktree removal\n",
				repoRaw,
			)
			continue
		}

		if err := r.RemoveWorktree(branch); err != nil {
			fmt.Fprintf(os.Stderr,
				"Warning: could not remove worktree %q in %s: %v\n",
				branch, r.Path, err,
			)
		} else {
			fmt.Printf("Removed worktree: %s\n", branch)
		}
	}

	return nil
}

// hasRepoBranchSessions returns true if any session name contains "/".
func hasRepoBranchSessions(sessions []string) bool {
	for _, s := range sessions {
		if strings.Contains(s, "/") {
			return true
		}
	}
	return false
}

// splitRepoSession extracts repo name and branch from a session name,
// stripping any configured session prefix. Returns false if not repo/branch.
func splitRepoSession(session string, cfg *config.Config) (string, string, bool) {
	slashIdx := strings.Index(session, "/")
	if slashIdx < 0 {
		return "", "", false
	}
	repoRaw := session[:slashIdx]
	branch := session[slashIdx+1:]
	if cfg != nil && cfg.SessionPrefix != "" {
		repoRaw = strings.TrimPrefix(repoRaw, cfg.SessionPrefix)
	}
	return repoRaw, branch, true
}