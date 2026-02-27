# Completion for twine command

# Complete verbose actions
complete \
    --command twine \
    --no-files \
    --condition "test (count (commandline -opc)) -eq 1" \
    --arguments "worktree session attach start init convert" \
    --description "Twine action"

# Complete shortcut actions
complete \
    --command twine \
    --no-files \
    --condition "test (count (commandline -opc)) -eq 1" \
    --arguments "tw t ts" \
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
