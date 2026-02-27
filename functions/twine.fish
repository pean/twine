function twine --description 'Git worktree + tmux session management'
    set -l action $argv[1]

    # Show help if no action or --help
    if test (count $argv) -eq 0; or test "$action" = "--help"; or test "$action" = "-h"
        __twine_help
        return 0
    end

    # Route to subcommands
    switch $action
        case worktree tw
            tw $argv[2..]
        case session t
            t $argv[2..]
        case attach
            __twine_attach $argv[2..]
        case start ts
            ts $argv[2..]
        case init
            __twine_init $argv[2..]
        case convert
            __twine_convert $argv[2..]
        case '*'
            echo "Error: Unknown action '$action'"
            echo ""
            echo "Run 'twine --help' for usage"
            return 1
    end
end

function __twine_help --description 'Show twine help'
    echo "Twine - Git worktree + tmux session management"
    echo ""
    echo "Usage: twine [action] [options]"
    echo ""
    echo "Actions:"
    echo "  worktree    Manage worktrees and sessions (shortcut: tw)"
    echo "  session     Switch to repo session (shortcut: t)"
    echo "  start       Start tmuxinator for repo (shortcut: ts)"
    echo "  attach      Attach to tmux session"
    echo "  init        Initialize new bare repo"
    echo "  convert     Convert regular repo to bare"
    echo ""
    echo "Shortcuts:"
    echo "  tw          Same as 'twine worktree'"
    echo "  t           Same as 'twine session'"
    echo "  ts          Same as 'twine start'"
    echo ""
    echo "Examples:"
    echo "  twine worktree my-project        # Interactive worktree selection"
    echo "  twine worktree my-proj feature/x # Switch to specific worktree"
    echo "  twine session my-project         # Quick session switch"
    echo "  twine attach my-session          # Attach to session"
    echo "  twine start                      # Start tmuxinator in current repo"
    echo "  twine init my-proj <url>         # Clone as bare repo"
    echo "  twine convert my-proj            # Convert to bare repo"
    echo ""
    echo "  tw my-project                    # Shortcut for 'twine worktree'"
    echo "  t my-project                     # Shortcut for 'twine session'"
    echo ""
    echo "Help:"
    echo "  twine --help              # This message"
    echo "  tw --help                 # Detailed help for worktree command"
    echo "  twine worktree --help     # Same thing"
    echo ""
    echo "Configuration:"
    echo "  TWINE_BASE_DIRS          Base directories (required)"
    echo "  TWINE_TMUXINATOR_LAYOUT  Tmuxinator layout (optional)"
    echo "  TWINE_SESSION_PREFIX     Session name prefix (optional)"
    echo "  TWINE_USE_TMUXINATOR     auto|true|false (optional)"
    echo ""
    echo "Documentation: https://github.com/pean/twine"
end
