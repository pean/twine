# 🌿 Twine

> Git worktree + tmux session management

```
Twine manages git worktrees and tmux sessions together.

Usage:
  twine [command]

Available Commands:
  agents      List and switch to AI coding agents in tmux
  install     Set up config, aliases, and completions
  kill        Kill tmux sessions with optional worktree removal
  prune       Clean up gone branches, worktrees, and sessions
  session     Switch to a tmux session (faster, no branch selection)
  worktree    Create worktree and switch tmux session

Flags:
  -h, --help   help for twine

Use "twine [command] --help" for more information about a command.
```

Twine makes it easy to work on multiple branches simultaneously by pairing git worktrees with dedicated tmux sessions — one session per branch, no context switching.

## Requirements

- [tmux](https://github.com/tmux/tmux)
- [git](https://git-scm.com/) (with worktree support, git 2.5+)

**Optional:**
- [tmuxinator](https://github.com/tmuxinator/tmuxinator) — for custom session layouts

## Installation

### Go install

```sh
go install github.com/pean/twine/cmd/twine@latest
```

Make sure `~/go/bin` is in your `PATH` (e.g. add `fish_add_path ~/go/bin` to `~/.config/fish/config.fish`).

### From source

```sh
git clone https://github.com/pean/twine
cd twine
make install
```

### Shell setup

Run the interactive installer to set up aliases and completions for your shell (fish, bash, or zsh):

```sh
twine install
```

## Configuration

Twine reads `~/.config/twine/config.toml`. The `twine install` command will create it for you, or write it manually:

```toml
# ~/.config/twine/config.toml

# Directories to search for repositories (required).
base_dirs = [
  "~/src/work",
  "~/src/personal",
]

# Prefix prepended to all tmux session names (optional).
# session_prefix = ""

# Tmuxinator integration (optional).
# use_tmuxinator = "auto"  # auto | true | false
# tmuxinator_layout = "dev"
```

`use_tmuxinator = "auto"` (the default) uses tmuxinator when it is installed
and `tmuxinator_layout` is set.

## Commands

### `twine worktree` (`tw`)

The main command. Creates worktrees and switches to their tmux sessions.

```sh
tw                              # interactive repo + branch selection
tw my-project                   # interactive branch selection
tw my-project main              # switch to main (creates worktree if needed)
tw my-project feature/x         # check out from remote
tw my-project feature/x -c      # create new branch from main/master
tw my-project feature/x -c -f develop  # create from develop
```

When the repository is not found, you are prompted for a git URL or
`org/repo` shorthand (uses `gh repo clone`), and it is cloned as bare
automatically.

When a regular (non-bare) repo is found, you are offered to convert it to the
bare + worktree layout.

Flags: `-c / --create`, `-f / --from <branch>`

### `twine session` (`t`)

Faster session switcher — no branch selection, just finds the repo session and attaches.

```sh
t my-project
```

### `twine kill` (`tk`)

Kill tmux sessions, optionally removing their worktrees.

```sh
tk                              # interactive multi-select, prompts for worktree removal
tk my-project/feature/x         # kill specific session
tk my-project/feature/x -w      # also remove the worktree without prompting
```

### `twine prune`

Remove branches whose remote has been deleted: kills sessions, removes worktrees, deletes local branches, and runs `git worktree prune` to clean up stale refs.

```sh
twine prune                     # all repos
twine prune my-project          # specific repo
twine prune --dry-run           # preview
```

### `twine agents`

Dashboard for AI coding agents running in tmux sessions. Detects Claude Code and OpenCode, shows their state, and lets you switch to them.

```sh
twine agents
```

Bind it to a tmux key for quick access — add to `~/.tmux.conf`:

```tmux
bind-key a display-popup -E -w 90% -h 90% "twine agents"
```

## How it works

### Repository layout

Twine works best with the bare + worktree pattern:

```
~/src/work/
└── my-project.git/     ← bare repository
    ├── main/           ← worktree for main
    ├── develop/        ← worktree for develop
    └── feature/xyz/    ← worktree for feature/xyz
```

Each worktree is a full working directory sharing the same git objects. You can have all branches checked out at once with no switching overhead.

### Session naming

Sessions are named `<prefix><repo>/<branch>`:

```
my-project/main
my-project/feature/xyz
```

### Worktree creation

When you ask for a branch that has no local worktree yet, twine handles three cases:

1. **Branch exists on remote and locally** — adds worktree and sets upstream tracking
2. **Branch exists on remote only** — creates a new tracking branch and worktree
3. **Branch doesn't exist anywhere + `--create`** — creates a new branch from main/master (or `--from`)

## Development

```sh
make build       # build to bin/twine
make test        # go test ./...
make install     # go install ./cmd/twine
make completions # regenerate completion files
```

## License

MIT
