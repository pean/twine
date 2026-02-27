function __twine_attach --description 'Internal: Attach or switch to tmux session'
    # Show help
    if test "$argv[1]" = "--help"; or test "$argv[1]" = "-h"
        echo "attach - Tmux attach/switch utility"
        echo ""
        echo "Usage: twine attach <session-name>"
        echo ""
        echo "Arguments:"
        echo "  session-name    Name of tmux session"
        echo ""
        echo "Examples:"
        echo "  twine attach my-session    # Attach or switch to session"
        echo "  twine attach work          # Creates session if doesn't exist"
        echo ""
        echo "Behavior:"
        echo "  - Creates session if it doesn't exist"
        echo "  - Switches to session if inside tmux (TMUX set)"
        echo "  - Attaches to session if outside tmux"
        echo ""
        echo "Note: This is an internal function used by tw and t commands."
        echo ""
        echo "See also: t, tw"
        return 0
    end

    if test (count $argv) -lt 1
        echo "Usage: twine attach <session-name>"
        echo "Run 'twine attach --help' for more information"
        return 1
    end

    if not tmux list-sessions | grep -q "^$argv:" 2>/dev/null
        echo "Creating new session $argv"
        tmux new-session -d -s $argv
    end

    if set -q TMUX
        echo "Switching to $argv"
        tmux switch-client -t $argv
    else
        echo "Attaching to $argv"
        tmux attach -t $argv
    end
end
