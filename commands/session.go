package commands

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/spf13/cobra"

	"github.com/pean/twine/internal/config"
	"github.com/pean/twine/internal/repo"
	"github.com/pean/twine/internal/tmux"
	"github.com/pean/twine/internal/ui"
)

var sessionCmd = &cobra.Command{
	Use:     "session [repo-or-session]",
	Aliases: []string{"ts"},
	Short:   "Switch to a tmux session (faster, no branch selection)",
	Long: `Switch to an existing tmux session or create one for a repo.

Unlike the worktree command, no branch selection is shown.
Interactive selection is offered when called without arguments.

Pass "." or an absolute path to create a session rooted at that directory,
named after its basename.`,
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

	// If target is a path (starts with . or /), resolve to an absolute dir
	// and use the basename as the session name.
	if target == "." || filepath.IsAbs(target) {
		dir, err := filepath.Abs(target)
		if err != nil {
			return fmt.Errorf("cannot resolve path: %w", err)
		}
		name := cfg.SessionPrefix + strings.TrimLeft(filepath.Base(dir), ".")
		if !tmux.HasSession(name) {
			if cfg.ShouldUseTmuxinator() {
				if err := runTmuxinator(name, dir, cfg); err != nil {
					return err
				}
			} else {
				if err := tmux.NewSession(name, dir); err != nil &&
					!strings.Contains(err.Error(), "duplicate session") {
					return fmt.Errorf("failed to create session: %w", err)
				}
			}
		}
		return tmux.AttachOrSwitch(name)
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
	var (
		sessions []string
		allRepos []*repo.Repo
		wg       sync.WaitGroup
	)
	wg.Add(2)
	go func() { defer wg.Done(); sessions, _ = tmux.ListSessions() }()
	go func() { defer wg.Done(); allRepos, _ = repo.FindAll(cfg.BaseDirs) }()
	wg.Wait()

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