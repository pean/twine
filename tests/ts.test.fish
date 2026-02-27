#!/usr/bin/env fish

# Test ts command (tmuxinator wrapper)

set -g test_dir (mktemp -d)
source functions/ts.fish

# Test: ts errors when tmuxinator not installed
set -l old_PATH $PATH
set -gx PATH /nonexistent
set output (ts 2>&1)
@test "ts errors when tmuxinator not installed" (string match -q "*tmuxinator not installed*" -- $output) $status -eq 0
set -gx PATH $old_PATH

# Test: ts errors when not in a git repo
set -l temp_bin (mktemp -d)
touch $temp_bin/tmuxinator
chmod +x $temp_bin/tmuxinator
set -l old_PATH $PATH
set -gx PATH $temp_bin $PATH
function git
    if test "$argv[1]" = "rev-parse"
        return 1
    end
end
cd $test_dir
set output (ts 2>&1)
@test "ts errors when not in a git repo" (string match -q "*Not a repo*" -- $output) $status -eq 0
functions -e git
set -gx PATH $old_PATH
rm -rf $temp_bin

# Test: ts extracts repo name correctly
mkdir -p $test_dir/test-repo
cd $test_dir/test-repo
set -l temp_bin (mktemp -d)
touch $temp_bin/tmuxinator
chmod +x $temp_bin/tmuxinator
set -l old_PATH $PATH
set -gx PATH $temp_bin $PATH
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
@test "ts extracts repo name correctly" (string match -q "*Starting tmuxinator for test-repo*" -- $output; and string match -q "*tmuxinator: start -n test-repo dev*" -- $output) $status -eq 0
functions -e git
functions -e tmuxinator
set -gx PATH $old_PATH
rm -rf $temp_bin

# Test: ts strips leading dot from repo name
mkdir -p $test_dir/.dotrepo
cd $test_dir/.dotrepo
set -l temp_bin (mktemp -d)
touch $temp_bin/tmuxinator
chmod +x $temp_bin/tmuxinator
set -l old_PATH $PATH
set -gx PATH $temp_bin $PATH
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
@test "ts strips leading dot from repo name" (string match -q "*tmuxinator: start -n dotrepo dev*" -- $output) $status -eq 0
functions -e git
functions -e tmuxinator
set -gx PATH $old_PATH
rm -rf $temp_bin

# Cleanup
rm -rf $test_dir
