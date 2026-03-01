# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
