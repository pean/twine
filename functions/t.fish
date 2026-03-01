function t --description 'Switch to tmux session for repo'
    # Show help
    if test "$argv[1]" = "--help"; or test "$argv[1]" = "-h"
        echo "t - Quick session switcher"
        echo ""
        echo "Usage: t [repo]"
        echo ""
        echo "Arguments:"
        echo "  repo    Repository or session name (optional)"
        echo ""
        echo "Examples:"
        echo "  t                # Interactive selection with fzf"
        echo "  t my-project     # Switch to existing session or create from repo"
        echo "  t session-name   # Switch to any tmux session"
        echo ""
        echo "Features:"
        echo "  - Interactive fzf selection when called without arguments"
        echo "  - Searches for existing tmux sessions first"
        echo "  - Searches all TWINE_BASE_DIRS for repos"
        echo "  - Supports TWINE_SESSION_PREFIX"
        echo "  - Creates session with tmuxinator if available"
        echo "  - Falls back to basic tmux session"
        echo ""
        echo "Tab completion shows:"
        echo "  - Running sessions first (▶)"
        echo "  - Available repositories (📁)"
        echo ""
        echo "See also: tw, ts"
        return 0
    end

    if test (count $argv) -lt 1
        if not command -v fzf >/dev/null
            echo "Usage: t <repo>"
            echo "Run 't --help' for more information"
            echo ""
            echo "Tip: Install fzf for interactive selection"
            return 1
        end

        # Get running sessions
        set sessions
        if tmux has-session >/dev/null 2>&1
            set sessions (tmux list-sessions -F '#S')
        end

        # Get available repos
        set repos
        if set -q TWINE_BASE_DIRS
            for base_dir in $TWINE_BASE_DIRS
                if test -d $base_dir
                    for dir in $base_dir/*/
                        set repo (basename $dir)
                        if not contains $repo $sessions
                            set repos $repos $repo
                        end
                    end
                end
            end
        end

        # Show selection menu
        set selection (begin
            echo $sessions | tr ' ' '\n' | sed 's|$| ▶|'
            echo $repos | tr ' ' '\n' | sed 's|$| 📁|'
        end | fzf --height=40% --prompt="Select session or repo: ")

        if test -z "$selection"
            return 1
        end

        # Remove the symbol and use as argument
        set argv (string replace -r ' [▶📁]$' '' $selection)
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
                __twine_attach $name
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
            __twine_attach $name
            return
        end
    end

    # No repo found, create basic session with the provided name
    __twine_attach $names[2]
end
