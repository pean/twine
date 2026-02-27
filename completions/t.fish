function __t_complete_t
    if tmux has-session >/dev/null 2>&1
        tmux list-sessions -F '#S'
    end

    # Add directories from all base directories
    if set -q TWINE_BASE_DIRS
        for base_dir in $TWINE_BASE_DIRS
            if test -d $base_dir
                for dir in $base_dir/*/
                    echo (basename $dir)
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
