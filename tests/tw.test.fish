#!/usr/bin/env fish

# Test tw command (worktree manager)

set -g test_dir (mktemp -d)

function setup
    # Create test directory structure
    mkdir -p $test_dir/repos

    # Create a mock bare repo
    cd $test_dir/repos
    git init --bare test-repo.git >/dev/null 2>&1

    # Configure test environment
    set -gx TWINE_BASE_DIRS $test_dir/repos
    set -gx TWINE_SESSION_PREFIX ""
    set -gx TWINE_USE_TMUXINATOR false

    # Source the functions
    source conf.d/twine.fish
    source functions/tw.fish
end

function teardown
    rm -rf $test_dir
    set -e TWINE_BASE_DIRS
    set -e TWINE_SESSION_PREFIX
    set -e TWINE_USE_TMUXINATOR
end

@test "tw shows usage when no arguments"
    setup
    set output (tw 2>&1)
    string match -q "*Usage: tw <repo>*" $output
    teardown

@test "tw errors when TWINE_BASE_DIRS not configured"
    setup
    set -e TWINE_BASE_DIRS
    set output (tw test-repo 2>&1)
    string match -q "*TWINE_BASE_DIRS not configured*" $output
    teardown

@test "tw strips .git suffix from repo name"
    setup

    # Mock the repo finding (since we need actual git operations for real test)
    function tw_test_strip
        set repo (string replace -r '\.git$' '' "test-repo.git")
        test "$repo" = "test-repo"
    end

    tw_test_strip
    functions -e tw_test_strip
    teardown

@test "tw finds repo with .git suffix (bare repo)"
    setup

    # Test that repo_path gets set correctly for bare repo
    set repo "test-repo"
    set repo_path ""

    for base_dir in $TWINE_BASE_DIRS
        if test -d $base_dir/$repo.git
            set repo_path $base_dir/$repo.git
            break
        end
    end

    test "$repo_path" = "$test_dir/repos/test-repo.git"
    teardown

@test "tw finds repo without .git suffix (regular repo)"
    setup

    # Create a regular repo
    mkdir -p $test_dir/repos/regular-repo
    cd $test_dir/repos/regular-repo
    git init >/dev/null 2>&1

    # Test that repo_path gets set correctly
    set repo "regular-repo"
    set repo_path ""

    for base_dir in $TWINE_BASE_DIRS
        if test -d $base_dir/$repo.git
            set repo_path $base_dir/$repo.git
            break
        else if test -d $base_dir/$repo
            set repo_path $base_dir/$repo
            break
        end
    end

    test "$repo_path" = "$test_dir/repos/regular-repo"
    teardown

@test "tw searches multiple base directories"
    setup

    # Create second base directory
    mkdir -p $test_dir/repos2
    cd $test_dir/repos2
    git init --bare other-repo.git >/dev/null 2>&1

    set -gx TWINE_BASE_DIRS $test_dir/repos $test_dir/repos2

    # Test finding in second directory
    set repo "other-repo"
    set repo_path ""

    for base_dir in $TWINE_BASE_DIRS
        if test -d $base_dir/$repo.git
            set repo_path $base_dir/$repo.git
            break
        end
    end

    test "$repo_path" = "$test_dir/repos2/other-repo.git"
    teardown

@test "tw applies session prefix when configured"
    setup
    set -gx TWINE_SESSION_PREFIX "work-"

    set repo "test-repo"
    set branch "main"
    set session_name "$TWINE_SESSION_PREFIX$repo/$branch"

    test "$session_name" = "work-test-repo/main"
    teardown

@test "tw builds correct worktree path"
    setup

    set repo_path "$test_dir/repos/test-repo.git"
    set branch "feature/test"
    set worktree_path $repo_path/$branch

    test "$worktree_path" = "$test_dir/repos/test-repo.git/feature/test"
    teardown
