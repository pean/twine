function __twine_prune --description 'Internal: Prune gone worktrees and branches across all repos'
    if test "$argv[1]" = "--help"; or test "$argv[1]" = "-h"
        echo "prune - Clean up branches whose remote has been deleted"
        echo ""
        echo "Usage: twine prune [--dry-run] [repo]"
        echo ""
        echo "Options:"
        echo "  -n, --dry-run   Show what would be removed without doing it"
        echo ""
        echo "Arguments:"
        echo "  repo    Repository name (optional, prunes all repos if omitted)"
        echo ""
        echo "What it does:"
        echo "  1. Fetches all remotes and prunes remote tracking branches"
        echo "  2. Finds branches marked as 'gone' (remote deleted)"
        echo "  3. Kills tmux sessions for gone branches"
        echo "  4. Removes worktrees for gone branches (bare repos)"
        echo "  5. Deletes local branches marked as gone"
        echo ""
        echo "Examples:"
        echo "  twine prune                  # Prune all repos in TWINE_BASE_DIRS"
        echo "  twine prune --dry-run        # Show what would be pruned"
        echo "  twine prune my-project       # Prune specific repository"
        return 0
    end

    set dry_run 0
    set remaining_argv
    for arg in $argv
        if test "$arg" = "--dry-run"; or test "$arg" = "-n"
            set dry_run 1
        else
            set -a remaining_argv $arg
        end
    end
    set argv $remaining_argv

    if not set -q TWINE_BASE_DIRS
        echo "Error: TWINE_BASE_DIRS not configured"
        echo "Add to ~/.config/fish/config.fish: set -gx TWINE_BASE_DIRS ~/src/repos"
        return 1
    end

    set repo_paths
    set repo_bares

    if test -n "$argv[1]"
        set found 0
        for base_dir in $TWINE_BASE_DIRS
            if test -d "$base_dir/$argv[1].git"
                set -a repo_paths "$base_dir/$argv[1].git"
                set -a repo_bares 1
                set found 1
                break
            else if test -d "$base_dir/$argv[1]"
                set -a repo_paths "$base_dir/$argv[1]"
                set -a repo_bares 0
                set found 1
                break
            end
        end
        if test $found -eq 0
            echo "Error: Repository not found: $argv[1]"
            return 1
        end
    else
        for base_dir in $TWINE_BASE_DIRS
            test -d "$base_dir" || continue
            for dir in "$base_dir"/*/
                set dir (string trim -r -c '/' -- $dir)
                test -d "$dir" || continue
                if string match -q "*.git" "$dir"
                    if test -f "$dir/config"; and grep -q "bare = true" "$dir/config"
                        set -a repo_paths "$dir"
                        set -a repo_bares 1
                    end
                else if test -d "$dir/.git"
                    set -a repo_paths "$dir"
                    set -a repo_bares 0
                end
            end
        end
    end

    if test (count $repo_paths) -eq 0
        echo "No repositories found in TWINE_BASE_DIRS"
        return 0
    end

    set total_pruned 0

    if test $dry_run -eq 1
        echo "(dry run)"
        echo ""
    end

    for i in (seq (count $repo_paths))
        set repo_path $repo_paths[$i]
        set is_bare $repo_bares[$i]
        set repo_name (basename $repo_path | string replace -r '\.git$' '')

        echo "→ $repo_name"

        if not git -C $repo_path fetch --all -p -q 2>/dev/null
            echo "  ⚠ Failed to fetch"
            continue
        end

        set gone_branches (git -C $repo_path branch -vv 2>/dev/null | grep ': gone]' | awk '{ print ($1 == "+" || $1 == "*") ? $2 : $1 }')

        if test (count $gone_branches) -eq 0
            echo "  ✓ Nothing to prune"
            continue
        end

        for branch in $gone_branches
            set session_name "$TWINE_SESSION_PREFIX$repo_name/$branch"

            if test $dry_run -eq 1
                if test $is_bare -eq 1; and test -d "$repo_path/$branch"
                    echo "  would remove worktree: $branch"
                end
                if tmux has-session -t "$session_name" 2>/dev/null
                    echo "  would kill session: $session_name"
                end
                echo "  would delete branch: $branch"
                set total_pruned (math $total_pruned + 1)
                continue
            end

            if tmux has-session -t "$session_name" 2>/dev/null
                if tmux kill-session -t "$session_name" 2>/dev/null
                    echo "  ✓ Killed session: $session_name"
                else
                    echo "  ⚠ Failed to kill session: $session_name"
                end
            end

            if test $is_bare -eq 1
                set worktree_path "$repo_path/$branch"
                if test -d "$worktree_path"
                    if git -C $repo_path worktree remove "$worktree_path" --force 2>/dev/null
                        echo "  ✓ Removed worktree: $branch"
                    else
                        echo "  ⚠ Failed to remove worktree: $branch"
                    end
                end
            end

            if git -C $repo_path branch -D "$branch" 2>/dev/null
                echo "  ✓ Deleted branch: $branch"
                set total_pruned (math $total_pruned + 1)
            else
                echo "  ⚠ Failed to delete branch: $branch"
            end
        end
    end

    echo ""
    if test $dry_run -eq 1
        echo "Would prune $total_pruned branch(es) across "(count $repo_paths)" repo(s)."
    else
        echo "Done. Pruned $total_pruned branch(es) across "(count $repo_paths)" repo(s)."
    end
end
