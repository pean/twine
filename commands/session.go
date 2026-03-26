package commands

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/pean/twine/internal/config"
	"github.com/pean/twine/internal/repo"
	"github.com/pean/twine/internal/tmux"
	"github.com/pean/twine/internal/ui"
)

var sessionCmd = &cobra.Command{
	Use:     "session [repo-or-session]",
	Aliases: []string{"t"},
	Short:   "Switch to a tmux session (faster, no branch selection)",
	Long: `Switch to an existing tmux session or create one for a repo.

Unlike the worktree command, no branch selection is shown.
Interactive selection is offered when called without arguments.`,
	RunE: runSession,
}

func runSession(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if len(cfg.BaseDirs) == 0 {
		return fmt.Errorf(
			"base_dirs not set in config\nRun: twine config init",
		)
	}

	target := ""
	if len(args) >= 1 {
		target = args[0]
	}

	if target == "" {
		chosen, err := selectSession(cfg)
		if errors.Is(err, ui.ErrCancelled) {
			return nil
		}
		if err != nil {
			return err
		}
		target = chosen
	}

	// Try session names with and without prefix.
	names := []string{cfg.SessionPrefix + target, target}
	for _, name := range names {
		if tmux.HasSession(name) {
			return tmux.AttachOrSwitch(name)
		}
	}

	// Search repos in base dirs.
	for _, name := range names {
		for _, base := range cfg.BaseDirs {
			// Worktree inside bare repo: base/name.git/name
			wtPath := filepath.Join(base, name+".git", name)
			regPath := filepath.Join(base, name)

			repoPath := ""
			if isDir(wtPath) {
				repoPath = wtPath
			} else if isDir(regPath) {
				repoPath = regPath
			}

			if repoPath == "" {
				continue
			}

			if cfg.ShouldUseTmuxinator() {
				if err := runTmuxinator(name, repoPath, cfg); err != nil {
					return err
				}
			} else {
				if err := tmux.NewSession(name, repoPath); err != nil {
					return fmt.Errorf(
						"failed to create session: %w", err,
					)
				}
			}
			return tmux.AttachOrSwitch(name)
		}
	}

	// No repo found – create a basic session.
	fallback := names[len(names)-1]
	if !tmux.HasSession(fallback) {
		if err := tmux.NewSession(fallback, "."); err != nil {
			return fmt.Errorf("failed to create session: %w", err)
		}
	}
	return tmux.AttachOrSwitch(fallback)
}

func selectSession(cfg *config.Config) (string, error) {
	sessions, _ := tmux.ListSessions()
	allRepos, _ := repo.FindAll(cfg.BaseDirs)

	sessionSet := map[string]bool{}
	var items []ui.Item
	for _, s := range sessions {
		sessionSet[s] = true
		items = append(items, ui.Item{
			Title: s, Value: s, Active: true,
		})
	}
	for _, r := range allRepos {
		if !sessionSet[r.Name] {
			items = append(items, ui.Item{
				Title: r.Name, Value: r.Name, Active: false,
			})
		}
	}

	chosen, err := ui.Select(items, "Select session or repo:")
	if err != nil {
		return "", err
	}
	return chosen.Value, nil
}

// isDir returns true when path is an existing directory.
func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// runTmuxinator launches tmuxinator for a named session rooted at dir.
func runTmuxinator(name, dir string, cfg *config.Config) error {
	layout := "dev"
	if cfg != nil && cfg.TmuxinatorLayout != "" {
		layout = cfg.TmuxinatorLayout
	}
	c := exec.Command("tmuxinator", "start", "-n", name, layout, "false")
	c.Dir = dir
	return c.Run()
}