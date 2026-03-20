function tk -d "Kill tmux sessions with optional worktree removal"
    if not command -q tmux
        echo "Error: tmux is not installed" >&2
        return 1
    end

    # Parse flags and explicit session names
    set -l do_worktree 0
    set -l sessions

    for arg in $argv
        switch $arg
            case --worktree -w
                set do_worktree 1
            case '*'
                set -a sessions $arg
        end
    end

    # If no explicit sessions, use fzf multi-select
    if test (count $sessions) -eq 0
        if not command -q fzf
            echo "Error: fzf is required for interactive selection" >&2
            return 1
        end

        set -l raw_sessions (tmux list-sessions -F \
            "#{session_name}|#{session_path}|#{session_windows}" 2>/dev/null)

        if test (count $raw_sessions) -eq 0
            echo "No tmux sessions running."
            return 0
        end

        # Format as "name | path | Nw" and display with fzf
        set -l formatted
        for s in $raw_sessions
            set -l parts (string split '|' $s)
            set -a formatted "$parts[1]|$parts[2]|$parts[3]w"
        end

        set -l selected (printf '%s\n' $formatted | \
            column -t -s '|' | \
            fzf --multi \
                --ansi \
                --reverse \
                --border \
                --prompt "Kill session(s) > " \
                --header "Tab to select multiple, Enter to kill, Esc to cancel")

        if test -z "$selected"
            return 0
        end

        # Recover session names (first column)
        for line in (string split '\n' $selected)
            set -l name (echo $line | awk '{print $1}')
            if test -n "$name"
                set -a sessions $name
            end
        end
    end

    # Kill each session (and optionally its worktree)
    for session in $sessions
        tmux kill-session -t $session 2>/dev/null
        if test $status -ne 0
            echo "Warning: could not kill session '$session'" >&2
        else
            echo "Killed session: $session"
        end

        if test $do_worktree -eq 1
            # Expect session name in "repo/branch" format
            set -l parts (string split '/' $session)
            if test (count $parts) -lt 2
                echo "Warning: cannot determine worktree for '$session' (not repo/branch format)" >&2
                continue
            end

            # Strip optional TWINE_SESSION_PREFIX from the repo part
            set -l repo $parts[1]
            if set -q TWINE_SESSION_PREFIX; and test -n "$TWINE_SESSION_PREFIX"
                set repo (string replace -- "$TWINE_SESSION_PREFIX" "" $repo)
            end
            set -l branch (string join '/' $parts[2..])

            set -l repo_path ""
            for base_dir in $TWINE_BASE_DIRS
                if test -d "$base_dir/$repo.git"
                    set repo_path "$base_dir/$repo.git"
                    break
                end
            end

            if test -z "$repo_path"
                echo "Warning: no bare repo found for '$repo', skipping worktree removal" >&2
                continue
            end

            git -C $repo_path worktree remove $branch --force 2>/dev/null
            if test $status -eq 0
                echo "Removed worktree: $branch"
            else
                echo "Warning: could not remove worktree '$branch' in $repo_path" >&2
            end
        end
    end
end
