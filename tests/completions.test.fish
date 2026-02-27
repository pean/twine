#!/usr/bin/env fish

# Test completions for tw and t commands

set -g test_dir (mktemp -d)
mkdir -p $test_dir/repos
set -gx TWINE_BASE_DIRS $test_dir/repos
source completions/tw.fish 2>/dev/null
source completions/t.fish 2>/dev/null
source completions/twine.fish 2>/dev/null

# Test: tw completion lists repos with .git suffix
mkdir -p $test_dir/repos/repo1.git
mkdir -p $test_dir/repos/repo2.git
set output (__tw_complete_repos)
@test "tw completion lists repos with .git suffix" (string match -q "*repo1*" -- $output; and string match -q "*repo2*" -- $output) $status -eq 0

# Test: tw completion strips .git suffix from repo names
mkdir -p $test_dir/repos/test-repo.git
set output (__tw_complete_repos)
@test "tw completion strips .git suffix from repo names" (string match -q "*test-repo*" -- $output; and not string match -q "*.git*" -- $output) $status -eq 0

# Test: tw completion includes tmux sessions
function tmux
    if test "$argv[1]" = "has-session"
        return 0
    else if test "$argv[1]" = "list-sessions"
        echo "repo1/main: 1 windows"
        echo "repo2/develop: 2 windows"
        return 0
    end
end
set output (__tw_complete_repos)
@test "tw completion includes tmux sessions" (string match -q "*repo1*" -- $output; and string match -q "*repo2*" -- $output) $status -eq 0
functions -e tmux

# Test: tw completion searches multiple base directories
mkdir -p $test_dir/repos2
set -gx TWINE_BASE_DIRS $test_dir/repos $test_dir/repos2
mkdir -p $test_dir/repos/repo1.git
mkdir -p $test_dir/repos2/repo2.git
set output (__tw_complete_repos)
@test "tw completion searches multiple base directories" (string match -q "*repo1*" -- $output; and string match -q "*repo2*" -- $output) $status -eq 0
set -gx TWINE_BASE_DIRS $test_dir/repos

# Test: tw branch completion finds repo in base directories
cd $test_dir/repos
git init --bare test-repo.git >/dev/null 2>&1
function commandline
    if test "$argv[1]" = "-opc"
        echo tw
        echo test-repo
    end
end
set repo (commandline -opc)[2]
set repo (string replace -r '\.git$' '' $repo)
set repo_path ""
for base_dir in $TWINE_BASE_DIRS
    if test -d $base_dir/$repo.git
        set repo_path $base_dir/$repo.git
        break
    end
end
@test "tw branch completion finds repo in base directories" (test "$repo_path" = "$test_dir/repos/test-repo.git") $status -eq 0
functions -e commandline

# Test: t completion lists directories from base dirs
mkdir -p $test_dir/repos/repo1
mkdir -p $test_dir/repos/repo2
function tmux
    if test "$argv[1]" = "has-session"
        return 1
    end
end
set output (__t_complete_t)
@test "t completion lists directories from base dirs" (string match -q "*repo1*" -- $output; and string match -q "*repo2*" -- $output) $status -eq 0
functions -e tmux

# Test: t completion includes tmux sessions
function tmux
    if test "$argv[1]" = "has-session"
        return 0
    else if test "$argv[1]" = "list-sessions"
        echo "session1: 1 windows"
        echo "session2: 2 windows"
        return 0
    end
end
set output (__t_complete_t)
@test "t completion includes tmux sessions" (string match -q "*session1*" -- $output; and string match -q "*session2*" -- $output) $status -eq 0
functions -e tmux

# Test: t completion searches multiple base directories
mkdir -p $test_dir/repos2
set -gx TWINE_BASE_DIRS $test_dir/repos $test_dir/repos2
mkdir -p $test_dir/repos/repo1
mkdir -p $test_dir/repos2/repo2
function tmux
    if test "$argv[1]" = "has-session"
        return 1
    end
end
set output (__t_complete_t)
@test "t completion searches multiple base directories" (string match -q "*repo1*" -- $output; and string match -q "*repo2*" -- $output) $status -eq 0
functions -e tmux
set -gx TWINE_BASE_DIRS $test_dir/repos

# Test: twine completion includes all verbose actions
set completions (complete -C "twine " 2>/dev/null | string match -v "*Shortcut*")
@test "twine completion includes all verbose actions" (string match -q "*worktree*" -- $completions; and string match -q "*session*" -- $completions; and string match -q "*attach*" -- $completions; and string match -q "*start*" -- $completions; and string match -q "*init*" -- $completions; and string match -q "*convert*" -- $completions) $status -eq 0

# Test: twine completion includes only tw t ts shortcuts
set shortcuts (complete -C "twine " 2>/dev/null | string match "*Shortcut*")
set shortcut_count (count $shortcuts)
@test "twine completion includes only tw t ts shortcuts" (test $shortcut_count -eq 3; and string match -q "*tw*" -- $shortcuts; and string match -q "*ts*" -- $shortcuts) $status -eq 0

# Cleanup
rm -rf $test_dir
set -e TWINE_BASE_DIRS
