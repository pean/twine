function __tw_complete_repos
    # Show existing tmux sessions (extract repo name from "repo/worktree" format)
    if tmux has-session >/dev/null 2>&1
        tmux list-sessions -F '#S' | grep '/' | sed 's|/.*||' | sort -u
    end

    # Add directories from all base directories that end with .git
    if set -q TWINE_BASE_DIRS
        for base_dir in $TWINE_BASE_DIRS
            if test -d $base_dir
                for dir in $base_dir/*.git/
                    if test -d $dir
                        set name (basename $dir)
                        # Remove .git suffix for completion
                        string replace '.git' '' $name
                    end
                end
            end
        end
    end
end

function __tw_complete_branches
    if not set -q TWINE_BASE_DIRS
        return
    end

    set repo (commandline -opc)[2]
    set repo_path ""

    if test -z "$repo"
        return
    end

    # Strip .git suffix
    set repo (string replace -r '\.git$' '' $repo)

    # Search all base directories for the repo
    for base_dir in $TWINE_BASE_DIRS
        if test -d $base_dir/$repo.git
            set repo_path $base_dir/$repo.git
            break
        else if test -d $base_dir/$repo
            set repo_path $base_dir/$repo
            break
        end
    end

    if test -z "$repo_path"
        return
    end

    # Get existing worktrees (exclude bare repo root)
    set existing_worktrees (git -C $repo_path worktree list --porcelain 2>/dev/null | \
        grep '^worktree ' | \
        sed 's|^worktree '"$repo_path"'/\{0,1\}||' | \
        grep -v '^\s*$')

    # Get remote branches
    set remote_branches (git -C $repo_path branch -r 2>/dev/null | \
        sed 's|^[* ]*origin/||' | \
        grep -v '^HEAD')

    # Output existing worktrees with label
    for wt in $existing_worktrees
        echo -e "$wt\tworktree"
    end

    # Output remote branches (excluding those with worktrees) with label
    for branch in $remote_branches
        if not contains $branch $existing_worktrees
            echo -e "$branch\tremote"
        end
    end
end

complete \
    --keep-order \
    --no-files \
    --command tw \
    --condition "test (count (commandline -opc)) -eq 1" \
    --arguments "(__tw_complete_repos)"

complete \
    --keep-order \
    --no-files \
    --command tw \
    --condition "test (count (commandline -opc)) -eq 2" \
    --arguments "(__tw_complete_branches)"
