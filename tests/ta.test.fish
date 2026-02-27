#!/usr/bin/env fish

# Test __twine_attach internal function (tmux attach)

source functions/__twine_attach.fish

# Test: __twine_attach accepts session name argument
function tmux
    if test "$argv[1]" = "list-sessions"
        return 1
    else if test "$argv[1]" = "new-session"
        echo "new-session: $argv[2..]"
        return 0
    end
end
set output (__twine_attach test-session 2>&1)
@test "__twine_attach accepts session name argument" (string match -q "*Creating new session test-session*" -- $output; and string match -q "*new-session: -d -s test-session*" -- $output) $status -eq 0
functions -e tmux

# Test: __twine_attach switches to existing session when in TMUX
function tmux
    if test "$argv[1]" = "list-sessions"
        echo "test-session: 1 windows"
        return 0
    else if test "$argv[1]" = "switch-client"
        echo "switch-client: $argv[2..]"
        return 0
    end
end
set -gx TMUX "fake-tmux-session"
set output (__twine_attach test-session 2>&1)
@test "__twine_attach switches to existing session when in TMUX" (string match -q "*Switching to test-session*" -- $output; and string match -q "*switch-client: -t test-session*" -- $output) $status -eq 0
functions -e tmux
set -e TMUX

# Test: __twine_attach attaches to session when not in TMUX
function tmux
    if test "$argv[1]" = "list-sessions"
        echo "test-session: 1 windows"
        return 0
    else if test "$argv[1]" = "attach"
        echo "attach: $argv[2..]"
        return 0
    end
end
set -e TMUX
set output (__twine_attach test-session 2>&1)
@test "__twine_attach attaches to session when not in TMUX" (string match -q "*Attaching to test-session*" -- $output; and string match -q "*attach: -t test-session*" -- $output) $status -eq 0
functions -e tmux

# Test: __twine_attach shows usage when called without arguments
set output (__twine_attach 2>&1)
@test "__twine_attach shows usage when called without arguments" (string match -q "*Usage: twine attach <session-name>*" -- $output; and string match -q "*--help*" -- $output) $status -eq 0
