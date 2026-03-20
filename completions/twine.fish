# Completion for twine command

# Complete verbose actions
complete \
    --command twine \
    --no-files \
    --condition "test (count (commandline -opc)) -eq 1" \
    --arguments "worktree session attach start init convert prune agents kill" \
    --description "Twine action"

# Complete shortcut actions
complete \
    --command twine \
    --no-files \
    --condition "test (count (commandline -opc)) -eq 1" \
    --arguments "tw t ts tk" \
    --description "Shortcut"

# Add --help option
complete \
    --command twine \
    --long-option help \
    --short-option h \
    --description "Show help message"

# Delegate to subcommand completions based on action
# worktree / tw action
complete \
    --command twine \
    --condition "test (count (commandline -opc)) -ge 2; and string match -qr '^(worktree|tw)\$' -- (commandline -opc)[2]" \
    --wraps tw

# session / t action
complete \
    --command twine \
    --condition "test (count (commandline -opc)) -ge 2; and string match -qr '^(session|t)\$' -- (commandline -opc)[2]" \
    --wraps t

# attach action (no completion needed for internal command)
# convert action (no completion needed for internal command)
# agents action (no completion needed for internal command)

# prune action - complete repo names
complete \
    --command twine \
    --condition "test (count (commandline -opc)) -eq 2; and string match -q 'prune' -- (commandline -opc)[2]" \
    --no-files \
    --arguments "(source completions/tw.fish; __tw_complete_repos)"

# kill / tk action - complete session names and --worktree flag
complete \
    --command twine \
    --condition "test (count (commandline -opc)) -ge 2; and string match -qr '^(kill|tk)$' -- (commandline -opc)[2]" \
    --no-files \
    --arguments "(tmux list-sessions -F '#{session_name}' 2>/dev/null)"

complete \
    --command twine \
    --condition "test (count (commandline -opc)) -ge 2; and string match -qr '^(kill|tk)$' -- (commandline -opc)[2]" \
    --long-option worktree \
    --short-option w \
    --description "Also remove git worktree"

complete \
    --command tk \
    --no-files \
    --arguments "(tmux list-sessions -F '#{session_name}' 2>/dev/null)"

complete \
    --command tk \
    --long-option worktree \
    --short-option w \
    --description "Also remove git worktree"
