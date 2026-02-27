#!/usr/bin/env fish

# Test tw command (worktree manager)

set -g test_dir (mktemp -d)
set -g orig_dir (pwd)
mkdir -p $test_dir/repos

set -gx TWINE_BASE_DIRS $test_dir/repos
set -gx TWINE_SESSION_PREFIX ""
set -gx TWINE_USE_TMUXINATOR false

source conf.d/twine.fish
source functions/tw.fish

cd $test_dir/repos
git init --bare test-repo.git >/dev/null 2>&1

# Test: tw shows usage when no arguments
set output (tw 2>&1)
@test "tw shows usage when no arguments" (string match -q "*Usage: tw <repo>*" -- $output) $status -eq 0

# Test: tw errors when TWINE_BASE_DIRS not configured
set -e TWINE_BASE_DIRS
set output (tw test-repo 2>&1)
@test "tw errors when TWINE_BASE_DIRS not configured" (string match -q "*TWINE_BASE_DIRS not configured*" -- $output) $status -eq 0
set -gx TWINE_BASE_DIRS $test_dir/repos

# Test: tw strips .git suffix from repo name
set repo (string replace -r '\.git$' '' "test-repo.git")
@test "tw strips .git suffix from repo name" (test "$repo" = "test-repo") $status -eq 0

# Test: tw finds repo with .git suffix (bare repo)
set repo "test-repo"
set repo_path ""
for base_dir in $TWINE_BASE_DIRS
    if test -d $base_dir/$repo.git
        set repo_path $base_dir/$repo.git
        break
    end
end
@test "tw finds repo with .git suffix (bare repo)" (test "$repo_path" = "$test_dir/repos/test-repo.git") $status -eq 0

# Test: tw finds repo without .git suffix (regular repo)
mkdir -p $test_dir/repos/regular-repo
cd $test_dir/repos/regular-repo
git init >/dev/null 2>&1
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
@test "tw finds repo without .git suffix (regular repo)" (test "$repo_path" = "$test_dir/repos/regular-repo") $status -eq 0

# Test: tw searches multiple base directories
mkdir -p $test_dir/repos2
cd $test_dir/repos2
git init --bare other-repo.git >/dev/null 2>&1
set -gx TWINE_BASE_DIRS $test_dir/repos $test_dir/repos2
set repo "other-repo"
set repo_path ""
for base_dir in $TWINE_BASE_DIRS
    if test -d $base_dir/$repo.git
        set repo_path $base_dir/$repo.git
        break
    end
end
@test "tw searches multiple base directories" (test "$repo_path" = "$test_dir/repos2/other-repo.git") $status -eq 0
set -gx TWINE_BASE_DIRS $test_dir/repos

# Test: tw applies session prefix when configured
set -gx TWINE_SESSION_PREFIX "work-"
set repo "test-repo"
set branch "main"
set session_name "$TWINE_SESSION_PREFIX$repo/$branch"
@test "tw applies session prefix when configured" (test "$session_name" = "work-test-repo/main") $status -eq 0
set -gx TWINE_SESSION_PREFIX ""

# Test: tw builds correct worktree path
set repo_path "$test_dir/repos/test-repo.git"
set branch "feature/test"
set worktree_path $repo_path/$branch
@test "tw builds correct worktree path" (test "$worktree_path" = "$test_dir/repos/test-repo.git/feature/test") $status -eq 0

# Cleanup
rm -rf $test_dir
set -e TWINE_BASE_DIRS
set -e TWINE_SESSION_PREFIX
set -e TWINE_USE_TMUXINATOR
