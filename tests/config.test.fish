#!/usr/bin/env fish

# Test configuration and initialization

@test "warns when TWINE_BASE_DIRS not configured"
    set -e TWINE_BASE_DIRS
    set output (source conf.d/twine.fish 2>&1)
    string match -q "*TWINE_BASE_DIRS not configured*" $output

@test "sets default TWINE_SESSION_PREFIX to empty"
    set -e TWINE_SESSION_PREFIX
    source conf.d/twine.fish
    test "$TWINE_SESSION_PREFIX" = ""

@test "sets default TWINE_USE_TMUXINATOR to auto"
    set -e TWINE_USE_TMUXINATOR
    source conf.d/twine.fish
    test "$TWINE_USE_TMUXINATOR" = "auto"

@test "__twine_use_tmuxinator returns 1 when explicitly disabled"
    set -gx TWINE_USE_TMUXINATOR false
    source conf.d/twine.fish
    __twine_use_tmuxinator

@test "__twine_use_tmuxinator returns 1 when tmuxinator not installed"
    set -gx TWINE_USE_TMUXINATOR true
    source conf.d/twine.fish

    # Mock tmuxinator not being available
    function command
        if test "$argv[1]" = "-v"
            if test "$argv[2]" = "tmuxinator"
                return 1
            end
        end
        builtin command $argv
    end

    __twine_use_tmuxinator

    functions -e command

@test "__twine_use_tmuxinator returns 0 when explicitly enabled and tmuxinator exists"
    set -gx TWINE_USE_TMUXINATOR true
    source conf.d/twine.fish

    # Mock tmuxinator being available
    function command
        if test "$argv[1]" = "-v"
            if test "$argv[2]" = "tmuxinator"
                return 0
            end
        end
        builtin command $argv
    end

    __twine_use_tmuxinator

    functions -e command

@test "__twine_use_tmuxinator auto mode returns 0 when tmuxinator exists and layout configured"
    set -gx TWINE_USE_TMUXINATOR auto
    set -gx TWINE_TMUXINATOR_LAYOUT dev
    source conf.d/twine.fish

    # Mock tmuxinator being available
    function command
        if test "$argv[1]" = "-v"
            if test "$argv[2]" = "tmuxinator"
                return 0
            end
        end
        builtin command $argv
    end

    __twine_use_tmuxinator

    functions -e command

@test "__twine_use_tmuxinator auto mode returns 1 when layout not configured"
    set -gx TWINE_USE_TMUXINATOR auto
    set -e TWINE_TMUXINATOR_LAYOUT
    source conf.d/twine.fish

    # Mock tmuxinator being available
    function command
        if test "$argv[1]" = "-v"
            if test "$argv[2]" = "tmuxinator"
                return 0
            end
        end
        builtin command $argv
    end

    __twine_use_tmuxinator

    functions -e command
