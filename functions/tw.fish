function tw --description 'Switch to tmux session for a worktree (creates worktree if needed)'
    # Show help
    if test "$argv[1]" = "--help"; or test "$argv[1]" = "-h"
        echo "tw - Worktree + tmux session manager"
        echo ""
        echo "Usage: tw <repo> [branch]"
        echo ""
        echo "Arguments:"
        echo "  repo     Repository name (searches TWINE_BASE_DIRS)"
        echo "  branch   Branch/worktree name (optional, uses fzf if not provided)"
        echo ""
        echo "Examples:"
        echo "  tw my-project              # Interactive branch selection"
        echo "  tw my-project main         # Switch to main branch worktree"
        echo "  tw my-project feature/x    # Creates worktree if doesn't exist"
        echo ""
        echo "Features:"
        echo "  - Auto-detects bare (.git) and regular repos"
        echo "  - Creates worktrees from remote branches"
        echo "  - Fetches latest branches before selection"
        echo "  - Creates or switches to tmux session"
        echo "  - Offers to convert regular repos to bare"
        echo ""
        echo "Session naming: \$TWINE_SESSION_PREFIX<repo>/<branch>"
        echo ""
        echo "See also: twine init, twine convert"
        return 0
    end

    if test (count $argv) -lt 1
        echo "Usage: tw <repo> [branch]"
        echo "Run 'tw --help' for more information"
        return 1
    end

    # Check for required configuration
    if not set -q TWINE_BASE_DIRS
        echo "Error: TWINE_BASE_DIRS not configured"
        echo "Add to ~/.config/fish/config.fish: set -gx TWINE_BASE_DIRS ~/src/repos"
        return 1
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

        # Combine and show in fzf
        set selection (begin
            echo $existing_worktrees | tr ' ' '\n' | sed 's|$| (worktree)|'
            echo $remote_branches | tr ' ' '\n' | sed 's|$| (remote)|'
        end | sort -u | fzf --height=40% --prompt="Select branch for $repo: ")

        if test -z "$selection"
            return 1
        end

        # Remove the label
        set branch (string replace -r ' \(.*\)$' '' $selection)
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
            echo "Error: Branch '$branch' not found on remote"
            echo ""
            echo "Available remote branches:"
            git -C $repo_path branch -r | sed 's|^[* ]*origin/||' | grep -v '^HEAD' | head -10
            return 1
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
