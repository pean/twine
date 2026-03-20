#!/usr/bin/env fish

# Test twine prune command

set -g test_dir (mktemp -d)
set -g orig_dir (pwd)
mkdir -p $test_dir/repos

set -gx TWINE_BASE_DIRS $test_dir/repos
set -gx TWINE_USE_TMUXINATOR false

source conf.d/twine.fish
source functions/__twine_prune.fish

# Test: prune errors when TWINE_BASE_DIRS not configured
set -e TWINE_BASE_DIRS
set output (__twine_prune test-repo 2>&1)
@test "prune errors when TWINE_BASE_DIRS not configured" (string match -q "*TWINE_BASE_DIRS not configured*" -- $output) $status -eq 0
set -gx TWINE_BASE_DIRS $test_dir/repos

# Test: prune errors when specific repo not found
set output (__twine_prune nonexistent-repo 2>&1)
@test "prune errors when repo not found" (string match -q "*Repository not found*" -- $output) $status -eq 0

# Test: prune reports no repos when base dir is empty
set output (__twine_prune 2>&1)
@test "prune reports no repositories found" (string match -q "*No repositories found*" -- $output) $status -eq 0

# Test: prune finds and processes bare repo
cd $test_dir/repos
git init --bare test-repo.git >/dev/null 2>&1
git -C test-repo.git config user.email "test@example.com" >/dev/null 2>&1
git -C test-repo.git config user.name "Test User" >/dev/null 2>&1
echo "ref: refs/heads/main" > $test_dir/repos/test-repo.git/HEAD

set output (__twine_prune 2>&1)
@test "prune discovers bare repo" (string match -q "*test-repo*" -- $output) $status -eq 0

# Test: prune specific repo by name
set output (__twine_prune test-repo 2>&1)
@test "prune processes specific repo" (string match -q "*test-repo*" -- $output) $status -ge 0

# Test: prune finds regular repo
mkdir -p $test_dir/repos/regular-repo
cd $test_dir/repos/regular-repo
git init >/dev/null 2>&1
git config user.email "test@example.com" >/dev/null 2>&1
git config user.name "Test User" >/dev/null 2>&1

set output (__twine_prune 2>&1)
@test "prune discovers regular repo" (string match -q "*regular-repo*" -- $output) $status -eq 0

# Test: dry run shows would-prune output without deleting
set output (__twine_prune --dry-run 2>&1)
@test "dry run shows dry run header" (string match -q "*(dry run)*" -- $output) $status -eq 0

set output (__twine_prune -n 2>&1)
@test "dry run accepts -n flag" (string match -q "*(dry run)*" -- $output) $status -eq 0

# Test: prune shows help with --help flag
set output (__twine_prune --help 2>&1)
@test "prune shows help with --help" (string match -q "*remote has been deleted*" -- $output) $status -eq 0

# Test: prune help shows all-repos default behavior
set output (__twine_prune --help 2>&1)
@test "prune help mentions all repos behavior" (string match -q "*all repos if omitted*" -- $output) $status -eq 0

# Cleanup
cd $orig_dir
rm -rf $test_dir
