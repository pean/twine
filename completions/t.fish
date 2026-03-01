function __t_complete_t
    # Get list of running sessions
    set running_sessions
    if tmux has-session >/dev/null 2>&1
        set running_sessions (tmux list-sessions -F '#S')
    end

    # Show running sessions first with ▶ symbol
    for session in $running_sessions
        echo -e "$session\t▶"
    end

    # Add directories from all base directories with 📁 symbol
    if set -q TWINE_BASE_DIRS
        for base_dir in $TWINE_BASE_DIRS
            if test -d $base_dir
                for dir in $base_dir/*/
                    set repo (basename $dir)
                    # Skip if already shown as running session
                    if not contains $repo $running_sessions
                        echo -e "$repo\t📁"
                    end
                end
            end
        end
    end
end

complete \
    --keep-order \
    --no-files \
    --command t \
    --arguments "(__t_complete_t)"
