#!/usr/bin/env fish

# Test __twine_attach internal function (tmux attach)

function setup
    source functions/__twine_attach.fish
end

@test "__twine_attach accepts session name argument"
    setup

    # Mock tmux commands to test behavior
    function tmux
        # Simulate tmux list-sessions failing (no sessions)
        if test "$argv[1]" = "list-sessions"
            return 1
        else if test "$argv[1]" = "new-session"
            # Capture arguments
            echo "new-session: $argv[2-]"
            return 0
        end
    end

    set output (__twine_attach test-session 2>&1)
    string match -q "*Creating new session test-session*" $output
    and string match -q "*new-session: -d -s test-session*" $output

    functions -e tmux

@test "__twine_attach switches to existing session when in TMUX"
    setup

    function tmux
        if test "$argv[1]" = "list-sessions"
            echo "test-session: 1 windows"
            return 0
        else if test "$argv[1]" = "switch-client"
            echo "switch-client: $argv[2-]"
            return 0
        end
    end

    set -gx TMUX "fake-tmux-session"
    set output (__twine_attach test-session 2>&1)

    string match -q "*Switching to test-session*" $output
    and string match -q "*switch-client: -t test-session*" $output

    functions -e tmux
    set -e TMUX

@test "__twine_attach attaches to session when not in TMUX"
    setup

    function tmux
        if test "$argv[1]" = "list-sessions"
            echo "test-session: 1 windows"
            return 0
        else if test "$argv[1]" = "attach"
            echo "attach: $argv[2-]"
            return 0
        end
    end

    set -e TMUX
    set output (__twine_attach test-session 2>&1)

    string match -q "*Attaching to test-session*" $output
    and string match -q "*attach: -t test-session*" $output

    functions -e tmux

@test "__twine_attach shows usage when called without arguments"
    setup
    set output (__twine_attach 2>&1)
    string match -q "*Usage: twine attach <session-name>*" $output
    and string match -q "*--help*" $output
    teardown
