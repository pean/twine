# 🌿 Twine

> Git worktree + tmux session management

```
Twine manages git worktrees and tmux sessions together.

Usage:
  twine [command]

Available Commands:
  agents      List and switch to AI coding agents in your sessions
  install     Set up config, aliases, and completions
  kill        Kill tmux sessions with optional worktree removal
  prune       Clean up gone branches, worktrees, and sessions
  session     Switch to a tmux session (faster, no branch selection)
  watch       Notify when an AI coding agent needs input or finishes
  worktree    Create worktree and switch tmux session

Flags:
  -h, --help   help for twine

Use "twine [command] --help" for more information about a command.
```

Twine makes it easy to work on multiple branches simultaneously by pairing git worktrees with dedicated tmux sessions — one session per branch, no context switching.

## Requirements

- [tmux](https://github.com/tmux/tmux) or [rex](https://www.superlogical.com/rex/docs)
- [git](https://git-scm.com/) (with worktree support, git 2.5+)

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

# Prefix prepended to all session names (optional).
# session_prefix = ""

# Terminal multiplexer for new sessions (optional): auto | tmux | rex.
# multiplexer = "auto"

# Layout new sessions start with (optional), defined under [layouts].
# layout = "dev"

# Per-repository overrides (optional). The first matching entry wins.
# [[repos]]
# match = "work-*"         # repo name or glob
# multiplexer = "rex"
# layout = "dev"
```

### Layouts

A layout is the windows and panes a new session starts with. It works the same
in tmux and rex.

```toml
layout = "dev"

[layouts.dev]
windows = [
  { name = "editor", panes = [
      { command = "nvim" },
      { split = "left", size = 40, command = "bin/rails console" },
      { split = "below", size = 66, command = "git diff" },
      { split = "below", size = 50, command = "git status" },
  ]},
  { name = "server", panes = [{ command = "ls" }] },
]
```

Each pane after the first splits the pane before it. `split` is `left`, `right`,
`above`, or `below` (default `right`), and `size` is the new pane's share of the
pane it splits, in percent (default 50). `command` is typed into the pane's shell,
which stays open after it. The first pane of the first window is selected.

Set `layout` at the top level for all repos, or per repo in `[[repos]]`. Sessions
without a layout get a single shell.

Tmuxinator is no longer supported. `use_tmuxinator` and `tmuxinator_layout` now
fail with an error that points here. A tmuxinator `on_project_start` hook has no
layout equivalent; put the command in a pane, e.g. `command = "mise install; nvim"`.

### Choosing a multiplexer

Twine drives a terminal multiplexer through a backend: tmux or
[rex](https://www.superlogical.com/rex/docs).

New sessions go to the first of these that is set:

1. `--mux <name>` on the command line. This is strict: only that
   multiplexer is used, also when looking for existing sessions.
2. `TWINE_MUX` in the environment, e.g. `set -gx TWINE_MUX tmux` in fish to
   pick a default for one shell.
3. The first matching `[[repos]]` entry in the config.
4. The top-level `multiplexer` setting.

`auto` (the default) picks the multiplexer twine is running inside, otherwise
the first installed one (tmux, then rex).

Without `--mux`, existing sessions are found in every installed multiplexer,
so `ts my-project` switches to `my-project` wherever it runs, and `list`,
`kill`, `prune`, and `agents` cover all of them. When more than one
multiplexer is installed, session lists show which one runs each session.

#### Rex

Sessions are labelled `<prefix><repo>/<branch>`, as in tmux. Rex has no terminal
attach, so switching asks the Rex app to show the session. The app only accepts that
when **Remote Control** is on in its settings (Rex Server). Without it, twine still
creates the session and prints the setting to turn on.

`twine agents` reads each agent's own status from Rex (Claude Code reports
`working`, `blocked`, and so on), so it does not read the screen unless an agent
reports nothing. `rex` must be on `PATH`; Rex adds it inside its own terminals.

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

### `twine session` (`ts`)

Faster session switcher — no branch selection, just finds the repo session and attaches.

```sh
ts my-project                   # switch to repo session
ts .                            # create/attach session for current directory
ts /path/to/dir                 # create/attach session for any directory
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

Dashboard for AI coding agents running in tmux or rex sessions. Detects Claude Code and OpenCode, shows their state, and lets you switch to them.

```sh
twine agents
```

Bind it to a tmux key for quick access — add to `~/.tmux.conf`:

```tmux
bind-key a display-popup -E -w 90% -h 90% "twine agents"
```

In Rex, `twine install` writes `twine_agents`, an action that opens the dashboard
in a floating pane. It does not edit your `init.lua`; add the two lines it prints:

```lua
require("twine")
rex.bind("ctrl+b>a", "twine_agents")
```

### `twine watch`

Watches the agents in every multiplexer and shows a desktop notification when one
needs input or finishes. Runs until interrupted, so start it in a spare pane.

```sh
twine watch                     # check every 3s
twine watch --interval 10s
```

A new state has to show up on two checks in a row, so a screen that flickers does
not notify. Notifications use `osascript` on macOS and `notify-send` on Linux.

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

## Release

Releases are automated via [release-please](https://github.com/googleapis/release-please).
The version is derived from conventional commit messages:
`feat:` → minor, `fix:` → patch, `BREAKING CHANGE` footer → major.

**One-time repo setup** — release-please needs permission to open PRs:
Settings → Actions → General → Workflow permissions →
enable **"Allow GitHub Actions to create and approve pull requests"**.

**To cut a release:**

1. Ensure all changes are on `master` with conventional commit messages.
   For a major version bump, add an empty breaking-change commit:
   ```sh
   git commit --allow-empty -m "feat!: <summary>

   BREAKING CHANGE: <description of what broke>"
   git push
   ```

2. Trigger the workflow:
   ```sh
   gh workflow run release-please.yml
   ```

3. release-please opens a PR with updated `CHANGELOG.md` and version bump.
   Merge it — that creates the git tag and GitHub release, and triggers
   cross-compiled binary uploads for linux/darwin × amd64/arm64.

**Manual release (fallback)** — if you need to tag without release-please:
```sh
gh release create v2.0.0 --title "v2.0.0" --notes-file CHANGELOG.md --draft
```
Note: the binary build job only runs when release-please creates the release.
To also upload binaries, run the build manually and attach them:
```sh
GOOS=darwin GOARCH=arm64 go build -o twine-darwin-arm64 ./cmd/twine
gh release upload v2.0.0 twine-darwin-arm64
```

## License

MIT
