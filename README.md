# 🌿 Twine

> Intertwined branches and sessions - Fish shell plugin for git worktree + tmux session management

Twine helps you seamlessly manage git worktrees and tmux sessions, making it easy to work on multiple branches simultaneously with dedicated terminal environments for each.

## Features

- **Multi-directory support**: Search for repositories across multiple base directories
- **Auto-create worktrees**: Automatically create git worktrees from remote branches
- **Interactive branch selection**: FZF-powered fuzzy finder for branches and worktrees
- **Smart session management**: Create and switch between tmux sessions for each worktree
- **Optional tmuxinator integration**: Use custom layouts or fall back to basic tmux sessions
- **Tab completion**: Intelligent completion for repos, branches, and sessions

## Requirements

**Required:**
- [Fish shell](https://fishshell.com/) 3.0+
- [tmux](https://github.com/tmux/tmux) - Terminal multiplexer
- [git](https://git-scm.com/) - Version control (with worktree support)
- [fzf](https://github.com/junegunn/fzf) - Fuzzy finder for interactive selection
- [Fisher](https://github.com/jorgebucaran/fisher) - Fish plugin manager

**Optional:**
- [tmuxinator](https://github.com/tmuxinator/tmuxinator) - For custom tmux session layouts

## Installation

Install via Fisher:

```fish
fisher install pean/twine
```

## Configuration

### Required

Add to `~/.config/fish/config.fish`:

```fish
# Base directories to search for repositories
set -gx TWINE_BASE_DIRS ~/src/work ~/src/personal ~/projects
```

### Optional

```fish
# Tmuxinator layout to use (if tmuxinator is installed)
set -gx TWINE_TMUXINATOR_LAYOUT dev

# Session name prefix (e.g., "work-" for "work-repo/branch")
set -gx TWINE_SESSION_PREFIX ""

# Control tmuxinator usage: auto (default), true, or false
set -gx TWINE_USE_TMUXINATOR auto
```

**Tmuxinator behavior:**
- `auto` (default): Use tmuxinator if installed and `TWINE_TMUXINATOR_LAYOUT` is set
- `true`: Always use tmuxinator (error if not installed)
- `false`: Never use tmuxinator, create basic tmux sessions

## Commands

Twine provides both a unified `twine` command and convenient shortcuts.

### Usage Patterns

**Unified command with verbose actions:**
```fish
twine worktree <repo> [branch]    # Manage worktrees
twine session <repo>               # Switch sessions
twine start                        # Start tmuxinator
twine attach <session>             # Attach to tmux (verbose only)
twine init <name> <url>            # Initialize bare repo
twine convert <repo>               # Convert to bare
```

**Shortcut commands:**
```fish
tw <repo> [branch]     # Same as 'twine worktree'
t <repo>               # Same as 'twine session'
ts                     # Same as 'twine start'
```

**Getting help:**
```fish
twine --help              # Overview of all actions
tw --help                 # Help for worktree command
twine worktree --help     # Same thing
```

### Actions

### `worktree` - Worktree Manager

Main command for creating and switching to git worktree tmux sessions.

**Usage:**
```fish
twine worktree <repo> [branch]
tw <repo> [branch]              # Shortcut
```

**Examples:**
```fish
# Interactive branch selection with fzf
tw my-project

# Switch to specific branch (creates worktree if needed)
tw my-project feature/new-feature

# Works with bare repos (repo.git) or regular repos
tw my-project.git develop
```

**Features:**
- Auto-detects bare repos (`.git` suffix) or regular repos
- Fetches latest remote branches before selection
- Creates worktrees from remote branches automatically
- Switches to existing tmux session or creates new one

### `session` - Session Switcher

Quick session switcher for existing repos.

**Usage:**
```fish
twine session <repo>
t <repo>                # Shortcut
```

**Examples:**
```fish
# Switch to repo session (searches all base directories)
t my-project

# Creates session if repo exists but session doesn't
t another-repo
```

### `attach` - Tmux Attach

Basic tmux session attach/switch utility.

**Usage:**
```fish
twine attach <session-name>
```

**Examples:**
```fish
# Attach to session (creates if doesn't exist)
twine attach my-session

# Switch to session from within tmux
twine attach another-session
```

**Note:** This is an internal function used by `tw` and `t` commands. Most users won't need to call this directly.

### `start` - Tmuxinator Launcher

Start tmuxinator for the current git repository.

**Usage:**
```fish
twine start [args]
ts [args]               # Shortcut
```

**Example:**
```fish
# From within a git repository
cd ~/src/my-project
ts                      # Shortcut
twine start             # Verbose
```

**Note:** Requires tmuxinator to be installed.

### `init` - Initialize New Repository

Clone a repository as bare and create initial worktree.

**Usage:**
```fish
twine init <repo-name> <git-url> [branch]
```

**Examples:**
```fish
# Clone as bare with default branch
twine init my-project git@github.com:user/my-project.git

# Clone with specific branch
twine init my-project git@github.com:user/my-project.git develop
```

**What it does:**
1. Clones repository as bare (`my-project.git`)
2. Creates worktree for initial branch
3. Ready to use with `tw`

### `convert` - Convert Regular Repo

Convert an existing regular repository to bare + worktree setup.

**Usage:**
```fish
twine convert <repo>
```

**Examples:**
```fish
# Convert existing repo
twine convert my-project
```

**What it does:**
1. Clones regular repo as bare (`my-project.git`)
2. Creates worktree for current branch
3. Optionally removes old regular repo

**Note:** The `worktree` command will also offer to convert when it detects a regular repo.

## Tmuxinator Setup (Optional)

If you want to use custom tmux layouts, install tmuxinator and create a layout file:

1. Install tmuxinator:
   ```bash
   gem install tmuxinator
   ```

2. Create a layout (e.g., `~/.config/tmuxinator/dev.yml`):
   ```yaml
   name: <%= ENV["TMUX_SESSION_NAME"] || "dev" %>
   root: <%= ENV["PWD"] %>

   windows:
     - editor:
         layout: main-vertical
         panes:
           - nvim
           -
     - terminal:
   ```

3. Configure twine to use it:
   ```fish
   set -gx TWINE_TMUXINATOR_LAYOUT dev
   ```

See `templates/dev.yml.example` for a full example.

## How It Works

### Git Worktrees

Twine works best with bare git repositories using the worktree pattern:

```
~/src/work/
├── my-project.git/         # Bare repository
│   ├── main/               # Worktree for main branch
│   ├── develop/            # Worktree for develop branch
│   └── feature/xyz/        # Worktree for feature branch
```

**Benefits:**
- Work on multiple branches simultaneously
- No branch switching overhead
- Each worktree has its own working directory
- Shared git history and objects

### Tmux Sessions

Each worktree gets its own tmux session named `repo/branch`:

```
my-project/main
my-project/develop
my-project/feature/xyz
```

This allows you to:
- Keep separate terminal environments per branch
- Switch between branches without losing context
- Run different processes for each branch simultaneously

## Tips

- Use `tw` without a branch argument for interactive fzf selection
- Tab completion shows existing worktrees and remote branches
- Sessions are automatically created with your configured layout
- Works with both bare repos (`.git`) and regular repos
- Supports multiple base directories - repos are searched in order

## Development

### Running Tests

Twine includes a comprehensive test suite using [Fishtape](https://github.com/jorgebucaran/fishtape).

**Install Fishtape:**
```fish
fisher install jorgebucaran/fishtape
```

**Run tests:**
```fish
./test
```

**Test coverage includes:**
- Configuration validation
- Repository finding across multiple directories
- Worktree detection and creation logic
- Session management
- Tmuxinator integration
- Tab completions

Tests run automatically on GitHub Actions for every push and pull request.

### Contributing

Contributions welcome! Please:
1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## Uninstall

```fish
fisher remove pean/twine
```

Remove configuration from `~/.config/fish/config.fish`.

## License

MIT
