#!/usr/bin/env fish

# Test unified twine command

function setup
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
end

function teardown
    functions -e tw
    functions -e t
    functions -e __twine_attach
    functions -e ts
    functions -e __twine_init
    functions -e __twine_convert
end

@test "twine routes worktree action to tw"
    setup

    set output (twine worktree my-repo 2>&1)
    string match -q "*tw called with: my-repo*" $output

    teardown

@test "twine routes tw shortcut to tw function"
    setup

    set output (twine tw my-repo feature/branch 2>&1)
    string match -q "*tw called with: my-repo feature/branch*" $output

    teardown

@test "twine routes session action to t"
    setup

    set output (twine session my-repo 2>&1)
    string match -q "*t called with: my-repo*" $output

    teardown

@test "twine routes t shortcut to t function"
    setup

    set output (twine t my-session 2>&1)
    string match -q "*t called with: my-session*" $output

    teardown

@test "twine routes attach action to __twine_attach"
    setup

    set output (twine attach my-session 2>&1)
    string match -q "*__twine_attach called with: my-session*" $output

    teardown

@test "twine routes start action to ts"
    setup

    set output (twine start 2>&1)
    string match -q "*ts called with:*" $output

    teardown

@test "twine routes ts shortcut to ts function"
    setup

    set output (twine ts 2>&1)
    string match -q "*ts called with:*" $output

    teardown

@test "twine routes init action to __twine_init"
    setup

    set output (twine init my-repo https://github.com/user/repo.git 2>&1)
    string match -q "*__twine_init called with: my-repo https://github.com/user/repo.git*" $output

    teardown

@test "twine routes convert action to __twine_convert"
    setup

    set output (twine convert my-repo 2>&1)
    string match -q "*__twine_convert called with: my-repo*" $output

    teardown

@test "twine shows error for unknown action"
    setup

    set output (twine unknown-action 2>&1)
    string match -q "*Unknown action 'unknown-action'*" $output

    teardown

@test "twine shows help when called without arguments"
    setup

    set output (twine 2>&1)
    string match -q "*Twine - Git worktree + tmux session management*" $output

    teardown

@test "twine shows help with --help flag"
    setup

    set output (twine --help 2>&1)
    string match -q "*Twine - Git worktree + tmux session management*" $output

    teardown
