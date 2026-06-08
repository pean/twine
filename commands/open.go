package commands

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/pean/twine/internal/config"
	"github.com/pean/twine/internal/tmux"
)

// matchWorktree reports whether the "repo/branch" candidate matches pattern.
// Patterns wrapped in /…/ are treated as regular expressions matched against
// the full candidate string.  Bare strings are matched as case-insensitive
// substrings.
func matchWorktree(pattern, candidate string) (bool, error) {
	if strings.HasPrefix(pattern, "/") && strings.HasSuffix(pattern, "/") && len(pattern) > 1 {
		re, err := regexp.Compile(pattern[1 : len(pattern)-1])
		if err != nil {
			return false, fmt.Errorf("invalid regex %q: %w", pattern, err)
		}
		return re.MatchString(candidate), nil
	}
	return strings.Contains(
		strings.ToLower(candidate),
		strings.ToLower(pattern),
	), nil
}

var openCmd = &cobra.Command{
	Use:     "open <pattern>",
	Aliases: []string{"to"},
	Short:   "Open all worktrees matching a glob pattern",
	Long: `Find all existing worktrees across all repos whose "repo/branch" path
matches the given pattern, then launch a tmux session for each.

Pattern forms:
  /regex/     treated as a regular expression matched against "repo/branch"
  string      case-insensitive substring match against "repo/branch"

Examples:
  twine open /bankid/                   # all worktrees containing "bankid"
  twine open /web-app.*bankid/          # bankid branches only in web-app
  twine open /DRE-346$/                 # branches ending in DRE-346
  twine open bankid                     # same as /bankid/ but case-insensitive`,
	Args: cobra.ExactArgs(1),
	RunE: runOpen,
}

type openMatch struct {
	repoName     string
	branch       string
	worktreePath string
	sessionName  string
}

func runOpen(cmd *cobra.Command, args []string) error {
	pattern := args[0]

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	entries, _, err := allWorktreesWithSessions()
	if err != nil {
		return err
	}

	var matches []openMatch
	for _, e := range entries {
		candidate := e.Repo.Name + "/" + e.Branch
		matched, err := matchWorktree(pattern, candidate)
		if err != nil {
			return err
		}
		if matched {
			matches = append(matches, openMatch{
				repoName:     e.Repo.Name,
				branch:       e.Branch,
				worktreePath: e.Path,
				sessionName:  cfg.SessionPrefix + e.Repo.Name + "/" + e.Branch,
			})
		}
	}

	if len(matches) == 0 {
		fmt.Printf("No worktrees match %q\n", pattern)
		return nil
	}

	fmt.Printf("Found %d worktree(s):\n", len(matches))
	for _, m := range matches {
		fmt.Printf("  %s/%s\n", m.repoName, m.branch)
	}
	fmt.Printf("Launch %d session(s)? [y/N] ", len(matches))

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	if strings.ToLower(strings.TrimSpace(scanner.Text())) != "y" {
		return nil
	}

	var lastSession string
	for _, m := range matches {
		if !tmux.HasSession(m.sessionName) {
			fmt.Printf("Creating tmux session: %s\n", m.sessionName)
			var sessionErr error
			if cfg.ShouldUseTmuxinator() {
				sessionErr = runTmuxinator(m.sessionName, m.worktreePath, cfg)
			} else {
				sessionErr = tmux.NewSession(m.sessionName, m.worktreePath)
			}
			if sessionErr != nil {
				return fmt.Errorf("failed to create session %s: %w", m.sessionName, sessionErr)
			}
		}
		lastSession = m.sessionName
	}

	return tmux.AttachOrSwitch(lastSession)
}
