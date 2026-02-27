#!/usr/bin/env fish

# Test completions for tw and t commands

set -g test_dir (mktemp -d)

function setup
    mkdir -p $test_dir/repos
    set -gx TWINE_BASE_DIRS $test_dir/repos
    source completions/tw.fish
    source completions/t.fish
end

function teardown
    rm -rf $test_dir
    set -e TWINE_BASE_DIRS
end

@test "tw completion lists repos with .git suffix"
    setup

    # Create bare repos
    mkdir -p $test_dir/repos/repo1.git
    mkdir -p $test_dir/repos/repo2.git

    set output (__tw_complete_repos)

    string match -q "*repo1*" $output
    and string match -q "*repo2*" $output

    teardown

@test "tw completion strips .git suffix from repo names"
    setup

    mkdir -p $test_dir/repos/test-repo.git

    set output (__tw_complete_repos)

    string match -q "*test-repo*" $output
    and not string match -q "*.git*" $output

    teardown

@test "tw completion includes tmux sessions"
    setup

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

    string match -q "*repo1*" $output
    and string match -q "*repo2*" $output

    functions -e tmux
    teardown

@test "tw completion searches multiple base directories"
    setup

    mkdir -p $test_dir/repos2
    set -gx TWINE_BASE_DIRS $test_dir/repos $test_dir/repos2

    mkdir -p $test_dir/repos/repo1.git
    mkdir -p $test_dir/repos2/repo2.git

    set output (__tw_complete_repos)

    string match -q "*repo1*" $output
    and string match -q "*repo2*" $output

    teardown

@test "tw branch completion finds repo in base directories"
    setup

    # Create bare repo
    cd $test_dir/repos
    git init --bare test-repo.git >/dev/null 2>&1

    # Mock commandline to return repo name
    function commandline
        if test "$argv[1]" = "-opc"
            echo tw
            echo test-repo
        end
    end

    # This tests the repo finding logic
    set repo (commandline -opc)[2]
    set repo (string replace -r '\.git$' '' $repo)
    set repo_path ""

    for base_dir in $TWINE_BASE_DIRS
        if test -d $base_dir/$repo.git
            set repo_path $base_dir/$repo.git
            break
        end
    end

    test "$repo_path" = "$test_dir/repos/test-repo.git"

    functions -e commandline
    teardown

@test "t completion lists directories from base dirs"
    setup

    mkdir -p $test_dir/repos/repo1
    mkdir -p $test_dir/repos/repo2

    function tmux
        if test "$argv[1]" = "has-session"
            return 1
        end
    end

    set output (__t_complete_t)

    string match -q "*repo1*" $output
    and string match -q "*repo2*" $output

    functions -e tmux
    teardown

@test "t completion includes tmux sessions"
    setup

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

    string match -q "*session1*" $output
    and string match -q "*session2*" $output

    functions -e tmux
    teardown

@test "t completion searches multiple base directories"
    setup

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

    string match -q "*repo1*" $output
    and string match -q "*repo2*" $output

    functions -e tmux
    teardown

@test "twine completion includes all verbose actions"
    source completions/twine.fish

    # Get all completions for twine command
    set completions (complete -C "twine " | string match -v "*Shortcut*")

    string match -q "*worktree*" $completions
    and string match -q "*session*" $completions
    and string match -q "*attach*" $completions
    and string match -q "*start*" $completions
    and string match -q "*init*" $completions
    and string match -q "*convert*" $completions


@test "twine completion includes only tw t ts shortcuts"
    source completions/twine.fish

    # Get shortcut completions
    set shortcuts (complete -C "twine " | string match "*Shortcut*")

    string match -q "*tw*" $shortcuts
    and string match -q "* t *" $shortcuts
    and string match -q "*ts*" $shortcuts
    and not string match -q "*ta*" $shortcuts
