#!/usr/bin/env fish

# Test agents command
source functions/agents.fish

@test "agents: requires fzf" (
    # Mock fzf as missing
    function command -a cmd
        if test "$cmd" = "-q"
            test "$argv[2]" != "fzf"
        else
            builtin command $argv
        end
    end

    set result (agents 2>&1)
    echo $result | string match -q "*fzf is required*"
) $status -eq 0

@test "agents: requires tmux" (
    # Mock tmux as missing
    function command -a cmd
        if test "$cmd" = "-q"
            test "$argv[2]" != "tmux"
        else
            builtin command $argv
        end
    end

    set result (agents 2>&1)
    echo $result | string match -q "*tmux is not installed*"
) $status -eq 0

@test "agents: handles no agents found" (
    # Mock empty sessions directory
    set -l temp_home (mktemp -d)
    mkdir -p $temp_home/.claude/sessions

    set -l original_home $HOME
    set -gx HOME $temp_home

    # Mock tmux to return no opencode processes
    function tmux
        # Return empty
    end

    set result (agents 2>&1)
    set status_code $status

    set -gx HOME $original_home
    rm -rf $temp_home

    test $status_code -eq 1
    and echo $result | string match -q "*No AI coding agents*"
) $status -eq 0
