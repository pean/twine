#!/usr/bin/env fish

# Test ts command (tmuxinator wrapper)

set -g test_dir (mktemp -d)

function setup
    source functions/ts.fish
end

function teardown
    rm -rf $test_dir
end

@test "ts errors when tmuxinator not installed"
    setup

    function command
        if test "$argv[1]" = "-v"
            if test "$argv[2]" = "tmuxinator"
                return 1
            end
        end
        builtin command $argv
    end

    set output (ts 2>&1)
    string match -q "*tmuxinator not installed*" $output

    functions -e command
    teardown

@test "ts errors when not in a git repo"
    setup

    function command
        if test "$argv[1]" = "-v"
            if test "$argv[2]" = "tmuxinator"
                return 0
            end
        end
        builtin command $argv
    end

    function git
        if test "$argv[1]" = "rev-parse"
            return 1
        end
    end

    cd $test_dir
    set output (ts 2>&1)
    string match -q "*Not a repo*" $output

    functions -e command
    functions -e git
    teardown

@test "ts extracts repo name correctly"
    setup

    mkdir -p $test_dir/test-repo
    cd $test_dir/test-repo

    function command
        if test "$argv[1]" = "-v"
            if test "$argv[2]" = "tmuxinator"
                return 0
            end
        end
        builtin command $argv
    end

    function git
        if test "$argv[1]" = "rev-parse"
            if test "$argv[2]" = "--git-dir"
                return 0
            else if test "$argv[2]" = "--show-toplevel"
                echo "$test_dir/test-repo"
                return 0
            end
        end
    end

    function tmuxinator
        echo "tmuxinator: $argv"
        return 0
    end

    set output (ts 2>&1)
    string match -q "*Starting tmuxinator for test-repo*" $output
    and string match -q "*tmuxinator: start -n test-repo dev*" $output

    functions -e command
    functions -e git
    functions -e tmuxinator
    teardown

@test "ts strips leading dot from repo name"
    setup

    mkdir -p $test_dir/.dotrepo
    cd $test_dir/.dotrepo

    function command
        if test "$argv[1]" = "-v"
            if test "$argv[2]" = "tmuxinator"
                return 0
            end
        end
        builtin command $argv
    end

    function git
        if test "$argv[1]" = "rev-parse"
            if test "$argv[2]" = "--git-dir"
                return 0
            else if test "$argv[2]" = "--show-toplevel"
                echo "$test_dir/.dotrepo"
                return 0
            end
        end
    end

    function tmuxinator
        echo "tmuxinator: $argv"
        return 0
    end

    set output (ts 2>&1)
    string match -q "*tmuxinator: start -n dotrepo dev*" $output

    functions -e command
    functions -e git
    functions -e tmuxinator
    teardown
