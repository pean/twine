#!/usr/bin/env fish

# Test t command (session switcher)

set -g test_dir (mktemp -d)
mkdir -p $test_dir/repos
set -gx TWINE_BASE_DIRS $test_dir/repos
set -gx TWINE_SESSION_PREFIX ""
set -gx TWINE_USE_TMUXINATOR false
source conf.d/twine.fish 2>/dev/null
source functions/t.fish

# Test: t errors when TWINE_BASE_DIRS not configured
set -e TWINE_BASE_DIRS
set output (t test 2>&1)
@test "t errors when TWINE_BASE_DIRS not configured" (string match -q "*TWINE_BASE_DIRS not configured*" -- $output) $status -eq 0
set -gx TWINE_BASE_DIRS $test_dir/repos

# Test: t builds session names with and without prefix
set -gx TWINE_SESSION_PREFIX "work-"
set names $TWINE_SESSION_PREFIX"test" "test"
@test "t builds session names with and without prefix" (test (count $names) -eq 2; and test "$names[1]" = "work-test"; and test "$names[2]" = "test") $status -eq 0
set -gx TWINE_SESSION_PREFIX ""

# Test: t searches for repo in base directories
mkdir -p $test_dir/repos/test-repo
cd $test_dir/repos/test-repo
git init >/dev/null 2>&1
set name "test-repo"
set found 0
for base_dir in $TWINE_BASE_DIRS
    if test -d $base_dir/$name
        set found 1
        break
    end
end
@test "t searches for repo in base directories" (test $found -eq 1) $status -eq 0

# Test: t finds worktree in bare repo
cd $test_dir/repos
git init --bare main.git >/dev/null 2>&1
mkdir -p main.git/main
set name "main"
set found 0
for base_dir in $TWINE_BASE_DIRS
    if test -d $base_dir/$name.git/$name
        set found 1
        break
    end
end
@test "t finds worktree in bare repo" (test $found -eq 1) $status -eq 0

# Test: t shows usage when called without arguments
set output (t 2>&1)
@test "t shows usage when called without arguments" (string match -q "*Usage: t <repo>*" -- $output; and string match -q "*--help*" -- $output) $status -eq 0

# Cleanup
rm -rf $test_dir
set -e TWINE_BASE_DIRS
set -e TWINE_SESSION_PREFIX
set -e TWINE_USE_TMUXINATOR
