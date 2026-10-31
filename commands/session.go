package commands

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/spf13/cobra"

	"github.com/pean/twine/internal/config"
	"github.com/pean/twine/internal/git"
	"github.com/pean/twine/internal/mux"
	"github.com/pean/twine/internal/repo"
	"github.com/pean/twine/internal/ui"
)

var sessionBackground bool

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

func init() {
	sessionCmd.Flags().BoolVarP(
		&sessionBackground, "background", "b", false,
		"create the session without attaching or switching to it",
	)
}

func attachOrSwitchUnlessBackground(m mux.Multiplexer, name string) error {
	if sessionBackground {
		return nil
	}
	return m.AttachOrSwitch(name)
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

	m := multiplexer(cfg)

	target := ""
	if len(args) >= 1 {
		target = args[0]
	}

	if target == "" {
		chosen, err := selectSession(cfg, m)
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
		if !m.HasSession(name) {
			// Layouts only apply to git repos.
			useLayout := git.IsGitRepo(dir)
			err := m.NewSession(name, dir, mux.NewSessionOptions{
				UseLayout: useLayout,
			})
			if err != nil && !errors.Is(err, mux.ErrSessionExists) {
				return fmt.Errorf("failed to create session: %w", err)
			}
		}
		return attachOrSwitchUnlessBackground(m, name)
	}

	// Try session names with and without prefix.
	names := []string{cfg.SessionPrefix + target, target}
	for _, name := range names {
		if m.HasSession(name) {
			return attachOrSwitchUnlessBackground(m, name)
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

			err := m.NewSession(name, repoPath, mux.NewSessionOptions{
				UseLayout: true,
			})
			if err != nil {
				return fmt.Errorf("failed to create session: %w", err)
			}
			return attachOrSwitchUnlessBackground(m, name)
		}
	}

	// No repo found – create a basic session.
	fallback := names[len(names)-1]
	if !m.HasSession(fallback) {
		err := m.NewSession(fallback, ".", mux.NewSessionOptions{})
		if err != nil {
			return fmt.Errorf("failed to create session: %w", err)
		}
	}
	return attachOrSwitchUnlessBackground(m, fallback)
}

func selectSession(cfg *config.Config, m mux.Multiplexer) (string, error) {
	var (
		sessions []string
		allRepos []*repo.Repo
		wg       sync.WaitGroup
	)
	wg.Add(2)
	go func() { defer wg.Done(); sessions = sessionNames(m) }()
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
