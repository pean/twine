# 🌿 Twine

> Intertwined branches and sessions - Fish shell plugin for git worktree + tmux session management

Twine helps you seamlessly manage git worktrees and tmux sessions, making it easy to work on multiple branches simultaneously with dedicated terminal environments for each.

## What You Can Do

### Work on Multiple Branches Simultaneously

**`tw` - Your main command for branch-based workflows**

Jump between branches instantly, each in their own tmux session. Perfect for when
you're juggling feature work, code review, and hotfixes.

```fish
# Pick from all your projects and branches interactively
tw

# Jump to a specific branch (creates worktree if needed)
tw my-project feature/login

# Start work on a remote branch you haven't checked out yet
tw my-project origin/hotfix/critical-bug
```

**Real scenario:** You're coding in `feature/payment` when a critical bug is
reported. Run `tw my-project hotfix/security-fix` to instantly switch to a fresh
environment. Your payment feature work stays running with its dev server and
tests untouched.

### Quick Project Navigation

**`t` - Fast switching between your projects**

Switch to any project's main session without specifying a branch. Great for when
you just need to jump into a project.

```fish
# Interactive selection of all your projects
t

# Jump directly to a project
t my-project
```

**Real scenario:** Need to quickly check something in your docs repo or run a
script? `t docs` gets you there instantly without the branch selection step.

### Set Up New Projects

**`init` - Start fresh with the worktree pattern**

Setting up a new repository for the first time? Get it configured correctly from
the start.

```fish
# Clone and set up for multi-branch work
twine init my-project git@github.com:user/my-project.git

# Start with a specific branch
twine init my-project git@github.com:user/my-project.git develop
```

**Real scenario:** Starting a new job or project. Use `init` to clone the repo
as bare with worktree support, so you're ready to work on multiple features from
day one.

### Migrate Existing Projects

**`convert` - Upgrade regular repos to use worktrees**

Already have a repo? Convert it to the bare + worktree pattern to unlock
parallel branch workflows.

```fish
# Convert your existing repo
twine convert my-project
```

**Real scenario:** You've been using a regular repo but now need to work on a
feature branch while keeping main available for hotfixes. Convert it once and
enjoy the worktree workflow.

### Launch Custom Layouts

**`ts` - Start tmuxinator sessions**

If you use tmuxinator for custom window/pane layouts, launch them in your
current repo.

```fish
# From within any git repository
cd ~/src/my-project/feature-branch
ts
```

**Real scenario:** You have a tmuxinator layout with editor, tests, and logs
panes. Jump into any worktree and run `ts` to get your preferred setup instantly.

## Directory Structure Explained

Twine uses git's worktree feature to let you work on multiple branches at the same
time. Here's how your repositories get organized:

### Bare Repository Pattern (Recommended)

```
~/src/work/
├── my-project.git/           # Bare repository (no working files)
│   ├── HEAD, config, objects/, refs/  # Git internals
│   ├── main/                 # Worktree: main branch
│   │   ├── src/
│   │   ├── package.json
│   │   └── ...               # Full working directory
│   ├── develop/              # Worktree: develop branch
│   │   ├── src/
│   │   └── ...
│   └── feature/
│       └── new-login/        # Worktree: feature/new-login branch
│           ├── src/
│           └── ...
```

**How it works:**
- `my-project.git/` is the bare repository (just git data, no files to edit)
- Each subdirectory (`main/`, `develop/`, etc.) is a complete working directory
- Each worktree is checked out to a different branch
- All worktrees share the same git history and objects (efficient!)
- Each worktree can have its own running processes, uncommitted changes, etc.

**Tmux sessions match the structure:**
- `my-project/main` - tmux session for main worktree
- `my-project/develop` - tmux session for develop worktree
- `my-project/feature/new-login` - tmux session for feature worktree

### Regular Repository (Also Supported)

```
~/src/work/
└── my-project/               # Regular git repository
    ├── .git/                 # Git data
    ├── src/
    └── package.json
```

Twine works with regular repos too, but you can only work on one branch at a
time. When you try to work with multiple branches, twine will offer to convert
it to the bare + worktree pattern.

### Configuration

Tell twine where to look for repositories:

```fish
set -gx TWINE_BASE_DIRS ~/src/work ~/src/personal ~/projects
```

Twine will search all these directories for repositories when you run `tw` or `t`.

## Features

- **Interactive selection**: Call `t` or `tw` without arguments for fzf-powered selection
- **Visual indicators**: Running sessions marked with ▶, available repos with 📁
- **Multi-directory support**: Search for repositories across multiple base directories
- **Auto-create worktrees**: Automatically create git worktrees from remote branches
- **Smart session management**: Create and switch between tmux sessions for each worktree
- **Optional tmuxinator integration**: Use custom layouts or fall back to basic tmux sessions
- **Tab completion**: Intelligent completion prioritizing active sessions over repos

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

## Command Reference

### Quick Reference

```fish
tw [repo] [branch]     # Work on a specific branch (main command)
t [repo]               # Switch to a project (no branch selection)
ts                     # Launch tmuxinator in current repo
twine init <name> <url> [branch]  # Set up new repo
twine convert <repo>   # Convert existing repo to worktrees
```

**Getting help:**
```fish
twine --help              # Overview of all actions
tw --help                 # Help for worktree command
```

### `tw` (worktree)

Branch-focused workflow - creates worktrees and switches to branch-specific sessions.

```fish
tw [repo] [branch]
twine worktree [repo] [branch]  # Verbose form
```

- Without args: Interactive fzf selection of all repos and branches
- With repo only: Interactive branch selection for that repo
- With repo and branch: Direct switch (creates worktree if needed)
- Fetches latest remote branches before selection
- Auto-creates worktrees from remote branches
- Visual indicators: ▶ for active, 📁 for available

### `t` (session)

Project-focused workflow - switches to main repo session without branch selection.

```fish
t [repo]
twine session [repo]  # Verbose form
```

- Without args: Interactive fzf selection of all repos
- With repo: Direct switch to that project
- Faster than `tw` when you don't need branch selection
- Creates session if repo exists but session doesn't
- Prioritizes running sessions in completions

### `ts` (start)

Launches tmuxinator with your configured layout in the current directory.

```fish
ts [args]
twine start [args]  # Verbose form
```

- Must be run from within a git repository
- Requires tmuxinator installation
- Uses layout specified in `TWINE_TMUXINATOR_LAYOUT`

### `init`

Sets up a new repository with bare + worktree structure from a remote URL.

```fish
twine init <repo-name> <git-url> [branch]
```

Steps performed:
1. Clones repository as bare (`repo-name.git`)
2. Creates worktree for initial branch (default or specified)
3. Repository is ready to use with `tw`

### `convert`

Converts an existing regular repository to bare + worktree structure.

```fish
twine convert <repo>
```

Steps performed:
1. Clones regular repo as bare (`repo.git`)
2. Creates worktree for current branch
3. Optionally removes old regular repo directory

The `tw` command will also offer to convert when it detects a regular repo.

### `attach`

Low-level tmux session attach/switch (rarely needed directly).

```fish
twine attach <session-name>
```

Used internally by `tw` and `t`. Use those commands instead for better
integration with git worktrees.

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

## Tips & Workflow Suggestions

- **Start interactive**: Call `t` or `tw` without arguments to see all options with visual indicators
- **Use tab completion**: Running sessions appear first (▶), then available repos (📁)
- **Choose the right command**: Use `tw` when working with branches, `t` for quick project navigation
- **Let it auto-create**: When you `tw repo branch`, worktrees are created automatically if needed
- **Multiple base dirs**: Configure `TWINE_BASE_DIRS` to search across work/personal/hobby projects
- **Convert when ready**: Regular repos work fine, but `convert` unlocks parallel branch workflows

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

**Commit Message Format:**

This project uses [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>: <description>

[optional body]
```

Types:
- `feat:` - New feature (bumps MINOR version)
- `fix:` - Bug fix (bumps PATCH version)
- `docs:` - Documentation changes
- `test:` - Test changes
- `chore:` - Maintenance tasks
- `refactor:` - Code refactoring
- `BREAKING CHANGE:` - Breaking changes (bumps MAJOR version)

Examples:
```bash
git commit -m "feat: add interactive branch selection"
git commit -m "fix: resolve session switching issue"
git commit -m "docs: update installation instructions"
```

### Releases

Releases are managed automatically using [release-please](https://github.com/googleapis/release-please).

**CHANGELOG:** Maintained automatically based on conventional commits.

**Creating a release:**

1. Ensure all changes are committed and pushed to master
2. Trigger the release workflow:
   ```bash
   gh workflow run release-please.yml
   ```
3. Review and merge the automated Release PR
4. Git tag and GitHub release are created automatically

**Version scheme:** [Semantic Versioning](https://semver.org/) (MAJOR.MINOR.PATCH)

## Uninstall

```fish
fisher remove pean/twine
```

Remove configuration from `~/.config/fish/config.fish`.

## License

MIT
