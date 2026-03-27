# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Standalone Go CLI binary replacing the Fish shell plugin, built with Cobra and
  Bubbletea for interactive TUI (no fzf dependency)
- Config file at `~/.config/twine/config.toml` replacing environment variables
- `twine install` command for interactive setup of config, aliases, and shell completions
- Shell completions for Fish, Bash, and Zsh (`completions/`)
- `-r/--remote` flag on `twine worktree` to fetch and list remote branches only
  (default now shows local branches/worktrees without a network call)
- Go unit tests for config loading, git helpers, repo discovery, and URL classification
- Cross-compiled binary releases for linux/darwin × amd64/arm64 via CI

### Changed

- `twine worktree` / `tw` and `twine session` / `t` now implemented in Go
- `twine kill` / `tk`, `twine prune`, and `twine agents` ported to Go
- Repo scan parallelized with goroutines; `ListSessions` and `FindAll` run concurrently
- CI switched from Fish/Fishtape to `go build` + `go test`
- `WorktreePathForBranch` now queries `git worktree list` instead of inferring from
  branch name, fixing branches with slashes and manually-deleted worktree directories

### Removed

- Fish-only commands: `attach`, `convert`, `init`, `start` (functionality folded into
  `worktree` or dropped)
- `install-aliases` command (merged into `twine install`)
- `config` subcommand (merged into `twine install`)
- Old Fish function implementations in `functions/t.fish`, `functions/tw.fish`,
  `functions/__tw_convert_to_bare.fish`, `functions/__twine_init.fish`
- Fish test files and `conf.d/twine.fish` dispatcher

## [1.0.0] - 2026-03-20

### Added

- Initial twine plugin implementation
- `t` command for quick tmux session switching
- `tw` command for worktree + tmux session management
- Interactive fzf selection when `t` is called without arguments
- Interactive fzf selection when `tw` is called without arguments
- Visual indicators in completions: ▶ for active sessions/worktrees, 📁 for available repos
- Tab completion for `t` command showing running sessions and available repos
- Tab completion for `tw` command with repo and branch support
- Auto-completion prioritizes running sessions at the top
- Symbol-based labels in all fzf selection menus
- Support for TWINE_BASE_DIRS configuration
- Support for TWINE_SESSION_PREFIX configuration
- Integration with tmuxinator when available
- Auto-detection of bare (.git) and regular repos
- Automatic worktree creation from remote branches

### Fixed

- Missing space between command and argument in plugin initialization
