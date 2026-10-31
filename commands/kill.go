package commands

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/pean/twine/internal/config"
	"github.com/pean/twine/internal/mux"
	"github.com/pean/twine/internal/repo"
	"github.com/pean/twine/internal/ui"
)

var killWorktree bool

var killCmd = &cobra.Command{
	Use:     "kill [session...]",
	Aliases: []string{"tk"},
	Short:   "Kill sessions with optional worktree removal",
	Long: `Kill one or more sessions, in whichever multiplexer runs them.

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

	cfg, _ := config.Load()
	set, err := muxSet(cfg)
	if err != nil {
		return err
	}

	// targets maps each session to kill to the backends running it. In
	// interactive mode that is the backend it was picked from; for names
	// given as arguments, every backend that has a session of that name.
	targets := map[string][]mux.Multiplexer{}

	if interactive {
		all := set.Sessions()
		if len(all) == 0 {
			fmt.Println("No sessions running.")
			return nil
		}

		items := make([]ui.Item, 0, len(all))
		for i, s := range all {
			title := s.Name + muxLabel(set, s.Backend)
			if s.Path != "" {
				title += fmt.Sprintf("  %s  %dw", s.Path, s.Windows)
			}
			items = append(items, ui.Item{
				Title:  title,
				Value:  strconv.Itoa(i),
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
			i, _ := strconv.Atoi(item.Value)
			s := all[i]
			if _, seen := targets[s.Name]; !seen {
				sessions = append(sessions, s.Name)
			}
			targets[s.Name] = append(targets[s.Name], s.Backend)
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

	for _, session := range sessions {
		owners, picked := targets[session]
		if !picked {
			owners = set.Owners(session)
		}
		if len(owners) == 0 {
			fmt.Fprintf(os.Stderr,
				"Warning: could not kill session %q: no such session\n", session,
			)
		}
		for _, m := range owners {
			if err := m.KillSession(session); err != nil {
				fmt.Fprintf(os.Stderr,
					"Warning: could not kill session %q%s: %v\n",
					session, muxLabel(set, m), err,
				)
			} else {
				fmt.Printf("Killed session: %s%s\n", session, muxLabel(set, m))
			}
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