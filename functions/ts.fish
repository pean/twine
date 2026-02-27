function ts --description 'Start tmuxinator for current repository'
    # Show help
    if test "$argv[1]" = "--help"; or test "$argv[1]" = "-h"
        echo "ts - Tmuxinator launcher"
        echo ""
        echo "Usage: ts [args]"
        echo "       twine ts [args]"
        echo ""
        echo "Description:"
        echo "  Starts tmuxinator with TWINE_TMUXINATOR_LAYOUT for the current repo"
        echo ""
        echo "Examples:"
        echo "  cd ~/src/my-project"
        echo "  ts                  # Starts tmuxinator session"
        echo "  ts false            # Start detached"
        echo "  twine ts            # Same thing"
        echo ""
        echo "Requirements:"
        echo "  - Must be run from within a git repository"
        echo "  - Requires tmuxinator to be installed"
        echo "  - Uses TWINE_TMUXINATOR_LAYOUT config"
        echo ""
        echo "See also: tw, t, ta"
        return 0
    end

    if not command -v tmuxinator >/dev/null
        echo "Error: tmuxinator not installed"
        return 1
    end

    if git rev-parse --git-dir >/dev/null 2>&1
        set reponame (basename (git rev-parse --show-toplevel) | sed 's/^\.//')
        echo "Starting tmuxinator for $reponame in $pwd"
        tmuxinator start -n $reponame dev $argv
    else
        echo "Not a repo"
    end
end
