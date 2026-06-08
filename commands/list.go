package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/pean/twine/internal/config"
	"github.com/pean/twine/internal/repo"
	"github.com/pean/twine/internal/tmux"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all worktrees across all repos",
	RunE:  runList,
}

func runList(cmd *cobra.Command, args []string) error {
	entries, sessions, err := allWorktreesWithSessions()
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		fmt.Println("No worktrees found.")
		return nil
	}
	for _, e := range entries {
		name := e.Repo.Name + "/" + e.Branch
		if sessions[name] {
			fmt.Printf("▶ %s\n", name)
		} else {
			fmt.Printf("  %s\n", name)
		}
	}
	return nil
}

// allWorktreesWithSessions returns all worktrees and a set of active session names.
// The session key is "repoName/branch" (without the configured prefix).
func allWorktreesWithSessions() ([]repo.WorktreeEntry, map[string]bool, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, nil, err
	}
	if len(cfg.BaseDirs) == 0 {
		return nil, nil, fmt.Errorf("base_dirs not set in config\nRun: twine config init")
	}

	entries, err := repo.ListAllWorktrees(cfg.BaseDirs)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	sessionNames, _ := tmux.ListSessions()
	sessions := make(map[string]bool, len(sessionNames))
	prefix := cfg.SessionPrefix
	for _, s := range sessionNames {
		name := s
		if prefix != "" {
			name = stripPrefix(s, prefix)
		}
		sessions[name] = true
	}

	return entries, sessions, nil
}

func stripPrefix(s, prefix string) string {
	if len(s) > len(prefix) && s[:len(prefix)] == prefix {
		return s[len(prefix):]
	}
	return s
}
