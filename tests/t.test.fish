#!/usr/bin/env fish

# Test t command (session switcher)

set -g test_dir (mktemp -d)

function setup
    mkdir -p $test_dir/repos
    set -gx TWINE_BASE_DIRS $test_dir/repos
    set -gx TWINE_SESSION_PREFIX ""
    set -gx TWINE_USE_TMUXINATOR false
    source conf.d/twine.fish
    source functions/t.fish
end

function teardown
    rm -rf $test_dir
    set -e TWINE_BASE_DIRS
    set -e TWINE_SESSION_PREFIX
    set -e TWINE_USE_TMUXINATOR
end

@test "t errors when TWINE_BASE_DIRS not configured"
    setup
    set -e TWINE_BASE_DIRS
    set output (t test 2>&1)
    string match -q "*TWINE_BASE_DIRS not configured*" $output
    teardown

@test "t builds session names with and without prefix"
    setup
    set -gx TWINE_SESSION_PREFIX "work-"

    set names $TWINE_SESSION_PREFIX"test" "test"

    test (count $names) -eq 2
    and test "$names[1]" = "work-test"
    and test "$names[2]" = "test"
    teardown

@test "t searches for repo in base directories"
    setup

    # Create test repo
    mkdir -p $test_dir/repos/test-repo
    cd $test_dir/repos/test-repo
    git init >/dev/null 2>&1

    # Test finding repo
    set name "test-repo"
    set found 0

    for base_dir in $TWINE_BASE_DIRS
        if test -d $base_dir/$name
            set found 1
            break
        end
    end

    test $found -eq 1
    teardown

@test "t finds worktree in bare repo"
    setup

    # Create bare repo with worktree
    cd $test_dir/repos
    git init --bare test-repo.git >/dev/null 2>&1
    mkdir -p test-repo.git/main

    set name "main"
    set found 0

    for base_dir in $TWINE_BASE_DIRS
        if test -d $base_dir/$name.git/$name
            set found 1
            break
        end
    end

    test $found -eq 1
    teardown

@test "t shows usage when called without arguments"
    setup
    set output (t 2>&1)
    string match -q "*Usage: t <repo>*" $output
    and string match -q "*--help*" $output
    teardown
