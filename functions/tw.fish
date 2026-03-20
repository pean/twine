function tw --description 'Switch to tmux session for a worktree (creates worktree if needed)'
    # Show help
    if test "$argv[1]" = "--help"; or test "$argv[1]" = "-h"
        echo "tw - Worktree + tmux session manager"
        echo ""
        echo "Usage: tw [repo] [branch] [options]"
        echo ""
        echo "Arguments:"
        echo "  repo     Repository name (optional, uses fzf if not provided)"
        echo "  branch   Branch/worktree name (optional, uses fzf if not provided)"
        echo ""
        echo "Options:"
        echo "  -c, --create       Create new branch if it doesn't exist"
        echo "  -f, --from BRANCH  Base branch for new branch (default: main/master)"
        echo ""
        echo "Examples:"
        echo "  tw                              # Interactive repo and branch selection"
        echo "  tw my-project                   # Interactive branch selection"
        echo "  tw my-project main              # Switch to main branch worktree"
        echo "  tw my-project feature/x         # Creates worktree from remote"
        echo "  tw my-project feature/x -c      # Create new branch from main/master"
        echo "  tw my-project feature/x -c -f develop  # Create from develop"
        echo ""
        echo "Features:"
        echo "  - Interactive fzf selection for repos and branches"
        echo "  - Auto-detects bare (.git) and regular repos"
        echo "  - Creates worktrees from remote branches"
        echo "  - Creates new branches with --create flag"
        echo "  - Fetches latest branches before selection"
        echo "  - Creates or switches to tmux session"
        echo "  - Offers to convert regular repos to bare"
        echo "  - Visual indicators: ▶ for active, 📁 for available"
        echo ""
        echo "Session naming: \$TWINE_SESSION_PREFIX<repo>/<branch>"
        echo ""
        echo "See also: twine init, twine convert"
        return 0
    end

    # Parse flags
    set create_branch 0
    set base_branch ""
    set positional_args

    for arg in $argv
        switch $arg
            case -c --create
                set create_branch 1
            case -f --from
                # Next arg will be the base branch
                set parse_from 1
            case '*'
                if set -q parse_from
                    set base_branch $arg
                    set -e parse_from
                else
                    set positional_args $positional_args $arg
                end
        end
    end

    # Replace argv with positional arguments
    set argv $positional_args

    # Check for required configuration
    if not set -q TWINE_BASE_DIRS
        echo "Error: TWINE_BASE_DIRS not configured"
        echo "Add to ~/.config/fish/config.fish: set -gx TWINE_BASE_DIRS ~/src/repos"
        return 1
    end

    if test (count $argv) -lt 1
        if not command -v fzf >/dev/null
            echo "Usage: tw <repo> [branch]"
            echo "Run 'tw --help' for more information"
            echo ""
            echo "Tip: Install fzf for interactive selection"
            return 1
        end

        # Get running sessions (extract repo name from "repo/worktree" format)
        set running_sessions
        if tmux has-session >/dev/null 2>&1
            set running_sessions (tmux list-sessions -F '#S' | grep '/' | sed 's|/.*||' | sort -u)
        end

        # Get available repos
        set repos
        for base_dir in $TWINE_BASE_DIRS
            if test -d $base_dir
                for dir in $base_dir/*.git/
                    if test -d $dir
                        set name (basename $dir)
                        set repo_name (string replace '.git' '' $name)
                        if not contains $repo_name $running_sessions
                            set repos $repos $repo_name
                        end
                    end
                end
            end
        end

        # Show selection menu
        set selection (begin
            echo $running_sessions | tr ' ' '\n' | sed 's|$| ▶|'
            echo $repos | tr ' ' '\n' | sed 's|$| 📁|'
        end | fzf --height=40% --prompt="Select repo: ")

        if test -z "$selection"
            return 1
        end

        # Remove the symbol and use as argument
        set argv (string replace -r ' [▶📁]$' '' $selection)
    end

    # Strip .git suffix if provided
    set repo (string replace -r '\.git$' '' $argv[1])

    # Find repo path across all base directories
    set repo_path ""
    set found_regular_repo 0
    for base_dir in $TWINE_BASE_DIRS
        if test -d $base_dir/$repo.git
            set repo_path $base_dir/$repo.git
            break
        else if test -d $base_dir/$repo
            set repo_path $base_dir/$repo
            set found_regular_repo 1
            break
        end
    end

    if test -z "$repo_path"
        echo "Error: Repository '$repo' not found in:"
        for base_dir in $TWINE_BASE_DIRS
            echo "  - $base_dir"
        end
        return 1
    end

    # Offer to convert regular repo to bare + worktree setup
    if test $found_regular_repo -eq 1
        echo "Found regular repository: $repo_path"
        echo ""
        echo "For better worktree support, convert to bare repo? (y/N)"
        echo "This will:"
        echo "  1. Clone as bare ($repo_path.git)"
        echo "  2. Create worktree for current branch"
        echo "  3. Optionally remove old repo"
        echo ""
        read -P "> " -n 1 convert

        if test "$convert" = "y"
            echo ""
            if __tw_convert_to_bare $repo_path
                # Update repo_path to the new bare repo
                set repo_path $repo_path.git
            else
                return 1
            end
        else
            echo ""
            echo "Continuing with regular repo (limited worktree support)"
            echo "Tip: Run 'twine convert $repo' later to convert"
            echo ""
        end
    end

    # Interactive branch/worktree selection with fzf if not provided
    if test (count $argv) -lt 2
        if not command -v fzf >/dev/null
            echo "Error: fzf required for interactive selection"
            return 1
        end

        # Get existing worktrees (exclude bare repo root)
        set existing_worktrees (git -C $repo_path worktree list --porcelain 2>/dev/null | \
            grep '^worktree ' | \
            sed 's|^worktree '"$repo_path"'/\{0,1\}||' | \
            grep -v '^\s*$')

        # Fetch latest branches with spinner
        echo -n "Fetching branches..."
        git -C $repo_path fetch --quiet 2>/dev/null &
        set fetch_pid $last_pid

        # Show spinner while fetching
        while kill -0 $fetch_pid 2>/dev/null
            echo -n "."
            sleep 0.2
        end
        echo " ✓"

        # Get remote branches
        set remote_branches (git -C $repo_path branch -r 2>/dev/null | \
            sed 's|^[* ]*origin/||' | \
            grep -v '^HEAD')

        # Get local branches
        set local_branches (git -C $repo_path branch 2>/dev/null | \
            sed 's|^[* ]*||')

        # Combine and show in fzf
        set selection (begin
            echo $existing_worktrees | tr ' ' '\n' | sed 's|$| ▶|'
            echo $local_branches | tr ' ' '\n' | sed 's|$| 📁|'
            echo $remote_branches | tr ' ' '\n' | sed 's|$| 📁|'
        end | sort -u | fzf --height=40% --prompt="Select branch for $repo: ")

        if test -z "$selection"
            return 1
        end

        # Remove the symbol
        set branch (string replace -r ' [▶📁]$' '' $selection)
    else
        set branch $argv[2]
    end

    set worktree_path $repo_path/$branch
    set session_name "$TWINE_SESSION_PREFIX$repo/$branch"

    # Create worktree if it doesn't exist
    if not test -d $worktree_path
        echo "Worktree '$branch' doesn't exist, checking for branch..."

        # Check if branch exists on remote
        if git -C $repo_path branch -r 2>/dev/null | grep -q "origin/$branch\$"
            echo "Creating worktree from origin/$branch..."

            # Check if local branch already exists
            if git -C $repo_path branch --list $branch | grep -q "."
                # Local branch exists, use it
                if git -C $repo_path worktree add $branch $branch 2>&1
                    echo "✓ Worktree created: $branch"
                    # Set upstream tracking if not already configured
                    git -C $repo_path branch --set-upstream-to=origin/$branch $branch 2>/dev/null
                else
                    echo "✗ Failed to create worktree"
                    return 1
                end
            else
                # Local branch doesn't exist, create tracking branch
                if git -C $repo_path worktree add -b $branch $branch origin/$branch 2>&1
                    echo "✓ Worktree created: $branch"
                else
                    echo "✗ Failed to create worktree"
                    return 1
                end
            end
        else
            # Branch not found on remote
            if test $create_branch -eq 1
                # Determine base branch
                if test -z "$base_branch"
                    # Auto-detect default branch - check main/master first
                    if git -C $repo_path branch -r 2>/dev/null | grep -q "origin/main\$"
                        set base_branch main
                    else if git -C $repo_path branch -r 2>/dev/null | grep -q "origin/master\$"
                        set base_branch master
                    else
                        # Try symbolic-ref as fallback
                        set base_branch (git -C $repo_path symbolic-ref refs/remotes/origin/HEAD 2>/dev/null | sed 's|^refs/remotes/origin/||')
                        if test -z "$base_branch"
                            echo "Error: Could not determine default branch"
                            echo "Use --from to specify base branch explicitly"
                            return 1
                        end
                    end
                end

                echo "Creating new branch '$branch' from '$base_branch'..."

                # Determine starting point (prefer remote, fall back to local)
                set start_point ""
                if git -C $repo_path branch -r 2>/dev/null | grep -q "origin/$base_branch\$"
                    set start_point origin/$base_branch
                else if git -C $repo_path branch --list $base_branch | grep -q "."
                    set start_point $base_branch
                else
                    echo "Error: Base branch '$base_branch' not found (neither origin/$base_branch nor local $base_branch exist)"
                    echo ""
                    echo "Available branches:"
                    git -C $repo_path branch -a | head -10
                    return 1
                end

                # Create worktree with new branch
                if git -C $repo_path worktree add -b $branch $branch $start_point 2>&1
                    echo "✓ Worktree and branch created: $branch (from $start_point)"
                else
                    echo "✗ Failed to create worktree"
                    return 1
                end
            else
                echo "Error: Branch '$branch' not found on remote"
                echo ""
                echo "Available remote branches:"
                git -C $repo_path branch -r | sed 's|^[* ]*origin/||' | grep -v '^HEAD' | head -10
                echo ""
                echo "Tip: Use -c or --create to create a new branch"
                return 1
            end
        end
    end

    # Switch to existing session or create new one
    if tmux list-sessions 2>/dev/null | grep -q "^$session_name:"
        if set -q TMUX
            tmux switch-client -t $session_name
        else
            tmux attach -t $session_name
        end
    else
        echo "Creating tmux session: $session_name"

        # Check if tmuxinator should be used
        if __twine_use_tmuxinator
            fish -c "cd $worktree_path && tmuxinator start -n $session_name $TWINE_TMUXINATOR_LAYOUT false"
        else
            # Create basic tmux session
            tmux new-session -d -s $session_name -c $worktree_path
        end

        if set -q TMUX
            tmux switch-client -t $session_name
        else
            tmux attach -t $session_name
        end
    end
end
