#!/usr/bin/env fish

# Test unified twine command

source functions/twine.fish

# Mock subcommands
function tw
    echo "tw called with: $argv"
end

function t
    echo "t called with: $argv"
end

function __twine_attach
    echo "__twine_attach called with: $argv"
end

function ts
    echo "ts called with: $argv"
end

function __twine_init
    echo "__twine_init called with: $argv"
end

function __twine_convert
    echo "__twine_convert called with: $argv"
end

# Test: twine routes worktree action to tw
set output (twine worktree my-repo 2>&1)
@test "twine routes worktree action to tw" (string match -q "*tw called with: my-repo*" -- $output) $status -eq 0

# Test: twine routes tw shortcut to tw function
set output (twine tw my-repo feature/branch 2>&1)
@test "twine routes tw shortcut to tw function" (string match -q "*tw called with: my-repo feature/branch*" -- $output) $status -eq 0

# Test: twine routes session action to ts
set output (twine session my-repo 2>&1)
@test "twine routes session action to ts" (string match -q "*ts called with: my-repo*" -- $output) $status -eq 0

# Test: twine routes ts shortcut to ts function
set output (twine ts my-session 2>&1)
@test "twine routes ts shortcut to ts function" (string match -q "*ts called with: my-session*" -- $output) $status -eq 0

# Test: twine routes attach action to __twine_attach
set output (twine attach my-session 2>&1)
@test "twine routes attach action to __twine_attach" (string match -q "*__twine_attach called with: my-session*" -- $output) $status -eq 0

# Test: twine routes start action to ts
set output (twine start 2>&1)
@test "twine routes start action to ts" (string match -q "*ts called with:*" -- $output) $status -eq 0

# Test: twine routes ts shortcut to ts function
set output (twine ts 2>&1)
@test "twine routes ts shortcut to ts function" (string match -q "*ts called with:*" -- $output) $status -eq 0

# Test: twine routes init action to __twine_init
set output (twine init my-repo https://github.com/user/repo.git 2>&1)
@test "twine routes init action to __twine_init" (string match -q "*__twine_init called with: my-repo https://github.com/user/repo.git*" -- $output) $status -eq 0

# Test: twine routes convert action to __twine_convert
set output (twine convert my-repo 2>&1)
@test "twine routes convert action to __twine_convert" (string match -q "*__twine_convert called with: my-repo*" -- $output) $status -eq 0

# Test: twine shows error for unknown action
set output (twine unknown-action 2>&1)
@test "twine shows error for unknown action" (string match -q "*Unknown action 'unknown-action'*" -- $output) $status -eq 0

# Test: twine shows help when called without arguments
set output (twine 2>&1)
@test "twine shows help when called without arguments" (string match -q "*Twine - Git worktree + tmux session management*" -- $output) $status -eq 0

# Test: twine shows help with --help flag
set output (twine --help 2>&1)
@test "twine shows help with --help flag" (string match -q "*Twine - Git worktree + tmux session management*" -- $output) $status -eq 0

# Cleanup
functions -e tw
functions -e t
functions -e __twine_attach
functions -e ts
functions -e __twine_init
functions -e __twine_convert
