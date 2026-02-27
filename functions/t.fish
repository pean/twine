function t --description 'Switch to tmux session for repo'
    # Show help
    if test "$argv[1]" = "--help"; or test "$argv[1]" = "-h"
        echo "t - Quick session switcher"
        echo ""
        echo "Usage: t <repo>"
        echo ""
        echo "Arguments:"
        echo "  repo    Repository or session name"
        echo ""
        echo "Examples:"
        echo "  t my-project     # Switch to existing session or create from repo"
        echo "  t session-name   # Switch to any tmux session"
        echo ""
        echo "Features:"
        echo "  - Searches for existing tmux sessions first"
        echo "  - Searches all TWINE_BASE_DIRS for repos"
        echo "  - Supports TWINE_SESSION_PREFIX"
        echo "  - Creates session with tmuxinator if available"
        echo "  - Falls back to basic tmux session"
        echo ""
        echo "Tab completion shows:"
        echo "  - Existing tmux sessions"
        echo "  - Available repositories"
        echo ""
        echo "See also: tw, ts"
        return 0
    end

    if test (count $argv) -lt 1
        echo "Usage: t <repo>"
        echo "Run 't --help' for more information"
        return 1
    end

    if not set -q TWINE_BASE_DIRS
        echo "Error: TWINE_BASE_DIRS not configured"
        echo "Add to ~/.config/fish/config.fish: set -gx TWINE_BASE_DIRS ~/src/repos"
        return 1
    end

    # Build list of session names to try (with and without prefix)
    set names $TWINE_SESSION_PREFIX$argv $argv

    # Check if any matching sessions exist
    if tmux has-session >/dev/null 2>&1
        for name in $names
            if tmux list-sessions | grep -q "^$name:" 2>/dev/null
                __twine_attach$name
                return
            end
        end
    end

    # Search for repository across all base directories
    for name in $names
        for base_dir in $TWINE_BASE_DIRS
            if test -d $base_dir/$name.git/$name
                # Found worktree in bare repo
                set repo_path $base_dir/$name.git/$name
            else if test -d $base_dir/$name
                # Found regular repo or worktree
                set repo_path $base_dir/$name
            else
                continue
            end

            # Create session for this repo
            if __twine_use_tmuxinator
                fish -c "cd $repo_path ; ts false"
            else
                tmux new-session -d -s $name -c $repo_path
            end
            __twine_attach$name
            return
        end
    end

    # No repo found, create basic session with the provided name
    __twine_attach$names[2]
end
